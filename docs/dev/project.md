# 项目架构文档

## 更新日志

### 2026-04-06 Task 5 新增：Agent核心循环模块

#### internal/agent/prompts.go
- `SystemPromptType` - 系统提示词类型（orchestrator/worker/reviewer）
- `OrchestratorSystemPrompt` - Orchestrator角色系统提示词（高效执行模式）
- `WorkerSystemPrompt` - Worker角色系统提示词
- `ReviewerSystemPrompt` - Reviewer角色系统提示词
- `YAMLStatePrompt` - YAML状态维护提示
- `GetSystemPrompt(promptType) string` - 根据类型获取系统提示词
- `BuildTaskPrompt(taskGoal, stateSummary) string` - 构建任务提示词
- `BuildToolResultPrompt(toolName, result) string` - 构建工具结果提示词
- `BuildErrorPrompt(err) string` - 构建错误提示词

#### internal/agent/history.go
- `HistoryManager` - 消息历史管理器（线程安全）
  - 使用sync.RWMutex保护并发访问
  - 支持消息添加、获取、截断等操作
- `NewHistoryManager() *HistoryManager` - 创建新的消息历史管理器
- `AddMessage(msg)` - 添加消息到历史
- `AddMessages(msgs)` - 批量添加消息
- `GetMessages() []Message` - 获取所有消息的副本
- `GetMessagesRef() []Message` - 获取消息引用（只读）
- `GetLastMessage() (Message, bool)` - 获取最后一条消息
- `GetMessageCount() int` - 获取消息数量
- `Clear()` - 清空消息历史
- `Truncate(maxTokens, tokenCounter)` - 截断历史以适应token限制
- `TruncateSimple(keepCount)` - 简单截断策略
- `GetTokenCount(tokenCounter) int` - 获取当前历史的token数量
- `GetMessagesByRole(role) []Message` - 获取指定角色的消息
- `RemoveLastMessage() bool` - 移除最后一条消息
- `ReplaceLastMessage(msg) bool` - 替换最后一条消息
- `GetRecentMessages(n) []Message` - 获取最近N条消息
- `Clone() *HistoryManager` - 克隆历史管理器

#### internal/agent/executor.go
- `ToolExecutor` - 工具调用执行器
- `NewToolExecutor(registry) *ToolExecutor` - 创建新的工具执行器
- `ExecuteToolCalls(ctx, toolCalls) ([]Message, error)` - 执行工具调用列表（并行）
- `ExecuteToolCallWithTimeout(ctx, tc, timeout) (Message, error)` - 执行工具调用（带超时）
- `ExecuteToolCallSequential(ctx, toolCalls) ([]Message, error)` - 顺序执行工具调用
- `ToolExecutionResult` - 工具执行结果（包含详细信息）
- `ExecuteToolCallsWithDetails(ctx, toolCalls) ([]ToolExecutionResult, error)` - 执行工具调用并返回详细信息
- `GetAvailableTools() []Tool` - 获取可用工具列表
- `GetToolSchemas() []model.Tool` - 获取工具Schema列表
- `HasTool(name) bool` - 检查工具是否存在
- `ParseToolCallArguments(tc) (map, error)` - 解析工具调用参数
- `BuildToolResultMessage(tc, result) Message` - 构建工具结果消息
- `BuildToolErrorMessage(tc, errMsg) Message` - 构建工具错误消息

#### internal/agent/core.go
- `Agent` - 核心Agent结构体
  - modelClient: 模型客户端
  - toolRegistry: 工具注册中心
  - stateManager: 状态管理器
  - executor: 工具执行器
  - history: 消息历史
  - maxIterations: 最大迭代次数
  - systemPrompt: 系统提示词
- `Config` - Agent配置结构体
- `NewAgent(config) (*Agent, error)` - 创建新的Agent实例
- `Run(ctx, taskGoal) (*RunResult, error)` - 执行任务主循环
- `RunResult` - 运行结果结构体
  - TaskID: 任务ID
  - Status: 状态（completed/failed/cancelled/max_iterations）
  - StartTime/EndTime: 开始/结束时间
  - Duration: 执行时长
  - Iterations: 迭代次数
  - FinalResponse: 最终响应内容
  - Error: 错误信息
- `Stop()` - 停止Agent运行
- `IsRunning() bool` - 检查Agent是否正在运行
- `SetTaskID(taskID)` - 设置任务ID
- `GetTaskID() string` - 获取当前任务ID
- `SetOnStreamChunk(callback)` - 设置流式输出回调
- `SetOnToolCall(callback)` - 设置工具调用回调
- `SetOnIteration(callback)` - 设置迭代回调
- `GetHistory() *HistoryManager` - 获取消息历史管理器
- `GetState() (*TaskState, error)` - 获取当前任务状态
- `RunSync(ctx, taskGoal) error` - 同步执行任务（简化接口）

