# TUI 界面规格文档

## Why

当前 Agent Plus 通过命令行交互，用户体验有限，无法直观查看对话历史、token 用量、子代理状态等信息。需要实现一个基于终端的用户界面（TUI），提供更好的交互体验和信息展示。

## What Changes

- **Bubble Tea 框架集成**：使用 charmbracelet/bubbletea 构建终端用户界面
- **对话界面**：下方用户输入框，上方对话列表滚动显示
- **流式输出展示**：实时显示模型思考、正文、工具调用的流式内容
- **工具调用格式化**：流式生成时显示原文，完成后格式化为友好显示
- **Token 统计显示**：每个对话显示 token 用量，界面显示总用量和推理次数
- **工作目录管理**：显示当前工作目录，支持切换
- **子对话管理**：列表展示所有对话，标注状态（活动、等待、结束）

## Impact

- Affected specs: 无现有规格影响，新增独立模块
- Affected code:
  - `cmd/agentplus/main.go` - 添加 TUI 启动模式
  - `internal/tui/` - 新增 TUI 模块目录
  - `internal/agent/core.go` - 添加流式输出回调接口

## ADDED Requirements

### Requirement: TUI 框架集成

系统应集成 Bubble Tea 框架实现终端用户界面。

#### Scenario: 启动 TUI 模式
- **WHEN** 用户运行 `agentplus tui` 命令
- **THEN** 系统启动 TUI 界面，显示欢迎界面和输入提示

#### Scenario: Bubble Tea 组件架构
- **WHEN** TUI 启动
- **THEN** 系统使用 Bubble Tea 的 Model-Update-View 架构管理界面状态

### Requirement: 对话界面布局

系统应提供清晰的对话界面布局。

#### Scenario: 界面分区
- **WHEN** TUI 显示
- **THEN** 界面分为三个主要区域：
  - 顶部状态栏：显示工作目录、总 token 用量、推理次数
  - 中部对话区：滚动显示对话历史
  - 底部输入区：用户输入框

#### Scenario: 对话列表显示
- **WHEN** 对话历史存在
- **THEN** 对话区显示所有对话消息，支持滚动查看

#### Scenario: 用户输入
- **WHEN** 用户在输入框输入文本
- **THEN** 支持多行输入、编辑、提交

### Requirement: 流式输出实时显示

系统应实时显示模型的流式输出内容。

#### Scenario: 思考内容显示
- **WHEN** 模型输出思考内容（`<think/>` 标签内）
- **THEN** 在独立的信息块中实时显示思考内容，使用不同样式区分

#### Scenario: 正文内容显示
- **WHEN** 模型输出正文内容
- **THEN** 在独立的信息块中实时显示正文内容

#### Scenario: 工具调用显示
- **WHEN** 模型输出工具调用
- **THEN** 在独立的信息块中显示工具调用信息

#### Scenario: 流式更新
- **WHEN** 模型流式输出内容
- **THEN** 界面实时更新显示，无需等待完整响应

### Requirement: 工具调用格式化

系统应对工具调用进行友好格式化显示。

#### Scenario: 流式生成中
- **WHEN** 工具调用正在流式生成
- **THEN** 显示原始 JSON 文本

#### Scenario: 工具调用完成
- **WHEN** 工具调用流式生成完成
- **THEN** 格式化显示为友好格式：
  - 工具名称
  - 参数列表（格式化 JSON）
  - 执行结果（如果有）

#### Scenario: 工具响应显示
- **WHEN** 工具执行完成返回结果
- **THEN** 显示工具执行结果，支持折叠/展开

### Requirement: Token 用量统计

系统应显示 token 用量统计信息。

#### Scenario: 单次对话 token 显示
- **WHEN** 一次对话完成
- **THEN** 在对话消息下方显示本次对话的 token 用量

#### Scenario: 总 token 用量显示
- **WHEN** TUI 运行中
- **THEN** 在顶部状态栏显示本次会话的总 token 用量

