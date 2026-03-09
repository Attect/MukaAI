# 基于用户口述的软件设计需求

## 文档信息
- 创建日期：2026-03-09
- 状态：可行性验证完成，等待项目启动
- 来源：`docs/user_say.md`

## 项目概述

### 项目名称 (暂定)
**Kotlin AI Assistant (KAA)** - Kotlin 多平台 AI 助手

### 项目定位
一个基于 Kotlin Multiplatform 的个人 AI 助手系统，参考 nanobot 和 openclaw 的设计理念，提供多工作区管理、可扩展技能系统、多模态交互能力。

## 核心架构设计

### 1. 技术架构
```
┌─────────────────────────────────────────────────────────┐
│                    客户端层 (KMP)                        │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │
│  │ Desktop  │ │ Android  │ │   iOS    │ │Web (WASM)│   │
│  │Compose MP│ │Compose MP│ │Swift UI  │ │Compose JS│   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘   │
└─────────────────────────────────────────────────────────┘
                        ↓ Ktor Client
┌─────────────────────────────────────────────────────────┐
│                   服务端层 (Kotlin Native)                │
│  ┌──────────────────────────────────────────────────┐   │
│  │              Ktor Server Core                     │   │
│  └──────────────────────────────────────────────────┘   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │
│  │Workspace │ │  Skill   │ │  Session │ │  Browser │   │
│  │ Manager  │ │  System  │ │ Manager  │ │ Controller│   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │
│  │  Shell   │ │  LM      │ │  Media   │ │  File    │   │
│  │ Executor │ │  Studio  │ │ Processor│ │  System  │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘   │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│                    外部服务层                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │
│  │LM Studio │ │  Web     │ │  System  │ │  Node.js │   │
│  │   API    │ │ Browser  │ │  Shell   │ │Playwright│   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘   │
└─────────────────────────────────────────────────────────┘
```

### 2. 模块划分

#### 2.1 共享模块 (shared)
- **数据模型**: 所有平台共享的数据类
- **业务逻辑**: 核心业务逻辑抽象
- **接口定义**: 平台特定实现的接口

#### 2.2 服务端模块 (server)
- **Ktor 服务端**: HTTP/WebSocket服务
- **工作区管理**: 多工作区支持
- **技能系统**: Skill 加载和执行
- **会话管理**: 多模态会话处理
- **AI 集成**: LM Studio 和其他 LLM 提供商

#### 2.3 客户端模块
- **desktop**: 桌面客户端 (Compose Desktop)
- **android**: Android 客户端 (Compose Android)
- **ios**: iOS 客户端 (SwiftUI + KMP)
- **web**: Web 客户端 (Compose Web/WASM)

### 3. 核心功能模块

#### 3.1 多工作区管理
**设计要点**:
- 工作区隔离：每个工作区有独立的文件系统访问范围
- 配置独立：每个工作区有自己的配置文件和技能列表
- 快速切换：支持在工作区之间快速切换
- 安全性：基于白名单的路径访问控制

**实现方案**:
```kotlin
data class Workspace(
    val id: String,
    val name: String,
    val rootPath: String,
    val config: WorkspaceConfig,
    val metadata: Map<String, String>
)

class WorkspaceManager {
    fun createWorkspace(name: String, rootPath: String): Workspace
    fun switchWorkspace(workspaceId: String): Boolean
    fun isPathAllowed(path: String): Boolean
}
```

#### 3.2 Skill 系统
**设计要点**:
- SKILL.md 格式：标准化的技能定义文件
- 动态加载：运行时发现和加载技能
- 安全执行：沙箱环境执行技能代码
- 可扩展：支持自定义技能和第三方技能

**实现方案**:
```kotlin
data class SkillDefinition(
    val name: String,
    val description: String,
    val parameters: Map<String, ParameterSpec>,
    val requiredPermissions: List<String>
)

interface SkillExecutor {
    suspend fun execute(parameters: Map<String, String>): SkillResult
}
```

#### 3.3 命令行执行
**设计要点**:
- 跨平台支持：Windows (PowerShell/CMD), Linux (bash), macOS (zsh)
- 安全限制：命令白名单、超时控制、资源限制
- 输出捕获：实时捕获 stdout/stderr
- 环境变量：支持自定义环境变量

**实现方案**:
```kotlin
suspend fun executeCommand(
    command: List<String>,
    timeout: Long = 30000,
    workingDir: String? = null,
    environment: Map<String, String>? = null
): CommandResult
```

#### 3.4 浏览器控制
**设计要点**:
- Playwright 集成：通过 Node.js 调用 Playwright
- 页面自动化：导航、点击、输入、截图
- 网络拦截：请求/响应拦截和修改
- 无头模式：支持无头浏览器运行

**实现方案**:
```kotlin
interface BrowserController {
    suspend fun navigate(url: String)
    suspend fun click(selector: String)
    suspend fun screenshot(): ByteArray
    suspend fun executeJavaScript(code: String): Any?
}
```

#### 3.5 多模态会话
**设计要点**:
- 文本和图片支持：同时处理文本和图片消息
- Base64 编码：图片数据传输
- 格式转换：适配不同 LLM 的多模态接口
- 历史记录：会话历史持久化

**实现方案**:
```kotlin
sealed class MessageContent {
    data class Text(val content: String) : MessageContent()
    data class Image(val data: ByteArray, val mimeType: String) : MessageContent()
    data class Multimodal(val text: String, val images: List<Image>) : MessageContent()
}
```

#### 3.6 LM Studio 集成
**设计要点**:
- 自动发现：检测本地运行的 LM Studio 服务
- 模型管理：获取和切换可用模型
- OpenAI 兼容：使用 OpenAI API 格式
- 备用方案：支持其他 LLM 提供商

