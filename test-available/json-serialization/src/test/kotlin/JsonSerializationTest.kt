package com.assistant.test.serialization

import kotlinx.serialization.*
import kotlinx.serialization.json.*
import kotlinx.serialization.encodeToString
import kotlinx.serialization.decodeFromString
import org.junit.jupiter.api.*
import org.junit.jupiter.api.Assertions.*
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * JSON 序列化可行性测试
 * 
 * 测试目标：
 * 1. 验证基础类型序列化
 * 2. 验证可空类型序列化
 * 3. 验证集合类型序列化
 * 4. 验证嵌套对象序列化
 * 5. 验证枚举类型序列化
 * 6. 验证密封类序列化
 * 7. 验证默认值处理
 * 8. 验证自定义序列化器
 * 9. 验证复杂业务对象序列化
 * 10. 验证序列化配置选项
 */
class JsonSerializationTest {
    
    private val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
        encodeDefaults = true
        prettyPrint = true
    }
    
    private val compactJson = Json {
        ignoreUnknownKeys = true
        isLenient = true
        encodeDefaults = false
        prettyPrint = false
    }
    
    /**
     * 测试 1: 基础类型序列化
     * 验证 String, Int, Long, Double, Boolean 等基础类型
     */
    @Test
    fun testBasicTypes() {
        println("\n=== 测试 1: 基础类型序列化 ===")
        
        val data = BasicTypes(
            string = "Hello, 世界!",
            int = 42,
            long = 1234567890123456789L,
            double = 3.141592653589793,
            boolean = true
        )
        
        val jsonString = json.encodeToString(data)
        println("序列化结果:\n$jsonString")
        
        val decoded = json.decodeFromString<BasicTypes>(jsonString)
        println("反序列化结果: $decoded")
        
        assertEquals("Hello, 世界!", decoded.string)
        assertEquals(42, decoded.int)
        assertEquals(1234567890123456789L, decoded.long)
        assertEquals(3.141592653589793, decoded.double, 0.0000000001)
        assertEquals(true, decoded.boolean)
        
        println("✓ 基础类型序列化测试通过")
    }
    
    /**
     * 测试 2: 可空类型序列化
     * 验证可空字段的序列化和反序列化
     */
    @Test
    fun testNullableTypes() {
        println("\n=== 测试 2: 可空类型序列化 ===")
        
        val withNulls = NullableTypes(
            nullableString = null,
            nullableInt = null,
            nullableLong = null,
            nullableDouble = null,
            nullableBoolean = null,
            nullableList = null,
            nullableMap = null
        )
        
        val jsonString = json.encodeToString(withNulls)
        println("全 null 序列化结果:\n$jsonString")
        
        val decoded = json.decodeFromString<NullableTypes>(jsonString)
        assertNull(decoded.nullableString)
        assertNull(decoded.nullableInt)
        assertNull(decoded.nullableLong)
        assertNull(decoded.nullableDouble)
        assertNull(decoded.nullableBoolean)
        assertNull(decoded.nullableList)
        assertNull(decoded.nullableMap)
        
        val withValues = NullableTypes(
            nullableString = "test",
            nullableInt = 123,
            nullableLong = 9876543210L,
            nullableDouble = 2.71828,
            nullableBoolean = false,
            nullableList = listOf("a", "b", "c"),
            nullableMap = mapOf("key1" to 1, "key2" to 2)
        )
        
        val jsonStringWithValues = json.encodeToString(withValues)
        println("有值序列化结果:\n$jsonStringWithValues")
        
        val decodedWithValues = json.decodeFromString<NullableTypes>(jsonStringWithValues)
        assertEquals("test", decodedWithValues.nullableString)
        assertEquals(123, decodedWithValues.nullableInt)
        assertEquals(9876543210L, decodedWithValues.nullableLong)
        assertNotNull(decodedWithValues.nullableDouble)
        assertEquals(2.71828, decodedWithValues.nullableDouble!!, 0.00001)
        assertEquals(false, decodedWithValues.nullableBoolean)
        assertEquals(listOf("a", "b", "c"), decodedWithValues.nullableList)
        assertEquals(mapOf("key1" to 1, "key2" to 2), decodedWithValues.nullableMap)
        
        println("✓ 可空类型序列化测试通过")
    }
    
    /**
     * 测试 3: 集合类型序列化
     * 验证 List, Set, Map 等集合类型
     */
    @Test
    fun testCollectionTypes() {
        println("\n=== 测试 3: 集合类型序列化 ===")
        
        val data = CollectionTypes(
            stringList = listOf("apple", "banana", "cherry"),
            intSet = setOf(1, 2, 3, 4, 5),
            stringMap = mapOf("name" to "Alice", "age" to "30"),
            nestedList = listOf(listOf(1, 2), listOf(3, 4), listOf(5, 6)),
            nestedMap = mapOf(
                "group1" to mapOf("member1" to 100, "member2" to 200),
                "group2" to mapOf("member3" to 300, "member4" to 400)
            ),
            mixedList = listOf("string", "values")
        )
        
        val jsonString = json.encodeToString(data)
        println("序列化结果:\n$jsonString")
        
        val decoded = json.decodeFromString<CollectionTypes>(jsonString)
        assertEquals(listOf("apple", "banana", "cherry"), decoded.stringList)
        assertEquals(setOf(1, 2, 3, 4, 5), decoded.intSet)
        assertEquals(mapOf("name" to "Alice", "age" to "30"), decoded.stringMap)
        assertEquals(3, decoded.nestedList.size)
        assertEquals(2, decoded.nestedMap.size)
        
        println("✓ 集合类型序列化测试通过")
    }
    
    /**
     * 测试 4: 嵌套对象序列化
     * 验证复杂嵌套结构的序列化
     */
    @Test
    fun testNestedObjects() {
        println("\n=== 测试 4: 嵌套对象序列化 ===")
        
        val profile = UserProfile(
            id = "user-001",
            name = "张三",
            age = 28,
            email = "zhangsan@example.com",
            address = Address(
                country = "中国",
                province = "北京",
                city = "北京市",
                street = "朝阳区某某路123号",
                zipCode = "100000"
            ),
            roles = listOf(UserRole.ADMIN, UserRole.USER),
            createdAt = System.currentTimeMillis(),
            updatedAt = System.currentTimeMillis()
        )
        
        val jsonString = json.encodeToString(profile)
        println("序列化结果:\n$jsonString")
        
        val decoded = json.decodeFromString<UserProfile>(jsonString)
        assertEquals("user-001", decoded.id)
        assertEquals("张三", decoded.name)
        assertEquals(28, decoded.age)
        assertNotNull(decoded.address)
        assertEquals("中国", decoded.address!!.country)
        assertEquals("北京", decoded.address.province)
        assertEquals(listOf(UserRole.ADMIN, UserRole.USER), decoded.roles)
        
        println("✓ 嵌套对象序列化测试通过")
    }
    
    /**
     * 测试 5: 枚举类型序列化
     * 验证枚举类型的序列化和反序列化
     */
    @Test
    fun testEnumSerialization() {
        println("\n=== 测试 5: 枚举类型序列化 ===")
        
        val roles = listOf(UserRole.ADMIN, UserRole.USER, UserRole.GUEST)
        
        val jsonString = json.encodeToString(roles)
        println("序列化结果: $jsonString")
        
        val decoded = json.decodeFromString<List<UserRole>>(jsonString)
        assertEquals(roles, decoded)
        
        // 测试单个枚举值
        val adminJson = json.encodeToString(UserRole.ADMIN)
        println("单个枚举序列化: $adminJson")
        assertEquals(UserRole.ADMIN, json.decodeFromString<UserRole>(adminJson))
        
        println("✓ 枚举类型序列化测试通过")
    }
    
    /**
     * 测试 6: 密封类多态序列化
     * 验证密封类的多态序列化
     */
    @Test
    fun testSealedClassSerialization() {
        println("\n=== 测试 6: 密封类多态序列化 ===")
        
        val messages: List<Message> = listOf(
            Message.TextMessage(
                content = "你好，世界！",
                timestamp = System.currentTimeMillis()
            ),
            Message.ImageMessage(
                url = "https://example.com/image.png",
                width = 1920,
                height = 1080,
                timestamp = System.currentTimeMillis()
            ),
            Message.ToolCallMessage(
                toolName = "weather_api",
                parameters = mapOf("city" to "Beijing", "unit" to "celsius"),
                timestamp = System.currentTimeMillis()
            )
        )
        
        val jsonString = json.encodeToString(messages)
        println("序列化结果:\n$jsonString")
        
        val decoded = json.decodeFromString<List<Message>>(jsonString)
        assertEquals(3, decoded.size)
        
        val textMsg = decoded[0] as Message.TextMessage
        assertEquals("你好，世界！", textMsg.content)
        
        val imageMsg = decoded[1] as Message.ImageMessage
        assertEquals("https://example.com/image.png", imageMsg.url)
        assertEquals(1920, imageMsg.width)
        
        val toolMsg = decoded[2] as Message.ToolCallMessage
        assertEquals("weather_api", toolMsg.toolName)
        assertEquals(mapOf("city" to "Beijing", "unit" to "celsius"), toolMsg.parameters)
        
        println("✓ 密封类多态序列化测试通过")
    }
    
    /**
     * 测试 7: 默认值处理
     * 验证默认值的序列化和反序列化
     */
    @Test
    fun testDefaultValues() {
        println("\n=== 测试 7: 默认值处理 ===")
        
        // 只提供必需字段
        val minimal = DefaultValues(requiredField = "test")
        
        val minimalJson = compactJson.encodeToString(minimal)
        println("最小化 JSON (encodeDefaults=false): $minimalJson")
        
        val decodedMinimal = json.decodeFromString<DefaultValues>(minimalJson)
        assertEquals("test", decodedMinimal.requiredField)
        assertEquals("default_value", decodedMinimal.optionalString)
        assertEquals(42, decodedMinimal.optionalInt)
        assertEquals(1234567890L, decodedMinimal.optionalLong)
        assertEquals(3.14159, decodedMinimal.optionalDouble, 0.00001)
        assertEquals(true, decodedMinimal.optionalBoolean)
        assertEquals(listOf("default1", "default2"), decodedMinimal.optionalList)
        assertEquals(mapOf("key1" to 1, "key2" to 2), decodedMinimal.optionalMap)
        
        // 提供所有字段
        val full = DefaultValues(
            requiredField = "test",
            optionalString = "custom_value",
            optionalInt = 100,
            optionalLong = 9999999999L,
            optionalDouble = 2.71828,
            optionalBoolean = false,
            optionalList = listOf("custom1", "custom2", "custom3"),
            optionalMap = mapOf("customKey" to 999)
        )
        
        val fullJson = json.encodeToString(full)
        println("完整 JSON:\n$fullJson")
        
        val decodedFull = json.decodeFromString<DefaultValues>(fullJson)
        assertEquals("custom_value", decodedFull.optionalString)
        assertEquals(100, decodedFull.optionalInt)
        assertEquals(9999999999L, decodedFull.optionalLong)
        assertEquals(2.71828, decodedFull.optionalDouble, 0.00001)
        assertEquals(false, decodedFull.optionalBoolean)
        assertEquals(listOf("custom1", "custom2", "custom3"), decodedFull.optionalList)
        assertEquals(mapOf("customKey" to 999), decodedFull.optionalMap)
        
        println("✓ 默认值处理测试通过")
    }
    
    /**
     * 测试 8: 自定义序列化器
     * 验证自定义序列化器的使用
     */
    @Test
    fun testCustomSerializer() {
        println("\n=== 测试 8: 自定义序列化器 ===")
        
        val data = CustomData(
            value = "Hello World",
            timestamp = System.currentTimeMillis()
        )
        
        val jsonString = json.encodeToString(data)
        println("序列化结果: $jsonString")
        
        // 验证序列化时转换为大写
        assertTrue(jsonString.contains("HELLO WORLD"))
        
        val decoded = json.decodeFromString<CustomData>(jsonString)
        println("反序列化结果: $decoded")
        
        // 验证反序列化时转换为小写
        assertEquals("hello world", decoded.value)
        assertEquals(data.timestamp, decoded.timestamp)
        
        println("✓ 自定义序列化器测试通过")
    }
    
    /**
     * 测试 9: AI 角色配置序列化（实际业务场景）
     * 验证实际业务对象的序列化
     */
    @Test
    fun testAIRoleConfigSerialization() {
        println("\n=== 测试 9: AI 角色配置序列化 ===")
        
        val config = AIRoleConfig(
            id = "ai-role-001",
            name = "助手小王",
            description = "一个友好的 AI 助手",
            systemPrompt = "你是一个专业的 AI 助手，请用中文回答用户的问题。",
            modelId = "qwen-2.5-72b-instruct",
            temperature = 0.7,
            maxTokens = 4096,
            skills = listOf("web_search", "code_execution", "file_management"),
            skillWhitelist = listOf("web_search", "code_execution"),
            skillBlacklist = listOf("system_access"),
            createdAt = System.currentTimeMillis(),
            updatedAt = System.currentTimeMillis(),
            isActive = true
        )
        
        val jsonString = json.encodeToString(config)
        println("序列化结果:\n$jsonString")
        
        val decoded = json.decodeFromString<AIRoleConfig>(jsonString)
        assertEquals("ai-role-001", decoded.id)
        assertEquals("助手小王", decoded.name)
        assertEquals("qwen-2.5-72b-instruct", decoded.modelId)
        assertEquals(0.7, decoded.temperature, 0.01)
        assertEquals(listOf("web_search", "code_execution", "file_management"), decoded.skills)
        assertEquals(listOf("web_search", "code_execution"), decoded.skillWhitelist)
        assertEquals(listOf("system_access"), decoded.skillBlacklist)
        assertTrue(decoded.isActive)
        
        println("✓ AI 角色配置序列化测试通过")
    }
    
    /**
     * 测试 10: 疲劳状态序列化（实际业务场景）
     * 验证疲劳状态的序列化和反序列化
     */
    @Test
    fun testFatigueStateSerialization() {
        println("\n=== 测试 10: 疲劳状态序列化 ===")
        
        val state = FatigueState(
            roleId = "ai-role-001",
            currentValue = 45.5,
            lastMessageTime = System.currentTimeMillis(),
            messageCountInWindow = 87,
            windowDurationMinutes = 120,
            maxMessagesInWindow = 200,
            decayRatePerMinute = 0.5
        )
        
        val jsonString = json.encodeToString(state)
        println("序列化结果:\n$jsonString")
        
        val decoded = json.decodeFromString<FatigueState>(jsonString)
        assertEquals("ai-role-001", decoded.roleId)
        assertEquals(45.5, decoded.currentValue, 0.01)
        assertEquals(state.lastMessageTime, decoded.lastMessageTime)
        assertEquals(87, decoded.messageCountInWindow)
        assertEquals(120, decoded.windowDurationMinutes)
        assertEquals(200, decoded.maxMessagesInWindow)
        assertEquals(0.5, decoded.decayRatePerMinute, 0.01)
        
        println("✓ 疲劳状态序列化测试通过")
    }
    
    /**
     * 测试 11: 会话消息序列化（实际业务场景）
     * 验证会话消息的序列化
     */
    @Test
    fun testSessionMessageSerialization() {
        println("\n=== 测试 11: 会话消息序列化 ===")
        
        val message = SessionMessage(
            id = "msg-001",
            sessionId = "session-001",
            roleId = "ai-role-001",
            content = Message.TextMessage(
                content = "你好！我是 AI 助手。",
                timestamp = System.currentTimeMillis()
            ),
            timestamp = System.currentTimeMillis(),
            metadata = mapOf(
                "source" to "user_input",
                "language" to "zh-CN"
            )
        )
        
        val jsonString = json.encodeToString(message)
        println("序列化结果:\n$jsonString")
        
        val decoded = json.decodeFromString<SessionMessage>(jsonString)
        assertEquals("msg-001", decoded.id)
        assertEquals("session-001", decoded.sessionId)
        assertEquals("ai-role-001", decoded.roleId)
        assertTrue(decoded.content is Message.TextMessage)
        assertEquals("你好！我是 AI 助手。", (decoded.content as Message.TextMessage).content)
        assertEquals(mapOf("source" to "user_input", "language" to "zh-CN"), decoded.metadata)
        
        println("✓ 会话消息序列化测试通过")
    }
    
    /**
     * 测试 12: 复杂业务对象序列化
     * 验证综合复杂对象的序列化
     */
    @Test
    fun testComplexBusinessObjectSerialization() {
        println("\n=== 测试 12: 复杂业务对象序列化 ===")
        
        val now = System.currentTimeMillis()
        
        val config = AIRoleConfig(
            id = "ai-role-001",
            name = "助手小王",
            systemPrompt = "你是一个专业的 AI 助手。",
            modelId = "qwen-2.5-72b-instruct",
            createdAt = now,
            updatedAt = now
        )
        
        val fatigue = FatigueState(
            roleId = "ai-role-001",
            currentValue = 30.0,
            lastMessageTime = now,
            messageCountInWindow = 50
        )
        
        val session = SessionState(
            sessionId = "session-001",
            roleIds = listOf("ai-role-001", "ai-role-002"),
            messages = listOf(
                SessionMessage(
                    id = "msg-001",
                    sessionId = "session-001",
                    roleId = "ai-role-001",
                    content = Message.TextMessage("测试消息", now),
                    timestamp = now
                )
            ),
            createdAt = now,
            updatedAt = now
        )
        
        val complexObj = ComplexBusinessObject(
            id = "complex-001",
            version = 1,
            config = config,
            fatigue = fatigue,
            sessions = listOf(session),
            metadata = mapOf("env" to "production", "region" to "cn-north"),
            createdAt = now,
            updatedAt = now
        )
        
        val jsonString = json.encodeToString(complexObj)
        println("序列化结果:\n$jsonString")
        
        val decoded = json.decodeFromString<ComplexBusinessObject>(jsonString)
        assertEquals("complex-001", decoded.id)
        assertEquals(1, decoded.version)
        assertEquals("ai-role-001", decoded.config.id)
        assertEquals("助手小王", decoded.config.name)
        assertEquals(30.0, decoded.fatigue.currentValue, 0.01)
        assertEquals(1, decoded.sessions.size)
        assertEquals("session-001", decoded.sessions[0].sessionId)
        assertEquals(mapOf("env" to "production", "region" to "cn-north"), decoded.metadata)
        
        println("✓ 复杂业务对象序列化测试通过")
    }
    
    /**
     * 测试 13: JSON 配置选项测试
     * 验证不同的 Json 配置选项
     */
    @Test
    fun testJsonConfigurationOptions() {
        println("\n=== 测试 13: JSON 配置选项测试 ===")
        
        val data = SimpleConfig(name = "测试")
        
        // 测试 ignoreUnknownKeys = true
        val jsonWithIgnore = Json {
            ignoreUnknownKeys = true
        }
        
        val jsonString = jsonWithIgnore.encodeToString(data)
        println("序列化 JSON: $jsonString")
        
        // 反序列化
        val decoded = jsonWithIgnore.decodeFromString<SimpleConfig>(jsonString)
        assertEquals("测试", decoded.name)
        
        // 测试 prettyPrint 选项
        val prettyJson = Json {
            prettyPrint = true
            prettyPrintIndent = "  "
        }
        
        val prettyString = prettyJson.encodeToString(data)
        println("格式化输出:\n$prettyString")
        assertTrue(prettyString.contains("\n"))
        
        // 测试 compact 输出
        val compactString = compactJson.encodeToString(data)
        println("紧凑输出: $compactString")
        assertTrue(!compactString.contains("\n"))
        
        println("✓ JSON 配置选项测试通过")
    }
    
    /**
     * 测试 14: 大数据量序列化性能测试
     * 验证大数据量的序列化性能
     */
    @Test
    fun testLargeDataSerialization() {
        println("\n=== 测试 14: 大数据量序列化性能测试 ===")
        
        val largeList = (1..10000).map { index ->
            SessionMessage(
                id = "msg-$index",
                sessionId = "session-001",
                roleId = "ai-role-001",
                content = Message.TextMessage(
                    content = "消息内容 $index",
                    timestamp = System.currentTimeMillis() + index
                ),
                timestamp = System.currentTimeMillis() + index,
                metadata = mapOf("index" to index.toString())
            )
        }
        
        val startTime = System.currentTimeMillis()
        val jsonString = compactJson.encodeToString(largeList)
        val serializeTime = System.currentTimeMillis() - startTime
        
        println("序列化 ${largeList.size} 条消息")
        println("JSON 大小: ${jsonString.length} 字符")
        println("序列化耗时: ${serializeTime}ms")
        
        val decodeStartTime = System.currentTimeMillis()
        val decoded = compactJson.decodeFromString<List<SessionMessage>>(jsonString)
        val deserializeTime = System.currentTimeMillis() - decodeStartTime
        
        println("反序列化耗时: ${deserializeTime}ms")
        
        assertEquals(10000, decoded.size)
        assertEquals("msg-1", decoded[0].id)
        assertEquals("msg-10000", decoded[9999].id)
        
        println("✓ 大数据量序列化性能测试通过")
    }
    
    /**
     * 测试 15: Unicode 和特殊字符处理
     * 验证 Unicode 和特殊字符的序列化
     */
    @Test
    fun testUnicodeAndSpecialCharacters() {
        println("\n=== 测试 15: Unicode 和特殊字符处理 ===")
        
        val data = mapOf(
            "chinese" to "中文测试：你好世界！",
            "japanese" to "日本語テスト：こんにちは世界！",
            "korean" to "한국어 테스트: 안녕하세요 세계!",
            "emoji" to "😀🎉🚀💡❤️",
            "special" to "特殊字符：\n\t\r\"\\/",
            "mixed" to "混合：Hello 世界 🌍 こんにちは \n\t"
        )
        
        val jsonString = json.encodeToString(data)
        println("序列化结果:\n$jsonString")
        
        val decoded = json.decodeFromString<Map<String, String>>(jsonString)
        assertEquals("中文测试：你好世界！", decoded["chinese"])
        assertEquals("日本語テスト：こんにちは世界！", decoded["japanese"])
        assertEquals("한국어 테스트: 안녕하세요 세계!", decoded["korean"])
        assertEquals("😀🎉🚀💡❤️", decoded["emoji"])
        assertEquals("特殊字符：\n\t\r\"\\/", decoded["special"])
        assertEquals("混合：Hello 世界 🌍 こんにちは \n\t", decoded["mixed"])
        
        println("✓ Unicode 和特殊字符处理测试通过")
    }
    
    /**
     * 测试 16: 往返一致性测试
     * 验证序列化和反序列化的往返一致性
     */
    @Test
    fun testRoundTripConsistency() {
        println("\n=== 测试 16: 往返一致性测试 ===")
        
        val original = AIRoleConfig(
            id = "test-001",
            name = "测试角色",
            description = "这是一个测试角色",
            systemPrompt = "你是一个测试助手",
            modelId = "test-model",
            temperature = 0.85,
            maxTokens = 8192,
            skills = listOf("skill1", "skill2", "skill3"),
            skillWhitelist = listOf("skill1"),
            skillBlacklist = listOf("skill3"),
            createdAt = 1234567890123L,
            updatedAt = 1234567890456L,
            isActive = false
        )
        
        // 第一次往返
        val json1 = json.encodeToString(original)
        val decoded1 = json.decodeFromString<AIRoleConfig>(json1)
        
        // 第二次往返
        val json2 = json.encodeToString(decoded1)
        val decoded2 = json.decodeFromString<AIRoleConfig>(json2)
        
        // 验证两次往返结果一致
        assertEquals(original, decoded1)
        assertEquals(decoded1, decoded2)
        assertEquals(json1, json2)
        
        println("第一次序列化:\n$json1")
        println("\n第二次序列化:\n$json2")
        println("\n往返一致性验证通过")
        
        println("✓ 往返一致性测试通过")
    }
}
