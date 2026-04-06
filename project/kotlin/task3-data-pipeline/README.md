# Task 3: 数据处理管道

## 题目描述

实现一个数据处理管道框架，支持链式操作、懒加载和并行处理，提供函数式编程风格的API。

## 功能要求

### 核心操作
1. **map(transform: (T) -> R)** - 转换元素
2. **filter(predicate: (T) -> Boolean)** - 过滤元素
3. **reduce(initial: R, accumulator: (R, T) -> R)** - 归约操作
4. **flatMap(transform: (T) -> Iterable<R>)** - 扁平化映射
5. **distinct()** - 去重
6. **sorted(comparator: Comparator<T>)** - 排序
7. **groupBy(keySelector: (T) -> K)** - 分组
8. **take(n: Int)** - 取前n个元素
9. **drop(n: Int)** - 跳过前n个元素

### 终止操作
1. **toList()** - 转换为列表
2. **toSet()** - 转换为集合
3. **toMap(keySelector: (T) -> K, valueSelector: (T) -> V)** - 转换为映射
4. **forEach(action: (T) -> Unit)** - 遍历元素
5. **count()** - 计数
6. **any(predicate: (T) -> Boolean)** - 是否存在满足条件的元素
7. **all(predicate: (T) -> Boolean)** - 是否所有元素都满足条件
8. **first()**, **firstOrNull()** - 获取第一个元素
9. **sum()**, **average()** - 数值统计

### 高级功能
1. **懒加载**: 中间操作不立即执行，直到终止操作时才执行
2. **并行处理**: 支持并行执行管道操作
3. **自定义操作**: 支持注册自定义管道操作
4. **调试支持**: 支持打印管道执行过程

## 技术要求

### 设计模式
- 使用Builder模式构建管道
- 使用装饰器模式实现操作链
- 使用迭代器模式实现懒加载

### 性能要求
- 懒加载：中间操作O(1)时间复杂度
- 并行处理：大数据集性能提升明显
- 内存效率：避免创建中间集合

### 扩展性
- 支持自定义操作扩展
- 支持自定义数据源
- 支持自定义并行策略

## 接口定义

```kotlin
interface Pipeline<T> {
    // 中间操作
    fun <R> map(transform: (T) -> R): Pipeline<R>
    fun filter(predicate: (T) -> Boolean): Pipeline<T>
    fun <R> reduce(initial: R, accumulator: (R, T) -> R): R
    fun <R> flatMap(transform: (T) -> Iterable<R>): Pipeline<R>
    fun distinct(): Pipeline<T>
    fun sorted(comparator: Comparator<T>): Pipeline<T>
    fun <K> groupBy(keySelector: (T) -> K): Pipeline<Map<K, List<T>>>
    fun take(n: Int): Pipeline<T>
    fun drop(n: Int): Pipeline<T>
    
    // 终止操作
    fun toList(): List<T>
    fun toSet(): Set<T>
    fun <K, V> toMap(keySelector: (T) -> K, valueSelector: (T) -> V): Map<K, V>
    fun forEach(action: (T) -> Unit)
    fun count(): Int
    fun any(predicate: (T) -> Boolean): Boolean
    fun all(predicate: (T) -> Boolean): Boolean
    fun first(): T
    fun firstOrNull(): T?
    
    // 数值操作（仅适用于数值类型）
    fun sum(): Double where T : Number
    fun average(): Double where T : Number
    
    // 并行处理
    fun parallel(): Pipeline<T>
    fun sequential(): Pipeline<T>
    
    // 调试支持
    fun peek(action: (T) -> Unit): Pipeline<T>
    fun debug(): Pipeline<T>
}

interface PipelineBuilder<T> {
    fun from(iterable: Iterable<T>): Pipeline<T>
    fun from(sequence: Sequence<T>): Pipeline<T>
    fun from(vararg elements: T): Pipeline<T>
    
    // 注册自定义操作
    fun <R> registerOperation(
        name: String,
        operation: (Pipeline<T>, List<Any?>) -> Pipeline<R>
    )
}

class DataPipeline<T> private constructor(
    private val source: Iterable<T>,
    private val operations: List<PipelineOperation<T>>
) : Pipeline<T> {
    // TODO: 实现细节
}

sealed class PipelineOperation<T> {
    data class Map<T, R>(val transform: (T) -> R) : PipelineOperation<T>()
    data class Filter<T>(val predicate: (T) -> Boolean) : PipelineOperation<T>()
    // ... 其他操作
}
```

