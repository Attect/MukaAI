#!/usr/bin/env kotlin

/**
 * 会话压缩功能可行性测试脚本
 * 
 * 使用方法:
 * 1. 确保 LM Studio 服务已启动，地址 http://127.0.0.1:11451
 * 2. 确保模型 jaahas/qwen3.5-uncensored:9b 已加载
 * 3. 运行此脚本：kotlinc -script test.kts
 */

import java.net.http.*
import java.net.URI
import java.time.Instant

// 数据类
data class Message(
    val id: String,
    val senderId: String,
    val senderType: String,
    val content: String,
    val timestamp: Instant
)

data class CompressionResult(
    val summary: String,
    val shouldAutoContinue: Boolean,
    val triggerType: String
)

enum class CompressionTriggerType {
    AUTO,
    USER_INITIATED
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

fun getPromptConfig(): CompressionPromptConfig {
    return CompressionPromptConfig(
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

fun createMockMessages(): List<Message> {
    return listOf(
        Message("msg-001", "user", "USER", "你好，我想学习 Kotlin 编程。", Instant.now().minusSeconds(3600)),
        Message("msg-002", "ai-assistant", "AI_ROLE", "你好！很高兴你想学习 Kotlin。Kotlin 是一门现代、简洁且功能强大的编程语言。你有什么编程基础吗？", Instant.now().minusSeconds(3500)),
        Message("msg-003", "user", "USER", "我有一些 Java 基础，学过面向对象编程。", Instant.now().minusSeconds(3400)),
        Message("msg-004", "ai-assistant", "AI_ROLE", "太好了！有 Java 基础学习 Kotlin 会很容易。Kotlin 和 Java 有很多相似之处，但语法更简洁。我们从哪里开始呢？", Instant.now().minusSeconds(3300)),
        Message("msg-005", "user", "USER", "先介绍一下 Kotlin 的基础语法吧。", Instant.now().minusSeconds(3200)),
        Message("msg-006", "ai-assistant", "AI_ROLE", "好的！Kotlin 基础语法要点：1. 变量声明 2. 类型推断 3. 字符串模板 4. 空安全 5. 函数", Instant.now().minusSeconds(3100)),
        Message("msg-007", "user", "USER", "空安全听起来很有用，详细讲讲吧。", Instant.now().minusSeconds(3000)),
        Message("msg-008", "ai-assistant", "AI_ROLE", "Kotlin 的空安全系统设计非常优秀：1. 可空类型 2. 安全调用 3. Elvis 操作符 4. 安全转换", Instant.now().minusSeconds(2900))
    )
}

fun formatMessages(messages: List<Message>): String {
    return messages.joinToString("\n") { msg ->
        val sender = when (msg.senderType) {
            "USER" -> "用户"
            "AI_ROLE" -> "AI"
            else -> msg.senderId
        }
        "[$sender]: ${msg.content}"
    }
}

fun callAIModel(systemPrompt: String, userPrompt: String): String {
    val client = HttpClient.newHttpClient()
    
    val jsonBody = """
        {
            "model": "jaahas/qwen3.5-uncensored:9b",
            "messages": [
                {
                    "role": "system",
                    "content": """ + "\"\"\"" + systemPrompt + "\"\"\"" + """
                },
                {
                    "role": "user",
                    "content": """ + "\"\"\"" + userPrompt + "\"\"\"" + """
                }
            ],
            "temperature": 0.7,
            "stream": false
        }
    """.trimIndent()
    
    val request = HttpRequest.newBuilder()
        .uri(URI.create("http://127.0.0.1:11451/v1/chat/completions"))
        .header("Content-Type", "application/json")
        .POST(HttpRequest.BodyPublishers.ofString(jsonBody))
        .build()
    
    try {
        val response = client.send(request, HttpResponse.BodyHandlers.ofString())
        val responseBody = response.body()
        
        // 简单解析 JSON 提取内容
        val contentRegex = "\"content\"\\s*:\\s*\"([^\"]*)\"".toRegex()
        val matches = contentRegex.findAll(responseBody).toList()
        
        if (matches.isNotEmpty()) {
            // 返回最后一个 content 匹配（AI 的回复）
            return matches.last().groupValues[1]
                .replace("\\n", "\n")
                .replace("\\\"", "\"")
        }
        
        return "无响应"
    } catch (e: Exception) {
        println("❌ 调用失败：${e.message}")
        throw e
    }
}

fun testCompression(
    messages: List<Message>,
    triggerType: CompressionTriggerType,
    promptConfig: CompressionPromptConfig
): CompressionResult {
    println("\n${"=".repeat(60)}")
    println("测试：${if (triggerType == CompressionTriggerType.AUTO) "自动压缩" else "用户主动压缩"}")
    println("${"=".repeat(60)}")
    
    val messagesToCompress = messages.dropLast(promptConfig.retainMessageCount)
    val recentMessages = messages.takeLast(promptConfig.retainMessageCount)
    
    println("\n📝 压缩 ${messagesToCompress.size} 条早期消息")
    println("💾 保留最近 ${recentMessages.size} 条消息的细节")
    
    if (messagesToCompress.isEmpty()) {
        return CompressionResult("没有需要压缩的消息", false, triggerType.name)
    }
    
    val formattedMessages = formatMessages(messagesToCompress)
    
    val (systemPrompt, userPromptTemplate) = when (triggerType) {
        CompressionTriggerType.AUTO -> 
            promptConfig.autoSystemPrompt to promptConfig.autoUserPromptTemplate
        CompressionTriggerType.USER_INITIATED -> 
            promptConfig.userSystemPrompt to promptConfig.userUserPromptTemplate
    }
    
    val userPrompt = userPromptTemplate
        .replace("{messages}", formattedMessages)
        .replace("{maxLength}", promptConfig.maxSummaryLength.toString())
        .replace("{retainCount}", promptConfig.retainMessageCount.toString())
    
    println("\n🤖 正在调用 AI 模型生成摘要...")
    val summary = callAIModel(systemPrompt, userPrompt)
    
    println("\n📄 生成的摘要:")
    println("-".repeat(60))
    println(summary)
    println("-".repeat(60))
    
    val shouldAutoContinue = when (triggerType) {
        CompressionTriggerType.AUTO -> true
        CompressionTriggerType.USER_INITIATED -> false
    }
    
    return CompressionResult(summary, shouldAutoContinue, triggerType.name)
}

fun validateSummary(summary: String, triggerType: CompressionTriggerType): Boolean {
    println("\n🔍 验证摘要是否符合预期...")
    
    return when (triggerType) {
        CompressionTriggerType.AUTO -> {
            val hasFirstPerson = summary.contains("我") || summary.contains("我们")
            val hasContinuation = summary.contains("继续") || 
                                 summary.contains("接下来") || 
                                 summary.contains("将")
            
            println("  - 使用第一人称：${if (hasFirstPerson) "✅" else "❌"}")
            println("  - 体现连续性：${if (hasContinuation) "✅" else "❌"}")
            
            hasFirstPerson && hasContinuation
        }
        CompressionTriggerType.USER_INITIATED -> {
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

fun main() {
    println("\n${"=".repeat(60)}")
    println("会话压缩功能可行性测试")
    println("${"=".repeat(60)}")
    println("\n🚀 开始测试...")
    println("📍 AI 服务：http://127.0.0.1:11451")
    println("🤖 模型：jaahas/qwen3.5-uncensored:9b")
    
    val promptConfig = getPromptConfig()
    val mockMessages = createMockMessages()
    
    try {
        // 测试 1: 自动压缩
        println("\n\n")
        println("📋 测试用例 1: 自动压缩")
        val autoResult = testCompression(mockMessages, CompressionTriggerType.AUTO, promptConfig)
        val autoValid = validateSummary(autoResult.summary, CompressionTriggerType.AUTO)
        
        if (autoResult.shouldAutoContinue) {
            println("\n🔄 自动压缩触发，应该发送\"继续\"指令")
        }
        
        Thread.sleep(2000)
        
        // 测试 2: 用户主动压缩
        println("\n\n")
        println("📋 测试用例 2: 用户主动压缩")
        val userResult = testCompression(mockMessages, CompressionTriggerType.USER_INITIATED, promptConfig)
        val userValid = validateSummary(userResult.summary, CompressionTriggerType.USER_INITIATED)
        
        if (!userResult.shouldAutoContinue) {
            println("\n⏸️  用户主动压缩，等待用户发起下一条消息")
        }
        
        // 测试总结
        println("\n\n")
        println("=".repeat(60))
        println("测试总结")
        println("=".repeat(60))
        println("测试用例 1 (自动压缩): ${if (autoValid) "✅ 通过" else "❌ 失败"}")
        println("测试用例 2 (用户主动压缩): ${if (userValid) "✅ 通过" else "❌ 失败"}")
        
        if (autoValid && userValid) {
            println("\n🎉 所有测试通过！会话压缩功能可行。")
        } else {
            println("\n⚠️  部分测试未通过，需要调整提示词。")
        }
        
    } catch (e: Exception) {
        println("\n❌ 测试执行失败：${e.message}")
        println("请确保 AI 服务已启动并可访问。")
        e.printStackTrace()
    }
}

main()
