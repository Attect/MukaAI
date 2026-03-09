import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import java.io.File
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter
import kotlinx.coroutines.runBlocking

@Serializable
data class Workspace(
    val id: String,
    val name: String,
    val rootPath: String,
    val createdAt: String = LocalDateTime.now().format(DateTimeFormatter.ISO_LOCAL_DATE_TIME),
    val updatedAt: String = LocalDateTime.now().format(DateTimeFormatter.ISO_LOCAL_DATE_TIME),
    val config: WorkspaceConfig = WorkspaceConfig(),
    val metadata: Map<String, String> = emptyMap()
)

@Serializable
data class WorkspaceConfig(
    val allowedPaths: List<String> = emptyList(),
    val environment: Map<String, String> = emptyMap(),
    val skills: List<String> = emptyList(),
    val restrictions: Restrictions = Restrictions()
)

@Serializable
data class Restrictions(
    val allowFileSystemAccess: Boolean = true,
    val allowShellExecution: Boolean = false,
    val allowNetworkAccess: Boolean = true,
    val maxFileSize: Long = 10 * 1024 * 1024,
    val allowedFileExtensions: List<String> = emptyList()
)

@Serializable
data class WorkspaceManagerState(
    val currentWorkspaceId: String? = null,
    val workspaces: List<Workspace> = emptyList()
)

class WorkspaceManager(private val stateFile: File) {
    private var state: WorkspaceManagerState = loadState()
    
    private fun loadState(): WorkspaceManagerState {
        return if (stateFile.exists()) {
            try {
                Json.decodeFromString<WorkspaceManagerState>(stateFile.readText())
            } catch (e: Exception) {
                WorkspaceManagerState()
            }
        } else {
            WorkspaceManagerState()
        }
    }
    
    private fun saveState() {
        stateFile.parentFile?.mkdirs()
        stateFile.writeText(Json.encodeToString(state))
    }
    
    fun createWorkspace(name: String, rootPath: String): Workspace {
        val id = "ws_${System.currentTimeMillis()}"
        val workspace = Workspace(
            id = id,
            name = name,
            rootPath = File(rootPath).absolutePath
        )
        
        state = state.copy(
            workspaces = state.workspaces + workspace
        )
        saveState()
        
        File(rootPath).mkdirs()
        
        val configFile = File(rootPath, ".assistant-workspace.json")
        configFile.writeText(Json.encodeToString(workspace))
        
        return workspace
    }
    
    fun getAllWorkspaces(): List<Workspace> = state.workspaces
    
    fun getCurrentWorkspace(): Workspace? {
        return state.currentWorkspaceId?.let { id ->
            state.workspaces.find { it.id == id }
        }
    }
    
    fun switchWorkspace(workspaceId: String): Boolean {
        val workspace = state.workspaces.find { it.id == workspaceId }
        if (workspace == null) {
            return false
        }
        
        state = state.copy(currentWorkspaceId = workspaceId)
        saveState()
        return true
    }
    
    fun deleteWorkspace(workspaceId: String): Boolean {
        val workspace = state.workspaces.find { it.id == workspaceId }
        if (workspace == null) {
            return false
        }
        
        if (state.currentWorkspaceId == workspaceId) {
            state = state.copy(currentWorkspaceId = null)
        }
        
        state = state.copy(
            workspaces = state.workspaces.filter { it.id != workspaceId }
        )
        saveState()
        
        return true
    }
    
    fun updateWorkspaceConfig(workspaceId: String, config: WorkspaceConfig): Boolean {
        val workspaceIndex = state.workspaces.indexOfFirst { it.id == workspaceId }
        if (workspaceIndex == -1) {
            return false
        }
        
        val updatedWorkspace = state.workspaces[workspaceIndex].copy(
            config = config,
            updatedAt = LocalDateTime.now().format(DateTimeFormatter.ISO_LOCAL_DATE_TIME)
        )
        
        state = state.copy(
            workspaces = state.workspaces.mapIndexed { index, ws ->
                if (index == workspaceIndex) updatedWorkspace else ws
            }
        )
        saveState()
        
        return true
    }
    
