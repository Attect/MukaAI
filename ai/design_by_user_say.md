# AgentPlus 需求规格说明书

> 版本: v1.0.0
> 生成时间: 2026-04-12
> 生成者: requirement-analyst（需求分析师AGENT）
> 分析方法: 源码逆向分析 + 架构文档交叉验证
> 状态: 初版

---

## 一、产品概述

### 1.1 产品定位

AgentPlus 是一款基于大语言模型（LLM）的 AI Agent 桌面应用程序，支持 CLI 命令行和 GUI 图形界面双模式运行。系统通过 OpenAI 兼容的 Chat Completion API 与 LLM 模型通信，具备完整的任务规划、工具调用、审查校验、自我修正和子代理 Fork 能力。

> 来源: `ai/dev/project.md` §1.1 + `cmd/agentplus/main.go`

### 1.2 目标用户

- 开发者：需要 AI Agent 辅助完成编程、文件操作、命令执行等任务的软件工程师
- 项目管理者：需要自动化执行重复性任务的技术管理人员
- AI 应用研究者：需要研究 Agent 架构、工具调用、自我修正机制的研究人员

### 1.3 核心价值

1. **自动化任务执行**：通过自然语言描述任务目标，Agent 自动分析、规划并执行
2. **质量保障闭环**：内置审查→校验→修正的完整质量保障机制，确保任务执行质量
3. **双模式交互**：CLI 适合脚本化场景，GUI 提供可视化交互体验
4. **可扩展架构**：支持自定义工具、自定义校验规则、自定义角色，灵活扩展

---

## 二、功能模块清单

### 2.1 模块总览

| 模块 | 功能数量 | 来源目录 | 实现状态 |
|------|---------|---------|---------|
| Agent核心 | 8 | `internal/agent/` | 已完成 |
| 工具系统 | 11 | `internal/tools/` | 已完成 |
| 审查校验 | 3 | `internal/agent/` | 已完成 |
| GUI界面 | 9 | `internal/gui/` + `frontend/src/components/` | 已完成 |
| CLI命令行 | 5 | `cmd/agentplus/main.go` | 已完成 |
| 团队角色 | 6 | `internal/team/` | 已完成 |
| 状态管理 | 4 | `internal/state/` | 已完成 |
| 模型通信 | 3 | `internal/model/` | 已完成 |
| 配置系统 | 2 | `internal/config/` | 已完成 |

---

## 三、功能详细描述

### 3.1 Agent核心功能模块

#### 3.1.1 双层主循环

- **功能目标**: Agent 执行任务的核心调度循环，外层循环支持强制校验重试，内层循环是主要的迭代处理
- **功能描述**:
  - 外层循环最大迭代数为 `maxIterations + 10`，提供强制校验保障
  - 内层循环最大迭代数为 `maxIterations`（默认50），处理模型调用、审查、工具执行、校验
  - 每次迭代检查上下文取消、上下文溢出、模型调用、审查、工具执行、校验
  - 任务完成后外层循环执行强制校验，确保任务真正完成
- **输入定义**:
  - `ctx context.Context`: 上下文，支持取消操作
  - `taskGoal string`: 任务目标描述
- **输出定义**:
  - `*RunResult`: 运行结果（任务ID、状态、迭代次数、时长、最终响应、错误信息）
  - `error`: 错误信息
- **来源文件**: `internal/agent/core.go` → `Agent.Run()`
- **实现状态**: 已完成

#### 3.1.2 流式模型调用

- **功能目标**: 通过 SSE（Server-Sent Events）协议流式调用 LLM 模型，实时处理思考内容和工具调用
- **功能描述**:
  - 通过 `StreamChatCompletion` 发起 SSE 流式请求
  - 处理 `reasoning_content` 字段（Qwen3.5 思考内容）
  - 处理 `<thinking>` 标签（通用思考标签模式）
  - 流式拼接 ToolCalls 参数
  - 估算 Token 用量（每4字符≈1 Token）
- **输入定义**: 消息历史 + 工具 Schema
- **输出定义**: 内容文本 + ToolCall 列表 + Token 用量
- **来源文件**: `internal/agent/core.go` → `Agent.callModel()` + `internal/model/client.go`
- **实现状态**: 已完成

#### 3.1.3 思考标签处理

- **功能目标**: 在流式输出中识别和处理 `<thinking>` 标签，实现跨块的标签分割处理
- **功能描述**:
  - 状态机模式跟踪当前是否处于 `<thinking>` 标签内
  - 处理标签跨SSE数据块分割的问题
  - 分离思考内容和正文内容，分别回调
  - 支持不完整标签的缓冲和拼接
- **输入定义**: 流式内容块字符串
- **输出定义**: 思考内容字符串 + 正文内容字符串
- **来源文件**: `internal/agent/thinking.go`
- **实现状态**: 已完成

#### 3.1.4 子代理Fork机制

- **功能目标**: 主Agent可以创建子代理执行特定角色任务，完成后合并结果
- **功能描述**:
  - `ForkManager` 管理子代理的创建、执行和合并
  - 子代理共享父 TaskID 和 StateManager，但拥有独立的 History
  - 子代理通过 `complete_as_agent` 工具提交执行总结
  - `Join` 方法将子代理结果合并回主Agent
  - 线程安全的活动子代理管理
- **输入定义**: 角色（Worker/Reviewer/Specialist）+ 任务描述
- **输出定义**: ForkID + 执行总结 + 状态 + 时长 + 迭代次数
- **来源文件**: `internal/agent/fork.go`
- **实现状态**: 已完成

#### 3.1.5 消息历史管理

- **功能目标**: 管理 Agent 的消息历史，支持上下文溢出时截断
- **功能描述**:
  - 维护消息列表（system/user/assistant/tool 四种角色）
  - 支持上下文溢出时按 Token 预算截断
  - 线程安全的消息操作
- **输入定义**: Message 对象
- **输出定义**: 消息列表
- **来源文件**: `internal/agent/history.go`（通过 `core.go` 引用）
- **实现状态**: 已完成

#### 3.1.6 上下文压缩

- **功能目标**: 在上下文过长时压缩消息历史，保留关键信息
- **功能描述**:
  - 触发阈值默认80%上下文使用率
  - 渐进式压缩策略：先轻度压缩（保留20条消息、3次工具调用），再深度压缩（保留10条消息、2条摘要）
  - 保留所有系统消息和上下文摘要
  - 从任务状态和消息历史提取关键决策、最近操作
  - 压缩统计（原始/压缩后消息数、Token数、压缩比率）
- **输入定义**: 消息历史 + 任务状态
- **输出定义**: 压缩后消息列表 + 摘要 + 统计信息
- **来源文件**: `internal/agent/compressor.go`
- **实现状态**: 已完成

#### 3.1.7 异步消息发送

- **功能目标**: 支持异步发送用户消息并启动推理（GUI模式使用）
- **功能描述**:
  - `SendMessage` 方法异步启动 `Agent.Run`
  - 推理完成后通过 `OnTaskDone` 通知 GUI
  - 防止重复运行（running 状态检查）
- **输入定义**: 用户消息内容
- **输出定义**: 错误信息（同步返回）/ 推理结果（异步通过 StreamHandler）
- **来源文件**: `internal/agent/core.go` → `Agent.SendMessage()`
- **实现状态**: 已完成

#### 3.1.8 运行日志记录

- **功能目标**: 以 JSON Lines 格式记录 Agent 运行过程中的所有关键事件
- **功能描述**:
  - 日志类型：message、tool_call、tool_result、review、verification、error、iteration、task_start、task_end、correction、session_start、session_end
  - 每条日志包含时间戳、类型、内容和元数据
  - 线程安全的文件写入，每条日志写入后立即 sync
  - 支持任务ID关联
- **输入定义**: 日志条目
- **输出定义**: JSON Lines 格式日志文件
- **来源文件**: `internal/agent/logger.go`
- **实现状态**: 已完成

---

### 3.2 工具系统功能模块

#### 3.2.1 文件读取工具 (`read_file`)

- **功能目标**: 读取指定路径的文件内容
- **输入定义**: `path`（绝对路径，必需）, `encoding`（默认utf-8）
- **输出定义**: 文件路径 + 内容 + 文件大小
- **来源文件**: `internal/tools/filesystem.go` → `ReadFileTool`
- **实现状态**: 已完成

#### 3.2.2 文件写入工具 (`write_file`)

- **功能目标**: 将内容写入指定路径的文件，自动创建父目录
- **输入定义**: `path`（绝对路径，必需）, `content`（写入内容，必需）
- **输出定义**: 文件路径 + 写入字节数
- **来源文件**: `internal/tools/filesystem.go` → `WriteFileTool`
- **实现状态**: 已完成

#### 3.2.3 目录列表工具 (`list_directory`)

