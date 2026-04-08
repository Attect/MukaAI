// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"agentplus/internal/tui/components"
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
	input      *components.InputComponent
	chatView   *components.ChatView
	statusBar  *components.StatusBar
	dialogList *components.DialogList

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

	// 流式更新管理器
	streamManager *StreamUpdateManager

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
	// 创建输入组件
	inputConfig := components.DefaultInputComponentConfig()
	inputConfig.Width = 80
	inputConfig.Height = 3
	input := components.NewInputComponent(inputConfig)

	// 创建对话显示组件
	chatView := components.NewChatView(80, 24)

	// 创建状态栏组件
	statusBar := components.NewStatusBar(components.DefaultStatusBarConfig())

	// 创建对话列表组件
	dialogList := components.NewDialogList()

	// 获取当前工作目录
	currentDir, err := os.Getwd()
	if err != nil {
		currentDir = "~"
	}
	statusBar.SetDirectory(currentDir)

	// 创建流式更新管理器
	streamManager := NewStreamUpdateManager(DefaultBatchUpdateConfig())

	return AppModel{
		input:         input,
		chatView:      chatView,
		statusBar:     statusBar,
		dialogList:    dialogList,
		conversations: make([]*Conversation, 0),
		inputMode:     InputModeSingleLine,
		currentDir:    currentDir,
		initialized:   false,
		streamManager: streamManager,
	}
}

// Init 初始化 TUI 应用
func (m AppModel) Init() tea.Cmd {
	// 设置流式更新管理器的回调
	m.streamManager.SetOnUpdate(func(result *FlushResult) {
		// 通过发送消息触发 UI 更新
		// 注意：这里不能直接调用 m.Update，需要通过消息机制
		// 实际的更新逻辑在 handleBatchUpdate 中
	})

	// 启动流式更新管理器
	m.streamManager.Start()

	return tea.Batch(
		m.input.Init(),
		// 启动定时器，定期检查缓冲区
		m.tickCmd(),
	)
}

// tickCmd 创建定时器命令
func (m AppModel) tickCmd() tea.Cmd {
	// 每 16ms 检查一次（约 60fps）
	return tea.Tick(16*time.Millisecond, func(t time.Time) tea.Msg {
		return NewTickMsg(t)
	})
}

