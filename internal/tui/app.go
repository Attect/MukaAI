// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
)

// InputMode 输入模式类型
type InputMode int

const (
	// InputModeSingleLine 单行输入模式
	InputModeSingleLine InputMode = iota
	// InputModeMultiLine 多行输入模式
	InputModeMultiLine
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

// MessageRole 消息角色类型
type MessageRole int

const (
	// MessageRoleUser 用户消息
	MessageRoleUser MessageRole = iota
	// MessageRoleAssistant 助手消息
	MessageRoleAssistant
	// MessageRoleTool 工具消息
	MessageRoleTool
)

// ToolCall 工具调用信息
type ToolCall struct {
	// ID 工具调用唯一标识
	ID string
	// Name 工具名称
	Name string
	// Arguments 工具参数（JSON 格式）
	Arguments string
	// IsComplete 是否已完成流式生成
	IsComplete bool
	// Result 工具执行结果
	Result string
	// ResultError 工具执行错误
	ResultError string
}

// Message 对话消息
type Message struct {
	// Role 消息角色
	Role MessageRole
	// Content 正文内容
	Content string
	// Thinking 思考内容
	Thinking string
	// ToolCalls 工具调用列表
	ToolCalls []ToolCall
	// TokenUsage token 用量
	TokenUsage int
	// Timestamp 时间戳
	Timestamp time.Time
	// IsStreaming 是否正在流式输出
	IsStreaming bool
	// StreamingContent 流式输出的当前内容
	StreamingContent string
	// StreamingType 流式输出类型（thinking/content/tool）
	StreamingType string
}

// Conversation 对话
type Conversation struct {
	// ID 对话唯一标识
	ID string
	// CreatedAt 创建时间
	CreatedAt time.Time
	// Status 对话状态
	Status ConvStatus
	// Messages 消息列表
	Messages []Message
	// TokenUsage token 用量
	TokenUsage int
	// Title 对话标题
	Title string
	// IsSubConversation 是否为子对话
	IsSubConversation bool
	// ParentID 父对话 ID（如果是子对话）
	ParentID string
	// AgentRole Agent 角色（如果是子对话）
	AgentRole string
	// currentMessage 当前正在构建的消息（用于流式输出）
	currentMessage *Message
}

// StatusBar 状态栏组件
type StatusBar struct {
	// Width 宽度
	Width int
	// CurrentDir 当前工作目录
	CurrentDir string
	// TotalTokens 总 token 用量
	TotalTokens int
	// InferenceCount 推理次数
	InferenceCount int
}

// AppModel TUI 主应用 Model
// 实现 Bubble Tea 的 Model-Update-View 架构
type AppModel struct {
	// 状态
	currentDir     string
	totalTokens    int
	inferenceCount int

	// 对话管理
	conversations []*Conversation
	activeConv    *Conversation

	// UI 组件
	input     textinput.Model
	chatView  viewport.Model
	statusBar StatusBar

	// 状态标志
	isStreaming  bool
	showConvList bool
	inputMode    InputMode

	// 窗口尺寸
	width  int
	height int

	// 错误信息
	lastError string

	// Agent 回调接口
	agent AgentInterface

	// 是否已初始化
	initialized bool
}

// AgentInterface Agent 接口定义
// 用于解耦 TUI 和 Agent 核心逻辑
type AgentInterface interface {
	// SendMessage 发送用户消息并启动推理
	SendMessage(content string) error

	// Cancel 取消当前推理
	Cancel() error

	// SetStreamHandler 设置流式输出处理器
	SetStreamHandler(handler StreamHandler)

	// GetConversations 获取所有对话列表
	GetConversations() []*Conversation

	// SwitchConversation 切换到指定对话
	SwitchConversation(id string) error

	// GetCurrentConversation 获取当前活动对话
	GetCurrentConversation() *Conversation

	// Close 关闭 Agent
	Close() error
}

// StreamHandler 流式消息处理器接口
type StreamHandler interface {
	// OnThinking 处理思考内容块
	OnThinking(chunk string)

	// OnContent 处理正文内容块
	OnContent(chunk string)

	// OnToolCall 处理工具调用
	// isComplete 表示工具调用是否已完成流式生成
	OnToolCall(call ToolCall, isComplete bool)

	// OnToolResult 处理工具执行结果
	OnToolResult(result ToolCall)

	// OnComplete 处理推理完成
	OnComplete(usage int)

	// OnError 处理错误
	OnError(err error)
}

// NewAppModel 创建新的 TUI 应用 Model
func NewAppModel() AppModel {
	// 创建输入框
	ti := textinput.New()
	ti.Placeholder = "请输入你的问题..."
	ti.Focus()
	ti.CharLimit = 10000

	// 创建视口（使用 functional options）
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(24))
	vp.SetContent("")

	return AppModel{
		input:        ti,
		chatView:     vp,
		conversations: make([]*Conversation, 0),
		inputMode:    InputModeSingleLine,
		initialized:  false,
	}
}

