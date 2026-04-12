# AgentPlus 项目架构文档

> 版本: v1.0.0
> 生成时间: 2026-04-11
> 生成者: code-framework（架构师AGENT）
> 分析方法: 源码逆向分析

---

## 一、系统概览

### 1.1 项目定位

AgentPlus是一个基于Go语言和React前端构建的**AI Agent桌面应用程序**，支持CLI命令行和GUI图形界面双模式运行。系统通过OpenAI兼容的API与LLM模型通信，具备完整的任务规划、工具调用、审查校验、自我修正和子代理Fork能力。

### 1.2 系统架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        AgentPlus 系统                           │
│                                                                 │
│  ┌─────────────────┐      ┌──────────────────────────────────┐  │
│  │   CLI 模式       │      │         GUI 模式 (Wails)         │  │
│  │  main.go →      │      │  App ←→ StreamBridge ←→ Wails    │  │
│  │  runCLICommand() │      │          事件系统                │  │
│  └────────┬────────┘      └──────────────┬───────────────────┘  │
│           │                              │                      │
│           └──────────┬───────────────────┘                      │
│                      ▼                                          │
│           ┌─────────────────────┐                               │
│           │   Agent 核心        │                               │
│           │  (双层主循环)       │                               │
│           │  Run() → callModel  │                               │
│           │  → Review → Execute │                               │
│           │  → Verify → Correct │                               │
│           └──┬──┬──┬──┬──┬──┬──┘                               │
│              │  │  │  │  │  │                                    │
│  ┌───────────┘  │  │  │  │  └──────────┐                       │
│  ▼              ▼  ▼  ▼  ▼             ▼                       │
│ Reviewer  Verifier  Compressor  ForkManager                    │
│ SelfCorrector   HistoryManager   AgentLogger                   │
│ ThinkingTagProcessor  FeedbackInjector                         │
│                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │  Model   │  │  Tools   │  │  State   │  │   Team   │       │
│  │  Client  │  │ Registry │  │ Manager  │  │ Manager  │       │
│  │ (SSE流式) │  │ (11工具)  │  │ (YAML)   │  │ (6角色)   │       │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘       │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              外部依赖                                    │   │
│  │  LLM API (OpenAI兼容)  │  文件系统  │  命令执行器       │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### 1.3 运行模式

| 模式 | 入口 | 说明 |
|------|------|------|
| CLI | `cmd/agentplus/main.go` → `runCLICommand()` | 命令行交互式Agent |
| GUI | `cmd/agentplus/main.go` → `runGUICommand()` | Wails桌面应用（默认） |

> 来源: `cmd/agentplus/main.go`

---

## 二、技术栈

### 2.1 后端 (Go)

| 依赖 | 版本 | 用途 | 来源 |
|------|------|------|------|
| Go | 1.25.0 | 编程语言 | `go.mod` |
| `wailsapp/wails/v2` | v2.12.0 | 桌面应用框架(Go后端+Web前端) | `go.mod` |
| `gorilla/websocket` | v1.5.3 | WebSocket通信 | `go.mod` |
| `labstack/echo/v4` | v4.13.3 | HTTP框架 | `go.mod` |
| `google/uuid` | v1.6.0 | UUID生成 | `go.mod` |
| `samber/lo` | v1.49.1 | 泛型工具库 | `go.mod` |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML解析 | `go.mod` |
| `go-toast/v2` | v2.0.3 | 系统通知 | `go.mod` |

### 2.2 前端 (TypeScript/React)

| 依赖 | 版本 | 用途 | 来源 |
|------|------|------|------|
| React | ^19.0.0 | UI框架 | `frontend/package.json` |
| Vite | ^6.0.0 | 构建工具 | `frontend/package.json` |
| TypeScript | ^5.0.0 | 类型系统 | `frontend/package.json` |
| TailwindCSS | ^4.0.0 | CSS框架 | `frontend/package.json` |
| react-markdown | ^9.0.0 | Markdown渲染 | `frontend/package.json` |
| highlight.js | ^11.0.0 | 代码高亮 | `frontend/package.json` |

---

## 三、目录结构

