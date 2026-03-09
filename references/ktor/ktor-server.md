# Ktor 服务端开发指南

本文档整理了 Ktor 官方文档的服务端开发相关内容。

## 文档来源

- **官方文档**: https://ktor.io/docs/
- **版本**: Ktor 3.4.1+ (最新版本)
- **最后更新**: 2026 年 1 月

## 概述

Ktor 是一个现代化的异步 Web 框架，用于构建服务端应用程序。它具有以下特点:

- **异步非阻塞**: 基于协程的异步编程模型
- **轻量级**: 核心库小巧，按需添加功能
- **可插拔**: 通过插件机制扩展功能
- **多平台**: 支持 JVM、Native 等平台
- **类型安全**: 利用 Kotlin 的类型系统

## 快速开始

### 创建项目

使用 Ktor 项目向导或手动创建项目:

```kotlin
// build.gradle.kts
plugins {
    kotlin("jvm") version "2.3.10"
    application
}

dependencies {
    implementation("io.ktor:ktor-server-core:$ktor_version")
    implementation("io.ktor:ktor-server-netty:$ktor_version")
    implementation("io.ktor:ktor-server-content-negotiation:$ktor_version")
    implementation("io.ktor:ktor-serialization-kotlinx-json:$ktor_version")
    implementation("io.ktor:ktor-server-cors:$ktor_version")
}
```

### 创建服务器

```kotlin
import io.ktor.server.application.*
import io.ktor.server.engine.*
import io.ktor.server.netty.*
import io.ktor.server.response.*
import io.ktor.server.routing.*

fun main() {
    embeddedServer(Netty, port = 8080) {
        configureRouting()
    }.start(wait = true)
}

fun Application.configureRouting() {
    routing {
        get("/") {
            call.respondText("Hello, Ktor!")
        }
    }
}
```

## 核心概念

### 路由 (Routing)

路由是 Ktor 的核心插件，用于处理传入的 HTTP 请求。

#### 安装路由

```kotlin
import io.ktor.server.routing.*

fun Application.module() {
    // 方式 1: 显式安装
    install(Routing) {
        get("/hello") {
            call.respondText("Hello!")
        }
    }
    
    // 方式 2: 使用便捷函数 (推荐)
    routing {
        get("/world") {
            call.respondText("World!")
        }
    }
}
```

#### 定义路由处理器

```kotlin
routing {
    // HTTP 动词 + 路径
    get("/users") { /* ... */ }
    post("/users") { /* ... */ }
    put("/users/{id}") { /* ... */ }
    delete("/users/{id}") { /* ... */ }
    
    // 所有动词
    route("/items") {
        method(HttpMethod.Get) { /* ... */ }
        method(HttpMethod.Post) { /* ... */ }
    }
}
```

#### 路径模式

```kotlin
routing {
    // 固定路径
    get("/hello") { /* ... */ }
    
    // 多段路径
    get("/order/shipment") { /* ... */ }
    
    // 路径参数
    get("/user/{login}") {
        val login = call.parameters["login"]
        call.respondText("User: $login")
    }
    
    // 可选参数 (必须在末尾)
    get("/user/{login?}") { /* ... */ }
    
    // 通配符 (匹配单个路径段)
    get("/user/*") { /* ... */ }
    
    // 尾卡 (匹配剩余所有路径)
    get("/user/{...}") { /* ... */ }
    
    // 带尾卡的参数
    get("/user/{param...}") {
        val segments = call.parameters.getAll("param")
        // segments = ["john", "settings"]
    }
    
    // 正则表达式
    get(Regex("/.+/hello")) {
        // 匹配任何以 /hello 结尾的路径
    }
    
    // 带命名组的正则
    get(Regex("/user/(?<id>\\d+)/hello")) {
        val id = call.parameters["id"]
        call.respondText("User ID: $id")
    }
}
```

#### 嵌套路由

```kotlin
routing {
    route("/order") {
        get("/shipment") { /* /order/shipment */ }
        post("/shipment") { /* /order/shipment */ }
    }
    
    // 或更简洁的写法
    route("/order") {
        route("/shipment") {
            get { /* ... */ }
            post { /* ... */ }
        }
    }
}
```

#### 路由扩展函数