// Init 初始化 TUI 应用
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
	)
}

// Update 处理消息和更新状态
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// 处理窗口大小变化
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.Width = msg.Width
		m.chatView.SetWidth(msg.Width)
		m.chatView.SetHeight(msg.Height - 4) // 减去状态栏和输入框的高度

	case tea.KeyMsg:
		// 处理键盘输入
		switch msg.String() {
		case "ctrl+c", "esc":
			// 退出 TUI
			return m, tea.Quit

		case "enter":
			// 提交输入
			if m.inputMode == InputModeSingleLine {
				content := m.input.Value()
				if content != "" {
					cmds = append(cmds, m.handleUserInput(content))
					m.input.SetValue("")
				}
			}

		case "tab":
			// 切换输入模式
			if m.inputMode == InputModeSingleLine {
				m.inputMode = InputModeMultiLine
			} else {
				m.inputMode = InputModeSingleLine
			}

		case "ctrl+l":
			// 显示对话列表
			m.showConvList = !m.showConvList
		}

	case StreamThinkingMsg:
		// 处理思考内容流式消息
		m.handleStreamThinking(msg.Chunk)

	case StreamContentMsg:
		// 处理正文内容流式消息
		m.handleStreamContent(msg.Chunk)

	case StreamToolCallMsg:
		// 处理工具调用流式消息
		m.handleStreamToolCall(msg.Call, msg.IsComplete)

	case StreamToolResultMsg:
		// 处理工具执行结果消息
		m.handleStreamToolResult(msg.Result)

	case StreamCompleteMsg:
		// 处理推理完成消息
		m.handleStreamComplete(msg.Usage)

	case StreamErrorMsg:
		// 处理错误消息
		m.handleStreamError(msg.Error)

	case UserInputMsg:
		// 处理用户输入消息
		cmds = append(cmds, m.processUserInput(msg.Content))
	}

	// 更新输入框
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	// 更新视口
	var vpCmd tea.Cmd
	m.chatView, vpCmd = m.chatView.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

// View 渲染 TUI 界面
func (m AppModel) View() tea.View {
	// 渲染状态栏
	statusBar := m.renderStatusBar()

	// 渲染对话区
	chatArea := m.renderChatArea()

	// 渲染输入框
	inputArea := m.renderInputArea()

	// 组合界面
	content := statusBar + "\n" + chatArea + "\n" + inputArea

	// 创建 View 对象
	v := tea.View{}
	v.SetContent(content)
	return v
}

// handleUserInput 处理用户输入
func (m AppModel) handleUserInput(content string) tea.Cmd {
	return func() tea.Msg {
		return UserInputMsg{Content: content}
	}
}

// processUserInput 处理用户输入消息
func (m AppModel) processUserInput(content string) tea.Cmd {
	// 检查是否为命令
	if len(content) > 0 && content[0] == '/' {
		return m.handleCommand(content)
	}

	// 添加用户消息到对话
	if m.activeConv != nil {
		userMsg := Message{
			Role:      MessageRoleUser,
			Content:   content,
			Timestamp: time.Now(),
		}
		m.activeConv.Messages = append(m.activeConv.Messages, userMsg)
		m.updateChatView()
	}

	// 发送到 Agent
	if m.agent != nil {
		m.isStreaming = true
		if err := m.agent.SendMessage(content); err != nil {
			m.lastError = err.Error()
		}
	}

	return nil
}

// handleCommand 处理命令
func (m AppModel) handleCommand(cmd string) tea.Cmd {
	// TODO: 实现命令解析和处理
	// 支持的命令：/cd, /conversations, /clear, /save, /help, /exit
	return nil
}

