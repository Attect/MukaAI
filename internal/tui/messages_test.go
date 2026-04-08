// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"errors"
	"testing"
	"time"
)

// TestNewStreamThinkingMsg 测试创建流式思考内容消息
func TestNewStreamThinkingMsg(t *testing.T) {
	chunk := "思考内容测试"
	msg := NewStreamThinkingMsg(chunk)

	if msg.Chunk != chunk {
		t.Errorf("Expected chunk %q, got %q", chunk, msg.Chunk)
	}
}

// TestNewStreamContentMsg 测试创建流式正文内容消息
func TestNewStreamContentMsg(t *testing.T) {
	chunk := "正文内容测试"
	msg := NewStreamContentMsg(chunk)

	if msg.Chunk != chunk {
		t.Errorf("Expected chunk %q, got %q", chunk, msg.Chunk)
	}
}

// TestNewStreamToolCallMsg 测试创建流式工具调用消息
func TestNewStreamToolCallMsg(t *testing.T) {
	call := ToolCall{
		ID:         "tool-123",
		Name:       "create_file",
		Arguments:  `{"path": "/test/file.go"}`,
		IsComplete: false,
	}
	isComplete := true

	msg := NewStreamToolCallMsg(call, isComplete)

	if msg.Call.ID != call.ID {
		t.Errorf("Expected call ID %q, got %q", call.ID, msg.Call.ID)
	}
	if msg.Call.Name != call.Name {
		t.Errorf("Expected call name %q, got %q", call.Name, msg.Call.Name)
	}
	if msg.IsComplete != isComplete {
		t.Errorf("Expected isComplete %v, got %v", isComplete, msg.IsComplete)
	}
}

// TestNewStreamToolResultMsg 测试创建工具执行结果消息
func TestNewStreamToolResultMsg(t *testing.T) {
	result := ToolCall{
		ID:          "tool-123",
		Name:        "create_file",
		Result:      "文件创建成功",
		ResultError: "",
	}

	msg := NewStreamToolResultMsg(result)

	if msg.Result.ID != result.ID {
		t.Errorf("Expected result ID %q, got %q", result.ID, msg.Result.ID)
	}
	if msg.Result.Name != result.Name {
		t.Errorf("Expected result name %q, got %q", result.Name, msg.Result.Name)
	}
}

// TestNewStreamCompleteMsg 测试创建流式输出完成消息
func TestNewStreamCompleteMsg(t *testing.T) {
	usage := 150
	msg := NewStreamCompleteMsg(usage)

	if msg.Usage != usage {
		t.Errorf("Expected usage %d, got %d", usage, msg.Usage)
	}
}

// TestNewStreamErrorMsg 测试创建流式输出错误消息
func TestNewStreamErrorMsg(t *testing.T) {
	err := errors.New("测试错误")
	msg := NewStreamErrorMsg(err)

	if msg.Error != err {
		t.Errorf("Expected error %v, got %v", err, msg.Error)
	}
}

// TestNewConversationCreatedMsg 测试创建对话创建消息
func TestNewConversationCreatedMsg(t *testing.T) {
	conv := &Conversation{
		ID:        "conv-123",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Title:     "测试对话",
	}

	msg := NewConversationCreatedMsg(conv)

	if msg.Conversation.ID != conv.ID {
		t.Errorf("Expected conversation ID %q, got %q", conv.ID, msg.Conversation.ID)
	}
	if msg.Conversation.Title != conv.Title {
		t.Errorf("Expected conversation title %q, got %q", conv.Title, msg.Conversation.Title)
	}
}

// TestNewConversationSwitchedMsg 测试创建对话切换消息
func TestNewConversationSwitchedMsg(t *testing.T) {
	id := "conv-456"
	msg := NewConversationSwitchedMsg(id)

	if msg.ConversationID != id {
		t.Errorf("Expected conversation ID %q, got %q", id, msg.ConversationID)
	}
}