### 2026-04-06 Task 4 新增：YAML状态管理模块

#### internal/state/task.go
- `TaskState` - 完整的任务状态文档结构
  - `Task TaskInfo` - 任务基本信息
  - `Progress ProgressInfo` - 任务进度信息
  - `Context ContextInfo` - 任务上下文信息
  - `Agents AgentsInfo` - Agent团队状态
- `TaskInfo` - 任务基本信息结构
  - `ID string` - 任务唯一标识符
  - `Goal string` - 任务目标描述
  - `Status string` - 任务状态（pending/in_progress/completed/failed）
  - `CreatedAt time.Time` - 创建时间
  - `UpdatedAt time.Time` - 最后更新时间
- `ProgressInfo` - 任务进度信息结构
  - `CurrentPhase string` - 当前阶段名称
  - `CompletedSteps []string` - 已完成的步骤列表
  - `PendingSteps []string` - 待完成的步骤列表
- `ContextInfo` - 任务上下文信息结构
  - `Decisions []string` - 决策记录
  - `Constraints []string` - 约束条件
  - `Files []FileInfo` - 相关文件信息
- `FileInfo` - 文件信息结构
  - `Path string` - 文件路径
  - `Description string` - 文件描述
  - `Status string` - 文件状态
- `AgentsInfo` - Agent团队状态结构
  - `Active string` - 当前活动的Agent角色
  - `History []AgentRecord` - Agent执行历史记录
- `AgentRecord` - Agent执行记录结构
  - `Role string` - Agent角色
  - `Summary string` - 执行摘要
  - `Duration string` - 执行时长
- `NewTaskState(id, goal string) *TaskState` - 创建新的任务状态实例
- `IsCompleted() bool` - 检查任务是否已完成
- `IsInProgress() bool` - 检查任务是否正在进行中
- `IsFailed() bool` - 检查任务是否失败
- `UpdateStatus(status string)` - 更新任务状态
- `AddCompletedStep(step string)` - 添加已完成的步骤
- `RemovePendingStep(step string)` - 从待办步骤中移除指定步骤
- `AddDecision(decision string)` - 添加决策记录
- `AddConstraint(constraint string)` - 添加约束条件
- `AddFile(path, description, status string)` - 添加相关文件信息
- `AddAgentRecord(role, summary, duration string)` - 添加Agent执行记录
- `SetActiveAgent(role string)` - 设置当前活动的Agent
- `SetCurrentPhase(phase string)` - 设置当前阶段

#### internal/state/yaml.go
- `LoadYAML(filePath string) (*TaskState, error)` - 从文件加载YAML
- `SaveYAML(state *TaskState, filePath string) error` - 保存YAML到文件
- `ParseYAML(data []byte) (*TaskState, error)` - 解析YAML字符串
- `ToYAML(state *TaskState) ([]byte, error)` - 序列化为YAML字节数据
- `ToYAMLString(state *TaskState) (string, error)` - 序列化为YAML字符串
- `GetYAMLSummary(state *TaskState) (string, error)` - 获取YAML摘要（用于上下文压缩）

#### internal/state/manager.go
- `StateManager` - 状态管理器（并发安全）
  - 使用sync.RWMutex保护并发访问
  - 支持内存缓存和文件持久化
  - 支持自动保存功能
- `NewStateManager(stateDir string, autoSave bool) (*StateManager, error)` - 创建状态管理器
- `CreateTask(id, goal string) (*TaskState, error)` - 创建新任务状态
- `Load(id string) (*TaskState, error)` - 从文件加载任务状态
- `Save(id string) error` - 保存任务状态到文件
- `UpdateProgress(id, phase, completedStep string) error` - 更新任务进度
- `AddDecision(id, decision string) error` - 添加决策记录
- `CompleteStep(id, step string) error` - 完成一个步骤
- `SwitchAgent(id, role, summary, duration string) error` - 切换活动Agent
- `GetState(id string) (*TaskState, error)` - 获取任务状态（只读）
- `GetYAMLSummary(id string) (string, error)` - 获取任务的YAML摘要
- `SetPendingSteps(id string, steps []string) error` - 设置待完成步骤列表
- `AddConstraint(id, constraint string) error` - 添加约束条件
- `AddFile(id, path, description, status string) error` - 添加相关文件信息
- `UpdateTaskStatus(id, status string) error` - 更新任务状态
- `ListTasks() ([]string, error)` - 列出所有已知任务ID
- `DeleteTask(id string) error` - 删除任务状态

### 2026-04-06 Task 2 新增：模型服务连接模块

