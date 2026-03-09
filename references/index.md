# 项目参考索引

本索引记录了项目开发和参考使用的所有文档和源码库。

## 目录结构

```
references/
├── index.md                    # 本索引文件
├── kotlin/                     # Kotlin 相关参考
│   └── kotlin-official-docs.md # Kotlin 官方文档整理
├── ktor/                       # Ktor 相关参考
│   ├── ktor-server.md          # Ktor 服务端开发指南
│   ├── ktor-client.md          # Ktor 客户端开发指南
│   └── ktor-integration.md     # Ktor 集成方案指南
├── lm-studio/                  # LM Studio 相关参考
│   └── lm-studio-rest-api.md   # LM Studio REST API 参考指南
├── mcp/                        # MCP 相关参考
│   └── mcp-kotlin-sdk.md       # MCP Kotlin SDK 参考指南
├── kermit/                     # Kermit 相关参考
│   └── kermit-logging.md       # Kermit 日志库参考指南
├── dokka/                      # Dokka 相关参考
│   └── dokka-documentation.md  # Dokka 文档生成工具参考指南
├── multiplatform-markdown-renderer/  # Multiplatform Markdown Renderer 相关参考
│   └── multiplatform-markdown-renderer.md  # Multiplatform Markdown Renderer 参考指南
└── nanobot/                    # nanobot 相关参考
    └── nanobot-analysis.md     # nanobot 项目分析报告
```

## 参考文档列表

### Kotlin 语言

| 文件路径 | 描述 | 版本/日期 |
|---------|------|----------|
| [kotlin/kotlin-official-docs.md](kotlin/kotlin-official-docs.md) | Kotlin 官方文档整理，包含语言特性、语法、编码规范等 | 2.3.10 / 2025-12 |

**主要内容**:
- Kotlin 语言特性和版本演进
- K2 编译器特性
- 基本语法和常用惯用写法
- 编码规范和最佳实践
- 多平台开发指南

**官方来源**:
- https://kotlinlang.org/docs/
- https://kotlinlang.org/docs/basic-syntax.html
- https://kotlinlang.org/docs/coding-conventions.html
- https://kotlinlang.org/docs/idioms.html

### Ktor 框架

| 文件路径 | 描述 | 版本/日期 |
|---------|------|----------|
| [ktor/ktor-server.md](ktor/ktor-server.md) | Ktor 服务端开发指南，包含路由、插件、请求处理等 | 3.4.1+ / 2026-01 |
| [ktor/ktor-client.md](ktor/ktor-client.md) | Ktor 客户端开发指南，包含 HTTP 请求、响应处理、引擎等 | 3.4.1+ / 2026-01 |
| [ktor/ktor-integration.md](ktor/ktor-integration.md) | Ktor 集成方案指南，包含 KMP 全栈开发、RPC 集成等 | 3.4.1+ / 2026-03 |

**主要内容**:

**ktor-server.md**:
- Ktor 服务端核心概念和快速开始
- 路由定义和路径模式
- 插件系统 (CORS、认证、内容协商等)
- 请求处理和响应生成
- WebSocket 和 SSE
- 模板引擎 (Thymeleaf、Kotlinx.html)
- 最佳实践和部署

**ktor-client.md**:
- HttpClient 创建和配置
- HTTP 请求方法 (GET、POST、PUT、DELETE 等)
- 响应处理和流式数据
- 客户端插件 (ContentNegotiation、Logging、Auth 等)
- 多平台引擎配置 (JVM、Android、iOS、JS、Native)
- 多平台项目集成方案

**ktor-integration.md**:
- Kotlin Multiplatform 全栈开发架构
- 共享模块配置和数据模型共享
- Ktor 客户端与服务端集成
- SQLDelight 数据库集成 (跨平台)
- Koin 依赖注入 (多平台)
- Compose Multiplatform 客户端实现
- Kotlin RPC 集成方案
- RPC 与 REST 对比和最佳实践
- 混合集成方案 (RPC + REST + WebSocket)
- Docker 部署配置

**官方来源**:
- https://ktor.io/docs/
- https://github.com/ktorio/ktor
- https://github.com/ktorio/ktor-samples

### LM Studio

| 文件路径 | 描述 | 版本/日期 |
|---------|------|----------|
| [lm-studio/lm-studio-rest-api.md](lm-studio/lm-studio-rest-api.md) | LM Studio REST API 参考指南，包含 API 端点、SDK 使用、Kotlin 集成等 | 0.4.0+ / 2026-03 |

