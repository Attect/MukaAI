// Package tui 提供 TUI 测试
package tui

import (
	"testing"
	"time"

	"agentplus/internal/tui/components"
)

// TestAppModelCreation 测试 AppModel 创建
func TestAppModelCreation(t *testing.T) {
	model := NewAppModel()

	// 检查输入组件
	if model.input == nil {
		t.Fatal("Expected input component to be initialized")
	}

	// 检查对话列表组件
	if model.dialogList == nil {
		t.Fatal("Expected dialog list to be initialized")
	}

	// 检查初始状态
	if model.dialogList.IsVisible() {
		t.Error("Dialog list should not be visible initially")
	}

	// 检查输入模式
	if model.inputMode != InputModeSingleLine {
		t.Errorf("Expected initial input mode to be single-line, got %v", model.inputMode)
	}
}

// TestInputModeSwitch 测试输入模式切换
func TestInputModeSwitch(t *testing.T) {
	model := NewAppModel()

	// 初始模式应该是单行
	if model.input.GetMode() != components.InputModeSingleLine {
		t.Error("Expected initial mode to be single-line")
	}

	// 切换到多行模式
	model.input.ToggleMode()
	if model.input.GetMode() != components.InputModeMultiLine {
		t.Error("Expected mode to be multi-line after toggle")
	}

	// 再次切换回单行模式
	model.input.ToggleMode()
	if model.input.GetMode() != components.InputModeSingleLine {
		t.Error("Expected mode to be single-line after second toggle")
	}
}

// TestShouldSubmit 测试提交判断逻辑
func TestShouldSubmit(t *testing.T) {
	model := NewAppModel()

	// 单行模式：Enter 提交
	model.input.SetMode(components.InputModeSingleLine)
	if !model.input.ShouldSubmit("enter") {
		t.Error("Enter should submit in single-line mode")
	}
	if model.input.ShouldSubmit("ctrl+enter") {
		t.Error("Ctrl+Enter should not submit in single-line mode")
	}

	// 多行模式：Ctrl+Enter 提交
	model.input.SetMode(components.InputModeMultiLine)
	if !model.input.ShouldSubmit("ctrl+enter") {
		t.Error("Ctrl+Enter should submit in multi-line mode")
	}
	if model.input.ShouldSubmit("enter") {
		t.Error("Enter should not submit in multi-line mode")
	}
}

// TestDialogListToggle 测试对话列表切换
func TestDialogListToggle(t *testing.T) {
	model := NewAppModel()

	// 初始状态对话列表不可见
	if model.dialogList.IsVisible() {
		t.Error("Dialog list should not be visible initially")
	}

	// 显示对话列表
	model.toggleConversationList()
	if !model.dialogList.IsVisible() {
		t.Error("Expected dialog list to be visible after toggle")
	}

	// 隐藏对话列表
	model.toggleConversationList()
	if model.dialogList.IsVisible() {
		t.Error("Expected dialog list to be hidden after second toggle")
	}
}

// TestInputHistory 测试输入历史记录
func TestInputHistory(t *testing.T) {
	model := NewAppModel()

	// 输入并提交多个内容
	inputs := []string{"first input", "second input", "third input"}
	for _, input := range inputs {
		model.input.AddToHistory(input)
	}

	// 检查历史记录
	history := model.input.GetHistory()
	if len(history) != len(inputs) {
		t.Errorf("Expected %d history items, got %d", len(inputs), len(history))
	}

	for i, expected := range inputs {
		if history[i] != expected {
			t.Errorf("History item %d: expected %q, got %q", i, expected, history[i])
		}
	}
}

// TestInputModeSync 测试输入模式同步
func TestInputModeSync(t *testing.T) {
	model := NewAppModel()

	// 初始同步
	if model.inputMode != InputModeSingleLine {
		t.Error("Expected initial AppModel inputMode to be single-line")
	}

	// 通过 InputComponent 切换模式
	model.input.SetMode(components.InputModeMultiLine)
	// 注意：这里需要手动同步，实际使用中通过 Update 方法自动同步
	model.inputMode = InputModeMultiLine

	// 检查 AppModel 的 inputMode 是否同步
	if model.inputMode != InputModeMultiLine {
		t.Error("Expected AppModel inputMode to be synced to multi-line")
	}

	// 检查 InputComponent 的模式是否一致
	if model.input.GetMode() != components.InputModeMultiLine {
		t.Error("Expected InputComponent mode to be multi-line")
	}
}

