// Package components 提供 TUI 组件实现
package components

import (
	"strings"
	"testing"
)

// TestNewChatView 测试创建对话显示组件
func TestNewChatView(t *testing.T) {
	chatView := NewChatView(80, 24)

	if chatView == nil {
		t.Fatal("NewChatView returned nil")
	}

	if chatView.GetWidth() != 80 {
		t.Errorf("Expected width 80, got %d", chatView.GetWidth())
	}

	if chatView.GetHeight() != 24 {
		t.Errorf("Expected height 24, got %d", chatView.GetHeight())
	}

	if !chatView.autoScroll {
		t.Error("autoScroll should be true by default")
	}
}

// TestNewChatViewWithStyles 测试创建带自定义样式的对话显示组件
func TestNewChatViewWithStyles(t *testing.T) {
	customStyles := DefaultChatStyles()
	chatView := NewChatViewWithStyles(100, 30, customStyles)

	if chatView == nil {
		t.Fatal("NewChatViewWithStyles returned nil")
	}

	if chatView.GetWidth() != 100 {
		t.Errorf("Expected width 100, got %d", chatView.GetWidth())
	}

	if chatView.GetHeight() != 30 {
		t.Errorf("Expected height 30, got %d", chatView.GetHeight())
	}
}

// TestChatViewSetSize 测试设置组件大小
func TestChatViewSetSize(t *testing.T) {
	chatView := NewChatView(80, 24)

	chatView.SetSize(100, 30)

	if chatView.GetWidth() != 100 {
		t.Errorf("Expected width 100, got %d", chatView.GetWidth())
	}

	if chatView.GetHeight() != 30 {
		t.Errorf("Expected height 30, got %d", chatView.GetHeight())
	}
}

// TestSetWidth 测试设置宽度
func TestSetWidth(t *testing.T) {
	chatView := NewChatView(80, 24)

	chatView.SetWidth(100)

	if chatView.GetWidth() != 100 {
		t.Errorf("Expected width 100, got %d", chatView.GetWidth())
	}
}

// TestSetHeight 测试设置高度
func TestSetHeight(t *testing.T) {
	chatView := NewChatView(80, 24)

	chatView.SetHeight(30)

	if chatView.GetHeight() != 30 {
		t.Errorf("Expected height 30, got %d", chatView.GetHeight())
	}
}

// TestSetAutoScroll 测试设置自动滚动
func TestSetAutoScroll(t *testing.T) {
	chatView := NewChatView(80, 24)

	chatView.SetAutoScroll(false)
	if chatView.autoScroll {
		t.Error("autoScroll should be false")
	}

	chatView.SetAutoScroll(true)
	if !chatView.autoScroll {
		t.Error("autoScroll should be true")
	}
}

// TestSetContent 测试设置内容
func TestSetContent(t *testing.T) {
	chatView := NewChatView(80, 24)

	content := "Hello, World!"
	chatView.SetContent(content)

	// 验证内容已设置（通过 View 方法）
	view := chatView.View()
	if !strings.Contains(view, "Hello, World!") {
		t.Error("Content not set correctly")
	}
}

// TestRenderUserMessage 测试渲染用户消息
func TestRenderUserMessage(t *testing.T) {
	chatView := NewChatView(80, 24)

	content := "This is a user message"
	rendered := chatView.RenderUserMessage(content)

	// 验证包含用户标识
	if !strings.Contains(rendered, "User:") {
		t.Error("User message should contain 'User:'")
	}

	// 验证包含内容
	if !strings.Contains(rendered, content) {
		t.Error("User message should contain the content")
	}
}

// TestRenderThinking 测试渲染思考内容
func TestRenderThinking(t *testing.T) {
	chatView := NewChatView(80, 24)

	thinking := "This is thinking content"
	rendered := chatView.RenderThinking(thinking, false)

	// 验证包含思考标识
	if !strings.Contains(rendered, "Thinking") {
		t.Error("Thinking content should contain 'Thinking'")
	}

	// 验证包含内容
	if !strings.Contains(rendered, thinking) {
		t.Error("Thinking content should contain the thinking text")
	}

	// 测试流式光标
	renderedStreaming := chatView.RenderThinking(thinking, true)
	if !strings.Contains(renderedStreaming, "▌") {
		t.Error("Streaming thinking should contain cursor")
	}
}

// TestRenderContent 测试渲染正文内容
func TestRenderContent(t *testing.T) {
	chatView := NewChatView(80, 24)

	content := "This is main content"
	rendered := chatView.RenderContent(content, false)

	// 验证包含内容
	if !strings.Contains(rendered, content) {
		t.Error("Content should contain the text")
	}

	// 测试流式光标
	renderedStreaming := chatView.RenderContent(content, true)
	if !strings.Contains(renderedStreaming, "▌") {
		t.Error("Streaming content should contain cursor")
	}
}

