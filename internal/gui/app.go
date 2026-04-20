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
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"gopkg.in/yaml.v3"
	"unicode/utf8"
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

	// toolWorkDirUpdater 工具工作目录更新回调
	// 由初始化代码设置，用于在切换工作目录时同步更新所有工具的workDir
	toolWorkDirUpdater func(string)
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
	} else {
		// 没有历史对话，创建一个新的默认对话
		a.createDefaultConversation()
	}

	// 通知前端对话列表已更新（Startup 后首次加载）
	a.emit("conversation:updated", a.GetConversationData())
	a.emit("conversations:updated", a.GetConversations())
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
	// 确保配置目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
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

// GenerateConversationTitle 根据对话内容自动生成标题
// 如果标题已经是"新对话"，使用第一条用户消息和第一条助手消息生成标题
// 如果标题已被手动设置过，使用完整对话内容（排除系统消息）生成标题
// 调用LLM非流式API生成简短中文标题（10字以内），24小时超时（适配本地模型慢速推理）
func (a *App) GenerateConversationTitle(id string) error {
	a.mu.RLock()
	var conv *conversation
	for _, c := range a.conversations {
		if c.id == id {
			conv = c
			break
		}
	}
	a.mu.RUnlock()

	if conv == nil {
		return fmt.Errorf("conversation not found: %s", id)
	}

	// 获取模型客户端
	modelClient := a.agent.GetModelClient()
	if modelClient == nil {
		return fmt.Errorf("model client not initialized")
	}

	// 构建标题生成请求的消息历史
	var messages []model.Message

	// 判断是否为新对话（只有1条用户消息+1条助手消息）
	userMsgCount := 0
	for _, msg := range conv.messages {
		if msg.role == "user" {
			userMsgCount++
		}
	}

	if conv.title == "新对话" && userMsgCount <= 1 && len(conv.messages) <= 2 {
		// 新对话：仅使用首条用户消息和首条助手消息
		messages = a.buildTitlePromptFromFirstPair(conv.messages)
	} else {
		// 已有标题的对话：使用完整对话内容（排除系统消息）
		messages = a.buildTitlePromptFromFullConversation(conv.messages)
	}

	if len(messages) == 0 {
		return fmt.Errorf("no messages available for title generation")
	}

	// 构建标题生成提示
	titleMessages := []model.Message{
		model.NewUserMessage(titleGenerationPrompt),
	}
	titleMessages = append(titleMessages, messages...)

	// 使用24小时超时调用LLM（适配本地模型慢速推理和排队场景）
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()

	resp, err := modelClient.ChatCompletion(ctx, titleMessages, nil)
	if err != nil {
		// 生成失败，不修改原有标题
		fmt.Printf("[App] 标题生成失败 [%s]: %v\n", id, err)
		return fmt.Errorf("title generation failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return fmt.Errorf("empty title response")
	}

	newTitle := strings.TrimSpace(resp.Choices[0].Message.Content)

	// 更新标题并持久化
	a.mu.Lock()
	for _, c := range a.conversations {
		if c.id == id {
			c.title = newTitle
			a.saveConv(c)
			break
		}
	}
	a.mu.Unlock()

	fmt.Printf("[App] 标题已生成 [%s]: %s\n", id, newTitle)
	return nil
}

// RegenerateConversationTitle 基于完整对话内容重新生成标题
// 无论当前标题是什么，都使用完整对话内容重新生成
func (a *App) RegenerateConversationTitle(id string) error {
	a.mu.RLock()
	var conv *conversation
	for _, c := range a.conversations {
		if c.id == id {
			conv = c
			break
		}
	}
	a.mu.RUnlock()

	if conv == nil {
		return fmt.Errorf("conversation not found: %s", id)
	}

	modelClient := a.agent.GetModelClient()
	if modelClient == nil {
		return fmt.Errorf("model client not initialized")
	}

	// 使用完整对话内容（排除系统消息）
	messages := a.buildTitlePromptFromFullConversation(conv.messages)
	if len(messages) == 0 {
		return fmt.Errorf("no messages available for title regeneration")
	}

	titleMessages := []model.Message{
		model.NewUserMessage(titleGenerationPrompt),
	}
	titleMessages = append(titleMessages, messages...)

	// 使用24小时超时调用LLM（适配本地模型慢速推理和排队场景）
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()

	resp, err := modelClient.ChatCompletion(ctx, titleMessages, nil)
	if err != nil {
		fmt.Printf("[App] 标题重新生成失败 [%s]: %v\n", id, err)
		return fmt.Errorf("title regeneration failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return fmt.Errorf("empty title response")
	}

	newTitle := strings.TrimSpace(resp.Choices[0].Message.Content)

	a.mu.Lock()
	for _, c := range a.conversations {
		if c.id == id {
			c.title = newTitle
			a.saveConv(c)
			break
		}
	}
	a.mu.Unlock()

	fmt.Printf("[App] 标题已重新生成 [%s]: %s\n", id, newTitle)
	return nil
}

// GetCompressionSummary 返回指定对话的最后一次压缩摘要（如果有）
func (a *App) GetCompressionSummary(id string) (string, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, conv := range a.conversations {
		if conv.id == id {
			var lastSummary string
			for _, msg := range conv.messages {
				if msg.role == "user" && strings.Contains(msg.content, "[上下文摘要") {
					lastSummary = msg.content
				}
			}
			return lastSummary, lastSummary != ""
		}
	}
	return "", false
}

// buildTitlePromptFromFirstPair 从第一条用户消息和第一条助手消息构建标题生成消息
func (a *App) buildTitlePromptFromFirstPair(msgs []*message) []model.Message {
	var result []model.Message
	var userMsg, assistantContent string

	for _, msg := range msgs {
		if msg.role == "user" && userMsg == "" {
			userMsg = msg.content
		} else if msg.role == "assistant" && assistantContent == "" {
			// 标题生成只使用文本内容，不使用 tool_calls
			// 因为 API 可能启用了 enable_thinking，与 tool_calls 冲突
			if msg.content != "" {
				assistantContent = msg.content
			} else if len(msg.thinking) > 0 {
				// 优先使用 thinking 内容作为助手回复摘要（按字符数截断）
				runes := []rune(msg.thinking)
				if utf8.RuneCountInString(msg.thinking) > 100 {
					runes = runes[:100]
				}
				assistantContent = "[思考中]" + string(runes)
			}
		}
		if userMsg != "" && assistantContent != "" {
			break
		}
	}

	if userMsg != "" {
		result = append(result, model.NewUserMessage(userMsg))
	}
	if assistantContent != "" {
		result = append(result, model.NewAssistantMessage(assistantContent))
	}
	return result
}

// buildTitlePromptFromFullConversation 从完整对话构建标题生成消息（排除系统消息）
// 如果消息超过10条，取前5对user-assistant+最近3对user-assistant
func (a *App) buildTitlePromptFromFullConversation(msgs []*message) []model.Message {
	// 过滤掉system消息
	var filtered []*message
	for _, msg := range msgs {
		if msg.role != "system" {
			filtered = append(filtered, msg)
		}
	}

	// 计算user-assistant对数
	var pairs [][2]*message
	var userMsg *message
	for _, msg := range filtered {
		if msg.role == "user" {
			userMsg = msg
		} else if msg.role == "assistant" && userMsg != nil {
			pairs = append(pairs, [2]*message{userMsg, msg})
			userMsg = nil
		}
	}

	if len(pairs) == 0 {
		return nil
	}

	var result []model.Message

	if len(pairs) <= 10 {
		for _, pair := range pairs {
			result = append(result, model.NewUserMessage(pair[0].content))
			asm := a.buildAssistantMessage(pair[1])
			// 只包含有文本内容的助手消息（标题生成不使用 tool_calls）
			if asm.Content != "" {
				result = append(result, asm)
			}
		}
	} else {
		selected := make(map[int]bool)
		for i := 0; i < 5 && i < len(pairs); i++ {
			selected[i] = true
		}
		for i := len(pairs) - 3; i < len(pairs); i++ {
			selected[i] = true
		}

		for i, pair := range pairs {
			if selected[i] {
				result = append(result, model.NewUserMessage(pair[0].content))
				asm := a.buildAssistantMessage(pair[1])
				if asm.Content != "" {
					result = append(result, asm)
				}
			}
		}
	}

	return result
}

// buildAssistantMessage 构建模型消息用于标题生成，只使用文本内容
// 不包含 tool_calls，因为 API 可能启用了 enable_thinking，与 tool_calls 冲突
func (a *App) buildAssistantMessage(msg *message) model.Message {
	if msg == nil {
		return model.Message{}
	}
	if msg.content != "" {
		return model.NewAssistantMessage(msg.content)
	}
	// 如果只有 tool_calls 没有 content，返回空消息供调用方过滤
	return model.Message{}
}

const titleGenerationPrompt = `请为以下对话生成一个简短的中文标题（不超过10个字），要求：
1. 准确概括对话主题
2. 简洁明了
3. 只输出标题本身，不要包含其他内容`

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

	// 构建对话历史（不包括当前用户消息），注入到 Agent
	history := a.buildConversationHistory(conv.messages[:len(conv.messages)-1])
	a.agent.SetExternalHistory(history)

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

// buildConversationHistory 将内部消息列表转换为 Agent 可用的历史消息格式
// 保留 tool_calls 字段，避免产生既无 content 也无 tool_calls 的非法 assistant 消息
func (a *App) buildConversationHistory(msgs []*message) []model.Message {
	history := make([]model.Message, 0, len(msgs))
	for _, m := range msgs {
		role := model.RoleAssistant
		if m.role == "user" {
			role = model.RoleUser
		}

		msg := model.Message{
			Role:    role,
			Content: m.content,
		}

		// 转换 tool_calls，保留给模型的工具备忘
		if len(m.toolCalls) > 0 {
			modelToolCalls := make([]model.ToolCall, 0, len(m.toolCalls))
			for _, tc := range m.toolCalls {
				// 解析 arguments JSON 字符串
				var args string
				if tc.Arguments != "" {
					args = tc.Arguments
				}
				modelToolCalls = append(modelToolCalls, model.ToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: model.FunctionCall{
						Name:      tc.Name,
						Arguments: args,
					},
				})
			}
			msg.ToolCalls = modelToolCalls
		}

		history = append(history, msg)
	}
	return history
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

// ChooseDirectory 弹出系统目录选择对话框，返回用户选择的目录路径
func (a *App) ChooseDirectory() string {
	if a.ctx == nil {
		return ""
	}
	chosen, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择工作目录",
	})
	if err != nil {
		fmt.Printf("[App] ChooseDirectory error: %v\n", err)
		return ""
	}
	return chosen
}

