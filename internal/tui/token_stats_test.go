// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"testing"
	"time"
)

// TestMessage_TokenUsage 测试消息的 token 用量字段
func TestMessage_TokenUsage(t *testing.T) {
	msg := Message{
		Role:       MessageRoleAssistant,
		Content:    "Hello, world!",
		TokenUsage: 100,
		Timestamp:  time.Now(),
	}

	if msg.TokenUsage != 100 {
		t.Errorf("Expected TokenUsage 100, got %d", msg.TokenUsage)
	}
}

// TestConversation_TokenUsage 测试对话的 token 用量字段
func TestConversation_TokenUsage(t *testing.T) {
	conv := Conversation{
		ID:         "test-conv-1",
		CreatedAt:  time.Now(),
		Status:     ConvStatusActive,
		TokenUsage: 500,
	}

	if conv.TokenUsage != 500 {
		t.Errorf("Expected TokenUsage 500, got %d", conv.TokenUsage)
	}
}

// TestAppModel_TokenStatistics 测试 AppModel 的 token 统计功能
func TestAppModel_TokenStatistics(t *testing.T) {
	model := NewAppModel()

	// 初始值应该为 0
	if model.totalTokens != 0 {
		t.Errorf("Expected initial totalTokens 0, got %d", model.totalTokens)
	}

	if model.inferenceCount != 0 {
		t.Errorf("Expected initial inferenceCount 0, got %d", model.inferenceCount)
	}
}

// TestAppModel_HandleStreamComplete 测试处理流式完成消息
func TestAppModel_HandleStreamComplete(t *testing.T) {
	model := NewAppModel()

	// 创建一个测试对话
	conv := &Conversation{
		ID:        "test-conv-1",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages:  make([]Message, 0),
	}
	model.activeConv = conv

	// 创建一个当前消息
	conv.currentMessage = &Message{
		Role:      MessageRoleAssistant,
		Content:   "Test response",
		Timestamp: time.Now(),
	}

	// 模拟流式完成
	usage := 150
	model.handleStreamComplete(usage)

	// 验证统计更新
	if model.totalTokens != usage {
		t.Errorf("Expected totalTokens %d, got %d", usage, model.totalTokens)
	}

	if model.inferenceCount != 1 {
		t.Errorf("Expected inferenceCount 1, got %d", model.inferenceCount)
	}

	if conv.TokenUsage != usage {
		t.Errorf("Expected conv.TokenUsage %d, got %d", usage, conv.TokenUsage)
	}

	if conv.currentMessage.TokenUsage != usage {
		t.Errorf("Expected currentMessage.TokenUsage %d, got %d", usage, conv.currentMessage.TokenUsage)
	}

	// 验证状态栏更新
	if model.statusBar.TotalTokens != usage {
		t.Errorf("Expected statusBar.TotalTokens %d, got %d", usage, model.statusBar.TotalTokens)
	}

	if model.statusBar.InferenceCount != 1 {
		t.Errorf("Expected statusBar.InferenceCount 1, got %d", model.statusBar.InferenceCount)
	}
}

// TestAppModel_MultipleInferences 测试多次推理的 token 累计
func TestAppModel_MultipleInferences(t *testing.T) {
	model := NewAppModel()

	// 创建一个测试对话
	conv := &Conversation{
		ID:        "test-conv-1",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages:  make([]Message, 0),
	}
	model.activeConv = conv

	// 模拟三次推理
	usages := []int{100, 150, 200}
	expectedTotal := 0

	for i, usage := range usages {
		// 创建新的当前消息
		conv.currentMessage = &Message{
			Role:      MessageRoleAssistant,
			Content:   "Test response",
			Timestamp: time.Now(),
		}

		model.handleStreamComplete(usage)
		expectedTotal += usage

		// 验证累计值
		if model.totalTokens != expectedTotal {
			t.Errorf("After inference %d: expected totalTokens %d, got %d", i+1, expectedTotal, model.totalTokens)
		}

		if model.inferenceCount != i+1 {
			t.Errorf("After inference %d: expected inferenceCount %d, got %d", i+1, i+1, model.inferenceCount)
		}
	}

	// 最终验证
	if model.totalTokens != 450 {
		t.Errorf("Expected final totalTokens 450, got %d", model.totalTokens)
	}

	if model.inferenceCount != 3 {
		t.Errorf("Expected final inferenceCount 3, got %d", model.inferenceCount)
	}
}