- **功能目标**: 列出指定目录的内容，支持递归
- **输入定义**: `path`（绝对路径，必需）, `recursive`（是否递归，默认false）
- **输出定义**: 目录路径 + 文件列表（名称、路径、是否目录、大小、权限、修改时间）+ 条目数量
- **来源文件**: `internal/tools/filesystem.go` → `ListDirectoryTool`
- **实现状态**: 已完成

#### 3.2.4 文件删除工具 (`delete_file`)

- **功能目标**: 删除指定路径的文件或目录
- **输入定义**: `path`（绝对路径，必需）, `recursive`（是否递归删除目录，默认false）
- **输出定义**: 路径 + 是否为目录 + 是否递归
- **来源文件**: `internal/tools/filesystem.go` → `DeleteFileTool`
- **实现状态**: 已完成

#### 3.2.5 目录创建工具 (`create_directory`)

- **功能目标**: 创建指定路径的目录（支持多级创建，类似 mkdir -p）
- **输入定义**: `path`（绝对路径，必需）
- **输出定义**: 路径 + 是否新建
- **来源文件**: `internal/tools/filesystem.go` → `CreateDirectoryTool`
- **实现状态**: 已完成

#### 3.2.6 命令执行工具 (`execute_command`)

- **功能目标**: 执行系统命令并返回结果，支持跨平台
- **输入定义**: `command`（命令，必需）, `args`（参数列表）, `working_dir`（工作目录）, `timeout`（超时秒数，1-600，默认60）, `env`（环境变量）
- **输出定义**: 命令 + 标准输出 + 标准错误 + 退出码 + 是否成功 + 执行时长 + 是否超时
- **边界条件**: Windows 使用 `cmd /c`，Unix 使用 `sh -c`；空命令返回错误
- **来源文件**: `internal/tools/command.go` → `ExecuteCommandTool`
- **实现状态**: 已完成

#### 3.2.7 Shell执行工具 (`shell_execute`)

- **功能目标**: 直接执行 Shell 命令字符串，支持管道、重定向等特性
- **输入定义**: `command`（Shell命令，必需）, `working_dir`, `timeout`
- **输出定义**: 同 `execute_command`
- **来源文件**: `internal/tools/command.go` → `ShellExecuteTool`
- **实现状态**: 已完成

#### 3.2.8 任务完成工具 (`complete_task`)

- **功能目标**: 标记任务已完成，可选执行校验
- **输入定义**: `summary`（完成总结，必需）
- **输出定义**: 状态 + 总结 + 校验结果（如有）
- **来源文件**: `internal/tools/state_tools.go` → `completeTaskTool`
- **实现状态**: 已完成

#### 3.2.9 任务失败工具 (`fail_task`)

- **功能目标**: 标记任务失败
- **输入定义**: `reason`（失败原因，必需）
- **输出定义**: 状态 + 原因
- **来源文件**: `internal/tools/state_tools.go` → `failTaskTool`
- **实现状态**: 已完成

#### 3.2.10 状态更新工具 (`update_state`)

- **功能目标**: 更新任务状态，记录完成的步骤或添加决策
- **输入定义**: `completed_step`（已完成步骤）, `decision`（决策）, `current_phase`（当前阶段）
- **输出定义**: 更新项列表
- **来源文件**: `internal/tools/state_tools.go` → `updateStateTool`
- **实现状态**: 已完成

#### 3.2.11 探索结束工具 (`end_exploration`)

- **功能目标**: Agent 主动声明探索阶段结束
- **输入定义**: `summary`（探索阶段总结，必需）
- **输出定义**: 确认消息 + 总结
- **来源文件**: `internal/tools/state_tools.go` → `endExplorationTool`
- **实现状态**: 已完成

---

### 3.3 审查校验功能模块

#### 3.3.1 程序逻辑审查器（Reviewer）

- **功能目标**: 审查 Agent 的输出和行为，检测6类问题
- **功能描述**:
  - **方向偏离检测**: 任务目标关键词匹配率 < 20% 时报告（仅在无工具调用且输出过短时检查）
  - **无限循环检测**: 窗口内相同操作重复 ≥ 3次
  - **无效工具调用检测**: 工具名为空或参数非 JSON
  - **重复失败检测**: 连续失败 ≥ 3次
  - **编造内容检测**: 声称存在的文件实际不存在
  - **无进度检测**: 迭代多次但任务步骤未推进
- **探索期机制**: 前3次迭代为探索期，放宽进度检查阈值（×2），支持通过 `end_exploration` 工具主动结束
- **审查结果**: pass/warning/block 三种状态
- **来源文件**: `internal/agent/reviewer.go`
- **实现状态**: 已完成

#### 3.3.2 成果校验器（Verifier）

- **功能目标**: 校验 Agent 的工作成果，确保任务完成质量
- **功能描述**:
  - **文件存在检查**: 检查声称创建的文件是否存在
  - **文件非空检查**: 检查文件大小 > 0 且内容非纯空白
  - **关键词匹配**: 支持 any/all 两种匹配模式
  - **JavaScript 语法检查**: 检查引号闭合、括号匹配、模板字符串对象字面量、file 协议兼容性
  - **HTML 结构检查**: 检查 DOCTYPE、必需标签（html/head/body）、标签闭合
  - **自定义规则**: 通过 `VerifyRule` 接口扩展
- **校验结果**: pass/warning/fail 三种状态
- **来源文件**: `internal/agent/verifier.go`
- **实现状态**: 已完成

#### 3.3.3 自我修正器（SelfCorrector）

- **功能目标**: 分析失败原因、生成修正指令、管理重试逻辑
- **功能描述**:
  - **分离重试计数**: 审查（最大3次）和校验（最大5次）各自独立计数
  - **失败模式检测**: 分析最近失败记录，识别重复出现的问题（窗口大小5）
  - **修正指令生成**: 包含失败原因、优先级、建议、行动指导
  - **指数退避**: 重试延迟支持指数退避，最大30秒
  - **统计信息**: 失败次数、修正次数、修正成功率
- **来源文件**: `internal/agent/selfcorrector.go`
- **实现状态**: 已完成

---

### 3.4 GUI界面功能模块

#### 3.4.1 主应用框架（App.tsx）

- **功能目标**: React 前端主应用，协调所有 UI 组件和后端交互
- **功能描述**:
  - Wails 运行时初始化检测
  - 对话数据管理（消息列表、流式状态）
  - Token 统计显示
  - 工作目录管理
  - 命令处理（/cd, /clear, /help, /exit）
  - 事件监听（conversation:updated, tokenstats:updated, stream:done, stream:error, workdir:changed）
- **来源文件**: `frontend/src/App.tsx`
- **实现状态**: 已完成

#### 3.4.2 消息列表组件（MessageList）

- **功能目标**: 显示对话消息列表，自动滚动到底部
- **来源文件**: `frontend/src/components/MessageList.tsx`
- **实现状态**: 已完成

#### 3.4.3 消息项组件（MessageItem）

- **功能目标**: 渲染单条消息，区分用户/助手角色
- **功能描述**:
  - 用户消息：简单文本展示
  - 助手消息：包含思考块、工具调用块、Markdown正文、Token用量
  - 流式状态展示（闪烁光标）
  - Markdown 渲染（react-markdown + remark-gfm + rehype-highlight）
- **来源文件**: `frontend/src/components/MessageItem.tsx`
- **实现状态**: 已完成

#### 3.4.4 思考内容组件（ThinkingBlock）

- **功能目标**: 可折叠展示模型的思考过程内容
- **来源文件**: `frontend/src/components/ThinkingBlock.tsx`
- **实现状态**: 已完成

#### 3.4.5 工具调用组件（ToolCallBlock）

- **功能目标**: 可折叠展示工具调用详情，包含参数和结果
- **功能描述**:
  - 展示工具名称和完成状态
  - 可展开查看参数（JSON 格式化）和结果
  - 错误结果红色高亮，成功结果绿色高亮
  - 流式状态展示（参数拼接中）
- **来源文件**: `frontend/src/components/ToolCallBlock.tsx`
- **实现状态**: 已完成

#### 3.4.6 输入区域组件（InputArea）

- **功能目标**: 用户输入消息或命令，支持快捷键
- **功能描述**:
  - 多行文本输入框
  - Enter 发送，Shift+Enter 换行
  - 斜杠命令自动补全（/cd, /clear, /save, /help, /exit）
  - 推理中禁用输入
  - 字符计数显示
- **来源文件**: `frontend/src/components/InputArea.tsx`
- **实现状态**: 已完成

#### 3.4.7 工具栏组件（Toolbar）

- **功能目标**: 显示工具状态信息和操作按钮
- **功能描述**:
  - 工作目录显示
  - Token 使用量和推理次数统计
  - 推理中显示"打断"按钮
  - 清空对话按钮
  - 侧边栏切换按钮
- **来源文件**: `frontend/src/components/Toolbar.tsx`
- **实现状态**: 已完成

