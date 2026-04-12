package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"agentplus/internal/model"
	"agentplus/internal/state"
	"agentplus/internal/tools"
)

// mockAPIServer 创建模拟API服务器
func mockAPIServer(responses []*model.ChatCompletionResponse) *httptest.Server {
	callCount := 0
	var mu sync.Mutex

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		var resp *model.ChatCompletionResponse
		if callCount < len(responses) {
			resp = responses[callCount]
			callCount++
		} else {
			// 默认响应
			resp = &model.ChatCompletionResponse{
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
		mu.Unlock()

		// 检查是否是流式请求
		var req model.ChatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Stream {
			// 流式响应
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			flusher, _ := w.(http.Flusher)

			// 发送内容
			if resp != nil && len(resp.Choices) > 0 {
				msg := resp.Choices[0].Message

				// 发送内容
				if msg.Content != "" {
					data, _ := json.Marshal(&model.StreamResponse{
						Choices: []model.Choice{
							{
								Delta: &model.Delta{
									Content: msg.Content,
								},
							},
						},
					})
					w.Write([]byte("data: " + string(data) + "\n\n"))
					flusher.Flush()
				}

				// 发送工具调用
				if len(msg.ToolCalls) > 0 {
					for _, tc := range msg.ToolCalls {
						data, _ := json.Marshal(&model.StreamResponse{
							Choices: []model.Choice{
								{
									Delta: &model.Delta{
										ToolCalls: []model.ToolCall{tc},
									},
								},
							},
						})
						w.Write([]byte("data: " + string(data) + "\n\n"))
						flusher.Flush()
					}
				}
			}

			// 发送完成信号
			w.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
		} else {
			// 非流式响应
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}
	}))
}

// mockTool 模拟工具
type mockTool struct {
	name        string
	description string
	executeFunc func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error)
}

func (t *mockTool) Name() string {
	return t.name
}

func (t *mockTool) Description() string {
	return t.description
}

func (t *mockTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *mockTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	if t.executeFunc != nil {
		return t.executeFunc(ctx, params)
	}
	return tools.NewSuccessResult("mock result"), nil
}

// setupTestAgent 创建测试用的Agent
func setupTestAgent(t *testing.T, responses []*model.ChatCompletionResponse) (*Agent, *tools.ToolRegistry, *state.StateManager, func()) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 创建状态管理器
	stateManager, err := state.NewStateManager(filepath.Join(tmpDir, "state"), true)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create state manager: %v", err)
	}

	// 创建工具注册中心
	registry := tools.NewToolRegistry()

	// 创建mock服务器
	server := mockAPIServer(responses)

	// 创建模型客户端
	config := model.DefaultConfig()
	config.Endpoint = server.URL + "/"
	config.APIKey = "test-key"
	config.ModelName = "test-model"

	client, err := model.NewClient(config)
	if err != nil {
		server.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create model client: %v", err)
	}

	// 创建Agent
	agent, err := NewAgent(&Config{
		ModelClient:   client,
		ToolRegistry:  registry,
		StateManager:  stateManager,
		MaxIterations: 10,
	})
	if err != nil {
		server.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create agent: %v", err)
	}

	cleanup := func() {
		server.Close()
		os.RemoveAll(tmpDir)
	}

	return agent, registry, stateManager, cleanup
}

func TestNewAgent(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stateManager, _ := state.NewStateManager(filepath.Join(tmpDir, "state"), true)
	registry := tools.NewToolRegistry()

	// 创建mock服务器
	server := mockAPIServer(nil)
	defer server.Close()

	config := model.DefaultConfig()
	config.Endpoint = server.URL + "/"
	config.APIKey = "test-key"

	client, _ := model.NewClient(config)

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				ModelClient:   client,
				ToolRegistry:  registry,
				StateManager:  stateManager,
				MaxIterations: 10,
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "nil model client",
			config: &Config{
				ToolRegistry: registry,
				StateManager: stateManager,
			},
			wantErr: true,
		},
		{
			name: "nil tool registry",
			config: &Config{
				ModelClient:  client,
				StateManager: stateManager,
			},
			wantErr: true,
		},
		{
			name: "nil state manager",
			config: &Config{
				ModelClient:  client,
				ToolRegistry: registry,
			},
			wantErr: true,
		},
		{
			name: "default max iterations",
			config: &Config{
				ModelClient:  client,
				ToolRegistry: registry,
				StateManager: stateManager,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewAgent(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && agent == nil {
				t.Error("NewAgent() returned nil agent")
			}
		})
	}
}

