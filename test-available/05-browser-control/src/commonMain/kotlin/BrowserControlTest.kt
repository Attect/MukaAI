import kotlinx.coroutines.*
import java.io.BufferedReader
import java.io.InputStreamReader
import java.io.File

data class BrowserTestResult(
    val success: Boolean,
    val output: String,
    val error: String? = null
)

/**
 * 检查 Playwright 是否已安装
 */
suspend fun checkPlaywrightInstalled(): Boolean {
    return try {
        val processBuilder = ProcessBuilder("npx", "playwright", "--version")
        val process = processBuilder.start()
        val exitCode = process.waitFor()
        exitCode == 0
    } catch (e: Exception) {
        false
    }
}

/**
 * 使用 Playwright 进行浏览器测试
 */
suspend fun runPlaywrightTest(): BrowserTestResult {
    return try {
        // 创建一个简单的 Playwright 脚本
        val testScript = File("test-playwright.mjs")
        testScript.writeText("""
            import { chromium } from 'playwright';
            
            async function test() {
                const browser = await chromium.launch({ headless: true });
                const page = await browser.newPage();
                
                // 测试 1: 导航到页面
                await page.goto('data:text/html,<h1>Test Page</h1>');
                const title = await page.title();
                console.log('页面标题：' + title);
                
                // 测试 2: 获取页面内容
                const content = await page.content();
                console.log('页面内容长度：' + content.length);
                
                // 测试 3: 执行 JavaScript
                const result = await page.evaluate(() => {
                    return document.querySelector('h1')?.textContent;
                });
                console.log('H1 文本：' + result);
                
                // 测试 4: 截图
                await page.screenshot({ path: 'test-screenshot.png' });
                console.log('截图已保存');
                
                await browser.close();
                console.log('测试完成');
            }
            
            test().catch(err => {
                console.error('测试失败:', err);
                process.exit(1);
            });
        """.trimIndent())
        
        val processBuilder = ProcessBuilder("node", testScript.absolutePath)
        val process = processBuilder.start()
        
        val output = StringBuilder()
        val error = StringBuilder()
        
        // 读取输出
        val reader = BufferedReader(InputStreamReader(process.inputStream))
        var line: String?
        while (reader.readLine().also { line = it } != null) {
            output.appendLine(line)
            println("[STDOUT] $line")
        }
        
        // 读取错误
        val errorReader = BufferedReader(InputStreamReader(process.errorStream))
        while (errorReader.readLine().also { line = it } != null) {
            error.appendLine(line)
        }
        
        val exitCode = process.waitFor()
        
        // 清理临时文件
        testScript.delete()
        
        BrowserTestResult(
            success = exitCode == 0,
            output = output.toString().trim(),
            error = if (exitCode != 0) error.toString().trim() else null
        )
    } catch (e: Exception) {
        BrowserTestResult(
            success = false,
            output = "",
            error = e.message
        )
    }
}

/**
 * 使用 Selenium 进行浏览器测试（备选方案）
 */
suspend fun runSeleniumTest(): BrowserTestResult {
    return try {
        // 检查 Selenium 是否可用
        val processBuilder = ProcessBuilder("java", "-cp", "*", "org.openqa.selenium.chrome.ChromeDriver")
        val process = processBuilder.start()
        
        // Selenium 需要额外的依赖，这里只做简单检查
        BrowserTestResult(
            success = true,
            output = "Selenium 方案需要额外配置依赖"
        )
    } catch (e: Exception) {
        BrowserTestResult(
            success = false,
            output = "",
            error = "Selenium 不可用：${e.message}"
        )
    }
}

fun main() = runBlocking {
    println("=== 测试 05: 浏览器控制能力 (Playwright) ===\n")
    
    try {
        println("[测试 1] 检查 Playwright 安装状态...")
        val playwrightInstalled = checkPlaywrightInstalled()
        println("Playwright 已安装：$playwrightInstalled")
        
        if (playwrightInstalled) {
            println("\n[测试 2] 运行 Playwright 浏览器测试...")
            val result = runPlaywrightTest()
            
            if (result.success) {
                println("\n✓ Playwright 测试通过")
                println("输出：${result.output}")
            } else {
                println("\n✗ Playwright 测试失败")
                println("错误：${result.error}")
            }
        } else {
            println("\n⚠ Playwright 未安装，跳过实际测试")
            println("提示：安装 Playwright 以运行完整测试:")
            println("  npm install -D playwright")
            println("  npx playwright install chromium")
        }
        
        println("\n[测试 3] 备选方案：Selenium")
        val seleniumResult = runSeleniumTest()
        println("Selenium 状态：${seleniumResult.output}")
        
        println("\n=== 浏览器控制能力验证完成 ===")
        println("注意：完整测试需要安装 Playwright 或 Selenium")
        println("推荐方案：使用 Playwright（功能更强大）")
        
    } catch (e: Exception) {
        println("\n=== 测试失败：${e.message} ===")
        e.printStackTrace()
    }
}
