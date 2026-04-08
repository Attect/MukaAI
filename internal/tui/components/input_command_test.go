// Package components 提供 TUI 组件
package components

import (
	"strings"
	"testing"
)

// TestParseCommand 测试命令解析
func TestParseCommand(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	tests := []struct {
		name         string
		input        string
		expectedType CommandType
		expectedName string
		expectedArgs []string
	}{
		{
			name:         "cd command",
			input:        "/cd /new/path",
			expectedType: CommandCD,
			expectedName: "cd",
			expectedArgs: []string{"/new/path"},
		},
		{
			name:         "cd command with spaces",
			input:        "/cd /path/with spaces",
			expectedType: CommandCD,
			expectedName: "cd",
			expectedArgs: []string{"/path/with", "spaces"},
		},
		{
			name:         "conversations command",
			input:        "/conversations",
			expectedType: CommandConversations,
			expectedName: "conversations",
			expectedArgs: []string{},
		},
		{
			name:         "conv alias",
			input:        "/conv",
			expectedType: CommandConversations,
			expectedName: "conversations",
			expectedArgs: []string{},
		},
		{
			name:         "clear command",
			input:        "/clear",
			expectedType: CommandClear,
			expectedName: "clear",
			expectedArgs: []string{},
		},
		{
			name:         "save command",
			input:        "/save /path/to/file.json",
			expectedType: CommandSave,
			expectedName: "save",
			expectedArgs: []string{"/path/to/file.json"},
		},
		{
			name:         "save command without args",
			input:        "/save",
			expectedType: CommandSave,
			expectedName: "save",
			expectedArgs: []string{},
		},
		{
			name:         "help command",
			input:        "/help",
			expectedType: CommandHelp,
			expectedName: "help",
			expectedArgs: []string{},
		},
		{
			name:         "exit command",
			input:        "/exit",
			expectedType: CommandExit,
			expectedName: "exit",
			expectedArgs: []string{},
		},
		{
			name:         "quit alias",
			input:        "/quit",
			expectedType: CommandExit,
			expectedName: "exit",
			expectedArgs: []string{},
		},
		{
			name:         "q alias",
			input:        "/q",
			expectedType: CommandExit,
			expectedName: "exit",
			expectedArgs: []string{},
		},
		{
			name:         "unknown command",
			input:        "/unknown",
			expectedType: CommandNone,
			expectedName: "",
			expectedArgs: nil,
		},
		{
			name:         "not a command",
			input:        "hello world",
			expectedType: CommandNone,
			expectedName: "",
			expectedArgs: nil,
		},
		{
			name:         "empty input",
			input:        "",
			expectedType: CommandNone,
			expectedName: "",
			expectedArgs: nil,
		},
		{
			name:         "whitespace input",
			input:        "   ",
			expectedType: CommandNone,
			expectedName: "",
			expectedArgs: nil,
		},
		{
			name:         "command with extra spaces",
			input:        "/cd   /path/to/dir   ",
			expectedType: CommandCD,
			expectedName: "cd",
			expectedArgs: []string{"/path/to/dir"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ic.ParseCommand(tt.input)

			if cmd.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, cmd.Type)
			}

			if cmd.Name != tt.expectedName {
				t.Errorf("Expected name %q, got %q", tt.expectedName, cmd.Name)
			}

			if len(cmd.Args) != len(tt.expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(tt.expectedArgs), len(cmd.Args))
			} else {
				for i, arg := range tt.expectedArgs {
					if cmd.Args[i] != arg {
						t.Errorf("Expected arg[%d] = %q, got %q", i, arg, cmd.Args[i])
					}
				}
			}
		})
	}
}

