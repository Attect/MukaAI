// Package helpers 提供项目级共享测试工具函数
// 集成测试和跨模块测试可引用此包以减少重复代码
package helpers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"

	"github.com/Attect/MukaAI/internal/model"
	"github.com/Attect/MukaAI/internal/state"
	"github.com/Attect/MukaAI/internal/tools"
)

// TempDirFixture 提供临时目录的创建和清理
// 使用方在测试结束后调用 Cleanup 移除临时目录
type TempDirFixture struct {
	Dir string
}

// NewTempDirFixture 创建临时目录夹具
// 在测试用例开头调用，defer fixture.Cleanup() 清理
func NewTempDirFixture(prefix string) (*TempDirFixture, error) {
	dir, err := os.MkdirTemp("", prefix+"-*")
	if err != nil {
		return nil, err
	}
	return &TempDirFixture{Dir: dir}, nil
}

// Cleanup 移除临时目录及其所有内容
func (f *TempDirFixture) Cleanup() {
	if f.Dir != "" {
		os.RemoveAll(f.Dir)
	}
}

// Path 返回临时目录下指定子路径
func (f *TempDirFixture) Path(parts ...string) string {
	return filepath.Join(append([]string{f.Dir}, parts...)...)
}

// CreateFile 在临时目录中创建指定文件并写入内容
func (f *TempDirFixture) CreateFile(relPath string, content []byte) error {
	fullPath := f.Path(relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, content, 0644)
}

// MockAPIServerConfig 模拟API服务器的配置
type MockAPIServerConfig struct {
	// Responses 按顺序返回的响应列表
	Responses []*model.ChatCompletionResponse
	// DefaultResponse 超出Responses列表时的默认响应
	DefaultResponse *model.ChatCompletionResponse
}

// NewMockAPIServer 创建模拟API服务器
// 支持：流式和非流式响应、工具调用、按顺序返回多个响应
func NewMockAPIServer(cfg MockAPIServerConfig) *httptest.Server {
	callCount := 0
	var mu sync.Mutex

	defaultResp := cfg.DefaultResponse
	if defaultResp == nil {
		defaultResp = &model.ChatCompletionResponse{
			Choices: []model.Choice{
				{
					Message: model.Message{
						Role:    model.RoleAssistant,
						Content: "任务已完成",
					},
					FinishReason: "stop",
				},
			},
		}
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		var resp *model.ChatCompletionResponse
		if callCount < len(cfg.Responses) {
			resp = cfg.Responses[callCount]
			callCount++
		} else {
			resp = defaultResp
		}
		mu.Unlock()

		var req model.ChatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			flusher, _ := w.(http.Flusher)

			if resp != nil && len(resp.Choices) > 0 {
				msg := resp.Choices[0].Message
				if msg.Content != "" {
					data, _ := json.Marshal(&model.StreamResponse{
						Choices: []model.Choice{
							{Delta: &model.Delta{Content: msg.Content}},
						},
					})
					w.Write([]byte("data: " + string(data) + "\n\n"))
					flusher.Flush()
				}
				if len(msg.ToolCalls) > 0 {
					for _, tc := range msg.ToolCalls {
						data, _ := json.Marshal(&model.StreamResponse{
							Choices: []model.Choice{
								{Delta: &model.Delta{ToolCalls: []model.ToolCall{tc}}},
							},
						})
						w.Write([]byte("data: " + string(data) + "\n\n"))
						flusher.Flush()
					}
				}
			}
			w.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}
	}))
}

// MockTool 模拟工具，用于测试工具注册和执行
type MockTool struct {
	NameField        string
	DescriptionField string
	ExecuteFunc      func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error)
}

func (t *MockTool) Name() string        { return t.NameField }
func (t *MockTool) Description() string { return t.DescriptionField }
func (t *MockTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}
func (t *MockTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	if t.ExecuteFunc != nil {
		return t.ExecuteFunc(ctx, params)
	}
	return tools.NewSuccessResult("mock result"), nil
}

// TestEnvironment 提供一个完整的测试环境
// 包含临时目录、状态管理器、工具注册中心、模拟API服务器和模型客户端
type TestEnvironment struct {
	Fixture      *TempDirFixture
	StateManager *state.StateManager
	ToolRegistry *tools.ToolRegistry
	ModelClient  *model.Client
	APIServer    *httptest.Server
}

// NewTestEnvironment 创建完整的测试环境
// 使用方在测试结束后调用 Cleanup 清理所有资源
func NewTestEnvironment(responses []*model.ChatCompletionResponse) (*TestEnvironment, error) {
	fixture, err := NewTempDirFixture("muka-test")
	if err != nil {
		return nil, err
	}

	stateManager, err := state.NewStateManager(filepath.Join(fixture.Dir, "state"), true)
	if err != nil {
		fixture.Cleanup()
		return nil, err
	}

	registry := tools.NewToolRegistry()
	server := NewMockAPIServer(MockAPIServerConfig{Responses: responses})

	config := model.DefaultConfig()
	config.Endpoint = server.URL + "/"
	config.APIKey = "test-key"
	config.ModelName = "test-model"

	client, err := model.NewClient(config)
	if err != nil {
		server.Close()
		fixture.Cleanup()
		return nil, err
	}

	return &TestEnvironment{
		Fixture:      fixture,
		StateManager: stateManager,
		ToolRegistry: registry,
		ModelClient:  client,
		APIServer:    server,
	}, nil
}

// Cleanup 清理所有测试资源
func (env *TestEnvironment) Cleanup() {
	if env.APIServer != nil {
		env.APIServer.Close()
	}
	if env.Fixture != nil {
		env.Fixture.Cleanup()
	}
}
