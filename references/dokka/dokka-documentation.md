# Dokka Kotlin 文档生成工具参考指南

> **版本**: 2.1.0+  
> **更新日期**: 2026-03-07  
> **官方文档**: https://kotlinlang.org/docs/dokka-get-started.html  
> **GitHub**: https://github.com/Kotlin/dokka

---

## 目录

- [概述](#概述)
- [快速开始](#快速开始)
- [系统要求](#系统要求)
- [安装配置](#安装配置)
- [生成文档](#生成文档)
- [输出格式](#输出格式)
- [单项目配置](#单项目配置)
- [多项目配置](#多项目配置)
- [聚合文档](#聚合文档)
- [高级配置](#高级配置)
- [发布 Javadoc JAR](#发布-javadoc-jar)
- [Dokka 插件](#dokka-插件)
- [最佳实践](#最佳实践)
- [常见问题](#常见问题)

---

## 概述

Dokka 是 JetBrains 官方推出的 Kotlin 文档生成工具，专为 Kotlin 语言特性设计。它能够从 Kotlin 代码中自动提取 KDoc 注释并生成美观易读的 API 文档。

### 主要特性

- **Kotlin 优先**: 专为 Kotlin 语言特性设计，完美支持 Kotlin 语法
- **多平台支持**: 支持 Kotlin Multiplatform 项目文档生成
- **多种输出格式**: HTML、Javadoc 等格式
- **高度可扩展**: 支持自定义插件和主题
- **Gradle 集成**: 与 Gradle 构建系统无缝集成
- **构建缓存**: 支持 Gradle 构建缓存和配置缓存
- **多项目聚合**: 可聚合多个子项目的文档

### 适用场景

- Kotlin 项目 API 文档生成
- Kotlin Multiplatform 跨平台项目文档
- 库和项目文档发布
- Maven Central 发布需求
- 团队内部 API 文档
- 开源项目文档

### 与 Javadoc 对比

| 特性 | Dokka | Javadoc |
|------|-------|---------|
| **Kotlin 支持** | 原生支持 | 有限支持 |
| **KDoc 语法** | 完整支持 | 不支持 |
| **多平台** | 支持 KMP | 不支持 |
| **输出格式** | HTML、Javadoc | HTML |
| **扩展性** | 插件系统 | 有限扩展 |
| **外观** | 现代化 UI | 传统样式 |

---

## 快速开始

### 1. 应用 Dokka 插件

在项目根目录的 `build.gradle.kts` 中添加:

```kotlin
plugins {
    id("org.jetbrains.dokka") version "2.1.0"
}
```

### 2. 生成文档

运行 Gradle 任务:

```bash
./gradlew :dokkaGenerate
```

### 3. 查看文档

生成的文档位于 `build/dokka/html` 目录:

```bash
# 在浏览器中打开
open build/dokka/html/index.html  # macOS
start build\dokka\html\index.html  # Windows
```

---

## 系统要求

确保项目满足最低版本要求:

| 工具 | 最低版本 |
|------|---------|
| **Gradle** | 7.6+ |
| **Android Gradle Plugin** | 7.0+ |
| **Kotlin Gradle Plugin** | 1.9+ |
| **Dokka Gradle Plugin** | 2.1.0+ |

### 版本兼容性

```kotlin
// gradle/libs.versions.toml
[versions]
kotlin = "2.3.10"
dokka = "2.1.0"
gradle = "8.5"

[plugins]
kotlin-jvm = { id = "org.jetbrains.kotlin.jvm", version.ref = "kotlin" }
kotlin-multiplatform = { id = "org.jetbrains.kotlin.multiplatform", version.ref = "kotlin" }
dokka = { id = "org.jetbrains.dokka", version.ref = "dokka" }
```

---

## 安装配置

### 应用 Dokka 插件

#### 方式 1: 版本目录 (推荐)

```kotlin
// settings.gradle.kts
pluginManagement {
    repositories {
        mavenCentral()
        gradlePluginPortal()
    }
}

// gradle/libs.versions.toml
[versions]
dokka = "2.1.0"

[plugins]
dokka = { id = "org.jetbrains.dokka", version.ref = "dokka" }

// build.gradle.kts (根项目)
plugins {
    alias(libs.plugins.dokka)
}
```

#### 方式 2: 直接声明

```kotlin
// build.gradle.kts (根项目)
plugins {
    id("org.jetbrains.dokka") version "2.1.0"
}
```

#### 方式 3: 多项目应用

```kotlin
// settings.gradle.kts
pluginManagement {
    repositories {
        mavenCentral()
        gradlePluginPortal()
    }
}

// build.gradle.kts (根项目)
plugins {
    id("org.jetbrains.dokka") version "2.1.0" apply false
}

// 子项目 build.gradle.kts
plugins {
    id("org.jetbrains.dokka")
}
```

### 启用构建缓存

Dokka 支持 Gradle 构建缓存和配置缓存，提升构建性能。

#### 启用构建缓存

```kotlin
// settings.gradle.kts
buildCache {
    local {
        isEnabled = true
    }
    remote<HttpBuildCache> {
        url = uri("http://your-build-cache-server:8080/cache/")
        isEnabled = true
    }
}
```

#### 启用配置缓存

```properties
# gradle.properties
org.gradle.configuration-cache=true
org.gradle.configuration-cache.problems=warn
```

---

## 生成文档

### Gradle 任务

Dokka 提供以下 Gradle 任务:

| 任务 | 描述 | 输出格式 |
|------|------|---------|
| `dokkaGenerate` | 生成所有已应用插件的文档 | 所有格式 |
| `dokkaGeneratePublicationHtml` | 生成 HTML 文档 | HTML |
| `dokkaGeneratePublicationJavadoc` | 生成 Javadoc 文档 | Javadoc |

### 使用示例

#### 生成 HTML 文档

```bash
# 单项目
./gradlew :dokkaGeneratePublicationHtml

# 多项目 (从根项目)
./gradlew :dokkaGenerate

# 或指定项目
./gradlew :shared:dokkaGeneratePublicationHtml
```

#### 生成 Javadoc 文档

```bash
./gradlew :dokkaGeneratePublicationJavadoc
```

#### 生成所有格式

```bash
./gradlew :dokkaGenerate
```

### 输出目录

默认输出目录:

```
build/
└── dokka/
    ├── html/          # HTML 文档
    └── javadoc/       # Javadoc 文档
```

自定义输出目录:

```kotlin
// build.gradle.kts
dokka {
    dokkaPublications.html {
        outputDirectory.set(layout.buildDirectory.dir("documentation/html"))
    }
}
```

### 避免的用法

```bash
# ❌ 不推荐：会运行所有子项目的 dokkaGenerate 任务
./gradlew dokkaGenerate

# ✅ 推荐：指定项目路径
./gradlew :dokkaGenerate
./gradlew :aggregatingProject:dokkaGenerate
```

---

## 输出格式

### HTML 格式 (默认)

现代化、响应式的 HTML 文档。

**特点**:
- 响应式设计
- 搜索功能
- 导航树
- 语法高亮
- 移动设备友好

**应用插件**:
```kotlin
plugins {
    id("org.jetbrains.dokka") version "2.1.0"
}
```

**生成任务**:
```bash
./gradlew :dokkaGeneratePublicationHtml
```

### Javadoc 格式

标准 Javadoc 格式，兼容 Java 工具。

**特点**:
- 标准 Javadoc 布局
- 兼容 javadoc.io
- Maven Central 要求
- 与 Java 项目一致

**应用插件**:
```kotlin
plugins {
    id("org.jetbrains.dokka-javadoc") version "2.1.0"
}
```

**生成任务**:
```bash
./gradlew :dokkaGeneratePublicationJavadoc
```

**注意事项**:
> ⚠️ Javadoc 格式目前处于 Alpha 阶段，可能存在 bug 和迁移问题。使用风险自负。

### 同时生成两种格式

```kotlin
plugins {
    // HTML 格式
    id("org.jetbrains.dokka") version "2.1.0"
    
    // Javadoc 格式
    id("org.jetbrains.dokka-javadoc") version "2.1.0"
}
```

生成文档:
```bash
# 生成两种格式
./gradlew :dokkaGenerate

# 仅生成 HTML
./gradlew :dokkaGeneratePublicationHtml

# 仅生成 Javadoc
./gradlew :dokkaGeneratePublicationJavadoc
```

---

## 单项目配置

### 项目结构

#### 单平台项目

```
my-project/
├── build.gradle.kts
├── settings.gradle.kts
└── src/
    └── main/
        └── kotlin/
            └── HelloWorld.kt
```

#### 多平台项目

```
my-project/
├── build.gradle.kts
├── settings.gradle.kts
└── src/
    ├── commonMain/
    │   └── kotlin/
    │       └── Common.kt
    ├── jvmMain/
    │   └── kotlin/
    │       └── JvmUtils.kt
    ├── jsMain/
    │   └── kotlin/
    │       └── JsUtils.kt
    └── nativeMain/
        └── kotlin/
            └── NativeUtils.kt
```

### 基础配置

```kotlin
// build.gradle.kts
plugins {
    kotlin("jvm") version "2.3.10"
    id("org.jetbrains.dokka") version "2.1.0"
}

repositories {
    mavenCentral()
}

dokka {
    dokkaPublications.html {
        // 设置模块名称
        moduleName.set("MyProject")
        
        // 设置输出目录
        outputDirectory.set(layout.buildDirectory.dir("documentation/html"))
        
        // 包含 README.md
        includes.from("README.md")
    }

    dokkaSourceSets.main {
        // 配置源代码链接 (用于"查看源码"功能)
        sourceLink {
            localDirectory.set(file("src/main/kotlin"))
            remoteUrl.set(URI("https://github.com/your-org/your-repo"))
            remoteLineSuffix.set("#L")
        }
    }
}
```

### 高级配置

```kotlin
dokka {
    // 文档发布配置
    dokkaPublications.html {
        // 基本配置
        moduleName.set("MyLibrary")
        moduleVersion.set("1.0.0")
        outputDirectory.set(layout.buildDirectory.dir("dokka/html"))
        
        // 包含额外内容
        includes.from("README.md", "CHANGELOG.md")
        
        // 文档源文件
        sourceRoots.from(
            file("src/commonMain/kotlin"),
            file("src/jvmMain/kotlin")
        )
        
        // 自定义样式
        customStyleSheets.from(
            file("docs/styles/custom-styles.css")
        )
        
        // 自定义脚本
        customAssets.from(
            file("docs/assets/logo.png")
        )
        
        // 外部链接
        externalDocumentation {
            url.set(URI("https://kotlinlang.org/api/latest/jvm/stdlib/").toURL())
        }
    }
    
    // 源代码集配置
    dokkaSourceSets.configureEach {
        // 包含的包
        includes.from("packages.md")
        
        // 文档可见性
        documentedVisibilities.set(
            setOf(
                org.jetbrains.dokka.gradle.DokkaSourceSet.Visibility.PUBLIC,
                org.jetbrains.dokka.gradle.DokkaSourceSet.Visibility.PROTECTED
            )
        )
        
        // 抑制 (不生成文档)
        suppress.set(false)
        suppressGeneratedFiles.set(true)
        
        // 跳过空包
        skipEmptyPackages.set(true)
        
        // 跳过废弃 API
        skipDeprecated.set(false)
        
        // 报告未文档化的 API
        reportUndocumented.set(true)
        
        // JDK 版本
        jdkVersion.set(8)
        
        // 语言版本
        languageVersion.set("2.3")
        
        // 源代码链接
        sourceLink {
            localDirectory.set(file("src/main/kotlin"))
            remoteUrl.set(URI("https://github.com/your-org/your-repo").toURL())
            remoteLineSuffix.set("#L")
        }
        
        // 外部文档链接
        externalDocumentation {
            url.set(URI("https://kotlinlang.org/api/latest/jvm/stdlib/").toURL())
            packageListUrl.set(URI("https://kotlinlang.org/api/latest/jvm/stdlib/package-list").toURL())
        }
    }
}
```

### 配置示例

#### 配置模块名称和版本

```kotlin
dokka {
    dokkaPublications.html {
        moduleName.set("MyLibrary")
        moduleVersion.set("1.0.0")
    }
}
```

#### 包含 README

```kotlin
dokka {
    dokkaPublications.html {
        includes.from("README.md")
    }
}
```

#### 配置源代码链接

```kotlin
dokka {
    dokkaSourceSets.configureEach {
        sourceLink {
            localDirectory.set(file("src/main/kotlin"))
            remoteUrl.set(URI("https://github.com/your-org/your-repo").toURL())
            remoteLineSuffix.set("#L")
        }
    }
}
```

这会在文档中添加"Source"链接，指向 GitHub 上的源代码。

#### 自定义输出目录

```kotlin
dokka {
    dokkaPublications.html {
        outputDirectory.set(layout.buildDirectory.dir("docs/api"))
    }
}
```

#### 配置包文档

创建 `packages.md`:

```markdown
# Module MyLibrary

这里是模块的概述文档。

## Package com.example.utils

工具类包，包含各种实用函数。

## Package com.example.models

数据模型包，包含所有数据类。
```

配置:

```kotlin
dokka {
    dokkaSourceSets.configureEach {
        includes.from("packages.md")
    }
}
```

---

## 多项目配置

### 项目结构

```
my-multi-project/
├── build.gradle.kts
├── settings.gradle.kts
├── subproject-A/
│   ├── build.gradle.kts
│   └── src/main/kotlin/...
├── subproject-B/
│   ├── build.gradle.kts
│   └── src/main/kotlin/...
└── subproject-C/
    ├── build.gradle.kts
    └── src/main/kotlin/...
```

### 配置方式

有两种方式配置多项目构建:

1. **约定插件** (推荐): 集中配置，避免重复
2. **手动配置**: 在每个子项目中重复配置

### 方式 1: 约定插件 (推荐)

#### 步骤 1: 设置 buildSrc

创建 `buildSrc` 目录:

```
buildSrc/
├── settings.gradle.kts
├── build.gradle.kts
└── src/main/kotlin/
    └── dokka-convention.gradle.kts
```

**buildSrc/settings.gradle.kts**:
```kotlin
rootProject.name = "buildSrc"
```

**buildSrc/build.gradle.kts**:
```kotlin
plugins {
    `kotlin-dsl`
}

repositories {
    mavenCentral()
    gradlePluginPortal()
}

dependencies {
    implementation("org.jetbrains.dokka:dokka-gradle-plugin:2.1.0")
}
```

#### 步骤 2: 创建约定插件

**buildSrc/src/main/kotlin/dokka-convention.gradle.kts**:
```kotlin
plugins {
    id("org.jetbrains.dokka")
}

dokka {
    // 共享配置
    dokkaPublications.html {
        moduleName.set("${project.name}")
        outputDirectory.set(layout.buildDirectory.dir("dokka/html"))
        includes.from("README.md")
    }
    
    dokkaSourceSets.configureEach {
        // 通用配置
        documentedVisibilities.set(
            setOf(
                org.jetbrains.dokka.gradle.DokkaSourceSet.Visibility.PUBLIC
            )
        )
        
        sourceLink {
            localDirectory.set(file("src/main/kotlin"))
            remoteUrl.set(URI("https://github.com/your-org/your-repo").toURL())
            remoteLineSuffix.set("#L")
        }
    }
}
```

#### 步骤 3: 应用约定插件到子项目

**subproject-A/build.gradle.kts**:
```kotlin
plugins {
    kotlin("jvm")
    id("dokka-convention")  // 应用约定插件
}
```

**subproject-B/build.gradle.kts**:
```kotlin
plugins {
    kotlin("jvm")
    id("dokka-convention")
}
```

### 方式 2: 手动配置

在每个子项目中重复配置:

**subproject-A/build.gradle.kts**:
```kotlin
plugins {
    kotlin("jvm")
    id("org.jetbrains.dokka") version "2.1.0"
}

dokka {
    dokkaPublications.html {
        moduleName.set("subproject-A")
        outputDirectory.set(layout.buildDirectory.dir("dokka/html"))
    }
    
    dokkaSourceSets.configureEach {
        // 配置...
    }
}
```

**subproject-B/build.gradle.kts**:
```kotlin
// 相同的配置
```

### 聚合文档

在多项目构建中，可以聚合所有子项目的文档到单一输出。

#### 配置聚合项目

创建一个专门的聚合项目或使用根项目:

**build.gradle.kts (聚合项目)**:
```kotlin
plugins {
    id("org.jetbrains.dokka") version "2.1.0"
}

dependencies {
    // 依赖所有需要聚合文档的子项目
    dokka(project(":subproject-A"))
    dokka(project(":subproject-B"))
    dokka(project(":subproject-C"))
}
```

#### 聚合输出目录结构

聚合后的文档保持项目结构:

```
build/dokka/html/
├── index.html
├── subproject-A/
│   ├── index.html
│   └── ...
├── subproject-B/
│   ├── index.html
│   └── ...
└── subproject-C/
    ├── index.html
    └── ...
```

#### 自定义子项目目录

默认情况下，Dokka 保留完整的项目路径。可以自定义:

**subproject-C/build.gradle.kts**:
```kotlin
dokka {
    dokkaPublications.html {
        outputDirectory.set(
            layout.buildDirectory.dir("dokka/html/custom-name")
        )
    }
}
```

---

## 高级配置

### 文档可见性

控制哪些可见性的 API 会生成文档:

```kotlin
dokka {
    dokkaSourceSets.configureEach {
        documentedVisibilities.set(
            setOf(
                DokkaSourceSet.Visibility.PUBLIC,
                DokkaSourceSet.Visibility.PROTECTED,
                DokkaSourceSet.Visibility.INTERNAL,
                DokkaSourceSet.Visibility.PRIVATE
            )
        )
    }
}
```

### 包含/排除包

```kotlin
dokka {
    dokkaSourceSets.configureEach {
        // 包含的包
        includes.from("packages.md")
        
        // 排除的包
        perPackageOption {
            matchingRegex.set(".*\\.internal.*")
            suppress.set(true)
        }
        
        // 多个包选项
        perPackageOption {
            matchingRegex.set(".*\\.api.*")
            suppress.set(false)
        }
    }
}
```

### 外部文档链接

链接到外部文档 (如 Kotlin 标准库):

```kotlin
dokka {
    dokkaSourceSets.configureEach {
        externalDocumentation {
            // Kotlin 标准库
            url.set(URI("https://kotlinlang.org/api/latest/jvm/stdlib/").toURL())
            packageListUrl.set(URI("https://kotlinlang.org/api/latest/jvm/stdlib/package-list").toURL())
            
            // 其他库
            url.set(URI("https://kotlinlang.org/api/kotlinx.coroutines/").toURL())
        }
        
        // 多个外部链接
        externalDocumentation {
            url.set(URI("https://kotlinlang.org/api/latest/jvm/stdlib/").toURL())
        }
        externalDocumentation {
            url.set(URI("https://kotlinlang.org/api/kotlinx.coroutines/").toURL())
        }
    }
}
```

### 自定义样式和资源

```kotlin
dokka {
    dokkaPublications.html {
        // 自定义样式表
        customStyleSheets.from(
            file("docs/styles/custom-styles.css"),
            file("docs/styles/colors.css")
        )
        
        // 自定义资源 (logo、图标等)
        customAssets.from(
            file("docs/assets/logo.png"),
            file("docs/assets/favicon.ico")
        )
        
        // 自定义模板
        templatesDir.set(file("docs/templates"))
    }
}
```

### 自定义 CSS 示例

```css
/* docs/styles/custom-styles.css */

/* 修改主色调 */
:root {
    --color-brand: #7F52FF;
    --color-brand-dark: #6633CC;
}

/* 修改导航栏 */
.sidebar {
    background: linear-gradient(135deg, #7F52FF 0%, #6633CC 100%);
}

/* 修改代码块 */
.code-block {
    border-radius: 8px;
    background: #1E1E1E;
}

/* 修改字体 */
body {
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
}
```

### 报告未文档化的 API

```kotlin
dokka {
    dokkaSourceSets.configureEach {
        // 启用未文档化 API 报告
        reportUndocumented.set(true)
        
        // 跳过废弃 API
        skipDeprecated.set(false)
        
        // 跳过空包
        skipEmptyPackages.set(true)
    }
}
```

### KDoc 标签

Dokka 支持标准 KDoc 标签:

```kotlin
/**
 * 计算两个数的和。
 * 
 * @param a 第一个加数
 * @param b 第二个加数
 * @return 两个数的和
 * @throws IllegalArgumentException 如果数字超出范围
 * 
 * @see subtract
 * @sample com.example.samples.addSample
 */
fun add(a: Int, b: Int): Int {
    return a + b
}
```

支持的标签:
- `@param` - 参数说明
- `@return` - 返回值说明
- `@throws` / `@exception` - 异常说明
- `@receiver` - 接收者说明
- `@sample` - 示例代码
- `@see` - 参考链接
- `@since` - 版本信息
- `@author` - 作者信息
- `@suppress` - 抑制文档生成

---

## 发布 Javadoc JAR

### 为什么需要 Javadoc JAR

发布到 Maven Central 等仓库时，通常需要提供:
- 源代码 JAR (`-sources.jar`)
- 文档 JAR (`-javadoc.jar`)

### 创建 Javadoc JAR 任务

Dokka 不直接提供创建 JAR 的任务，但可以通过自定义 Gradle 任务实现:

```kotlin
// build.gradle.kts

// HTML 文档 JAR
val dokkaHtmlJar by tasks.registering(Jar::class) {
    description = "A HTML Documentation JAR containing Dokka HTML"
    
    // 从 Dokka HTML 输出创建 JAR
    from(tasks.dokkaGeneratePublicationHtml.flatMap { it.outputDirectory })
    
    // 设置分类器
    archiveClassifier.set("html-doc")
    
    // 设置归档名称
    archiveBaseName.set("${project.name}")
}

// Javadoc 文档 JAR
val dokkaJavadocJar by tasks.registering(Jar::class) {
    description = "A Javadoc JAR containing Dokka Javadoc"
    
    // 从 Dokka Javadoc 输出创建 JAR
    from(tasks.dokkaGeneratePublicationJavadoc.flatMap { it.outputDirectory })
    
    // 设置分类器 (Maven Central 要求使用 javadoc)
    archiveClassifier.set("javadoc")
    
    // 设置归档名称
    archiveBaseName.set("${project.name}")
}
```

### 配置发布

```kotlin
// build.gradle.kts
plugins {
    `maven-publish`
}

publishing {
    publications {
        create<MavenPublication>("maven") {
            groupId = "com.example"
            artifactId = "my-library"
            version = "1.0.0"
            
            // 主构件
            from(components["java"])
            
            // 源代码 JAR
            artifact(tasks.named("sourcesJar"))
            
            // 文档 JAR
            artifact(tasks.named("dokkaJavadocJar"))
            // 或
            artifact(tasks.named("dokkaHtmlJar"))
        }
    }
    
    repositories {
        maven {
            name = "OSSRH"
            url = uri("https://s01.oss.sonatype.org/service/local/staging/deploy/maven2/")
            credentials {
                username = findProperty("ossrhUsername") as String?
                password = findProperty("ossrhPassword") as String?
            }
        }
    }
}
```

### 完整示例

```kotlin
// build.gradle.kts
plugins {
    `java-library`
    `maven-publish`
    signing
    id("org.jetbrains.dokka") version "2.1.0"
    id("org.jetbrains.dokka-javadoc") version "2.1.0"
}

// 创建源代码 JAR
val sourcesJar by tasks.registering(Jar::class) {
    archiveClassifier.set("sources")
    from(sourceSets.main.get().allSource)
}

// 创建 Javadoc JAR
val dokkaJavadocJar by tasks.registering(Jar::class) {
    description = "A Javadoc JAR containing Dokka Javadoc"
    from(tasks.dokkaGeneratePublicationJavadoc.flatMap { it.outputDirectory })
    archiveClassifier.set("javadoc")
}

publishing {
    publications {
        create<MavenPublication>("mavenJava") {
            from(components["java"])
            
            artifact(sourcesJar)
            artifact(dokkaJavadocJar)
            
            pom {
                name.set("My Library")
                description.set("My awesome Kotlin library")
                url.set("https://github.com/your-org/your-repo")
                
                licenses {
                    license {
                        name.set("MIT License")
                        url.set("https://opensource.org/licenses/MIT")
                    }
                }
                
                developers {
                    developer {
                        id.set("your-id")
                        name.set("Your Name")
                        email.set("your.email@example.com")
                    }
                }
                
                scm {
                    connection.set("scm:git:git://github.com/your-org/your-repo.git")
                    developerConnection.set("scm:git:ssh://github.com/your-org/your-repo.git")
                    url.set("https://github.com/your-org/your-repo")
                }
            }
        }
    }
    
    repositories {
        maven {
            name = "OSSRH"
            url = uri("https://s01.oss.sonatype.org/service/local/staging/deploy/maven2/")
            credentials {
                username = findProperty("ossrhUsername") as String?
                password = findProperty("ossrhPassword") as String?
            }
        }
    }
}

// 签名 (Maven Central 要求)
signing {
    sign(publishing.publications["mavenJava"])
}
```

### 使用 javadoc.io

[javadoc.io](https://www.javadoc.io/) 可以免费托管 Maven Central 库的文档:

1. 发布包含 `javadoc.jar` 到 Maven Central
2. javadoc.io 自动抓取并托管文档
3. 访问：`https://www.javadoc.io/doc/com.example/my-library`

**优点**:
- 免费托管
- 无需额外配置
- 支持 HTML 格式
- 自动更新

---

## Dokka 插件

Dokka 具有高度可扩展性，支持社区开发的插件。

### 官方插件

- **dokka-base**: 基础插件
- **dokka-javadoc**: Javadoc 输出格式
- **dokka-html**: HTML 输出格式
- **dokka-javadoc-plugin**: Javadoc 增强

### 社区插件

#### Kotlinx Coroutines

```kotlin
dependencies {
    dokkaPlugin("org.jetbrains.dokka:kotlinx-coroutines-plugin:2.1.0")
}
```

#### Android

```kotlin
dependencies {
    dokkaPlugin("org.jetbrains.dokka:android-documentation-plugin:2.1.0")
}
```

#### 自定义插件

创建自定义 Dokka 插件:

```kotlin
// build.gradle.kts
plugins {
    kotlin("jvm")
    `maven-publish`
}

dependencies {
    compileOnly("org.jetbrains.dokka:dokka-core:2.1.0")
}

// 插件实现
class CustomPlugin : DokkaPlugin() {
    val customExtension by extending {
        pluginConfiguration<CustomExtension, CustomExtension.Configuration> {
            enabled.set(true)
        }
    }
}
```

### 使用插件

```kotlin
// build.gradle.kts
dependencies {
    dokkaPlugin("com.example:custom-plugin:1.0.0")
}

dokka {
    pluginsConfiguration {
        // 插件配置
    }
}
```

---

## 最佳实践

### 1. 使用 KDoc 注释

```kotlin
/**
 * 用户数据类。
 * 
 * @property id 用户唯一标识符
 * @property name 用户名称
 * @property email 用户邮箱地址
 * @property createdAt 用户创建时间
 * 
 * @see UserRepository
 * @sample com.example.samples.userSample
 */
data class User(
    val id: String,
    val name: String,
    val email: String,
    val createdAt: Instant = Clock.System.now()
)
```

### 2. 提供示例代码

```kotlin
/**
 * 将字符串转换为驼峰命名。
 * 
 * @receiver 要转换的字符串
 * @return 驼峰命名字符串
 * 
 * @sample com.example.samples.toCamelCaseSample
 */
fun String.toCamelCase(): String {
    return this.split("_").mapIndexed { index, s ->
        if (index == 0) s else s.replaceFirstChar { it.uppercase() }
    }.joinToString("")
}

// 示例代码
/**
 * Sample:
 * ```kotlin
 * val result = "hello_world".toCamelCase()
 * println(result) // 输出：helloWorld
 * ```
 */
```

### 3. 包含 README

```kotlin
dokka {
    dokkaPublications.html {
        includes.from("README.md")
    }
}
```

**README.md** 内容:
```markdown
# MyLibrary

这里是库的总体介绍。

## 安装

```kotlin
dependencies {
    implementation("com.example:mylibrary:1.0.0")
}
```

## 快速开始

```kotlin
val result = MyLibrary.doSomething()
```

## 功能特性

- 功能 1
- 功能 2
- 功能 3
```

### 4. 配置源代码链接

```kotlin
dokka {
    dokkaSourceSets.configureEach {
        sourceLink {
            localDirectory.set(file("src/main/kotlin"))
            remoteUrl.set(URI("https://github.com/your-org/your-repo").toURL())
            remoteLineSuffix.set("#L")
        }
    }
}
```

### 5. 使用约定插件

对于多项目:

```kotlin
// buildSrc/src/main/kotlin/dokka-convention.gradle.kts
plugins {
    id("org.jetbrains.dokka")
}

dokka {
    dokkaPublications.html {
        moduleName.set("${project.name}")
        outputDirectory.set(layout.buildDirectory.dir("dokka/html"))
        includes.from("README.md")
    }
}
```

### 6. 排除内部 API

```kotlin
dokka {
    dokkaSourceSets.configureEach {
        perPackageOption {
            matchingRegex.set(".*\\.internal.*")
            suppress.set(true)
        }
    }
}
```

### 7. 报告未文档化的 API

```kotlin
dokka {
    dokkaSourceSets.configureEach {
        reportUndocumented.set(true)
    }
}
```

### 8. 启用构建缓存

```properties
# gradle.properties
org.gradle.caching=true
org.gradle.configuration-cache=true
```

### 9. 文档版本控制

```kotlin
dokka {
    dokkaPublications.html {
        moduleName.set("MyLibrary")
        moduleVersion.set(project.version.toString())
    }
}
```

### 10. 自动化文档发布

```kotlin
// build.gradle.kts
tasks.register("publishDocumentation") {
    dependsOn("dokkaGeneratePublicationHtml")
    
    doLast {
        // 发布到 GitHub Pages、Netlify 等
        // 例如：git push 到 gh-pages 分支
    }
}
```

---

## 常见问题

### 1. 文档不生成

**问题**: 运行 `dokkaGenerate` 后没有生成文档

**解决方案**:
```kotlin
// 检查是否应用了插件
plugins {
    id("org.jetbrains.dokka") version "2.1.0"
}

// 检查 Gradle 版本
// Gradle >= 7.6

// 检查 Kotlin 插件
plugins {
    kotlin("jvm") version "2.3.10"
}
```

### 2. 多项目文档重复

**问题**: 多项目构建中生成重复文档

**解决方案**:
```kotlin
// 使用聚合项目
dependencies {
    dokka(project(":subproject-A"))
    dokka(project(":subproject-B"))
}

// 或在子项目中抑制
dokka {
    dokkaSourceSets.configureEach {
        suppress.set(true)
    }
}
```

### 3. 自定义样式不生效

**问题**: 自定义 CSS 没有应用

**解决方案**:
```kotlin
dokka {
    dokkaPublications.html {
        customStyleSheets.from(
            file("docs/styles/custom.css")
        )
    }
}

// 确保文件路径正确
// 检查 CSS 语法
```

### 4. 源代码链接错误

**问题**: "Source" 链接指向错误的 URL

**解决方案**:
```kotlin
sourceLink {
    localDirectory.set(file("src/main/kotlin"))
    remoteUrl.set(URI("https://github.com/your-org/your-repo").toURL())
    remoteLineSuffix.set("#L")  // GitHub 使用 #L, GitLab 使用 #L
}
```

### 5. 外部链接不工作

**问题**: 外部库文档链接失效

**解决方案**:
```kotlin
externalDocumentation {
    url.set(URI("https://kotlinlang.org/api/latest/jvm/stdlib/").toURL())
    packageListUrl.set(URI("https://kotlinlang.org/api/latest/jvm/stdlib/package-list").toURL())
}
```

### 6. Javadoc JAR 为空

**问题**: 创建的 Javadoc JAR 为空

**解决方案**:
```kotlin
val dokkaJavadocJar by tasks.registering(Jar::class) {
    // 确保依赖 dokkaGeneratePublicationJavadoc
    dependsOn(tasks.dokkaGeneratePublicationJavadoc)
    
    from(tasks.dokkaGeneratePublicationJavadoc.flatMap { it.outputDirectory })
    archiveClassifier.set("javadoc")
}
```

### 7. 构建性能慢

**问题**: Dokka 生成文档很慢

**解决方案**:
```properties
# gradle.properties
# 启用构建缓存
org.gradle.caching=true

# 启用配置缓存
org.gradle.configuration-cache=true

# 并行构建
org.gradle.parallel=true
```

```kotlin
// 跳过不需要的包
dokka {
    dokkaSourceSets.configureEach {
        perPackageOption {
            matchingRegex.set(".*\\.internal.*")
            suppress.set(true)
        }
        
        skipEmptyPackages.set(true)
    }
}
```

### 8. KDoc 不显示

**问题**: KDoc 注释没有出现在文档中

**解决方案**:
```kotlin
// 确保使用正确的 KDoc 格式
/**
 * 这是 KDoc 注释
 * 
 * @param name 参数说明
 * @return 返回值说明
 */
fun foo(name: String): Int { }

// 检查可见性
dokka {
    dokkaSourceSets.configureEach {
        documentedVisibilities.set(
            setOf(DokkaSourceSet.Visibility.PUBLIC)
        )
    }
}
```

### 9. 多平台文档不完整

**问题**: Kotlin Multiplatform 项目文档不完整

**解决方案**:
```kotlin
// 为每个源集配置 Dokka
dokka {
    dokkaSourceSets {
        configureEach {
            // 通用配置
        }
        
        named("commonMain") {
            // commonMain 特定配置
        }
        
        named("jvmMain") {
            // jvmMain 特定配置
        }
    }
}
```

### 10. 从 v1 升级到 v2

**问题**: 从 Dokka Gradle Plugin v1 升级到 v2

**解决方案**:

v1 语法:
```kotlin
// ❌ 旧版本
tasks.dokkaHtml.configure {
    outputDirectory.set(buildDir.resolve("dokka"))
}
```

v2 语法:
```kotlin
// ✅ 新版本
dokka {
    dokkaPublications.html {
        outputDirectory.set(layout.buildDirectory.dir("dokka/html"))
    }
}
```

参考官方迁移指南: https://kotlinlang.org/docs/dokka-migration.html

---

## 参考资源

### 官方文档

- [Dokka 官方文档](https://kotlinlang.org/docs/dokka-get-started.html)
- [Dokka GitHub](https://github.com/Kotlin/dokka)
- [Dokka 示例项目](https://github.com/Kotlin/dokka/tree/master/examples)
- [迁移指南](https://kotlinlang.org/docs/dokka-migration.html)

### 相关资源

- [KDoc 语法](https://kotlinlang.org/docs/kotlin-doc.html)
- [Gradle 构建缓存](https://docs.gradle.org/current/userguide/build_cache.html)
- [Gradle 配置缓存](https://docs.gradle.org/current/userguide/configuration_cache.html)
- [Maven Central 发布指南](https://central.sonatype.org/publish/publish-guide/)

### 示例项目

- [Dokka 示例](https://github.com/Kotlin/dokka/tree/master/examples)
- [Kotlinx Coroutines 文档](https://kotlinlang.org/api/kotlinx.coroutines/)
- [Kotlin 标准库文档](https://kotlinlang.org/api/latest/jvm/stdlib/)

---

## 总结

Dokka 是 Kotlin 项目的标准文档生成工具，具有:

1. **Kotlin 原生支持**: 完美支持 Kotlin 语言特性
2. **多平台**: 支持 Kotlin Multiplatform 项目
3. **多种格式**: HTML、Javadoc 输出
4. **高度可扩展**: 插件系统和自定义主题
5. **Gradle 集成**: 与 Gradle 无缝集成
6. **构建缓存**: 支持 Gradle 构建缓存
7. **多项目聚合**: 可聚合多个子项目文档

**推荐使用场景**:
- Kotlin 库和项目文档
- Kotlin Multiplatform 跨平台项目
- Maven Central 发布
- 团队内部 API 文档
- 开源项目文档

对于 Kotlin 项目，Dokka 是生成 API 文档的标准选择。

---

## 更新记录

| 日期 | 版本 | 描述 |
|------|------|------|
| 2026-03-07 | 1.0 | 初始版本，整理 Dokka 文档生成工具文档 |
