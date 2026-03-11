# LM Studio 模型信息查询测试

## 测试目的
验证从 LM Studio 获取模型最大 token 长度的可行性

## 测试端点

### 1. 列出所有模型
**端点**: `GET /v1/models`

**响应示例**:
```json
{
  "object": "list",
  "data": [
    {
      "id": "qwen3.5-9b-uncensored-hauhaucs-aggressive",
      "object": "model",
      "created": 1234567890,
      "owned_by": "lmstudio"
    }
  ]
}
```

### 2. 获取单个模型详情
**端点**: `GET /api/v1/models/{modelId}`

**预期响应**:
```json
{
  "id": "qwen3.5-9b",
  "context_length": 32768,
  "max_tokens": 32768
}
```

## 测试代码

```kotlin
import io.ktor.client.*
import io.ktor.client.engine.cio.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import kotlinx.serialization.*
import kotlinx.serialization.json.*

@Serializable
data class ModelList(
    val `object`: String,
    val data: List<ModelInfo>
)

@Serializable
data class ModelInfo(
    val id: String,
    val `object`: String,
    val created: Long,
    val owned_by: String
)

@Serializable
data class ModelDetails(
    val id: String,
    val context_length: Int?,
    val max_tokens: Int?
)

suspend fun testGetModels() {
    val client = HttpClient(CIO) {
        engine {
            endpoint {
                connectTimeout = 5000
                requestTimeout = 10000
            }
        }
    }
    
    try {
        // 测试 1: 获取模型列表
        println("=== 测试 1: 获取模型列表 ===")
        val response = client.get("http://localhost:1234/v1/models")
        println("状态码：${response.status}")
        println("响应：${response.bodyAsText()}")
        
        // 测试 2: 获取模型详情
        println("\n=== 测试 2: 获取模型详情 ===")
        val modelId = "qwen3.5-9b-uncensored-hauhaucs-aggressive"
        val detailsResponse = client.get("http://localhost:1234/api/v1/models/$modelId")
        println("状态码：${detailsResponse.status}")
        println("响应：${detailsResponse.bodyAsText()}")
        
    } catch (e: Exception) {
        println("错误：${e.message}")
        e.printStackTrace()
    } finally {
        client.close()
    }
}
```

## 备选方案

如果 LM Studio 不提供模型详情 API，使用以下备选方案：

### 方案 1: 配置文件手动指定
```hocon
aiService {
    modelName = "qwen3.5-9b"
    modelMaxTokens = 32768  # 手动配置
}
```

### 方案 2: 从模型文件名推断
某些模型文件名包含上下文长度信息：
- `qwen3.5-9b-32k.gguf` → 32768 tokens
- `llama-3-8b-8k.gguf` → 8192 tokens

### 方案 3: 使用默认值
根据模型大小使用保守的默认值：
- 小型模型 (1-3B): 4096
- 中型模型 (7-14B): 8192
- 大型模型 (20B+): 16384

## 测试结果记录

### 测试环境
- LM Studio 版本：待填写
- 模型：待填写
- 测试时间：2026-03-10

### 测试执行

#### 测试 1: 获取模型列表
**命令**: `curl http://localhost:1234/v1/models`

**结果**: 待填写

#### 测试 2: 获取模型详情
**命令**: `curl http://localhost:1234/api/v1/models/{modelId}`

**结果**: 待填写

### 结论
待测试完成后填写
