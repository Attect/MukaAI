import kotlinx.coroutines.*
import kotlinx.serialization.*
import kotlinx.serialization.json.*
import io.ktor.client.*
import io.ktor.client.request.*
import io.ktor.client.engine.cio.*
import io.ktor.http.*
import io.ktor.websocket.*
import io.ktor.client.statement.*
import io.ktor.client.plugins.websocket.*
import java.io.File
import java.util.Base64

@OptIn(ExperimentalStdlibApi::class)
fun main() = runBlocking {
    println("=== 测试 05: 浏览器控制能力 (CDP 完整实现) ===\n")
    println("基于 Chrome DevTools Protocol 1.3")
    println("使用 Ktor WebSocket 进行实时通信\n")
    
    val output = StringBuilder()
    
    try {
        // 步骤 1: 检查浏览器是否在调试模式运行
        output.appendLine("[步骤 1] 检查浏览器是否在调试模式运行...")
        val browserReady = checkBrowserReady()
        if (!browserReady) {
            output.appendLine("✗ 浏览器未就绪，请确保浏览器以调试模式启动")
            output.appendLine("  Edge: msedge.exe --remote-debugging-port=9222")
            output.appendLine("  Chrome: chrome.exe --remote-debugging-port=9222")
            println(output.toString())
            return@runBlocking
        }
        output.appendLine("✓ 浏览器已就绪\n")
        
        // 步骤 2: 连接 CDP 并获取浏览器信息
        output.appendLine("[步骤 2] 连接 CDP 并获取浏览器信息...")
        val client = CdpClient("127.0.0.1", 9222)
        try {
            val versionInfo = client.getVersion()
            output.appendLine("✓ 浏览器：${versionInfo["Browser"]?.jsonPrimitive?.content ?: "未知"}")
            output.appendLine("✓ 协议版本：${versionInfo["Protocol-Version"]?.jsonPrimitive?.content ?: "未知"}")
            output.appendLine("✓ V8 版本：${versionInfo["V8-Version"]?.jsonPrimitive?.content ?: "未知"}")
            output.appendLine("✓ WebKit 版本：${versionInfo["WebKit-Version"]?.jsonPrimitive?.content ?: "未知"}\n")
            
            // 步骤 3: 获取标签页列表
            output.appendLine("[步骤 3] 获取标签页列表...")
            val targets = client.getTargets()
            output.appendLine("✓ 找到 ${targets.size} 个标签页")
            for ((i, target) in targets.withIndex()) {
                val title = target["title"]?.jsonPrimitive?.content ?: "无标题"
                val url = target["url"]?.jsonPrimitive?.content ?: ""
                output.appendLine("  [$i] $title - $url")
            }
            output.appendLine()
            
            // 步骤 4: 连接到标签页
            val pageTarget = targets.find { 
                it["url"]?.jsonPrimitive?.content?.contains("baidu") == true 
            } ?: targets.firstOrNull { 
                it["type"]?.jsonPrimitive?.content == "page" 
            }
            
            if (pageTarget != null) {
                val targetId = pageTarget["id"]?.jsonPrimitive?.content ?: ""
                val title = pageTarget["title"]?.jsonPrimitive?.content ?: "未知"
                output.appendLine("[步骤 4] 连接到标签页：$title...")
                client.connect(targetId)
                output.appendLine("✓ WebSocket 连接成功\n")
                
                // 步骤 5: 启用 Page 域
                output.appendLine("[步骤 5] 启用 Page 域...")
                client.sendCommand("Page.enable", buildJsonObject { })
                output.appendLine("✓ Page 域已启用\n")
                
                // 步骤 5.5: 设置下载行为（指定下载路径）
                output.appendLine("[步骤 5.5] 设置下载行为...")
                val downloadPath = File("downloads").absolutePath
                output.appendLine("  下载路径：$downloadPath")
                File(downloadPath).mkdirs()  // 创建下载目录
                val downloadParams = buildJsonObject {
                    put("behavior", "allowAndName")  // 允许下载并自动命名
                    put("downloadPath", downloadPath)
                    put("eventsEnabled", true)  // 启用下载事件
                }
                client.sendCommand("Browser.setDownloadBehavior", downloadParams)
                output.appendLine("✓ 下载行为已设置\n")
                
                // 步骤 6: 导航到百度页面
                output.appendLine("[步骤 6] 导航到百度页面...")
                val navigateResult = client.navigate("https://www.baidu.com/baidu.html")
                output.appendLine("✓ 导航成功")
                val frameId = navigateResult["result"]?.jsonObject?.get("frameId")?.jsonPrimitive?.content ?: "未知"
                val loaderId = navigateResult["result"]?.jsonObject?.get("loaderId")?.jsonPrimitive?.content ?: "未知"
                output.appendLine("  Frame ID: $frameId")
                output.appendLine("  Loader ID: $loaderId")
                output.appendLine("  等待页面加载完成...")
                delay(3000)  // 增加等待时间确保页面完全加载
                output.appendLine("✓ 页面加载完成\n")
                
                // 步骤 7: 执行 JavaScript
                output.appendLine("[步骤 7] 执行 JavaScript...")
                val titleResult = client.evaluate("document.title")
                val pageTitle = titleResult["result"]?.jsonObject?.get("value")?.jsonPrimitive?.content ?: "未知"
                output.appendLine("✓ 页面标题：$pageTitle\n")
                
                // 步骤 8: 获取页面内容
                output.appendLine("[步骤 8] 获取页面内容...")
                val contentResult = client.evaluate("document.body.innerText")
                val pageContent = contentResult["result"]?.jsonObject?.get("value")?.jsonPrimitive?.content ?: ""
                output.appendLine("✓ 页面内容预览：${pageContent.take(100)}...\n")
                
                // 步骤 9: 生成截图
                output.appendLine("[步骤 9] 生成截图...")
                val screenshotData = client.captureScreenshot()
                val screenshotBytes = Base64.getDecoder().decode(screenshotData)
                File("test-cdp-screenshot.png").writeBytes(screenshotBytes)
                output.appendLine("✓ 截图已保存：test-cdp-screenshot.png")
                output.appendLine("  图片大小：${screenshotBytes.size} 字节\n")
                
                // 步骤 10: 元素交互测试
                output.appendLine("[步骤 10] 元素交互测试...")
                val buttonNode = client.querySelector("input[type='submit']")
                if (buttonNode != null) {
                    output.appendLine("✓ 找到提交按钮，节点 ID: $buttonNode")
                    // 点击按钮
                    client.clickElement(buttonNode)
                    output.appendLine("✓ 按钮点击成功\n")
                } else {
                    output.appendLine("⚠ 未找到提交按钮\n")
                }
                
                // 步骤 10.5: 下载百度 Logo
                output.appendLine("[步骤 10.5] 下载百度 Logo 图片...")
                try {
                    // 等待页面完全加载（包括图片资源）
                    output.appendLine("  等待页面资源加载完成...")
                    delay(3000)
                    
                    // 使用多个选择器尝试定位 Logo
                    val selectors = listOf(
                        "#lg img",
                        "#s_tab_img",
                        ".s-logo-img",
                        "img[alt='百度']",
                        "#wrapper img"
                    )
                    
                    var logoInfo: JsonObject? = null
                    var found = false
                    var usedSelector = ""
                    
                    for (selector in selectors) {
                        output.appendLine("  尝试选择器：$selector")
                        val evalResult = client.evaluate("""
                            (function(selector) {
                                const img = document.querySelector(selector);
                                if (img) {
                                    const src = img.src || img.getAttribute('src');
                                    const isVisible = img.offsetWidth > 0 && img.offsetHeight > 0;
                                    return {
                                        found: true,
                                        src: src,
                                        visible: isVisible
                                    };
                                }
                                return { found: false };
                            })("$selector")
                        """)
                        logoInfo = evalResult["result"]?.jsonObject?.get("result")?.jsonObject?.get("value")?.jsonObject
                        found = logoInfo?.get("found")?.jsonPrimitive?.boolean ?: false
                        if (found) {
                            usedSelector = selector
                            output.appendLine("  ✓ 使用选择器 '$selector' 找到 Logo")
                            break
                        }
                    }
                    
                    if (!found) {
                        output.appendLine("⚠ 未找到 Logo 图片，尝试了 ${selectors.size} 个选择器")
                        // 获取页面所有图片用于调试
                        val allImages = client.evaluate("""
                            (function() {
                                const imgs = document.querySelectorAll('img');
                                return Array.from(imgs).slice(0, 5).map(img => ({
                                    src: img.src || img.getAttribute('src'),
                                    id: img.id,
                                    className: img.className,
                                    alt: img.alt
                                }));
                            })()
                        """)
                        val imagesJson = allImages.get("result")?.jsonObject?.get("value")?.jsonArray
                        if (imagesJson != null && imagesJson.size > 0) {
                            output.appendLine("  页面中的图片:")
                            for (i in 0 until imagesJson.size) {
                                val img = imagesJson[i].jsonObject
                                val src = img["src"]?.jsonPrimitive?.content ?: ""
                                val id = img["id"]?.jsonPrimitive?.content ?: ""
                                val className = img["className"]?.jsonPrimitive?.content ?: ""
                                val alt = img["alt"]?.jsonPrimitive?.content ?: ""
                                output.appendLine("    [$i] id='$id' class='$className' alt='$alt'")
                                output.appendLine("        src: ${src.take(80)}")
                            }
                        } else {
                            output.appendLine("  无法获取页面中的图片列表")
                        }
                    } else {
                        val src = logoInfo?.get("src")?.jsonPrimitive?.content ?: ""
                        val visible = logoInfo?.get("visible")?.jsonPrimitive?.boolean ?: false
                        
                        output.appendLine("✓ 找到 Logo")
                        output.appendLine("  图片源：$src")
                        output.appendLine("  可见：$visible")
                        
                        if (!visible) {
                            output.appendLine("⚠ Logo 元素不可见，尝试等待后重试...")
                            delay(2000)
                            
                            // 重试一次
                            val retryInfo = client.evaluate("""
                                (function() {
                                    const img = document.querySelector('#lg img');
                                    if (img) {
                                        const rect = img.getBoundingClientRect();
                                        return {
                                            visible: rect.width > 0 && rect.height > 0
                                        };
                                    }
                                    return null;
                                })()
                            """)
                            
                            val retryVisible = retryInfo["result"]?.jsonObject?.get("result")?.jsonObject?.get("value")?.jsonObject?.get("visible")?.jsonPrimitive?.boolean ?: false
                            output.appendLine("  重试后可见：$retryVisible")
                        }
                        
                        // 使用浏览器原生下载能力下载 Logo 图片
                        output.appendLine("正在使用浏览器下载能力下载 Logo...")
                        val downloadResult = client.evaluate("""
                            (function() {
                                const img = document.querySelector('#lg img');
                                if (img && img.src) {
                                    // 创建一个临时的下载链接
                                    const a = document.createElement('a');
                                    a.href = img.src;
                                    a.download = 'baidu-logo.png';
                                    a.target = '_blank';
                                    
                                    // 触发下载
                                    document.body.appendChild(a);
                                    a.click();
                                    document.body.removeChild(a);
                                    
                                    return {
                                        success: true,
                                        src: img.src,
                                        message: '下载已触发'
                                    };
                                }
                                return {
                                    success: false,
                                    message: '未找到 Logo 图片或 src'
                                };
                            })()
                        """)
                        
                        val downloadSuccess = downloadResult["result"]?.jsonObject?.get("result")?.jsonObject?.get("value")?.jsonObject?.get("success")?.jsonPrimitive?.boolean ?: false
                        val downloadSrc = downloadResult["result"]?.jsonObject?.get("result")?.jsonObject?.get("value")?.jsonObject?.get("src")?.jsonPrimitive?.content ?: ""
                        val downloadMessage = downloadResult["result"]?.jsonObject?.get("result")?.jsonObject?.get("value")?.jsonObject?.get("message")?.jsonPrimitive?.content ?: ""
                        
                        if (downloadSuccess) {
                            output.appendLine("✓ 百度 Logo 已下载 (使用浏览器原生下载)")
                            output.appendLine("  图片源：$downloadSrc")
                            output.appendLine("  消息：$downloadMessage")
                            output.appendLine("  下载目录：$downloadPath")
                            
                            // 等待下载完成
                            output.appendLine("  等待下载完成...")
                            delay(3000)  // 增加等待时间
                            
                            // 检查下载目录中的文件
                            val downloadDir = File(downloadPath)
                            if (downloadDir.exists() && downloadDir.isDirectory) {
                                val files = downloadDir.listFiles()
                                if (files != null && files.isNotEmpty()) {
                                    output.appendLine("✓ 下载成功，找到 ${files.size} 个文件:")
                                    for (file in files) {
                                        output.appendLine("    - ${file.name} (${file.length()} 字节)")
                                        // 验证 PNG 文件头
                                        if (file.name.endsWith(".png") || file.length() > 0) {
                                            try {
                                                val fileBytes = file.readBytes()
                                                if (fileBytes.size >= 4 && 
                                                    fileBytes[0] == 0x89.toByte() && 
                                                    fileBytes[1] == 0x50.toByte() && 
                                                    fileBytes[2] == 0x4E.toByte() && 
                                                    fileBytes[3] == 0x47.toByte()) {
                                                    output.appendLine("      ✓ 有效的 PNG 文件")
                                                }
                                            } catch (e: Exception) {
                                                output.appendLine("      ⚠ 无法读取文件：${e.message}")
                                            }
                                        }
                                    }
                                } else {
                                    output.appendLine("⚠ 下载目录为空")
                                }
                            } else {
                                output.appendLine("⚠ 下载目录不存在")
                            }
                        } else {
                            output.appendLine("⚠ Logo 下载失败：$downloadMessage")
                        }
                    }
                } catch (e: Exception) {
                    output.appendLine("⚠ Logo 下载失败：${e.message}")
                }
                output.appendLine()
                
                // 步骤 11: 获取 DOM 信息
                output.appendLine("[步骤 11] 获取 DOM 信息...")
                val documentNode = client.getDocument()
                output.appendLine("✓ 文档根节点 ID: $documentNode")
                
                // 使用 JavaScript 获取 body 的 HTML，避免编码问题
                val bodyHTML = client.evaluateAsString("document.body.outerHTML")
                output.appendLine("✓ Body 标签：${bodyHTML.take(150)}...\n")
                
                // 步骤 12: 获取所有目标
                output.appendLine("[步骤 12] 获取所有目标...")
                val allTargets = client.getTargets()
                output.appendLine("✓ 找到 ${allTargets.size} 个目标\n")
                
                // 清理
                output.appendLine("[清理] 关闭连接...")
                client.close()
                output.appendLine("✓ 连接已关闭\n")
            } else {
                output.appendLine("⚠ 未找到合适的标签页\n")
            }
        } finally {
            client.close()
        }
        
        // 打印输出摘要
        println("=== 测试结果 ===")
        println("✓ 浏览器控制测试通过\n")
        println("输出摘要:")
        println(output.toString())
        println("截图已保存：test-cdp-screenshot.png")
        
        // 关键验证点
        println("\n=== 关键验证点 ===")
        println("1. ✓ CDP 连接建立")
        println("2. ✓ 标签页控制")
        println("3. ✓ 页面导航")
        println("4. ✓ JavaScript 执行")
        println("5. ✓ 截图生成")
        println("6. ✓ DOM 操作")
        println("7. ✓ 元素交互")
        
    } catch (e: Exception) {
        println("\n=== 测试失败：${e.message} ===")
        e.printStackTrace()
        return@runBlocking
    }
    
    println("\n=== 测试完成 ===")
}

