# MukaAI 项目架构文档

## 元数据

- **标题**: MukaAI 项目架构文档
- **作者**: 架构师
- **日期**: 2026-03-13
- **版本**: 1.0.0

---

## 1. 项目概述

MukaAI 是一个基于 Kotlin Multiplatform (KMP) 的跨平台 AI 助手项目，采用前后端分离架构，支持多平台部署。

### 1.1 技术选型

| 层级 | 技术框架 | 版本 | 说明 |
|------|----------|------|------|
| 编程语言 | Kotlin | 2.3.0 | 主开发语言 |
| 跨平台框架 | Kotlin Multiplatform | 2.3.0 | 代码共享 |
| UI框架 | Jetpack Compose Multiplatform | 1.10.0 | 跨平台UI |
| 后端框架 | Ktor | 3.3.3 | 服务端框架 |
| 构建工具 | Gradle | 8.x | 构建管理 |
| 包管理 | Gradle Version Catalog | - | 版本统一管理 |

---

## 2. 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         MukaAI 项目架构                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    composeApp (UI层)                     │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │   │
│  │  │ Android │ │   iOS   │ │ Desktop │ │   Web   │       │   │
│  │  │  (APP)  │ │(Framework)│ │  (JVM)  │ │(JS/Wasm)│       │   │
│  │  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘       │   │
│  │       └─────────────┴─────────┴─────────────┘           │   │
│  │                     Jetpack Compose                      │   │
│  └─────────────────────────┬───────────────────────────────┘   │
│                            │                                    │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    shared (共享层)                       │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │   │
│  │  │commonMain│ │androidMain│ │ iosMain │ │ jvmMain │       │   │
│  │  │ (通用代码)│ │(平台特定) │ │(平台特定) │ │(平台特定) │       │   │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘       │   │
│  │  ┌─────────┐ ┌─────────┐                                 │   │
│  │  │  jsMain │ │wasmJsMain│                                 │   │
│  │  └─────────┘ └─────────┘                                 │   │
│  └─────────────────────────┬───────────────────────────────┘   │
│                            │                                    │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    server (服务层)                       │   │
│  │              Ktor + Netty (JVM Only)                     │   │
│  │                    REST API 服务                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. 目录结构

```
MukaAI/
├── composeApp/                          # UI应用模块
│   ├── src/
│   │   ├── androidMain/kotlin/          # Android入口
│   │   │   └── app/muka/ai/MainActivity.kt
│   │   ├── iosMain/kotlin/              # iOS入口
│   │   │   └── app/muka/ai/MainViewController.kt
│   │   ├── jvmMain/kotlin/              # Desktop入口
│   │   │   └── app/muka/ai/main.kt
│   │   ├── webMain/kotlin/              # Web入口
│   │   │   └── app/muka/ai/main.kt
│   │   ├── commonMain/kotlin/           # 共享UI代码
│   │   │   └── app/muka/ai/App.kt       # 主UI组件
│   │   └── commonTest/kotlin/           # UI测试
│   └── build.gradle.kts
│
├── server/                              # 后端服务模块
│   ├── src/
│   │   ├── main/kotlin/                 # 服务端代码
│   │   │   └── app/muka/ai/Application.kt
│   │   └── test/kotlin/                 # 服务端测试
│   │       └── app/muka/ai/ApplicationTest.kt
│   └── build.gradle.kts
│
├── shared/                              # 共享代码模块
│   ├── src/
│   │   ├── commonMain/kotlin/           # 通用代码
│   │   │   └── app/muka/ai/
│   │   │       ├── Platform.kt          # 平台抽象接口
│   │   │       ├── Greeting.kt          # 示例业务类
│   │   │       └── Constants.kt         # 常量定义
│   │   ├── commonTest/kotlin/           # 通用测试
│   │   ├── androidMain/kotlin/          # Android平台实现
│   │   ├── iosMain/kotlin/              # iOS平台实现
│   │   ├── jvmMain/kotlin/              # JVM平台实现
│   │   ├── jsMain/kotlin/               # JS平台实现
│   │   └── wasmJsMain/kotlin/           # Wasm平台实现
│   └── build.gradle.kts
│
├── gradle/
│   └── libs.versions.toml               # 版本目录管理
├── build.gradle.kts                     # 根构建脚本
├── settings.gradle.kts                  # 项目设置
└── gradle.properties                    # Gradle配置
```

---

## 4. 核心类/方法签名

### 4.1 shared模块

#### Platform.kt
```kotlin
package app.muka.ai

/**
 * 平台信息接口
 * 定义各平台的基本信息
 */
interface Platform {
    /** 平台名称 */
    val name: String
}

/**
 * 获取当前平台实例
 * @return 平台实现对象
 */
expect fun getPlatform(): Platform
```

#### Greeting.kt
```kotlin
package app.muka.ai

/**
 * 问候语生成类
 * 演示跨平台业务逻辑共享
 */
class Greeting {
    private val platform = getPlatform()

    /**
     * 生成问候语
     * @return 包含平台信息的问候字符串
     */
    fun greet(): String
}
```

### 4.2 composeApp模块

