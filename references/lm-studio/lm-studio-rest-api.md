# LM Studio REST API 参考指南

> **版本**: 0.4.0+ (v1 API)  
> **更新日期**: 2026-03-07  
> **官方文档**: https://lmstudio.ai/docs/developer/rest

---

## 目录

- [概述](#概述)
- [API 版本](#api 版本)
- [核心特性](#核心特性)
- [端点列表](#端点列表)
- [推理端点对比](#推理端点对比)
- [快速开始](#快速开始)
- [SDK 使用](#sdk 使用)
- [REST API 详细用法](#rest-api 详细用法)
- [模型管理](#模型管理)
- [认证配置](#认证配置)
- [Kotlin 集成方案](#kotlin 集成方案)
- [最佳实践](#最佳实践)

---

## 概述

LM Studio 提供强大的 REST API，支持本地推理和模型管理。除了原生 API 外，还提供与 OpenAI 和 Anthropic 兼容的端点。

### 主要优势

- **本地推理**: 在本地运行大语言模型，无需联网
- **模型管理**: 下载、加载、卸载模型的完整生命周期管理
- **多协议支持**: 原生 API + OpenAI 兼容 + Anthropic 兼容
- **流式输出**: 支持实时流式响应
- **状态化聊天**: 支持有状态的对话管理
- **MCP 集成**: 支持 Model Context Protocol

---

## API 版本

### v1 REST API (推荐)

LM Studio 0.4.0+ 版本正式发布了原生 v1 REST API，端点路径为 `/api/v1/*`。

**v1 API 新增特性**:
- MCP via API (Model Context Protocol)
- Stateful chats (状态化聊天)
- Authentication configuration with API tokens (API Token 认证)
- Model download, load and unload endpoints (模型下载、加载、卸载)

### v0 REST API (已弃用)

0.4.0 版本之前使用 v0 API，现已不推荐使用。

---

## 核心特性

### 1. 聊天和文本生成

- 支持流式输出
- 支持状态化聊天
- 支持自定义工具调用
- 支持在请求中包含助手消息

### 2. 模型管理

- 列出可用模型
- 加载模型到内存
- 从内存卸载模型
- 下载新模型
- 查看下载状态

### 3. MCP 支持

- Remote MCPs (远程 MCP)
- MCPs you have in LM Studio (本地 MCP)
- 通过 API 调用 MCP

### 4. 高级特性

- 自定义工具调用
- 指定上下文长度
- 模型加载流式事件
- 提示处理流式事件

---

## 端点列表

### 原生 v1 API

| 端点 | 方法 | 描述 | 文档 |
|------|------|------|------|
| `/api/v1/chat` | POST | 聊天/文本生成 | [文档](https://lmstudio.ai/docs/developer/rest) |
| `/api/v1/models` | GET | 列出所有模型 | [文档](https://lmstudio.ai/docs/developer/rest) |
| `/api/v1/models/load` | POST | 加载模型 | [文档](https://lmstudio.ai/docs/developer/rest) |
| `/api/v1/models/unload` | POST | 卸载模型 | [文档](https://lmstudio.ai/docs/developer/rest) |
| `/api/v1/models/download` | POST | 下载模型 | [文档](https://lmstudio.ai/docs/developer/rest) |
| `/api/v1/models/download/status` | GET | 下载状态 | [文档](https://lmstudio.ai/docs/developer/rest) |

### OpenAI 兼容端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/v1/chat/completions` | POST | 聊天完成 (OpenAI 兼容) |
| `/v1/completions` | POST | 文本完成 (OpenAI 兼容) |
| `/v1/embeddings` | POST | 嵌入向量 (OpenAI 兼容) |

### Anthropic 兼容端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/v1/messages` | POST | 消息 (Anthropic 兼容) |

---

## 推理端点对比

| 特性 | `/api/v1/chat` (原生) | `/v1/responses` | `/v1/chat/completions` (OpenAI) | `/v1/messages` (Anthropic) |
|------|----------------------|----------------|--------------------------------|---------------------------|
| **流式输出** | ✅ | ✅ | ✅ | ✅ |
| **状态化聊天** | ✅ | ✅ | ❌ | ❌ |
| **远程 MCPs** | ✅ | ✅ | ❌ | ❌ |
| **本地 MCPs** | ✅ | ✅ | ❌ | ❌ |
| **自定义工具** | ❌ | ✅ | ✅ | ✅ |
| **包含助手消息** | ❌ | ✅ | ✅ | ✅ |
| **模型加载流式事件** | ✅ | ❌ | ❌ | ❌ |
| **提示处理流式事件** | ✅ | ❌ | ❌ | ❌ |
| **指定上下文长度** | ✅ | ❌ | ❌ | ❌ |

**推荐使用**:
- **原生 API** (`/api/v1/chat`): 需要 LM Studio 特有功能时 (模型管理、MCP、流式事件等)
- **OpenAI 兼容**: 已有 OpenAI 集成代码，需要快速迁移
- **Anthropic 兼容**: 已有 Anthropic 集成代码

---

## 快速开始

### 1. 安装 LM Studio

**Windows**:
```powershell
irm https://lmstudio.ai/install.ps1 | iex
```

**macOS / Linux**:
```bash
curl -fsSL https://lmstudio.ai/install.sh | bash
```

### 2. 启动本地服务器

```bash
lms server start --port 1234
```

### 3. 下载模型

```bash
lms get <model>
# 例如：lms get llama-3.2-1b-instruct
```

### 4. 基本使用

#### 使用 cURL

```bash
curl http://localhost:1234/api/v1/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $LM_API_TOKEN" \
  -d '{
    "model": "openai/gpt-oss-20b",
    "input": "Who are you, and what can you do?"
  }'
```

#### 使用 Python (官方 SDK)

```python
import lmstudio as lms

with lms.Client() as client:
    model = client.llm.model("openai/gpt-oss-20b")
    result = model.respond("Who are you, and what can you do?")
    print(result)
```

#### 使用 TypeScript (官方 SDK)

```typescript
import { LMStudioClient } from "@lmstudio/sdk";

const client = new LMStudioClient();
const model = await client.llm.model("openai/gpt-oss-20b");
const result = await model.respond("Who are you, and what can you do?");

console.info(result.content);
```

#### 使用 OpenAI SDK (兼容模式)

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:1234/v1",
    api_key="not-needed"  # LM Studio 不需要 API 密钥
)

response = client.chat.completions.create(
    model="local-model",  # 填写已加载的本地模型名
    messages=[
        {"role": "user", "content": "Hello!"}
    ]
)

print(response.choices[0].message.content)
```

---

## SDK 使用

### lmstudio-python

#### 安装

```bash
pip install lmstudio
```

#### 基本用法

```python
import lmstudio as lms

# 创建客户端 (同步)
with lms.Client() as client:
    # 获取模型
    model = client.llm.model("llama-3.2-1b-instruct")
    
    # 文本完成
    result = model.complete("Once upon a time,")
    print(result)
    
    # 聊天响应
    chat = lms.Chat("You are a helpful assistant")
    chat.add_user_message("Hello!")
    response = model.respond(chat)
    chat.add_assistant_response(response)
    print(response)
```

#### 流式输出

```python
import lmstudio as lms

model = lms.llm()
chat = lms.Chat()
chat.add_user_message("Tell me a story")

# 流式响应
for chunk in model.respond_stream(chat):
    print(chunk, end="", flush=True)
```

#### 模型管理

```python
import lmstudio as lms

client = lms.Client()

# 列出模型
models = client.llm.list_loaded_models()
for model in models:
    print(f"Model: {model.identifier}")

# 加载模型
model = client.llm.model("llama-3.2-1b-instruct")

# 卸载模型
client.llm.unload("llama-3.2-1b-instruct")
```

### lmstudio-js (TypeScript/JavaScript)

#### 安装

```bash
npm install @lmstudio/sdk
```

#### 基本用法

```typescript
import { LMStudioClient } from "@lmstudio/sdk";

const client = new LMStudioClient();

// 获取模型
const model = await client.llm.model("llama-3.2-1b-instruct");

// 文本响应
const result = await model.respond("What is the meaning of life?");
console.info(result.content);

// 流式响应
const stream = await model.respondStream("Tell me a story");
for await (const chunk of stream) {
    process.stdout.write(chunk);
}
```

#### 聊天管理

```typescript
import { LMStudioClient } from "@lmstudio/sdk";

const client = new LMStudioClient();
const model = await client.llm.model("llama-3.2-1b-instruct");

// 创建聊天会话
const chat = client.llm.chatSystem();
chat.addUserMessage("Hello!");

const response = await model.respond(chat.getHistory());
chat.addAssistantMessage(response.content);
```

#### 模型管理

```typescript
import { LMStudioClient } from "@lmstudio/sdk";

const client = new LMStudioClient();

// 列出模型
const models = await client.llm.listLoadedModels();
models.forEach(model => {
    console.log(`Model: ${model.identifier}`);
});

// 加载模型
const model = await client.llm.model("llama-3.2-1b-instruct");

// 卸载模型
await client.llm.unload("llama-3.2-1b-instruct");

// 下载模型
const download = await client.system.downloadModel("llama-3.2-1b-instruct");
download.watch((progress) => {
    console.log(`Download progress: ${progress.percent}%`);
});
```

---

## REST API 详细用法

### 聊天端点 (`/api/v1/chat`)

#### 请求示例

```bash
curl http://localhost:1234/api/v1/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-token" \
  -d '{
    "model": "llama-3.2-1b-instruct",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "Hello!"}
    ],
    "stream": false,
    "maxTokens": 100,
    "temperature": 0.7,
    "contextLength": 4096
  }'
```

#### 响应示例 (非流式)

```json
{
  "id": "chat-123",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "llama-3.2-1b-instruct",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finishReason": "stop"
    }
  ],
  "usage": {
    "promptTokens": 10,
    "completionTokens": 8,
    "totalTokens": 18
  }
}
```

#### 流式响应

```bash
curl http://localhost:1234/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama-3.2-1b-instruct",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "stream": true
  }'
```

流式响应会返回多个 SSE (Server-Sent Events) 数据块:

```
data: {"id":"chat-123","choices":[{"delta":{"content":"Hello"}}]}

data: {"id":"chat-123","choices":[{"delta":{"content":"!"}}]}

data: {"id":"chat-123","choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

### 模型列表端点 (`/api/v1/models`)

#### 请求示例

```bash
curl http://localhost:1234/api/v1/models \
  -H "Authorization: Bearer your-api-token"
```

#### 响应示例

```json
{
  "data": [
    {
      "id": "llama-3.2-1b-instruct",
      "object": "model",
      "created": 1234567890,
      "ownedBy": "meta",
      "type": "llm"
    },
    {
      "id": "gemma-2b",
      "object": "model",
      "created": 1234567891,
      "ownedBy": "google",
      "type": "llm"
    }
  ]
}
```

### 模型加载端点 (`/api/v1/models/load`)

#### 请求示例

```bash
curl http://localhost:1234/api/v1/models/load \
  -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-token" \
  -d '{
    "model": "llama-3.2-1b-instruct",
    "contextLength": 4096,
    "gpuOffload": {
      "enabled": true,
      "layers": 10
    }
  }'
```

#### 响应示例

```json
{
  "id": "llama-3.2-1b-instruct",
  "status": "loaded",
  "memoryUsage": {
    "ram": 2048000000,
    "vram": 1024000000
  }
}
```

### 模型卸载端点 (`/api/v1/models/unload`)

#### 请求示例

```bash
curl http://localhost:1234/api/v1/models/unload \
  -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-token" \
  -d '{
    "model": "llama-3.2-1b-instruct"
  }'
```

#### 响应示例

```json
{
  "id": "llama-3.2-1b-instruct",
  "status": "unloaded"
}
```

### 模型下载端点 (`/api/v1/models/download`)

#### 请求示例

```bash
curl http://localhost:1234/api/v1/models/download \
  -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-token" \
  -d '{
    "model": "llama-3.2-1b-instruct"
  }'
```

#### 响应示例

```json
{
  "id": "download-123",
  "model": "llama-3.2-1b-instruct",
  "status": "downloading",
  "progress": {
    "downloadedBytes": 1048576,
    "totalBytes": 2097152000,
    "percent": 0.05
  }
}
```

### 下载状态端点 (`/api/v1/models/download/status`)

#### 请求示例

```bash
curl http://localhost:1234/api/v1/models/download/status/download-123 \
  -H "Authorization: Bearer your-api-token"
```

#### 响应示例

```json
{
  "id": "download-123",
  "model": "llama-3.2-1b-instruct",
  "status": "completed",
  "progress": {
    "downloadedBytes": 2097152000,
    "totalBytes": 2097152000,
    "percent": 100
  }
}
```

---

## 模型管理

### 命令行工具 (lms CLI)

#### 基本命令

```bash
# 启动守护进程
lms daemon up

# 下载模型
lms get <model>

# 启动本地服务器
lms server start

# 打开交互式聊天
lms chat

# 列出已下载模型
lms list

# 查看系统状态
lms status
```

#### 常用命令示例

```bash
# 下载 Qwen 模型
lms get Qwen2.5-7B-Instruct

# 下载 Llama 模型
lms get llama-3.2-1b-instruct

# 下载 Gemma 模型
lms get gemma-2b

# 启动服务器并指定端口
lms server start --port 1234

# 查看已加载的模型
lms list

# 查看系统资源使用情况
lms status
```

### 模型推荐

#### 小型模型 (1-3B 参数)

- `llama-3.2-1b-instruct`: Meta 的轻量级模型
- `Qwen2.5-1.5B-Instruct`: 阿里巴巴的轻量级模型
- `gemma-2b`: Google 的轻量级模型

#### 中型模型 (7-14B 参数)

- `llama-3.2-3b-instruct`: Meta 的中型模型
- `Qwen2.5-7B-Instruct`: 阿里巴巴的中型模型
- `mistral-7b-instruct`: Mistral AI 的中型模型

#### 大型模型 (20B+ 参数)

- `openai/gpt-oss-20b`: OpenAI 的开源模型
- `Qwen2.5-14B-Instruct`: 阿里巴巴的大型模型
- `mixtral-8x7b`: Mistral AI 的混合专家模型

---

## 认证配置

### API Token 认证

LM Studio v1 API 支持 API Token 认证。

#### 设置 API Token

```bash
# 设置环境变量
export LM_API_TOKEN="your-api-token"

# Windows PowerShell
$env:LM_API_TOKEN="your-api-token"
```

#### 使用 API Token

```bash
curl http://localhost:1234/api/v1/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $LM_API_TOKEN" \
  -d '{
    "model": "llama-3.2-1b-instruct",
    "input": "Hello!"
  }'
```

#### 无需认证 (本地开发)

在本地开发环境中，可以跳过认证:

```bash
curl http://localhost:1234/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama-3.2-1b-instruct",
    "input": "Hello!"
  }'
```

---

## Kotlin 集成方案

### 使用 Ktor 客户端集成

#### 添加依赖

在 `gradle/libs.versions.toml` 中添加:

```toml
[versions]
ktor = "3.4.1"
kotlinx-serialization = "1.8.1"

[libraries]
ktor-client-core = { module = "io.ktor:ktor-client-core", version.ref = "ktor" }
ktor-client-cio = { module = "io.ktor:ktor-client-cio", version.ref = "ktor" }
ktor-client-content-negotiation = { module = "io.ktor:ktor-client-content-negotiation", version.ref = "ktor" }
ktor-serialization-kotlinx-json = { module = "io.ktor:ktor-serialization-kotlinx-json", version.ref = "ktor" }
kotlinx-serialization-json = { module = "org.jetbrains.kotlinx:kotlinx-serialization-json", version.ref = "kotlinx-serialization" }
```

在共享模块的 `build.gradle.kts` 中添加:

```kotlin
sourceSets {
    commonMain {
        dependencies {
            implementation(libs.ktor.client.core)
            implementation(libs.ktor.client.content.negotiation)
            implementation(libs.ktor.serialization.kotlinx.json)
            implementation(libs.kotlinx.serialization.json)
        }
    }
    
    jvmMain {
        dependencies {
            implementation(libs.ktor.client.cio)
        }
    }
}
```

#### 创建 LM Studio 客户端

```kotlin
import io.ktor.client.*
import io.ktor.client.engine.cio.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.request.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.serialization.json.Json

class LMStudioClient(
    private val baseUrl: String = "http://localhost:1234",
    private val apiToken: String? = null
) {
    private val client = HttpClient(CIO) {
        install(ContentNegotiation) {
            json(Json {
                prettyPrint = true
                isLenient = true
                ignoreUnknownKeys = true
            })
        }
    }
    
    suspend fun chat(
        model: String,
        messages: List<Message>,
        stream: Boolean = false,
        maxTokens: Int = 100,
        temperature: Double = 0.7
    ): ChatResponse {
        return client.post("$baseUrl/api/v1/chat") {
            contentType(ContentType.Application.Json)
            apiToken?.let { header("Authorization", "Bearer $it") }
            setBody(
                ChatRequest(
                    model = model,
                    messages = messages,
                    stream = stream,
                    maxTokens = maxTokens,
                    temperature = temperature
                )
            )
        }.body()
    }
    
    suspend fun listModels(): List<ModelInfo> {
        return client.get("$baseUrl/api/v1/models") {
            apiToken?.let { header("Authorization", "Bearer $it") }
        }.body<ListModelsResponse>().data
    }
    
    suspend fun loadModel(
        model: String,
        contextLength: Int = 4096,
        gpuOffload: Boolean = true,
        gpuLayers: Int = 10
    ): LoadModelResponse {
        return client.post("$baseUrl/api/v1/models/load") {
            contentType(ContentType.Application.Json)
            apiToken?.let { header("Authorization", "Bearer $it") }
            setBody(
                LoadModelRequest(
                    model = model,
                    contextLength = contextLength,
                    gpuOffload = GpuOffload(enabled = gpuOffload, layers = gpuLayers)
                )
            )
        }.body()
    }
    
    suspend fun unloadModel(model: String): UnloadModelResponse {
        return client.post("$baseUrl/api/v1/models/unload") {
            contentType(ContentType.Application.Json)
            apiToken?.let { header("Authorization", "Bearer $it") }
            setBody(UnloadModelRequest(model = model))
        }.body()
    }
    
    fun close() {
        client.close()
    }
}
```

#### 数据模型

```kotlin
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class Message(
    val role: String,
    val content: String
) {
    companion object {
        fun system(content: String) = Message("system", content)
        fun user(content: String) = Message("user", content)
        fun assistant(content: String) = Message("assistant", content)
    }
}

@Serializable
data class ChatRequest(
    val model: String,
    val messages: List<Message>,
    val stream: Boolean = false,
    val maxTokens: Int = 100,
    val temperature: Double = 0.7,
    val contextLength: Int = 4096
)

@Serializable
data class ChatResponse(
    val id: String,
    val `object`: String,
    val created: Long,
    val model: String,
    val choices: List<Choice>,
    val usage: Usage
)

@Serializable
data class Choice(
    val index: Int,
    val message: Message,
    val finishReason: String
)

@Serializable
data class Usage(
    val promptTokens: Int,
    val completionTokens: Int,
    val totalTokens: Int
)

@Serializable
data class ListModelsResponse(
    val data: List<ModelInfo>
)

@Serializable
data class ModelInfo(
    val id: String,
    val `object`: String,
    val created: Long,
    val ownedBy: String,
    val type: String
)

@Serializable
data class LoadModelRequest(
    val model: String,
    val contextLength: Int = 4096,
    val gpuOffload: GpuOffload
)

@Serializable
data class GpuOffload(
    val enabled: Boolean,
    val layers: Int
)

@Serializable
data class LoadModelResponse(
    val id: String,
    val status: String,
    val memoryUsage: MemoryUsage
)

@Serializable
data class MemoryUsage(
    val ram: Long,
    val vram: Long
)

@Serializable
data class UnloadModelRequest(
    val model: String
)

@Serializable
data class UnloadModelResponse(
    val id: String,
    val status: String
)
```

#### 使用示例

```kotlin
import kotlinx.coroutines.runBlocking

fun main() = runBlocking {
    val client = LMStudioClient()
    
    try {
        // 列出可用模型
        val models = client.listModels()
        println("可用模型:")
        models.forEach { model ->
            println("  - ${model.id} (${model.ownedBy})")
        }
        
        // 加载模型
        println("\n加载模型...")
        val loadResult = client.loadModel("llama-3.2-1b-instruct")
        println("模型加载状态：${loadResult.status}")
        println("内存使用：RAM=${loadResult.memoryUsage.ram / 1024 / 1024}MB, VRAM=${loadResult.memoryUsage.vram / 1024 / 1024}MB")
        
        // 聊天
        println("\n发送消息...")
        val messages = listOf(
            Message.system("You are a helpful assistant."),
            Message.user("Hello! How are you?")
        )
        
        val response = client.chat(
            model = "llama-3.2-1b-instruct",
            messages = messages,
            maxTokens = 200,
            temperature = 0.7
        )
        
        val assistantMessage = response.choices.first().message
        println("助手回复：${assistantMessage.content}")
        println("Token 使用：${response.usage.totalTokens}")
        
        // 卸载模型
        println("\n卸载模型...")
        val unloadResult = client.unloadModel("llama-3.2-1b-instruct")
        println("模型卸载状态：${unloadResult.status}")
        
    } finally {
        client.close()
    }
}
```

### 使用 OpenAI 兼容模式

#### 添加依赖

```kotlin
// 使用 Ktor 客户端 (同上)
// 或使用 OpenAI SDK 的 Kotlin 封装
```

#### 创建 OpenAI 兼容客户端

```kotlin
import io.ktor.client.*
import io.ktor.client.request.*
import io.ktor.http.*

class OpenAICompatibleClient(
    private val baseUrl: String = "http://localhost:1234/v1",
    private val apiKey: String = "not-needed"
) {
    private val client = HttpClient {
        install(ContentNegotiation) {
            json(Json {
                prettyPrint = true
                isLenient = true
                ignoreUnknownKeys = true
            })
        }
    }
    
    suspend fun chat(
        model: String,
        messages: List<Message>
    ): ChatCompletionResponse {
        return client.post("$baseUrl/chat/completions") {
            contentType(ContentType.Application.Json)
            header("Authorization", "Bearer $apiKey")
            setBody(
                ChatCompletionRequest(
                    model = model,
                    messages = messages
                )
            )
        }.body()
    }
}

@Serializable
data class ChatCompletionRequest(
    val model: String,
    val messages: List<Message>
)

@Serializable
data class ChatCompletionResponse(
    val id: String,
    val `object`: String,
    val created: Long,
    val model: String,
    val choices: List<ChatCompletionChoice>,
    val usage: Usage
)

@Serializable
data class ChatCompletionChoice(
    val index: Int,
    val message: Message,
    val finishReason: String
)
```

---

## 最佳实践

### 1. 模型选择

- **开发测试**: 使用小型模型 (1-3B)，速度快，资源占用少
- **生产环境**: 根据任务复杂度选择中型 (7-14B) 或大型 (20B+) 模型
- **资源受限**: 使用量化版本 (如 Q4_K_M、Q5_K_M)

### 2. 性能优化

```kotlin
// 启用 GPU 加速
val loadResult = client.loadModel(
    model = "llama-3.2-1b-instruct",
    contextLength = 4096,
    gpuOffload = true,
    gpuLayers = 10  // 根据显存调整
)

// 调整上下文长度
val response = client.chat(
    model = "llama-3.2-1b-instruct",
    messages = messages,
    maxTokens = 200,
    temperature = 0.7,
    contextLength = 2048  // 减少上下文长度以提高速度
)
```

### 3. 错误处理

```kotlin
try {
    val response = client.chat(
        model = "llama-3.2-1b-instruct",
        messages = messages
    )
} catch (e: Exception) {
    when {
        e is HttpRequestTimeoutException -> {
            // 请求超时，重试或降级
            println("请求超时，请检查模型是否已加载")
        }
        e is ClientRequestException -> {
            // HTTP 错误，检查请求参数
            println("HTTP 错误：${e.response.status}")
        }
        else -> {
            // 其他错误
            println("未知错误：${e.message}")
        }
    }
}
```

### 4. 资源管理

```kotlin
// 使用完模型后及时卸载
client.unloadModel("llama-3.2-1b-instruct")

// 关闭客户端释放资源
client.close()

// 或使用 Kotlin 的 use 函数
LMStudioClient().use { client ->
    // 使用客户端
}
```

### 5. 流式处理

```kotlin
// 使用 Ktor 的 SSE 支持
client.post("$baseUrl/api/v1/chat") {
    contentType(ContentType.Application.Json)
    setBody(ChatRequest(model, messages, stream = true))
}.bodyAsText().let { response ->
    // 解析 SSE 流
    response.split("\n").forEach { line ->
        if (line.startsWith("data: ")) {
            val data = line.removePrefix("data: ")
            if (data != "[DONE]") {
                // 处理数据块
                println(data)
            }
        }
    }
}
```

### 6. 并发控制

```kotlin
// 限制并发请求数
val semaphore = Semaphore(2)  // 最多 2 个并发请求

suspend fun safeChat(messages: List<Message>): ChatResponse {
    semaphore.acquire()
    return try {
        client.chat(model = "llama-3.2-1b-instruct", messages = messages)
    } finally {
        semaphore.release()
    }
}
```

### 7. 缓存策略

```kotlin
// 缓存常用模型的响应
val cache = ConcurrentHashMap<String, ChatResponse>()

suspend fun cachedChat(key: String, messages: List<Message>): ChatResponse {
    return cache.getOrPut(key) {
        client.chat(model = "llama-3.2-1b-instruct", messages = messages)
    }
}
```

---

## 无头部署 (Headless Deployment)

### llmster

`llmster` 是 LM Studio 的核心，作为守护进程打包用于无头部署。

**特点**:
- 独立运行，不依赖 LM Studio GUI
- 适用于服务器、云实例、CI/CD
- 完整的 API 支持

**安装**:

```bash
# Mac / Linux
curl -fsSL https://lmstudio.ai/install.sh | bash

# Windows
irm https://lmstudio.ai/install.ps1 | iex
```

**基本用法**:

```bash
# 启动守护进程
lms daemon up

# 下载模型
lms get <model>

# 启动本地服务器
lms server start

# 打开交互式会话
lms chat
```

---

## 常见问题

### 1. 模型加载失败

**原因**:
- 内存不足
- 显存不足
- 模型文件损坏

**解决方案**:
```bash
# 使用更小的模型
lms get llama-3.2-1b-instruct

# 减少 GPU 层数
# 在加载请求中设置 gpuOffload.layers = 5

# 重新下载模型
lms get <model> --force
```

### 2. 响应速度慢

**原因**:
- 模型过大
- CPU/GPU 性能不足
- 上下文长度过长

**解决方案**:
```kotlin
// 使用量化模型
val model = "llama-3.2-1b-instruct-Q4_K_M"

// 减少上下文长度
val response = client.chat(
    model = model,
    messages = messages,
    contextLength = 2048  // 从 4096 减少到 2048
)

// 启用 GPU 加速
val loadResult = client.loadModel(
    model = model,
    gpuOffload = true,
    gpuLayers = 10
)
```

### 3. API 连接失败

**原因**:
- 服务器未启动
- 端口被占用
- 防火墙阻止

**解决方案**:
```bash
# 检查服务器状态
lms status

# 重启服务器
lms server restart

# 更改端口
lms server start --port 1235

# 检查防火墙设置
# Windows: 允许 LM Studio 通过防火墙
# macOS: 系统偏好设置 -> 安全性与隐私 -> 防火墙
```

---

## 参考资源

### 官方文档

- [LM Studio REST API](https://lmstudio.ai/docs/developer/rest)
- [LM Studio 开发者文档](https://lmstudio.ai/docs/developer)
- [lmstudio-js 文档](https://github.com/lmstudio-ai/lmstudio-js)
- [lmstudio-python 文档](https://github.com/lmstudio-ai/lmstudio-python)
- [本地服务器基础](https://lmstudio.ai/docs/developer/rest-api/local-server-basics)
- [API 更新日志](https://lmstudio.ai/docs/developer/rest-api/api-changelog)

### 社区资源

- [LM Studio Discord 社区](https://discord.gg/lmstudio)
- [LM Studio GitHub](https://github.com/lmstudio-ai)

### 相关项目

- [OpenAI Python SDK](https://github.com/openai/openai-python)
- [Anthropic Python SDK](https://github.com/anthropics/anthropic-sdk-python)

---

## 更新记录

| 日期 | 版本 | 描述 |
|------|------|------|
| 2026-03-07 | 1.0 | 初始版本，整理 LM Studio REST API 文档 |
| 2026-03-07 | 1.1 | 添加 Kotlin 集成方案和完整示例 |

---

## 总结

LM Studio 提供了强大的本地推理 API，支持:

1. **多种 API 格式**: 原生 v1 API、OpenAI 兼容、Anthropic 兼容
2. **完整模型管理**: 下载、加载、卸载、状态查询
3. **多语言 SDK**: Python、TypeScript/JavaScript
4. **高级特性**: 流式输出、MCP 支持、状态化聊天
5. **灵活部署**: GUI 模式、无头模式 (llmster)

**推荐使用场景**:
- 本地开发和测试
- 隐私敏感应用
- 离线环境部署
- 成本敏感项目
- 自定义模型部署

对于 Kotlin 项目，推荐使用 Ktor 客户端进行集成，或采用 OpenAI 兼容模式快速迁移现有代码。
