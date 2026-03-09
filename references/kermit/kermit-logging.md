# Kermit Kotlin Multiplatform 日志库参考指南

> **版本**: 2.0.4+  
> **更新日期**: 2026-03-07  
> **官方文档**: https://kermit.touchlab.co/docs/  
> **GitHub**: https://github.com/touchlab/Kermit

---

## 目录

- [概述](#概述)
- [快速开始](#快速开始)
- [核心概念](#核心概念)
- [日志级别](#日志级别)
- [使用方法](#使用方法)
- [标签 (Tag) 管理](#标签-tag-管理)
- [配置指南](#配置指南)
- [自定义 LogWriter](#自定义-logwriter)
- [多平台配置](#多平台配置)
- [非 Kotlin 环境调用](#非-kotlin-环境调用)
- [最佳实践](#最佳实践)
- [与项目集成](#与项目集成)

---

## 概述

Kermit 是一个 Kotlin Multiplatform 日志库，主要功能是允许 Kotlin 代码中的日志语句写入到可组合的日志输出中。

### 主要特性

- **跨平台支持**: 一次配置，多平台使用 (Android、iOS、JS、JVM、Native)
- **平台默认输出**: 
  - Android: Logcat
  - iOS: OSLog
  - JS: console
  - JVM: System.out
- **易于扩展**: 可轻松添加自定义 LogWriter
- **Kotlin 友好**: 支持 lambda 语法和默认参数
- **可组合性**: 支持多个 LogWriter 组合使用
- **崩溃日志**: 支持 Crashlytics 和 Bugsnag 集成

### 适用场景

- Kotlin Multiplatform 项目日志记录
- 跨平台业务逻辑调试
- 生产环境日志收集
- 崩溃信息追踪
- 多环境日志配置

---

## 快速开始

### 1. 添加依赖

在 Kotlin Multiplatform 模块的 `commonMain` 源集中添加 Kermit 依赖:

```kotlin
// gradle/libs.versions.toml
[versions]
kermit = "2.0.4"  // 使用最新版本

[libraries]
kermit = { module = "co.touchlab:kermit", version.ref = "kermit" }
```

```kotlin
// shared/build.gradle.kts 或 common/build.gradle.kts
kotlin {
    sourceSets {
        commonMain {
            dependencies {
                implementation(libs.kermit)
            }
        }
    }
}
```

### 2. 开始记录日志

```kotlin
import co.touchlab.kermit.Logger

// 最简单的日志记录
Logger.i { "Hello World" }

// 带异常的日志
try {
    somethingRisky()
} catch (t: Throwable) {
    Logger.w(t) { "That could've gone better" }
}
```

### 3. 默认配置

Kermit 默认配置非常简单，开箱即用:
- 无需额外配置即可使用
- 自动使用平台特定的日志输出
- 适合开发环境

---

## 核心概念

Kermit 的三个基本组件：**Logger**、**LogWriter**、**Severity**。

### Logger (日志记录器)

Logger 是主要的日志记录组件，负责:
- 接收代码中的日志调用
- 将日志消息分派到 LogWriter 实例
- 提供不同严重级别的日志方法

```kotlin
// 基本使用
Logger.i { "Hello World" }

// 带异常
Logger.e(exception) { "Something failed" }

// 带标签
Logger.w("MyTag") { "Warning message" }
```

### LogWriter (日志写入器)

LogWriter 负责将日志消息发送到不同的输出目标。

**内置 LogWriter**:
- **CommonWriter**: 通用写入器
- **Platform-specific**: 平台特定写入器
  - Android: LogcatWriter
  - iOS: OSLogWriter
  - JS: ConsoleWriter
  - JVM: SystemOutWriter
- **第三方集成**:
  - CrashlyticsWriter
  - BugsnagWriter

```kotlin
// 添加自定义 LogWriter
val logger = Logger(
    tag = "MyApp",
    logWriter = CustomLogWriter()
)
```

### Severity (严重级别)

严重级别控制日志的过滤和输出。

**级别排序** (从低到高):
1. `v()` - Verbose (详细)
2. `d()` - Debug (调试)
3. `i()` - Info (信息)
4. `w()` - Warning (警告)
5. `e()` - Error (错误)
6. `a()` - Assert (断言)

```kotlin
Logger.v { "Verbose message" }  // 最详细
Logger.d { "Debug message" }
Logger.i { "Info message" }
Logger.w { "Warning message" }
Logger.e { "Error message" }
Logger.a { "Assert message" }   // 最严重
```

---

## 日志级别

### 级别详解

#### Verbose (v) - 详细

最低级别，用于详细的调试信息。

```kotlin
Logger.v { "Entering function foo()" }
Logger.v { "Variable x = $x, y = $y" }
```

**使用场景**:
- 函数入口/出口跟踪
- 变量值详细记录
- 执行流程跟踪

#### Debug (d) - 调试

用于调试信息，比 Verbose 级别稍高。

```kotlin
Logger.d { "Processing item: $item" }
Logger.d { "Network response: $response" }
```

**使用场景**:
- 调试过程中的临时日志
- 数据处理的中间状态
- 开发阶段的详细信息

#### Info (i) - 信息

用于一般信息性消息。

```kotlin
Logger.i { "Application started" }
Logger.i { "User logged in: ${user.name}" }
```

**使用场景**:
- 应用生命周期事件
- 用户操作记录
- 系统状态变更

#### Warning (w) - 警告

用于警告信息，表示可能的问题。

```kotlin
Logger.w { "Cache miss, fetching from network" }
Logger.w(exception) { "Retry attempt $retryCount" }
```

**使用场景**:
- 非致命错误
- 降级处理
- 性能问题
- 异常情况但可恢复

#### Error (e) - 错误

用于错误信息，表示操作失败。

```kotlin
Logger.e(exception) { "Failed to load data" }
Logger.e("Database connection failed", exception)
```

**使用场景**:
- 操作失败
- 系统错误
- 数据损坏
- 服务不可用

#### Assert (a) - 断言

最高级别，用于严重错误。

```kotlin
Logger.a { "Invariant violated: x should never be null" }
```

**使用场景**:
- 不变量被破坏
- 严重逻辑错误
- 不应该发生的情况

### 级别过滤

可以根据环境配置日志级别过滤:

```kotlin
import co.touchlab.kermit.Severity

// 只记录 Warning 及以上级别
val logger = Logger(minSeverity = Severity.Warn)

logger.d { "This won't be logged" }  // 被过滤
logger.w { "This will be logged" }   // 记录
logger.e { "This will be logged" }   // 记录
```

---

## 使用方法

### 基本日志方法

每个严重级别都有两种方法签名:

```kotlin
// 1. 使用 lambda (推荐)
fun i(message: () -> String)
fun i(throwable: Throwable? = null, tag: String = this.tag, message: () -> String)

// 2. 直接使用字符串
fun i(messageString: String, throwable: Throwable? = null, tag: String = this.tag)
```

### Lambda vs 字符串

#### Lambda 方式 (推荐)

```kotlin
Logger.i { "Hello World" }
Logger.w { "Warning: $someExpensiveOperation()" }
```

**优点**:
- 延迟求值 (只有在日志会被写入时才计算)
- 避免不必要的字符串创建
- 性能更好
- Kotlin 风格

#### 字符串方式

```kotlin
Logger.i("Hello World")
Logger.w("Warning: ${someExpensiveOperation()}")
```

**缺点**:
- 立即求值 (即使日志被过滤)
- 可能创建不必要的字符串
- 性能稍差

**适用场景**:
- 简单日志
- 与非 Kotlin 代码互操作
- 个人偏好

### 带异常的日志

```kotlin
try {
    riskyOperation()
} catch (e: Exception) {
    // 方式 1: lambda + 异常
    Logger.e(e) { "Operation failed" }
    
    // 方式 2: 字符串 + 异常
    Logger.e("Operation failed", e)
}
```

### 带标签的日志

```kotlin
// 方式 1: 指定标签
Logger.w("MyTag") { "Warning message" }

// 方式 2: 创建带标签的 Logger
class MyViewModel : ViewModel {
    private val log = Logger.withTag("MyViewModel")
    
    fun loadData() {
        log.i { "Loading data..." }
    }
}
```

### 完整示例

```kotlin
import co.touchlab.kermit.Logger

class UserRepository(
    private val api: UserApi,
    private val database: UserDatabase
) {
    private val log = Logger.withTag("UserRepository")
    
    suspend fun loadUser(userId: String): User {
        log.d { "Loading user: $userId" }
        
        return try {
            // 尝试从数据库加载
            val cached = database.getUser(userId)
            if (cached != null) {
                log.i { "Loaded from cache: $userId" }
                return cached
            }
            
            // 从 API 加载
            log.d { "Fetching from API: $userId" }
            val user = api.getUser(userId)
            
            // 缓存到数据库
            database.saveUser(user)
            log.i { "Cached user: $userId" }
            
            user
        } catch (e: Exception) {
            log.e(e) { "Failed to load user: $userId" }
            throw e
        }
    }
    
    suspend fun updateUser(user: User) {
        log.d { "Updating user: ${user.id}" }
        
        try {
            database.updateUser(user)
            log.i { "User updated: ${user.id}" }
        } catch (e: Exception) {
            log.w(e) { "Update failed, will retry" }
            // 降级处理
            scheduleRetry(user)
        }
    }
}
```

---

## 标签 (Tag) 管理

### 标签的作用

标签用于标识日志来源，便于过滤和搜索。

**平台差异**:
- **Android**: Logcat 原生支持标签，非常重要
- **iOS**: OSLog 不常用标签，但可自定义显示
- **JS**: console 不原生支持标签
- **JVM**: System.out 不原生支持标签

### 标签设置方式

#### 1. 每次调用指定标签

```kotlin
Logger.d("MyTag") { "Message" }
Logger.i("MyTag") { "Info message" }
```

**适用场景**:
- 临时日志
- 少量日志调用

#### 2. 创建带标签的 Logger 实例

```kotlin
class MyViewModel : ViewModel {
    private val log = Logger.withTag("MyViewModel")
    
    fun onCreate() {
        log.i { "ViewModel created" }
    }
    
    fun onDestroy() {
        log.i { "ViewModel destroyed" }
    }
}
```

**适用场景**:
- 类级别的日志
- 多个日志调用
- 推荐模式

#### 3. 使用类名作为标签

```kotlin
// 扩展函数
inline fun <reified T> T.logger(): Logger {
    return Logger.withTag(T::class.java.simpleName)
}

// 使用
class UserRepository {
    private val log = logger()
    
    fun loadUser() {
        log.i { "Loading user..." }  // 标签：UserRepository
    }
}
```

**优点**:
- 自动使用类名
- 无需手动维护标签
- 易于识别日志来源

#### 4. 层级标签

```kotlin
// 使用点分隔表示层级
val log = Logger.withTag("App.Repository.User")

log.i { "Loading user" }  // 输出：App.Repository.User: Loading user
```

**适用场景**:
- 大型项目
- 模块化架构
- 便于分类过滤

### 标签最佳实践

```kotlin
// ✅ 推荐：使用类名
class UserService {
    private val log = Logger.withTag("UserService")
}

// ✅ 推荐：使用扩展函数
inline fun <reified T> T.logger(): Logger = 
    Logger.withTag(T::class.java.simpleName)

class DataService {
    private val log = logger()
}

// ❌ 不推荐：硬编码字符串
Logger.i("SomeRandomTag") { "Message" }

// ❌ 不推荐：过长的标签
Logger.withTag("com.example.myapp.data.repository.user.UserRepositoryImpl")
```

---

## 配置指南

### 基础配置

Kermit 默认配置开箱即用，但可以根据需求进行配置。

#### 最小配置

```kotlin
// 无需配置，直接使用
Logger.i { "Hello World" }
```

#### 配置日志级别

```kotlin
import co.touchlab.kermit.Severity

// 只记录 Warning 及以上
val logger = Logger(minSeverity = Severity.Warn)

// 记录所有级别
val verboseLogger = Logger(minSeverity = Severity.Verbose)
```

#### 配置 LogWriter

```kotlin
import co.touchlab.kermit.Logger
import co.touchlab.kermit.LogWriter

// 使用自定义 LogWriter
val customWriter = CustomLogWriter()
val logger = Logger(logWriter = customWriter)

// 使用多个 LogWriter
val writers = listOf(
    CommonWriter(),
    CrashlyticsWriter(),
    CustomFileWriter()
)
val logger = Logger(logWriters = writers)
```

### 环境配置

根据不同环境配置日志行为。

#### 开发环境

```kotlin
expect object PlatformLogger {
    fun createLogger(): Logger
}

// iOS 实际实现
actual object PlatformLogger {
    actual fun createLogger(): Logger {
        return Logger(minSeverity = Severity.Verbose)
    }
}

// Android 实际实现
actual object PlatformLogger {
    actual fun createLogger(): Logger {
        return Logger(minSeverity = Severity.Debug)
    }
}
```

#### 生产环境

```kotlin
// commonMain
expect object PlatformLogger {
    fun createLogger(): Logger
}

// 生产环境配置
actual object PlatformLogger {
    actual fun createLogger(): Logger {
        return Logger(
            minSeverity = Severity.Warn,  // 只记录警告和错误
            logWriters = listOf(
                CommonWriter(),            // 平台默认输出
                CrashlyticsWriter()        // 崩溃报告
            )
        )
    }
}
```

### 消息格式化

使用 `MessageStringFormatter` 配置日志消息格式。

```kotlin
import co.touchlab.kermit.Logger
import co.touchlab.kermit.MessageStringFormatter

class CustomFormatter : MessageStringFormatter {
    override fun formatMessage(
        severity: Severity,
        tag: String,
        message: String,
        throwable: Throwable?
    ): String {
        val timestamp = SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.getDefault())
            .format(Date())
        
        return "[$timestamp] [$severity] [$tag] $message"
    }
}

// 使用自定义格式化器
val logger = Logger(
    messageStringFormatter = CustomFormatter()
)

logger.i { "Hello World" }
// 输出：[2026-03-07 10:30:00] [Info] [MyTag] Hello World
```

---

## 自定义 LogWriter

### 创建自定义 LogWriter

```kotlin
import co.touchlab.kermit.LogWriter
import co.touchlab.kermit.Severity
import co.touchlab.kermit.MessageStringFormatter

class FileLogWriter(
    private val logFile: File,
    private val formatter: MessageStringFormatter = DefaultFormatter()
) : LogWriter() {
    
    override fun log(
        severity: Severity,
        message: String,
        tag: String,
        throwable: Throwable?
    ) {
        try {
            val formattedMessage = formatter.formatMessage(
                severity = severity,
                tag = tag,
                message = message,
                throwable = throwable
            )
            
            // 写入文件
            logFile.appendText(formattedMessage + "\n")
        } catch (e: Exception) {
            // 处理写入失败
            e.printStackTrace()
        }
    }
}
```

### 使用自定义 LogWriter

```kotlin
// 创建日志文件
val logFile = File(context.filesDir, "app.log")

// 创建 LogWriter
val fileWriter = FileLogWriter(logFile)

// 创建 Logger
val logger = Logger(
    tag = "MyApp",
    logWriters = listOf(
        CommonWriter(),  // 平台默认
        fileWriter       // 文件输出
    )
)

// 使用
logger.i { "Application started" }
```

### 网络日志写入器

```kotlin
import co.touchlab.kermit.LogWriter
import co.touchlab.kermit.Severity
import io.ktor.client.*
import io.ktor.client.request.*

class NetworkLogWriter(
    private val client: HttpClient,
    private val logEndpoint: String
) : LogWriter() {
    
    override fun log(
        severity: Severity,
        message: String,
        tag: String,
        throwable: Throwable?
    ) {
        // 只发送 Error 及以上级别
        if (severity.ordinal < Severity.Error.ordinal) {
            return
        }
        
        // 异步发送日志
        CoroutineScope(Dispatchers.IO).launch {
            try {
                client.post(logEndpoint) {
                    setBody(
                        mapOf(
                            "severity" to severity.name,
                            "tag" to tag,
                            "message" to message,
                            "timestamp" to System.currentTimeMillis(),
                            "error" to throwable?.stackTraceToString()
                        )
                    )
                }
            } catch (e: Exception) {
                // 网络失败，降级到本地
                println("Failed to send log: ${e.message}")
            }
        }
    }
}
```

### Crashlytics 集成

```kotlin
import co.touchlab.kermit.LogWriter
import co.touchlab.kermit.Severity
import com.google.firebase.crashlytics.FirebaseCrashlytics

class CrashlyticsLogWriter : LogWriter() {
    private val crashlytics = FirebaseCrashlytics.getInstance()
    
    override fun log(
        severity: Severity,
        message: String,
        tag: String,
        throwable: Throwable?
    ) {
        // 记录日志到 Crashlytics
        crashlytics.log("[$tag] $message")
        
        // 记录异常
        throwable?.let {
            crashlytics.recordException(it)
        }
    }
}
```

---

## 多平台配置

### 项目结构

```
shared/
├── src/
│   ├── commonMain/
│   │   └── kotlin/
│   │       └── com/example/
│   │           └── Logger.kt          # 通用日志配置
│   ├── androidMain/
│   │   └── kotlin/
│   │       └── com/example/
│   │           └── Logger.android.kt  # Android 实现
│   ├── iosMain/
│   │   └── kotlin/
│   │       └── com/example/
│   │           └── Logger.ios.kt      # iOS 实现
│   └── jvmMain/
│       └── kotlin/
│           └── com/example/
│               └── Logger.jvm.kt      # JVM 实现
```

### commonMain

```kotlin
// commonMain/kotlin/com/example/Logger.kt
package com.example

import co.touchlab.kermit.Logger
import co.touchlab.kermit.Severity

expect object PlatformLogger {
    fun createLogger(tag: String = ""): Logger
}

// 便捷函数
inline fun <reified T> T.logger(): Logger {
    return PlatformLogger.createLogger(T::class.java.simpleName)
}
```

### Android 实现

```kotlin
// androidMain/kotlin/com/example/Logger.android.kt
package com.example

import co.touchlab.kermit.Logger
import co.touchlab.kermit.Severity

actual object PlatformLogger {
    actual fun createLogger(tag: String): Logger {
        return Logger(
            tag = tag,
            minSeverity = if (BuildConfig.DEBUG) Severity.Debug else Severity.Warn,
            logWriters = listOf(
                AndroidLogcatWriter()  // Android Logcat
            )
        )
    }
}
```

### iOS 实现

```kotlin
// iosMain/kotlin/com/example/Logger.ios.kt
package com.example

import co.touchlab.kermit.Logger
import co.touchlab.kermit.Severity

actual object PlatformLogger {
    actual fun createLogger(tag: String): Logger {
        return Logger(
            tag = tag,
            minSeverity = Severity.Debug,
            logWriters = listOf(
                OSLogWriter()  // iOS OSLog
            )
        )
    }
}
```

### JS 实现

```kotlin
// jsMain/kotlin/com/example/Logger.kt
package com.example

import co.touchlab.kermit.Logger
import co.touchlab.kermit.Severity

actual object PlatformLogger {
    actual fun createLogger(tag: String): Logger {
        return Logger(
            tag = tag,
            minSeverity = if (isDevelopment()) Severity.Debug else Severity.Warn,
            logWriters = listOf(
                ConsoleWriter()  // Browser console
            )
        )
    }
    
    private fun isDevelopment(): Boolean {
        return js("process.env.NODE_ENV === 'development'")
    }
}
```

### JVM 实现

```kotlin
// jvmMain/kotlin/com/example/Logger.kt
package com.example

import co.touchlab.kermit.Logger
import co.touchlab.kermit.Severity
import java.util.logging.ConsoleHandler
import java.util.logging.Level
import java.util.logging.Logger as JdkLogger

actual object PlatformLogger {
    actual fun createLogger(tag: String): Logger {
        return Logger(
            tag = tag,
            minSeverity = Severity.Info,
            logWriters = listOf(
                SystemOutWriter()  // JVM System.out
            )
        )
    }
}
```

---

## 非 Kotlin 环境调用

### 问题

Kermit 的 API 使用 Kotlin 特性 (默认参数、lambda),在非 Kotlin 环境 (如 Swift、JS) 中调用不便。

### 解决方案：kermit-simple

添加 `kermit-simple` 模块:

```kotlin
// gradle/libs.versions.toml
[libraries]
kermit-simple = { module = "co.touchlab:kermit-simple", version.ref = "kermit" }
```

```kotlin
// build.gradle.kts
commonMain {
    dependencies {
        implementation(libs.kermit)
        api(libs.kermit.simple)  // 导出给非 Kotlin 环境
    }
}
```

### 从 Swift 调用

```swift
// iOS Swift 代码
import shared

// 简单日志
Logger.shared.i("Hello from Swift")

// 带标签
Logger.shared.iWithTag("MyTag", message: "Hello World")

// 带异常
do {
    try riskyOperation()
} catch {
    Logger.shared.eWithThrowable(error, tag: "MyTag", message: "Failed")
}
```

### 从 JavaScript 调用

```javascript
// JS 代码
import { Logger } from 'shared'

// 简单日志
Logger.i("Hello from JS")

// 带标签
Logger.iWithTag("MyTag", "Hello World")
```

---

## 最佳实践

### 1. 使用 Logger 实例

```kotlin
// ✅ 推荐：创建类级别的 Logger
class UserRepository {
    private val log = Logger.withTag("UserRepository")
    
    fun loadUser() {
        log.i { "Loading user" }
    }
}

// ❌ 不推荐：每次都使用全局 Logger
class UserRepository {
    fun loadUser() {
        Logger.i("UserRepository") { "Loading user" }
    }
}
```

### 2. 使用 Lambda 语法

```kotlin
// ✅ 推荐：延迟求值
Logger.i { "Expensive operation: ${expensiveOperation()}" }

// ❌ 不推荐：立即求值
Logger.i("Expensive operation: ${expensiveOperation()}")
```

### 3. 合理的日志级别

```kotlin
// ✅ 推荐：根据重要性选择级别
Logger.d { "Entering function" }           // 调试信息
Logger.i { "User logged in" }              // 一般信息
Logger.w { "Cache miss" }                  // 警告
Logger.e(e) { "Failed to load data" }      // 错误

// ❌ 不推荐：所有日志都用同一级别
Logger.i { "Everything is info" }
```

### 4. 包含上下文信息

```kotlin
// ✅ 推荐：包含足够的上下文
Logger.e(e) { "Failed to load user: userId=$userId, retry=$retryCount" }

// ❌ 不推荐：信息不足
Logger.e(e) { "Failed" }
```

### 5. 避免敏感信息

```kotlin
// ✅ 推荐：脱敏处理
Logger.i { "User logged in: id=${user.id}" }

// ❌ 不推荐：记录敏感信息
Logger.i { "User password: ${user.password}" }
Logger.i { "API key: $apiKey" }
```

### 6. 生产环境配置

```kotlin
// ✅ 推荐：根据环境配置
actual object PlatformLogger {
    actual fun createLogger(tag: String): Logger {
        return Logger(
            minSeverity = if (isProduction()) Severity.Warn else Severity.Debug,
            logWriters = buildList {
                add(CommonWriter())
                if (isProduction()) {
                    add(CrashlyticsWriter())
                }
            }
        )
    }
}
```

### 7. 结构化日志

```kotlin
// ✅ 推荐：结构化日志便于解析
data class LogEntry(
    val timestamp: Long,
    val severity: String,
    val tag: String,
    val message: String,
    val userId: String?,
    val action: String?
)

class StructuredLogWriter : LogWriter() {
    override fun log(
        severity: Severity,
        message: String,
        tag: String,
        throwable: Throwable?
    ) {
        val entry = LogEntry(
            timestamp = System.currentTimeMillis(),
            severity = severity.name,
            tag = tag,
            message = message,
            userId = getCurrentUserId(),
            action = getCurrentAction()
        )
        
        // 发送到日志服务
        logService.send(entry)
    }
}
```

### 8. 性能优化

```kotlin
// ✅ 推荐：避免频繁创建 Logger
class MyViewModel : ViewModel {
    private val log = Logger.withTag("MyViewModel")  // 只创建一次
    
    fun action1() {
        log.i { "Action 1" }
    }
    
    fun action2() {
        log.i { "Action 2" }
    }
}

// ❌ 不推荐：每次都创建
class MyViewModel : ViewModel {
    fun action1() {
        Logger.withTag("MyViewModel").i { "Action 1" }
    }
}
```

### 9. 异步日志

```kotlin
// 对于网络日志等耗时操作，使用异步
class AsyncLogWriter(
    private val dispatcher: CoroutineDispatcher = Dispatchers.IO
) : LogWriter() {
    private val scope = CoroutineScope(dispatcher + SupervisorJob())
    
    override fun log(
        severity: Severity,
        message: String,
        tag: String,
        throwable: Throwable?
    ) {
        scope.launch {
            // 异步发送日志
            sendToServer(severity, message, tag, throwable)
        }
    }
}
```

### 10. 日志轮转

```kotlin
class RotatingFileLogWriter(
    private val logDir: File,
    private val maxFileSize: Long = 10 * 1024 * 1024,  // 10MB
    private val maxFiles: Int = 5
) : LogWriter() {
    
    private var currentFile: File? = null
    private var currentSize: Long = 0
    
    override fun log(
        severity: Severity,
        message: String,
        tag: String,
        throwable: Throwable?
    ) {
        val logFile = getCurrentLogFile()
        val logMessage = formatMessage(severity, message, tag, throwable)
        
        logFile.appendText(logMessage + "\n")
        currentSize += logMessage.length
        
        // 检查是否需要轮转
        if (currentSize > maxFileSize) {
            rotateLogs()
        }
    }
    
    private fun rotateLogs() {
        // 删除最旧的文件
        val files = logDir.listFiles()?.sortedBy { it.lastModified() } ?: return
        if (files.size >= maxFiles) {
            files.first().delete()
        }
        
        // 创建新文件
        currentFile = null
        currentSize = 0
    }
}
```

---

## 与项目集成

### 1. 在 KMP 项目中添加 Kermit

#### 版本目录配置

```toml
# gradle/libs.versions.toml
[versions]
kermit = "2.0.4"

[libraries]
kermit = { module = "co.touchlab:kermit", version.ref = "kermit" }
```

#### 共享模块配置

```kotlin
// shared/build.gradle.kts
kotlin {
    sourceSets {
        commonMain {
            dependencies {
                implementation(libs.kermit)
            }
        }
    }
}
```

### 2. 创建日志工具类

```kotlin
// shared/src/commonMain/kotlin/com/example/app/Logger.kt
package com.example.app

import co.touchlab.kermit.Logger
import co.touchlab.kermit.Severity

expect object PlatformLogger {
    fun createLogger(tag: String): Logger
}

// 便捷扩展
inline fun <reified T> T.logger(): Logger {
    return PlatformLogger.createLogger(T::class.java.simpleName)
}

// 全局日志配置
object AppLogger {
    private val logger = PlatformLogger.createLogger("App")
    
    fun v(message: () -> String) = logger.v(message = message)
    fun d(message: () -> String) = logger.d(message = message)
    fun i(message: () -> String) = logger.i(message = message)
    fun w(message: () -> String) = logger.w(message = message)
    fun e(message: () -> String) = logger.e(message = message)
    fun a(message: () -> String) = logger.a(message = message)
    
    fun e(throwable: Throwable, message: () -> String) = 
        logger.e(throwable = throwable, message = message)
}
```

### 3. 平台实现

#### Android

```kotlin
// shared/src/androidMain/kotlin/com/example/app/Logger.android.kt
package com.example.app

import co.touchlab.kermit.Logger
import co.touchlab.kermit.Severity
import com.example.app.BuildConfig

actual object PlatformLogger {
    actual fun createLogger(tag: String): Logger {
        return Logger(
            tag = tag,
            minSeverity = if (BuildConfig.DEBUG) Severity.Debug else Severity.Warn,
            logWriters = listOf(
                AndroidLogcatWriter()
            )
        )
    }
}
```

#### iOS

```kotlin
// shared/src/iosMain/kotlin/com/example/app/Logger.ios.kt
package com.example.app

import co.touchlab.kermit.Logger
import co.touchlab.kermit.Severity

actual object PlatformLogger {
    actual fun createLogger(tag: String): Logger {
        return Logger(
            tag = tag,
            minSeverity = Severity.Debug,
            logWriters = listOf(
                OSLogWriter()
            )
        )
    }
}
```

### 4. 使用示例

```kotlin
// shared/src/commonMain/kotlin/com/example/app/data/UserRepository.kt
package com.example.app.data

import com.example.app.logger
import com.example.app.AppLogger

class UserRepository(
    private val api: UserApi,
    private val database: UserDatabase
) {
    private val log = logger()
    
    suspend fun loadUser(userId: String): User {
        log.d { "Loading user: $userId" }
        
        return try {
            val cached = database.getUser(userId)
            if (cached != null) {
                log.i { "Loaded from cache: $userId" }
                return cached
            }
            
            log.d { "Fetching from API: $userId" }
            val user = api.getUser(userId)
            
            database.saveUser(user)
            log.i { "Cached user: $userId" }
            
            user
        } catch (e: Exception) {
            log.e(e) { "Failed to load user: $userId" }
            AppLogger.e(e) { "Critical error in UserRepository" }
            throw e
        }
    }
}
```

### 5. 与 Ktor 集成

```kotlin
// shared/src/commonMain/kotlin/com/example/app/network/HttpClient.kt
package com.example.app.network

import co.touchlab.kermit.Logger
import io.ktor.client.*
import io.ktor.client.plugins.logging.*

class HttpLogger : Logger {
    private val log = Logger.withTag("HttpLogger")
    
    override fun log(message: String) {
        log.d { message }
    }
}

fun createHttpClient(): HttpClient {
    return HttpClient {
        install(Logging) {
            logger = HttpLogger()
            level = LogLevel.ALL
        }
    }
}
```

### 6. 与 SQLDelight 集成

```kotlin
// shared/src/commonMain/kotlin/com/example/app/database/DatabaseFactory.kt
package com.example.app.database

import co.touchlab.kermit.Logger
import app.cash.sqldelight.db.SqlDriver

class DatabaseLogger : SqlDriver.Logger {
    private val log = Logger.withTag("DatabaseLogger")
    
    override fun logQuery(message: String) {
        log.v { message }
    }
}
```

---

## 常见问题

### 1. 日志不显示

**问题**: 日志没有输出到控制台

**解决方案**:
```kotlin
// 检查日志级别
val logger = Logger(minSeverity = Severity.Verbose)  // 设置为最低级别

// 检查 LogWriter
val logger = Logger(
    logWriters = listOf(CommonWriter())  // 确保有 LogWriter
)

// 检查平台配置
// Android: 确保 Logcat 过滤器没有屏蔽标签
// iOS: 检查 Xcode 控制台输出
// JS: 检查浏览器控制台
```

### 2. 性能问题

**问题**: 日志影响应用性能

**解决方案**:
```kotlin
// ✅ 使用 lambda 延迟求值
Logger.i { "Expensive: ${expensiveOperation()}" }

// ✅ 生产环境提高日志级别
val logger = Logger(minSeverity = Severity.Warn)

// ✅ 异步日志
class AsyncLogWriter : LogWriter() {
    private val scope = CoroutineScope(Dispatchers.IO)
    
    override fun log(...) {
        scope.launch { /* 异步处理 */ }
    }
}
```

### 3. 标签混乱

**问题**: 标签不统一，难以过滤

**解决方案**:
```kotlin
// 使用统一的标签策略
inline fun <reified T> T.logger(): Logger {
    return Logger.withTag(T::class.java.simpleName)
}

// 所有类使用相同方式
class UserRepository {
    private val log = logger()  // 标签：UserRepository
}

class UserService {
    private val log = logger()  // 标签：UserService
}
```

### 4. 敏感信息泄露

**问题**: 日志中包含敏感信息

**解决方案**:
```kotlin
// 创建脱敏函数
fun String.maskEmail(): String {
    return this.replace(Regex("(?<=.{3}).(?=.*@)"), "*")
}

fun String.maskPhone(): String {
    return this.replace(Regex("(\\d{3})\\d{4}(\\d{4})"), "$1****$2")
}

// 使用脱敏
Logger.i { "User email: ${user.email.maskEmail()}" }
Logger.i { "User phone: ${user.phone.maskPhone()}" }

// ❌ 避免
Logger.i { "User password: ${user.password}" }
```

### 5. 日志文件过大

**问题**: 日志文件占用过多空间

**解决方案**:
```kotlin
// 使用轮转日志
class RotatingFileLogWriter(
    maxFileSize = 10 * 1024 * 1024,  // 10MB
    maxFiles = 5                      // 最多 5 个文件
) : LogWriter() {
    // 实现轮转逻辑
}

// 定期清理
fun cleanupOldLogs(maxAgeDays: Int = 7) {
    val cutoffTime = System.currentTimeMillis() - (maxAgeDays * 24 * 60 * 60 * 1000)
    logDir.listFiles()?.forEach { file ->
        if (file.lastModified() < cutoffTime) {
            file.delete()
        }
    }
}
```

---

## 参考资源

### 官方文档

- [Kermit 官方文档](https://kermit.touchlab.co/docs/)
- [Kermit GitHub 仓库](https://github.com/touchlab/Kermit)
- [KaMP Kit 示例](https://github.com/touchlab/KaMPKit)

### 相关项目

- [Napier](https://github.com/AAkira/Napier) - 另一个 KMP 日志库
- [KaMP Kit](https://github.com/touchlab/KaMPKit) - KMP 项目模板
- [Touchlab 博客](https://touchlab.co/blog)

### 版本信息

- **当前版本**: 2.0.4
- **发布日期**: 2024+
- **Kotlin 版本**: 1.9.0+
- **多平台**: Android、iOS、JS、JVM、Native

---

## 总结

Kermit 是一个功能强大的 Kotlin Multiplatform 日志库，具有:

1. **跨平台支持**: 一次编写，多平台运行
2. **开箱即用**: 默认配置适合大多数场景
3. **易于扩展**: 可自定义 LogWriter
4. **Kotlin 友好**: 支持 lambda 和默认参数
5. **灵活配置**: 支持级别过滤、格式化、多 LogWriter
6. **生产就绪**: 支持 Crashlytics、Bugsnag 集成

**推荐使用场景**:
- Kotlin Multiplatform 项目日志
- 跨平台业务逻辑调试
- 生产环境日志收集
- 崩溃追踪
- 多环境日志管理

对于 KMP 项目，Kermit 是日志记录的理想选择。

---

## 更新记录

| 日期 | 版本 | 描述 |
|------|------|------|
| 2026-03-07 | 1.0 | 初始版本，整理 Kermit 日志库文档 |
