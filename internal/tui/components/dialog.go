// Package components 提供 TUI 组件
package components

import (
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ConvStatus 对话状态类型
type ConvStatus int

const (
	// ConvStatusActive 活动（正在推理）
	ConvStatusActive ConvStatus = iota
	// ConvStatusWaiting 等待（等待子代理完成）
	ConvStatusWaiting
	// ConvStatusFinished 结束（对话内容结束）
	ConvStatusFinished
)

// Conversation 对话信息（简化版，用于列表显示）
type Conversation struct {
	// ID 对话唯一标识
	ID string
	// CreatedAt 创建时间
	CreatedAt time.Time
	// Status 对话状态
	Status ConvStatus
	// Title 对话标题
	Title string
	// IsSubConversation 是否为子对话
	IsSubConversation bool
	// AgentRole Agent 角色（如果是子对话）
	AgentRole string
	// MessageCount 消息数量
	MessageCount int
	// TokenUsage token 用量
	TokenUsage int
}

// DialogList 对话列表弹窗组件
type DialogList struct {
	// conversations 对话列表（已排序）
	conversations []*Conversation
	// selectedIndex 当前选中的索引
	selectedIndex int
	// width 宽度
	width int
	// height 高度
	height int
	// visible 是否可见
	visible bool
	// title 标题
	title string
}

// NewDialogList 创建新的对话列表组件
func NewDialogList() *DialogList {
	return &DialogList{
		conversations:  make([]*Conversation, 0),
		selectedIndex:  0,
		visible:        false,
		title:          "对话列表",
	}
}

// SetConversations 设置对话列表
// 会自动按创建时间排序（最新的在前）
func (d *DialogList) SetConversations(conversations []*Conversation) {
	// 复制切片以避免修改原始数据
	d.conversations = make([]*Conversation, len(conversations))
	copy(d.conversations, conversations)

	// 按创建时间降序排序（最新的在前）
	sort.Slice(d.conversations, func(i, j int) bool {
		return d.conversations[i].CreatedAt.After(d.conversations[j].CreatedAt)
	})

	// 确保选中索引有效
	if d.selectedIndex >= len(d.conversations) {
		d.selectedIndex = len(d.conversations) - 1
	}
	if d.selectedIndex < 0 {
		d.selectedIndex = 0
	}
}

// GetConversations 获取对话列表
func (d *DialogList) GetConversations() []*Conversation {
	return d.conversations
}

// SetSize 设置组件大小
func (d *DialogList) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// Show 显示对话列表
func (d *DialogList) Show() {
	d.visible = true
}

// Hide 隐藏对话列表
func (d *DialogList) Hide() {
	d.visible = false
}

// Toggle 切换显示状态
func (d *DialogList) Toggle() {
	d.visible = !d.visible
}

// IsVisible 是否可见
func (d *DialogList) IsVisible() bool {
	return d.visible
}

// GetSelected 获取当前选中的对话
func (d *DialogList) GetSelected() *Conversation {
	if len(d.conversations) == 0 || d.selectedIndex < 0 || d.selectedIndex >= len(d.conversations) {
		return nil
	}
	return d.conversations[d.selectedIndex]
}

// GetSelectedIndex 获取当前选中的索引
func (d *DialogList) GetSelectedIndex() int {
	return d.selectedIndex
}

// SetSelectedIndex 设置选中的索引
func (d *DialogList) SetSelectedIndex(index int) {
	if index >= 0 && index < len(d.conversations) {
		d.selectedIndex = index
	}
}

// Update 处理消息更新
func (d *DialogList) Update(msg tea.Msg) (*Conversation, bool) {
	if !d.visible {
		return nil, false
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			// 向上移动
			d.moveUp()
			return nil, false

		case "down", "j":
			// 向下移动
			d.moveDown()
			return nil, false

		case "enter":
			// 选择当前项
			selected := d.GetSelected()
			if selected != nil {
				return selected, true
			}
			return nil, false

		case "esc", "q":
			// 关闭列表
			d.Hide()
			return nil, false
		}
	}

	return nil, false
}

// moveUp 向上移动选择
func (d *DialogList) moveUp() {
	if len(d.conversations) == 0 {
		return
	}
	d.selectedIndex--
	if d.selectedIndex < 0 {
		d.selectedIndex = len(d.conversations) - 1
	}
}

// moveDown 向下移动选择
func (d *DialogList) moveDown() {
	if len(d.conversations) == 0 {
		return
	}
	d.selectedIndex++
	if d.selectedIndex >= len(d.conversations) {
		d.selectedIndex = 0
	}
}

// View 渲染对话列表
func (d *DialogList) View() string {
	if !d.visible {
		return ""
	}

	// 定义样式
	styleBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(0, 1)

	styleTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Padding(0, 1).
		MarginBottom(1)

	styleItem := lipgloss.NewStyle().
		Padding(0, 1)

	styleItemSelected := lipgloss.NewStyle().
		Background(lipgloss.Color("#7C3AED")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		Bold(true)

	// 构建内容
	var lines []string

	// 添加标题
	lines = append(lines, styleTitle.Render("╔═ "+d.title+" ═╗"))
	lines = append(lines, "")

	// 添加对话列表
	if len(d.conversations) == 0 {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true).
			Render("  暂无对话"))
	} else {
		for i, conv := range d.conversations {
			line := d.formatConversationItem(conv, i == d.selectedIndex)

			if i == d.selectedIndex {
				lines = append(lines, styleItemSelected.Render("▶ "+line))
			} else {
				lines = append(lines, styleItem.Render("  "+line))
			}
		}
	}

	// 添加帮助信息
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Render("  ↑/k: 上移  ↓/j: 下移  Enter: 选择  Esc: 关闭"))

	// 应用边框
	content := strings.Join(lines, "\n")
	return styleBorder.Render(content)
}

