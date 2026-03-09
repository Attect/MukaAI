# Ktor 客户端开发指南

本文档整理了 Ktor 官方文档的客户端开发相关内容。

## 文档来源

- **官方文档**: https://ktor.io/docs/
- **版本**: Ktor 3.4.1+ (最新版本)
- **最后更新**: 2026 年 1 月

## 概述

Ktor Client 是一个现代化的异步 HTTP 客户端，具有以下特点:

- **多平台**: 支持 JVM、Android、JavaScript、Native (iOS、macOS、Linux、Windows)
- **异步非阻塞**: 基于 Kotlin 协程
- **可插拔**: 通过插件机制扩展功能
- **类型安全**: 利用 Kotlin 的类型系统
- **引擎可选**: 支持多种 HTTP 引擎

## 快速开始

### 添加依赖

```kotlin
// build.gradle.kts (多平台项目)
kotlin {
    sourceSets {
        val commonMain by getting {
            dependencies {
                implementation("io.ktor:ktor-client-core:$ktor_version")
                implementation("io.ktor:ktor-client-content-negotiation:$ktor_version")
                implementation("io.ktor:ktor-serialization-kotlinx-json:$ktor_version")
            }
        }
        
        val jvmMain by getting {
            dependencies {
                implementation("io.ktor:ktor-client-cio:$ktor_version")
            }
        }
        
        val androidMain by getting {
            dependencies {
                implementation("io.ktor:ktor-client-okhttp:$ktor_version")
            }
        }
        
        val iosMain by getting {
            dependencies {
                implementation("io.ktor:ktor-client-darwin:$ktor_version")
            }
        }
    }
}
```

### 创建客户端

```kotlin
import io.ktor.client.*
import io.ktor.client.engine.cio.*
import io.ktor.client.plugins.*
import io.ktor.client.plugins.logging.*
import io.ktor.client.request.*
import io.ktor.client.statement.*

// 方式 1: 指定引擎
val client = HttpClient(CIO) {
    // 客户端配置
}

// 方式 2: 自动选择引擎 (根据依赖)
val client = HttpClient()

// 方式 3: 多平台项目中使用期望/实际
expect fun createHttpClient(): HttpClient

// JVM/Android 实际
actual fun createHttpClient() = HttpClient(OkHttp)

// iOS 实际
actual fun createHttpClient() = HttpClient(Darwin)
```

### 基本使用

```kotlin
import io.ktor.client.request.*
import io.ktor.client.statement.*

suspend fun fetchData() {
    val client = HttpClient()
    
    try {
        // GET 请求
        val response: HttpResponse = client.get("https://api.example.com/data")
        
        // 获取响应体
        val body: String = response.bodyAsText()
        println("Response: $body")
        
    } finally {
        // 关闭客户端
        client.close()
    }
}

// 使用 use 自动关闭
suspend fun fetchData() {
    HttpClient().use { client ->
        val response = client.get("https://api.example.com/data")
        println(response.bodyAsText())
    }
}
```

## 核心概念

### HttpClient

`HttpClient` 是客户端的核心类，负责发送 HTTP 请求和接收响应。

```kotlin
// 创建客户端
val client = HttpClient(CIO) {
    // 基本配置
    expectSuccess = true  // 验证响应状态码
    
    // 引擎配置
    engine {
        maxConnectionsCount = 1000
        endpoint {
            maxConnectionsPerRoute = 100
            keepAliveTime = 5000
            connectTimeout = 5000
        }
    }
    
    // 安装插件
    install(Logging) {
        level = LogLevel.INFO
    }
}
```

### HttpRequestBuilder

用于配置 HTTP 请求。

```kotlin
client.request("https://api.example.com/users") {
    // HTTP 方法
    method = HttpMethod.Get
    
    // 或者使用便捷函数
    // get("..."), post("..."), put("..."), delete("...")
    
    // 头部
    headers {
        append("Authorization", "Bearer token")
        append("Content-Type", "application/json")
    }
    
    // 快速设置头部
    header("X-Custom-Header", "value")
    
    // 请求体
    setBody(MyData("value"))
    
    // 超时
    timeout {
        requestTimeoutMillis = 30000
        connectTimeoutMillis = 10000
        socketTimeoutMillis = 10000
    }
}
```

### HttpResponse

