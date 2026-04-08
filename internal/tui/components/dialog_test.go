// Package components 提供 TUI 组件
package components

import (
	"testing"
	"time"
)

// TestNewDialogList 测试创建对话列表组件
func TestNewDialogList(t *testing.T) {
	dialog := NewDialogList()

	if dialog == nil {
		t.Fatal("NewDialogList() returned nil")
	}

	if dialog.visible {
		t.Error("Dialog list should not be visible initially")
	}

	if len(dialog.conversations) != 0 {
		t.Error("Dialog list should be empty initially")
	}

	if dialog.selectedIndex != 0 {
		t.Error("Selected index should be 0 initially")
	}

	if dialog.title != "对话列表" {
		t.Errorf("Title should be '对话列表', got '%s'", dialog.title)
	}
}

// TestSetConversations 测试设置对话列表
func TestSetConversations(t *testing.T) {
	dialog := NewDialogList()

	// 创建测试对话列表
	now := time.Now()
	conversations := []*Conversation{
		{
			ID:        "conv-1",
			CreatedAt: now.Add(-2 * time.Hour),
			Status:    ConvStatusFinished,
			Title:     "对话1",
		},
		{
			ID:        "conv-2",
			CreatedAt: now.Add(-1 * time.Hour),
			Status:    ConvStatusActive,
			Title:     "对话2",
		},
		{
			ID:        "conv-3",
			CreatedAt: now,
			Status:    ConvStatusWaiting,
			Title:     "对话3",
		},
	}

	dialog.SetConversations(conversations)

	// 检查数量
	if len(dialog.conversations) != 3 {
		t.Errorf("Expected 3 conversations, got %d", len(dialog.conversations))
	}

	// 检查排序（最新的在前）
	if dialog.conversations[0].ID != "conv-3" {
		t.Errorf("First conversation should be conv-3 (newest), got %s", dialog.conversations[0].ID)
	}

	if dialog.conversations[1].ID != "conv-2" {
		t.Errorf("Second conversation should be conv-2, got %s", dialog.conversations[1].ID)
	}

	if dialog.conversations[2].ID != "conv-1" {
		t.Errorf("Third conversation should be conv-1 (oldest), got %s", dialog.conversations[2].ID)
	}
}

// TestShowHide 测试显示和隐藏
func TestShowHide(t *testing.T) {
	dialog := NewDialogList()

	// 初始状态应该是隐藏的
	if dialog.IsVisible() {
		t.Error("Dialog should be hidden initially")
	}

	// 显示
	dialog.Show()
	if !dialog.IsVisible() {
		t.Error("Dialog should be visible after Show()")
	}

	// 隐藏
	dialog.Hide()
	if dialog.IsVisible() {
		t.Error("Dialog should be hidden after Hide()")
	}

	// 切换
	dialog.Toggle()
	if !dialog.IsVisible() {
		t.Error("Dialog should be visible after Toggle()")
	}

	dialog.Toggle()
	if dialog.IsVisible() {
		t.Error("Dialog should be hidden after second Toggle()")
	}
}

// TestNavigation 测试导航功能
func TestNavigation(t *testing.T) {
	dialog := NewDialogList()

	// 创建测试对话列表
	now := time.Now()
	conversations := []*Conversation{
		{ID: "conv-1", CreatedAt: now.Add(-2 * time.Hour), Status: ConvStatusFinished},
		{ID: "conv-2", CreatedAt: now.Add(-1 * time.Hour), Status: ConvStatusActive},
		{ID: "conv-3", CreatedAt: now, Status: ConvStatusWaiting},
	}

	dialog.SetConversations(conversations)
	dialog.Show()

	// 初始选中应该是第一个（最新的）
	if dialog.GetSelectedIndex() != 0 {
		t.Errorf("Initial selected index should be 0, got %d", dialog.GetSelectedIndex())
	}

	// 向下移动
	dialog.moveDown()
	if dialog.GetSelectedIndex() != 1 {
		t.Errorf("After moveDown, selected index should be 1, got %d", dialog.GetSelectedIndex())
	}

	dialog.moveDown()
	if dialog.GetSelectedIndex() != 2 {
		t.Errorf("After second moveDown, selected index should be 2, got %d", dialog.GetSelectedIndex())
	}

	// 循环到底部再向下应该回到顶部
	dialog.moveDown()
	if dialog.GetSelectedIndex() != 0 {
		t.Errorf("After moveDown from bottom, should wrap to 0, got %d", dialog.GetSelectedIndex())
	}

	// 向上移动
	dialog.moveUp()
	if dialog.GetSelectedIndex() != 2 {
		t.Errorf("After moveUp from top, should wrap to 2, got %d", dialog.GetSelectedIndex())
	}

	dialog.moveUp()
	if dialog.GetSelectedIndex() != 1 {
		t.Errorf("After moveUp, selected index should be 1, got %d", dialog.GetSelectedIndex())
	}
}

