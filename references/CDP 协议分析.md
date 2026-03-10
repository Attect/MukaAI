# Chrome DevTools Protocol (CDP) 协议分析

## 概述

Chrome DevTools Protocol 是一个基于 JSON 的协议，允许客户端与 Chrome/Chromium 浏览器进行通信，实现浏览器自动化、调试和分析功能。

**协议版本**: 1.3 (当前最新版本)

**官方仓库**: https://github.com/ChromeDevTools/devtools-protocol

## 一、协议架构

### 1.1 通信方式

CDP 支持两种通信方式:

#### HTTP REST API (用于发现和创建)
```
GET http://localhost:9222/json/version          # 获取浏览器版本信息
GET http://localhost:9222/json/list             # 获取所有标签页列表
GET http://localhost:9222/json/new              # 创建新标签页
POST http://localhost:9222/json/activate/:id    # 激活标签页
POST http://localhost:9222/json/close/:id       # 关闭标签页
```

#### WebSocket (用于实时命令和事件)
```
ws://localhost:9222/devtools/page/:pageId       # 连接到特定页面
ws://localhost:9222/devtools/browser/:browserId # 连接到浏览器
```

### 1.2 消息格式

#### WebSocket 命令格式
```json
{
  "id": 1,                    // 命令 ID (整数)
  "method": "Page.navigate",  // 方法名 (域名。方法名)
  "params": {                 // 参数对象 (可选)
    "url": "https://example.com"
  }
}
```

#### WebSocket 响应格式
```json
{
  "id": 1,                    // 对应请求的 ID
  "result": {                 // 返回结果
    "frameId": "xxx",
    "loaderId": "yyy"
  }
  // 或者错误情况:
  // "error": {
  //   "code": -32602,
  //   "message": "Invalid parameters"
  // }
}
```

#### WebSocket 事件格式
```json
{
  "method": "Page.loadEventFired",  // 事件名
  "params": {                       // 事件参数
    "timestamp": 123456.789
  }
}
```

## 二、核心 Domain 和命令

### 2.1 Target 域 (目标管理)

用于管理浏览器的目标 (标签页、iframe、worker 等)。

#### 关键命令:

**1. `Target.getTargetInfo`** - 获取目标信息
```json
{
  "id": 1,
  "method": "Target.getTargetInfo",
  "params": {
    "targetId": "page-id-xxx"
  }
}
```

返回:
```json
{
  "id": 1,
  "result": {
    "targetInfo": {
      "targetId": "page-id-xxx",
      "type": "page",
      "title": "Example Domain",
      "url": "https://www.example.com/",
      "attached": true,
      "canAccessOpener": false
    }
  }
}
```

**2. `Target.attachToTarget`** - 附加到目标
```json
{
  "id": 2,
  "method": "Target.attachToTarget",
  "params": {
    "targetId": "page-id-xxx",
    "flatten": true
  }
}
```

返回:
```json
{
  "id": 2,
  "result": {
    "sessionId": "session-id-yyy"
  }
}
```

**3. `Target.createTarget`** - 创建新标签页
```json
{
  "id": 3,
  "method": "Target.createTarget",
  "params": {
    "url": "https://www.example.com",
    "width": 1280,
    "height": 720,
    "newWindow": true,
    "background": false
  }
}
```

返回:
```json
{
  "id": 3,
  "result": {
    "targetId": "new-page-id-zzz"
  }
}
```

**4. `Target.closeTarget`** - 关闭目标
```json
{
  "id": 4,
  "method": "Target.closeTarget",
  "params": {
    "targetId": "page-id-xxx"
  }
}
```

返回:
```json
{
  "id": 4,
  "result": {
    "success": true
  }
}
```

**5. `Target.getTargets`** - 获取所有目标
```json
{
  "id": 5,
  "method": "Target.getTargets",
  "params": {
    "filter": [
      {"type": "page"}
    ]
  }
}
```

