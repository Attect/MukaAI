package com.assistant.test.lmstudio

import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.engine.cio.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.plugins.HttpTimeout
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.longOrNull
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.booleanOrNull

/**
 * LM Studio 模型信息数据类 (根据实际响应调整)
 */
@Serializable
data class LMStudioModel(
    val id: String? = null,
    val `object`: String? = null,
    val created: Long? = null,
    val ownedBy: String? = null,
    val name: String? = null,
    // LM Studio 特定字段
    val type: String? = null,
    val publisher: String? = null,
    val key: String? = null,
    val displayName: String? = null,
    val architecture: String? = null,
    val quantization: QuantizationInfo? = null,
    val sizeBytes: Long? = null,
    val paramsString: String? = null,
    val maxContextLength: Int? = null,
    val format: String? = null,
    val capabilities: CapabilitiesInfo? = null
) {
    // 使用 key 或 displayName 作为模型标识
    val identifier: String get() = key ?: displayName ?: name ?: id ?: "unknown"
}

@Serializable
data class QuantizationInfo(
    val name: String? = null,
    val bitsPerWeight: Int? = null
)

@Serializable
data class CapabilitiesInfo(
    val vision: Boolean? = null,
    val trainedForToolUse: Boolean? = null
)

/**
 * LM Studio 模型列表响应数据类
 */
@Serializable
data class LMStudioModelsResponse(
    val models: List<LMStudioModel>
)

/**
 * 聊天消息数据类
 */
@Serializable
data class ChatMessage(
    val role: String,
    val content: String
)

/**
 * 聊天完成请求数据类
 */
@Serializable
data class ChatCompletionRequest(
    val model: String,
    val messages: List<ChatMessage>,
    val temperature: Double = 0.7,
    val maxTokens: Int = -1,
    val stream: Boolean = false
)

/**
 * 聊天完成响应数据类
 */
@Serializable
data class Choice(
    val message: ChatMessage? = null,
    val finishReason: String? = null
)

@Serializable
data class Usage(
    val promptTokens: Int = 0,
    val completionTokens: Int = 0,
    val totalTokens: Int = 0
)

@Serializable
data class ChatCompletionResponse(
    val id: String? = null,
    val `object`: String? = null,
    val created: Long? = null,
    val model: String? = null,
    val choices: List<Choice>? = null,
    val usage: Usage? = null
)

/**
 * LM Studio 客户端类
 * @param baseUrl API 基础地址，默认：http://127.0.0.1:11452/api/v1
 * @param apiKey API 认证密钥，默认：sk-lm-ocCyCxvA:LFkKFpG7abUUr3kwp1ek
 */