**主要内容**:
- LM Studio REST API 概述和核心特性
- v1 API 端点列表和使用方法
- 推理端点对比 (原生 vs OpenAI vs Anthropic)
- 快速开始和安装指南
- Python/TypeScript SDK 使用示例
- REST API 详细用法 (聊天、模型管理等)
- 模型管理和命令行工具
- 认证配置
- Kotlin 集成方案 (Ktor 客户端、OpenAI 兼容模式)
- 最佳实践和性能优化
- 无头部署 (llmster)
- 常见问题和解决方案

**官方来源**:
- https://lmstudio.ai/docs/developer/rest
- https://lmstudio.ai/docs/developer
- https://github.com/lmstudio-ai/lmstudio-js
- https://github.com/lmstudio-ai/lmstudio-python

### MCP (Model Context Protocol)

| 文件路径 | 描述 | 版本/日期 |
|---------|------|----------|
| [mcp/mcp-kotlin-sdk.md](mcp/mcp-kotlin-sdk.md) | MCP Kotlin SDK 参考指南，包含协议介绍、安装配置、快速开始、核心概念等 | 最新 / 2026-03 |

**主要内容**:
- MCP 协议概述和核心架构
- MCP 原语 (Prompts、Resources、Tools、Sampling)
- 能力系统 (服务端能力、客户端能力)
- 安装配置 (JVM、Kotlin Multiplatform)
- 快速开始 (客户端、服务端创建)
- 服务器功能 (提示、资源、工具、完成、日志、分页)
- 客户端功能 (根目录、采样)
- 传输层 (STDIO、Streamable HTTP、SSE、WebSocket、ChannelTransport)
- 完整示例 (天气查询服务器、客户端、测试)
- 最佳实践 (错误处理、进度报告、资源管理、日志记录、安全性、性能优化)
- 与 LM Studio 集成方案
- 常见问题和解决方案

**官方来源**:
- https://github.com/modelcontextprotocol/kotlin-sdk
- https://modelcontextprotocol.io/docs
- https://spec.modelcontextprotocol.io/

### Kermit 日志库

| 文件路径 | 描述 | 版本/日期 |
|---------|------|----------|
| [kermit/kermit-logging.md](kermit/kermit-logging.md) | Kermit Kotlin Multiplatform 日志库参考指南，包含快速开始、核心概念、配置指南等 | 2.0.4+ / 2026-03 |

**主要内容**:
- Kermit 日志库概述和特性
- 快速开始 (依赖配置、基本使用)
- 核心概念 (Logger、LogWriter、Severity)
- 日志级别详解 (Verbose、Debug、Info、Warning、Error、Assert)
- 使用方法 (lambda 语法、异常处理、标签管理)
- 标签管理策略 (类名标签、层级标签、最佳实践)
- 配置指南 (日志级别、LogWriter、环境配置、消息格式化)
- 自定义 LogWriter (文件输出、网络日志、Crashlytics 集成)
- 多平台配置 (Android、iOS、JS、JVM、Native)
- 非 Kotlin 环境调用 (kermit-simple、Swift、JS)
- 最佳实践 (性能优化、结构化日志、日志轮转)
- 与项目集成 (KMP 项目、Ktor、SQLDelight)
- 常见问题和解决方案

**官方来源**:
- https://kermit.touchlab.co/docs/
- https://github.com/touchlab/Kermit
- https://touchlab.co/blog

### Dokka 文档生成工具

| 文件路径 | 描述 | 版本/日期 |
|---------|------|----------|
| [dokka/dokka-documentation.md](dokka/dokka-documentation.md) | Dokka Kotlin 文档生成工具参考指南，包含快速开始、配置指南、多项目配置等 | 2.1.0+ / 2026-03 |

**主要内容**:
- Dokka 概述和主要特性
- 快速开始 (插件应用、文档生成)
- 系统要求和版本兼容性
- 安装配置 (版本目录、构建缓存)
- 生成文档 (Gradle 任务、输出目录)
- 输出格式 (HTML、Javadoc)
- 单项目配置 (基础配置、高级配置)
- 多项目配置 (约定插件、手动配置、聚合文档)
- 高级配置 (可见性、包选项、外部链接、自定义样式)
- 发布 Javadoc JAR (Maven Central 发布、javadoc.io)
- Dokka 插件 (官方插件、社区插件、自定义插件)
- 最佳实践 (KDoc 注释、示例代码、自动化发布)
- 常见问题和解决方案
- 从 v1 升级到 v2 迁移指南

