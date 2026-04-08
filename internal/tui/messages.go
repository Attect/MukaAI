// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"time"
)

// UserInputMsg 用户输入消息
type UserInputMsg struct {
	// Content 用户输入内容
	Content string
}

// StreamThinkingMsg 流式思考内容消息
type StreamThinkingMsg struct {
	// Chunk 思考内容块
	Chunk string
}

// StreamContentMsg 流式正文内容消息
type StreamContentMsg struct {
	// Chunk 正文内容块
	Chunk string
}

// StreamToolCallMsg 流式工具调用消息
type StreamToolCallMsg struct {
	// Call 工具调用信息
	Call ToolCall
	// IsComplete 是否已完成流式生成
	IsComplete bool
}

// StreamToolResultMsg 工具执行结果消息
type StreamToolResultMsg struct {
	// Result 工具执行结果
	Result ToolCall
}

// StreamCompleteMsg 流式输出完成消息
type StreamCompleteMsg struct {
	// Usage token 用量
	Usage int
}

// StreamErrorMsg 流式输出错误消息
type StreamErrorMsg struct {
	// Error 错误信息
	Error error
}

// ConversationCreatedMsg 对话创建消息
type ConversationCreatedMsg struct {
	// Conversation 新创建的对话
	Conversation *Conversation
}

// ConversationSwitchedMsg 对话切换消息
type ConversationSwitchedMsg struct {
	// ConversationID 切换到的对话 ID
	ConversationID string
}

// ConversationStatusUpdatedMsg 对话状态更新消息
type ConversationStatusUpdatedMsg struct {
	// ConversationID 对话 ID
	ConversationID string
	// Status 新状态
	Status ConvStatus
}

// WorkingDirChangedMsg 工作目录变更消息
type WorkingDirChangedMsg struct {
	// OldDir 旧目录
	OldDir string
	// NewDir 新目录
	NewDir string
}

// TokenUsageUpdatedMsg token 用量更新消息
type TokenUsageUpdatedMsg struct {
	// TotalTokens 总 token 用量
	TotalTokens int
	// Delta 增量
	Delta int
}

// InferenceCountUpdatedMsg 推理次数更新消息
type InferenceCountUpdatedMsg struct {
	// Count 推理次数
	Count int
}

// InputModeChangedMsg 输入模式变更消息
type InputModeChangedMsg struct {
	// Mode 新的输入模式
	Mode InputMode
}

// ShowConversationListMsg 显示对话列表消息
type ShowConversationListMsg struct {
	// Show 是否显示
	Show bool
}

// CommandExecutedMsg 命令执行消息
type CommandExecutedMsg struct {
	// Command 命令名称
	Command string
	// Args 命令参数
	Args []string
	// Result 执行结果
	Result string
	// Error 执行错误
	Error error
}

// BatchUpdateMsg 批量更新消息
// 用于批量处理缓冲的流式消息
type BatchUpdateMsg struct {
	// Result 刷新结果
	Result *FlushResult
}

// TickMsg 定时器消息
// 用于定时检查缓冲区是否需要刷新
type TickMsg struct {
	// Time 当前时间
	Time time.Time
}

// NewBatchUpdateMsg 创建批量更新消息
func NewBatchUpdateMsg(result *FlushResult) BatchUpdateMsg {
	return BatchUpdateMsg{Result: result}
}

// NewTickMsg 创建定时器消息
func NewTickMsg(t time.Time) TickMsg {
	return TickMsg{Time: t}
}

// NewStreamThinkingMsg 创建流式思考内容消息
func NewStreamThinkingMsg(chunk string) StreamThinkingMsg {
	return StreamThinkingMsg{Chunk: chunk}
}

// NewStreamContentMsg 创建流式正文内容消息
func NewStreamContentMsg(chunk string) StreamContentMsg {
	return StreamContentMsg{Chunk: chunk}
}

// NewStreamToolCallMsg 创建流式工具调用消息
func NewStreamToolCallMsg(call ToolCall, isComplete bool) StreamToolCallMsg {
	return StreamToolCallMsg{Call: call, IsComplete: isComplete}
}

// NewStreamToolResultMsg 创建工具执行结果消息
func NewStreamToolResultMsg(result ToolCall) StreamToolResultMsg {
	return StreamToolResultMsg{Result: result}
}

// NewStreamCompleteMsg 创建流式输出完成消息
func NewStreamCompleteMsg(usage int) StreamCompleteMsg {
	return StreamCompleteMsg{Usage: usage}
}

// NewStreamErrorMsg 创建流式输出错误消息
func NewStreamErrorMsg(err error) StreamErrorMsg {
	return StreamErrorMsg{Error: err}
}

// NewConversationCreatedMsg 创建对话创建消息
func NewConversationCreatedMsg(conv *Conversation) ConversationCreatedMsg {
	return ConversationCreatedMsg{Conversation: conv}
}

// NewConversationSwitchedMsg 创建对话切换消息
func NewConversationSwitchedMsg(id string) ConversationSwitchedMsg {
	return ConversationSwitchedMsg{ConversationID: id}
}

// NewConversationStatusUpdatedMsg 创建对话状态更新消息
func NewConversationStatusUpdatedMsg(id string, status ConvStatus) ConversationStatusUpdatedMsg {
	return ConversationStatusUpdatedMsg{ConversationID: id, Status: status}
}

// NewWorkingDirChangedMsg 创建工作目录变更消息
func NewWorkingDirChangedMsg(oldDir, newDir string) WorkingDirChangedMsg {
	return WorkingDirChangedMsg{OldDir: oldDir, NewDir: newDir}
}