#### 3.4.8 侧边栏组件（Sidebar）

- **功能目标**: 显示对话列表，支持新建和切换对话
- **来源文件**: `frontend/src/components/Sidebar.tsx`
- **实现状态**: 已完成

#### 3.4.9 流式桥接（StreamBridge）

- **功能目标**: 将 Agent 的 StreamHandler 接口桥接到 Wails 前端事件系统
- **功能描述**:
  - 实现 7 个 StreamHandler 方法
  - 将事件转换为 Wails `runtime.EventsEmit` 调用
  - 同步更新 App 中的对话状态（消息内容、思考内容、工具调用）
  - 事件类型：stream:thinking, stream:content, stream:toolcall, stream:toolresult, stream:complete, stream:error, stream:done
- **来源文件**: `internal/gui/stream_bridge.go`
- **实现状态**: 已完成

---

### 3.5 CLI命令行功能模块

#### 3.5.1 CLI模式入口

- **功能目标**: 命令行交互式 Agent 运行模式
- **功能描述**:
  - 解析命令行参数（配置路径、任务ID、工作目录、详细输出、最大迭代次数）
  - 加载配置并初始化组件（模型客户端、工具注册中心、状态管理器）
  - 交互式多行输入（空行提交）
  - 支持内建命令（/help, /quit, /status, /clear）
  - 信号处理（Ctrl+C 优雅退出）
  - 执行结果格式化输出
- **来源文件**: `cmd/agentplus/main.go` → `runCLICommand()`
- **实现状态**: 已完成

#### 3.5.2 GUI模式入口

- **功能目标**: Wails 桌面 GUI 应用启动
- **功能描述**:
  - 解析 GUI 子命令参数（配置路径、工作目录）
  - 初始化所有后端组件
  - 创建 Agent、App、StreamBridge 实例
  - 启动 Wails 应用（窗口 1024×768，最小 640×480）
- **来源文件**: `cmd/agentplus/main.go` → `runGUICommand()`
- **实现状态**: 已完成

#### 3.5.3 版本和帮助信息

- **功能目标**: 显示版本信息和使用说明
- **来源文件**: `cmd/agentplus/main.go` → `printUsage()`, `printInteractiveHelp()`
- **实现状态**: 已完成

#### 3.5.4 命令行参数解析

- **功能目标**: 解析 CLI 和 GUI 模式的命令行参数
- **参数列表**:
  - CLI: `-c/config`, `-t/task`, `-w/workdir`, `-v/verbose`, `--no-supervisor`, `--max-iterations`
  - GUI: `-c/config`, `-w/workdir`
- **来源文件**: `cmd/agentplus/main.go` → `parseFlags()`, `parseGUIFlags()`
- **实现状态**: 已完成

#### 3.5.5 模式路由

- **功能目标**: 根据命令行参数路由到 CLI 或 GUI 模式
- **路由规则**: `gui` 子命令→GUI模式；`help`/`version`→信息输出；默认→CLI模式
- **来源文件**: `cmd/agentplus/main.go` → `main()`
- **实现状态**: 已完成

---

### 3.6 团队角色功能模块

#### 3.6.1 Orchestrator（主控Agent）

- **功能目标**: 协调任务流程，维护全局状态
- **权限**: 所有工具可用，可 Fork 子代理
- **最大迭代**: 50
- **优先级**: 100（最高）
- **来源文件**: `internal/team/roles.go` → `NewOrchestratorRole()`
- **实现状态**: 已完成

#### 3.6.2 Architect（架构师）

- **功能目标**: 系统架构设计、技术选型、模块划分
- **权限**: 文件读写 + 命令执行，可 Fork 子代理
- **最大迭代**: 30
- **优先级**: 90
- **来源文件**: `internal/team/roles.go` → `NewArchitectRole()`
- **实现状态**: 已完成

#### 3.6.3 Developer（开发者）

- **功能目标**: 代码实现、功能开发、Bug 修复
- **权限**: 文件读写 + 命令执行，不可 Fork
- **最大迭代**: 40
- **优先级**: 80
- **来源文件**: `internal/team/roles.go` → `NewDeveloperRole()`
- **实现状态**: 已完成

#### 3.6.4 Tester（测试者）

- **功能目标**: 编写测试用例、执行测试、验证功能
- **权限**: 文件读写 + 命令执行，不可 Fork
- **最大迭代**: 30
- **优先级**: 70
- **来源文件**: `internal/team/roles.go` → `NewTesterRole()`
- **实现状态**: 已完成

#### 3.6.5 Reviewer（审查者）

- **功能目标**: 代码审查、质量把控
- **权限**: 文件只读，不可 Fork
- **最大迭代**: 20
- **优先级**: 60
- **来源文件**: `internal/team/roles.go` → `NewReviewerRole()`
- **实现状态**: 已完成

#### 3.6.6 Supervisor（监督者）

- **功能目标**: 监督 Agent 行为、检查输出质量、纠偏
- **权限**: 文件只读，不可 Fork
- **最大迭代**: 15
- **优先级**: 50
- **来源文件**: `internal/team/roles.go` → `NewSupervisorRole()`
- **实现状态**: 已完成

#### 3.6.7 默认工作流

- **功能目标**: 定义标准的开发流程工作流
- **工作流步骤**:
  1. `task_analysis`（Orchestrator）→ 分析任务，制定执行计划
  2. `architecture_design`（Architect）→ 架构设计和技术选型
  3. `implementation`（Developer）→ 实现功能代码
  4. `testing`（Tester）→ 编写和执行测试
  5. `code_review`（Reviewer）→ 代码审查
  6. `task_complete`（Orchestrator）→ 任务完成
- **条件分支**: 测试失败→回到实现；审查失败→回到实现
- **来源文件**: `internal/team/definition.go` → `DefaultTeam()`
- **实现状态**: 已完成

---

### 3.7 状态管理功能模块

#### 3.7.1 任务状态管理器

- **功能目标**: 以 YAML 格式持久化管理任务状态
- **功能描述**:
  - 任务 CRUD 操作（创建、加载、更新状态）
  - 进度管理（更新阶段、完成步骤）
  - 决策记录
  - 文件跟踪
  - Agent 身份切换
  - 自动保存
- **来源文件**: `internal/state/manager.go`
- **实现状态**: 已完成

#### 3.7.2 任务状态数据结构

- **功能目标**: 定义任务状态的数据结构
- **数据结构**: 任务信息（ID、目标、状态、时间）+ 进度（当前阶段、已完成步骤、待完成步骤）+ 上下文（决策、约束、文件）+ Agent 信息（活跃Agent、执行历史）
- **来源文件**: `internal/state/task.go`
- **实现状态**: 已完成

#### 3.7.3 YAML序列化

- **功能目标**: YAML 格式的序列化和反序列化
- **来源文件**: `internal/state/yaml.go`
- **实现状态**: 已完成

#### 3.7.4 对话管理（GUI层）

- **功能目标**: 在 GUI 层管理对话状态和消息流
- **功能描述**:
  - 对话创建和切换
  - 消息列表管理
  - 流式消息状态跟踪
  - Token 统计
  - 推理中断
  - 对话清空
  - 工作目录切换
- **来源文件**: `internal/gui/app.go`
- **实现状态**: 已完成

---

### 3.8 模型通信功能模块

#### 3.8.1 OpenAI兼容客户端

- **功能目标**: 与 OpenAI 兼容的 Chat Completion API 通信
- **功能描述**:
  - 支持标准 Chat Completion 请求
  - 支持带温度参数的请求
  - HTTP 超时设置 300 分钟（适配慢速本地模型）
  - Token 估算（每4字符≈1 Token）
  - 上下文溢出检查
- **来源文件**: `internal/model/client.go`
- **实现状态**: 已完成

#### 3.8.2 SSE流式响应

- **功能目标**: 解析 Server-Sent Events 格式的流式响应
- **功能描述**:
  - 解析 `data: ` 前缀的 SSE 数据行
  - `[DONE]` 结束标记处理
  - JSON 反序列化
  - 思考标签保留（交由 Agent 核心处理）
- **来源文件**: `internal/model/client.go` → `doStreamChatCompletion()`
- **实现状态**: 已完成

#### 3.8.3 消息类型系统

- **功能目标**: 定义 OpenAI 兼容的消息类型
- **类型**: Message（system/user/assistant/tool）, ToolCall, FunctionCall, StreamResponse, Delta（含 ReasoningContent）
- **来源文件**: `internal/model/message.go`
- **实现状态**: 已完成

---

### 3.9 配置系统功能模块

#### 3.9.1 YAML配置加载

- **功能目标**: 从 YAML 文件加载应用配置
- **功能描述**:
  - 配置项包括：模型端点、API Key、模型名称、上下文大小、最大迭代次数、温度、工作目录、允许命令白名单
  - 支持环境变量覆盖
  - 支持命令行参数覆盖