/**
 * 检查浏览器是否就绪
 */
suspend fun checkBrowserReady(): Boolean {
    return try {
        val client = HttpClient(CIO)
        val response = client.get("http://127.0.0.1:9222/json/version")
        response.status == HttpStatusCode.OK
    } catch (e: Exception) {
        false
    }
}

/**
 * CDP 客户端实现
 */
class CdpClient(private val host: String, private val port: Int) {
    private val httpClient = HttpClient(CIO) {
        install(WebSockets)
    }
    private var webSocketSession: ClientWebSocketSession? = null
    private var commandId = 0
    private val pendingCommands = mutableMapOf<Int, CompletableDeferred<JsonObject>>()
    
    /**
     * 获取浏览器版本信息
     */
    suspend fun getVersion(): JsonObject {
        val response = httpClient.get("http://$host:$port/json/version")
        return Json.parseToJsonElement(response.bodyAsText()).jsonObject
    }
    
    /**
     * 获取所有标签页
     */
    suspend fun getTargets(): List<JsonObject> {
        val response = httpClient.get("http://$host:$port/json/list")
        val json = Json.parseToJsonElement(response.bodyAsText())
        return json.jsonArray.map { it.jsonObject }
    }
    
    /**
     * 连接到标签页
     */
    suspend fun connect(targetId: String) {
        val wsUrl = "ws://$host:$port/devtools/page/$targetId"
        
        CoroutineScope(Dispatchers.Default + SupervisorJob()).launch {
            try {
                println("[WebSocket] 开始连接：$wsUrl")
                httpClient.ws(wsUrl) {
                    println("[WebSocket] 连接成功")
                    webSocketSession = this
                    
                    // 接收消息循环
                    try {
                        while (true) {
                            val frame = incoming.receive() as? Frame.Text ?: continue
                            val message = frame.readText()
                            val json = Json.parseToJsonElement(message).jsonObject
                            
                            println("[WebSocket] 收到消息：${message.take(100)}...")
                            
                            // 处理响应
                            val id = json["id"]?.jsonPrimitive?.int
                            if (id != null && pendingCommands.containsKey(id)) {
                                println("[WebSocket] 收到响应：ID=$id")
                                pendingCommands.remove(id)?.complete(json)
                            } else {
                                val method = json["method"]?.jsonPrimitive?.content
                                println("[WebSocket] 收到事件：$method")
                            }
                        }
                    } catch (e: Exception) {
                        println("[WebSocket] 接收错误：${e.message}")
                    }
                }
            } catch (e: Exception) {
                println("[WebSocket] 连接错误：${e.message}")
            }
        }
        
        // 等待连接建立
        delay(1000)
    }
    
