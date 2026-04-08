// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"sync"
	"testing"
	"time"
)

// TestNewConversationManager 测试创建对话管理器
func TestNewConversationManager(t *testing.T) {
	cm := NewConversationManager()

	if cm == nil {
		t.Fatal("NewConversationManager() returned nil")
	}
	if cm.conversations == nil {
		t.Error("conversations map should be initialized")
	}
	if cm.rootConversations == nil {
		t.Error("rootConversations slice should be initialized")
	}
}

// TestCreateConversation 测试创建对话
func TestCreateConversation(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("测试对话")

	if conv == nil {
		t.Fatal("CreateConversation() returned nil")
	}
	if conv.ID == "" {
		t.Error("Conversation ID should not be empty")
	}
	if conv.Title != "测试对话" {
		t.Errorf("Expected title '测试对话', got %q", conv.Title)
	}
	if conv.Status != ConvStatusActive {
		t.Errorf("Expected status %v, got %v", ConvStatusActive, conv.Status)
	}
	if conv.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if len(conv.Messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(conv.Messages))
	}
}

// TestCreateConversationWithoutTitle 测试创建无标题对话
func TestCreateConversationWithoutTitle(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("")

	if conv == nil {
		t.Fatal("CreateConversation() returned nil")
	}
	if conv.Title != "" {
		t.Errorf("Expected empty title, got %q", conv.Title)
	}
}

// TestCreateSubConversation 测试创建子对话
func TestCreateSubConversation(t *testing.T) {
	cm := NewConversationManager()

	// 创建父对话
	parent := cm.CreateConversation("父对话")

	// 创建子对话
	subConv, err := cm.CreateSubConversation(parent.ID, "Developer", "子任务")
	if err != nil {
		t.Fatalf("CreateSubConversation() failed: %v", err)
	}

	if subConv == nil {
		t.Fatal("CreateSubConversation() returned nil")
	}
	if !subConv.IsSubConversation {
		t.Error("SubConversation should be marked as sub-conversation")
	}
	if subConv.ParentID != parent.ID {
		t.Errorf("Expected parent ID %q, got %q", parent.ID, subConv.ParentID)
	}
	if subConv.AgentRole != "Developer" {
		t.Errorf("Expected agent role 'Developer', got %q", subConv.AgentRole)
	}
	if subConv.Title != "子任务" {
		t.Errorf("Expected title '子任务', got %q", subConv.Title)
	}

	// 检查父对话状态是否更新为等待
	updatedParent, _ := cm.GetConversation(parent.ID)
	if updatedParent.Status != ConvStatusWaiting {
		t.Errorf("Parent status should be waiting, got %v", updatedParent.Status)
	}
}

// TestCreateSubConversationWithInvalidParent 测试使用无效父对话创建子对话
func TestCreateSubConversationWithInvalidParent(t *testing.T) {
	cm := NewConversationManager()

	_, err := cm.CreateSubConversation("invalid-id", "Developer", "子任务")
	if err == nil {
		t.Error("Expected error for invalid parent ID")
	}
}

// TestUpdateConversationStatus 测试更新对话状态
func TestUpdateConversationStatus(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("测试对话")

	// 更新状态
	err := cm.UpdateConversationStatus(conv.ID, ConvStatusFinished)
	if err != nil {
		t.Fatalf("UpdateConversationStatus() failed: %v", err)
	}

	// 检查状态已更新
	updatedConv, _ := cm.GetConversation(conv.ID)
	if updatedConv.Status != ConvStatusFinished {
		t.Errorf("Expected status %v, got %v", ConvStatusFinished, updatedConv.Status)
	}
}

// TestUpdateConversationStatusWithInvalidID 测试使用无效ID更新状态
func TestUpdateConversationStatusWithInvalidID(t *testing.T) {
	cm := NewConversationManager()

	err := cm.UpdateConversationStatus("invalid-id", ConvStatusFinished)
	if err == nil {
		t.Error("Expected error for invalid conversation ID")
	}
}

