// Package components 提供 TUI 组件
package components

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
)

// InputMode 输入模式类型
type InputMode int

const (
	// InputModeSingleLine 单行输入模式
	InputModeSingleLine InputMode = iota
	// InputModeMultiLine 多行输入模式
	InputModeMultiLine
)

// CommandType 命令类型
type CommandType int

const (
	// CommandNone 无命令
	CommandNone CommandType = iota
	// CommandCD 切换目录命令
	CommandCD
	// CommandConversations 显示对话列表
	CommandConversations
	// CommandClear 清空对话
	CommandClear
	// CommandSave 保存对话
	CommandSave
	// CommandHelp 显示帮助
	CommandHelp
	// CommandExit 退出程序
	CommandExit
)

// Command 表示解析后的命令
type Command struct {
	// Type 命令类型
	Type CommandType
	// Name 命令名称
	Name string
	// Args 命令参数
	Args []string
	// Raw 原始输入
	Raw string
}

// InputComponent 输入组件
// 支持单行/多行输入模式切换、输入历史记录和命令解析
type InputComponent struct {
	// textarea 文本输入区域
	textarea textarea.Model
	// mode 当前输入模式
	mode InputMode
	// width 宽度
	width int
	// height 高度（多行模式）
	height int
	// focused 是否聚焦
	focused bool

	// history 输入历史记录
	history []string
	// historyIndex 当前历史索引（-1 表示不在浏览历史）
	historyIndex int
	// maxHistory 最大历史记录数
	maxHistory int
	// tempInput 临时输入（浏览历史时保存当前输入）
	tempInput string

	// placeholder 占位符文本
	placeholder string
	// prompt 提示符
	prompt string

	// 样式
	style          lipgloss.Style
	focusedStyle   lipgloss.Style
	modeStyle      lipgloss.Style
	promptStyle    lipgloss.Style
	placeholderStyle lipgloss.Style
}

// InputComponentConfig 输入组件配置
type InputComponentConfig struct {
	// Width 宽度
	Width int
	// Height 高度（多行模式）
	Height int
	// Placeholder 占位符
	Placeholder string
	// Prompt 提示符
	Prompt string
	// MaxHistory 最大历史记录数
	MaxHistory int
	// InitialMode 初始输入模式
	InitialMode InputMode
}

// DefaultInputComponentConfig 返回默认配置
func DefaultInputComponentConfig() InputComponentConfig {
	return InputComponentConfig{
		Width:       80,
		Height:      3,
		Placeholder: "请输入你的问题...",
		Prompt:      ">",
		MaxHistory:  100,
		InitialMode: InputModeSingleLine,
	}
}

// NewInputComponent 创建新的输入组件
func NewInputComponent(config InputComponentConfig) *InputComponent {
	// 创建 textarea
	ta := textarea.New()
	ta.SetWidth(config.Width)
	ta.SetHeight(config.Height)
	ta.Placeholder = config.Placeholder
	ta.ShowLineNumbers = false
	ta.Prompt = config.Prompt + " "

	// 设置初始模式
	if config.InitialMode == InputModeSingleLine {
		ta.SetHeight(1)
	}

	// 初始化样式
	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		Padding(0, 1)

	focusedStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(0, 1)

	modeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)

	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true)

	placeholderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF"))

	return &InputComponent{
		textarea:        ta,
		mode:            config.InitialMode,
		width:           config.Width,
		height:          config.Height,
		focused:         true,
		history:         make([]string, 0),
		historyIndex:    -1,
		maxHistory:      config.MaxHistory,
		tempInput:       "",
		placeholder:     config.Placeholder,
		prompt:          config.Prompt,
		style:           style,
		focusedStyle:    focusedStyle,
		modeStyle:       modeStyle,
		promptStyle:     promptStyle,
		placeholderStyle: placeholderStyle,
	}
}

// Init 初始化组件
func (ic *InputComponent) Init() tea.Cmd {
	return textarea.Blink
}