封装 HTTP 响应。

```kotlin
val response: HttpResponse = client.get("https://api.example.com/data")

// 状态码
val status: HttpStatusCode = response.status
println("Status: ${status.value} ${status.description}")

// 头部
val contentType = response.contentType()
val charset = response.charset()
val etag = response.etag()
val cookies = response.setCookie()
val allHeaders = response.headers

// 获取特定头部
val customHeader = response.headers["X-Custom-Header"]
val splitValues = response.headers.getSplitValues("Set-Cookie")

// 响应体
val text: String = response.bodyAsText()
val bytes: ByteArray = response.readBytes()
val json: MyData = response.body<MyData>()
```

## 发送请求

### GET 请求

```kotlin
// 简单 GET
val response = client.get("https://api.example.com/users")

// 带路径参数
val userId = 123
val response = client.get("https://api.example.com/users/$userId")

// 带查询参数
val response = client.get("https://api.example.com/users") {
    parameter("page", 1)
    parameter("size", 20)
    parameter("sort", "name")
    
    // 多个值
    parameters {
        append("tags", "kotlin")
        append("tags", "ktor")
    }
}

// 带头部
val response = client.get("https://api.example.com/data") {
    header("Authorization", "Bearer token")
    header("Accept", "application/json")
}
```

### POST 请求

```kotlin
// 发送 JSON
val newUser = User("John", 25)
val response = client.post("https://api.example.com/users") {
    contentType(ContentType.Application.Json)
    setBody(newUser)
}

// 发送表单数据
val response = client.post("https://api.example.com/login") {
    contentType(ContentType.Application.FormUrlEncoded)
    setBody(FormDataContent(Parameters.build {
        append("username", "john")
        append("password", "secret")
    }))
}

// 发送 multipart 表单
val response = client.post("https://api.example.com/upload") {
    setBody(MultiPartFormDataContent(formData {
        append("username", "john")
        append("file", File("path/to/file.jpg"), ContentType.Image.JPEG)
    }))
}

// 发送纯文本
val response = client.post("https://api.example.com/text") {
    contentType(ContentType.Text.Plain)
    setBody("Hello, World!")
}
```

### PUT 请求

```kotlin
val updatedUser = User("John", 26)
val response = client.put("https://api.example.com/users/123") {
    contentType(ContentType.Application.Json)
    setBody(updatedUser)
}
```

### DELETE 请求

```kotlin
val response = client.delete("https://api.example.com/users/123")
```

### PATCH 请求

```kotlin
val updateData = mapOf("age" to 26)
val response = client.patch("https://api.example.com/users/123") {
    contentType(ContentType.Application.Json)
    setBody(updateData)
}
```

## 接收响应

### 响应体

```kotlin
// String
val text: String = response.bodyAsText()

// ByteArray
val bytes: ByteArray = response.readBytes()

// 数据类 (需要 ContentNegotiation 插件)
val user: User = response.body<User>()

// Map
val map: Map<String, Any> = response.body()

// List
val users: List<User> = response.body()
```

### 流式处理

```kotlin
import io.ktor.utils.io.*

// 分块处理
client.get("https://api.example.com/large-file").use { response ->
    val channel: ByteReadChannel = response.bodyAsChannel()
    
    while (!channel.isClosedForRead) {
        val packet = channel.readRemaining(4096)
        while (packet.isNotEmpty) {
            val bytes = packet.readBytes()
            // 处理字节
        }
    }
}

// 直接写入文件
client.get("https://api.example.com/file.pdf").use { response ->
    val channel = response.bodyAsChannel()
    File("download.pdf").writeChannel().use { output ->
        channel.copyAndClose(output)
    }
}

// 使用 Sink
client.get("https://api.example.com/file.pdf").use { response ->
    val channel = response.bodyAsChannel()
    File("download.pdf").sink().use { sink ->
        channel.readTo(sink)
    }
}
```

### 下载进度

```kotlin
import io.ktor.utils.io.*

client.get("https://api.example.com/large-file").use { response ->
    val channel = response.bodyAsChannel()
    val totalBytes = response.contentLength ?: -1L
    var downloadedBytes = 0L
    
    val file = File("download.pdf")
    file.outputStream().use { output ->
        while (!channel.isClosedForRead) {
            val packet = channel.readRemaining(4096)
            while (packet.isNotEmpty) {
                val bytes = packet.readBytes()
                output.write(bytes)
                downloadedBytes += bytes.size
                
                // 更新进度
                if (totalBytes > 0) {
                    val progress = (downloadedBytes * 100.0 / totalBytes).toInt()
                    println("Download progress: $progress%")
                }
            }
        }
    }
}
```

