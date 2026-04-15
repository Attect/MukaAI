package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Attect/MukaAI/internal/model"
	"github.com/Attect/MukaAI/internal/state"
)

// TestNewCompressor 测试压缩器创建
func TestNewCompressor(t *testing.T) {
	// 创建测试用的模型客户端
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 4096,
	}
	client, err := model.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create model client: %v", err)
	}

	tests := []struct {
		name        string
		client      *model.Client
		config      *CompressorConfig
		wantErr     bool
		errContains string
	}{
		{
			name:    "默认配置",
			client:  client,
			config:  nil,
			wantErr: false,
		},
		{
			name:   "自定义配置",
			client: client,
			config: &CompressorConfig{
				TriggerThreshold:    0.7,
				MinMessagesToKeep:   5,
				MaxMessagesToKeep:   15,
				KeepRecentToolCalls: 2,
				SummaryMaxLength:    1000,
			},
			wantErr: false,
		},
		{
			name:        "空客户端",
			client:      nil,
			config:      nil,
			wantErr:     true,
			errContains: "model client cannot be nil",
		},
		{
			name:   "无效阈值",
			client: client,
			config: &CompressorConfig{
				TriggerThreshold: 1.5,
				SummaryMaxLength: 1000,
			},
			wantErr:     true,
			errContains: "trigger threshold",
		},
		{
			name:   "无效消息数量",
			client: client,
			config: &CompressorConfig{
				MinMessagesToKeep: 0,
				SummaryMaxLength:  1000,
			},
			wantErr:     true,
			errContains: "min messages to keep",
		},
		{
			name:   "最大消息小于最小消息",
			client: client,
			config: &CompressorConfig{
				MinMessagesToKeep: 20,
				MaxMessagesToKeep: 10,
				SummaryMaxLength:  1000,
			},
			wantErr:     true,
			errContains: "max messages to keep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressor, err := NewCompressor(tt.client, tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewCompressor() expected error, got nil")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("NewCompressor() error = %v, want contains %v", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("NewCompressor() unexpected error: %v", err)
				return
			}
			if compressor == nil {
				t.Error("NewCompressor() returned nil")
				return
			}
		})
	}
}