- **来源文件**: `internal/config/loader.go`
- **实现状态**: 已完成

#### 3.9.2 Wails配置

- **功能目标**: Wails 桌面应用框架配置
- **来源文件**: `wails.json`
- **实现状态**: 已完成

---

## 四、辅助功能

### 4.1 反馈注入器（FeedbackInjector）

- **功能目标**: 将审查结果转换为用户消息注入到对话历史中
- **功能描述**:
  - 多种反馈级别：info/warning/error/critical
  - 阻断反馈（阻断级别问题）
  - 警告反馈（提醒但不阻断）
  - 进度反馈（迭代次数提醒）
  - 循环检测反馈
  - 失败反馈
  - 方向偏离反馈
  - 批量反馈注入
- **来源文件**: `internal/agent/feedback.go`
- **实现状态**: 已完成

### 4.2 错误边界组件（ErrorBoundary）

- **功能目标**: React 错误边界，捕获子组件渲染错误
- **来源文件**: `frontend/src/components/ErrorBoundary.tsx`
- **实现状态**: 已完成

---

## 五、非功能性需求

### 5.1 性能需求

| 需求 | 描述 | 当前实现 |
|------|------|---------|
| 长时间推理支持 | HTTP 超时设置 300 分钟 | `internal/model/client.go` |
| 上下文压缩 | 渐进式压缩策略，阈值 80% | `internal/agent/compressor.go` |
| 流式输出 | SSE 流式响应，实时展示 | `internal/model/client.go` + `internal/gui/stream_bridge.go` |
| 工具执行超时 | 默认 60 秒，最大 600 秒 | `internal/tools/command.go` |
| Token 估算 | 每 4 字符约 1 Token | `internal/model/client.go` |

### 5.2 安全需求

| 需求 | 描述 | 当前实现 |
|------|------|---------|
| 命令白名单 | 配置允许执行的命令列表 | `configs/config.yaml` → `tools.allow_commands` |
| 绝对路径要求 | 文件操作工具必须使用绝对路径 | `internal/tools/filesystem.go` |
| 子代理权限隔离 | 不同角色有不同的工具权限 | `internal/team/roles.go` |
| 线程安全 | 关键组件使用 `sync.RWMutex` 保护 | 全局 |

> **风险提示**: 命令白名单的验证逻辑在 `main.go` 的 `initToolRegistry` 中未实际使用，`allow_commands` 配置目前仅作为声明，未在工具执行时强制校验。此为安全隐患。

### 5.3 可用性需求

| 需求 | 描述 | 当前实现 |
|------|------|---------|
| 双模式运行 | CLI 和 GUI 模式共享核心逻辑 | `cmd/agentplus/main.go` |
| 交互式输入 | CLI 支持多行输入，空行提交 | `interactiveInput()` |
| 流式光标 | GUI 流式输出时显示闪烁光标 | `MessageItem.tsx`, `ThinkingBlock.tsx` |
| 错误展示 | 错误信息红色横幅展示，可关闭 | `App.tsx` |
| 自动滚动 | 消息列表自动滚动到底部 | `MessageList.tsx` |

### 5.4 可靠性需求

| 需求 | 描述 | 当前实现 |
|------|------|---------|
| 审查-校验-修正闭环 | Reviewer → Verifier → SelfCorrector | `internal/agent/` |
| 双层循环保障 | 内层循环 + 外层强制校验 | `internal/agent/core.go` |
| 指数退避 | 重试延迟指数增长 | `internal/agent/selfcorrector.go` |
| 优雅退出 | Ctrl+C 信号处理 | `cmd/agentplus/main.go` |
| 推理中断 | GUI 支持用户主动打断推理 | `internal/gui/app.go` → `InterruptInference()` |

### 5.5 可扩展性需求

| 需求 | 描述 | 当前实现 |
|------|------|---------|
| 自定义工具 | 实现 `Tool` 接口即可注册 | `internal/tools/types.go` |
| 自定义校验规则 | 实现 `VerifyRule` 接口即可注册 | `internal/agent/verifier.go` |
| 自定义角色 | 创建 `AgentRole` 并添加到团队 | `internal/team/definition.go` |
| 自定义工作流 | 定义 `WorkflowStep` 条件分支 | `internal/team/definition.go` |
| 自定义 StreamHandler | 实现 `StreamHandler` 接口 | `internal/agent/stream.go` |
| 项目模板 | 多语言项目模板脚手架 | `project/` 目录 |

---

## 六、功能完整度评估

### 6.1 已完成功能（功能完整）

| 模块 | 功能 | 完整度 |
|------|------|--------|
| Agent核心 | 双层主循环、流式调用、思考处理、Fork机制、历史管理、上下文压缩、异步发送、日志记录 | 100% |
| 工具系统 | 5个文件系统工具 + 2个命令执行工具 + 4个状态管理工具 = 11个工具 | 100% |
| 审查校验 | 6类问题检测 + 6类成果校验 + 分离重试修正 | 100% |
| GUI界面 | 9个前端组件 + 流式桥接 + 对话管理 | 100% |
| CLI命令行 | 交互式输入 + 参数解析 + 模式路由 | 100% |
| 团队角色 | 6种预定义角色 + 6步默认工作流 | 100% |
| 模型通信 | OpenAI兼容客户端 + SSE流式 + 消息类型 | 100% |

### 6.2 部分完成/需改进功能

| 功能 | 问题描述 | 优先级 | 来源 |
|------|---------|--------|------|
| 命令白名单校验 | `allow_commands` 配置存在但未在工具执行时强制校验 | P1（安全） | `cmd/agentplus/main.go` → `initToolRegistry()` |
| GUI对话持久化 | 对话数据仅存在于内存，重启后丢失 | P2 | `internal/gui/app.go` |
| 对话切换 | 侧边栏 onSelect 回调为空函数，无法切换对话 | P2 | `frontend/src/App.tsx` L116 |
| Token精确计算 | 使用字符数/4的粗略估算，不精确 | P3 | `internal/model/client.go` |
| 工具执行结果展示 | `ToolResultBlock.tsx` 组件未在 `MessageItem` 中被引用 | P2 | `frontend/src/components/` |

### 6.3 待实现/缺失功能

| 功能 | 描述 | 优先级 |
|------|------|--------|
| 单元测试 | 项目无任何测试文件 | P1 |
| 状态清理机制 | `state/` 目录有30个历史任务文件，缺少自动清理 | P2 |
| 对话保存/导出 | GUI 不支持保存或导出对话记录 | P2 |
| 多语言模型模板 | `project/` 模板目录中 Kotlin/Java 模板的具体使用方式未集成 | P3 |
| Supervisor模块 | `internal/supervisor/monitor.go` 提及6类检查和4级干预，但实际集成程度待确认 | P3 |
| 错误恢复 | Agent 运行异常退出后无法恢复到上次状态继续执行 | P3 |
| 并行工具执行 | `executor.go` 支持并行执行但具体策略未完全启用 | P3 |

---

## 七、数据流概览

### 7.1 用户任务执行流程

```
用户输入任务目标
  → Agent.Run() 创建任务状态
  → 构建系统提示词 + 任务提示
  → [内层循环]
    → callModel() SSE流式调用
    → Reviewer.ReviewOutput() 审查
      → 阻断? → SelfCorrector → 注入修正 → continue
    → 有ToolCalls? → executeTools()
      → complete_task? → Verifier → 校验
        → 失败? → SelfCorrector → 注入修正 → continue
        → 通过? → break内层
      → fail_task? → 直接失败返回
      → 其他工具 → continue
    → 无ToolCalls → isTaskComplete()?
      → 是 → Verifier → 通过/修正
      → 否 → 提示继续
  → [外层循环]
    → 强制校验
      → 通过 → 任务真正完成
      → 失败 → 重置状态 → 注入修正 → 继续外层
```

> 来源: `internal/agent/core.go` → `Agent.Run()`

### 7.2 GUI流式数据流

```
Agent.callModel()
  → StreamHandler.OnThinking() → StreamBridge.OnThinking()
    → App.currentMessage.thinking += chunk
    → Wails EventsEmit("stream:thinking")
    → React useStreamEvents → setThinking

  → StreamHandler.OnContent() → StreamBridge.OnContent()
    → App.currentMessage.content += chunk
    → Wails EventsEmit("stream:content")
    → React useStreamEvents → appendContent

  → StreamHandler.OnToolCall() → StreamBridge.OnToolCall()
    → App.currentMessage.toolCalls 更新
    → Wails EventsEmit("stream:toolcall")
    → React useStreamEvents → setToolCalls

  → StreamHandler.OnToolResult() → StreamBridge.OnToolResult()
    → App.currentMessage.toolCalls 结果更新
    → Wails EventsEmit("stream:toolresult")
    → React useStreamEvents → setToolResult

  → StreamHandler.OnComplete() → StreamBridge.OnComplete()
    → 固化消息到列表，更新Token统计
    → Wails EventsEmit("stream:complete")

  → StreamHandler.OnTaskDone() → StreamBridge.OnTaskDone()
    → 重置 isStreaming
    → Wails EventsEmit("stream:done")
```