// TestSwitchConversation 测试切换对话
func TestSwitchConversation(t *testing.T) {
	cm := NewConversationManager()

	conv1 := cm.CreateConversation("对话1")
	conv2 := cm.CreateConversation("对话2")

	// 切换到对话2
	err := cm.SwitchConversation(conv2.ID)
	if err != nil {
		t.Fatalf("SwitchConversation() failed: %v", err)
	}

	// 检查活动对话
	activeConv := cm.GetActiveConversation()
	if activeConv.ID != conv2.ID {
		t.Errorf("Expected active conversation ID %q, got %q", conv2.ID, activeConv.ID)
	}

	// 切换回对话1
	err = cm.SwitchConversation(conv1.ID)
	if err != nil {
		t.Fatalf("SwitchConversation() failed: %v", err)
	}

	activeConv = cm.GetActiveConversation()
	if activeConv.ID != conv1.ID {
		t.Errorf("Expected active conversation ID %q, got %q", conv1.ID, activeConv.ID)
	}
}

// TestSwitchConversationWithInvalidID 测试使用无效ID切换对话
func TestSwitchConversationWithInvalidID(t *testing.T) {
	cm := NewConversationManager()

	err := cm.SwitchConversation("invalid-id")
	if err == nil {
		t.Error("Expected error for invalid conversation ID")
	}
}

// TestGetConversation 测试获取对话
func TestGetConversation(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("测试对话")

	// 获取对话
	retrievedConv, err := cm.GetConversation(conv.ID)
	if err != nil {
		t.Fatalf("GetConversation() failed: %v", err)
	}

	if retrievedConv.ID != conv.ID {
		t.Errorf("Expected conversation ID %q, got %q", conv.ID, retrievedConv.ID)
	}
}

// TestGetConversationWithInvalidID 测试使用无效ID获取对话
func TestGetConversationWithInvalidID(t *testing.T) {
	cm := NewConversationManager()

	_, err := cm.GetConversation("invalid-id")
	if err == nil {
		t.Error("Expected error for invalid conversation ID")
	}
}

// TestGetActiveConversation 测试获取活动对话
func TestGetActiveConversation(t *testing.T) {
	cm := NewConversationManager()

	// 没有对话时
	activeConv := cm.GetActiveConversation()
	if activeConv != nil {
		t.Error("Expected nil when no conversations exist")
	}

	// 创建第一个对话（自动成为活动对话）
	conv1 := cm.CreateConversation("对话1")
	activeConv = cm.GetActiveConversation()
	if activeConv.ID != conv1.ID {
		t.Errorf("Expected active conversation ID %q, got %q", conv1.ID, activeConv.ID)
	}

	// 创建第二个对话（不会自动成为活动对话）
	conv2 := cm.CreateConversation("对话2")
	_ = conv2 // 避免未使用变量警告
	activeConv = cm.GetActiveConversation()
	if activeConv.ID != conv1.ID {
		t.Error("Active conversation should not change after creating new conversation")
	}
}

// TestGetAllConversations 测试获取所有对话
func TestGetAllConversations(t *testing.T) {
	cm := NewConversationManager()

	// 创建多个对话
	conv1 := cm.CreateConversation("对话1")
	time.Sleep(10 * time.Millisecond) // 确保时间不同
	_ = cm.CreateConversation("对话2")
	time.Sleep(10 * time.Millisecond)
	conv3 := cm.CreateConversation("对话3")

	// 获取所有对话
	allConvs := cm.GetAllConversations()
	if len(allConvs) != 3 {
		t.Errorf("Expected 3 conversations, got %d", len(allConvs))
	}

	// 检查排序（按创建时间降序）
	if allConvs[0].ID != conv3.ID {
		t.Error("First conversation should be the newest")
	}
	if allConvs[2].ID != conv1.ID {
		t.Error("Third conversation should be the oldest")
	}
}