// Update 处理消息和更新状态
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// 如果对话列表可见，优先处理对话列表的输入
	if m.dialogList.IsVisible() {
		selected, handled := m.dialogList.Update(msg)
		if handled {
			// 用户选择了对话
			if selected != nil {
				m.SwitchConversation(selected.ID)
				m.dialogList.Hide()
			}
			return m, tea.Batch(cmds...)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// 处理窗口大小变化
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)
		m.chatView.SetSize(msg.Width, m.height-6) // 减去状态栏和输入框的高度
		m.input.SetWidth(msg.Width)
		m.dialogList.SetSize(msg.Width-4, m.height-6)

	case tea.KeyMsg:
		// 处理键盘输入
		keyStr := msg.String()

		// 全局快捷键
		switch keyStr {
		case "ctrl+c":
			// 退出 TUI
			m.streamManager.Stop()
			return m, tea.Quit

		case "esc":
			// 如果对话列表可见，关闭对话列表
			if m.dialogList.IsVisible() {
				m.dialogList.Hide()
				return m, nil
			}
			// 否则退出 TUI
			m.streamManager.Stop()
			return m, tea.Quit

		case "ctrl+l":
			// 显示/隐藏对话列表
			m.toggleConversationList()
			return m, nil
		}

		// 输入相关快捷键（仅在对话列表不可见时处理）
		if !m.dialogList.IsVisible() {
			// Tab 键切换输入模式
			if keyStr == "tab" {
				m.input.ToggleMode()
				// 同步 inputMode 状态
				if m.input.GetMode() == components.InputModeSingleLine {
					m.inputMode = InputModeSingleLine
				} else {
					m.inputMode = InputModeMultiLine
				}
				return m, nil
			}

			// Enter 和 Ctrl+Enter 提交逻辑
			if m.input.ShouldSubmit(keyStr) {
				content := m.input.GetValue()
				if content != "" {
					// 添加到历史记录
					m.input.AddToHistory(content)
					cmds = append(cmds, m.handleUserInput(content))
					m.input.Clear()
				}
				return m, tea.Batch(cmds...)
			}
		}

		// 更新输入组件（处理其他按键）
		var inputCmd tea.Cmd
		m.input, inputCmd = m.input.Update(msg)
		if inputCmd != nil {
			cmds = append(cmds, inputCmd)
		}

	case TickMsg:
		// 定时检查缓冲区
		if m.streamManager.GetBuffer().ShouldFlush() {
			result := m.streamManager.ForceFlush()
			if result.HasData() {
				cmds = append(cmds, func() tea.Msg {
					return NewBatchUpdateMsg(result)
				})
			}
		}
		// 继续定时器
		cmds = append(cmds, m.tickCmd())

	case BatchUpdateMsg:
		// 处理批量更新消息
		m.handleBatchUpdate(msg.Result)

	case StreamThinkingMsg:
		// 处理思考内容流式消息（缓冲）
		m.streamManager.AddThinking(msg.Chunk)

	case StreamContentMsg:
		// 处理正文内容流式消息（缓冲）
		m.streamManager.AddContent(msg.Chunk)

	case StreamToolCallMsg:
		// 处理工具调用流式消息（缓冲）
		m.streamManager.AddToolCall(msg.Call, msg.IsComplete)

	case StreamToolResultMsg:
		// 处理工具执行结果消息（缓冲）
		m.streamManager.AddToolResult(msg.Result)

	case StreamCompleteMsg:
		// 处理推理完成消息（缓冲）
		m.streamManager.AddComplete(msg.Usage)

	case StreamErrorMsg:
		// 处理错误消息（缓冲）
		m.streamManager.AddError(msg.Error)

	case UserInputMsg:
		// 处理用户输入消息
		cmds = append(cmds, m.processUserInput(msg.Content))

	case WorkingDirChangedMsg:
		// 处理工作目录变更消息
		m.handleWorkingDirChanged(msg.OldDir, msg.NewDir)

	case ShowConversationListMsg:
		// 处理显示对话列表消息
		if msg.Show {
			// 更新对话列表数据
			m.updateDialogList()
			// 如果有活动对话，选中它
			if m.activeConv != nil {
				m.dialogList.SelectConversationByID(m.activeConv.ID)
			}
			m.dialogList.Show()
		} else {
			m.dialogList.Hide()
		}

	case CommandExecutedMsg:
		// 处理命令执行消息
		m.handleCommandExecuted(msg.Command, msg.Args, msg.Result, msg.Error)
	}

	// 更新对话显示组件
	_, vpCmd := m.chatView.Update(msg)
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

	// 如果对话列表可见，渲染对话列表覆盖层
	if m.dialogList.IsVisible() {
		overlay := m.dialogList.RenderOverlay(m.width, m.height)
		content = overlay
	}

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

		// 初始化当前消息（用于接收流式输出）
		m.activeConv.currentMessage = &Message{
			Role:      MessageRoleAssistant,
			Timestamp: time.Now(),
		}

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
	// 解析命令
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}

	cmdName := strings.ToLower(parts[0])
	args := parts[1:]

	// 根据命令类型路由
	switch cmdName {
	case "/cd":
		return m.handleCDCommand(args)
	case "/conversations", "/conv":
		// 显示对话列表
		return func() tea.Msg {
			return NewShowConversationListMsg(true)
		}
	case "/clear":
		// 清空当前对话
		return m.handleClearCommand()
	case "/save":
		// 保存对话
		return m.handleSaveCommand(args)
	case "/help":
		// 显示帮助
		return m.handleHelpCommand()
	case "/exit", "/quit", "/q":
		// 退出程序
		return tea.Quit
	default:
		// 未知命令，显示错误
		return func() tea.Msg {
			return NewCommandExecutedMsg(cmdName, args, "", fmt.Errorf("未知命令: %s", cmdName))
		}
	}
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

	// 更新状态栏
	m.statusBar.SetTokens(m.totalTokens)
	m.statusBar.SetInferenceCount(m.inferenceCount)

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

