package gui

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"agentplus/internal/agent"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Conversation 对话信息，暴露给前端的JSON结构
type Conversation struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	CreatedAt    time.Time `json:"createdAt"`
	Status       string    `json:"status"`
	TokenUsage   int       `json:"tokenUsage"`
	MessageCount int       `json:"messageCount"`
}

// Message 消息信息，暴露给前端的JSON结构
type Message struct {
	Role          string     `json:"role"`
	Content       string     `json:"content"`
	Thinking      string     `json:"thinking"`
	ToolCalls     []ToolCall `json:"toolCalls"`
	TokenUsage    int        `json:"tokenUsage"`
	IsStreaming   bool       `json:"isStreaming"`
	StreamingType string     `json:"streamingType"`
	Timestamp     time.Time  `json:"timestamp"`
}

// ToolCall 工具调用信息，暴露给前端的JSON结构
type ToolCall struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Arguments   string `json:"arguments"`
	IsComplete  bool   `json:"isComplete"`
	Result      string `json:"result"`
	ResultError string `json:"resultError"`
}

// TokenStats Token使用统计
type TokenStats struct {
	TotalTokens    int `json:"totalTokens"`
	InferenceCount int `json:"inferenceCount"`
}

// App Wails应用绑定层
// 作为前端与后端Agent之间的桥梁，管理对话状态和消息流
type App struct {
	ctx   context.Context
	agent *agent.Agent
	mu    sync.RWMutex

	conversations  []*conversation
	activeConvID   string
	currentDir     string
	totalTokens    int
	inferenceCount int
	isStreaming    bool
}

// conversation 内部对话结构，包含消息列表和当前流式消息
type conversation struct {
	id             string
	title          string
	createdAt      time.Time
	status         string
	tokenUsage     int
	messages       []*message
	currentMessage *message
}

// message 内部消息结构，记录单条消息的完整状态
type message struct {
	role          string
	content       string
	thinking      string
	toolCalls     []ToolCall
	tokenUsage    int
	isStreaming   bool
	streamingType string
	timestamp     time.Time
}

// NewApp 创建新的App实例
func NewApp() *App {
	currentDir, _ := os.Getwd()
	return &App{
		currentDir: currentDir,
	}
}

// startup Wails生命周期回调，在应用启动时调用
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// SetAgent 设置Agent实例
// 必须在调用SendMessage之前设置，否则会返回错误
func (a *App) SetAgent(ag *agent.Agent) {
	a.agent = ag
}

// SetCurrentDir 设置当前工作目录
func (a *App) SetCurrentDir(dir string) {
	a.currentDir = dir
}

// SendMessage 发送用户消息并启动推理
// 这是前端调用的主要入口，会异步启动Agent推理过程
func (a *App) SendMessage(content string) error {
	if a.agent == nil {
		return fmt.Errorf("agent not initialized")
	}
	if a.isStreaming {
		return fmt.Errorf("agent is already running")
	}

	a.mu.Lock()
	conv := a.getOrCreateActiveConversation()
	conv.messages = append(conv.messages, &message{
		role:      "user",
		content:   content,
		timestamp: time.Now(),
	})
	conv.currentMessage = nil
	a.isStreaming = true
	a.mu.Unlock()

	runtime.EventsEmit(a.ctx, "conversation:updated", a.GetConversationData())

	go func() {
		if err := a.agent.SendMessage(content); err != nil {
			a.mu.Lock()
			a.isStreaming = false
			a.mu.Unlock()
			runtime.EventsEmit(a.ctx, "stream:error", err.Error())
		}
	}()

	return nil
}

// GetConversations 获取所有对话列表
// 返回前端可用的Conversation结构数组
func (a *App) GetConversations() []Conversation {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]Conversation, 0, len(a.conversations))
	for _, conv := range a.conversations {
		result = append(result, Conversation{
			ID:           conv.id,
			Title:        conv.title,
			CreatedAt:    conv.createdAt,
			Status:       conv.status,
			TokenUsage:   conv.tokenUsage,
			MessageCount: len(conv.messages),
		})
	}
	return result
}

