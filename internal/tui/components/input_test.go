// Package components 提供 TUI 组件
package components

import (
	"strings"
	"testing"
)

// TestNewInputComponent 测试创建输入组件
func TestNewInputComponent(t *testing.T) {
	config := DefaultInputComponentConfig()
	ic := NewInputComponent(config)

	if ic == nil {
		t.Fatal("Expected non-nil InputComponent")
	}

	if ic.mode != InputModeSingleLine {
		t.Errorf("Expected mode %v, got %v", InputModeSingleLine, ic.mode)
	}

	if ic.focused != true {
		t.Error("Expected focused to be true")
	}

	if len(ic.history) != 0 {
		t.Errorf("Expected empty history, got %d items", len(ic.history))
	}

	if ic.maxHistory != config.MaxHistory {
		t.Errorf("Expected maxHistory %d, got %d", config.MaxHistory, ic.maxHistory)
	}
}

// TestInputComponentModeSwitch 测试输入模式切换
func TestInputComponentModeSwitch(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 初始模式应该是单行
	if ic.GetMode() != InputModeSingleLine {
		t.Errorf("Expected mode %v, got %v", InputModeSingleLine, ic.GetMode())
	}

	// 切换到多行模式
	ic.ToggleMode()
	if ic.GetMode() != InputModeMultiLine {
		t.Errorf("Expected mode %v, got %v", InputModeMultiLine, ic.GetMode())
	}

	// 切换回单行模式
	ic.ToggleMode()
	if ic.GetMode() != InputModeSingleLine {
		t.Errorf("Expected mode %v, got %v", InputModeSingleLine, ic.GetMode())
	}

	// 测试 SetMode
	ic.SetMode(InputModeMultiLine)
	if ic.GetMode() != InputModeMultiLine {
		t.Errorf("Expected mode %v, got %v", InputModeMultiLine, ic.GetMode())
	}

	ic.SetMode(InputModeSingleLine)
	if ic.GetMode() != InputModeSingleLine {
		t.Errorf("Expected mode %v, got %v", InputModeSingleLine, ic.GetMode())
	}
}

// TestInputComponentValue 测试输入值操作
func TestInputComponentValue(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 测试设置和获取值
	testValue := "Hello, World!"
	ic.SetValue(testValue)
	if ic.GetValue() != testValue {
		t.Errorf("Expected value %q, got %q", testValue, ic.GetValue())
	}

	// 测试清空
	ic.Clear()
	if ic.GetValue() != "" {
		t.Errorf("Expected empty value, got %q", ic.GetValue())
	}
}

// TestInputComponentHistory 测试历史记录功能
func TestInputComponentHistory(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 添加历史记录
	inputs := []string{"input1", "input2", "input3"}
	for _, input := range inputs {
		ic.AddToHistory(input)
	}

	// 检查历史记录数量
	history := ic.GetHistory()
	if len(history) != len(inputs) {
		t.Errorf("Expected %d history items, got %d", len(inputs), len(history))
	}

	// 检查历史记录内容
	for i, input := range inputs {
		if history[i] != input {
			t.Errorf("Expected history[%d] = %q, got %q", i, input, history[i])
		}
	}

	// 测试重复输入不会添加
	ic.AddToHistory("input3")
	if len(ic.GetHistory()) != len(inputs) {
		t.Error("Expected duplicate input not to be added")
	}

	// 测试空输入不会添加
	ic.AddToHistory("")
	ic.AddToHistory("   ")
	if len(ic.GetHistory()) != len(inputs) {
		t.Error("Expected empty input not to be added")
	}

	// 测试清空历史
	ic.ClearHistory()
	if len(ic.GetHistory()) != 0 {
		t.Error("Expected empty history after clear")
	}
}

// TestInputComponentHistoryNavigation 测试历史记录导航
func TestInputComponentHistoryNavigation(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 添加历史记录
	inputs := []string{"input1", "input2", "input3"}
	for _, input := range inputs {
		ic.AddToHistory(input)
	}

	// 设置当前输入
	currentInput := "current"
	ic.SetValue(currentInput)

	// 向上导航（更早的历史）
	ic.navigateHistory(-1)
	if ic.GetValue() != inputs[2] {
		t.Errorf("Expected value %q, got %q", inputs[2], ic.GetValue())
	}

	// 继续向上
	ic.navigateHistory(-1)
	if ic.GetValue() != inputs[1] {
		t.Errorf("Expected value %q, got %q", inputs[1], ic.GetValue())
	}

	// 向下导航（更新的历史）
	ic.navigateHistory(1)
	if ic.GetValue() != inputs[2] {
		t.Errorf("Expected value %q, got %q", inputs[2], ic.GetValue())
	}

	// 导航到末尾（恢复当前输入）
	ic.navigateHistory(1)
	if ic.GetValue() != currentInput {
		t.Errorf("Expected value %q, got %q", currentInput, ic.GetValue())
	}
}