// toggleConversationList 切换对话列表显示
func (m *AppModel) toggleConversationList() {
	if m.dialogList.IsVisible() {
		m.dialogList.Hide()
	} else {
		// 更新对话列表数据
		m.updateDialogList()
		m.dialogList.Show()
	}
}

// updateDialogList 更新对话列表数据
func (m *AppModel) updateDialogList() {
	// 将 Conversation 转换为 components.Conversation
	dialogConvs := make([]*components.Conversation, 0, len(m.conversations))
	for _, conv := range m.conversations {
		dialogConv := &components.Conversation{
			ID:                conv.ID,
			CreatedAt:         conv.CreatedAt,
			Status:            components.ConvStatus(conv.Status),
			Title:             conv.Title,
			IsSubConversation: conv.IsSubConversation,
			AgentRole:         conv.AgentRole,
			MessageCount:      len(conv.Messages),
			TokenUsage:        conv.TokenUsage,
		}
		dialogConvs = append(dialogConvs, dialogConv)
	}
	m.dialogList.SetConversations(dialogConvs)
}

// showConversationList 显示对话列表
func (m AppModel) showConversationList() {
	// 更新对话列表数据
	m.updateDialogList()
	// 显示对话列表
	m.dialogList.Show()
}

// renderConversation 渲染对话内容
func (m AppModel) renderConversation(conv *Conversation) string {
	// 将 Conversation 的消息转换为 ChatView 的消息格式
	messages := make([]components.MessageData, 0, len(conv.Messages))

	for _, msg := range conv.Messages {
		// 转换消息角色
		var role string
		switch msg.Role {
		case MessageRoleUser:
			role = "user"
		case MessageRoleAssistant:
			role = "assistant"
		case MessageRoleTool:
			role = "tool"
		}

		// 转换工具调用
		toolCalls := make([]components.ToolCallData, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			toolCalls = append(toolCalls, components.ToolCallData{
				ID:          tc.ID,
				Name:        tc.Name,
				Arguments:   tc.Arguments,
				IsComplete:  tc.IsComplete,
				Result:      tc.Result,
				ResultError: tc.ResultError,
			})
		}

		// 创建消息数据
		msgData := components.MessageData{
			Role:          role,
			Content:       msg.Content,
			Thinking:      msg.Thinking,
			ToolCalls:     toolCalls,
			TokenUsage:    msg.TokenUsage,
			IsStreaming:   msg.IsStreaming,
			StreamingType: msg.StreamingType,
		}

		messages = append(messages, msgData)
	}

	// 使用 ChatView 渲染消息
	return m.chatView.RenderMessages(messages)
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

// handleBatchUpdate 处理批量更新消息
// 这是实时 UI 更新的核心方法，处理缓冲后的流式消息
func (m *AppModel) handleBatchUpdate(result *FlushResult) {
	if m.activeConv == nil {
		return
	}

	// 确保有当前消息
	if m.activeConv.currentMessage == nil {
		m.activeConv.currentMessage = &Message{
			Role:      MessageRoleAssistant,
			Timestamp: time.Now(),
		}
	}

	// 更新思考内容
	if result.HasThinking {
		m.activeConv.currentMessage.Thinking += result.Thinking
		m.activeConv.currentMessage.IsStreaming = true
		m.activeConv.currentMessage.StreamingType = "thinking"
	}

	// 更新正文内容
	if result.HasContent {
		m.activeConv.currentMessage.Content += result.Content
		m.activeConv.currentMessage.IsStreaming = true
		m.activeConv.currentMessage.StreamingType = "content"
	}

	// 更新工具调用
	if result.HasToolCalls {
		for _, tc := range result.ToolCalls {
			// 查找或添加工具调用
			found := false
			for i, existingTc := range m.activeConv.currentMessage.ToolCalls {
				if existingTc.ID == tc.ID {
					m.activeConv.currentMessage.ToolCalls[i] = tc
					found = true
					break
				}
			}

			if !found {
				m.activeConv.currentMessage.ToolCalls = append(m.activeConv.currentMessage.ToolCalls, tc)
			}
		}
		m.activeConv.currentMessage.IsStreaming = true
		m.activeConv.currentMessage.StreamingType = "tool"
	}

	// 更新工具结果
	if result.HasToolResult {
		for _, tr := range result.ToolResults {
			for i, tc := range m.activeConv.currentMessage.ToolCalls {
				if tc.ID == tr.ID {
					m.activeConv.currentMessage.ToolCalls[i].Result = tr.Result
					m.activeConv.currentMessage.ToolCalls[i].ResultError = tr.ResultError
					break
				}
			}
		}
	}

	// 处理其他消息（完成、错误等）
	for _, msg := range result.Messages {
		switch msg.Type {
		case "complete":
			// 完成当前消息
			m.activeConv.currentMessage.IsStreaming = false
			m.activeConv.currentMessage.TokenUsage = msg.Usage
			m.activeConv.currentMessage.Timestamp = time.Now()

			// 更新统计
			m.activeConv.TokenUsage += msg.Usage
			m.totalTokens += msg.Usage
			m.inferenceCount++

			// 更新状态栏
			m.statusBar.SetTokens(m.totalTokens)
			m.statusBar.SetInferenceCount(m.inferenceCount)

			m.isStreaming = false

		case "error":
			// 处理错误
			m.lastError = msg.Error.Error()
			m.isStreaming = false
		}
	}

	// 更新对话视图
	m.updateChatView()

	// 自动滚动到底部
	if m.activeConv.currentMessage.IsStreaming {
		m.chatView.ScrollToBottom()
	}
}

// ========== 工作目录管理 ==========

// handleCDCommand 处理 /cd 命令
func (m AppModel) handleCDCommand(args []string) tea.Cmd {
	// 检查参数
	if len(args) == 0 {
		return func() tea.Msg {
			return NewCommandExecutedMsg("cd", args, "", fmt.Errorf("缺少目录参数"))
		}
	}

	// 获取目标目录
	targetDir := args[0]

	// 处理特殊路径
	if targetDir == "~" {
		// 用户主目录
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return func() tea.Msg {
				return NewCommandExecutedMsg("cd", args, "", fmt.Errorf("无法获取用户主目录: %w", err))
			}
		}
		targetDir = homeDir
	} else if targetDir == "-" {
		// 上一个目录（暂不支持，返回错误）
		return func() tea.Msg {
			return NewCommandExecutedMsg("cd", args, "", fmt.Errorf("暂不支持切换到上一个目录"))
		}
	}

	// 解析相对路径为绝对路径
	if !filepath.IsAbs(targetDir) {
		targetDir = filepath.Join(m.currentDir, targetDir)
	}

	// 清理路径（处理 . 和 ..）
	targetDir = filepath.Clean(targetDir)

	// 验证目录
	if err := m.validateDirectory(targetDir); err != nil {
		return func() tea.Msg {
			return NewCommandExecutedMsg("cd", args, "", err)
		}
	}

	// 保存旧目录
	oldDir := m.currentDir

	// 切换目录
	if err := os.Chdir(targetDir); err != nil {
		return func() tea.Msg {
			return NewCommandExecutedMsg("cd", args, "", fmt.Errorf("切换目录失败: %w", err))
		}
	}

	// 返回工作目录变更消息
	return func() tea.Msg {
		return NewWorkingDirChangedMsg(oldDir, targetDir)
	}
}

