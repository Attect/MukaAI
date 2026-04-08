// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme 主题配置
// 定义完整的颜色主题和样式配置
type Theme struct {
	// 基础颜色
	Primary    color.Color
	Secondary  color.Color
	Success    color.Color
	Warning    color.Color
	Error      color.Color
	Info       color.Color
	Muted      color.Color
	Background color.Color
	Border     color.Color

	// 消息类型颜色
	UserMessage color.Color
	Thinking    color.Color
	Content     color.Color
	ToolCall    color.Color
	ToolResult  color.Color
	ToolError   color.Color

	// 状态颜色
	Active   color.Color
	Waiting  color.Color
	Finished color.Color

	// 布局配置
	Layout LayoutConfig
}

// LayoutConfig 布局配置
// 定义边框、间距、对齐等布局属性
type LayoutConfig struct {
	// 边框配置
	BorderWidth  int
	BorderRadius int
	BorderStyle  lipgloss.Border

	// 间距配置
	PaddingHorizontal int
	PaddingVertical   int
	MarginHorizontal  int
	MarginVertical    int

	// 对齐配置
	HorizontalAlign lipgloss.Position
	VerticalAlign   lipgloss.Position
}

// DefaultTheme 返回默认主题配置
func DefaultTheme() *Theme {
	return &Theme{
		// 基础颜色
		Primary:    lipgloss.Color("#7C3AED"), // 紫色 - 主色调
		Secondary:  lipgloss.Color("#3B82F6"), // 蓝色 - 次要色调
		Success:    lipgloss.Color("#10B981"), // 绿色 - 成功
		Warning:    lipgloss.Color("#F59E0B"), // 黄色 - 警告
		Error:      lipgloss.Color("#EF4444"), // 红色 - 错误
		Info:       lipgloss.Color("#6B7280"), // 灰色 - 信息
		Muted:      lipgloss.Color("#9CA3AF"), // 浅灰色 - 弱化文本
		Background: lipgloss.Color("#1F2937"), // 深灰色 - 背景
		Border:     lipgloss.Color("#374151"), // 灰色 - 边框

		// 消息类型颜色
		UserMessage: lipgloss.Color("#3B82F6"), // 蓝色 - 用户消息
		Thinking:    lipgloss.Color("#6B7280"), // 灰色 - 思考内容
		Content:     lipgloss.Color("#F3F4F6"), // 浅灰色 - 正文内容
		ToolCall:    lipgloss.Color("#F59E0B"), // 黄色 - 工具调用
		ToolResult:  lipgloss.Color("#10B981"), // 绿色 - 工具结果
		ToolError:   lipgloss.Color("#EF4444"), // 红色 - 工具错误

		// 状态颜色
		Active:   lipgloss.Color("#10B981"), // 绿色 - 活动状态
		Waiting:  lipgloss.Color("#F59E0B"), // 黄色 - 等待状态
		Finished: lipgloss.Color("#6B7280"), // 灰色 - 结束状态

		// 布局配置
		Layout: LayoutConfig{
			BorderWidth:       1,
			BorderRadius:      0,
			BorderStyle:       lipgloss.NormalBorder(),
			PaddingHorizontal: 1,
			PaddingVertical:   0,
			MarginHorizontal:  0,
			MarginVertical:    0,
			HorizontalAlign:   lipgloss.Left,
			VerticalAlign:     lipgloss.Top,
		},
	}
}

// DarkTheme 返回深色主题
func DarkTheme() *Theme {
	theme := DefaultTheme()
	theme.Background = lipgloss.Color("#0F172A") // 更深的背景
	theme.Border = lipgloss.Color("#334155")     // 更深的边框
	theme.Content = lipgloss.Color("#E2E8F0")    // 更亮的文本
	return theme
}