## 插件

### ContentNegotiation

处理请求和响应的序列化/反序列化。

```kotlin
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.serialization.json.Json

val client = HttpClient(CIO) {
    install(ContentNegotiation) {
        json(Json {
            prettyPrint = true
            isLenient = true
            ignoreUnknownKeys = true
            encodeDefaults = true
        })
        
        // XML (需要额外依赖)
        // xml { /* 配置 */ }
    }
}

// 使用
val user = client.get<User>("https://api.example.com/users/123")
client.post("https://api.example.com/users") {
    setBody(User("John", 25))
}
```

### Logging

记录 HTTP 请求和响应。

```kotlin
import io.ktor.client.plugins.logging.*

val client = HttpClient(CIO) {
    install(Logging) {
        logger = Logger.DEFAULT  // 或 Logger.SIMPLE, Logger.EMPTY
        level = LogLevel.INFO    // ALL, HEADERS, BODY, INFO, ERROR, NONE
        
        // 过滤器
        filter { request ->
            request.url.host.contains("example.com")
        }
        
        // 格式化
        sanitizeHeader { header -> header == HttpHeaders.Authorization }
    }
}
```

### Auth

处理认证。

```kotlin
import io.ktor.client.plugins.auth.*
import io.ktor.client.plugins.auth.providers.*

val client = HttpClient(CIO) {
    install(Auth) {
        // Basic 认证
        basic {
            sendWithoutRequest { true }
            credentials {
                BasicAuthCredentials("username", "password")
            }
        }
        
        // Bearer 认证
        bearer {
            sendWithoutRequest { false }
            
            // 加载 token
            loadTokens {
                val token = loadTokenFromStorage()
                BearerTokens(
                    accessToken = token,
                    refreshToken = null
                )
            }
            
            // 刷新 token
            refreshTokens {
                val newToken = refreshToken()
                BearerTokens(
                    accessToken = newToken,
                    refreshToken = null
                )
            }
        }
    }
}

// 访问认证提供者
val authProvider = client.authProvider<BearerAuthProvider>()
authProvider?.clearToken()  // 清除缓存的 token

// 清除所有认证
client.clearAuthTokens()
```

### HttpTimeout

设置超时。

```kotlin
import io.ktor.client.plugins.*

val client = HttpClient(CIO) {
    install(HttpTimeout) {
        requestTimeoutMillis = 30000
        connectTimeoutMillis = 10000
        socketTimeoutMillis = 10000
    }
}

// 或使用便捷函数
val client = HttpClient(CIO) {
    timeout {
        requestTimeoutMillis = 30000
        connectTimeoutMillis = 10000
        socketTimeoutMillis = 10000
    }
}
```

### HttpCookies

管理 Cookie。

```kotlin
import io.ktor.client.plugins.cookies.*
import io.ktor.http.*

val client = HttpClient(CIO) {
    install(HttpCookies) {
        // 使用自定义 Cookie 存储
        storage = AcceptAllCookiesStorage()
    }
}

// 设置 Cookie
client.get("https://api.example.com/data") {
    cookie("session", "abc123") {
        domain = "api.example.com"
        path = "/"
        secure = true
        httpOnly = true
    }
}

// 获取 Cookie
val cookies = client.cookieStorage.get("https://api.example.com")
```

## 引擎

### JVM 引擎

**CIO (默认推荐)**:

```kotlin
val client = HttpClient(CIO) {
    engine {
        maxConnectionsCount = 1000
        endpoint {
            maxConnectionsPerRoute = 100
            keepAliveTime = 5000
            connectTimeout = 5000
        }
        https {
            serverName = "api.example.com"
            trustManager = myCustomTrustManager
        }
    }
}
```

**OkHttp**:

