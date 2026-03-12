# 可行性测试：Ollama 集成

## 测试目的

验证 Ollama 作为 OpenAI API 格式模型提供商的可行性，包括:

1. **OpenAI 兼容 API**: 验证 `/v1/chat/completions` 和 `/v1/models` 接口
2. **Embedding API**: 验证 `/v1/embeddings` 接口
3. **原生 API**: 验证 `/api/show` 接口获取模型详情

## 测试背景

### 需求来源

用户希望项目支持 OpenAI API 格式的模型提供商 (如 Ollama)，但这类 API 的 `/v1/models` 接口不返回模型上下文和支持能力信息，需要手动配置。

同时，需要解决 Embedding 模型来源问题 (不使用 LM Studio 时)。

### 测试环境

- **Ollama 地址**: `http://127.0.0.1:11451`
- **测试模型**: `jaahas/qwen3.5-uncensored:9b`
- **超时时间**: 20 分钟 (因模型运行较慢)
- **Embedding 模型**: `nomic-embed-text` (可选)

## 项目结构

```
ollama-integration/
├── src/main/kotlin/
│   └── OllamaIntegrationTest.kt    # 测试源代码
├── build.gradle.kts                 # Gradle 构建配置
├── settings.gradle.kts              # Gradle 设置
├── gradle/                          # Gradle Wrapper
├── gradlew                          # Gradle Wrapper (Unix)
├── gradlew.bat                      # Gradle Wrapper (Windows)
└── README.md                        # 本说明文件
```

## 构建和运行

### 前置要求

- JDK 17 或更高版本
- Ollama 服务已启动
- 测试模型已下载 (`ollama pull jaahas/qwen3.5-uncensored:9b`)
- Embedding 模型已下载 (可选): `ollama pull nomic-embed-text`

### 运行测试

#### Windows:
```bash
gradlew.bat run
```

#### Linux/macOS:
```bash
./gradlew run
```

### 预期输出

```
=== 测试：Ollama 集成可行性 ===

Ollama 地址：http://127.0.0.1:11451
测试模型：jaahas/qwen3.5-uncensored:9b
超时时间：20 分钟

[测试 1] 检查 Ollama 服务可用性...
✓ Ollama 服务可用

[测试 2] 获取模型列表 (OpenAI 兼容 API)...
发现 X 个模型:
  1. jaahas/qwen3.5-uncensored:9b
  2. nomic-embed-text
✓ 测试模型 'jaahas/qwen3.5-uncensored:9b' 可用

[测试 3] 聊天补全测试 (OpenAI 兼容 API)...
模型：jaahas/qwen3.5-uncensored:9b
注意：首次请求可能需要加载模型，请耐心等待 (最多 20 分钟)

✓ 聊天响应成功!
响应 ID: chatcmpl-xxx
使用模型：jaahas/qwen3.5-uncensored:9b
Token 使用:
  提示词：XX
  完成：XX
  总计：XX

助手回复:
──────────────────────────────────────────────────
你好！Kotlin 是一种现代、简洁、安全的编程语言...
──────────────────────────────────────────────────

[测试 4] Embedding 测试 (OpenAI 兼容 API)...
模型：nomic-embed-text

✓ Embedding 响应成功!
模型：nomic-embed-text
生成 2 个嵌入向量
  向量 0: 维度 768, 前 5 个值：0.1234, -0.5678, 0.9012, -0.3456, 0.7890
  向量 1: 维度 768, 前 5 个值：0.2345, -0.6789, 0.0123, -0.4567, 0.8901
Token 使用:
  提示词：XX
  总计：XX

[测试 5] 获取模型详情 (Ollama 原生 API)...
使用 /api/show 接口获取模型详细信息

✓ 模型详情获取成功!
模型详细信息:
  格式：gguf
  家族：qwen2
  参数量：9B
  量化级别：Q4_K_M

=== ✓ 所有测试完成 ===

测试结论:
1. ✓ Ollama OpenAI 兼容 API 可用 (chat/completions, models)
2. ✓ Ollama Embedding API 可用 (embeddings)
3. ✓ Ollama 原生 API 可获取模型详情 (/api/show)

关键发现:
- OpenAI 兼容 API 的 /v1/models 接口不返回 context_length 等详细能力
- 需要使用原生 API /api/show 获取模型详细信息
- Embedding 模型需要单独下载和配置
- 模型加载较慢，建议生产环境预加载模型

配置建议:
1. 手动配置模型的 context_length、supports_vision 等能力
2. 使用 /api/show 接口在启动时自动探测模型能力
3. Embedding 模型独立配置，推荐使用 nomic-embed-text
```