```
AgentPlus/
├── cmd/
│   └── agentplus/
│       └── main.go                    # 程序入口（CLI + GUI 双模式）
├── configs/
│   └── config.yaml                    # 应用配置文件
├── internal/                          # 核心业务逻辑（Go内包）
│   ├── agent/                         # Agent核心模块（14个文件）
│   │   ├── core.go                    # Agent结构体、Run()双层主循环
│   │   ├── prompts.go                 # 3种角色提示词 + YAML状态提示
│   │   ├── executor.go                # ToolExecutor工具执行器
│   │   ├── stream.go                  # StreamHandler接口(7方法)
│   │   ├── fork.go                    # ForkManager子代理Fork机制
│   │   ├── reviewer.go                # Reviewer程序逻辑审查器
│   │   ├── verifier.go                # Verifier成果校验器
│   │   ├── selfcorrector.go           # SelfCorrector自我修正器
│   │   ├── history.go                 # HistoryManager消息历史管理
│   │   ├── thinking.go                # ThinkingTagProcessor思考标签处理
│   │   ├── compressor.go              # Compressor上下文压缩器
│   │   ├── feedback.go                # FeedbackInjector反馈注入
│   │   └── logger.go                  # AgentLogger运行日志(JSON Lines)
│   ├── config/                        # 配置加载
│   │   └── loader.go                  # YAML配置 + 环境变量覆盖
│   ├── gui/                           # Wails GUI绑定层（2个文件）
│   │   ├── app.go                     # App(Wails绑定)、对话管理
│   │   └── stream_bridge.go           # StreamBridge流式事件桥接
│   ├── model/                         # LLM模型客户端（3个文件）
│   │   ├── client.go                  # OpenAI兼容API客户端、SSE流式
│   │   ├── config.go                  # Config(Endpoint/APIKey/ModelName)
│   │   └── message.go                 # Message/ToolCall/Delta等类型
│   ├── state/                         # 状态管理（3个文件）
│   │   ├── manager.go                 # StateManager CRUD + 自动保存
│   │   ├── task.go                    # TaskState/TaskInfo数据结构
│   │   └── yaml.go                    # YAML序列化/反序列化/摘要
│   ├── supervisor/                    # 监督模块
│   │   └── monitor.go                 # Supervisor 6类检查 + 4级干预
│   ├── team/                          # 团队与角色管理（3个文件）
│   │   ├── definition.go              # Team/AgentRole/WorkflowStep
│   │   ├── roles.go                   # 6种预定义角色
│   │   └── manager.go                 # RoleManager角色管理器
│   └── tools/                         # 工具系统（4个文件）
│       ├── types.go                   # Tool接口、ToolParameter、Schema
│       ├── registry.go                # ToolRegistry线程安全注册中心
│       ├── filesystem.go              # 文件系统工具(5个)
│       ├── command.go                 # 命令执行工具(2个)
│       └── state_tools.go             # 状态管理工具(4个)
├── frontend/                          # React前端
│   ├── src/
│   │   ├── App.tsx                    # 主应用组件
│   │   ├── components/                # UI组件(9个)
│   │   ├── hooks/                     # 自定义Hooks
│   │   ├── types/                     # TypeScript类型定义
│   │   └── styles/                    # 样式文件
│   ├── index.html
│   └── package.json
├── project/                           # 项目模板（语言脚手架）
│   ├── html-tools/
│   ├── java/
│   ├── javascript/
│   └── kotlin/
├── state/                             # 运行时任务状态存储(YAML)
├── logs/                              # 运行日志
├── go.mod / go.sum                    # Go模块依赖
├── wails.json                         # Wails构建配置
└── frontend_assets.go                 # 前端资源嵌入
```

---

## 四、模块架构

### 4.1 模块依赖关系

```
cmd/agentplus (入口)
    ├── internal/agent (核心)
    │   ├── internal/model (模型通信)
    │   ├── internal/tools (工具系统)
    │   ├── internal/state (状态管理)
    │   └── internal/team (角色管理)
    ├── internal/gui (GUI绑定)
    │   └── internal/agent
    ├── internal/config (配置)
    └── internal/supervisor (监督)
        └── internal/agent
```

### 4.2 Agent核心模块 (`internal/agent/`)

Agent核心模块是系统的心脏，包含14个文件，负责Agent的完整生命周期管理。

#### 核心组件

