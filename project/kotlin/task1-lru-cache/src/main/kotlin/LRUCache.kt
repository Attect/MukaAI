package task1.lru.cache

import java.util.LinkedHashMap
import java.util.concurrent.locks.ReentrantReadWriteLock

/**
 * 线程安全的 LRU 缓存实现
 * 
 * @param K 键的类型
 * @param V 值的类型
 */
class LRUCache<K, V> private constructor(
    private val capacity: Int
) {
    // 使用 LinkedHashMap 维护访问顺序，accessOrder=true 表示按访问顺序排序
    private val cache: MutableMap<K, V> = LinkedHashMap(capacity, 0.75f, true)
    
    // 使用 ReentrantReadWriteLock 保证线程安全
    private val lock = ReentrantReadWriteLock()
    private val readLock = lock.readLock()
    private val writeLock = lock.writeLock()

    /**
     * 获取缓存值
     * 
     * @param key 键
     * @return 如果存在返回对应的值，否则返回 null
     */
    fun get(key: K): V? {
        readLock.lock()
        try {
            return cache[key]
        } finally {
            readLock.unlock()
        }
    }

    /**
     * 添加缓存值
     * 
     * @param key 键
     * @param value 值
     */
    fun put(key: K, value: V) {
        writeLock.lock()
        try {
            // 如果缓存已满，移除最久未使用的项
            if (cache.size >= capacity && !cache.containsKey(key)) {
                // LinkedHashMap 的 firstKey 返回最久未使用的键（当 accessOrder=true 时）
                val oldestKey = cache.keys.first()
                cache.remove(oldestKey)
            }
            
            // 添加或更新缓存项
            cache[key] = value
        } finally {
            writeLock.unlock()
        }
    }

    /**
     * 移除缓存值
     * 
     * @param key 键
     * @return 如果成功移除返回 true，否则返回 false
     */
    fun remove(key: K): Boolean {
        writeLock.lock()
        try {
            return cache.remove(key) != null
        } finally {
            writeLock.unlock()
        }
    }

    /**
     * 获取当前缓存大小
     * 
     * @return 缓存中的元素数量
     */
    fun size(): Int {
        readLock.lock()
        try {
            return cache.size
        } finally {
            readLock.unlock()
        }
    }

    /**
     * LRUCache 构建器
     */
    class Builder<K, V> {
        private var capacity: Int = 16

        fun setCapacity(capacity: Int): Builder<K, V> {
            this.capacity = capacity
            return this
        }

        fun build(): LRUCache<K, V> {
            return LRUCache(capacity)
        }
    }

    companion object {
        /**
         * 创建构建器实例
         */
        inline fun <reified K, reified V> builder(): Builder<K, V> {
            @Suppress("UNCHECKED_CAST")
            return Builder<K, V>() as Builder<K, V>
        }
    }
}