// TestAppModel_MultipleConversations 测试多个对话的 token 统计
func TestAppModel_MultipleConversations(t *testing.T) {
	model := NewAppModel()

	// 创建第一个对话
	conv1 := &Conversation{
		ID:        "conv-1",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages:  make([]Message, 0),
	}

	// 创建第二个对话
	conv2 := &Conversation{
		ID:        "conv-2",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages:  make([]Message, 0),
	}

	model.conversations = []*Conversation{conv1, conv2}

	// 在第一个对话中进行推理
	model.activeConv = conv1
	conv1.currentMessage = &Message{
		Role:      MessageRoleAssistant,
		Content:   "Response 1",
		Timestamp: time.Now(),
	}
	model.handleStreamComplete(100)

	// 切换到第二个对话
	model.activeConv = conv2
	conv2.currentMessage = &Message{
		Role:      MessageRoleAssistant,
		Content:   "Response 2",
		Timestamp: time.Now(),
	}
	model.handleStreamComplete(200)

	// 验证全局统计
	if model.totalTokens != 300 {
		t.Errorf("Expected totalTokens 300, got %d", model.totalTokens)
	}

	if model.inferenceCount != 2 {
		t.Errorf("Expected inferenceCount 2, got %d", model.inferenceCount)
	}

	// 验证各对话的统计
	if conv1.TokenUsage != 100 {
		t.Errorf("Expected conv1.TokenUsage 100, got %d", conv1.TokenUsage)
	}

	if conv2.TokenUsage != 200 {
		t.Errorf("Expected conv2.TokenUsage 200, got %d", conv2.TokenUsage)
	}
}

// TestStreamCompleteMsg 测试流式完成消息
func TestStreamCompleteMsg(t *testing.T) {
	msg := NewStreamCompleteMsg(250)

	if msg.Usage != 250 {
		t.Errorf("Expected Usage 250, got %d", msg.Usage)
	}
}

// TestTokenUsageUpdatedMsg 测试 token 用量更新消息
func TestTokenUsageUpdatedMsg(t *testing.T) {
	msg := NewTokenUsageUpdatedMsg(1000, 100)

	if msg.TotalTokens != 1000 {
		t.Errorf("Expected TotalTokens 1000, got %d", msg.TotalTokens)
	}

	if msg.Delta != 100 {
		t.Errorf("Expected Delta 100, got %d", msg.Delta)
	}
}

// TestInferenceCountUpdatedMsg 测试推理次数更新消息
func TestInferenceCountUpdatedMsg(t *testing.T) {
	msg := NewInferenceCountUpdatedMsg(5)

	if msg.Count != 5 {
		t.Errorf("Expected Count 5, got %d", msg.Count)
	}
}

// TestMessageBuilder_SetTokenUsage 测试消息构建器的 token 用量设置
func TestMessageBuilder_SetTokenUsage(t *testing.T) {
	builder := NewMessageBuilder(MessageRoleAssistant)
	msg := builder.
		SetContent("Test content").
		SetTokenUsage(300).
		Build()

	if msg.TokenUsage != 300 {
		t.Errorf("Expected TokenUsage 300, got %d", msg.TokenUsage)
	}
}

// TestConversationBuilder_SetTokenUsage 测试对话构建器的 token 用量设置
func TestConversationBuilder_SetTokenUsage(t *testing.T) {
	builder := NewConversationBuilder("test-conv")
	conv := builder.
		SetTitle("Test Conversation").
		SetTokenUsage(500).
		Build()

	if conv.TokenUsage != 500 {
		t.Errorf("Expected TokenUsage 500, got %d", conv.TokenUsage)
	}
}