#### Scenario: 推理次数显示
- **WHEN** TUI 运行中
- **THEN** 在顶部状态栏显示本次会话的推理次数

### Requirement: 工作目录管理

系统应提供工作目录显示和切换功能。

#### Scenario: 当前目录显示
- **WHEN** TUI 显示
- **THEN** 在顶部状态栏显示当前工作目录

#### Scenario: 目录切换命令
- **WHEN** 用户输入 `/cd <path>` 命令
- **THEN** 切换工作目录到指定路径

#### Scenario: 目录切换确认
- **WHEN** 目录切换成功
- **THEN** 显示切换成功提示，更新状态栏显示

### Requirement: 子对话管理

系统应提供子代理和子对话的管理界面。

#### Scenario: 对话列表查看
- **WHEN** 用户输入 `/conversations` 命令或按快捷键
- **THEN** 显示所有对话列表，包括主对话和子对话

#### Scenario: 对话状态标注
- **WHEN** 显示对话列表
- **THEN** 每个对话标注状态：
  - 活动（正在推理）
  - 等待（等待子代理完成）
  - 结束（对话内容结束）

#### Scenario: 对话时间排序
- **WHEN** 显示对话列表
- **THEN** 按创建时间排序，最新的在前

#### Scenario: 对话切换
- **WHEN** 用户选择某个对话
- **THEN** 切换到该对话视图，显示该对话的内容

### Requirement: 快捷键支持

系统应支持常用快捷键操作。

#### Scenario: 提交输入
- **WHEN** 用户按下 `Enter` 键（单行模式）或 `Ctrl+Enter`（多行模式）
- **THEN** 提交用户输入

#### Scenario: 切换输入模式
- **WHEN** 用户按下 `Tab` 键
- **THEN** 在单行输入和多行输入模式间切换

#### Scenario: 查看对话列表
- **WHEN** 用户按下 `Ctrl+L` 键
- **THEN** 显示对话列表界面

#### Scenario: 退出 TUI
- **WHEN** 用户按下 `Ctrl+C` 或 `Esc` 键
- **THEN** 退出 TUI 界面

### Requirement: 样式与主题

系统应提供清晰的视觉样式。

#### Scenario: 消息类型样式
- **WHEN** 显示不同类型的消息
- **THEN** 使用不同颜色和样式区分：
  - 用户消息：蓝色
  - 模型思考：灰色斜体
  - 模型正文：默认样式
  - 工具调用：黄色
  - 工具响应：绿色
  - 错误信息：红色

#### Scenario: 状态指示
- **WHEN** 显示对话状态
- **THEN** 使用不同符号和颜色：
  - 活动：🔄 绿色
  - 等待：⏳ 黄色
  - 结束：✓ 灰色

## 技术规格

### 依赖库

```go
require (
    charm.land/bubbletea v2.x.x
    charm.land/bubbles v0.x.x  // UI 组件库
    charm.space/lipgloss v0.x.x  // 样式库
)
```

### 项目结构

```
internal/tui/
├── app.go           # TUI 主应用 Model
├── components/
│   ├── input.go     # 输入组件
│   ├── chat.go      # 对话显示组件
│   ├── statusbar.go # 状态栏组件
│   └── dialog.go    # 对话列表弹窗组件
├── messages.go      # Bubble Tea 消息定义
├── styles.go        # 样式定义
└── formatter.go     # 工具调用格式化
```

### Bubble Tea Model 结构

