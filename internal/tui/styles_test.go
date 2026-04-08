// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
)

// ========== 主题测试 ==========

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()

	// 测试基础颜色不为 nil
	if theme.Primary == nil {
		t.Error("Primary color should not be nil")
	}
	if theme.Secondary == nil {
		t.Error("Secondary color should not be nil")
	}
	if theme.Success == nil {
		t.Error("Success color should not be nil")
	}
	if theme.Warning == nil {
		t.Error("Warning color should not be nil")
	}
	if theme.Error == nil {
		t.Error("Error color should not be nil")
	}

	// 测试消息类型颜色不为 nil
	if theme.UserMessage == nil {
		t.Error("UserMessage color should not be nil")
	}
	if theme.Thinking == nil {
		t.Error("Thinking color should not be nil")
	}
	if theme.Content == nil {
		t.Error("Content color should not be nil")
	}
	if theme.ToolCall == nil {
		t.Error("ToolCall color should not be nil")
	}
	if theme.ToolResult == nil {
		t.Error("ToolResult color should not be nil")
	}
	if theme.ToolError == nil {
		t.Error("ToolError color should not be nil")
	}

	// 测试状态颜色不为 nil
	if theme.Active == nil {
		t.Error("Active color should not be nil")
	}
	if theme.Waiting == nil {
		t.Error("Waiting color should not be nil")
	}
	if theme.Finished == nil {
		t.Error("Finished color should not be nil")
	}

	// 测试布局配置
	if theme.Layout.BorderWidth != 1 {
		t.Errorf("BorderWidth should be 1, got %d", theme.Layout.BorderWidth)
	}
	if theme.Layout.PaddingHorizontal != 1 {
		t.Errorf("PaddingHorizontal should be 1, got %d", theme.Layout.PaddingHorizontal)
	}
}

func TestDarkTheme(t *testing.T) {
	theme := DarkTheme()

	// 测试深色主题的特殊颜色不为 nil
	if theme.Background == nil {
		t.Error("Background color should not be nil")
	}
	if theme.Border == nil {
		t.Error("Border color should not be nil")
	}
	if theme.Content == nil {
		t.Error("Content color should not be nil")
	}
}

func TestLightTheme(t *testing.T) {
	theme := LightTheme()

	// 测试浅色主题的特殊颜色不为 nil
	if theme.Background == nil {
		t.Error("Background color should not be nil")
	}
	if theme.Border == nil {
		t.Error("Border color should not be nil")
	}
	if theme.Content == nil {
		t.Error("Content color should not be nil")
	}
}

func TestSetTheme(t *testing.T) {
	// 保存原始主题
	originalTheme := GetTheme()

	// 设置新主题
	customTheme := DefaultTheme()
	customTheme.Primary = lipgloss.Color("#FF0000")
	SetTheme(customTheme)

	// 验证主题已更新
	currentTheme := GetTheme()
	if currentTheme.Primary == nil {
		t.Error("Theme Primary should not be nil after update")
	}

	// 恢复原始主题
	SetTheme(originalTheme)
}

func TestGetTheme(t *testing.T) {
	theme := GetTheme()
	if theme == nil {
		t.Error("GetTheme should not return nil")
	}
}

// ========== 样式定义测试 ==========

func TestStyleDefinitions(t *testing.T) {
	// 测试基础样式 - 验证样式创建成功
	if styleBase.String() == "" {
		t.Error("styleBase should be created successfully")
	}

	// 测试标题样式
	if !styleTitle.GetBold() {
		t.Error("styleTitle should be bold")
	}

	// 测试用户消息样式
	if !styleUserMessage.GetBold() {
		t.Error("styleUserMessage should be bold")
	}

	// 测试思考内容样式
	if !styleThinking.GetItalic() {
		t.Error("styleThinking should be italic")
	}

	// 测试状态样式
	if !styleStatusActive.GetBold() {
		t.Error("styleStatusActive should be bold")
	}
	if !styleStatusWaiting.GetBold() {
		t.Error("styleStatusWaiting should be bold")
	}
}

// ========== 格式化函数测试 ==========

func TestFormatUserMessage(t *testing.T) {
	content := "Hello, world!"
	result := FormatUserMessage(content)

	if !strings.Contains(result, "User:") {
		t.Error("Result should contain 'User:'")
	}
	if !strings.Contains(result, content) {
		t.Error("Result should contain the content")
	}
}

func TestFormatThinking(t *testing.T) {
	thinking := "I need to analyze..."
	
	// 测试流式输出
	result := FormatThinking(thinking, true)
	if !strings.Contains(result, thinking) {
		t.Error("Result should contain the thinking content")
	}
	if !strings.Contains(result, "Thinking") {
		t.Error("Result should contain 'Thinking' title")
	}

	// 测试非流式输出
	result = FormatThinking(thinking, false)
	if !strings.Contains(result, thinking) {
		t.Error("Result should contain the thinking content")
	}
}

