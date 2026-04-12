# AgentPlus 部署文档

> 版本: v1.0.0
> 生成时间: 2026-04-12
> 生成者: dev-ops
> 分析方法: 构建配置文件 + 源码入口分析

---

## 一、项目概述

AgentPlus是一个基于Go + React构建的AI Agent桌面应用程序，支持CLI命令行和Wails GUI图形界面双模式运行。项目通过OpenAI兼容API与LLM模型通信，具备任务规划、工具调用、审查校验、自我修正和子代理Fork能力。

| 属性 | 值 |
|------|-----|
| 项目名称 | AgentPlus |
| 版本 | v1.0.0 |
| 后端语言 | Go 1.25.0 |
| 前端框架 | React 19 + Vite 6 + TailwindCSS 4 |
| 桌面框架 | Wails v2.12.0 |
| 目标平台 | Windows（当前）/ 跨平台（Wails支持） |
| 构建入口 | `cmd/agentplus/main.go` |

---

## 二、环境要求

### 2.1 开发环境

| 工具 | 最低版本 | 用途 | 验证命令 |
|------|---------|------|---------|
| Go | 1.25.0 | 后端编译 | `go version` |
| Node.js | 18.x+ | 前端构建 | `node --version` |
| npm | 9.x+ | 前端依赖管理 | `npm --version` |
| Wails CLI | v2.12.0 | GUI模式构建和开发 | `wails version` |

### 2.2 运行环境

| 依赖 | 说明 |
|------|------|
| LLM API服务 | 兼容OpenAI Chat Completion API的模型服务（如本地部署的vLLM、Ollama等） |
| 网络访问 | 需要能访问LLM API端点（默认`http://127.0.0.1:11453/v1/`） |
| 文件系统 | 读写权限（工具系统需要文件操作能力） |
| 命令执行 | 支持白名单内命令执行（go、git、ls等） |

### 2.3 Wails CLI安装

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

---

## 三、构建指南

### 3.1 依赖安装

#### 3.1.1 Go依赖

```bash
# 在项目根目录执行
go mod download
```

Go依赖通过`go.mod`管理，`go.sum`锁定版本。当前共37个依赖（含传递依赖）。

主要依赖清单：

| 依赖 | 版本 | 类型 | 用途 |
|------|------|------|------|
| `wailsapp/wails/v2` | v2.12.0 | 间接 | 桌面应用框架 |
| `gorilla/websocket` | v1.5.3 | 间接 | WebSocket通信 |
| `labstack/echo/v4` | v4.13.3 | 间接 | HTTP框架 |
| `google/uuid` | v1.6.0 | 间接 | UUID生成 |
| `samber/lo` | v1.49.1 | 间接 | 泛型工具库 |
| `gopkg.in/yaml.v3` | v3.0.1 | 直接 | YAML解析 |
| `go-toast/v2` | v2.0.3 | 间接 | 系统通知 |

> **注意**: `go.mod`中仅`gopkg.in/yaml.v3`为直接依赖(`require`)，其余均为间接依赖(`require ... // indirect`)。Wails框架相关依赖全部通过传递依赖引入。

#### 3.1.2 前端依赖

```bash
# 在frontend/目录执行
cd frontend && npm install
```

前端依赖通过`frontend/package.json`管理，版本锁定在`frontend/package-lock.json`。

运行时依赖：

| 包名 | 版本范围 | 用途 |
|------|---------|------|
| `react` | ^19.0.0 | UI框架 |
| `react-dom` | ^19.0.0 | DOM渲染 |
| `react-markdown` | ^9.0.0 | Markdown渲染 |
| `remark-gfm` | ^4.0.0 | GitHub风格Markdown扩展 |
| `rehype-highlight` | ^7.0.0 | 代码高亮rehype插件 |
| `highlight.js` | ^11.0.0 | 代码高亮引擎 |

开发依赖：

| 包名 | 版本范围 | 用途 |
|------|---------|------|
| `@vitejs/plugin-react` | ^4.0.0 | Vite React插件 |
| `@tailwindcss/vite` | ^4.0.0 | TailwindCSS Vite插件 |
| `tailwindcss` | ^4.0.0 | CSS框架 |
| `typescript` | ^5.0.0 | TypeScript编译器 |
| `vite` | ^6.0.0 | 前端构建工具 |
| `@types/react` | ^19.0.0 | React类型定义 |
| `@types/react-dom` | ^19.0.0 | ReactDOM类型定义 |

### 3.2 CLI模式构建

CLI模式不需要Wails构建标签，直接使用Go编译即可：

