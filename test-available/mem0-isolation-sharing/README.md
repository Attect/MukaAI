# Mem0 隔离/共享测试

## 测试目的

验证 Mem0 AI 记忆系统在多角色场景下的隔离和共享机制，确保：
1. 每个 AI 角色的私有记忆完全隔离
2. 群聊中的共享记忆对所有参与者可见
3. 混合模式下私有和共享记忆正确分离
4. 不同群组的记忆互不干扰

## 测试范围

### 1. 私有记忆隔离测试
- 每个角色使用独立的 `user_id` (即 `role_id`)
- 验证角色只能访问自己的记忆
- 验证记忆不会泄露给其他角色

### 2. 群聊共享记忆测试
- 群聊使用共享的 `group_id` 和 `session_id`
- 验证共享记忆对所有参与者可见
- 验证共享记忆的搜索和访问

### 3. 混合模式测试
- 角色既有私有记忆，也能访问群聊共享记忆
- 验证私有和共享记忆的正确隔离
- 验证混合搜索功能

### 4. 群组隔离测试
- 不同群组的记忆完全隔离
- 验证群组间记忆不会泄露

### 5. 多角色群聊测试
- 多个角色在同一个群聊中
- 验证所有角色都能访问共享记忆

### 6. 记忆管理测试
- 记忆的创建、搜索、删除操作
- 验证记忆管理的正确性

### 7. 性能测试
- 大量记忆的创建和搜索
- 验证系统性能

## 技术栈

- **Kotlin**: 2.3.10
- **Ktor Client**: 3.4.1
- **Kotlinx Serialization**: 1.7.3
- **JUnit 5**: 5.10.2
- **Mem0 Server**: Python FastAPI 服务

## 前置条件

### 1. 启动 Mem0 Server

```bash
cd test-available/mem0-integration/mem0-server-local

# 方式 1: Python 脚本运行
python run_server.py

# 方式 2: 运行打包后的可执行文件
dist/mem0-server.exe
```

### 2. 启动 LM Studio

1. 加载 LLM 模型（如 `qwen3.5-9b-uncensored-hauhaucs-aggressive`）
2. 加载 Embedding 模型（如 `nomic-embed-text-v1.5`）
3. 启动本地服务器（端口 1234 或 11452）
4. 确保 Function Calling 已启用

### 3. 配置环境变量

创建 `.env` 文件：

```bash
LM_STUDIO_BASE_URL=http://localhost:1234
LM_STUDIO_MODEL=qwen3.5-9b-uncensored-hauhaucs-aggressive
EMBEDDER_MODEL=nomic-embed-text-v1.5
FAISS_PATH=./data/faiss_memories
```

## 构建和运行

### 构建项目

```bash
cd test-available/mem0-isolation-sharing
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
| 1 | testPrivateMemoryIsolation | 角色私有记忆隔离 |
| 2 | testGroupSharedMemory | 群聊共享记忆 |
| 3 | testHybridMemoryMode | 混合模式（私有 + 共享） |
| 4 | testGroupMemoryIsolation | 群组记忆隔离 |
| 5 | testMultiRoleGroupChat | 多角色群聊场景 |
| 6 | testMemoryUpdateAndDelete | 记忆更新和删除 |
| 7 | testPerformance | 性能测试 |

## 验证标准

### 成功标准

1. 所有测试用例通过（7/7）
2. 私有记忆完全隔离，不同角色无法访问彼此的记忆
3. 共享记忆对所有群组成员可见
4. 混合模式下私有和共享记忆正确分离
5. 不同群组的记忆互不干扰
6. 性能测试在合理范围内

### 失败标准

1. 任何测试用例失败
2. 记忆泄露给未授权的角色
3. 共享记忆无法被群组成员访问
4. 性能测试超时

## 隔离策略

### 1. 严格隔离模式 (STRICT)

```kotlin
// 每个角色只能访问自己的记忆
val memories = memoryManager.getPrivateMemories(roleId)
```

**适用场景**：
- 单角色对话
- 需要完全隐私的场景

### 2. 群组共享模式 (GROUP_SHARED)

```kotlin
// 群聊中的所有角色都能访问共享记忆
val memories = memoryManager.searchSharedMemories(groupId, query)
```

**适用场景**：
- 群聊对话
- 团队协作场景

### 3. 混合模式 (HYBRID)

```kotlin
// 角色既有私有记忆，也能访问群聊共享记忆
val (privateMemories, sharedMemories) = memoryManager.searchHybridMemories(
    roleId = roleId,
    groupId = groupId,
    query = query
)
```

**适用场景**：
- 复杂对话场景
- 需要个人偏好 + 团队信息的场景

## 关键发现

### 1. 隔离机制

Mem0 通过 `user_id` 实现记忆隔离：
- 每个 AI 角色使用独立的 `role_id` 作为 `user_id`
- 搜索时通过 `user_id` 过滤，确保只返回该角色的记忆
- 不同角色的记忆存储在同一个 FAISS 索引中，但通过元数据隔离

### 2. 共享机制

群聊共享记忆通过 `group_id` 和 `session_id` 实现：
- 创建记忆时添加 `groupId` 和 `sessionId` 元数据
- 搜索时通过 `groupId` 过滤，返回群组共享记忆
- 所有群组成员都能搜索到相同的共享记忆

### 3. 混合模式实现

```kotlin
// 私有记忆
val privateMemories = searchPrivateMemories(roleId, query)

// 共享记忆
val sharedMemories = searchSharedMemories(groupId, query)

// 合并结果
val allMemories = privateMemories + sharedMemories
```

## 注意事项

1. **Mem0 Server 必须运行**: 测试前确保 mem0 server 已启动
2. **LM Studio 必须运行**: LLM 和 Embedding 模型必须加载
3. **测试间隔**: 测试之间有延迟，避免请求过快
4. **数据清理**: 每个测试前会重置所有记忆
5. **性能考虑**: 大量记忆创建可能耗时较长

## 后续工作

- [ ] 在实际项目中集成记忆管理
- [ ] 添加记忆过期和清理策略
- [ ] 实现记忆优先级和权重
- [ ] 添加记忆访问日志和审计
- [ ] 性能优化和压力测试

## 相关文档

- [test-available/index.md](../index.md) - 可行性测试总索引
- [test-available/mem0-integration/](../mem0-integration/) - Mem0 集成测试
- [test-available/ai-role-system/](../ai-role-system/) - AI 角色系统测试
