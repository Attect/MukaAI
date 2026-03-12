package main

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import java.time.Instant

/**
 * TODO 列表数据类
 */
@Serializable
data class TodoList(
    val id: String,                    // TODO 列表 ID（语义化命名，如 "task-20260312-001"）
    val title: String,                 // 总标题
    val roleId: String,                // AI 角色 ID
    val sessionId: String,             // 所属会话 ID
    val items: List<TodoItem>,         // 事项列表
    val status: TodoListStatus,        // 列表状态
    val createdAt: Long,               // 创建时间（Unix 时间戳毫秒）
    val updatedAt: Long,               // 最后更新时间
    val parentListId: String? = null   // 父列表 ID（用于恢复上级内容）
)

/**
 * TODO 事项数据类
 */
@Serializable
data class TodoItem(
    val id: String,                    // 事项 ID
    val title: String,                 // 事项标题
    val description: String? = null,   // 事项描述
    val status: TodoItemStatus,        // 事项状态
    val priority: Int = 0,             // 优先级（数字越大优先级越高）
    val createdAt: Long,               // 创建时间
    val updatedAt: Long,               // 最后更新时间
    val completedAt: Long? = null,     // 完成时间
    val blockedReason: String? = null  // 阻塞原因（当状态为 Blocked 时）
)

/**
 * TODO 列表状态
 */
@Serializable
enum class TodoListStatus {
    ACTIVE,         // 活跃中
    COMPLETED,      // 全部完成
    CANCELLED       // 已取消
}

/**
 * TODO 事项状态
 */
@Serializable
enum class TodoItemStatus {
    PENDING,        // 待执行
    IN_PROGRESS,    // 进行中
    COMPLETED,      // 已完成
    CANCELLED,      // 已取消
    BLOCKED         // 阻塞中
}

/**
 * 事项输入数据类
 */
@Serializable
data class TodoItemInput(
    val title: String,
    val description: String? = null,
    val priority: Int = 0
)

/**
 * 事项更新数据类
 */
@Serializable
data class TodoItemUpdates(
    val title: String? = null,
    val description: String? = null,
    val priority: Int? = null,
    val status: TodoItemStatus? = null,
    val blockedReason: String? = null
)
