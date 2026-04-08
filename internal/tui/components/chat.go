// Package components 提供 TUI 组件实现
package components

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ChatView 对话显示组件
// 封装 viewport 组件，提供消息渲染和自动滚动功能
type ChatView struct {
	// viewport 视口组件
	viewport viewport.Model

	// width 宽度
	width int
	// height 高度
	height int

	// autoScroll 是否自动滚动到底部
	autoScroll bool

	// lastContentHeight 上次内容高度（用于检测是否需要滚动）
	lastContentHeight int

	// 消息样式
	styles ChatStyles

	// formatter 工具调用格式化器
	formatter *Formatter
}

// ChatStyles 对话样式配置
type ChatStyles struct {
	// UserMessage 用户消息样式
	UserMessage lipgloss.Style
	// UserContent 用户消息内容样式
	UserContent lipgloss.Style

	// Thinking 思考内容样式
	Thinking lipgloss.Style
	// ThinkingBox 思考内容框样式
	ThinkingBox lipgloss.Style
	// ThinkingTitle 思考标题样式
	ThinkingTitle lipgloss.Style

	// Content 正文内容样式
	Content lipgloss.Style

	// ToolCall 工具调用样式
	ToolCall lipgloss.Style
	// ToolCallBox 工具调用框样式
	ToolCallBox lipgloss.Style
	// ToolCallTitle 工具调用标题样式
	ToolCallTitle lipgloss.Style
	// ToolArgs 工具参数样式
	ToolArgs lipgloss.Style

	// ToolResult 工具结果样式
	ToolResult lipgloss.Style
	// ToolResultBox 工具结果框样式
	ToolResultBox lipgloss.Style
	// ToolError 工具错误样式
	ToolError lipgloss.Style

	// TokenUsage token 用量样式
	TokenUsage lipgloss.Style

	// Error 错误消息样式
	Error lipgloss.Style
	// ErrorBox 错误框样式
	ErrorBox lipgloss.Style

	// StreamingCursor 流式光标样式
	StreamingCursor lipgloss.Style
}

// MessageData 消息数据
type MessageData struct {
	// Role 消息角色（user/assistant/tool）
	Role string
	// Content 正文内容
	Content string
	// Thinking 思考内容
	Thinking string
	// ToolCalls 工具调用列表
	ToolCalls []ToolCallData
	// TokenUsage token 用量
	TokenUsage int
	// IsStreaming 是否正在流式输出
	IsStreaming bool
	// StreamingType 流式输出类型（thinking/content/tool）
	StreamingType string
}

// ToolCallData 工具调用数据
type ToolCallData struct {
	// ID 工具调用唯一标识
	ID string
	// Name 工具名称
	Name string
	// Arguments 工具参数（JSON 格式）
	Arguments string
	// IsComplete 是否已完成流式生成
	IsComplete bool
	// Result 工具执行结果
	Result string
	// ResultError 工具执行错误
	ResultError string
}

// DefaultChatStyles 返回默认对话样式
func DefaultChatStyles() ChatStyles {
	// 颜色定义
	colorUserMessage := lipgloss.Color("#3B82F6") // 蓝色 - 用户消息
	colorThinking := lipgloss.Color("#6B7280")    // 灰色 - 思考内容
	colorContent := lipgloss.Color("#F3F4F6")     // 浅灰色 - 正文内容
	colorToolCall := lipgloss.Color("#F59E0B")    // 黄色 - 工具调用
	colorToolResult := lipgloss.Color("#10B981")  // 绿色 - 工具结果
	colorToolError := lipgloss.Color("#EF4444")   // 红色 - 工具错误
	colorMuted := lipgloss.Color("#9CA3AF")       // 浅灰色 - 弱化文本

	return ChatStyles{
		UserMessage: lipgloss.NewStyle().
			Foreground(colorUserMessage).
			Bold(true).
			Padding(0, 1),

		UserContent: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F3F4F6")).
			Padding(0, 2),

		Thinking: lipgloss.NewStyle().
			Foreground(colorThinking).
			Italic(true).
			Padding(0, 1),

		ThinkingBox: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1).
			MarginLeft(2),

		ThinkingTitle: lipgloss.NewStyle().
			Foreground(colorMuted).
			Bold(true).
			Padding(0, 1),

		Content: lipgloss.NewStyle().
			Foreground(colorContent).
			Padding(0, 2),

		ToolCall: lipgloss.NewStyle().
			Foreground(colorToolCall).
			Bold(true).
			Padding(0, 1),

		ToolCallBox: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorToolCall).
			Padding(0, 1).
			MarginLeft(2),

		ToolCallTitle: lipgloss.NewStyle().
			Foreground(colorToolCall).
			Bold(true).
			Padding(0, 1),

		ToolArgs: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB")).
			Padding(0, 2),

		ToolResult: lipgloss.NewStyle().
			Foreground(colorToolResult).
			Padding(0, 2),

		ToolResultBox: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorToolResult).
			Padding(0, 1).
			MarginLeft(2),

		ToolError: lipgloss.NewStyle().
			Foreground(colorToolError).
			Padding(0, 2),

		TokenUsage: lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 2).
			Italic(true),

		Error: lipgloss.NewStyle().
			Foreground(colorToolError).
			Bold(true).
			Padding(0, 1),

		ErrorBox: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorToolError).
			Padding(0, 1).
			MarginLeft(2),

		StreamingCursor: lipgloss.NewStyle().
			Foreground(colorUserMessage).
			Bold(true),
	}
}