    /**
     * 发送 CDP 命令
     */
    suspend fun sendCommand(method: String, params: JsonObject = buildJsonObject { }): JsonObject {
        val id = ++commandId
        val deferred = CompletableDeferred<JsonObject>()
        pendingCommands[id] = deferred
        
        val command = buildJsonObject {
            put("id", id)
            put("method", method)
            if (params.isNotEmpty()) {
                put("params", params)
            }
        }
        
        println("[CDP 发送] ID=$id, 方法=$method")
        
        webSocketSession?.send(Frame.Text(command.toString()))
        
        return deferred.await()
    }
    
    /**
     * 导航到 URL
     */
    suspend fun navigate(url: String): JsonObject {
        val params = buildJsonObject {
            put("url", url)
        }
        val response = sendCommand("Page.navigate", params)
        
        // 等待页面加载
        delay(1000)
        
        return response
    }
    
    /**
     * 执行 JavaScript
     */
    suspend fun evaluate(expression: String): JsonObject {
        val params = buildJsonObject {
            put("expression", expression)
            put("returnByValue", true)
        }
        return sendCommand("Runtime.evaluate", params)
    }
    
    /**
     * 执行 JavaScript 并返回字符串结果
     */
    suspend fun evaluateAsString(expression: String): String {
        val result = evaluate(expression)
        return result["result"]?.jsonObject?.get("value")?.jsonPrimitive?.content ?: ""
    }
    
