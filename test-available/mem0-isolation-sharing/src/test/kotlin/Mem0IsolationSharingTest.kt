package com.assistant.test.mem0

import kotlinx.coroutines.*
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.*
import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.Assumptions.assumeTrue
import kotlin.system.exitProcess

/**
 * Mem0 隔离/共享测试
 * 
 * 测试目标：
 * 1. 验证角色私有记忆隔离
 * 2. 验证群聊共享记忆
 * 3. 验证混合模式（私有 + 共享）
 * 4. 验证记忆不会泄露给其他角色
 */
class Mem0IsolationSharingTest {
    
    private lateinit var mem0Client: Mem0Client
    private lateinit var memoryManager: MemoryManager
    
    companion object {
        private const val MEM0_SERVER_URL = "http://localhost:8000"
        
        @BeforeAll
        @JvmStatic
        fun checkServer() {
            println("\n========================================")
            println("Mem0 隔离/共享测试")
            println("========================================")
            println("\n前置条件:")
            println("1. Mem0 Server 必须运行在 $MEM0_SERVER_URL")
            println("2. LM Studio 必须运行并加载模型")
            println("3. Embedding 模型必须加载")
            println("\n启动 mem0 server:")
            println("  cd test-available/mem0-integration/mem0-server-local")
            println("  python run_server.py")
            println("\n或者运行打包后的可执行文件:")
            println("  dist/mem0-server.exe")
            println("\n========================================\n")
        }
    }
    
    @BeforeEach
    fun setup() = runTest {
        mem0Client = Mem0Client(MEM0_SERVER_URL)
        memoryManager = MemoryManager(mem0Client)
        
        // 检查服务器是否可用
        val isHealthy = mem0Client.healthCheck()
        if (!isHealthy) {
            println("⚠️  Mem0 Server 未运行，跳过测试")
            println("请先启动 mem0 server: python run_server.py")
        }
        assumeTrue(isHealthy, "Mem0 Server 未运行")
        
        // 清理测试数据
        println("清理测试数据...")
        mem0Client.resetAllMemories()
        delay(500)
    }
    
    @AfterEach
    fun tearDown() {
        mem0Client.close()
    }
    
    /**
     * 测试 1: 角色私有记忆隔离
     * 验证每个角色只能访问自己的记忆
     */
    @Test
    fun testPrivateMemoryIsolation() = runTest {
        println("\n=== 测试 1: 角色私有记忆隔离 ===")
        
        val role1Id = "role-001-alice"
        val role2Id = "role-002-bob"
        
        // 角色 1 创建私有记忆
        println("\n角色 1 ($role1Id) 创建私有记忆...")
        val memories1 = memoryManager.createPrivateMemory(
            roleId = role1Id,
            messages = listOf(
                Message("user", "我喜欢吃披萨，特别是意大利香肠口味"),
                Message("assistant", "好的，我记住了")
            )
        )
        println("✓ 角色 1 创建了 ${memories1.size} 条记忆")
        delay(500)
        
        // 角色 2 创建私有记忆
        println("\n角色 2 ($role2Id) 创建私有记忆...")
        val memories2 = memoryManager.createPrivateMemory(
            roleId = role2Id,
            messages = listOf(
                Message("user", "我喜欢吃寿司，特别是三文鱼寿司"),
                Message("assistant", "好的，我记住了")
            )
        )
        println("✓ 角色 2 创建了 ${memories2.size} 条记忆")
        delay(500)
        
        // 验证角色 1 只能访问自己的记忆
        println("\n验证角色 1 只能访问自己的记忆...")
        val role1Memories = memoryManager.getPrivateMemories(role1Id)
        println("角色 1 的记忆数: ${role1Memories.size}")
        
        assertTrue(role1Memories.isNotEmpty(), "角色 1 应该有记忆")
        // 验证角色 1 的记忆包含 userId
        assertTrue(
            role1Memories.all { it.userId == role1Id },
            "角色 1 的所有记忆应该都属于 role1Id"
        )
        
        // 验证角色 2 只能访问自己的记忆
        println("\n验证角色 2 只能访问自己的记忆...")
        val role2Memories = memoryManager.getPrivateMemories(role2Id)
        println("角色 2 的记忆数: ${role2Memories.size}")
        
        assertTrue(role2Memories.isNotEmpty(), "角色 2 应该有记忆")
        // 验证角色 2 的记忆包含 userId
        assertTrue(
            role2Memories.all { it.userId == role2Id },
            "角色 2 的所有记忆应该都属于 role2Id"
        )
        
        // 验证两个角色的记忆不重叠
        val role1MemoryIds = role1Memories.map { it.id }.toSet()
        val role2MemoryIds = role2Memories.map { it.id }.toSet()
        assertTrue(
            role1MemoryIds.intersect(role2MemoryIds).isEmpty(),
            "角色 1 和角色 2 的记忆不应该重叠"
        )
        
        println("\n✓ 私有记忆隔离测试通过")
    }
    