```bash
# 开发构建（带调试信息）
go build -o agentplus.exe ./cmd/agentplus

# 生产构建（优化体积）
go build -ldflags "-w -s" -o agentplus.exe ./cmd/agentplus

# 带版本信息的构建
go build -ldflags "-w -s -X main.Version=1.0.0" -o agentplus.exe ./cmd/agentplus
```

**构建产物**: `agentplus.exe`（Windows平台）

> **重要**: CLI模式构建**不需要**前端资源。`frontend_assets.go`中的`//go:embed all:frontend/dist`在CLI模式下不会触发编译错误，因为当构建标签不包含`desktop`时，Wails相关代码可能被条件编译排除。但需注意，如果`frontend/dist`目录不存在，带有embed指令的文件会导致编译失败——详见第3.4节注意事项。

### 3.3 GUI模式构建

GUI模式使用Wails CLI构建，自动处理前端构建和Go编译：

```bash
# 开发模式（热重载）
wails dev

# 生产构建
wails build

# 生产构建（指定输出）
wails build -o agentplus-gui.exe

# 清理构建后重新构建
wails build -clean
```

Wails构建流程（`wails build`执行时自动进行）：

```
1. npm install         ← 安装前端依赖（frontend/目录）
2. tsc && vite build   ← TypeScript编译 + Vite打包
3. 前端产物 → frontend/dist/
4. go:embed 指令       ← 将 frontend/dist/ 嵌入Go二进制
5. go build            ← 编译Go二进制（含嵌入的前端资源）
```

**构建产物**: `agentplus-gui.exe`（由`wails.json`的`outputfilename`决定）

### 3.4 前端独立构建

```bash
# 在frontend/目录下
cd frontend
npm run build        # 执行 tsc && vite build
```

构建产物输出到`frontend/dist/`目录。

> **注意**: `frontend_assets.go`使用了`//go:embed all:frontend/dist`指令。这意味着：
> - 如果`frontend/dist/`目录不存在，**任何Go编译操作都会失败**（包括CLI模式构建）
> - 解决方案：在首次构建CLI模式前，需要先创建`frontend/dist/`目录，或先执行一次前端构建
> - 建议在`frontend/dist/`中放置一个占位文件（如`.gitkeep`），确保embed指令不会因空目录而失败

### 3.5 跨平台构建

Wails支持跨平台构建（需安装对应平台的依赖）：

```bash
# Windows
wails build -platform windows/amd64

# Linux
wails build -platform linux/amd64

# macOS
wails build -platform darwin/universal
```

---

## 四、配置说明

### 4.1 配置文件结构

配置文件路径: `configs/config.yaml`

```yaml
# LLM模型服务配置
model:
  endpoint: "http://127.0.0.1:11453/v1/"    # API端点地址
  api_key: "no-key"                          # API密钥
  model_name: "Qwen3.5-27B"                 # 模型名称/标识
  context_size: 200000                       # 上下文窗口大小(tokens)

# Agent行为配置
agent:
  max_iterations: 100                        # 最大迭代次数
  temperature: 0.7                           # 温度参数(0-2)

# 状态管理配置
state:
  dir: "./state"                             # 状态文件存储目录
  auto_save: true                            # 是否自动保存状态

# 工具系统配置
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

### 4.2 配置加载优先级

配置按以下优先级从低到高加载：

```
1. 默认配置 (DefaultConfig())
2. 配置文件 (configs/config.yaml)
3. 环境变量覆盖
4. 命令行参数覆盖（仅CLI模式）
```

### 4.3 环境变量覆盖

所有配置项均支持环境变量覆盖，格式为`AGENTPLUS_<SECTION>_<KEY>`：

| 环境变量 | 对应配置项 | 类型 | 示例 |
|---------|-----------|------|------|
| `AGENTPLUS_MODEL_ENDPOINT` | `model.endpoint` | string | `http://localhost:8080/v1/` |
| `AGENTPLUS_MODEL_API_KEY` | `model.api_key` | string | `sk-xxxxx` |
| `AGENTPLUS_MODEL_NAME` | `model.model_name` | string | `gpt-4` |
| `AGENTPLUS_MODEL_CONTEXT_SIZE` | `model.context_size` | int | `128000` |
| `AGENTPLUS_AGENT_MAX_ITERATIONS` | `agent.max_iterations` | int | `50` |
| `AGENTPLUS_AGENT_TEMPERATURE` | `agent.temperature` | float | `0.5` |
| `AGENTPLUS_STATE_DIR` | `state.dir` | string | `/data/state` |
| `AGENTPLUS_STATE_AUTO_SAVE` | `state.auto_save` | bool | `true` |
| `AGENTPLUS_TOOLS_WORK_DIR` | `tools.work_dir` | string | `/workspace` |