// TestCompressorConfig_Validate 测试配置验证
func TestCompressorConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  CompressorConfig
		wantErr bool
	}{
		{
			name: "有效配置",
			config: CompressorConfig{
				TriggerThreshold:    0.8,
				MinMessagesToKeep:   10,
				MaxMessagesToKeep:   20,
				KeepRecentToolCalls: 3,
				SummaryMaxLength:    2000,
			},
			wantErr: false,
		},
		{
			name: "阈值为0",
			config: CompressorConfig{
				TriggerThreshold:  0,
				MinMessagesToKeep: 10,
				MaxMessagesToKeep: 20,
				SummaryMaxLength:  1000,
			},
			wantErr: false,
		},
		{
			name: "阈值为1",
			config: CompressorConfig{
				TriggerThreshold:  1,
				MinMessagesToKeep: 10,
				MaxMessagesToKeep: 20,
				SummaryMaxLength:  1000,
			},
			wantErr: false,
		},
		{
			name: "阈值小于0",
			config: CompressorConfig{
				TriggerThreshold: -0.1,
			},
			wantErr: true,
		},
		{
			name: "阈值大于1",
			config: CompressorConfig{
				TriggerThreshold: 1.1,
			},
			wantErr: true,
		},
		{
			name: "最小消息数为0",
			config: CompressorConfig{
				MinMessagesToKeep: 0,
			},
			wantErr: true,
		},
		{
			name: "摘要长度过小",
			config: CompressorConfig{
				SummaryMaxLength: 50,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

// TestShouldCompress 测试压缩判断
func TestShouldCompress(t *testing.T) {
	// 创建测试客户端
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 1000, // 较小的上下文用于测试
	}
	client, _ := model.NewClient(config)

	compressorConfig := &CompressorConfig{
		TriggerThreshold:    0.8,
		MinMessagesToKeep:   5,
		MaxMessagesToKeep:   10,
		KeepRecentToolCalls: 2,
		SummaryMaxLength:    1000,
	}
	compressor, err := NewCompressor(client, compressorConfig)
	if err != nil {
		t.Fatalf("Failed to create compressor: %v", err)
	}

	tests := []struct {
		name         string
		messages     []model.Message
		wantCompress bool
		wantMinRatio float64
		wantMaxRatio float64
	}{
		{
			name:         "空消息",
			messages:     []model.Message{},
			wantCompress: false,
			wantMinRatio: 0,
			wantMaxRatio: 0,
		},
		{
			name: "少量消息",
			messages: []model.Message{
				model.NewSystemMessage("You are a helpful assistant."),
				model.NewUserMessage("Hello"),
				model.NewAssistantMessage("Hi! How can I help you?"),
			},
			wantCompress: false,
			wantMinRatio: 0,
			wantMaxRatio: 0.5,
		},
		{
			name:         "大量消息",
			messages:     generateLargeMessages(100),
			wantCompress: true,
			wantMinRatio: 0.8,
			wantMaxRatio: 200.0, // 可能超过100%，因为生成的消息很长
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldCompress, ratio := compressor.ShouldCompress(tt.messages)

			if shouldCompress != tt.wantCompress {
				t.Errorf("ShouldCompress() = %v, want %v", shouldCompress, tt.wantCompress)
			}

			if ratio < tt.wantMinRatio || ratio > tt.wantMaxRatio {
				t.Errorf("ShouldCompress() ratio = %v, want between %v and %v",
					ratio, tt.wantMinRatio, tt.wantMaxRatio)
			}
		})
	}
}

// TestCompress 测试压缩功能
func TestCompress(t *testing.T) {
	// 创建测试客户端
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 500, // 小上下文触发压缩
	}
	client, _ := model.NewClient(config)

	compressorConfig := &CompressorConfig{
		TriggerThreshold:             0.8,
		MinMessagesToKeep:            5,
		MaxMessagesToKeep:            10,
		KeepRecentToolCalls:          2,
		EnableProgressiveCompression: true,
		SummaryMaxLength:             1000,
	}
	compressor, _ := NewCompressor(client, compressorConfig)

	// 创建测试任务状态
	taskState := state.NewTaskState("test-task", "测试任务目标")
	taskState.AddCompletedStep("步骤1: 初始化")
	taskState.AddCompletedStep("步骤2: 分析")
	taskState.AddDecision("决定使用Go语言实现")

	tests := []struct {
		name           string
		messages       []model.Message
		taskState      *state.TaskState
		wantCompressed bool
		wantMinCount   int
		wantMaxCount   int
		wantSummary    bool
	}{
		{
			name:           "空消息不压缩",
			messages:       []model.Message{},
			taskState:      taskState,
			wantCompressed: false,
			wantMinCount:   0,
			wantMaxCount:   0,
			wantSummary:    false,
		},
		{
			name: "少量消息不压缩",
			messages: []model.Message{
				model.NewSystemMessage("You are a helpful assistant."),
				model.NewUserMessage("Hello"),
			},
			taskState:      taskState,
			wantCompressed: false,
			wantMinCount:   2,
			wantMaxCount:   2,
			wantSummary:    false,
		},
		{
			name:           "大量消息触发压缩",
			messages:       generateLargeMessages(50),
			taskState:      taskState,
			wantCompressed: true,
			wantMinCount:   5,
			wantMaxCount:   15,
			wantSummary:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compressor.Compress(tt.messages, tt.taskState)
			if err != nil {
				t.Errorf("Compress() error = %v", err)
				return
			}

			if result.WasCompressed != tt.wantCompressed {
				t.Errorf("Compress() WasCompressed = %v, want %v",
					result.WasCompressed, tt.wantCompressed)
			}

			if len(result.CompressedMessages) < tt.wantMinCount ||
				len(result.CompressedMessages) > tt.wantMaxCount {
				t.Errorf("Compress() message count = %v, want between %v and %v",
					len(result.CompressedMessages), tt.wantMinCount, tt.wantMaxCount)
			}

			if tt.wantSummary && result.Summary == "" {
				t.Error("Compress() expected summary, got empty")
			}

			if !tt.wantSummary && result.Summary != "" {
				t.Errorf("Compress() unexpected summary: %v", result.Summary)
			}

			// 验证压缩比率
			if result.WasCompressed {
				if result.CompressionRatio <= 0 || result.CompressionRatio > 1 {
					t.Errorf("Compress() invalid compression ratio: %v", result.CompressionRatio)
				}
				if result.OriginalTokens <= result.CompressedTokens {
					t.Errorf("Compress() should reduce tokens: original=%d, compressed=%d",
						result.OriginalTokens, result.CompressedTokens)
				}
			}
		})
	}
}