    /**
     * 测试 2: 群聊共享记忆
     * 验证群聊中的所有角色都能访问共享记忆
     */
    @Test
    fun testGroupSharedMemory() = runTest {
        println("\n=== 测试 2: 群聊共享记忆 ===")
        
        val groupId = "group-001-project-alpha"
        val role1Id = "role-001-alice"
        val role2Id = "role-002-bob"
        
        // 创建群聊共享记忆
        println("\n创建群聊共享记忆 (groupId: $groupId)...")
        val sharedMemories = memoryManager.createSharedMemory(
            groupId = groupId,
            messages = listOf(
                Message("user", "我们团队的项目截止日期是下周五"),
                Message("assistant", "好的，我记住了团队共享信息")
            )
        )
        println("✓ 创建了 ${sharedMemories.size} 条共享记忆")
        delay(500)
        
        // 角色 1 搜索群聊共享记忆
        println("\n角色 1 搜索群聊共享记忆...")
        val role1SharedMemories = memoryManager.searchSharedMemories(
            groupId = groupId,
            query = "项目截止日期"
        )
        println("角色 1 找到 ${role1SharedMemories.size} 条共享记忆")
        
        assertTrue(role1SharedMemories.isNotEmpty(), "角色 1 应该能找到群聊共享记忆")
        // 验证搜索结果属于正确的群组
        assertTrue(
            role1SharedMemories.all { it.agentId == groupId },
            "搜索结果应该都属于 groupId"
        )
        
        // 角色 2 搜索群聊共享记忆
        println("\n角色 2 搜索群聊共享记忆...")
        val role2SharedMemories = memoryManager.searchSharedMemories(
            groupId = groupId,
            query = "团队项目"
        )
        println("角色 2 找到 ${role2SharedMemories.size} 条共享记忆")
        
        assertTrue(role2SharedMemories.isNotEmpty(), "角色 2 应该能找到群聊共享记忆")
        // 验证两个角色搜索到相同的记忆
        val role1Ids = role1SharedMemories.map { it.id }.toSet()
        val role2Ids = role2SharedMemories.map { it.id }.toSet()
        assertTrue(
            role1Ids.intersect(role2Ids).isNotEmpty(),
            "两个角色应该能搜索到相同的共享记忆"
        )
        
        println("\n✓ 群聊共享记忆测试通过")
    }
    
    /**
     * 测试 3: 混合模式（私有 + 共享）
     * 验证角色既有私有记忆，也能访问群聊共享记忆
     */
    @Test
    fun testHybridMemoryMode() = runTest {
        println("\n=== 测试 3: 混合模式（私有 + 共享） ===")
        
        val groupId = "group-002-hybrid-test"
        val roleId = "role-003-charlie"
        
        // 创建角色私有记忆
        println("\n创建角色私有记忆...")
        val privateMemories = memoryManager.createPrivateMemory(
            roleId = roleId,
            messages = listOf(
                Message("user", "我个人的生日是 3 月 15 日"),
                Message("assistant", "好的，我记住了你的个人信息")
            )
        )
        println("✓ 创建了 ${privateMemories.size} 条私有记忆")
        delay(500)
        
        // 创建群聊共享记忆
        println("\n创建群聊共享记忆...")
        val sharedMemories = memoryManager.createSharedMemory(
            groupId = groupId,
            messages = listOf(
                Message("user", "我们团队每周一上午 10 点开会"),
                Message("assistant", "好的，我记住了团队会议安排")
            )
        )
        println("✓ 创建了 ${sharedMemories.size} 条共享记忆")
        delay(500)
        
        // 搜索混合记忆
        println("\n搜索混合记忆...")
        val (foundPrivate, foundShared) = memoryManager.searchHybridMemories(
            roleId = roleId,
            groupId = groupId,
            query = "日期"
        )
        
        println("找到私有记忆: ${foundPrivate.size} 条")
        println("找到共享记忆: ${foundShared.size} 条")
        
        // 验证私有记忆
        assertTrue(foundPrivate.isNotEmpty(), "应该找到私有记忆")
        // 验证私有记忆属于正确的角色
        assertTrue(
            foundPrivate.all { it.userId == roleId },
            "私有记忆应该都属于 roleId"
        )
        
        // 验证共享记忆
        assertTrue(foundShared.isNotEmpty(), "应该找到共享记忆")
        // 验证共享记忆属于正确的群组
        assertTrue(
            foundShared.all { it.agentId == groupId },
            "共享记忆应该都属于 groupId"
        )
        
        println("\n✓ 混合模式测试通过")
    }
    