```kotlin
val client = HttpClient(OkHttp) {
    engine {
        config {
            followRedirects(true)
            connectTimeout(10, TimeUnit.SECONDS)
            readTimeout(30, TimeUnit.SECONDS)
        }
        addInterceptor(myInterceptor)
        addNetworkInterceptor(myNetworkInterceptor)
        
        // 使用预配置的 OkHttpClient
        // preconfigured = okHttpClientInstance
    }
}
```

**Apache5**:

```kotlin
val client = HttpClient(Apache5) {
    engine {
        followRedirects = true
        socketTimeout = 10000
        connectTimeout = 10000
        connectionRequestTimeout = 20000
        
        configureConnectionManager {
            setMaxConnPerRoute(1000)
            setMaxConnTotal(2000)
        }
        
        customizeClient {
            setProxy(HttpHost("proxy.example.com", 8080))
        }
    }
}
```

**Java (Java 11+)**:

```kotlin
val client = HttpClient(Java) {
    engine {
        threadsCount = 8
        pipelining = true
        proxy = ProxyBuilder.http("http://proxy.example.com/")
        protocolVersion = java.net.http.HttpClient.Version.HTTP_2
    }
}
```

### Android 引擎

**OkHttp (推荐)**:

```kotlin
val client = HttpClient(OkHttp) {
    engine {
        config {
            retryOnConnectionFailure(true)
            connectTimeout(0, TimeUnit.SECONDS)
        }
    }
}
```

**Android**:

```kotlin
val client = HttpClient(Android) {
    engine {
        connectTimeout = 100000
        socketTimeout = 100000
        proxy = Proxy(Proxy.Type.HTTP, InetSocketAddress("localhost", 8080))
    }
}
```

### Native 引擎

**Darwin (iOS、macOS)**:

```kotlin
val client = HttpClient(Darwin) {
    engine {
        configureRequest {
            setAllowsCellularAccess(true)
            setTimeoutInterval(30.0)
        }
        configureSession {
            setHTTPMaximumConnectionsPerHost(10)
        }
    }
}
```

**Curl (Linux、macOS、Windows)**:

```kotlin
val client = HttpClient(Curl) {
    engine {
        sslVerify = false  // 仅用于测试
    }
}
```

**WinHttp (Windows)**:

```kotlin
val client = HttpClient(WinHttp) {
    engine {
        protocolVersion = HttpProtocolVersion.HTTP_1_1
    }
}
```

### JavaScript 引擎

```kotlin
val client = HttpClient(Js) {
    // 使用 fetch API
}

// 或使用便捷函数
val client = JsClient()
```

## 多平台项目

### 期望/实际模式

```kotlin
// commonMain
expect fun createHttpClient(): HttpClient

// jvmMain
actual fun createHttpClient(): HttpClient {
    return HttpClient(OkHttp) {
        engine {
            config {
                followRedirects(true)
            }
        }
    }
}

// androidMain
actual fun createHttpClient(): HttpClient {
    return HttpClient(OkHttp) {
        engine {
            config {
                retryOnConnectionFailure(true)
                connectTimeout(0, TimeUnit.SECONDS)
            }
        }
    }
}

// iosMain (iosArm64, iosX64, iosSimulatorArm64)
actual fun createHttpClient(): HttpClient {
    return HttpClient(Darwin) {
        engine {
            configureRequest {
                setAllowsCellularAccess(true)
            }
        }
    }
}

// 使用
suspend fun fetchData() {
    val client = createHttpClient()
    try {
        val response = client.get("https://api.example.com/data")
        // 处理响应
    } finally {
        client.close()
    }
}
```

### 共享客户端配置

```kotlin
// commonMain
expect class PlatformHttpClient {
    companion object {
        fun create(): HttpClient
    }
}

// commonMain
class ApiClient(private val client: HttpClient) {
    suspend fun getUsers(): List<User> {
        return client.get("https://api.example.com/users").body()
    }
}

// 使用
val client = PlatformHttpClient.create()
val apiClient = ApiClient(client)
```

## 高级特性

### WebSocket

```kotlin
import io.ktor.client.plugins.websocket.*
import io.ktor.websocket.*

val client = HttpClient(CIO) {
    install(WebSockets)
}

client.webSocket("wss://api.example.com/ws") {
    // 发送消息
    send(Frame.Text("Hello"))
    send(Frame.Binary(byteArrayOf(1, 2, 3)))
    
    // 接收消息
    for (frame in incoming) {
        when (frame) {
            is Frame.Text -> {
                val text = frame.readText()
                println("Received: $text")
            }
            is Frame.Binary -> {
                val data = frame.data
                // 处理二进制数据
            }
            is Frame.Close -> {
                // 处理关闭
            }
        }
    }
}
```