// TestExtractKeyInfo 测试关键信息提取
func TestExtractKeyInfo(t *testing.T) {
	// 创建测试客户端
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 4096,
	}
	client, _ := model.NewClient(config)
	compressor, _ := NewCompressor(client, nil)

	messages := []model.Message{
		model.NewSystemMessage("You are a helpful assistant."),
		model.NewUserMessage("请帮我实现一个功能"),
		model.NewAssistantMessage("我决定使用Go语言来实现这个功能。选择Gin框架作为Web框架。"),
		model.NewUserMessage("好的，开始实现吧"),
		createAssistantMessageWithToolCalls("",
			[]model.ToolCall{
				{
					ID:   "call-1",
					Type: "function",
					Function: model.FunctionCall{
						Name:      "write_file",
						Arguments: `{"path": "main.go", "content": "package main"}`,
					},
				},
			},
		),
		model.NewToolResultMessage("call-1", "write_file", "文件写入成功"),
	}

	keyInfo := compressor.ExtractKeyInfo(messages)

	if keyInfo == nil {
		t.Fatal("ExtractKeyInfo() returned nil")
	}

	// 验证决策提取
	if len(keyInfo.Decisions) == 0 {
		t.Error("ExtractKeyInfo() should extract decisions")
	}

	// 验证最近操作提取
	if len(keyInfo.RecentActions) == 0 {
		t.Error("ExtractKeyInfo() should extract recent actions")
	}

	// 验证工具历史提取
	if len(keyInfo.ToolHistory) == 0 {
		t.Error("ExtractKeyInfo() should extract tool history")
	}
}

// TestCompressor_UpdateConfig 测试配置更新
func TestCompressor_UpdateConfig(t *testing.T) {
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 4096,
	}
	client, _ := model.NewClient(config)
	compressor, _ := NewCompressor(client, nil)

	tests := []struct {
		name    string
		config  *CompressorConfig
		wantErr bool
	}{
		{
			name: "有效配置",
			config: &CompressorConfig{
				TriggerThreshold:    0.7,
				MinMessagesToKeep:   8,
				MaxMessagesToKeep:   15,
				KeepRecentToolCalls: 4,
				SummaryMaxLength:    1000,
			},
			wantErr: false,
		},
		{
			name:    "空配置",
			config:  nil,
			wantErr: true,
		},
		{
			name: "无效配置",
			config: &CompressorConfig{
				TriggerThreshold: -0.1,
				SummaryMaxLength: 1000,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := compressor.UpdateConfig(tt.config)
			if tt.wantErr && err == nil {
				t.Errorf("UpdateConfig() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("UpdateConfig() unexpected error: %v", err)
			}
		})
	}
}

// TestGetCompressionStats 测试压缩统计
func TestGetCompressionStats(t *testing.T) {
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 10000, // 增大上下文大小
	}
	client, _ := model.NewClient(config)
	compressor, _ := NewCompressor(client, nil)

	messages := []model.Message{
		model.NewSystemMessage("You are a helpful assistant."),
		model.NewUserMessage("Hello"),
		model.NewAssistantMessage("Hi!"),
	}
	stats := compressor.GetCompressionStats(messages)

	if stats == nil {
		t.Fatal("GetCompressionStats() returned nil")
	}

	if stats.MessageCount != len(messages) {
		t.Errorf("MessageCount = %v, want %v", stats.MessageCount, len(messages))
	}

	if stats.ContextSize != 10000 {
		t.Errorf("ContextSize = %v, want 10000", stats.ContextSize)
	}

	if stats.UsageRatio < 0 {
		t.Errorf("UsageRatio = %v, want >= 0", stats.UsageRatio)
	}
}

