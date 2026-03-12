# AI 角色和会话系统 - 设计需求

> 基于用户口述需求整理  
> 创建时间：2026-03-10  
> 最后更新时间：2026-03-12  
> 状态：会话压缩功能设计完成，待进行可行性测试
> 
> ## 更新历史
> - 2026-03-12: 细化会话压缩功能，区分自动压缩和用户主动压缩两种触发方式，实现不同的压缩提示词和后续处理逻辑

## 需求来源

详见 [`docs/user_say.md`](../user_say.md#2026-03-10-ai-角色和会话系统详细需求)

## 需求概述

### 核心架构

系统采用多 AI 角色架构，每个角色具有独立的工作区、记忆、配置和模型选择。支持用户与单个 AI 角色的一对一会话，以及多个 AI 角色参与的群聊会话。

### 关键技术决策

1. **后端技术栈**: Kotlin/JVM（利用 JVM 生态系统的成熟库）
2. **网络框架**: Ktor（支持 JVM，异步非阻塞）
3. **异步处理**: Kotlin 协程
4. **记忆系统**: mem0（独立进程，REST API 调用）
5. **数据存储**: 统一使用 mem0 存储所有数据（会话历史、记忆、配置等）
6. **限流算法**: 令牌桶算法（替代疲劳值系统）

## 详细设计

### 1. AI 角色工作区设计

#### 1.1 文件系统结构

```
workspaces/
└── {ai_role_id}/
    ├── config/
    │   ├── profile.json          # 角色配置（名称、描述、头像等）
    │   ├── system_prompt.md      # 系统提示词
    │   ├── behavior_rules.md     # 行为准则和约束
    │   └── model_config.json     # 模型配置（model_id、参数等）
    ├── skills/
    │   ├── whitelist.json        # 技能白名单
    │   ├── blacklist.json        # 技能黑名单
    │   ├── installed/            # 已安装技能
    │   │   ├── skill_1/
    │   │   ├── skill_2/
    │   │   └── ...
    │   └── custom/               # 角色自定义技能
    │       ├── custom_skill_1/
    │       │   ├── manifest.json
    │       │   ├── main.py
    │       │   └── versions/
    │       └── ...
    ├── tools/                    # 工具配置和状态
    │   ├── browser/              # 浏览器数据
    │   │   └── user_data/        # 独立的浏览器用户数据
    │   ├── python_venv/          # Python虚拟环境（技能执行用）
    │   └── tool_state.json       # 工具状态（CDP端口等）
    └── token_bucket.json         # 令牌桶状态数据
```

#### 1.2 会话架构设计

**会话类型**

系统采用固定会话架构：

1. **一对一会话**：每个 AI 角色创建后，自动与用户建立一个永久会话
2. **群聊会话**：所有 AI 角色默认加入群聊会话

**会话规则**

- AI 角色创建后，永久存在于系统中
- 每个 AI 角色与用户拥有且仅拥有一个一对一会话
- 所有 AI 角色自动加入群聊会话
- 会话不存在"结束"状态，只有"存在"状态
- 会话历史统一存储在 mem0 中

#### 1.3 角色配置数据类

```kotlin
data class AIRoleConfig(
    val id: String,
    val name: String,
    val description: String,
    val avatarPath: String?,
    val personality: PersonalityType,
    val modelId: String,
    val systemPrompt: String,
    val behaviorRules: List<String>,
    val skills: SkillConfig = SkillConfig(),
    val createdAt: Instant,
    val updatedAt: Instant
)

data class SkillConfig(
    val whitelist: List<String> = emptyList(),
    val blacklist: List<String> = emptyList(),
    val installedSkills: List<InstalledSkill> = emptyList()
)

enum class PersonalityType {
    FRIENDLY,           // 友好型
    PROFESSIONAL,       // 专业型
    CREATIVE,           // 创意型
    ANALYTICAL,         // 分析型
    EMPATHETIC,         // 共情型
    HUMOROUS,           // 幽默型
    // 更多模板...
}
```

#### 1.4 会话数据模型

**会话信息**

```kotlin
/**
 * 会话信息
 */
data class SessionInfo(
    val id: String,                    // 会话ID
    val type: SessionType,             // 会话类型
    val participants: List<String>,    // 参与者角色ID列表
    val createdAt: Instant,            // 创建时间
    val updatedAt: Instant             // 最后更新时间
)

enum class SessionType {
    ONE_ON_ONE,     // 一对一会话
    GROUP_CHAT      // 群聊会话
}
```

**消息模型**

```kotlin
/**
 * 消息数据类
 */
data class Message(
    val id: String,                    // 消息唯一ID
    val senderId: String,              // 发送者ID（用户或AI角色ID）
    val senderType: SenderType,        // 发送者类型
    val sessionId: String,             // 所属会话ID
    val content: MessageContent,       // 消息内容
    val timestamp: Instant,            // 发送时间
    val status: MessageStatus,         // 消息状态
    val replyTo: String? = null,       // 回复的消息ID（可选）
    val mentions: List<String> = emptyList()  // @提及的角色ID列表
)

enum class SenderType {
    USER,           // 用户
    AI_ROLE,        // AI角色
    SYSTEM          // 系统消息
}

/**
 * 消息内容
 */
sealed class MessageContent {
    data class Text(val text: String) : MessageContent()
    data class Image(val url: String, val description: String? = null) : MessageContent()
    data class Link(val url: String, val title: String? = null) : MessageContent()
    data class File(val fileName: String, val filePath: String, val fileSize: Long) : MessageContent()
    data class Audio(val url: String, val duration: Int? = null) : MessageContent()
}

enum class MessageStatus {
    COMPLETE,       // 完整消息
    PARTIAL         // 未完全生成消息（不可删除）
}
```

**会话状态（针对AI角色与用户的会话关系）**

```kotlin
/**
 * AI角色与会话的交互状态
 * 用于记录AI是否已读消息、是否正在处理等
 */
data class SessionInteractionState(
    val roleId: String,                // AI角色ID
    val sessionId: String,             // 会话ID
    val lastReadMessageId: String?,    // 最后已读消息ID
    val hasUnreadMessages: Boolean,    // 是否有未读消息
    val isProcessing: Boolean,         // 是否正在处理中（正在生成回应）
    val processingMessageId: String?,  // 正在处理的消息ID
    val lastActivityAt: Instant        // 最后活动时间
)

/**
 * 会话状态管理器
 */
class SessionStateManager(
    private val mem0Client: Mem0Client
) {
    /**
     * 标记消息已读
     */
    suspend fun markAsRead(roleId: String, sessionId: String, messageId: String) {
        val state = getState(roleId, sessionId)
        val newState = state.copy(
            lastReadMessageId = messageId,
            hasUnreadMessages = false,
            lastActivityAt = Instant.now()
        )
        saveState(newState)
    }
    
    /**
     * 开始处理消息
     */
    suspend fun startProcessing(roleId: String, sessionId: String, messageId: String) {
        val state = getState(roleId, sessionId)
        val newState = state.copy(
            isProcessing = true,
            processingMessageId = messageId,
            lastActivityAt = Instant.now()
        )
        saveState(newState)
    }
    
    /**
     * 结束处理消息
     */
    suspend fun finishProcessing(roleId: String, sessionId: String) {
        val state = getState(roleId, sessionId)
        val newState = state.copy(
            isProcessing = false,
            processingMessageId = null,
            lastActivityAt = Instant.now()
        )
        saveState(newState)
    }
    
    /**
     * 新消息到达时更新未读状态
     */
    suspend fun onNewMessage(sessionId: String, messageId: String) {
        // 获取该会话的所有参与者
        val participants = getSessionParticipants(sessionId)
        
        participants.forEach { roleId ->
            val state = getState(roleId, sessionId)
            if (state.lastReadMessageId != messageId) {
                saveState(state.copy(hasUnreadMessages = true))
            }
        }
    }
    
    private suspend fun getState(roleId: String, sessionId: String): SessionInteractionState {
        // 从mem0读取状态
        val memories = mem0Client.searchMemories(
            userId = "state_$roleId",
            query = "session_state_$sessionId",
            limit = 1
        )
        
        return if (memories.isNotEmpty()) {
            Json.decodeFromString(memories.first().text)
        } else {
            SessionInteractionState(
                roleId = roleId,
                sessionId = sessionId,
                lastReadMessageId = null,
                hasUnreadMessages = false,
                isProcessing = false,
                processingMessageId = null,
                lastActivityAt = Instant.now()
            )
        }
    }
    
    private suspend fun saveState(state: SessionInteractionState) {
        mem0Client.addMemory(
            userId = "state_${state.roleId}",
            data = MemoryData(
                text = Json.encodeToString(state),
                metadata = mapOf(
                    "type" to "session_state",
                    "sessionId" to state.sessionId,
                    "timestamp" to Instant.now().toString()
                )
            )
        )
    }
}
```

### 2. 错误处理和重试机制

#### 2.1 模型请求失败重试

**网络错误重试策略**

```kotlin
/**
 * 模型请求重试管理器
 */
class ModelRequestRetryManager {
    companion object {
        const val RETRY_INTERVAL_MINUTES = 5L  // 每5分钟重试一次
        const val MAX_RETRY_DURATION_HOURS = 1L  // 持续一小时
    }
    
    private val retryJobs = ConcurrentHashMap<String, Job>()
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    
    /**
     * 执行带重试的模型请求
     * @param requestId 请求唯一标识
     * @param request 请求执行块
     */
    suspend fun <T> executeWithRetry(
        requestId: String,
        request: suspend () -> T
    ): T {
        val startTime = Instant.now()
        val maxDuration = Duration.ofHours(MAX_RETRY_DURATION_HOURS)
        
        while (Duration.between(startTime, Instant.now()) < maxDuration) {
            try {
                return request()
            } catch (e: NetworkException) {
                // 网络错误，等待后重试
                Logger.warn("模型请求网络错误，将在 ${RETRY_INTERVAL_MINUTES} 分钟后重试: ${e.message}")
                delay(Duration.ofMinutes(RETRY_INTERVAL_MINUTES).toMillis())
            } catch (e: Exception) {
                // 其他错误，直接抛出
                throw e
            }
        }
        
        throw RetryExhaustedException("模型请求在 ${MAX_RETRY_DURATION_HOURS} 小时内重试耗尽")
    }
    
    /**
     * 取消指定请求的重试
     */
    fun cancelRetry(requestId: String) {
        retryJobs[requestId]?.cancel()
        retryJobs.remove(requestId)
    }
}

class RetryExhaustedException(message: String) : Exception(message)
```

#### 2.2 工具调用失败处理

**工具错误重试机制**

```kotlin
/**
 * 工具调用重试处理器
 */
class ToolCallRetryHandler {
    /**
     * 处理工具调用失败
     * 在会话中重新附加工具错误原因和使用说明后再次请求
     */
    suspend fun handleToolFailure(
        originalRequest: AIRequest,
        toolError: ToolExecutionError,
        sessionMessages: MutableList<Message>,
        requestAI: suspend (AIRequest) -> AIResponse
    ): AIResponse {
        // 构建错误提示
        val errorPrompt = buildString {
            appendLine("=== 工具调用失败 ===")
            appendLine("工具: ${toolError.toolName}")
            appendLine("错误: ${toolError.errorMessage}")
            appendLine()
            appendLine("=== 工具使用说明 ===")
            appendLine(getToolUsageGuide(toolError.toolName))
            appendLine()
            appendLine("请根据错误信息和说明重新尝试，或采取其他方式完成任务。")
        }
        
        // 将错误信息添加到会话上下文
        val errorMessage = Message(
            id = generateId(),
            senderId = "system",
            senderType = SenderType.SYSTEM,
            sessionId = originalRequest.sessionId,
            content = MessageContent.Text(errorPrompt),
            timestamp = Instant.now(),
            status = MessageStatus.COMPLETE
        )
        sessionMessages.add(errorMessage)
        
        // 重新构造请求，包含错误上下文
        val retryRequest = originalRequest.copy(
            messages = sessionMessages,
            retryCount = originalRequest.retryCount + 1
        )
        
        // 再次请求
        return requestAI(retryRequest)
    }
    
    private fun getToolUsageGuide(toolName: String): String {
        return when (toolName) {
            "read" -> """
                read 工具使用说明:
                - path: 文件路径（相对于工作区）
                - offset: 起始行号（可选，从1开始）
                - limit: 读取行数（可选）
                注意: 只能读取工作区内的文件
            """.trimIndent()
            "find" -> """
                find 工具使用说明:
                - pattern: 通配符模式，如 "*.kt", "**/*.md"
                - path: 搜索起始路径（可选，默认为当前目录）
            """.trimIndent()
            "edit" -> """
                edit 工具使用说明:
                - path: 文件路径
                - oldString: 原文内容（用于定位，必须唯一匹配）
                - newString: 新内容（用于替换）
                注意: oldString 必须在文件中有且仅有一处匹配
            """.trimIndent()
            else -> "请参考工具定义获取使用说明"
        }
    }
}

data class ToolExecutionError(
    val toolName: String,
    val errorMessage: String,
    val originalParams: Map<String, JsonElement>
)
```

#### 2.3 mem0 不可用处理

**mem0 服务监控和退出机制**

```kotlin
/**
 * mem0 服务健康检查器
 */
class Mem0HealthChecker(
    private val mem0Client: Mem0Client,
    private val checkInterval: Duration = Duration.ofSeconds(30)
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private var healthCheckJob: Job? = null
    
    /**
     * 启动健康检查
     */
    fun start() {
        healthCheckJob = scope.launch {
            while (isActive) {
                if (!checkHealth()) {
                    // mem0 不可用，终止程序
                    Logger.error("mem0 服务不可用，程序即将终止")
                    shutdownApplication()
                }
                delay(checkInterval.toMillis())
            }
        }
    }
    
    /**
     * 检查 mem0 健康状态
     */
    private suspend fun checkHealth(): Boolean {
        return try {
            // 尝试一个简单的API调用来检查服务状态
            mem0Client.healthCheck()
            true
        } catch (e: Exception) {
            Logger.error("mem0 健康检查失败: ${e.message}")
            false
        }
    }
    
    /**
     * 终止应用程序
     */
    private fun shutdownApplication() {
        // 终止所有任务
        Runtime.getRuntime().exit(1)
    }
    
    fun stop() {
        healthCheckJob?.cancel()
    }
}

interface Mem0Client {
    /**
     * 健康检查
     */
    suspend fun healthCheck(): Boolean
    
    /**
     * 添加记忆
     */
    suspend fun addMemory(userId: String, data: MemoryData): Memory
    
    /**
     * 搜索记忆
     */
    suspend fun searchMemories(userId: String, query: String, limit: Int = 10): List<Memory>
    
    /**
     * 获取所有记忆
     */
    suspend fun getAllMemories(userId: String, limit: Int? = null): List<Memory>
}
```

### 3. AI 群聊调度机制

#### 3.1 调度架构

```
用户消息
  ↓
消息分发器
  ↓
┌──────────────────────────────────────┐
│  调度器 AI                            │
│  - 拥有群聊记忆                       │
│  - 分析话题，识别相关角色             │
│  - 处理@提及                          │
└──────────────────────────────────────┘
  ↓
遍历选中的 AI 角色
  ↓
┌──────────────────────────────────────┐
│  角色 A: 是否需要回应？               │
│  角色 B: 是否需要回应？               │
│  角色 C: 是否需要回应？               │
└──────────────────────────────────────┘
  ↓
收集回应，按顺序发送
```

#### 3.2 调度器 AI 设计

**调度器特性**

- 调度器自身**无疲劳值/令牌桶限制**
- 调度器**拥有群聊记忆**，可以访问群聊历史
- 调度器根据话题相关性选择角色（曾经发表过相关话题的角色优先）
- 支持 **@提及机制**：用户或AI角色可以@一个或多个角色触发回应

**实现方式**:

```kotlin
/**
 * 调度器上下文
 */
data class DispatcherContext(
    val message: Message,                              // 当前消息
    val recentMessages: List<Message>,                 // 最近上下文
    val allAvailableRoles: List<AIRoleConfig>,         // 所有可用角色
    val mentionedRoles: List<String>,                  // @提及的角色ID
    val groupChatMemories: List<Memory>                // 群聊相关记忆
)

/**
 * 调度结果
 */
data class DispatchResult(
    val selectedRoles: List<String>,                   // 选中的角色ID列表
    val reason: String                                 // 选择原因
)

/**
 * 群聊调度器
 */
class GroupChatDispatcher(
    private val mem0Client: Mem0Client,
    private val dispatcherRoleConfig: AIRoleConfig,
    private val aiClient: AIClient
) {
    companion object {
        const val GROUP_CHAT_USER_ID = "group_chat_dispatcher"
    }
    
    /**
     * 分析消息并选择参与讨论的角色
     */
    suspend fun dispatch(context: DispatcherContext): DispatchResult {
        // 1. 优先处理@提及
        if (context.mentionedRoles.isNotEmpty()) {
            return DispatchResult(
                selectedRoles = context.mentionedRoles,
                reason = "用户/角色明确@提及"
            )
        }
        
        // 2. 使用调度器AI分析话题
        val analysisPrompt = buildAnalysisPrompt(context)
        val response = aiClient.generate(
            modelId = dispatcherRoleConfig.modelId,
            systemPrompt = dispatcherRoleConfig.systemPrompt,
            userPrompt = analysisPrompt
        )
        
        // 3. 解析AI响应，获取选中的角色
        val selectedRoles = parseSelectedRoles(response.text)
        
        return DispatchResult(
            selectedRoles = selectedRoles,
            reason = "调度器AI分析话题相关性"
        )
    }
    
    /**
     * 构建分析提示词
     */
    private fun buildAnalysisPrompt(context: DispatcherContext): String {
        return buildString {
            appendLine("请分析以下群聊消息，决定哪些AI角色应该参与回应。")
            appendLine()
            appendLine("=== 当前消息 ===")
            appendLine(formatMessage(context.message))
            appendLine()
            appendLine("=== 最近上下文 ===")
            context.recentMessages.takeLast(10).forEach {
                appendLine(formatMessage(it))
            }
            appendLine()
            appendLine("=== 群聊相关记忆 ===")
            context.groupChatMemories.take(5).forEach {
                appendLine("- ${it.text}")
            }
            appendLine()
            appendLine("=== 可用角色 ===")
            context.allAvailableRoles.forEach { role ->
                appendLine("- ${role.id}: ${role.name} - ${role.description}")
            }
            appendLine()
            appendLine("=== 任务 ===")
            appendLine("1. 分析消息话题")
            appendLine("2. 根据话题相关性选择应该回应的角色（曾经讨论过相关话题的角色优先）")
            appendLine("3. 返回选中的角色ID列表，格式: [\"role_id_1\", \"role_id_2\"]")
            appendLine("4. 如果没有角色适合回应，返回空列表 []")
        }
    }
    
    private fun formatMessage(message: Message): String {
        val sender = when (message.senderType) {
            SenderType.USER -> "用户"
            SenderType.AI_ROLE -> message.senderId
            SenderType.SYSTEM -> "系统"
        }
        return "[$sender]: ${message.content}"
    }
    
    private fun parseSelectedRoles(response: String): List<String> {
        // 从AI响应中解析角色ID列表
        val regex = Regex("\\[([^\\]]*)\\]")
        val match = regex.find(response)
        return match?.groupValues?.get(1)
            ?.split(",")
            ?.map { it.trim().trim('"', '\'') }
            ?.filter { it.isNotEmpty() }
            ?: emptyList()
    }
    
    /**
     * 获取群聊记忆
     */
    suspend fun getGroupChatMemories(query: String, limit: Int = 10): List<Memory> {
        return mem0Client.searchMemories(
            userId = GROUP_CHAT_USER_ID,
            query = query,
            limit = limit
        )
    }
    
    /**
     * 存储群聊记忆
     */
    suspend fun storeGroupChatMemory(content: String, metadata: Map<String, String> = emptyMap()) {
        mem0Client.addMemory(
            userId = GROUP_CHAT_USER_ID,
            data = MemoryData(
                text = content,
                metadata = metadata + mapOf("type" to "group_chat", "timestamp" to Instant.now().toString())
            )
        )
    }
}

/**
 * @提及解析器
 */
object MentionParser {
    private val mentionRegex = Regex("@([a-zA-Z0-9_]+)")
    
    /**
     * 解析消息中的@提及
     * @param content 消息内容
     * @param availableRoles 所有可用角色ID列表
     * @return 被提及的角色ID列表
     */
    fun parse(content: String, availableRoles: List<String>): List<String> {
        val mentions = mentionRegex.findAll(content).map { it.groupValues[1] }.toList()
        return mentions.filter { it in availableRoles }
    }
}
```

#### 3.3 令牌桶限流配置

```hocon
groupChat {
    # 调度器配置
    dispatcher {
        enabled = true
        dispatcherRoleId = "dispatcher_001"
    }
    
    # 令牌桶限流配置
    tokenBucket {
        # 全局令牌桶（限制整个群聊的响应频率）
        global {
            capacity = 10           # 桶容量（最大令牌数）
            refillRate = 2          # 每秒补充令牌数
        }
        
        # 角色级别令牌桶
        perRole {
            capacity = 5            # 每个角色的桶容量
            refillRate = 0.5        # 每秒补充令牌数（每2秒1个）
        }
    }
}
```
```

### 4. 令牌桶限流系统

#### 4.1 令牌桶算法实现

```kotlin
/**
 * 令牌桶状态
 */
data class TokenBucketState(
    val roleId: String,                  // 角色ID
    var tokens: Double,                  // 当前令牌数
    val capacity: Double,                // 桶容量（最大令牌数）
    val refillRate: Double,              // 每秒补充令牌数
    var lastRefillTime: Instant          // 最后补充时间
)

/**
 * 令牌桶限流器
 */
class TokenBucketRateLimiter(
    private val statePath: Path
) {
    private val buckets = ConcurrentHashMap<String, TokenBucketState>()
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    
    /**
     * 初始化角色令牌桶
     * @param roleId 角色ID
     * @param capacity 桶容量
     * @param refillRate 每秒补充令牌数
     */
    suspend fun initBucket(
        roleId: String,
        capacity: Double,
        refillRate: Double
    ) {
        val state = loadState(roleId)
        val bucket = if (state != null) {
            // 恢复已有状态，但更新配置
            state.copy(capacity = capacity, refillRate = refillRate)
        } else {
            TokenBucketState(
                roleId = roleId,
                tokens = capacity,  // 初始满桶
                capacity = capacity,
                refillRate = refillRate,
                lastRefillTime = Instant.now()
            )
        }
        buckets[roleId] = bucket
        saveState(bucket)
    }
    
    /**
     * 尝试消费令牌
     * @param roleId 角色ID
     * @param tokens 需要消费的令牌数（默认为1）
     * @return 是否消费成功
     */
    suspend fun tryConsume(roleId: String, tokens: Double = 1.0): Boolean {
        val bucket = buckets[roleId] ?: return false
        
        // 先补充令牌
        refill(bucket)
        
        return if (bucket.tokens >= tokens) {
            bucket.tokens -= tokens
            saveState(bucket)
            true
        } else {
            false
        }
    }
    
    /**
     * 获取当前令牌数
     */
    fun getAvailableTokens(roleId: String): Double {
        val bucket = buckets[roleId] ?: return 0.0
        refill(bucket)
        return bucket.tokens
    }
    
    /**
     * 补充令牌
     */
    private fun refill(bucket: TokenBucketState) {
        val now = Instant.now()
        val elapsedSeconds = Duration.between(bucket.lastRefillTime, now).toMillis() / 1000.0
        
        if (elapsedSeconds > 0) {
            val tokensToAdd = elapsedSeconds * bucket.refillRate
            bucket.tokens = minOf(bucket.capacity, bucket.tokens + tokensToAdd)
            bucket.lastRefillTime = now
        }
    }
    
    /**
     * 从文件加载状态
     */
    private suspend fun loadState(roleId: String): TokenBucketState? {
        return withContext(Dispatchers.IO) {
            val file = statePath.resolve("$roleId.json")
            if (Files.exists(file)) {
                try {
                    val json = Files.readString(file)
                    Json.decodeFromString<TokenBucketState>(json)
                } catch (e: Exception) {
                    null
                }
            } else {
                null
            }
        }
    }
    
    /**
     * 保存状态到文件
     */
    private suspend fun saveState(state: TokenBucketState) {
        withContext(Dispatchers.IO) {
            Files.createDirectories(statePath)
            val file = statePath.resolve("${state.roleId}.json")
            Files.writeString(file, Json.encodeToString(state))
        }
    }
    
    /**
     * 定期持久化所有桶状态
     */
    fun startPersistenceJob(interval: Duration = Duration.ofMinutes(1)) {
        scope.launch {
            while (isActive) {
                delay(interval.toMillis())
                buckets.values.forEach { saveState(it) }
            }
        }
    }
}
```

#### 4.2 令牌桶配置和提示词注入

```kotlin
/**
 * 构建包含令牌桶状态的系统提示词
 */
fun buildSystemPromptWithTokenBucket(
    basePrompt: String,
    availableTokens: Double,
    capacity: Double
): String {
    val tokenRatio = availableTokens / capacity
    
    val availabilityInstruction = when {
        tokenRatio > 0.7 -> 
            "你当前精力充沛，可以积极回应。"
        tokenRatio > 0.3 -> 
            "你当前状态正常，可以适度回应。"
        tokenRatio > 0.1 -> 
            "你当前有些疲劳，建议减少回应频率。"
        else -> 
            "你当前非常疲劳，建议简短回应或等待恢复。"
    }
    
    return """
        $basePrompt
        
        === 当前状态 ===
        可用响应额度：${availableTokens.toInt()}/${capacity.toInt()}
        $availabilityInstruction
    """.trimIndent()
}

/**
 * 构建工具使用约束提示词
 * 告知AI角色工具使用的安全约束
 */
fun buildToolConstraintsPrompt(): String {
    return """
        === 工具使用约束 ===
        
        你拥有以下基础工具能力：
        1. read - 读取工作区内的文件内容
        2. find - 在工作区内查找文件
        3. edit - 修改工作区内的文件内容
        4. delete - 删除工作区内的文件或目录（软删除到回收站）
        5. browser - 浏览器控制工具（导航、获取内容、执行脚本、截图等）
        6. skill - 创建和管理自定义Python技能
        7. memory - 从mem0检索你的记忆
        8. selfAware - 查看和修改你的认知和行为准则
        
        重要约束：
        - 所有文件操作仅限于你的工作区内，禁止访问工作区外的文件系统
        - 浏览器工具仅用于合法的信息获取和自动化操作，禁止用于恶意攻击
        - Python技能创建约束：
          * 禁止执行越界行为（如访问系统文件、网络攻击等）
          * 技能应专注于完成特定任务
          * 禁止在技能中执行有害操作或恶意代码
          * 技能代码将在独立进程中执行，但请自觉遵守安全规范
        - 记忆查找仅用于获取与你相关的上下文信息
        - 自我认知修改仅限于可配置属性（name, description, personality, behaviorRules, systemPrompt）
          核心属性（id, createdAt）不可修改
        
        请负责任地使用这些工具，专注于帮助用户完成任务。
    """.trimIndent()
}
```

### 5. 实时打断功能

#### 5.1 打断架构

```kotlin
class AIExecutionContext(
    private val roleId: String,
    private val sessionId: String
) {
    // 协程 Job，用于取消模型请求
    private var currentJob: Job? = null
    
    // 正在执行的工具调用
    private val runningTools = ConcurrentHashMap<String, ToolExecution>()
    
    // 子进程管理（用于 shell 命令等）
    private val childProcesses = ConcurrentHashMap<String, Process>()
    
    // 中断信号通道
    private val interruptionChannel = Channel<Unit>(Channel.CONFLATED)
    
    /**
     * 中断所有执行
     */
    suspend fun interruptAll() {
        Logger.info("中断AI角色[$roleId]在会话[$sessionId]中的所有执行")
        
        // 1. 发送中断信号
        interruptionChannel.trySend(Unit)
        
        // 2. 取消模型请求
        currentJob?.cancel()
        currentJob = null
        
        // 3. 中断工具调用
        runningTools.values.forEach { it.cancel() }
        runningTools.clear()
        
        // 4. 终止子进程
        childProcesses.values.forEach { process ->
            try {
                process.destroyForcibly()
            } catch (e: Exception) {
                Logger.warn("终止子进程失败: ${e.message}")
            }
        }
        childProcesses.clear()
    }
    
    /**
     * 执行模型请求（可中断）
     */
    suspend fun executeModelRequest(
        request: AIRequest,
        block: suspend () -> AIResponse
    ): AIResponse {
        return coroutineScope {
            val job = launch {
                try {
                    block()
                } catch (e: CancellationException) {
                    // 被中断，清理部分回复
                    cleanupPartialResponse()
                    throw e
                }
            }
            
            currentJob = job
            
            // 监听中断信号
            val interruptionListener = launch {
                interruptionChannel.receive()
                job.cancel()
            }
            
            try {
                job.join()
                job.getCompletionOrNull() ?: throw CancellationException()
            } finally {
                currentJob = null
                interruptionListener.cancel()
            }
        }
    }
    
    /**
     * 注册工具执行
     */
    fun registerToolExecution(toolId: String, execution: ToolExecution) {
        runningTools[toolId] = execution
    }
    
    /**
     * 注册子进程
     */
    fun registerChildProcess(processId: String, process: Process) {
        childProcesses[processId] = process
    }
    
    /**
     * 清理部分回复
     */
    private suspend fun cleanupPartialResponse() {
        // 通知会话管理器清理该角色的部分回复
        SessionManager.markPartialMessagesAsInterrupted(roleId, sessionId)
    }
}

/**
 * 工具执行接口
 */
interface ToolExecution {
    fun cancel()
    suspend fun execute(): Any
}
```
```

#### 4.2 消息撤回机制

```kotlin
interface MessageManager {
    /**
     * 撤回消息
     */
    suspend fun recallMessage(messageId: String): Boolean
    
    /**
     * 撤回 AI 的所有部分回复
     */
    suspend fun recallPartialResponses(roleId: String, sessionId: String): List<String>
}

class MessageManagerImpl(
    private val sessionStorage: SessionStorage
) : MessageManager {
    override suspend fun recallMessage(messageId: String): Boolean {
        return sessionStorage.markAsRecalled(messageId)
    }
    
    override suspend fun recallPartialResponses(
        roleId: String, 
        sessionId: String
    ): List<String> {
        val session = sessionStorage.getSession(sessionId)
        val partialMessages = session.messages
            .filter { it.roleId == roleId && it.status == MessageStatus.PARTIAL }
        
        partialMessages.forEach { 
            sessionStorage.markAsRecalled(it.id)
        }
        
        return partialMessages.map { it.id }
    }
}
```

#### 4.3 打断事件总线

```kotlin
class InterruptionEventBus {
    private val scope = CoroutineScope(Dispatchers.Default + SupervisorJob())
    private val channel = Channel<InterruptionEvent>(Channel.BUFFERED)
    
    // 用户消息触发打断
    suspend fun onUserMessage(userId: String, sessionId: String) {
        channel.send(InterruptionEvent.UserMessage(userId, sessionId))
    }
    
    // AI 监听打断事件
    fun subscribe(roleId: String): Flow<InterruptionEvent> {
        return channel.consumeAsFlow()
            .filter { it.shouldInterrupt(roleId) }
    }
}

sealed class InterruptionEvent {
    data class UserMessage(val userId: String, val sessionId: String) : InterruptionEvent()
    
    fun shouldInterrupt(roleId: String): Boolean {
        return this is UserMessage  // 只有用户消息能打断 AI
    }
}
```

### 5. 技能黑白名单系统

#### 5.1 技能管理架构

```kotlin
data class InstalledSkill(
    val id: String,
    val name: String,
    val version: String,
    val path: Path,
    val manifest: SkillManifest,
    val installedAt: Instant
)

data class SkillManifest(
    val id: String,
    val name: String,
    val description: String,
    val version: String,
    val requiredPermissions: List<Permission>,
    val entryPoint: String  // 入口函数或脚本
)

class SkillManager(private val roleWorkspace: Path) {
    private val skillsPath = roleWorkspace.resolve("skills/installed")
    private val whitelistPath = roleWorkspace.resolve("skills/whitelist.json")
    private val blacklistPath = roleWorkspace.resolve("skills/blacklist.json")
    
    /**
     * 检查技能是否可用
     */
    suspend fun isSkillAvailable(skillId: String): Boolean {
        val whitelist = loadWhitelist()
        val blacklist = loadBlacklist()
        
        // 白名单优先
        if (whitelist.isNotEmpty()) {
            return skillId in whitelist
        }
        
        // 没有白名单时，检查黑名单
        return skillId !in blacklist
    }
    
    /**
     * 获取可用的技能列表
     */
    suspend fun getAvailableSkills(): List<InstalledSkill> {
        val allSkills = listInstalledSkills()
        return allSkills.filter { isSkillAvailable(it.id) }
    }
    
    /**
     * 安装技能（不检查黑白名单，由使用时检查）
     */
    suspend fun installSkill(skillPackage: Path): InstalledSkill {
        // 解压、验证、安装技能
        // 不阻止安装，使用时才检查黑白名单
    }
}
```

#### 5.2 技能调用拦截器

```kotlin
class SkillCallInterceptor(
    private val skillManager: SkillManager
) {
    /**
     * 拦截技能调用，检查黑白名单
     */
    suspend fun intercept(
        roleId: String,
        skillId: String,
        args: Map<String, Any>
    ): Any? {
        if (!skillManager.isSkillAvailable(skillId)) {
            throw SkillUnavailableException(
                "技能 '$skillId' 不可用（受黑白名单限制）"
            )
        }
        
        return executeSkill(roleId, skillId, args)
    }
}

class SkillUnavailableException(message: String) : Exception(message)
```

### 6. 模型选择限制

#### 6.1 模型验证器

```kotlin
data class ModelCapability(
    val modelId: String,
    val contextLength: Int,          // 上下文长度（tokens）
    val supportsMultimodal: Boolean, // 是否支持多模态（图片）
    val supportsToolCalling: Boolean, // 是否支持工具调用
    val provider: String             // 提供者（如 LM Studio）
)

class ModelValidator(private val lmStudioClient: LMStudioClient) {
    companion object {
        const val MIN_CONTEXT_LENGTH = 100_000  // 至少 100K
    }
    
    /**
     * 验证模型是否满足要求
     */
    suspend fun validateModel(modelId: String): ModelValidationResult {
        val capability = lmStudioClient.getModelCapability(modelId)
        
        val errors = mutableListOf<String>()
        
        if (capability.contextLength < MIN_CONTEXT_LENGTH) {
            errors.add("上下文长度不足：${capability.contextLength} < $MIN_CONTEXT_LENGTH")
        }
        
        if (!capability.supportsMultimodal) {
            errors.add("不支持多模态（图片）")
        }
        
        if (!capability.supportsToolCalling) {
            errors.add("不支持工具调用")
        }
        
        return ModelValidationResult(
            valid = errors.isEmpty(),
            capability = capability,
            errors = errors
        )
    }
    
    /**
     * 获取所有可用的模型（过滤不满足要求的）
     */
    suspend fun getAvailableModels(): List<ModelCapability> {
        val allModels = lmStudioClient.listModels()
        
        return allModels.mapNotNull { model ->
            val result = validateModel(model.id)
            if (result.valid) model else null
        }
    }
}

data class ModelValidationResult(
    val valid: Boolean,
    val capability: ModelCapability,
    val errors: List<String>
)
```

#### 6.2 LM Studio API 集成

```kotlin
interface LMStudioClient {
    /**
     * 获取模型列表
     */
    suspend fun listModels(): List<ModelInfo>
    
    /**
     * 获取模型能力信息
     */
    suspend fun getModelCapability(modelId: String): ModelCapability
}

class LMStudioClientImpl(
    private val baseUrl: String,
    private val httpClient: HttpClient
) : LMStudioClient {
    override suspend fun listModels(): List<ModelInfo> {
        val response = httpClient.get("$baseUrl/api/v1/models")
        return Json.decodeFromString<ModelsResponse>(response).data
    }
    
    override suspend fun getModelCapability(modelId: String): ModelCapability {
        // 方案 1: 从模型信息 API 获取
        val response = httpClient.get("$baseUrl/api/v1/models/$modelId")
        val modelInfo = Json.decodeFromString<ModelInfoResponse>(response)
        
        return ModelCapability(
            modelId = modelId,
            contextLength = modelInfo.context_length ?: modelInfo.max_tokens ?: 0,
            supportsMultimodal = modelInfo.supports_vision ?: false,
            supportsToolCalling = modelInfo.supports_tool_calling ?: false,
            provider = "LM Studio"
        )
    }
}
```

### 6. 统一存储系统（基于mem0）

#### 6.1 存储架构设计

系统统一使用 mem0 作为唯一数据存储，包括：
- 会话历史记录
- AI角色记忆
- 群聊共享记忆
- 会话状态
- 配置历史

#### 6.2 记忆隔离架构

```kotlin
class Mem0Integration(
    private val mem0Client: Mem0Client
) {
    /**
     * 获取 AI 角色的记忆用户 ID（隔离）
     */
    fun getRoleUserId(roleId: String): String {
        return "role_$roleId"
    }
    
    /**
     * 获取群聊记忆用户 ID（共享）
     */
    fun getGroupChatUserId(): String {
        return "group_chat_shared"
    }
    
    /**
     * 获取会话存储用户ID
     */
    fun getSessionUserId(sessionId: String): String {
        return "session_$sessionId"
    }
    
    /**
     * 存储会话消息
     */
    suspend fun storeMessage(message: Message) {
        val userId = getSessionUserId(message.sessionId)
        mem0Client.addMemory(
            userId = userId,
            data = MemoryData(
                text = Json.encodeToString(message),
                metadata = mapOf(
                    "type" to "message",
                    "messageId" to message.id,
                    "senderId" to message.senderId,
                    "timestamp" to message.timestamp.toString()
                )
            )
        )
    }
    
    /**
     * 获取会话历史
     */
    suspend fun getSessionMessages(
        sessionId: String,
        limit: Int = 100,
        before: Instant? = null
    ): List<Message> {
        val userId = getSessionUserId(sessionId)
        val memories = mem0Client.getAllMemories(userId = userId, limit = limit * 2)
        
        return memories
            .filter { it.metadata["type"] == "message" }
            .sortedBy { it.metadata["timestamp"] }
            .takeLast(limit)
            .mapNotNull { memory ->
                try {
                    Json.decodeFromString<Message>(memory.text)
                } catch (e: Exception) {
                    null
                }
            }
    }
    
    /**
     * 存储会话压缩内容
     */
    suspend fun storeCompressionMemory(
        roleId: String,
        sessionId: String,
        summary: String
    ) {
        val userId = getRoleUserId(roleId)
        mem0Client.addMemory(
            userId = userId,
            data = MemoryData(
                text = summary,
                metadata = mapOf(
                    "type" to "session_summary",
                    "sessionId" to sessionId,
                    "timestamp" to Instant.now().toString()
                )
            )
        )
    }
    
    /**
     * 检索相关记忆
     */
    suspend fun searchMemories(
        roleId: String,
        query: String,
        limit: Int = 10
    ): List<Memory> {
        val userId = getRoleUserId(roleId)
        return mem0Client.searchMemories(userId = userId, query = query, limit = limit)
    }
    
    /**
     * 检索群聊记忆
     */
    suspend fun searchGroupChatMemories(
        query: String,
        limit: Int = 10
    ): List<Memory> {
        val userId = getGroupChatUserId()
        return mem0Client.searchMemories(userId = userId, query = query, limit = limit)
    }
}
```

#### 7.2 记忆注入上下文

```kotlin
class MemoryContextInjector(
    private val mem0Integration: Mem0Integration
) {
    /**
     * 在对话前注入相关记忆
     */
    suspend fun injectMemories(
        roleId: String,
        currentMessage: String,
        context: List<Message>
    ): String {
        // 检索相关记忆
        val memories = mem0Integration.searchMemories(
            roleId = roleId,
            query = currentMessage,
            limit = 5
        )
        
        if (memories.isEmpty()) {
            return ""  // 没有记忆，不注入
        }
        
        // 构建记忆上下文
        val memoryContext = memories.joinToString("\n") { memory ->
            "[记忆] ${memory.text} (时间：${memory.timestamp})"
        }
        
        return """
            
            === 相关记忆 ===
            $memoryContext
            """.trimIndent()
    }
}
```

#### 6.3 会话压缩触发机制

**两种压缩触发方式**

系统支持两种会话压缩触发方式，每种方式对应不同的压缩策略和后续处理逻辑：

1. **自动压缩（AUTO）**: 由系统根据 token 阈值自动触发
   - 压缩后程序会自动发送"继续"指令，让 AI 平滑继续对话
   - 使用 AI 口吻结尾，确保对话连续性

2. **用户主动压缩（USER_INITIATED）**: 由用户通过界面或命令触发
   - 压缩后等待用户主动发起下一句话
   - 使用等待用户回应的口吻结尾

**特殊情况处理**

- **用户消息触发压缩**: 当用户发送消息时，因用户消息长度导致会话总长度超过阈值
  - 处理方式：视为用户先主动压缩，再发送此条消息
  - 执行顺序：
    1. 先执行会话压缩（使用用户主动压缩指导）
    2. 压缩完成后，再处理用户发送的这条消息
  - 原因：避免自动压缩后自动继续，导致用户的新消息被忽略或打断

```kotlin
/**
 * 压缩触发源类型
 */
enum class CompressionTriggerType {
    /**
     * 自动压缩：由系统根据 token 阈值自动触发
     * 压缩后程序会自动发送"继续"指令，让 AI 平滑继续对话
     */
    AUTO,
    
    /**
     * 用户主动压缩：由用户通过界面或命令触发
     * 压缩后等待用户主动发起下一句话
     */
    USER_INITIATED
}

/**
 * 会话压缩配置
 */
data class SessionCompressionConfig(
    val enabled: Boolean = true,
    val thresholdPercent: Double = 0.7,      // 达到模型最大 token 的 70% 时触发
    val retainPercent: Double = 0.3,          // 保留最近 30% 的消息
    val promptConfigPath: Path                // 压缩提示词配置文件路径
)

/**
 * 压缩提示词配置
 * 针对不同的触发方式，使用不同的提示词模板
 */
data class CompressionPromptConfig(
    val autoSystemPrompt: String,              // 自动压缩的系统提示词
    val autoUserPromptTemplate: String,        // 自动压缩的用户提示词模板
    val userSystemPrompt: String,              // 用户主动压缩的系统提示词
    val userUserPromptTemplate: String,        // 用户主动压缩的用户提示词模板
    val maxSummaryLength: Int = 2000,          // 摘要最大长度
    val retainMessageCount: Int = 5            // 保留最近的消息数量（用于细节保留）
)

/**
 * 压缩提示词加载器
 */
class CompressionPromptLoader(
    private val configPath: Path
) {
    private var currentConfig: CompressionPromptConfig = loadDefaultConfig()
    
    /**
     * 加载提示词配置
     */
    fun loadConfig(): CompressionPromptConfig {
        return try {
            if (Files.exists(configPath)) {
                val json = Files.readString(configPath)
                Json.decodeFromString<CompressionPromptConfig>(json)
            } else {
                loadDefaultConfig()
            }
        } catch (e: Exception) {
            Logger.warn("加载压缩提示词配置失败，使用默认配置: ${e.message}")
            loadDefaultConfig()
        }
    }
    
    /**
     * 重新加载配置（热更新）
     */
    fun reloadConfig() {
        currentConfig = loadConfig()
        Logger.info("压缩提示词配置已重新加载")
    }
    
    fun getCurrentConfig(): CompressionPromptConfig = currentConfig
    
    /**
     * 根据触发类型获取对应的系统提示词
     */
    fun getSystemPrompt(triggerType: CompressionTriggerType): String {
        return when (triggerType) {
            CompressionTriggerType.AUTO -> currentConfig.autoSystemPrompt
            CompressionTriggerType.USER_INITIATED -> currentConfig.userSystemPrompt
        }
    }
    
    /**
     * 根据触发类型获取对应的用户提示词模板
     */
    fun getUserPromptTemplate(triggerType: CompressionTriggerType): String {
        return when (triggerType) {
            CompressionTriggerType.AUTO -> currentConfig.autoUserPromptTemplate
            CompressionTriggerType.USER_INITIATED -> currentConfig.userUserPromptTemplate
        }
    }
    
    private fun loadDefaultConfig(): CompressionPromptConfig {
        return CompressionPromptConfig(
            // 自动压缩提示词：AI 口吻，强调连续性
            autoSystemPrompt = """
                你是一个会话摘要助手。你的任务是对对话历史进行压缩摘要，以便 AI 模型能够平滑继续对话。
                
                请保留以下关键信息：
                1. 重要的决策和结论
                2. 用户的偏好和要求
                3. 未完成的任务和行动项
                4. 关键的时间点和事件
                5. 最近对话的细节（特别是最后几条消息的具体内容）
                
                摘要要求：
                - 使用 AI 的第一人称口吻（例如："我刚刚帮助用户完成了..."）
                - 结尾应体现对话的连续性（例如："接下来我将继续帮助用户..."）
                - 确保 AI 模型阅读摘要后能够立即继续之前的任务，不中断思路
                - 摘要应该简洁明了，便于后续对话时快速理解上下文
            """.trimIndent(),
            
            autoUserPromptTemplate = """
                请对以下对话进行摘要，以便 AI 能够平滑继续对话：
                
                {messages}
                
                请生成一个简洁的摘要，长度不超过{maxLength}字。
                注意：
                1. 保留最近{retainCount}条消息的完整细节
                2. 使用 AI 的第一人称口吻
                3. 结尾应体现"接下来我将继续..."的语义
            """.trimIndent(),
            
            // 用户主动压缩提示词：等待用户回应的口吻
            userSystemPrompt = """
                你是一个会话摘要助手。你的任务是对对话历史进行压缩摘要，以便用户快速回顾。
                
                请保留以下关键信息：
                1. 重要的决策和结论
                2. 用户的偏好和要求
                3. 未完成的任务和行动项
                4. 关键的时间点和事件
                5. 最近对话的细节（特别是最后几条消息的具体内容）
                
                摘要要求：
                - 使用第三人称或客观描述（例如："用户和 AI 讨论了..."）
                - 结尾应体现等待用户回应（例如："我们已经讨论了...请问您还有什么需要补充的吗？"）
                - 确保用户阅读摘要后能够快速回顾之前的对话内容
                - 摘要应该简洁明了，便于用户快速理解
            """.trimIndent(),
            
            userUserPromptTemplate = """
                请对以下对话进行摘要，以便用户快速回顾：
                
                {messages}
                
                请生成一个简洁的摘要，长度不超过{maxLength}字。
                注意：
                1. 保留最近{retainCount}条消息的完整细节
                2. 使用客观描述的语气
                3. 结尾应体现"等待用户回应"的语义
            """.trimIndent(),
            
            maxSummaryLength = 2000,
            retainMessageCount = 5
        )
    }
}

class SessionCompressionManager(
    private val tokenCounter: TokenCounter,
    private val mem0Integration: Mem0Integration,
    private val promptLoader: CompressionPromptLoader,
    private val config: SessionCompressionConfig,
    private val aiClient: AIClient
) {
    /**
     * 检查是否需要压缩
     */
    suspend fun shouldCompress(
        roleId: String,
        sessionId: String,
        messages: List<Message>
    ): Boolean {
        if (!config.enabled) return false
        
        val currentTokens = tokenCounter.countTokens(messages)
        val modelMaxTokens = getModelMaxTokens(roleId)
        val threshold = modelMaxTokens * config.thresholdPercent
        
        return currentTokens >= threshold
    }
    
    /**
     * 执行会话压缩（自动压缩）
     * 压缩后会返回是否需要自动继续的标记
     */
    suspend fun compressSession(
        roleId: String,
        sessionId: String,
        messages: List<Message>,
        triggerType: CompressionTriggerType = CompressionTriggerType.AUTO
    ): CompressionResult {
        val promptConfig = promptLoader.getCurrentConfig()
        
        // 1. 使用 AI 对早期对话进行摘要
        val summary = generateSummary(messages, triggerType, promptConfig, aiClient)
        
        // 2. 存储到 mem0
        mem0Integration.storeCompressionMemory(
            roleId = roleId,
            sessionId = sessionId,
            summary = summary
        )
        
        // 3. 保留最近的对话，替换早期对话为摘要
        // 使用配置中的 retainMessageCount 确保保留足够的最近消息细节
        val retainedCount = maxOf(
            (messages.size * config.retainPercent).toInt(),
            promptConfig.retainMessageCount
        )
        val recentMessages = messages.takeLast(retainedCount)
        
        val summaryMessage = Message(
            id = generateId(),
            senderId = "system",
            senderType = SenderType.SYSTEM,
            sessionId = sessionId,
            content = MessageContent.Text("[会话摘要] $summary"),
            timestamp = Instant.now(),
            status = MessageStatus.COMPLETE
        )
        
        // 4. 根据触发类型决定后续行为
        val shouldAutoContinue = when (triggerType) {
            CompressionTriggerType.AUTO -> true  // 自动压缩后自动继续
            CompressionTriggerType.USER_INITIATED -> false  // 用户主动压缩后等待用户
        }
        
        return CompressionResult(
            compressedMessages = listOf(summaryMessage) + recentMessages,
            summary = summary,
            originalTokenCount = tokenCounter.countTokens(messages),
            newTokenCount = tokenCounter.countTokens(listOf(summaryMessage) + recentMessages),
            shouldAutoContinue = shouldAutoContinue,
            triggerType = triggerType
        )
    }
    
    /**
     * 处理用户消息时的压缩检查
     * 如果用户消息导致 token 超限，则先执行用户主动压缩，再处理用户消息
     * @return 返回是否需要先压缩，以及压缩后的结果
     */
    suspend fun handleUserMessageWithCompression(
        roleId: String,
        sessionId: String,
        currentMessages: List<Message>,
        newUserMessage: Message
    ): CompressionHandlingResult {
        // 合并用户新消息后检查是否需要压缩
        val messagesWithNew = currentMessages + newUserMessage
        
        if (shouldCompress(roleId, sessionId, messagesWithNew)) {
            // 用户消息触发的压缩，视为用户主动压缩
            val compressionResult = compressSession(
                roleId = roleId,
                sessionId = sessionId,
                messages = currentMessages,  // 压缩现有消息
                triggerType = CompressionTriggerType.USER_INITIATED
            )
            
            // 压缩后，再添加用户的新消息
            val finalMessages = compressionResult.compressedMessages + newUserMessage
            
            return CompressionHandlingResult(
                shouldCompressFirst = true,
                compressionResult = compressionResult,
                finalMessages = finalMessages,
                shouldAutoContinue = false  // 用户触发的压缩，不自动继续
            )
        } else {
            // 不需要压缩，直接返回
            return CompressionHandlingResult(
                shouldCompressFirst = false,
                finalMessages = messagesWithNew,
                shouldAutoContinue = false
            )
        }
    }
    
    /**
     * 生成摘要
     */
    private suspend fun generateSummary(
        messages: List<Message>,
        triggerType: CompressionTriggerType,
        promptConfig: CompressionPromptConfig,
        aiClient: AIClient
    ): String {
        // 计算需要压缩的消息范围
        // 保留最近的 retainMessageCount 条消息不压缩
        val messagesToCompress = messages.dropLast(promptConfig.retainMessageCount)
        
        if (messagesToCompress.isEmpty()) {
            // 如果没有需要压缩的消息，返回空摘要
            return ""
        }
        
        val formattedMessages = messagesToCompress.joinToString("\n") { msg ->
            val sender = when (msg.senderType) {
                SenderType.USER -> "用户"
                SenderType.AI_ROLE -> msg.senderId
                SenderType.SYSTEM -> "系统"
            }
            "[$sender]: ${msg.content}"
        }
        
        // 根据触发类型选择提示词模板
        val systemPrompt = promptLoader.getSystemPrompt(triggerType)
        val userPromptTemplate = promptLoader.getUserPromptTemplate(triggerType)
        
        val userPrompt = userPromptTemplate
            .replace("{messages}", formattedMessages)
            .replace("{maxLength}", promptConfig.maxSummaryLength.toString())
            .replace("{retainCount}", promptConfig.retainMessageCount.toString())
        
        val response = aiClient.generate(
            systemPrompt = systemPrompt,
            userPrompt = userPrompt
        )
        
        return response.text
    }
}

/**
 * 压缩结果数据类
 */
data class CompressionResult(
    val compressedMessages: List<Message>,
    val summary: String,
    val originalTokenCount: Int,
    val newTokenCount: Int,
    val shouldAutoContinue: Boolean = false,  // 是否应该自动继续
    val triggerType: CompressionTriggerType = CompressionTriggerType.AUTO
)

/**
 * 用户消息压缩处理结果
 */
data class CompressionHandlingResult(
    val shouldCompressFirst: Boolean,          // 是否需要先压缩
    val compressionResult: CompressionResult? = null,  // 压缩结果（如果需要压缩）
    val finalMessages: List<Message>,          // 最终消息列表
    val shouldAutoContinue: Boolean = false    // 是否应该自动继续
)
```

#### 6.4 压缩后的自动继续逻辑

**自动压缩后的处理流程**

当自动压缩触发时，系统需要自动发送"继续"指令，让 AI 模型能够平滑继续对话：

```kotlin
/**
 * 自动继续处理器
 */
class AutoContinueHandler(
    private val aiClient: AIClient,
    private val sessionManager: SessionManager
) {
    /**
     * 在自动压缩后，自动发送继续指令
     */
    suspend fun handleAutoContinue(
        roleId: String,
        sessionId: String,
        compressionResult: CompressionResult
    ) {
        if (!compressionResult.shouldAutoContinue) {
            Logger.debug("压缩类型为${compressionResult.triggerType}，不自动继续")
            return
        }
        
        Logger.info("自动压缩完成，正在发送继续指令...")
        
        // 构建继续提示词
        val continuePrompt = buildContinuePrompt(compressionResult.summary)
        
        // 创建系统消息
        val continueMessage = Message(
            id = generateId(),
            senderId = "system",
            senderType = SenderType.SYSTEM,
            sessionId = sessionId,
            content = MessageContent.Text(continuePrompt),
            timestamp = Instant.now(),
            status = MessageStatus.COMPLETE
        )
        
        // 添加到会话
        sessionManager.addMessage(sessionId, continueMessage)
        
        // 请求 AI 继续
        val aiResponse = aiClient.generate(
            roleId = roleId,
            sessionId = sessionId,
            messages = compressionResult.compressedMessages + continueMessage
        )
        
        Logger.info("AI 已响应继续指令，对话平滑继续")
    }
    
    /**
     * 构建继续提示词
     */
    private fun buildContinuePrompt(summary: String): String {
        return """
            [系统提示] 会话已自动压缩以节省 token。
            
            摘要内容：
            $summary
            
            请根据上述摘要继续之前的对话或任务。
        """.trimIndent()
    }
}
```

**用户主动压缩后的处理流程**

用户主动压缩后，系统仅显示压缩摘要，等待用户主动发起下一条消息：

```kotlin
/**
 * 用户主动压缩处理器
 */
class UserInitiatedCompressionHandler(
    private val compressionManager: SessionCompressionManager,
    private val notificationService: NotificationService
) {
    /**
     * 处理用户主动压缩请求
     */
    suspend fun handleUserInitiatedCompression(
        roleId: String,
        sessionId: String,
        currentMessages: List<Message>
    ): CompressionResult {
        Logger.info("用户主动触发会话压缩")
        
        // 执行压缩（使用 USER_INITIATED 类型）
        val compressionResult = compressionManager.compressSession(
            roleId = roleId,
            sessionId = sessionId,
            messages = currentMessages,
            triggerType = CompressionTriggerType.USER_INITIATED
        )
        
        // 通知用户压缩完成
        notificationService.sendNotification(
            title = "会话压缩完成",
            message = "会话已压缩，保留了最近的对话细节。请继续您的提问。",
            type = NotificationType.INFO
        )
        
        // 不自动继续，等待用户主动发起
        Logger.info("用户主动压缩完成，等待用户继续")
        
        return compressionResult
    }
}
```

#### 6.5 用户消息触发压缩的完整流程

当用户发送消息时，如果消息长度导致会话总 token 数超过阈值，系统按以下流程处理：

```kotlin
/**
 * 用户消息处理器（带压缩检查）
 */
class UserMessageHandler(
    private val compressionManager: SessionCompressionManager,
    private val aiService: AIService,
    private val sessionManager: SessionManager
) {
    /**
     * 处理用户发送的消息
     */
    suspend fun handleUserMessage(
        roleId: String,
        sessionId: String,
        userMessage: Message
    ) {
        // 获取当前会话消息
        val currentMessages = sessionManager.getSessionMessages(sessionId)
        
        // 检查是否需要先压缩
        val compressionResult = compressionManager.handleUserMessageWithCompression(
            roleId = roleId,
            sessionId = sessionId,
            currentMessages = currentMessages,
            newUserMessage = userMessage
        )
        
        if (compressionResult.shouldCompressFirst) {
            Logger.info("用户消息触发压缩，先执行压缩")
            
            // 压缩已完成，使用压缩后的消息列表
            // 注意：compressionResult.finalMessages 已包含压缩摘要 + 用户新消息
            
            // 不自动继续，因为这是用户触发的压缩
            // 等待 AI 正常响应用户的新消息
            val aiResponse = aiService.generateResponse(
                roleId = roleId,
                sessionId = sessionId,
                messages = compressionResult.finalMessages
            )
            
            // 发送 AI 响应
            sessionManager.addMessage(sessionId, aiResponse.toMessage())
        } else {
            // 不需要压缩，正常处理
            Logger.debug("不需要压缩，正常处理用户消息")
            
            val aiResponse = aiService.generateResponse(
                roleId = roleId,
                sessionId = sessionId,
                messages = compressionResult.finalMessages
            )
            
            sessionManager.addMessage(sessionId, aiResponse.toMessage())
        }
    }
}
```

**流程图**

```
用户发送消息
    ↓
检查 (当前消息 + 新消息) 的 token 数
    ↓
是否超过阈值？
    ├─ 否 → 正常处理，AI 响应用户消息
    └─ 是 → 执行用户主动压缩
            ↓
        使用 USER_INITIATED 提示词压缩
            ↓
        压缩后添加用户新消息
            ↓
        AI 响应（不自动继续）
            ↓
        等待用户下一条消息
```

### 7. 日志和监控系统

#### 7.1 日志记录规范

**日志级别**

```kotlin
enum class LogLevel {
    DEBUG,      // 调试信息
    INFO,       // 一般信息
    WARN,       // 警告
    ERROR       // 错误
}
```

**日志内容**

需要记录以下行为：

1. **AI行为日志**
   - 角色启动/停止
   - 会话开始/结束处理
   - 消息接收和响应
   - 工具调用请求和结果
   - 配置变更

2. **对话日志**
   - 所有收发的消息内容
   - 消息状态变更
   - 会话压缩事件

3. **工具调用日志**
   - 工具名称和参数
   - 执行结果或错误
   - 执行耗时

4. **程序输出日志**
   - 系统启动/关闭
   - 服务健康状态
   - 异常和错误

**重复行为监控**

```kotlin
/**
 * 重复行为检测器
 */
class DuplicateBehaviorDetector {
    private val recentActions = ConcurrentHashMap<String, CircularFifoQueue<ActionRecord>>()
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    
    data class ActionRecord(
        val action: String,
        val content: String,
        val timestamp: Instant
    )
    
    /**
     * 记录行为
     */
    fun recordAction(roleId: String, action: String, content: String) {
        val queue = recentActions.getOrPut(roleId) { 
            CircularFifoQueue(100)  // 保留最近100条记录
        }
        queue.add(ActionRecord(action, content, Instant.now()))
        
        // 检查重复
        checkForDuplicates(roleId, action, content)
    }
    
    /**
     * 检查重复行为
     */
    private fun checkForDuplicates(roleId: String, action: String, content: String) {
        val queue = recentActions[roleId] ?: return
        val recentRecords = queue.toList().takeLast(10)
        
        // 检查相同行为重复次数
        val sameActionCount = recentRecords.count { 
            it.action == action && it.content == content 
        }
        
        if (sameActionCount >= 3) {
            Logger.warn("检测到角色[$roleId]重复执行相同行为[$action] $sameActionCount 次")
        }
        
        // 检查相同文本重复
        val sameContentCount = recentRecords.count { it.content == content }
        if (sameContentCount >= 3) {
            Logger.warn("检测到角色[$roleId]重复输出相同内容 $sameContentCount 次")
        }
    }
}
```

#### 7.2 日志存储

日志采用文件存储，按日期和角色分类：

```
logs/
├── system/
│   ├── 2026-03-11.log
│   └── 2026-03-12.log
├── roles/
│   ├── role_001/
│   │   ├── 2026-03-11.log
│   │   └── 2026-03-12.log
│   └── role_002/
│       └── ...
└── conversations/
    ├── session_001/
    │   └── 2026-03-11.log
    └── group_chat/
        └── 2026-03-11.log
```

### 8. 数据备份和升级机制

#### 8.1 每日备份

**备份策略**

```kotlin
/**
 * 备份管理器
 */
class BackupManager(
    private val workspacesPath: Path,
    private val backupPath: Path
) {
    companion object {
        const val BACKUP_TIME_HOUR = 2  // 凌晨2点备份
        const val KEEP_DAYS = 30        // 保留30天
    }
    
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    
    /**
     * 启动定时备份任务
     */
    fun startScheduledBackup() {
        scope.launch {
            while (isActive) {
                val now = LocalDateTime.now()
                val nextBackup = now.withHour(BACKUP_TIME_HOUR)
                    .withMinute(0)
                    .withSecond(0)
                
                val delayMillis = if (now.hour >= BACKUP_TIME_HOUR) {
                    // 今天已过了备份时间，预约明天
                    Duration.between(now, nextBackup.plusDays(1)).toMillis()
                } else {
                    Duration.between(now, nextBackup).toMillis()
                }
                
                delay(delayMillis)
                performBackup()
            }
        }
    }
    
    /**
     * 执行备份
     */
    suspend fun performBackup(): Path = withContext(Dispatchers.IO) {
        val timestamp = LocalDateTime.now().format(DateTimeFormatter.ofPattern("yyyyMMdd_HHmmss"))
        val backupFileName = "workspaces_backup_$timestamp.7z"
        val backupFile = backupPath.resolve(backupFileName)
        
        Files.createDirectories(backupPath)
        
        // 使用7z压缩
        val process = ProcessBuilder(
            "7z", "a", "-t7z", "-m0=lzma2", "-mx=9",
            backupFile.toString(),
            workspacesPath.toString()
        ).start()
        
        val success = process.waitFor(300, TimeUnit.SECONDS)
        
        if (!success || process.exitValue() != 0) {
            throw RuntimeException("备份失败")
        }
        
        Logger.info("备份完成: $backupFileName")
        
        // 清理旧备份
        cleanupOldBackups()
        
        backupFile
    }
    
    /**
     * 清理旧备份
     */
    private suspend fun cleanupOldBackups() = withContext(Dispatchers.IO) {
        val cutoffDate = LocalDateTime.now().minusDays(KEEP_DAYS.toLong())
        
        Files.list(backupPath)
            .filter { it.fileName.toString().startsWith("workspaces_backup_") }
            .filter { 
                val fileTime = Files.getLastModifiedTime(it).toInstant()
                fileTime.isBefore(cutoffDate.atZone(ZoneId.systemDefault()).toInstant())
            }
            .forEach { 
                Files.deleteIfExists(it)
                Logger.info("删除旧备份: ${it.fileName}")
            }
    }
}
```

#### 8.2 数据升级机制

**版本检测和升级**

```kotlin
/**
 * 数据版本管理器
 */
class DataVersionManager(
    private val workspacesPath: Path
) {
    companion object {
        const val CURRENT_VERSION = 1
    }
    
    /**
     * 检测并执行升级
     */
    suspend fun checkAndUpgrade() {
        val currentVersion = readCurrentVersion()
        
        if (currentVersion < CURRENT_VERSION) {
            Logger.info("检测到数据版本 $currentVersion，需要升级到 $CURRENT_VERSION")
            
            for (version in currentVersion until CURRENT_VERSION) {
                upgradeFrom(version)
            }
            
            writeCurrentVersion(CURRENT_VERSION)
            Logger.info("数据升级完成")
        }
    }
    
    /**
     * 从指定版本升级
     */
    private suspend fun upgradeFrom(fromVersion: Int) {
        Logger.info("执行从版本 $fromVersion 的升级")
        
        when (fromVersion) {
            0 -> upgradeFromV0ToV1()
            // 未来版本添加更多升级路径
        }
    }
    
    /**
     * V0 -> V1 升级
     * 示例：添加 token_bucket.json 文件
     */
    private suspend fun upgradeFromV0ToV1() = withContext(Dispatchers.IO) {
        Files.list(workspacesPath).forEach { roleDir ->
            val tokenBucketFile = roleDir.resolve("token_bucket.json")
            if (!Files.exists(tokenBucketFile)) {
                // 创建默认令牌桶配置
                val defaultConfig = TokenBucketState(
                    roleId = roleDir.fileName.toString(),
                    tokens = 5.0,
                    capacity = 5.0,
                    refillRate = 0.5,
                    lastRefillTime = Instant.now()
                )
                Files.writeString(tokenBucketFile, Json.encodeToString(defaultConfig))
            }
        }
    }
    
    private fun readCurrentVersion(): Int {
        val versionFile = workspacesPath.resolve(".version")
        return try {
            if (Files.exists(versionFile)) {
                Files.readString(versionFile).toInt()
            } else {
                0
            }
        } catch (e: Exception) {
            0
        }
    }
    
    private fun writeCurrentVersion(version: Int) {
        val versionFile = workspacesPath.resolve(".version")
        Files.writeString(versionFile, version.toString())
    }
}
```

### 9. 配置热加载机制

#### 9.1 配置变更处理

```kotlin
/**
 * 配置热加载管理器
 */
class ConfigHotReloadManager(
    private val roleId: String,
    private val configPath: Path,
    private val onConfigChanged: (AIRoleConfig) -> Unit
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private var watchJob: Job? = null
    
    /**
     * 启动配置监控
     */
    fun startWatching() {
        watchJob = scope.launch {
            val watchService = FileSystems.getDefault().newWatchService()
            configPath.register(watchService, StandardWatchEventKinds.ENTRY_MODIFY)
            
            while (isActive) {
                val key = watchService.poll(1, TimeUnit.SECONDS) ?: continue
                
                key.pollEvents().forEach { event ->
                    val fileName = event.context() as? Path
                    if (fileName != null) {
                        handleConfigFileChange(fileName.toString())
                    }
                }
                
                key.reset()
            }
        }
    }
    
    /**
     * 处理配置文件变更
     */
    private fun handleConfigFileChange(fileName: String) {
        when (fileName) {
            "profile.json", "system_prompt.md", "behavior_rules.md" -> {
                Logger.info("检测到角色[$roleId]配置变更: $fileName")
                // 重新加载配置
                scope.launch {
                    val newConfig = loadRoleConfig(configPath)
                    onConfigChanged(newConfig)
                }
            }
        }
    }
    
    /**
     * 加载角色配置
     */
    private suspend fun loadRoleConfig(path: Path): AIRoleConfig = withContext(Dispatchers.IO) {
        val profilePath = path.resolve("profile.json")
        val systemPromptPath = path.resolve("system_prompt.md")
        val behaviorRulesPath = path.resolve("behavior_rules.md")
        
        val profile = Json.decodeFromString<AIRoleProfile>(
            Files.readString(profilePath)
        )
        
        val systemPrompt = if (Files.exists(systemPromptPath)) {
            Files.readString(systemPromptPath)
        } else ""
        
        val behaviorRules = if (Files.exists(behaviorRulesPath)) {
            Files.readAllLines(behaviorRulesPath)
        } else emptyList()
        
        AIRoleConfig(
            id = profile.id,
            name = profile.name,
            description = profile.description,
            avatarPath = null,
            personality = PersonalityType.valueOf(profile.personality),
            modelId = "",  // 从model_config.json读取
            systemPrompt = systemPrompt,
            behaviorRules = behaviorRules,
            createdAt = profile.createdAt,
            updatedAt = profile.updatedAt
        )
    }
    
    fun stop() {
        watchJob?.cancel()
    }
}

/**
 * 配置变更通知
 * 
 * 说明：
 * - 修改的配置都是与提示词相关，影响系统提示词
 * - 已有的会话内容保持不变
 * - 配置切换只是文本更替
 * - 不考虑因变动系统提示词导致的模型缓存失效增加的Token使用
 */
class ConfigChangeNotifier {
    private val listeners = ConcurrentHashMap<String, MutableList<suspend (ConfigChangeEvent) -> Unit>>()
    
    data class ConfigChangeEvent(
        val roleId: String,
        val changedFiles: List<String>,
        val newConfig: AIRoleConfig
    )
    
    fun subscribe(roleId: String, listener: suspend (ConfigChangeEvent) -> Unit) {
        listeners.getOrPut(roleId) { mutableListOf() }.add(listener)
    }
    
    suspend fun notify(event: ConfigChangeEvent) {
        listeners[event.roleId]?.forEach { listener ->
            try {
                listener(event)
            } catch (e: Exception) {
                Logger.error("配置变更通知失败: ${e.message}")
            }
        }
    }
}
```

## 实施计划

### 阶段 1: 基础架构（优先级：高）

- [ ] 设计工作区文件系统结构
- [ ] 实现 AI 角色配置管理
- [ ] 实现模型验证器（100K 上下文、多模态、工具调用）
- [ ] 实现技能黑白名单基础功能
- [ ] 集成 mem0 客户端
- [ ] 实现统一存储系统（mem0）

### 阶段 2: 会话系统（优先级：高）

- [ ] 实现永久会话管理（一对一会话、群聊会话）
- [ ] 实现消息模型和存储
- [ ] 实现会话状态流转（已读、处理中）
- [ ] 实现会话压缩基础功能（配置化提示词）
- [ ] 实现 token 计数和阈值检测

### 阶段 3: 群聊调度（优先级：中）

- [ ] 实现调度器 AI（群聊记忆、@提及机制）
- [ ] 实现令牌桶限流算法
- [ ] 实现调度器话题分析
- [ ] 实现令牌桶状态注入提示词

### 阶段 4: 实时打断（优先级：中）

- [ ] 实现协程取消机制
- [ ] 实现工具调用中断
- [ ] 实现子进程终止
- [ ] 实现消息撤回
- [ ] 实现打断事件总线

### 阶段 5: 工具系统（优先级：中）

- [ ] 实现浏览器工具（自动启动、CDP协议）
- [ ] 实现文件操作工具（读、查、改、删）
- [ ] 实现技能管理工具
- [ ] 实现记忆检索工具
- [ ] 实现自我认知工具

### 阶段 6: 基础设施（优先级：中）

- [ ] 实现错误处理和重试机制
- [ ] 实现日志和监控系统
- [ ] 实现数据备份机制
- [ ] 实现数据升级机制
- [ ] 实现配置热加载

### 阶段 7: 测试和优化（优先级：低）

- [ ] 编写单元测试
- [ ] 编写集成测试
- [ ] 性能优化
- [ ] 用户体验优化
- [ ] 文档完善

## 10. 数据流图

### 10.1 消息处理流程

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   用户输入   │────▶│  消息解析器  │────▶│  会话路由   │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                               │
                    ┌──────────────────────────┼──────────────────────────┐
                    │                          │                          │
                    ▼                          ▼                          ▼
            ┌───────────────┐          ┌───────────────┐          ┌───────────────┐
            │  一对一会话    │          │   群聊会话     │          │  @提及处理    │
            │  (直接发送)    │          │  (调度器分析)  │          │  (强制调度)    │
            └───────┬───────┘          └───────┬───────┘          └───────┬───────┘
                    │                          │                          │
                    │    ┌─────────────────────┘                          │
                    │    │                                                │
                    ▼    ▼                                                ▼
            ┌───────────────────────────────────────────────────────────────────────┐
            │                          群聊调度器 AI                                 │
            │  - 分析话题相关性                                                    │
            │  - 查询群聊记忆                                                      │
            │  - 选择参与角色                                                      │
            └─────────────────────────────────┬─────────────────────────────────────┘
                                              │
                                              ▼
                              ┌───────────────────────────────┐
                              │    遍历选中的 AI 角色          │
                              │  1. 检查令牌桶余额             │
                              │  2. 注入记忆和状态             │
                              │  3. 调用模型生成响应           │
                              └───────────────┬───────────────┘
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
                    │                         │                         │
                    ▼                         ▼                         ▼
            ┌───────────────┐         ┌───────────────┐         ┌───────────────┐
            │   工具调用     │         │   正常响应     │         │   令牌不足     │
            │  (执行并返回)  │         │  (直接返回)    │         │  (跳过该角色)  │
            └───────┬───────┘         └───────┬───────┘         └───────────────┘
                    │                         │
                    └───────────┬─────────────┘
                                │
                                ▼
                    ┌───────────────────────┐
                    │    存储到 mem0        │
                    │  - 会话历史           │
                    │  - 角色记忆           │
                    │  - 群聊记忆           │
                    └───────────┬───────────┘
                                │
                                ▼
                    ┌───────────────────────┐
                    │      返回给用户        │
                    └───────────────────────┘
```

### 10.2 AI 角色内部处理流程

```
┌─────────────────────────────────────────────────────────────────┐
│                        AI 角色处理流程                           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 1. 接收消息                                                      │
│    - 更新会话状态（标记已读）                                     │
│    - 设置处理中状态                                              │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. 构建上下文                                                    │
│    - 获取会话历史（从 mem0）                                      │
│    - 检索相关记忆                                                │
│    - 获取令牌桶状态                                              │
│    - 加载系统提示词                                              │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. 调用模型                                                      │
│    - 发送请求到 LM Studio                                        │
│    - 支持流式响应                                                │
│    - 可中断处理                                                  │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. 处理响应                                                      │
│    - 解析工具调用请求                                            │
│    - 执行工具（文件、浏览器、技能等）                              │
│    - 工具结果返回模型                                            │
│    - 生成最终响应                                                │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. 完成处理                                                      │
│    - 存储消息到 mem0                                             │
│    - 更新会话状态（处理完成）                                     │
│    - 消费令牌                                                    │
│    - 检查是否需要会话压缩                                         │
└─────────────────────────────────────────────────────────────────┘
```

## 11. 并发控制策略

### 11.1 并发场景分析

**需要控制的并发场景：**

1. **多角色同时响应群聊消息**
2. **同一角色同时处理多个会话**
3. **工具调用的并发执行**
4. **配置热加载的并发访问**

### 11.2 并发控制实现

```kotlin
/**
 * 角色并发控制器
 * 确保同一角色不会同时处理多个请求
 */
class RoleConcurrencyController {
    private val processingRoles = ConcurrentHashMap<String, Mutex>()
    
    /**
     * 获取角色的互斥锁
     */
    fun getMutex(roleId: String): Mutex {
        return processingRoles.getOrPut(roleId) { Mutex() }
    }
    
    /**
     * 检查角色是否正在处理
     */
    fun isProcessing(roleId: String): Boolean {
        val mutex = processingRoles[roleId] ?: return false
        return mutex.isLocked
    }
    
    /**
     * 执行受控操作
     */
    suspend fun <T> withRoleLock(roleId: String, block: suspend () -> T): T {
        val mutex = getMutex(roleId)
        return mutex.withLock {
            block()
        }
    }
}

/**
 * 会话并发控制器
 * 管理每个会话的并发访问
 */
class SessionConcurrencyController {
    private val sessionLocks = ConcurrentHashMap<String, Mutex>()
    private val messageQueues = ConcurrentHashMap<String, Channel<Message>>()
    
    /**
     * 初始化会话队列
     */
    fun initSession(sessionId: String) {
        sessionLocks.getOrPut(sessionId) { Mutex() }
        messageQueues.getOrPut(sessionId) { Channel(Channel.BUFFERED) }
    }
    
    /**
     * 发送消息到会话队列
     */
    suspend fun sendMessage(sessionId: String, message: Message) {
        val queue = messageQueues[sessionId] 
            ?: throw IllegalStateException("会话 $sessionId 未初始化")
        queue.send(message)
    }
    
    /**
     * 接收会话消息
     */
    suspend fun receiveMessage(sessionId: String): Message {
        val queue = messageQueues[sessionId] 
            ?: throw IllegalStateException("会话 $sessionId 未初始化")
        return queue.receive()
    }
    
    /**
     * 在会话锁保护下执行操作
     */
    suspend fun <T> withSessionLock(sessionId: String, block: suspend () -> T): T {
        val mutex = sessionLocks.getOrPut(sessionId) { Mutex() }
        return mutex.withLock {
            block()
        }
    }
}

/**
 * 工具调用并发控制器
 */
class ToolConcurrencyController {
    // 限制同时执行的工具调用数量
    private val semaphore = Semaphore(10)
    
    /**
     * 执行受控的工具调用
     */
    suspend fun <T> executeWithControl(block: suspend () -> T): T {
        semaphore.acquire()
        try {
            return block()
        } finally {
            semaphore.release()
        }
    }
}

/**
 * 全局并发管理器
 */
class ConcurrencyManager(
    private val roleController: RoleConcurrencyController,
    private val sessionController: SessionConcurrencyController,
    private val toolController: ToolConcurrencyController
) {
    /**
     * 处理群聊消息
     * 并行调度多个角色，但每个角色串行处理
     */
    suspend fun processGroupChatMessage(
        message: Message,
        selectedRoles: List<String>,
        processRole: suspend (String, Message) -> Unit
    ) {
        coroutineScope {
            selectedRoles.forEach { roleId ->
                launch {
                    // 每个角色在独立的协程中处理，但受角色锁保护
                    roleController.withRoleLock(roleId) {
                        processRole(roleId, message)
                    }
                }
            }
        }
    }
    
    /**
     * 处理工具调用
     */
    suspend fun <T> executeTool(block: suspend () -> T): T {
        return toolController.executeWithControl {
            block()
        }
    }
}
```

### 11.3 打断机制的并发处理

```kotlin
/**
 * 打断信号管理器
 */
class InterruptionManager {
    private val interruptionSignals = ConcurrentHashMap<String, Channel<Unit>>()
    
    /**
     * 获取或创建打断信号通道
     */
    fun getSignalChannel(roleId: String): Channel<Unit> {
        return interruptionSignals.getOrPut(roleId) { 
            Channel(Channel.CONFLATED) 
        }
    }
    
    /**
     * 发送打断信号
     */
    fun interrupt(roleId: String) {
        getSignalChannel(roleId).trySend(Unit)
    }
    
    /**
     * 监听打断信号
     */
    suspend fun watchForInterruption(roleId: String, onInterrupt: suspend () -> Unit) {
        val channel = getSignalChannel(roleId)
        channel.receive()
        onInterrupt()
    }
}
```

## 12. 单例部署结构

### 12.1 部署架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        单服务器部署                              │
│                     (Single Instance)                            │
└─────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│   AI 角色服务  │    │   mem0 服务   │    │  LM Studio   │
│  (Kotlin/JVM) │    │  (独立进程)   │    │  (模型服务)   │
└───────┬───────┘    └───────────────┘    └───────────────┘
        │
        │    ┌──────────────────────────────────────────┐
        │    │           工作区文件系统                  │
        │    │  workspaces/                             │
        │    │  ├── role_001/                           │
        │    │  ├── role_002/                           │
        │    │  └── ...                                 │
        │    │                                          │
        │    │  logs/                                   │
        │    │  backups/                                │
        │    └──────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────────┐
│                      服务组件关系                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐     │
│  │  HTTP API   │◀────▶│  核心业务逻辑 │◀────▶│  角色管理器  │     │
│  │  (Ktor)     │      │             │      │             │     │
│  └─────────────┘      └──────┬──────┘      └──────┬──────┘     │
│                              │                    │            │
│                              ▼                    ▼            │
│                       ┌─────────────┐      ┌─────────────┐     │
│                       │  会话管理器  │◀────▶│  工具管理器  │     │
│                       │             │      │             │     │
│                       └──────┬──────┘      └──────┬──────┘     │
│                              │                    │            │
│                              ▼                    ▼            │
│                       ┌─────────────┐      ┌─────────────┐     │
│                       │   mem0 客户端 │    │  浏览器管理  │     │
│                       │             │      │             │     │
│                       └──────┬──────┘      └─────────────┘     │
│                              │                                 │
│                              ▼                                 │
│                       ┌─────────────┐                          │
│                       │  LM Studio  │                          │
│                       │   客户端    │                          │
│                       └─────────────┘                          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 12.2 服务启动顺序

```kotlin
/**
 * 应用启动器
 */
class ApplicationBootstrapper {
    suspend fun start() {
        Logger.info("========== 启动 AI 角色系统 ==========")
        
        // 1. 检查并升级数据
        Logger.info("[1/7] 检查数据版本...")
        DataVersionManager(workspacesPath).checkAndUpgrade()
        
        // 2. 启动 mem0 健康检查
        Logger.info("[2/7] 连接 mem0 服务...")
        val mem0Client = createMem0Client()
        val healthChecker = Mem0HealthChecker(mem0Client)
        healthChecker.start()
        
        // 3. 初始化令牌桶
        Logger.info("[3/7] 初始化令牌桶...")
        val tokenBucketLimiter = TokenBucketRateLimiter(tokenBucketPath)
        loadAllRoles().forEach { role ->
            tokenBucketLimiter.initBucket(
                roleId = role.id,
                capacity = config.tokenBucket.capacity,
                refillRate = config.tokenBucket.refillRate
            )
        }
        
        // 4. 初始化浏览器工具（结束残留进程）
        Logger.info("[4/7] 初始化浏览器工具...")
        BrowserTool.killAllBrowserProcesses()
        
        // 5. 加载 AI 角色
        Logger.info("[5/7] 加载 AI 角色...")
        val roleManager = RoleManager(workspacesPath, mem0Client)
        roleManager.loadAllRoles()
        
        // 6. 启动会话管理器
        Logger.info("[6/7] 启动会话管理器...")
        val sessionManager = SessionManager(mem0Client)
        sessionManager.initializeSessions()
        
        // 7. 启动 HTTP API 服务
        Logger.info("[7/7] 启动 API 服务...")
        val apiServer = APIServer(roleManager, sessionManager)
        apiServer.start()
        
        // 8. 启动备份任务
        Logger.info("启动定时备份任务...")
        val backupManager = BackupManager(workspacesPath, backupPath)
        backupManager.startScheduledBackup()
        
        Logger.info("========== 系统启动完成 ==========")
    }
}
```

### 12.3 配置文件结构

```hocon
# application.conf

server {
    host = "0.0.0.0"
    port = 8080
}

storage {
    workspacesPath = "./workspaces"
    logsPath = "./logs"
    backupPath = "./backups"
}

mem0 {
    baseUrl = "http://localhost:8000"
    healthCheckInterval = "30s"
}

lmStudio {
    baseUrl = "http://localhost:1234"
    minContextLength = 100000  # 可从配置文件指定
    requireMultimodal = true
    requireToolCalling = true
}

tokenBucket {
    capacity = 5.0
    refillRate = 0.5  # 每秒
}

compression {
    enabled = true
    thresholdPercent = 0.7
    retainPercent = 0.3
    promptConfigPath = "./config/compression_prompt.json"
}

dispatcher {
    roleId = "dispatcher_001"
    modelId = "default"
}

backup {
    enabled = true
    scheduleHour = 2
    keepDays = 30
}
```

## 可行性测试清单

### 必需测试

- [ ] **文件系统隔离测试**: 验证工作区目录结构
- [ ] **令牌桶算法测试**: 验证令牌补充和消费逻辑
- [ ] **实时打断测试**: 验证协程取消、工具中断、进程终止
- [ ] **技能黑白名单测试**: 验证白名单优先、黑名单过滤
- [ ] **模型验证测试**: 验证 100K 上下文、多模态、工具调用检测
- [ ] **mem0 隔离测试**: 验证不同角色的记忆隔离
- [ ] **mem0 共享测试**: 验证群聊记忆共享
- [ ] **会话压缩测试**: 验证压缩触发和 mem0 存储
- [ ] **浏览器自动启动测试**: 验证CDP检测和自动启动
- [ ] **配置热加载测试**: 验证配置文件变更检测和加载
- [ ] **数据升级测试**: 验证版本检测和数据迁移
- [ ] **并发控制测试**: 验证多角色并发和锁机制

### 可选测试

- [ ] **调度器 AI 测试**: 验证话题分析和角色选择
- [ ] **令牌桶限流测试**: 验证限流效果和提示词注入
- [ ] **多角色并发测试**: 验证多个 AI 同时回应的场景
- [ ] **备份恢复测试**: 验证备份创建和恢复流程

## 技术依赖

### JVM 库

```toml
[dependencies]
# Ktor (网络框架)
ktor-client-core = "3.4.1"
ktor-client-cio = "3.4.1"
ktor-server-core = "3.4.1"
ktor-server-cio = "3.4.1"

# Kotlin 协程
kotlinx-coroutines-core = "1.9.0"
kotlinx-coroutines-jdk8 = "1.9.0"

# 序列化
kotlinx-serialization-json = "1.7.3"

# 配置 (HOCON)
typesafe-config = "1.4.3"
kotlinx-config = "0.5.0"

# 日志
kotlin-logging = "3.0.5"
slf4j = "2.0.16"
logback = "1.5.16"

# 时间处理
kotlinx-datetime = "0.6.2"

# JSON 处理
kotlinx-serialization-json = "1.7.3"

# 文件系统
kotlinx-io = "0.6.0"
```

## 风险评估

### 低风险

- ✅ 文件系统隔离实现简单
- ✅ Kotlin 协程取消机制成熟
- ✅ mem0 REST API 调用直接
- ✅ JVM 生态系统库丰富

### 中风险

- ⚠️ 调度器 AI 的话题分析准确性
  - 缓解：提供手动配置和规则匹配
- ⚠️ 疲劳值算法需要调优
  - 缓解：提供配置参数，支持动态调整
- ⚠️ 工具调用中断可能不完整
  - 缓解：为每种工具类型实现专门的取消逻辑

### 高风险

- ❓ 实时打断在极端情况下的稳定性
  - 缓解：进行充分的压力测试
- ❓ 多个 AI 角色并发场景下的性能
  - 缓解：实现并发控制和队列管理

## 验收标准

### 功能验收

- [ ] 成功创建多个 AI 角色，每个角色有独立工作区
- [ ] 角色间记忆完全隔离
- [ ] 群聊中记忆共享
- [ ] 疲劳值系统正常工作（增长、衰减、持久化）
- [ ] 实时打断功能正常（模型请求、工具调用、子进程）
- [ ] 技能黑白名单正常工作
- [ ] 模型选择限制正常工作（过滤不满足要求的模型）
- [ ] 会话压缩正常触发并存储到 mem0

### 性能验收

- [ ] 打断响应时间 < 100ms
- [ ] 记忆检索时间 < 500ms
- [ ] 群聊调度时间 < 1s
- [ ] 疲劳值计算时间 < 10ms

### 文档验收

- [ ] 用户手册完整
- [ ] 开发者文档完整
- [ ] API 文档完整
- [ ] 配置示例完整

## 8. AI 角色基础工具系统

### 8.1 工具架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                     AI 角色工具管理器                        │
│                  (AIToolManager)                             │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│   文件工具     │    │   浏览器工具   │    │   技能工具     │
│ FileToolKit   │    │ BrowserTool   │    │  SkillTool    │
└───────────────┘    └───────────────┘    └───────────────┘
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│   记忆工具     │    │   自我认知工具 │    │   删除工具     │
│  MemoryTool   │    │ SelfAwareTool │    │  DeleteTool   │
└───────────────┘    └───────────────┘    └───────────────┘
```

### 8.2 读取工具 (ReadTool)

```kotlin
/**
 * 读取工具 - 用于读取工作区内的文件内容
 */
class ReadTool(
    private val workspacePath: Path
) {
    /**
     * 读取文件内容
     * @param relativePath 相对于工作区的文件路径
     * @param offset 起始行号（从1开始）
     * @param limit 读取行数
     * @return 文件内容
     */
    suspend fun read(
        relativePath: String,
        offset: Int? = null,
        limit: Int? = null
    ): ReadResult {
        // 验证路径在工作区内
        val filePath = validateAndResolvePath(relativePath)
        
        return withContext(Dispatchers.IO) {
            if (!Files.exists(filePath)) {
                return@withContext ReadResult.Error("文件不存在: $relativePath")
            }
            
            if (Files.isDirectory(filePath)) {
                return@withContext ReadResult.Error("路径是目录，不是文件: $relativePath")
            }
            
            val content = if (offset != null && limit != null) {
                // 读取指定范围
                readRange(filePath, offset, limit)
            } else {
                // 读取全部内容
                Files.readString(filePath)
            }
            
            ReadResult.Success(
                content = content,
                totalLines = Files.readAllLines(filePath).size,
                readLines = content.lines().size
            )
        }
    }
    
    /**
     * 验证并解析路径，确保在工作区内
     */
    private fun validateAndResolvePath(relativePath: String): Path {
        val resolved = workspacePath.resolve(relativePath).normalize()
        if (!resolved.startsWith(workspacePath)) {
            throw SecurityException("路径超出工作区范围: $relativePath")
        }
        return resolved
    }
}

sealed class ReadResult {
    data class Success(
        val content: String,
        val totalLines: Int,
        val readLines: Int
    ) : ReadResult()
    
    data class Error(val message: String) : ReadResult()
}
```

### 8.3 查找工具 (FindTool)

```kotlin
/**
 * 查找工具 - 用于在工作区内查找文件
 */
class FindTool(
    private val workspacePath: Path
) {
    /**
     * 查找文件
     * @param pattern 通配符模式，如 "*.kt", "**/*.md"
     * @param path 搜索的起始路径（相对于工作区）
     * @return 匹配的文件列表
     */
    suspend fun find(
        pattern: String,
        path: String = "."
    ): FindResult {
        val searchPath = workspacePath.resolve(path).normalize()
        
        if (!searchPath.startsWith(workspacePath)) {
            return FindResult.Error("搜索路径超出工作区范围")
        }
        
        return withContext(Dispatchers.IO) {
            try {
                val matcher = FileSystems.getDefault()
                    .getPathMatcher("glob:$pattern")
                
                val matches = Files.walk(searchPath)
                    .filter { matcher.matches(it.fileName) }
                    .map { workspacePath.relativize(it).toString() }
                    .toList()
                
                FindResult.Success(matches)
            } catch (e: Exception) {
                FindResult.Error("查找失败: ${e.message}")
            }
        }
    }
}

sealed class FindResult {
    data class Success(val files: List<String>) : FindResult()
    data class Error(val message: String) : FindResult()
}
```

### 8.4 修改工具 (EditTool)

```kotlin
/**
 * 修改工具 - 用于修改工作区内的文件
 */
class EditTool(
    private val workspacePath: Path
) {
    /**
     * 修改文件内容
     * @param relativePath 相对于工作区的文件路径
     * @param oldString 原文内容（用于定位）
     * @param newString 新内容（用于替换）
     * @return 修改结果
     */
    suspend fun edit(
        relativePath: String,
        oldString: String,
        newString: String
    ): EditResult {
        val filePath = validateAndResolvePath(relativePath)
        
        return withContext(Dispatchers.IO) {
            if (!Files.exists(filePath)) {
                return@withContext EditResult.Error("文件不存在: $relativePath")
            }
            
            val content = Files.readString(filePath)
            
            // 检查匹配次数
            val matchCount = content.split(oldString).size - 1
            
            when {
                matchCount == 0 -> {
                    EditResult.Error("未找到匹配的内容，请检查原文是否正确")
                }
                matchCount > 1 -> {
                    EditResult.Error("找到 $matchCount 处匹配，请提供更多上下文以精确定位")
                }
                else -> {
                    // 唯一匹配，执行替换
                    val newContent = content.replaceFirst(oldString, newString)
                    Files.writeString(filePath, newContent)
                    EditResult.Success(
                        originalContent = content,
                        newContent = newContent
                    )
                }
            }
        }
    }
    
    private fun validateAndResolvePath(relativePath: String): Path {
        val resolved = workspacePath.resolve(relativePath).normalize()
        if (!resolved.startsWith(workspacePath)) {
            throw SecurityException("路径超出工作区范围: $relativePath")
        }
        return resolved
    }
}

sealed class EditResult {
    data class Success(
        val originalContent: String,
        val newContent: String
    ) : EditResult()
    
    data class Error(val message: String) : EditResult()
}
```

### 8.5 删除工具 (DeleteTool)

```kotlin
/**
 * 删除工具 - 软删除工作区内的文件或目录
 */
class DeleteTool(
    private val workspacePath: Path,
    private val recycleBinPath: Path,
    private val roleId: String
) {
    /**
     * 删除文件或目录（软删除，移动到回收站）
     * @param relativePath 相对于工作区的路径
     * @return 删除结果
     */
    suspend fun delete(relativePath: String): DeleteResult {
        val sourcePath = validateAndResolvePath(relativePath)
        
        return withContext(Dispatchers.IO) {
            if (!Files.exists(sourcePath)) {
                return@withContext DeleteResult.Error("路径不存在: $relativePath")
            }
            
            // 生成回收站中的新名称
            val timestamp = System.currentTimeMillis()
            val originalName = sourcePath.fileName.toString()
            val recycleName = "${roleId}-${timestamp}-${originalName}"
            val targetPath = recycleBinPath.resolve(recycleName)
            
            try {
                // 确保回收站目录存在
                Files.createDirectories(recycleBinPath)
                
                // 移动到回收站
                Files.move(sourcePath, targetPath, StandardCopyOption.ATOMIC_MOVE)
                
                DeleteResult.Success(
                    originalPath = relativePath,
                    recyclePath = targetPath.toString(),
                    recycleName = recycleName
                )
            } catch (e: Exception) {
                DeleteResult.Error("删除失败: ${e.message}")
            }
        }
    }
    
    private fun validateAndResolvePath(relativePath: String): Path {
        val resolved = workspacePath.resolve(relativePath).normalize()
        if (!resolved.startsWith(workspacePath)) {
            throw SecurityException("路径超出工作区范围: $relativePath")
        }
        return resolved
    }
}

sealed class DeleteResult {
    data class Success(
        val originalPath: String,
        val recyclePath: String,
        val recycleName: String
    ) : DeleteResult()
    
    data class Error(val message: String) : DeleteResult()
}
```

### 8.6 浏览器工具 (BrowserTool)

**设计方案B**: 共享浏览器实例，使用独立的用户数据目录 + 端口持久化和动态分配

```kotlin
/**
 * 浏览器工具 - 通过CDP协议控制浏览器
 * 
 * 设计方案：
 * - 共享浏览器实例，每个角色使用独立的用户数据目录
 * - 通过 --user-data-dir 参数为每个角色创建独立的浏览器配置
 * - 端口持久化：优先使用上次分配的端口，被占用时再动态分配
 * - 启动时结束所有浏览器进程以确保端口释放
 * - 使用共享的Chrome实例，通过不同的CDP端口连接
 * - 平衡资源消耗和隔离性
 */
class BrowserTool(
    private val roleId: String,
    private val workspacePath: Path,
    private val baseCdpPort: Int = 9222,
    private val maxPortRange: Int = 1000
) {
    private val httpClient = HttpClient(CIO)
    private var webSocket: WebSocketSession? = null
    
    // 每个角色有独立的用户数据目录
    private val userDataDir: Path = workspacePath.resolve("tools/browser/user_data")
    
    // 工具状态文件路径（用于持久化端口分配）
    private val toolStatePath: Path = workspacePath.resolve("tools/tool_state.json")
    
    // 实际分配的CDP端口
    private var assignedCdpPort: Int? = null
    
    companion object {
        /**
         * 系统启动时调用：结束所有浏览器进程以确保端口释放
         * 应在应用启动时执行一次
         */
        fun killAllBrowserProcesses() {
            try {
                val os = System.getProperty("os.name").lowercase()
                val processBuilder = when {
                    os.contains("win") -> ProcessBuilder(
                        "taskkill", "/F", "/IM", "chrome.exe", "/T"
                    )
                    os.contains("mac") -> ProcessBuilder(
                        "pkill", "-9", "Google Chrome"
                    )
                    else -> ProcessBuilder(
                        "pkill", "-9", "chrome"
                    )
                }
                processBuilder.inheritIO().start()
                // 等待进程结束
                Thread.sleep(1000)
            } catch (e: Exception) {
                // 忽略错误（可能没有运行的浏览器进程）
            }
        }
    }
    
    /**
     * 获取当前角色的CDP端口
     * 策略：
     * 1. 如果已分配，直接返回
     * 2. 尝试读取持久化的端口配置
     * 3. 如果持久化端口可用，使用它
     * 4. 否则动态分配新端口并持久化
     */
    suspend fun getCdpPort(): Int {
        assignedCdpPort?.let { return it }
        
        // 尝试读取持久化的端口
        val persistedPort = loadPersistedPort()
        
        if (persistedPort != null && isPortAvailable(persistedPort)) {
            // 使用持久化的端口
            assignedCdpPort = persistedPort
            return persistedPort
        }
        
        // 动态分配新端口
        val newPort = allocatePort()
        assignedCdpPort = newPort
        
        // 持久化端口配置
        persistPort(newPort)
        
        return newPort
    }
    
    /**
     * 从配置文件加载持久化的端口
     */
    private suspend fun loadPersistedPort(): Int? {
        return withContext(Dispatchers.IO) {
            try {
                if (!Files.exists(toolStatePath)) {
                    return@withContext null
                }
                
                val content = Files.readString(toolStatePath)
                val state = Json.decodeFromString<ToolState>(content)
                state.browserCdpPort
            } catch (e: Exception) {
                null
            }
        }
    }
    
    /**
     * 持久化端口配置
     */
    private suspend fun persistPort(port: Int) {
        withContext(Dispatchers.IO) {
            try {
                Files.createDirectories(toolStatePath.parent)
                
                val state = ToolState(
                    roleId = roleId,
                    browserCdpPort = port,
                    updatedAt = Instant.now()
                )
                
                Files.writeString(toolStatePath, Json.encodeToString(state))
            } catch (e: Exception) {
                // 忽略持久化错误
            }
        }
    }
    
    /**
     * 动态分配可用端口
     * 策略：基于角色ID哈希起始，扫描找到第一个可用端口
     */
    private suspend fun allocatePort(): Int {
        // 计算起始端口（基于角色ID的哈希，增加随机性）
        val hashOffset = (roleId.hashCode() and 0x7FFFFFFF) % maxPortRange
        
        // 扫描可用端口
        for (offset in 0 until maxPortRange) {
            val port = baseCdpPort + ((hashOffset + offset) % maxPortRange)
            
            if (isPortAvailable(port)) {
                return port
            }
        }
        
        throw IllegalStateException(
            "无法为角色 $roleId 分配可用的CDP端口，" +
            "端口范围 ${baseCdpPort}-${baseCdpPort + maxPortRange} 已被占用"
        )
    }
    
    /**
     * 检测端口是否可用
     * 端口可用条件：未被CDP占用且未被系统其他进程占用
     */
    private suspend fun isPortAvailable(port: Int): Boolean {
        // 1. 检测是否被CDP占用（有浏览器实例在运行）
        val cdpInUse = try {
            val response = httpClient.get("http://localhost:$port/json/version") {
                timeout { requestTimeoutMillis = 1000 }
            }
            response.status == HttpStatusCode.OK
        } catch (e: Exception) {
            false
        }
        
        if (cdpInUse) {
            return false
        }
        
        // 2. 检测是否被系统其他进程占用
        return !isPortInUseBySystem(port)
    }
    
    /**
     * 检测端口是否被系统其他进程占用
     */
    private fun isPortInUseBySystem(port: Int): Boolean {
        return try {
            ServerSocket(port).use { 
                // 端口可用
                false 
            }
        } catch (e: Exception) {
            // 端口被占用
            true
        }
    }
    
    /**
     * 获取浏览器启动参数
     * @return Chrome启动参数列表
     */
    suspend fun getBrowserArgs(): List<String> {
        val port = getCdpPort()
        return listOf(
            "--remote-debugging-port=$port",
            "--user-data-dir=$userDataDir",
            "--no-first-run",
            "--no-default-browser-check",
            "--disable-default-apps",
            "--disable-extensions",
            "--disable-background-networking",
            "--disable-background-timer-throttling",
            "--disable-backgrounding-occluded-windows",
            "--disable-breakpad",
            "--disable-component-update",
            "--disable-default-apps",
            "--disable-features=TranslateUI",
            "--disable-hang-monitor",
            "--disable-ipc-flooding-protection",
            "--disable-popup-blocking",
            "--disable-prompt-on-repost",
            "--disable-renderer-backgrounding",
            "--force-color-profile=srgb",
            "--metrics-recording-only",
            "--safebrowsing-disable-auto-update",
            "--enable-automation",
            "--password-store=basic",
            "--use-mock-keychain",
            "--headless=new"  // 使用新的无头模式
        )
    }
    
    // 浏览器进程
    private var browserProcess: Process? = null
    
    /**
     * 确保浏览器连接
     * 如果浏览器未运行，自动启动并恢复连接
     */
    private suspend fun ensureConnection(): Boolean {
        val port = getCdpPort()
        
        // 尝试连接CDP
        val isConnected = try {
            val response = httpClient.get("http://localhost:$port/json/version")
            response.status == HttpStatusCode.OK
        } catch (e: Exception) {
            false
        }
        
        if (isConnected) {
            return true
        }
        
        // 浏览器未运行，尝试自动启动
        return try {
            startBrowser()
        } catch (e: Exception) {
            Logger.error("自动启动浏览器失败: ${e.message}")
            false
        }
    }
    
    /**
     * 启动浏览器
     */
    private suspend fun startBrowser(): Boolean {
        return withContext(Dispatchers.IO) {
            try {
                val port = getCdpPort()
                val chromePath = findChromeExecutable()
                    ?: throw IllegalStateException("未找到Chrome可执行文件")
                
                val args = getBrowserArgs()
                
                Logger.info("启动浏览器，角色: $roleId, 端口: $port")
                
                browserProcess = ProcessBuilder(listOf(chromePath.toString()) + args)
                    .redirectOutput(ProcessBuilder.Redirect.DISCARD)
                    .redirectError(ProcessBuilder.Redirect.DISCARD)
                    .start()
                
                // 等待浏览器启动
                var attempts = 0
                while (attempts < 30) {
                    delay(500)
                    try {
                        val response = httpClient.get("http://localhost:$port/json/version")
                        if (response.status == HttpStatusCode.OK) {
                            Logger.info("浏览器启动成功")
                            return@withContext true
                        }
                    } catch (e: Exception) {
                        // 继续等待
                    }
                    attempts++
                }
                
                throw IllegalStateException("浏览器启动超时")
            } catch (e: Exception) {
                Logger.error("启动浏览器失败: ${e.message}")
                false
            }
        }
    }
    
    /**
     * 查找Chrome可执行文件
     */
    private fun findChromeExecutable(): Path? {
        val os = System.getProperty("os.name").lowercase()
        val possiblePaths = when {
            os.contains("win") -> listOf(
                Paths.get("C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe"),
                Paths.get("C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe"),
                Paths.get(System.getenv("LOCALAPPDATA") ?: "", "Google\\Chrome\\Application\\chrome.exe")
            )
            os.contains("mac") -> listOf(
                Paths.get("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome")
            )
            else -> listOf(
                Paths.get("/usr/bin/google-chrome"),
                Paths.get("/usr/bin/chromium-browser"),
                Paths.get("/usr/bin/chromium")
            )
        }
        
        return possiblePaths.firstOrNull { Files.exists(it) }
    }
    
    /**
     * 关闭浏览器
     */
    suspend fun closeBrowser() {
        withContext(Dispatchers.IO) {
            try {
                // 先尝试通过CDP优雅关闭
                try {
                    sendCDPCommand("Browser.close", emptyMap())
                } catch (e: Exception) {
                    // 忽略
                }
                
                // 强制终止进程
                browserProcess?.destroyForcibly()
                browserProcess = null
                
                // 关闭WebSocket
                webSocket?.close()
                webSocket = null
                
                Logger.info("浏览器已关闭")
            } catch (e: Exception) {
                Logger.warn("关闭浏览器时出错: ${e.message}")
            }
        }
    }
    
    /**
     * 打开网页
     * @param url 目标URL
     * @return 页面加载结果
     */
    suspend fun navigate(url: String): BrowserResult {
        return try {
            if (!ensureConnection()) {
                return BrowserResult.Error("浏览器未连接，请确保浏览器已启动")
            }
            
            // 通过CDP发送导航命令
            val response = sendCDPCommand(
                "Page.navigate",
                mapOf("url" to url)
            )
            BrowserResult.Success("页面加载完成: $url")
        } catch (e: Exception) {
            BrowserResult.Error("导航失败: ${e.message}")
        }
    }
    
    /**
     * 获取页面内容
     * @return 页面HTML内容
     */
    suspend fun getContent(): BrowserResult {
        return try {
            if (!ensureConnection()) {
                return BrowserResult.Error("浏览器未连接")
            }
            
            val response = sendCDPCommand(
                "Runtime.evaluate",
                mapOf("expression" to "document.documentElement.outerHTML")
            )
            BrowserResult.Success(response)
        } catch (e: Exception) {
            BrowserResult.Error("获取内容失败: ${e.message}")
        }
    }
    
    /**
     * 执行JavaScript
     * @param script JavaScript代码
     * @return 执行结果
     */
    suspend fun executeScript(script: String): BrowserResult {
        return try {
            if (!ensureConnection()) {
                return BrowserResult.Error("浏览器未连接")
            }
            
            val response = sendCDPCommand(
                "Runtime.evaluate",
                mapOf("expression" to script)
            )
            BrowserResult.Success(response)
        } catch (e: Exception) {
            BrowserResult.Error("脚本执行失败: ${e.message}")
        }
    }
    
    /**
     * 截图
     * @param outputPath 截图保存路径（相对于工作区）
     * @return 截图结果
     */
    suspend fun screenshot(outputPath: String): BrowserResult {
        return try {
            if (!ensureConnection()) {
                return BrowserResult.Error("浏览器未连接")
            }
            
            val response = sendCDPCommand(
                "Page.captureScreenshot",
                mapOf("format" to "png", "fromSurface" to true)
            )
            
            // 解码base64并保存到工作区
            val screenshotData = Json.decodeFromString<CDPScreenshotResponse>(response)
            val imageBytes = Base64.getDecoder().decode(screenshotData.data)
            
            val fullOutputPath = workspacePath.resolve(outputPath)
            Files.createDirectories(fullOutputPath.parent)
            Files.write(fullOutputPath, imageBytes)
            
            BrowserResult.Success("截图已保存到: $outputPath")
        } catch (e: Exception) {
            BrowserResult.Error("截图失败: ${e.message}")
        }
    }
    
    /**
     * 点击元素
     * @param selector CSS选择器
     * @return 操作结果
     */
    suspend fun click(selector: String): BrowserResult {
        return try {
            if (!ensureConnection()) {
                return BrowserResult.Error("浏览器未连接")
            }
            
            val script = """
                (function() {
                    const element = document.querySelector('$selector');
                    if (element) {
                        element.click();
                        return '点击成功';
                    } else {
                        return '元素未找到: $selector';
                    }
                })()
            """.trimIndent()
            
            val response = sendCDPCommand(
                "Runtime.evaluate",
                mapOf("expression" to script)
            )
            BrowserResult.Success(response)
        } catch (e: Exception) {
            BrowserResult.Error("点击失败: ${e.message}")
        }
    }
    
    /**
     * 输入文本
     * @param selector CSS选择器
     * @param text 要输入的文本
     * @return 操作结果
     */
    suspend fun type(selector: String, text: String): BrowserResult {
        return try {
            if (!ensureConnection()) {
                return BrowserResult.Error("浏览器未连接")
            }
            
            val script = """
                (function() {
                    const element = document.querySelector('$selector');
                    if (element) {
                        element.value = '$text';
                        element.dispatchEvent(new Event('input', { bubbles: true }));
                        element.dispatchEvent(new Event('change', { bubbles: true }));
                        return '输入成功';
                    } else {
                        return '元素未找到: $selector';
                    }
                })()
            """.trimIndent()
            
            val response = sendCDPCommand(
                "Runtime.evaluate",
                mapOf("expression" to script)
            )
            BrowserResult.Success(response)
        } catch (e: Exception) {
            BrowserResult.Error("输入失败: ${e.message}")
        }
    }
    
    private suspend fun sendCDPCommand(method: String, params: Map<String, Any>): String {
        val port = getCdpPort()
        
        // 1. 获取可用的WebSocket调试页面
        val pagesResponse = httpClient.get("http://localhost:$port/json/list")
        val pages = Json.decodeFromString<List<CDPPage>>(pagesResponse.bodyAsText())
        
        if (pages.isEmpty()) {
            throw IllegalStateException("没有可用的浏览器页面")
        }
        
        val page = pages.first()
        
        // 2. 建立WebSocket连接（如果尚未连接）
        if (webSocket == null || webSocket?.isActive != true) {
            webSocket = httpClient.webSocketSession(page.webSocketDebuggerUrl)
        }
        
        // 3. 发送CDP命令
        val command = CDPCommand(
            id = System.currentTimeMillis().toInt(),
            method = method,
            params = params
        )
        
        webSocket?.send(Json.encodeToString(command))
        
        // 4. 接收响应
        val response = webSocket?.incoming?.receive() as? Frame.Text
            ?: throw IllegalStateException("未收到响应")
        
        return response.readText()
    }
    
    /**
     * 关闭浏览器连接
     */
    suspend fun close() {
        webSocket?.close()
        httpClient.close()
    }
}

@Serializable
data class CDPCommand(
    val id: Int,
    val method: String,
    val params: Map<String, @Contextual Any>
)

@Serializable
data class CDPPage(
    val id: String,
    val title: String,
    val type: String,
    val url: String,
    @SerialName("webSocketDebuggerUrl") val webSocketDebuggerUrl: String
)

@Serializable
data class CDPScreenshotResponse(
    val data: String
)

/**
 * 工具状态数据类
 * 用于持久化工具配置（如CDP端口分配）
 */
@Serializable
data class ToolState(
    val roleId: String,
    val browserCdpPort: Int? = null,
    val updatedAt: Instant
)

sealed class BrowserResult {
    data class Success(val data: String) : BrowserResult()
    data class Error(val message: String) : BrowserResult()
}
```

### 8.7 技能创建和修改工具 (SkillTool)

**设计方案**: 使用系统Python + 角色工作区虚拟环境隔离依赖 + 提示词约束

```kotlin
/**
 * 技能工具 - 创建和管理角色的独有技能
 * 
 * 设计方案：
 * - 使用系统Python解释器执行技能代码
 * - 每个角色有独立的虚拟环境（venv），隔离依赖
 * - 虚拟环境位于角色工作区内
 * - 技能代码在独立进程中执行，确保安全
 * 
 * 安全约束（通过系统提示词告知AI角色）：
 * - AI角色不应通过Python技能执行越界行为
 * - 禁止访问工作区外的文件系统
 * - 禁止执行网络攻击、恶意代码等有害操作
 * - 技能应专注于完成特定任务，而非破坏系统
 */
class SkillTool(
    private val workspacePath: Path,
    private val roleId: String
) {
    private val skillsPath = workspacePath.resolve("skills/custom")
    private val venvPath = workspacePath.resolve("tools/python_venv")
    
    /**
     * 获取Python解释器路径
     */
    private fun getPythonExecutable(): Path {
        return if (System.getProperty("os.name").lowercase().contains("win")) {
            venvPath.resolve("Scripts/python.exe")
        } else {
            venvPath.resolve("bin/python")
        }
    }
    
    /**
     * 确保虚拟环境存在
     */
    private suspend fun ensureVenv(): Boolean {
        return withContext(Dispatchers.IO) {
            if (Files.exists(getPythonExecutable())) {
                return@withContext true
            }
            
            // 创建虚拟环境
            try {
                val process = ProcessBuilder(
                    "python", "-m", "venv", venvPath.toString()
                ).start()
                
                process.waitFor(60, TimeUnit.SECONDS)
                Files.exists(getPythonExecutable())
            } catch (e: Exception) {
                false
            }
        }
    }
    
    /**
     * 安装Python依赖
     * @param packages 依赖包列表
     * @return 安装结果
     */
    suspend fun installDependencies(packages: List<String>): SkillResult {
        return withContext(Dispatchers.IO) {
            try {
                if (!ensureVenv()) {
                    return@withContext SkillResult.Error("无法创建Python虚拟环境")
                }
                
                val pipExecutable = if (System.getProperty("os.name").lowercase().contains("win")) {
                    venvPath.resolve("Scripts/pip.exe")
                } else {
                    venvPath.resolve("bin/pip")
                }
                
                val process = ProcessBuilder(
                    listOf(pipExecutable.toString(), "install") + packages
                ).start()
                
                val success = process.waitFor(120, TimeUnit.SECONDS)
                
                if (success && process.exitValue() == 0) {
                    SkillResult.Success("依赖安装成功: ${packages.joinToString(", ")}")
                } else {
                    val error = process.errorStream.bufferedReader().readText()
                    SkillResult.Error("依赖安装失败: $error")
                }
            } catch (e: Exception) {
                SkillResult.Error("安装依赖失败: ${e.message}")
            }
        }
    }
    
    /**
     * 创建新技能
     * @param skillId 技能ID
     * @param name 技能名称
     * @param description 技能描述
     * @param pythonCode Python实现代码
     * @param dependencies 依赖包列表（可选）
     * @return 创建结果
     */
    suspend fun createSkill(
        skillId: String,
        name: String,
        description: String,
        pythonCode: String,
        dependencies: List<String> = emptyList()
    ): SkillResult {
        return withContext(Dispatchers.IO) {
            try {
                // 确保虚拟环境存在
                if (!ensureVenv()) {
                    return@withContext SkillResult.Error("无法创建Python虚拟环境")
                }
                
                val skillDir = skillsPath.resolve(skillId)
                Files.createDirectories(skillDir)
                
                // 创建技能清单
                val manifest = SkillManifest(
                    id = skillId,
                    name = name,
                    description = description,
                    version = "1.0.0",
                    createdBy = roleId,
                    createdAt = Instant.now(),
                    entryPoint = "main.py",
                    dependencies = dependencies
                )
                
                // 保存清单
                val manifestPath = skillDir.resolve("manifest.json")
                Files.writeString(
                    manifestPath,
                    Json.encodeToString(manifest)
                )
                
                // 保存Python代码
                val codePath = skillDir.resolve("main.py")
                Files.writeString(codePath, pythonCode)
                
                // 安装依赖
                if (dependencies.isNotEmpty()) {
                    val depResult = installDependencies(dependencies)
                    if (depResult is SkillResult.Error) {
                        return@withContext depResult
                    }
                }
                
                // 立即加载技能到当前角色
                loadSkill(skillId, skillDir)
                
                SkillResult.Success("技能 '$name' 创建成功并已加载")
            } catch (e: Exception) {
                SkillResult.Error("创建技能失败: ${e.message}")
            }
        }
    }
    
    /**
     * 修改技能代码
     * @param skillId 技能ID
     * @param newCode 新的Python代码
     * @return 修改结果
     */
    suspend fun updateSkill(
        skillId: String,
        newCode: String
    ): SkillResult {
        return withContext(Dispatchers.IO) {
            try {
                val skillDir = skillsPath.resolve(skillId)
                val codePath = skillDir.resolve("main.py")
                
                if (!Files.exists(codePath)) {
                    return@withContext SkillResult.Error("技能不存在: $skillId")
                }
                
                // 备份旧代码（使用版本号管理）
                val manifestPath = skillDir.resolve("manifest.json")
                val manifest = Json.decodeFromString<SkillManifest>(
                    Files.readString(manifestPath)
                )
                val backupDir = skillDir.resolve("versions")
                Files.createDirectories(backupDir)
                val backupPath = backupDir.resolve("main_v${manifest.version}.py")
                Files.copy(codePath, backupPath, StandardCopyOption.REPLACE_EXISTING)
                
                // 写入新代码
                Files.writeString(codePath, newCode)
                
                // 更新版本号
                val newManifest = manifest.copy(
                    version = incrementVersion(manifest.version),
                    updatedAt = Instant.now()
                )
                Files.writeString(manifestPath, Json.encodeToString(newManifest))
                
                // 重新加载技能
                reloadSkill(skillId)
                
                SkillResult.Success("技能 '$skillId' 更新成功并已重新加载")
            } catch (e: Exception) {
                SkillResult.Error("更新技能失败: ${e.message}")
            }
        }
    }
    
    /**
     * 执行技能
     * @param skillId 技能ID
     * @param params 执行参数
     * @return 执行结果
     */
    suspend fun executeSkill(
        skillId: String,
        params: Map<String, Any>
    ): SkillExecutionResult {
        return withContext(Dispatchers.IO) {
            try {
                val skillDir = skillsPath.resolve(skillId)
                val codePath = skillDir.resolve("main.py")
                
                if (!Files.exists(codePath)) {
                    return@withContext SkillExecutionResult.Error("技能不存在: $skillId")
                }
                
                // 创建临时参数文件
                val paramsPath = skillDir.resolve("params.json")
                Files.writeString(paramsPath, Json.encodeToString(params))
                
                // 执行Python脚本
                val process = ProcessBuilder(
                    getPythonExecutable().toString(),
                    codePath.toString(),
                    paramsPath.toString()
                )
                    .directory(skillDir.toFile())
                    .redirectErrorStream(true)
                    .start()
                
                // 设置超时
                val completed = process.waitFor(30, TimeUnit.SECONDS)
                
                if (!completed) {
                    process.destroyForcibly()
                    return@withContext SkillExecutionResult.Error("技能执行超时")
                }
                
                val output = process.inputStream.bufferedReader().readText()
                
                if (process.exitValue() == 0) {
                    SkillExecutionResult.Success(output)
                } else {
                    SkillExecutionResult.Error("执行失败: $output")
                }
            } catch (e: Exception) {
                SkillExecutionResult.Error("执行技能失败: ${e.message}")
            }
        }
    }
    
    /**
     * 获取技能历史版本
     * @param skillId 技能ID
     * @return 版本列表
     */
    suspend fun getSkillVersions(skillId: String): List<String> {
        return withContext(Dispatchers.IO) {
            try {
                val versionsDir = skillsPath.resolve(skillId).resolve("versions")
                if (!Files.exists(versionsDir)) {
                    return@withContext emptyList()
                }
                
                Files.list(versionsDir)
                    .map { it.fileName.toString() }
                    .filter { it.startsWith("main_v") && it.endsWith(".py") }
                    .sorted()
                    .toList()
            } catch (e: Exception) {
                emptyList()
            }
        }
    }
    
    /**
     * 回滚到指定版本
     * @param skillId 技能ID
     * @param version 版本号
     * @return 回滚结果
     */
    suspend fun rollbackSkill(skillId: String, version: String): SkillResult {
        return withContext(Dispatchers.IO) {
            try {
                val skillDir = skillsPath.resolve(skillId)
                val backupPath = skillDir.resolve("versions/main_v$version.py")
                val codePath = skillDir.resolve("main.py")
                
                if (!Files.exists(backupPath)) {
                    return@withContext SkillResult.Error("版本不存在: $version")
                }
                
                // 备份当前版本
                val manifestPath = skillDir.resolve("manifest.json")
                val manifest = Json.decodeFromString<SkillManifest>(
                    Files.readString(manifestPath)
                )
                val currentBackupPath = skillDir.resolve(
                    "versions/main_v${manifest.version}.py"
                )
                Files.copy(codePath, currentBackupPath, StandardCopyOption.REPLACE_EXISTING)
                
                // 恢复指定版本
                Files.copy(backupPath, codePath, StandardCopyOption.REPLACE_EXISTING)
                
                // 更新版本号
                val newManifest = manifest.copy(
                    version = incrementVersion(manifest.version),
                    updatedAt = Instant.now()
                )
                Files.writeString(manifestPath, Json.encodeToString(newManifest))
                
                // 重新加载
                reloadSkill(skillId)
                
                SkillResult.Success("技能已回滚到版本 $version")
            } catch (e: Exception) {
                SkillResult.Error("回滚失败: ${e.message}")
            }
        }
    }
    
    /**
     * 版本号递增
     */
    private fun incrementVersion(version: String): String {
        val parts = version.split(".").map { it.toInt() }.toMutableList()
        parts[2] = parts[2] + 1
        return parts.joinToString(".")
    }
    
    /**
     * 加载技能到当前角色
     */
    private suspend fun loadSkill(skillId: String, skillDir: Path) {
        // 实现技能加载逻辑
        // 验证技能代码语法，注册到技能管理器
    }
    
    /**
     * 重新加载技能
     */
    private suspend fun reloadSkill(skillId: String) {
        // 实现技能重载逻辑
    }
}

@Serializable
data class SkillManifest(
    val id: String,
    val name: String,
    val description: String,
    val version: String,
    val createdBy: String,
    val createdAt: Instant,
    val updatedAt: Instant? = null,
    val entryPoint: String,
    val dependencies: List<String> = emptyList()
)

sealed class SkillResult {
    data class Success(val message: String) : SkillResult()
    data class Error(val message: String) : SkillResult()
}

sealed class SkillExecutionResult {
    data class Success(val output: String) : SkillExecutionResult()
    data class Error(val message: String) : SkillExecutionResult()
}
```

### 8.8 记忆查找工具 (MemoryTool)

```kotlin
/**
 * 记忆工具 - 从mem0检索角色记忆
 */
class MemoryTool(
    private val mem0Client: Mem0Client,
    private val roleId: String
) {
    /**
     * 搜索相关记忆
     * @param query 查询内容
     * @param limit 返回数量限制
     * @return 记忆列表
     */
    suspend fun search(
        query: String,
        limit: Int = 10
    ): MemoryResult {
        return try {
            val memories = mem0Client.searchMemories(
                userId = "role_$roleId",
                query = query,
                limit = limit
            )
            MemoryResult.Success(memories)
        } catch (e: Exception) {
            MemoryResult.Error("搜索记忆失败: ${e.message}")
        }
    }
    
    /**
     * 获取最近添加的记忆
     * @param limit 返回数量限制
     * @return 记忆列表
     */
    suspend fun getRecent(limit: Int = 10): MemoryResult {
        return try {
            val memories = mem0Client.getAllMemories(
                userId = "role_$roleId",
                limit = limit
            )
            MemoryResult.Success(memories)
        } catch (e: Exception) {
            MemoryResult.Error("获取记忆失败: ${e.message}")
        }
    }
    
    /**
     * 获取特定类型的记忆
     * @param type 记忆类型（如 "session_summary", "user_preference"）
     * @param limit 返回数量限制
     * @return 记忆列表
     */
    suspend fun getByType(
        type: String,
        limit: Int = 10
    ): MemoryResult {
        return try {
            val allMemories = mem0Client.getAllMemories(
                userId = "role_$roleId"
            )
            val filtered = allMemories.filter { 
                it.metadata["type"] == type 
            }.take(limit)
            MemoryResult.Success(filtered)
        } catch (e: Exception) {
            MemoryResult.Error("获取记忆失败: ${e.message}")
        }
    }
}

sealed class MemoryResult {
    data class Success(val memories: List<Memory>) : MemoryResult()
    data class Error(val message: String) : MemoryResult()
}
```

### 8.9 自我认知和行为准则修改工具 (SelfAwareTool)

**设计方案**: 
- 不允许修改核心属性（id, createdAt）
- 只允许修改与提示词相关的可配置内容
- 保留版本历史供用户回退
- 无需用户审核，自动生效

```kotlin
/**
 * 自我认知工具 - 查看和修改角色的认知和行为准则
 * 
 * 限制说明：
 * - 核心属性（id, createdAt）不可修改
 * - 只允许修改与提示词相关的内容（name, description, personality, behaviorRules, systemPrompt）
 * - 修改自动生效，无需审核
 * - 保留版本历史供用户回退
 */
class SelfAwareTool(
    private val workspacePath: Path,
    private val roleId: String
) {
    private val configPath = workspacePath.resolve("config")
    private val profilePath = configPath.resolve("profile.json")
    private val behaviorRulesPath = configPath.resolve("behavior_rules.md")
    private val systemPromptPath = configPath.resolve("system_prompt.md")
    private val versionsPath = configPath.resolve("versions")
    
    /**
     * 获取当前角色配置
     * @return 角色配置（只读视图，核心属性不可修改）
     */
    suspend fun getProfile(): SelfAwareResult {
        return withContext(Dispatchers.IO) {
            try {
                if (!Files.exists(profilePath)) {
                    return@withContext SelfAwareResult.Error("角色配置文件不存在")
                }
                
                val content = Files.readString(profilePath)
                val profile = Json.decodeFromString<AIRoleProfile>(content)
                SelfAwareResult.ProfileSuccess(profile)
            } catch (e: Exception) {
                SelfAwareResult.Error("读取配置失败: ${e.message}")
            }
        }
    }
    
    /**
     * 更新角色可配置属性
     * 可修改：name, description, personality
     * 不可修改：id, createdAt
     * @param updates 要更新的字段
     * @return 更新结果
     */
    suspend fun updateProfile(
        updates: ProfileUpdates
    ): SelfAwareResult {
        return withContext(Dispatchers.IO) {
            try {
                val currentProfile = when (val result = getProfile()) {
                    is SelfAwareResult.ProfileSuccess -> result.profile
                    else -> return@withContext SelfAwareResult.Error("无法读取当前配置")
                }
                
                // 备份当前配置
                backupConfigFile(profilePath, "profile")
                
                // 只更新允许修改的字段
                val newProfile = currentProfile.copy(
                    name = updates.name ?: currentProfile.name,
                    description = updates.description ?: currentProfile.description,
                    personality = updates.personality ?: currentProfile.personality,
                    updatedAt = Instant.now()
                    // id 和 createdAt 不允许修改
                )
                
                Files.writeString(
                    profilePath,
                    Json.encodeToString(newProfile)
                )
                
                // 触发角色重新加载配置
                notifyConfigChanged()
                
                SelfAwareResult.Success("角色配置已更新")
            } catch (e: Exception) {
                SelfAwareResult.Error("更新配置失败: ${e.message}")
            }
        }
    }
    
    /**
     * 获取行为准则
     * @return 行为准则内容
     */
    suspend fun getBehaviorRules(): SelfAwareResult {
        return withContext(Dispatchers.IO) {
            try {
                if (!Files.exists(behaviorRulesPath)) {
                    return@withContext SelfAwareResult.BehaviorRulesSuccess("")
                }
                
                val content = Files.readString(behaviorRulesPath)
                SelfAwareResult.BehaviorRulesSuccess(content)
            } catch (e: Exception) {
                SelfAwareResult.Error("读取行为准则失败: ${e.message}")
            }
        }
    }
    
    /**
     * 更新行为准则
     * @param newRules 新的行为准则内容
     * @return 更新结果
     */
    suspend fun updateBehaviorRules(
        newRules: String
    ): SelfAwareResult {
        return withContext(Dispatchers.IO) {
            try {
                // 备份旧规则（版本化管理）
                backupConfigFile(behaviorRulesPath, "behavior_rules")
                
                Files.writeString(behaviorRulesPath, newRules)
                
                // 触发角色重新加载配置
                notifyConfigChanged()
                
                SelfAwareResult.Success("行为准则已更新")
            } catch (e: Exception) {
                SelfAwareResult.Error("更新行为准则失败: ${e.message}")
            }
        }
    }
    
    /**
     * 获取系统提示词
     * @return 系统提示词内容
     */
    suspend fun getSystemPrompt(): SelfAwareResult {
        return withContext(Dispatchers.IO) {
            try {
                if (!Files.exists(systemPromptPath)) {
                    return@withContext SelfAwareResult.SystemPromptSuccess("")
                }
                
                val content = Files.readString(systemPromptPath)
                SelfAwareResult.SystemPromptSuccess(content)
            } catch (e: Exception) {
                SelfAwareResult.Error("读取系统提示词失败: ${e.message}")
            }
        }
    }
    
    /**
     * 更新系统提示词
     * @param newPrompt 新的系统提示词
     * @return 更新结果
     */
    suspend fun updateSystemPrompt(
        newPrompt: String
    ): SelfAwareResult {
        return withContext(Dispatchers.IO) {
            try {
                // 备份旧提示词（版本化管理）
                backupConfigFile(systemPromptPath, "system_prompt")
                
                Files.writeString(systemPromptPath, newPrompt)
                
                // 触发角色重新加载配置
                notifyConfigChanged()
                
                SelfAwareResult.Success("系统提示词已更新")
            } catch (e: Exception) {
                SelfAwareResult.Error("更新系统提示词失败: ${e.message}")
            }
        }
    }
    
    /**
     * 获取配置历史版本列表
     * @param configType 配置类型（profile, behavior_rules, system_prompt）
     * @return 版本列表（按时间倒序）
     */
    suspend fun getConfigVersions(configType: String): List<ConfigVersion> {
        return withContext(Dispatchers.IO) {
            try {
                val typeVersionsPath = versionsPath.resolve(configType)
                if (!Files.exists(typeVersionsPath)) {
                    return@withContext emptyList()
                }
                
                Files.list(typeVersionsPath)
                    .filter { it.fileName.toString().matches(Regex("\\d{17}_.*")) }
                    .map { path ->
                        val fileName = path.fileName.toString()
                        val timestamp = fileName.substring(0, 17).toLong()
                        ConfigVersion(
                            timestamp = timestamp,
                            fileName = fileName,
                            path = path.toString()
                        )
                    }
                    .sortedByDescending { it.timestamp }
                    .toList()
            } catch (e: Exception) {
                emptyList()
            }
        }
    }
    
    /**
     * 回滚配置到指定版本
     * @param configType 配置类型
     * @param timestamp 版本时间戳
     * @return 回滚结果
     */
    suspend fun rollbackConfig(
        configType: String,
        timestamp: Long
    ): SelfAwareResult {
        return withContext(Dispatchers.IO) {
            try {
                val typeVersionsPath = versionsPath.resolve(configType)
                val backupFile = typeVersionsPath.resolve("${timestamp}_${configType}.backup")
                
                if (!Files.exists(backupFile)) {
                    return@withContext SelfAwareResult.Error("版本不存在: $timestamp")
                }
                
                // 备份当前版本
                val targetPath = when (configType) {
                    "profile" -> profilePath
                    "behavior_rules" -> behaviorRulesPath
                    "system_prompt" -> systemPromptPath
                    else -> return@withContext SelfAwareResult.Error("未知的配置类型: $configType")
                }
                backupConfigFile(targetPath, configType)
                
                // 恢复指定版本
                Files.copy(backupFile, targetPath, StandardCopyOption.REPLACE_EXISTING)
                
                // 触发角色重新加载配置
                notifyConfigChanged()
                
                SelfAwareResult.Success("配置已回滚到版本 ${formatTimestamp(timestamp)}")
            } catch (e: Exception) {
                SelfAwareResult.Error("回滚失败: ${e.message}")
            }
        }
    }
    
    /**
     * 备份配置文件
     */
    private suspend fun backupConfigFile(sourcePath: Path, configType: String) {
        if (!Files.exists(sourcePath)) {
            return
        }
        
        Files.createDirectories(versionsPath.resolve(configType))
        
        val timestamp = System.currentTimeMillis()
        val backupFileName = "${timestamp}_${configType}.backup"
        val backupPath = versionsPath.resolve(configType).resolve(backupFileName)
        
        Files.copy(sourcePath, backupPath, StandardCopyOption.REPLACE_EXISTING)
        
        // 清理旧版本（保留最近20个版本）
        cleanupOldVersions(configType, keepCount = 20)
    }
    
    /**
     * 清理旧版本
     */
    private suspend fun cleanupOldVersions(configType: String, keepCount: Int) {
        val typeVersionsPath = versionsPath.resolve(configType)
        if (!Files.exists(typeVersionsPath)) {
            return
        }
        
        val versions = Files.list(typeVersionsPath)
            .sorted { p1, p2 ->
                val t1 = p1.fileName.toString().substring(0, 17).toLongOrNull() ?: 0
                val t2 = p2.fileName.toString().substring(0, 17).toLongOrNull() ?: 0
                t2.compareTo(t1) // 倒序
            }
            .toList()
        
        if (versions.size > keepCount) {
            versions.drop(keepCount).forEach { Files.deleteIfExists(it) }
        }
    }
    
    /**
     * 格式化时间戳
     */
    private fun formatTimestamp(timestamp: Long): String {
        val instant = Instant.ofEpochMilli(timestamp)
        return instant.toString()
    }
    
    /**
     * 通知配置已更改
     */
    private fun notifyConfigChanged() {
        // 发送配置变更事件，触发角色重新加载
    }
}

@Serializable
data class AIRoleProfile(
    val id: String,              // 核心属性，不可修改
    val name: String,            // 可修改
    val description: String,     // 可修改
    val personality: String,     // 可修改
    val createdAt: Instant,      // 核心属性，不可修改
    val updatedAt: Instant       // 自动更新
)

data class ProfileUpdates(
    val name: String? = null,
    val description: String? = null,
    val personality: String? = null
    // 注意：不包含 id 和 createdAt
)

data class ConfigVersion(
    val timestamp: Long,
    val fileName: String,
    val path: String
)

sealed class SelfAwareResult {
    data class ProfileSuccess(val profile: AIRoleProfile) : SelfAwareResult()
    data class BehaviorRulesSuccess(val content: String) : SelfAwareResult()
    data class SystemPromptSuccess(val content: String) : SelfAwareResult()
    data class Success(val message: String) : SelfAwareResult()
    data class Error(val message: String) : SelfAwareResult()
}
```

### 8.10 工具管理器

```kotlin
/**
 * AI角色工具管理器 - 统一管理角色的所有工具
 * 
 * 说明：
 * - 所有AI角色默认拥有8个基础工具
 * - 基础工具不受技能黑白名单控制
 * - 扩展技能（installed目录）受黑白名单控制
 * - 自定义技能（custom目录）不受黑白名单限制
 */
class AIToolManager(
    private val roleId: String,
    private val workspacePath: Path,
    private val recycleBinPath: Path,
    private val mem0Client: Mem0Client,
    private val baseCdpPort: Int = 9222
) {
    // 基础工具 - 所有AI角色默认拥有，不受黑白名单控制
    val readTool: ReadTool = ReadTool(workspacePath)
    val findTool: FindTool = FindTool(workspacePath)
    val editTool: EditTool = EditTool(workspacePath)
    val deleteTool: DeleteTool = DeleteTool(workspacePath, recycleBinPath, roleId)
    val browserTool: BrowserTool = BrowserTool(roleId, workspacePath, baseCdpPort)
    val skillTool: SkillTool = SkillTool(workspacePath, roleId)
    val memoryTool: MemoryTool = MemoryTool(mem0Client, roleId)
    val selfAwareTool: SelfAwareTool = SelfAwareTool(workspacePath, roleId)
    
    /**
     * 获取所有基础工具列表
     * 用于向AI模型注册工具调用能力
     */
    fun getBasicTools(): List<ToolDefinition> {
        return listOf(
            ToolDefinition(
                name = "read",
                description = "读取工作区内的文件内容，支持指定行数范围",
                parameters = listOf(
                    ToolParameter("path", "string", "文件路径（相对于工作区）", true),
                    ToolParameter("offset", "integer", "起始行号（可选）", false),
                    ToolParameter("limit", "integer", "读取行数（可选）", false)
                )
            ),
            ToolDefinition(
                name = "find",
                description = "在工作区内查找文件，支持通配符",
                parameters = listOf(
                    ToolParameter("pattern", "string", "通配符模式，如 *.kt", true),
                    ToolParameter("path", "string", "搜索起始路径（可选，默认为当前目录）", false)
                )
            ),
            ToolDefinition(
                name = "edit",
                description = "修改工作区内的文件内容",
                parameters = listOf(
                    ToolParameter("path", "string", "文件路径", true),
                    ToolParameter("oldString", "string", "原文内容（用于定位）", true),
                    ToolParameter("newString", "string", "新内容（用于替换）", true)
                )
            ),
            ToolDefinition(
                name = "delete",
                description = "删除工作区内的文件或目录（软删除到回收站）",
                parameters = listOf(
                    ToolParameter("path", "string", "要删除的路径", true)
                )
            ),
            ToolDefinition(
                name = "browser",
                description = "浏览器控制工具，支持导航、获取内容、执行脚本、截图等",
                parameters = listOf(
                    ToolParameter("action", "string", "操作类型：navigate/getContent/executeScript/screenshot/click/type", true),
                    ToolParameter("url", "string", "URL（navigate操作）", false),
                    ToolParameter("script", "string", "JavaScript代码（executeScript操作）", false),
                    ToolParameter("outputPath", "string", "截图保存路径（screenshot操作）", false),
                    ToolParameter("selector", "string", "CSS选择器（click/type操作）", false),
                    ToolParameter("text", "string", "输入文本（type操作）", false)
                )
            ),
            ToolDefinition(
                name = "skill",
                description = "创建和管理角色的自定义技能",
                parameters = listOf(
                    ToolParameter("action", "string", "操作类型：create/update/execute/getVersions/rollback", true),
                    ToolParameter("skillId", "string", "技能ID", false),
                    ToolParameter("name", "string", "技能名称（create操作）", false),
                    ToolParameter("description", "string", "技能描述（create操作）", false),
                    ToolParameter("pythonCode", "string", "Python代码（create/update操作）", false),
                    ToolParameter("dependencies", "array", "依赖包列表（create操作）", false),
                    ToolParameter("params", "object", "执行参数（execute操作）", false),
                    ToolParameter("version", "string", "版本号（rollback操作）", false)
                )
            ),
            ToolDefinition(
                name = "memory",
                description = "从mem0检索角色记忆",
                parameters = listOf(
                    ToolParameter("action", "string", "操作类型：search/getRecent/getByType", true),
                    ToolParameter("query", "string", "查询内容（search操作）", false),
                    ToolParameter("type", "string", "记忆类型（getByType操作）", false),
                    ToolParameter("limit", "integer", "返回数量限制（可选）", false)
                )
            ),
            ToolDefinition(
                name = "selfAware",
                description = "查看和修改角色的认知和行为准则",
                parameters = listOf(
                    ToolParameter("action", "string", "操作类型：getProfile/updateProfile/getBehaviorRules/updateBehaviorRules/getSystemPrompt/updateSystemPrompt/getConfigVersions/rollbackConfig", true),
                    ToolParameter("name", "string", "角色名称（updateProfile操作）", false),
                    ToolParameter("description", "string", "角色描述（updateProfile操作）", false),
                    ToolParameter("personality", "string", "性格特征（updateProfile操作）", false),
                    ToolParameter("newRules", "string", "新行为准则（updateBehaviorRules操作）", false),
                    ToolParameter("newPrompt", "string", "新系统提示词（updateSystemPrompt操作）", false),
                    ToolParameter("configType", "string", "配置类型（getConfigVersions/rollbackConfig操作）", false),
                    ToolParameter("timestamp", "integer", "版本时间戳（rollbackConfig操作）", false)
                )
            )
        )
    }
    
    /**
     * 关闭所有工具资源
     */
    suspend fun close() {
        browserTool.close()
    }
}

data class ToolDefinition(
    val name: String,
    val description: String,
    val parameters: List<ToolParameter>
)

data class ToolParameter(
    val name: String,
    val type: String,
    val description: String,
    val required: Boolean
)
```

### 8.11 工具调用协议

```kotlin
/**
 * 工具调用请求
 */
@Serializable
data class ToolCallRequest(
    val toolName: String,
    val action: String,
    val parameters: Map<String, JsonElement>
)

/**
 * 工具调用响应
 */
@Serializable
data class ToolCallResponse(
    val success: Boolean,
    val data: JsonElement? = null,
    val error: String? = null
)

/**
 * 工具调用处理器
 */
class ToolCallHandler(
    private val toolManager: AIToolManager
) {
    suspend fun handle(request: ToolCallRequest): ToolCallResponse {
        return try {
            val result = when (request.toolName) {
                "read" -> handleReadTool(request)
                "find" -> handleFindTool(request)
                "edit" -> handleEditTool(request)
                "delete" -> handleDeleteTool(request)
                "browser" -> handleBrowserTool(request)
                "skill" -> handleSkillTool(request)
                "memory" -> handleMemoryTool(request)
                "selfAware" -> handleSelfAwareTool(request)
                else -> throw IllegalArgumentException("未知工具: ${request.toolName}")
            }
            
            ToolCallResponse(
                success = true,
                data = Json.encodeToJsonElement(result)
            )
        } catch (e: Exception) {
            ToolCallResponse(
                success = false,
                error = e.message
            )
        }
    }
    
    private suspend fun handleReadTool(request: ToolCallRequest): ReadResult {
        val path = request.parameters["path"]?.jsonPrimitive?.content
            ?: throw IllegalArgumentException("缺少path参数")
        val offset = request.parameters["offset"]?.jsonPrimitive?.intOrNull
        val limit = request.parameters["limit"]?.jsonPrimitive?.intOrNull
        
        return toolManager.readTool.read(path, offset, limit)
    }
    
    private suspend fun handleFindTool(request: ToolCallRequest): FindResult {
        val pattern = request.parameters["pattern"]?.jsonPrimitive?.content
            ?: throw IllegalArgumentException("缺少pattern参数")
        val path = request.parameters["path"]?.jsonPrimitive?.content ?: "."
        
        return toolManager.findTool.find(pattern, path)
    }
    
    private suspend fun handleEditTool(request: ToolCallRequest): EditResult {
        val path = request.parameters["path"]?.jsonPrimitive?.content
            ?: throw IllegalArgumentException("缺少path参数")
        val oldString = request.parameters["oldString"]?.jsonPrimitive?.content
            ?: throw IllegalArgumentException("缺少oldString参数")
        val newString = request.parameters["newString"]?.jsonPrimitive?.content
            ?: throw IllegalArgumentException("缺少newString参数")
        
        return toolManager.editTool.edit(path, oldString, newString)
    }
    
    private suspend fun handleDeleteTool(request: ToolCallRequest): DeleteResult {
        val path = request.parameters["path"]?.jsonPrimitive?.content
            ?: throw IllegalArgumentException("缺少path参数")
        
        return toolManager.deleteTool.delete(path)
    }
    
    private suspend fun handleBrowserTool(request: ToolCallRequest): BrowserResult {
        return when (request.action) {
            "navigate" -> {
                val url = request.parameters["url"]?.jsonPrimitive?.content
                    ?: throw IllegalArgumentException("缺少url参数")
                toolManager.browserTool.navigate(url)
            }
            "getContent" -> toolManager.browserTool.getContent()
            "executeScript" -> {
                val script = request.parameters["script"]?.jsonPrimitive?.content
                    ?: throw IllegalArgumentException("缺少script参数")
                toolManager.browserTool.executeScript(script)
            }
            "screenshot" -> {
                val outputPath = request.parameters["outputPath"]?.jsonPrimitive?.content
                    ?: throw IllegalArgumentException("缺少outputPath参数")
                toolManager.browserTool.screenshot(outputPath)
            }
            else -> throw IllegalArgumentException("未知的浏览器操作: ${request.action}")
        }
    }
    
    private suspend fun handleSkillTool(request: ToolCallRequest): SkillResult {
        return when (request.action) {
            "create" -> {
                val skillId = request.parameters["skillId"]?.jsonPrimitive?.content
                    ?: throw IllegalArgumentException("缺少skillId参数")
                val name = request.parameters["name"]?.jsonPrimitive?.content
                    ?: throw IllegalArgumentException("缺少name参数")
                val description = request.parameters["description"]?.jsonPrimitive?.content
                    ?: throw IllegalArgumentException("缺少description参数")
                val pythonCode = request.parameters["pythonCode"]?.jsonPrimitive?.content
                    ?: throw IllegalArgumentException("缺少pythonCode参数")
                toolManager.skillTool.createSkill(skillId, name, description, pythonCode)
            }
            "update" -> {
                val skillId = request.parameters["skillId"]?.jsonPrimitive?.content
                    ?: throw IllegalArgumentException("缺少skillId参数")
                val pythonCode = request.parameters["pythonCode"]?.jsonPrimitive?.content
                    ?: throw IllegalArgumentException("缺少pythonCode参数")
                toolManager.skillTool.updateSkill(skillId, pythonCode)
            }
            else -> throw IllegalArgumentException("未知的技能操作: ${request.action}")
        }
    }
    
    private suspend fun handleMemoryTool(request: ToolCallRequest): MemoryResult {
        return when (request.action) {
            "search" -> {
                val query = request.parameters["query"]?.jsonPrimitive?.content
                    ?: throw IllegalArgumentException("缺少query参数")
                val limit = request.parameters["limit"]?.jsonPrimitive?.intOrNull ?: 10
                toolManager.memoryTool.search(query, limit)
            }
            "getRecent" -> {
                val limit = request.parameters["limit"]?.jsonPrimitive?.intOrNull ?: 10
                toolManager.memoryTool.getRecent(limit)
            }
            "getByType" -> {
                val type = request.parameters["type"]?.jsonPrimitive?.content
                    ?: throw IllegalArgumentException("缺少type参数")
                val limit = request.parameters["limit"]?.jsonPrimitive?.intOrNull ?: 10
                toolManager.memoryTool.getByType(type, limit)
            }
            else -> throw IllegalArgumentException("未知的记忆操作: ${request.action}")
        }
    }
    
    private suspend fun handleSelfAwareTool(request: ToolCallRequest): SelfAwareResult {
        return when (request.action) {
            "getProfile" -> toolManager.selfAwareTool.getProfile()
            "updateProfile" -> {
                val name = request.parameters["name"]?.jsonPrimitive?.content
                val description = request.parameters["description"]?.jsonPrimitive?.content
                val personality = request.parameters["personality"]?.jsonPrimitive?.content
                toolManager.selfAwareTool.updateProfile(
                    ProfileUpdates(name, description, personality)
                )
            }
            "getBehaviorRules" -> toolManager.selfAwareTool.getBehaviorRules()
            "updateBehaviorRules" -> {
                val newRules = request.parameters["newRules"]?.jsonPrimitive?.content
                    ?: throw IllegalArgumentException("缺少newRules参数")
                toolManager.selfAwareTool.updateBehaviorRules(newRules)
            }
            "getSystemPrompt" -> toolManager.selfAwareTool.getSystemPrompt()
            "updateSystemPrompt" -> {
                val newPrompt = request.parameters["newPrompt"]?.jsonPrimitive?.content
                    ?: throw IllegalArgumentException("缺少newPrompt参数")
                toolManager.selfAwareTool.updateSystemPrompt(newPrompt)
            }
            else -> throw IllegalArgumentException("未知的自我认知操作: ${request.action}")
        }
    }
}
```

### 8.12 工作区目录结构（含工具）

```
workspaces/
└── {ai_role_id}/
    ├── config/
    │   ├── profile.json          # 角色配置（核心属性不可修改）
    │   ├── system_prompt.md      # 系统提示词（可修改，有版本历史）
    │   ├── behavior_rules.md     # 行为准则（可修改，有版本历史）
    │   ├── model_config.json     # 模型配置
    │   └── versions/             # 配置文件版本历史
    │       ├── profile/          # profile.json 的历史版本
    │       ├── behavior_rules/   # behavior_rules.md 的历史版本
    │       └── system_prompt/    # system_prompt.md 的历史版本
    ├── skills/
    │   ├── whitelist.json        # 技能白名单（黑白名单控制）
    │   ├── blacklist.json        # 技能黑名单（黑白名单控制）
    │   ├── installed/            # 已安装技能（黑白名单控制）
    │   │   ├── skill_1/
    │   │   ├── skill_2/
    │   │   └── ...
    │   └── custom/               # 角色自定义技能（不受黑白名单限制）
    │       ├── custom_skill_1/
    │       │   ├── manifest.json
    │       │   ├── main.py
    │       │   └── versions/     # 技能代码版本历史
    │       └── ...
    ├── memory/                   # (mem0 管理的记忆数据)
    ├── sessions/                 # 会话历史
    ├── tools/                    # 工具配置和状态
    │   ├── browser/              # 浏览器数据
    │   │   └── user_data/        # 独立的浏览器用户数据
    │   ├── python_venv/          # Python虚拟环境（技能执行用）
    │   └── tool_state.json       # 工具状态
    └── fatigue.json              # 疲劳值数据

recycle_bin/                      # 回收站目录
├── {role_id}-{timestamp}-file1.txt
├── {role_id}-{timestamp}-folder1/
└── ...
```

### 8.13 基础工具与技能黑白名单的关系

**基础工具**：所有AI角色默认拥有，不受黑白名单控制
- 读取工具 (ReadTool)
- 查找工具 (FindTool)
- 修改工具 (EditTool)
- 删除工具 (DeleteTool)
- 浏览器工具 (BrowserTool)
- 技能创建和修改工具 (SkillTool) - 用于创建自定义技能
- 记忆查找工具 (MemoryTool)
- 自我认知和行为准则修改工具 (SelfAwareTool)

**扩展技能**：通过黑白名单控制访问权限
- 位于 `skills/installed/` 目录
- 受 `whitelist.json` 和 `blacklist.json` 控制
- 白名单优先：如果在白名单中，允许使用
- 黑名单过滤：如果不在白名单中，检查是否在黑名单中

**自定义技能**：角色自己创建的技能
- 位于 `skills/custom/` 目录
- 不受黑白名单限制（因为是角色自己创建的）
- 有独立的版本管理机制

## 9. 客户端界面设计

### 9.1 设计原则

#### 界面风格
- **Notion卡片式布局**：采用类似Notion的卡片式布局，内容以卡片形式组织，层次分明
- **Apple Human Interface Guidelines**：遵循Apple的设计语言，注重清晰、尊重、深度三大原则
- **易用性**：用户操作直观，交互流畅，降低学习成本
- **跨平台一致性**：在不同平台（Desktop/Android/iOS/Web）上保持一致的视觉体验

#### 设计目标
- 采用卡片式布局，内容模块化呈现
- 使用圆角、阴影创造层次感
- 提供流畅的动画过渡效果
- 确保可访问性（Accessibility）
- 多色填充图标，视觉丰富但不过度

### 9.2 主题系统

#### 主题模式
- **亮色主题（Light）**：适合日间使用，高对比度确保可读性
- **暗色主题（Dark）**：适合夜间使用，减少眼睛疲劳
- **自动切换**：根据系统主题自动适配
- **手动设置**：用户可在设置中手动选择主题模式

#### 主题配置结构

```kotlin
/**
 * 主题配置数据类
 */
@Serializable
data class ThemeConfig(
    val mode: ThemeMode = ThemeMode.AUTO,
    val lightColors: ColorScheme = LightColorScheme,
    val darkColors: ColorScheme = DarkColorScheme,
    val typography: Typography = DefaultTypography,
    val shapes: Shapes = DefaultShapes
)

enum class ThemeMode {
    LIGHT,      // 强制亮色
    DARK,       // 强制暗色
    AUTO        // 跟随系统
}

/**
 * 颜色方案 - 基于 Apple Human Interface Guidelines
 * 使用更柔和、自然的色调
 */
@Serializable
data class ColorScheme(
    val primary: String,           // 主色调
    val onPrimary: String,         // 主色调上的文字
    val secondary: String,         // 次色调
    val onSecondary: String,       // 次色调上的文字
    val background: String,        // 背景色
    val onBackground: String,      // 背景上的文字
    val surface: String,           // 表面色（卡片、对话框等）
    val onSurface: String,         // 表面上的文字
    val error: String,             // 错误色
    val onError: String,           // 错误色上的文字
    val outline: String,           // 边框色
    val surfaceVariant: String,    // 表面变体色
    val onSurfaceVariant: String   // 表面变体上的文字
)

// 亮色主题默认配色 - Apple风格
val LightColorScheme = ColorScheme(
    primary = "#007AFF",           // iOS系统蓝
    onPrimary = "#FFFFFF",
    secondary = "#5856D6",         // iOS紫色
    onSecondary = "#FFFFFF",
    background = "#F5F5F7",        // Apple背景灰
    onBackground = "#1D1D1F",      // Apple文字黑
    surface = "#FFFFFF",           // 纯白卡片
    onSurface = "#1D1D1F",
    error = "#FF3B30",             // iOS红
    onError = "#FFFFFF",
    outline = "#E5E5EA",           // iOS分隔线色
    surfaceVariant = "#F2F2F7",    // iOS系统灰
    onSurfaceVariant = "#8E8E93"   // iOS次级文字
)

// 暗色主题默认配色 - Apple风格
val DarkColorScheme = ColorScheme(
    primary = "#0A84FF",           // iOS系统蓝（暗色）
    onPrimary = "#FFFFFF",
    secondary = "#5E5CE6",         // iOS紫色（暗色）
    onSecondary = "#FFFFFF",
    background = "#000000",        // 纯黑背景
    onBackground = "#FFFFFF",
    surface = "#1C1C1E",           // iOS卡片色（暗色）
    onSurface = "#FFFFFF",
    error = "#FF453A",             // iOS红（暗色）
    onError = "#FFFFFF",
    outline = "#38383A",           // iOS分隔线色（暗色）
    surfaceVariant = "#2C2C2E",    // iOS系统灰（暗色）
    onSurfaceVariant = "#8E8E93"   // iOS次级文字（暗色）
)
    background = "#1C1B1F",
    onBackground = "#E6E1E5",
    surface = "#1C1B1F",
    onSurface = "#E6E1E5",
    error = "#F2B8B5",
    onError = "#601410",
    outline = "#938F99",
    surfaceVariant = "#49454F",
    onSurfaceVariant = "#CAC4D0"
)
```

#### 主题管理器

使用 `multiplatform-settings` 进行跨平台主题持久化：

```kotlin
// build.gradle.kts 依赖
dependencies {
    implementation("com.russhwolf:multiplatform-settings:1.3.0")
    implementation("com.russhwolf:multiplatform-settings-coroutines:1.3.0")
    
    // Desktop平台
    implementation("com.russhwolf:multiplatform-settings-jvm:1.3.0")
    // Web平台
    implementation("com.russhwolf:multiplatform-settings-js:1.3.0")
}
```

```kotlin
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.ui.graphics.Color
import com.russhwolf.settings.ObservableSettings
import com.russhwolf.settings.Settings
import com.russhwolf.settings.coroutines.FlowSettings
import com.russhwolf.settings.coroutines.toFlowSettings
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/**
 * 主题模式
 */
@Serializable
enum class ThemeMode {
    LIGHT, DARK, AUTO
}

/**
 * 主题配置
 */
@Serializable
data class ThemeConfig(
    val mode: ThemeMode = ThemeMode.AUTO
)

/**
 * Apple风格颜色方案
 */
private val LightColors = lightColorScheme(
    primary = Color(0xFF007AFF),
    onPrimary = Color(0xFFFFFFFF),
    secondary = Color(0xFF5856D6),
    onSecondary = Color(0xFFFFFFFF),
    background = Color(0xFFF5F5F7),
    onBackground = Color(0xFF1D1D1F),
    surface = Color(0xFFFFFFFF),
    onSurface = Color(0xFF1D1D1F),
    error = Color(0xFFFF3B30),
    onError = Color(0xFFFFFFFF),
    outline = Color(0xFFE5E5EA),
    surfaceVariant = Color(0xFFF2F2F7),
    onSurfaceVariant = Color(0xFF8E8E93)
)

private val DarkColors = darkColorScheme(
    primary = Color(0xFF0A84FF),
    onPrimary = Color(0xFFFFFFFF),
    secondary = Color(0xFF5E5CE6),
    onSecondary = Color(0xFFFFFFFF),
    background = Color(0xFF000000),
    onBackground = Color(0xFFFFFFFF),
    surface = Color(0xFF1C1C1E),
    onSurface = Color(0xFFFFFFFF),
    error = Color(0xFFFF453A),
    onError = Color(0xFFFFFFFF),
    outline = Color(0xFF38383A),
    surfaceVariant = Color(0xFF2C2C2E),
    onSurfaceVariant = Color(0xFF8E8E93)
)

/**
 * 主题管理器
 * 使用 multiplatform-settings 实现跨平台主题持久化
 * 
 * 注意：经过可行性验证，使用基础API而非序列化扩展函数更稳定
 */
class ThemeManager(private val settings: Settings) {
    private val flowSettings: FlowSettings = (settings as ObservableSettings).toFlowSettings()
    private val json = Json { ignoreUnknownKeys = true }
    
    /**
     * 当前主题配置流
     * 使用Flow实现实时响应主题变更
     */
    val themeConfig: Flow<ThemeConfig> = flowSettings
        .getStringOrNullFlow(KEY_THEME_CONFIG)
        .map { jsonString ->
            jsonString?.let {
                try {
                    json.decodeFromString<ThemeConfig>(it)
                } catch (e: Exception) {
                    ThemeConfig()
                }
            } ?: ThemeConfig()
        }
    
    /**
     * 获取当前有效的颜色方案
     * 根据主题模式和系统主题返回对应的颜色方案
     */
    @Composable
    fun getColorScheme(config: ThemeConfig): androidx.compose.material3.ColorScheme {
        val isSystemDark = isSystemInDarkTheme()
        return when (config.mode) {
            ThemeMode.LIGHT -> LightColors
            ThemeMode.DARK -> DarkColors
            ThemeMode.AUTO -> if (isSystemDark) DarkColors else LightColors
        }
    }
    
    /**
     * 切换主题模式
     * 保存到Settings并自动通知所有订阅者
     */
    suspend fun setThemeMode(mode: ThemeMode) {
        val currentJson = settings.getStringOrNull(KEY_THEME_CONFIG)
        val current = currentJson?.let {
            try {
                json.decodeFromString<ThemeConfig>(it)
            } catch (e: Exception) {
                ThemeConfig()
            }
        } ?: ThemeConfig()
        
        val newConfig = current.copy(mode = mode)
        settings.putString(
            KEY_THEME_CONFIG, 
            json.encodeToString(ThemeConfig.serializer(), newConfig)
        )
    }
    
    companion object {
        private const val KEY_THEME_CONFIG = "theme_config"
    }
}

/**
 * 应用主题
 * 使用示例：
 * ```
 * AppTheme(themeManager) {
 *     // 应用内容
 * }
 * ```
 */
@Composable
fun AppTheme(
    themeManager: ThemeManager,
    content: @Composable () -> Unit
) {
    val config = themeManager.themeConfig.collectAsState(ThemeConfig()).value
    val colorScheme = themeManager.getColorScheme(config)
    
    MaterialTheme(
        colorScheme = colorScheme,
        content = content
    )
}
```

#### 平台特定实现

**Desktop平台** (使用Java Preferences)：
```kotlin
// desktopMain/kotlin/main.kt
import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application
import com.russhwolf.settings.PreferencesSettings
import java.util.prefs.Preferences

fun main() = application {
    val preferences = Preferences.userRoot().node("com.example.app")
    val settings = PreferencesSettings(preferences)
    
    Window(onCloseRequest = ::exitApplication) {
        App(settings)
    }
}
```

**Web平台** (使用localStorage)：
```kotlin
// wasmJsMain/kotlin/main.kt
import androidx.compose.ui.ExperimentalComposeUiApi
import androidx.compose.ui.window.CanvasBasedWindow
import com.russhwolf.settings.StorageSettings
import kotlinx.browser.localStorage
import org.jetbrains.skiko.wasm.onWasmReady

@OptIn(ExperimentalComposeUiApi::class)
fun main() {
    onWasmReady {
        val settings = StorageSettings(localStorage)
        
        CanvasBasedWindow(canvasElementId = "ComposeTarget") {
            App(settings)
        }
    }
}
```
```

### 9.3 图标系统

#### 图标设计规范

**风格**：填充多色图标（Filled Multi-color Icons）
- 使用填充形状而非描边
- 多色设计增加视觉丰富度
- 保持Apple HIG的清晰度和易识别性
- 圆角处理，与整体设计语言一致

**示例图标特征**：
- 文件夹图标：蓝色填充主体 + 黄色标签
- 消息图标：蓝色气泡 + 白色文字区域
- AI机器人：紫色头部 + 蓝色身体 + 橙色细节

#### SVG图标策略

**优势**：
- 矢量格式，任意缩放不失真
- 文件体积小，适合网络传输
- 支持透明通道
- 跨平台通用，一套图标全平台使用
- 多色支持，视觉表现更丰富

**技术实现**：
- 使用Jetpack Compose的`VectorPainter`加载SVG
- 图标资源统一存放在`resources/icons/`目录
- 按功能分类组织（navigation, action, content, file等）
- 每个SVG保留原始多色设计，不使用单一色调

#### 图标资源结构

```
resources/
└── icons/
    ├── svg/                  # SVG源文件
    │   ├── navigation/
    │   │   ├── home.svg
    │   │   ├── settings.svg
    │   │   ├── back.svg
    │   │   └── menu.svg
    │   ├── action/
    │   │   ├── add.svg
    │   │   ├── delete.svg
    │   │   ├── edit.svg
    │   │   ├── search.svg
    │   │   └── refresh.svg
    │   ├── content/
    │   │   ├── message.svg
    │   │   ├── image.svg
    │   │   ├── file.svg
    │   │   └── folder.svg
    │   ├── status/
    │   │   ├── success.svg
    │   │   ├── error.svg
    │   │   ├── warning.svg
    │   │   └── info.svg
    │   └── ai/
    │       ├── robot.svg
    │       ├── brain.svg
    │       ├── chat.svg
    │       └── skill.svg
    └── png/                  # 预转换的PNG（多分辨率）
        ├── navigation/
        │   ├── home/
        │   │   ├── 16x16.png
        │   │   ├── 24x24.png
        │   │   ├── 32x32.png
        │   │   ├── 48x48.png
        │   │   ├── 64x64.png
        │   │   ├── 128x128.png
        │   │   ├── 256x256.png
        │   │   └── 512x512.png
        │   └── ...
        └── ...
```

#### 图标组件

```kotlin
/**
 * 多色SVG图标组件
 * 保留SVG原始颜色，不使用tint
 */
@Composable
fun MultiColorSvgIcon(
    name: String,
    modifier: Modifier = Modifier,
    size: Dp = 24.dp
) {
    val painter = rememberSvgPainter(
        resourcePath = "icons/svg/$name.svg"
    )
    
    Image(
        painter = painter,
        contentDescription = name,
        modifier = modifier.size(size)
        // 不使用colorFilter，保留SVG原始颜色
    )
}

/**
 * 预转换PNG图标（用于不支持SVG的平台）
 */
@Composable
fun PngIcon(
    name: String,
    modifier: Modifier = Modifier,
    size: Dp = 24.dp
) {
    // 根据尺寸选择最接近的PNG
    val sizePx = with(LocalDensity.current) { size.roundToPx() }
    val pngSize = selectOptimalPngSize(sizePx)
    
    val painter = rememberAsyncImagePainter(
        model = "icons/png/$name/${pngSize}x${pngSize}.png"
    )
    
    Image(
        painter = painter,
        contentDescription = name,
        modifier = modifier.size(size)
    )
}

/**
 * 自适应图标组件
 * 优先使用SVG，在不支持的平台使用PNG
 */
@Composable
fun AdaptiveIcon(
    name: String,
    modifier: Modifier = Modifier,
    size: Dp = 24.dp
) {
    if (LocalPlatform.current.supportsSvg) {
        MultiColorSvgIcon(name, modifier, size)
    } else {
        PngIcon(name, modifier, size)
    }
}

/**
 * 选择最优PNG尺寸
 */
private fun selectOptimalPngSize(requestedSize: Int): Int {
    val availableSizes = listOf(16, 24, 32, 48, 64, 128, 256, 512)
    return availableSizes.find { it >= requestedSize } 
        ?: availableSizes.last()
}
```

### 9.4 SVG转位图格式

#### 预转换策略

**核心原则**：每个SVG图标在资源提交时就预转换为多分辨率PNG，而非运行时转换。

**优势**：
- 构建时无需转换，加快构建速度
- 运行时直接读取PNG，无性能损耗
- 版本控制包含所有格式，确保一致性

**预转换规格**：
每个SVG图标预转换为以下尺寸的PNG（带透明通道）：
- 16x16 - 工具栏小图标
- 24x24 - 标准图标（默认）
- 32x32 - 工具栏大图标
- 48x48 - 列表/菜单图标
- 64x64 - 侧边栏图标
- 128x128 - 大图标展示
- 256x256 - 高分辨率显示
- 512x512 - 应用图标/启动图标

**目录结构**：
```
resources/icons/
├── svg/                    # SVG源文件（版本控制）
│   └── navigation/
│       └── home.svg
└── png/                    # 预转换PNG（版本控制）
    └── navigation/
        └── home/
            ├── 16x16.png
            ├── 24x24.png
            ├── 32x32.png
            ├── 48x48.png
            ├── 64x64.png
            ├── 128x128.png
            ├── 256x256.png
            └── 512x512.png
```

#### 转换需求场景

1. **Windows桌面应用图标**：需要ICO格式
2. **macOS应用图标**：需要ICNS格式
3. **Android应用图标**：需要PNG格式（不同密度）
4. **iOS应用图标**：需要PNG格式
5. **Web Favicon**：需要ICO或PNG格式
6. **系统托盘图标**：需要PNG格式

#### 转换工具设计

```kotlin
/**
 * SVG转换器
 * 将SVG矢量图转换为各种位图格式
 */
class SvgConverter {
    
    companion object {
        /**
         * SVG转PNG
         * @param svgPath SVG文件路径
         * @param outputPath 输出PNG路径
         * @param width 输出宽度
         * @param height 输出高度
         * @param backgroundColor 背景色（null表示透明）
         */
        suspend fun convertToPng(
            svgPath: Path,
            outputPath: Path,
            width: Int,
            height: Int,
            backgroundColor: Color? = null
        ): ConversionResult
        
        /**
         * SVG转ICO（Windows图标）
         * @param svgPath SVG文件路径
         * @param outputPath 输出ICO路径
         * @param sizes 包含的尺寸列表，默认 [16, 32, 48, 256]
         */
        suspend fun convertToIco(
            svgPath: Path,
            outputPath: Path,
            sizes: List<Int> = listOf(16, 32, 48, 256)
        ): ConversionResult
        
        /**
         * SVG转ICNS（macOS图标）
         * @param svgPath SVG文件路径
         * @param outputPath 输出ICNS路径
         */
        suspend fun convertToIcns(
            svgPath: Path,
            outputPath: Path
        ): ConversionResult
        
        /**
         * 批量生成Android图标
         * 生成不同密度的PNG图标
         */
        suspend fun generateAndroidIcons(
            svgPath: Path,
            outputDir: Path,
            baseSize: Int = 48
        ): ConversionResult
        
        /**
         * 批量生成iOS图标
         * 生成各种尺寸的PNG图标
         */
        suspend fun generateIosIcons(
            svgPath: Path,
            outputDir: Path
        ): ConversionResult
    }
}

sealed class ConversionResult {
    data class Success(val outputPath: Path) : ConversionResult()
    data class Error(val message: String) : ConversionResult()
}
```

#### 技术实现方案

**方案一：使用SVG Salamander（推荐用于JVM）**

```kotlin
// build.gradle.kts 依赖
dependencies {
    implementation("guru.nidi.com.kitfox:svg-salamander:1.1.3")
}
```

```kotlin
import com.kitfox.svg.SVGDiagram
import com.kitfox.svg.SVGUniverse
import java.awt.Graphics2D
import java.awt.RenderingHints
import java.awt.image.BufferedImage
import java.io.File
import javax.imageio.ImageIO

/**
 * 使用SVG Salamander进行转换
 */
class SvgSalamanderConverter {
    
    fun convertToPng(
        svgFile: File,
        outputFile: File,
        width: Int,
        height: Int
    ) {
        val universe = SVGUniverse()
        val uri = universe.loadSVG(svgFile.toURI().toURL())
        val diagram: SVGDiagram = universe.getDiagram(uri)
        
        // 创建BufferedImage
        val image = BufferedImage(width, height, BufferedImage.TYPE_INT_ARGB)
        val g2d: Graphics2D = image.createGraphics()
        
        // 设置高质量渲染
        g2d.setRenderingHint(
            RenderingHints.KEY_ANTIALIASING,
            RenderingHints.VALUE_ANTIALIAS_ON
        )
        g2d.setRenderingHint(
            RenderingHints.KEY_RENDERING,
            RenderingHints.VALUE_RENDER_QUALITY
        )
        
        // 计算缩放比例
        val scaleX = width / diagram.width
        val scaleY = height / diagram.height
        g2d.scale(scaleX.toDouble(), scaleY.toDouble())
        
        // 渲染SVG
        diagram.render(g2d)
        g2d.dispose()
        
        // 保存PNG
        ImageIO.write(image, "PNG", outputFile)
    }
}
```

**方案二：使用Apache Batik**

```kotlin
// build.gradle.kts 依赖
dependencies {
    implementation("org.apache.xmlgraphics:batik-transcoder:1.17")
    implementation("org.apache.xmlgraphics:batik-codec:1.17")
}
```

```kotlin
import org.apache.batik.transcoder.TranscoderInput
import org.apache.batik.transcoder.TranscoderOutput
import org.apache.batik.transcoder.image.PNGTranscoder
import java.io.File
import java.io.FileInputStream
import java.io.FileOutputStream

/**
 * 使用Apache Batik进行转换
 */
class BatikSvgConverter {
    
    fun convertToPng(
        svgFile: File,
        outputFile: File,
        width: Int,
        height: Int
    ) {
        val transcoder = PNGTranscoder()
        
        // 设置输出尺寸
        transcoder.addTranscodingHint(
            PNGTranscoder.KEY_WIDTH,
            width.toFloat()
        )
        transcoder.addTranscodingHint(
            PNGTranscoder.KEY_HEIGHT,
            height.toFloat()
        )
        
        // 执行转换
        val input = TranscoderInput(FileInputStream(svgFile))
        val output = TranscoderOutput(FileOutputStream(outputFile))
        
        transcoder.transcode(input, output)
    }
}
```

**方案三：Kotlin Native方案（使用librsvg）**

对于Kotlin Native目标，可以通过CInterop调用librsvg库：

```kotlin
// 适用于Native目标的转换
expect class NativeSvgConverter() {
    fun convertToPng(
        svgPath: String,
        outputPath: String,
        width: Int,
        height: Int
    )
}
```

#### Gradle预转换任务

在 `build.gradle.kts` 中添加预转换任务：

```kotlin
// build.gradle.kts
import java.io.File

plugins {
    kotlin("multiplatform")
    // ... 其他插件
}

// SVG预转换配置
val svgSourceDir = file("src/commonMain/resources/icons/svg")
val pngOutputDir = file("src/commonMain/resources/icons/png")
val pngSizes = listOf(16, 24, 32, 48, 64, 128, 256, 512)

/**
 * 预转换所有SVG图标任务
 */
tasks.register<JavaExec>("convertSvgIcons") {
    group = "resource processing"
    description = "预转换所有SVG图标为多分辨率PNG"
    
    classpath = configurations["runtimeClasspath"]
    mainClass.set("com.example.build.SvgIconConverter")
    
    args = listOf(
        svgSourceDir.absolutePath,
        pngOutputDir.absolutePath,
        pngSizes.joinToString(",")
    )
    
    // 增量构建支持
    inputs.dir(svgSourceDir)
    outputs.dir(pngOutputDir)
}

/**
 * 在资源处理前执行转换
 */
tasks.named("processResources") {
    dependsOn("convertSvgIcons")
}
```

#### 图标转换工具类（构建时使用）

```kotlin
// buildSrc/src/main/kotlin/com/example/build/SvgIconConverter.kt
package com.example.build

import org.apache.batik.transcoder.TranscoderInput
import org.apache.batik.transcoder.TranscoderOutput
import org.apache.batik.transcoder.image.PNGTranscoder
import java.io.File
import java.io.FileInputStream
import java.io.FileOutputStream

/**
 * SVG图标预转换工具
 * 在构建时将SVG转换为多分辨率PNG
 */
object SvgIconConverter {
    
    @JvmStatic
    fun main(args: Array<String>) {
        if (args.size < 3) {
            println("用法: SvgIconConverter <svg目录> <png输出目录> <尺寸列表>")
            return
        }
        
        val svgDir = File(args[0])
        val pngDir = File(args[1])
        val sizes = args[2].split(",").map { it.toInt() }
        
        if (!svgDir.exists()) {
            println("SVG目录不存在: ${svgDir.absolutePath}")
            return
        }
        
        println("开始转换SVG图标...")
        println("源目录: ${svgDir.absolutePath}")
        println("输出目录: ${pngDir.absolutePath}")
        println("尺寸: $sizes")
        
        convertAllSvgs(svgDir, pngDir, sizes)
        
        println("转换完成!")
    }
    
    private fun convertAllSvgs(svgDir: File, pngDir: File, sizes: List<Int>) {
        svgDir.walkTopDown()
            .filter { it.isFile && it.extension.lowercase() == "svg" }
            .forEach { svgFile ->
                val relativePath = svgFile.relativeTo(svgDir).parent ?: ""
                val iconName = svgFile.nameWithoutExtension
                
                sizes.forEach { size ->
                    val outputDir = File(pngDir, relativePath).resolve(iconName)
                    outputDir.mkdirs()
                    
                    val outputFile = File(outputDir, "${size}x${size}.png")
                    
                    // 检查是否需要更新（增量转换）
                    if (needsConversion(svgFile, outputFile)) {
                        convertSvgToPng(svgFile, outputFile, size, size)
                        println("✓ ${svgFile.name} -> ${size}x${size}.png")
                    }
                }
            }
    }
    
    private fun needsConversion(svgFile: File, pngFile: File): Boolean {
        if (!pngFile.exists()) return true
        return svgFile.lastModified() > pngFile.lastModified()
    }
    
    private fun convertSvgToPng(
        svgFile: File,
        outputFile: File,
        width: Int,
        height: Int
    ) {
        val transcoder = PNGTranscoder()
        
        transcoder.addTranscodingHint(
            PNGTranscoder.KEY_WIDTH,
            width.toFloat()
        )
        transcoder.addTranscodingHint(
            PNGTranscoder.KEY_HEIGHT,
            height.toFloat()
        )
        
        // 启用抗锯齿
        transcoder.addTranscodingHint(
            PNGTranscoder.KEY_ANTIALIASING,
            true
        )
        
        FileInputStream(svgFile).use { inputStream ->
            FileOutputStream(outputFile).use { outputStream ->
                val input = TranscoderInput(inputStream)
                val output = TranscoderOutput(outputStream)
                transcoder.transcode(input, output)
            }
        }
    }
}
```

#### .gitignore配置

```gitignore
# 不忽略预转换的PNG（纳入版本控制）
# resources/icons/png/

# 但忽略构建时生成的临时文件
build/
*.tmp
```

### 9.5 界面布局架构

#### 响应式布局

```kotlin
/**
 * 窗口尺寸类别
 */
enum class WindowSizeClass {
    COMPACT,    // 手机尺寸
    MEDIUM,     // 平板尺寸
    EXPANDED    // 桌面尺寸
}

/**
 * 根据窗口尺寸确定布局方式
 */
@Composable
fun AdaptiveLayout(
    windowSizeClass: WindowSizeClass,
    compactContent: @Composable () -> Unit,
    mediumContent: @Composable () -> Unit,
    expandedContent: @Composable () -> Unit
) {
    when (windowSizeClass) {
        WindowSizeClass.COMPACT -> compactContent()
        WindowSizeClass.MEDIUM -> mediumContent()
        WindowSizeClass.EXPANDED -> expandedContent()
    }
}
```

#### 主界面结构

```
┌─────────────────────────────────────────────────────────────┐
│  [Sidebar]  │              [Main Content]                    │
│             │                                                │
│  ┌─────────┐│  ┌──────────────────────────────────────────┐  │
│  │ AI角色1 ││  │                                          │  │
│  ├─────────┤│  │           会话区域                        │  │
│  │ AI角色2 ││  │                                          │  │
│  ├─────────┤│  │                                          │  │
│  │ AI角色3 ││  │                                          │  │
│  └─────────┘│  │                                          │  │
│             │  └──────────────────────────────────────────┘  │
│  ┌─────────┐│  ┌──────────────────────────────────────────┐  │
│  │ 群聊    ││  │  [输入框]                    [发送按钮]   │  │
│  └─────────┘│  └──────────────────────────────────────────┘  │
│             │                                                │
└─────────────────────────────────────────────────────────────┘
```

### 9.6 动画与过渡

```kotlin
/**
 * 主题切换动画
 */
@Composable
fun AnimatedThemeSwitch(
    isDark: Boolean,
    onToggle: () -> Unit
) {
    val rotation by animateFloatAsState(
        targetValue = if (isDark) 180f else 0f,
        animationSpec = tween(durationMillis = 500)
    )
    
    IconButton(onClick = onToggle) {
        SvgIcon(
            name = if (isDark) "moon" else "sun",
            modifier = Modifier.rotate(rotation)
        )
    }
}

/**
 * 页面切换过渡
 */
@Composable
fun PageTransition(
    targetState: Int,
    content: @Composable (Int) -> Unit
) {
    AnimatedContent(
        targetState = targetState,
        transitionSpec = {
            fadeIn(animationSpec = tween(300)) +
            slideInHorizontally { it } with
            fadeOut(animationSpec = tween(300)) +
            slideOutHorizontally { -it }
        }
    ) { page ->
        content(page)
    }
}
```

### 9.7 可访问性支持

```kotlin
/**
 * 可访问性配置
 */
@Composable
fun AccessibleIconButton(
    onClick: () -> Unit,
    iconName: String,
    contentDescription: String
) {
    IconButton(
        onClick = onClick,
        modifier = Modifier.semantics {
            this.contentDescription = contentDescription
        }
    ) {
        SvgIcon(name = iconName)
    }
}

/**
 * 高对比度模式支持
 */
@Composable
fun HighContrastAwareContent(
    content: @Composable () -> Unit
) {
    val isHighContrast = LocalAccessibilityHighContrast.current
    
    CompositionLocalProvider(
        LocalColorScheme provides if (isHighContrast) {
            HighContrastColorScheme
        } else {
            LocalColorScheme.current
        }
    ) {
        content()
    }
}
```

### 9.8 相关技能

- **svg-converter**：SVG 转 PNG/ICO 等位图格式的技能，详见 `.trae/skills/svg-converter/SKILL.md`

## 10. 类 OAuth2.0 认证系统设计

### 10.1 需求概述

用户希望客户端和服务端使用类似 OAuth2.0 的方式进行认证，采用方案 A（简化版 Token 交换机制）。

**核心需求**:
1. 服务端预设一个固定 Token，首次启动时生成，每天自动更换
2. 客户端第一次通过预设 Token 向服务端索取一个独属于自己的 Token（永不过期）
3. 服务端将客户端 Token 在文件中持久化，启动时加载
4. 支持管理已授权的客户端（查看、撤销）

### 10.2 认证流程

```
┌─────────────┐                    ┌─────────────┐                    ┌─────────────┐
│   服务端    │                    │    客户端    │                    │    用户     │
└──────┬──────┘                    └──────┬──────┘                    └──────┬──────┘
       │                                  │                                  │
       │ 1. 启动时生成预设 Token           │                                  │
       │    (每天更换)                    │                                  │
       │                                  │                                  │
       │                                  │ 2. 用户输入预设 Token ───────────>│
       │                                  │                                  │
       │ 3. 使用预设 Token 请求专属 Token ─>│                                  │
       │                                  │                                  │
       │ 4. 验证预设 Token                │                                  │
       │    生成专属 Token                │                                  │
       │    保存到授权列表                │                                  │
       │                                  │                                  │
       │ 5. 返回专属 Token ───────────────>│                                  │
       │    (永不过期)                    │                                  │
       │                                  │                                  │
       │                                  │ 6. 持久化存储专属 Token           │
       │                                  │                                  │
       │ 7. 后续请求使用专属 Token ───────>│                                  │
       │                                  │                                  │
       │ 8. 验证专属 Token                │                                  │
       │    响应请求                      │                                  │
       │                                  │                                  │
```

### 10.3 技术架构

#### 10.3.1 服务端 Token 管理器

**职责**:
- 生成和管理预设 Token（每日更换）
- 管理已授权的客户端 Token 列表
- 验证预设 Token 和专属 Token
- 支持 Token 撤销

**数据结构**:
```kotlin
data class TokenData(
    val token: String,           // Token 字符串
    val clientId: String,        // 客户端 ID
    val createdAt: String,       // 创建时间戳
    val expiresAt: String? = null // 过期时间（null 表示永不过期）
)

class ServerTokenManager {
    var presetToken: String              // 当前预设 Token
    var presetTokenDate: LocalDate       // 预设 Token 生成日期
    
    fun initialize()                     // 初始化：加载/生成预设 Token
    fun verifyPresetToken(token: String): Boolean
    fun verifyClientToken(token: String): Boolean
    fun exchangePresetToken(presetToken: String, clientId: String?): TokenData?
    fun revokeToken(token: String): Boolean
    fun getAuthorizedTokens(): List<Map<String, String>>
}
```

#### 10.3.2 客户端 Token 管理器

**职责**:
- 管理预设 Token（用户输入）
- 管理专属 Token（从服务端获取）
- 持久化存储 Token
- Token 丢失后重新获取

**数据结构**:
```kotlin
class ClientTokenManager {
    var presetToken: String?      // 预设 Token（用户输入）
    var clientToken: TokenData?   // 专属 Token
    var clientId: String          // 客户端 ID
    
    fun initialize()              // 初始化：加载已保存的 Token
    fun updatePresetToken(token: String)
    fun updateClientToken(token: TokenData)
    fun clearClientToken()        // 清除专属 Token
}
```

### 10.4 持久化存储

#### 10.4.1 服务端存储

**预设 Token 文件**: `server-config/preset_token.json`
```json
{
  "token": "4835392f...",
  "date": "2026-03-12",
  "createdAt": "1773282753904"
}
```

**已授权 Token 文件**: `server-config/authorized_tokens.json`
```json
[
  {
    "token": "226252a2...",
    "clientId": "client_3c90accb...",
    "addedAt": "1773282753904"
  }
]
```

#### 10.4.2 客户端存储

**Token 文件**: `client-token.json`
```json
{
  "presetToken": "4835392f...",
  "clientToken": "{\"token\":\"226252a2...\",\"clientId\":\"client_3c90accb...\",\"createdAt\":\"1773282753904\"}",
  "clientId": "client_3c90accb...",
  "lastUpdated": "1773282753904"
}
```

**客户端 ID 文件**: `client_id.txt`
```
client_3c90accb71ccc1ee95aa7e8e3bd3bc13
```

### 10.5 Token 生成算法

**预设 Token**:
- 长度：32 字节（64 字符十六进制）
- 生成器：`SecureRandom`
- 更换策略：每天自动更换

**专属 Token**:
- 长度：32 字节（64 字符十六进制）
- 生成器：`SecureRandom`
- 过期策略：永不过期

**客户端 ID**:
- 格式：`client_` + 16 字节随机数（32 字符十六进制）
- 生成器：`SecureRandom`
- 持久化：每个设备独立保存

### 10.6 认证中间件（Ktor 集成）

```kotlin
fun Application.configureAuthentication() {
    install(Authentication) {
        bearer("auth") {
            realm = "Access"
            authenticate { token ->
                val tokenManager = application.tokenManager
                if (tokenManager.verifyClientToken(token)) {
                    UserIdPrincipal(token)
                } else {
                    null
                }
            }
        }
    }
}

// 使用示例
authenticate("auth") {
    get("/api/protected") {
        call.respondText("受保护的资源")
    }
}
```

### 10.7 管理 API

```kotlin
// 获取所有已授权的客户端
get("/api/admin/authorized-clients") {
    val tokens = tokenManager.getAuthorizedTokens()
    call.respond(tokens)
}

// 撤销某个客户端的 Token
post("/api/admin/revoke-token") {
    val request = call.receive<TokenRevokeRequest>()
    val success = tokenManager.revokeToken(request.token)
    call.respond(mapOf("success" to success))
}

// 获取当前预设 Token（用于管理界面）
get("/api/admin/preset-token") {
    call.respond(mapOf("token" to tokenManager.presetToken))
}
```

### 10.8 安全特性

1. **Token 安全性**: 使用 `SecureRandom` 生成，不可预测
2. **Token 隔离**: 每个客户端有独立的专属 Token
3. **Token 管理**: 支持查看和撤销已授权的 Token
4. **每日更换**: 预设 Token 每天自动更换，降低泄露风险
5. **持久化存储**: JSON 格式，易于管理和审计

### 10.9 安全建议（生产环境）

1. **传输加密**: 必须使用 HTTPS 传输 Token
2. **存储加密**: 客户端 Token 存储应加密
3. **访问日志**: 记录所有认证请求，便于审计
4. **速率限制**: 防止暴力破解预设 Token
5. **IP 白名单**: 可选，限制特定 IP 访问

### 10.10 可行性测试

**测试位置**: `test-available/auth-system/`

**测试结果**: ✅ 全部通过 (13/13, 100%)

**测试覆盖**:
- ✅ 预设 Token 生成和每日更换
- ✅ 专属 Token 生成和验证
- ✅ Token 持久化和加载
- ✅ 预设 Token 验证
- ✅ 专属 Token 换取
- ✅ 多客户端支持
- ✅ Token 撤销
- ✅ Token 丢失后重新获取

**详细报告**: [`test-available/auth-system/verification-report.md`](../test-available/auth-system/verification-report.md)

### 10.11 实施计划

1. **Phase 1: 核心实现**
   - [ ] 实现 `ServerTokenManager`
   - [ ] 实现 `ClientTokenManager`
   - [ ] 实现 Token 持久化

2. **Phase 2: Ktor 集成**
   - [ ] 实现认证中间件
   - [ ] 实现管理 API
   - [ ] 添加错误处理

3. **Phase 3: 客户端集成**
   - [ ] 为各平台客户端提供 Token 管理
   - [ ] 实现认证 UI 流程
   - [ ] 添加 Token 自动刷新

4. **Phase 4: 安全增强**
   - [ ] Token 加密存储
   - [ ] HTTPS 强制
   - [ ] 访问日志
   - [ ] 速率限制

### 10.12 验收标准

- [ ] 预设 Token 每日自动更换
- [ ] 专属 Token 永不过期
- [ ] Token 持久化可靠
- [ ] 多客户端支持正常
- [ ] Token 撤销功能正常
- [ ] 认证流程安全可靠
- [ ] 所有测试通过

## 11. AI 角色 TODO 列表系统

### 11.1 需求来源

详见 [`docs/user_say.md`](../user_say.md#2026-03-12-ai-角色-todo-列表功能需求)

### 11.2 系统概述

TODO 列表系统为 AI 角色提供任务规划和跟踪能力，支持 AI 和用户双向交互，使用 mem0 记忆系统存储，支持多列表并发和优先级管理。

### 11.3 核心特性

1. **存储方式**: 使用 mem0 记忆系统存储（选项 A）
2. **双向交互**: 用户和 AI 都可以创建、修改、操作 TODO 列表
3. **多列表并发**: 支持多个 TODO 列表同时存在
4. **优先级规则**: 按创建时间排列，最后创建的为最高优先级
5. **状态管理**: 支持待执行、进行中、已完成、已取消、阻塞中五种状态
6. **语义化 ID**: 使用语义化命名（如 "task-20260312-001"）

### 11.4 数据模型

#### TODO 列表数据类

```kotlin
/**
 * TODO 列表数据类
 */
data class TodoList(
    val id: String,                    // TODO 列表 ID（语义化命名，如 "task-20260312-001"）
    val title: String,                 // 总标题
    val roleId: String,                // AI 角色 ID
    val sessionId: String,             // 所属会话 ID
    val items: List<TodoItem>,         // 事项列表
    val status: TodoListStatus,        // 列表状态
    val createdAt: Instant,            // 创建时间
    val updatedAt: Instant,            // 最后更新时间
    val parentListId: String? = null   // 父列表 ID（用于恢复上级内容）
)

/**
 * TODO 事项数据类
 */
data class TodoItem(
    val id: String,                    // 事项 ID
    val title: String,                 // 事项标题
    val description: String? = null,   // 事项描述
    val status: TodoItemStatus,        // 事项状态
    val priority: Int = 0,             // 优先级（数字越大优先级越高）
    val createdAt: Instant,            // 创建时间
    val updatedAt: Instant,            // 最后更新时间
    val completedAt: Instant? = null,  // 完成时间
    val blockedReason: String? = null  // 阻塞原因（当状态为 Blocked 时）
)

/**
 * TODO 列表状态
 */
enum class TodoListStatus {
    ACTIVE,         // 活跃中
    COMPLETED,      // 全部完成
    CANCELLED       // 已取消
}

/**
 * TODO 事项状态
 */
enum class TodoItemStatus {
    PENDING,        // 待执行
    IN_PROGRESS,    // 进行中
    COMPLETED,      // 已完成
    CANCELLED,      // 已取消
    BLOCKED         // 阻塞中
}
```

### 11.5 mem0 存储架构

#### 存储键设计

```kotlin
/**
 * TODO 列表存储管理器
 */
class TodoListStorageManager(
    private val mem0Client: Mem0Client
) {
    companion object {
        // mem0 用户 ID 命名空间
        const val USER_ID_PREFIX = "todo_"
        
        // 记忆类型元数据
        const val MEMORY_TYPE_LIST = "todo_list"
        const val MEMORY_TYPE_ITEM = "todo_item"
    }
    
    /**
     * 获取 mem0 用户 ID
     */
    fun getMem0UserId(roleId: String): String {
        return "${USER_ID_PREFIX}${roleId}"
    }
    
    /**
     * 保存 TODO 列表
     */
    suspend fun saveTodoList(todoList: TodoList) {
        val userId = getMem0UserId(todoList.roleId)
        
        mem0Client.addMemory(
            userId = userId,
            data = MemoryData(
                text = Json.encodeToString(todoList),
                metadata = mapOf(
                    "type" to MEMORY_TYPE_LIST,
                    "listId" to todoList.id,
                    "sessionId" to todoList.sessionId,
                    "status" to todoList.status.name,
                    "timestamp" to todoList.updatedAt.toString()
                )
            )
        )
    }
    
    /**
     * 获取 TODO 列表
     */
    suspend fun getTodoList(roleId: String, listId: String): TodoList? {
        val userId = getMem0UserId(roleId)
        val memories = mem0Client.searchMemories(
            userId = userId,
            query = "listId:$listId",
            limit = 10
        )
        
        return memories
            .filter { it.metadata["type"] == MEMORY_TYPE_LIST && it.metadata["listId"] == listId }
            .sortedByDescending { it.metadata["timestamp"] }
            .firstOrNull()
            ?.let { Json.decodeFromString<TodoList>(it.text) }
    }
    
    /**
     * 获取所有活跃的 TODO 列表（按创建时间倒序排列）
     */
    suspend fun getActiveTodoLists(roleId: String, sessionId: String): List<TodoList> {
        val userId = getMem0UserId(roleId)
        val memories = mem0Client.getAllMemories(userId = userId, limit = 100)
        
        return memories
            .filter { 
                it.metadata["type"] == MEMORY_TYPE_LIST && 
                it.metadata["sessionId"] == sessionId &&
                it.metadata["status"] == TodoListStatus.ACTIVE.name
            }
            .sortedByDescending { it.metadata["timestamp"] }  // 最新的在前
            .mapNotNull { 
                try {
                    Json.decodeFromString<TodoList>(it.text)
                } catch (e: Exception) {
                    null
                }
            }
    }
    
    /**
     * 删除 TODO 列表（标记为已取消）
     */
    suspend fun cancelTodoList(roleId: String, listId: String) {
        val todoList = getTodoList(roleId, listId) ?: return
        
        val cancelledList = todoList.copy(
            status = TodoListStatus.CANCELLED,
            updatedAt = Instant.now(),
            items = todoList.items.map { item ->
                if (item.status == TodoItemStatus.PENDING || item.status == TodoItemStatus.IN_PROGRESS) {
                    item.copy(status = TodoItemStatus.CANCELLED, updatedAt = Instant.now())
                } else {
                    item
                }
            }
        )
        
        saveTodoList(cancelledList)
    }
}
```

### 11.6 TODO 列表操作工具

#### 工具定义

```kotlin
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
 * 事项输入数据类
 */
data class TodoItemInput(
    val title: String,
    val description: String? = null,
    val priority: Int = 0
)

/**
 * 事项更新数据类
 */
data class TodoItemUpdates(
    val title: String? = null,
    val description: String? = null,
    val priority: Int? = null,
    val status: TodoItemStatus? = null,
    val blockedReason: String? = null
)
```

#### 工具实现

```kotlin
/**
 * TODO 列表工具实现
 */
class TodoListToolsImpl(
    private val storageManager: TodoListStorageManager,
    private val idGenerator: TodoListIdGenerator
) : TodoListTools {
    
    override suspend fun createTodoList(
        roleId: String,
        sessionId: String,
        title: String,
        items: List<TodoItemInput>,
        parentListId: String?
    ): TodoList {
        val now = Instant.now()
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
                    updatedAt = Instant.now()
                )
            } else {
                item
            }
        }
        
        val updatedList = todoList.copy(
            items = updatedItems,
            updatedAt = Instant.now()
        )
        
        storageManager.saveTodoList(updatedList)
        
        return updatedItems.first { it.id == itemId }
    }
    
    override suspend fun completeTodoItem(
        roleId: String,
        listId: String,
        itemId: String
    ): TodoItem {
        val now = Instant.now()
        
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
            }
        }
        
        return updatedItem
    }
    
    override suspend fun cancelTodoList(
        roleId: String,
        listId: String
    ): Boolean {
        val todoList = storageManager.getTodoList(roleId, listId) ?: return false
        
        storageManager.cancelTodoList(roleId, listId)
        return true
    }
    
    override suspend fun getCurrentActiveTodoList(
        roleId: String,
        sessionId: String
    ): TodoList? {
        val activeLists = storageManager.getActiveTodoLists(roleId, sessionId)
        return activeLists.firstOrNull()  // 返回最新创建的（优先级最高）
    }
    
    override suspend fun getAllTodoLists(
        roleId: String,
        sessionId: String,
        status: TodoListStatus?
    ): List<TodoList> {
        val allLists = storageManager.getActiveTodoLists(roleId, sessionId)
        return if (status != null) {
            allLists.filter { it.status == status }
        } else {
            allLists
        }
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
```

### 11.7 AI 提示词集成

#### 系统提示词模板

```kotlin
/**
 * TODO 列表系统提示词管理器
 */
class TodoListSystemPromptManager {
    
    /**
     * 生成包含 TODO 列表功能的系统提示词
     */
    fun generateSystemPrompt(basePrompt: String): String {
        return """
            $basePrompt
            
            === TODO 列表功能 ===
            
            你拥有 TODO 列表功能，可以帮助你规划和跟踪任务执行。
            
            **使用偏好**：
            - 当你开始计划做一些事情时，应该主动创建 TODO 列表
            - 使用提供的工具来管理 TODO 列表：创建、修改事项、标记完成、取消列表
            - TODO 列表包含总标题和多个事项
            - 每个事项都有状态：待执行、进行中、已完成、已取消、阻塞中
            
            **多列表管理**：
            - 你可以同时有多个 TODO 列表
            - 最后创建的列表优先级最高，你应该优先执行
            - 通过 TODO 列表 ID 来指定操作哪个列表
            
            **打断处理**：
            - 如果你正在处理 TODO 列表时被用户打断，先关注用户交流内容
            - 在事情结束时，询问用户是否应该继续处理 TODO 列表
            - 仅当用户主动提示继续时，才继续处理 TODO 列表
            
            **取消机制**：
            - 你可以自主决定取消某个 TODO 列表
            - 用户也可以下达指令让你取消列表
            - 取消后，如果有上一级的 TODO 列表，则继续上一级的内容
            
            **工具列表**：
            - createTodoList(title, items, parentListId?): 创建 TODO 列表
            - updateTodoItem(listId, itemId, updates): 修改事项
            - completeTodoItem(listId, itemId): 标记事项完成
            - cancelTodoList(listId): 取消 TODO 列表
            - getCurrentActiveTodoList(): 获取当前优先级最高的活跃列表
            - getAllTodoLists(): 获取所有 TODO 列表
            
            请负责任地使用这些工具，专注于帮助用户完成任务。
        """.trimIndent()
    }
}
```

### 11.8 用户交互设计

#### 用户可见的 TODO 列表 UI

```kotlin
/**
 * TODO 列表 UI 状态
 */
data class TodoListUiState(
    val lists: List<TodoListDisplay>,
    val currentListId: String?,
    val isLoading: Boolean,
    val error: String?
)

/**
 * TODO 列表显示数据类
 */
data class TodoListDisplay(
    val id: String,
    val title: String,
    val items: List<TodoItemDisplay>,
    val status: TodoListStatus,
    val createdAt: String,
    val isCurrentPriority: Boolean  // 是否是当前优先级最高的
)

/**
 * TODO 事项显示数据类
 */
data class TodoItemDisplay(
    val id: String,
    val title: String,
    val description: String?,
    val status: TodoItemStatus,
    val priority: Int,
    val completedAt: String?
)

/**
 * TODO 列表 UI 组件（Jetpack Compose）
 */
@Composable
fun TodoListPanel(
    viewModel: TodoListViewModel,
    modifier: Modifier = Modifier
) {
    val state by viewModel.state.collectAsState()
    
    Column(modifier = modifier) {
        // 标题
        Text(
            text = "TODO 列表",
            style = MaterialTheme.typography.titleLarge
        )
        
        // TODO 列表列表
        state.lists.forEach { list ->
            TodoListItem(
                list = list,
                isCurrentPriority = list.isCurrentPriority,
                onItemClick = { itemId -> viewModel.toggleItemStatus(itemId) },
                onCancelClick = { viewModel.cancelList(list.id) }
            )
        }
        
        // 创建新列表按钮
        Button(
            onClick = { viewModel.showCreateDialog() },
            enabled = !state.isLoading
        ) {
            Text("创建 TODO 列表")
        }
    }
}

@Composable
fun TodoListItem(
    list: TodoListDisplay,
    isCurrentPriority: Boolean,
    onItemClick: (String) -> Unit,
    onCancelClick: () -> Unit
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(8.dp),
        colors = if (isCurrentPriority) {
            CardDefaults.cardColors(
                containerColor = MaterialTheme.colorScheme.primaryContainer
            )
        } else {
            CardDefaults.cardColors()
        }
    ) {
        Column(
            modifier = Modifier.padding(16.dp)
        ) {
            // 列表标题和状态
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween
            ) {
                Text(
                    text = list.title,
                    style = MaterialTheme.typography.titleMedium
                )
                
                if (isCurrentPriority) {
                    Badge {
                        Text("当前优先")
                    }
                }
                
                IconButton(onClick = onCancelClick) {
                    Icon(
                        imageVector = Icons.Default.Close,
                        contentDescription = "取消列表"
                    )
                }
            }
            
            // 事项列表
            list.items.forEach { item ->
                TodoItemRow(
                    item = item,
                    onClick = { onItemClick(item.id) }
                )
            }
        }
    }
}