    /**
     * 测试 4: 记忆不会泄露给其他群组
     * 验证不同群组的记忆是隔离的
     */
    @Test
    fun testGroupMemoryIsolation() = runTest {
        println("\n=== 测试 4: 群组记忆隔离 ===")
        
        val group1Id = "group-003-marketing"
        val group2Id = "group-004-development"
        
        // 群组 1 创建共享记忆
        println("\n群组 1 ($group1Id) 创建共享记忆...")
        val group1Memories = memoryManager.createSharedMemory(
            groupId = group1Id,
            messages = listOf(
                Message("user", "市场营销预算是 50 万"),
                Message("assistant", "好的，我记住了")
            )
        )
        println("✓ 群组 1 创建了 ${group1Memories.size} 条记忆")
        delay(500)
        
        // 群组 2 创建共享记忆
        println("\n群组 2 ($group2Id) 创建共享记忆...")
        val group2Memories = memoryManager.createSharedMemory(
            groupId = group2Id,
            messages = listOf(
                Message("user", "技术债务需要 2 周时间清理"),
                Message("assistant", "好的，我记住了")
            )
        )
        println("✓ 群组 2 创建了 ${group2Memories.size} 条记忆")
        delay(500)
        
        // 验证群组 1 只能访问自己的记忆
        println("\n验证群组 1 只能访问自己的记忆...")
        val group1SearchResults = memoryManager.searchSharedMemories(
            groupId = group1Id,
            query = "预算"
        )
        
        assertTrue(group1SearchResults.isNotEmpty(), "群组 1 应该能找到预算相关记忆")
        // 验证搜索结果都属于群组 1
        assertTrue(
            group1SearchResults.all { it.agentId == group1Id },
            "群组 1 的搜索结果应该都属于 group1Id"
        )
        
        // 验证群组 2 只能访问自己的记忆
        println("\n验证群组 2 只能访问自己的记忆...")
        val group2SearchResults = memoryManager.searchSharedMemories(
            groupId = group2Id,
            query = "技术"
        )
        
        assertTrue(group2SearchResults.isNotEmpty(), "群组 2 应该能找到技术相关记忆")
        // 验证搜索结果都属于群组 2
        assertTrue(
            group2SearchResults.all { it.agentId == group2Id },
            "群组 2 的搜索结果应该都属于 group2Id"
        )
        
        // 验证两个群组的搜索结果不重叠
        val group1Ids = group1SearchResults.map { it.id }.toSet()
        val group2Ids = group2SearchResults.map { it.id }.toSet()
        assertTrue(
            group1Ids.intersect(group2Ids).isEmpty(),
            "群组 1 和群组 2 的搜索结果不应该重叠"
        )
        
        println("\n✓ 群组记忆隔离测试通过")
    }
    