// TestNewConversationStatusUpdatedMsg 测试创建对话状态更新消息
func TestNewConversationStatusUpdatedMsg(t *testing.T) {
	id := "conv-789"
	status := ConvStatusFinished
	msg := NewConversationStatusUpdatedMsg(id, status)

	if msg.ConversationID != id {
		t.Errorf("Expected conversation ID %q, got %q", id, msg.ConversationID)
	}
	if msg.Status != status {
		t.Errorf("Expected status %v, got %v", status, msg.Status)
	}
}

// TestNewWorkingDirChangedMsg 测试创建工作目录变更消息
func TestNewWorkingDirChangedMsg(t *testing.T) {
	oldDir := "/old/path"
	newDir := "/new/path"
	msg := NewWorkingDirChangedMsg(oldDir, newDir)

	if msg.OldDir != oldDir {
		t.Errorf("Expected old dir %q, got %q", oldDir, msg.OldDir)
	}
	if msg.NewDir != newDir {
		t.Errorf("Expected new dir %q, got %q", newDir, msg.NewDir)
	}
}

// TestNewTokenUsageUpdatedMsg 测试创建 token 用量更新消息
func TestNewTokenUsageUpdatedMsg(t *testing.T) {
	total := 1000
	delta := 150
	msg := NewTokenUsageUpdatedMsg(total, delta)

	if msg.TotalTokens != total {
		t.Errorf("Expected total tokens %d, got %d", total, msg.TotalTokens)
	}
	if msg.Delta != delta {
		t.Errorf("Expected delta %d, got %d", delta, msg.Delta)
	}
}

// TestNewInferenceCountUpdatedMsg 测试创建推理次数更新消息
func TestNewInferenceCountUpdatedMsg(t *testing.T) {
	count := 5
	msg := NewInferenceCountUpdatedMsg(count)

	if msg.Count != count {
		t.Errorf("Expected count %d, got %d", count, msg.Count)
	}
}

// TestNewInputModeChangedMsg 测试创建输入模式变更消息
func TestNewInputModeChangedMsg(t *testing.T) {
	mode := InputModeMultiLine
	msg := NewInputModeChangedMsg(mode)

	if msg.Mode != mode {
		t.Errorf("Expected mode %v, got %v", mode, msg.Mode)
	}
}

// TestNewShowConversationListMsg 测试创建显示对话列表消息
func TestNewShowConversationListMsg(t *testing.T) {
	show := true
	msg := NewShowConversationListMsg(show)

	if msg.Show != show {
		t.Errorf("Expected show %v, got %v", show, msg.Show)
	}
}

// TestNewCommandExecutedMsg 测试创建命令执行消息
func TestNewCommandExecutedMsg(t *testing.T) {
	cmd := "cd"
	args := []string{"/new/path"}
	result := "目录切换成功"
	err := errors.New("测试错误")

	// 测试成功情况
	msg := NewCommandExecutedMsg(cmd, args, result, nil)
	if msg.Command != cmd {
		t.Errorf("Expected command %q, got %q", cmd, msg.Command)
	}
	if len(msg.Args) != len(args) {
		t.Errorf("Expected %d args, got %d", len(args), len(msg.Args))
	}
	if msg.Result != result {
		t.Errorf("Expected result %q, got %q", result, msg.Result)
	}
	if msg.Error != nil {
		t.Errorf("Expected no error, got %v", msg.Error)
	}

	// 测试失败情况
	msg2 := NewCommandExecutedMsg(cmd, args, "", err)
	if msg2.Error != err {
		t.Errorf("Expected error %v, got %v", err, msg2.Error)
	}
}

// TestNewBatchUpdateMsg 测试创建批量更新消息
func TestNewBatchUpdateMsg(t *testing.T) {
	result := &FlushResult{
		Thinking:    "思考内容",
		Content:     "正文内容",
		HasThinking: true,
		HasContent:  true,
	}

	msg := NewBatchUpdateMsg(result)

	if msg.Result != result {
		t.Error("Expected result to be set")
	}
	if msg.Result.Thinking != result.Thinking {
		t.Errorf("Expected thinking %q, got %q", result.Thinking, msg.Result.Thinking)
	}
}