| 组件 | 文件 | 职责 |
|------|------|------|
| Agent | `core.go` | Agent结构体、Run()双层主循环、callModel()流式收集、executeTools() |
| ToolExecutor | `executor.go` | 工具执行器，支持并行/顺序/超时执行 |
| StreamHandler | `stream.go` | 流式消息处理器接口(7个方法) |
| ForkManager | `fork.go` | 子代理Fork机制，创建/执行/合并子代理 |
| Reviewer | `reviewer.go` | 程序逻辑审查器，6类问题检测 |
| Verifier | `verifier.go` | 成果校验器，文件/语法/HTML/自定义规则校验 |
| SelfCorrector | `selfcorrector.go` | 自我修正器，分离的审查/校验重试计数 |
| HistoryManager | `history.go` | 消息历史管理，token感知截断 |
| ThinkingTagProcessor | `thinking.go` | 跨块`<thinking>`标签处理 |
| Compressor | `compressor.go` | 上下文压缩器，渐进式压缩(轻度→深度) |
| FeedbackInjector | `feedback.go` | 审查结果→用户消息转换 |
| AgentLogger | `logger.go` | JSON Lines格式日志记录 |

> 来源: `internal/agent/` 目录全部文件

#### Agent结构体定义

```go
// 来源: internal/agent/core.go
type Agent struct {
    modelClient  *model.Client       // 模型客户端
    toolRegistry *tools.ToolRegistry // 工具注册中心
    stateManager *state.StateManager // 状态管理器
    executor     *ToolExecutor       // 工具执行器
    history      *HistoryManager     // 消息历史

    reviewer  *Reviewer      // 程序逻辑审查器
    verifier  *Verifier      // 成果校验器
    corrector *SelfCorrector // 自我修正器

    logger *AgentLogger // 运行日志记录器

    maxIterations int              // 最大迭代次数
    systemPrompt  string           // 系统提示词
    taskID        string           // 当前任务ID
    promptType    SystemPromptType // 提示词类型

    streamHandler StreamHandler // 流式消息处理器
    // ... 回调、状态字段
}
```

---

## 五、核心流程

### 5.1 Agent主循环

Agent采用**双层循环**架构：外层循环支持强制校验重试，内层循环是主要的迭代处理。

```
┌──────────────── 外层循环 (maxIterations + 10) ─────────────────┐
│                                                                  │
│  ┌──────────── 内层循环 (maxIterations) ──────────────────┐     │
│  │                                                         │     │
│  │  1. 检查上下文取消                                       │     │
│  │  2. 检查上下文溢出 → Truncate历史                       │     │
│  │  3. callModel() → SSE流式调用                           │     │
│  │     ├─ 处理 reasoning_content (Qwen思考内容)             │     │
│  │     ├─ 处理 <thinking> 标签 (跨块处理)                   │     │
│  │     ├─ 处理正文内容                                      │     │
│  │     └─ 处理 ToolCalls (增量拼接)                         │     │
│  │  4. Reviewer.ReviewOutput() → 审查模型输出              │     │
│  │     ├─ IsBlocked? → SelfCorrector → 注入修正指令        │     │
│  │     └─ ShouldRetryReview? → continue / fail             │     │
│  │  5. 有ToolCalls? → executeTools()                       │     │
│  │     ├─ end_exploration → 结束探索期                      │     │
│  │     ├─ complete_task → verifyTaskCompletion()            │     │
│  │     │   ├─ 校验通过 → break内层循环                      │     │
│  │     │   └─ 校验失败 → SelfCorrector → 注入修正指令      │     │
│  │     └─ fail_task → 直接返回失败                          │     │
│  │  6. 无ToolCalls → isTaskComplete()                      │     │
│  │     └─ 检测完成标志 → verifyTaskCompletion()             │     │
│  │                                                         │     │
│  └─────────────────────────────────────────────────────────┘     │
│                                                                  │
│  7. 强制校验 (即使内层已通过)                                    │
│     ├─ 通过 → 真正完成任务，返回结果                            │
│     └─ 失败 → 重置状态，注入修正指令，继续外层循环               │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

> 来源: `internal/agent/core.go` → `Agent.Run()` 方法

### 5.2 审查-校验-修正闭环

这是AgentPlus最核心的质量保障机制：

```
                    模型输出
                       │
                       ▼
            ┌──────────────────┐
            │   Reviewer       │ ← 程序逻辑审查
            │  6类问题检测      │
            └──────┬───────────┘
                   │
           ┌───────┴───────┐
           │ IsBlocked?    │
           └───┬───────┬───┘
         No    │       │ Yes
               │       ▼
               │  SelfCorrector.AnalyzeFailure()
               │  → GenerateCorrectionInstruction()
               │  → 注入修正到历史 → continue
               │
               ▼
          工具执行
               │
               ▼
         complete_task?
               │
               ▼
            ┌──────────────────┐
            │   Verifier       │ ← 成果校验
            │  文件/语法/HTML   │
            └──────┬───────────┘
                   │
           ┌───────┴───────┐
           │ IsFailed?     │
           └───┬───────┬───┘
         No    │       │ Yes
               │       ▼
               │  SelfCorrector.AnalyzeFailure()
               │  → GenerateCorrectionInstruction()
               │  → ShouldRetryVerify?
               │     ├─ Yes → 注入修正到历史 → continue
               │     └─ No → 任务失败
               │
               ▼
         内层循环break
               │
               ▼
            ┌──────────────────┐
            │  强制校验         │ ← 外层循环的最终保障
            │  (再次Verify)    │
            └──────┬───────────┘
                   │
           ┌───────┴───────┐
           │ 通过?          │
           └───┬───────┬───┘
         Yes   │       │ No
               │       → 重置状态，注入修正，继续外层
               ▼
          任务真正完成
