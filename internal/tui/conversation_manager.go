// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ConversationManager 对话管理器
// 负责管理对话的创建、状态更新、切换和父子关系
type ConversationManager struct {
	// mu 读写锁，保证并发安全
	mu sync.RWMutex

	// conversations 所有对话的映射（ID -> Conversation）
	conversations map[string]*Conversation

	// rootConversations 根对话列表（非子对话）
	rootConversations []*Conversation

	// activeConvID 当前活动对话ID
	activeConvID string

	// onConversationCreated 对话创建回调
	onConversationCreated func(conv *Conversation)

	// onConversationStatusChanged 对话状态变更回调
	onConversationStatusChanged func(conv *Conversation)

	// onConversationSwitched 对话切换回调
	onConversationSwitched func(conv *Conversation)
}

// NewConversationManager 创建新的对话管理器
func NewConversationManager() *ConversationManager {
	return &ConversationManager{
		conversations:      make(map[string]*Conversation),
		rootConversations:  make([]*Conversation, 0),
	}
}

// CreateConversation 创建新对话
// title: 对话标题（可选）
// 返回创建的对话对象
func (cm *ConversationManager) CreateConversation(title string) *Conversation {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 生成唯一ID
	id := uuid.New().String()

	// 创建对话对象
	conv := &Conversation{
		ID:              id,
		CreatedAt:       time.Now(),
		Status:          ConvStatusActive,
		Messages:        make([]Message, 0),
		TokenUsage:      0,
		Title:           title,
		IsSubConversation: false,
		ParentID:        "",
		AgentRole:       "",
	}

	// 添加到映射和列表
	cm.conversations[id] = conv
	cm.rootConversations = append(cm.rootConversations, conv)

	// 如果没有活动对话，设置为活动对话
	if cm.activeConvID == "" {
		cm.activeConvID = id
	}

	// 触发回调
	if cm.onConversationCreated != nil {
		cm.onConversationCreated(conv)
	}

	return conv
}

// CreateSubConversation 创建子对话
// parentID: 父对话ID
// agentRole: 子代理角色
// task: 任务描述（用作标题）
// 返回创建的子对话对象
func (cm *ConversationManager) CreateSubConversation(parentID, agentRole, task string) (*Conversation, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 检查父对话是否存在
	parent, exists := cm.conversations[parentID]
	if !exists {
		return nil, fmt.Errorf("父对话不存在: %s", parentID)
	}

	// 生成唯一ID
	id := uuid.New().String()

	// 创建子对话对象
	conv := &Conversation{
		ID:              id,
		CreatedAt:       time.Now(),
		Status:          ConvStatusActive,
		Messages:        make([]Message, 0),
		TokenUsage:      0,
		Title:           task,
		IsSubConversation: true,
		ParentID:        parentID,
		AgentRole:       agentRole,
	}

	// 添加到映射
	cm.conversations[id] = conv

	// 更新父对话状态为等待
	parent.Status = ConvStatusWaiting
	if cm.onConversationStatusChanged != nil {
		cm.onConversationStatusChanged(parent)
	}

	// 触发回调
	if cm.onConversationCreated != nil {
		cm.onConversationCreated(conv)
	}

	return conv, nil
}

// UpdateConversationStatus 更新对话状态
// convID: 对话ID
// status: 新状态
func (cm *ConversationManager) UpdateConversationStatus(convID string, status ConvStatus) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conv, exists := cm.conversations[convID]
	if !exists {
		return fmt.Errorf("对话不存在: %s", convID)
	}

	// 更新状态
	oldStatus := conv.Status
	conv.Status = status

	// 如果是子对话且状态变为结束，更新父对话状态
	if conv.IsSubConversation && status == ConvStatusFinished {
		cm.updateParentStatus(conv.ParentID)
	}

	// 触发回调
	if oldStatus != status && cm.onConversationStatusChanged != nil {
		cm.onConversationStatusChanged(conv)
	}

	return nil
}

// updateParentStatus 更新父对话状态
// 当所有子对话都结束时，父对话恢复为活动状态
func (cm *ConversationManager) updateParentStatus(parentID string) {
	parent, exists := cm.conversations[parentID]
	if !exists {
		return
	}

	// 检查是否所有子对话都已结束
	allSubFinished := true
	for _, conv := range cm.conversations {
		if conv.ParentID == parentID && conv.Status != ConvStatusFinished {
			allSubFinished = false
			break
		}
	}

	// 如果所有子对话都结束，父对话恢复活动状态
	if allSubFinished {
		parent.Status = ConvStatusActive
		if cm.onConversationStatusChanged != nil {
			cm.onConversationStatusChanged(parent)
		}
	}
}

