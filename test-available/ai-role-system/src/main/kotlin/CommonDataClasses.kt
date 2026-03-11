package com.assistant.test

import kotlinx.serialization.*

/**
 * AI 角色配置
 */
@Serializable
data class AIRoleConfig(
    val id: String,
    val name: String,
    val description: String,
    val avatarPath: String?,
    val personality: PersonalityType,
    val modelId: String,
    val systemPrompt: String,
    val behaviorRules: List<String>,
    val skills: SkillConfig,
    val createdAt: Long,  // Unix 时间戳（毫秒）
    val updatedAt: Long   // Unix 时间戳（毫秒）
)

/**
 * 技能配置
 */
@Serializable
data class SkillConfig(
    val whitelist: List<String> = emptyList(),
    val blacklist: List<String> = emptyList(),
    val installedSkills: List<InstalledSkill> = emptyList()
)

/**
 * 已安装技能
 */
@Serializable
data class InstalledSkill(
    val id: String,
    val name: String,
    val version: String,
    val installedAt: Long  // Unix 时间戳（毫秒）
)

/**
 * 性格类型
 */
@Serializable
enum class PersonalityType {
    FRIENDLY,           // 友好型
    PROFESSIONAL,       // 专业型
    CREATIVE,           // 创意型
    ANALYTICAL,         // 分析型
    EMPATHETIC,         // 共情型
    HUMOROUS            // 幽默型
}

/**
 * 会话数据
 */
@Serializable
data class SessionData(
    val id: String,
    val roleId: String,
    val messages: List<Message>,
    val createdAt: Long,  // Unix 时间戳（毫秒）
    val updatedAt: Long   // Unix 时间戳（毫秒）
)

/**
 * 消息
 */
@Serializable
data class Message(
    val id: String,
    val role: String,
    val content: String,
    val timestamp: Long  // Unix 时间戳（毫秒）
)

/**
 * 疲劳值数据
 */
@Serializable
data class FatigueState(
    val roleId: String,
    var currentValue: Double = 0.0,
    var lastMessageTime: Long,  // Unix 时间戳（毫秒）
    var messageCountInWindow: Int = 0,
    val windowDurationMinutes: Long = 120,
    val maxMessagesInWindow: Int = 200,
    val decayRatePerMinute: Double = 0.5
)

/**
 * 疲劳等级
 */
@Serializable
enum class FatigueLevel {
    LOW,          // 低疲劳：积极回应
    MEDIUM,       // 中等疲劳：正常回应
    HIGH,         // 高疲劳：减少回应
    VERY_HIGH,    // 很高疲劳：简短回应，期望结束
    EXHAUSTED     // 精疲力竭：不再回应
}