## 测试用例

### 基础操作测试

```kotlin
@Test
fun testMapAndFilter() {
    val pipeline = PipelineBuilder.from(1, 2, 3, 4, 5)
    
    val result = pipeline
        .filter { it % 2 == 0 }
        .map { it * 2 }
        .toList()
    
    assertEquals(listOf(4, 8), result)
}

@Test
fun testReduce() {
    val pipeline = PipelineBuilder.from(1, 2, 3, 4, 5)
    
    val sum = pipeline.reduce(0) { acc, value -> acc + value }
    assertEquals(15, sum)
    
    val product = pipeline.reduce(1) { acc, value -> acc * value }
    assertEquals(120, product)
}

@Test
fun testFlatMap() {
    val pipeline = PipelineBuilder.from(listOf(1, 2), listOf(3, 4))
    
    val result = pipeline
        .flatMap { it }
        .toList()
    
    assertEquals(listOf(1, 2, 3, 4), result)
}

@Test
fun testDistinct() {
    val pipeline = PipelineBuilder.from(1, 2, 2, 3, 3, 3, 4)
    
    val result = pipeline.distinct().toList()
    
    assertEquals(listOf(1, 2, 3, 4), result)
}

@Test
fun testSorted() {
    val pipeline = PipelineBuilder.from(3, 1, 4, 1, 5, 9, 2, 6)
    
    val ascending = pipeline.sorted(naturalOrder()).toList()
    assertEquals(listOf(1, 1, 2, 3, 4, 5, 6, 9), ascending)
    
    val descending = pipeline.sorted(reverseOrder()).toList()
    assertEquals(listOf(9, 6, 5, 4, 3, 2, 1, 1), descending)
}
```

### 分组和聚合测试

```kotlin
@Test
fun testGroupBy() {
    data class Person(val name: String, val age: Int)
    
    val pipeline = PipelineBuilder.from(
        Person("Alice", 25),
        Person("Bob", 30),
        Person("Charlie", 25),
        Person("David", 30)
    )
    
    val groups = pipeline.groupBy { it.age }.toList().first()
    
    assertEquals(2, groups[25]?.size)
    assertEquals(2, groups[30]?.size)
    assertTrue(groups[25]?.any { it.name == "Alice" } == true)
}

@Test
fun testTakeAndDrop() {
    val pipeline = PipelineBuilder.from(1, 2, 3, 4, 5)
    
    val first3 = pipeline.take(3).toList()
    assertEquals(listOf(1, 2, 3), first3)
    
    val last2 = pipeline.drop(3).toList()
    assertEquals(listOf(4, 5), last2)
}
```

### 终止操作测试

```kotlin
@Test
fun testTerminationOperations() {
    val pipeline = PipelineBuilder.from(1, 2, 3, 4, 5)
    
    // count
    assertEquals(5, pipeline.count())
    
    // any/all
    assertTrue(pipeline.any { it > 4 })
    assertFalse(pipeline.any { it > 10 })
    assertTrue(pipeline.all { it > 0 })
    assertFalse(pipeline.all { it > 3 })
    
    // first
    assertEquals(1, pipeline.first())
    assertEquals(1, pipeline.firstOrNull())
    
    // sum/average
    assertEquals(15.0, pipeline.sum(), 0.0001)
    assertEquals(3.0, pipeline.average(), 0.0001)
}

@Test
fun testToCollections() {
    val pipeline = PipelineBuilder.from(1, 2, 2, 3, 3, 3)
    
    val list = pipeline.toList()
    assertEquals(listOf(1, 2, 2, 3, 3, 3), list)
    
    val set = pipeline.toSet()
    assertEquals(setOf(1, 2, 3), set)
    
    val map = pipeline.toMap({ it }, { it * 2 })
    assertEquals(mapOf(1 to 2, 2 to 4, 3 to 6), map)
}
```

### 懒加载测试