> 来源: `internal/gui/stream_bridge.go` + `frontend/src/App.tsx`

---

## 八、架构决策记录

| 编号 | 决策 | 原因 | 来源 |
|------|------|------|------|
| ADR-001 | 双层循环设计 | 内层处理正常迭代，外层提供"强制校验"保障 | `internal/agent/core.go` |
| ADR-002 | 分离的审查/校验重试计数 | 审查和校验失败原因不同，容错需求不同 | `internal/agent/selfcorrector.go` |
| ADR-003 | 探索期机制 | Agent 需要先了解环境，避免误报"无进度" | `internal/agent/reviewer.go` |
| ADR-004 | 渐进式上下文压缩 | 先轻度压缩保留更多上下文，超限再深度压缩 | `internal/agent/compressor.go` |
| ADR-005 | Fork子代理共享状态 | 子代理需要看到和更新全局任务状态，但消息历史私有 | `internal/agent/fork.go` |
| ADR-006 | 双模式思考内容处理 | 不同模型思考内容输出方式不同（reasoning_content vs thinking标签） | `internal/agent/thinking.go` |

---

## 九、新增功能需求：文件语法检查系统

> 版本: v1.1.0
> 更新时间: 2026-04-12
> 任务编号: SYNTAX-001
> 优先级: P0（紧急）
> 来源: `ai/user_say.md` 用户需求 + `ai/tasks/active/task-SYNTAX-001.yaml`

### 9.1 功能概述

#### 9.1.1 功能目标

在文件写入（`write_file`）和编辑（`edit_file`）操作完成后，系统自动根据文件扩展名执行真正的语法检查（AST解析级别），将检查结果嵌入工具返回值（`ToolResult`），辅助模型发现并修正基础语法错误，形成"写入→检查→反馈→修正"的闭环。

#### 9.1.2 功能范围

| 范围项 | 说明 |
|--------|------|
| 影响工具 | `write_file`（行为变更）、`edit_file`（新增工具） |
| 涉及模块 | `internal/tools/`（工具层）、新增 `internal/tools/syntax/`（语法检查器） |
| 不影响 | `read_file`、`list_directory`、`delete_file`、`create_directory` 等只读/目录工具 |
| 与Verifier的关系 | 本功能为工具层即时检查，Verifier仍负责任务完成时的综合校验，两者互补 |

#### 9.1.3 核心原则

1. **真正语法检查**：必须使用AST解析或标准库解析器，严禁使用正则匹配作为检查手段
2. **优雅降级**：外部工具不可用时不影响文件操作本身，仅跳过检查并标记降级状态
3. **结果嵌入**：语法检查结果直接嵌入 `ToolResult`，模型无需额外调用即可看到
4. **性能优先**：Go原生解析器优先，外部工具作为后备方案

---

### 9.2 语法检查器模块设计

#### 9.2.1 模块位置与架构

```
internal/tools/syntax/
├── checker.go          # SyntaxChecker 接口定义 + 调度器
├── result.go           # SyntaxCheckResult 数据结构定义
├── json_checker.go     # JSON语法检查（Go原生）
├── yaml_checker.go     # YAML语法检查（gopkg.in/yaml.v3）
├── xml_checker.go      # XML语法检查（encoding/xml）
├── html_checker.go     # HTML语法检查（golang.org/x/net/html）
├── go_checker.go       # Go语言语法检查（go/parser）
├── toml_checker.go     # TOML语法检查（第三方库）
├── css_checker.go      # CSS语法检查（第三方库）
├── properties_checker.go # Properties格式检查（Go原生实现）
├── sql_checker.go      # SQL语法检查（基础解析）
├── external_checker.go # 外部工具调用检查器（node/python/javac等）
└── registry.go         # 检查器注册表（扩展名→检查器映射）
```

#### 9.2.2 核心接口定义

```go
// SyntaxChecker 语法检查器接口
type SyntaxChecker interface {
    // SupportedExtensions 返回支持的文件扩展名列表（含.号）
    SupportedExtensions() []string
    
    // Check 对内容进行语法检查
    // content: 文件内容
    // filePath: 文件路径（用于错误提示）
    Check(content string, filePath string) *SyntaxCheckResult
}

// SyntaxCheckResult 语法检查结果
type SyntaxCheckResult struct {
    // Language 检查的语言类型
    Language    string               `json:"language"`
    
    // HasErrors 是否存在语法错误
    HasErrors   bool                 `json:"has_errors"`
    
    // Errors 语法错误列表
    Errors      []SyntaxError        `json:"errors,omitempty"`
    
    // Warnings 语法警告列表
    Warnings    []SyntaxWarning      `json:"warnings,omitempty"`
    
    // CheckMethod 检查方法：native(Go原生) / external(外部工具) / skipped(跳过)
    CheckMethod string               `json:"check_method"`
    
    // Degraded 是否降级（外部工具不可用）
    Degraded    bool                 `json:"degraded,omitempty"`
    
    // DegradedReason 降级原因
    DegradedReason string            `json:"degraded_reason,omitempty"`
}

// SyntaxError 语法错误
type SyntaxError struct {
    // Line 行号（1-based）
    Line        int                  `json:"line,omitempty"`
    
    // Column 列号（1-based，可选）
    Column      int                  `json:"column,omitempty"`
    
    // Message 错误描述
    Message     string               `json:"message"`
    
    // Severity 严重程度：error / warning
    Severity    string               `json:"severity"`
    
    // Suggestion 修正建议（可选）
    Suggestion  string               `json:"suggestion,omitempty"`
}

// SyntaxWarning 语法警告（不影响运行但建议修正）
type SyntaxWarning = SyntaxError  // 复用SyntaxError结构
```

#### 9.2.3 调度器设计

```go
// SyntaxCheckDispatcher 语法检查调度器
// 根据文件扩展名选择合适的检查器并执行检查
type SyntaxCheckDispatcher struct {
    // checkers 扩展名→检查器映射
    checkers map[string]SyntaxChecker
    
    // externalChecker 外部工具检查器（用于无原生解析器的语言）
    externalChecker *ExternalChecker
}
```

**调度逻辑**:
1. 根据文件扩展名查找已注册的检查器
2. 如果存在原生检查器，使用原生检查器执行
3. 如果不存在原生检查器，尝试外部工具检查器
4. 如果外部工具不可用，返回降级结果
5. 未知文件类型直接跳过检查

---

### 9.3 各语言语法检查策略

#### 9.3.1 策略总览

| 语言 | 扩展名 | 主策略 | 备选策略 | 依赖 |
|------|--------|--------|---------|------|
| **JSON** | `.json` | Go原生 `encoding/json` | 无 | 无（标准库） |
| **YAML** | `.yaml`, `.yml` | Go原生 `gopkg.in/yaml.v3` | 无 | 已有依赖 |
| **XML** | `.xml`, `.xsd`, `.xsl`, `.xslt`, `.svg`, `.pom` | Go原生 `encoding/xml` | 无 | 无（标准库） |
| **HTML** | `.html`, `.htm` | `golang.org/x/net/html` | 无 | 需新增依赖 |
| **Go** | `.go` | Go原生 `go/parser` + `go/token` | 无 | 无（标准库） |
| **TOML** | `.toml` | `github.com/BurntSushi/toml` | 无 | 需新增依赖 |
| **CSS** | `.css` | `github.com/tdewolff/parse/v2/css` | 无 | 需新增依赖 |
| **Properties** | `.properties` | Go原生逐行解析 | 无 | 无（标准库） |
| **SQL** | `.sql` | Go原生基础解析（分号/括号匹配） | 无 | 无（标准库） |
| **JavaScript** | `.js`, `.mjs` | 外部 `node --check` | 降级跳过 | 需要 Node.js |
| **TypeScript** | `.ts`, `.tsx` | 外部 `npx tsc --noEmit` | 降级跳过 | 需要 TypeScript |
| **Python** | `.py`, `.pyw` | 外部 `python -m py_compile` | 降级跳过 | 需要 Python |
| **Java** | `.java` | 外部 `javac -Xstdout /dev/null` | 降级跳过 | 需要 JDK |
| **Kotlin** | `.kt`, `.kts` | 外部 `kotlinc -nowarn` | 降级跳过 | 需要 Kotlin |
| **Rust** | `.rs` | 外部 `rustc --crate-type lib` | 降级跳过 | 需要 Rust |
| **Shell** | `.sh`, `.bash` | 外部 `bash -n` | 降级跳过 | 需要 Bash |
| **PowerShell** | `.ps1`, `.psm1` | 外部 `pwsh -NoExec -Command` | 降级跳过 | 需要 PowerShell |
| **BAT** | `.bat`, `.cmd` | 外部 `cmd /c call /?` 验证 | 降级跳过 | Windows原生 |
| **Gradle** | `.gradle` | 外部 `gradle --dry-run` | 降级跳过 | 需要 Gradle |
| **Gradle(KTS)** | `.gradle.kts` | 外部 `gradle --dry-run` | 降级跳过 | 需要 Gradle |
| **HTML+JS** | HTML中嵌入 | HTML检查 + 提取`<script>`后JS检查 | 仅HTML检查 | 同HTML+JS |

