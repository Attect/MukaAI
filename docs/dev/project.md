# 项目架构文档

## 更新日志

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
- internal/state: 待实现
