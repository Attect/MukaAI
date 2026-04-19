package gui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Attect/MukaAI/internal/agent"
	"github.com/Attect/MukaAI/internal/model"

	"gopkg.in/yaml.v3"
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
	ctx          context.Context
	agent        *agent.Agent
	mu           sync.RWMutex
	eventEmitter EventEmitter // 事件发射器抽象，替代直接调用runtime.EventsEmit

	conversations  []*conversation
	activeConvID   string
	currentDir     string
	totalTokens    int
	inferenceCount int
	isStreaming    bool
	convStore      *ConversationStore // 对话持久化存储
	configPath     string             // 配置文件路径，用于GetSettings/SaveSettings
	terminalWSUrl  string             // 终端 WebSocket 连接地址
	logPath        string             // 日志文件路径
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

// Startup Wails生命周期回调，在应用启动时调用
// 必须为导出方法，以便外部包（如cmd/agentplus）在OnStartup回调中调用
// 初始化对话持久化存储并加载历史对话
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.eventEmitter = NewWailsEventEmitter(ctx)

	// 初始化对话持久化存储
	convDir := filepath.Join(a.currentDir, "state", "conversations")
	store, err := NewConversationStore(convDir)
	if err != nil {
		fmt.Printf("[App] 初始化对话存储失败: %v\n", err)
		return
	}
	a.convStore = store

	// 加载历史对话
	conversations, err := store.LoadAllConversations()
	if err != nil {
		fmt.Printf("[App] 加载历史对话失败: %v\n", err)
		return
	}
	if len(conversations) > 0 {
		a.conversations = conversations
		// 激活最后一个（最新的）对话
		a.activeConvID = conversations[len(conversations)-1].id
	}
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

// Shutdown 关闭应用资源
// 应在应用退出时调用，确保所有对话数据完整落盘
func (a *App) Shutdown() {
	if a.convStore != nil {
		a.convStore.Close()
	}
}

// SetConfigPath 设置配置文件路径
// 由外部初始化代码调用，用于GetSettings/SaveSettings操作
func (a *App) SetConfigPath(path string) {
	a.mu.Lock()
	a.configPath = path
	a.mu.Unlock()
}

// GetSettings 获取当前配置
// 读取配置文件，返回扁平化的配置map供前端设置页面使用
func (a *App) GetSettings() map[string]interface{} {
	a.mu.RLock()
	configPath := a.configPath
	a.mu.RUnlock()

	if configPath == "" {
		return map[string]interface{}{}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return map[string]interface{}{}
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return map[string]interface{}{}
	}

	// 将嵌套的YAML结构扁平化
	result := make(map[string]interface{})
	if m, ok := raw["model"].(map[string]interface{}); ok {
		result["endpoint"] = m["endpoint"]
		result["api_key"] = m["api_key"]
		result["model_name"] = m["model_name"]
		result["context_size"] = m["context_size"]
	}
	if ag, ok := raw["agent"].(map[string]interface{}); ok {
		result["temperature"] = ag["temperature"]
		result["max_iterations"] = ag["max_iterations"]
	}
	if t, ok := raw["tools"].(map[string]interface{}); ok {
		result["work_dir"] = t["work_dir"]
		if ac, ok := t["allow_commands"].([]interface{}); ok {
			result["allow_commands"] = ac
		}
	}

	return result
}