#### 9.3.2 Go原生解析器详细规格

##### 9.3.2.1 JSON检查器 (`json_checker.go`)

- **功能目标**: 使用 `encoding/json` 对JSON内容进行语法验证
- **实现方式**: `json.Unmarshal` 或 `json.Decoder`（支持流式，可报告错误位置）
- **错误信息提取**: 从 `json.SyntaxError` 获取偏移量，转换为行号和列号
- **支持检测**: 缺少逗号、未闭合的括号/花括号、非法值、尾随逗号、字符串未闭合、非法Unicode转义
- **优先级**: P0
- **依赖**: Go标准库，无额外依赖

##### 9.3.2.2 YAML检查器 (`yaml_checker.go`)

- **功能目标**: 使用 `gopkg.in/yaml.v3` 对YAML内容进行语法验证
- **实现方式**: `yaml.Unmarshal` 并捕获 `yaml.TypeError`
- **错误信息提取**: 从错误中提取行号、错误描述
- **支持检测**: 缩进错误、未闭合的引号、非法的YAML语法、重复键、非法锚点/别名
- **优先级**: P0
- **依赖**: 已在go.mod中

##### 9.3.2.3 XML检查器 (`xml_checker.go`)

- **功能目标**: 使用 `encoding/xml` 对XML内容进行语法验证
- **实现方式**: `xml.NewDecoder` 逐Token解析，捕获错误
- **错误信息提取**: 从 `xml.SyntaxError` 获取行号和错误信息
- **支持检测**: 未闭合标签、非法XML声明、编码错误、实体引用错误、属性引号未闭合
- **优先级**: P0
- **依赖**: Go标准库，无额外依赖

##### 9.3.2.4 HTML检查器 (`html_checker.go`)

- **功能目标**: 使用 `golang.org/x/net/html` 对HTML内容进行语法验证
- **实现方式**: `html.Parse` 构建DOM树，捕获解析错误
- **支持检测**: 未闭合标签、嵌套错误、非法属性、DOCTYPE缺失
- **HTML+JS检查**: 解析完成后提取所有 `<script>` 标签内容，调用JS外部检查器进行语法检查
- **优先级**: P0
- **依赖**: 需新增 `golang.org/x/net`

##### 9.3.2.5 Go语言检查器 (`go_checker.go`)

- **功能目标**: 使用 `go/parser` + `go/token` 对Go源码进行语法验证
- **实现方式**: `parser.ParseFile` 解析为AST
- **错误信息提取**: 从 `scanner.ErrorList` 获取每个错误的位置（文件:行:列）和描述
- **支持检测**: 语法错误、未闭合的括号/花括号、非法标识符、类型错误、导入路径错误
- **注意**: 仅做语法级检查，不做类型检查（避免需要完整包上下文）
- **优先级**: P1
- **依赖**: Go标准库，无额外依赖

##### 9.3.2.6 TOML检查器 (`toml_checker.go`)

- **功能目标**: 使用第三方库对TOML内容进行语法验证
- **实现方式**: `toml.Decode` 并捕获错误
- **支持检测**: 键值对格式错误、表头格式错误、数组/内联表语法错误、日期格式错误
- **优先级**: P1
- **依赖**: 需新增 `github.com/BurntSushi/toml`

##### 9.3.2.7 CSS检查器 (`css_checker.go`)

- **功能目标**: 对CSS样式表进行语法验证
- **实现方式**: 使用 `github.com/tdewolff/parse/v2/css` 解析CSS Token
- **支持检测**: 未闭合的花括号、非法选择器、属性值错误、缺少分号
- **优先级**: P2
- **依赖**: 需新增 `github.com/tdewolff/parse/v2`

##### 9.3.2.8 Properties检查器 (`properties_checker.go`)

- **功能目标**: 对Java Properties文件格式进行验证
- **实现方式**: Go原生逐行解析，验证键值对格式
- **支持检测**: 非法转义序列、行延续符错误、编码问题
- **优先级**: P2
- **依赖**: Go标准库，无额外依赖

##### 9.3.2.9 SQL基础检查器 (`sql_checker.go`)

- **功能目标**: 对SQL语句进行基础语法验证
- **实现方式**: Go原生实现基础的分号闭合、括号匹配、字符串闭合检查
- **支持检测**: 未闭合的字符串、未闭合的括号、缺少分号
- **限制**: 不做完整的SQL语法解析（SQL方言差异大），仅做基础结构检查
- **优先级**: P2
- **依赖**: Go标准库，无额外依赖

#### 9.3.3 外部工具检查器详细规格

##### 9.3.3.1 外部检查器通用框架 (`external_checker.go`)

- **功能目标**: 统一管理通过调用外部进程执行语法检查的逻辑
- **核心流程**:
  1. 将待检查内容写入临时文件
  2. 执行外部工具命令
  3. 解析stdout/stderr获取错误信息
  4. 清理临时文件
  5. 返回检查结果
- **降级策略**:
  - 首次调用时检测外部工具是否可用（缓存可用性状态）
  - 不可用时直接返回降级结果，不再重复检测
  - 可用性缓存有效期：整个进程生命周期
- **超时控制**: 单次检查最长5秒
- **优先级**: P0

##### 9.3.3.2 JavaScript检查（外部）

- **检查命令**: `node --check <file>`
- **支持版本**: ES2024+ 现代语法
- **错误解析**: 解析 `node` 输出的错误行号和描述
- **降级条件**: `node` 命令不存在或执行失败

##### 9.3.3.3 TypeScript检查（外部）

- **检查命令**: `npx tsc --noEmit --strict <file>` 或 `npx -p typescript tsc --noEmit <file>`
- **支持版本**: TypeScript 5.x+ 最新语法
- **错误解析**: 解析 `tsc` 输出的错误行号和描述
- **降级条件**: `tsc` 或 `npx` 命令不存在

##### 9.3.3.4 Python检查（外部）

- **检查命令**: `python -m py_compile <file>`
- **支持版本**: Python 3.x
- **错误解析**: 解析 `SyntaxError` 输出的行号和描述
- **降级条件**: `python` 命令不存在

##### 9.3.3.5 Java检查（外部）

- **检查命令**: `javac -Xstdout /dev/null -d /tmp <file>`（仅语法级）
- **支持版本**: Java 26（使用当前安装的javac版本）
- **错误解析**: 解析javac输出的错误行号、列号和描述
- **降级条件**: `javac` 命令不存在
- **注意**: Java编译需要类路径上下文，仅做最基础的语法验证

##### 9.3.3.6 Kotlin检查（外部）

- **检查命令**: `kotlinc -nowarn -script <file>` 或 `kotlinc -Xno-warn <file>`
- **支持版本**: Kotlin 2.x
- **错误解析**: 解析kotlinc输出的错误信息
- **降级条件**: `kotlinc` 命令不存在

##### 9.3.3.7 Rust检查（外部）

- **检查命令**: `rustc --crate-type lib --edition 2024 <file>`（仅语法级）
- **错误解析**: 解析rustc输出的错误行号和描述
- **降级条件**: `rustc` 命令不存在

##### 9.3.3.8 Shell检查（外部）

- **检查命令**: `bash -n <file>` (Bash) 或 `sh -n <file>` (POSIX Shell)
- **错误解析**: 解析shell输出的错误行号和描述
- **降级条件**: `bash` / `sh` 命令不存在

##### 9.3.3.9 PowerShell检查（外部）

- **检查命令**: `pwsh -NoExec -Command "[System.Management.Automation.Language.Parser]::ParseFile('<file>', [ref]$null, [ref]$null)"`
- **错误解析**: 解析PowerShell输出
- **降级条件**: `pwsh` 命令不存在
- **平台注意**: Windows PowerShell (`powershell`) 和 PowerShell Core (`pwsh`) 均可

##### 9.3.3.10 BAT检查（外部）

- **检查命令**: Windows下使用 `cmd /c echo @echo off > nul && <file>` 进行基础验证
- **限制**: BAT语法检查能力有限，仅能检测明显错误
- **降级条件**: 非Windows平台

##### 9.3.3.11 Gradle检查（外部）

- **检查命令**: `gradle --dry-run -b <file>`（Groovy DSL）或 `gradle --dry-run -b <file>`（Kotlin DSL）
- **错误解析**: 解析gradle输出的错误信息
- **降级条件**: `gradle` 命令不存在
- **注意**: Gradle检查较重，可能需要较长超时时间