// validateDirectory 验证目录是否存在且可访问
func (m AppModel) validateDirectory(dir string) error {
	// 检查路径是否存在
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("目录不存在: %s", dir)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("没有权限访问目录: %s", dir)
		}
		return fmt.Errorf("无法访问目录: %w", err)
	}

	// 检查是否为目录
	if !info.IsDir() {
		return fmt.Errorf("路径不是目录: %s", dir)
	}

	return nil
}

// handleWorkingDirChanged 处理工作目录变更
func (m *AppModel) handleWorkingDirChanged(oldDir, newDir string) {
	// 更新当前目录
	m.currentDir = newDir

	// 更新状态栏显示
	m.statusBar.SetDirectory(newDir)
}

// handleCommandExecuted 处理命令执行结果
func (m *AppModel) handleCommandExecuted(cmd string, args []string, result string, err error) {
	if m.activeConv == nil {
		return
	}

	// 创建系统消息
	var systemMsg Message
	if err != nil {
		// 错误消息
		systemMsg = Message{
			Role:      MessageRoleAssistant,
			Content:   fmt.Sprintf("❌ 命令执行失败: %s\n错误: %v", cmd, err),
			Timestamp: time.Now(),
		}
	} else if result != "" {
		// 成功消息
		systemMsg = Message{
			Role:      MessageRoleAssistant,
			Content:   fmt.Sprintf("✓ %s", result),
			Timestamp: time.Now(),
		}
	} else {
		// 无结果消息
		systemMsg = Message{
			Role:      MessageRoleAssistant,
			Content:   fmt.Sprintf("✓ 命令 %s 执行成功", cmd),
			Timestamp: time.Now(),
		}
	}

	// 添加到当前对话
	m.activeConv.Messages = append(m.activeConv.Messages, systemMsg)
	m.updateChatView()
}