// TestGetRootConversations 测试获取根对话
func TestGetRootConversations(t *testing.T) {
	cm := NewConversationManager()

	// 创建根对话
	root1 := cm.CreateConversation("根对话1")
	_ = root1 // 避免未使用变量警告
	root2 := cm.CreateConversation("根对话2")
	_ = root2 // 避免未使用变量警告

	// 创建子对话
	cm.CreateSubConversation(root1.ID, "Developer", "子对话1")

	// 获取根对话
	rootConvs := cm.GetRootConversations()
	if len(rootConvs) != 2 {
		t.Errorf("Expected 2 root conversations, got %d", len(rootConvs))
	}
}

// TestGetSubConversations 测试获取子对话
func TestGetSubConversations(t *testing.T) {
	cm := NewConversationManager()

	// 创建父对话
	parent := cm.CreateConversation("父对话")

	// 创建子对话
	sub1, _ := cm.CreateSubConversation(parent.ID, "Developer", "子对话1")
	sub2, _ := cm.CreateSubConversation(parent.ID, "Architect", "子对话2")

	// 获取子对话
	subConvs := cm.GetSubConversations(parent.ID)
	if len(subConvs) != 2 {
		t.Errorf("Expected 2 sub-conversations, got %d", len(subConvs))
	}

	// 检查子对话ID
	subIDs := make(map[string]bool)
	for _, sub := range subConvs {
		subIDs[sub.ID] = true
	}
	if !subIDs[sub1.ID] || !subIDs[sub2.ID] {
		t.Error("Missing expected sub-conversations")
	}
}

// TestGetConversationTree 测试获取对话树
func TestGetConversationTree(t *testing.T) {
	cm := NewConversationManager()

	// 创建对话树
	root := cm.CreateConversation("根对话")
	sub1, _ := cm.CreateSubConversation(root.ID, "Developer", "子对话1")
	cm.CreateSubConversation(sub1.ID, "Tester", "子子对话") // 嵌套子对话
	cm.CreateSubConversation(root.ID, "Architect", "子对话2")

	// 获取对话树
	tree := cm.GetConversationTree()
	if len(tree) != 1 {
		t.Errorf("Expected 1 root node, got %d", len(tree))
	}

	// 检查根节点
	rootNode := tree[0]
	if rootNode.Conversation.ID != root.ID {
		t.Error("Root node should match root conversation")
	}
	if len(rootNode.Children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(rootNode.Children))
	}

	// 检查嵌套子对话
	for _, child := range rootNode.Children {
		if child.Conversation.ID == sub1.ID {
			if len(child.Children) != 1 {
				t.Errorf("Expected 1 nested child, got %d", len(child.Children))
			}
		}
	}
}

// TestDeleteConversation 测试删除对话
func TestDeleteConversation(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("测试对话")

	// 删除对话
	err := cm.DeleteConversation(conv.ID)
	if err != nil {
		t.Fatalf("DeleteConversation() failed: %v", err)
	}

	// 检查对话已删除
	_, err = cm.GetConversation(conv.ID)
	if err == nil {
		t.Error("Conversation should be deleted")
	}

	// 检查根对话列表
	rootConvs := cm.GetRootConversations()
	if len(rootConvs) != 0 {
		t.Errorf("Expected 0 root conversations, got %d", len(rootConvs))
	}
}