// NewChatView 创建新的对话显示组件
func NewChatView(width, height int) *ChatView {
	// 创建视口
	vp := viewport.New(viewport.WithWidth(width), viewport.WithHeight(height))
	vp.SetContent("")

	return &ChatView{
		viewport:   vp,
		width:      width,
		height:     height,
		autoScroll: true,
		styles:     DefaultChatStyles(),
		formatter:  NewFormatter(),
	}
}

// NewChatViewWithStyles 创建带自定义样式的对话显示组件
func NewChatViewWithStyles(width, height int, styles ChatStyles) *ChatView {
	vp := viewport.New(viewport.WithWidth(width), viewport.WithHeight(height))
	vp.SetContent("")

	return &ChatView{
		viewport:   vp,
		width:      width,
		height:     height,
		autoScroll: true,
		styles:     styles,
		formatter:  NewFormatter(),
	}
}

// Init 初始化组件
func (c *ChatView) Init() tea.Cmd {
	return nil
}

// Update 更新组件状态
func (c *ChatView) Update(msg tea.Msg) (*ChatView, tea.Cmd) {
	var cmd tea.Cmd

	// 更新视口
	c.viewport, cmd = c.viewport.Update(msg)

	return c, cmd
}

// View 渲染组件
func (c *ChatView) View() string {
	return c.viewport.View()
}

// SetSize 设置组件大小
func (c *ChatView) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.viewport.SetWidth(width)
	c.viewport.SetHeight(height)
}

// SetWidth 设置宽度
func (c *ChatView) SetWidth(width int) {
	c.width = width
	c.viewport.SetWidth(width)
}

// SetHeight 设置高度
func (c *ChatView) SetHeight(height int) {
	c.height = height
	c.viewport.SetHeight(height)
}

// SetContent 设置内容
func (c *ChatView) SetContent(content string) {
	// 计算新内容高度
	newHeight := strings.Count(content, "\n") + 1

	// 更新内容
	c.viewport.SetContent(content)

	// 如果启用了自动滚动，滚动到底部
	if c.autoScroll && newHeight > c.lastContentHeight {
		c.ScrollToBottom()
	}

	c.lastContentHeight = newHeight
}

// SetAutoScroll 设置自动滚动
func (c *ChatView) SetAutoScroll(autoScroll bool) {
	c.autoScroll = autoScroll
}

// ScrollToBottom 滚动到底部
func (c *ChatView) ScrollToBottom() {
	// viewport v2 使用 GotoBottom 方法
	c.viewport.GotoBottom()
}

// ScrollToTop 滚动到顶部
func (c *ChatView) ScrollToTop() {
	c.viewport.GotoTop()
}

// PageDown 向下翻页
func (c *ChatView) PageDown() {
	c.viewport.HalfPageDown()
}

// PageUp 向上翻页
func (c *ChatView) PageUp() {
	c.viewport.HalfPageUp()
}

// LineDown 向下滚动一行
func (c *ChatView) LineDown() {
	c.viewport.ScrollDown(1)
}

// LineUp 向上滚动一行
func (c *ChatView) LineUp() {
	c.viewport.ScrollUp(1)
}

// GetViewport 获取视口组件
func (c *ChatView) GetViewport() *viewport.Model {
	return &c.viewport
}

// RenderMessages 渲染消息列表
func (c *ChatView) RenderMessages(messages []MessageData) string {
	var builder strings.Builder

	for i, msg := range messages {
		// 渲染单条消息
		rendered := c.RenderMessage(msg)
		builder.WriteString(rendered)

		// 消息之间添加空行（最后一条消息除外）
		if i < len(messages)-1 {
			builder.WriteString("\n\n")
		}
	}

	return builder.String()
}