func TestHistoryManager(t *testing.T) {
	history := NewHistoryManager()

	// 测试添加消息
	msg := model.NewUserMessage("test message")
	history.AddMessage(msg)

	if history.GetMessageCount() != 1 {
		t.Errorf("expected 1 message, got %d", history.GetMessageCount())
	}

	// 测试获取消息
	messages := history.GetMessages()
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}

	// 测试清空
	history.Clear()
	if history.GetMessageCount() != 0 {
		t.Errorf("expected 0 messages after clear, got %d", history.GetMessageCount())
	}
}

func TestHistoryManagerTruncate(t *testing.T) {
	history := NewHistoryManager()

	// 添加系统消息
	history.AddMessage(model.NewSystemMessage("system prompt"))

	// 添加多条用户消息
	for i := 0; i < 10; i++ {
		history.AddMessage(model.NewUserMessage("message"))
	}

	// 简单截断
	history.TruncateSimple(5)

	messages := history.GetMessages()

	// 应该有1条系统消息 + 5条用户消息
	if len(messages) != 6 {
		t.Errorf("expected 6 messages after truncate, got %d", len(messages))
	}

	// 第一条应该是系统消息
	if messages[0].Role != model.RoleSystem {
		t.Error("first message should be system message")
	}
}

func TestToolExecutor(t *testing.T) {
	registry := tools.NewToolRegistry()

	// 注册模拟工具
	mockTool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
	}
	registry.RegisterTool(mockTool)

	executor := NewToolExecutor(registry)

	// 测试获取工具Schema
	schemas := executor.GetToolSchemas()
	if len(schemas) != 1 {
		t.Errorf("expected 1 tool schema, got %d", len(schemas))
	}

	// 测试检查工具是否存在
	if !executor.HasTool("test_tool") {
		t.Error("expected tool to exist")
	}

	if executor.HasTool("nonexistent") {
		t.Error("expected nonexistent tool to not exist")
	}
}