// SwitchConversation 切换到指定对话
// convID: 目标对话ID
func (cm *ConversationManager) SwitchConversation(convID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conv, exists := cm.conversations[convID]
	if !exists {
		return fmt.Errorf("对话不存在: %s", convID)
	}

	// 更新活动对话ID
	cm.activeConvID = convID

	// 触发回调
	if cm.onConversationSwitched != nil {
		cm.onConversationSwitched(conv)
	}

	return nil
}

// GetConversation 获取指定对话
func (cm *ConversationManager) GetConversation(convID string) (*Conversation, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conv, exists := cm.conversations[convID]
	if !exists {
		return nil, fmt.Errorf("对话不存在: %s", convID)
	}

	return conv, nil
}

// GetActiveConversation 获取当前活动对话
func (cm *ConversationManager) GetActiveConversation() *Conversation {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.activeConvID == "" {
		return nil
	}

	return cm.conversations[cm.activeConvID]
}

// GetAllConversations 获取所有对话列表
// 返回按创建时间降序排序的对话列表
func (cm *ConversationManager) GetAllConversations() []*Conversation {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 创建结果切片
	result := make([]*Conversation, 0, len(cm.conversations))
	for _, conv := range cm.conversations {
		result = append(result, conv)
	}

	// 按创建时间降序排序
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].CreatedAt.Before(result[j].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// GetRootConversations 获取根对话列表（非子对话）
func (cm *ConversationManager) GetRootConversations() []*Conversation {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 返回副本
	result := make([]*Conversation, len(cm.rootConversations))
	copy(result, cm.rootConversations)
	return result
}

// GetSubConversations 获取指定对话的所有子对话
func (cm *ConversationManager) GetSubConversations(parentID string) []*Conversation {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]*Conversation, 0)
	for _, conv := range cm.conversations {
		if conv.ParentID == parentID {
			result = append(result, conv)
		}
	}
	return result
}

// GetConversationTree 获取对话树（包含父子关系）
// 返回树形结构的对话列表
func (cm *ConversationManager) GetConversationTree() []*ConversationTreeNode {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 构建树形结构
	tree := make([]*ConversationTreeNode, 0)
	for _, conv := range cm.rootConversations {
		node := cm.buildConversationTreeNode(conv)
		tree = append(tree, node)
	}

	return tree
}

// ConversationTreeNode 对话树节点
type ConversationTreeNode struct {
	Conversation *Conversation
	Children     []*ConversationTreeNode
}

// buildConversationTreeNode 构建对话树节点
func (cm *ConversationManager) buildConversationTreeNode(conv *Conversation) *ConversationTreeNode {
	node := &ConversationTreeNode{
		Conversation: conv,
		Children:     make([]*ConversationTreeNode, 0),
	}

	// 查找子对话
	for _, child := range cm.conversations {
		if child.ParentID == conv.ID {
			childNode := cm.buildConversationTreeNode(child)
			node.Children = append(node.Children, childNode)
		}
	}

	return node
}

// DeleteConversation 删除对话
// 如果对话有子对话，也会一并删除
func (cm *ConversationManager) DeleteConversation(convID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conv, exists := cm.conversations[convID]
	if !exists {
		return fmt.Errorf("对话不存在: %s", convID)
	}

	// 递归删除子对话
	cm.deleteConversationRecursive(convID)

	// 如果是根对话，从根对话列表中移除
	if !conv.IsSubConversation {
		for i, rootConv := range cm.rootConversations {
			if rootConv.ID == convID {
				cm.rootConversations = append(cm.rootConversations[:i], cm.rootConversations[i+1:]...)
				break
			}
		}
	}

	// 如果删除的是活动对话，切换到其他对话
	if cm.activeConvID == convID {
		cm.activeConvID = ""
		if len(cm.rootConversations) > 0 {
			cm.activeConvID = cm.rootConversations[0].ID
		}
	}

	return nil
}

// deleteConversationRecursive 递归删除对话及其子对话
func (cm *ConversationManager) deleteConversationRecursive(convID string) {
	// 先删除所有子对话
	for _, conv := range cm.conversations {
		if conv.ParentID == convID {
			cm.deleteConversationRecursive(conv.ID)
		}
	}

	// 删除当前对话
	delete(cm.conversations, convID)
}