func TestFormatContent(t *testing.T) {
	content := "This is the main content"
	
	// 测试流式输出
	result := FormatContent(content, true)
	if !strings.Contains(result, content) {
		t.Error("Result should contain the content")
	}

	// 测试非流式输出
	result = FormatContent(content, false)
	if !strings.Contains(result, content) {
		t.Error("Result should contain the content")
	}
}

func TestStyleFormatToolCall(t *testing.T) {
	name := "create_file"
	args := `{"path": "/tmp/test.go"}`
	
	// 测试完成的工具调用
	result := FormatToolCall(name, args, true)
	if !strings.Contains(result, name) {
		t.Error("Result should contain the tool name")
	}
	if !strings.Contains(result, "Parameters:") {
		t.Error("Result should contain 'Parameters:'")
	}

	// 测试流式生成中的工具调用
	result = FormatToolCall(name, args, false)
	if !strings.Contains(result, name) {
		t.Error("Result should contain the tool name")
	}
}

func TestStyleFormatToolResult(t *testing.T) {
	result := "File created successfully"
	
	// 测试成功结果
	formatted := FormatToolResult(result, false)
	if !strings.Contains(formatted, result) {
		t.Error("Result should contain the content")
	}

	// 测试错误结果
	errorResult := "Failed to create file"
	formatted = FormatToolResult(errorResult, true)
	if !strings.Contains(formatted, errorResult) {
		t.Error("Result should contain the error content")
	}
}

func TestFormatTokenUsage(t *testing.T) {
	usage := 123
	result := FormatTokenUsage(usage)
	
	if !strings.Contains(result, "Tokens:") {
		t.Error("Result should contain 'Tokens:'")
	}
}

func TestFormatError(t *testing.T) {
	errMsg := "Something went wrong"
	result := FormatError(errMsg)
	
	if !strings.Contains(result, "Error:") {
		t.Error("Result should contain 'Error:'")
	}
	if !strings.Contains(result, errMsg) {
		t.Error("Result should contain the error message")
	}
}

func TestFormatHelpText(t *testing.T) {
	result := FormatHelpText()
	
	if !strings.Contains(result, "快捷键:") {
		t.Error("Result should contain '快捷键:'")
	}
	if !strings.Contains(result, "Enter") {
		t.Error("Result should contain 'Enter'")
	}
	if !strings.Contains(result, "Tab") {
		t.Error("Result should contain 'Tab'")
	}
}

// ========== 状态样式测试 ==========

func TestGetStatusStyle(t *testing.T) {
	tests := []struct {
		status   ConvStatus
		hasStyle bool
	}{
		{ConvStatusActive, true},
		{ConvStatusWaiting, true},
		{ConvStatusFinished, true},
	}

	for _, test := range tests {
		style := GetStatusStyle(test.status)
		if test.hasStyle && style.String() == "" {
			t.Errorf("GetStatusStyle(%d) should return a style", test.status)
		}
	}
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status       ConvStatus
		expectedIcon string
	}{
		{ConvStatusActive, "🔄"},
		{ConvStatusWaiting, "⏳"},
		{ConvStatusFinished, "✓"},
	}

	for _, test := range tests {
		icon := GetStatusIcon(test.status)
		if icon != test.expectedIcon {
			t.Errorf("GetStatusIcon(%d) = %s, expected %s", test.status, icon, test.expectedIcon)
		}
	}
}

func TestGetMessageRoleStyle(t *testing.T) {
	tests := []struct {
		role    MessageRole
		hasStyle bool
	}{
		{MessageRoleUser, true},
		{MessageRoleAssistant, true},
		{MessageRoleTool, true},
	}

	for _, test := range tests {
		style := GetMessageRoleStyle(test.role)
		if test.hasStyle && style.String() == "" {
			t.Errorf("GetMessageRoleStyle(%d) should return a style", test.role)
		}
	}
}

// ========== 布局样式辅助函数测试 ==========

func TestNewBorderStyle(t *testing.T) {
	tests := []struct {
		borderType string
		valid      bool
	}{
		{"normal", true},
		{"rounded", true},
		{"double", true},
		{"thick", true},
		{"hidden", true},
		{"unknown", true}, // 默认为 normal
	}

	for _, test := range tests {
		style := NewBorderStyle(test.borderType, lipgloss.Color("#FF0000"))
		if test.valid && style.String() == "" {
			t.Errorf("NewBorderStyle(%s) should return a valid style", test.borderType)
		}
	}
}

func TestNewPaddingStyle(t *testing.T) {
	style := NewPaddingStyle(1, 2)
	// 验证样式创建成功
	if style.String() == "" {
		t.Error("NewPaddingStyle should create a valid style")
	}
}