**官方来源**:
- https://kotlinlang.org/docs/dokka-get-started.html
- https://kotlinlang.org/docs/dokka-gradle.html
- https://github.com/Kotlin/dokka
- https://kotlinlang.org/docs/dokka-migration.html

### Multiplatform Markdown Renderer

| 文件路径 | 描述 | 版本/日期 |
|---------|------|----------|
| [multiplatform-markdown-renderer/multiplatform-markdown-renderer.md](multiplatform-markdown-renderer/multiplatform-markdown-renderer.md) | Multiplatform Markdown Renderer 参考指南，包含特性介绍、安装配置、快速开始、高级用法等 | 最新 / 2026-03 |

**主要内容**:
- Multiplatform Markdown Renderer 概述和特性
- 安装配置 (多平台、JVM、Android 依赖配置)
- 快速开始 (基础使用、Material 主题集成)
- 高级用法 (rememberMarkdownState、异步解析、状态保持)
- 大型文档懒加载 (LazyColumn 支持)
- ViewModel 中解析 Markdown
- 自定义样式 (颜色、排版、动画)
- 扩展跨度 (Extended Spans) 支持
- 自定义注解处理
- 列表顺序调整
- 自定义组件 (段落、列表等)
- 表格支持
- 图片加载 (Coil2、Coil3 集成)
- 语法高亮 (Highlights 集成)
- Kotlin 集成示例和最佳实践

**官方来源**:
- https://github.com/mikepenz/multiplatform-markdown-renderer
- https://www.jetbrains.com/compose-multiplatform/
- https://github.com/JetBrains/markdown
- https://coil-kt.github.io/coil/
- https://github.com/mikepenz/Highlights

### nanobot (AI 助理)

| 文件路径 | 描述 | 版本/日期 |
|---------|------|----------|
| [nanobot/nanobot-analysis.md](nanobot/nanobot-analysis.md) | nanobot 超轻量级 AI 助理项目分析报告，包含架构设计、工具系统、记忆系统等 | **v0.1.4.post4** / 2026-03 |

**项目概述**:
nanobot 是由香港大学数据智能实验室 (HKUDS) 开发的超轻量级个人 AI 助手框架，灵感来源于 OpenClaw。核心代码仅约 4,000 行，比 OpenClaw 少 99%。

**v0.1.4.post4 新特性** (2026-03-08):
- 🔒 **更安全的默认设置**: 修复授权绕过漏洞，默认拒绝所有访问
- 🚀 **多实例支持**: `--config` 参数，支持多机器人、多租户部署
- 💪 **MCP 可靠性增强**: SSE 传输支持，改进的取消异常处理
- 🛠️ **工具系统改进**: 参数自动类型转换，OOM 防护
- 🤖 **提供商扩展**: Azure OpenAI、阿里云编码计划、GitHub Copilot 改进
- ⏰ **Cron 加固**: 自动重载、防递归调度
- 📱 **渠道改进**: Telegram/Feishu/QQ/DingTalk/WhatsApp 等全面改进
- 🌍 **跨平台稳定**: Windows 信号处理、依赖修复

**核心特性**:
- 🪶 **超轻量级**: 核心代理代码仅约 4,000 行 (OpenClaw 的 1%)
- 🔬 **研究就绪**: 代码清晰易读，易于理解、修改和扩展
- ⚡️ **闪电快速**: 最小资源占用，更快启动，更低资源使用
- 💎 **易于使用**: 一键部署，2 分钟即可运行

**核心架构**:
- **Agent Loop**: 核心处理引擎 (LLM ↔ 工具执行)
- **Context Builder**: Prompt 构建器 (系统提示、记忆、技能)
- **Memory System**: 短期记忆 + 长期记忆 + 记忆巩固
- **Tool Registry**: 工具注册表 (文件系统、Shell、Web、MCP 等)
- **Session Manager**: 会话管理 (历史、状态持久化)
- **Channel Adapters**: 9+ 渠道适配器 (Telegram/Discord/WhatsApp/Feishu/等)
- **Provider Registry**: LLM 提供商管理 (20+ 提供商支持)

**工具系统**:
- 文件系统工具 (Read/Write/Edit/List)
- Shell 执行工具 (Exec)
- Web 工具 (Search/Fetch)
- 消息工具 (Send Message)
- 子代理工具 (Spawn)
- MCP 工具 (Model Context Protocol) - v0.1.4.post4 改进
- 定时任务工具 (Cron)

