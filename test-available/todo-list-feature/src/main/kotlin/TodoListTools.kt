package main

import java.time.LocalDate
import java.time.format.DateTimeFormatter
import java.util.concurrent.atomic.AtomicInteger

/**
 * TODO 列表工具接口
 */
interface TodoListTools {
    /**
     * 创建 TODO 列表
     */
    suspend fun createTodoList(
        roleId: String,
        sessionId: String,
        title: String,
        items: List<TodoItemInput>,
        parentListId: String? = null
    ): TodoList
    
    /**
     * 修改事项
     */
    suspend fun updateTodoItem(
        roleId: String,
        listId: String,
        itemId: String,
        updates: TodoItemUpdates
    ): TodoItem
    
    /**
     * 标记事项完成
     */
    suspend fun completeTodoItem(
        roleId: String,
        listId: String,
        itemId: String
    ): TodoItem
    
    /**
     * 取消 TODO 列表
     */
    suspend fun cancelTodoList(
        roleId: String,
        listId: String
    ): Boolean
    
    /**
     * 获取当前活跃的 TODO 列表（优先级最高的）
     */
    suspend fun getCurrentActiveTodoList(
        roleId: String,
        sessionId: String
    ): TodoList?
    
    /**
     * 获取所有 TODO 列表
     */
    suspend fun getAllTodoLists(
        roleId: String,
        sessionId: String,
        status: TodoListStatus? = null
    ): List<TodoList>
}

/**
 * TODO 列表工具实现
 */
class TodoListToolsImpl(
    private val storageManager: TodoListStorageManager = TodoListStorageManager(),
    private val idGenerator: TodoListIdGenerator = TodoListIdGenerator()
) : TodoListTools {
    
    override suspend fun createTodoList(
        roleId: String,
        sessionId: String,
        title: String,
        items: List<TodoItemInput>,
        parentListId: String?
    ): TodoList {
        val now = System.currentTimeMillis()
        val listId = idGenerator.generateId()
        
        val todoList = TodoList(
            id = listId,
            title = title,
            roleId = roleId,
            sessionId = sessionId,
            items = items.map { input ->
                TodoItem(
                    id = generateItemId(),
                    title = input.title,
                    description = input.description,
                    status = TodoItemStatus.PENDING,
                    priority = input.priority,
                    createdAt = now,
                    updatedAt = now
                )
            },
            status = TodoListStatus.ACTIVE,
            createdAt = now,
            updatedAt = now,
            parentListId = parentListId
        )
        
        storageManager.saveTodoList(todoList)
        println("[工具] 创建 TODO 列表成功：$listId, 标题：$title, 事项数：${items.size}")
        return todoList
    }
    
    override suspend fun updateTodoItem(
        roleId: String,
        listId: String,
        itemId: String,
        updates: TodoItemUpdates
    ): TodoItem {
        val todoList = storageManager.getTodoList(roleId, listId)
            ?: throw IllegalArgumentException("TODO 列表不存在：$listId")
        
        val updatedItems = todoList.items.map { item ->
            if (item.id == itemId) {
                item.copy(
                    title = updates.title ?: item.title,
                    description = updates.description ?: item.description,
                    priority = updates.priority ?: item.priority,
                    status = updates.status ?: item.status,
                    blockedReason = updates.blockedReason ?: item.blockedReason,
                    updatedAt = System.currentTimeMillis()
                )
            } else {
                item
            }
        }
        
        val updatedList = todoList.copy(
            items = updatedItems,
            updatedAt = System.currentTimeMillis()
        )
        
        storageManager.saveTodoList(updatedList)
        
        val updatedItem = updatedItems.first { it.id == itemId }
        println("[工具] 更新事项成功：$itemId, 状态：${updatedItem.status}")
        return updatedItem
    }
    
    override suspend fun completeTodoItem(
        roleId: String,
        listId: String,
        itemId: String
    ): TodoItem {
        val now = System.currentTimeMillis()
        
        val updatedItem = updateTodoItem(
            roleId = roleId,
            listId = listId,
            itemId = itemId,
            updates = TodoItemUpdates(
                status = TodoItemStatus.COMPLETED,
                blockedReason = null
            )
        ).copy(
            completedAt = now
        )
        
        // 检查是否所有事项都已完成
        val todoList = storageManager.getTodoList(roleId, listId)
        if (todoList != null) {
            val allCompleted = todoList.items.all { 
                it.status == TodoItemStatus.COMPLETED || it.status == TodoItemStatus.CANCELLED 
            }
            
            if (allCompleted) {
                storageManager.saveTodoList(
                    todoList.copy(
                        status = TodoListStatus.COMPLETED,
                        updatedAt = now
                    )
                )
                println("[工具] 所有事项已完成，列表状态更新为 COMPLETED")
            }
        }
        
        println("[工具] 标记事项完成：$itemId")
        return updatedItem
    }
    
    override suspend fun cancelTodoList(
        roleId: String,
        listId: String
    ): Boolean {
        val result = storageManager.cancelTodoList(roleId, listId)
        if (result) {
            println("[工具] 取消 TODO 列表成功：$listId")
        } else {
            println("[工具] 取消 TODO 列表失败：列表不存在")
        }
        return result
    }
    
    override suspend fun getCurrentActiveTodoList(
        roleId: String,
        sessionId: String
    ): TodoList? {
        val activeLists = storageManager.getActiveTodoLists(roleId)
        val currentList = activeLists.firstOrNull()
        
        if (currentList != null) {
            println("[工具] 获取当前优先级最高的列表：${currentList.id}, 标题：${currentList.title}")
        } else {
            println("[工具] 没有找到活跃的 TODO 列表")
        }
        
        return currentList
    }
    
    override suspend fun getAllTodoLists(
        roleId: String,
        sessionId: String,
        status: TodoListStatus?
    ): List<TodoList> {
        val allLists = storageManager.getAllTodoLists(roleId)
        val filteredLists = if (status != null) {
            allLists.filter { it.status == status }
        } else {
            allLists
        }
        
        println("[工具] 获取 TODO 列表：共 ${filteredLists.size} 个")
        return filteredLists
    }
    
    private fun generateItemId(): String {
        return "item_${System.currentTimeMillis()}_${(0..9999).random()}"
    }
}

/**
 * TODO 列表 ID 生成器
 */
class TodoListIdGenerator {
    private val counter = AtomicInteger(0)
    private val dateFormat = DateTimeFormatter.ofPattern("yyyyMMdd")
    
    /**
     * 生成语义化 ID
     * 格式：task-YYYYMMDD-NNN
     */
    fun generateId(): String {
        val date = LocalDate.now().format(dateFormat)
        val sequence = counter.incrementAndGet()
        return "task-${date}-${String.format("%03d", sequence)}"
    }
    
    /**
     * 重置计数器（每日重置）
     */
    fun resetForNewDay() {
        counter.set(0)
    }
}
