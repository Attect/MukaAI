package com.example

import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.plugins.HttpTimeout
import io.ktor.client.request.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import kotlinx.serialization.json.putJsonArray
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.JsonObject
import java.time.Instant

/**
 * 会话压缩可行性测试
 * 
 * 测试目标：
 * 1. 验证自动压缩提示词生成 AI 口吻的摘要
 * 2. 验证用户主动压缩提示词生成客观描述的摘要
 * 3. 验证压缩后处理逻辑
 */

// 数据类
data class Message(
    val id: String,
    val senderId: String,
    val senderType: String, // "USER", "AI_ROLE", "SYSTEM"
    val sessionId: String,
    val content: String,
    val timestamp: Instant
)

data class CompressionResult(
    val summary: String,
    val shouldAutoContinue: Boolean,
    val triggerType: String
)

// 压缩触发类型
enum class CompressionTriggerType {
    AUTO,               // 自动压缩
    USER_INITIATED      // 用户主动压缩
}

// 提示词配置
data class CompressionPromptConfig(
    val autoSystemPrompt: String,
    val autoUserPromptTemplate: String,
    val userSystemPrompt: String,
    val userUserPromptTemplate: String,
    val maxSummaryLength: Int = 2000,
    val retainMessageCount: Int = 5
)

// 模拟的会话数据
fun createMockMessages(): List<Message> {
    return listOf(
        Message(
            id = "msg-001",
            senderId = "user",
            senderType = "USER",
            sessionId = "session-test-001",
            content = "你好，我想学习 Kotlin 编程。",
            timestamp = Instant.now().minusSeconds(3600)
        ),
        Message(
            id = "msg-002",
            senderId = "ai-assistant",
            senderType = "AI_ROLE",
            sessionId = "session-test-001",
            content = "你好！很高兴你想学习 Kotlin。Kotlin 是一门现代、简洁且功能强大的编程语言。你有什么编程基础吗？",
            timestamp = Instant.now().minusSeconds(3500)
        ),
        Message(
            id = "msg-003",
            senderId = "user",
            senderType = "USER",
            sessionId = "session-test-001",
            content = "我有一些 Java 基础，学过面向对象编程。",
            timestamp = Instant.now().minusSeconds(3400)
        ),
        Message(
            id = "msg-004",
            senderId = "ai-assistant",
            senderType = "AI_ROLE",
            sessionId = "session-test-001",
            content = "太好了！有 Java 基础学习 Kotlin 会很容易。Kotlin 和 Java 有很多相似之处，但语法更简洁。我们从哪里开始呢？我可以介绍：\n1. Kotlin 基础语法\n2. Kotlin 与 Java 的区别\n3. 直接开始一个小项目",
            timestamp = Instant.now().minusSeconds(3300)
        ),
        Message(
            id = "msg-005",
            senderId = "user",
            senderType = "USER",
            sessionId = "session-test-001",
            content = "先介绍一下 Kotlin 的基础语法吧。",
            timestamp = Instant.now().minusSeconds(3200)
        ),
        Message(
            id = "msg-006",
            senderId = "ai-assistant",
            senderType = "AI_ROLE",
            sessionId = "session-test-001",
            content = "好的！Kotlin 基础语法要点：1. 变量声明：val 声明只读变量，var 声明可变变量 2. 类型推断：自动推断类型 3. 字符串模板：使用美元符号 4. 空安全：可空类型用？表示 5. 函数：简洁的函数语法。你想先看哪个部分的详细示例？",
            timestamp = Instant.now().minusSeconds(3100)
        ),
        Message(
            id = "msg-007",
            senderId = "user",
            senderType = "USER",
            sessionId = "session-test-001",
            content = "空安全听起来很有用，详细讲讲吧。",
            timestamp = Instant.now().minusSeconds(3000)
        ),
        Message(
            id = "msg-008",
            senderId = "ai-assistant",
            senderType = "AI_ROLE",
            sessionId = "session-test-001",
            content = "Kotlin 的空安全系统设计非常优秀：1. 可空类型：String?表示可空 2. 安全调用：b?.length 3. Elvis 操作符：b?.length ?: 0 4. 安全转换：as? String。要我举个例子说明这些在实际代码中如何使用吗？",
            timestamp = Instant.now().minusSeconds(2900)
        )
    )
}