// TestDeleteConversationWithSubConversations 测试删除带子对话的对话
func TestDeleteConversationWithSubConversations(t *testing.T) {
	cm := NewConversationManager()

	// 创建对话树
	root := cm.CreateConversation("根对话")
	sub1, _ := cm.CreateSubConversation(root.ID, "Developer", "子对话1")
	sub2, _ := cm.CreateSubConversation(root.ID, "Architect", "子对话2")

	// 删除根对话
	err := cm.DeleteConversation(root.ID)
	if err != nil {
		t.Fatalf("DeleteConversation() failed: %v", err)
	}

	// 检查所有对话都已删除
	_, err = cm.GetConversation(root.ID)
	if err == nil {
		t.Error("Root conversation should be deleted")
	}
	_, err = cm.GetConversation(sub1.ID)
	if err == nil {
		t.Error("Sub-conversation 1 should be deleted")
	}
	_, err = cm.GetConversation(sub2.ID)
	if err == nil {
		t.Error("Sub-conversation 2 should be deleted")
	}
}

// TestDeleteConversationWithInvalidID 测试使用无效ID删除对话
func TestDeleteConversationWithInvalidID(t *testing.T) {
	cm := NewConversationManager()

	err := cm.DeleteConversation("invalid-id")
	if err == nil {
		t.Error("Expected error for invalid conversation ID")
	}
}

// TestAddMessageToConversation 测试向对话添加消息
func TestAddMessageToConversation(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("测试对话")

	// 添加消息
	msg := Message{
		Role:      MessageRoleUser,
		Content:   "测试消息",
		Timestamp: time.Now(),
	}
	err := cm.AddMessageToConversation(conv.ID, msg)
	if err != nil {
		t.Fatalf("AddMessageToConversation() failed: %v", err)
	}

	// 检查消息已添加
	updatedConv, _ := cm.GetConversation(conv.ID)
	if len(updatedConv.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(updatedConv.Messages))
	}
	if updatedConv.Messages[0].Content != "测试消息" {
		t.Errorf("Expected message content '测试消息', got %q", updatedConv.Messages[0].Content)
	}
}

// TestUpdateTokenUsage 测试更新 token 用量
func TestUpdateTokenUsage(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("测试对话")

	// 更新 token 用量
	err := cm.UpdateTokenUsage(conv.ID, 100)
	if err != nil {
		t.Fatalf("UpdateTokenUsage() failed: %v", err)
	}

	// 检查用量已更新
	updatedConv, _ := cm.GetConversation(conv.ID)
	if updatedConv.TokenUsage != 100 {
		t.Errorf("Expected token usage 100, got %d", updatedConv.TokenUsage)
	}

	// 再次更新
	err = cm.UpdateTokenUsage(conv.ID, 50)
	if err != nil {
		t.Fatalf("UpdateTokenUsage() failed: %v", err)
	}

	updatedConv, _ = cm.GetConversation(conv.ID)
	if updatedConv.TokenUsage != 150 {
		t.Errorf("Expected token usage 150, got %d", updatedConv.TokenUsage)
	}
}

// TestSetConversationTitle 测试设置对话标题
func TestSetConversationTitle(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("原标题")

	// 设置标题
	err := cm.SetConversationTitle(conv.ID, "新标题")
	if err != nil {
		t.Fatalf("SetConversationTitle() failed: %v", err)
	}

	// 检查标题已更新
	updatedConv, _ := cm.GetConversation(conv.ID)
	if updatedConv.Title != "新标题" {
		t.Errorf("Expected title '新标题', got %q", updatedConv.Title)
	}
}

// TestGetStatistics 测试获取统计信息
func TestGetStatistics(t *testing.T) {
	cm := NewConversationManager()

	// 创建不同状态的对话
	conv1 := cm.CreateConversation("对话1")
	_ = conv1 // 避免未使用变量警告
	conv2 := cm.CreateConversation("对话2")
	conv3 := cm.CreateConversation("对话3")

	cm.UpdateConversationStatus(conv2.ID, ConvStatusWaiting)
	cm.UpdateConversationStatus(conv3.ID, ConvStatusFinished)

	// 获取统计
	stats := cm.GetStatistics()
	if stats[ConvStatusActive] != 1 {
		t.Errorf("Expected 1 active conversation, got %d", stats[ConvStatusActive])
	}
	if stats[ConvStatusWaiting] != 1 {
		t.Errorf("Expected 1 waiting conversation, got %d", stats[ConvStatusWaiting])
	}
	if stats[ConvStatusFinished] != 1 {
		t.Errorf("Expected 1 finished conversation, got %d", stats[ConvStatusFinished])
	}
}