```

> 来源: `internal/agent/core.go` + `reviewer.go` + `verifier.go` + `selfcorrector.go`

### 5.3 Reviewer 审查器 - 6类问题检测

| 问题类型 | 常量 | 严重度 | 检测逻辑 |
|---------|------|--------|---------|
| 方向偏离 | `IssueTypeDirection` | low | 任务目标关键词匹配率 < 20% |
| 无限循环 | `IssueTypeInfiniteLoop` | critical | 窗口内相同操作重复 ≥ 3次 |
| 无效工具调用 | `IssueTypeInvalidToolCall` | high | 工具名为空或参数非JSON |
| 重复失败 | `IssueTypeRepeatedFailure` | high | 连续失败 ≥ 3次 |
| 编造内容 | `IssueTypeFabrication` | high | 声称存在的文件实际不存在 |
| 无进度 | `IssueTypeNoProgress` | medium | 迭代多次但任务步骤未推进 |

> 来源: `internal/agent/reviewer.go`

**探索期机制**: 前3次迭代为探索期，期间放宽`no_progress`检查阈值（×2），支持通过`end_exploration`工具声明探索结束。

### 5.4 Verifier 校验器 - 多层校验

| 校验类型 | 配置项 | 说明 |
|---------|--------|------|
| 文件存在 | `CheckFileExists` | 检查声称创建的文件是否存在 |
| 文件非空 | `CheckFileNonEmpty` | 检查文件大小 > 0 且内容非纯空白 |
| 关键词匹配 | `CheckKeywords` | 支持any/all两种匹配模式 |
| JavaScript语法 | `CheckJSSyntax` | 检查引号闭合、括号匹配、常见错误 |
| HTML结构 | `CheckHTMLStructure` | 检查DOCTYPE、必需标签、标签闭合 |
| 自定义规则 | `EnableCustomRules` | 通过`VerifyRule`接口扩展 |

> 来源: `internal/agent/verifier.go`

### 5.5 SelfCorrector 自我修正器

**分离重试计数**: 审查和校验各自维护独立的重试计数器。

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `MaxReviewRetries` | 3 | 审查最大重试次数 |
| `MaxVerifyRetries` | 5 | 校验最大重试次数 |
| `ExponentialBackoff` | true | 是否启用指数退避 |
| `FailurePatternWindow` | 5 | 失败模式检测窗口 |

**失败模式检测**: `DetectFailurePattern()`方法分析最近的失败记录，识别重复出现的失败模式。

> 来源: `internal/agent/selfcorrector.go`

### 5.6 Fork子代理机制

```
┌─────────────────┐     spawn_agent      ┌─────────────────┐
│   主Agent        │ ──────────────────→  │  子Agent实例     │
│  (Orchestrator)  │                      │  (Worker/等)     │
│                  │                      │                  │
│  共享 StateManager│                      │  共享 TaskID     │
│  独立 History     │                      │  独立 History    │
│                  │  ←────────────────  │                  │
│  Join合并结果     │   complete_as_agent  │  提交总结        │
└─────────────────┘                      └─────────────────┘
```

**关键流程**:
1. `SpawnAgentTool.Execute()` → `ForkManager.Fork()` 创建子Agent实例
2. 子Agent共享父TaskID和StateManager，但拥有独立的History
3. 子Agent通过`complete_as_agent`工具提交执行总结
4. `ForkManager.Join()` 将结果合并回主Agent，注入合并提示到主Agent历史
5. 更新StateManager中的Agent记录（身份切换回Orchestrator）

> 来源: `internal/agent/fork.go`

### 5.7 上下文压缩

Compressor采用**渐进式压缩**策略：

```
触发条件: 上下文使用率 >= TriggerThreshold (默认80%)

