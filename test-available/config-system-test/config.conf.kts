import app.muka.ai.config.*

// 喵卡助手 Kotlin DSL 配置文件
// 展示动态配置、环境感知、条件配置等高级功能

appConfig {

    // 根据环境动态设置服务器配置
    server {
        when (System.getenv("ENVIRONMENT") ?: "dev") {
            "prod" -> {
                host = "0.0.0.0"
                port = 8080
                timeout = 60000
            }
            "test" -> {
                host = "localhost"
                port = 9000
                timeout = 30000
            }
            else -> { // dev
                host = "0.0.0.0"
                port = 8080
                timeout = 30000
            }
        }
    }

    // 根据操作系统动态设置路径
    paths {
        val osName = System.getProperty("os.name").toLowerCase()
        workspaceRoot = if (osName.contains("win")) {
            System.getenv("USERPROFILE") + "\\muka\\workspaces"
        } else {
            System.getenv("HOME") + "/muka/workspaces"
        }
        
        skillDirectory = if (osName.contains("win")) {
            System.getenv("APPDATA") + "\\muka\\skills"
        } else {
            System.getenv("HOME") + "/.config/muka/skills"
        }
    }

    // 根据环境动态设置 AI 服务
    aiService {
        val env = System.getenv("ENVIRONMENT") ?: System.getProperty("env", "dev")
        when (env) {
            "prod" -> {
                lmStudioUrl = "https://lmstudio.example.com/v1"
                apiKey = System.getenv("LMSTUDIO_API_KEY") ?: "missing-api-key"
                modelName = "production-model"
            }
            "test" -> {
                lmStudioUrl = "http://test-lmstudio:1234/v1"
                apiKey = "test-api-key"
                modelName = "test-model"
            }
            else -> { // dev
                lmStudioUrl = "http://localhost:1234/v1"
                apiKey = "dev-api-key"
                modelName = "development-model"
            }
        }
    }

    // 根据环境设置代理
    proxy {
        val env = System.getenv("ENVIRONMENT") ?: System.getProperty("env", "dev")
        if (env == "prod") {
            general {
                enabled = true
                host = "proxy.company.com"
                port = 8080
                username = System.getenv("PROXY_USER")
                password = System.getenv("PROXY_PASS")
            }
        } else {
            general {
                enabled = false
            }
        }
        
        aiService {
            enabled = false // AI 服务通常不需要代理
        }
    }

    // 根据系统自动检测浏览器路径
    browser {
        val osName = System.getProperty("os.name").toLowerCase()
        executablePath = when {
            osName.contains("win") -> {
                // Windows 系统检测多个浏览器
                val browsers = listOf(
                    "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
                    "C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
                    "C:\\Program Files\\Mozilla Firefox\\firefox.exe",
                    "C:\\Program Files (x86)\\Mozilla Firefox\\firefox.exe"
                )
                browsers.firstOrNull { java.io.File(it).exists() }
            }
            osName.contains("mac") -> {
                // macOS 检测浏览器
                val browsers = listOf(
                    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
                    "/Applications/Firefox.app/Contents/MacOS/Firefox"
                )
                browsers.firstOrNull { java.io.File(it).exists() }
            }
            else -> {
                // Linux 检测浏览器
                val browsers = listOf("google-chrome", "chromium-browser", "firefox")
                browsers.firstOrNull { 
                    Runtime.getRuntime().exec(arrayOf("which", it)).waitFor() == 0 
                }
            }
        }
        
        headless = System.getProperty("browser.headless", "false").toBoolean()
    }

    // 根据环境设置日志级别
    log {
        val env = System.getenv("ENVIRONMENT") ?: System.getProperty("env", "dev")
        level = when (env) {
            "prod" -> "INFO"
            "test" -> "WARN"
            else -> "DEBUG" // dev 环境详细日志
        }
    }

    // 动态配置示例
    dynamic {
        println("正在根据环境动态配置...")
        println("当前环境: ${System.getenv("ENVIRONMENT") ?: "dev"}")
        println("操作系统: ${System.getProperty("os.name")}")
        println("JVM 版本: ${System.getProperty("java.version")}")
    }
}