    /**
     * 测试 5: 多角色群聊场景
     * 验证多个角色在同一个群聊中共享记忆
     */
    @Test
    fun testMultiRoleGroupChat() = runTest {
        println("\n=== 测试 5: 多角色群聊场景 ===")
        
        val groupId = "group-005-multi-role"
        val role1Id = "role-004-david"
        val role2Id = "role-005-eva"
        val role3Id = "role-006-frank"
        
        // 创建群聊共享记忆
        println("\n创建群聊共享记忆...")
        val sharedMemories = memoryManager.createSharedMemory(
            groupId = groupId,
            messages = listOf(
                Message("user", "我们决定使用 Kotlin 作为主要开发语言"),
                Message("assistant", "好的，我记住了团队决策")
            )
        )
        println("✓ 创建了 ${sharedMemories.size} 条共享记忆")
        delay(500)
        
        // 验证所有角色都能访问共享记忆
        println("\n验证所有角色都能访问共享记忆...")
        
        val role1Results = memoryManager.searchSharedMemories(groupId, "开发语言")
        println("角色 1 找到 ${role1Results.size} 条记忆")
        assertTrue(role1Results.isNotEmpty(), "角色 1 应该能找到共享记忆")
        
        val role2Results = memoryManager.searchSharedMemories(groupId, "Kotlin")
        println("角色 2 找到 ${role2Results.size} 条记忆")
        assertTrue(role2Results.isNotEmpty(), "角色 2 应该能找到共享记忆")
        
        val role3Results = memoryManager.searchSharedMemories(groupId, "团队决策")
        println("角色 3 找到 ${role3Results.size} 条记忆")
        assertTrue(role3Results.isNotEmpty(), "角色 3 应该能找到共享记忆")
        
        println("\n✓ 多角色群聊测试通过")
    }
    
    /**
     * 测试 6: 记忆更新和删除
     * 验证记忆的更新和删除操作
     */
    @Test
    fun testMemoryUpdateAndDelete() = runTest {
        println("\n=== 测试 6: 记忆更新和删除 ===")
        
        val roleId = "role-007-george"
        
        // 创建记忆
        println("\n创建记忆...")
        val memories = memoryManager.createPrivateMemory(
            roleId = roleId,
            messages = listOf(
                Message("user", "我的电话号码是 123-456-7890"),
                Message("assistant", "好的，我记住了")
            )
        )
        println("✓ 创建了 ${memories.size} 条记忆")
        delay(500)
        
        // 验证记忆存在
        println("\n验证记忆存在...")
        val searchResults = memoryManager.searchPrivateMemories(roleId, "电话号码")
        assertTrue(searchResults.isNotEmpty(), "应该能找到电话号码记忆")
        println("✓ 找到记忆: ${searchResults.first().memory}")
        
        // 删除记忆
        println("\n删除记忆...")
        val memoryToDelete = searchResults.first()
        val deleteSuccess = mem0Client.deleteMemory(memoryToDelete.id)
        assertTrue(deleteSuccess, "删除记忆应该成功")
        println("✓ 记忆已删除")
        delay(500)
        
        // 验证记忆已删除
        println("\n验证记忆已删除...")
        val afterDelete = memoryManager.searchPrivateMemories(roleId, "电话号码")
        assertTrue(afterDelete.isEmpty(), "删除后不应该找到记忆")
        println("✓ 记忆已成功删除")
        
        println("\n✓ 记忆更新和删除测试通过")
    }
    
    /**
     * 测试 7: 性能测试
     * 验证大量记忆的搜索性能
     */
    @Test
    fun testPerformance() = runTest {
        println("\n=== 测试 7: 性能测试 ===")
        
        val roleId = "role-008-performance-test"
        val memoryCount = 10
        
        // 创建多条记忆
        println("\n创建 $memoryCount 条记忆...")
        val startTime = System.currentTimeMillis()
        
        repeat(memoryCount) { index ->
            memoryManager.createPrivateMemory(
                roleId = roleId,
                messages = listOf(
                    Message("user", "这是第 ${index + 1} 条测试记忆，主题是主题${index % 3}"),
                    Message("assistant", "好的，我记住了")
                )
            )
            delay(200) // 避免请求过快
        }
        
        val createTime = System.currentTimeMillis() - startTime
        println("✓ 创建完成，耗时: ${createTime}ms")
        
        // 搜索记忆
        println("\n搜索记忆...")
        val searchStartTime = System.currentTimeMillis()
        
        val searchResults = memoryManager.searchPrivateMemories(roleId, "主题1")
        
        val searchTime = System.currentTimeMillis() - searchStartTime
        println("✓ 搜索完成，耗时: ${searchTime}ms")
        println("找到 ${searchResults.size} 条记忆")
        
        assertTrue(searchResults.isNotEmpty(), "应该找到记忆")
        println("\n✓ 性能测试通过")
    }
}