┌──────────────┐    仍超限     ┌──────────────┐
│  轻度压缩     │ ──────────→ │  深度压缩     │
│  保留20条消息  │             │  保留10条消息  │
│  保留3次工具   │             │  保留2条消息   │
│  简要摘要     │             │  详细摘要     │
└──────────────┘              └──────────────┘
```

**压缩内容保留策略**:
- 所有系统消息（含指令）
- 上下文摘要（从TaskState和消息历史提取）
- 最近N次工具调用及结果
- 最近的对话消息

> 来源: `internal/agent/compressor.go`

---

## 六、数据模型

### 6.1 消息类型 (`internal/model/message.go`)

```go
// OpenAI兼容的消息类型
type Message struct {
    Role      string       // "system" | "user" | "assistant" | "tool"
    Content   string       // 消息内容
    ToolCalls []ToolCall   // 工具调用列表(assistant消息)
    ToolCallID string      // 工具调用ID(tool消息)
    Name      string       // 工具名称(tool消息)
}

type ToolCall struct {
    ID       string        // 工具调用唯一ID
    Type     string        // 类型，固定"function"
    Function FunctionCall  // 函数调用信息
}

type FunctionCall struct {
    Name      string       // 函数名
    Arguments string       // JSON格式参数
}

// SSE流式响应
type StreamResponse struct {
    Choices []Choice      // 响应选项
    Error   error         // 错误信息
    Done    bool          // 是否完成
}

type Delta struct {
    Content          string       // 正文内容
    ReasoningContent string       // 思考内容(Qwen3.5)
    ToolCalls        []ToolCallDelta // 工具调用增量
}
```

> 来源: `internal/model/message.go`

### 6.2 任务状态 (`internal/state/task.go`)

```yaml
# 任务状态YAML结构
task:
  id: "task-1234567890"          # 任务唯一标识
  goal: "任务目标描述"             # 任务目标
  status: "in_progress"          # pending | in_progress | completed | failed
  created_at: "2026-04-11T10:00:00Z"
  updated_at: "2026-04-11T10:30:00Z"

progress:
  current_phase: "实现阶段"       # 当前阶段
  completed_steps:               # 已完成步骤
    - "分析需求"
    - "设计架构"
  pending_steps:                 # 待完成步骤
    - "编写测试"

context:
  decisions:                     # 关键决策记录
    - "选择React作为前端框架"
  constraints:                   # 约束条件
    - "必须兼容IE11"
  files:                         # 相关文件
    - path: "src/main.go"
      description: "主入口文件"
      status: "created"          # created | modified | deleted

agents:
  active: "Orchestrator"         # 当前活动Agent
  history:                       # Agent执行历史
    - role: "Worker"
      summary: "完成了API开发"
      duration: "5m30s"
```

> 来源: `internal/state/task.go` + `yaml.go`

### 6.3 工具系统数据结构 (`internal/tools/types.go`)

```go
type Tool interface {
    Name() string                                    // 工具名称
    Description() string                             // 工具描述
    Parameters() map[string]interface{}              // JSON Schema参数
    Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
}

type ToolResult struct {
    Success bool                    // 是否成功
    Data    map[string]interface{}  // 结果数据
    Error   string                  // 错误信息
}

