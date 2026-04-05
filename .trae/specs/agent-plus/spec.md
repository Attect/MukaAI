# Agent Plus 开发工具规格文档

## Why

当前AI Agent工具完全由模型决策运行，存在以下局限性：
- 模型容易偏离基础方向
- 模型可能编造内容或产生错误
- 长链路任务容易跑偏
- 缺乏外部监督机制

Agent Plus通过"程序逻辑+AGENT"共同决策机制，在模型输出后由程序进行审查、纠偏和监督，引导模型完成复杂软件项目开发。

## What Changes

### 核心架构
- **双层决策系统**：程序逻辑层 + 模型推理层共同决策
- **YAML状态管理**：使用结构化YAML文档维护任务状态，避免Markdown格式自由导致的混乱
- **子代理Fork机制**：身份切换式子代理调用，保持上下文清晰
- **监督团队**：对话外部检查对话内容，确保质量

### 功能模块
- 模型服务连接模块（OpenAI API兼容）
- 基础工具集（文件读写、命令执行）
- 状态管理引擎（YAML解析与维护）
- 子代理调度器
- 团队定义与角色管理
- 上下文压缩优化
- 测试验证系统

## Impact

- Affected specs: 全新项目，无现有规格影响
- Affected code: 
  - `cmd/` - 命令行入口
  - `internal/agent/` - Agent核心逻辑
  - `internal/tools/` - 工具实现
  - `internal/model/` - 模型连接
  - `internal/state/` - 状态管理
  - `internal/supervisor/` - 监督系统
  - `internal/team/` - 团队定义
  - `project/` - 测试题目

## ADDED Requirements

### Requirement: 模型服务连接

系统应提供与OpenAI API兼容的模型服务连接能力。

#### Scenario: 成功连接模型
- **WHEN** 系统启动并配置模型服务地址http://127.0.0.1:11453/v1/
- **THEN** 系统能够成功建立连接并进行推理请求

#### Scenario: 模型参数配置
- **WHEN** 系统配置模型参数
- **THEN** 支持配置模型名称(Huihui-Qwen3.5-27B-abliterated.Q4_K_M)、API Key(no-key)、上下文大小(200k)

### Requirement: 基础工具集

系统应提供基础操作工具供Agent调用。

#### Scenario: 文件读写工具
- **WHEN** Agent需要操作文件系统
- **THEN** 提供读取文件、写入文件、列出目录、删除文件等工具

#### Scenario: 命令执行工具
- **WHEN** Agent需要执行系统命令
- **THEN** 提供命令执行工具，返回执行结果

### Requirement: YAML状态管理

系统应使用YAML格式维护任务状态，确保长链路工作不跑偏。

#### Scenario: 状态文档初始化
- **WHEN** 用户提出新任务
- **THEN** 系统创建YAML状态文档，记录任务目标、当前进度、待办事项

#### Scenario: 状态实时更新
- **WHEN** Agent完成某个步骤
- **THEN** 系统自动更新YAML文档中的进度状态

#### Scenario: 状态恢复
- **WHEN** Agent需要继续之前的工作
- **THEN** 系统能够从YAML文档恢复上下文状态

### Requirement: 程序逻辑审查

系统应在模型输出后进行程序逻辑审查。

#### Scenario: 方向偏离检测
- **WHEN** 模型输出内容偏离基础方向
- **THEN** 程序逻辑检测并引导模型回归正确方向

#### Scenario: 错误纠正
- **WHEN** 模型输出包含错误
- **THEN** 程序逻辑检测并提示模型纠正

#### Scenario: 编造内容检测
- **WHEN** 模型编造不存在的内容
- **THEN** 程序逻辑检测并要求模型验证或更正

### Requirement: 子代理Fork机制

系统应支持子代理调用，通过身份切换保持上下文清晰。

#### Scenario: 子代理调用
- **WHEN** 主Agent需要委派子任务
- **THEN** 系统创建子代理会话，明确身份切换提示

#### Scenario: 子代理完成
- **WHEN** 子代理完成任务
- **THEN** 系统返回总结给主Agent，主Agent继续原任务

#### Scenario: 上下文隔离
- **WHEN** 子代理执行过程中
- **THEN** 子代理的详细输出不污染主Agent上下文

### Requirement: 团队定义与角色

系统应定义Agent团队及其职责。

#### Scenario: 团队成员定义
- **WHEN** 系统初始化
- **THEN** 定义各Agent角色：架构师、开发者、测试者、审查者等

#### Scenario: 角色职责执行
- **WHEN** 任务需要特定角色
- **THEN** 对应Agent按职责执行任务

### Requirement: 监督团队

系统应提供监督团队在对话外部检查对话内容。

#### Scenario: 实时监督
- **WHEN** Agent执行任务过程中
- **THEN** 监督Agent并行检查输出质量