// TestCompressionWithToolCalls 测试包含工具调用的压缩
func TestCompressionWithToolCalls(t *testing.T) {
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 200, // 减小上下文大小以触发压缩
	}
	client, _ := model.NewClient(config)

	compressorConfig := &CompressorConfig{
		TriggerThreshold:    0.8,
		MinMessagesToKeep:   5,
		MaxMessagesToKeep:   10,
		KeepRecentToolCalls: 2,
		SummaryMaxLength:    1000,
	}
	compressor, _ := NewCompressor(client, compressorConfig)

	// 创建包含工具调用的消息
	messages := []model.Message{
		model.NewSystemMessage("You are a helpful assistant."),
		model.NewUserMessage("请帮我创建一个文件"),
		createAssistantMessageWithToolCalls("",
			[]model.ToolCall{
				{
					ID:   "call-1",
					Type: "function",
					Function: model.FunctionCall{
						Name:      "write_file",
						Arguments: `{"path": "test.txt", "content": "Hello"}`,
					},
				},
			},
		),
		model.NewToolResultMessage("call-1", "write_file", "文件创建成功"),
		model.NewUserMessage("现在读取这个文件"),
		createAssistantMessageWithToolCalls("",
			[]model.ToolCall{
				{
					ID:   "call-2",
					Type: "function",
					Function: model.FunctionCall{
						Name:      "read_file",
						Arguments: `{"path": "test.txt"}`,
					},
				},
			},
		),
		model.NewToolResultMessage("call-2", "read_file", "Hello"),
	}

	// 添加更多消息以触发压缩（增加消息长度）
	for i := 0; i < 30; i++ {
		messages = append(messages, model.NewUserMessage(fmt.Sprintf("这是一条较长的测试消息编号%d，用于触发压缩机制", i)))
		messages = append(messages, model.NewAssistantMessage(fmt.Sprintf("这是对测试消息%d的回复内容，同样较长以增加token数量", i)))
	}

	taskState := state.NewTaskState("test-task", "测试任务")
	result, err := compressor.Compress(messages, taskState)
	if err != nil {
		t.Errorf("Compress() error = %v", err)
		return
	}

	// 验证系统消息被保留
	hasSystemMessage := false
	for _, msg := range result.CompressedMessages {
		if msg.Role == model.RoleSystem {
			hasSystemMessage = true
			break
		}
	}
	if !hasSystemMessage {
		t.Error("Compress() should keep system messages")
	}

	// 验证压缩发生
	if !result.WasCompressed {
		t.Error("Compress() should have compressed the messages")
	}
}

// TestConcurrentCompression 测试并发压缩
func TestConcurrentCompression(t *testing.T) {
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 500,
	}
	client, _ := model.NewClient(config)
	compressor, _ := NewCompressor(client, nil)

	taskState := state.NewTaskState("test-task", "测试任务")

	// 并发执行压缩
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			messages := generateLargeMessages(30)
			result, err := compressor.Compress(messages, taskState)
			if err != nil {
				t.Errorf("Concurrent compress failed: %v", err)
			}
			if result == nil {
				t.Error("Concurrent compress returned nil result")
			}
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

// 辅助函数

// generateLargeMessages 生成大量测试消息
func generateLargeMessages(count int) []model.Message {
	messages := []model.Message{
		model.NewSystemMessage("You are a helpful assistant."),
	}

	for i := 0; i < count; i++ {
		// 生成较长的内容以增加token数量
		content := ""
		for j := 0; j < 50; j++ {
			content += "这是一段测试内容，用于增加消息的长度和token数量。"
		}
		messages = append(messages, model.NewUserMessage(content))
		messages = append(messages, model.NewAssistantMessage(content))
	}

	return messages
}

// containsString 检查字符串是否包含子串
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// createAssistantMessageWithToolCalls 创建带工具调用的助手消息（测试辅助函数）
func createAssistantMessageWithToolCalls(content string, toolCalls []model.ToolCall) model.Message {
	return model.Message{
		Role:      model.RoleAssistant,
		Content:   content,
		ToolCalls: toolCalls,
	}
}

// === C1: 中英文关键词测试 ===

func TestExtractKeyDecisions_EnglishKeywords(t *testing.T) {
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 4096,
	}
	client, _ := model.NewClient(config)
	compressor, _ := NewCompressor(client, nil)

	tests := []struct {
		name      string
		message   string
		wantMatch bool
	}{
		{"中文决定", "我决定使用Go语言实现", true},
		{"中文选择", "选择Gin框架作为Web框架", true},
		{"中文方案", "这是一个好的方案", true},
		{"英文decision", "I made the decision to use REST API", true},
		{"英文decided", "We decided to implement caching", true},
		{"英文will use", "I will use PostgreSQL for storage", true},
		{"英文approach", "The approach we take is microservices", true},
		{"英文fix", "I will fix the authentication bug", true},
		{"英文fixed", "The issue has been fixed in the latest commit", true},
		{"英文implement", "I plan to implement rate limiting", true},
		{"英文resolved", "The conflict has been resolved", true},
		{"无关键词", "This is a normal message", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := []model.Message{
				model.NewAssistantMessage(tt.message),
			}
			decisions := compressor.ExtractKeyInfo(messages).Decisions
			if tt.wantMatch && len(decisions) == 0 {
				t.Errorf("Expected to extract decision from %q, got none", tt.message)
			}
			if !tt.wantMatch && len(decisions) > 0 {
				t.Errorf("Did not expect decision from %q, got: %v", tt.message, decisions)
			}
		})
	}
}

