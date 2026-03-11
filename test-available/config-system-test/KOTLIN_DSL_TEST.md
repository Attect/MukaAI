# Kotlin DSL 配置系统可行性测试

## 测试目的
验证 Kotlin DSL 配置文件的可行性，特别是其动态配置能力。

## Kotlin DSL vs HOCON

### Kotlin DSL 优势
1. **动态配置能力** - 可根据环境变量、操作系统等动态生成配置
2. **类型安全** - 编译时检查，避免配置错误
3. **代码复用** - 可提取公共配置逻辑
4. **条件配置** - 支持 when/if 表达式
5. **环境感知** - 自动检测运行环境
6. **强大的表达能力** - 可实现复杂逻辑

### HOCON 优势
1. **简单易用** - 语法简洁，无需编程知识
2. **成熟稳定** - Typesafe Config 库被广泛使用
3. **加载快速** - 无需编译，直接解析

## 测试场景

### 场景 1: 环境感知配置
```kotlin
server {
    when (System.getenv("ENVIRONMENT") ?: "dev") {
        "prod" -> {
            host = "0.0.0.0"
            port = 8080
        }
        "test" -> {
            host = "localhost"
            port = 9000
        }
        else -> { // dev
            host = "0.0.0.0"
            port = 8080
        }
    }
}
```

### 场景 2: 跨平台路径配置
```kotlin
paths {
    val osName = System.getProperty("os.name").toLowerCase()
    workspaceRoot = if (osName.contains("win")) {
        System.getenv("USERPROFILE") + "\\muka\\workspaces"
    } else {
        System.getenv("HOME") + "/muka/workspaces"
    }
}
```

### 场景 3: 动态浏览器检测
```kotlin
browser {
    val osName = System.getProperty("os.name").toLowerCase()
    executablePath = when {
        osName.contains("win") -> {
            listOf(
                "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
                "C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe"
            ).firstOrNull { File(it).exists() }
        }
        osName.contains("mac") -> {
            "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
        }
        else -> "google-chrome"
    }
}
```

### 场景 4: 从环境变量读取敏感信息
```kotlin
aiService {
    apiKey = System.getenv("LMSTUDIO_API_KEY") ?: "default-key"
}

proxy {
    general {
        username = System.getenv("PROXY_USER")
        password = System.getenv("PROXY_PASS")
    }
}
```

## 技术实现

### Kotlin Scripting 方案

使用 Kotlin 官方的 Scripting API：

```kotlin
dependencies {
    implementation("org.jetbrains.kotlin:kotlin-scripting-jvm:2.3.10")
    implementation("org.jetbrains.kotlin:kotlin-scripting-common:2.3.10")
    implementation("org.jetbrains.kotlin:kotlin-scripting-jvm-host:2.3.10")
}
```

### 实现步骤

1. **定义 DSL 语法** - 使用类型安全的构建器模式
2. **编译脚本** - 使用 Kotlin 编译器 API
3. **执行脚本** - 在隔离的类加载器中运行
4. **获取配置** - 从脚本中提取配置对象

## 测试命令

### 测试 1: 默认环境配置
```bash
.\gradlew.bat run
```

### 测试 2: 生产环境配置
```bash
.\gradlew.bat run --args="--env prod"
```

### 测试 3: 测试环境配置
```bash
.\gradlew.bat run --args="--env test"
```

### 测试 4: 设置环境变量
```bash
# Windows PowerShell
$env:ENVIRONMENT="prod"; .\gradlew.bat run

# Linux/macOS
ENVIRONMENT=prod ./gradlew run
```

### 测试 5: 命令行参数覆盖
```bash
.\gradlew.bat run --args="--env dev --port 9999 --host 127.0.0.1"
```

## 预期结果

### 开发环境 (dev)
- host: 0.0.0.0
- port: 8080
- logLevel: DEBUG
- 代理：禁用

### 生产环境 (prod)
- host: 0.0.0.0
- port: 8080
- logLevel: INFO
- 代理：启用

### 测试环境 (test)
- host: localhost
- port: 9000
- logLevel: WARN
- 代理：禁用

## 验收标准

- ✅ Kotlin DSL 配置文件能被正确加载
- ✅ 环境感知配置正常工作
- ✅ 跨平台路径配置正确
- ✅ 动态浏览器检测有效
- ✅ 环境变量读取成功
- ✅ 命令行参数覆盖生效
- ✅ 配置优先级正确：命令行 > DSL > 默认值

## 风险评估

### 已识别风险

1. **Kotlin Scripting 配置复杂**
   - 风险等级：中
   - 影响：增加项目复杂性和构建时间
   - 缓解：使用成熟的 Scripting API，提供详细文档

2. **脚本执行性能**
   - 风险等级：低
   - 影响：启动时编译脚本可能较慢
   - 缓解：缓存编译结果，支持预编译

3. **安全性**
   - 风险等级：中
   - 影响：恶意脚本可能执行危险操作
   - 缓解：限制脚本权限，提供沙箱环境

## 结论

Kotlin DSL 配置系统提供了强大的动态配置能力，特别适合：
- 多环境部署（dev/test/prod）
- 跨平台应用
- 需要复杂配置逻辑的场景
- 对类型安全要求高的项目

虽然配置相对复杂，但带来的灵活性和安全性是值得的。

---

**测试状态**: 进行中  
**创建时间**: 2026-03-10  
**更新时间**: 2026-03-10