### 4.4 命令行参数（CLI模式）

| 参数 | 短参数 | 默认值 | 说明 |
|------|--------|--------|------|
| `--config` | `-c` | `./configs/config.yaml` | 配置文件路径 |
| `--task` | `-t` | 无 | 继续已有任务ID |
| `--workdir` | `-w` | 配置文件值 | 工作目录 |
| `--verbose` | `-v` | false | 详细输出 |
| `--no-supervisor` | 无 | false | 禁用监督 |
| `--max-iterations` | 无 | 配置文件值 | 最大迭代次数 |

### 4.5 命令行参数（GUI模式）

通过`agentplus gui [options]`启动：

| 参数 | 短参数 | 默认值 | 说明 |
|------|--------|--------|------|
| `--config` | `-c` | `./configs/config.yaml` | 配置文件路径 |
| `--workdir` | `-w` | 配置文件值 | 工作目录 |

---

## 五、部署指南

### 5.1 CLI模式部署

#### 部署步骤

```
1. 构建: go build -ldflags "-w -s" -o agentplus.exe ./cmd/agentplus
2. 准备配置: configs/config.yaml（修改模型端点等）
3. 分发文件:
   - agentplus.exe
   - configs/config.yaml
4. 运行: agentplus.exe [选项] "任务描述"
```

#### 最小部署文件

```
agentplus/
├── agentplus.exe           # 可执行文件
├── configs/
│   └── config.yaml         # 配置文件
├── state/                  # 状态目录（自动创建）
└── logs/                   # 日志目录（自动创建）
```

#### 运行方式

```bash
# 直接运行（交互式输入任务）
agentplus.exe

# 命令行传入任务
agentplus.exe "创建一个Hello World程序"

# 指定配置和工作目录
agentplus.exe -c ./config.yaml -w ./workspace "分析项目"

# 查看版本
agentplus.exe version
```

### 5.2 GUI模式部署

#### 部署步骤

```
1. 构建: wails build
2. 分发文件:
   - agentplus-gui.exe（内嵌前端资源，单文件分发）
   - configs/config.yaml（外部配置）
3. 运行: agentplus-gui.exe
```

#### 最小部署文件

```
agentplus/
├── agentplus-gui.exe       # GUI可执行文件（内嵌前端）
├── configs/
│   └── config.yaml         # 配置文件
├── state/                  # 状态目录（自动创建）
└── logs/                   # 日志目录（自动创建）
```

#### 运行方式

```bash
# 直接启动GUI
agentplus-gui.exe

# 通过命令行启动GUI
agentplus.exe gui

# 指定配置
agentplus-gui.exe -c ./config.yaml
```

### 5.3 配置文件部署注意事项

1. **模型端点**: 必须根据部署环境修改`model.endpoint`，指向可用的LLM API服务
2. **API密钥**: 如果LLM服务需要认证，修改`model.api_key`，或通过环境变量`AGENTPLUS_MODEL_API_KEY`传入
3. **工作目录**: `tools.work_dir`决定Agent操作文件的根目录，建议设置为绝对路径
4. **命令白名单**: `tools.allow_commands`控制Agent可执行的命令，按需调整
5. **状态目录**: `state.dir`用于存储任务状态，确保有写入权限

---

## 六、构建配置文件详解

### 6.1 wails.json

```json
{
  "$schema": "https://wails.io/schemas/config.v2.json",
  "name": "AgentPlus",
  "outputfilename": "agentplus-gui",      // GUI构建产物文件名
  "frontend:install": "npm install",       // 前端依赖安装命令
  "frontend:build": "npm run build",       // 前端构建命令
  "frontend:dev:watcher": "npm run dev",   // 开发模式前端监听命令
  "frontend:dev:serverUrl": "auto",        // 开发模式Vite服务器URL（自动检测）
  "author": {
    "name": "AgentPlus"
  }
}
```

### 6.2 vite.config.ts

```typescript
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  clearScreen: false,  // Wails开发模式下不清屏
});
```

### 6.3 tsconfig.json

| 配置项 | 值 | 说明 |
|--------|-----|------|
| `target` | ES2020 | 编译目标 |
| `lib` | ES2020, DOM, DOM.Iterable | 类型库 |
| `module` | ESNext | 模块系统 |
| `moduleResolution` | bundler | 模块解析策略 |
| `jsx` | react-jsx | JSX转换模式 |
| `strict` | true | 严格模式 |
| `noEmit` | true | 不输出文件（由Vite处理） |
| `include` | src, wailsjs | 源码和Wails运行时类型 |

