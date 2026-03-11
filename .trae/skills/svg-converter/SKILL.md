---
name: "svg-converter"
description: "将SVG图像转换为PNG、ICO等带透明通道的位图格式。Invoke when user needs to convert SVG icons to PNG/ICO for platforms that don't support SVG, or when preparing cross-platform app resources."
---

# SVG 图像转换器

本技能用于将SVG矢量图像转换为带透明通道的PNG、ICO等位图格式，以支持在不支持SVG的平台上使用。

## 项目设计决策

本项目采用**预转换策略**：
- 每个SVG图标在资源提交时就预转换为多分辨率PNG
- 预转换的PNG纳入版本控制
- 运行时直接使用PNG，无性能损耗
- 构建时无需转换，加快构建速度

## 预转换规格

每个SVG图标预转换为以下尺寸的PNG（带透明通道）：
- 16x16 - 工具栏小图标
- 24x24 - 标准图标（默认）
- 32x32 - 工具栏大图标
- 48x48 - 列表/菜单图标
- 64x64 - 侧边栏图标
- 128x128 - 大图标展示
- 256x256 - 高分辨率显示
- 512x512 - 应用图标/启动图标

## 目录结构

```
resources/icons/
├── svg/                    # SVG源文件（版本控制）
│   └── navigation/
│       └── home.svg
└── png/                    # 预转换PNG（版本控制）
    └── navigation/
        └── home/
            ├── 16x16.png
            ├── 24x24.png
            ├── 32x32.png
            ├── 48x48.png
            ├── 64x64.png
            ├── 128x128.png
            ├── 256x256.png
            └── 512x512.png
```

## 技术方案

### 推荐库

在Kotlin Multiplatform项目中，推荐使用以下方案进行SVG转换：

1. **Apache Batik** (推荐用于构建时转换)
   - Apache开源项目
   - 功能完整的SVG处理库
   - 支持转换多种格式
   - 适合Gradle构建任务使用

2. **SVG Salamander** (Java库)
   - 纯Java实现，可在Kotlin中使用
   - 支持SVG渲染为BufferedImage
   - 可导出为PNG

3. **Kotlin Native方案**
   - 使用librsvg (C库) 通过CInterop调用
   - 或使用Skia库进行渲染

## Gradle预转换任务

在 `build.gradle.kts` 中添加预转换任务：

```kotlin
// build.gradle.kts
import java.io.File

plugins {
    kotlin("multiplatform")
}

// SVG预转换配置
val svgSourceDir = file("src/commonMain/resources/icons/svg")
val pngOutputDir = file("src/commonMain/resources/icons/png")
val pngSizes = listOf(16, 24, 32, 48, 64, 128, 256, 512)

/**
 * 预转换所有SVG图标任务
 */
tasks.register<JavaExec>("convertSvgIcons") {
    group = "resource processing"
    description = "预转换所有SVG图标为多分辨率PNG"
    
    classpath = configurations["runtimeClasspath"]
    mainClass.set("com.example.build.SvgIconConverter")
    
    args = listOf(
        svgSourceDir.absolutePath,
        pngOutputDir.absolutePath,
        pngSizes.joinToString(",")
    )
    
    // 增量构建支持
    inputs.dir(svgSourceDir)
    outputs.dir(pngOutputDir)
}

/**
 * 在资源处理前执行转换
 */
tasks.named("processResources") {
    dependsOn("convertSvgIcons")
}
```

## 图标转换工具类

