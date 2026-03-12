package main

import kotlinx.serialization.json.Json
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.encodeToString
import kotlinx.serialization.decodeFromString
import java.io.File
import java.nio.file.Files
import java.nio.file.Path

// 全局 JSON 序列化配置
val JsonFormat = Json {
    ignoreUnknownKeys = true
    encodeDefaults = true
}

/**
 * TODO 列表存储管理器
 * 使用文件系统存储 TODO 列表数据
 * 存储位置：每个 AI 角色的工作区目录
 */
class TodoListStorageManager {
    companion object {
        private const val TODO_LISTS_DIR = "todo-lists"
        private const val FILE_EXTENSION = ".json"
    }
    
    private val workspaceBasePath: Path = Path.of("workspace")
    
    /**
     * 获取角色的 TODO 列表存储目录
     * @param roleId 角色 ID
     * @return 角色的 TODO 列表存储目录路径
     */
    private fun getRoleTodoDir(roleId: String): Path {
        val roleDir = workspaceBasePath.resolve(roleId)
        val todoDir = roleDir.resolve(TODO_LISTS_DIR)
        
        // 确保目录存在
        if (!Files.exists(todoDir)) {
            Files.createDirectories(todoDir)
        }
        
        return todoDir
    }
    
    /**
     * 获取 TODO 列表文件路径
     * @param roleId 角色 ID
     * @param listId 列表 ID
     * @return TODO 列表文件路径
     */
    private fun getTodoListFile(roleId: String, listId: String): Path {
        return getRoleTodoDir(roleId).resolve("$listId$FILE_EXTENSION")
    }
    
    /**
     * 保存 TODO 列表到文件系统
     * @param todoList 要保存的 TODO 列表
     */
    suspend fun saveTodoList(todoList: TodoList): Unit = withContext(Dispatchers.IO) {
        val file = getTodoListFile(todoList.roleId, todoList.id).toFile()
        
        // 确保父目录存在
        file.parentFile?.mkdirs()
        
        // 写入 JSON 文件
        val jsonContent = JsonFormat.encodeToString<TodoList>(todoList)
        file.writeText(jsonContent)
        
        println("[存储] 保存 TODO 列表成功：${todoList.id} (文件：${file.absolutePath})")
    }
    
    /**
     * 从文件系统获取 TODO 列表
     * @param roleId 角色 ID
     * @param listId 列表 ID
     * @return TODO 列表，如果不存在则返回 null
     */
    suspend fun getTodoList(roleId: String, listId: String): TodoList? = withContext(Dispatchers.IO) {
        val file = getTodoListFile(roleId, listId).toFile()
        
        if (!file.exists()) {
            println("[存储] TODO 列表不存在：$listId")
            return@withContext null
        }
        
        try {
            val jsonContent = file.readText()
            val todoList = JsonFormat.decodeFromString<TodoList>(jsonContent)
            println("[存储] 获取 TODO 列表成功：$listId")
            todoList
        } catch (e: Exception) {
            println("[存储] 读取 TODO 列表失败：${e.message}")
            null
        }
    }
    
    /**
     * 获取角色的所有 TODO 列表
     * @param roleId 角色 ID
     * @return TODO 列表列表，按更新时间降序排序
     */
    suspend fun getAllTodoLists(roleId: String): List<TodoList> = withContext(Dispatchers.IO) {
        val todoDir = getRoleTodoDir(roleId)
        
        if (!Files.exists(todoDir)) {
            return@withContext emptyList()
        }
        
        val files = todoDir.toFile().listFiles()?.filter { it.name.endsWith(FILE_EXTENSION) } ?: emptyList()
        
        val lists = files.mapNotNull { file ->
            try {
                val jsonContent = file.readText()
                JsonFormat.decodeFromString<TodoList>(jsonContent)
            } catch (e: Exception) {
                println("[存储] 读取文件失败 ${file.name}: ${e.message}")
                null
            }
        }
        
        // 按更新时间降序排序
        lists.sortedByDescending { it.updatedAt }
    }
    
    /**
     * 获取角色所有活跃的 TODO 列表
     * @param roleId 角色 ID
     * @return 活跃的 TODO 列表列表，按更新时间降序排序
     */
    suspend fun getActiveTodoLists(roleId: String): List<TodoList> = withContext(Dispatchers.IO) {
        val allLists = getAllTodoLists(roleId)
        allLists.filter { it.status == TodoListStatus.ACTIVE }
    }
    
    /**
     * 删除 TODO 列表
     * @param roleId 角色 ID
     * @param listId 列表 ID
     * @return 是否删除成功
     */
    suspend fun deleteTodoList(roleId: String, listId: String): Boolean = withContext(Dispatchers.IO) {
        val file = getTodoListFile(roleId, listId).toFile()
        
        if (!file.exists()) {
            println("[存储] TODO 列表不存在：$listId")
            return@withContext false
        }
        
        file.delete()
        println("[存储] 删除 TODO 列表成功：$listId")
        true
    }
    
    /**
     * 取消 TODO 列表（标记为已取消）
     * @param roleId 角色 ID
     * @param listId 列表 ID
     * @return 是否取消成功
     */
    suspend fun cancelTodoList(roleId: String, listId: String): Boolean = withContext(Dispatchers.IO) {
        val todoList = getTodoList(roleId, listId) ?: return@withContext false
        
        val now = System.currentTimeMillis()
        val cancelledList = todoList.copy(
            status = TodoListStatus.CANCELLED,
            updatedAt = now,
            items = todoList.items.map { item ->
                if (item.status == TodoItemStatus.PENDING || item.status == TodoItemStatus.IN_PROGRESS) {
                    item.copy(
                        status = TodoItemStatus.CANCELLED,
                        updatedAt = now
                    )
                } else {
                    item
                }
            }
        )
        
        saveTodoList(cancelledList)
        println("[存储] 取消 TODO 列表成功：$listId")
        true
    }
}
