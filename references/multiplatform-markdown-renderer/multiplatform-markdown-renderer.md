# Multiplatform Markdown Renderer 参考指南

## 概述

**Multiplatform Markdown Renderer** 是一个强大的 Kotlin Multiplatform Markdown 渲染库，使用 Compose Multiplatform 构建。它支持在 Android、iOS、Desktop 和 Web 平台上渲染 Markdown 内容。

**项目地址**: https://github.com/mikepenz/multiplatform-markdown-renderer

**许可证**: Apache License, Version 2.0

---

## 特性

### 核心功能

- **跨平台 Markdown 渲染** - 支持 Android、iOS、Desktop 和 Web
- **Material Design 集成** - 无缝集成 Material 2 和 Material 3 主题
- **丰富的 Markdown 支持** - 渲染标题、列表、代码块、表格、图片等
- **语法高亮** - 支持多种编程语言的代码语法高亮
- **图片加载** - 灵活的图片加载，支持 Coil2 和 Coil3 集成
- **自定义选项** - 丰富的颜色、排版、组件等自定义选项
- **性能优化** - 支持懒加载，高效渲染大型文档
- **扩展文本跨度** - 支持高级文本样式
- **轻量级** - 最小依赖，性能优化

---

## 安装配置

### Gradle 依赖配置

#### 多平台项目

对于多平台项目，在 `build.gradle.kts` 中添加：

```kotlin
dependencies {
    // 核心库
    implementation("com.mikepenz:multiplatform-markdown-renderer:${version}")

    // 根据 Material 主题选择以下其中一个：

    // Material 2 主题应用
    implementation("com.mikepenz:multiplatform-markdown-renderer-m2:${version}")

    // 或 Material 3 主题应用
    implementation("com.mikepenz:multiplatform-markdown-renderer-m3:${version}")
}
```

#### JVM (Desktop) 项目

对于纯 JVM 项目：

```kotlin
dependencies {
    implementation("com.mikepenz:multiplatform-markdown-renderer-jvm:${version}")
}
```

#### Android 项目

对于纯 Android 项目：

```kotlin
dependencies {
    implementation("com.mikepenz:multiplatform-markdown-renderer-android:${version}")
}
```

### 重要说明

**从版本 0.13.0 开始**，核心库不再依赖 Material 主题。必须包含 `-m2` 或 `-m3` 模块才能获得默认样式。

---

## 快速开始

### 基础使用

最简单的使用方式是将 Markdown 字符串传递给 `Markdown` composable：

```kotlin
// 在 composable 中 (使用适合主题的 Markdown 实现)
import com.mikepenz.markdown.m3.Markdown  // Material 3
// 或
import com.mikepenz.markdown.m2.Markdown  // Material 2

Markdown(
    """
    # Hello Markdown

    This is a simple markdown example with:

    - Bullet points
    - **Bold text**
    - *Italic text*

    [Check out this link](https://github.com/mikepenz/multiplatform-markdown-renderer)
    """.trimIndent()
)
```

### 注意

- 导入 `com.mikepenz.markdown.m3.Markdown` (Material 3) 或 `com.mikepenz.markdown.m2.Markdown` (Material 2)
- 默认情况下，当 Markdown 内容变化时，组件会显示加载状态
- 设置 `retainState = true` 可在更新时保持之前内容可见

---

## 高级用法

### 使用 rememberMarkdownState

为了更好的性能（特别是处理大型 Markdown 内容），使用 `rememberMarkdownState` 或将解析移到 ViewModel 中：

```kotlin
val markdownState = rememberMarkdownState(markdown)
Markdown(markdownState)
```

### 异步解析

**从版本 0.33.0 开始**，Markdown 内容默认异步解析，解析前会显示加载状态。

```kotlin
// 默认异步解析（推荐）
val markdownState = rememberMarkdownState(markdown)

// 强制立即解析（不推荐，可能阻塞 UI）
val markdownState = rememberMarkdownState(markdown, immediate = true)
```

### 保持状态避免闪烁

当内容变化时，使用 `retainState` 参数保持之前的渲染内容可见：