// === C2: LLM摘要测试（使用mock） ===

// mockLLMSummaryFunc 创建mock LLM摘要函数
func mockLLMSummaryFunc(returnValue string, callCount *int) LLMSummaryFunc {
	return func(ctx context.Context, messages []model.Message, prompt string) string {
		if callCount != nil {
			*callCount++
		}
		return returnValue
	}
}

// mockLLMSummaryFuncWithError 创建mock LLM摘要函数（模拟失败）
func mockLLMSummaryFuncWithError() LLMSummaryFunc {
	return func(ctx context.Context, messages []model.Message, prompt string) string {
		return "" // 模拟LLM失败
	}
}

// mockLLMSummaryFuncWithDelay 创建mock LLM摘要函数（模拟超时）
func mockLLMSummaryFuncWithDelay() LLMSummaryFunc {
	return func(ctx context.Context, messages []model.Message, prompt string) string {
		select {
		case <-ctx.Done():
			return "" // 超时返回空
		case <-time.After(30 * time.Second):
			return "delayed summary"
		}
	}
}

func TestCompressor_LLMSummarySuccess(t *testing.T) {
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 500,
	}
	client, _ := model.NewClient(config)

	compressorConfig := &CompressorConfig{
		TriggerThreshold:             0.8,
		MinMessagesToKeep:            5,
		MaxMessagesToKeep:            10,
		KeepRecentToolCalls:          2,
		SummaryMaxLength:             1000,
		LLMSummaryTimeout:            5,
		EnableProgressiveCompression: true,
	}

	callCount := 0
	mockSummary := "LLM摘要: 已完成用户登录功能，当前正在进行订单模块开发。"
	compressor, _ := NewCompressor(client, compressorConfig, mockLLMSummaryFunc(mockSummary, &callCount))

	taskState := state.NewTaskState("test-task", "测试任务")
	result, err := compressor.Compress(generateLargeMessages(50), taskState)
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	if !result.WasCompressed {
		t.Error("Expected compression to occur")
	}

	// 验证LLM被调用了
	if callCount == 0 {
		t.Error("Expected LLM summary to be called")
	}

	// 验证摘要包含LLM生成的内容
	if result.Summary != mockSummary {
		t.Errorf("Summary = %q, want %q", result.Summary, mockSummary)
	}
}

func TestCompressor_LLMSummaryFallback(t *testing.T) {
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 500,
	}
	client, _ := model.NewClient(config)

	compressorConfig := &CompressorConfig{
		TriggerThreshold:             0.8,
		MinMessagesToKeep:            5,
		MaxMessagesToKeep:            10,
		KeepRecentToolCalls:          2,
		SummaryMaxLength:             1000,
		LLMSummaryTimeout:            5,
		EnableProgressiveCompression: true,
	}

	// 使用会失败的mock
	compressor, _ := NewCompressor(client, compressorConfig, mockLLMSummaryFuncWithError())

	taskState := state.NewTaskState("test-task", "测试任务")
	taskState.AddDecision("决定使用Go语言实现")
	result, err := compressor.Compress(generateLargeMessages(50), taskState)
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	if !result.WasCompressed {
		t.Error("Expected compression to occur")
	}

	// 验证回退到规则提取（摘要不为空）
	if result.Summary == "" {
		t.Error("Expected fallback to rule-based summary, got empty summary")
	}
}