class LMStudioClient(
    private val baseUrl: String = "http://127.0.0.1:11452",
    private val apiKey: String = "sk-lm-ocCyCxvA:LFkKFpG7abUUr3kwp1ek"
) {
    // LM Studio 的实际 API 路径
    private val modelsEndpoint = "$baseUrl/api/v1/models"
    private val chatCompletionsEndpoint = "$baseUrl/v1/chat/completions"
    private val client = HttpClient(CIO) {
        install(ContentNegotiation) {
            json(Json {
                ignoreUnknownKeys = true
                isLenient = true
            })
        }
        install(HttpTimeout) {
            connectTimeoutMillis = 5000
            requestTimeoutMillis = 120000 // 增加到 2 分钟，支持长对话
            socketTimeoutMillis = 120000
        }
    }
    
    /**
     * 检查 LM Studio 服务是否可用
     * @return 如果服务可用返回 true，否则返回 false
     */
    suspend fun isAvailable(): Boolean {
        return try {
            val response = client.get(modelsEndpoint) {
                header("Authorization", "Bearer $apiKey")
            }
            response.status.isSuccess()
        } catch (e: Exception) {
            println("检查 LM Studio 可用性失败：${e.message}")
            false
        }
    }
    
    /**
     * 获取模型列表
     */
    suspend fun getModels(): List<LMStudioModel> {
        val httpResponse = client.get(modelsEndpoint) {
            header("Authorization", "Bearer $apiKey")
        }.bodyAsText()
        
        // 解析 JSON
        val json = Json { ignoreUnknownKeys = true; isLenient = true }
        val jsonObject = json.parseToJsonElement(httpResponse).jsonObject
        val modelsArray = jsonObject["models"]?.jsonArray
        
        if (modelsArray != null) {
            return modelsArray.map { element ->
                val modelObj = element.jsonObject
                val quantizationObj = modelObj["quantization"]?.jsonObject
                val capabilitiesObj = modelObj["capabilities"]?.jsonObject
                LMStudioModel(
                    id = modelObj["id"]?.jsonPrimitive?.contentOrNull,
                    name = modelObj["name"]?.jsonPrimitive?.contentOrNull,
                    `object` = modelObj["object"]?.jsonPrimitive?.contentOrNull,
                    created = modelObj["created"]?.jsonPrimitive?.longOrNull,
                    ownedBy = modelObj["owned_by"]?.jsonPrimitive?.contentOrNull,
                    type = modelObj["type"]?.jsonPrimitive?.contentOrNull,
                    publisher = modelObj["publisher"]?.jsonPrimitive?.contentOrNull,
                    key = modelObj["key"]?.jsonPrimitive?.contentOrNull,
                    displayName = modelObj["display_name"]?.jsonPrimitive?.contentOrNull,
                    architecture = modelObj["architecture"]?.jsonPrimitive?.contentOrNull,
                    quantization = quantizationObj?.let { q ->
                        QuantizationInfo(
                            name = q["name"]?.jsonPrimitive?.contentOrNull,
                            bitsPerWeight = q["bits_per_weight"]?.jsonPrimitive?.intOrNull
                        )
                    },
                    sizeBytes = modelObj["size_bytes"]?.jsonPrimitive?.longOrNull,
                    paramsString = modelObj["params_string"]?.jsonPrimitive?.contentOrNull,
                    maxContextLength = modelObj["max_context_length"]?.jsonPrimitive?.intOrNull,
                    format = modelObj["format"]?.jsonPrimitive?.contentOrNull,
                    capabilities = capabilitiesObj?.let { c ->
                        CapabilitiesInfo(
                            vision = c["vision"]?.jsonPrimitive?.booleanOrNull,
                            trainedForToolUse = c["trained_for_tool_use"]?.jsonPrimitive?.booleanOrNull
                        )
                    }
                )
            }
        }
        return emptyList()
    }
    
    /**
     * 发送聊天请求
     * @param model 使用的模型 ID
     * @param messages 消息列表
     * @param temperature 温度参数，默认 0.7
     * @return 聊天响应
     */
    suspend fun chat(
        model: String,
        messages: List<ChatMessage>,
        temperature: Double = 0.7
    ): ChatCompletionResponse {
        val request = ChatCompletionRequest(
            model = model,
            messages = messages,
            temperature = temperature
        )
        
        // 发送请求
        val httpResponse = client.post(chatCompletionsEndpoint) {
            contentType(ContentType.Application.Json)
            header("Authorization", "Bearer $apiKey")
            setBody(request)
        }.bodyAsText()
        
        // 解析 JSON
        val json = Json { ignoreUnknownKeys = true; isLenient = true }
        val jsonObject = json.parseToJsonElement(httpResponse).jsonObject
        
        return ChatCompletionResponse(
            id = jsonObject["id"]?.jsonPrimitive?.contentOrNull,
            `object` = jsonObject["object"]?.jsonPrimitive?.contentOrNull,
            created = jsonObject["created"]?.jsonPrimitive?.longOrNull,
            model = jsonObject["model"]?.jsonPrimitive?.contentOrNull,
            choices = jsonObject["choices"]?.jsonArray?.map { choiceObj ->
                val choiceJson = choiceObj.jsonObject
                val messageObj = choiceJson["message"]?.jsonObject
                Choice(
                    message = messageObj?.let { msg ->
                        ChatMessage(
                            role = msg["role"]?.jsonPrimitive?.content ?: "assistant",
                            content = msg["content"]?.jsonPrimitive?.content ?: ""
                        )
                    },
                    finishReason = choiceJson["finish_reason"]?.jsonPrimitive?.contentOrNull
                )
            },
            usage = jsonObject["usage"]?.jsonObject?.let { usageObj ->
                Usage(
                    promptTokens = usageObj["prompt_tokens"]?.jsonPrimitive?.intOrNull ?: 0,
                    completionTokens = usageObj["completion_tokens"]?.jsonPrimitive?.intOrNull ?: 0,
                    totalTokens = usageObj["total_tokens"]?.jsonPrimitive?.intOrNull ?: 0
                )
            }
        )
    }
    
    /**
     * 关闭客户端
     */
    fun close() {
        client.close()
    }
}