```kotlin
// 更新时保持之前内容可见（避免显示加载状态）
val markdownState = rememberMarkdownState(
    markdown,
    retainState = true
)
Markdown(markdownState)

// 动态内容加载
val markdownState = rememberMarkdownState(
    key, // 触发重新解析的 key
    retainState = true
) {
    "# Dynamic content $counter"
}
Markdown(markdownState)
```

这在内容频繁更新或希望避免新旧内容之间闪烁时特别有用。

### 大型文档的懒加载

**从版本 0.33.0 开始**，库支持使用 `LazyColumn` 替代 `Column` 来高效渲染大型 Markdown 文档：

```kotlin
Markdown(
    markdownState = markdownState,
    success = { state, components, modifier ->
        LazyMarkdownSuccess(
            state, 
            components, 
            modifier, 
            contentPadding = PaddingValues(16.dp)
        )
    },
    modifier = Modifier.fillMaxSize()
)
```

### 在 ViewModel 中解析

在 ViewModel 中解析 Markdown 可以保持滚动位置，即使导航离开后再返回：

```kotlin
// 在 ViewModel 中设置解析 Markdown 的 flow
val markdownFlow = parseMarkdownFlow("# Markdown")
    .stateIn(lifecycleScope, SharingStarted.Eagerly, State.Loading())

// 在 Composable 中使用 flow
val state by markdownFlow.collectAsStateWithLifecycle()
Markdown(state)
```

---

## 自定义样式

### 提供自定义样式

库允许修改渲染 Markdown 时的不同行为：

```kotlin
Markdown(
    content,
    colors = markdownColor(text = Color.Red),
    typography = markdownTypography(
        h1 = MaterialTheme.typography.body1
    )
)
```

### 禁用动画

默认情况下，MarkdownText 会动画化大小变化（如果图片加载）：

```kotlin
Markdown(
    content,
    animations = markdownAnimations(
        animateTextSize = {
            this
            /** 无动画 */
        }
    ),
)
```

---

## 扩展跨度 (Extended Spans)

**从版本 0.16.0 开始**，库支持 extended-spans（由 Saket Narayan 开发）。

### 启用扩展跨度

```kotlin
Markdown(
    content,
    extendedSpans = markdownExtendedSpans {
        val animator = rememberSquigglyUnderlineAnimator()
        remember {
            ExtendedSpans(
                RoundedCornerSpanPainter(),
                SquigglyUnderlineSpanPainter(animator = animator)
            )
        }
    }
)
```

---

## 自定义注解处理

库已经处理了大量不同的 token，但可以通过自定义注解器扩展：

```kotlin
Markdown(
    content,
    annotator = markdownAnnotator { content, child ->
        if (child.type == GFMElementTypes.STRIKETHROUGH) {
            append("Replaced you :)")
            true // 返回 true 以消耗此 ASTNode child
        } else false
    }
)
```

---

## 列表顺序调整

### 使用原始 Markdown 符号

```kotlin
// 使用原始 Markdown 中的无序列表符号
CompositionLocalProvider(LocalBulletListHandler provides { 
    type, bullet, index, listNumber, depth -> "$bullet " 
}) {
    Markdown(content)
}

// 将有序列表符号替换为 `A.)`
CompositionLocalProvider(LocalOrderedListHandler provides { 
    type, bullet, index, listNumber, depth -> "A.) " 
}) {
    Markdown(content, Modifier.fillMaxSize().padding(16.dp).verticalScroll(scrollState))
}
```

---

## 自定义组件

**从版本 0.9.0 开始**，可以提供自定义组件替代默认组件。

### 自定义段落组件

```kotlin
// 简单调整段落，使用不同的 Modifier
val customParagraphComponent: MarkdownComponent = {
    Box(modifier = Modifier.fillMaxWidth()) {
        MarkdownParagraph(
            it.content, 
            it.node, 
            Modifier.align(Alignment.CenterEnd)
        )
    }
}

// 完整的自定义段落示例
val customParagraphComponentComplex: MarkdownComponent = {
    // 构建带样式的文本
    val styledText = buildAnnotatedString {
        pushStyle(LocalMarkdownTypography.current.paragraph.toSpanStyle())
        buildMarkdownAnnotatedString(it.content, it.node, annotatorSettings())
        pop()
    }

    // 定义 Text composable
    Text(
        styledText,
        textAlign = TextAlign.End
    )
}

// 使用自定义组件
Markdown(
    content,
    components = markdownComponents(
        paragraph = customParagraphComponent
    )
)
```