// Update 更新组件状态
func (ic *InputComponent) Update(msg tea.Msg) (*InputComponent, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			// 切换输入模式
			ic.ToggleMode()
			return ic, nil

		case "enter":
			// 单行模式：直接提交
			// 多行模式：需要 Ctrl+Enter
			if ic.mode == InputModeSingleLine {
				return ic, nil // 由外部处理提交
			}
			// 多行模式下，Enter 换行
			var cmd tea.Cmd
			ic.textarea, cmd = ic.textarea.Update(msg)
			cmds = append(cmds, cmd)
			return ic, tea.Batch(cmds...)

		case "ctrl+enter":
			// 多行模式：提交
			if ic.mode == InputModeMultiLine {
				return ic, nil // 由外部处理提交
			}

		case "up":
			// 上箭头：浏览历史记录（仅在行首时）
			if ic.mode == InputModeSingleLine || ic.textarea.Line() == 0 {
				ic.navigateHistory(-1)
				return ic, nil
			}

		case "down":
			// 下箭头：浏览历史记录（仅在行首时）
			if ic.mode == InputModeSingleLine || ic.textarea.Line() == ic.textarea.LineCount()-1 {
				ic.navigateHistory(1)
				return ic, nil
			}

		case "ctrl+l":
			// 显示对话列表（由外部处理）
			return ic, nil
		}
	}

	// 更新 textarea
	var cmd tea.Cmd
	ic.textarea, cmd = ic.textarea.Update(msg)
	cmds = append(cmds, cmd)

	return ic, tea.Batch(cmds...)
}

// View 渲染组件
func (ic *InputComponent) View() string {
	var builder strings.Builder

	// 渲染模式提示
	modeText := "单行模式"
	if ic.mode == InputModeMultiLine {
		modeText = "多行模式"
	}
	modeHint := ic.modeStyle.Render("[" + modeText + "]")
	builder.WriteString(modeHint)
	builder.WriteString("\n")

	// 渲染输入框
	var inputView string
	if ic.focused {
		inputView = ic.focusedStyle.Render(ic.textarea.View())
	} else {
		inputView = ic.style.Render(ic.textarea.View())
	}
	builder.WriteString(inputView)
	builder.WriteString("\n")

	// 渲染帮助提示
	helpText := ic.renderHelpText()
	builder.WriteString(helpText)

	return builder.String()
}

// renderHelpText 渲染帮助文本
func (ic *InputComponent) renderHelpText() string {
	var hints []string

	if ic.mode == InputModeSingleLine {
		hints = append(hints, "Tab 切换多行")
		hints = append(hints, "Enter 提交")
		hints = append(hints, "↑↓ 浏览历史")
	} else {
		hints = append(hints, "Tab 切换单行")
		hints = append(hints, "Ctrl+Enter 提交")
		hints = append(hints, "Enter 换行")
	}

	hints = append(hints, "Ctrl+L 对话列表")
	hints = append(hints, "Ctrl+C 退出")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)

	return helpStyle.Render(strings.Join(hints, "  |  "))
}

// ToggleMode 切换输入模式
func (ic *InputComponent) ToggleMode() {
	if ic.mode == InputModeSingleLine {
		ic.mode = InputModeMultiLine
		ic.textarea.SetHeight(ic.height)
	} else {
		ic.mode = InputModeSingleLine
		ic.textarea.SetHeight(1)
	}
}

// SetMode 设置输入模式
func (ic *InputComponent) SetMode(mode InputMode) {
	if ic.mode != mode {
		ic.mode = mode
		if mode == InputModeSingleLine {
			ic.textarea.SetHeight(1)
		} else {
			ic.textarea.SetHeight(ic.height)
		}
	}
}

// GetMode 获取当前输入模式
func (ic *InputComponent) GetMode() InputMode {
	return ic.mode
}

// GetValue 获取输入内容
func (ic *InputComponent) GetValue() string {
	return ic.textarea.Value()
}

// SetValue 设置输入内容
func (ic *InputComponent) SetValue(value string) {
	ic.textarea.SetValue(value)
	ic.historyIndex = -1
	ic.tempInput = ""
}

// Clear 清空输入
func (ic *InputComponent) Clear() {
	ic.textarea.SetValue("")
	ic.historyIndex = -1
	ic.tempInput = ""
}

// Focus 获取焦点
func (ic *InputComponent) Focus() tea.Cmd {
	ic.focused = true
	return ic.textarea.Focus()
}

// Blur 失去焦点
func (ic *InputComponent) Blur() {
	ic.focused = false
	ic.textarea.Blur()
}

// SetWidth 设置宽度
func (ic *InputComponent) SetWidth(width int) {
	ic.width = width
	ic.textarea.SetWidth(width)
}

// SetHeight 设置高度
func (ic *InputComponent) SetHeight(height int) {
	ic.height = height
	if ic.mode == InputModeMultiLine {
		ic.textarea.SetHeight(height)
	}
}