func TestToolExecutorExecute(t *testing.T) {
	registry := tools.NewToolRegistry()

	// 注册模拟工具
	mockTool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		executeFunc: func(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
			return tools.NewSuccessResult("executed"), nil
		},
	}
	registry.RegisterTool(mockTool)

	executor := NewToolExecutor(registry)

	// 执行工具调用
	toolCalls := []model.ToolCall{
		{
			ID:   "call-1",
			Type: "function",
			Function: model.FunctionCall{
				Name:      "test_tool",
				Arguments: "{}",
			},
		},
	}

	results, err := executor.ExecuteToolCalls(context.Background(), toolCalls)
	if err != nil {
		t.Fatalf("failed to execute tool calls: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	// 检查结果
	if results[0].Role != model.RoleTool {
		t.Error("expected tool role")
	}
}

func TestAgentRunSimple(t *testing.T) {
	responses := []*model.ChatCompletionResponse{
		{
			Choices: []model.Choice{
				{
					Message: model.Message{
						Role:    model.RoleAssistant,
						Content: "任务已完成",
					},
					FinishReason: "stop",
				},
			},
		},
	}

	agent, _, _, cleanup := setupTestAgent(t, responses)
	defer cleanup()

	// 运行Agent
	result, err := agent.Run(context.Background(), "test task")
	if err != nil {
		t.Fatalf("agent run failed: %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", result.Status)
	}
}

func TestAgentRunWithToolCall(t *testing.T) {
	responses := []*model.ChatCompletionResponse{
		{
			Choices: []model.Choice{
				{
					Message: model.Message{
						Role: model.RoleAssistant,
						ToolCalls: []model.ToolCall{
							{
								ID:   "call-1",
								Type: "function",
								Function: model.FunctionCall{
									Name:      "complete_task",
									Arguments: "{}",
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
		},
	}

	agent, registry, _, cleanup := setupTestAgent(t, responses)
	defer cleanup()

	// 注册complete_task工具
	completeTool := &mockTool{
		name:        "complete_task",
		description: "Mark task as complete",
	}
	registry.RegisterTool(completeTool)

	// 运行Agent
	result, err := agent.Run(context.Background(), "test task")
	if err != nil {
		t.Fatalf("agent run failed: %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", result.Status)
	}
}

func TestAgentStop(t *testing.T) {
	// 使用一个阻塞的mock服务器来测试取消
	callCount := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		currentCall := callCount
		mu.Unlock()

		// 第一个请求阻塞较长时间
		if currentCall == 1 {
			time.Sleep(500 * time.Millisecond)
		}

		// 流式响应
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, _ := w.(http.Flusher)

		data, _ := json.Marshal(&model.StreamResponse{
			Choices: []model.Choice{
				{
					Delta: &model.Delta{
						Content: "working...",
					},
				},
			},
		})
		w.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()

		w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stateManager, _ := state.NewStateManager(filepath.Join(tmpDir, "state"), true)
	registry := tools.NewToolRegistry()

	config := model.DefaultConfig()
	config.Endpoint = server.URL + "/"
	config.APIKey = "test-key"
	config.ModelName = "test-model"

	client, _ := model.NewClient(config)

	agent, _ := NewAgent(&Config{
		ModelClient:   client,
		ToolRegistry:  registry,
		StateManager:  stateManager,
		MaxIterations: 100,
	})

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 在另一个goroutine中运行Agent
	done := make(chan error, 1)
	go func() {
		_, err := agent.Run(ctx, "test task")
		done <- err
	}()

	// 等待一小段时间后取消
	time.Sleep(50 * time.Millisecond)
	cancel()

	// 等待Agent完成
	select {
	case err := <-done:
		if err == nil {
			t.Error("expected error due to cancellation")
		}
	case <-time.After(3 * time.Second):
		t.Error("test timeout - agent did not respond to cancellation")
	}
}

func TestAgentMaxIterations(t *testing.T) {
	// 创建不会完成的响应
	responses := make([]*model.ChatCompletionResponse, 5)
	for i := 0; i < 5; i++ {
		responses[i] = &model.ChatCompletionResponse{
			Choices: []model.Choice{
				{
					Message: model.Message{
						Role:    model.RoleAssistant,
						Content: "working...",
					},
					FinishReason: "stop",
				},
			},
		}
	}

	agent, _, _, cleanup := setupTestAgent(t, responses)
	defer cleanup()

	// 设置较小的最大迭代次数
	agent.maxIterations = 2

	// 运行Agent
	result, err := agent.Run(context.Background(), "test task")

	// 应该返回错误
	if err == nil {
		t.Error("expected error due to max iterations")
	}

	if result.Status != "max_iterations" {
		t.Errorf("expected status 'max_iterations', got '%s'", result.Status)
	}
}

func TestPrompts(t *testing.T) {
	// 测试获取系统提示词
	prompt := GetSystemPrompt(PromptTypeOrchestrator)
	if prompt == "" {
		t.Error("expected non-empty orchestrator prompt")
	}

	// 测试构建任务提示词
	taskPrompt := BuildTaskPrompt("test goal", "test state", "")
	if taskPrompt == "" {
		t.Error("expected non-empty task prompt")
	}

	// 测试构建工具结果提示词
	toolPrompt := BuildToolResultPrompt("test_tool", "test result")
	if toolPrompt == "" {
		t.Error("expected non-empty tool result prompt")
	}
}

func TestAgentCallbacks(t *testing.T) {
	responses := []*model.ChatCompletionResponse{
		{
			Choices: []model.Choice{
				{
					Message: model.Message{
						Role:    model.RoleAssistant,
						Content: "任务已完成",
					},
					FinishReason: "stop",
				},
			},
		},
	}

	agent, _, _, cleanup := setupTestAgent(t, responses)
	defer cleanup()

	// 设置回调
	var streamChunks []string
	var iterations []int

	agent.SetOnStreamChunk(func(chunk string) {
		streamChunks = append(streamChunks, chunk)
	})

	agent.SetOnIteration(func(iteration int) {
		iterations = append(iterations, iteration)
	})

	// 运行Agent
	_, err := agent.Run(context.Background(), "test task")
	if err != nil {
		t.Fatalf("agent run failed: %v", err)
	}

	// 检查回调是否被调用
	if len(streamChunks) == 0 {
		t.Error("stream callback was not called")
	}

	if len(iterations) == 0 {
		t.Error("iteration callback was not called")
	}
}

func TestAgentConcurrency(t *testing.T) {
	responses := []*model.ChatCompletionResponse{
		{
			Choices: []model.Choice{
				{
					Message: model.Message{
						Role:    model.RoleAssistant,
						Content: "任务已完成",
					},
					FinishReason: "stop",
				},
			},
		},
	}

	agent, _, _, cleanup := setupTestAgent(t, responses)
	defer cleanup()

	// 尝试并发运行（应该失败）
	var wg sync.WaitGroup
	errors := make([]error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := agent.Run(context.Background(), "test task")
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// 至少有一个应该失败（因为Agent已经在运行）
	hasRunningError := false
	for _, err := range errors {
		if err != nil && err.Error() == "agent is already running" {
			hasRunningError = true
			break
		}
	}

	if !hasRunningError {
		t.Error("expected 'agent is already running' error for concurrent runs")
	}
}