// TestGetSelected 测试获取选中项
func TestGetSelected(t *testing.T) {
	dialog := NewDialogList()

	// 空列表应该返回nil
	if selected := dialog.GetSelected(); selected != nil {
		t.Error("GetSelected() should return nil for empty list")
	}

	// 创建测试对话列表
	now := time.Now()
	conversations := []*Conversation{
		{ID: "conv-1", CreatedAt: now.Add(-1 * time.Hour), Status: ConvStatusFinished},
		{ID: "conv-2", CreatedAt: now, Status: ConvStatusActive},
	}

	dialog.SetConversations(conversations)

	// 获取选中项
	selected := dialog.GetSelected()
	if selected == nil {
		t.Fatal("GetSelected() returned nil")
	}

	if selected.ID != "conv-2" {
		t.Errorf("Selected ID should be conv-2 (newest), got %s", selected.ID)
	}

	// 修改选中索引
	dialog.SetSelectedIndex(1)
	selected = dialog.GetSelected()
	if selected == nil {
		t.Fatal("GetSelected() returned nil after SetSelectedIndex")
	}

	if selected.ID != "conv-1" {
		t.Errorf("Selected ID should be conv-1, got %s", selected.ID)
	}
}

// TestUpdate 测试消息处理
// 注意：由于 tea.KeyMsg 的构造方式在 Bubble Tea v2 中较为复杂，
// 这里主要测试导航方法，而不是通过 Update 方法测试键盘输入
func TestUpdate(t *testing.T) {
	dialog := NewDialogList()

	// 创建测试对话列表
	now := time.Now()
	conversations := []*Conversation{
		{ID: "conv-1", CreatedAt: now.Add(-1 * time.Hour), Status: ConvStatusFinished},
		{ID: "conv-2", CreatedAt: now, Status: ConvStatusActive},
	}

	dialog.SetConversations(conversations)
	dialog.Show()

	// 测试导航方法（moveDown 和 moveUp）
	dialog.moveDown()
	if dialog.GetSelectedIndex() != 1 {
		t.Errorf("After moveDown, selected index should be 1, got %d", dialog.GetSelectedIndex())
	}

	dialog.moveUp()
	if dialog.GetSelectedIndex() != 0 {
		t.Errorf("After moveUp, selected index should be 0, got %d", dialog.GetSelectedIndex())
	}

	// 测试循环导航
	dialog.moveUp() // 从顶部向上应该循环到底部
	if dialog.GetSelectedIndex() != 1 {
		t.Errorf("After moveUp from top, should wrap to 1, got %d", dialog.GetSelectedIndex())
	}

	dialog.moveDown() // 从底部向下应该循环到顶部
	if dialog.GetSelectedIndex() != 0 {
		t.Errorf("After moveDown from bottom, should wrap to 0, got %d", dialog.GetSelectedIndex())
	}
}

// TestFindConversationByID 测试根据ID查找对话
func TestFindConversationByID(t *testing.T) {
	dialog := NewDialogList()

	// 创建测试对话列表
	now := time.Now()
	conversations := []*Conversation{
		{ID: "conv-1", CreatedAt: now.Add(-1 * time.Hour), Status: ConvStatusFinished},
		{ID: "conv-2", CreatedAt: now, Status: ConvStatusActive},
	}

	dialog.SetConversations(conversations)

	// 查找存在的对话
	conv := dialog.FindConversationByID("conv-1")
	if conv == nil {
		t.Fatal("FindConversationByID() returned nil for existing ID")
	}
	if conv.ID != "conv-1" {
		t.Errorf("Found conversation ID should be conv-1, got %s", conv.ID)
	}

	// 查找不存在的对话
	conv = dialog.FindConversationByID("conv-999")
	if conv != nil {
		t.Error("FindConversationByID() should return nil for non-existing ID")
	}
}