// NewTokenUsageUpdatedMsg 创建 token 用量更新消息
func NewTokenUsageUpdatedMsg(total, delta int) TokenUsageUpdatedMsg {
	return TokenUsageUpdatedMsg{TotalTokens: total, Delta: delta}
}

// NewInferenceCountUpdatedMsg 创建推理次数更新消息
func NewInferenceCountUpdatedMsg(count int) InferenceCountUpdatedMsg {
	return InferenceCountUpdatedMsg{Count: count}
}

// NewInputModeChangedMsg 创建输入模式变更消息
func NewInputModeChangedMsg(mode InputMode) InputModeChangedMsg {
	return InputModeChangedMsg{Mode: mode}
}

// NewShowConversationListMsg 创建显示对话列表消息
func NewShowConversationListMsg(show bool) ShowConversationListMsg {
	return ShowConversationListMsg{Show: show}
}

// NewCommandExecutedMsg 创建命令执行消息
func NewCommandExecutedMsg(cmd string, args []string, result string, err error) CommandExecutedMsg {
	return CommandExecutedMsg{Command: cmd, Args: args, Result: result, Error: err}
}

// StreamHandlerImpl 流式处理器实现
// 实现 StreamHandler 接口，用于将流式事件转换为 Bubble Tea 消息
type StreamHandlerImpl struct {
	// sendMsg 发送消息的函数
	sendMsg func(msg interface{})
}

// NewStreamHandler 创建新的流式处理器
func NewStreamHandler(sendMsg func(msg interface{})) *StreamHandlerImpl {
	return &StreamHandlerImpl{sendMsg: sendMsg}
}

// OnThinking 处理思考内容块
func (h *StreamHandlerImpl) OnThinking(chunk string) {
	h.sendMsg(NewStreamThinkingMsg(chunk))
}

// OnContent 处理正文内容块
func (h *StreamHandlerImpl) OnContent(chunk string) {
	h.sendMsg(NewStreamContentMsg(chunk))
}

// OnToolCall 处理工具调用
func (h *StreamHandlerImpl) OnToolCall(call ToolCall, isComplete bool) {
	h.sendMsg(NewStreamToolCallMsg(call, isComplete))
}

// OnToolResult 处理工具执行结果
func (h *StreamHandlerImpl) OnToolResult(result ToolCall) {
	h.sendMsg(NewStreamToolResultMsg(result))
}

// OnComplete 处理推理完成
func (h *StreamHandlerImpl) OnComplete(usage int) {
	h.sendMsg(NewStreamCompleteMsg(usage))
}

// OnError 处理错误
func (h *StreamHandlerImpl) OnError(err error) {
	h.sendMsg(NewStreamErrorMsg(err))
}

// MessageBuilder 消息构建器
// 用于构建对话消息
type MessageBuilder struct {
	message *Message
}

// NewMessageBuilder 创建新的消息构建器
func NewMessageBuilder(role MessageRole) *MessageBuilder {
	return &MessageBuilder{
		message: &Message{
			Role:      role,
			Timestamp: time.Now(),
		},
	}
}

// SetContent 设置正文内容
func (b *MessageBuilder) SetContent(content string) *MessageBuilder {
	b.message.Content = content
	return b
}

// SetThinking 设置思考内容
func (b *MessageBuilder) SetThinking(thinking string) *MessageBuilder {
	b.message.Thinking = thinking
	return b
}

// AddToolCall 添加工具调用
func (b *MessageBuilder) AddToolCall(call ToolCall) *MessageBuilder {
	b.message.ToolCalls = append(b.message.ToolCalls, call)
	return b
}

// SetTokenUsage 设置 token 用量
func (b *MessageBuilder) SetTokenUsage(usage int) *MessageBuilder {
	b.message.TokenUsage = usage
	return b
}

// SetStreaming 设置流式状态
func (b *MessageBuilder) SetStreaming(streaming bool, streamingType string) *MessageBuilder {
	b.message.IsStreaming = streaming
	b.message.StreamingType = streamingType
	return b
}

// Build 构建消息
func (b *MessageBuilder) Build() *Message {
	return b.message
}

// ConversationBuilder 对话构建器
// 用于构建对话
type ConversationBuilder struct {
	conversation *Conversation
}

// NewConversationBuilder 创建新的对话构建器
func NewConversationBuilder(id string) *ConversationBuilder {
	return &ConversationBuilder{
		conversation: &Conversation{
			ID:        id,
			CreatedAt: time.Now(),
			Status:    ConvStatusActive,
			Messages:  make([]Message, 0),
		},
	}
}

// SetTitle 设置标题
func (b *ConversationBuilder) SetTitle(title string) *ConversationBuilder {
	b.conversation.Title = title
	return b
}

// SetStatus 设置状态
func (b *ConversationBuilder) SetStatus(status ConvStatus) *ConversationBuilder {
	b.conversation.Status = status
	return b
}

// SetSubConversation 设置为子对话
func (b *ConversationBuilder) SetSubConversation(parentID, agentRole string) *ConversationBuilder {
	b.conversation.IsSubConversation = true
	b.conversation.ParentID = parentID
	b.conversation.AgentRole = agentRole
	return b
}

// AddMessage 添加消息
func (b *ConversationBuilder) AddMessage(msg Message) *ConversationBuilder {
	b.conversation.Messages = append(b.conversation.Messages, msg)
	return b
}

// SetTokenUsage 设置 token 用量
func (b *ConversationBuilder) SetTokenUsage(usage int) *ConversationBuilder {
	b.conversation.TokenUsage = usage
	return b
}

// Build 构建对话
func (b *ConversationBuilder) Build() *Conversation {
	return b.conversation
}