#### Scenario: 监督干预
- **WHEN** 监督Agent发现问题
- **THEN** 能够中断或引导当前Agent行为

### Requirement: 高效执行模式

Agent Plus应高效执行，避免冗余输出。

#### Scenario: 直接执行
- **WHEN** Agent收到任务
- **THEN** 立即调用工具执行操作，不奉承、不评价、不出报告

#### Scenario: 状态维护
- **WHEN** Agent执行过程中
- **THEN** 所有动态仅维护YAML状态文档

### Requirement: 上下文压缩

系统应结合Agent行为特点进行上下文压缩优化。

#### Scenario: 状态摘要压缩
- **WHEN** 上下文接近限制
- **THEN** 基于YAML状态生成摘要，压缩历史对话

#### Scenario: 关键信息保留
- **WHEN** 进行上下文压缩
- **THEN** 保留关键决策、工具调用结果等重要信息

### Requirement: 测试验证系统

系统应提供测试题目验证Agent Plus能力。

#### Scenario: Kotlin测试题目
- **WHEN** 运行Kotlin测试
- **THEN** Agent Plus能够完成复杂的Kotlin编程任务

#### Scenario: Java测试题目
- **WHEN** 运行Java测试
- **THEN** Agent Plus能够完成复杂的Java编程任务

#### Scenario: JavaScript测试题目
- **WHEN** 运行JavaScript测试
- **THEN** Agent Plus能够完成复杂的JavaScript编程任务

## 技术规格

### 模型配置
```yaml
model:
  endpoint: "http://127.0.0.1:11453/v1/"
  api_key: "no-key"
  model_name: "Huihui-Qwen3.5-27B-abliterated.Q4_K_M"
  context_size: 200000
```

### 项目结构
```
AgentPlus/
├── cmd/
│   └── agentplus/        # 命令行入口
├── internal/
│   ├── agent/            # Agent核心逻辑
│   │   ├── core.go       # Agent主循环
│   │   ├── reviewer.go   # 程序逻辑审查
│   │   └── fork.go       # 子代理Fork
│   ├── model/
│   │   ├── client.go     # 模型客户端
│   │   └── message.go    # 消息结构
│   ├── tools/
│   │   ├── filesystem.go # 文件系统工具
│   │   ├── command.go    # 命令执行工具
│   │   └── registry.go   # 工具注册
│   ├── state/
│   │   ├── yaml.go       # YAML状态管理
│   │   └── task.go       # 任务状态
│   ├── supervisor/
│   │   └── monitor.go    # 监督系统
│   └── team/
│       ├── definition.go # 团队定义
│       └── roles.go      # 角色管理
├── project/
│   ├── kotlin/           # Kotlin测试题目
│   ├── java/             # Java测试题目
│   └── javascript/       # JavaScript测试题目
├── configs/
│   └── config.yaml       # 配置文件
├── go.mod
└── go.sum
```

### Agent团队定义

| 角色 | 职责 |
|------|------|
| Orchestrator | 主控Agent，协调任务流程，维护YAML状态 |
| Architect | 架构设计，技术选型，模块划分 |
| Developer | 代码实现，功能开发 |
| Tester | 测试编写，验证功能 |
| Reviewer | 代码审查，质量把控 |
| Supervisor | 监督Agent，检查输出质量，纠偏 |

### YAML状态文档格式

```yaml
task:
  id: "task-001"
  goal: "任务目标描述"
  status: "in_progress"
  created_at: "2026-04-06T10:00:00Z"
  updated_at: "2026-04-06T10:30:00Z"

progress:
  current_phase: "implementation"
  completed_steps:
    - "需求分析完成"
    - "架构设计完成"
  pending_steps:
    - "代码实现"
    - "测试验证"

context:
  decisions:
    - "选择使用Go标准库实现HTTP客户端"
  constraints:
    - "不引入外部依赖"

agents:
  active: "Developer"
  history:
    - role: "Architect"
      summary: "完成架构设计"
      duration: "5m"
```

### 程序逻辑审查规则

1. **方向检查**：验证输出是否与YAML中记录的目标一致
2. **工具调用验证**：检查工具调用参数是否合理
3. **编造检测**：验证声称存在的文件/函数是否真实存在
4. **错误模式识别**：识别常见错误模式（无限循环、重复操作等）
5. **进度验证**：检查是否真正推进任务进度

### 子代理Fork流程

```
主Agent -> [身份切换提示] -> 子代理会话 -> [执行任务] -> [生成总结] -> 主Agent继续
```

身份切换提示模板：
```
接下来我转变身份为{role}，需要进行{task}操作。
```

完成提示模板：
```
我以{role}身份的操作结束，现在继续任务，检查yaml。
```
