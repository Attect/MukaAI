# Model Context Protocol (MCP) Kotlin SDK 参考指南

> **版本**: 最新 (2026-03)  
> **更新日期**: 2026-03-07  
> **官方仓库**: https://github.com/modelcontextprotocol/kotlin-sdk  
> **官方文档**: https://modelcontextprotocol.io/docs

---

## 目录

- [概述](#概述)
- [什么是 MCP](#什么是-mcp)
- [核心架构](#核心架构)
- [安装配置](#安装配置)
- [快速开始](#快速开始)
- [核心概念](#核心概念)
- [服务器功能](#服务器功能)
- [客户端功能](#客户端功能)
- [传输层](#传输层)
- [完整示例](#完整示例)
- [最佳实践](#最佳实践)
- [与 LM Studio 集成](#与-lm-studio-集成)
- [常见问题](#常见问题)

---

## 概述

Model Context Protocol (MCP) Kotlin SDK 是 Model Context Protocol 的官方 Kotlin 实现，支持 Kotlin Multiplatform，可以 targeting JVM、Native、JS 和 Wasm 平台。

### 主要特性

- **跨平台支持**: 单个代码库支持 JVM、Native、JS、Wasm
- **客户端和服务端**: 完整的 MCP 客户端和服务端实现
- **多种传输层**: stdio、SSE、Streamable HTTP、WebSocket
- **协程友好**: 使用 Kotlin 协程处理 MCP 协议消息和生命周期事件
- **类型安全**: 完整的 Kotlin 类型系统支持
- **Ktor 集成**: 基于 Ktor 构建，与现有 Ktor 项目无缝集成

### 适用场景

- AI 助手连接外部数据源 (文件、数据库、API)
- 智能体执行工具调用和操作
- 企业聊天机器人连接多个数据源
- AI 模型与外部系统交互 (3D 设计、打印等)
- 标准化 AI 应用与外部系统的连接

---

## 什么是 MCP

### 定义

Model Context Protocol (MCP) 是由 Anthropic (Claude 的母公司) 在 2024 年推出的开放标准，旨在标准化 AI 应用程序与外部系统的连接。

### 核心理念

MCP 的核心理念是：**不要让模型去适应数据源，而要让数据源适应模型**。就像 USB-C 接口统一了电子设备的连接标准，MCP 统一了 AI 模型与外部系统的连接标准。

### 解决的问题

**MCP 出现之前**:
- 每个 AI 应用都需要自定义集成方式
- 数据源需要为不同 AI 开发多个适配器
- 开发成本高，维护复杂
- 缺乏标准化，互操作性差

**MCP 出现之后**:
- 统一的协议标准
- 一次开发，多处使用
- 降低开发成本和复杂度
- 提高互操作性

### 应用场景

1. **个人助理**: AI 访问 Google Calendar、Notion，提供个性化服务
2. **代码生成**: Claude Code 使用 Figma 设计生成完整 Web 应用
3. **企业聊天**: 连接多个数据库，支持数据分析
4. **创意设计**: AI 创建 3D 设计并打印

---

## 核心架构

### MCP 架构组件

```
┌─────────────┐         MCP Protocol          ┌─────────────┐
│   Client    │ ◄──────────────────────────► │   Server    │
│  (AI App)   │                              │ (Data/Tool) │
└─────────────┘                              └─────────────┘
     │                                              │
     │  - Claude Desktop                            │  - 文件系统
     │  - ChatGPT                                   │  - 数据库
     │  - VS Code                                   │  - API 服务
     │  - Cursor                                    │  - 专业工具
     └                                              └
```

### MCP 原语 (Primitives)

MCP 协议定义了 4 个核心原语:

| 原语 | 服务端角色 | 客户端角色 | 描述 |
|------|-----------|-----------|------|
| **Prompts** (提示) | 提供带参数的提示模板 | 请求和使用提示 | 交互式模板，用于 LLM 交互 (类似斜杠命令) |
| **Resources** (资源) | 暴露数据源 (文件、API 等) | 读取和订阅资源 | 为 LLM 提供上下文数据 |
| **Tools** (工具) | 定义可执行函数 | 调用工具执行操作 | LLM 可以调用以执行操作的函数 |
| **Sampling** (采样) | 向客户端请求 LLM 完成 | 执行 LLM 调用并返回结果 | 服务端发起的 LLM 请求 (反向) |

### 通信流程

```
Client                          Server
   │                              │
   │──── Initialize Request ────►│
   │  (client capabilities)       │
   │                              │
   │◄─── Initialize Response ────│
   │  (server capabilities)       │
   │                              │
   │──── List Tools ────────────►│
   │◄─── Tools List ─────────────│
   │                              │
   │──── Call Tool ─────────────►│
   │◄─── Tool Result ────────────│
   │                              │
   │──── List Resources ────────►│
   │◄─── Resources List ─────────│
   │                              │
   │──── Read Resource ─────────►│
   │◄─── Resource Content ───────│
   │                              │
   │◄─── Sampling Request ───────│
   │──── Sampling Result ───────►│
```

---

## 安装配置

### 依赖版本

在 `gradle/libs.versions.toml` 中添加:

```toml
[versions]
mcp = "0.6.0"  # 或最新版本
ktor = "3.4.1"
kotlinx-serialization = "1.8.1"

[libraries]
# MCP SDK
mcp-kotlin-sdk = { module = "io.modelcontextprotocol:kotlin-sdk", version.ref = "mcp" }
mcp-kotlin-sdk-client = { module = "io.modelcontextprotocol:kotlin-sdk-client", version.ref = "mcp" }
mcp-kotlin-sdk-server = { module = "io.modelcontextprotocol:kotlin-sdk-server", version.ref = "mcp" }

# Ktor (MCP 需要显式声明 Ktor 依赖)
ktor-client-core = { module = "io.ktor:ktor-client-core", version.ref = "ktor" }
ktor-client-cio = { module = "io.ktor:ktor-client-cio", version.ref = "ktor" }
ktor-client-sse = { module = "io.ktor:ktor-client-sse", version.ref = "ktor" }
ktor-server-core = { module = "io.ktor:ktor-server-core", version.ref = "ktor" }
ktor-server-netty = { module = "io.ktor:ktor-server-netty", version.ref = "ktor" }
ktor-server-cio = { module = "io.ktor:ktor-server-cio", version.ref = "ktor" }
ktor-server-sse = { module = "io.ktor:ktor-server-sse", version.ref = "ktor" }

# Kotlinx Serialization
kotlinx-serialization-json = { module = "org.jetbrains.kotlinx:kotlinx-serialization-json", version.ref = "kotlinx-serialization" }
```

### JVM 项目配置

在 `build.gradle.kts` 中添加:

```kotlin
repositories {
    mavenCentral()
}

dependencies {
    // 完整 SDK (客户端 + 服务端)
    implementation(libs.mcp.kotlin.sdk)
    
    // 或分别引入
    // implementation(libs.mcp.kotlin.sdk.client)
    // implementation(libs.mcp.kotlin.sdk.server)
    
    // Ktor 依赖 (MCP 不传递引入)
    implementation(libs.ktor.client.cio)
    implementation(libs.ktor.server.netty)
    
    // Kotlinx Serialization
    implementation(libs.kotlinx.serialization.json)
}
```

### Kotlin Multiplatform 配置

在共享模块的 `build.gradle.kts` 中:

```kotlin
kotlin {
    sourceSets {
        commonMain {
            dependencies {
                // 跨平台通用依赖
                implementation(libs.mcp.kotlin.sdk)
                implementation(libs.kotlinx.serialization.json)
            }
        }
        
        jvmMain {
            dependencies {
                // JVM 特定依赖
                implementation(libs.ktor.client.cio)
                implementation(libs.ktor.server.netty)
            }
        }
        
        nativeMain {
            dependencies {
                // Native 特定依赖
                implementation(libs.ktor.client.darwin)
                implementation(libs.ktor.server.cio)
            }
        }
    }
}
```

### 仅客户端或服务端

如果只需要客户端或服务端，可以单独引入:

```kotlin
dependencies {
    // 仅客户端
    implementation("io.modelcontextprotocol:kotlin-sdk-client:0.6.0")
    
    // 仅服务端
    implementation("io.modelcontextprotocol:kotlin-sdk-server:0.6.0")
}
```

---

## 快速开始

### 创建 MCP 客户端

```kotlin
import io.ktor.client.HttpClient
import io.ktor.client.plugins.sse.SSE
import io.modelcontextprotocol.kotlin.sdk.client.Client
import io.modelcontextprotocol.kotlin.sdk.client.StreamableHttpClientTransport
import io.modelcontextprotocol.kotlin.sdk.types.Implementation
import kotlinx.coroutines.runBlocking

fun main(args: Array<String>) = runBlocking {
    // 服务器地址
    val url = args.firstOrNull() ?: "http://localhost:3000/mcp"
    
    // 创建 HttpClient (需要安装 SSE 插件)
    val httpClient = HttpClient { install(SSE) }
    
    // 创建 MCP 客户端
    val client = Client(
        clientInfo = Implementation(
            name = "example-client",
            version = "1.0.0"
        )
    )
    
    // 创建传输层
    val transport = StreamableHttpClientTransport(
        client = httpClient,
        url = url
    )
    
    // 连接到服务器
    client.connect(transport)
    
    // 列出可用工具
    val tools = client.listTools().tools
    println("可用工具：$tools")
    
    // 调用工具
    val result = client.callTool(
        name = "example-tool",
        arguments = mapOf("input" to "Hello")
    )
    println("工具结果：$result")
    
    // 列出资源
    val resources = client.listResources().resources
    println("可用资源：$resources")
    
    // 读取资源
    val resource = client.readResource("note://release/latest")
    println("资源内容：$resource")
}
```

### 创建 MCP 服务端

```kotlin
import io.ktor.server.cio.CIO
import io.ktor.server.engine.embeddedServer
import io.modelcontextprotocol.kotlin.sdk.server.Server
import io.modelcontextprotocol.kotlin.sdk.server.ServerOptions
import io.modelcontextprotocol.kotlin.sdk.server.mcp
import io.modelcontextprotocol.kotlin.sdk.types.CallToolResult
import io.modelcontextprotocol.kotlin.sdk.types.Implementation
import io.modelcontextprotocol.kotlin.sdk.types.ServerCapabilities
import io.modelcontextprotocol.kotlin.sdk.types.TextContent
import io.modelcontextprotocol.kotlin.sdk.types.ToolSchema
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

fun main(args: Array<String>) {
    val port = args.firstOrNull()?.toIntOrNull() ?: 3000
    
    // 创建 MCP 服务器
    val mcpServer = Server(
        serverInfo = Implementation(
            name = "example-server",
            version = "1.0.0"
        ),
        options = ServerOptions(
            capabilities = ServerCapabilities(
                tools = ServerCapabilities.Tools(listChanged = true),
            ),
        )
    )
    
    // 添加工具
    mcpServer.addTool(
        name = "example-tool",
        description = "An example tool",
        inputSchema = ToolSchema(
            properties = buildJsonObject {
                put("input", buildJsonObject { put("type", "string") })
            }
        )
    ) { request ->
        CallToolResult(content = listOf(TextContent("Hello, world!")))
    }
    
    // 启动嵌入式 Ktor 服务器
    embeddedServer(CIO, host = "127.0.0.1", port = port) {
        mcp {
            mcpServer
        }
    }.start(wait = true)
}
```

### 测试 MCP 服务器

使用 MCP Inspector 测试:

```bash
# 安装并运行 MCP Inspector
npx -y @modelcontextprotocol/inspector

# 在 Inspector UI 中连接到 http://localhost:3000
```

或使用 Claude Desktop:

```bash
# 添加 MCP 服务器
claude mcp add --transport http kotlin-mcp http://localhost:3000
```

---

## 核心概念

### 能力 (Capabilities)

能力定义了服务端或客户端支持的功能，在初始化时声明。

#### 服务端能力

| 能力 | 功能标志 | 描述 |
|------|---------|------|
| **prompts** | `listChanged` | 提示模板管理和通知 |
| **resources** | `subscribe`, `listChanged` | 资源暴露、订阅和更新通知 |
| **tools** | `listChanged` | 工具发现、执行和列表变更通知 |
| **logging** | - | 服务端日志输出到客户端控制台 |
| **completions** | - | 参数自动完成建议 |
| **experimental** | 自定义属性 | 非标准实验性功能 |

#### 客户端能力

| 能力 | 功能标志 | 描述 |
|------|---------|------|
| **sampling** | - | 客户端可以执行 LLM 请求 |
| **roots** | `listChanged` | 客户端暴露根目录并可通知变更 |
| **elicitation** | - | 客户端可以显示模式/表单对话框 |
| **experimental** | 自定义属性 | 非标准实验性功能 |

### 能力声明示例

```kotlin
// 服务端能力
val serverOptions = ServerOptions(
    capabilities = ServerCapabilities(
        prompts = ServerCapabilities.Prompts(listChanged = true),
        resources = ServerCapabilities.Resources(subscribe = true, listChanged = true),
        tools = ServerCapabilities.Tools(listChanged = true),
        logging = ServerCapabilities.Logging,
        completions = ServerCapabilities.Completions,
    )
)

// 客户端能力
val clientOptions = ClientOptions(
    capabilities = ClientCapabilities(
        roots = ClientCapabilities.Roots(listChanged = true),
        sampling = ClientCapabilities.Sampling,
        elicitation = ClientCapabilities.Elicitation,
    )
)
```

---

## 服务器功能

### 提示 (Prompts)

提示是用户控制的模板，客户端通过 `prompts/list` 发现，通过 `prompts/get` 获取。

**适用场景**:
- 代码审查模板
- Bug 分类问题
- 入职检查清单
- 保存的搜索

```kotlin
val server = Server(
    serverInfo = Implementation("example-server", "1.0.0"),
    options = ServerOptions(
        capabilities = ServerCapabilities(
            prompts = ServerCapabilities.Prompts(listChanged = true),
        ),
    )
)

server.addPrompt(
    name = "code-review",
    description = "Ask the model to review a diff",
    arguments = listOf(
        PromptArgument(
            name = "diff",
            description = "Unified diff",
            required = true
        ),
    ),
) { request ->
    GetPromptResult(
        description = "Quick code review helper",
        messages = listOf(
            PromptMessage(
                role = Role.User,
                content = TextContent(
                    text = "Review this change:\n${request.arguments?.get("diff")}"
                ),
            ),
        ),
    )
}
```

**使用建议**:
- 仅当提示目录在运行时会变化时才设置 `listChanged = true`
- 当提示目录变化时发送 `notifications/prompts/list_changed` 通知

### 资源 (Resources)

资源是应用程序驱动的数据源，客户端通过 `resources/list` 或 `resources/templates/list` 发现，通过 `resources/read` 获取。

**适用场景**:
- 文件系统内容
- 数据库记录
- API 响应
- 部署报告

```kotlin
val server = Server(
    serverInfo = Implementation("example-server", "1.0.0"),
    options = ServerOptions(
        capabilities = ServerCapabilities(
            resources = ServerCapabilities.Resources(
                subscribe = true,
                listChanged = true
            ),
        ),
    )
)

server.addResource(
    uri = "note://release/latest",
    name = "Release notes",
    description = "Last deployment summary",
    mimeType = "text/markdown",
) { request ->
    ReadResourceResult(
        contents = listOf(
            TextResourceContents(
                text = "Ship 42 reached production successfully.",
                uri = request.uri,
                mimeType = "text/markdown",
            ),
        ),
    )
}
```

**资源类型**:
- 静态文本
- 生成的 JSON
- 二进制数据

**通知**:
- `subscribe = true`: 当特定 URI 变化时发送 `notifications/resources/updated`
- `listChanged = true`: 当目录变化时发送 `notifications/resources/list_changed`

### 工具 (Tools)

工具是模型控制的函数，客户端通过 `tools/list` 发现，通过 `tools/call` 调用。

**适用场景**:
- 计算器
- 搜索引擎
- 数据库查询
- 文件操作
- API 调用

```kotlin
val server = Server(
    serverInfo = Implementation("example-server", "1.0.0"),
    options = ServerOptions(
        capabilities = ServerCapabilities(
            tools = ServerCapabilities.Tools(listChanged = true),
        ),
    )
)

server.addTool(
    name = "echo",
    description = "Return whatever the user sent back to them",
    inputSchema = ToolSchema(
        properties = buildJsonObject {
            put("text", buildJsonObject { put("type", "string") })
        }
    )
) { request ->
    val text = request.arguments?.get("text")?.jsonPrimitive?.content ?: "(empty)"
    CallToolResult(content = listOf(TextContent(text = "Echo: $text")))
}
```

**高级用法**:

```kotlin
server.addTool(
    name = "search",
    description = "Search for information",
    inputSchema = ToolSchema(
        properties = buildJsonObject {
            put("query", buildJsonObject { put("type", "string") })
            put("limit", buildJsonObject { 
                put("type", "integer")
                put("default", 10)
            })
        },
        required = listOf("query")
    )
) { request ->
    val query = request.arguments?.get("query")?.jsonPrimitive?.content ?: ""
    val limit = request.arguments?.get("limit")?.jsonPrimitive?.int ?: 10
    
    // 执行搜索
    val results = searchEngine.search(query, limit)
    
    // 返回结果
    CallToolResult(
        content = listOf(
            TextContent(text = "Found ${results.size} results:")
        ) + results.map { result ->
            TextContent(text = "- ${result.title}: ${result.url}")
        }
    )
}
```

**使用建议**:
- 敏感操作需要人工确认
- 长任务通过请求上下文报告进度
- 仅当工具目录变化时才设置 `listChanged = true`

### 完成 (Completion)

Completion 为提示或资源模板提供参数建议。

**适用场景**:
- 参数自动完成
- 下拉建议
- 上下文相关的选项

```kotlin
val server = Server(
    serverInfo = Implementation("example-server", "1.0.0"),
    options = ServerOptions(
        capabilities = ServerCapabilities(
            completions = ServerCapabilities.Completions,
        ),
    )
)

val session = server.createSession(
    StdioServerTransport(
        inputStream = System.`in`.asSource().buffered(),
        outputStream = System.out.asSink().buffered()
    )
)

session.setRequestHandler<CompleteRequest>(Method.Defined.CompletionComplete) { request, _ ->
    val options = listOf("kotlin", "compose", "coroutine")
    val matches = options.filter { 
        it.startsWith(request.argument.value.lowercase()) 
    }
    
    CompleteResult(
        completion = CompleteResult.Completion(
            values = matches.take(3),
            total = matches.size,
            hasMore = matches.size > 3,
        ),
    )
}
```

**使用建议**:
- 最多返回 100 个排名值
- 使用 `total` 和 `hasMore` 进行分页
- 使用 `context.arguments` 为依赖字段提供建议

### 日志 (Logging)

日志允许服务端使用 RFC 5424 级别 (debug → emergency) 向客户端发送结构化日志通知。

```kotlin
val server = Server(
    serverInfo = Implementation("example-server", "1.0.0"),
    options = ServerOptions(
        capabilities = ServerCapabilities(
            logging = ServerCapabilities.Logging,
        ),
    )
)

val session = server.createSession(
    StdioServerTransport(
        inputStream = System.`in`.asSource().buffered(),
        outputStream = System.out.asSink().buffered()
    )
)

// 发送日志
session.sendLoggingMessage(
    LoggingMessageNotification(
        params = LoggingMessageNotificationParams(
            level = LoggingLevel.Info,
            logger = "startup",
            data = buildJsonObject { 
                put("message", "Server started")
                put("timestamp", System.currentTimeMillis())
            },
        ),
    ),
)
```

**日志级别**:
- `Debug`: 调试信息
- `Info`: 一般信息
- `Notice`: 正常但重要的事件
- `Warning`: 警告
- `Error`: 错误
- `Critical`: 严重错误
- `Alert`: 需要立即行动
- `Emergency`: 系统不可用

**使用建议**:
- 日志中不包含敏感数据
- 客户端会在自己的 UI 中显示日志
- 客户端可以通过 `logging/setLevel` 调整最低级别

### 分页 (Pagination)

列表操作返回带 `nextCursor` 的分页结果。

**支持的分页调用**:
- `resources/list`
- `resources/templates/list`
- `prompts/list`
- `tools/list`

```kotlin
val server = Server(
    serverInfo = Implementation("example-server", "1.0.0"),
    options = ServerOptions(
        capabilities = ServerCapabilities(
            resources = ServerCapabilities.Resources(),
        ),
    )
)

val session = server.createSession(
    StdioServerTransport(
        inputStream = System.`in`.asSource().buffered(),
        outputStream = System.out.asSink().buffered()
    )
)

val resources = listOf(
    Resource(uri = "note://1", name = "Note 1", description = "First"),
    Resource(uri = "note://2", name = "Note 2", description = "Second"),
    Resource(uri = "note://3", name = "Note 3", description = "Third"),
)
val pageSize = 2

session.setRequestHandler<ListResourcesRequest>(Method.Defined.ResourcesList) { request, _ ->
    val start = request.params?.cursor?.toIntOrNull() ?: 0
    val page = resources.drop(start).take(pageSize)
    val next = if (start + page.size < resources.size) {
        (start + page.size).toString()
    } else {
        null
    }
    
    ListResourcesResult(
        resources = page,
        nextCursor = next,
    )
}
```

**使用建议**:
- `nextCursor` 仅在还有更多项目时包含
- 游标是不透明的，不要解析或持久化
- 空游标结束分页

---

## 客户端功能

### 根目录 (Roots)

根目录让客户端声明服务器允许操作的位置。

**适用场景**:
- 项目目录
- 文件系统访问限制
- 权限控制

```kotlin
val client = Client(
    clientInfo = Implementation("demo-client", "1.0.0"),
    options = ClientOptions(
        capabilities = ClientCapabilities(
            roots = ClientCapabilities.Roots(listChanged = true)
        ),
    ),
)

// 添加根目录
client.addRoot(
    uri = "file:///Users/demo/projects",
    name = "Projects",
)

// 通知服务器根目录列表已变化
client.sendRootsListChanged()
```

**使用建议**:
- URI 必须是 `file://` 路径
- 当文件系统视图变化时调用 `addRoot`/`removeRoot`
- 使用 `sendRootsListChanged()` 通知服务器
- 根目录列表应由用户控制

### 采样 (Sampling)

采样允许服务器请求客户端调用其首选的 LLM。

**适用场景**:
- 服务器需要 LLM 能力
- 反向 LLM 调用
- 模型选择委托给客户端

```kotlin
val client = Client(
    clientInfo = Implementation("demo-client", "1.0.0"),
    options = ClientOptions(
        capabilities = ClientCapabilities(
            sampling = ClientCapabilities.Sampling(
                tools = buildJsonObject { } // 不支持工具使用时省略
            ),
        ),
    ),
)

client.setRequestHandler<CreateMessageRequest>(Method.Defined.SamplingCreateMessage) { request, _ ->
    // 获取最后一条消息
    val content = request.messages.lastOrNull()?.content
    val prompt = if (content is TextContent) {
        content.text
    } else {
        "your topic"
    }
    
    // 返回采样结果
    CreateMessageResult(
        model = "gpt-4o-mini",
        role = Role.Assistant,
        content = TextContent(text = "Here is a short note about $prompt"),
    )
}
```

**使用建议**:
- 在处理器中可以选择任何模型/提供商
- 需要人工批准时可以要求确认
- 可以拒绝请求
- 不支持工具使用时省略 `sampling.tools`

---

## 传输层

所有传输层共享相同的 API 表面，可以在不改变业务逻辑的情况下切换部署方式。

### STDIO 传输

**适用场景**:
- 编辑器插件
- CLI 工具
- 本地进程通信

```kotlin
// 服务端
val server = Server(
    serverInfo = Implementation("stdio-server", "1.0.0"),
    options = ServerOptions(capabilities = ServerCapabilities())
)

val transport = StdioServerTransport(
    inputStream = System.`in`.asSource().buffered(),
    outputStream = System.out.asSink().buffered()
)

val session = server.createSession(transport)

// 客户端
val client = Client(
    clientInfo = Implementation("stdio-client", "1.0.0"),
    options = ClientOptions(capabilities = ClientCapabilities())
)

val clientTransport = StdioClientTransport(
    processBuilder = ProcessBuilder("node", "server.js")
)

client.connect(clientTransport)
```

**特点**:
- 无需网络配置
- 适合本地进程间通信
- 编辑器集成首选

### Streamable HTTP 传输

**适用场景**:
- 远程部署
- 代理服务
- 服务网格

```kotlin
// 服务端
embeddedServer(CIO, port = 3000) {
    mcpStreamableHttp(path = "/api/mcp") {
        MyServer()
    }
}.start(wait = true)

// 客户端
val httpClient = HttpClient { install(SSE) }

val client = Client(
    clientInfo = Implementation("http-client", "1.0.0")
)

val transport = StreamableHttpClientTransport(
    client = httpClient,
    url = "http://localhost:3000/api/mcp"
)

client.connect(transport)
```

**特点**:
- 推荐用于新项目
- 单个 HTTP 端点
- 支持 JSON-only 或 SSE 流式响应
- 可自定义路径 (默认：`/mcp`)

### SSE (Server-Sent Events) 传输

**适用场景**:
- 向后兼容旧 MCP 客户端
- 单向流式传输

```kotlin
// 方式 1: 自动安装 SSE 插件
embeddedServer(CIO, port = 3000) {
    mcp { MyServer() }  // 自动安装 SSE 插件，端点为 /.
}.start(wait = true)

// 方式 2: 手动安装，自定义路径
embeddedServer(CIO, port = 3000) {
    install(SSE)
    routing {
        route("/api/mcp") {
            mcp { MyServer() }
        }
    }
}.start(wait = true)
```

**特点**:
- 向后兼容
- 单向流 (服务器 → 客户端)
- 新项目推荐使用 Streamable HTTP

### WebSocket 传输

**适用场景**:
- 全双工通信
- 低延迟连接
- 长会话

```kotlin
// 服务端
embeddedServer(CIO, port = 3000) {
    mcpWebSocket(path = "/ws/mcp") {
        MyServer()
    }
}.start(wait = true)

// 客户端
val client = Client(
    clientInfo = Implementation("ws-client", "1.0.0")
)

val transport = WebSocketClientTransport(
    url = "ws://localhost:3000/ws/mcp"
)

client.connect(transport)
```

**特点**:
- 全双工通信
- 低延迟
- 适合大量通知
- 需要 WebSocket 反向代理支持

### ChannelTransport (测试)

**适用场景**:
- 单元测试
- 本地开发
- 无需网络

```kotlin
import io.modelcontextprotocol.kotlin.sdk.testing.ChannelTransport

// 创建配对的客户端和服务器传输
val (clientTransport, serverTransport) = ChannelTransport.createPair()

// 创建服务器
val server = Server(
    serverInfo = Implementation("test-server", "1.0.0")
)
val serverSession = server.createSession(serverTransport)

// 创建客户端
val client = Client(
    clientInfo = Implementation("test-client", "1.0.0")
)
client.connect(clientTransport)

// 现在可以直接测试，无需网络
```

**特点**:
- 使用 Kotlin 协程通道
- 全双工连接
- 无需网络配置
- 适合测试

---

## 完整示例

### 天气查询 MCP 服务器

```kotlin
package com.example.weather

import io.ktor.server.cio.CIO
import io.ktor.server.engine.embeddedServer
import io.modelcontextprotocol.kotlin.sdk.server.Server
import io.modelcontextprotocol.kotlin.sdk.server.ServerOptions
import io.modelcontextprotocol.kotlin.sdk.server.mcp
import io.modelcontextprotocol.kotlin.sdk.types.*
import kotlinx.serialization.json.*

// 天气数据模型
data class WeatherData(
    val temperature: Double,
    val condition: String,
    val humidity: Int,
    val city: String
)

// 模拟天气服务
object WeatherService {
    fun getWeather(city: String): WeatherData {
        return WeatherData(
            temperature = 25.0,
            condition = "Sunny",
            humidity = 60,
            city = city
        )
    }
    
    fun getAlerts(city: String): List<String> {
        return listOf("No active alerts for $city")
    }
}

fun main() {
    val port = 3000
    
    // 创建 MCP 服务器
    val server = Server(
        serverInfo = Implementation(
            name = "weather-server",
            version = "1.0.0"
        ),
        options = ServerOptions(
            capabilities = ServerCapabilities(
                tools = ServerCapabilities.Tools(listChanged = false),
                resources = ServerCapabilities.Resources(subscribe = false, listChanged = false),
                prompts = ServerCapabilities.Prompts(listChanged = false),
                logging = ServerCapabilities.Logging,
            ),
        )
    )
    
    // 添加天气查询工具
    server.addTool(
        name = "get_weather",
        description = "Get current weather for a city",
        inputSchema = ToolSchema(
            properties = buildJsonObject {
                put("city", buildJsonObject { 
                    put("type", "string")
                    put("description", buildJsonPrimitive("City name"))
                })
            },
            required = listOf("city")
        )
    ) { request ->
        val city = request.arguments?.get("city")?.jsonPrimitive?.content ?: "Unknown"
        val weather = WeatherService.getWeather(city)
        
        CallToolResult(
            content = listOf(
                TextContent(
                    text = """
                        Weather in ${weather.city}:
                        - Temperature: ${weather.temperature}°C
                        - Condition: ${weather.condition}
                        - Humidity: ${weather.humidity}%
                    """.trimIndent()
                )
            )
        )
    }
    
    // 添加天气警报工具
    server.addTool(
        name = "get_weather_alerts",
        description = "Get weather alerts for a city",
        inputSchema = ToolSchema(
            properties = buildJsonObject {
                put("city", buildJsonObject { 
                    put("type", "string")
                    put("description", buildJsonPrimitive("City name"))
                })
            },
            required = listOf("city")
        )
    ) { request ->
        val city = request.arguments?.get("city")?.jsonPrimitive?.content ?: "Unknown"
        val alerts = WeatherService.getAlerts(city)
        
        CallToolResult(
            content = listOf(
                TextContent(text = alerts.joinToString("\n"))
            )
        )
    }
    
    // 添加天气资源
    server.addResource(
        uri = "weather://current",
        name = "Current Weather",
        description = "Current weather data",
        mimeType = "application/json"
    ) { request ->
        val weather = WeatherService.getWeather("Default City")
        val json = Json.encodeToString(WeatherData.serializer(), weather)
        
        ReadResourceResult(
            contents = listOf(
                TextResourceContents(
                    text = json,
                    uri = request.uri,
                    mimeType = "application/json"
                )
            )
        )
    }
    
    // 添加天气提示
    server.addPrompt(
        name = "weather-check",
        description = "Check weather for a trip",
        arguments = listOf(
            PromptArgument(
                name = "destination",
                description = "Trip destination city",
                required = true
            ),
            PromptArgument(
                name = "date",
                description = "Travel date",
                required = false
            )
        )
    ) { request ->
        val destination = request.arguments?.get("destination")?.jsonPrimitive?.content ?: "Unknown"
        val date = request.arguments?.get("date")?.jsonPrimitive?.content ?: "today"
        
        GetPromptResult(
            description = "Weather check for your trip",
            messages = listOf(
                PromptMessage(
                    role = Role.User,
                    content = TextContent(
                        text = "What's the weather like in $destination on $date? Should I pack an umbrella?"
                    )
                )
            )
        )
    }
    
    // 启动服务器
    println("Starting MCP Weather Server on port $port...")
    embeddedServer(CIO, host = "127.0.0.1", port = port) {
        mcp {
            server
        }
    }.start(wait = true)
}
```

### MCP 客户端示例

```kotlin
package com.example.weather.client

import io.ktor.client.HttpClient
import io.ktor.client.plugins.sse.SSE
import io.modelcontextprotocol.kotlin.sdk.client.Client
import io.modelcontextprotocol.kotlin.sdk.client.StreamableHttpClientTransport
import io.modelcontextprotocol.kotlin.sdk.types.Implementation
import kotlinx.coroutines.runBlocking

fun main() = runBlocking {
    val url = "http://localhost:3000/mcp"
    
    // 创建 HttpClient
    val httpClient = HttpClient { install(SSE) }
    
    // 创建 MCP 客户端
    val client = Client(
        clientInfo = Implementation(
            name = "weather-client",
            version = "1.0.0"
        )
    )
    
    // 创建传输层
    val transport = StreamableHttpClientTransport(
        client = httpClient,
        url = url
    )
    
    try {
        // 连接到服务器
        println("Connecting to MCP server at $url...")
        client.connect(transport)
        println("Connected!")
        
        // 列出工具
        println("\n=== Available Tools ===")
        val tools = client.listTools().tools
        tools.forEach { tool ->
            println("Tool: ${tool.name}")
            println("  Description: ${tool.description}")
            println("  Input Schema: ${tool.inputSchema}")
        }
        
        // 调用天气查询工具
        println("\n=== Calling get_weather ===")
        val weatherResult = client.callTool(
            name = "get_weather",
            arguments = mapOf("city" to "Beijing")
        )
        println("Weather Result:")
        weatherResult.content.forEach { content ->
            println(content)
        }
        
        // 调用天气警报工具
        println("\n=== Calling get_weather_alerts ===")
        val alertsResult = client.callTool(
            name = "get_weather_alerts",
            arguments = mapOf("city" to "Beijing")
        )
        println("Alerts Result:")
        alertsResult.content.forEach { content ->
            println(content)
        }
        
        // 列出资源
        println("\n=== Available Resources ===")
        val resources = client.listResources().resources
        resources.forEach { resource ->
            println("Resource: ${resource.name} (${resource.uri})")
        }
        
        // 读取资源
        println("\n=== Reading Resource ===")
        val resourceResult = client.readResource("weather://current")
        println("Resource Content:")
        resourceResult.contents.forEach { content ->
            println(content)
        }
        
        // 列出提示
        println("\n=== Available Prompts ===")
        val prompts = client.listPrompts().prompts
        prompts.forEach { prompt ->
            println("Prompt: ${prompt.name}")
            println("  Description: ${prompt.description}")
            println("  Arguments: ${prompt.arguments}")
        }
        
        // 获取提示
        println("\n=== Getting Prompt ===")
        val promptResult = client.getPrompt(
            name = "weather-check",
            arguments = mapOf(
                "destination" to "Paris",
                "date" to "2026-03-15"
            )
        )
        println("Prompt Result:")
        println("  Description: ${promptResult.description}")
        promptResult.messages.forEach { message ->
            println("  Message [${message.role}]: ${message.content}")
        }
        
    } finally {
        // 关闭客户端
        client.close()
        httpClient.close()
        println("\nClient closed.")
    }
}
```

### 测试示例

```kotlin
package com.example.weather.test

import io.modelcontextprotocol.kotlin.sdk.client.Client
import io.modelcontextprotocol.kotlin.sdk.server.Server
import io.modelcontextprotocol.kotlin.sdk.testing.ChannelTransport
import io.modelcontextprotocol.kotlin.sdk.types.Implementation
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull

class WeatherServerTest {
    
    @Test
    fun testWeatherTool() = runTest {
        // 创建传输层
        val (clientTransport, serverTransport) = ChannelTransport.createPair()
        
        // 创建服务器
        val server = Server(
            serverInfo = Implementation("test-weather-server", "1.0.0")
        )
        
        server.addTool(
            name = "get_weather",
            description = "Get weather"
        ) { request ->
            // 测试实现
            io.modelcontextprotocol.kotlin.sdk.types.CallToolResult(
                content = listOf(
                    io.modelcontextprotocol.kotlin.sdk.types.TextContent(
                        text = "Sunny, 25°C"
                    )
                )
            )
        }
        
        val serverSession = server.createSession(serverTransport)
        
        // 创建客户端
        val client = Client(
            clientInfo = Implementation("test-client", "1.0.0")
        )
        client.connect(clientTransport)
        
        // 测试工具调用
        val result = client.callTool(
            name = "get_weather",
            arguments = mapOf("city" to "Beijing")
        )
        
        // 验证结果
        assertNotNull(result)
        assertEquals(1, result.content.size)
    }
}
```

---

## 最佳实践

### 1. 错误处理

```kotlin
server.addTool(
    name = "safe-operation",
    description = "A tool with error handling"
) { request ->
    try {
        // 执行操作
        val result = performOperation()
        
        CallToolResult(
            content = listOf(TextContent(text = "Success: $result")),
            isError = false
        )
    } catch (e: Exception) {
        CallToolResult(
            content = listOf(TextContent(text = "Error: ${e.message}")),
            isError = true
        )
    }
}
```

### 2. 进度报告

```kotlin
server.addTool(
    name = "long-running-task",
    description = "A tool that reports progress"
) { request ->
    val context = request.context
    
    for (i in 1..100) {
        // 报告进度
        context.sendProgress(
            ProgressNotification(
                params = ProgressNotificationParams(
                    progressToken = request.progressToken,
                    progress = i,
                    total = 100
                )
            )
        )
        
        // 执行任务
        performStep(i)
    }
    
    CallToolResult(content = listOf(TextContent("Completed")))
}
```

### 3. 资源管理

```kotlin
// 使用 use 函数确保资源正确关闭
fun useClient() {
    val client = Client(
        clientInfo = Implementation("client", "1.0.0")
    )
    
    try {
        // 使用客户端
        client.connect(transport)
        // ...
    } finally {
        client.close()
    }
}
```

### 4. 日志记录

```kotlin
server.addTool(
    name = "logged-operation",
    description = "An operation with logging"
) { request ->
    val session = request.context.session
    
    // 发送日志
    session.sendLoggingMessage(
        LoggingMessageNotification(
            params = LoggingMessageNotificationParams(
                level = LoggingLevel.Info,
                logger = "operation",
                data = buildJsonObject {
                    put("operation", "started")
                    put("timestamp", System.currentTimeMillis())
                }
            )
        )
    )
    
    try {
        // 执行操作
        val result = performOperation()
        
        session.sendLoggingMessage(
            LoggingMessageNotification(
                params = LoggingMessageNotificationParams(
                    level = LoggingLevel.Info,
                    logger = "operation",
                    data = buildJsonObject {
                        put("operation", "completed")
                        put("result", result)
                    }
                )
            )
        )
        
        CallToolResult(content = listOf(TextContent("Success")))
    } catch (e: Exception) {
        session.sendLoggingMessage(
            LoggingMessageNotification(
                params = LoggingMessageNotificationParams(
                    level = LoggingLevel.Error,
                    logger = "operation",
                    data = buildJsonObject {
                        put("operation", "failed")
                        put("error", e.message ?: "Unknown error")
                    }
                )
            )
        )
        
        CallToolResult(
            content = listOf(TextContent("Error: ${e.message}")),
            isError = true
        )
    }
}
```

### 5. 安全性

```kotlin
server.addTool(
    name = "sensitive-operation",
    description = "Requires user confirmation"
) { request ->
    // 检查权限
    if (!hasPermission(request)) {
        return@addTool CallToolResult(
            content = listOf(TextContent("Permission denied")),
            isError = true
        )
    }
    
    // 敏感操作需要确认
    val confirmed = requestUserConfirmation()
    if (!confirmed) {
        return@addTool CallToolResult(
            content = listOf(TextContent("Operation cancelled")),
            isError = false
        )
    }
    
    // 执行操作
    performSensitiveOperation()
    
    CallToolResult(content = listOf(TextContent("Operation completed")))
}
```

### 6. 性能优化

```kotlin
// 缓存资源内容
class CachedResourceServer {
    private val cache = ConcurrentHashMap<String, String>()
    
    fun addCachedResource(server: Server) {
        server.addResource(
            uri = "data://expensive",
            name = "Expensive Resource"
        ) { request ->
            // 检查缓存
            val cached = cache[request.uri]
            if (cached != null) {
                return@addResource ReadResourceResult(
                    contents = listOf(
                        TextResourceContents(
                            text = cached,
                            uri = request.uri
                        )
                    )
                )
            }
            
            // 生成内容
            val content = generateExpensiveContent()
            cache[request.uri] = content
            
            ReadResourceResult(
                contents = listOf(
                    TextResourceContents(
                        text = content,
                        uri = request.uri
                    )
                )
            )
        }
    }
}
```

---

## 与 LM Studio 集成

### 架构

```
┌─────────────┐         MCP Protocol          ┌─────────────┐
│  LM Studio  │ ◄──────────────────────────► │ MCP Server  │
│   (Client)  │                              │  (Kotlin)   │
└─────────────┘                              └─────────────┘
     │                                              │
     │  - 本地 LLM                                  │  - 文件系统
     │  - 模型管理                                  │  - 数据库
     │  - 采样能力                                  │  - 工具调用
     └                                              └
```

### 配置 LM Studio 客户端

```kotlin
import io.ktor.client.HttpClient
import io.ktor.client.plugins.sse.SSE
import io.modelcontextprotocol.kotlin.sdk.client.Client
import io.modelcontextprotocol.kotlin.sdk.client.StreamableHttpClientTransport
import io.modelcontextprotocol.kotlin.sdk.types.Implementation

class LMStudioMCPClient {
    private val httpClient = HttpClient { install(SSE) }
    
    private val client = Client(
        clientInfo = Implementation(
            name = "lm-studio-mcp-client",
            version = "1.0.0"
        ),
        options = ClientOptions(
            capabilities = ClientCapabilities(
                sampling = ClientCapabilities.Sampling
            )
        )
    )
    
    suspend fun connect(url: String = "http://localhost:3000/mcp") {
        val transport = StreamableHttpClientTransport(
            client = httpClient,
            url = url
        )
        client.connect(transport)
    }
    
    // 处理采样请求
    fun setupSampling() {
        client.setRequestHandler<CreateMessageRequest>(
            Method.Defined.SamplingCreateMessage
        ) { request, _ ->
            // 这里可以集成 LM Studio 的 API
            // 调用本地 LLM 返回结果
            CreateMessageResult(
                model = "local-model",
                role = Role.Assistant,
                content = TextContent(text = "Response from local LLM")
            )
        }
    }
}
```

### 使用 LM Studio REST API

```kotlin
import io.ktor.client.call.body
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import kotlinx.serialization.Serializable

@Serializable
data class ChatRequest(
    val model: String,
    val messages: List<Message>,
    val maxTokens: Int = 100
)

@Serializable
data class Message(
    val role: String,
    val content: String
)

@Serializable
data class ChatResponse(
    val choices: List<Choice>
)

@Serializable
data class Choice(
    val message: Message
)

class LMStudioSampler(private val baseUrl: String = "http://localhost:1234") {
    private val client = HttpClient { install(ContentNegotiation) { json() } }
    
    suspend fun sample(prompt: String): String {
        val response = client.post("$baseUrl/v1/chat/completions") {
            setBody(
                ChatRequest(
                    model = "local-model",
                    messages = listOf(
                        Message(role = "user", content = prompt)
                    )
                )
            )
        }.body<ChatResponse>()
        
        return response.choices.first().message.content
    }
}
```

### 完整集成示例

```kotlin
import io.ktor.server.cio.CIO
import io.ktor.server.engine.embeddedServer
import io.modelcontextprotocol.kotlin.sdk.server.Server
import io.modelcontextprotocol.kotlin.sdk.server.ServerOptions
import io.modelcontextprotocol.kotlin.sdk.server.mcp
import io.modelcontextprotocol.kotlin.sdk.types.*
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

fun main() = runBlocking {
    // 创建 MCP 服务器
    val server = Server(
        serverInfo = Implementation("lm-studio-integration", "1.0.0"),
        options = ServerOptions(
            capabilities = ServerCapabilities(
                tools = ServerCapabilities.Tools(),
                resources = ServerCapabilities.Resources(),
                prompts = ServerCapabilities.Prompts(),
            )
        )
    )
    
    // LM Studio 采样器
    val sampler = LMStudioSampler()
    
    // 添加使用 LM Studio 的工具
    server.addTool(
        name = "ask-local-llm",
        description = "Ask a question to the local LLM via LM Studio"
    ) { request ->
        val question = request.arguments?.get("question")?.jsonPrimitive?.content ?: ""
        
        try {
            val answer = sampler.sample(question)
            CallToolResult(
                content = listOf(TextContent(text = answer))
            )
        } catch (e: Exception) {
            CallToolResult(
                content = listOf(TextContent(text = "Error: ${e.message}")),
                isError = true
            )
        }
    }
    
    // 启动服务器
    embeddedServer(CIO, port = 3000) {
        mcp {
            server
        }
    }.start(wait = true)
}
```

---

## 常见问题

### 1. 连接失败

**问题**: 客户端无法连接到服务器

**解决方案**:
```kotlin
// 检查服务器是否启动
println("Server starting on port 3000...")

// 使用正确的 URL
val url = "http://localhost:3000/mcp"  // 确保路径正确

// 检查 HttpClient 配置
val httpClient = HttpClient {
    install(SSE)  // 必须安装 SSE 插件
    engine {
        // 根据需要配置引擎
    }
}
```

### 2. 工具调用失败

**问题**: 工具调用返回错误

**解决方案**:
```kotlin
// 确保输入 Schema 正确
inputSchema = ToolSchema(
    properties = buildJsonObject {
        put("param", buildJsonObject { 
            put("type", "string")
            put("description", buildJsonPrimitive("Parameter description"))
        })
    },
    required = listOf("param")  // 必需参数
)

// 处理参数
server.addTool(name = "tool") { request ->
    val param = request.arguments?.get("param")?.jsonPrimitive?.content
    if (param == null) {
        return@addTool CallToolResult(
            content = listOf(TextContent("Missing required parameter")),
            isError = true
        )
    }
    // ...
}
```

### 3. 资源读取失败

**问题**: 无法读取资源内容

**解决方案**:
```kotlin
// 确保 URI 匹配
server.addResource(
    uri = "resource://my-resource",  // 使用一致的 URI 格式
    name = "My Resource"
) { request ->
    // request.uri 应该与添加时的 uri 匹配
    println("Reading resource: ${request.uri}")
    
    ReadResourceResult(
        contents = listOf(
            TextResourceContents(
                text = "Content",
                uri = request.uri,  // 返回相同的 URI
                mimeType = "text/plain"
            )
        )
    )
}
```

### 4. 内存泄漏

**问题**: 长时间运行后内存占用过高

**解决方案**:
```kotlin
// 及时关闭客户端和服务器
fun useMCP() {
    val client = Client(...)
    val httpClient = HttpClient()
    
    try {
        // 使用
    } finally {
        client.close()      // 关闭客户端
        httpClient.close()  // 关闭 HTTP 客户端
    }
}

// 使用 Kotlin 的 use 函数
fun useMCPWithUse() {
    Client(...).use { client ->
        HttpClient().use { httpClient ->
            // 使用
        }
    }
}
```

### 5. 并发问题

**问题**: 多个客户端同时连接导致问题

**解决方案**:
```kotlin
// 为每个连接创建独立的服务器实例
class MCPServerManager {
    private val servers = ConcurrentHashMap<String, Server>()
    
    fun createServer(clientId: String): Server {
        return servers.getOrPut(clientId) {
            Server(
                serverInfo = Implementation("server-$clientId", "1.0.0")
            )
        }
    }
    
    fun removeServer(clientId: String) {
        servers.remove(clientId)
    }
}
```

---

## 参考资源

### 官方文档

- [MCP Kotlin SDK GitHub](https://github.com/modelcontextprotocol/kotlin-sdk)
- [Model Context Protocol 官方文档](https://modelcontextprotocol.io/docs)
- [MCP 规范](https://spec.modelcontextprotocol.io/)
- [MCP Inspector](https://github.com/modelcontextprotocol/inspector)

### 示例项目

- [kotlin-mcp-server](https://github.com/modelcontextprotocol/kotlin-sdk/tree/main/samples/kotlin-mcp-server) - Streamable HTTP 服务器示例
- [weather-stdio-server](https://github.com/modelcontextprotocol/kotlin-sdk/tree/main/samples/weather-stdio-server) - STDIO 天气服务器
- [kotlin-mcp-client](https://github.com/modelcontextprotocol/kotlin-sdk/tree/main/samples/kotlin-mcp-client) - 交互式 STDIO 客户端
- [McpClient Notebook](https://github.com/modelcontextprotocol/kotlin-sdk/tree/main/samples/notebooks/McpClient.ipynb) - 可运行的 Notebook 演示

### 社区资源

- [MCP 官方 Discord](https://discord.gg/modelcontextprotocol)
- [MCP GitHub 组织](https://github.com/modelcontextprotocol)

### 相关项目

- [LM Studio](https://lmstudio.ai/) - 本地 LLM 推理
- [Claude Desktop](https://claude.ai/download) - 支持 MCP 的 Claude 桌面应用
- [Cursor](https://cursor.sh/) - 支持 MCP 的代码编辑器

---

## 总结

MCP Kotlin SDK 提供了完整的 Model Context Protocol 实现，支持:

1. **跨平台**: JVM、Native、JS、Wasm
2. **完整功能**: 客户端和服务端
3. **多种传输**: stdio、SSE、Streamable HTTP、WebSocket
4. **协程友好**: 基于 Kotlin 协程的异步 API
5. **类型安全**: 完整的 Kotlin 类型系统
6. **Ktor 集成**: 与 Ktor 无缝集成

**推荐使用场景**:
- AI 应用连接外部数据源
- 智能体工具调用
- 企业聊天机器人
- 本地 LLM 集成 (如 LM Studio)
- 标准化 AI 连接

对于 Kotlin 项目，MCP 提供了标准化的方式来连接 AI 模型和外部系统，是构建 AI 应用的理想选择。

---

## 更新记录

| 日期 | 版本 | 描述 |
|------|------|------|
| 2026-03-07 | 1.0 | 初始版本，整理 MCP Kotlin SDK 文档 |
