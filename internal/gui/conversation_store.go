// Package gui 提供Wails GUI应用的对话持久化存储
package gui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// conversationFile 对话持久化文件格式，包含版本号以便未来扩展
type conversationFile struct {
	Version    int           `json:"version"`
	ID         string        `json:"id"`
	Title      string        `json:"title"`
	CreatedAt  time.Time     `json:"createdAt"`
	Status     string        `json:"status"`
	TokenUsage int           `json:"tokenUsage"`
	Messages   []messageFile `json:"messages"`
}

// messageFile 持久化的消息格式
type messageFile struct {
	Role          string     `json:"role"`
	Content       string     `json:"content"`
	Thinking      string     `json:"thinking"`
	ToolCalls     []ToolCall `json:"toolCalls"`
	TokenUsage    int        `json:"tokenUsage"`
	Timestamp     time.Time  `json:"timestamp"`
	StreamingType string     `json:"streamingType,omitempty"`
}

const (
	// conversationFileVersion 当前持久化格式版本号
	conversationFileVersion = 1
	// debounceInterval 防抖间隔，同一对话在此时间内多次变更只保存最后一次
	debounceInterval = 500 * time.Millisecond
)

// ConversationStore 对话持久化存储
// 负责将对话数据异步保存到磁盘，使用防抖机制减少IO频率
// 每个对话保存为一个独立的JSON文件，格式包含版本号以便未来扩展
type ConversationStore struct {
	dir    string                       // 存储目录
	mu     sync.Mutex                   // 保护timers和buffer
	timers map[string]*time.Timer       // 每个对话的防抖定时器
	buffer map[string]*conversationFile // 待保存的快照缓冲区（不可变快照）
}

// NewConversationStore 创建新的对话存储实例
// dir: 对话文件存储目录，不存在时会自动创建
func NewConversationStore(dir string) (*ConversationStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建对话存储目录失败: %w", err)
	}
	return &ConversationStore{
		dir:    dir,
		timers: make(map[string]*time.Timer),
		buffer: make(map[string]*conversationFile),
	}, nil
}

// SaveConversation 异步保存对话（防抖500ms）
// 调用方应持有App.mu锁以保证快照一致性
// 同一对话在500ms内多次调用只会保存最后一次
func (s *ConversationStore) SaveConversation(conv *conversation) {
	if conv == nil || conv.id == "" {
		return
	}
	// 立即创建不可变快照，确保后续修改不影响待写入的数据
	snapshot := s.toSnapshot(conv)

	s.mu.Lock()
	s.buffer[conv.id] = snapshot
	// 如果已有定时器，先停止（防抖核心逻辑：取消上一次，重新计时）
	if timer, exists := s.timers[conv.id]; exists {
		timer.Stop()
	}
	// 创建新的防抖定时器
	s.timers[conv.id] = time.AfterFunc(debounceInterval, func() {
		s.flush(conv.id)
	})
	s.mu.Unlock()
}

// toSnapshot 将内部conversation结构转换为可序列化的不可变快照
func (s *ConversationStore) toSnapshot(conv *conversation) *conversationFile {
	msgs := make([]messageFile, 0, len(conv.messages))
	for _, m := range conv.messages {
		toolCalls := m.toolCalls
		if toolCalls == nil {
			toolCalls = make([]ToolCall, 0)
		}
		msgs = append(msgs, messageFile{
			Role:          m.role,
			Content:       m.content,
			Thinking:      m.thinking,
			ToolCalls:     toolCalls,
			TokenUsage:    m.tokenUsage,
			Timestamp:     m.timestamp,
			StreamingType: m.streamingType,
		})
	}

	return &conversationFile{
		Version:    conversationFileVersion,
		ID:         conv.id,
		Title:      conv.title,
		CreatedAt:  conv.createdAt,
		Status:     conv.status,
		TokenUsage: conv.tokenUsage,
		Messages:   msgs,
	}
}

// flush 将缓冲区中指定对话的快照写入磁盘
// 由防抖定时器触发，在独立goroutine中执行
func (s *ConversationStore) flush(id string) {
	s.mu.Lock()
	snapshot, exists := s.buffer[id]
	if !exists {
		s.mu.Unlock()
		return
	}
	// 从缓冲区移除，防止重复写入
	delete(s.buffer, id)
	delete(s.timers, id)
	s.mu.Unlock()

	// 序列化快照（快照是不可变的，无需加锁）
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		fmt.Printf("[ConversationStore] 序列化对话 %s 失败: %v\n", id, err)
		return
	}

	path := filepath.Join(s.dir, id+".json")
	// 先写临时文件再重命名，确保写入原子性
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		fmt.Printf("[ConversationStore] 写入临时文件失败: %v\n", err)
		return
	}
	if err := os.Rename(tmpPath, path); err != nil {
		// 重命名失败（可能跨文件系统），降级为直接写入目标文件
		if writeErr := os.WriteFile(path, data, 0644); writeErr != nil {
			fmt.Printf("[ConversationStore] 降级写入对话文件失败: %v (rename错误: %v)\n", writeErr, err)
			_ = os.Remove(tmpPath)
			return
		}
		// 降级写入成功，同步确保数据落盘
		if f, syncErr := os.OpenFile(path, os.O_WRONLY, 0644); syncErr == nil {
			_ = f.Sync()
			_ = f.Close()
		}
		_ = os.Remove(tmpPath)
	}
}