#### internal/model/config.go
- `Config` - 模型服务配置结构体
  - `Endpoint string` - API端点地址
  - `APIKey string` - API密钥
  - `ModelName string` - 模型名称
  - `ContextSize int` - 上下文大小
- `DefaultConfig() *Config` - 返回默认配置
- `Validate() error` - 验证配置有效性
- `ConfigError` - 配置错误类型

#### internal/model/message.go
- `Role` - 消息角色类型（system/user/assistant/tool）
- `Message` - 聊天消息结构体
  - `Role Role` - 消息角色
  - `Content string` - 消息内容
  - `ToolCalls []ToolCall` - 工具调用请求
  - `ToolCallID string` - 工具调用ID
- `ToolCall` - 工具调用请求结构体
- `FunctionCall` - 函数调用详情
- `Tool` - 工具定义结构体
- `FunctionDef` - 函数定义结构体
- `ChatCompletionRequest` - 聊天补全请求
- `ChatCompletionResponse` - 聊天补全响应
- `Choice` - 响应选项
- `Delta` - 流式响应增量内容
- `Usage` - token使用统计
- `StreamResponse` - 流式响应结构
- `NewSystemMessage(content string) Message` - 创建系统消息
- `NewUserMessage(content string) Message` - 创建用户消息
- `NewAssistantMessage(content string) Message` - 创建助手消息
- `NewToolResultMessage(toolCallID, name, content string) Message` - 创建工具结果消息
- `ParseToolCallArguments() (map[string]interface{}, error)` - 解析工具调用参数

#### internal/model/client.go
- `Client` - OpenAI API兼容客户端
- `NewClient(config *Config) (*Client, error)` - 创建新的模型客户端
- `ChatCompletion(ctx context.Context, messages []Message, tools []Tool) (*ChatCompletionResponse, error)` - 发送聊天补全请求
- `ChatCompletionWithTemperature(ctx context.Context, messages []Message, tools []Tool, temperature float64) (*ChatCompletionResponse, error)` - 发送带温度参数的聊天补全请求
- `StreamChatCompletion(ctx context.Context, messages []Message, tools []Tool) (<-chan StreamEvent, error)` - 流式聊天补全
- `StreamEvent` - 流式事件结构
- `GetConfig() *Config` - 获取客户端配置
- `CountTokens(messages []Message) int` - 估算消息的token数量
- `IsContextOverflow(messages []Message) bool` - 检查是否超出上下文限制
- `RequestError` - 请求错误类型
- `APIError` - API错误类型

## 架构说明

### 模块划分

#### internal/model - 模型服务连接模块
负责与OpenAI API兼容的模型服务进行通信，支持：
- 标准Chat Completion API调用
- 流式响应处理（SSE）
- Function Calling/Tool Calling
- 思考标签处理
- 上下文管理

#### internal/tools - 工具集模块
提供Agent可调用的基础工具：
- 文件系统操作（读写、列表、删除）
- 命令执行
- 工具注册机制

#### internal/state - 状态管理模块
使用YAML格式维护任务状态：
- YAML解析与序列化
- 任务状态结构定义
- 状态自动更新机制

#### internal/agent - Agent核心模块
Agent主循环和核心逻辑：
- Agent主循环（Run方法）
- 消息历史管理
- 工具调用执行
- 系统提示词管理
- 状态自动更新机制
- 流式响应处理
- 思考标签处理
- 高效执行模式

#### internal/supervisor - 监督系统（待实现）
外部监督Agent行为：
- 实时监督
- 监督干预

#### internal/team - 团队定义（待实现）
定义Agent团队和角色：
- 团队结构定义
- 角色职责管理

## 数据流

1. **用户输入** -> Agent核心
2. Agent核心 -> **模型服务** (ChatCompletion/StreamChatCompletion)
3. 模型服务 -> **工具调用** (Tool Calling)
4. 工具执行 -> **状态更新** (YAML State)
5. 状态更新 -> Agent核心 -> **循环或完成**

## 配置说明

配置文件位于 `configs/config.yaml`，包含：
- model: 模型服务配置
- agent: Agent行为配置
- state: 状态管理配置
- tools: 工具配置

## 测试覆盖

所有模块均包含单元测试，测试文件位于对应模块目录下，命名格式为 `*_test.go`。

当前测试覆盖：
- internal/model: 100% 核心功能覆盖
- internal/tools: 待实现
- internal/state: 100% 核心功能覆盖
  - 任务状态创建和操作
  - YAML序列化与反序列化
  - 文件读写操作
  - 状态管理器功能
  - 并发安全性验证
  - 时间字段序列化
- internal/agent: 100% 核心功能覆盖
  - Agent创建和配置
  - 消息历史管理
  - 工具执行器
  - Agent主循环
  - 取消和并发控制
  - 回调机制