type ToolParameter struct {
    Type        string   // 参数类型
    Description string   // 参数描述
    Required    bool     // 是否必需
    Enum        []string // 枚举值
    Default     interface{} // 默认值
}
```

> 来源: `internal/tools/types.go`

---

## 七、工具系统

### 7.1 工具注册中心

`ToolRegistry`提供线程安全的工具注册、查询和执行机制，维护全局默认实例。

> 来源: `internal/tools/registry.go`

### 7.2 工具清单

| 工具名 | 类别 | 文件 | 参数 | 说明 |
|--------|------|------|------|------|
| `read_file` | 文件系统 | `filesystem.go` | `file_path` | 读取文件内容 |
| `write_file` | 文件系统 | `filesystem.go` | `file_path`, `content` | 写入文件 |
| `list_directory` | 文件系统 | `filesystem.go` | `path` | 列出目录内容 |
| `delete_file` | 文件系统 | `filesystem.go` | `file_path` | 删除文件 |
| `create_directory` | 文件系统 | `filesystem.go` | `path` | 创建目录 |
| `execute_command` | 命令执行 | `command.go` | `command`, `args` | 执行命令 |
| `shell_execute` | 命令执行 | `command.go` | `cmd` | 跨平台Shell执行 |
| `complete_task` | 状态管理 | `state_tools.go` | `summary` | 标记任务完成 |
| `fail_task` | 状态管理 | `state_tools.go` | `reason` | 标记任务失败 |
| `update_state` | 状态管理 | `state_tools.go` | `phase`, `decision`, `completed_step` | 更新任务状态 |
| `end_exploration` | 状态管理 | `state_tools.go` | - | 声明探索期结束 |

### 7.3 Fork相关工具

| 工具名 | 参数 | 说明 |
|--------|------|------|
| `spawn_agent` | `role` (Worker/Reviewer/Specialist), `task` | 创建子代理执行特定任务 |
| `complete_as_agent` | `summary` | 子代理提交执行总结 |

> 来源: `internal/tools/` 全部文件 + `internal/agent/fork.go`

---

## 八、角色与团队系统

### 8.1 预定义角色

| 角色 | 提示词类型 | 职责 | 来源 |
|------|-----------|------|------|
| orchestrator | `PromptTypeOrchestrator` | 主控Agent，协调和执行任务，维护状态 | `prompts.go` |
| worker | `PromptTypeWorker` | 执行者，专注于高质量完成分配的具体任务 | `prompts.go` |
| reviewer | `PromptTypeReviewer` | 审查者，审查工作成果，确保质量和正确性 | `prompts.go` |
| architect | - | 架构师（待确认具体实现） | `roles.go` |
| developer | - | 开发者（待确认具体实现） | `roles.go` |
| tester | - | 测试者（待确认具体实现） | `roles.go` |

### 8.2 系统提示词设计

**Orchestrator角色**采用高效执行模式：
- 不奉承、不评价、不出报告
- 使用`<thinking>`标签输出思考过程
- 优先使用工具而非空谈计划
- 通过`update_state`工具维护YAML格式任务状态

> 来源: `internal/agent/prompts.go` + `internal/team/roles.go`

### 8.3 工作流

默认团队(`DefaultTeam`)定义了6步工作流（待确认具体步骤定义）。

> 来源: `internal/team/definition.go`

---

## 九、前后端交互

### 9.1 架构桥接

```
┌──────────────────────────────────────────────────────┐
│                    Go 后端                            │
│                                                       │
│  App (Wails绑定结构体)                                │
│    ├── SendMessage(content) → go a.Run()             │
│    ├── GetConversations() []ConversationData          │
│    └── StreamHandler 接口                             │
│              │                                        │
│              ▼                                        │
│  StreamBridge (实现 StreamHandler)                    │
│    ├── OnThinking → runtime.EventsEmit("stream:thinking")    │
│    ├── OnContent → runtime.EventsEmit("stream:content")       │
│    ├── OnToolCall → runtime.EventsEmit("stream:toolCall")     │
│    ├── OnToolResult → runtime.EventsEmit("stream:toolResult") │
│    ├── OnComplete → runtime.EventsEmit("stream:complete")     │
│    ├── OnError → runtime.EventsEmit("stream:error")           │
│    └── OnTaskDone → runtime.EventsEmit("stream:taskDone")     │
│                                                       │
└──────────────────────┬───────────────────────────────┘
                       │ Wails事件系统
                       ▼