// LoadAllConversations 加载所有持久化的对话
// 返回按创建时间升序排列的对话列表
func (s *ConversationStore) LoadAllConversations() ([]*conversation, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取存储目录失败: %w", err)
	}

	var conversations []*conversation
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(s.dir, entry.Name())
		conv, err := s.readFromFile(path)
		if err != nil {
			fmt.Printf("[ConversationStore] 加载对话 %s 失败: %v\n", path, err)
			continue
		}
		conversations = append(conversations, conv)
	}

	// 按创建时间升序排列，最新的在最后
	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].createdAt.Before(conversations[j].createdAt)
	})

	return conversations, nil
}

// readFromFile 从JSON文件读取并解析对话
func (s *ConversationStore) readFromFile(path string) (*conversation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	var fileData conversationFile
	if err := json.Unmarshal(data, &fileData); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	// 版本兼容性检查
	if fileData.Version > conversationFileVersion {
		return nil, fmt.Errorf("不支持的对话文件版本: %d (当前支持: %d)", fileData.Version, conversationFileVersion)
	}

	conv := &conversation{
		id:         fileData.ID,
		title:      fileData.Title,
		createdAt:  fileData.CreatedAt,
		status:     fileData.Status,
		tokenUsage: fileData.TokenUsage,
	}

	for _, mf := range fileData.Messages {
		toolCalls := mf.ToolCalls
		if toolCalls == nil {
			toolCalls = make([]ToolCall, 0)
		}
		conv.messages = append(conv.messages, &message{
			role:          mf.Role,
			content:       mf.Content,
			thinking:      mf.Thinking,
			toolCalls:     toolCalls,
			tokenUsage:    mf.TokenUsage,
			timestamp:     mf.Timestamp,
			streamingType: mf.StreamingType,
		})
	}

	return conv, nil
}

// DeleteConversation 删除指定对话的持久化文件
// 同时取消该对话的待保存定时器和缓冲区数据
func (s *ConversationStore) DeleteConversation(id string) error {
	s.mu.Lock()
	// 取消防抖定时器
	if timer, exists := s.timers[id]; exists {
		timer.Stop()
		delete(s.timers, id)
	}
	// 清除缓冲区中的快照
	delete(s.buffer, id)
	s.mu.Unlock()

	path := filepath.Join(s.dir, id+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除对话文件失败: %w", err)
	}
	return nil
}

// Close 关闭存储，同步刷新所有待保存的对话
// 应在应用退出时调用，确保所有数据完整落盘
func (s *ConversationStore) Close() {
	s.mu.Lock()
	// 停止所有防抖定时器
	for id, timer := range s.timers {
		timer.Stop()
		delete(s.timers, id)
	}
	// 取出所有待保存的快照
	pending := s.buffer
	s.buffer = make(map[string]*conversationFile)
	s.mu.Unlock()

	// 同步写入所有待保存的快照
	for id, snapshot := range pending {
		data, err := json.MarshalIndent(snapshot, "", "  ")
		if err != nil {
			fmt.Printf("[ConversationStore] 关闭时序列化对话 %s 失败: %v\n", id, err)
			continue
		}
		path := filepath.Join(s.dir, id+".json")
		tmpPath := path + ".tmp"
		if err := os.WriteFile(tmpPath, data, 0644); err == nil {
			if renameErr := os.Rename(tmpPath, path); renameErr != nil {
				// 重命名失败，降级为直接写入
				if writeErr := os.WriteFile(path, data, 0644); writeErr != nil {
					fmt.Printf("[ConversationStore] 关闭时降级写入对话 %s 失败: %v (rename错误: %v)\n", id, writeErr, renameErr)
				} else if f, syncErr := os.OpenFile(path, os.O_WRONLY, 0644); syncErr == nil {
					_ = f.Sync()
					_ = f.Close()
				}
				_ = os.Remove(tmpPath)
			}
		} else {
			fmt.Printf("[ConversationStore] 关闭时写入临时文件 %s 失败: %v\n", tmpPath, err)
		}
	}
}
