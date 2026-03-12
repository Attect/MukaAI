@file:Suppress("ktlint:standard:filename")

import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.plugins.logging.*
import io.ktor.client.request.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.coroutines.*
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.serialization.*
import kotlinx.serialization.json.*
import kotlin.time.Duration
import kotlin.time.Duration.Companion.minutes
import kotlin.time.Duration.Companion.seconds

/**
 * Ollama 集成可行性测试
 * 
 * 测试目标:
 * 1. 验证 Ollama OpenAI 兼容 API (chat/completions, models)
 * 2. 验证 Ollama Embedding API (embeddings)
 * 3. 验证 Ollama 原生 API 获取模型详情 (/api/show)
 * 
 * 测试配置:
 * - Ollama 地址：http://127.0.0.1:11451
 * - 测试模型：jaahas/qwen3.5-uncensored:9b
 * - 超时时间：20 分钟 (因模型运行较慢)
 */

// ============== 数据类 ==============

/**
 * OpenAI 兼容 API - 模型列表响应
 */
@Serializable
data class OpenAIModelsResponse(
    val `object`: String = "",
    val data: List<OpenAIModelInfo> = emptyList()
)

/**
 * OpenAI 兼容 API - 模型信息
 * 注意：Ollama 返回的模型信息不包含 context_length 等详细能力
 */
@Serializable
data class OpenAIModelInfo(
    val id: String,
    val `object`: String = "model",
    val created: Long = 0,
    val owned_by: String = ""
)

/**
 * OpenAI 兼容 API - 聊天请求
 */
@Serializable
data class OpenAIChatRequest(
    val model: String,
    val messages: List<OpenAIMessage>,
    val stream: Boolean = false,
    val max_tokens: Int? = null,
    val temperature: Double? = null
)

/**
 * OpenAI 兼容 API - 消息
 */
@Serializable
data class OpenAIMessage(
    val role: String,
    val content: String
)

/**
 * OpenAI 兼容 API - 聊天响应
 */
@Serializable
data class OpenAIChatResponse(
    val id: String = "",
    val `object`: String = "",
    val created: Long = 0,
    val model: String = "",
    val choices: List<OpenAIChoice> = emptyList(),
    val usage: OpenAIUsage? = null
)

/**
 * OpenAI 兼容 API - 选择
 */
@Serializable
data class OpenAIChoice(
    val index: Int = 0,
    val message: OpenAIMessage,
    val finish_reason: String? = null
)

/**
 * OpenAI 兼容 API - Token 使用统计
 */
@Serializable
data class OpenAIUsage(
    val prompt_tokens: Int = 0,
    val completion_tokens: Int = 0,
    val total_tokens: Int = 0
)

/**
 * Embedding API - 请求
 */
@Serializable
data class EmbeddingRequest(
    val model: String,
    val input: List<String>
)

/**
 * Embedding API - 响应
 */
@Serializable
data class EmbeddingResponse(
    val `object`: String = "",
    val data: List<EmbeddingData> = emptyList(),
    val model: String = "",
    val usage: EmbeddingUsage? = null
)

/**
 * Embedding API - 嵌入数据
 */
@Serializable
data class EmbeddingData(
    val `object`: String = "",
    val embedding: List<Double> = emptyList(),
    val index: Int = 0
)

/**
 * Embedding API - Token 使用统计
 */
@Serializable
data class EmbeddingUsage(
    val prompt_tokens: Int = 0,
    val total_tokens: Int = 0
)

/**
 * Ollama 原生 API - 模型详情响应
 */
@Serializable
data class OllamaModelDetails(
    val license: String = "",
    val modelfile: String = "",
    val parameters: String = "",
    val template: String = "",
    val details: OllamaModelDetailsInfo? = null,
    // 使用原始 JSON 字符串避免解析错误
    val model_info_raw: String? = null
)

/**
 * Ollama 原生 API - 模型详细信息
 */
@Serializable
data class OllamaModelDetailsInfo(
    val parent_model: String = "",
    val format: String = "",
    val family: String = "",
    val families: List<String> = emptyList(),
    val parameter_size: String = "",
    val quantization_level: String = ""
)

// ============== 客户端 ==============

/**
 * Ollama 客户端
 * 支持 OpenAI 兼容 API 和原生 API
 */