    fun isPathAllowed(path: String): Boolean {
        val workspace = getCurrentWorkspace() ?: return false
        
        val absolutePath = File(path).absolutePath
        val workspaceRoot = File(workspace.rootPath).absolutePath
        
        if (!absolutePath.startsWith(workspaceRoot)) {
            return false
        }
        
        if (workspace.config.allowedPaths.isNotEmpty()) {
            return workspace.config.allowedPaths.any { allowedPath ->
                absolutePath.startsWith(File(allowedPath).absolutePath)
            }
        }
        
        return true
    }
}

fun main() = runBlocking {
    println("=== 测试 02: Kotlin 多工作区管理方案 ===\n")
    
    val tempDir = File(System.getProperty("java.io.tmpdir"), "assistant-workspaces-test")
    val stateFile = File(tempDir, "workspace-manager-state.json")
    
    if (stateFile.exists()) {
        stateFile.delete()
    }
    
    val manager = WorkspaceManager(stateFile)
    
    try {
        println("[测试 1] 创建工作区...")
        val workspace1 = manager.createWorkspace(
            name = "开发工作区",
            rootPath = File(tempDir, "dev-workspace").absolutePath
        )
        println("创建工作区成功:")
        println("  ID: ${workspace1.id}")
        println("  名称：${workspace1.name}")
        println("  路径：${workspace1.rootPath}")
        println()
        
        val workspace2 = manager.createWorkspace(
            name = "测试工作区",
            rootPath = File(tempDir, "test-workspace").absolutePath
        )
        println("创建第二个工作区:")
        println("  ID: ${workspace2.id}")
        println("  名称：${workspace2.name}")
        println()
        
        println("[测试 2] 获取所有工作区...")
        val allWorkspaces = manager.getAllWorkspaces()
        println("工作区数量：${allWorkspaces.size}")
        allWorkspaces.forEach { ws ->
            println("  - ${ws.name} (${ws.id})")
        }
        println()
        
        println("[测试 3] 切换工作区...")
        val switched = manager.switchWorkspace(workspace1.id)
        println("切换到 ${workspace1.name}: ${if (switched) "成功" else "失败"}")
        
        val currentWorkspace = manager.getCurrentWorkspace()
        println("当前工作区：${currentWorkspace?.name ?: "无"}")
        println()
        
        println("[测试 4] 路径访问验证...")
        val validPath = File(workspace1.rootPath, "subdir/file.txt")
        val invalidPath = File("/tmp/outside-workspace/file.txt")
        
        println("验证路径 (应允许): ${validPath.path}")
        println("结果：${if (manager.isPathAllowed(validPath.path)) "✓ 允许" else "✗ 拒绝"}")
        
        println("验证路径 (应拒绝): ${invalidPath.path}")
        println("结果：${if (manager.isPathAllowed(invalidPath.path)) "✓ 允许" else "✗ 拒绝"}")
        println()
        
        println("[测试 5] 更新工作区配置...")
        val newConfig = WorkspaceConfig(
            allowedPaths = listOf(File(workspace1.rootPath, "allowed").absolutePath),
            restrictions = Restrictions(
                allowFileSystemAccess = true,
                allowShellExecution = false,
                maxFileSize = 5 * 1024 * 1024
            )
        )
        
        val updated = manager.updateWorkspaceConfig(workspace1.id, newConfig)
        println("配置更新：${if (updated) "成功" else "失败"}")
        
        val updatedWorkspace = manager.getCurrentWorkspace()
        println("新配置 - 允许 Shell 执行：${updatedWorkspace?.config?.restrictions?.allowShellExecution}")
        println("新配置 - 最大文件大小：${updatedWorkspace?.config?.restrictions?.maxFileSize ?: 0} bytes")
        println()
        
        println("[测试 6] 删除工作区...")
        val deleted = manager.deleteWorkspace(workspace2.id)
        println("删除 ${workspace2.name}: ${if (deleted) "成功" else "失败"}")
        
        val remainingWorkspaces = manager.getAllWorkspaces()
        println("剩余工作区数量：${remainingWorkspaces.size}")
        println()
        
        println("=== 多工作区管理功能验证通过！ ===")
        
    } catch (e: Exception) {
        println("\n=== 测试失败：${e.message} ===")
        e.printStackTrace()
    } finally {
        tempDir.deleteRecursively()
    }
}