// TestInputComponentHistoryLimit 测试历史记录限制
func TestInputComponentHistoryLimit(t *testing.T) {
	config := DefaultInputComponentConfig()
	config.MaxHistory = 5
	ic := NewInputComponent(config)

	// 添加超过限制的历史记录
	for i := 0; i < 10; i++ {
		ic.AddToHistory("input")
	}

	// 检查历史记录数量
	history := ic.GetHistory()
	if len(history) != config.MaxHistory {
		t.Errorf("Expected %d history items, got %d", config.MaxHistory, len(history))
	}
}

// TestInputComponentCommandParsing 测试命令解析
func TestInputComponentCommandParsing(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	tests := []struct {
		input         string
		expectedType  CommandType
		expectedName  string
		expectedArgs  []string
		expectedIsCmd bool
	}{
		{
			input:         "/cd /path/to/dir",
			expectedType:  CommandCD,
			expectedName:  "cd",
			expectedArgs:  []string{"/path/to/dir"},
			expectedIsCmd: true,
		},
		{
			input:         "/conversations",
			expectedType:  CommandConversations,
			expectedName:  "conversations",
			expectedArgs:  []string{},
			expectedIsCmd: true,
		},
		{
			input:         "/conv",
			expectedType:  CommandConversations,
			expectedName:  "conversations",
			expectedArgs:  []string{},
			expectedIsCmd: true,
		},
		{
			input:         "/clear",
			expectedType:  CommandClear,
			expectedName:  "clear",
			expectedArgs:  []string{},
			expectedIsCmd: true,
		},
		{
			input:         "/save conversation.yaml",
			expectedType:  CommandSave,
			expectedName:  "save",
			expectedArgs:  []string{"conversation.yaml"},
			expectedIsCmd: true,
		},
		{
			input:         "/help",
			expectedType:  CommandHelp,
			expectedName:  "help",
			expectedArgs:  []string{},
			expectedIsCmd: true,
		},
		{
			input:         "/exit",
			expectedType:  CommandExit,
			expectedName:  "exit",
			expectedArgs:  []string{},
			expectedIsCmd: true,
		},
		{
			input:         "/quit",
			expectedType:  CommandExit,
			expectedName:  "exit",
			expectedArgs:  []string{},
			expectedIsCmd: true,
		},
		{
			input:         "/q",
			expectedType:  CommandExit,
			expectedName:  "exit",
			expectedArgs:  []string{},
			expectedIsCmd: true,
		},
		{
			input:         "not a command",
			expectedType:  CommandNone,
			expectedName:  "",
			expectedArgs:  nil,
			expectedIsCmd: false,
		},
		{
			input:         "/unknown",
			expectedType:  CommandNone,
			expectedName:  "",
			expectedArgs:  nil,
			expectedIsCmd: true,
		},
	}

	for _, test := range tests {
		cmd := ic.ParseCommand(test.input)

		if cmd.Type != test.expectedType {
			t.Errorf("Input %q: expected type %v, got %v", test.input, test.expectedType, cmd.Type)
		}

		if cmd.Name != test.expectedName {
			t.Errorf("Input %q: expected name %q, got %q", test.input, test.expectedName, cmd.Name)
		}

		if len(cmd.Args) != len(test.expectedArgs) {
			t.Errorf("Input %q: expected %d args, got %d", test.input, len(test.expectedArgs), len(cmd.Args))
		} else {
			for i, arg := range test.expectedArgs {
				if cmd.Args[i] != arg {
					t.Errorf("Input %q: expected arg[%d] = %q, got %q", test.input, i, arg, cmd.Args[i])
				}
			}
		}

		if ic.IsCommand(test.input) != test.expectedIsCmd {
			t.Errorf("Input %q: expected IsCommand = %v, got %v", test.input, test.expectedIsCmd, ic.IsCommand(test.input))
		}
	}
}

// TestInputComponentCommandParsingWithSpaces 测试带空格的命令解析
func TestInputComponentCommandParsingWithSpaces(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 测试带空格的输入
	input := "  /cd   /path/to/dir  "
	cmd := ic.ParseCommand(input)

	if cmd.Type != CommandCD {
		t.Errorf("Expected type %v, got %v", CommandCD, cmd.Type)
	}

	if cmd.Name != "cd" {
		t.Errorf("Expected name %q, got %q", "cd", cmd.Name)
	}

	if len(cmd.Args) != 1 || cmd.Args[0] != "/path/to/dir" {
		t.Errorf("Expected args [%q], got %v", "/path/to/dir", cmd.Args)
	}
}

