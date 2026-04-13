package agent

import (
	"fmt"
	"testing"

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
