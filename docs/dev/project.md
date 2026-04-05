# 项目架构文档

## 更新日志

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

#### internal/agent - Agent核心模块（待实现）
Agent主循环和核心逻辑：
- Agent主循环
- 程序逻辑审查
- 子代理Fork机制

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