### 6.4 frontend_assets.go

```go
package agentplus

import "embed"

//go:embed all:frontend/dist
var FrontendAssets embed.FS
```

**关键说明**:
- 使用Go 1.16+的`embed`包将前端构建产物嵌入二进制文件
- 仅在GUI模式（Wails构建）时有效利用
- `frontend/dist/`目录必须存在，否则Go编译失败
- 开发模式下（`wails dev`），Vite开发服务器提供资源，嵌入不生效

---

## 七、依赖清单汇总

### 7.1 构建时依赖

| 依赖 | 版本要求 | 类型 | 说明 |
|------|---------|------|------|
| Go | 1.25.0+ | 工具链 | 后端编译 |
| Node.js | 18.x+ | 工具链 | 前端构建 |
| npm | 9.x+ | 工具链 | 前端包管理 |
| Wails CLI | v2.12.0 | 工具链 | GUI构建（仅GUI模式需要） |

### 7.2 Go运行时依赖

| 包 | 版本 | 说明 |
|----|------|------|
| `gopkg.in/yaml.v3` | v3.0.1 | YAML配置解析（唯一直接依赖） |
| `github.com/wailsapp/wails/v2` | v2.12.0 | 桌面框架（GUI模式） |
| `github.com/gorilla/websocket` | v1.5.3 | WebSocket |
| `github.com/labstack/echo/v4` | v4.13.3 | HTTP服务 |
| `github.com/google/uuid` | v1.6.0 | UUID生成 |
| `github.com/samber/lo` | v1.49.1 | 泛型工具 |
| `git.sr.ht/~jackmordaunt/go-toast/v2` | v2.0.3 | 系统通知 |

### 7.3 前端运行时依赖

| 包 | 版本 | 说明 |
|----|------|------|
| react | ^19.0.0 | UI框架 |
| react-dom | ^19.0.0 | DOM渲染 |
| react-markdown | ^9.0.0 | Markdown渲染 |
| remark-gfm | ^4.0.0 | GFM扩展 |
| rehype-highlight | ^7.0.0 | 代码高亮 |
| highlight.js | ^11.0.0 | 高亮引擎 |

### 7.4 外部服务依赖

| 服务 | 说明 | 默认地址 |
|------|------|---------|
| LLM API | OpenAI兼容的模型推理服务 | `http://127.0.0.1:11453/v1/` |

---

## 八、开发工作流

### 8.1 CLI模式开发

```bash
# 1. 安装Go依赖
go mod download

# 2. 直接运行（无需前端构建）
go run ./cmd/agentplus "测试任务"

# 3. 或编译后运行
go build -o agentplus.exe ./cmd/agentplus
./agentplus.exe "测试任务"
```

### 8.2 GUI模式开发

```bash
# 1. 安装前端依赖
cd frontend && npm install && cd ..

# 2. 安装Wails CLI（首次）
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 3. 启动开发模式（热重载）
wails dev

# 4. 生产构建
wails build
```

### 8.3 前端独立开发

```bash
cd frontend
npm install
npm run dev      # 启动Vite开发服务器
npm run build    # 构建生产版本
npm run preview  # 预览生产构建
```

---

## 九、.gitignore 配置分析

当前`.gitignore`配置的构建相关忽略规则：

| 规则 | 说明 | 评估 |
|------|------|------|
| `*.exe` | 忽略所有可执行文件 | 正确 |
| `*.dll` / `*.so` / `*.dylib` | 忽略动态库 | 正确 |
| `vendor/` | 忽略Go vendor目录 | 正确 |
| `agentplus` / `agentplus.exe` | 忽略构建产物 | 正确 |
| `state/` | 忽略运行时状态文件 | 正确 |
| `*.yaml` + `!configs/config.yaml` | 忽略所有YAML但保留配置模板 | 正确 |

> **注意**: `*.yaml`的忽略规则配合`!configs/config.yaml`的排除规则，确保了任务状态文件不被提交，同时保留配置模板。但需要注意，`ai/`目录下的任务YAML文件也会被忽略——如果需要版本控制任务文档，需要添加排除规则如`!ai/tasks/**/*.yaml`。

---

## 十、CI/CD状态