返回:
```json
{
  "id": 5,
  "result": {
    "targetInfos": [
      {
        "targetId": "page-1",
        "type": "page",
        "title": "Tab 1",
        "url": "https://example.com/1"
      },
      {
        "targetId": "page-2",
        "type": "page",
        "title": "Tab 2",
        "url": "https://example.com/2"
      }
    ]
  }
}
```

### 2.2 Page 域 (页面操作)

用于处理页面相关的操作，如导航、截图、打印等。

#### 关键命令:

**1. `Page.navigate`** - 导航到 URL
```json
{
  "id": 10,
  "method": "Page.navigate",
  "params": {
    "url": "https://www.example.com",
    "referrer": "",
    "transitionType": "auto",
    "frameId": "frame-id-xxx",
    "referrerPolicy": "strict-origin-when-cross-origin"
  }
}
```

返回:
```json
{
  "id": 10,
  "result": {
    "frameId": "frame-id-xxx",
    "loaderId": "loader-id-yyy",
    "errorText": ""  // 如果有错误
  }
}
```

**2. `Page.reload`** - 刷新页面
```json
{
  "id": 11,
  "method": "Page.reload",
  "params": {
    "ignoreCache": false,
    "scriptToEvaluateOnLoad": ""
  }
}
```

**3. `Page.captureScreenshot`** - 截图
```json
{
  "id": 12,
  "method": "Page.captureScreenshot",
  "params": {
    "format": "png",        // png 或 jpeg
    "quality": 80,          // JPEG 质量 (0-100)
    "fromSurface": true,    // 从表面截图
    "captureBeyondViewport": false,
    "clip": {               // 可选的裁剪区域
      "x": 0,
      "y": 0,
      "width": 800,
      "height": 600,
      "scale": 1
    }
  }
}
```

返回:
```json
{
  "id": 12,
  "result": {
    "data": "iVBORw0KGgoAAAANSUhEUgAA..."  // Base64 编码的图片
  }
}
```

**4. `Page.printToPDF`** - 打印为 PDF
```json
{
  "id": 13,
  "method": "Page.printToPDF",
  "params": {
    "landscape": false,
    "displayHeaderFooter": false,
    "printBackground": false,
    "scale": 1,
    "paperWidth": 8.5,
    "paperHeight": 11,
    "marginTop": 0.4,
    "marginBottom": 0.4,
    "marginLeft": 0.4,
    "marginRight": 0.4,
    "pageRanges": "",
    "headerTemplate": "",
    "footerTemplate": "",
    "preferCSSPageSize": false
  }
}
```

返回:
```json
{
  "id": 13,
  "result": {
    "data": "JVBERi0xLjQKJeLjz9MK..."  // Base64 编码的 PDF
  }
}
```

**5. `Page.close`** - 关闭页面
```json
{
  "id": 14,
  "method": "Page.close"
}
```

**6. `Page.getFrameTree`** - 获取帧树
```json
{
  "id": 15,
  "method": "Page.getFrameTree"
}
```

返回:
```json
{
  "id": 15,
  "result": {
    "frameTree": {
      "frame": {
        "id": "main-frame-id",
        "url": "https://example.com",
        "urlFragment": "",
        "domainAndRegistry": "example.com",
        "securityOrigin": "https://example.com",
        "mimeType": "text/html",
        "adFrameStatus": {
          "adFrameType": "none"
        }
      },
      "childFrames": []
    }
  }
}
```

### 2.3 Runtime 域 (JavaScript 执行)

用于执行 JavaScript 代码和评估表达式。

#### 关键命令:

**1. `Runtime.evaluate`** - 执行 JavaScript
```json
{
  "id": 20,
  "method": "Runtime.evaluate",
  "params": {
    "expression": "document.title",
    "objectGroup": "console",
    "includeCommandLineAPI": false,
    "silent": false,
    "returnByValue": true,
    "generatePreview": true,
    "userGesture": true,
    "awaitPromise": false,
    "throwOnSideEffect": false,
    "timeout": 500,
    "disableBreaks": false,
    "replMode": false,
    "allowUnsafeEvalBlockedByCSP": false,
    "serializationOptions": {
      "serialization": "json"
    }
  }
}
```