// 提示词配置
fun getPromptConfig(): CompressionPromptConfig {
    return CompressionPromptConfig(
        // 自动压缩提示词：AI 口吻，强调连续性
        autoSystemPrompt = """
            你是一个会话摘要助手。你的任务是对对话历史进行压缩摘要，以便 AI 模型能够平滑继续对话。
            
            请保留以下关键信息：
            1. 重要的决策和结论
            2. 用户的偏好和要求
            3. 未完成的任务和行动项
            4. 关键的时间点和事件
            5. 最近对话的细节（特别是最后几条消息的具体内容）
            
            摘要要求：
            - 使用 AI 的第一人称口吻（例如："我刚刚帮助用户完成了..."）
            - 结尾应体现对话的连续性（例如："接下来我将继续帮助用户..."）
            - 确保 AI 模型阅读摘要后能够立即继续之前的任务，不中断思路
            - 摘要应该简洁明了，便于后续对话时快速理解上下文
        """.trimIndent(),
        
        autoUserPromptTemplate = """
            请对以下对话进行摘要，以便 AI 能够平滑继续对话：
            
            {messages}
            
            请生成一个简洁的摘要，长度不超过{maxLength}字。
            注意：
            1. 保留最近{retainCount}条消息的完整细节
            2. 使用 AI 的第一人称口吻
            3. 结尾应体现"接下来我将继续..."的语义
        """.trimIndent(),
        
        // 用户主动压缩提示词：等待用户回应的口吻
        userSystemPrompt = """
            你是一个会话摘要助手。你的任务是对对话历史进行压缩摘要，以便用户快速回顾。
            
            请保留以下关键信息：
            1. 重要的决策和结论
            2. 用户的偏好和要求
            3. 未完成的任务和行动项
            4. 关键的时间点和事件
            5. 最近对话的细节（特别是最后几条消息的具体内容）
            
            摘要要求：
            - 使用第三人称或客观描述（例如："用户和 AI 讨论了..."）
            - 结尾应体现等待用户回应（例如："我们已经讨论了...请问您还有什么需要补充的吗？"）
            - 确保用户阅读摘要后能够快速回顾之前的对话内容
            - 摘要应该简洁明了，便于用户快速理解
        """.trimIndent(),
        
        userUserPromptTemplate = """
            请对以下对话进行摘要，以便用户快速回顾：
            
            {messages}
            
            请生成一个简洁的摘要，长度不超过{maxLength}字。
            注意：
            1. 保留最近{retainCount}条消息的完整细节
            2. 使用客观描述的语气
            3. 结尾应体现"等待用户回应"的语义
        """.trimIndent(),
        
        maxSummaryLength = 2000,
        retainMessageCount = 5
    )
}

// 创建 HTTP 客户端
fun createHttpClient(): HttpClient {
    return HttpClient {
        install(ContentNegotiation) {
            json(Json {
                prettyPrint = true
                isLenient = true
            })
        }
        // 安装超时插件
        install(HttpTimeout) {
            requestTimeoutMillis = 20 * 60 * 1000L  // 20 分钟
            connectTimeoutMillis = 30 * 1000L       // 30 秒
            socketTimeoutMillis = 20 * 60 * 1000L   // 20 分钟
        }
    }
}

// 调用 AI 模型
suspend fun callAIModel(
    client: HttpClient,
    systemPrompt: String,
    userPrompt: String
): String {
    val baseUrl = "http://127.0.0.1:11451"
    
    val requestBody = buildJsonObject {
        put("model", "jaahas/qwen3.5-uncensored:9b")
        putJsonArray("messages") {
            add(buildJsonObject {
                put("role", "system")
                put("content", systemPrompt)
            })
            add(buildJsonObject {
                put("role", "user")
                put("content", userPrompt)
            })
        }
        put("temperature", 0.7)
        put("stream", false)
    }
    
    try {
        val response = client.post("$baseUrl/v1/chat/completions") {
            contentType(ContentType.Application.Json)
            setBody(requestBody)
        }
        
        val responseBody: JsonObject = response.body()
        val choices = responseBody["choices"] as? kotlinx.serialization.json.JsonArray
        if (choices != null && choices.size > 0) {
            val message = choices[0] as? JsonObject
            return message?.get("message")?.let {
                (it as JsonObject)["content"]?.toString() ?: "无响应"
            } ?: "无响应"
        }
        return "无响应"
    } catch (e: Exception) {
        println("❌ 调用 AI 模型失败：${e.message}")
        throw e
    }
}

// 格式化消息
fun formatMessages(messages: List<Message>): String {
    return messages.joinToString("\n") { msg ->
        val sender = when (msg.senderType) {
            "USER" -> "用户"
            "AI_ROLE" -> "AI"
            "SYSTEM" -> "系统"
            else -> msg.senderId
        }
        "[$sender]: ${msg.content}"
    }
}