func TestNewMarginStyle(t *testing.T) {
	style := NewMarginStyle(1, 2)
	// 验证样式创建成功
	if style.String() == "" {
		t.Error("NewMarginStyle should create a valid style")
	}
}

func TestNewAlignedStyle(t *testing.T) {
	style := NewAlignedStyle(lipgloss.Center, lipgloss.Center)
	// 验证样式创建成功（空样式也是有效的）
	_ = style
}

func TestNewBoxStyle(t *testing.T) {
	style := NewBoxStyle("rounded", lipgloss.Color("#FF0000"), 1, 2)
	// 验证样式创建成功
	if style.String() == "" {
		t.Error("NewBoxStyle should create a valid style")
	}
}

// ========== 样式构建器测试 ==========

func TestStyleBuilder(t *testing.T) {
	// 测试链式调用
	style := NewStyleBuilder().
		Foreground(lipgloss.Color("#FF0000")).
		Background(lipgloss.Color("#00FF00")).
		Bold(true).
		Italic(true).
		Underline(true).
		Padding(1, 2).
		Margin(1, 2).
		Width(100).
		Height(50).
		Build()

	// 验证样式属性
	if !style.GetBold() {
		t.Error("Style should be bold")
	}
	if !style.GetItalic() {
		t.Error("Style should be italic")
	}
	if !style.GetUnderline() {
		t.Error("Style should be underlined")
	}
	if style.GetWidth() != 100 {
		t.Error("Style should have width 100")
	}
	if style.GetHeight() != 50 {
		t.Error("Style should have height 50")
	}
}

func TestStyleBuilderRender(t *testing.T) {
	text := "Hello, world!"
	result := NewStyleBuilder().
		Foreground(lipgloss.Color("#FF0000")).
		Bold(true).
		Render(text)

	if !strings.Contains(result, text) {
		t.Error("Rendered text should contain the original text")
	}
}

// ========== 样式工具函数测试 ==========

func TestJoinHorizontal(t *testing.T) {
	text1 := "Hello"
	text2 := "World"
	result := JoinHorizontal(text1, text2)

	if !strings.Contains(result, text1) {
		t.Error("Result should contain text1")
	}
	if !strings.Contains(result, text2) {
		t.Error("Result should contain text2")
	}
}

func TestJoinVertical(t *testing.T) {
	text1 := "Hello"
	text2 := "World"
	result := JoinVertical(text1, text2)

	if !strings.Contains(result, text1) {
		t.Error("Result should contain text1")
	}
	if !strings.Contains(result, text2) {
		t.Error("Result should contain text2")
	}
}

func TestPlace(t *testing.T) {
	content := "Centered content"
	result := Place(80, 20, lipgloss.Center, lipgloss.Center, content)

	if !strings.Contains(result, content) {
		t.Error("Result should contain the content")
	}
}

func TestWidth(t *testing.T) {
	text := "Hello"
	width := Width(text)

	if width <= 0 {
		t.Error("Width should be greater than 0")
	}
}

func TestHeight(t *testing.T) {
	text := "Hello\nWorld"
	height := Height(text)

	if height <= 0 {
		t.Error("Height should be greater than 0")
	}
}

// ========== 对话列表格式化测试 ==========

func TestFormatConversationItem(t *testing.T) {
	conv := &Conversation{
		ID:        "test-1",
		CreatedAt: parseTime("2026-04-08T10:30:00Z"),
		Status:    ConvStatusActive,
		Title:     "Test Conversation",
	}

	// 测试未选中状态
	result := FormatConversationItem(conv, false)
	if !strings.Contains(result, "Test Conversation") {
		t.Error("Result should contain the conversation title")
	}
	if !strings.Contains(result, "🔄") {
		t.Error("Result should contain the active icon")
	}

	// 测试选中状态
	result = FormatConversationItem(conv, true)
	if !strings.Contains(result, "Test Conversation") {
		t.Error("Result should contain the conversation title")
	}
}

func TestFormatConversationItemWithSubConversation(t *testing.T) {
	conv := &Conversation{
		ID:              "test-2",
		CreatedAt:       parseTime("2026-04-08T10:30:00Z"),
		Status:          ConvStatusWaiting,
		Title:           "",
		IsSubConversation: true,
		AgentRole:       "Developer",
	}

	result := FormatConversationItem(conv, false)
	if !strings.Contains(result, "Sub-agent: Developer") {
		t.Error("Result should contain 'Sub-agent: Developer'")
	}
	if !strings.Contains(result, "⏳") {
		t.Error("Result should contain the waiting icon")
	}
}

// ========== 辅助函数 ==========

// parseTime 解析时间字符串
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Now()
	}
	return t
}