@Composable
fun TodoItemRow(
    item: TodoItemDisplay,
    onClick: () -> Unit
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp)
            .clickable(onClick = onClick),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // 状态图标
        Icon(
            imageVector = when (item.status) {
                TodoItemStatus.PENDING -> Icons.Default.RadioButtonUnchecked
                TodoItemStatus.IN_PROGRESS -> Icons.Default.PlayArrow
                TodoItemStatus.COMPLETED -> Icons.Default.CheckCircle
                TodoItemStatus.CANCELLED -> Icons.Default.Close
                TodoItemStatus.BLOCKED -> Icons.Default.Lock
            },
            contentDescription = item.status.name,
            tint = when (item.status) {
                TodoItemStatus.COMPLETED -> Color.Green
                TodoItemStatus.BLOCKED -> Color.Gray
                else -> Color.Unspecified
            }
        )
        
        Spacer(modifier = Modifier.width(8.dp))
        
        // 事项标题
        Text(
            text = item.title,
            style = MaterialTheme.typography.bodyMedium,
            textDecoration = if (item.status == TodoItemStatus.COMPLETED) {
                TextDecoration.LineThrough
            } else {
                TextDecoration.None
            }
        )
    }
}
```

### 11.9 打断和恢复机制

#### 打断处理流程

```kotlin
/**
 * TODO 列表打断管理器
 */