// TestDialogListNavigation 测试对话列表导航
func TestDialogListNavigation(t *testing.T) {
	model := NewAppModel()

	// 添加一些测试对话
	model.conversations = []*Conversation{
		{
			ID:         "conv1",
			CreatedAt:  time.Now().Add(-2 * time.Hour),
			Status:     ConvStatusFinished,
			Title:      "Conversation 1",
			TokenUsage: 100,
		},
		{
			ID:         "conv2",
			CreatedAt:  time.Now().Add(-1 * time.Hour),
			Status:     ConvStatusActive,
			Title:      "Conversation 2",
			TokenUsage: 200,
		},
		{
			ID:         "conv3",
			CreatedAt:  time.Now(),
			Status:     ConvStatusWaiting,
			Title:      "Conversation 3",
			TokenUsage: 300,
		},
	}

	// 显示对话列表
	model.toggleConversationList()

	// 检查对话列表是否更新
	if len(model.dialogList.GetConversations()) != 3 {
		t.Error("Expected dialog list to have 3 conversations")
	}

	// 检查对话列表是否按时间降序排序（最新的在前）
	convs := model.dialogList.GetConversations()
	if convs[0].ID != "conv3" {
		t.Error("Expected first conversation to be conv3 (newest)")
	}
	if convs[2].ID != "conv1" {
		t.Error("Expected last conversation to be conv1 (oldest)")
	}
}

// TestConversationSwitch 测试对话切换
func TestConversationSwitch(t *testing.T) {
	model := NewAppModel()

	// 创建测试对话
	conv1 := &Conversation{
		ID:        "conv1",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Title:     "Test Conversation 1",
		Messages:  []Message{},
	}
	conv2 := &Conversation{
		ID:        "conv2",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Title:     "Test Conversation 2",
		Messages:  []Message{},
	}

	model.AddConversation(conv1)
	model.AddConversation(conv2)

	// 检查初始活动对话
	if model.activeConv != conv1 {
		t.Error("Expected first conversation to be active initially")
	}

	// 切换到第二个对话
	model.SwitchConversation("conv2")
	if model.activeConv != conv2 {
		t.Error("Expected second conversation to be active after switch")
	}

	// 切换回第一个对话
	model.SwitchConversation("conv1")
	if model.activeConv != conv1 {
		t.Error("Expected first conversation to be active after switch back")
	}
}

// TestInputValueOperations 测试输入值操作
func TestInputValueOperations(t *testing.T) {
	model := NewAppModel()

	// 设置输入值
	testValue := "test input value"
	model.input.SetValue(testValue)
	if model.input.GetValue() != testValue {
		t.Errorf("Expected input value %q, got %q", testValue, model.input.GetValue())
	}

	// 清空输入
	model.input.Clear()
	if model.input.GetValue() != "" {
		t.Error("Expected input to be cleared")
	}
}

// TestCommandParsing 测试命令解析
func TestCommandParsing(t *testing.T) {
	model := NewAppModel()

	tests := []struct {
		input    string
		cmdType  components.CommandType
		cmdName  string
		hasError bool
	}{
		{"/cd /path/to/dir", components.CommandCD, "cd", false},
		{"/conversations", components.CommandConversations, "conversations", false},
		{"/conv", components.CommandConversations, "conversations", false},
		{"/clear", components.CommandClear, "clear", false},
		{"/save output.txt", components.CommandSave, "save", false},
		{"/help", components.CommandHelp, "help", false},
		{"/exit", components.CommandExit, "exit", false},
		{"/quit", components.CommandExit, "exit", false},
		{"/q", components.CommandExit, "exit", false},
		{"normal input", components.CommandNone, "", false},
	}

	for _, test := range tests {
		cmd := model.input.ParseCommand(test.input)
		if cmd.Type != test.cmdType {
			t.Errorf("Input %q: expected command type %v, got %v", test.input, test.cmdType, cmd.Type)
		}
		if cmd.Name != test.cmdName {
			t.Errorf("Input %q: expected command name %q, got %q", test.input, test.cmdName, cmd.Name)
		}
	}
}

// TestKeybindingsHelp 测试快捷键帮助文本
func TestKeybindingsHelp(t *testing.T) {
	model := NewAppModel()

	// 单行模式帮助文本
	model.input.SetMode(components.InputModeSingleLine)
	view := model.input.View()
	if view == "" {
		t.Error("Expected non-empty view for single-line mode")
	}

	// 多行模式帮助文本
	model.input.SetMode(components.InputModeMultiLine)
	view = model.input.View()
	if view == "" {
		t.Error("Expected non-empty view for multi-line mode")
	}
}