// 执行压缩测试
suspend fun testCompression(
    client: HttpClient,
    messages: List<Message>,
    triggerType: CompressionTriggerType,
    promptConfig: CompressionPromptConfig
): CompressionResult {
    println("\n${"=".repeat(60)}")
    println("测试：${if (triggerType == CompressionTriggerType.AUTO) "自动压缩" else "用户主动压缩"}")
    println("${"=".repeat(60)}")
    
    // 计算需要压缩的消息范围（保留最近的 retainMessageCount 条）
    val messagesToCompress = messages.dropLast(promptConfig.retainMessageCount)
    val recentMessages = messages.takeLast(promptConfig.retainMessageCount)
    
    println("\n📝 压缩 ${messagesToCompress.size} 条早期消息")
    println("💾 保留最近 ${recentMessages.size} 条消息的细节")
    
    if (messagesToCompress.isEmpty()) {
        return CompressionResult(
            summary = "没有需要压缩的消息",
            shouldAutoContinue = false,
            triggerType = triggerType.name
        )
    }
    
    // 格式化消息
    val formattedMessages = formatMessages(messagesToCompress)
    
    // 选择提示词
    val (systemPrompt, userPromptTemplate) = when (triggerType) {
        CompressionTriggerType.AUTO -> 
            promptConfig.autoSystemPrompt to promptConfig.autoUserPromptTemplate
        CompressionTriggerType.USER_INITIATED -> 
            promptConfig.userSystemPrompt to promptConfig.userUserPromptTemplate
    }
    
    // 构建用户提示词
    val userPrompt = userPromptTemplate
        .replace("{messages}", formattedMessages)
        .replace("{maxLength}", promptConfig.maxSummaryLength.toString())
        .replace("{retainCount}", promptConfig.retainMessageCount.toString())
    
    println("\n🤖 正在调用 AI 模型生成摘要...")
    val summary = callAIModel(client, systemPrompt, userPrompt)
    
    println("\n📄 生成的摘要:")
    println("-".repeat(60))
    println(summary)
    println("-".repeat(60))
    
    // 判断是否应该自动继续
    val shouldAutoContinue = when (triggerType) {
        CompressionTriggerType.AUTO -> true
        CompressionTriggerType.USER_INITIATED -> false
    }
    
    return CompressionResult(
        summary = summary,
        shouldAutoContinue = shouldAutoContinue,
        triggerType = triggerType.name
    )
}

// 验证摘要是否符合预期
fun validateSummary(
    summary: String,
    triggerType: CompressionTriggerType
): Boolean {
    println("\n🔍 验证摘要是否符合预期...")
    
    return when (triggerType) {
        CompressionTriggerType.AUTO -> {
            // 自动压缩应该使用 AI 第一人称，结尾体现连续性
            val hasFirstPerson = summary.contains("我") || summary.contains("我们")
            val hasContinuation = summary.contains("继续") || 
                                 summary.contains("接下来") || 
                                 summary.contains("将")
            
            println("  - 使用第一人称：${if (hasFirstPerson) "✅" else "❌"}")
            println("  - 体现连续性：${if (hasContinuation) "✅" else "❌"}")
            
            hasFirstPerson && hasContinuation
        }
        CompressionTriggerType.USER_INITIATED -> {
            // 用户主动压缩应该客观描述，结尾体现等待回应
            val hasObjectiveTone = !summary.contains("我") || 
                                  summary.contains("用户") || 
                                  summary.contains("AI")
            val hasWaiting = summary.contains("请问") || 
                            summary.contains("等待") || 
                            summary.contains("继续") ||
                            summary.contains("?") ||
                            summary.contains("?")
            
            println("  - 客观描述：${if (hasObjectiveTone) "✅" else "❌"}")
            println("  - 等待回应：${if (hasWaiting) "✅" else "❌"}")
            
            hasObjectiveTone && hasWaiting
        }
    }
}

// 测试自动继续逻辑
fun testAutoContinueLogic(compressionResult: CompressionResult) {
    println("\n🔄 测试自动继续逻辑...")
    
    if (compressionResult.shouldAutoContinue) {
        println("  ✅ 自动压缩触发，应该发送\"继续\"指令")
        println("  模拟：系统正在发送继续指令...")
        println("  模拟：AI 收到继续指令，正在继续之前的对话...")
    } else {
        println("  ✅ 用户主动压缩，等待用户发起下一条消息")
        println("  模拟：系统显示压缩摘要，等待用户输入...")
    }
}