// TestGetTotalTokenUsage 测试获取总 token 用量
func TestGetTotalTokenUsage(t *testing.T) {
	cm := NewConversationManager()

	conv1 := cm.CreateConversation("对话1")
	conv2 := cm.CreateConversation("对话2")

	cm.UpdateTokenUsage(conv1.ID, 100)
	cm.UpdateTokenUsage(conv2.ID, 200)

	// 获取总用量
	total := cm.GetTotalTokenUsage()
	if total != 300 {
		t.Errorf("Expected total token usage 300, got %d", total)
	}
}

// TestConversationCallbacks 测试对话回调
func TestConversationCallbacks(t *testing.T) {
	cm := NewConversationManager()

	var createdConv *Conversation
	var statusChangedConv *Conversation
	var switchedConv *Conversation

	// 设置回调
	cm.SetOnConversationCreated(func(conv *Conversation) {
		createdConv = conv
	})
	cm.SetOnConversationStatusChanged(func(conv *Conversation) {
		statusChangedConv = conv
	})
	cm.SetOnConversationSwitched(func(conv *Conversation) {
		switchedConv = conv
	})

	// 创建对话（触发创建回调）
	conv := cm.CreateConversation("测试对话")
	if createdConv == nil || createdConv.ID != conv.ID {
		t.Error("OnConversationCreated callback not called correctly")
	}

	// 更新状态（触发状态变更回调）
	cm.UpdateConversationStatus(conv.ID, ConvStatusFinished)
	if statusChangedConv == nil || statusChangedConv.ID != conv.ID {
		t.Error("OnConversationStatusChanged callback not called correctly")
	}

	// 切换对话（触发切换回调）
	conv2 := cm.CreateConversation("对话2")
	_ = cm.SwitchConversation(conv2.ID)
	if switchedConv == nil || switchedConv.ID != conv2.ID {
		t.Error("OnConversationSwitched callback not called correctly")
	}
}

// TestCompleteConversation 测试完成对话
func TestCompleteConversation(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("测试对话")

	err := cm.CompleteConversation(conv.ID)
	if err != nil {
		t.Fatalf("CompleteConversation() failed: %v", err)
	}

	updatedConv, _ := cm.GetConversation(conv.ID)
	if updatedConv.Status != ConvStatusFinished {
		t.Errorf("Expected status %v, got %v", ConvStatusFinished, updatedConv.Status)
	}
}

// TestActivateConversation 测试激活对话
func TestActivateConversation(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("测试对话")
	cm.UpdateConversationStatus(conv.ID, ConvStatusFinished)

	err := cm.ActivateConversation(conv.ID)
	if err != nil {
		t.Fatalf("ActivateConversation() failed: %v", err)
	}

	updatedConv, _ := cm.GetConversation(conv.ID)
	if updatedConv.Status != ConvStatusActive {
		t.Errorf("Expected status %v, got %v", ConvStatusActive, updatedConv.Status)
	}
}

// TestWaitConversation 测试等待对话
func TestWaitConversation(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("测试对话")

	err := cm.WaitConversation(conv.ID)
	if err != nil {
		t.Fatalf("WaitConversation() failed: %v", err)
	}

	updatedConv, _ := cm.GetConversation(conv.ID)
	if updatedConv.Status != ConvStatusWaiting {
		t.Errorf("Expected status %v, got %v", ConvStatusWaiting, updatedConv.Status)
	}
}