func TestCompressor_LLMSummaryTimeout(t *testing.T) {
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 500,
	}
	client, _ := model.NewClient(config)

	compressorConfig := &CompressorConfig{
		TriggerThreshold:             0.8,
		MinMessagesToKeep:            5,
		MaxMessagesToKeep:            10,
		KeepRecentToolCalls:          2,
		SummaryMaxLength:             1000,
		LLMSummaryTimeout:            1, // 1秒超时
		EnableProgressiveCompression: true,
	}

	// 使用会超时的mock
	compressor, _ := NewCompressor(client, compressorConfig, mockLLMSummaryFuncWithDelay())

	taskState := state.NewTaskState("test-task", "测试任务")
	taskState.AddDecision("决定使用Go语言实现")

	start := time.Now()
	result, err := compressor.Compress(generateLargeMessages(50), taskState)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	// 超时应该快速返回，不应该等30秒
	if elapsed > 5*time.Second {
		t.Errorf("Compress took too long: %v, expected < 5s with 1s timeout", elapsed)
	}

	// 应该回退到规则提取
	if result.Summary == "" {
		t.Error("Expected fallback to rule-based summary after timeout")
	}
}

func TestCompressor_NoLLMStillWorks(t *testing.T) {
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 500,
	}
	client, _ := model.NewClient(config)

	compressorConfig := &CompressorConfig{
		TriggerThreshold:             0.8,
		MinMessagesToKeep:            5,
		MaxMessagesToKeep:            10,
		KeepRecentToolCalls:          2,
		SummaryMaxLength:             1000,
		EnableProgressiveCompression: true,
	}

	// 不传入LLM函数
	compressor, _ := NewCompressor(client, compressorConfig)

	taskState := state.NewTaskState("test-task", "测试任务")
	taskState.AddDecision("决定使用Go语言实现")
	result, err := compressor.Compress(generateLargeMessages(50), taskState)
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	if !result.WasCompressed {
		t.Error("Expected compression to occur")
	}

	// 规则提取应该仍然工作
	if result.Summary == "" {
		t.Error("Expected rule-based summary when no LLM function provided")
	}
}

// === C3: 工具历史保留最近N次测试 ===

func TestExtractToolHistory_KeepRecentN(t *testing.T) {
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 4096,
	}
	client, _ := model.NewClient(config)
	compressor, _ := NewCompressor(client, nil)

	// 创建多个相同工具的调用
	messages := []model.Message{
		model.NewSystemMessage("You are a helpful assistant."),
		// 第一次 read_file 调用
		createAssistantMessageWithToolCalls("", []model.ToolCall{
			{ID: "call-1", Type: "function", Function: model.FunctionCall{Name: "read_file", Arguments: `{"path": "a.go"}`}},
		}),
		model.NewToolResultMessage("call-1", "read_file", "content of a.go"),
		// 第二次 read_file 调用
		createAssistantMessageWithToolCalls("", []model.ToolCall{
			{ID: "call-2", Type: "function", Function: model.FunctionCall{Name: "read_file", Arguments: `{"path": "b.go"}`}},
		}),
		model.NewToolResultMessage("call-2", "read_file", "content of b.go"),
		// 第三次 read_file 调用
		createAssistantMessageWithToolCalls("", []model.ToolCall{
			{ID: "call-3", Type: "function", Function: model.FunctionCall{Name: "read_file", Arguments: `{"path": "c.go"}`}},
		}),
		model.NewToolResultMessage("call-3", "read_file", "content of c.go"),
		// write_file 调用
		createAssistantMessageWithToolCalls("", []model.ToolCall{
			{ID: "call-4", Type: "function", Function: model.FunctionCall{Name: "write_file", Arguments: `{"path": "d.go"}`}},
		}),
		model.NewToolResultMessage("call-4", "write_file", "file written"),
	}

	keyInfo := compressor.ExtractKeyInfo(messages)

	// 应该保留所有工具调用（read_file 3次 + write_file 1次 = 4条）
	if len(keyInfo.ToolHistory) != 4 {
		t.Errorf("Expected 4 tool history entries, got %d: %v", len(keyInfo.ToolHistory), keyInfo.ToolHistory)
	}

	// 验证包含调用次数标记
	for _, entry := range keyInfo.ToolHistory {
		if !containsString(entry, "次调用") {
			t.Errorf("Tool history entry should contain call count: %q", entry)
		}
	}

	// 验证read_file有3次调用
	readFileCount := 0
	for _, entry := range keyInfo.ToolHistory {
		if containsString(entry, "read_file") {
			readFileCount++
		}
	}
	if readFileCount != 3 {
		t.Errorf("Expected 3 read_file entries, got %d", readFileCount)
	}
}