// RenderMessage 渲染单条消息
func (c *ChatView) RenderMessage(msg MessageData) string {
	var builder strings.Builder

	switch msg.Role {
	case "user":
		// 用户消息
		builder.WriteString(c.RenderUserMessage(msg.Content))

	case "assistant":
		// 助手消息
		builder.WriteString(c.RenderAssistantMessage(msg))

	case "tool":
		// 工具消息
		builder.WriteString(c.RenderToolMessage(msg))
	}

	return builder.String()
}

// RenderUserMessage 渲染用户消息
func (c *ChatView) RenderUserMessage(content string) string {
	header := c.styles.UserMessage.Render("User:")
	body := c.styles.UserContent.Render(content)
	return header + "\n" + body
}

// RenderAssistantMessage 渲染助手消息
func (c *ChatView) RenderAssistantMessage(msg MessageData) string {
	var builder strings.Builder

	// 渲染思考内容（如果有）
	if msg.Thinking != "" {
		builder.WriteString(c.RenderThinking(msg.Thinking, msg.IsStreaming && msg.StreamingType == "thinking"))
		builder.WriteString("\n\n")
	}

	// 渲染工具调用（如果有）
	if len(msg.ToolCalls) > 0 {
		for _, tc := range msg.ToolCalls {
			builder.WriteString(c.RenderToolCall(tc, msg.IsStreaming && msg.StreamingType == "tool"))
			builder.WriteString("\n")

			// 如果有结果，渲染结果
			if tc.Result != "" || tc.ResultError != "" {
				builder.WriteString(c.RenderToolResult(tc))
				builder.WriteString("\n")
			}
		}
	}

	// 渲染正文内容（如果有）
	if msg.Content != "" {
		builder.WriteString(c.RenderContent(msg.Content, msg.IsStreaming && msg.StreamingType == "content"))
	}

	// 渲染 token 用量（如果有）
	if msg.TokenUsage > 0 {
		builder.WriteString("\n")
		builder.WriteString(c.RenderTokenUsage(msg.TokenUsage))
	}

	return builder.String()
}

// RenderToolMessage 渲染工具消息
func (c *ChatView) RenderToolMessage(msg MessageData) string {
	// 工具消息通常是工具执行结果
	if len(msg.ToolCalls) > 0 {
		return c.RenderToolResult(msg.ToolCalls[0])
	}
	return ""
}

// RenderThinking 渲染思考内容
func (c *ChatView) RenderThinking(thinking string, isStreaming bool) string {
	title := c.styles.ThinkingTitle.Render("┌─ Thinking ─────────────────────────────────────────┐")

	content := c.styles.Thinking.Render(thinking)
	if isStreaming {
		content += c.styles.StreamingCursor.Render("▌") // 流式光标
	}

	footer := c.styles.ThinkingTitle.Render("└────────────────────────────────────────────────────┘")

	return title + "\n" + c.styles.ThinkingBox.Render(content) + "\n" + footer
}

// RenderContent 渲染正文内容
func (c *ChatView) RenderContent(content string, isStreaming bool) string {
	if isStreaming {
		content += c.styles.StreamingCursor.Render("▌") // 流式光标
	}
	return c.styles.Content.Render(content)
}

// RenderToolCall 渲染工具调用
func (c *ChatView) RenderToolCall(tc ToolCallData, isStreaming bool) string {
	// 使用格式化器格式化工具调用
	return c.formatter.FormatToolCall(tc.Name, tc.Arguments, tc.IsComplete)
}

// RenderToolResult 渲染工具结果
func (c *ChatView) RenderToolResult(tc ToolCallData) string {
	// 使用格式化器格式化工具结果
	isError := tc.ResultError != ""
	result := tc.Result
	if isError {
		result = tc.ResultError
	}
	return c.formatter.FormatToolResult(result, isError)
}

// RenderTokenUsage 渲染 token 用量
func (c *ChatView) RenderTokenUsage(usage int) string {
	return c.styles.TokenUsage.Render(fmt.Sprintf("Tokens: %d", usage))
}

// RenderError 渲染错误消息
func (c *ChatView) RenderError(err string) string {
	title := c.styles.Error.Render("Error:")
	content := c.styles.ErrorBox.Render(err)
	return title + "\n" + content
}

// GetWidth 获取宽度
func (c *ChatView) GetWidth() int {
	return c.width
}

// GetHeight 获取高度
func (c *ChatView) GetHeight() int {
	return c.height
}

// IsAtBottom 检查是否在底部
func (c *ChatView) IsAtBottom() bool {
	// viewport v2 没有直接的 AtBottom 方法，需要计算
	// 获取当前滚动位置和内容高度
	return c.viewport.AtBottom()
}

// SetStyles 设置样式
func (c *ChatView) SetStyles(styles ChatStyles) {
	c.styles = styles
}

// GetStyles 获取样式
func (c *ChatView) GetStyles() ChatStyles {
	return c.styles
}