---

### 9.4 新增工具：`edit_file`

#### 9.4.1 功能目标

提供基于行号或字符串替换的文件编辑能力，避免为修改少量内容而需要重写整个文件。编辑完成后自动进行语法检查。

#### 9.4.2 输入定义

| 参数名 | 类型 | 必需 | 描述 |
|--------|------|------|------|
| `path` | string | 是 | 文件的绝对路径 |
| `mode` | string | 是 | 编辑模式：`line_replace`（行号替换）或 `string_replace`（字符串替换） |
| `start_line` | integer | 条件必需 | 起始行号（1-based），`line_replace` 模式必需 |
| `end_line` | integer | 条件必需 | 结束行号（1-based，包含），`line_replace` 模式必需 |
| `old_string` | string | 条件必需 | 被替换的原始字符串，`string_replace` 模式必需 |
| `new_string` | string | 否 | 替换后的新字符串，默认为空（即删除操作） |
| `replace_all` | boolean | 否 | `string_replace`模式下是否替换所有匹配项，默认false |

**参数约束**:
- `line_replace` 模式：`start_line` 和 `end_line` 必需，且 `start_line <= end_line`
- `string_replace` 模式：`old_string` 必需且非空
- `path` 必须是绝对路径且在工作目录范围内
- 文件必须存在（不同于 `write_file` 会自动创建）

#### 9.4.3 输出定义

**成功时**（包含语法检查结果）:

```json
{
  "success": true,
  "data": {
    "path": "/path/to/file.java",
    "mode": "string_replace",
    "replacements_made": 1,
    "lines_affected": [5, 8],
    "bytes_written": 1024,
    "message": "file edited successfully",
    "syntax_check": {
      "language": "java",
      "has_errors": false,
      "errors": [],
      "warnings": [],
      "check_method": "external",
      "degraded": false
    }
  }
}
```

**成功但有语法错误时**:

```json
{
  "success": true,
  "data": {
    "path": "/path/to/file.java",
    "mode": "line_replace",
    "lines_affected": [10, 15],
    "bytes_written": 1024,
    "message": "file edited successfully, but syntax errors detected. Please fix them immediately.",
    "syntax_check": {
      "language": "java",
      "has_errors": true,
      "errors": [
        {
          "line": 12,
          "column": 5,
          "message": "';' expected",
          "severity": "error",
          "suggestion": "Add a semicolon at the end of the statement"
        }
      ],
      "warnings": [],
      "check_method": "external",
      "degraded": false
    }
  }
}
```

**降级时**:

```json
{
  "success": true,
  "data": {
    "path": "/path/to/file.java",
    "mode": "string_replace",
    "replacements_made": 1,
    "bytes_written": 1024,
    "message": "file edited successfully (syntax check skipped: javac not available)",
    "syntax_check": {
      "language": "java",
      "has_errors": false,
      "errors": [],
      "warnings": [],
      "check_method": "skipped",
      "degraded": true,
      "degraded_reason": "external tool 'javac' is not available"
    }
  }
}
```

#### 9.4.4 边界条件

| 场景 | 处理方式 |
|------|---------|
| 文件不存在 | 返回错误 `file does not exist: <path>` |
| `start_line` 超出文件行数 | 返回错误 `start_line exceeds file line count` |
| `end_line` 超出文件行数 | 返回错误 `end_line exceeds file line count` |
| `start_line > end_line` | 返回错误 `start_line must be <= end_line` |
| `old_string` 未找到 | 返回错误 `old_string not found in file` |
| `old_string` 存在多个匹配且 `replace_all=false` | 返回错误 `found multiple matches for old_string. Provide more context or use replace_all=true` |
| `new_string` 为空 | 视为删除操作（删除匹配行或替换为空） |
| 文件为空 | `line_replace` 模式返回错误；`string_replace` 模式在空文件中搜索 |
| 路径为目录 | 返回错误 `path is a directory, not a file` |
| 编码问题 | 按 UTF-8 处理，非UTF-8文件返回错误 |

#### 9.4.5 异常处理

| 错误码 | 描述 | HTTP类比 |
|--------|------|---------|
| `file_not_found` | 文件不存在 | 404 |
| `path_is_directory` | 路径是目录 | 400 |
| `invalid_mode` | 不支持的编辑模式 | 400 |
| `missing_parameter` | 缺少必需参数 | 400 |
| `line_out_of_range` | 行号超出范围 | 400 |
| `old_string_not_found` | 原始字符串未找到 | 404 |
| `multiple_matches` | 多处匹配且未指定replace_all | 409 |
| `edit_failed` | 编辑操作失败 | 500 |

#### 9.4.6 验收标准

- [ ] `line_replace` 模式正确替换指定行范围
- [ ] `string_replace` 模式正确替换匹配的字符串
- [ ] `replace_all=true` 时替换所有匹配项
- [ ] 编辑完成后自动执行语法检查
- [ ] 语法检查结果嵌入返回值
- [ ] 有语法错误时 `message` 字段包含提示信息
- [ ] 工具注册到 `ToolRegistry`，模型可调用

#### 9.4.7 工具Description

```
编辑指定文件的内容。支持两种编辑模式：
1. line_replace模式：通过指定起始行(start_line)和结束行(end_line)替换指定范围的行。
2. string_replace模式：通过指定原始字符串(old_string)和新字符串(new_string)进行精确替换。
   设置replace_all=true可替换所有匹配项。

编辑完成后会自动进行语法检查，如果发现错误会在返回结果中提示。
**重要：必须使用此工具或write_file工具进行文件编辑操作，不应通过execute_command等命令行方式绕过文件操作工具。**
路径必须是绝对路径。
```

---

### 9.5 工具行为变更：`write_file`

#### 9.5.1 行为变更概述

`write_file` 工具在现有文件写入功能的基础上，增加写入后自动语法检查。文件写入本身不受语法检查结果影响（写入始终成功），语法检查结果作为附加信息嵌入返回值。

#### 9.5.2 输出定义变更

**变更前**:

```json
{
  "success": true,
  "data": {
    "path": "/path/to/file.js",
    "bytes_write": 1024,
    "message": "file written successfully"
  }
}
```

**变更后**:

```json
{
  "success": true,
  "data": {
    "path": "/path/to/file.js",
    "bytes_write": 1024,
    "message": "file written successfully, but syntax errors detected. Please fix them immediately.",
    "syntax_check": {
      "language": "javascript",
      "has_errors": true,
      "errors": [
        {
          "line": 5,
          "column": 10,
          "message": "Unexpected token '}'",
          "severity": "error",
          "suggestion": "Check for missing semicolons or parentheses before this token"
        }
      ],
      "warnings": [],
      "check_method": "external",
      "degraded": false
    }
  }
}
```

**新增字段**:
- `data.syntax_check`: 语法检查结果对象（结构同 9.2.2 定义）
- `data.message`: 当存在语法错误时，消息追加提示 `"but syntax errors detected. Please fix them immediately."`
- 当语法检查被跳过（不支持的文件类型）时，不包含 `syntax_check` 字段
- 当语法检查降级时，`data.message` 追加 `"(syntax check skipped: <reason>)"`

#### 9.5.3 行为变更详细规格

| 场景 | 变更前行为 | 变更后行为 |
|------|-----------|-----------|
| 写入支持检查的文件类型 | 返回写入成功 | 写入成功 + 自动语法检查 + 结果嵌入返回值 |
| 写入不支持的文件类型（如.txt, .md） | 返回写入成功 | 返回写入成功（无 `syntax_check` 字段） |
| 写入有语法错误的文件 | 返回写入成功 | 返回写入成功 + 语法错误列表 + 错误提示 |
| 写入语法正确的文件 | 返回写入成功 | 返回写入成功 + `has_errors: false` |
| 外部工具不可用 | N/A | 写入成功 + 降级标记 + 降级原因 |

#### 9.5.4 Description变更

**变更前**:
```
将内容写入指定路径的文件。如果文件不存在则创建，如果存在则覆盖。路径必须是绝对路径。会自动创建所需的父目录。
```

**变更后**:
```
将内容写入指定路径的文件。如果文件不存在则创建，如果存在则覆盖。路径必须是绝对路径。会自动创建所需的父目录。
写入完成后会自动根据文件扩展名进行语法检查（支持JSON/YAML/XML/HTML/Go/JS/TS/Python/Java/Kotlin/Rust等），检查结果会附加在返回数据中。如果发现语法错误，会在返回消息中提示，请根据错误信息修正文件。
**重要：必须使用此工具进行文件创建/写入操作，不应通过execute_command等命令行方式绕过文件操作工具。**
```

---

### 9.6 防绕过机制

#### 9.6.1 机制说明