### 事件流 (Server-Sent Events)

```kotlin
import io.ktor.client.plugins.sse.*

val client = HttpClient(CIO) {
    install(SSE) {
        // 配置
    }
}

client.sse("https://api.example.com/events") {
    for (event in events) {
        println("Event: ${event.event}, Data: ${event.data}")
    }
}
```

### 自定义插件

```kotlin
import io.ktor.client.*
import io.ktor.client.plugins.*
import io.ktor.util.*

class CustomPlugin {
    class Config {
        var customValue: String = "default"
    }
    
    companion object Plugin : HttpClientPlugin<Config, CustomPlugin> {
        override fun prepare(block: Config.() -> Unit) = CustomPlugin()
        
        override fun install(
            plugin: CustomPlugin,
            scope: HttpClient
        ) {
            scope.requestPipeline.intercept(HttpRequestPipeline.Before) {
                context.headers.append("X-Custom", plugin.customValue)
            }
        }
    }
}

// 使用
val client = HttpClient(CIO) {
    install(CustomPlugin) {
        customValue = "custom"
    }
}
```

## 最佳实践

### 客户端复用

```kotlin
// ❌ 不要为每个请求创建新客户端
suspend fun fetchData() {
    HttpClient().use { client ->
        client.get("...")
    }
}

// ✅ 复用客户端实例
class ApiService {
    private val client = HttpClient(CIO) {
        install(ContentNegotiation) { json() }
        install(Logging)
    }
    
    suspend fun fetchData(): String {
        return client.get("...").bodyAsText()
    }
    
    fun close() {
        client.close()
    }
}
```

### 错误处理

```kotlin
import io.ktor.client.call.*
import io.ktor.client.statement.*
import io.ktor.util.network.*

suspend fun safeRequest() {
    try {
        val response = client.get("https://api.example.com/data")
        
        // 检查状态码
        if (!response.status.isSuccess()) {
            throw Exception("Request failed: ${response.status}")
        }
        
        val data = response.body<Data>()
        
    } catch (e: ClientRequestException) {
        // 4xx 错误
        println("Client error: ${e.response.status}")
    } catch (e: ServerResponseException) {
        // 5xx 错误
        println("Server error: ${e.response.status}")
    } catch (e: HttpRequestTimeoutException) {
        // 超时
        println("Request timeout")
    } catch (e: Exception) {
        // 其他错误
        println("Error: ${e.message}")
    }
}
```

### 取消协程

```kotlin
import kotlinx.coroutines.*

val job = launch {
    try {
        val response = client.get("https://api.example.com/data")
        // 处理响应
    } catch (e: CancellationException) {
        // 协程被取消
        println("Request cancelled")
    }
}

// 取消请求
job.cancel()
```

### 依赖注入

```kotlin
// 使用 Koin
val appModule = module {
    single {
        HttpClient(CIO) {
            install(ContentNegotiation) { json() }
            install(Logging)
        }
    }
    
    single { ApiService(get()) }
}

class ApiService(private val client: HttpClient) {
    suspend fun fetchData(): Data {
        return client.get("...").body()
    }
}
```

## 测试

```kotlin
import io.ktor.client.engine.mock.*

// 使用 MockEngine
val mockEngine = MockEngine { request ->
    respond(
        content = """{"name": "John", "age": 25}""",
        status = HttpStatusCode.OK,
        headers = headersOf("Content-Type" to listOf("application/json"))
    )
}

val client = HttpClient(mockEngine) {
    install(ContentNegotiation) { json() }
}

// 测试
val user = client.get<User>("https://api.example.com/users/123")
assertEquals("John", user.name)
```

## 参考资源

- [Ktor 客户端文档](https://ktor.io/docs/client.html)
- [Ktor GitHub](https://github.com/ktorio/ktor)
- [Ktor 示例项目](https://github.com/ktorio/ktor-samples)
- [Ktor 插件注册表](https://ktor.io/docs/client-plugins.html)

## 更新日志

- **2026-03-07**: 初始版本，整理 Ktor 客户端开发文档
