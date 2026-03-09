# Kotlin 官方文档参考

本文档整理了 Kotlin 官方最新文档 (版本 2.3.10) 的核心内容，用于项目开发参考。

## 文档来源

- **官方文档**: https://kotlinlang.org/docs/
- **版本**: Kotlin 2.3.10 (最新版本)
- **发布日期**: 2025 年 12 月 16 日

## 核心特性

### 1. 现代化语言特性

Kotlin 是一门现代的编程语言，具有以下特点:

- **简洁性**: 代码简洁，减少样板代码
- **多平台支持**: Kotlin Multiplatform (KMP) 支持跨平台开发
- **互操作性**: 与 Java 及其他语言完全兼容
- **安全性**: 空安全设计，避免 NullPointerException
- **函数式编程**: 支持高阶函数、Lambda 表达式等

### 2. K2 编译器

Kotlin 2.0 开始使用全新的 K2 编译器，带来以下改进:

- **性能提升**: 编译速度更快，内存使用更优
- **统一架构**: JVM、Native、Wasm、JS 所有平台统一使用 K2 编译器
- **智能转换增强**: 更多场景支持自动类型转换
- **更好的错误提示**: 更清晰的编译错误和警告信息

### 3. 语言版本演进

#### Kotlin 2.3.0 (最新版本)

**主要更新**:

- **语言特性**:
  - 更多稳定和默认启用的功能
  - 未使用返回值检查器
  - 显式 backing fields
  - 上下文敏感解析的改进

- **Kotlin/JVM**:
  - 支持 Java 25 字节码

- **Kotlin/Native**:
  - 通过 Swift 导出改进互操作性
  - 发布任务构建时间更快
  - C 和 Objective-C 库导入进入 Beta

- **Kotlin/Wasm**:
  - 默认启用完全限定名和新的异常处理提案
  - Latin-1 字符的新型紧凑存储

- **Kotlin/JS**:
  - 新的实验性 suspend 函数导出
  - LongArray 表示
  - 统一的 companion object 访问

- **Gradle**:
  - 兼容 Gradle 9.0
  - 注册生成源代码的新 API

- **Compose 编译器**:
  - 支持精简版 Android 应用的堆栈跟踪

- **标准库**:
  - 稳定的时间追踪功能
  - 改进的 UUID 生成和解析

#### Kotlin 2.2.0

**主要特性**:

- **上下文参数 (Context Parameters)**: 预览功能，简化依赖注入
- **上下文敏感解析**: 预览功能，自动推断枚举和密封类成员
- **注解使用站点目标**: 改进注解处理
- **嵌套类型别名**: 支持在类内部定义类型别名
- **稳定功能**:
  - when 表达式中的守卫条件
  - 非局部 break 和 continue
  - 多美元符号字符串插值

#### Kotlin 2.1.0

**主要特性**:

- **Swift 导出基础支持**: 简化 iOS 开发
- **编译器选项 Gradle DSL 稳定**: 简化多平台项目配置
- **iosArm64 提升为 Tier 1**: 最高级别支持
- **LLVM 更新**: 从 16 升级到 19
- **增量编译**: Kotlin/Wasm 支持增量编译

#### Kotlin 2.0.0

**里程碑版本**:

- **K2 编译器稳定**: 所有平台默认使用 K2 编译器
- **智能转换改进**: 更多场景支持自动类型转换
- **多平台改进**:
  - 公共和平台源代码分离
  - 预期和实际声明的不同可见性级别
- **Compose 编译器 Gradle 插件**: 独立的 Compose 编译器插件

## 基本语法

### 包定义和导入

```kotlin
// 包定义必须位于源文件顶部
package org.example

// 导入语句
import kotlin.collections.List
```

### 程序入口点

```kotlin
// 基本形式
fun main() {
    // ...
}

// 接受命令行参数
fun main(args: Array<String>) {
    // ...
}
```

### 变量声明

