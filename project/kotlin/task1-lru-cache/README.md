# Task 1: LRU缓存实现

## 题目描述

实现一个线程安全的LRU（Least Recently Used）缓存，支持并发访问和自动淘汰机制。

## 功能要求

### 核心功能
1. **get(key: K): V?** - 获取缓存值，如果存在则更新访问顺序
2. **put(key: K, value: V)** - 添加缓存项，如果超出容量则淘汰最久未使用的项
3. **remove(key: K): V?** - 移除指定缓存项
4. **clear()** - 清空缓存
5. **size(): Int** - 获取当前缓存大小

### 扩展功能
1. **contains(key: K): Boolean** - 检查缓存是否包含指定键
2. **getAll(): Map<K, V>** - 获取所有缓存项的快照
3. **evict(count: Int)** - 手动淘汰指定数量的最久未使用项

## 技术要求

### 数据结构设计
- 使用双向链表维护访问顺序
- 使用HashMap实现O(1)时间复杂度的查找
- 链表节点需包含键值对信息

### 线程安全
- 支持多线程并发访问
- 使用适当的锁机制（推荐使用ReentrantReadWriteLock）
- 避免死锁和性能瓶颈

### 性能要求
- get操作: O(1)时间复杂度
- put操作: O(1)时间复杂度
- remove操作: O(1)时间复杂度
- 内存占用: O(capacity)

## 接口定义

```kotlin
interface LRUCache<K, V> {
    fun get(key: K): V?
    fun put(key: K, value: V)
    fun remove(key: K): V?
    fun clear()
    fun size(): Int
    fun contains(key: K): Boolean
    fun getAll(): Map<K, V>
    fun evict(count: Int)
}

class LRUCacheImpl<K, V>(
    private val capacity: Int
) : LRUCache<K, V> {
    // TODO: 实现细节
}
```

## 测试用例

### 基础功能测试

```kotlin
@Test
fun testBasicOperations() {
    val cache = LRUCacheImpl<String, Int>(capacity = 3)
    
    // 测试put和get
    cache.put("a", 1)
    cache.put("b", 2)
    cache.put("c", 3)
    assertEquals(1, cache.get("a"))
    assertEquals(2, cache.get("b"))
    assertEquals(3, cache.get("c"))
    
    // 测试容量限制和淘汰
    cache.put("d", 4) // 应该淘汰最久未使用的"a"
    assertNull(cache.get("a"))
    assertEquals(4, cache.get("d"))
    
    // 测试访问顺序更新
    cache.get("b") // "b"变为最近使用
    cache.put("e", 5) // 应该淘汰"c"
    assertNull(cache.get("c"))
    assertEquals(2, cache.get("b"))
}

@Test
fun testRemove() {
    val cache = LRUCacheImpl<String, Int>(capacity = 3)
    cache.put("a", 1)
    assertEquals(1, cache.remove("a"))
    assertNull(cache.get("a"))
    assertEquals(0, cache.size())
}

@Test
fun testClear() {
    val cache = LRUCacheImpl<String, Int>(capacity = 3)
    cache.put("a", 1)
    cache.put("b", 2)
    cache.clear()
    assertEquals(0, cache.size())
    assertNull(cache.get("a"))
}
```

### 并发测试

```kotlin
@Test
fun testConcurrentAccess() {
    val cache = LRUCacheImpl<Int, Int>(capacity = 100)
    val threadCount = 10
    val operationsPerThread = 1000
    
    val threads = (1..threadCount).map { threadId ->
        thread {
            repeat(operationsPerThread) { i ->
                val key = threadId * 1000 + i
                cache.put(key, key * 2)
                assertEquals(key * 2, cache.get(key))
            }
        }
    }
    
    threads.forEach { it.join() }
    
    // 验证缓存大小不超过容量
    assertTrue(cache.size() <= 100)
}

@Test
fun testConcurrentReadAndWrite() {
    val cache = LRUCacheImpl<Int, Int>(capacity = 50)
    
    // 并发写入
    val writeThreads = (1..5).map { threadId ->
        thread {
            repeat(100) { i ->
                cache.put(threadId * 100 + i, i)
            }
        }
    }
    
    // 并发读取
    val readThreads = (1..5).map { threadId ->
        thread {
            repeat(100) { i ->
                cache.get(threadId * 100 + i)
            }
        }
    }
    
    (writeThreads + readThreads).forEach { it.join() }
}
```

### 性能测试

```kotlin
@Test
fun testPerformance() {
    val cache = LRUCacheImpl<Int, Int>(capacity = 1000)
    val operations = 10000
    
    val startTime = System.currentTimeMillis()
    
    repeat(operations) { i ->
        cache.put(i, i * 2)
        cache.get(i)
    }
    
    val endTime = System.currentTimeMillis()
    val duration = endTime - startTime
    
    // 10000次操作应在100ms内完成
    assertTrue(duration < 100, "Performance test failed: ${duration}ms")
}
```

## 验收标准

### 功能验收
- [ ] 所有基础功能测试用例通过
- [ ] 正确实现LRU淘汰策略
- [ ] 边界情况处理正确（空缓存、满缓存、重复键等）

### 并发验收
- [ ] 并发访问测试通过
- [ ] 无死锁和竞态条件
- [ ] 线程安全验证通过

### 性能验收
- [ ] get/put/remove操作时间复杂度为O(1)
- [ ] 10000次操作耗时<100ms
- [ ] 内存占用符合预期

### 代码质量
- [ ] 代码结构清晰，命名规范
- [ ] 包含必要的注释和文档
- [ ] 测试覆盖率≥80%
- [ ] 无代码异味和警告

## 提示

1. 使用`ReentrantReadWriteLock`实现读写锁，提高并发性能
2. 双向链表节点建议定义为内部类
3. 注意处理链表头尾节点的特殊情况
4. 考虑使用`@Synchronized`或显式锁保护临界区
5. 建议先实现单线程版本，再添加线程安全支持

## 参考资料

- LRU缓存原理：https://en.wikipedia.org/wiki/Cache_replacement_policies#LRU
- Kotlin并发编程：https://kotlinlang.org/docs/shared-mutable-state-and-concurrency.html