// TestIsCommand 测试命令识别
func TestIsCommand(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	tests := []struct {
		input    string
		expected bool
	}{
		{"/cd /path", true},
		{"/help", true},
		{"hello world", false},
		{"", false},
		{"  /cd  ", true},
		{"not a command", false},
		{"/unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ic.IsCommand(tt.input)
			if result != tt.expected {
				t.Errorf("IsCommand(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestGetCommandHelp 测试命令帮助文本
func TestGetCommandHelp(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	help := ic.GetCommandHelp()

	// 检查帮助文本包含所有命令
	expectedCommands := []string{
		"/cd",
		"/conversations",
		"/clear",
		"/save",
		"/help",
		"/exit",
	}

	for _, cmd := range expectedCommands {
		if !strings.Contains(help, cmd) {
			t.Errorf("Command help should contain %q", cmd)
		}
	}

	// 检查帮助文本非空
	if help == "" {
		t.Error("Command help should not be empty")
	}
}

// TestShouldSubmit 测试提交判断
func TestShouldSubmit(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 测试单行模式
	ic.SetMode(InputModeSingleLine)
	if !ic.ShouldSubmit("enter") {
		t.Error("Should submit on Enter in single-line mode")
	}
	if ic.ShouldSubmit("ctrl+enter") {
		t.Error("Should not submit on Ctrl+Enter in single-line mode")
	}

	// 测试多行模式
	ic.SetMode(InputModeMultiLine)
	if ic.ShouldSubmit("enter") {
		t.Error("Should not submit on Enter in multi-line mode")
	}
	if !ic.ShouldSubmit("ctrl+enter") {
		t.Error("Should submit on Ctrl+Enter in multi-line mode")
	}
}

// TestInputModeToggle 测试输入模式切换
func TestInputModeToggle(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 初始为单行模式
	if ic.GetMode() != InputModeSingleLine {
		t.Error("Initial mode should be single-line")
	}

	// 切换到多行模式
	ic.ToggleMode()
	if ic.GetMode() != InputModeMultiLine {
		t.Error("Mode should be multi-line after toggle")
	}

	// 切换回单行模式
	ic.ToggleMode()
	if ic.GetMode() != InputModeSingleLine {
		t.Error("Mode should be single-line after second toggle")
	}
}

// TestInputModeSet 测试设置输入模式
func TestInputModeSet(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 设置为多行模式
	ic.SetMode(InputModeMultiLine)
	if ic.GetMode() != InputModeMultiLine {
		t.Error("Mode should be multi-line")
	}

	// 再次设置为多行模式（无变化）
	ic.SetMode(InputModeMultiLine)
	if ic.GetMode() != InputModeMultiLine {
		t.Error("Mode should still be multi-line")
	}

	// 设置为单行模式
	ic.SetMode(InputModeSingleLine)
	if ic.GetMode() != InputModeSingleLine {
		t.Error("Mode should be single-line")
	}
}

// TestInputValueOperations 测试输入值操作
func TestInputValueOperations(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 设置值
	ic.SetValue("test input")
	if ic.GetValue() != "test input" {
		t.Errorf("Expected value 'test input', got %q", ic.GetValue())
	}

	// 清空值
	ic.Clear()
	if ic.GetValue() != "" {
		t.Errorf("Expected empty value, got %q", ic.GetValue())
	}
}

// TestInputHistory 测试输入历史
func TestInputHistory(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 添加历史记录
	ic.AddToHistory("input 1")
	ic.AddToHistory("input 2")
	ic.AddToHistory("input 3")

	history := ic.GetHistory()
	if len(history) != 3 {
		t.Errorf("Expected 3 history items, got %d", len(history))
	}

	// 检查历史记录顺序
	expected := []string{"input 1", "input 2", "input 3"}
	for i, h := range expected {
		if history[i] != h {
			t.Errorf("Expected history[%d] = %q, got %q", i, h, history[i])
		}
	}

	// 清空历史
	ic.ClearHistory()
	if len(ic.GetHistory()) != 0 {
		t.Error("History should be empty after clear")
	}
}

// TestInputHistoryDuplicate 测试历史记录去重
func TestInputHistoryDuplicate(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 添加相同输入
	ic.AddToHistory("input")
	ic.AddToHistory("input")
	ic.AddToHistory("input")

	history := ic.GetHistory()
	if len(history) != 1 {
		t.Errorf("Expected 1 history item (deduplicated), got %d", len(history))
	}
}

// TestInputHistoryEmpty 测试空输入历史
func TestInputHistoryEmpty(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	// 添加空输入
	ic.AddToHistory("")
	ic.AddToHistory("   ")
	ic.AddToHistory("\t")

	history := ic.GetHistory()
	if len(history) != 0 {
		t.Errorf("Expected 0 history items (empty inputs ignored), got %d", len(history))
	}
}

// TestInputHistoryLimit 测试历史记录限制
func TestInputHistoryLimit(t *testing.T) {
	config := DefaultInputComponentConfig()
	config.MaxHistory = 5
	ic := NewInputComponent(config)

	// 添加超过限制的历史记录
	for i := 0; i < 10; i++ {
		ic.AddToHistory("input")
	}

	history := ic.GetHistory()
	if len(history) > 5 {
		t.Errorf("Expected at most 5 history items, got %d", len(history))
	}
}

// TestInputComponentStringRepresentation 测试字符串表示
func TestInputComponentStringRepresentation(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())
	ic.AddToHistory("test")

	str := ic.String()
	if !strings.Contains(str, "InputComponent") {
		t.Error("String representation should contain 'InputComponent'")
	}
	if !strings.Contains(str, "history: 1 items") {
		t.Error("String representation should contain history count")
	}
}

// TestCommandRawField 测试命令原始字段
func TestCommandRawField(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	input := "/cd /path/to/dir  "
	cmd := ic.ParseCommand(input)

	// 原始字段应该包含完整的输入（包括空格）
	if cmd.Raw != strings.TrimSpace(input) {
		t.Errorf("Expected raw %q, got %q", strings.TrimSpace(input), cmd.Raw)
	}
}

// TestCommandTypeConstants 测试命令类型常量
func TestCommandTypeConstants(t *testing.T) {
	// 确保命令类型常量正确
	if CommandNone != 0 {
		t.Error("CommandNone should be 0")
	}
	if CommandCD != 1 {
		t.Error("CommandCD should be 1")
	}
	if CommandConversations != 2 {
		t.Error("CommandConversations should be 2")
	}
	if CommandClear != 3 {
		t.Error("CommandClear should be 3")
	}
	if CommandSave != 4 {
		t.Error("CommandSave should be 4")
	}
	if CommandHelp != 5 {
		t.Error("CommandHelp should be 5")
	}
	if CommandExit != 6 {
		t.Error("CommandExit should be 6")
	}
}

// TestInputModeConstants 测试输入模式常量
func TestInputModeConstants(t *testing.T) {
	if InputModeSingleLine != 0 {
		t.Error("InputModeSingleLine should be 0")
	}
	if InputModeMultiLine != 1 {
		t.Error("InputModeMultiLine should be 1")
	}
}

// TestCommandParsingEdgeCases 测试命令解析边界情况
func TestCommandParsingEdgeCases(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	tests := []struct {
		name  string
		input string
	}{
		{"only slash", "/"},
		{"slash with spaces", "/   "},
		{"multiple slashes", "///"},
		{"mixed case command", "/CD"},
		{"command with tab", "/cd\t/path"},
		{"command with newline", "/cd /path\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 确保不会崩溃
			cmd := ic.ParseCommand(tt.input)
			_ = cmd // 只要不崩溃就算通过
		})
	}
}

// TestInputComponentConfigCustom 测试输入组件配置
func TestInputComponentConfigCustom(t *testing.T) {
	config := InputComponentConfig{
		Width:       100,
		Height:      5,
		Placeholder: "自定义占位符",
		Prompt:      "$",
		MaxHistory:  50,
		InitialMode: InputModeMultiLine,
	}

	ic := NewInputComponent(config)

	if ic.GetMode() != InputModeMultiLine {
		t.Error("Initial mode should be multi-line")
	}

	// 检查宽度设置
	if ic.width != 100 {
		t.Errorf("Expected width 100, got %d", ic.width)
	}

	// 检查高度设置
	if ic.height != 5 {
		t.Errorf("Expected height 5, got %d", ic.height)
	}

	// 检查最大历史记录
	if ic.maxHistory != 50 {
		t.Errorf("Expected maxHistory 50, got %d", ic.maxHistory)
	}
}

// TestDefaultInputComponentConfig 测试默认配置
func TestDefaultInputComponentConfig(t *testing.T) {
	config := DefaultInputComponentConfig()

	if config.Width != 80 {
		t.Errorf("Expected default width 80, got %d", config.Width)
	}
	if config.Height != 3 {
		t.Errorf("Expected default height 3, got %d", config.Height)
	}
	if config.Placeholder != "请输入你的问题..." {
		t.Errorf("Expected default placeholder, got %q", config.Placeholder)
	}
	if config.Prompt != ">" {
		t.Errorf("Expected default prompt '>', got %q", config.Prompt)
	}
	if config.MaxHistory != 100 {
		t.Errorf("Expected default maxHistory 100, got %d", config.MaxHistory)
	}
	if config.InitialMode != InputModeSingleLine {
		t.Errorf("Expected default initial mode single-line, got %v", config.InitialMode)
	}
}

// TestCommandWithMultipleArgs 测试多参数命令
func TestCommandWithMultipleArgs(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	cmd := ic.ParseCommand("/cd /path1 /path2 /path3")

	if cmd.Type != CommandCD {
		t.Errorf("Expected type CommandCD, got %v", cmd.Type)
	}
	if len(cmd.Args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(cmd.Args))
	}
	expectedArgs := []string{"/path1", "/path2", "/path3"}
	for i, arg := range expectedArgs {
		if cmd.Args[i] != arg {
			t.Errorf("Expected arg[%d] = %q, got %q", i, arg, cmd.Args[i])
		}
	}
}

// TestCommandCaseInsensitive 测试命令大小写不敏感
func TestCommandCaseInsensitive(t *testing.T) {
	ic := NewInputComponent(DefaultInputComponentConfig())

	tests := []struct {
		input    string
		expected CommandType
	}{
		{"/CD", CommandCD},
		{"/Cd", CommandCD},
		{"/HELP", CommandHelp},
		{"/Help", CommandHelp},
		{"/EXIT", CommandExit},
		{"/Exit", CommandExit},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := ic.ParseCommand(tt.input)
			if cmd.Type != tt.expected {
				t.Errorf("Expected type %v for input %q, got %v", tt.expected, tt.input, cmd.Type)
			}
		})
	}
}