// TestAppModel_StatusBarUpdate 测试状态栏的 token 统计更新
func TestAppModel_StatusBarUpdate(t *testing.T) {
	model := NewAppModel()

	// 初始值应该为 0
	if model.totalTokens != 0 {
		t.Errorf("Expected initial totalTokens 0, got %d", model.totalTokens)
	}

	if model.inferenceCount != 0 {
		t.Errorf("Expected initial inferenceCount 0, got %d", model.inferenceCount)
	}

	// 创建测试对话
	conv := &Conversation{
		ID:        "test-conv",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages:  make([]Message, 0),
	}
	model.activeConv = conv
	conv.currentMessage = &Message{
		Role:      MessageRoleAssistant,
		Content:   "Test",
		Timestamp: time.Now(),
	}

	// 触发流式完成
	model.handleStreamComplete(50)

	// 验证状态栏更新
	if model.totalTokens != 50 {
		t.Errorf("Expected totalTokens 50, got %d", model.totalTokens)
	}

	if model.inferenceCount != 1 {
		t.Errorf("Expected inferenceCount 1, got %d", model.inferenceCount)
	}
}

// TestAppModel_RenderConversation 测试对话渲染中的 token 显示
func TestAppModel_RenderConversation(t *testing.T) {
	model := NewAppModel()

	// 创建测试对话
	conv := &Conversation{
		ID:        "test-conv",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages: []Message{
			{
				Role:      MessageRoleUser,
				Content:   "Hello",
				Timestamp: time.Now(),
			},
			{
				Role:       MessageRoleAssistant,
				Content:    "Hi there!",
				TokenUsage: 50,
				Timestamp:  time.Now(),
			},
		},
	}

	// 渲染对话
	rendered := model.renderConversation(conv)

	// 验证渲染结果不为空
	if rendered == "" {
		t.Error("renderConversation returned empty string")
	}

	// 注意：由于 lipgloss 样式渲染包含 ANSI 转义码，
	// 我们只验证渲染结果包含基本内容
	// 实际的 token 显示测试在 ChatView 组件测试中
}

// TestTokenUsage_ZeroValue 测试 token 用量为 0 的情况
func TestTokenUsage_ZeroValue(t *testing.T) {
	model := NewAppModel()

	conv := &Conversation{
		ID:        "test-conv",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages:  make([]Message, 0),
	}
	model.activeConv = conv
	conv.currentMessage = &Message{
		Role:      MessageRoleAssistant,
		Content:   "",
		Timestamp: time.Now(),
	}

	// 触发流式完成，usage 为 0
	model.handleStreamComplete(0)

	// 验证统计更新（即使是 0 也应该更新）
	if model.totalTokens != 0 {
		t.Errorf("Expected totalTokens 0, got %d", model.totalTokens)
	}

	if model.inferenceCount != 1 {
		t.Errorf("Expected inferenceCount 1, got %d", model.inferenceCount)
	}
}

// TestTokenUsage_LargeValue 测试大数值 token 用量
func TestTokenUsage_LargeValue(t *testing.T) {
	model := NewAppModel()

	conv := &Conversation{
		ID:        "test-conv",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages:  make([]Message, 0),
	}
	model.activeConv = conv
	conv.currentMessage = &Message{
		Role:      MessageRoleAssistant,
		Content:   "Large response",
		Timestamp: time.Now(),
	}

	// 测试大数值
	largeUsage := 999999
	model.handleStreamComplete(largeUsage)

	if model.totalTokens != largeUsage {
		t.Errorf("Expected totalTokens %d, got %d", largeUsage, model.totalTokens)
	}

	if model.statusBar.TotalTokens != largeUsage {
		t.Errorf("Expected statusBar.TotalTokens %d, got %d", largeUsage, model.statusBar.TotalTokens)
	}
}