// LightTheme 返回浅色主题
func LightTheme() *Theme {
	return &Theme{
		// 基础颜色
		Primary:    lipgloss.Color("#7C3AED"), // 紫色 - 主色调
		Secondary:  lipgloss.Color("#3B82F6"), // 蓝色 - 次要色调
		Success:    lipgloss.Color("#059669"), // 深绿色 - 成功
		Warning:    lipgloss.Color("#D97706"), // 深黄色 - 警告
		Error:      lipgloss.Color("#DC2626"), // 深红色 - 错误
		Info:       lipgloss.Color("#4B5563"), // 深灰色 - 信息
		Muted:      lipgloss.Color("#6B7280"), // 灰色 - 弱化文本
		Background: lipgloss.Color("#F9FAFB"), // 浅灰色 - 背景
		Border:     lipgloss.Color("#D1D5DB"), // 浅灰色 - 边框

		// 消息类型颜色
		UserMessage: lipgloss.Color("#2563EB"), // 深蓝色 - 用户消息
		Thinking:    lipgloss.Color("#6B7280"), // 灰色 - 思考内容
		Content:     lipgloss.Color("#1F2937"), // 深灰色 - 正文内容
		ToolCall:    lipgloss.Color("#D97706"), // 深黄色 - 工具调用
		ToolResult:  lipgloss.Color("#059669"), // 深绿色 - 工具结果
		ToolError:   lipgloss.Color("#DC2626"), // 深红色 - 工具错误

		// 状态颜色
		Active:   lipgloss.Color("#059669"), // 深绿色 - 活动状态
		Waiting:  lipgloss.Color("#D97706"), // 深黄色 - 等待状态
		Finished: lipgloss.Color("#6B7280"), // 灰色 - 结束状态

		// 布局配置
		Layout: LayoutConfig{
			BorderWidth:       1,
			BorderRadius:      0,
			BorderStyle:       lipgloss.NormalBorder(),
			PaddingHorizontal: 1,
			PaddingVertical:   0,
			MarginHorizontal:  0,
			MarginVertical:    0,
			HorizontalAlign:   lipgloss.Left,
			VerticalAlign:     lipgloss.Top,
		},
	}
}

// currentTheme 当前主题（默认为 DefaultTheme）
var currentTheme = DefaultTheme()

// SetTheme 设置当前主题
func SetTheme(theme *Theme) {
	if theme != nil {
		currentTheme = theme
		// 更新所有样式定义
		updateStylesFromTheme()
	}
}

// GetTheme 获取当前主题
func GetTheme() *Theme {
	return currentTheme
}