| 项目 | 状态 | 说明 |
|------|------|------|
| CI/CD配置 | 不存在 | 无`.github/workflows`、`Jenkinsfile`等 |
| Dockerfile | 不存在 | 无容器化构建配置 |
| Makefile | 不存在 | 无标准化构建脚本 |
| 版本发布 | 不存在 | 无`goreleaser`等发布工具配置 |

当前项目完全依赖手动构建和分发。

---

## 十一、问题与改进建议

### 11.1 当前问题

| 编号 | 问题 | 严重度 | 说明 |
|------|------|--------|------|
| P-001 | `go:embed`依赖`frontend/dist`目录 | 高 | CLI模式构建时，如果`frontend/dist/`不存在会导致编译失败，即使CLI模式不需要前端资源 |
| P-002 | 无构建脚本 | 中 | 缺少`Makefile`或类似构建脚本，构建命令依赖开发者记忆 |
| P-003 | 无CI/CD | 中 | 没有自动化构建和测试流程 |
| P-004 | `go.mod`依赖标记不准确 | 低 | 大部分直接使用的依赖被标记为`indirect`（如Wails），可能导致`go mod tidy`时行为异常 |
| P-005 | 配置文件中的敏感信息 | 中 | `config.yaml`中API密钥为明文（当前为"no-key"），如果切换到真实密钥需要额外处理 |
| P-006 | `.gitignore`忽略所有YAML | 低 | `*.yaml`规则会影响`ai/tasks/`下的任务文档的版本控制 |

### 11.2 改进建议

#### 建议1: 解决`go:embed`对CLI构建的影响

**方案A**: 在`frontend/dist/`中放置`.gitkeep`占位文件，确保目录始终存在

**方案B**: 使用构建标签(Build Tags)分离CLI和GUI代码：

```go
// frontend_assets.go (添加构建标签)
//go:build desktop

package agentplus

import "embed"

//go:embed all:frontend/dist
var FrontendAssets embed.FS
```

```go
// frontend_assets_stub.go (CLI模式桩文件)
//go:build !desktop

package agentplus

import "embed"

// FrontendAssets CLI模式下的空实现
var FrontendAssets = embed.FS{}
```

#### 建议2: 添加Makefile标准化构建

建议创建`Makefile`统一构建流程：

```makefile
.PHONY: build-cli build-gui dev clean

build-cli:
	go build -ldflags "-w -s" -o agentplus.exe ./cmd/agentplus

build-gui:
	wails build -clean

dev:
	wails dev

clean:
	rm -f agentplus.exe agentplus-gui.exe
	rm -rf frontend/dist
```

#### 建议3: 敏感信息管理

- 配置文件中的API密钥改为环境变量优先
- 在`.gitignore`中添加对包含真实密钥的配置文件的忽略规则
- 文档中说明通过环境变量传入敏感信息的方式

---

## 十二、版本管理

### 12.1 当前版本

| 属性 | 值 | 来源 |
|------|-----|------|
| 版本号 | 1.0.0 | `cmd/agentplus/main.go` 中的 `Version` 常量 |
| 产品名 | AgentPlus | `cmd/agentplus/main.go` 中的 `Name` 常量 |

### 12.2 版本号构建注入

当前版本号通过Go常量硬编码在`main.go`中。如需动态注入版本号，可使用`-ldflags`：

```bash
VERSION=$(git describe --tags --always)
go build -ldflags "-w -s -X main.Version=$VERSION" -o agentplus.exe ./cmd/agentplus
```

---

## 十三、快速参考

### 常用命令速查

```bash
# === 开发 ===
go run ./cmd/agentplus                          # CLI开发运行
wails dev                                       # GUI开发运行（热重载）
cd frontend && npm run dev                      # 前端独立开发

# === 构建 ===
go build -o agentplus.exe ./cmd/agentplus       # CLI构建
wails build                                     # GUI构建

# === 依赖 ===
go mod download                                 # 安装Go依赖
cd frontend && npm install                      # 安装前端依赖

# === 运行 ===
./agentplus.exe "任务描述"                       # CLI运行
./agentplus.exe gui                             # GUI运行
./agentplus.exe -c config.yaml -w ./workspace   # 指定配置和工作目录
./agentplus.exe version                         # 查看版本
```

### 文件分发清单

| 场景 | 文件 | 说明 |
|------|------|------|
| CLI分发 | `agentplus.exe` + `configs/config.yaml` | 可执行文件 + 配置模板 |
| GUI分发 | `agentplus-gui.exe` + `configs/config.yaml` | 单文件可执行（内嵌前端） + 配置模板 |

---

> 变更记录:
> - 2026-04-12 | dev-ops | 初始版本，基于项目源码分析生成