// TestCurrentMessage 测试当前消息（流式输出）
func TestCurrentMessage(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("测试对话")

	// 创建当前消息
	msg, err := cm.CreateCurrentMessage(conv.ID)
	if err != nil {
		t.Fatalf("CreateCurrentMessage() failed: %v", err)
	}

	if msg == nil {
		t.Fatal("CreateCurrentMessage() returned nil")
	}
	if msg.Role != MessageRoleAssistant {
		t.Errorf("Expected role %v, got %v", MessageRoleAssistant, msg.Role)
	}

	// 获取当前消息
	currentMsg, err := cm.GetCurrentMessage(conv.ID)
	if err != nil {
		t.Fatalf("GetCurrentMessage() failed: %v", err)
	}
	if currentMsg != msg {
		t.Error("Current message should match created message")
	}

	// 完成当前消息
	err = cm.FinalizeCurrentMessage(conv.ID)
	if err != nil {
		t.Fatalf("FinalizeCurrentMessage() failed: %v", err)
	}

	// 检查消息已添加到列表
	updatedConv, _ := cm.GetConversation(conv.ID)
	if len(updatedConv.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(updatedConv.Messages))
	}

	// 当前消息应该被清空
	currentMsg, _ = cm.GetCurrentMessage(conv.ID)
	if currentMsg != nil {
		t.Error("Current message should be nil after finalization")
	}
}

// TestParentStatusUpdate 测试父对话状态更新
func TestParentStatusUpdate(t *testing.T) {
	cm := NewConversationManager()

	// 创建父对话
	parent := cm.CreateConversation("父对话")

	// 创建子对话（父对话状态变为等待）
	sub1, _ := cm.CreateSubConversation(parent.ID, "Developer", "子对话1")

	updatedParent, _ := cm.GetConversation(parent.ID)
	if updatedParent.Status != ConvStatusWaiting {
		t.Errorf("Parent status should be waiting, got %v", updatedParent.Status)
	}

	// 完成第一个子对话（父对话仍应为等待，因为还有其他子对话）
	cm.CompleteConversation(sub1.ID)

	updatedParent, _ = cm.GetConversation(parent.ID)
	if updatedParent.Status != ConvStatusActive {
		t.Errorf("Parent status should be active when all sub-conversations finished, got %v", updatedParent.Status)
	}
}

// TestConversationManagerConcurrency 测试并发安全性
func TestConversationManagerConcurrency(t *testing.T) {
	cm := NewConversationManager()

	var wg sync.WaitGroup
	numOps := 100

	// 并发创建对话
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cm.CreateConversation("并发对话")
		}()
	}

	// 并发获取对话
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cm.GetAllConversations()
		}()
	}

	// 并发获取统计
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cm.GetStatistics()
		}()
	}

	wg.Wait()

	// 检查最终状态
	allConvs := cm.GetAllConversations()
	if len(allConvs) != numOps {
		t.Errorf("Expected %d conversations, got %d", numOps, len(allConvs))
	}
}

// TestDeleteActiveConversation 测试删除活动对话
func TestDeleteActiveConversation(t *testing.T) {
	cm := NewConversationManager()

	conv1 := cm.CreateConversation("对话1")
	conv2 := cm.CreateConversation("对话2")

	// 设置对话1为活动对话
	_ = cm.SwitchConversation(conv1.ID)

	// 删除活动对话
	_ = cm.DeleteConversation(conv1.ID)

	// 检查活动对话已切换
	activeConv := cm.GetActiveConversation()
	if activeConv == nil {
		t.Error("Should have an active conversation after deleting the active one")
	} else if activeConv.ID != conv2.ID {
		t.Errorf("Expected active conversation ID %q, got %q", conv2.ID, activeConv.ID)
	}
}

// TestDeleteLastConversation 测试删除最后一个对话
func TestDeleteLastConversation(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("唯一对话")

	// 删除唯一对话
	_ = cm.DeleteConversation(conv.ID)

	// 检查没有活动对话
	activeConv := cm.GetActiveConversation()
	if activeConv != nil {
		t.Error("Should have no active conversation after deleting the last one")
	}
}