### 自定义无序列表组件

```kotlin
// 定义自定义无序列表组件
val customUnorderedListComponent: MarkdownComponent = {
    // 使用 MarkdownListItems composable 渲染列表项
    MarkdownListItems(it.content, it.node, depth = 0) { 
        startNumber, index, child ->
        // 渲染带绿色色调的图标
        Icon(
            imageVector = icon,
            tint = Color.Green,
            contentDescription = null,
            modifier = Modifier.size(20.dp),
        )
    }
}

// 使用自定义组件
Markdown(
    content,
    components = markdownComponents(
        unorderedList = customUnorderedListComponent
    )
)
```

---

## 表格支持

**从版本 0.30.0 开始**，库支持渲染 Markdown 中的表格：

```kotlin
val markdown = """
| Header 1 | Header 2 |
|----------|----------|
| Cell 1   | Cell 2   |
| Cell 3   | Cell 4   |
""".trimIndent()

Markdown(markdown)
```

---

## 图片加载

库提供不同的图片加载实现，提供灵活性。

### Coil3 集成

```kotlin
// 添加依赖
implementation("com.mikepenz:multiplatform-markdown-renderer-coil3:${version}")

// 使用 Coil3
Markdown(
    MARKDOWN,
    imageTransformer = Coil3ImageTransformerImpl,
)
```

### Coil2 集成

```kotlin
// 添加依赖
implementation("com.mikepenz:multiplatform-markdown-renderer-coil2:${version}")

// 使用 Coil2
Markdown(
    MARKDOWN,
    imageTransformer = Coil2ImageTransformerImpl,
)
```

---

## 语法高亮

**从版本 0.27.0 开始**，库通过 Highlights 项目提供可选的语法高亮支持。

### 启用语法高亮

```kotlin
// 添加依赖
implementation("com.mikepenz:multiplatform-markdown-renderer-code:${version}")
```

### 配置高亮显示

```kotlin
// 使用默认配色方案
Markdown(
    MARKDOWN,
    components = markdownComponents(
        codeBlock = highlightedCodeBlock,
        codeFence = highlightedCodeFence,
    )
)

// 高级：自定义 Highlights 库主题
val isDarkTheme = isSystemInDarkTheme()
val highlightsBuilder = remember(isDarkTheme) {
    Highlights.Builder().theme(
        SyntaxThemes.atom(darkMode = isDarkTheme)
    )
}

Markdown(
    MARKDOWN,
    components = markdownComponents(
        codeBlock = {
            MarkdownHighlightedCodeBlock(
                content = it.content,
                node = it.node,
                highlightsBuilder = highlightsBuilder,
                showHeader = true, // 可选：启用头部显示语言 + 复制按钮
            )
        },
        codeFence = {
            MarkdownHighlightedCodeFence(
                content = it.content,
                node = it.node,
                highlightsBuilder = highlightsBuilder,
                showHeader = true, // 可选：启用头部显示语言 + 复制按钮
            )
        },
    )
)
```

---

## 核心依赖

库使用以下关键依赖：

- **JetBrains Markdown** - 多平台 Markdown 处理器，用于解析 Markdown 内容
- **Compose Multiplatform** - 用于跨平台 UI 渲染
- **Extended Spans** - 用于高级文本样式（集成为多平台）
- **Highlights** - 用于代码语法高亮（可选）

---

## Kotlin 集成示例

### 基础 Markdown 组件

```kotlin
@Composable
fun MarkdownContent(markdown: String) {
    val markdownState = rememberMarkdownState(
        markdown = markdown,
        retainState = true
    )
    
    Markdown(
        markdownState = markdownState,
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp)
    )
}
```

### 带语法高亮的代码块

