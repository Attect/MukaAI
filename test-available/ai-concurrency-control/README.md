# AI 模型请求并发控制和会话压缩阈值测试

## 测试目的
验证以下功能的可行性：
1. 从 LM Studio 获取模型最大 token 长度
2. 基于百分比的会话压缩阈值配置
3. AI 模型请求并发控制和队列管理

## 测试时间
2026-03-10

## 技术可行性

### 1. LM Studio 模型信息查询

#### 方案一：通过 REST API 获取模型信息
LM Studio 提供模型信息端点，可以获取模型的上下文长度（context length）。

**端点**: `GET /v1/models/{modelId}` 或 `GET /api/v1/models/{modelId}`

**响应示例**:
```json
{
  "id": "qwen3.5-9b",
  "context_length": 32768,
  "max_tokens": 32768
}
```

**可行性**: ✅ 可行
- LM Studio REST API 支持查询模型信息
- 包含 `context_length` 或 `max_tokens` 字段

#### 方案二：配置文件中指定最大 token 数
如果无法动态获取，可以在配置文件中手动指定。

**配置示例**:
```hocon
aiService {
    lmStudioUrl = "http://localhost:1234/v1"
    modelName = "qwen3.5-9b"
    modelMaxTokens = 32768  # 手动指定
}
```

**可行性**: ✅ 可行
- 简单直接
- 需要用户手动维护

### 2. 会话压缩阈值配置

**配置方案**:
```hocon
aiService {
    # 方法 1: 百分比方式（推荐）
    sessionCompression {
        enabled = true
        thresholdPercent = 0.8  # 80% 时触发压缩
    }
    
    # 方法 2: 固定 token 数
    sessionCompression {
        enabled = true
        thresholdTokens = 26214  # 32768 * 0.8
    }
    
    # 方法 3: 混合方式
    sessionCompression {
        enabled = true
        thresholdPercent = 0.8
        minThresholdTokens = 10000  # 最小阈值
        maxThresholdTokens = 30000  # 最大阈值
    }
}
```

**可行性**: ✅ 可行
- HOCON 配置支持嵌套结构
- 支持数值类型（Double/Int）

### 3. 并发控制和队列管理

#### 技术方案
使用 Kotlin 协程和 Channel 实现并发控制。

**核心实现**:
```kotlin
class AIRequestQueue(
    private val maxConcurrent: Int
) {
    private val semaphore = Semaphore(maxConcurrent)
    private val requestQueue = Channel<Request>(capacity = Channel.UNLIMITED)
    
    suspend fun submit(request: Request): Response {
        // 如果超过最大并发数，排队等待
        semaphore.acquire()
        try {
            return executeRequest(request)
        } finally {
            semaphore.release()
        }
    }
}
```

**配置方案**:
```hocon
aiService {
    # 并发控制
    concurrency {
        maxConcurrentRequests = 3  # 最大并发请求数
        queueEnabled = true        # 启用队列
        queueTimeout = 300000      # 队列超时时间（毫秒）
    }
}
```

**可行性**: ✅ 可行
- Kotlin 协程提供 Semaphore 原语
- Channel 支持无界队列
- 可实现 FIFO 先到先得

## 测试验证

### 测试 1: LM Studio 模型信息查询

**测试代码**:
```kotlin
import io.ktor.client.*
import io.ktor.client.request.*
import kotlinx.serialization.Serializable

@Serializable
data class ModelInfo(
    val id: String,
    val contextLength: Int,
    val maxTokens: Int
)

suspend fun getModelInfo(client: HttpClient, modelId: String): ModelInfo {
    return client.get("http://localhost:1234/api/v1/models/$modelId")
}
```

**预期结果**:
- ✅ 成功获取模型信息
- ✅ 包含 `contextLength` 字段

### 测试 2: 会话压缩阈值计算