// updateStylesFromTheme 根据当前主题更新样式定义
func updateStylesFromTheme() {
	// 更新基础样式
	styleBase = lipgloss.NewStyle().
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal)

	styleTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(currentTheme.Primary).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal)

	styleStatusBar = lipgloss.NewStyle().
		Background(currentTheme.Background).
		Foreground(currentTheme.Content).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal).
		Bold(true)

	styleStatusItem = lipgloss.NewStyle().
		Foreground(currentTheme.Content).
		Padding(currentTheme.Layout.PaddingVertical, 2)

	styleInput = lipgloss.NewStyle().
		Border(currentTheme.Layout.BorderStyle).
		BorderForeground(currentTheme.Border).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal)

	styleInputFocused = lipgloss.NewStyle().
		Border(currentTheme.Layout.BorderStyle).
		BorderForeground(currentTheme.Primary).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal)

	styleChatArea = lipgloss.NewStyle().
		Border(currentTheme.Layout.BorderStyle).
		BorderForeground(currentTheme.Border).
		Padding(1, 2)

	// 更新消息样式
	styleUserMessage = lipgloss.NewStyle().
		Foreground(currentTheme.UserMessage).
		Bold(true).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal)

	styleUserContent = lipgloss.NewStyle().
		Foreground(currentTheme.Content).
		Padding(currentTheme.Layout.PaddingVertical, 2)

	styleThinking = lipgloss.NewStyle().
		Foreground(currentTheme.Thinking).
		Italic(true).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal)

	styleThinkingBox = lipgloss.NewStyle().
		Border(currentTheme.Layout.BorderStyle).
		BorderForeground(currentTheme.Muted).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal).
		MarginLeft(2)

	styleThinkingTitle = lipgloss.NewStyle().
		Foreground(currentTheme.Muted).
		Bold(true).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal)

	styleContent = lipgloss.NewStyle().
		Foreground(currentTheme.Content).
		Padding(currentTheme.Layout.PaddingVertical, 2)

	styleToolCall = lipgloss.NewStyle().
		Foreground(currentTheme.ToolCall).
		Bold(true).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal)

	styleToolCallBox = lipgloss.NewStyle().
		Border(currentTheme.Layout.BorderStyle).
		BorderForeground(currentTheme.ToolCall).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal).
		MarginLeft(2)

	styleToolCallTitle = lipgloss.NewStyle().
		Foreground(currentTheme.ToolCall).
		Bold(true).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal)

	styleToolArgs = lipgloss.NewStyle().
		Foreground(currentTheme.Muted).
		Padding(currentTheme.Layout.PaddingVertical, 2)

	styleToolResult = lipgloss.NewStyle().
		Foreground(currentTheme.ToolResult).
		Padding(currentTheme.Layout.PaddingVertical, 2)

	styleToolResultBox = lipgloss.NewStyle().
		Border(currentTheme.Layout.BorderStyle).
		BorderForeground(currentTheme.ToolResult).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal).
		MarginLeft(2)

	styleToolError = lipgloss.NewStyle().
		Foreground(currentTheme.ToolError).
		Padding(currentTheme.Layout.PaddingVertical, 2)

	styleTokenUsage = lipgloss.NewStyle().
		Foreground(currentTheme.Muted).
		Padding(currentTheme.Layout.PaddingVertical, 2).
		Italic(true)

	// 更新状态样式
	styleStatusActive = lipgloss.NewStyle().
		Foreground(currentTheme.Active).
		Bold(true)

	styleStatusWaiting = lipgloss.NewStyle().
		Foreground(currentTheme.Warning).
		Bold(true)

	styleStatusFinished = lipgloss.NewStyle().
		Foreground(currentTheme.Finished)

	// 更新对话列表样式
	styleConversationList = lipgloss.NewStyle().
		Border(currentTheme.Layout.BorderStyle).
		BorderForeground(currentTheme.Primary).
		Padding(1, 2)

	styleConversationListTitle = lipgloss.NewStyle().
		Foreground(currentTheme.Primary).
		Bold(true).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal).
		MarginBottom(1)

	styleConversationItem = lipgloss.NewStyle().
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal)

	styleConversationItemSelected = lipgloss.NewStyle().
		Background(currentTheme.Primary).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal)

	styleConversationTitle = lipgloss.NewStyle().
		Foreground(currentTheme.Content).
		Bold(true)

	styleConversationTime = lipgloss.NewStyle().
		Foreground(currentTheme.Muted)

	// 更新错误样式
	styleError = lipgloss.NewStyle().
		Foreground(currentTheme.Error).
		Bold(true).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal)

	styleErrorBox = lipgloss.NewStyle().
		Border(currentTheme.Layout.BorderStyle).
		BorderForeground(currentTheme.Error).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal).
		MarginLeft(2)

	// 更新帮助文本样式
	styleHelp = lipgloss.NewStyle().
		Foreground(currentTheme.Muted).
		Padding(currentTheme.Layout.PaddingVertical, currentTheme.Layout.PaddingHorizontal)

	styleKeybinding = lipgloss.NewStyle().
		Foreground(currentTheme.Secondary).
		Bold(true)

	styleDescription = lipgloss.NewStyle().
		Foreground(currentTheme.Muted)
}