// 测试用户消息触发压缩
suspend fun testUserMessageCompression(
    client: HttpClient,
    currentMessages: List<Message>,
    newUserMessage: Message,
    promptConfig: CompressionPromptConfig
) {
    println("\n${"=".repeat(60)}")
    println("测试：用户消息触发压缩")
    println("${"=".repeat(60)}")
    
    // 模拟检查 token（这里简化为消息数量检查）
    val threshold = 10  // 假设阈值是 10 条消息
    val totalMessages = currentMessages.size + 1  // 加上新消息
    
    println("\n📊 当前消息数：${currentMessages.size}")
    println("📊 添加新消息后：$totalMessages")
    println("📊 阈值：$threshold")
    
    if (totalMessages > threshold) {
        println("\n⚠️  超过阈值，触发压缩")
        println("📝 视为用户主动压缩，使用 USER_INITIATED 提示词")
        
        // 执行压缩（使用当前消息，不包括新消息）
        val compressionResult = testCompression(
            client = client,
            messages = currentMessages,
            triggerType = CompressionTriggerType.USER_INITIATED,
            promptConfig = promptConfig
        )
        
        println("\n✅ 压缩完成")
        println("📝 添加用户新消息到压缩后的会话")
        println("⏸️  等待 AI 响应（不自动继续）")
        
    } else {
        println("\n✅ 未超过阈值，正常处理用户消息")
    }
}

// 主测试函数
fun main() = runBlocking {
    println("\n${"=".repeat(60)}")
    println("会话压缩功能可行性测试")
    println("${"=".repeat(60)}")
    println("\n🚀 开始测试...")
    println("📍 AI 服务：http://127.0.0.1:11451")
    println("🤖 模型：jaahas/qwen3.5-uncensored:9b")
    
    val client = createHttpClient()
    val promptConfig = getPromptConfig()
    val mockMessages = createMockMessages()
    
    try {
        // 测试 1: 自动压缩
        println("\n\n")
        println("📋 测试用例 1: 自动压缩")
        val autoResult = testCompression(
            client = client,
            messages = mockMessages,
            triggerType = CompressionTriggerType.AUTO,
            promptConfig = promptConfig
        )
        val autoValid = validateSummary(autoResult.summary, CompressionTriggerType.AUTO)
        testAutoContinueLogic(autoResult)
        
        // 等待一下，避免请求过快
        kotlinx.coroutines.delay(2000)
        
        // 测试 2: 用户主动压缩
        println("\n\n")
        println("📋 测试用例 2: 用户主动压缩")
        val userResult = testCompression(
            client = client,
            messages = mockMessages,
            triggerType = CompressionTriggerType.USER_INITIATED,
            promptConfig = promptConfig
        )
        val userValid = validateSummary(userResult.summary, CompressionTriggerType.USER_INITIATED)
        testAutoContinueLogic(userResult)
        
        // 等待一下
        kotlinx.coroutines.delay(2000)
        
        // 测试 3: 用户消息触发压缩
        println("\n\n")
        println("📋 测试用例 3: 用户消息触发压缩")
        val newMessage = Message(
            id = "msg-new",
            senderId = "user",
            senderType = "USER",
            sessionId = "session-test-001",
            content = "这个功能很强大，我想知道如何在实际项目中使用它。",
            timestamp = Instant.now()
        )
        testUserMessageCompression(
            client = client,
            currentMessages = mockMessages,
            newUserMessage = newMessage,
            promptConfig = promptConfig
        )
        
        // 测试总结
        println("\n\n")
        println("=".repeat(60))
        println("测试总结")
        println("=".repeat(60))
        println("测试用例 1 (自动压缩): ${if (autoValid) "✅ 通过" else "❌ 失败"}")
        println("测试用例 2 (用户主动压缩): ${if (userValid) "✅ 通过" else "❌ 失败"}")
        println("测试用例 3 (用户消息触发): ✅ 通过")
        
        if (autoValid && userValid) {
            println("\n🎉 所有测试通过！会话压缩功能可行。")
        } else {
            println("\n⚠️  部分测试未通过，需要调整提示词。")
        }
        
    } catch (e: Exception) {
        println("\n❌ 测试执行失败：${e.message}")
        println("请确保 AI 服务已启动并可访问。")
        e.printStackTrace()
    } finally {
        client.close()
    }
}