// formatConversationItem 格式化对话列表项
func (d *DialogList) formatConversationItem(conv *Conversation, isSelected bool) string {
	// 获取状态图标和颜色
	icon, statusColorStr := getStatusIconAndColor(conv.Status)

	// 构建标题
	title := conv.Title
	if title == "" {
		if conv.IsSubConversation {
			title = "子代理: " + conv.AgentRole
		} else {
			title = "主对话"
		}
	}

	// 格式化时间
	timeStr := formatTime(conv.CreatedAt)

	// 构建状态文本
	var statusText string
	switch conv.Status {
	case ConvStatusActive:
		statusText = "活动"
	case ConvStatusWaiting:
		statusText = "等待"
	case ConvStatusFinished:
		statusText = "结束"
	}

	// 构建行内容
	line := icon + " [" + statusText + "] " + title + " (" + timeStr + ")"

	// 如果不是选中状态，应用状态颜色
	if !isSelected {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColorStr))
		return style.Render(icon) + " [" + statusText + "] " + title + " (" + timeStr + ")"
	}

	return line
}

// getStatusIconAndColor 获取状态图标和颜色
func getStatusIconAndColor(status ConvStatus) (string, string) {
	switch status {
	case ConvStatusActive:
		return "🔄", "#10B981" // 绿色
	case ConvStatusWaiting:
		return "⏳", "#F59E0B" // 黄色
	case ConvStatusFinished:
		return "✓", "#6B7280" // 灰色
	default:
		return "✓", "#6B7280"
	}
}

// formatTime 格式化时间显示
func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	// 小于1分钟
	if diff < time.Minute {
		return "刚刚"
	}

	// 小于1小时
	if diff < time.Hour {
		minutes := int(diff.Minutes())
		return string(rune(minutes)) + "分钟前"
	}

	// 小于24小时
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return string(rune(hours)) + "小时前"
	}

	// 小于7天
	if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return string(rune(days)) + "天前"
	}

	// 其他情况，显示具体日期
	return t.Format("2006-01-02 15:04")
}

// RenderOverlay 渲染为覆盖层（居中显示）
func (d *DialogList) RenderOverlay(parentWidth, parentHeight int) string {
	if !d.visible {
		return ""
	}

	// 计算列表内容的高度
	listHeight := len(d.conversations) + 6 // 标题 + 空行 + 列表项 + 空行 + 帮助
	if listHeight < 10 {
		listHeight = 10
	}
	if listHeight > parentHeight-4 {
		listHeight = parentHeight - 4
	}

	// 计算列表宽度
	listWidth := parentWidth - 4
	if listWidth > 80 {
		listWidth = 80
	}
	if listWidth < 40 {
		listWidth = 40
	}

	// 设置组件大小
	d.SetSize(listWidth, listHeight)

	// 渲染列表内容
	content := d.View()

	// 计算居中位置
	// 使用 lipgloss.Place 居中
	return lipgloss.Place(
		parentWidth,
		parentHeight,
		lipgloss.Center,
		lipgloss.Center,
		content,
		lipgloss.WithWhitespaceChars(" "),
	)
}

// GetConversationCount 获取对话数量
func (d *DialogList) GetConversationCount() int {
	return len(d.conversations)
}

// FindConversationByID 根据ID查找对话
func (d *DialogList) FindConversationByID(id string) *Conversation {
	for _, conv := range d.conversations {
		if conv.ID == id {
			return conv
		}
	}
	return nil
}

// SelectConversationByID 根据ID选择对话
func (d *DialogList) SelectConversationByID(id string) bool {
	for i, conv := range d.conversations {
		if conv.ID == id {
			d.selectedIndex = i
			return true
		}
	}
	return false
}

// FilterByStatus 按状态过滤对话
func (d *DialogList) FilterByStatus(status ConvStatus) []*Conversation {
	var result []*Conversation
	for _, conv := range d.conversations {
		if conv.Status == status {
			result = append(result, conv)
		}
	}
	return result
}

// FilterSubConversations 过滤子对话
func (d *DialogList) FilterSubConversations(onlySub bool) []*Conversation {
	var result []*Conversation
	for _, conv := range d.conversations {
		if onlySub && conv.IsSubConversation {
			result = append(result, conv)
		} else if !onlySub && !conv.IsSubConversation {
			result = append(result, conv)
		}
	}
	return result
}

// GetStatistics 获取统计信息
func (d *DialogList) GetStatistics() map[ConvStatus]int {
	stats := make(map[ConvStatus]int)
	for _, conv := range d.conversations {
		stats[conv.Status]++
	}
	return stats
}