**实现方案**:
```kotlin
class LMStudioClient(baseUrl: String = "http://localhost:1234/v1") {
    suspend fun isAvailable(): Boolean
    suspend fun getModels(): List<ModelInfo>
    suspend fun chat(messages: List<Message>): ChatResponse
}
```

## 技术栈选型

### 核心技术
- **语言**: Kotlin 2.3.10+
- **多平台**: Kotlin Multiplatform
- **网络**: Ktor 3.4.1+
- **序列化**: Kotlinx Serialization 1.7.3+
- **协程**: Kotlinx Coroutines 1.9.0+

### UI 框架
- **Desktop**: Compose Multiplatform Desktop
- **Android**: Jetpack Compose
- **iOS**: SwiftUI (原生) + KMP 共享逻辑
- **Web**: Compose Multiplatform Web (WASM)

### 服务端
- **框架**: Ktor Server (Netty/CIO)
- **平台**: Kotlin Native (推荐) 或 JVM

### 数据库 (可选)
- **跨平台**: SQLDelight (推荐)
- **轻量级**: Kotlinx Serialization JSON 文件

### 日志
- **多平台**: Kermit (推荐)
- **服务端**: Ktor 内置日志

## 安全性设计

### 1. 工作区隔离
- 文件系统访问限制在工作区根目录内
- 支持配置允许的额外路径
- 禁止访问系统敏感目录

### 2. 命令执行安全
- 命令白名单机制
- 超时和取消控制
- 资源使用限制
- 环境变量隔离

### 3. Skill 沙箱
- 独立的执行环境
- 权限分级控制
- 资源使用配额
- 异常隔离

### 4. 网络安全
- CORS 配置
- 认证和授权
- 请求速率限制
- SSRF 防护

## 项目结构规划

```
project-root/
├── shared/                    # 共享模块
│   ├── src/
│   │   ├── commonMain/       # 公共代码
│   │   ├── jvmMain/          # JVM 特定代码
│   │   ├── androidMain/      # Android 特定代码
│   │   ├── iosMain/          # iOS 特定代码
│   │   └── jsMain/           # Web 特定代码
│   └── build.gradle.kts
├── server/                    # 服务端模块
│   ├── src/
│   │   └── nativeMain/       # Kotlin Native 代码
│   └── build.gradle.kts
├── desktop/                   # 桌面客户端
│   ├── src/
│   │   └── jvmMain/          # JVM + Compose Desktop
│   └── build.gradle.kts
├── android/                   # Android 客户端
│   ├── src/
│   │   └── androidMain/      # Android + Compose
│   └── build.gradle.kts
├── ios/                       # iOS 客户端
│   ├── src/
│   │   └── iosMain/          # iOS + SwiftUI
│   └── build.gradle.kts
├── web/                       # Web 客户端
│   ├── src/
│   │   └── jsMain/           # Web + Compose JS/WASM
│   └── build.gradle.kts
├── gradle/
│   └── libs.versions.toml    # 版本目录
├── settings.gradle.kts        # 项目设置
└── build.gradle.kts           # 根构建脚本
```

## 开发阶段规划

### 阶段 1: 基础架构 (2-3 周)
- [ ] 项目初始化和 Gradle 配置
- [ ] 共享模块基础数据模型
- [ ] Ktor 服务端基础框架
- [ ] 工作区管理实现

### 阶段 2: 核心功能 (3-4 周)
- [ ] Skill 系统实现
- [ ] 命令行执行器
- [ ] LM Studio 集成
- [ ] 多模态会话处理

### 阶段 3: 浏览器控制 (2-3 周)
- [ ] Node.js Playwright 集成
- [ ] 浏览器控制 API
- [ ] 网页自动化功能

### 阶段 4: 客户端开发 (4-6 周)
- [ ] Desktop 客户端 UI
- [ ] Android 客户端 UI
- [ ] iOS 客户端 UI
- [ ] Web 客户端 UI

### 阶段 5: 集成测试 (2-3 周)
- [ ] 端到端测试
- [ ] 跨平台兼容性测试
- [ ] 性能优化
- [ ] 安全性加固

## 风险评估

### 技术风险
1. **Kotlin Native 成熟度**: 某些平台可能支持不完善
   - 缓解方案：使用 JVM 作为备选
   
2. **Compose Web 限制**: Web 平台功能可能受限
   - 缓解方案：使用 React/Vue 作为备选

3. **浏览器控制依赖**: Node.js 依赖增加复杂性
   - 缓解方案：提供 Selenium 备选方案

### 安全风险
1. **命令执行**: 恶意命令执行风险
   - 缓解方案：严格的白名单和沙箱

2. **文件系统访问**: 未授权文件访问
   - 缓解方案：工作区隔离和权限控制

3. **Skill 安全**: 恶意技能代码
   - 缓解方案：代码审查和沙箱执行

## 下一步行动

### 立即行动
1. ✅ 完成可行性测试 (已完成)
2. 🔄 创建详细的项目计划
3. ⏳ 初始化项目结构和 Gradle 配置
4. ⏳ 实现基础服务端框架

### 需要用户确认
1. 项目名称和 Logo 设计
2. 优先开发的客户端平台
3. 是否需要数据库支持
4. 部署方式 (本地/云端)

## 参考文档
- [可行性测试报告](../test-available/index.md)
- [nanobot 分析](../references/nanobot/nanobot-analysis.md)
- [openclaw 设计模式](../references/openclaw/docs/patterns/INDEX.md)
- [Ktor 集成指南](../references/ktor/ktor-integration.md)
- [LM Studio API](../references/lm-studio/lm-studio-rest-api.md)
