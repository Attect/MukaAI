// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"sync"
	"time"
)

// BatchUpdateConfig 批量更新配置
type BatchUpdateConfig struct {
	// BufferDuration 缓冲时间窗口（毫秒）
	// 在此时间窗口内的消息会被批量处理
	BufferDuration int

	// MaxBufferSize 最大缓冲区大小
	// 当缓冲区达到此大小时，立即刷新
	MaxBufferSize int

	// EnableBatching 是否启用批量更新
	EnableBatching bool

	// MinUpdateInterval 最小更新间隔（毫秒）
	// 防止过于频繁的更新
	MinUpdateInterval int
}

// DefaultBatchUpdateConfig 返回默认批量更新配置
func DefaultBatchUpdateConfig() BatchUpdateConfig {
	return BatchUpdateConfig{
		BufferDuration:    50, // 50ms 缓冲窗口
		MaxBufferSize:     10, // 最多缓冲 10 条消息
		EnableBatching:    true,
		MinUpdateInterval: 16, // 约 60fps，确保流畅
	}
}

// BufferedMessage 缓冲的消息
type BufferedMessage struct {
	// Type 消息类型
	Type string
	// Content 消息内容
	Content string
	// ToolCall 工具调用信息（如果是工具调用消息）
	ToolCall ToolCall
	// IsComplete 是否完成（用于工具调用）
	IsComplete bool
	// Usage token 用量（用于完成消息）
	Usage int
	// Error 错误信息（用于错误消息）
	Error error
	// Timestamp 消息时间戳
	Timestamp time.Time
}

// MessageBuffer 消息缓冲器
// 用于缓冲流式消息，避免频繁更新 UI
type MessageBuffer struct {
	// mu 互斥锁
	mu sync.RWMutex

	// config 配置
	config BatchUpdateConfig

	// messages 缓冲的消息列表
	messages []BufferedMessage

	// lastUpdateTime 上次更新时间
	lastUpdateTime time.Time

	// thinkingBuffer 思考内容缓冲
	thinkingBuffer string

	// contentBuffer 正文内容缓冲
	contentBuffer string

	// toolCallBuffer 工具调用缓冲
	toolCallBuffer map[string]ToolCall

	// pendingToolResults 待处理的工具结果
	pendingToolResults map[string]ToolCall

	// hasNewData 是否有新数据
	hasNewData bool
}

// NewMessageBuffer 创建新的消息缓冲器
func NewMessageBuffer(config BatchUpdateConfig) *MessageBuffer {
	return &MessageBuffer{
		config:             config,
		messages:           make([]BufferedMessage, 0),
		lastUpdateTime:     time.Now(),
		toolCallBuffer:     make(map[string]ToolCall),
		pendingToolResults: make(map[string]ToolCall),
	}
}

// AddThinking 添加思考内容
func (b *MessageBuffer) AddThinking(chunk string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.thinkingBuffer += chunk
	b.hasNewData = true
	b.checkAutoFlush()
}

// AddContent 添加正文内容
func (b *MessageBuffer) AddContent(chunk string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.contentBuffer += chunk
	b.hasNewData = true
	b.checkAutoFlush()
}

// AddToolCall 添加工具调用
func (b *MessageBuffer) AddToolCall(call ToolCall, isComplete bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	call.IsComplete = isComplete
	b.toolCallBuffer[call.ID] = call
	b.hasNewData = true
	b.checkAutoFlush()
}

// AddToolResult 添加工具结果
func (b *MessageBuffer) AddToolResult(result ToolCall) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.pendingToolResults[result.ID] = result
	b.hasNewData = true
	b.checkAutoFlush()
}

// AddComplete 添加完成消息
func (b *MessageBuffer) AddComplete(usage int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.messages = append(b.messages, BufferedMessage{
		Type:      "complete",
		Usage:     usage,
		Timestamp: time.Now(),
	})
	b.hasNewData = true
	b.checkAutoFlush()
}

// AddError 添加错误消息
func (b *MessageBuffer) AddError(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.messages = append(b.messages, BufferedMessage{
		Type:      "error",
		Error:     err,
		Timestamp: time.Now(),
	})
	b.hasNewData = true
	b.checkAutoFlush()
}

// checkAutoFlush 检查是否需要自动刷新
func (b *MessageBuffer) checkAutoFlush() {
	// 如果达到最大缓冲区大小，立即刷新
	if len(b.messages) >= b.config.MaxBufferSize {
		return // 由调用者决定是否刷新
	}
}

// ShouldFlush 检查是否应该刷新缓冲区
func (b *MessageBuffer) ShouldFlush() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.hasNewData {
		return false
	}

	// 检查时间间隔
	elapsed := time.Since(b.lastUpdateTime).Milliseconds()
	return elapsed >= int64(b.config.BufferDuration)
}

// Flush 刷新缓冲区并返回累积的消息
func (b *MessageBuffer) Flush() *FlushResult {
	b.mu.Lock()
	defer b.mu.Unlock()

	result := &FlushResult{
		Thinking:      b.thinkingBuffer,
		Content:       b.contentBuffer,
		ToolCalls:     make([]ToolCall, 0),
		ToolResults:   make([]ToolCall, 0),
		Messages:      make([]BufferedMessage, 0),
		HasThinking:   b.thinkingBuffer != "",
		HasContent:    b.contentBuffer != "",
		HasToolCalls:  len(b.toolCallBuffer) > 0,
		HasToolResult: len(b.pendingToolResults) > 0,
	}

	// 收集工具调用
	for _, tc := range b.toolCallBuffer {
		result.ToolCalls = append(result.ToolCalls, tc)
	}

	// 收集工具结果
	for _, tr := range b.pendingToolResults {
		result.ToolResults = append(result.ToolResults, tr)
	}

	// 复制消息列表
	result.Messages = append(result.Messages, b.messages...)

	// 清空缓冲区
	b.thinkingBuffer = ""
	b.contentBuffer = ""
	b.toolCallBuffer = make(map[string]ToolCall)
	b.pendingToolResults = make(map[string]ToolCall)
	b.messages = make([]BufferedMessage, 0)
	b.hasNewData = false
	b.lastUpdateTime = time.Now()

	return result
}