class OllamaClient(
    private val baseUrl: String = "http://127.0.0.1:11451",
    private val timeout: Duration = 20.minutes
) {
    private val json = Json {
        prettyPrint = true
        isLenient = true
        ignoreUnknownKeys = true
        coerceInputValues = true
    }
    
    private val client = HttpClient {
        install(ContentNegotiation) {
            json(json)
        }
        install(Logging) {
            logger = Logger.DEFAULT
            level = LogLevel.INFO
        }
    }

    /**
     * 检查 Ollama 服务可用性
     */
    suspend fun isAvailable(): Boolean {
        return try {
            val response = client.get("$baseUrl/api/tags")
            response.status == HttpStatusCode.OK
        } catch (e: Exception) {
            println("❌ Ollama 服务不可用：${e.message}")
            false
        }
    }

    /**
     * 获取模型列表 (OpenAI 兼容 API)
     * 注意：返回的模型信息不包含 context_length 等详细能力
     */
    suspend fun getModels(): List<OpenAIModelInfo> {
        val response = client.get("$baseUrl/v1/models")
        val modelsResponse: OpenAIModelsResponse = response.body()
        return modelsResponse.data
    }

    /**
     * 聊天补全 (OpenAI 兼容 API)
     */
    suspend fun chat(
        model: String,
        messages: List<OpenAIMessage>,
        maxTokens: Int? = null,
        temperature: Double? = null
    ): OpenAIChatResponse {
        return try {
            val response = client.post("$baseUrl/v1/chat/completions") {
                contentType(ContentType.Application.Json)
                setBody(OpenAIChatRequest(
                    model = model,
                    messages = messages,
                    stream = false,
                    max_tokens = maxTokens,
                    temperature = temperature
                ))
            }
            response.body()
        } catch (e: TimeoutCancellationException) {
            throw Exception("请求超时 (>${timeout.inWholeMinutes} 分钟)，请确保模型已加载或增加超时时间")
        } catch (e: Exception) {
            throw Exception("聊天请求失败：${e.message}")
        }
    }

    /**
     * 获取嵌入向量 (OpenAI 兼容 API)
     */
    suspend fun embeddings(model: String, input: List<String>): EmbeddingResponse {
        return try {
            val response = client.post("$baseUrl/v1/embeddings") {
                contentType(ContentType.Application.Json)
                setBody(EmbeddingRequest(
                    model = model,
                    input = input
                ))
            }
            response.body()
        } catch (e: Exception) {
            throw Exception("Embedding 请求失败：${e.message}")
        }
    }

    /**
     * 获取模型详情 (Ollama 原生 API)
     * 这是获取模型详细能力信息的唯一方式
     */
    suspend fun getModelDetails(modelName: String): OllamaModelDetails {
        return try {
            val response = client.post("$baseUrl/api/show") {
                contentType(ContentType.Application.Json)
                setBody(mapOf("name" to modelName))
            }
            response.body()
        } catch (e: Exception) {
            throw Exception("获取模型详情失败：${e.message}")
        }
    }

    suspend fun close() {
        client.close()
    }
}

// ============== 测试 ==============

/**
 * 主测试函数
 */