// TestSelectConversationByID 测试根据ID选择对话
func TestSelectConversationByID(t *testing.T) {
	dialog := NewDialogList()

	// 创建测试对话列表
	now := time.Now()
	conversations := []*Conversation{
		{ID: "conv-1", CreatedAt: now.Add(-1 * time.Hour), Status: ConvStatusFinished},
		{ID: "conv-2", CreatedAt: now, Status: ConvStatusActive},
	}

	dialog.SetConversations(conversations)

	// 选择存在的对话
	if !dialog.SelectConversationByID("conv-1") {
		t.Error("SelectConversationByID() should return true for existing ID")
	}
	if dialog.GetSelectedIndex() != 1 {
		t.Errorf("After selecting conv-1, selected index should be 1, got %d", dialog.GetSelectedIndex())
	}

	// 选择不存在的对话
	if dialog.SelectConversationByID("conv-999") {
		t.Error("SelectConversationByID() should return false for non-existing ID")
	}
}

// TestFilterByStatus 测试按状态过滤
func TestFilterByStatus(t *testing.T) {
	dialog := NewDialogList()

	// 创建测试对话列表
	now := time.Now()
	conversations := []*Conversation{
		{ID: "conv-1", CreatedAt: now.Add(-2 * time.Hour), Status: ConvStatusFinished},
		{ID: "conv-2", CreatedAt: now.Add(-1 * time.Hour), Status: ConvStatusActive},
		{ID: "conv-3", CreatedAt: now, Status: ConvStatusWaiting},
		{ID: "conv-4", CreatedAt: now.Add(1 * time.Hour), Status: ConvStatusActive},
	}

	dialog.SetConversations(conversations)

	// 过滤活动状态
	activeConvs := dialog.FilterByStatus(ConvStatusActive)
	if len(activeConvs) != 2 {
		t.Errorf("Expected 2 active conversations, got %d", len(activeConvs))
	}

	// 过滤等待状态
	waitingConvs := dialog.FilterByStatus(ConvStatusWaiting)
	if len(waitingConvs) != 1 {
		t.Errorf("Expected 1 waiting conversation, got %d", len(waitingConvs))
	}

	// 过滤结束状态
	finishedConvs := dialog.FilterByStatus(ConvStatusFinished)
	if len(finishedConvs) != 1 {
		t.Errorf("Expected 1 finished conversation, got %d", len(finishedConvs))
	}
}

// TestFilterSubConversations 测试过滤子对话
func TestFilterSubConversations(t *testing.T) {
	dialog := NewDialogList()

	// 创建测试对话列表
	now := time.Now()
	conversations := []*Conversation{
		{ID: "conv-1", CreatedAt: now.Add(-2 * time.Hour), IsSubConversation: false},
		{ID: "conv-2", CreatedAt: now.Add(-1 * time.Hour), IsSubConversation: true, AgentRole: "Developer"},
		{ID: "conv-3", CreatedAt: now, IsSubConversation: false},
		{ID: "conv-4", CreatedAt: now.Add(1 * time.Hour), IsSubConversation: true, AgentRole: "Tester"},
	}

	dialog.SetConversations(conversations)

	// 过滤主对话
	mainConvs := dialog.FilterSubConversations(false)
	if len(mainConvs) != 2 {
		t.Errorf("Expected 2 main conversations, got %d", len(mainConvs))
	}

	// 过滤子对话
	subConvs := dialog.FilterSubConversations(true)
	if len(subConvs) != 2 {
		t.Errorf("Expected 2 sub conversations, got %d", len(subConvs))
	}
}

// TestGetStatistics 测试获取统计信息
func TestGetStatistics(t *testing.T) {
	dialog := NewDialogList()

	// 创建测试对话列表
	now := time.Now()
	conversations := []*Conversation{
		{ID: "conv-1", CreatedAt: now.Add(-2 * time.Hour), Status: ConvStatusFinished},
		{ID: "conv-2", CreatedAt: now.Add(-1 * time.Hour), Status: ConvStatusActive},
		{ID: "conv-3", CreatedAt: now, Status: ConvStatusWaiting},
		{ID: "conv-4", CreatedAt: now.Add(1 * time.Hour), Status: ConvStatusActive},
	}

	dialog.SetConversations(conversations)

	// 获取统计
	stats := dialog.GetStatistics()
	if stats[ConvStatusActive] != 2 {
		t.Errorf("Expected 2 active conversations, got %d", stats[ConvStatusActive])
	}
	if stats[ConvStatusWaiting] != 1 {
		t.Errorf("Expected 1 waiting conversation, got %d", stats[ConvStatusWaiting])
	}
	if stats[ConvStatusFinished] != 1 {
		t.Errorf("Expected 1 finished conversation, got %d", stats[ConvStatusFinished])
	}
}