```kotlin
// 定义扩展函数
fun Route.userRoutes() {
    route("/users") {
        get { /* 获取所有用户 */ }
        post { /* 创建用户 */ }
        get("/{id}") { /* 获取单个用户 */ }
    }
}

// 使用
fun Application.module() {
    routing {
        userRoutes()
    }
}
```

### 插件 (Plugins)

插件用于在请求/响应管道中添加通用功能。

#### 安装插件

```kotlin
import io.ktor.server.plugins.*
import io.ktor.server.plugins.cors.*
import io.ktor.server.plugins.contentnegotiation.*
import io.ktor.serialization.kotlinx.json.*

fun Application.module() {
    // 全局安装
    install(CORS) {
        anyHost()
        allowMethod(HttpMethod.Get)
        allowMethod(HttpMethod.Post)
    }
    
    install(ContentNegotiation) {
        json(Json {
            prettyPrint = true
            isLenient = true
        })
    }
    
    // 安装到特定路由
    routing {
        route("/api") {
            install(CachingHeaders) {
                options { _, _ ->
                    CachingOptions(CacheControl.MaxAge(maxAgeSeconds = 3600))
                }
            }
            get("/data") {
                call.respondText("Cached data")
            }
        }
    }
}
```

#### 常用插件

**内容协商 (ContentNegotiation)**:

```kotlin
install(ContentNegotiation) {
    json(Json {
        prettyPrint = true
        isLenient = true
        ignoreUnknownKeys = true
    })
    xml {
        // XML 序列化配置
    }
}
```

**CORS (跨域资源共享)**:

```kotlin
install(CORS) {
    // 允许特定主机
    allowHost("example.com", schemes = listOf("https"))
    
    // 允许所有主机 (开发环境)
    anyHost()
    
    // 允许的方法
    allowMethod(HttpMethod.Get)
    allowMethod(HttpMethod.Post)
    allowMethod(HttpMethod.Put)
    allowMethod(HttpMethod.Delete)
    
    // 允许的头部
    allowHeader(HttpHeaders.ContentType)
    allowHeader(HttpHeaders.Authorization)
    
    // 允许自定义头部
    allowHeadersPrefixed("custom-")
    
    // 暴露头部给客户端
    exposeHeader(HttpHeaders.ContentDisposition)
    
    // 允许凭证
    allowCredentials = true
    
    // 预检请求缓存时间
    maxAgeInSeconds = 24 * 60 * 60
}
```

**认证 (Authentication)**:

```kotlin
install(Authentication) {
    // Basic 认证
    basic("auth-basic") {
        realm = "Access to the API"
        validate { credentials ->
            if (credentials.name == "admin" && credentials.password == "secret") {
                UserIdPrincipal(credentials.name)
            } else {
                null
            }
        }
    }
    
    // Bearer 认证
    bearer("auth-bearer") {
        realm = "Access to the API"
        authenticate { token ->
            // 验证 token
            if (token.isValid()) {
                UserIdPrincipal(token.userId)
            } else {
                null
            }
        }
    }
}
```

### ApplicationCall

`ApplicationCall` 封装了 HTTP 请求和响应。

```kotlin
get("/example") {
    // 访问请求
    val request = call.request
    val method = request.httpMethod
    val uri = request.uri
    val headers = request.headers
    
    // 访问响应
    val response = call.response
    response.headers.append("X-Custom-Header", "value")
    
    // 发送响应
    call.respondText("Hello")
    call.respondJson(mapOf("key" to "value"))
    call.respondFile(File("path/to/file"))
}
```

## 请求处理

### 获取请求参数

```kotlin
get("/user/{id}") {
    // 路径参数
    val id = call.parameters["id"]
    
    // 查询参数
    val name = call.request.queryParameters["name"]
    val age = call.request.queryParameters["age"]?.toInt()
    
    // 多个查询参数
    val tags = call.request.queryParameters.getAll("tags")
    
    call.respondText("User $id: $name, $age years old")
}
```

### 读取请求体