在 `write_file` 和 `edit_file` 工具的 Description 中增加防绕过提示，引导模型使用工具而非命令行进行文件操作。

#### 9.6.2 防绕过提示内容

在两个工具的Description末尾追加以下文本：

```
**重要：必须使用此工具进行文件操作，不应通过execute_command等命令行方式绕过文件操作工具。**
```

#### 9.6.3 语法检查结果中的提示

当发现语法错误时，在返回结果的 `message` 字段中追加：

```
Please fix the syntax errors immediately by using edit_file or write_file tool.
```

---

### 9.7 非功能需求

#### 9.7.1 性能需求

| 需求 | 指标 | 说明 |
|------|------|------|
| Go原生检查延迟 | < 50ms（1MB文件） | JSON/YAML/XML/HTML等标准库解析器 |
| 外部工具检查延迟 | < 5s（含启动开销） | node/python/javac等外部进程 |
| 语法检查超时 | 5秒硬超时 | 超时后标记降级，不影响文件操作 |
| 内存占用 | < 50MB | 解析大文件时的内存限制 |
| 文件大小限制 | 默认10MB | 超过限制跳过检查，可配置 |

#### 9.7.2 降级策略

```
降级优先级链（以JavaScript为例）：

1. [首选] 外部工具 node --check
   ↓ node不可用
2. [降级] 跳过检查，返回 degraded=true
   ↓
   返回结果包含 degraded_reason="external tool 'node' is not available"

降级关键规则：
- 降级不影响文件操作本身（写入/编辑始终执行）
- 降级状态仅记录在返回值的 syntax_check 中
- 外部工具可用性在整个进程生命周期内缓存（首次检测后不再重试）
- 不向模型报告降级为错误，仅作为信息提示
```

#### 9.7.3 可靠性需求

| 需求 | 说明 |
|------|------|
| 文件操作原子性 | 语法检查不影响写入操作，写入失败时不执行检查 |
| 临时文件清理 | 外部检查器使用的临时文件必须在检查完成后清理（使用 `defer`） |
| 并发安全 | 检查器注册表需支持并发访问 |
| 错误隔离 | 语法检查器的panic必须被recover，不能导致工具执行崩溃 |

#### 9.7.4 可扩展性需求

| 需求 | 说明 |
|------|------|
| 检查器插件化 | `SyntaxChecker` 接口设计允许独立开发和注册新检查器 |
| 检查器注册 | 通过 `SyntaxCheckDispatcher.RegisterChecker()` 动态注册 |
| 配置化 | 可通过配置文件启用/禁用特定语言的检查器 |
| 检查器优先级 | 同一扩展名可注册多个检查器，按优先级执行 |

#### 9.7.5 兼容性需求

| 需求 | 说明 |
|------|------|
| Go版本 | 项目使用 Go 1.25，新代码需兼容 |
| 现有API | `write_file` 的输出结构为增量扩展，不删除/修改已有字段 |
| Verifier兼容 | 新增语法检查不影响现有 `verifier.go` 中的JS/HTML检查逻辑 |
| 跨平台 | Windows/Linux/macOS 均需支持，外部工具命令需适配平台差异 |

---

### 9.8 与现有系统的集成

#### 9.8.1 工具注册集成

在 `filesystem.go` 的 `RegisterFilesystemToolsWithWorkDir()` 函数中，新增 `EditFileTool` 的注册：

```go
func RegisterFilesystemToolsWithWorkDir(registry *ToolRegistry, workDir string) error {
    toolList := []Tool{
        NewReadFileToolWithWorkDir(workDir),
        NewWriteFileToolWithWorkDir(workDir),   // 行为变更：增加语法检查
        NewEditFileToolWithWorkDir(workDir),     // 新增工具
        NewListDirectoryToolWithWorkDir(workDir),
        NewDeleteFileToolWithWorkDir(workDir),
        NewCreateDirectoryToolWithWorkDir(workDir),
    }
    // ...
}
```

#### 9.8.2 语法检查调度器初始化

语法检查调度器在工具初始化时创建，通过依赖注入传递给 `WriteFileTool` 和 `EditFileTool`：

```go
// 在工具初始化时创建调度器
dispatcher := syntax.NewDispatcher()
// 注册所有原生检查器
dispatcher.RegisterChecker(syntax.NewJSONChecker())
dispatcher.RegisterChecker(syntax.NewYAMLChecker())
dispatcher.RegisterChecker(syntax.NewXMLChecker())
// ...
// 将调度器注入到需要语法检查的工具中
writeTool := NewWriteFileToolWithWorkDirAndSyntaxChecker(workDir, dispatcher)
editTool := NewEditFileToolWithWorkDirAndSyntaxChecker(workDir, dispatcher)
```

#### 9.8.3 ToolResult扩展

语法检查结果嵌入 `ToolResult.Data` 的 `map[string]interface{}` 中，无需修改 `ToolResult` 结构体定义。

---

### 9.9 架构决策记录（新增）

| 编号 | 决策 | 原因 | 关联 |
|------|------|------|------|
| ADR-007 | 语法检查结果嵌入ToolResult.Data | 不修改ToolResult结构体定义，保持向后兼容 | SYNTAX-001 |
| ADR-008 | Go原生优先、外部工具降级 | 避免强制依赖外部工具，保证基本功能可用 | SYNTAX-001 |
| ADR-009 | 外部工具可用性进程级缓存 | 避免每次调用都检测工具可用性，减少开销 | SYNTAX-001 |
| ADR-010 | 独立syntax子包 | 与工具层解耦，便于独立测试和扩展 | SYNTAX-001 |
| ADR-011 | edit_file双模式设计（行号/字符串替换） | 兼顾精确编辑（行号）和安全编辑（字符串匹配防歧义）两种场景 | SYNTAX-001 |
| ADR-012 | 语法检查不影响写入操作 | 写入成功是首要目标，语法检查是辅助反馈 | SYNTAX-001 |

---

### 9.10 实施建议

#### 9.10.1 实施优先级

| 阶段 | 内容 | 优先级 | 预计工时 |
|------|------|--------|---------|
| P0-1 | `SyntaxCheckResult` 数据结构 + 调度器框架 + 检查器注册表 | P0 | 2h |
| P0-2 | JSON/YAML/XML 原生检查器（标准库，无额外依赖） | P0 | 2h |
| P0-3 | HTML 原生检查器（需新增 `golang.org/x/net` 依赖） | P0 | 2h |
| P0-4 | 外部工具检查器框架 + `node --check` JS检查 | P0 | 3h |
| P0-5 | `edit_file` 工具实现 | P0 | 3h |
| P0-6 | `write_file` 集成语法检查 + Description更新 | P0 | 1h |
| P1-1 | Go/TOML/CSS/Properties/SQL 原生检查器 | P1 | 3h |
| P1-2 | Python/Java/TypeScript 外部检查器 | P1 | 2h |
| P2-1 | Kotlin/Rust/Shell/PowerShell/BAT/Gradle 外部检查器 | P2 | 3h |
| P2-2 | HTML中嵌入JS的联合检查 | P2 | 2h |

#### 9.10.2 新增Go依赖清单

| 依赖 | 用途 | 当前状态 |
|------|------|---------|
| `golang.org/x/net/html` | HTML DOM解析 | 需新增 |
| `github.com/BurntSushi/toml` | TOML解析 | 需新增 |
| `github.com/tdewolff/parse/v2` | CSS Token解析 | 需新增（P2阶段） |

---

### 9.11 风险评估

| 风险ID | 描述 | 严重程度 | 缓解措施 |
|--------|------|---------|---------|
| R-001 | 外部工具版本差异导致检查结果不一致 | 中 | 明确工具版本要求，降级策略兜底 |
| R-002 | 大文件语法检查耗时过长影响用户体验 | 中 | 文件大小限制 + 超时控制 |
| R-003 | Java/Kotlin编译需要完整类路径上下文 | 高 | 仅做语法级检查，不做语义/类型检查 |
| R-004 | 临时文件泄露（外部工具检查器） | 中 | 使用 `defer os.Remove()` 确保清理 |
| R-005 | Windows平台下Bash不可用 | 低 | PowerShell作为备选，最终降级跳过 |
| R-006 | 模型忽略语法错误提示 | 中 | Description中强调，错误消息明确引导 |
| R-007 | 新增依赖增加构建复杂度 | 低 | 仅3个轻量级依赖，均为成熟库 |

---

## 变更日志

| 日期 | 变更类型 | 变更描述 | 变更人 |
|------|---------|---------|--------|
| 2026-04-12 | 新建 | 初版需求规格说明书，基于源码逆向分析生成 | requirement-analyst |
| 2026-04-12 | 新增 | 新增第九章：文件语法检查系统需求规格（SYNTAX-001），包含语法检查器模块设计、各语言检查策略、edit_file工具规格、write_file行为变更、防绕过机制、非功能需求 | requirement-analyst |