class TodoListInterruptionManager {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    
    // 正在处理的 TODO 事项
    private val processingItems = ConcurrentHashMap<String, String>()  // roleId -> itemId
    
    // 等待恢复的 TODO 列表
    private val pendingLists = ConcurrentHashMap<String, String>()  // roleId -> listId
    
    /**
     * 开始处理事项
     */
    fun startProcessingItem(roleId: String, listId: String, itemId: String) {
        processingItems[roleId] = itemId
        pendingLists[roleId] = listId
    }
    
    /**
     * 用户打断
     */
    suspend fun onUserInterruption(
        roleId: String,
        sessionId: String,
        sendMessage: suspend (Message) -> Unit
    ) {
        val currentListId = pendingLists[roleId]
        
        if (currentListId != null) {
            // 询问用户是否继续
            sendMessage(
                Message(
                    id = generateId(),
                    senderId = roleId,
                    senderType = SenderType.AI_ROLE,
                    sessionId = sessionId,
                    content = MessageContent.Text(
                        "我注意到我正在处理 TODO 列表中的任务。您希望我继续处理之前的 TODO 列表吗？"
                    ),
                    timestamp = Instant.now(),
                    status = MessageStatus.COMPLETE
                )
            )
        }
    }
    
    /**
     * 用户确认继续
     */
    fun userConfirmedContinue(roleId: String): String? {
        return pendingLists[roleId]
    }
    