// navigateHistory 导航历史记录
// direction: -1 向上（更早），1 向下（更新）
func (ic *InputComponent) navigateHistory(direction int) {
	if len(ic.history) == 0 {
		return
	}

	// 第一次向上导航时，保存当前输入
	if direction == -1 && ic.historyIndex == -1 {
		ic.tempInput = ic.textarea.Value()
	}

	// 计算新索引
	newIndex := ic.historyIndex + direction

	// 边界检查
	if newIndex < -1 {
		newIndex = -1
	} else if newIndex >= len(ic.history) {
		newIndex = len(ic.history) - 1
	}

	ic.historyIndex = newIndex

	// 更新输入框内容
	if newIndex == -1 {
		// 恢复临时输入
		ic.textarea.SetValue(ic.tempInput)
	} else {
		// 显示历史记录
		ic.textarea.SetValue(ic.history[newIndex])
	}
}

// AddToHistory 添加到历史记录
func (ic *InputComponent) AddToHistory(input string) {
	// 忽略空输入
	if strings.TrimSpace(input) == "" {
		return
	}

	// 避免重复添加相同的连续输入
	if len(ic.history) > 0 && ic.history[len(ic.history)-1] == input {
		return
	}

	// 添加到历史
	ic.history = append(ic.history, input)

	// 限制历史记录数量
	if len(ic.history) > ic.maxHistory {
		ic.history = ic.history[1:]
	}

	// 重置历史索引
	ic.historyIndex = -1
	ic.tempInput = ""
}

// GetHistory 获取历史记录
func (ic *InputComponent) GetHistory() []string {
	return ic.history
}

// ClearHistory 清空历史记录
func (ic *InputComponent) ClearHistory() {
	ic.history = make([]string, 0)
	ic.historyIndex = -1
	ic.tempInput = ""
}

// ParseCommand 解析命令
// 如果输入以 / 开头，则解析为命令
func (ic *InputComponent) ParseCommand(input string) Command {
	input = strings.TrimSpace(input)

	// 不是命令
	if !strings.HasPrefix(input, "/") {
		return Command{
			Type: CommandNone,
			Raw:  input,
		}
	}

	// 解析命令
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return Command{
			Type: CommandNone,
			Raw:  input,
		}
	}

	cmdName := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmdName {
	case "/cd":
		return Command{
			Type: CommandCD,
			Name: "cd",
			Args: args,
			Raw:  input,
		}
	case "/conversations", "/conv":
		return Command{
			Type: CommandConversations,
			Name: "conversations",
			Args: args,
			Raw:  input,
		}
	case "/clear":
		return Command{
			Type: CommandClear,
			Name: "clear",
			Args: args,
			Raw:  input,
		}
	case "/save":
		return Command{
			Type: CommandSave,
			Name: "save",
			Args: args,
			Raw:  input,
		}
	case "/help":
		return Command{
			Type: CommandHelp,
			Name: "help",
			Args: args,
			Raw:  input,
		}
	case "/exit", "/quit", "/q":
		return Command{
			Type: CommandExit,
			Name: "exit",
			Args: args,
			Raw:  input,
		}
	default:
		// 未知命令，返回原始输入
		return Command{
			Type: CommandNone,
			Raw:  input,
		}
	}
}

// IsCommand 检查输入是否为命令
func (ic *InputComponent) IsCommand(input string) bool {
	return strings.HasPrefix(strings.TrimSpace(input), "/")
}

// GetCommandHelp 获取命令帮助文本
func (ic *InputComponent) GetCommandHelp() string {
	help := []struct {
		cmd  string
		desc string
	}{
		{"/cd <path>", "切换工作目录"},
		{"/conversations", "显示对话列表"},
		{"/clear", "清空当前对话"},
		{"/save [file]", "保存对话历史"},
		{"/help", "显示帮助信息"},
		{"/exit", "退出 TUI"},
	}

	var lines []string
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true)

	cmdStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B82F6"))

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	lines = append(lines, titleStyle.Render("内置命令:"))
	for _, h := range help {
		cmd := cmdStyle.Render(h.cmd)
		desc := descStyle.Render(" - " + h.desc)
		lines = append(lines, "  "+cmd+desc)
	}

	return strings.Join(lines, "\n")
}

// ShouldSubmit 检查是否应该提交输入
// 根据当前模式和按键判断
func (ic *InputComponent) ShouldSubmit(key string) bool {
	if ic.mode == InputModeSingleLine {
		return key == "enter"
	}
	return key == "ctrl+enter"
}

// String 返回输入组件的字符串表示
func (ic *InputComponent) String() string {
	return fmt.Sprintf("InputComponent{mode: %v, history: %d items}", ic.mode, len(ic.history))
}