```kotlin
@Test
fun testLazyEvaluation() {
    var mapCount = 0
    var filterCount = 0
    
    val pipeline = PipelineBuilder.from(1, 2, 3, 4, 5)
        .map { 
            mapCount++
            it * 2 
        }
        .filter { 
            filterCount++
            it > 5 
        }
    
    // 此时操作尚未执行
    assertEquals(0, mapCount)
    assertEquals(0, filterCount)
    
    // 执行终止操作
    val result = pipeline.toList()
    
    // 现在操作才执行
    assertTrue(mapCount > 0)
    assertTrue(filterCount > 0)
    assertEquals(listOf(6, 8, 10), result)
}

@Test
fun testShortCircuit() {
    var count = 0
    
    val result = PipelineBuilder.from(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
        .map { 
            count++
            it * 2 
        }
        .take(3)
        .toList()
    
    // 由于懒加载和take(3)，map应该只执行3次
    assertEquals(3, count)
    assertEquals(listOf(2, 4, 6), result)
}
```

### 并行处理测试

```kotlin
@Test
fun testParallelProcessing() {
    val size = 10000
    val data = (1..size).toList()
    
    val sequentialTime = measureTime {
        PipelineBuilder.from(data)
            .map { it * 2 }
            .filter { it % 4 == 0 }
            .toList()
    }
    
    val parallelTime = measureTime {
        PipelineBuilder.from(data)
            .parallel()
            .map { it * 2 }
            .filter { it % 4 == 0 }
            .toList()
    }
    
    // 并行处理应该更快（至少在某些情况下）
    println("Sequential: $sequentialTime, Parallel: $parallelTime")
}

@Test
fun testParallelCorrectness() {
    val data = (1..100).toList()
    
    val sequential = PipelineBuilder.from(data)
        .map { it * 2 }
        .filter { it % 4 == 0 }
        .toList()
        .sorted()
    
    val parallel = PipelineBuilder.from(data)
        .parallel()
        .map { it * 2 }
        .filter { it % 4 == 0 }
        .toList()
        .sorted()
    
    assertEquals(sequential, parallel)
}
```

### 自定义操作测试

```kotlin
@Test
fun testCustomOperation() {
    val builder = PipelineBuilder<Int>()
    
    // 注册自定义操作：每n个元素取一个
    builder.registerOperation<Int, Int>("sample") { pipeline, args ->
        val n = args[0] as Int
        pipeline.filterIndexed { index, _ -> index % n == 0 }
    }
    
    val pipeline = builder.from(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
    val result = pipeline.sample(3).toList()
    
    assertEquals(listOf(1, 4, 7, 10), result)
}
```

### 调试测试

```kotlin
@Test
fun testDebug() {
    val output = mutableListOf<String>()
    
    val result = PipelineBuilder.from(1, 2, 3, 4, 5)
        .peek { output.add("Before filter: $it") }
        .filter { it % 2 == 0 }
        .peek { output.add("After filter: $it") }
        .map { it * 2 }
        .peek { output.add("After map: $it") }
        .toList()
    
    assertEquals(listOf(4, 8), result)
    assertTrue(output.contains("Before filter: 1"))
    assertTrue(output.contains("After filter: 2"))
    assertTrue(output.contains("After map: 4"))
}
```

## 验收标准

### 功能验收
- [ ] 所有核心操作正确实现
- [ ] 所有终止操作正确实现
- [ ] 链式调用正常工作
- [ ] 懒加载机制正确

### 性能验收
- [ ] 中间操作O(1)时间复杂度
- [ ] 并行处理性能提升明显
- [ ] 内存使用高效
- [ ] 短路优化正常工作

### 扩展性验收
- [ ] 支持自定义操作
- [ ] 支持自定义数据源
- [ ] API设计合理易用

### 代码质量
- [ ] 设计模式应用恰当
- [ ] 代码结构清晰
- [ ] 测试覆盖率≥80%
- [ ] 包含详细文档

## 提示

1. 使用Sequence实现懒加载，避免创建中间集合
2. 使用线程池实现并行处理，注意线程安全
3. 使用装饰器模式包装操作链
4. 考虑使用inline函数优化性能
5. 注意处理异常和边界情况

## 参考资料

- Kotlin Sequence：https://kotlinlang.org/docs/sequences.html
- Java Stream API：https://docs.oracle.com/javase/8/docs/api/java/util/stream/Stream.html
- 并行流处理：https://kotlinlang.org/docs/coroutines-guide.html