// TestNewTickMsg 测试创建定时器消息
func TestNewTickMsg(t *testing.T) {
	now := time.Now()
	msg := NewTickMsg(now)

	if !msg.Time.Equal(now) {
		t.Errorf("Expected time %v, got %v", now, msg.Time)
	}
}

// TestStreamHandlerImpl 测试流式处理器实现
func TestStreamHandlerImpl(t *testing.T) {
	var receivedMessages []interface{}
	sendMsg := func(msg interface{}) {
		receivedMessages = append(receivedMessages, msg)
	}

	handler := NewStreamHandler(sendMsg)

	// 测试 OnThinking
	handler.OnThinking("思考内容")
	if len(receivedMessages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(receivedMessages))
	}
	if thinkingMsg, ok := receivedMessages[0].(StreamThinkingMsg); !ok {
		t.Error("Expected StreamThinkingMsg")
	} else if thinkingMsg.Chunk != "思考内容" {
		t.Errorf("Expected chunk '思考内容', got %q", thinkingMsg.Chunk)
	}

	// 测试 OnContent
	handler.OnContent("正文内容")
	if len(receivedMessages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(receivedMessages))
	}
	if contentMsg, ok := receivedMessages[1].(StreamContentMsg); !ok {
		t.Error("Expected StreamContentMsg")
	} else if contentMsg.Chunk != "正文内容" {
		t.Errorf("Expected chunk '正文内容', got %q", contentMsg.Chunk)
	}

	// 测试 OnToolCall
	call := ToolCall{ID: "tool-1", Name: "test"}
	handler.OnToolCall(call, true)
	if len(receivedMessages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(receivedMessages))
	}
	if toolCallMsg, ok := receivedMessages[2].(StreamToolCallMsg); !ok {
		t.Error("Expected StreamToolCallMsg")
	} else if toolCallMsg.Call.ID != call.ID {
		t.Errorf("Expected call ID %q, got %q", call.ID, toolCallMsg.Call.ID)
	}

	// 测试 OnToolResult
	result := ToolCall{ID: "tool-1", Result: "成功"}
	handler.OnToolResult(result)
	if len(receivedMessages) != 4 {
		t.Errorf("Expected 4 messages, got %d", len(receivedMessages))
	}
	if resultMsg, ok := receivedMessages[3].(StreamToolResultMsg); !ok {
		t.Error("Expected StreamToolResultMsg")
	} else if resultMsg.Result.ID != result.ID {
		t.Errorf("Expected result ID %q, got %q", result.ID, resultMsg.Result.ID)
	}

	// 测试 OnComplete
	handler.OnComplete(100)
	if len(receivedMessages) != 5 {
		t.Errorf("Expected 5 messages, got %d", len(receivedMessages))
	}
	if completeMsg, ok := receivedMessages[4].(StreamCompleteMsg); !ok {
		t.Error("Expected StreamCompleteMsg")
	} else if completeMsg.Usage != 100 {
		t.Errorf("Expected usage 100, got %d", completeMsg.Usage)
	}

	// 测试 OnError
	err := errors.New("测试错误")
	handler.OnError(err)
	if len(receivedMessages) != 6 {
		t.Errorf("Expected 6 messages, got %d", len(receivedMessages))
	}
	if errorMsg, ok := receivedMessages[5].(StreamErrorMsg); !ok {
		t.Error("Expected StreamErrorMsg")
	} else if errorMsg.Error != err {
		t.Errorf("Expected error %v, got %v", err, errorMsg.Error)
	}
}