// TestInputComponentShouldSubmit 测试提交判断
func TestInputComponentShouldSubmit(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 单行模式
	ic.SetMode(InputModeSingleLine)
	if !ic.ShouldSubmit("enter") {
		t.Error("Expected ShouldSubmit(enter) to be true in single-line mode")
	}
	if ic.ShouldSubmit("ctrl+enter") {
		t.Error("Expected ShouldSubmit(ctrl+enter) to be false in single-line mode")
	}

	// 多行模式
	ic.SetMode(InputModeMultiLine)
	if ic.ShouldSubmit("enter") {
		t.Error("Expected ShouldSubmit(enter) to be false in multi-line mode")
	}
	if !ic.ShouldSubmit("ctrl+enter") {
		t.Error("Expected ShouldSubmit(ctrl+enter) to be true in multi-line mode")
	}
}

// TestInputComponentFocus 测试焦点管理
func TestInputComponentFocus(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 初始状态应该是聚焦的
	if !ic.focused {
		t.Error("Expected initial focus state to be true")
	}

	// 失去焦点
	ic.Blur()
	if ic.focused {
		t.Error("Expected focused to be false after Blur")
	}

	// 获取焦点
	ic.Focus()
	if !ic.focused {
		t.Error("Expected focused to be true after Focus")
	}
}

// TestInputComponentSize 测试尺寸设置
func TestInputComponentSize(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 测试设置宽度
	newWidth := 100
	ic.SetWidth(newWidth)
	if ic.width != newWidth {
		t.Errorf("Expected width %d, got %d", newWidth, ic.width)
	}

	// 测试设置高度
	newHeight := 5
	ic.SetHeight(newHeight)
	if ic.height != newHeight {
		t.Errorf("Expected height %d, got %d", newHeight, ic.height)
	}

	// 多行模式下设置高度应该影响 textarea
	ic.SetMode(InputModeMultiLine)
	ic.SetHeight(10)
	// 注意：这里无法直接测试 textarea 的高度，但可以确保不会崩溃
}

// TestInputComponentView 测试视图渲染
func TestInputComponentView(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 渲染视图
	view := ic.View()
	if view == "" {
		t.Error("Expected non-empty view")
	}

	// 检查模式提示
	if !strings.Contains(view, "单行模式") {
		t.Error("Expected view to contain '单行模式'")
	}

	// 切换到多行模式
	ic.SetMode(InputModeMultiLine)
	view = ic.View()
	if !strings.Contains(view, "多行模式") {
		t.Error("Expected view to contain '多行模式'")
	}
}

// TestInputComponentString 测试字符串表示
func TestInputComponentString(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	str := ic.String()
	if str == "" {
		t.Error("Expected non-empty string representation")
	}

	// 添加历史记录
	ic.AddToHistory("test1")
	ic.AddToHistory("test2")

	str = ic.String()
	if !strings.Contains(str, "2 items") {
		t.Errorf("Expected string to contain '2 items', got %q", str)
	}
}

// TestInputComponentGetCommandHelp 测试命令帮助
func TestInputComponentGetCommandHelp(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	help := ic.GetCommandHelp()
	if help == "" {
		t.Error("Expected non-empty help text")
	}

	// 检查是否包含所有命令
	commands := []string{"/cd", "/conversations", "/clear", "/save", "/help", "/exit"}
	for _, cmd := range commands {
		if !strings.Contains(help, cmd) {
			t.Errorf("Expected help to contain %q", cmd)
		}
	}
}

// TestInputComponentConfig 测试配置
func TestInputComponentConfig(t *testing.T) {
	config := InputComponentConfig{
		Width:       120,
		Height:      5,
		Placeholder: "Custom placeholder",
		Prompt:      "$",
		MaxHistory:  50,
		InitialMode: InputModeMultiLine,
	}

	ic := NewInputComponent(config)

	if ic.width != config.Width {
		t.Errorf("Expected width %d, got %d", config.Width, ic.width)
	}

	if ic.height != config.Height {
		t.Errorf("Expected height %d, got %d", config.Height, ic.height)
	}

	if ic.placeholder != config.Placeholder {
		t.Errorf("Expected placeholder %q, got %q", config.Placeholder, ic.placeholder)
	}

	if ic.prompt != config.Prompt {
		t.Errorf("Expected prompt %q, got %q", config.Prompt, ic.prompt)
	}

	if ic.maxHistory != config.MaxHistory {
		t.Errorf("Expected maxHistory %d, got %d", config.MaxHistory, ic.maxHistory)
	}

	if ic.mode != config.InitialMode {
		t.Errorf("Expected mode %v, got %v", config.InitialMode, ic.mode)
	}
}