// AddMessageToConversation 向对话添加消息
func (cm *ConversationManager) AddMessageToConversation(convID string, msg Message) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conv, exists := cm.conversations[convID]
	if !exists {
		return fmt.Errorf("对话不存在: %s", convID)
	}

	conv.Messages = append(conv.Messages, msg)
	return nil
}

// UpdateTokenUsage 更新对话的 token 用量
func (cm *ConversationManager) UpdateTokenUsage(convID string, usage int) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conv, exists := cm.conversations[convID]
	if !exists {
		return fmt.Errorf("对话不存在: %s", convID)
	}

	conv.TokenUsage += usage
	return nil
}

// SetConversationTitle 设置对话标题
func (cm *ConversationManager) SetConversationTitle(convID, title string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conv, exists := cm.conversations[convID]
	if !exists {
		return fmt.Errorf("对话不存在: %s", convID)
	}

	conv.Title = title
	return nil
}

// GetStatistics 获取统计信息
func (cm *ConversationManager) GetStatistics() map[ConvStatus]int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := make(map[ConvStatus]int)
	for _, conv := range cm.conversations {
		stats[conv.Status]++
	}
	return stats
}

// GetTotalTokenUsage 获取总 token 用量
func (cm *ConversationManager) GetTotalTokenUsage() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	total := 0
	for _, conv := range cm.conversations {
		total += conv.TokenUsage
	}
	return total
}

// SetOnConversationCreated 设置对话创建回调
func (cm *ConversationManager) SetOnConversationCreated(callback func(conv *Conversation)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.onConversationCreated = callback
}

// SetOnConversationStatusChanged 设置对话状态变更回调
func (cm *ConversationManager) SetOnConversationStatusChanged(callback func(conv *Conversation)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.onConversationStatusChanged = callback
}

// SetOnConversationSwitched 设置对话切换回调
func (cm *ConversationManager) SetOnConversationSwitched(callback func(conv *Conversation)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.onConversationSwitched = callback
}

// CompleteConversation 完成对话
// 将对话状态设置为结束，并更新父对话状态
func (cm *ConversationManager) CompleteConversation(convID string) error {
	return cm.UpdateConversationStatus(convID, ConvStatusFinished)
}

// ActivateConversation 激活对话
// 将对话状态设置为活动
func (cm *ConversationManager) ActivateConversation(convID string) error {
	return cm.UpdateConversationStatus(convID, ConvStatusActive)
}

// WaitConversation 设置对话为等待状态
func (cm *ConversationManager) WaitConversation(convID string) error {
	return cm.UpdateConversationStatus(convID, ConvStatusWaiting)
}

// GetCurrentMessage 获取对话的当前消息（用于流式输出）
func (cm *ConversationManager) GetCurrentMessage(convID string) (*Message, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conv, exists := cm.conversations[convID]
	if !exists {
		return nil, fmt.Errorf("对话不存在: %s", convID)
	}

	return conv.currentMessage, nil
}

// SetCurrentMessage 设置对话的当前消息（用于流式输出）
func (cm *ConversationManager) SetCurrentMessage(convID string, msg *Message) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conv, exists := cm.conversations[convID]
	if !exists {
		return fmt.Errorf("对话不存在: %s", convID)
	}

	conv.currentMessage = msg
	return nil
}

// CreateCurrentMessage 创建当前消息（用于流式输出开始）
func (cm *ConversationManager) CreateCurrentMessage(convID string) (*Message, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conv, exists := cm.conversations[convID]
	if !exists {
		return nil, fmt.Errorf("对话不存在: %s", convID)
	}

	msg := &Message{
		Role:      MessageRoleAssistant,
		Timestamp: time.Now(),
	}
	conv.currentMessage = msg
	return msg, nil
}

// FinalizeCurrentMessage 完成当前消息并添加到消息列表
func (cm *ConversationManager) FinalizeCurrentMessage(convID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conv, exists := cm.conversations[convID]
	if !exists {
		return fmt.Errorf("对话不存在: %s", convID)
	}

	if conv.currentMessage == nil {
		return fmt.Errorf("对话没有当前消息: %s", convID)
	}

	// 将当前消息添加到消息列表
	conv.Messages = append(conv.Messages, *conv.currentMessage)
	conv.currentMessage = nil

	return nil
}