// SaveSettings 保存配置到YAML文件
// 接收前端传来的扁平化配置map，映射到YAML嵌套结构并写入文件
func (a *App) SaveSettings(settings map[string]interface{}) error {
	a.mu.RLock()
	configPath := a.configPath
	a.mu.RUnlock()

	if configPath == "" {
		return fmt.Errorf("config path not set")
	}

	// 读取现有配置以保留注释和其他字段
	raw := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		yaml.Unmarshal(data, &raw)
	}

	// 更新 model 部分
	if _, ok := raw["model"]; !ok {
		raw["model"] = make(map[string]interface{})
	}
	modelSection := raw["model"].(map[string]interface{})
	if v, ok := settings["endpoint"]; ok {
		modelSection["endpoint"] = v
	}
	if v, ok := settings["api_key"]; ok {
		modelSection["api_key"] = v
	}
	if v, ok := settings["model_name"]; ok {
		modelSection["model_name"] = v
	}
	if v, ok := settings["context_size"]; ok {
		modelSection["context_size"] = v
	}

	// 更新 agent 部分
	if _, ok := raw["agent"]; !ok {
		raw["agent"] = make(map[string]interface{})
	}
	agentSection := raw["agent"].(map[string]interface{})
	if v, ok := settings["temperature"]; ok {
		agentSection["temperature"] = v
	}
	if v, ok := settings["max_iterations"]; ok {
		agentSection["max_iterations"] = v
	}

	// 更新 tools 部分
	if _, ok := raw["tools"]; !ok {
		raw["tools"] = make(map[string]interface{})
	}
	toolsSection := raw["tools"].(map[string]interface{})
	if v, ok := settings["work_dir"]; ok {
		toolsSection["work_dir"] = v
	}
	if v, ok := settings["allow_commands"]; ok {
		toolsSection["allow_commands"] = v
	}

	// 写回文件
	outData, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, outData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// 热更新：仅model_name和context_size变更可即时对后续推理生效
	// endpoint和api_key变更需要重启（因为httpClient需要重建），热更新时会保留当前值
	if a.agent != nil {
		currentCfg := a.agent.GetModelConfig()
		if currentCfg != nil {
			// 使用当前config的endpoint和api_key，仅热更新model_name和context_size
			newCfg := &model.Config{
				Endpoint:    currentCfg.Endpoint, // 保持当前值，不允许热更新
				APIKey:      currentCfg.APIKey,   // 保持当前值，不允许热更新
				ModelName:   getString(modelSection, "model_name", currentCfg.ModelName),
				ContextSize: getInt(modelSection, "context_size", currentCfg.ContextSize),
			}

			if err := a.agent.UpdateModelConfig(newCfg); err != nil {
				// 热更新失败不影响文件保存，但需要通知前端显示警告
				a.emit("settings:hot-update-warning", err.Error())
			}
		}
	}

	return nil
}

// UpdateConversationTitle 更新对话标题
func (a *App) UpdateConversationTitle(id string, title string) error {
	a.mu.Lock()
	for _, conv := range a.conversations {
		if conv.id == id {
			conv.title = title
			a.saveConv(conv)
			a.mu.Unlock()
			a.emit("conversation:updated", a.GetConversationData())
			return nil
		}
	}
	a.mu.Unlock()
	return fmt.Errorf("conversation not found: %s", id)
}

// ExportConversation 导出对话为Markdown文件
// id为空时导出当前活跃对话，filename为空时自动生成文件名
func (a *App) ExportConversation(id string, filename string) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// 查找目标对话
	var conv *conversation
	if id == "" || id == a.activeConvID {
		conv = a.getActiveConversation()
	} else {
		for _, c := range a.conversations {
			if c.id == id {
				conv = c
				break
			}
		}
	}
	if conv == nil {
		return fmt.Errorf("conversation not found")
	}

	// 构建Markdown内容
	var sb strings.Builder
	sb.WriteString("# " + conv.title + "\n\n")
	sb.WriteString("导出时间: " + time.Now().Format("2006-01-02 15:04:05") + "\n\n---\n\n")

	for _, msg := range conv.messages {
		if msg.role == "user" {
			sb.WriteString("## User\n\n")
			sb.WriteString(msg.content + "\n\n")
		} else if msg.role == "assistant" {
			sb.WriteString("## Assistant\n\n")
			if msg.thinking != "" {
				sb.WriteString("<details><summary>思考过程</summary>\n\n")
				sb.WriteString(msg.thinking + "\n\n")
				sb.WriteString("</details>\n\n")
			}
			sb.WriteString(msg.content + "\n\n")
		}
	}

	// 确定文件名
	if filename == "" {
		safeTitle := strings.ReplaceAll(conv.title, " ", "_")
		safeTitle = strings.ReplaceAll(safeTitle, "/", "_")
		filename = fmt.Sprintf("%s_%s.md", safeTitle, time.Now().Format("20060102_150405"))
	}
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	path := filepath.Join(a.currentDir, filename)
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// saveConv 触发异步保存指定对话到磁盘
// 必须在持有a.mu锁的情况下调用，以保证快照数据一致性
func (a *App) saveConv(conv *conversation) {
	if a.convStore != nil && conv != nil {
		a.convStore.SaveConversation(conv)
	}
}