```go
type AppModel struct {
    // 状态
    currentDir    string
    totalTokens   int
    inferenceCount int
    
    // 对话管理
    conversations []*Conversation
    activeConv    *Conversation
    
    // UI 组件
    input         textinput.Model
    chatView      viewport.Model
    statusBar     StatusBar
    
    // 状态标志
    isStreaming   bool
    showConvList  bool
    inputMode     InputMode  // single-line / multi-line
}

type Conversation struct {
    id          string
    createdAt   time.Time
    status      ConvStatus  // active / waiting / finished
    messages    []Message
    tokenUsage  int
}

type Message struct {
    role        MessageRole  // user / assistant / tool
    content     string
    thinking    string
    toolCalls   []ToolCall
    tokenUsage  int
    timestamp   time.Time
}
```

### 消息流处理

```go
// 流式消息回调接口
type StreamHandler interface {
    OnThinking(chunk string)
    OnContent(chunk string)
    OnToolCall(call ToolCall, isComplete bool)
    OnToolResult(result ToolResult)
    OnComplete(usage TokenUsage)
}

// TUI 实现流式处理
func (m *AppModel) OnThinking(chunk string) {
    // 更新当前消息的思考内容
    m.activeConv.currentMessage.thinking += chunk
    // 触发 UI 更新
    m.updateChatView()
}
```

### 界面布局

```
┌─────────────────────────────────────────────────────────┐
│ 📁 /path/to/workdir  │  Tokens: 12345  │  Inferences: 5 │ <- 状态栏
├─────────────────────────────────────────────────────────┤
│                                                         │
│ User: 请帮我创建一个新功能                              │
│                                                         │
│ Assistant:                                              │
│ ┌─ Thinking ─────────────────────────────────────────┐ │
│ │ 我需要分析用户的需求...                            │ │
│ └────────────────────────────────────────────────────┘ │
│                                                         │
│ 好的，我将帮你创建新功能...                            │
│                                                         │
│ ┌─ Tool Call: create_file ──────────────────────────┐ │
│ │ path: /path/to/file.go                            │ │
│ │ content: package main...                          │ │
│ └────────────────────────────────────────────────────┘ │
│                                                         │
│ ┌─ Tool Result ─────────────────────────────────────┐ │
│ │ ✓ 文件创建成功                                    │ │
│ └────────────────────────────────────────────────────┘ │
│                                                         │
│ Tokens: 234                                             │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ > 请输入你的问题... _                                  │ <- 输入框
└─────────────────────────────────────────────────────────┘
```

### 对话列表界面

```
┌─ Conversations ─────────────────────────────────────────┐
│                                                         │
│ 🔄 [Active]   Main Conversation        2026-04-08 10:30│
│ ⏳ [Waiting]  Sub-agent: Developer     2026-04-08 10:28│
│ ✓  [Finished] Sub-agent: Architect    2026-04-08 10:25│
│ ✓  [Finished] Main Conversation       2026-04-08 10:20│
│                                                         │
│ [Enter] Select  [Esc] Close                            │
└─────────────────────────────────────────────────────────┘
```

### 工具调用格式化示例

流式生成中：
```
{"name": "create_file", "arguments": "{\"path\": \"/path/to/file.go\", \"content\": \"package main...\""}
```

格式化后：
```
┌─ Tool: create_file ─────────────────────────────────────┐
│ Parameters:                                             │
│   path: /path/to/file.go                               │
│   content:                                             │
│     package main                                       │
│     ...                                                │
└─────────────────────────────────────────────────────────┘
```

### 命令行参数

```bash
# 启动 TUI 模式
agentplus tui

# 指定初始工作目录
agentplus tui --dir /path/to/project

# 加载历史对话
agentplus tui --load conversation.yaml
```

### 内置命令

在 TUI 输入框中支持的命令：

- `/cd <path>` - 切换工作目录
- `/conversations` 或 `/conv` - 显示对话列表
- `/clear` - 清空当前对话
- `/save [file]` - 保存对话历史
- `/help` - 显示帮助信息
- `/exit` - 退出 TUI

### 性能要求

- 流式输出延迟 < 100ms
- 滚动流畅，无明显卡顿
- 支持 1000+ 条消息历史
- 内存占用 < 100MB（正常使用场景）
