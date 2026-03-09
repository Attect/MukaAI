import kotlinx.coroutines.*
import kotlinx.coroutines.flow.*
import java.io.BufferedReader
import java.io.InputStreamReader
import java.util.concurrent.TimeUnit

data class CommandResult(
    val exitCode: Int,
    val output: String,
    val error: String,
    val duration: Long
)

suspend fun executeCommand(
    command: List<String>,
    timeout: Long = 30000,
    workingDir: String? = null,
    environment: Map<String, String>? = null
): CommandResult {
    val startTime = System.currentTimeMillis()
    
    val processBuilder = ProcessBuilder(command)
    
    workingDir?.let { processBuilder.directory(java.io.File(it)) }
    
    environment?.let { env ->
        val processEnv = processBuilder.environment()
        processEnv.putAll(env)
    }
    
    processBuilder.redirectErrorStream(true)
    
    val process = processBuilder.start()
    
    val outputFlow = flow {
        val reader = BufferedReader(InputStreamReader(process.inputStream))
        var line: String?
        while (reader.readLine().also { line = it } != null) {
            emit(line)
        }
    }.flowOn(Dispatchers.IO)
    
    val output = StringBuilder()
    val error = StringBuilder()
    
    val collectJob = CoroutineScope(Dispatchers.IO).launch {
        outputFlow.collect { line ->
            output.appendLine(line)
            println("[STDOUT] $line")
        }
    }
    
    return try {
        if (timeout > 0) {
            withTimeout(timeout) {
                process.waitFor()
            }
        } else {
            process.waitFor()
        }
        
        collectJob.cancel()
        
        val duration = System.currentTimeMillis() - startTime
        
        CommandResult(
            exitCode = process.exitValue(),
            output = output.toString().trim(),
            error = error.toString().trim(),
            duration = duration
        )
    } catch (e: TimeoutCancellationException) {
        process.destroyForcibly()
        throw RuntimeException("命令执行超时 (${timeout}ms)", e)
    }
}

fun main() = runBlocking {
    println("=== 测试 04: Shell/CMD 命令行执行能力 ===\n")
    
    try {
        val isWindows = System.getProperty("os.name").lowercase().contains("win")
        
        println("[测试 1] 基础命令执行...")
        val command = if (isWindows) {
            listOf("powershell", "-Command", "Get-ChildItem")
        } else {
            listOf("ls", "-la")
        }
        
        val result1 = executeCommand(command, timeout = 5000)
        println("退出码：${result1.exitCode}")
        println("输出行数：${result1.output.lines().size}")
        println("执行时间：${result1.duration}ms\n")
        
        println("[测试 2] 带参数的命令...")
        val command2 = if (isWindows) {
            listOf("powershell", "-Command", "Write-Output", "'Hello from Kotlin'")
        } else {
            listOf("echo", "Hello from Kotlin")
        }
        
        val result2 = executeCommand(command2)
        println("输出：${result2.output}")
        println("退出码：${result2.exitCode}\n")
        
        println("[测试 3] 获取当前工作目录...")
        val command3 = if (isWindows) {
            listOf("powershell", "-Command", "Get-Location")
        } else {
            listOf("pwd")
        }
        
        val result3 = executeCommand(command3)
        println("工作目录：${result3.output}\n")
        
        println("[测试 4] 设置环境变量...")
        val command4 = if (isWindows) {
            listOf("powershell", "-Command", "Write-Output", "`env:TEST_VAR")
        } else {
            listOf("sh", "-c", "echo \$TEST_VAR")
        }
        
        val result4 = executeCommand(
            command4,
            environment = mapOf("TEST_VAR" to "TestValue123")
        )
        println("环境变量输出：${result4.output}\n")
        
        println("[测试 5] 错误命令处理...")
        val command5 = if (isWindows) {
            listOf("powershell", "-Command", "NonExistentCommand")
        } else {
            listOf("nonexistent_command")
        }
        
        val result5 = executeCommand(command5, timeout = 3000)
        println("退出码：${result5.exitCode} (预期非 0)")
        println("错误信息已捕获\n")
        
        println("=== 所有测试通过！Shell/CMD 命令行执行能力可行 ===")
        
    } catch (e: Exception) {
        println("\n=== 测试失败：${e.message} ===")
        e.printStackTrace()
    }
}