```kotlin
// val - 不可变变量 (只读)
val name: String = "Kotlin"
val age = 25  // 类型推断

// var - 可变变量
var count: Int = 0
count = 1  // 可以重新赋值

// 延迟初始化
lateinit var config: Config
```

### 函数

```kotlin
// 基本函数
fun add(a: Int, b: Int): Int {
    return a + b
}

// 表达式函数体
fun multiply(a: Int, b: Int) = a * b

// 无返回值 (Unit 可省略)
fun printHello() {
    println("Hello")
}

// 默认参数值
fun greet(name: String = "Guest") {
    println("Hello, $name")
}

// 命名参数
greet(name = "Alice")
```

### 类和对象

```kotlin
// 类定义
class Person(val name: String, var age: Int) {
    // 属性
    val greeting: String = "Hello"
    
    // 方法
    fun introduce() {
        println("Hi, I'm $name")
    }
}

// 继承 (类默认为 final)
open class Animal {
    open fun speak() = println("Animal speaks")
}

class Dog : Animal() {
    override fun speak() = println("Dog barks")
}

// 数据类
data class User(val id: Int, val name: String)

// 单例对象
object Singleton {
    fun doSomething() { /* ... */ }
}

// 伴生对象
class MyClass {
    companion object {
        fun create() = MyClass()
    }
}
```

### 空安全

```kotlin
// 可空类型
var nullable: String? = null

// 安全调用操作符
val length = nullable?.length

// Elvis 操作符
val name = nullable ?: "Default"

// 非空断言
val length = nullable!!.length

// 安全转换
val str = any as? String
```

### 控制流

```kotlin
// if 表达式
val max = if (a > b) a else b

// when 表达式
val description = when (x) {
    1 -> "One"
    2, 3 -> "Two or Three"
    in 4..10 -> "Range 4-10"
    else -> "Other"
}

// for 循环
for (i in 0..10) { /* ... */ }
for (i in 10 downTo 1) { /* ... */ }
for (i in 1..10 step 2) { /* ... */ }

// while 循环
while (condition) { /* ... */ }
do { /* ... */ } while (condition)
```

### 集合

```kotlin
// 列表
val list = listOf(1, 2, 3)
val mutableList = mutableListOf(1, 2, 3)

// 集合
val set = setOf("a", "b", "c")
val mutableSet = mutableSetOf("a", "b", "c")

// 映射
val map = mapOf("key" to "value")
val mutableMap = mutableMapOf("key" to "value")

// 集合操作
val filtered = list.filter { it > 1 }
val mapped = list.map { it * 2 }
val first = list.firstOrNull()
```

### Lambda 表达式和高阶函数

```kotlin
// Lambda 表达式
val sum = { x: Int, y: Int -> x + y }

// 高阶函数
fun operate(a: Int, b: Int, operation: (Int, Int) -> Int): Int {
    return operation(a, b)
}

// 使用 Lambda
val result = operate(2, 3) { x, y -> x + y }

// it 关键字 (单参数 Lambda)
list.filter { it > 0 }

// 作用域函数
// let - 执行 Lambda 并返回结果
val result = nullable?.let { it.length }

// apply - 配置对象并返回对象本身
val person = Person("Alice").apply {
    age = 25
}

// run - 配置对象并返回结果
val result = person.run { "$name is $age years old" }

// with - 在对象上执行 Lambda
with(person) {
    println("$name is $age years old")
}

// also - 执行副作用并返回对象
person.also { println("Created person: $it") }
```

### 扩展函数

```kotlin
// 定义扩展函数
fun String.addPrefix(): String = "Prefix: $this"

// 使用扩展函数
val result = "Hello".addPrefix()

// 扩展属性
val String.lastChar: Char
    get() = this[length - 1]
```

### 协程