┌──────────────────────────────────────────────────────┐
│                 React 前端                            │
│                                                       │
│  useStreamEvents Hook                                 │
│    ├── EventsOn("stream:thinking") → setThinking     │
│    ├── EventsOn("stream:content") → appendContent    │
│    ├── EventsOn("stream:toolCall") → setToolCalls    │
│    ├── EventsOn("stream:toolResult") → setToolResult │
│    ├── EventsOn("stream:complete") → setTokenStats   │
│    ├── EventsOn("stream:error") → setError           │
│    └── EventsOn("stream:taskDone") → setRunning(false)│
│                                                       │
│  useConversation Hook                                 │
│    ├── 管理对话列表                                    │
│    ├── sendMessage() → App.SendMessage()              │
│    └── 管理消息和工具调用状态                          │
│                                                       │
│  UI组件                                               │
│    ├── MessageList → 消息渲染                         │
│    ├── MessageItem → 单条消息                         │
│    ├── ToolCallBlock → 工具调用展示                   │
│    ├── ToolResultBlock → 工具结果展示                 │
│    ├── ThinkingBlock → 思考过程展示                   │
│    ├── InputArea → 输入框                             │
│    ├── Sidebar → 侧边栏对话列表                      │
│    └── Toolbar → 工具栏                               │
│                                                       │
└──────────────────────────────────────────────────────┘
```

> 来源: `internal/gui/app.go` + `stream_bridge.go` + `frontend/src/hooks/`

### 9.2 事件类型

| 事件名 | 数据 | 方向 | 说明 |
|--------|------|------|------|
| `stream:thinking` | `{chunk: string}` | Go→前端 | 思考内容块 |
| `stream:content` | `{chunk: string}` | Go→前端 | 正文内容块 |
| `stream:toolCall` | `{ToolCallInfo, isComplete}` | Go→前端 | 工具调用 |
| `stream:toolResult` | `{ToolCallInfo}` | Go→前端 | 工具执行结果 |
| `stream:complete` | `{usage: int}` | Go→前端 | 单次推理完成 |
| `stream:error` | `{error: string}` | Go→前端 | 错误 |
| `stream:taskDone` | - | Go→前端 | 整个任务完成 |

### 9.3 前端数据类型

```typescript
// 来源: frontend/src/types/index.ts
interface Message {
  id: string;
  role: "user" | "assistant" | "system";
  content: string;
  thinking?: string;
  toolCalls?: ToolCall[];
  timestamp: number;
}

interface ToolCall {
  id: string;
  name: string;
  arguments: string;
  result?: string;
  error?: string;
  isComplete: boolean;
}

interface ConversationData {
  id: string;
  title: string;
  messages: Message[];
  createdAt: number;
  updatedAt: number;
}

interface TokenStats {
  usage: number;
}
```

---

## 十、配置系统

### 10.1 配置加载

配置通过YAML文件加载，支持环境变量覆盖。

> 来源: `internal/config/loader.go`

### 10.2 配置项

```yaml
# 来源: configs/config.yaml
model:
  endpoint: "http://127.0.0.1:11453/v1/"    # LLM API端点
  api_key: "no-key"                          # API密钥
  model_name: "Qwen3.5-27B"                 # 模型名称
  context_size: 200000                       # 上下文窗口大小

agent:
  max_iterations: 100                        # 最大迭代次数
  temperature: 0.7                           # 温度参数

tools:
  work_dir: "."                              # 工作目录
  allow_commands:                            # 允许执行的命令白名单
    - "go"
    - "git"
    - "ls"
    - "cat"
    - "mkdir"
    - "rm"
