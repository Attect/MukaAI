package test

import kotlinx.serialization.*
import kotlinx.serialization.json.*

/**
 * 工具调用协议 Unicode-Escaped 验证测试
 * 
 * 测试目的：
 * 1. 模拟 AI 模型输出包含 Unicode-escaped 路径的工具调用
 * 2. 验证调度器能正确解码并执行
 * 3. 验证路径解码后能正确处理
 */

// 模拟工具调用请求
@Serializable
data class ToolCallRequest(
    val callId: String,
    val name: String,
    val args: Map<String, String>
)

// 模拟工具调用响应
@Serializable
data class ToolCallResponse(
    val callId: String,
    val success: Boolean,
    val result: String? = null,
    val error: String? = null
)



/**
 * 模拟工具调度器
 */
class MockToolScheduler {
    
    /**
     * 处理工具调用：解码参数 -> 验证 -> 执行
     */
    fun handleToolCall(request: ToolCallRequest): ToolCallResponse {
        println("\n=== 接收到工具调用 ===")
        println("调用 ID: ${request.callId}")
        println("工具名：${request.name}")
        println("原始参数:")
        request.args.forEach { (key, value) ->
            println("  $key = $value")
        }
        
        return try {
            // 步骤 1: 解码参数中的 Unicode-escaped 字符串
            val decodedArgs = request.args.mapValues { (_, value) ->
                value.fromUnicodeEscaped()
            }
            
            println("解码后参数:")
            decodedArgs.forEach { (key, value) ->
                println("  $key = $value")
            }
            
            // 步骤 2: 验证参数（这里简化处理）
            validateToolCall(request.name, decodedArgs)
            
            // 步骤 3: 执行工具（模拟）
            val result = executeTool(request.name, decodedArgs)
            
            ToolCallResponse(
                callId = request.callId,
                success = true,
                result = result
            )
        } catch (e: Exception) {
            ToolCallResponse(
                callId = request.callId,
                success = false,
                error = e.message
            )
        }
    }
    
    private fun validateToolCall(name: String, args: Map<String, String>) {
        // 简化的验证逻辑
        if (name == "read_file" && !args.containsKey("path")) {
            throw Exception("缺少必需参数：path")
        }
        if (name == "run_shell_command" && !args.containsKey("command")) {
            throw Exception("缺少必需参数：command")
        }
    }
    
    private fun executeTool(name: String, args: Map<String, String>): String {
        return when (name) {
            "read_file" -> {
                val path = args["path"] ?: throw Exception("缺少 path 参数")
                "模拟读取文件：$path"
            }
            "run_shell_command" -> {
                val command = args["command"] ?: throw Exception("缺少 command 参数")
                "模拟执行命令：$command"
            }
            else -> throw Exception("未知工具：$name")
        }
    }
}

/**
 * 测试用例执行
 */
fun runTest(testName: String, testBlock: () -> Boolean) {
    print("测试：$testName ... ")
    try {
        val result = testBlock()
        if (result) {
            println("✅ 通过")
        } else {
            println("❌ 失败")
        }
    } catch (e: Exception) {
        println("❌ 异常：${e.message}")
        e.printStackTrace()
    }
}

fun main() {
    println("=== 工具调用协议 Unicode-Escaped 测试 ===\n")
    
    val scheduler = MockToolScheduler()
    val json = Json { ignoreUnknownKeys = true }
    
    // 测试 1: read_file 工具 - 包含中文的路径
    runTest("read_file 工具 - 中文路径") {
        // 模拟 AI 模型输出的工具调用（路径已转换为 Unicode-escaped）
        val aiOutputPath = "C:\\Users\\测试\\文档\\文件.md".toUnicodeEscaped()
        val requestJson = """
            {
                "callId": "call-001",
                "name": "read_file",
                "args": {
                    "path": "$aiOutputPath"
                }
            }
        """.trimIndent()
        
        println("\n--- AI 输出的 JSON ---")
        println(requestJson)
        
        val request = json.decodeFromString<ToolCallRequest>(requestJson)
        val response = scheduler.handleToolCall(request)
        
        println("\n--- 执行结果 ---")
        println("成功：${response.success}")
        println("结果：${response.result}")
        
        response.success && response.result?.contains("测试") == true
    }
    
    // 测试 2: run_shell_command 工具 - 包含中文和数字的命令
    runTest("run_shell_command 工具 - 中文和数字混合") {
        // 模拟 AI 模型输出的工具调用
        val command = "cat '/tmp/中文 123 文档.md'".toUnicodeEscaped()
        val requestJson = """
            {
                "callId": "call-002",
                "name": "run_shell_command",
                "args": {
                    "command": "$command",
                    "is_background": "false"
                }
            }
        """.trimIndent()
        
        println("\n--- AI 输出的 JSON ---")
        println(requestJson)
        
        val request = json.decodeFromString<ToolCallRequest>(requestJson)
        val response = scheduler.handleToolCall(request)
        
        println("\n--- 执行结果 ---")
        println("成功：${response.success}")
        println("结果：${response.result}")
        
        response.success && response.result?.contains("中文 123") == true
    }
    
    // 测试 3: 多个参数都包含中文
    runTest("多参数中文处理") {
        val path = "C:\\Users\\用户\\文档.txt".toUnicodeEscaped()
        val content = "这是内容".toUnicodeEscaped()
        val requestJson = """
            {
                "callId": "call-003",
                "name": "write_file",
                "args": {
                    "path": "$path",
                    "content": "$content"
                }
            }
        """.trimIndent()
        
        println("\n--- AI 输出的 JSON ---")
        println(requestJson)
        
        val request = json.decodeFromString<ToolCallRequest>(requestJson)
        val response = scheduler.handleToolCall(request)
        
        // write_file 不是已知工具，应该失败
        !response.success
    }
    
    // 测试 4: 纯 ASCII 路径（确保不影响正常情况）
    runTest("纯 ASCII 路径 - 无 Unicode 编码") {
        val requestJson = """
            {
                "callId": "call-004",
                "name": "read_file",
                "args": {
                    "path": "C:\\Users\\test\\document.md"
                }
            }
        """.trimIndent()
        
        println("\n--- AI 输出的 JSON ---")
        println(requestJson)
        
        val request = json.decodeFromString<ToolCallRequest>(requestJson)
        val response = scheduler.handleToolCall(request)
        
        println("\n--- 执行结果 ---")
        println("成功：${response.success}")
        println("结果：${response.result}")
        
        response.success
    }
    
    // 测试 5: 混合场景 - 部分参数有 Unicode，部分没有
    runTest("混合 Unicode 和 ASCII 参数") {
        val path = "/home/用户/config.json".toUnicodeEscaped()
        val requestJson = """
            {
                "callId": "call-005",
                "name": "read_file",
                "args": {
                    "path": "$path"
                }
            }
        """.trimIndent()
        
        println("\n--- AI 输出的 JSON ---")
        println(requestJson)
        
        val request = json.decodeFromString<ToolCallRequest>(requestJson)
        val response = scheduler.handleToolCall(request)
        
        println("\n--- 执行结果 ---")
        println("成功：${response.success}")
        println("结果：${response.result}")
        
        response.success && response.result?.contains("用户") == true
    }
    
    println("\n=== 测试完成 ===")
}