```kotlin
// 启动协程
lifecycleScope.launch {
    // 异步代码
    val result = withContext(Dispatchers.IO) {
        // 耗时操作
        fetchData()
    }
    // 更新 UI
    updateUI(result)
}

// Flow
val flow = flow {
    emit(1)
    emit(2)
    emit(3)
}

flow
    .map { it * 2 }
    .filter { it > 2 }
    .collect { value ->
        println(value)
    }
```

## 编码规范

### 命名规范

- **类和对象**: 大驼峰命名 (UpperCamelCase)
  ```kotlin
  class Person, interface Readable
  ```

- **函数、属性和局部变量**: 小驼峰命名 (lowerCamelCase)
  ```kotlin
  fun calculateTotal(), val itemCount
  ```

- **常量**: 全大写，下划线分隔 (SCREAMING_SNAKE_CASE)
  ```kotlin
  const val MAX_COUNT = 100
  ```

- **包名**: 全小写，不使用下划线
  ```kotlin
  package org.example.project
  ```

### 代码格式

- **缩进**: 使用 4 个空格，不使用 Tab
- **大括号**: 左大括号在行尾，右大括号单独一行
- **空格**:
  - 二元运算符周围加空格：`a + b`
  - 控制流关键字后加空格：`if (condition)`
  - 冒号后加空格：`String: Type`

### 文件组织

- 单个类的文件：文件名与类名相同
- 多个类的文件：使用描述性名称
- 多平台项目：平台特定文件添加后缀
  - `Platform.jvm.kt`
  - `Platform.android.kt`
  - `Platform.ios.kt`

### 类布局

类内容按以下顺序排列:

1. 属性声明和初始化块
2. 次级构造函数
3. 方法声明
4. 伴生对象

### 注释规范

```kotlin
/**
 * 类的 KDoc 注释
 * 
 * @property name 属性描述
 * @constructor 构造函数描述
 */
class Person(val name: String) {
    /**
     * 方法的 KDoc 注释
     * 
     * @param param 参数描述
     * @return 返回值描述
     */
    fun method(param: String): Int {
        // 单行注释
        return 42
    }
}
```

## 常用惯用写法

### 创建 DTO

```kotlin
data class Customer(
    val name: String,
    val postalCode: Int
)
```

### 过滤列表

```kotlin
val fruits = basket.filter { it.startsWith("apple") }
```

### 检查元素存在

```kotlin
if (collection.contains(element)) { /* ... */ }
// 或
if (element in collection) { /* ... */ }
```

### 字符串模板

```kotlin
val name = "Kotlin"
println("Hello, $name!")
println("Length: ${name.length}")
```

### 安全读取标准输入

```kotlin
val input = readlnOrNull()
if (input != null) {
    println("You entered: $input")
}
```

### 实例检查

```kotlin
if (obj is String) {
    // 自动类型转换
    println(obj.length)
}

// 或使用 when
when (obj) {
    is String -> println("String: ${obj.length}")
    is Int -> println("Int: $obj")
    else -> println("Unknown type")
}
```

### 只读集合

```kotlin
val list: List<String> = listOf("a", "b", "c")
val map: Map<String, Int> = mapOf("key" to 42)
```

### 遍历映射

```kotlin
for ((k, v) in map) {
    println("$k -> $v")
}
```

### 懒加载属性

```kotlin
val lazyValue: String by lazy {
    println("computed!")
    "Hello"
}
```

### 单例模式

```kotlin
object Resource {
    val data = loadData()
    fun doSomething() { /* ... */ }
}
```

### 内联值类

```kotlin
@JvmInline
value class UserId(val id: String)

@JvmInline
value class OrderId(val id: String)

// 类型安全，无法混用
fun findUser(id: UserId) { /* ... */ }
```

### Not-null 简写

```kotlin
val length = nullable?.length ?: 0
nullable?.let { println(it.length) }
```

### try-catch 表达式

```kotlin
val result = try {
    parseInt(str)
} catch (e: NumberFormatException) {
    null
}
```

### 标记代码未完成

```kotlin
fun test() = TODO("Not implemented yet")
```

