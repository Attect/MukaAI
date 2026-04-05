package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "有效配置",
			config: &Config{
				Endpoint:    "http://localhost:11453/v1/",
				ModelName:   "test-model",
				ContextSize: 200000,
			},
			wantErr: false,
		},
		{
			name: "缺少endpoint",
			config: &Config{
				ModelName:   "test-model",
				ContextSize: 200000,
			},
			wantErr: true,
		},
		{
			name: "缺少model_name",
			config: &Config{
				Endpoint:    "http://localhost:11453/v1/",
				ContextSize: 200000,
			},
			wantErr: true,
		},
		{
			name: "无效context_size",
			config: &Config{
				Endpoint:    "http://localhost:11453/v1/",
				ModelName:   "test-model",
				ContextSize: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Endpoint == "" {
		t.Error("默认endpoint不应为空")
	}
	if config.ModelName == "" {
		t.Error("默认model_name不应为空")
	}
	if config.ContextSize <= 0 {
		t.Error("默认context_size应大于0")
	}
}

// TestMessageCreation 测试消息创建
func TestMessageCreation(t *testing.T) {
	// 测试系统消息
	sysMsg := NewSystemMessage("你是一个助手")
	if sysMsg.Role != RoleSystem {
		t.Errorf("系统消息角色错误: %v", sysMsg.Role)
	}

	// 测试用户消息
	userMsg := NewUserMessage("你好")
	if userMsg.Role != RoleUser {
		t.Errorf("用户消息角色错误: %v", userMsg.Role)
	}

	// 测试助手消息
	assistantMsg := NewAssistantMessage("你好！有什么可以帮你的？")
	if assistantMsg.Role != RoleAssistant {
		t.Errorf("助手消息角色错误: %v", assistantMsg.Role)
	}

	// 测试工具结果消息
	toolMsg := NewToolResultMessage("call-123", "read_file", "文件内容")
	if toolMsg.Role != RoleTool {
		t.Errorf("工具消息角色错误: %v", toolMsg.Role)
	}
	if toolMsg.ToolCallID != "call-123" {
		t.Errorf("工具调用ID错误: %v", toolMsg.ToolCallID)
	}
}

// TestToolCallArguments 测试工具调用参数解析
func TestToolCallArguments(t *testing.T) {
	tc := ToolCall{
		ID:   "call-123",
		Type: "function",
		Function: FunctionCall{
			Name:      "read_file",
			Arguments: `{"path": "/tmp/test.txt"}`,
		},
	}

	args, err := tc.ParseToolCallArguments()
	if err != nil {
		t.Fatalf("解析工具调用参数失败: %v", err)
	}

	if path, ok := args["path"].(string); !ok || path != "/tmp/test.txt" {
		t.Errorf("参数解析错误: %v", args)
	}
}

// TestChatCompletion 测试聊天补全
func TestChatCompletion(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法
		if r.Method != "POST" {
			t.Errorf("预期POST请求，得到: %s", r.Method)
		}

		// 验证请求路径
		if !strings.HasSuffix(r.URL.Path, "chat/completions") {
			t.Errorf("错误的请求路径: %s", r.URL.Path)
		}

		// 验证Authorization头
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("错误的Authorization头: %s", auth)
		}

		// 返回模拟响应
		resp := ChatCompletionResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "test-model",
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role:    RoleAssistant,
						Content: "这是一个测试响应",
					},
					FinishReason: "stop",
				},
			},
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建客户端
	config := &Config{
		Endpoint:    server.URL + "/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 200000,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	// 发送请求
	messages := []Message{
		NewUserMessage("你好"),
	}

	resp, err := client.ChatCompletion(context.Background(), messages, nil)
	if err != nil {
		t.Fatalf("聊天补全失败: %v", err)
	}

	// 验证响应
	if len(resp.Choices) == 0 {
		t.Fatal("响应中没有选项")
	}

	if resp.Choices[0].Message.Content != "这是一个测试响应" {
		t.Errorf("响应内容错误: %s", resp.Choices[0].Message.Content)
	}
}