// SetCurrentDir 设置当前工作目录（供外部调用）
func (m *AppModel) SetCurrentDir(dir string) error {
	// 验证目录
	if err := m.validateDirectory(dir); err != nil {
		return err
	}

	// 切换目录
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("切换目录失败: %w", err)
	}

	// 更新状态
	m.currentDir = dir
	m.statusBar.SetDirectory(dir)

	// 发送工作目录变更消息（如果需要）
	// 这里不发送消息，因为这是外部调用，不需要触发消息循环

	return nil
}

// GetCurrentDir 获取当前工作目录
func (m *AppModel) GetCurrentDir() string {
	return m.currentDir
}

// ========== 其他命令处理 ==========

// handleClearCommand 处理 /clear 命令
func (m AppModel) handleClearCommand() tea.Cmd {
	if m.activeConv != nil {
		// 清空当前对话的消息
		m.activeConv.Messages = make([]Message, 0)
		m.activeConv.TokenUsage = 0
		m.updateChatView()
	}

	return func() tea.Msg {
		return NewCommandExecutedMsg("clear", nil, "对话已清空", nil)
	}
}

// handleSaveCommand 处理 /save 命令
func (m AppModel) handleSaveCommand(args []string) tea.Cmd {
	// 检查是否有活动对话
	if m.activeConv == nil {
		return func() tea.Msg {
			return NewCommandExecutedMsg("save", args, "", fmt.Errorf("没有活动对话"))
		}
	}

	// 获取文件路径（可选）
	var filePath string
	if len(args) > 0 {
		filePath = args[0]
	}

	// 保存对话到文件
	savedPath, err := SaveConversationToFile(filePath, m.activeConv)
	if err != nil {
		return func() tea.Msg {
			return NewCommandExecutedMsg("save", args, "", fmt.Errorf("保存失败: %w", err))
		}
	}

	// 返回成功消息
	result := fmt.Sprintf("对话已保存到: %s", savedPath)
	return func() tea.Msg {
		return NewCommandExecutedMsg("save", args, result, nil)
	}
}

// handleHelpCommand 处理 /help 命令
func (m AppModel) handleHelpCommand() tea.Cmd {
	// 创建输入组件实例以获取帮助文本
	inputComp := components.NewInputComponent(components.DefaultInputComponentConfig())
	helpText := inputComp.GetCommandHelp()

	return func() tea.Msg {
		return NewCommandExecutedMsg("help", nil, helpText, nil)
	}
}