// 颜色主题定义
var (
	// 基础颜色
	colorPrimary    = lipgloss.Color("#7C3AED") // 紫色 - 主色调
	colorSecondary  = lipgloss.Color("#3B82F6") // 蓝色 - 次要色调
	colorSuccess    = lipgloss.Color("#10B981") // 绿色 - 成功
	colorWarning    = lipgloss.Color("#F59E0B") // 黄色 - 警告
	colorError      = lipgloss.Color("#EF4444") // 红色 - 错误
	colorInfo       = lipgloss.Color("#6B7280") // 灰色 - 信息
	colorMuted      = lipgloss.Color("#9CA3AF") // 浅灰色 - 弱化文本
	colorBackground = lipgloss.Color("#1F2937") // 深灰色 - 背景
	colorBorder     = lipgloss.Color("#374151") // 灰色 - 边框

	// 消息类型颜色
	colorUserMessage = lipgloss.Color("#3B82F6") // 蓝色 - 用户消息
	colorThinking    = lipgloss.Color("#6B7280") // 灰色 - 思考内容
	colorContent     = lipgloss.Color("#F3F4F6") // 浅灰色 - 正文内容
	colorToolCall    = lipgloss.Color("#F59E0B") // 黄色 - 工具调用
	colorToolResult  = lipgloss.Color("#10B981") // 绿色 - 工具结果
	colorToolError   = lipgloss.Color("#EF4444") // 红色 - 工具错误

	// 状态颜色
	colorActive   = lipgloss.Color("#10B981") // 绿色 - 活动状态
	colorWaiting  = lipgloss.Color("#F59E0B") // 黄色 - 等待状态
	colorFinished = lipgloss.Color("#6B7280") // 灰色 - 结束状态
)

// 样式定义
var (
	// 基础样式
	styleBase = lipgloss.NewStyle().
			Padding(0, 1)

	// 标题样式
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Padding(0, 1)

	// 状态栏样式
	styleStatusBar = lipgloss.NewStyle().
			Background(colorBackground).
			Foreground(lipgloss.Color("#F3F4F6")).
			Padding(0, 1).
			Bold(true)

	// 状态栏项样式
	styleStatusItem = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F3F4F6")).
			Padding(0, 2)

	// 输入框样式
	styleInput = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	// 输入框焦点样式
	styleInputFocused = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	// 对话区样式
	styleChatArea = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2)
)

// 消息样式定义
var (
	// 用户消息样式
	styleUserMessage = lipgloss.NewStyle().
				Foreground(colorUserMessage).
				Bold(true).
				Padding(0, 1)

	// 用户消息内容样式
	styleUserContent = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F3F4F6")).
				Padding(0, 2)

	// 思考内容样式
	styleThinking = lipgloss.NewStyle().
			Foreground(colorThinking).
			Italic(true).
			Padding(0, 1)

	// 思考内容框样式
	styleThinkingBox = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorMuted).
				Padding(0, 1).
				MarginLeft(2)

	// 思考标题样式
	styleThinkingTitle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Bold(true).
				Padding(0, 1)

	// 正文内容样式
	styleContent = lipgloss.NewStyle().
			Foreground(colorContent).
			Padding(0, 2)

	// 工具调用样式
	styleToolCall = lipgloss.NewStyle().
			Foreground(colorToolCall).
			Bold(true).
			Padding(0, 1)

	// 工具调用框样式
	styleToolCallBox = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorToolCall).
				Padding(0, 1).
				MarginLeft(2)

	// 工具调用标题样式
	styleToolCallTitle = lipgloss.NewStyle().
				Foreground(colorToolCall).
				Bold(true).
				Padding(0, 1)

	// 工具参数样式
	styleToolArgs = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB")).
			Padding(0, 2)

	// 工具结果样式
	styleToolResult = lipgloss.NewStyle().
			Foreground(colorToolResult).
			Padding(0, 2)

	// 工具结果框样式
	styleToolResultBox = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorToolResult).
				Padding(0, 1).
				MarginLeft(2)

	// 工具错误样式
	styleToolError = lipgloss.NewStyle().
			Foreground(colorToolError).
			Padding(0, 2)

	// Token 用量样式
	styleTokenUsage = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 2).
			Italic(true)
)