**测试代码**:
```kotlin
class SessionCompressionConfig {
    val thresholdPercent: Double = 0.8
    val modelMaxTokens: Int = 32768
    
    fun calculateThreshold(): Int {
        return (modelMaxTokens * thresholdPercent).toInt()
    }
}
```

**预期结果**:
- ✅ 32768 * 0.8 = 26214
- ✅ 阈值计算正确

### 测试 3: 并发控制队列

**测试场景**:
- 配置最大并发数：3
- 同时提交 10 个请求
- 验证前 3 个立即执行，后 7 个排队等待

**测试代码**:
```kotlin
class ConcurrencyTest {
    private val queue = AIRequestQueue(maxConcurrent = 3)
    
    @Test
    fun testConcurrencyLimit() = runBlocking {
        val startTime = System.currentTimeMillis()
        
        // 同时提交 10 个请求
        val jobs = List(10) { i ->
            async {
                queue.submit(Request("Request $i"))
            }
        }
        
        jobs.awaitAll()
        
        val elapsed = System.currentTimeMillis() - startTime
        // 验证总耗时 > 3 个请求的执行时间（证明有排队）
        assert(elapsed > 3000)
    }
}
```

**预期结果**:
- ✅ 前 3 个请求立即执行
- ✅ 后 7 个请求排队等待
- ✅ 先到先得顺序

## 推荐配置方案

### 完整配置示例

```hocon
app {
    aiService {
        lmStudioUrl = "http://localhost:1234/v1"
        apiKey = "sk-test-key-123456"
        modelName = "qwen3.5-9b-uncensored-hauhaucs-aggressive"
        
        # 模型配置（可选：如果无法自动获取）
        modelMaxTokens = 32768
        
        # 会话压缩配置
        sessionCompression {
            enabled = true
            thresholdPercent = 0.8      # 80% 时触发压缩
            minThresholdTokens = 10000  # 最小阈值（可选）
            maxThresholdTokens = 30000  # 最大阈值（可选）
            
            # 压缩策略
            strategy = "summarize"      # summarize / truncate / smart
        }
        
        # 并发控制配置
        concurrency {
            maxConcurrentRequests = 3   # 最大并发请求数
            queueEnabled = true         # 启用队列
            queueTimeout = 300000       # 队列超时（5 分钟）
            queueMode = "fifo"          # fifo / priority
        }
    }
}
```

## 实现建议

### 1. 自动获取模型信息（推荐）
- 启动时查询 LM Studio API 获取模型最大 token 数
- 缓存结果，避免重复查询
- 支持手动配置覆盖

### 2. 会话压缩
- 监控会话 token 使用量
- 达到阈值时触发压缩
- 提供多种压缩策略（摘要、截断、智能）

### 3. 并发控制
- 使用信号量控制并发数
- 使用 Channel 实现队列
- 支持超时和取消
- 提供队列状态监控

## 风险评估

### 低风险项
1. ✅ HOCON 配置支持所有需要的字段类型
2. ✅ Kotlin 协程提供成熟的并发控制原语
3. ✅ LM Studio REST API 支持模型查询

### 中风险项
1. ⚠️ LM Studio 不同版本 API 可能有差异
   - 缓解：支持手动配置覆盖
2. ⚠️ 模型信息端点可能返回格式不一致
   - 缓解：添加容错和默认值

### 待验证项
1. ❓ LM Studio 是否所有模型都提供 `context_length` 字段
2. ❓ 队列超时后的错误处理策略

## 结论

**可行性**: ✅ **完全可行**

所有需要的功能都有成熟的技术方案：
1. ✅ LM Studio 模型信息查询 - REST API 支持
2. ✅ 会话压缩阈值配置 - HOCON 配置支持
3. ✅ 并发控制和队列管理 - Kotlin 协程支持

**推荐实施顺序**:
1. 实现基础配置结构
2. 添加会话压缩阈值配置
3. 实现并发控制队列
4. 添加 LM Studio 模型信息自动获取
5. 完善错误处理和监控