返回:
```json
{
  "id": 20,
  "result": {
    "result": {
      "type": "string",
      "value": "Example Domain"
    },
    "exceptionDetails": null
  }
}
```

**2. `Runtime.callFunctionOn`** - 调用函数
```json
{
  "id": 21,
  "method": "Runtime.callFunctionOn",
  "params": {
    "functionDeclaration": "function() { return this; }",
    "objectId": "object-id-xxx",
    "arguments": [],
    "returnByValue": true
  }
}
```

### 2.4 DOM 域 (DOM 操作)

用于查询和操作 DOM 树。

#### 关键命令:

**1. `DOM.getDocument`** - 获取 DOM 文档
```json
{
  "id": 30,
  "method": "DOM.getDocument",
  "params": {
    "depth": -1,
    "pierce": true
  }
}
```

**2. `DOM.querySelector`** - 查询选择器
```json
{
  "id": 31,
  "method": "DOM.querySelector",
  "params": {
    "nodeId": 1,
    "selector": "#main"
  }
}
```

返回:
```json
{
  "id": 31,
  "result": {
    "nodeId": 5
  }
}
```

**3. `DOM.getOuterHTML`** - 获取外部 HTML
```json
{
  "id": 32,
  "method": "DOM.getOuterHTML",
  "params": {
    "nodeId": 5
  }
}
```

返回:
```json
{
  "id": 32,
  "result": {
    "outerHTML": "<div id=\"main\">...</div>"
  }
}
```

### 2.5 Network 域 (网络监控)

用于监控和分析网络请求。

#### 关键命令:

**1. `Network.enable`** - 启用网络事件
```json
{
  "id": 40,
  "method": "Network.enable",
  "params": {
    "maxTotalBufferSize": 10000000,
    "maxResourceBufferSize": 5000000,
    "maxPostDataSize": 5000000
  }
}
```

**2. `Network.getRequestResponseBody`** - 获取请求/响应体
```json
{
  "id": 41,
  "method": "Network.getRequestResponseBody",
  "params": {
    "requestId": "request-id-xxx"
  }
}
```

### 2.6 Input 域 (输入事件)

用于模拟用户输入事件。

#### 关键命令:

**1. `Input.dispatchMouseEvent`** - 发送鼠标事件
```json
{
  "id": 50,
  "method": "Input.dispatchMouseEvent",
  "params": {
    "type": "mousePressed",  // mousePressed, mouseReleased, mouseMoved
    "x": 100,
    "y": 200,
    "button": "left",        // left, right, middle
    "clickCount": 1
  }
}
```

**2. `Input.dispatchKeyEvent`** - 发送键盘事件
```json
{
  "id": 51,
  "method": "Input.dispatchKeyEvent",
  "params": {
    "type": "keyDown",       // keyDown, keyUp, char
    "text": "a",
    "unmodifiedText": "a",
    "windowsVirtualKeyCode": 65
  }
}
```

## 三、HTTP 端点详解

### 3.1 版本信息

**请求**:
```
GET http://localhost:9222/json/version
```

**响应**:
```json
{
  "Browser": "Chrome/145.0.7632.160",
  "Protocol-Version": "1.3",
  "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36",
  "V8-Version": "14.5.201.17",
  "WebKit-Version": "537.36 (@662e0d7961bd91ebe77fe6c52f369e45647af51c)",
  "webSocketDebuggerUrl": "ws://127.0.0.1:9222/devtools/browser/a359cba2-89ca-492e-8ffb-fb106bdf1227"
}
```

### 3.2 标签页列表

**请求**:
```
GET http://localhost:9222/json/list
```