#### App.kt
```kotlin
package app.muka.ai

import androidx.compose.runtime.Composable
import androidx.compose.ui.tooling.preview.Preview

/**
 * 主应用UI组件
 * 跨平台共享的Compose UI入口
 */
@Composable
@Preview
fun App()
```

### 4.3 server模块

#### Application.kt
```kotlin
package app.muka.ai

import io.ktor.server.application.Application

/**
 * 服务器主入口
 * 启动Ktor服务
 */
fun main()

/**
 * Ktor应用模块配置
 * 配置路由、中间件等
 */
fun Application.module()
```

---

## 5. 数据流图

```
┌─────────────────────────────────────────────────────────────────┐
│                         数据流向图                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   用户操作                                                        │
│      │                                                           │
│      ▼                                                           │
│  ┌─────────────┐    Compose UI    ┌─────────────┐               │
│  │  composeApp │ ◄──────────────► │   shared    │               │
│  │   (UI层)    │   业务逻辑调用    │  (业务逻辑)  │               │
│  └──────┬──────┘                  └──────┬──────┘               │
│         │                                │                       │
│         │ HTTP/WebSocket                 │                       │
│         ▼                                ▼                       │
│  ┌─────────────┐                  ┌─────────────┐               │
│  │    server   │ ◄──────────────► │   shared    │               │
│  │   (API服务)  │   共享数据模型    │  (数据模型)  │               │
│  └─────────────┘                  └─────────────┘               │
│         │                                                        │
│         ▼                                                        │
│   外部AI服务                                                      │
│   (Ollama/LMStudio等)                                             │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 6. 平台支持矩阵

| 平台 | 类型 | 技术栈 | 输出格式 | 状态 |
|------|------|--------|----------|------|
| Android | 移动应用 | Kotlin/JVM + Compose | APK/AAB | ✅ 支持 |
| iOS | 移动应用 | Kotlin/Native + Compose | Framework | ✅ 支持 |
| Desktop (Windows) | 桌面应用 | Kotlin/JVM + Compose | MSI | ✅ 支持 |
| Desktop (macOS) | 桌面应用 | Kotlin/JVM + Compose | DMG | ✅ 支持 |
| Desktop (Linux) | 桌面应用 | Kotlin/JVM + Compose | DEB | ✅ 支持 |
| Web (JavaScript) | Web应用 | Kotlin/JS + Compose | JS Bundle | ✅ 支持 |
| Web (WebAssembly) | Web应用 | Kotlin/Wasm + Compose | Wasm Bundle | ✅ 支持 |
| Server | 后端服务 | Kotlin/JVM + Ktor | JAR | ✅ 支持 |

---

## 7. 依赖版本管理

版本统一配置于 `gradle/libs.versions.toml`：

```toml
[versions]
kotlin = "2.3.0"
ktor = "3.3.3"
composeMultiplatform = "1.10.0"
agp = "8.11.2"
androidx-lifecycle = "2.9.6"
kotlinx-coroutines = "1.10.2"
```

---

## 8. 构建命令

```bash
# 构建所有模块
./gradlew build

# 运行服务端
./gradlew :server:run

# 运行桌面端
./gradlew :composeApp:run

# 构建桌面安装包
./gradlew :composeApp:package

# 运行测试
./gradlew test
```

---

## 9. 开发规范

1. **代码语言**: 所有代码注释使用中文
2. **包命名**: 统一使用 `app.muka.ai` 作为根包名
3. **平台实现**: shared模块中的平台特定代码使用 `expect/actual` 机制
4. **依赖管理**: 所有依赖版本必须在 `libs.versions.toml` 中定义
5. **禁止事项**:
   - 禁止使用 `Map` 返回数据
   - 禁止使用 `Any` 类型
   - 禁止使用 Java 反射
   - 禁止使用 HttpURLConnection、OkHttp、JavaWebSocket

---

## 10. 后续开发指引

### 10.1 添加共享业务逻辑

在 `shared/src/commonMain/kotlin/app/muka/ai/` 下创建业务类：

```kotlin
// 示例：添加用户服务
interface UserService {
    suspend fun getUser(id: String): User
    suspend fun saveUser(user: User): Result<Unit>
}

// 数据模型
data class User(
    val id: String,
    val name: String,
    val email: String
)
```

### 10.2 添加服务端API

在 `server/src/main/kotlin/app/muka/ai/` 下扩展路由：

```kotlin
fun Application.module() {
    routing {
        // 现有路由
        get("/") { ... }
        
        // 新增API
        route("/api/v1") {
            userRoutes()
        }
    }
}
```

### 10.3 添加UI组件

在 `composeApp/src/commonMain/kotlin/app/muka/ai/` 下创建Compose组件：

```kotlin
@Composable
fun UserProfile(user: User) {
    // UI实现
}
```

---

## 参考文档

- [Kotlin Multiplatform 官方文档](https://kotlinlang.org/docs/multiplatform.html)
- [Compose Multiplatform 官方文档](https://www.jetbrains.com/lp/compose-multiplatform/)
- [Ktor 官方文档](https://ktor.io/docs/)