// 状态样式定义
var (
	// 活动状态样式
	styleStatusActive = lipgloss.NewStyle().
				Foreground(colorActive).
				Bold(true)

	// 等待状态样式
	styleStatusWaiting = lipgloss.NewStyle().
				Foreground(colorWarning).
				Bold(true)

	// 结束状态样式
	styleStatusFinished = lipgloss.NewStyle().
				Foreground(colorFinished)

	// 状态图标
	iconActive   = "🔄"
	iconWaiting  = "⏳"
	iconFinished = "✓"
)

// 对话列表样式定义
var (
	// 对话列表框样式
	styleConversationList = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorPrimary).
				Padding(1, 2)

	// 对话列表标题样式
	styleConversationListTitle = lipgloss.NewStyle().
					Foreground(colorPrimary).
					Bold(true).
					Padding(0, 1).
					MarginBottom(1)

	// 对话项样式
	styleConversationItem = lipgloss.NewStyle().
				Padding(0, 1)

	// 对话项选中样式
	styleConversationItemSelected = lipgloss.NewStyle().
					Background(colorPrimary).
					Foreground(lipgloss.Color("#FFFFFF")).
					Padding(0, 1)

	// 对话标题样式
	styleConversationTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F3F4F6")).
				Bold(true)

	// 对话时间样式
	styleConversationTime = lipgloss.NewStyle().
				Foreground(colorMuted)
)

// 错误样式定义
var (
	// 错误消息样式
	styleError = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true).
			Padding(0, 1)

	// 错误框样式
	styleErrorBox = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorError).
			Padding(0, 1).
			MarginLeft(2)
)

// 帮助文本样式定义
var (
	// 帮助文本样式
	styleHelp = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	// 快捷键样式
	styleKeybinding = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	// 描述文本样式
	styleDescription = lipgloss.NewStyle().
				Foreground(colorMuted)
)

// GetStatusStyle 根据状态获取样式
func GetStatusStyle(status ConvStatus) lipgloss.Style {
	switch status {
	case ConvStatusActive:
		return styleStatusActive
	case ConvStatusWaiting:
		return styleStatusWaiting
	case ConvStatusFinished:
		return styleStatusFinished
	default:
		return styleStatusFinished
	}
}

// GetStatusIcon 根据状态获取图标
func GetStatusIcon(status ConvStatus) string {
	switch status {
	case ConvStatusActive:
		return iconActive
	case ConvStatusWaiting:
		return iconWaiting
	case ConvStatusFinished:
		return iconFinished
	default:
		return iconFinished
	}
}

// GetMessageRoleStyle 根据消息角色获取样式
func GetMessageRoleStyle(role MessageRole) lipgloss.Style {
	switch role {
	case MessageRoleUser:
		return styleUserMessage
	case MessageRoleAssistant:
		return styleContent
	case MessageRoleTool:
		return styleToolResult
	default:
		return styleContent
	}
}

// FormatUserMessage 格式化用户消息
func FormatUserMessage(content string) string {
	header := styleUserMessage.Render("User:")
	body := styleUserContent.Render(content)
	return header + "\n" + body
}

// FormatThinking 格式化思考内容
func FormatThinking(thinking string, isStreaming bool) string {
	title := styleThinkingTitle.Render("┌─ Thinking ─────────────────────────────────────────┐")
	content := styleThinking.Render(thinking)
	if isStreaming {
		content += styleThinking.Render("▌") // 流式光标
	}
	footer := styleThinkingTitle.Render("└────────────────────────────────────────────────────┘")
	return title + "\n" + styleThinkingBox.Render(content) + "\n" + footer
}

// FormatContent 格式化正文内容
func FormatContent(content string, isStreaming bool) string {
	if isStreaming {
		content += styleContent.Render("▌") // 流式光标
	}
	return styleContent.Render(content)
}