**响应**:
```json
[
  {
    "description": "",
    "devtoolsFrontendUrl": "https://chrome-devtools-frontend.appspot.com/serve_rev/@662e0d7961bd91ebe77fe6c52f369e45647af51c/inspector.html?ws=127.0.0.1:9222/devtools/page/4C0A9CC0087A2CD74C4E29CB111FF168",
    "id": "4C0A9CC0087A2CD74C4E29CB111FF168",
    "title": "Example Domain",
    "type": "page",
    "url": "https://www.example.com/",
    "webSocketDebuggerUrl": "ws://127.0.0.1:9222/devtools/page/4C0A9CC0087A2CD74C4E29CB111FF168"
  }
]
```

### 3.3 创建标签页

**请求**:
```
GET http://localhost:9222/json/new?https://www.example.com
```

**响应**:
```json
{
  "description": "",
  "devtoolsFrontendUrl": "https://chrome-devtools-frontend.appspot.com/serve_rev/...",
  "id": "new-page-id",
  "title": "",
  "type": "page",
  "url": "https://www.example.com/",
  "webSocketDebuggerUrl": "ws://127.0.0.1:9222/devtools/page/new-page-id"
}
```

### 3.4 关闭标签页

**请求**:
```
POST http://localhost:9222/json/close/:pageId
```

**响应**:
```
Target is closing
```

### 3.5 激活标签页

**请求**:
```
POST http://localhost:9222/json/activate/:pageId
```

**响应**:
```
Target is activated
```

## 四、WebSocket 连接流程

### 4.1 连接建立

```
1. 从 HTTP 端点获取 WebSocket URL
   GET http://localhost:9222/json/list
   -> 获取 webSocketDebuggerUrl

2. 连接到 WebSocket
   ws://127.0.0.1:9222/devtools/page/4C0A9CC0087A2CD74C4E29CB111FF168

3. 发送命令
   {"id": 1, "method": "Page.enable"}

4. 接收响应
   {"id": 1, "result": {}}

5. 接收事件
   {"method": "Page.frameNavigated", "params": {...}}
```

### 4.2 会话管理

```
1. 使用 Target.attachToTarget 获取 sessionId
   {"id": 1, "method": "Target.attachToTarget", "params": {"targetId": "xxx"}}
   -> {"id": 1, "result": {"sessionId": "session-yyy"}}

2. 使用 sessionId 发送命令 (扁平模式)
   {"id": 2, "sessionId": "session-yyy", "method": "Page.navigate", "params": {...}}
```

## 五、常见使用场景

### 5.1 导航并等待加载完成

```json
// 1. 启用 Page 域
{"id": 1, "method": "Page.enable"}

// 2. 导航
{"id": 2, "method": "Page.navigate", "params": {"url": "https://example.com"}}

// 3. 等待加载完成事件 (从服务器推送)
{"method": "Page.loadEventFired", "params": {"timestamp": 123456.789}}
```

### 5.2 执行 JavaScript 并获取结果

```json
// 执行 JavaScript
{
  "id": 10,
  "method": "Runtime.evaluate",
  "params": {
    "expression": "document.querySelector('h1').textContent",
    "returnByValue": true
  }
}

// 响应
{
  "id": 10,
  "result": {
    "result": {
      "type": "string",
      "value": "Example Domain"
    }
  }
}
```

### 5.3 截图

```json
// 截图
{
  "id": 20,
  "method": "Page.captureScreenshot",
  "params": {
    "format": "png",
    "fromSurface": true
  }
}

// 响应
{
  "id": 20,
  "result": {
    "data": "iVBORw0KGgoAAAANSUhEUgAA..."
  }
}
```

### 5.4 点击元素