/**
 * 主函数 - LM Studio 模型发现测试
 */
fun main() = runBlocking {
    println("=== 测试 06: LM Studio 模型自动发现 ===\n")
    println("API 地址：http://127.0.0.1:11452/api/v1")
    println("认证方式：Bearer Token\n")
    
    val client = LMStudioClient()
    
    try {
        // 测试 1: 检查 LM Studio 可用性
        println("[测试 1] 检查 LM Studio 服务可用性...")
        val available = client.isAvailable()
        if (available) {
            println("✓ LM Studio 服务可用\n")
        } else {
            println("✗ LM Studio 服务不可用，请确保服务已启动")
            println("提示：检查 LM Studio 是否运行在 http://127.0.0.1:11452\n")
            return@runBlocking
        }
        
        // 测试 2: 获取模型列表
        println("[测试 2] 获取可用模型列表...")
        val models = client.getModels()
        println("发现 ${models.size} 个模型:")
        models.forEachIndexed { index, model ->
            val displayName = model.displayName ?: model.name ?: model.key ?: model.id ?: "Unknown"
            val publisher = model.publisher ?: ""
            val architecture = model.architecture ?: ""
            println("  ${index + 1}. $displayName (by $publisher)")
            if (architecture.isNotEmpty()) {
                println("      架构：$architecture, 最大上下文：${model.maxContextLength ?: "N/A"}")
            }
        }
        println()
        
        if (models.isEmpty()) {
            println("警告：未发现任何模型，请在 LM Studio 中加载模型")
            return@runBlocking
        }
        
        // 测试 3: 使用第一个模型进行聊天测试
        val firstModel = models[0]
        println("[测试 3] 使用模型 '${firstModel.identifier}' 进行聊天测试...")
        val messages = listOf(
            ChatMessage(role = "system", content = "你是一个有帮助的助手，使用中文回答。"),
            ChatMessage(role = "user", content = "你好！请简单介绍一下你自己。")
        )
        
        val response = client.chat(
            model = firstModel.identifier,
            messages = messages
        )
        
        println("\n响应详情:")
        println("  响应 ID: ${response.id}")
        println("  使用模型：${response.model}")
        println("  助手回复：${response.choices?.getOrNull(0)?.message?.content ?: "无回复内容"}")
        response.usage?.let { usage ->
            println("  Token 使用:")
            println("    提示词：${usage.promptTokens}")
            println("    完成：${usage.completionTokens}")
            println("    总计：${usage.totalTokens}")
        }
        println()
        
        // 测试 4: 简单对话测试 (简化版，避免超时)
        println("[测试 4] 简单对话测试...")
        val simpleMessages = listOf(
            ChatMessage(role = "system", content = "你是一个有帮助的助手。"),
            ChatMessage(role = "user", content = "谢谢你的帮助！")
        )
        
        val simpleResponse = client.chat(
            model = firstModel.identifier,
            messages = simpleMessages
        )
        
        println("对话回复：${simpleResponse.choices?.getOrNull(0)?.message?.content ?: "无回复内容"}\n")
        
        println("=== ✓ 所有测试通过！LM Studio 模型自动发现功能验证成功 ===")
        
    } catch (e: Exception) {
        println("\n=== ✗ 测试失败 ===")
        println("错误信息：${e.message}")
        println("\n可能的原因:")
        println("1. LM Studio 服务未启动")
        println("2. API 地址或端口不正确")
        println("3. API Key 无效")
        println("4. 网络连接问题")
        println("\n请检查以上配置后重试。")
        e.printStackTrace()
    } finally {
        client.close()
    }
}