// FormatToolCall 格式化工具调用
func FormatToolCall(name, args string, isComplete bool) string {
	title := styleToolCallTitle.Render("┌─ Tool: " + name + " ────────────────────────────────────┐")

	var content string
	if isComplete {
		// 格式化显示参数
		content = styleToolArgs.Render("Parameters:\n" + args)
	} else {
		// 流式生成中，显示原文
		content = styleToolArgs.Render(args + "▌")
	}

	footer := styleToolCallTitle.Render("└────────────────────────────────────────────────────┘")
	return title + "\n" + styleToolCallBox.Render(content) + "\n" + footer
}

// FormatToolResult 格式化工具结果
func FormatToolResult(result string, isError bool) string {
	title := styleToolCallTitle.Render("┌─ Tool Result ──────────────────────────────────────┐")

	var content string
	if isError {
		content = styleToolError.Render(result)
	} else {
		content = styleToolResult.Render(result)
	}

	footer := styleToolCallTitle.Render("└────────────────────────────────────────────────────┘")
	return title + "\n" + styleToolResultBox.Render(content) + "\n" + footer
}

// FormatTokenUsage 格式化 token 用量
func FormatTokenUsage(usage int) string {
	return styleTokenUsage.Render("Tokens: " + string(rune(usage)))
}

// FormatError 格式化错误消息
func FormatError(err string) string {
	title := styleError.Render("Error:")
	content := styleErrorBox.Render(err)
	return title + "\n" + content
}

// FormatStatusBar 格式化状态栏
// 注意：此函数已废弃，请使用 components.StatusBar 组件
func FormatStatusBar(dir string, totalTokens, inferenceCount int, width int) string {
	// 工作目录
	dirPart := styleStatusItem.Render("📁 " + dir)

	// Token 用量
	tokenPart := styleStatusItem.Render(fmt.Sprintf("Tokens: %d", totalTokens))

	// 推理次数
	inferencePart := styleStatusItem.Render(fmt.Sprintf("Inferences: %d", inferenceCount))

	// 组合状态栏
	return lipgloss.JoinHorizontal(lipgloss.Top, dirPart, tokenPart, inferencePart)
}

// FormatConversationItem 格式化对话列表项
func FormatConversationItem(conv *Conversation, isSelected bool) string {
	icon := GetStatusIcon(conv.Status)
	statusStyle := GetStatusStyle(conv.Status)

	var statusText string
	switch conv.Status {
	case ConvStatusActive:
		statusText = "Active"
	case ConvStatusWaiting:
		statusText = "Waiting"
	case ConvStatusFinished:
		statusText = "Finished"
	}

	title := conv.Title
	if title == "" {
		if conv.IsSubConversation {
			title = "Sub-agent: " + conv.AgentRole
		} else {
			title = "Main Conversation"
		}
	}

	timeStr := conv.CreatedAt.Format("2006-01-02 15:04")

	line := icon + " [" + statusText + "]  " + title + "  " + timeStr

	if isSelected {
		return styleConversationItemSelected.Render(line)
	}

	return statusStyle.Render(icon) + styleConversationItem.Render(" ["+statusText+"]  "+title+"  "+timeStr)
}