    /**
     * 用户取消或完成交互
     */
    fun userCancelledOrFinished(roleId: String) {
        pendingLists.remove(roleId)
        processingItems.remove(roleId)
    }
    
    /**
     * 完成事项
     */
    fun completeItem(roleId: String) {
        processingItems.remove(roleId)
        // pendingLists 保留，因为可能还有其他事项
    }
}
```

### 11.10 可行性测试要求

#### 测试场景

1. **存储测试**
   - TODO 列表在 mem0 中的存储和检索
   - 多列表并发存储
   - 状态持久化验证

2. **优先级测试**
   - 多 TODO 列表按创建时间排序
   - 获取当前优先级最高的列表
   - 列表取消后的优先级切换

3. **交互测试**
   - AI 主动创建 TODO 列表
   - 用户创建 TODO 列表
   - 双向修改事项
   - 标记完成流程

4. **打断恢复测试**
   - AI 处理事项时用户打断
   - 用户确认继续处理
   - 用户取消处理

5. **取消机制测试**
   - AI 主动取消列表
   - 用户指令取消列表
   - 取消后恢复上级列表

#### 测试用例示例

```kotlin
class TodoListFeatureTest {
    @Test
    fun testCreateTodoListAndStoreInMem0() = runTest {
        // 测试创建 TODO 列表并存储到 mem0
    }
    