```

### 10.3 Wails配置

```json
// 来源: wails.json
{
  "name": "AgentPlus",
  "outputfilename": "agentplus",
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "frontend:dev:watcher": "npm run dev",
  "frontend:dev:serverUrl": "auto",
  "author": {
    "name": "Attect"
  }
}
```

---

## 十一、StreamHandler 接口

`StreamHandler`是前后端交互的核心接口，定义了7个方法：

```go
// 来源: internal/agent/stream.go
type StreamHandler interface {
    OnThinking(chunk string)              // 思考内容块
    OnContent(chunk string)               // 正文内容块
    OnToolCall(call ToolCallInfo, isComplete bool)  // 工具调用
    OnToolResult(result ToolCallInfo)     // 工具执行结果
    OnComplete(usage int)                 // 单次推理完成
    OnError(err error)                    // 错误
    OnTaskDone()                          // 整个任务完成
}
```

**实现链**: `StreamHandlerFunc`(建造者模式) → `streamHandlerFuncImpl` → `StreamBridge`(GUI桥接) → Wails事件系统 → React Hooks

---

## 十二、架构决策记录

### ADR-001: 双层循环设计

**决策**: 采用外层+内层双层循环架构
**原因**: 内层循环处理正常迭代，外层循环提供"强制校验"保障层。即使内层循环通过了校验，外层仍会再次验证，确保任务真正完成。
**来源**: `internal/agent/core.go` → `Agent.Run()`

### ADR-002: 分离的审查/校验重试计数

**决策**: 审查(Review)和校验(Verify)各自维护独立的重试计数器
**原因**: 审查阻断通常是因为方向偏差或循环行为，校验失败通常是因为文件缺失或内容不完整，两者的容错需求不同。审查更严格（3次），校验更宽松（5次）。
**来源**: `internal/agent/selfcorrector.go`

### ADR-003: 探索期机制

**决策**: Agent启动初期设置探索期（前3次迭代），期间放宽进度检查
**原因**: Agent需要先了解环境和文件结构才能开始正式工作，探索期的宽松检查避免了误报"无进度"。支持通过`end_exploration`工具主动声明探索结束。
**来源**: `internal/agent/reviewer.go`

### ADR-004: 渐进式上下文压缩

**决策**: 先轻度压缩，仍超限再深度压缩
**原因**: 轻度压缩保留更多上下文，对Agent执行影响更小；深度压缩只保留最关键信息，确保不超限。
**来源**: `internal/agent/compressor.go`

### ADR-005: Fork子代理共享状态

**决策**: 子代理共享父TaskID和StateManager，但拥有独立History
**原因**: 子代理需要看到和更新全局任务状态，但消息历史是私有的，避免上下文污染。
**来源**: `internal/agent/fork.go`

### ADR-006: 流式处理的双模式思考内容

**决策**: 同时支持`reasoning_content`字段和`<thinking>`标签两种思考内容模式
**原因**: 不同模型的思考内容输出方式不同。Qwen3.5使用`reasoning_content`字段，其他模型可能使用`<thinking>`标签。`ThinkingTagProcessor`处理跨块的标签分割问题。
**来源**: `internal/agent/thinking.go` + `core.go` → `callModel()`

---

## 十三、扩展点

### 13.1 自定义校验规则

通过`VerifyRule`接口可注册自定义校验规则：

```go
// 来源: internal/agent/verifier.go
type VerifyRule interface {
    Name() string
    Description() string
    Execute(ctx *VerifyContext) error
}

// 注册方式
verifier.AddRule(myRule)
```

### 13.2 自定义工具

通过`Tool`接口可注册自定义工具：

```go
// 来源: internal/tools/types.go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}
    Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
}

// 注册方式
registry.RegisterTool(myTool)
```

### 13.3 自定义StreamHandler

通过`StreamHandler`接口可实现自定义的流式处理：

```go
// 来源: internal/agent/stream.go
type StreamHandler interface {
    OnThinking(chunk string)
    OnContent(chunk string)
    OnToolCall(call ToolCallInfo, isComplete bool)
    OnToolResult(result ToolCallInfo)
    OnComplete(usage int)
    OnError(err error)
    OnTaskDone()
}
```

### 13.4 项目模板扩展

`project/`目录用于存放多语言项目模板，Agent创建新项目时可作为脚手架使用。当前支持html-tools、java、javascript、kotlin。

---

## 十四、待确认项

| 编号 | 项目 | 说明 |
|------|------|------|
| 1 | supervisor模块 | `internal/supervisor/monitor.go`中的6类检查和4级干预具体实现待深入分析 |
| 2 | team工作流 | `DefaultTeam`的6步工作流具体步骤定义待确认 |
| 3 | 反馈注入 | `feedback.go`中`FeedbackInjector`的具体注入策略待确认 |
| 4 | 项目模板使用 | `project/`目录下模板的具体使用流程待确认 |
| 5 | executor并行 | `executor.go`中工具并行执行的具体策略(并行/顺序判断条件)待确认 |
| 6 | 测试覆盖 | 项目当前无测试文件，测试覆盖率待评估 |
| 7 | 命令白名单 | `tools.allow_commands`的验证逻辑待确认 |