// TestMessageBuilder 测试消息构建器
func TestMessageBuilder(t *testing.T) {
	// 创建用户消息
	msg := NewMessageBuilder(MessageRoleUser).
		SetContent("用户输入").
		Build()

	if msg.Role != MessageRoleUser {
		t.Errorf("Expected role %v, got %v", MessageRoleUser, msg.Role)
	}
	if msg.Content != "用户输入" {
		t.Errorf("Expected content '用户输入', got %q", msg.Content)
	}

	// 创建助手消息
	toolCall := ToolCall{
		ID:        "tool-1",
		Name:      "create_file",
		Arguments: `{"path": "/test/file.go"}`,
	}

	msg2 := NewMessageBuilder(MessageRoleAssistant).
		SetContent("助手响应").
		SetThinking("思考内容").
		AddToolCall(toolCall).
		SetTokenUsage(150).
		SetStreaming(true, "content").
		Build()

	if msg2.Role != MessageRoleAssistant {
		t.Errorf("Expected role %v, got %v", MessageRoleAssistant, msg2.Role)
	}
	if msg2.Content != "助手响应" {
		t.Errorf("Expected content '助手响应', got %q", msg2.Content)
	}
	if msg2.Thinking != "思考内容" {
		t.Errorf("Expected thinking '思考内容', got %q", msg2.Thinking)
	}
	if len(msg2.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(msg2.ToolCalls))
	}
	if msg2.ToolCalls[0].ID != toolCall.ID {
		t.Errorf("Expected tool call ID %q, got %q", toolCall.ID, msg2.ToolCalls[0].ID)
	}
	if msg2.TokenUsage != 150 {
		t.Errorf("Expected token usage 150, got %d", msg2.TokenUsage)
	}
	if !msg2.IsStreaming {
		t.Error("Expected IsStreaming to be true")
	}
	if msg2.StreamingType != "content" {
		t.Errorf("Expected streaming type 'content', got %q", msg2.StreamingType)
	}
}

// TestConversationBuilder 测试对话构建器
func TestConversationBuilder(t *testing.T) {
	// 创建主对话
	msg := Message{
		Role:      MessageRoleUser,
		Content:   "用户消息",
		Timestamp: time.Now(),
	}

	conv := NewConversationBuilder("conv-1").
		SetTitle("测试对话").
		SetStatus(ConvStatusActive).
		AddMessage(msg).
		SetTokenUsage(200).
		Build()

	if conv.ID != "conv-1" {
		t.Errorf("Expected ID 'conv-1', got %q", conv.ID)
	}
	if conv.Title != "测试对话" {
		t.Errorf("Expected title '测试对话', got %q", conv.Title)
	}
	if conv.Status != ConvStatusActive {
		t.Errorf("Expected status %v, got %v", ConvStatusActive, conv.Status)
	}
	if len(conv.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(conv.Messages))
	}
	if conv.TokenUsage != 200 {
		t.Errorf("Expected token usage 200, got %d", conv.TokenUsage)
	}

	// 创建子对话
	subConv := NewConversationBuilder("conv-2").
		SetTitle("子对话").
		SetSubConversation("conv-1", "Developer").
		Build()

	if !subConv.IsSubConversation {
		t.Error("Expected IsSubConversation to be true")
	}
	if subConv.ParentID != "conv-1" {
		t.Errorf("Expected parent ID 'conv-1', got %q", subConv.ParentID)
	}
	if subConv.AgentRole != "Developer" {
		t.Errorf("Expected agent role 'Developer', got %q", subConv.AgentRole)
	}
}

// TestMessageBuilderChaining 测试消息构建器链式调用
func TestMessageBuilderChaining(t *testing.T) {
	msg := NewMessageBuilder(MessageRoleAssistant).
		SetContent("内容1").
		SetThinking("思考1").
		AddToolCall(ToolCall{ID: "1"}).
		AddToolCall(ToolCall{ID: "2"}).
		SetTokenUsage(100).
		SetStreaming(false, "").
		Build()

	if msg.Content != "内容1" {
		t.Errorf("Expected content '内容1', got %q", msg.Content)
	}
	if msg.Thinking != "思考1" {
		t.Errorf("Expected thinking '思考1', got %q", msg.Thinking)
	}
	if len(msg.ToolCalls) != 2 {
		t.Errorf("Expected 2 tool calls, got %d", len(msg.ToolCalls))
	}
	if msg.TokenUsage != 100 {
		t.Errorf("Expected token usage 100, got %d", msg.TokenUsage)
	}
	if msg.IsStreaming {
		t.Error("Expected IsStreaming to be false")
	}
}

