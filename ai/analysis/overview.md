# AgentPlus 项目概览

> 分析时间: 2026-04-11
> 分析者: project-analyzer
> 项目版本: v1.0.0

---

## 一、项目基本信息

| 项目属性 | 值 |
|---------|-----|
| 项目名称 | AgentPlus |
| 项目类型 | AI Agent 桌面应用（CLI + GUI） |
| 编程语言 | Go 1.25.0 + TypeScript/React |
| UI框架 | Wails v2.12.0 |
| 前端框架 | React 19 + Vite 6 + TailwindCSS 4 |
| 配置格式 | YAML |
| 状态管理 | YAML文件持久化 |
| 模型协议 | OpenAI Chat Completion API（兼容） |
| 目标平台 | Windows（当前）/ 跨平台（Wails支持） |

---

## 二、技术栈清单

### 后端（Go）

| 依赖 | 版本 | 用途 |
|------|------|------|
| `wailsapp/wails/v2` | v2.12.0 | 桌面应用框架，Go后端 + Web前端 |
| `gorilla/websocket` | v1.5.3 | WebSocket通信 |
| `labstack/echo/v4` | v4.13.3 | HTTP框架 |
| `google/uuid` | v1.6.0 | UUID生成 |
| `samber/lo` | v1.49.1 | 泛型工具库 |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML解析 |
| `go-toast/v2` | v2.0.3 | 系统通知 |

### 前端（TypeScript/React）

| 依赖 | 版本 | 用途 |
|------|------|------|
| `react` | ^19.0.0 | UI框架 |
| `react-dom` | ^19.0.0 | DOM渲染 |
| `react-markdown` | ^9.0.0 | Markdown渲染 |
| `remark-gfm` | ^4.0.0 | GitHub风格Markdown |
| `rehype-highlight` | ^7.0.0 | 代码高亮 |
| `highlight.js` | ^11.0.0 | 代码高亮引擎 |
| `tailwindcss` | ^4.0.0 | CSS框架 |
| `vite` | ^6.0.0 | 构建工具 |
| `typescript` | ^5.0.0 | 类型系统 |

---

## 三、目录结构

```
AgentPlus/
├── cmd/
│   └── agentplus/
│       └── main.go              # 程序入口（CLI + GUI 双模式）
├── configs/
│   └── config.yaml              # 应用配置文件
├── internal/                    # 核心业务逻辑
│   ├── agent/                   # Agent核心（23个文件）
│   │   ├── core.go              # Agent主循环、Run方法
│   │   ├── executor.go          # 工具执行器
│   │   ├── prompts.go           # 系统提示词定义
│   │   ├── history.go           # 消息历史管理
│   │   ├── stream.go            # 流式消息处理器接口
│   │   ├── fork.go              # 子代理Fork机制
│   │   ├── reviewer.go          # 程序逻辑审查器
│   │   ├── verifier.go          # 成果校验器
│   │   ├── selfcorrector.go     # 自我修正器
│   │   ├── feedback.go          # 反馈处理
│   │   ├── thinking.go          # 思考标签处理
│   │   ├── compressor.go        # 上下文压缩
│   │   └── logger.go            # 运行日志记录器
│   ├── config/                  # 配置加载
│   │   └── loader.go
│   ├── gui/                     # Wails GUI绑定层
│   │   ├── app.go               # 前后端桥接、对话管理
│   │   └── stream_bridge.go     # 流式事件桥接
│   ├── model/                   # LLM模型客户端
│   │   ├── client.go            # OpenAI API兼容客户端
│   │   ├── config.go            # 模型配置
│   │   └── message.go           # 消息类型定义
│   ├── state/                   # 状态管理
│   │   ├── manager.go           # 状态管理器
│   │   ├── task.go              # 任务状态数据结构
│   │   └── yaml.go              # YAML序列化
│   ├── supervisor/              # 监督模块
│   │   └── monitor.go
│   ├── team/                    # 团队与角色管理
│   │   ├── definition.go        # 团队、角色、工作流定义
│   │   ├── roles.go             # 预定义角色（6种）
│   │   └── manager.go           # 角色管理器
│   └── tools/                   # 工具系统
│       ├── types.go             # 工具接口、Schema定义
│       ├── registry.go          # 工具注册中心
│       ├── filesystem.go        # 文件系统工具（5个）
│       ├── command.go           # 命令执行工具（2个）
│       └── state_tools.go       # 状态管理工具（4个）
├── frontend/                    # React前端
│   ├── src/
│   │   ├── App.tsx              # 主应用组件
│   │   ├── main.tsx             # 入口
│   │   ├── components/          # UI组件
│   │   ├── hooks/               # 自定义Hooks
│   │   ├── types/               # TypeScript类型
│   │   ├── styles/              # 样式文件
│   │   └── wailsRuntime.ts      # Wails运行时桥接
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
├── project/                     # 项目模板（语言模板）
│   ├── html-tools/
│   ├── java/
│   ├── javascript/
│   └── kotlin/
├── state/                       # 运行时任务状态存储（30个YAML文件）
├── logs/                        # 运行日志
├── docs/                        # 项目文档
├── go.mod / go.sum              # Go模块依赖
├── wails.json                   # Wails配置
├── frontend_assets.go           # 前端资源嵌入
└── agentplus.exe                # 编译产物
```