**记忆系统**:
- 短期记忆：memory/YYYY-MM-DD.md (每日日志，保留 7 天)
- 长期记忆：MEMORY.md (精选记忆，永久保存)
- 记忆巩固：自动提取重要信息到长期记忆

**技能系统**:
- SKILL.md 格式定义技能
- 自动技能发现
- 内置技能：Weather/GitHub/Tmux/Cron/Memory/Skill Creator/Summarize/ClawHub

**支持的渠道**:
- Telegram (推荐，支持语音转文字) - v0.1.4.post4: 代理修复、群组主题、流式消息
- Discord (支持群聊@提及) - v0.1.4.post4: 群组策略、附件支持
- WhatsApp (需要 Node.js bridge) - v0.1.4.post4: 媒体支持 (图片/文档/视频)
- Feishu/飞书 (WebSocket 长连接) - v0.1.4.post4: 富文本解析、Groq Whisper 转录
- Mochat/爪哇 IM (Socket.IO)
- DingTalk/钉钉 (Stream 模式) - v0.1.4.post4: 群聊支持
- Slack (Socket Mode)
- Email (IMAP/SMTP)
- QQ (botpy SDK) - v0.1.4.post4: 群消息、Markdown 发送
- Matrix/Element (支持端到端加密) - v0.1.4.post4: 媒体标准化

**LLM 提供商**:
- OpenRouter (推荐，访问所有模型)
- Anthropic (Claude 直接)
- OpenAI (GPT 直接)
- Azure OpenAI - v0.1.4.post4 新增
- DeepSeek、Groq、Gemini、MiniMax
- SiliconFlow、VolcEngine、Dashscope、Moonshot、Zhipu
- 阿里云编码计划 - v0.1.4.post4 新增
- vLLM (本地部署)
- OpenAI Codex、GitHub Copilot (OAuth) - v0.1.4.post4 改进

**安全特性**:
- 工作空间限制 (restrictToWorkspace)
- 用户白名单 (allowFrom) - v0.1.4.post4: 默认拒绝
- 路径安全检查
- 命令过滤
- 会话隔离

**部署方式**:
- 本地安装 (pip install / uv / 源码)
- Docker 部署 (Docker Compose / 纯 Docker)
- Linux Systemd 服务

**多实例支持** - v0.1.4.post4:
- `--config` 参数切换实例
- CLI agent 支持 `--workspace` / `--config`
- 独立的运行时数据
- 支持同时运行多个机器人

**分析文档内容**:
- 项目概述和与 OpenClaw 对比
- 核心架构详解 (Agent Loop/Context/Memory/Tools/Session)
- 工具系统详解 (内置工具、安全、MCP v0.1.4.post4 改进)
- 渠道系统详解 (9+ 渠道适配器，v0.1.4.post4 改进)
- 记忆系统 (短期/长期/巩固机制)
- 技能系统 (SKILL.md 格式、内置技能)
- 提供商系统 (20+ 提供商、注册表模式、v0.1.4.post4 新增)
- 配置系统 (Schema、示例)
- 心跳系统和定时任务系统 (v0.1.4.post4 加固)
- 安全与权限设计 (v0.1.4.post4 修复)
- v0.1.4.post4 新特性详解
- 项目结构和学习价值

**官方来源**:
- GitHub: https://github.com/HKUDS/nanobot
- PyPI: https://pypi.org/project/nanobot-ai/
- Discord: https://discord.gg/MnCvHqpUGB
- v0.1.4.post4 发布说明：https://github.com/HKUDS/nanobot/releases/tag/v0.1.4.post4

## 参考项目列表

### OpenClaw

| 项目路径 | 描述 | 版本/日期 |
|---------|------|----------|
| [openclaw/](openclaw/) | 个人 AI 助理项目源码，包含 Gateway 架构、多渠道支持、代理运行时等 | 2026.3.3 / 2026-03 |

**项目概述**:
OpenClaw 是一个个人 AI 助理系统，用户可以运行在自己的设备上。核心价值包括 Local-first(本地优先)、Multi-channel(多渠道 20+)、Agent-native(代理原生)、Self-hosted(自托管)、Open source(开源 MIT)。