## 多平台开发

### 期望/实际机制

```kotlin
// commonMain
expect class Platform() {
    val name: String
}

// jvmMain
actual class Platform actual constructor() {
    actual val name: String = "JVM"
}

// androidMain
actual class Platform actual constructor() {
    actual val name: String = "Android"
}

// iosMain
actual class Platform actual constructor() {
    actual val name: String = "iOS"
}
```

### 平台特定代码

```kotlin
// 使用 @OptIn 标记实验性 API
@OptIn(ExperimentalForeignApi::class)
fun platformSpecific() {
    // 平台特定实现
}
```

## 最佳实践

### 1. 优先使用不可变性

```kotlin
// 推荐
val list = listOf(1, 2, 3)

// 避免
var list = mutableListOf(1, 2, 3)
```

### 2. 使用默认参数值代替重载

```kotlin
// 推荐
fun greet(name: String = "Guest", greeting: String = "Hello") {
    println("$greeting, $name")
}

// 避免
fun greet() { /* ... */ }
fun greet(name: String) { /* ... */ }
fun greet(name: String, greeting: String) { /* ... */ }
```

### 3. 使用扩展函数

```kotlin
// 推荐
fun String.addPrefix() = "Prefix: $this"

// 避免创建工具类
object StringUtils {
    fun addPrefix(str: String) = "Prefix: $str"
}
```

### 4. 使用高阶函数代替循环

```kotlin
// 推荐
val positives = numbers.filter { it > 0 }

// 避免
val positives = mutableListOf<Int>()
for (n in numbers) {
    if (n > 0) positives.add(n)
}
```

### 5. 使用字符串模板

```kotlin
// 推荐
println("Hello, $name!")

// 避免
println("Hello, " + name + "!")
```

### 6. 使用多行字符串

```kotlin
// 推荐
val html = """
    <html>
        <body>
            <h1>Title</h1>
        </body>
    </html>
""".trimIndent()

// 避免
val html = "<html>\n    <body>\n        <h1>Title</h1>\n    </body>\n</html>"
```

## 开发环境

### IDE 支持

- **IntelliJ IDEA**: 2023.3 及以上版本 (内置 Kotlin 插件)
- **Android Studio**: Iguana (2023.2.1) 及以上版本 (内置 Kotlin 插件)

### 构建工具

- **Gradle**: 7.6.3 到 9.0
- **Kotlin DSL**: 推荐使用 Kotlin 编写构建脚本
- **Kotlin 插件**: 
  ```kotlin
  plugins {
      kotlin("multiplatform") version "2.3.10"
      kotlin("plugin.serialization") version "2.3.10"
  }
  ```

### 依赖管理

使用 `gradle/libs.versions.toml` 进行版本管理:

```toml
[versions]
kotlin = "2.3.10"
ktor = "3.4.1"
compose = "2.3.10"

[libraries]
kotlin-stdlib = { group = "org.jetbrains.kotlin", name = "kotlin-stdlib", version.ref = "kotlin" }
ktor-client-core = { group = "io.ktor", name = "ktor-client-core", version.ref = "ktor" }
```

## 参考资源

- [Kotlin 官方文档](https://kotlinlang.org/docs/)
- [Kotlin 基础语法](https://kotlinlang.org/docs/basic-syntax.html)
- [Kotlin 编码规范](https://kotlinlang.org/docs/coding-conventions.html)
- [Kotlin 惯用写法](https://kotlinlang.org/docs/idioms.html)
- [Kotlin 多平台开发](https://kotlinlang.org/docs/multiplatform.html)
- [Kotlin 协程](https://kotlinlang.org/docs/coroutines-overview.html)
- [Kotlin 序列化](https://kotlinlang.org/docs/serialization.html)

## 更新日志

- **2026-03-07**: 初始版本，整理 Kotlin 2.3.10 核心文档
- 基于 Kotlin 官方文档最新版本