// TestStreamChatCompletion 测试流式聊天补全
func TestStreamChatCompletion(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求
		if r.Method != "POST" {
			t.Errorf("预期POST请求，得到: %s", r.Method)
		}

		// 设置SSE响应头
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// 发送流式数据
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("不支持流式响应")
		}

		// 模拟流式响应
		chunks := []string{
			`{"id":"test-id","object":"chat.completion.chunk","created":1234567890,"model":"test-model","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":""}]}`,
			`{"id":"test-id","object":"chat.completion.chunk","created":1234567890,"model":"test-model","choices":[{"index":0,"delta":{"content":"你好"},"finish_reason":""}]}`,
			`{"id":"test-id","object":"chat.completion.chunk","created":1234567890,"model":"test-model","choices":[{"index":0,"delta":{"content":"！"},"finish_reason":""}]}`,
			`[DONE]`,
		}

		for _, chunk := range chunks {
			_, err := w.Write([]byte("data: " + chunk + "\n\n"))
			if err != nil {
				t.Errorf("写入流式数据失败: %v", err)
				return
			}
			flusher.Flush()
		}
	}))
	defer server.Close()

	// 创建客户端
	config := &Config{
		Endpoint:    server.URL + "/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 200000,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	// 发送流式请求
	messages := []Message{
		NewUserMessage("你好"),
	}

	eventChan, err := client.StreamChatCompletion(context.Background(), messages, nil)
	if err != nil {
		t.Fatalf("流式聊天补全失败: %v", err)
	}

	// 收集响应
	var content strings.Builder
	eventCount := 0

	for event := range eventChan {
		if event.Error != nil {
			t.Errorf("流式事件错误: %v", event.Error)
			continue
		}

		if event.Done {
			break
		}

		if event.Response != nil && len(event.Response.Choices) > 0 {
			if event.Response.Choices[0].Delta != nil {
				content.WriteString(event.Response.Choices[0].Delta.Content)
			}
			eventCount++
		}
	}

	// 验证接收到的内容
	if content.String() != "你好！" {
		t.Errorf("流式响应内容错误: %s", content.String())
	}

	if eventCount < 2 {
		t.Errorf("接收到的事件数量太少: %d", eventCount)
	}
}

// TestThinkingTags 测试思考标签处理
func TestThinkingTags(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatCompletionResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "test-model",
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role:    RoleAssistant,
						Content: "<thinking>这是思考过程</thinking>这是实际回复",
					},
					FinishReason: "stop",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建客户端
	config := &Config{
		Endpoint:    server.URL + "/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 200000,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	// 发送请求
	messages := []Message{
		NewUserMessage("测试"),
	}

	resp, err := client.ChatCompletion(context.Background(), messages, nil)
	if err != nil {
		t.Fatalf("聊天补全失败: %v", err)
	}

	// 验证思考标签已被移除
	expected := "这是思考过程这是实际回复"
	if resp.Choices[0].Message.Content != expected {
		t.Errorf("思考标签处理错误，预期: %s, 得到: %s", expected, resp.Choices[0].Message.Content)
	}
}

// TestToolCalling 测试工具调用
func TestToolCalling(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 解析请求
		var req ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("解析请求失败: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 验证工具定义
		if len(req.Tools) == 0 {
			t.Error("请求中应包含工具定义")
		}

		// 返回工具调用响应
		resp := ChatCompletionResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "test-model",
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role: RoleAssistant,
						ToolCalls: []ToolCall{
							{
								ID:   "call-123",
								Type: "function",
								Function: FunctionCall{
									Name:      "read_file",
									Arguments: `{"path":"/tmp/test.txt"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建客户端
	config := &Config{
		Endpoint:    server.URL + "/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 200000,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	// 定义工具
	tools := []Tool{
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "read_file",
				Description: "读取文件内容",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "文件路径",
						},
					},
					"required": []string{"path"},
				},
			},
		},
	}

	// 发送请求
	messages := []Message{
		NewUserMessage("读取/tmp/test.txt文件"),
	}

	resp, err := client.ChatCompletion(context.Background(), messages, tools)
	if err != nil {
		t.Fatalf("聊天补全失败: %v", err)
	}

	// 验证工具调用
	if len(resp.Choices[0].Message.ToolCalls) == 0 {
		t.Fatal("响应中应包含工具调用")
	}

	tc := resp.Choices[0].Message.ToolCalls[0]
	if tc.Function.Name != "read_file" {
		t.Errorf("工具名称错误: %s", tc.Function.Name)
	}

	// 解析工具参数
	args, err := tc.ParseToolCallArguments()
	if err != nil {
		t.Fatalf("解析工具参数失败: %v", err)
	}

	if path, ok := args["path"].(string); !ok || path != "/tmp/test.txt" {
		t.Errorf("工具参数错误: %v", args)
	}
}

// TestCountTokens 测试token计数
func TestCountTokens(t *testing.T) {
	config := DefaultConfig()
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	messages := []Message{
		NewUserMessage("这是一个测试消息"),
		NewAssistantMessage("收到，这是回复"),
	}

	tokens := client.CountTokens(messages)
	if tokens <= 0 {
		t.Error("token计数应大于0")
	}

	// 粗略估算：中文约2个字符/token，英文约4个字符/token
	// 总字符数约20，预期约5-15个token（考虑到估算的粗略性）
	if tokens > 20 {
		t.Errorf("token计数可能过高: %d", tokens)
	}
}

// TestIsContextOverflow 测试上下文溢出检测
func TestIsContextOverflow(t *testing.T) {
	config := &Config{
		Endpoint:    "http://localhost:11453/v1/",
		ModelName:   "test-model",
		ContextSize: 100, // 设置较小的上下文大小用于测试
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	// 创建大量消息
	messages := []Message{}
	for i := 0; i < 100; i++ {
		messages = append(messages, NewUserMessage("这是一条测试消息，用于测试上下文溢出检测功能"))
	}

	if !client.IsContextOverflow(messages) {
		t.Error("应检测到上下文溢出")
	}
}