**核心架构**:
- **Gateway Layer**: 单一控制平面，WebSocket + HTTP 混合架构，单端口多路复用
- **Agent Runtime**: 基于 pi-mono 的嵌入式代理，独立工作空间，会话管理
- **Channel Adapters**: 20+ 渠道适配器 (WhatsApp/Telegram/Discord/Slack 等)
- **Tools System**: 第一类工具系统 (browser/canvas/nodes/cron/exec 等)
- **Skills Platform**: 技能平台 (bundled/managed/workspace)
- **Plugin System**: 插件架构 (npm 分发 + 本地扩展)

**设计理念**:
- Local-first Gateway (本地优先网关)
- Security and Safe Defaults (安全与默认安全)
- Terminal-first by Design (终端优先设计)
- Plugin Architecture (插件架构)
- Multi-agent Routing (多代理路由)

**值得借鉴的设计**:
- Gateway 架构模式 (单一控制平面，多路复用)
- 代理工作空间 (AGENTS.md/SOUL.md/TOOLS.md 等)
- 工具流式传输 (减少延迟，实时反馈)
- 安全配对机制 (DM pairing，允许列表)
- 多渠道统一接口 (统一消息接口，自动分块)
- 队列转向模式 (支持用户中断，灵活对话)
- 插件化架构 (核心精简，功能插件化)
- 节点系统 (设备作为节点，移动端集成)

**分析文档**:
- [OpenClaw 项目分析报告](openclaw/analysis.md) - 详细的设计理念、架构分析、AI 助理设计模式总结
- [OpenClaw 设计模式索引](openclaw/docs/patterns/INDEX.md) - 6 个核心设计模式文档的统一索引

**设计模式文档**:
1. [代理工作空间模式](openclaw/docs/patterns/01-agent-workspace.md) - 文件化的配置和记忆系统 (1200 行)
2. [工作流传输模式](openclaw/docs/patterns/02-streaming.md) - 减少延迟的流式传输
3. [队列转向模式](openclaw/docs/patterns/03-queue-steering.md) - 支持用户中断的对话管理
4. [上下文窗口管理](openclaw/docs/patterns/04-context-window.md) - 优化 token 使用
5. [工具系统](openclaw/docs/patterns/05-tools-system.md) - 第一类工具设计
6. [技能平台](openclaw/docs/patterns/06-skills-platform.md) - 可扩展能力系统

**官方来源**:
- GitHub: https://github.com/openclaw/openclaw
- 文档：https://docs.openclaw.ai
- Discord: https://discord.gg/clawd

## 分类规则

参考文档和项目源码按照以下规则分类存放:

1. **语言相关**: 按语言名称创建子目录 (如 `kotlin/`, `java/`)
2. **框架相关**: 按框架名称创建子目录 (如 `ktor/`, `compose/`)
3. **平台相关**: 按平台名称创建子目录 (如 `android/`, `ios/`, `web/`)
4. **工具相关**: 按工具名称创建子目录 (如 `gradle/`, `docker/`)

## 更新记录

| 日期 | 操作 | 描述 |
|------|------|------|
| 2026-03-07 | 创建 | 创建参考索引和 Kotlin 官方文档整理 |
| 2026-03-07 | 更新 | 添加 Ktor 服务端、客户端、集成方案文档 |
| 2026-03-07 | 重写 | 重写 ktor-integration.md，重点添加 Kotlin Multiplatform 全栈开发和 Kotlin RPC 集成方案 |
| 2026-03-07 | 更新 | 添加 LM Studio REST API 参考指南 |
| 2026-03-07 | 更新 | 添加 MCP Kotlin SDK 参考指南 |
| 2026-03-07 | 更新 | 添加 Kermit 日志库参考指南 |
| 2026-03-07 | 更新 | 添加 Dokka 文档生成工具参考指南 |
| 2026-03-07 | 更新 | 克隆 OpenClaw 源码并创建项目分析报告 |
| 2026-03-07 | 更新 | 创建 OpenClaw 设计模式文档系统 (6 个核心模式文档) |
| 2026-03-07 | 更新 | 添加 Multiplatform Markdown Renderer 参考指南 |
| 2026-03-08 | 更新 | 克隆 nanobot 源码并创建详细分析报告 |
| 2026-03-08 | 更新 | 更新 nanobot 到 v0.1.4.post4，添加新特性详解 |

## 使用说明

1. 添加新参考文档时，应在此索引中记录
2. 参考文档应按照分类规则存放
3. 每个参考文档都应包含来源链接和版本信息
4. 参考源码库应记录仓库地址和 commit 哈希
