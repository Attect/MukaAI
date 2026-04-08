// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"charm.land/lipgloss/v2"
)

// 颜色主题定义
var (
	// 基础颜色
	colorPrimary   = lipgloss.Color("#7C3AED") // 紫色 - 主色调
	colorSecondary = lipgloss.Color("#3B82F6") // 蓝色 - 次要色调
	colorSuccess   = lipgloss.Color("#10B981") // 绿色 - 成功
	colorWarning   = lipgloss.Color("#F59E0B") // 黄色 - 警告
	colorError     = lipgloss.Color("#EF4444") // 红色 - 错误
	colorInfo      = lipgloss.Color("#6B7280") // 灰色 - 信息
	colorMuted     = lipgloss.Color("#9CA3AF") // 浅灰色 - 弱化文本
	colorBackground = lipgloss.Color("#1F2937") // 深灰色 - 背景
	colorBorder    = lipgloss.Color("#374151") // 灰色 - 边框

	// 消息类型颜色
	colorUserMessage    = lipgloss.Color("#3B82F6") // 蓝色 - 用户消息
	colorThinking       = lipgloss.Color("#6B7280") // 灰色 - 思考内容
	colorContent        = lipgloss.Color("#F3F4F6") // 浅灰色 - 正文内容
	colorToolCall       = lipgloss.Color("#F59E0B") // 黄色 - 工具调用
	colorToolResult     = lipgloss.Color("#10B981") // 绿色 - 工具结果
	colorToolError      = lipgloss.Color("#EF4444") // 红色 - 工具错误

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
func FormatStatusBar(dir string, totalTokens, inferenceCount int, width int) string {
	// 工作目录
	dirPart := styleStatusItem.Render("📁 " + dir)

	// Token 用量
	tokenPart := styleStatusItem.Render("Tokens: " + string(rune(totalTokens)))

	// 推理次数
	inferencePart := styleStatusItem.Render("Inferences: " + string(rune(inferenceCount)))

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