// TestRenderToolCall 测试渲染工具调用
func TestRenderToolCall(t *testing.T) {
	chatView := NewChatView(80, 24)

	tc := ToolCallData{
		ID:        "tool-1",
		Name:      "read_file",
		Arguments: `{"path": "/test/file.txt"}`,
		IsComplete: true,
	}

	rendered := chatView.RenderToolCall(tc, false)

	// 验证包含工具名称
	if !strings.Contains(rendered, "read_file") {
		t.Error("Tool call should contain the tool name")
	}

	// 验证包含参数
	if !strings.Contains(rendered, "Parameters:") {
		t.Error("Tool call should contain 'Parameters:'")
	}

	// 测试流式生成中
	tc.IsComplete = false
	renderedStreaming := chatView.RenderToolCall(tc, true)
	if !strings.Contains(renderedStreaming, "▌") {
		t.Error("Streaming tool call should contain cursor")
	}
}

// TestRenderToolResult 测试渲染工具结果
func TestRenderToolResult(t *testing.T) {
	chatView := NewChatView(80, 24)

	// 测试成功结果
	tc := ToolCallData{
		ID:     "tool-1",
		Name:   "read_file",
		Result: "File content here",
	}

	rendered := chatView.RenderToolResult(tc)

	// 验证包含结果标识
	if !strings.Contains(rendered, "Tool Result") {
		t.Error("Tool result should contain 'Tool Result'")
	}

	// 验证包含内容
	if !strings.Contains(rendered, "File content here") {
		t.Error("Tool result should contain the result text")
	}

	// 测试错误结果
	tcError := ToolCallData{
		ID:          "tool-2",
		Name:        "read_file",
		ResultError: "File not found",
	}

	renderedError := chatView.RenderToolResult(tcError)
	if !strings.Contains(renderedError, "File not found") {
		t.Error("Tool error should contain the error text")
	}
}

// TestRenderTokenUsage 测试渲染 token 用量
func TestRenderTokenUsage(t *testing.T) {
	chatView := NewChatView(80, 24)

	usage := 150
	rendered := chatView.RenderTokenUsage(usage)

	// 验证包含 token 标识
	if !strings.Contains(rendered, "Tokens:") {
		t.Error("Token usage should contain 'Tokens:'")
	}

	// 验证包含数量
	if !strings.Contains(rendered, "150") {
		t.Error("Token usage should contain the usage number")
	}
}

// TestRenderError 测试渲染错误消息
func TestRenderError(t *testing.T) {
	chatView := NewChatView(80, 24)

	errMsg := "Something went wrong"
	rendered := chatView.RenderError(errMsg)

	// 验证包含错误标识
	if !strings.Contains(rendered, "Error:") {
		t.Error("Error message should contain 'Error:'")
	}

	// 验证包含错误内容
	if !strings.Contains(rendered, errMsg) {
		t.Error("Error message should contain the error text")
	}
}

// TestRenderMessages 测试渲染消息列表
func TestRenderMessages(t *testing.T) {
	chatView := NewChatView(80, 24)

	messages := []MessageData{
		{
			Role:    "user",
			Content: "Hello",
		},
		{
			Role:      "assistant",
			Thinking:  "Let me think...",
			Content:   "Hi there!",
			TokenUsage: 50,
		},
	}

	rendered := chatView.RenderMessages(messages)

	// 验证包含用户消息
	if !strings.Contains(rendered, "User:") {
		t.Error("Should contain user message")
	}

	// 验证包含助手消息
	if !strings.Contains(rendered, "Thinking") {
		t.Error("Should contain thinking content")
	}

	// 验证包含正文
	if !strings.Contains(rendered, "Hi there!") {
		t.Error("Should contain assistant content")
	}
}

// TestRenderAssistantMessage 测试渲染助手消息
func TestRenderAssistantMessage(t *testing.T) {
	chatView := NewChatView(80, 24)

	// 测试包含思考内容的助手消息
	msg := MessageData{
		Role:     "assistant",
		Thinking: "Let me think about this...",
		Content:  "Here's my answer",
	}

	rendered := chatView.RenderAssistantMessage(msg)

	// 验证包含思考内容
	if !strings.Contains(rendered, "Thinking") {
		t.Error("Should contain thinking content")
	}

	// 验证包含正文
	if !strings.Contains(rendered, "Here's my answer") {
		t.Error("Should contain assistant content")
	}

	// 测试包含工具调用的助手消息
	msgWithTool := MessageData{
		Role: "assistant",
		ToolCalls: []ToolCallData{
			{
				ID:         "tool-1",
				Name:       "read_file",
				Arguments:  `{"path": "/test/file.txt"}`,
				IsComplete: true,
				Result:     "File content",
			},
		},
	}

	renderedWithTool := chatView.RenderAssistantMessage(msgWithTool)

	// 验证包含工具调用
	if !strings.Contains(renderedWithTool, "read_file") {
		t.Error("Should contain tool call")
	}

	// 验证包含工具结果
	if !strings.Contains(renderedWithTool, "File content") {
		t.Error("Should contain tool result")
	}
}