```kotlin
// buildSrc/src/main/kotlin/com/example/build/SvgIconConverter.kt
package com.example.build

import org.apache.batik.transcoder.TranscoderInput
import org.apache.batik.transcoder.TranscoderOutput
import org.apache.batik.transcoder.image.PNGTranscoder
import java.io.File
import java.io.FileInputStream
import java.io.FileOutputStream

/**
 * SVG图标预转换工具
 * 在构建时将SVG转换为多分辨率PNG
 */
object SvgIconConverter {
    
    @JvmStatic
    fun main(args: Array<String>) {
        if (args.size < 3) {
            println("用法: SvgIconConverter <svg目录> <png输出目录> <尺寸列表>")
            return
        }
        
        val svgDir = File(args[0])
        val pngDir = File(args[1])
        val sizes = args[2].split(",").map { it.toInt() }
        
        if (!svgDir.exists()) {
            println("SVG目录不存在: ${svgDir.absolutePath}")
            return
        }
        
        println("开始转换SVG图标...")
        convertAllSvgs(svgDir, pngDir, sizes)
        println("转换完成!")
    }
    
    private fun convertAllSvgs(svgDir: File, pngDir: File, sizes: List<Int>) {
        svgDir.walkTopDown()
            .filter { it.isFile && it.extension.lowercase() == "svg" }
            .forEach { svgFile ->
                val relativePath = svgFile.relativeTo(svgDir).parent ?: ""
                val iconName = svgFile.nameWithoutExtension
                
                sizes.forEach { size ->
                    val outputDir = File(pngDir, relativePath).resolve(iconName)
                    outputDir.mkdirs()
                    
                    val outputFile = File(outputDir, "${size}x${size}.png")
                    
                    // 增量转换：只转换更新的文件
                    if (needsConversion(svgFile, outputFile)) {
                        convertSvgToPng(svgFile, outputFile, size, size)
                        println("✓ ${svgFile.name} -> ${size}x${size}.png")
                    }
                }
            }
    }
    
    private fun needsConversion(svgFile: File, pngFile: File): Boolean {
        if (!pngFile.exists()) return true
        return svgFile.lastModified() > pngFile.lastModified()
    }
    
    private fun convertSvgToPng(
        svgFile: File,
        outputFile: File,
        width: Int,
        height: Int
    ) {
        val transcoder = PNGTranscoder()
        
        transcoder.addTranscodingHint(
            PNGTranscoder.KEY_WIDTH, width.toFloat()
        )
        transcoder.addTranscodingHint(
            PNGTranscoder.KEY_HEIGHT, height.toFloat()
        )
        transcoder.addTranscodingHint(
            PNGTranscoder.KEY_ANTIALIASING, true
        )
        
        FileInputStream(svgFile).use { inputStream ->
            FileOutputStream(outputFile).use { outputStream ->
                val input = TranscoderInput(inputStream)
                val output = TranscoderOutput(outputStream)
                transcoder.transcode(input, output)
            }
        }
    }
}
```

## 运行时图标组件

```kotlin
/**
 * 多色SVG图标组件
 * 保留SVG原始颜色，不使用tint
 */
@Composable
fun MultiColorSvgIcon(
    name: String,
    modifier: Modifier = Modifier,
    size: Dp = 24.dp
) {
    val painter = rememberSvgPainter(
        resourcePath = "icons/svg/$name.svg"
    )
    
    Image(
        painter = painter,
        contentDescription = name,
        modifier = modifier.size(size)
    )
}

/**
 * 预转换PNG图标（用于不支持SVG的平台）
 */
@Composable
fun PngIcon(
    name: String,
    modifier: Modifier = Modifier,
    size: Dp = 24.dp
) {
    val sizePx = with(LocalDensity.current) { size.roundToPx() }
    val pngSize = selectOptimalPngSize(sizePx)
    
    val painter = rememberAsyncImagePainter(
        model = "icons/png/$name/${pngSize}x${pngSize}.png"
    )
    
    Image(
        painter = painter,
        contentDescription = name,
        modifier = modifier.size(size)
    )
}

private fun selectOptimalPngSize(requestedSize: Int): Int {
    val availableSizes = listOf(16, 24, 32, 48, 64, 128, 256, 512)
    return availableSizes.find { it >= requestedSize } ?: availableSizes.last()
}
```

## 依赖配置

```kotlin
// buildSrc/build.gradle.kts
plugins {
    kotlin("jvm")
}

repositories {
    mavenCentral()
}

dependencies {
    implementation("org.apache.xmlgraphics:batik-transcoder:1.17")
    implementation("org.apache.xmlgraphics:batik-codec:1.17")
}
```

## 输出规范

- PNG格式：保留Alpha透明通道
- ICO格式：包含多尺寸（16x16, 32x32, 48x48, 256x256）
- 输出质量：抗锯齿渲染，确保清晰度
- 多色图标：保留SVG原始颜色，不使用单色tint