func TestExtractToolHistory_Max20Limit(t *testing.T) {
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 4096,
	}
	client, _ := model.NewClient(config)
	compressor, _ := NewCompressor(client, nil)

	// 创建25次工具调用（超过20条限制）
	var messages []model.Message
	for i := 0; i < 25; i++ {
		toolCallID := fmt.Sprintf("call-%d", i)
		messages = append(messages,
			createAssistantMessageWithToolCalls("", []model.ToolCall{
				{ID: toolCallID, Type: "function", Function: model.FunctionCall{
					Name:      fmt.Sprintf("tool_%d", i%5),
					Arguments: fmt.Sprintf(`{"arg": "value_%d"}`, i),
				}},
			}),
			model.NewToolResultMessage(toolCallID, fmt.Sprintf("tool_%d", i%5), fmt.Sprintf("result_%d", i)),
		)
	}

	keyInfo := compressor.ExtractKeyInfo(messages)

	// 应该最多保留20条
	if len(keyInfo.ToolHistory) > 20 {
		t.Errorf("Expected at most 20 tool history entries, got %d", len(keyInfo.ToolHistory))
	}
}

// === C4: 消息顺序测试 ===

func TestMergeMessagesInOrder(t *testing.T) {
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 4096,
	}
	client, _ := model.NewClient(config)
	compressor, _ := NewCompressor(client, nil)

	base := []model.Message{
		model.NewSystemMessage("system"),
	}

	recent := []model.Message{
		model.NewUserMessage("user msg 1"),
		model.NewAssistantMessage("assistant msg 1"),
	}

	toolMessages := []model.Message{
		createAssistantMessageWithToolCalls("", []model.ToolCall{
			{ID: "call-1", Type: "function", Function: model.FunctionCall{Name: "read_file", Arguments: `{}`}},
		}),
		model.NewToolResultMessage("call-1", "read_file", "result"),
	}

	result := compressor.mergeMessagesInOrder(base, recent, toolMessages)

	// 应该有: system + user1 + assistant1 + tool_call + tool_result = 5
	if len(result) != 5 {
		t.Errorf("Expected 5 messages, got %d", len(result))
	}

	// 第一个应该是system消息
	if result[0].Role != model.RoleSystem {
		t.Errorf("First message should be system, got %s", result[0].Role)
	}
}

// === C5: 配置可调测试 ===

func TestConfigCompressorConfig(t *testing.T) {
	// 测试config包的CompressorConfig
	cfg := DefaultCompressorConfig()
	if cfg.TriggerThreshold != 0.8 {
		t.Errorf("Default TriggerThreshold = %f, want 0.8", cfg.TriggerThreshold)
	}
	if cfg.MinMessagesToKeep != 10 {
		t.Errorf("Default MinMessagesToKeep = %d, want 10", cfg.MinMessagesToKeep)
	}
	if cfg.MaxMessagesToKeep != 20 {
		t.Errorf("Default MaxMessagesToKeep = %d, want 20", cfg.MaxMessagesToKeep)
	}
	if cfg.KeepRecentToolCalls != 3 {
		t.Errorf("Default KeepRecentToolCalls = %d, want 3", cfg.KeepRecentToolCalls)
	}
	if cfg.SummaryMaxLength != 2000 {
		t.Errorf("Default SummaryMaxLength = %d, want 2000", cfg.SummaryMaxLength)
	}
	if cfg.LLMSummaryTimeout != 10 {
		t.Errorf("Default LLMSummaryTimeout = %d, want 10", cfg.LLMSummaryTimeout)
	}
}

// === SetLLMSummarize 测试 ===

func TestSetLLMSummarize(t *testing.T) {
	config := &model.Config{
		Endpoint:    "http://localhost:8080/v1/",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 4096,
	}
	client, _ := model.NewClient(config)
	compressor, _ := NewCompressor(client, nil)

	// 初始时没有LLM函数
	ctx := context.Background()
	result := compressor.summarizeWithLLM(ctx, nil, "test")
	if result != "" {
		t.Error("Expected empty result when no LLM function set")
	}

	// 设置LLM函数
	callCount := 0
	compressor.SetLLMSummarize(mockLLMSummaryFunc("mock summary", &callCount))

	result = compressor.summarizeWithLLM(ctx, nil, "test")
	if result != "mock summary" {
		t.Errorf("Expected 'mock summary', got %q", result)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}