// GetConversationData 获取当前活跃对话的完整数据
// 返回包含消息列表和流式状态的map，供前端渲染使用
func (a *App) GetConversationData() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	conv := a.getActiveConversation()
	if conv == nil {
		return map[string]interface{}{
			"messages":    []Message{},
			"isStreaming": a.isStreaming,
		}
	}

	messages := make([]Message, 0, len(conv.messages)+1)
	for _, msg := range conv.messages {
		messages = append(messages, Message{
			Role:          msg.role,
			Content:       msg.content,
			Thinking:      msg.thinking,
			ToolCalls:     msg.toolCalls,
			TokenUsage:    msg.tokenUsage,
			IsStreaming:   msg.isStreaming,
			StreamingType: msg.streamingType,
			Timestamp:     msg.timestamp,
		})
	}

	// 如果存在正在流式输出的消息，也包含在返回结果中
	if conv.currentMessage != nil {
		messages = append(messages, Message{
			Role:          conv.currentMessage.role,
			Content:       conv.currentMessage.content,
			Thinking:      conv.currentMessage.thinking,
			ToolCalls:     conv.currentMessage.toolCalls,
			TokenUsage:    conv.currentMessage.tokenUsage,
			IsStreaming:   conv.currentMessage.isStreaming,
			StreamingType: conv.currentMessage.streamingType,
			Timestamp:     conv.currentMessage.timestamp,
		})
	}

	return map[string]interface{}{
		"id":          conv.id,
		"messages":    messages,
		"isStreaming": a.isStreaming,
	}
}

// SetWorkDir 设置工作目录
// 切换工作目录并通知前端
func (a *App) SetWorkDir(path string) error {
	if err := os.Chdir(path); err != nil {
		return err
	}
	absPath, _ := os.Getwd()
	a.currentDir = absPath
	runtime.EventsEmit(a.ctx, "workdir:changed", absPath)
	return nil
}

// GetWorkDir 获取当前工作目录
func (a *App) GetWorkDir() string {
	return a.currentDir
}

// GetTokenStats 获取Token使用统计
func (a *App) GetTokenStats() TokenStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return TokenStats{
		TotalTokens:    a.totalTokens,
		InferenceCount: a.inferenceCount,
	}
}

// InterruptInference 中断当前推理
// 用户主动打断时调用，将当前流式消息标记为已中断并追加打断标记
func (a *App) InterruptInference() {
	a.mu.Lock()
	a.isStreaming = false
	conv := a.getActiveConversation()
	if conv != nil && conv.currentMessage != nil {
		conv.currentMessage.isStreaming = false
		conv.currentMessage.content += "\n\n[用户打断]"
		conv.messages = append(conv.messages, conv.currentMessage)
		conv.currentMessage = nil
	}
	a.mu.Unlock()
	runtime.EventsEmit(a.ctx, "stream:interrupted")
	runtime.EventsEmit(a.ctx, "conversation:updated", a.GetConversationData())
}

// ClearConversation 清空当前对话的消息
// 保留对话本身，仅清除消息列表
func (a *App) ClearConversation() {
	a.mu.Lock()
	conv := a.getActiveConversation()
	if conv != nil {
		conv.messages = nil
		conv.currentMessage = nil
	}
	a.mu.Unlock()
	runtime.EventsEmit(a.ctx, "conversation:updated", a.GetConversationData())
}

// getOrCreateActiveConversation 获取或创建活跃对话
// 如果不存在任何对话，则创建一个新的默认对话
// 调用方必须持有a.mu写锁
func (a *App) getOrCreateActiveConversation() *conversation {
	if len(a.conversations) == 0 {
		conv := &conversation{
			id:        fmt.Sprintf("conv-%d", time.Now().UnixMilli()),
			title:     "新对话",
			createdAt: time.Now(),
			status:    "active",
		}
		a.conversations = append(a.conversations, conv)
		a.activeConvID = conv.id
		return conv
	}
	for _, conv := range a.conversations {
		if conv.id == a.activeConvID {
			return conv
		}
	}
	return a.conversations[0]
}

// getActiveConversation 获取当前活跃对话
// 如果没有匹配的活跃对话，返回nil
// 调用方必须持有a.mu读锁或写锁
func (a *App) getActiveConversation() *conversation {
	for _, conv := range a.conversations {
		if conv.id == a.activeConvID {
			return conv
		}
	}
	return nil
}