---

## 四、模块划分

### 核心模块（internal/）

| 模块 | 职责 | 关键文件数 |
|------|------|-----------|
| `agent` | Agent核心循环、审查、校验、修正、Fork | 23 |
| `model` | LLM模型通信（OpenAI兼容API） | 4 |
| `tools` | 工具注册与执行（文件系统、命令、状态） | 6 |
| `state` | YAML持久化状态管理 | 4 |
| `gui` | Wails GUI桥接层 | 2 |
| `team` | 角色定义、团队管理、工作流 | 5 |
| `config` | 配置加载 | 2 |
| `supervisor` | 监督监控 | 2 |

### 入口点

| 入口 | 路径 | 说明 |
|------|------|------|
| CLI模式 | `cmd/agentplus/main.go` → `runCLICommand()` | 命令行交互式Agent |
| GUI模式 | `cmd/agentplus/main.go` → `runGUICommand()` | Wails桌面GUI应用 |
| 子命令 | `agentplus gui` | 启动GUI模式 |

---

## 五、项目模板（project/）

项目包含多语言的项目模板目录，用于Agent创建新项目时的脚手架：

- `html-tools/` - HTML工具项目模板
- `java/` - Java项目模板
- `javascript/` - JavaScript项目模板
- `kotlin/` - Kotlin项目模板

---

## 六、运行时数据

| 目录/文件 | 用途 |
|-----------|------|
| `state/` | 存储Agent任务状态（YAML格式），当前有30个历史任务 |
| `logs/` | Agent运行日志 |
| `configs/config.yaml` | 运行配置（模型端点、工具白名单等） |

### 当前配置

```yaml
model:
  endpoint: "http://127.0.0.1:11453/v1/"    # 本地模型服务
  api_key: "no-key"
  model_name: "mradermacher/Huihui-Qwen3.5-27B-abliterated-GGUF/..."  # Qwen3.5 27B量化模型
  context_size: 200000

agent:
  max_iterations: 100
  temperature: 0.7

tools:
  work_dir: "."
  allow_commands: ["go", "git", "ls", "cat", "mkdir", "rm"]
```

---

## 七、初步观察

### 架构亮点
1. **审查-校验-修正闭环**：Agent执行过程中有完整的 Reviewer → Verifier → SelfCorrector 质量闭环
2. **Fork机制**：支持主Agent创建子代理执行特定角色任务
3. **流式输出**：完整的SSE流式处理，支持思考内容分离
4. **双模式运行**：CLI和GUI共享核心逻辑

### 潜在风险
1. `state/`目录下有大量任务文件，缺少清理机制
2. 部分配置硬编码（如默认配置路径）
3. 前端组件结构需要进一步分析
4. `project/`模板目录的用途和使用方式待确认

---

## 八、待确认项

- [ ] 前端组件的具体功能和交互流程
- [ ] `supervisor`模块的完整功能（文件较少）
- [ ] `compressor`（上下文压缩）的实现状态
- [ ] `feedback`模块的用途
- [ ] 日志系统的详细配置
- [ ] 测试覆盖率和CI/CD流程