// TestRenderToolMessage 测试渲染工具消息
func TestRenderToolMessage(t *testing.T) {
	chatView := NewChatView(80, 24)

	msg := MessageData{
		Role: "tool",
		ToolCalls: []ToolCallData{
			{
				ID:     "tool-1",
				Result: "Tool execution result",
			},
		},
	}

	rendered := chatView.RenderToolMessage(msg)

	// 验证包含工具结果
	if !strings.Contains(rendered, "Tool Result") {
		t.Error("Should contain tool result")
	}
}

// TestRenderMessage 测试渲染单条消息
func TestRenderMessage(t *testing.T) {
	chatView := NewChatView(80, 24)

	// 测试用户消息
	userMsg := MessageData{
		Role:    "user",
		Content: "Test message",
	}

	rendered := chatView.RenderMessage(userMsg)
	if !strings.Contains(rendered, "User:") {
		t.Error("Should render user message")
	}

	// 测试助手消息
	assistantMsg := MessageData{
		Role:    "assistant",
		Content: "Assistant response",
	}

	rendered = chatView.RenderMessage(assistantMsg)
	if !strings.Contains(rendered, "Assistant response") {
		t.Error("Should render assistant message")
	}

	// 测试工具消息
	toolMsg := MessageData{
		Role: "tool",
		ToolCalls: []ToolCallData{
			{
				Result: "Tool result",
			},
		},
	}

	rendered = chatView.RenderMessage(toolMsg)
	if !strings.Contains(rendered, "Tool Result") {
		t.Error("Should render tool message")
	}
}

// TestDefaultChatStyles 测试默认样式
func TestDefaultChatStyles(t *testing.T) {
	styles := DefaultChatStyles()

	// 验证样式已初始化
	if styles.UserMessage.GetForeground() == nil {
		t.Error("UserMessage style should have foreground color")
	}

	if styles.Thinking.GetForeground() == nil {
		t.Error("Thinking style should have foreground color")
	}

	if styles.Content.GetForeground() == nil {
		t.Error("Content style should have foreground color")
	}

	if styles.ToolCall.GetForeground() == nil {
		t.Error("ToolCall style should have foreground color")
	}

	if styles.ToolResult.GetForeground() == nil {
		t.Error("ToolResult style should have foreground color")
	}
}

// TestSetStyles 测试设置样式
func TestSetStyles(t *testing.T) {
	chatView := NewChatView(80, 24)

	newStyles := DefaultChatStyles()
	chatView.SetStyles(newStyles)

	// 验证样式已设置
	styles := chatView.GetStyles()
	if styles.UserMessage.GetForeground() == nil {
		t.Error("Styles should be set correctly")
	}
}

// TestGetViewport 测试获取视口组件
func TestGetViewport(t *testing.T) {
	chatView := NewChatView(80, 24)

	vp := chatView.GetViewport()
	if vp == nil {
		t.Error("GetViewport should not return nil")
	}
}

// TestAutoScroll 测试自动滚动功能
func TestAutoScroll(t *testing.T) {
	chatView := NewChatView(80, 24)

	// 启用自动滚动
	chatView.SetAutoScroll(true)

	// 设置多行内容
	content := strings.Repeat("Line\n", 100)
	chatView.SetContent(content)

	// 验证内容已设置
	view := chatView.View()
	if view == "" {
		t.Error("Content should be set")
	}

	// 禁用自动滚动
	chatView.SetAutoScroll(false)

	// 设置新内容
	newContent := strings.Repeat("New Line\n", 50)
	chatView.SetContent(newContent)

	// 验证新内容已设置
	view = chatView.View()
	if view == "" {
		t.Error("New content should be set")
	}
}

// TestScrollMethods 测试滚动方法
func TestScrollMethods(t *testing.T) {
	chatView := NewChatView(80, 24)

	// 设置多行内容
	content := strings.Repeat("Line\n", 100)
	chatView.SetContent(content)

	// 测试滚动到顶部
	chatView.ScrollToTop()

	// 测试滚动到底部
	chatView.ScrollToBottom()

	// 测试翻页
	chatView.PageDown()
	chatView.PageUp()

	// 测试单行滚动
	chatView.LineDown()
	chatView.LineUp()

	// 如果没有 panic，测试通过
}