// SetWorkDir 设置工作目录
// 更新App的currentDir、Agent的工作目录以及所有工具的workDir（不直接调用os.Chdir）
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

	// 同步更新Agent的工作目录（影响后续任务提示中的路径信息）
	if a.agent != nil {
		a.agent.SetWorkDir(absPath)
	}

	// 同步更新所有工具的工作目录（文件系统工具、Git工具、安全审查器等）
	a.mu.RLock()
	updater := a.toolWorkDirUpdater
	a.mu.RUnlock()
	if updater != nil {
		updater(absPath)
	}

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

	// 切换对话时，将目标对话的历史注入到 Agent
	if a.agent != nil {
		a.agent.SetExternalHistory(a.buildConversationHistory(target.messages))
	}

	a.emit("conversation:updated", a.GetConversationData())
	return nil
}

// NewConversation 创建一个新的对话并设为活跃状态
// 如果 Agent 已初始化，会清空 Agent 的外部历史
func (a *App) NewConversation() error {
	a.mu.Lock()
	if a.isStreaming {
		a.mu.Unlock()
		return fmt.Errorf("cannot create conversation while streaming")
	}

	a.createDefaultConversation()
	a.mu.Unlock()

	// 清空 Agent 的外部历史（新对话没有历史）
	if a.agent != nil {
		a.agent.SetExternalHistory(nil)
	}

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

// createDefaultConversation 创建一个新的默认对话并设为活跃状态
// 调用方必须持有a.mu写锁
func (a *App) createDefaultConversation() {
	conv := &conversation{
		id:        fmt.Sprintf("conv-%d", time.Now().UnixMilli()),
		title:     "新对话",
		createdAt: time.Now(),
		status:    "active",
	}
	a.conversations = append(a.conversations, conv)
	a.activeConvID = conv.id
}

// getOrCreateActiveConversation 获取或创建活跃对话
// 如果不存在任何对话，则创建一个新的默认对话
// 调用方必须持有a.mu写锁
func (a *App) getOrCreateActiveConversation() *conversation {
	if len(a.conversations) == 0 || a.activeConvID == "" {
		a.createDefaultConversation()
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

// SetToolWorkDirUpdater 设置工具工作目录更新回调
// 由初始化代码调用，传入一个函数用于在切换工作目录时同步更新所有工具的workDir
func (a *App) SetToolWorkDirUpdater(updater func(string)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.toolWorkDirUpdater = updater
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
