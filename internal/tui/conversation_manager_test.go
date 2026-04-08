// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"testing"
	"time"
)

// TestConversationManager_CreateConversation 测试创建对话
func TestConversationManager_CreateConversation(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("Test Conversation")

	if conv == nil {
		t.Fatal("CreateConversation returned nil")
	}

	if conv.ID == "" {
		t.Error("Conversation ID should not be empty")
	}

	if conv.Title != "Test Conversation" {
		t.Errorf("Expected title 'Test Conversation', got '%s'", conv.Title)
	}

	if conv.Status != ConvStatusActive {
		t.Errorf("Expected status ConvStatusActive, got %d", conv.Status)
	}

	if conv.TokenUsage != 0 {
		t.Errorf("Expected initial TokenUsage 0, got %d", conv.TokenUsage)
	}
}

// TestConversationManager_CreateSubConversation 测试创建子对话
func TestConversationManager_CreateSubConversation(t *testing.T) {
	cm := NewConversationManager()

	// 创建父对话
	parent := cm.CreateConversation("Parent Conversation")

	// 创建子对话
	child, err := cm.CreateSubConversation(parent.ID, "Developer", "Implement feature X")
	if err != nil {
		t.Fatalf("CreateSubConversation failed: %v", err)
	}

	if child == nil {
		t.Fatal("CreateSubConversation returned nil")
	}

	if !child.IsSubConversation {
		t.Error("Child should be marked as sub-conversation")
	}

	if child.ParentID != parent.ID {
		t.Errorf("Expected ParentID %s, got %s", parent.ID, child.ParentID)
	}

	if child.AgentRole != "Developer" {
		t.Errorf("Expected AgentRole 'Developer', got '%s'", child.AgentRole)
	}

	// 验证父对话状态更新为等待
	updatedParent, _ := cm.GetConversation(parent.ID)
	if updatedParent.Status != ConvStatusWaiting {
		t.Errorf("Parent status should be ConvStatusWaiting, got %d", updatedParent.Status)
	}
}

// TestConversationManager_UpdateTokenUsage 测试更新 token 用量
func TestConversationManager_UpdateTokenUsage(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("Test")

	// 更新 token 用量
	err := cm.UpdateTokenUsage(conv.ID, 100)
	if err != nil {
		t.Fatalf("UpdateTokenUsage failed: %v", err)
	}

	// 验证更新
	updated, _ := cm.GetConversation(conv.ID)
	if updated.TokenUsage != 100 {
		t.Errorf("Expected TokenUsage 100, got %d", updated.TokenUsage)
	}

	// 再次更新
	err = cm.UpdateTokenUsage(conv.ID, 50)
	if err != nil {
		t.Fatalf("UpdateTokenUsage failed: %v", err)
	}

	updated, _ = cm.GetConversation(conv.ID)
	if updated.TokenUsage != 150 {
		t.Errorf("Expected TokenUsage 150, got %d", updated.TokenUsage)
	}
}

// TestConversationManager_GetTotalTokenUsage 测试获取总 token 用量
func TestConversationManager_GetTotalTokenUsage(t *testing.T) {
	cm := NewConversationManager()

	// 创建多个对话
	conv1 := cm.CreateConversation("Conv1")
	conv2 := cm.CreateConversation("Conv2")

	// 更新 token 用量
	cm.UpdateTokenUsage(conv1.ID, 100)
	cm.UpdateTokenUsage(conv2.ID, 200)

	// 获取总用量
	total := cm.GetTotalTokenUsage()
	if total != 300 {
		t.Errorf("Expected total TokenUsage 300, got %d", total)
	}
}

// TestConversationManager_SwitchConversation 测试切换对话
func TestConversationManager_SwitchConversation(t *testing.T) {
	cm := NewConversationManager()

	conv1 := cm.CreateConversation("Conv1")
	conv2 := cm.CreateConversation("Conv2")

	// 初始活动对话应该是第一个
	active := cm.GetActiveConversation()
	if active.ID != conv1.ID {
		t.Errorf("Expected active conversation %s, got %s", conv1.ID, active.ID)
	}

	// 切换到第二个对话
	err := cm.SwitchConversation(conv2.ID)
	if err != nil {
		t.Fatalf("SwitchConversation failed: %v", err)
	}

	active = cm.GetActiveConversation()
	if active.ID != conv2.ID {
		t.Errorf("Expected active conversation %s, got %s", conv2.ID, active.ID)
	}
}

// TestConversationManager_AddMessageToConversation 测试添加消息
func TestConversationManager_AddMessageToConversation(t *testing.T) {
	cm := NewConversationManager()

	conv := cm.CreateConversation("Test")

	msg := Message{
		Role:      MessageRoleUser,
		Content:   "Hello",
		Timestamp: time.Now(),
	}

	err := cm.AddMessageToConversation(conv.ID, msg)
	if err != nil {
		t.Fatalf("AddMessageToConversation failed: %v", err)
	}

	// 验证消息已添加
	updated, _ := cm.GetConversation(conv.ID)
	if len(updated.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(updated.Messages))
	}

	if updated.Messages[0].Content != "Hello" {
		t.Errorf("Expected message content 'Hello', got '%s'", updated.Messages[0].Content)
	}
}
