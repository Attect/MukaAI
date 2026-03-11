package com.assistant.test.mem0

import kotlinx.serialization.*

/**
 * Mem0 创建记忆响应
 */
@Serializable
data class CreateMemoryResponse(
    val results: List<MemoryResult>
)

/**
 * 记忆结果对象
 */
@Serializable
data class MemoryResult(
    val id: String,
    val memory: String,
    val event: String
)

/**
 * 记忆对象（用于获取和搜索）
 */
@Serializable
data class Memory(
    val id: String,
    val memory: String,
    val hash: String? = null,
    @SerialName("user_id")
    val userId: String? = null,
    @SerialName("agent_id")
    val agentId: String? = null,
    @SerialName("run_id")
    val runId: String? = null,
    val metadata: Map<String, String>? = null,
    val score: Double? = null,
    @SerialName("created_at")
    val createdAt: String? = null,
    @SerialName("updated_at")
    val updatedAt: String? = null
)

/**
 * 创建记忆请求
 */
@Serializable
data class CreateMemoryRequest(
    val messages: List<Message>,
    @SerialName("user_id")
    val userId: String? = null,
    @SerialName("agent_id")
    val agentId: String? = null,
    @SerialName("run_id")
    val runId: String? = null,
    val metadata: Map<String, String>? = null
)

/**
 * 消息对象
 */
@Serializable
data class Message(
    val role: String,
    val content: String
)

/**
 * 搜索请求
 */
@Serializable
data class SearchRequest(
    val query: String,
    @SerialName("user_id")
    val userId: String? = null,
    @SerialName("agent_id")
    val agentId: String? = null,
    @SerialName("run_id")
    val runId: String? = null,
    val filters: Map<String, String>? = null,
    val limit: Int? = 10
)

/**
 * 搜索响应
 */
@Serializable
data class SearchResponse(
    val results: List<Memory>
)

/**
 * 获取所有记忆响应
 */
@Serializable
data class GetAllMemoriesResponse(
    val results: List<Memory>
)