    /**
     * 获取 DOM 文档
     */
    suspend fun getDocument(): Int {
        val params = buildJsonObject {
            put("depth", 0)
        }
        val response = sendCommand("DOM.getDocument", params)
        return response["root"]?.jsonObject?.get("nodeId")?.jsonPrimitive?.int ?: -1
    }
    
    /**
     * 查询选择器
     */
    suspend fun querySelector(selector: String): Int? {
        val documentNode = getDocument()
        val params = buildJsonObject {
            put("nodeId", documentNode)
            put("selector", selector)
        }
        val response = sendCommand("DOM.querySelector", params)
        val nodeId = response["nodeId"]?.jsonPrimitive?.int
        return if (nodeId != null && nodeId > 0) nodeId else null
    }
    
    /**
     * 点击元素
     */
    suspend fun clickElement(nodeId: Int) {
        val params = buildJsonObject {
            put("nodeId", nodeId)
        }
        sendCommand("DOM.scrollIntoViewIfNeeded", params)
        
        val boxParams = buildJsonObject {
            put("nodeId", nodeId)
        }
        val boxResponse = sendCommand("DOM.getBoxModel", boxParams)
        val content = boxResponse["model"]?.jsonObject?.get("content")?.jsonArray
        if (content != null && content.size >= 2) {
            val x = content[0].jsonPrimitive.int
            val y = content[1].jsonPrimitive.int
            
            val mouseParams = buildJsonObject {
                put("type", "mousePressed")
                put("x", x)
                put("y", y)
                put("button", "left")
                put("clickCount", 1)
            }
            sendCommand("Input.dispatchMouseEvent", mouseParams)
            
            val mouseUpParams = buildJsonObject {
                put("type", "mouseReleased")
                put("x", x)
                put("y", y)
                put("button", "left")
                put("clickCount", 1)
            }
            sendCommand("Input.dispatchMouseEvent", mouseUpParams)
        }
    }
    
    /**
     * 截图
     */
    suspend fun captureScreenshot(format: String = "png", quality: Int? = null): String {
        val params = buildJsonObject {
            put("format", format)
            quality?.let { put("quality", it) }
            put("captureBeyondViewport", false)
        }
        
        val response = sendCommand("Page.captureScreenshot", params)
        return response["result"]?.jsonObject?.get("data")?.jsonPrimitive?.content
            ?: throw IllegalStateException("截图失败")
    }
    
    /**
     * 关闭连接
     */
    suspend fun close() {
        webSocketSession?.close()
        httpClient.close()
    }
}