## 测试内容详解

### 测试 1: Ollama 服务可用性检查

验证 Ollama 服务是否正常运行。

**实现方式**: 调用 `/api/tags` 接口

**预期结果**: 返回 HTTP 200 OK

### 测试 2: 模型列表获取 (OpenAI 兼容 API)

验证 `/v1/models` 接口返回模型列表。

**关键点**:
- Ollama 返回的模型信息**不包含** `context_length`、`supports_vision` 等详细能力
- 仅返回模型 ID、创建时间、所有者等基本信息

**实现方式**: 调用 `/v1/models` 接口

**预期响应**:
```json
{
  "object": "list",
  "data": [
    {
      "id": "jaahas/qwen3.5-uncensored:9b",
      "object": "model",
      "created": 1234567890,
      "owned_by": "ollama"
    }
  ]
}
```

### 测试 3: 聊天补全 (OpenAI 兼容 API)

验证 `/v1/chat/completions` 接口进行对话。

**实现方式**: 使用 OpenAI 兼容格式发送聊天请求

**请求示例**:
```json
{
  "model": "jaahas/qwen3.5-uncensored:9b",
  "messages": [
    {"role": "user", "content": "你好！请用中文介绍一下 Kotlin 的特点。"}
  ],
  "stream": false,
  "max_tokens": 500,
  "temperature": 0.7
}
```

**响应示例**:
```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "jaahas/qwen3.5-uncensored:9b",
  "choices": [
    {
      "index": 0,
      "message": {"role": "assistant", "content": "你好！Kotlin 是一种..."},
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 100,
    "total_tokens": 120
  }
}
```

### 测试 4: Embedding 测试 (OpenAI 兼容 API)

验证 `/v1/embeddings` 接口生成嵌入向量。

**实现方式**: 使用 OpenAI 兼容格式发送 embedding 请求

**请求示例**:
```json
{
  "model": "nomic-embed-text",
  "input": ["你好，世界", "这是一个测试"]
}
```

**预期响应**:
```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [0.1234, -0.5678, ...],
      "index": 0
    }
  ],
  "model": "nomic-embed-text",
  "usage": {
    "prompt_tokens": 10,
    "total_tokens": 10
  }
}
```

### 测试 5: 获取模型详情 (Ollama 原生 API)

验证 `/api/show` 接口获取模型详细信息。

**关键点**:
- 这是获取模型详细能力信息的**唯一方式**
- 返回模型格式、家族、参数量、量化级别、聊天模板等信息

**实现方式**: 调用 `/api/show` 接口

**请求示例**:
```json
{
  "name": "jaahas/qwen3.5-uncensored:9b"
}
```

**预期响应**:
```json
{
  "license": "...",
  "modelfile": "...",
  "parameters": "...",
  "template": "...",
  "details": {
    "parent_model": "",
    "format": "gguf",
    "family": "qwen2",
    "families": ["qwen2"],
    "parameter_size": "9B",
    "quantization_level": "Q4_K_M"
  },
  "model_info": {
    "general.architecture": "qwen2",
    "general.parameter_count": 9000000000,
    ...
  }
}
```

## 配置建议

### 方案 1: 手动配置模型能力

由于 OpenAI 兼容 API 不返回模型详细能力，建议在配置文件中手动指定:

```hocon
aiService {
  models {
    "jaahas/qwen3.5-uncensored:9b" {
      provider = "ollama"
      contextLength = 131072  # 128K (根据模型实际配置)
      supportsVision = true
      supportsTools = true
      maxTokens = 32768
      inputModalities = ["text", "image"]
      outputModalities = ["text"]
    }
  }
}
```

### 方案 2: 自动探测模型能力

在应用启动时，使用 `/api/show` 接口自动探测模型能力:

