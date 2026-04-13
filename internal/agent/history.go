package agent

import (
	"sync"

	"github.com/Attect/MukaAI/internal/model"
)

// HistoryManager 管理消息历史
// 线程安全，支持并发访问
type HistoryManager struct {
	mu       sync.RWMutex
	messages []model.Message
}

// NewHistoryManager 创建新的消息历史管理器
func NewHistoryManager() *HistoryManager {
	return &HistoryManager{
		messages: make([]model.Message, 0),
	}
}

// AddMessage 添加消息到历史
// 消息会被追加到历史末尾
func (h *HistoryManager) AddMessage(msg model.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = append(h.messages, msg)
}

// AddMessages 批量添加消息
func (h *HistoryManager) AddMessages(msgs []model.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = append(h.messages, msgs...)
}

// GetMessages 获取所有消息的副本
// 返回副本以避免外部修改影响内部状态
func (h *HistoryManager) GetMessages() []model.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 返回副本
	result := make([]model.Message, len(h.messages))
	copy(result, h.messages)
	return result
}

// GetMessagesRef 获取消息的引用（只读）
// 调用者不应修改返回的切片
// 用于性能敏感场景，避免复制
func (h *HistoryManager) GetMessagesRef() []model.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.messages
}

// GetLastMessage 获取最后一条消息
// 如果历史为空，返回空消息和false
func (h *HistoryManager) GetLastMessage() (model.Message, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.messages) == 0 {
		return model.Message{}, false
	}
	return h.messages[len(h.messages)-1], true
}

// GetMessageCount 获取消息数量
func (h *HistoryManager) GetMessageCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.messages)
}

// Clear 清空消息历史
func (h *HistoryManager) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = make([]model.Message, 0)
}

// Truncate 截断历史以适应token限制
// 保留系统消息和最近的对话
// maxTokens: 最大token数量
// tokenCounter: token计数函数
func (h *HistoryManager) Truncate(maxTokens int, tokenCounter func([]model.Message) int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.messages) == 0 {
		return
	}

	// 分离系统消息和非系统消息
	var systemMessages []model.Message
	var otherMessages []model.Message

	for _, msg := range h.messages {
		if msg.Role == model.RoleSystem {
			systemMessages = append(systemMessages, msg)
		} else {
			otherMessages = append(otherMessages, msg)
		}
	}

	// 计算系统消息的token数
	systemTokens := tokenCounter(systemMessages)
	remainingTokens := maxTokens - systemTokens

	if remainingTokens <= 0 {
		// 系统消息已经超出限制，只保留第一条系统消息
		if len(systemMessages) > 0 {
			h.messages = []model.Message{systemMessages[0]}
		} else {
			h.messages = make([]model.Message, 0)
		}
		return
	}

	// 从最新的消息开始保留
	var keptMessages []model.Message
	currentTokens := 0

	// 倒序遍历非系统消息
	for i := len(otherMessages) - 1; i >= 0; i-- {
		msgTokens := tokenCounter([]model.Message{otherMessages[i]})
		if currentTokens+msgTokens > remainingTokens {
			break
		}
		// 将消息插入到开头（保持顺序）
		keptMessages = append([]model.Message{otherMessages[i]}, keptMessages...)
		currentTokens += msgTokens
	}

	// 合并系统消息和保留的消息
	h.messages = append(systemMessages, keptMessages...)
}

// TruncateSimple 简单截断策略
// 保留系统消息和最近N条消息
func (h *HistoryManager) TruncateSimple(keepCount int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.messages) <= keepCount {
		return
	}

	// 分离系统消息和非系统消息
	var systemMessages []model.Message
	var otherMessages []model.Message

	for _, msg := range h.messages {
		if msg.Role == model.RoleSystem {
			systemMessages = append(systemMessages, msg)
		} else {
			otherMessages = append(otherMessages, msg)
		}
	}

	// 保留最近的N条非系统消息
	start := 0
	if len(otherMessages) > keepCount {
		start = len(otherMessages) - keepCount
	}
	keptMessages := otherMessages[start:]

	// 合并
	h.messages = append(systemMessages, keptMessages...)
}

// GetTokenCount 获取当前历史的token数量
func (h *HistoryManager) GetTokenCount(tokenCounter func([]model.Message) int) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return tokenCounter(h.messages)
}

// GetMessagesByRole 获取指定角色的消息
func (h *HistoryManager) GetMessagesByRole(role model.Role) []model.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []model.Message
	for _, msg := range h.messages {
		if msg.Role == role {
			result = append(result, msg)
		}
	}
	return result
}

// RemoveLastMessage 移除最后一条消息
// 用于回滚错误的消息
func (h *HistoryManager) RemoveLastMessage() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.messages) == 0 {
		return false
	}

	h.messages = h.messages[:len(h.messages)-1]
	return true
}

// ReplaceLastMessage 替换最后一条消息
func (h *HistoryManager) ReplaceLastMessage(msg model.Message) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.messages) == 0 {
		return false
	}

	h.messages[len(h.messages)-1] = msg
	return true
}

// GetRecentMessages 获取最近N条消息
func (h *HistoryManager) GetRecentMessages(n int) []model.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if n <= 0 {
		return []model.Message{}
	}

	if n >= len(h.messages) {
		result := make([]model.Message, len(h.messages))
		copy(result, h.messages)
		return result
	}

	start := len(h.messages) - n
	result := make([]model.Message, n)
	copy(result, h.messages[start:])
	return result
}

// Clone 克隆历史管理器
// 返回一个新的独立副本
func (h *HistoryManager) Clone() *HistoryManager {
	h.mu.RLock()
	defer h.mu.RUnlock()

	newManager := &HistoryManager{
		messages: make([]model.Message, len(h.messages)),
	}
	copy(newManager.messages, h.messages)
	return newManager
}