// FormatHelpText 格式化帮助文本
func FormatHelpText() string {
	help := []struct {
		key  string
		desc string
	}{
		{"Enter", "提交输入"},
		{"Tab", "切换输入模式"},
		{"Ctrl+L", "查看对话列表"},
		{"Ctrl+C / Esc", "退出 TUI"},
	}

	var lines []string
	for _, h := range help {
		key := styleKeybinding.Render(h.key)
		desc := styleDescription.Render(h.desc)
		lines = append(lines, key+"  "+desc)
	}

	return styleHelp.Render("快捷键: ") + "\n" + lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ========== 布局样式辅助函数 ==========

// NewBorderStyle 创建边框样式
// borderType: "normal", "rounded", "double", "thick", "hidden"
func NewBorderStyle(borderType string, borderColor color.Color) lipgloss.Style {
	var border lipgloss.Border
	switch borderType {
	case "rounded":
		border = lipgloss.RoundedBorder()
	case "double":
		border = lipgloss.DoubleBorder()
	case "thick":
		border = lipgloss.ThickBorder()
	case "hidden":
		border = lipgloss.HiddenBorder()
	default:
		border = lipgloss.NormalBorder()
	}

	return lipgloss.NewStyle().
		Border(border).
		BorderForeground(borderColor)
}

// NewPaddingStyle 创建内边距样式
func NewPaddingStyle(vertical, horizontal int) lipgloss.Style {
	return lipgloss.NewStyle().
		Padding(vertical, horizontal)
}

// NewMarginStyle 创建外边距样式
func NewMarginStyle(vertical, horizontal int) lipgloss.Style {
	return lipgloss.NewStyle().
		Margin(vertical, horizontal)
}

// NewAlignedStyle 创建对齐样式
func NewAlignedStyle(horizontal, vertical lipgloss.Position) lipgloss.Style {
	return lipgloss.NewStyle().
		Align(horizontal, vertical)
}

// NewBoxStyle 创建盒子样式（带边框和内边距）
func NewBoxStyle(borderType string, borderColor color.Color, paddingVertical, paddingHorizontal int) lipgloss.Style {
	return NewBorderStyle(borderType, borderColor).
		Padding(paddingVertical, paddingHorizontal)
}

// ========== 样式构建器 ==========

// StyleBuilder 样式构建器
// 提供链式调用的样式构建接口
type StyleBuilder struct {
	style lipgloss.Style
}

// NewStyleBuilder 创建样式构建器
func NewStyleBuilder() *StyleBuilder {
	return &StyleBuilder{
		style: lipgloss.NewStyle(),
	}
}

// Foreground 设置前景色
func (b *StyleBuilder) Foreground(foregroundColor color.Color) *StyleBuilder {
	b.style = b.style.Foreground(foregroundColor)
	return b
}

// Background 设置背景色
func (b *StyleBuilder) Background(backgroundColor color.Color) *StyleBuilder {
	b.style = b.style.Background(backgroundColor)
	return b
}

// Bold 设置粗体
func (b *StyleBuilder) Bold(bold bool) *StyleBuilder {
	b.style = b.style.Bold(bold)
	return b
}

// Italic 设置斜体
func (b *StyleBuilder) Italic(italic bool) *StyleBuilder {
	b.style = b.style.Italic(italic)
	return b
}

// Underline 设置下划线
func (b *StyleBuilder) Underline(underline bool) *StyleBuilder {
	b.style = b.style.Underline(underline)
	return b
}

// Padding 设置内边距
func (b *StyleBuilder) Padding(vertical, horizontal int) *StyleBuilder {
	b.style = b.style.Padding(vertical, horizontal)
	return b
}

// Margin 设置外边距
func (b *StyleBuilder) Margin(vertical, horizontal int) *StyleBuilder {
	b.style = b.style.Margin(vertical, horizontal)
	return b
}

// Border 设置边框
func (b *StyleBuilder) Border(border lipgloss.Border) *StyleBuilder {
	b.style = b.style.Border(border)
	return b
}

// BorderForeground 设置边框前景色
func (b *StyleBuilder) BorderForeground(borderColor color.Color) *StyleBuilder {
	b.style = b.style.BorderForeground(borderColor)
	return b
}

// Width 设置宽度
func (b *StyleBuilder) Width(width int) *StyleBuilder {
	b.style = b.style.Width(width)
	return b
}

// Height 设置高度
func (b *StyleBuilder) Height(height int) *StyleBuilder {
	b.style = b.style.Height(height)
	return b
}

// Align 设置对齐
func (b *StyleBuilder) Align(horizontal, vertical lipgloss.Position) *StyleBuilder {
	b.style = b.style.Align(horizontal, vertical)
	return b
}

// Build 构建最终样式
func (b *StyleBuilder) Build() lipgloss.Style {
	return b.style
}

// Render 渲染文本
func (b *StyleBuilder) Render(text string) string {
	return b.style.Render(text)
}

// ========== 样式工具函数 ==========

// JoinHorizontal 水平连接多个文本
func JoinHorizontal(texts ...string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, texts...)
}