    @Test
    fun testMultipleTodoListsPriorityOrder() = runTest {
        // 测试多列表优先级排序
    }
    
    @Test
    fun testUserInterruptionAndResume() = runTest {
        // 测试用户打断和恢复
    }
    
    @Test
    fun testCancelTodoListAndResumeParent() = runTest {
        // 测试取消列表后恢复上级
    }
}
```

### 11.11 技术约束

1. **mem0 依赖**: 所有 TODO 列表数据存储在 mem0 中
2. **双向交互**: 支持用户和 AI 双向操作
3. **会话隔离**: 不同会话的 TODO 列表完全隔离
4. **Ktor 集成**: 通过 Ktor 提供 REST API
5. **协程异步**: 所有操作使用 Kotlin 协程异步处理

### 11.12 验收标准

1. ✅ AI 能够主动创建 TODO 列表
2. ✅ 支持多个 TODO 列表同时存在
3. ✅ 优先级按创建时间正确排序
4. ✅ 用户和 AI 都可以操作 TODO 列表
5. ✅ 打断机制正常工作
6. ✅ 取消机制正常工作
7. ✅ 所有数据正确存储到 mem0
8. ✅ UI 正确显示 TODO 列表状态

## 参考资料

- [Mem0 官方文档](https://docs.mem0.ai/)
- [LM Studio API 文档](https://lmstudio.ai/docs)
- [Ktor 官方文档](https://ktor.io/)
- [Kotlin 协程文档](https://kotlinlang.org/docs/coroutines-overview.html)
- [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/)
- [OpenClaw 项目参考](../references/openclaw/)
- [NanoBot 项目参考](../references/nanobot/)
- [Jetpack Compose 主题指南](https://developer.android.com/jetpack/compose/themes)
- [Material Design 3 颜色系统](https://m3.material.io/styles/color/overview)
- [OAuth2.0 规范](https://oauth.net/2/)
- [Ktor 认证文档](https://ktor.io/docs/authentication.html)
- [GTD 时间管理方法](https://gettingthingsdone.com/)
- [Kotlinx 序列化文档](https://kotlinlang.org/docs/serialization.html)