```json
// 1. 获取 DOM 文档
{"id": 30, "method": "DOM.getDocument"}

// 2. 查询元素
{
  "id": 31,
  "method": "DOM.querySelector",
  "params": {
    "nodeId": 1,
    "selector": "#button"
  }
}

// 3. 获取元素的边界
{
  "id": 32,
  "method": "DOM.getBoxModel",
  "params": {
    "nodeId": 5
  }
}

// 4. 发送鼠标点击事件
{
  "id": 33,
  "method": "Input.dispatchMouseEvent",
  "params": {
    "type": "mousePressed",
    "x": 150,
    "y": 200,
    "button": "left",
    "clickCount": 1
  }
}

{
  "id": 34,
  "method": "Input.dispatchMouseEvent",
  "params": {
    "type": "mouseReleased",
    "x": 150,
    "y": 200,
    "button": "left",
    "clickCount": 1
  }
}
```

## 六、错误处理

### 6.1 错误响应格式

```json
{
  "id": 1,
  "error": {
    "code": -32602,
    "message": "Invalid parameters",
    "data": "Invalid url"
  }
}
```

### 6.2 常见错误码

- `-32700`: Parse error
- `-32600`: Invalid Request
- `-32601`: Method not found
- `-32602`: Invalid parameters
- `-32603`: Internal error
- `-32000`: Server error (Chrome 特定错误)

## 七、安全考虑

### 7.1 调试端口访问控制

- 默认情况下，调试端口只接受 localhost 连接
- 如需远程访问，需启动时添加 `--remote-allow-origins=*`
- 建议使用防火墙限制访问

### 7.2 认证机制

- CDP 本身没有内置认证
- 需要通过网络层进行保护
- 可以使用 WebSocket token 或 HTTP Basic Auth

## 八、Kotlin 实现示例

### 8.1 HTTP 请求示例

```kotlin
// 获取标签页列表
suspend fun getTabs(port: Int = 9222): List<CdpTab> {
    val url = URL("http://127.0.0.1:$port/json/list")
    val connection = url.openConnection() as HttpURLConnection
    connection.requestMethod = "GET"
    
    val response = connection.inputStream.bufferedReader().readText()
    return parseTabsFromJson(response)
}

// 创建新标签页
suspend fun createTab(port: Int = 9222, url: String): CdpTab? {
    val createUrl = URL("http://127.0.0.1:$port/json/new?$url")
    val connection = createUrl.openConnection() as HttpURLConnection
    connection.requestMethod = "POST"
    
    val response = connection.inputStream.bufferedReader().readText()
    return parseTabFromJson(response)
}
```

### 8.2 WebSocket 通信示例

```kotlin
// 使用 ktor-client-websockets
suspend fun sendCdpCommand(
    webSocketUrl: String,
    method: String,
    params: Map<String, Any> = emptyMap()
): CdpResponse {
    return HttpClient(CIO).ws(webSocketUrl) {
        val command = CdpCommand(
            id = 1,
            method = method,
            params = params
        )
        send(command.toJson())
        
        val response = incoming.receive().frame?.readText()
        return parseCdpResponse(response)
    }
}
```

## 九、参考资料

- **官方协议文档**: https://chromedevtools.github.io/devtools-protocol/
- **GitHub 仓库**: https://github.com/ChromeDevTools/devtools-protocol
- **Chrome 调试协议指南**: https://developer.chrome.com/docs/devtools/protocol/
- **Playwright CDP 实现**: https://playwright.dev/docs/api/class-browsertype#browser-type-connect-over-cdp
- **Puppeteer CDP 实现**: https://pptr.dev/api/puppeteer.cdpsession

## 十、总结

CDP 是一个强大的浏览器自动化协议，提供了:

1. **完整的浏览器控制**: 导航、截图、打印、DOM 操作等
2. **实时事件监听**: 页面加载、网络请求、控制台消息等
3. **JavaScript 执行**: 评估表达式、调用函数
4. **调试能力**: 断点、单步执行、变量检查
5. **网络监控**: 请求/响应拦截、性能分析

通过 HTTP + WebSocket 的组合，可以实现复杂的浏览器自动化场景。