```kotlin
@Composable
fun MarkdownWithCodeHighlight(markdown: String) {
    val isDarkTheme = isSystemInDarkTheme()
    val highlightsBuilder = remember(isDarkTheme) {
        Highlights.Builder().theme(
            SyntaxThemes.atom(darkMode = isDarkTheme)
        )
    }
    
    val markdownState = rememberMarkdownState(markdown)
    
    Markdown(
        markdownState = markdownState,
        components = markdownComponents(
            codeBlock = {
                MarkdownHighlightedCodeBlock(
                    content = it.content,
                    node = it.node,
                    highlightsBuilder = highlightsBuilder,
                    showHeader = true
                )
            }
        ),
        modifier = Modifier.fillMaxSize()
    )
}
```

### 大型文档懒加载

```kotlin
@Composable
fun LargeMarkdownDocument(markdown: String) {
    val markdownState = rememberMarkdownState(
        markdown = markdown,
        retainState = true
    )
    
    Markdown(
        markdownState = markdownState,
        success = { state, components, modifier ->
            LazyMarkdownSuccess(
                state = state,
                components = components,
                modifier = modifier,
                contentPadding = PaddingValues(16.dp)
            )
        },
        modifier = Modifier.fillMaxSize()
    )
}
```

### ViewModel 中解析

```kotlin
// ViewModel
class MarkdownViewModel : ViewModel() {
    private val _markdownFlow = MutableStateFlow<State<MarkdownContent>>(State.Loading())
    val markdownFlow: StateFlow<State<MarkdownContent>> = _markdownFlow.asStateFlow()
    
    fun loadMarkdown(content: String) {
        viewModelScope.launch {
            _markdownFlow.value = State.Loading()
            try {
                val parsed = parseMarkdown(content)
                _markdownFlow.value = State.Success(parsed)
            } catch (e: Exception) {
                _markdownFlow.value = State.Error(e)
            }
        }
    }
}

// Composable
@Composable
fun MarkdownScreen(viewModel: MarkdownViewModel) {
    val state by viewModel.markdownFlow.collectAsStateWithLifecycle()
    
    Markdown(
        markdownState = state,
        modifier = Modifier.fillMaxSize()
    )
}
```

---

## 最佳实践

### 性能优化

1. **使用 `rememberMarkdownState`**：避免重复解析
2. **启用 `retainState`**：避免内容闪烁
3. **大型文档使用懒加载**：使用 `LazyMarkdownSuccess`
4. **在 ViewModel 中解析**：保持滚动位置，避免重新解析

### 自定义建议

1. **保持默认样式**：使用 `markdownComponents()` 保留未覆盖的组件
2. **主题一致性**：确保自定义组件与 Material 主题一致
3. **图片优化**：使用 Coil 的图片缓存和缩略图功能
4. **代码高亮**：为技术文档启用语法高亮

### 常见问题

1. **内容闪烁**：设置 `retainState = true`
2. **滚动位置丢失**：在 ViewModel 中解析
3. **性能问题**：使用懒加载和异步解析
4. **样式不一致**：检查 Material 主题配置

---

## 版本历史

### v0.33.0+
- 默认异步解析 Markdown 内容
- 支持懒加载大型文档
- 改进状态保持

### v0.30.0+
- 添加表格支持

### v0.27.0+
- 添加语法高亮支持（通过 Highlights）

### v0.16.0+
- 集成 Extended Spans

### v0.13.0+
- 核心库不再依赖 Material 主题
- 需要单独添加 `-m2` 或 `-m3` 模块

### v0.9.0+
- 支持自定义组件

---

## 开发者信息

**作者**: Mike Penz
- 网站：mikepenz.com
- 邮箱：mikepenz@gmail.com
- PayPal: paypal.me/mikepenz

**贡献者**: 详见 CONTRIBUTORS.md 文件

**致谢**:
- Erik Hellman 的 [Rendering Markdown with Jetpack Compose](https://github.com/erev0s/MarkdownComposer)
- Saket Narayan 的 [extended-spans](https://github.com/saket/extended-spans) 项目

---

## 许可证

Copyright 2025 Mike Penz

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

---

## 参考链接

- [GitHub 仓库](https://github.com/mikepenz/multiplatform-markdown-renderer)
- [Compose Multiplatform](https://www.jetbrains.com/compose-multiplatform/)
- [JetBrains Markdown](https://github.com/JetBrains/markdown)
- [Coil](https://coil-kt.github.io/coil/)
- [Highlights](https://github.com/mikepenz/Highlights)