// JoinVertical 垂直连接多个文本
func JoinVertical(texts ...string) string {
	return lipgloss.JoinVertical(lipgloss.Left, texts...)
}

// Place 将内容放置在指定大小的区域内
func Place(width, height int, hPos, vPos lipgloss.Position, content string) string {
	return lipgloss.Place(width, height, hPos, vPos, content)
}

// Width 获取文本渲染后的宽度
func Width(text string) int {
	return lipgloss.Width(text)
}

// Height 获取文本渲染后的高度
func Height(text string) int {
	return lipgloss.Height(text)
}

// ========== 样式使用示例 ==========

/*
样式使用示例：

1. 基础样式使用：

	// 使用预定义样式
	userMsg := styleUserMessage.Render("User:")
	content := styleUserContent.Render("Hello, world!")
	fmt.Println(userMsg + "\n" + content)

2. 使用主题系统：

	// 切换到深色主题
	SetTheme(DarkTheme())

	// 切换到浅色主题
	SetTheme(LightTheme())

	// 自定义主题
	customTheme := DefaultTheme()
	customTheme.Primary = lipgloss.Color("#FF6B6B")
	customTheme.UserMessage = lipgloss.Color("#4ECDC4")
	SetTheme(customTheme)

3. 使用样式构建器：

	// 创建自定义样式
	customStyle := NewStyleBuilder().
		Foreground(lipgloss.Color("#FF6B6B")).
		Background(lipgloss.Color("#2C3E50")).
		Bold(true).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3498DB")).
		Build()

	fmt.Println(customStyle.Render("Custom styled text"))

4. 使用布局辅助函数：

	// 创建带边框的盒子
	box := NewBoxStyle("rounded", lipgloss.Color("#3498DB"), 1, 2)
	fmt.Println(box.Render("Box content"))

	// 创建内边距样式
	padded := NewPaddingStyle(1, 2)
	fmt.Println(padded.Render("Padded text"))

	// 创建对齐样式
	centered := NewAlignedStyle(lipgloss.Center, lipgloss.Center)
	fmt.Println(centered.Render("Centered text"))

5. 格式化消息：

	// 格式化用户消息
	userMsg := FormatUserMessage("请帮我创建一个新功能")

	// 格式化思考内容
	thinking := FormatThinking("我需要分析用户的需求...", true)

	// 格式化工具调用
	toolCall := FormatToolCall("create_file", `{"path": "/tmp/test.go"}`, true)

	// 格式化工具结果
	toolResult := FormatToolResult("文件创建成功", false)

	// 格式化错误消息
	errorMsg := FormatError("无法访问文件")

6. 状态样式：

	// 获取状态样式
	activeStyle := GetStatusStyle(ConvStatusActive)
	waitingStyle := GetStatusStyle(ConvStatusWaiting)
	finishedStyle := GetStatusStyle(ConvStatusFinished)

	// 获取状态图标
	activeIcon := GetStatusIcon(ConvStatusActive)   // 🔄
	waitingIcon := GetStatusIcon(ConvStatusWaiting) // ⏳
	finishedIcon := GetStatusIcon(ConvStatusFinished) // ✓

7. 布局组合：

	// 水平布局
	statusBar := JoinHorizontal(
		styleStatusItem.Render("📁 /path/to/dir"),
		styleStatusItem.Render("Tokens: 123"),
		styleStatusItem.Render("Inferences: 5"),
	)

	// 垂直布局
	chatArea := JoinVertical(
		FormatUserMessage("Hello"),
		FormatContent("Hi there!", false),
	)

	// 居中放置
	centeredContent := Place(80, 20, lipgloss.Center, lipgloss.Center, "Centered content")
*/