// TestConversationBuilderChaining 测试对话构建器链式调用
func TestConversationBuilderChaining(t *testing.T) {
	msg1 := Message{Role: MessageRoleUser, Content: "消息1"}
	msg2 := Message{Role: MessageRoleAssistant, Content: "消息2"}

	conv := NewConversationBuilder("conv-test").
		SetTitle("链式测试").
		SetStatus(ConvStatusWaiting).
		AddMessage(msg1).
		AddMessage(msg2).
		SetTokenUsage(300).
		Build()

	if conv.Title != "链式测试" {
		t.Errorf("Expected title '链式测试', got %q", conv.Title)
	}
	if conv.Status != ConvStatusWaiting {
		t.Errorf("Expected status %v, got %v", ConvStatusWaiting, conv.Status)
	}
	if len(conv.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(conv.Messages))
	}
	if conv.TokenUsage != 300 {
		t.Errorf("Expected token usage 300, got %d", conv.TokenUsage)
	}
}

// TestMessageTimestamp 测试消息时间戳
func TestMessageTimestamp(t *testing.T) {
	before := time.Now()
	msg := NewMessageBuilder(MessageRoleUser).Build()
	after := time.Now()

	if msg.Timestamp.Before(before) {
		t.Error("Timestamp should not be before creation")
	}
	if msg.Timestamp.After(after) {
		t.Error("Timestamp should not be after creation")
	}
}

// TestConversationTimestamp 测试对话时间戳
func TestConversationTimestamp(t *testing.T) {
	before := time.Now()
	conv := NewConversationBuilder("conv-time").Build()
	after := time.Now()

	if conv.CreatedAt.Before(before) {
		t.Error("CreatedAt should not be before creation")
	}
	if conv.CreatedAt.After(after) {
		t.Error("CreatedAt should not be after creation")
	}
}

// TestEmptyMessageBuilder 测试空消息构建器
func TestEmptyMessageBuilder(t *testing.T) {
	msg := NewMessageBuilder(MessageRoleUser).Build()

	if msg.Role != MessageRoleUser {
		t.Errorf("Expected role %v, got %v", MessageRoleUser, msg.Role)
	}
	if msg.Content != "" {
		t.Errorf("Expected empty content, got %q", msg.Content)
	}
	if msg.Thinking != "" {
		t.Errorf("Expected empty thinking, got %q", msg.Thinking)
	}
	if len(msg.ToolCalls) != 0 {
		t.Errorf("Expected 0 tool calls, got %d", len(msg.ToolCalls))
	}
	if msg.TokenUsage != 0 {
		t.Errorf("Expected token usage 0, got %d", msg.TokenUsage)
	}
}

// TestEmptyConversationBuilder 测试空对话构建器
func TestEmptyConversationBuilder(t *testing.T) {
	conv := NewConversationBuilder("conv-empty").Build()

	if conv.ID != "conv-empty" {
		t.Errorf("Expected ID 'conv-empty', got %q", conv.ID)
	}
	if conv.Title != "" {
		t.Errorf("Expected empty title, got %q", conv.Title)
	}
	if conv.Status != ConvStatusActive {
		t.Errorf("Expected default status %v, got %v", ConvStatusActive, conv.Status)
	}
	if len(conv.Messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(conv.Messages))
	}
	if conv.TokenUsage != 0 {
		t.Errorf("Expected token usage 0, got %d", conv.TokenUsage)
	}
	if conv.IsSubConversation {
		t.Error("Expected IsSubConversation to be false")
	}
}