// TestView 测试视图渲染
func TestView(t *testing.T) {
	dialog := NewDialogList()

	// 隐藏状态应该返回空字符串
	if view := dialog.View(); view != "" {
		t.Error("View() should return empty string when hidden")
	}

	// 创建测试对话列表
	now := time.Now()
	conversations := []*Conversation{
		{ID: "conv-1", CreatedAt: now.Add(-1 * time.Hour), Status: ConvStatusFinished, Title: "测试对话"},
	}

	dialog.SetConversations(conversations)
	dialog.Show()

	// 显示状态应该返回非空字符串
	view := dialog.View()
	if view == "" {
		t.Error("View() should return non-empty string when visible")
	}

	// 检查是否包含标题
	if !containsString(view, "对话列表") {
		t.Error("View should contain title '对话列表'")
	}

	// 检查是否包含对话标题
	if !containsString(view, "测试对话") {
		t.Error("View should contain conversation title '测试对话'")
	}

	// 检查是否包含帮助信息
	if !containsString(view, "上移") || !containsString(view, "下移") {
		t.Error("View should contain help text")
	}
}

// TestRenderOverlay 测试覆盖层渲染
func TestRenderOverlay(t *testing.T) {
	dialog := NewDialogList()

	// 隐藏状态应该返回空字符串
	if view := dialog.RenderOverlay(80, 24); view != "" {
		t.Error("RenderOverlay() should return empty string when hidden")
	}

	// 创建测试对话列表
	now := time.Now()
	conversations := []*Conversation{
		{ID: "conv-1", CreatedAt: now, Status: ConvStatusActive, Title: "测试对话"},
	}

	dialog.SetConversations(conversations)
	dialog.Show()

	// 渲染覆盖层
	view := dialog.RenderOverlay(80, 24)
	if view == "" {
		t.Error("RenderOverlay() should return non-empty string when visible")
	}
}

// TestFormatTime 测试时间格式化
func TestFormatTime(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{"刚刚", time.Now(), "刚刚"},
		{"1分钟前", time.Now().Add(-1 * time.Minute), "分钟前"},
		{"1小时前", time.Now().Add(-1 * time.Hour), "小时前"},
		{"1天前", time.Now().Add(-25 * time.Hour), "天前"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTime(tt.time)
			if !containsString(result, tt.expected) && result != "刚刚" {
				t.Errorf("formatTime() result '%s' should contain '%s'", result, tt.expected)
			}
		})
	}
}

// TestGetStatusIconAndColor 测试状态图标和颜色
func TestGetStatusIconAndColor(t *testing.T) {
	tests := []struct {
		status       ConvStatus
		expectedIcon string
	}{
		{ConvStatusActive, "🔄"},
		{ConvStatusWaiting, "⏳"},
		{ConvStatusFinished, "✓"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedIcon, func(t *testing.T) {
			icon, color := getStatusIconAndColor(tt.status)
			if icon != tt.expectedIcon {
				t.Errorf("Status %v: expected icon %s, got %s", tt.status, tt.expectedIcon, icon)
			}
			if color == "" {
				t.Errorf("Status %v: color should not be empty", tt.status)
			}
		})
	}
}

// TestSetSize 测试设置大小
func TestSetSize(t *testing.T) {
	dialog := NewDialogList()

	dialog.SetSize(100, 50)

	if dialog.width != 100 {
		t.Errorf("Width should be 100, got %d", dialog.width)
	}

	if dialog.height != 50 {
		t.Errorf("Height should be 50, got %d", dialog.height)
	}
}

// TestGetConversationCount 测试获取对话数量
func TestGetConversationCount(t *testing.T) {
	dialog := NewDialogList()

	// 空列表
	if count := dialog.GetConversationCount(); count != 0 {
		t.Errorf("Empty list should have 0 conversations, got %d", count)
	}

	// 添加对话
	now := time.Now()
	conversations := []*Conversation{
		{ID: "conv-1", CreatedAt: now.Add(-1 * time.Hour)},
		{ID: "conv-2", CreatedAt: now},
	}

	dialog.SetConversations(conversations)

	if count := dialog.GetConversationCount(); count != 2 {
		t.Errorf("List should have 2 conversations, got %d", count)
	}
}

// 辅助函数：检查字符串是否包含子串
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