```kotlin
import io.ktor.server.request.*

post("/users") {
    // 读取为 String
    val text = call.receiveText()
    
    // 读取为数据类
    val user = call.receive<User>()
    
    // 读取为 Map
    val map = call.receive<Map<String, Any>>()
    
    // 读取为表单数据
    val formData = call.receiveParameters()
    val username = formData["username"]
    
    // 读取 multipart 表单
    val multipart = call.receiveMultipart()
    multipart.forEachPart { part ->
        when (part) {
            is PartData.FormItem -> {
                println("Form field: ${part.name} = ${part.value}")
            }
            is PartData.FileItem -> {
                val fileContent = part.streamProvider().readBytes()
                // 处理文件
            }
        }
        part.dispose()
    }
}
```

### 发送响应

```kotlin
import io.ktor.server.response.*

get("/example") {
    // 文本响应
    call.respondText("Hello, World!")
    call.respondText("HTML content", ContentType.Text.Html)
    
    // JSON 响应
    call.respondJson(mapOf("key" to "value"))
    call.respond(User("John", 25))
    
    // 文件响应
    call.respondFile(File("path/to/file.pdf"))
    
    // 下载文件
    call.response.header(
        HttpHeaders.ContentDisposition,
        ContentDisposition.Attachment.withParameter(
            ContentDisposition.Parameters.FileName,
            "file.pdf"
        ).toString()
    )
    call.respondFile(File("path/to/file.pdf"))
    
    // 重定向
    call.respondRedirect("/new-location")
    call.respondRedirect("/permanent", true) // 永久重定向
    
    // 自定义状态码
    call.respond(HttpStatusCode.NotFound, "Not found")
    
    // 流式响应
    call.respondOutputStream {
        write("Chunk 1".toByteArray())
        flush()
        write("Chunk 2".toByteArray())
    }
}
```

## 高级特性

### WebSocket

```kotlin
import io.ktor.server.websocket.*
import io.ktor.websocket.*

install(WebSockets)

routing {
    webSocket("/ws") {
        for (frame in incoming) {
            when (frame) {
                is Frame.Text -> {
                    val text = frame.readText()
                    outgoing.send(Frame.Text("Echo: $text"))
                }
                is Frame.Binary -> {
                    // 处理二进制数据
                }
                is Frame.Close -> {
                    // 处理关闭
                }
            }
        }
    }
}
```

### Server-Sent Events (SSE)

```kotlin
import io.ktor.server.sse.*

install(SSE)

routing {
    sse("/events") {
        send("event", "data", "event-id")
        
        // 定时发送
        repeat(10) { i ->
            delay(1000)
            send("message", "Event $i")
        }
    }
}
```

### 会话管理

```kotlin
import io.ktor.server.sessions.*

data class UserSession(val userId: String, val username: String)

install(Sessions) {
    cookie<UserSession>("user_session") {
        cookie.path = "/"
        cookie.maxAgeInSeconds = 3600
        cookie.secure = true
        transform(SessionTransformer.encrypted("secret-key"))
    }
}

routing {
    get("/login") {
        val session = UserSession("123", "john")
        call.sessions.set(session)
        call.respondText("Logged in")
    }
    
    get("/profile") {
        val session = call.sessions.get<UserSession>()
        if (session != null) {
            call.respondText("Welcome, ${session.username}")
        } else {
            call.respondRedirect("/login")
        }
    }
    
    get("/logout") {
        call.sessions.clear<UserSession>()
        call.respondText("Logged out")
    }
}
```

### 静态文件

```kotlin
import io.ktor.server.static.*

routing {
    // 提供静态文件
    staticFiles("/static", File("path/to/static"))
    
    // 提供资源
    staticResources("/assets", "assets") {
        default("index.html")
        extensions("html", "css", "js")
    }
    
    // 单文件
    staticFile("/favicon.ico", File("path/to/favicon.ico"))
}
```

### 模板引擎

**Thymeleaf**:

```kotlin
import io.ktor.server.thymeleaf.*

install(Thymeleaf) {
    setTemplateResolver {
        FileTemplateResolver().apply {
            prefix = "templates/"
            suffix = ".html"
            templateMode = TemplateMode.HTML
        }
    }
}

routing {
    get("/hello") {
        call.respond(
            "hello",
            mapOf("name" to "John")
        )
    }
}
```

**Kotlinx.html**:

```kotlin
import io.ktor.server.html.*

routing {
    get("/page") {
        call.respondHtml {
            head {
                title { +"My Page" }
            }
            body {
                h1 { +"Hello, World!" }
            }
        }
    }
}
```

## 最佳实践

### 项目结构

```
src/
├── main/
│   ├── kotlin/
│   │   └── com/
│   │       └── example/
│   │           ├── Application.kt      # 主入口
│   │           ├── plugins/            # 插件配置
│   │           │   ├── ContentNegotiation.kt
│   │           │   ├── CORS.kt
│   │           │   └── Authentication.kt
│   │           ├── routes/             # 路由定义
│   │           │   ├── UserRoutes.kt
│   │           │   └── ProductRoutes.kt
│   │           ├── models/             # 数据模型
│   │           │   └── User.kt
│   │           ├── services/           # 业务逻辑
│   │           │   └── UserService.kt
│   │           └── utils/              # 工具类
│   └── resources/
│       ├── application.conf            # 配置文件
│       └── templates/                  # 模板文件
```

### 模块化配置

```kotlin
// Application.kt
fun main() {
    embeddedServer(Netty, port = 8080) {
        configurePlugins()
        configureRouting()
    }.start(wait = true)
}

// Plugins.kt
fun Application.configurePlugins() {
    install(ContentNegotiation) { /* ... */ }
    install(CORS) { /* ... */ }
    install(Authentication) { /* ... */ }
}

// Routing.kt
fun Application.configureRouting() {
    routing {
        authenticate {
            userRoutes()
            productRoutes()
        }
    }
}
```

### 错误处理

```kotlin
import io.ktor.server.plugins.statuspages.*
import io.ktor.server.plugins.callloging.*

install(StatusPages) {
    exception<Throwable> { call, cause ->
        call.respondText(
            text = "Internal Server Error",
            status = HttpStatusCode.InternalServerError
        )
    }
    
    status(HttpStatusCode.NotFound) { call ->
        call.respondText("404 Not Found")
    }
}

install(CallLogging) {
    level = Level.INFO
    filter { call -> call.request.path().startsWith("/api") }
}
```

### 依赖注入

```kotlin
// 使用 Koin
import org.koin.koin.inject
import org.koin.dsl.module

val appModule = module {
    single { UserService() }
    single { Database() }
}

fun Application.module() {
    startKoin {
        modules(appModule)
    }
    
    val userService = inject<UserService>()
    
    routing {
        get("/users") {
            val users = userService.getAll()
            call.respond(users)
        }
    }
}
```

### 测试

```kotlin
import io.ktor.server.testing.*

class ApplicationTest {
    @Test
    fun testRoot() = testApplication {
        application {
            configureRouting()
        }
        
        client.get("/").apply {
            assertEquals(HttpStatusCode.OK, status)
            assertEquals("Hello, Ktor!", bodyAsText())
        }
    }
}
```

## 部署

### 构建 Fat JAR

```kotlin
// build.gradle.kts
plugins {
    application {
        mainClass.set("com.example.ApplicationKt")
    }
}

tasks.jar {
    manifest {
        attributes("Main-Class" to "com.example.ApplicationKt")
    }
    from(configurations.runtimeClasspath.get().map { 
        if (it.isDirectory) it else zipTree(it) 
    })
    duplicatesStrategy = DuplicatesStrategy.EXCLUDE
}
```

### Docker 部署

```dockerfile
FROM eclipse-temurin:17-jdk-alpine
COPY build/libs/app.jar /app.jar
EXPOSE 8080
ENTRYPOINT ["java", "-jar", "/app.jar"]
```

### 环境变量配置

```kotlin
val port = System.getenv("PORT")?.toInt() ?: 8080
val host = System.getenv("HOST") ?: "0.0.0.0"

embeddedServer(Netty, port = port, host = host) {
    // ...
}
```

## 参考资源

- [Ktor 官方文档](https://ktor.io/docs/)
- [Ktor GitHub](https://github.com/ktorio/ktor)
- [Ktor 示例项目](https://github.com/ktorio/ktor-samples)
- [Ktor 插件注册表](https://ktor.io/docs/plugins.html)

## 更新日志

- **2026-03-07**: 初始版本，整理 Ktor 服务端开发文档
