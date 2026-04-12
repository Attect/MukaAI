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

## 变更日志

| 日期 | 变更类型 | 变更描述 | 变更人 |
|------|---------|---------|--------|
| 2026-04-12 | 新建 | 初版需求规格说明书，基于源码逆向分析生成 | requirement-analyst |