// handleStreamThinking 处理思考内容流式消息
func (m *AppModel) handleStreamThinking(chunk string) {
	if m.activeConv == nil || m.activeConv.currentMessage == nil {
		return
	}

	m.activeConv.currentMessage.Thinking += chunk
	m.activeConv.currentMessage.IsStreaming = true
	m.activeConv.currentMessage.StreamingType = "thinking"
	m.updateChatView()
}

// handleStreamContent 处理正文内容流式消息
func (m *AppModel) handleStreamContent(chunk string) {
	if m.activeConv == nil || m.activeConv.currentMessage == nil {
		return
	}

	m.activeConv.currentMessage.Content += chunk
	m.activeConv.currentMessage.IsStreaming = true
	m.activeConv.currentMessage.StreamingType = "content"
	m.updateChatView()
}

// handleStreamToolCall 处理工具调用流式消息
func (m *AppModel) handleStreamToolCall(call ToolCall, isComplete bool) {
	if m.activeConv == nil || m.activeConv.currentMessage == nil {
		return
	}

	// 查找或添加工具调用
	found := false
	for i, tc := range m.activeConv.currentMessage.ToolCalls {
		if tc.ID == call.ID {
			m.activeConv.currentMessage.ToolCalls[i] = call
			found = true
			break
		}
	}

	if !found {
		m.activeConv.currentMessage.ToolCalls = append(m.activeConv.currentMessage.ToolCalls, call)
	}

	m.activeConv.currentMessage.IsStreaming = true
	m.activeConv.currentMessage.StreamingType = "tool"
	m.updateChatView()
}

// handleStreamToolResult 处理工具执行结果消息
func (m *AppModel) handleStreamToolResult(result ToolCall) {
	if m.activeConv == nil || m.activeConv.currentMessage == nil {
		return
	}

	// 更新工具调用结果
	for i, tc := range m.activeConv.currentMessage.ToolCalls {
		if tc.ID == result.ID {
			m.activeConv.currentMessage.ToolCalls[i].Result = result.Result
			m.activeConv.currentMessage.ToolCalls[i].ResultError = result.ResultError
			break
		}
	}

	m.updateChatView()
}

// handleStreamComplete 处理推理完成消息
func (m *AppModel) handleStreamComplete(usage int) {
	if m.activeConv == nil || m.activeConv.currentMessage == nil {
		return
	}

	// 完成当前消息
	m.activeConv.currentMessage.IsStreaming = false
	m.activeConv.currentMessage.TokenUsage = usage
	m.activeConv.currentMessage.Timestamp = time.Now()

	// 更新统计
	m.activeConv.TokenUsage += usage
	m.totalTokens += usage
	m.inferenceCount++

	m.isStreaming = false
	m.updateChatView()
}

// handleStreamError 处理错误消息
func (m *AppModel) handleStreamError(err error) {
	m.lastError = err.Error()
	m.isStreaming = false
}

// updateChatView 更新对话视图
func (m *AppModel) updateChatView() {
	if m.activeConv == nil {
		return
	}

	content := m.renderConversation(m.activeConv)
	m.chatView.SetContent(content)
}

// renderStatusBar 渲染状态栏
func (m AppModel) renderStatusBar() string {
	return m.statusBar.Render()
}

// renderChatArea 渲染对话区
func (m AppModel) renderChatArea() string {
	return m.chatView.View()
}

// renderInputArea 渲染输入框
func (m AppModel) renderInputArea() string {
	return m.input.View()
}

// renderConversation 渲染对话内容
func (m AppModel) renderConversation(conv *Conversation) string {
	// TODO: 实现对话内容的格式化渲染
	// 包括用户消息、思考内容、正文、工具调用等
	return ""
}

// SetAgent 设置 Agent 接口
func (m *AppModel) SetAgent(agent AgentInterface) {
	m.agent = agent
}

// AddConversation 添加新对话
func (m *AppModel) AddConversation(conv *Conversation) {
	m.conversations = append(m.conversations, conv)
	if m.activeConv == nil {
		m.activeConv = conv
	}
}

// SwitchConversation 切换到指定对话
func (m *AppModel) SwitchConversation(id string) {
	for _, conv := range m.conversations {
		if conv.ID == id {
			m.activeConv = conv
			m.updateChatView()
			break
		}
	}
}

// Render 渲染状态栏
func (s StatusBar) Render() string {
	// TODO: 实现状态栏渲染
	// 显示：工作目录、总 token 用量、推理次数
	return ""
}
