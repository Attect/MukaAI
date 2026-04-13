# MukaAI

**MukaAI** 是一个基于大语言模型的智能编程助手，支持 CLI 和 GUI 两种交互模式，通过工具调用实现代码编写、文件操作、命令执行等自动化任务。

🌐 **[ai.muka.app](https://ai.muka.app)**

## 特性

- **多模式交互** — CLI 命令行模式 + Wails 桌面 GUI 模式
- **工具调用系统** — 内置文件读写、命令执行、Git 操作等工具
- **MCP 协议支持** — 通过 Model Context Protocol 连接外部工具服务器
- **LSP 代码诊断** — 集成 Language Server Protocol，支持 Go/TypeScript/Python 代码诊断
- **上下文感知** — 自动索引项目文件，根据任务注入相关代码上下文
- **监督与审查** — Agent 行为监督、输出审查、成果校验三级质量保障
- **终端集成** — GUI 模式内置终端面板，实时查看工具执行过程
- **对话持久化** — GUI 模式支持多对话管理和历史记录恢复
- **流式响应** — 支持思考过程展示、工具调用实时追踪

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.25+ |
| 前端 | React 19 + TypeScript + Tailwind CSS 4 + Vite 6 |
| 桌面框架 | Wails v2 |
| 终端 | xterm.js + WebSocket |

## 快速开始

### 前置要求

- Go 1.25+
- Node.js 18+（GUI 模式构建需要）
- Wails CLI v2（GUI 模式需要，参考 [Wails 安装文档](https://wails.io/docs/gettingstarted/installation)）
- 兼容 OpenAI API 格式的 LLM 服务端点

### 配置

```bash
# 复制配置模板
cp configs/config.yaml.example configs/config.yaml

# 编辑配置，填入模型端点和 API Key
# 也支持环境变量覆盖：MUKAAI_MODEL_ENDPOINT, MUKAAI_MODEL_API_KEY 等
```

### CLI 模式

```bash
# 构建
go build -o mukaai ./cmd/agentplus

# 运行（交互式输入）
./mukaai

# 直接指定任务
./mukaai "创建一个Hello World程序"

# 指定配置和工作目录
./mukaai -c ./configs/config.yaml -w /path/to/project "分析项目结构"

# 继续已有任务
./mukaai -t task-123 "继续执行任务"
```

### GUI 模式

```bash
# 开发模式（需要 Wails CLI）
wails dev

# 生产构建
wails build
# 产物位于 build/bin/ 目录

# 或直接使用 Go 构建（需要 frontend/dist 目录）
cd frontend && npm install && npm run build && cd ..
go build -tags gui -ldflags "-w -s" -o mukaai.exe ./cmd/agentplus
```

## 配置说明

配置文件使用 YAML 格式，支持以下主要配置项：

```yaml
model:
  endpoint: "http://127.0.0.1:11453/v1/"  # LLM API 端点
  api_key: "your-api-key"                  # API 密钥
  model_name: "your-model"                 # 模型名称
  context_size: 200000                     # 上下文窗口大小

agent:
  max_iterations: 100    # 最大工具调用迭代次数
  temperature: 0.7       # 生成温度

tools:
  work_dir: "."          # 工作目录
  allow_commands: []     # 命令白名单

mcp:                     # MCP 服务器配置
  enabled: false

lsp:                     # LSP 代码诊断配置
  enabled: false
```

所有配置项均可通过环境变量覆盖，格式为 `MUKAAI_<SECTION>_<KEY>`，例如 `MUKAAI_MODEL_ENDPOINT`。

## 项目结构

```
.
├── cmd/agentplus/         # 程序入口
│   ├── main.go            # CLI 模式入口
│   ├── gui.go             # GUI 模式入口（构建标签: gui）
│   └── gui_stub.go        # GUI 模式桩实现（非 GUI 构建）
├── internal/
│   ├── agent/             # Agent 核心循环和业务逻辑
│   ├── config/            # 配置加载与管理
│   ├── context/           # 代码上下文索引与注入
│   ├── gui/               # Wails GUI 绑定层
│   ├── lsp/               # Language Server Protocol 客户端
│   ├── mcp/               # Model Context Protocol 客户端
│   ├── model/             # LLM API 客户端
│   ├── state/             # 任务状态管理
│   ├── supervisor/        # Agent 行为监督
│   ├── terminal/          # 终端管理（PTY + WebSocket）
│   ├── tools/             # 工具注册与实现
│   │   ├── git/           # Git 工具
│   │   └── syntax/        # 语法检查工具
│   └── team/              # 多 Agent 团队协作
├── frontend/              # React 前端（GUI 模式）
│   ├── src/
│   │   ├── components/    # UI 组件
│   │   ├── hooks/         # React Hooks
│   │   └── styles/        # 样式文件
│   └── wailsjs/           # Wails 生成的 JS 绑定
├── configs/               # 配置文件
│   └── config.yaml.example
├── docs/                  # 文档
├── go.mod
├── wails.json
└── README.md
```

## 许可证

本项目采用专有许可证，详见 LICENSE 文件。

Copyright (c) 2026 Attect
