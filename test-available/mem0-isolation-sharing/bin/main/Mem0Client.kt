package com.assistant.test.mem0

import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.engine.cio.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.serialization.json.Json
import kotlinx.coroutines.*

/**
 * Mem0 REST API 客户端
 * 用于与 mem0 server 进行通信
 */
class Mem0Client(
    private val baseUrl: String = "http://localhost:8000"
) {
    private val client = HttpClient(CIO) {
        install(ContentNegotiation) {
            json(Json {
                ignoreUnknownKeys = true
                isLenient = true
                encodeDefaults = true
            })
        }
    }
    
    /**
     * 创建记忆
     * 返回创建的记忆结果列表
     */
    suspend fun createMemory(request: CreateMemoryRequest): List<MemoryResult> {
        val response = client.post("$baseUrl/memories") {
            contentType(ContentType.Application.Json)
            setBody(request)
        }
        
        if (!response.status.isSuccess()) {
            throw Exception("创建记忆失败: ${response.status} - ${response.bodyAsText()}")
        }
        
        val createResponse: CreateMemoryResponse = response.body()
        return createResponse.results
    }
    
    /**
     * 获取所有记忆
     * 返回记忆列表
     */
    suspend fun getAllMemories(userId: String? = null): List<Memory> {
        val url = buildString {
            append("$baseUrl/memories")
            if (userId != null) {
                append("?user_id=$userId")
            }
        }
        
        val response = client.get(url)
        
        if (!response.status.isSuccess()) {
            throw Exception("获取记忆失败: ${response.status} - ${response.bodyAsText()}")
        }
        
        val getResponse: GetAllMemoriesResponse = response.body()
        return getResponse.results
    }
    
    /**
     * 搜索记忆
     * 返回搜索结果
     */
    suspend fun searchMemories(request: SearchRequest): SearchResponse {
        val response = client.post("$baseUrl/search") {
            contentType(ContentType.Application.Json)
            setBody(request)
        }
        
        if (!response.status.isSuccess()) {
            throw Exception("搜索记忆失败: ${response.status} - ${response.bodyAsText()}")
        }
        
        return response.body()
    }
    
    /**
     * 删除记忆
     */
    suspend fun deleteMemory(memoryId: String): Boolean {
        val response = client.delete("$baseUrl/memories/$memoryId")
        return response.status.isSuccess()
    }
    
    /**
     * 删除所有记忆
     */
    suspend fun deleteAllMemories(userId: String? = null): Boolean {
        val url = buildString {
            append("$baseUrl/memories")
            if (userId != null) {
                append("?user_id=$userId")
            }
        }
        
        val response = client.delete(url)
        return response.status.isSuccess()
    }
    
    /**
     * 重置所有记忆
     */
    suspend fun resetAllMemories(): Boolean {
        val response = client.post("$baseUrl/reset") {
            contentType(ContentType.Application.Json)
        }
        return response.status.isSuccess()
    }
    
    /**
     * 健康检查
     */
    suspend fun healthCheck(): Boolean {
        return try {
            val response = client.get(baseUrl)
            response.status.isSuccess()
        } catch (e: Exception) {
            false
        }
    }
    
    /**
     * 关闭客户端
     */
    fun close() {
        client.close()
    }
}

/**
 * 记忆管理器
 * 提供高级记忆管理功能，支持隔离和共享
 */
class MemoryManager(
    private val mem0Client: Mem0Client
) {
    /**
     * 为 AI 角色创建私有记忆
     */
    suspend fun createPrivateMemory(
        roleId: String,
        messages: List<Message>,
        additionalMetadata: Map<String, String> = emptyMap()
    ): List<MemoryResult> {
        val request = CreateMemoryRequest(
            messages = messages,
            userId = roleId,
            metadata = mapOf(
                "roleId" to roleId,
                "memoryType" to "private"
            ) + additionalMetadata
        )
        
        return mem0Client.createMemory(request)
    }
    
    /**
     * 为群聊创建共享记忆
     * 使用 agent_id 作为群组标识符
     */
    suspend fun createSharedMemory(
        groupId: String,
        messages: List<Message>,
        additionalMetadata: Map<String, String> = emptyMap()
    ): List<MemoryResult> {
        val request = CreateMemoryRequest(
            messages = messages,
            agentId = groupId,
            metadata = mapOf(
                "groupId" to groupId,
                "memoryType" to "shared"
            ) + additionalMetadata
        )
        
        return mem0Client.createMemory(request)
    }
    
    /**
     * 获取角色的私有记忆
     */
    suspend fun getPrivateMemories(roleId: String): List<Memory> {
        return mem0Client.getAllMemories(userId = roleId)
    }
    
    /**
     * 搜索角色的私有记忆
     */
    suspend fun searchPrivateMemories(
        roleId: String,
        query: String,
        limit: Int = 10
    ): List<Memory> {
        val request = SearchRequest(
            query = query,
            userId = roleId,
            limit = limit
        )
        
        return mem0Client.searchMemories(request).results
    }
    
    /**
     * 搜索群聊共享记忆
     * 使用 agent_id 作为群组标识符
     */
    suspend fun searchSharedMemories(
        groupId: String,
        query: String,
        limit: Int = 10
    ): List<Memory> {
        val request = SearchRequest(
            query = query,
            agentId = groupId,
            limit = limit
        )
        
        return mem0Client.searchMemories(request).results
    }
    
    /**
     * 搜索混合记忆（私有 + 共享）
     */
    suspend fun searchHybridMemories(
        roleId: String,
        groupId: String,
        query: String,
        limit: Int = 10
    ): Pair<List<Memory>, List<Memory>> {
        val privateMemories = searchPrivateMemories(roleId, query, limit)
        val sharedMemories = searchSharedMemories(groupId, query, limit)
        
        return Pair(privateMemories, sharedMemories)
    }
    
    /**
     * 删除角色的所有私有记忆
     */
    suspend fun deletePrivateMemories(roleId: String): Boolean {
        return mem0Client.deleteAllMemories(userId = roleId)
    }
}