// FlushResult 刷新结果
type FlushResult struct {
	// Thinking 累积的思考内容
	Thinking string
	// Content 累积的正文内容
	Content string
	// ToolCalls 累积的工具调用
	ToolCalls []ToolCall
	// ToolResults 累积的工具结果
	ToolResults []ToolCall
	// Messages 其他消息（完成、错误等）
	Messages []BufferedMessage

	// HasThinking 是否有思考内容
	HasThinking bool
	// HasContent 是否有正文内容
	HasContent bool
	// HasToolCalls 是否有工具调用
	HasToolCalls bool
	// HasToolResult 是否有工具结果
	HasToolResult bool
}

// HasData 检查是否有数据
func (r *FlushResult) HasData() bool {
	return r.HasThinking || r.HasContent || r.HasToolCalls || r.HasToolResult || len(r.Messages) > 0
}

// Clear 清空缓冲区
func (b *MessageBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.thinkingBuffer = ""
	b.contentBuffer = ""
	b.toolCallBuffer = make(map[string]ToolCall)
	b.pendingToolResults = make(map[string]ToolCall)
	b.messages = make([]BufferedMessage, 0)
	b.hasNewData = false
}

// GetBufferSize 获取缓冲区大小
func (b *MessageBuffer) GetBufferSize() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.messages)
}

// HasPendingData 检查是否有待处理的数据
func (b *MessageBuffer) HasPendingData() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.hasNewData
}

// StreamUpdateManager 流式更新管理器
// 管理流式消息的缓冲和批量更新
type StreamUpdateManager struct {
	// mu 互斥锁
	mu sync.RWMutex

	// buffer 消息缓冲器
	buffer *MessageBuffer

	// config 配置
	config BatchUpdateConfig

	// onUpdate 更新回调函数
	onUpdate func(result *FlushResult)

	// ticker 定时器
	ticker *time.Ticker

	// stopChan 停止通道
	stopChan chan struct{}

	// isRunning 是否正在运行
	isRunning bool
}

// NewStreamUpdateManager 创建新的流式更新管理器
func NewStreamUpdateManager(config BatchUpdateConfig) *StreamUpdateManager {
	return &StreamUpdateManager{
		buffer:   NewMessageBuffer(config),
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// SetOnUpdate 设置更新回调函数
func (m *StreamUpdateManager) SetOnUpdate(onUpdate func(result *FlushResult)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onUpdate = onUpdate
}

// Start 启动更新管理器
func (m *StreamUpdateManager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return
	}

	// 创建定时器，定时检查是否需要刷新
	// 使用 MinUpdateInterval 作为检查间隔
	m.ticker = time.NewTicker(time.Duration(m.config.MinUpdateInterval) * time.Millisecond)
	m.isRunning = true

	go m.run()
}

// run 运行更新循环
func (m *StreamUpdateManager) run() {
	for {
		select {
		case <-m.ticker.C:
			// 定时检查是否需要刷新
			if m.buffer.ShouldFlush() {
				result := m.buffer.Flush()
				if result.HasData() && m.onUpdate != nil {
					m.onUpdate(result)
				}
			}

		case <-m.stopChan:
			// 停止信号
			return
		}
	}
}

// Stop 停止更新管理器
func (m *StreamUpdateManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return
	}

	// 刷新剩余数据
	if m.buffer.HasPendingData() {
		result := m.buffer.Flush()
		if result.HasData() && m.onUpdate != nil {
			m.onUpdate(result)
		}
	}

	// 停止定时器
	if m.ticker != nil {
		m.ticker.Stop()
	}

	// 发送停止信号
	close(m.stopChan)
	m.isRunning = false
}

// ForceFlush 强制刷新缓冲区
func (m *StreamUpdateManager) ForceFlush() *FlushResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := m.buffer.Flush()
	if result.HasData() && m.onUpdate != nil {
		m.onUpdate(result)
	}
	return result
}

// GetBuffer 获取消息缓冲器
func (m *StreamUpdateManager) GetBuffer() *MessageBuffer {
	return m.buffer
}

// IsRunning 检查是否正在运行
func (m *StreamUpdateManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// AddThinking 添加思考内容
func (m *StreamUpdateManager) AddThinking(chunk string) {
	m.buffer.AddThinking(chunk)
}

// AddContent 添加正文内容
func (m *StreamUpdateManager) AddContent(chunk string) {
	m.buffer.AddContent(chunk)
}

// AddToolCall 添加工具调用
func (m *StreamUpdateManager) AddToolCall(call ToolCall, isComplete bool) {
	m.buffer.AddToolCall(call, isComplete)
}

// AddToolResult 添加工具结果
func (m *StreamUpdateManager) AddToolResult(result ToolCall) {
	m.buffer.AddToolResult(result)
}

// AddComplete 添加完成消息
func (m *StreamUpdateManager) AddComplete(usage int) {
	m.buffer.AddComplete(usage)
}

// AddError 添加错误消息
func (m *StreamUpdateManager) AddError(err error) {
	m.buffer.AddError(err)
}

// Clear 清空缓冲区
func (m *StreamUpdateManager) Clear() {
	m.buffer.Clear()
}
