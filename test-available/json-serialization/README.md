# JSON 序列化可行性测试

## 测试目的

验证 Kotlinx Serialization 在各种场景下的序列化和反序列化功能，确保项目中的 JSON 序列化方案可行且稳定。

## 测试范围

### 1. 基础类型序列化
- String, Int, Long, Double, Boolean 等基础类型
- Unicode 字符和特殊字符处理

### 2. 可空类型序列化
- null 值的序列化和反序列化
- 可空字段的默认值处理

### 3. 集合类型序列化
- List, Set, Map 等集合类型
- 嵌套集合（List<List<T>>, Map<String, Map<String, T>>）

### 4. 嵌套对象序列化
- 复杂嵌套结构的序列化
- 多层嵌套对象

### 5. 枚举类型序列化
- 枚举值的序列化和反序列化
- 枚举列表的序列化

### 6. 密封类多态序列化
- 密封类的多态序列化
- 不同子类的序列化和反序列化

### 7. 默认值处理
- 默认值的序列化和反序列化
- encodeDefaults 配置选项

### 8. 自定义序列化器
- KSerializer 接口实现
- 自定义序列化逻辑

### 9. 实际业务对象序列化
- AI 角色配置（AIRoleConfig）
- 疲劳状态（FatigueState）
- 会话消息（SessionMessage）
- 会话状态（SessionState）
- 复杂业务对象（ComplexBusinessObject）

### 10. JSON 配置选项
- ignoreUnknownKeys：忽略未知字段
- isLenient：宽松解析
- encodeDefaults：编码默认值
- prettyPrint：格式化输出

### 11. 性能测试
- 大数据量序列化性能
- 序列化和反序列化耗时

### 12. 往返一致性
- 序列化 -> 反序列化 -> 序列化的一致性

## 技术栈

- **Kotlin**: 2.3.10
- **Kotlinx Serialization**: 1.7.3
- **JUnit 5**: 5.10.2
- **Gradle**: 8.13

## 构建和运行

### 前置要求

- JDK 17 或更高版本
- Gradle 8.13

### 构建项目

```bash
cd test-available/json-serialization
../../gradlew clean build
```

### 运行测试

```bash
../../gradlew test --console=plain
```

### 查看测试报告

测试报告位于：`build/reports/tests/test/index.html`

## 测试用例

| 测试编号 | 测试名称 | 测试内容 |
|---------|---------|---------|
| 1 | testBasicTypes | 基础类型序列化 |
| 2 | testNullableTypes | 可空类型序列化 |
| 3 | testCollectionTypes | 集合类型序列化 |
| 4 | testNestedObjects | 嵌套对象序列化 |
| 5 | testEnumSerialization | 枚举类型序列化 |
| 6 | testSealedClassSerialization | 密封类多态序列化 |
| 7 | testDefaultValues | 默认值处理 |
| 8 | testCustomSerializer | 自定义序列化器 |
| 9 | testAIRoleConfigSerialization | AI 角色配置序列化 |
| 10 | testFatigueStateSerialization | 疲劳状态序列化 |
| 11 | testSessionMessageSerialization | 会话消息序列化 |
| 12 | testComplexBusinessObjectSerialization | 复杂业务对象序列化 |
| 13 | testJsonConfigurationOptions | JSON 配置选项测试 |
| 14 | testLargeDataSerialization | 大数据量序列化性能测试 |
| 15 | testUnicodeAndSpecialCharacters | Unicode 和特殊字符处理 |
| 16 | testRoundTripConsistency | 往返一致性测试 |

## 验证标准

### 成功标准

1. 所有测试用例通过（16/16）
2. 序列化和反序列化结果一致
3. 性能测试在合理范围内（10000 条消息序列化 < 5 秒）
4. Unicode 和特殊字符正确处理
5. 往返一致性验证通过

### 失败标准

1. 任何测试用例失败
2. 序列化结果与预期不符
3. 反序列化抛出异常
4. 性能测试超时
5. 往返一致性验证失败

## 关键发现

### Unix 时间戳方案

项目中所有时间字段统一使用 `Long` 类型（Unix 时间戳，毫秒），避免使用 `Instant` 等复杂对象：

```kotlin
@Serializable
data class AIRoleConfig(
    val id: String,
    val name: String,
    val createdAt: Long,  // Unix 时间戳（毫秒）
    val updatedAt: Long   // Unix 时间戳（毫秒）
)
```

**优势**：
- 无需自定义序列化器
- 跨平台兼容性好
- 性能更优
- 代码更简洁

### Kotlinx Serialization 配置建议

```kotlin
val json = Json {
    ignoreUnknownKeys = true      // 忽略未知字段，提高兼容性
    isLenient = true              // 宽松解析，允许非标准 JSON
    encodeDefaults = true         // 编码默认值，确保数据完整性
    prettyPrint = true            // 格式化输出，便于调试
}
```

### 密封类多态序列化

使用 `@SerialName` 注解标记密封类子类：

```kotlin
@Serializable
sealed class Message {
    @Serializable
    @SerialName("text")
    data class TextMessage(...) : Message()
    
    @Serializable
    @SerialName("image")
    data class ImageMessage(...) : Message()
}
```

## 注意事项

1. **编译器插件**：必须在 `build.gradle.kts` 中添加 `kotlin("plugin.serialization")` 插件
2. **数据类位置**：数据类应放在 `src/main/kotlin` 目录，确保编译器插件生成序列化代码
3. **默认值**：使用 `encodeDefaults = false` 可以减少 JSON 大小，但可能导致默认值丢失
4. **性能优化**：大数据量序列化时，使用 `prettyPrint = false` 提高性能

## 后续工作

- [ ] 在实际项目中集成序列化方案
- [ ] 添加更多边界情况测试
- [ ] 性能优化和基准测试
- [ ] 与 Ktor ContentNegotiation 集成测试

## 相关文档

- [test-available/index.md](../index.md) - 可行性测试总索引
- [test-available/ai-role-system/](../ai-role-system/) - AI 角色系统测试
- [gradle/libs.versions.toml](../../gradle/libs.versions.toml) - 依赖版本管理