// DeleteConversation 删除指定对话
// 同时从内存和磁盘删除，如果删除的是当前活跃对话则切换到最后一个
func (a *App) DeleteConversation(id string) error {
	a.mu.Lock()
	if a.isStreaming {
		a.mu.Unlock()
		return fmt.Errorf("cannot delete conversation while streaming")
	}

	// 从内存切片中移除
	idx := -1
	for i, conv := range a.conversations {
		if conv.id == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		a.mu.Unlock()
		return fmt.Errorf("conversation not found: %s", id)
	}
	a.conversations = append(a.conversations[:idx], a.conversations[idx+1:]...)

	// 从磁盘删除
	if a.convStore != nil {
		a.convStore.DeleteConversation(id)
	}

	// 如果删除的是活跃对话，切换到剩余的最后一个
	if a.activeConvID == id {
		if len(a.conversations) > 0 {
			a.activeConvID = a.conversations[len(a.conversations)-1].id
		} else {
			a.activeConvID = ""
		}
	}
	a.mu.Unlock()

	a.emit("conversation:updated", a.GetConversationData())
	return nil
}

// SendMessage 发送用户消息并启动推理
// 这是前端调用的主要入口，会异步启动Agent推理过程
func (a *App) SendMessage(content string) error {
	if a.agent == nil {
		return fmt.Errorf("agent not initialized")
	}

	a.mu.Lock()
	if a.isStreaming {
		a.mu.Unlock()
		return fmt.Errorf("agent is already running")
	}

	conv := a.getOrCreateActiveConversation()
	conv.messages = append(conv.messages, &message{
		role:      "user",
		content:   content,
		timestamp: time.Now(),
	})
	conv.currentMessage = nil
	a.isStreaming = true
	a.saveConv(conv) // 持久化：保存用户消息
	a.mu.Unlock()

	a.emit("conversation:updated", a.GetConversationData())

	go func() {
		// 使用defer作为最终保障
		// StreamBridge.OnTaskDone会先执行重置，这里作为兜底防止isStreaming永久卡死
		defer func() {
			a.mu.Lock()
			if a.isStreaming {
				a.isStreaming = false
				a.mu.Unlock()
				a.emit("stream:done")
				a.emit("conversation:updated", a.GetConversationData())
			} else {
				a.mu.Unlock()
			}
		}()

		if err := a.agent.SendMessage(content); err != nil {
			a.mu.Lock()
			a.isStreaming = false
			a.mu.Unlock()
			a.emit("stream:error", err.Error())
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
		toolCalls := msg.toolCalls
		if toolCalls == nil {
			toolCalls = make([]ToolCall, 0)
		}
		messages = append(messages, Message{
			Role:          msg.role,
			Content:       msg.content,
			Thinking:      msg.thinking,
			ToolCalls:     toolCalls,
			TokenUsage:    msg.tokenUsage,
			IsStreaming:   msg.isStreaming,
			StreamingType: msg.streamingType,
			Timestamp:     msg.timestamp,
		})
	}

	if conv.currentMessage != nil {
		toolCalls := conv.currentMessage.toolCalls
		if toolCalls == nil {
			toolCalls = make([]ToolCall, 0)
		}
		messages = append(messages, Message{
			Role:          conv.currentMessage.role,
			Content:       conv.currentMessage.content,
			Thinking:      conv.currentMessage.thinking,
			ToolCalls:     toolCalls,
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
// 仅更新App的currentDir字段，不调用os.Chdir以避免竞态条件
func (a *App) SetWorkDir(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	// 验证路径是否存在
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", absPath)
	}
	a.mu.Lock()
	a.currentDir = absPath
	a.mu.Unlock()
	a.emit("workdir:changed", absPath)
	return nil
}

// GetWorkDir 获取当前工作目录
func (a *App) GetWorkDir() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
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
		a.saveConv(conv) // 持久化：保存打断时的消息
	}
	a.mu.Unlock()
	a.emit("stream:interrupted")
	a.emit("conversation:updated", a.GetConversationData())
}

// SwitchConversation 切换到指定ID的对话
// 如果正在推理中则拒绝切换，切换成功后返回新的对话数据
func (a *App) SwitchConversation(id string) error {
	a.mu.Lock()
	if a.isStreaming {
		a.mu.Unlock()
		return fmt.Errorf("cannot switch conversation while streaming")
	}

	// 查找目标对话
	var target *conversation
	for _, conv := range a.conversations {
		if conv.id == id {
			target = conv
			break
		}
	}
	if target == nil {
		a.mu.Unlock()
		return fmt.Errorf("conversation not found: %s", id)
	}

	a.activeConvID = id
	a.mu.Unlock()

	a.emit("conversation:updated", a.GetConversationData())
	return nil
}

// ClearConversation 清空当前对话的消息
// 保留对话本身，仅清除消息列表
func (a *App) ClearConversation() {
	a.mu.Lock()
	conv := a.getActiveConversation()
	if conv != nil {
		conv.messages = nil
		conv.currentMessage = nil
		a.saveConv(conv) // 持久化：保存清空后的状态
	}
	a.mu.Unlock()
	a.emit("conversation:updated", a.GetConversationData())
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

// SetTerminalWSUrl 设置终端 WebSocket 连接地址
// 由 GUI 初始化代码调用，将 WebSocket 服务器地址传递给前端
func (a *App) SetTerminalWSUrl(url string) {
	a.mu.Lock()
	a.terminalWSUrl = url
	a.mu.Unlock()
}

// GetTerminalWSUrl 获取终端 WebSocket 连接地址
// 前端通过 Wails 绑定调用此方法获取连接地址
func (a *App) GetTerminalWSUrl() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.terminalWSUrl
}

// SetEventEmitter 设置事件发射器
// 用于测试时注入MockEventEmitter，生产环境通过Startup自动设置
func (a *App) SetEventEmitter(emitter EventEmitter) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.eventEmitter = emitter
}

// emit 通过EventEmitter发射事件
// 内部辅助方法，统一事件发射入口
func (a *App) emit(event string, data ...interface{}) {
	a.mu.RLock()
	emitter := a.eventEmitter
	a.mu.RUnlock()

	if emitter != nil {
		emitter.Emit(event, data...)
	}
}

// SetLogPath 设置日志文件路径
// 由外部初始化代码调用，用于配置日志输出位置
func (a *App) SetLogPath(path string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.logPath = path
	fmt.Printf("[App] Log path configured: %s\n", path)
}

// GetLogPath 获取日志文件路径
func (a *App) GetLogPath() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.logPath
}

// getString 从YAML map中安全提取字符串值，缺失或类型不匹配时返回默认值
func getString(m map[string]interface{}, key string, defaultVal string) string {
	v, ok := m[key]
	if !ok {
		return defaultVal
	}
	s, ok := v.(string)
	if !ok {
		return defaultVal
	}
	return s
}

// getInt 从YAML map中安全提取整数值，缺失或类型不匹配时返回默认值
// YAML解析int可能产生int、float64或int64，需统一处理
func getInt(m map[string]interface{}, key string, defaultVal int) int {
	v, ok := m[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	default:
		return defaultVal
	}
}