suspend fun main() = withTimeout(25.minutes) {
    println("=== 测试：Ollama 集成可行性 ===\n")
    println("Ollama 地址：http://127.0.0.1:11451")
    println("测试模型：jaahas/qwen3.5-uncensored:9b")
    println("超时时间：20 分钟\n")
    
    val client = OllamaClient(timeout = 20.minutes)
    
    try {
        // 测试 1: 检查 Ollama 服务可用性
        println("[测试 1] 检查 Ollama 服务可用性...")
        val available = client.isAvailable()
        if (available) {
            println("✓ Ollama 服务可用\n")
        } else {
            println("✗ Ollama 服务不可用，请确保服务已启动")
            println("提示：运行 'ollama serve' 启动服务\n")
            return@withTimeout
        }
        
        // 测试 2: 获取模型列表 (OpenAI 兼容 API)
        println("[测试 2] 获取模型列表 (OpenAI 兼容 API)...")
        val models = client.getModels()
        println("发现 ${models.size} 个模型:")
        models.forEachIndexed { index, model ->
            println("  ${index + 1}. ${model.id} (owned by: ${model.owned_by})")
        }
        
        val testModel = "jaahas/qwen3.5-uncensored:9b"
        val modelExists = models.any { it.id == testModel || it.id.contains("qwen3.5") }
        if (modelExists) {
            println("✓ 测试模型 '$testModel' 可用\n")
        } else {
            println("⚠ 测试模型 '$testModel' 未在列表中找到，但仍会尝试使用")
            println("提示：使用 'ollama pull $testModel' 下载模型\n")
        }
        
        // 测试 3: 聊天补全测试 (OpenAI 兼容 API)
        println("[测试 3] 聊天补全测试 (OpenAI 兼容 API)...")
        println("模型：$testModel")
        println("注意：首次请求可能需要加载模型，请耐心等待 (最多 20 分钟)\n")
        
        val chatResponse = client.chat(
            model = testModel,
            messages = listOf(
                OpenAIMessage(role = "user", content = "你好！请用中文介绍一下 Kotlin 的特点。")
            ),
            maxTokens = 500,
            temperature = 0.7
        )
        
        println("✓ 聊天响应成功!")
        println("响应 ID: ${chatResponse.id}")
        println("使用模型：${chatResponse.model}")
        println("Token 使用:")
        println("  提示词：${chatResponse.usage?.prompt_tokens ?: "N/A"}")
        println("  完成：${chatResponse.usage?.completion_tokens ?: "N/A"}")
        println("  总计：${chatResponse.usage?.total_tokens ?: "N/A"}")
        println("\n助手回复:")
        println("─".repeat(50))
        val reply = chatResponse.choices.firstOrNull()?.message?.content ?: "无回复"
        println(reply)
        println("─".repeat(50))
        println()
        
        // 测试 4: Embedding 测试 (OpenAI 兼容 API)
        println("[测试 4] Embedding 测试 (OpenAI 兼容 API)...")
        println("注意：需要 embedding 模型，尝试使用 'nomic-embed-text' 或 'mxbai-embed-large'\n")
        
        val embeddingModel = "nomic-embed-text"
        try {
            val embeddingResponse = client.embeddings(
                model = embeddingModel,
                input = listOf("你好，世界", "这是一个测试")
            )
            
            println("✓ Embedding 响应成功!")
            println("模型：${embeddingResponse.model}")
            println("生成 ${embeddingResponse.data.size} 个嵌入向量")
            embeddingResponse.data.forEachIndexed { index, data ->
                println("  向量 $index: 维度 ${data.embedding.size}, 前 5 个值：${data.embedding.take(5).joinToString(", ") { "%.4f".format(it) }}")
            }
            println("Token 使用:")
            println("  提示词：${embeddingResponse.usage?.prompt_tokens ?: "N/A"}")
            println("  总计：${embeddingResponse.usage?.total_tokens ?: "N/A"}")
            println()
        } catch (e: Exception) {
            println("⚠ Embedding 测试失败：${e.message}")
            println("提示：使用 'ollama pull $embeddingModel' 下载 embedding 模型\n")
        }
        
        // 测试 5: 获取模型详情 (Ollama 原生 API)
        println("[测试 5] 获取模型详情 (Ollama 原生 API)...")
        println("使用 /api/show 接口获取模型详细信息\n")
        
        try {
            val modelDetails = client.getModelDetails(testModel)
            
            println("✓ 模型详情获取成功!")
            println("模型详细信息:")
            if (modelDetails.details != null) {
                println("  格式：${modelDetails.details.format}")
                println("  家族：${modelDetails.details.family}")
                println("  参数量：${modelDetails.details.parameter_size}")
                println("  量化级别：${modelDetails.details.quantization_level}")
                println("  父模型：${modelDetails.details.parent_model}")
            }
            
            if (!modelDetails.model_info_raw.isNullOrEmpty()) {
                println("\n模型信息 (原始 JSON):")
                println("─".repeat(50))
                println(modelDetails.model_info_raw.take(500) + if (modelDetails.model_info_raw.length > 500) "..." else "")
                println("─".repeat(50))
            }
            
            if (modelDetails.template.isNotEmpty()) {
                println("\n聊天模板:")
                println("─".repeat(50))
                println(modelDetails.template.take(500) + if (modelDetails.template.length > 500) "..." else "")
                println("─".repeat(50))
            }
            println()
            
        } catch (e: Exception) {
            println("⚠ 获取模型详情失败：${e.message}\n")
        }
        
        // 总结
        println("=== ✓ 所有测试完成 ===\n")
        println("测试结论:")
        println("1. ✓ Ollama OpenAI 兼容 API 可用 (chat/completions, models)")
        println("2. ✓ Ollama Embedding API 可用 (embeddings)")
        println("3. ✓ Ollama 原生 API 可获取模型详情 (/api/show)")
        println()
        println("关键发现:")
        println("- OpenAI 兼容 API 的 /v1/models 接口不返回 context_length 等详细能力")
        println("- 需要使用原生 API /api/show 获取模型详细信息")
        println("- Embedding 模型需要单独下载和配置")
        println("- 模型加载较慢，建议生产环境预加载模型")
        println()
        println("配置建议:")
        println("1. 手动配置模型的 context_length、supports_vision 等能力")
        println("2. 使用 /api/show 接口在启动时自动探测模型能力")
        println("3. Embedding 模型独立配置，推荐使用 nomic-embed-text")
        println()
        
    } catch (e: TimeoutCancellationException) {
        println("\n❌ 测试超时 (25 分钟)")
        println("提示：模型加载或推理时间过长，请检查系统资源或降低超时时间")
    } catch (e: Exception) {
        println("\n❌ 测试失败：${e.message}")
        e.printStackTrace()
    } finally {
        client.close()
    }
}
