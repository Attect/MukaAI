package com.assistant.test.serialization

import kotlinx.serialization.*
import kotlinx.serialization.json.*
import kotlinx.serialization.descriptors.*
import kotlinx.serialization.encoding.*

/**
 * 基础类型测试
 * 用于验证基础类型的序列化和反序列化
 */
@Serializable
data class BasicTypes(
    val string: String,
    val int: Int,
    val long: Long,
    val double: Double,
    val boolean: Boolean
)

/**
 * 枚举类型测试
 * 用于验证枚举类型的序列化和反序列化
 */
@Serializable
enum class UserRole {
    ADMIN,
    USER,
    GUEST
}

/**
 * 密封类测试
 * 用于验证多态序列化
 */
@Serializable
sealed class Message {
    @Serializable
    @SerialName("text")
    data class TextMessage(
        val content: String,
        val timestamp: Long
    ) : Message()
    
    @Serializable
    @SerialName("image")
    data class ImageMessage(
        val url: String,
        val width: Int,
        val height: Int,
        val timestamp: Long
    ) : Message()
    
    @Serializable
    @SerialName("tool_call")
    data class ToolCallMessage(
        val toolName: String,
        val parameters: Map<String, String>,
        val timestamp: Long
    ) : Message()
}

/**
 * 嵌套对象测试
 * 用于验证复杂嵌套结构的序列化
 */
@Serializable
data class Address(
    val country: String,
    val province: String,
    val city: String,
    val street: String? = null,
    val zipCode: String? = null
)

@Serializable
data class UserProfile(
    val id: String,
    val name: String,
    val age: Int,
    val email: String? = null,
    val address: Address? = null,
    val roles: List<UserRole> = emptyList(),
    val createdAt: Long,
    val updatedAt: Long
)

/**
 * 集合类型测试
 * 用于验证各种集合类型的序列化
 */
@Serializable
data class CollectionTypes(
    val stringList: List<String> = emptyList(),
    val intSet: Set<Int> = emptySet(),
    val stringMap: Map<String, String> = emptyMap(),
    val nestedList: List<List<Int>> = emptyList(),
    val nestedMap: Map<String, Map<String, Int>> = emptyMap(),
    val mixedList: List<String> = emptyList()
)

/**
 * 默认值测试
 * 用于验证默认值的序列化和反序列化
 */
@Serializable
data class DefaultValues(
    val requiredField: String,
    val optionalString: String = "default_value",
    val optionalInt: Int = 42,
    val optionalLong: Long = 1234567890L,
    val optionalDouble: Double = 3.14159,
    val optionalBoolean: Boolean = true,
    val optionalList: List<String> = listOf("default1", "default2"),
    val optionalMap: Map<String, Int> = mapOf("key1" to 1, "key2" to 2)
)

/**
 * 可空类型测试
 * 用于验证可空类型的序列化
 */
@Serializable
data class NullableTypes(
    val nullableString: String?,
    val nullableInt: Int?,
    val nullableLong: Long?,
    val nullableDouble: Double?,
    val nullableBoolean: Boolean?,
    val nullableList: List<String>?,
    val nullableMap: Map<String, Int>?
)

/**
 * AI 角色配置（实际业务场景）
 * 使用 Unix 时间戳
 */
@Serializable
data class AIRoleConfig(
    val id: String,
    val name: String,
    val description: String = "",
    val systemPrompt: String,
    val modelId: String,
    val temperature: Double = 0.7,
    val maxTokens: Int = 4096,
    val skills: List<String> = emptyList(),
    val skillWhitelist: List<String> = emptyList(),
    val skillBlacklist: List<String> = emptyList(),
    val createdAt: Long,
    val updatedAt: Long,
    val isActive: Boolean = true
)

/**
 * 疲劳状态（实际业务场景）
 * 使用 Unix 时间戳
 */
@Serializable
data class FatigueState(
    val roleId: String,
    var currentValue: Double = 0.0,
    var lastMessageTime: Long,
    var messageCountInWindow: Int = 0,
    val windowDurationMinutes: Long = 120,
    val maxMessagesInWindow: Int = 200,
    val decayRatePerMinute: Double = 0.5
)

/**
 * 会话消息（实际业务场景）
 * 使用 Unix 时间戳
 */
@Serializable
data class SessionMessage(
    val id: String,
    val sessionId: String,
    val roleId: String,
    val content: Message,
    val timestamp: Long,
    val metadata: Map<String, String> = emptyMap()
)

/**
 * 会话状态（实际业务场景）
 * 使用 Unix 时间戳
 */
@Serializable
data class SessionState(
    val sessionId: String,
    val roleIds: List<String>,
    val messages: List<SessionMessage> = emptyList(),
    val createdAt: Long,
    val updatedAt: Long,
    val isActive: Boolean = true
)

/**
 * 自定义序列化器测试
 * 用于验证自定义序列化器的使用
 */
@Serializable(with = CustomDataSerializer::class)
data class CustomData(
    val value: String,
    val timestamp: Long
)

object CustomDataSerializer : KSerializer<CustomData> {
    override val descriptor: SerialDescriptor = buildClassSerialDescriptor("CustomData") {
        element<String>("value")
        element<Long>("timestamp")
    }
    
    override fun serialize(encoder: Encoder, value: CustomData) {
        val composite = encoder.beginStructure(descriptor)
        composite.encodeStringElement(descriptor, 0, value.value.uppercase())
        composite.encodeLongElement(descriptor, 1, value.timestamp)
        composite.endStructure(descriptor)
    }
    
    override fun deserialize(decoder: Decoder): CustomData {
        val composite = decoder.beginStructure(descriptor)
        var value = ""
        var timestamp = 0L
        while (true) {
            when (val index = composite.decodeElementIndex(descriptor)) {
                CompositeDecoder.DECODE_DONE -> break
                0 -> value = composite.decodeStringElement(descriptor, 0).lowercase()
                1 -> timestamp = composite.decodeLongElement(descriptor, 1)
            }
        }
        composite.endStructure(descriptor)
        return CustomData(value, timestamp)
    }
}

/**
 * 复杂嵌套业务对象
 * 综合测试各种序列化场景
 */
@Serializable
data class ComplexBusinessObject(
    val id: String,
    val version: Int,
    val config: AIRoleConfig,
    val fatigue: FatigueState,
    val sessions: List<SessionState> = emptyList(),
    val metadata: Map<String, String> = emptyMap(),
    val createdAt: Long,
    val updatedAt: Long
)

/**
 * 简单配置测试类
 * 用于测试 JSON 配置选项
 */
@Serializable
data class SimpleConfig(
    val name: String
)