```kotlin
suspend fun detectModelCapabilities(modelName: String): ModelCapabilities {
    val details = ollamaClient.getModelDetails(modelName)
    
    // 从模型信息中推断能力
    val contextLength = details.model_info?.get("general.context_length") as? Int ?: 128000
    val parameterSize = details.details?.parameter_size ?: "unknown"
    
    // 根据模型家族推断能力
    val supportsVision = details.details?.family?.contains("qwen") == true
    val supportsTools = details.details?.family?.contains("qwen") == true
    
    return ModelCapabilities(
        contextLength = contextLength,
        supportsVision = supportsVision,
        supportsTools = supportsTools,
        parameterSize = parameterSize
    )
}
```

### Embedding 模型配置

推荐独立配置 Embedding 模型:

```hocon
embedding {
  provider = "ollama"
  baseUrl = "http://localhost:11434/v1"
  model = "nomic-embed-text"
  apiKey = ""  # Ollama 默认不需要 API Key
}
```

## 推荐模型

### LLM 模型 (支持 100K+ 上下文)

| 模型 | 上下文 | 参数量 | 备注 |
|------|--------|--------|------|
| `qwen2.5:72b` | 128K | 72B | 需要约 140GB 显存 |
| `qwen2.5:32b` | 128K | 32B | 需要约 64GB 显存 |
| `qwen2.5:14b` | 128K | 14B | 需要约 28GB 显存 |
| `qwen2.5:7b` | 128K | 7B | 需要约 14GB 显存 |

### Embedding 模型

| 模型 | 维度 | 大小 | 备注 |
|------|------|------|------|
| `nomic-embed-text` | 768 | ~270MB | 轻量高效，推荐 |
| `mxbai-embed-large` | 1024 | ~670MB | 效果更好，稍大 |

## 已知限制

1. **模型信息不完整**: OpenAI 兼容 API 的 `/v1/models` 接口不返回详细能力
2. **模型加载慢**: 首次请求需要加载模型，可能需要数分钟
3. **资源消耗**: 大模型需要大量显存和内存
4. **Embedding 独立**: Embedding 模型需要单独下载和配置

## 解决方案

1. **手动配置或自动探测**: 使用配置文件手动指定，或启动时调用 `/api/show` 自动探测
2. **预加载模型**: 生产环境预加载模型，避免首次请求延迟
3. **模型量化**: 使用量化版本降低资源消耗 (如 Q4_K_M)
4. **独立 Embedding 配置**: Embedding 模型独立配置，与 LLM provider 解耦

## 测试结果

### 最终测试状态 (2026-03-12)

✅ **所有测试通过 (5/5)**

| 测试项 | 状态 | 说明 |
|--------|------|------|
| 1. Ollama 服务可用性检查 | ✅ 完全通过 | 服务正常运行 |
| 2. 模型列表获取 (OpenAI API) | ✅ 完全通过 | 成功获取 15 个模型 |
| 3. 聊天补全 (OpenAI API) | ✅ 完全通过 | 响应成功，Token 使用正常 |
| 4. Embedding 测试 (OpenAI API) | ✅ **已修复** | 成功生成 2 个 768 维向量 |
| 5. 获取模型详情 (原生 API) | ✅ **已修复** | 成功获取模型详情 |

### 问题修复记录

#### 问题 1: Embedding 返回 0 个向量 (已修复)

**原因**: Embedding 模型未加载

**解决方案**: 
```bash
ollama pull nomic-embed-text
```

**验证结果**: ✅ 成功生成 2 个 768 维向量

#### 问题 2: 模型详情 JSON 解析失败 (已修复)

**原因**: Ollama 返回的 JSON 包含非标准格式

**解决方案**: 使用原始 JSON 字符串方式处理 `model_info` 字段

**验证结果**: ✅ 成功获取模型详情

## 参考

- [Ollama 官方文档](https://ollama.ai/docs)
- [Ollama API 参考](https://github.com/ollama/ollama/blob/main/docs/api.md)
- [OpenAI 兼容 API](https://platform.openai.com/docs/api-reference)
- [nomic-embed-text 模型](https://ollama.ai/library/nomic-embed-text)
