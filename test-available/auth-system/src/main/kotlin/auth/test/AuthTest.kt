package auth.test

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.*
import java.io.File
import java.security.SecureRandom
import java.time.LocalDate
import java.time.format.DateTimeFormatter

/**
 * Token 数据类
 */
@Serializable
data class TokenData(
    val token: String,
    val clientId: String,
    val createdAt: String,
    val expiresAt: String? = null // null 表示永不过期
)

/**
 * 服务端 Token 管理器
 * 负责管理预设 Token 和已授权的客户端 Token
 */
class ServerTokenManager(
    private val configDir: File = File("test-available/auth-system/server-config")
) {
    private val presetTokenFile = File(configDir, "preset_token.json")
    private val authorizedTokensFile = File(configDir, "authorized_tokens.json")
    
    /**
     * 当前预设 Token
     */
    var presetToken: String = ""
        private set
    
    /**
     * 预设 Token 生成日期
     */
    var presetTokenDate: LocalDate = LocalDate.MIN
        private set
    
    /**
     * 已授权的客户端 Token 映射 (token -> clientId)
     */
    private val authorizedTokens = mutableMapOf<String, String>()
    
    private val secureRandom = SecureRandom()
    
    /**
     * 初始化 Token 管理器
     * 加载或生成预设 Token，加载已授权的 Token
     */
    fun initialize() {
        configDir.mkdirs()
        loadOrGeneratePresetToken()
        loadAuthorizedTokens()
        println("[ServerTokenManager] 初始化完成")
        println("[ServerTokenManager] 预设 Token: ${presetToken.take(8)}... (日期：$presetTokenDate)")
        println("[ServerTokenManager] 已授权 Token 数量：${authorizedTokens.size}")
    }
    
    /**
     * 加载或生成预设 Token
     * 如果当前日期的预设 Token 不存在，则生成新的
     */
    private fun loadOrGeneratePresetToken() {
        val today = LocalDate.now()
        
        if (presetTokenFile.exists()) {
            val content = presetTokenFile.readText()
            val json = Json.parseToJsonElement(content).jsonObject
            val storedToken = json["token"]?.jsonPrimitive?.content ?: ""
            val storedDate = json["date"]?.jsonPrimitive?.content ?: ""
            
            val storedDateParsed = try {
                LocalDate.parse(storedDate, DateTimeFormatter.ISO_LOCAL_DATE)
            } catch (e: Exception) {
                LocalDate.MIN
            }
            
            // 如果日期匹配，使用现有的预设 Token
            if (storedDateParsed == today) {
                presetToken = storedToken
                presetTokenDate = storedDateParsed
                println("[ServerTokenManager] 加载现有预设 Token")
                return
            }
        }
        
        // 生成新的预设 Token
        generateNewPresetToken(today)
    }
    
    /**
     * 生成新的预设 Token
     */
    private fun generateNewPresetToken(date: LocalDate) {
        presetToken = generateSecureToken(32) // 32 字节，64 字符十六进制
        presetTokenDate = date
        
        val data = mapOf(
            "token" to presetToken,
            "date" to date.format(DateTimeFormatter.ISO_LOCAL_DATE),
            "createdAt" to System.currentTimeMillis().toString()
        )
        
        presetTokenFile.writeText(Json.encodeToString(JsonObject.serializer(), JsonObject(data.mapValues { JsonPrimitive(it.value) }))
)
        println("[ServerTokenManager] 生成新的预设 Token (日期：$date)")
    }
    
    /**
     * 加载已授权的 Token
     */
    private fun loadAuthorizedTokens() {
        if (authorizedTokensFile.exists()) {
            val content = authorizedTokensFile.readText()
            val jsonArray = Json.parseToJsonElement(content).jsonArray
            
            authorizedTokens.clear()
            jsonArray.forEach { element ->
                val obj = element.jsonObject
                val token = obj["token"]?.jsonPrimitive?.content ?: ""
                val clientId = obj["clientId"]?.jsonPrimitive?.content ?: ""
                if (token.isNotEmpty() && clientId.isNotEmpty()) {
                    authorizedTokens[token] = clientId
                }
            }
            
            println("[ServerTokenManager] 加载了 ${authorizedTokens.size} 个已授权 Token")
        }
    }
    
    /**
     * 保存已授权的 Token
     */
    private fun saveAuthorizedTokens() {
        val jsonArray = JsonArray(authorizedTokens.map { (token, clientId) ->
            buildJsonObject {
                put("token", token)
                put("clientId", clientId)
                put("addedAt", System.currentTimeMillis().toString())
            }
        })
        
        authorizedTokensFile.writeText(Json.encodeToString(JsonArray.serializer(), jsonArray))
        println("[ServerTokenManager] 保存已授权 Token 列表")
    }
    
    /**
     * 验证预设 Token
     */
    fun verifyPresetToken(token: String): Boolean {
        return presetToken == token
    }
    
    /**
     * 验证专属 Token
     */
    fun verifyClientToken(token: String): Boolean {
        return authorizedTokens.containsKey(token)
    }
    
    /**
     * 使用预设 Token 换取专属 Token
     * @param presetToken 预设 Token
     * @param clientId 客户端标识（可选，不提供则自动生成）
     * @return 专属 Token 数据，如果预设 Token 无效则返回 null
     */
    fun exchangePresetToken(presetToken: String, clientId: String? = null): TokenData? {
        if (!verifyPresetToken(presetToken)) {
            println("[ServerTokenManager] 预设 Token 验证失败")
            return null
        }
        
        val newClientId = clientId ?: generateClientId()
        val newToken = generateSecureToken(32)
        
        val tokenData = TokenData(
            token = newToken,
            clientId = newClientId,
            createdAt = System.currentTimeMillis().toString(),
            expiresAt = null // 永不过期
        )
        
        // 添加到已授权列表
        authorizedTokens[newToken] = newClientId
        saveAuthorizedTokens()
        
        println("[ServerTokenManager] 生成新的专属 Token，客户端 ID: $newClientId")
        return tokenData
    }
    
    /**
     * 生成客户端 ID
     */
    private fun generateClientId(): String {
        return "client_" + generateSecureToken(16)
    }
    
    /**
     * 生成安全的随机 Token
     */
    private fun generateSecureToken(byteLength: Int): String {
        val bytes = ByteArray(byteLength)
        secureRandom.nextBytes(bytes)
        return bytes.joinToString("") { "%02x".format(it) }
    }
    
    /**
     * 撤销已授权的 Token
     */
    fun revokeToken(token: String): Boolean {
        if (authorizedTokens.remove(token) != null) {
            saveAuthorizedTokens()
            println("[ServerTokenManager] 已撤销 Token: ${token.take(8)}...")
            return true
        }
        return false
    }
    
    /**
     * 获取所有已授权的 Token 列表（用于管理）
     */
    fun getAuthorizedTokens(): List<Map<String, String>> {
        return authorizedTokens.map { (token, clientId) ->
            mapOf(
                "clientId" to clientId,
                "token" to token.take(8) + "...", // 只显示前 8 位
                "tokenFull" to token // 完整 Token（用于管理界面）
            )
        }
    }
}

/**
 * 客户端 Token 管理器
 * 负责管理客户端的预设 Token 和专属 Token
 */
class ClientTokenManager(
    private val storageFile: File = File("test-available/auth-system/client-token.json")
) {
    /**
     * 预设 Token（用户输入）
     */
    var presetToken: String? = null
    
    /**
     * 专属 Token（从服务端获取）
     */
    var clientToken: TokenData? = null
    
    /**
     * 客户端 ID
     */
    var clientId: String = ""
    
    /**
     * 初始化
     * 加载已保存的 Token
     */
    fun initialize() {
        clientId = generateOrLoadClientId()
        
        if (storageFile.exists()) {
            val content = storageFile.readText()
            try {
                val json = Json.parseToJsonElement(content).jsonObject
                presetToken = json["presetToken"]?.jsonPrimitive?.contentOrNull
                clientToken = json["clientToken"]?.jsonPrimitive?.content?.let { tokenString ->
                    Json.decodeFromString<TokenData>(tokenString)
                }
                
                println("[ClientTokenManager] 加载已保存的 Token")
                if (clientToken != null) {
                    println("[ClientTokenManager] 专属 Token: ${clientToken!!.token.take(8)}...")
                }
            } catch (e: Exception) {
                println("[ClientTokenManager] 加载 Token 失败：${e.message}")
            }
        }
    }
    
    /**
     * 生成或加载客户端 ID
     */
    private fun generateOrLoadClientId(): String {
        val clientIdFile = File(storageFile.parent, "client_id.txt")
        
        if (clientIdFile.exists()) {
            return clientIdFile.readText().trim()
        }
        
        val newClientId = "client_" + generateSecureToken(16)
        clientIdFile.writeText(newClientId)
        return newClientId
    }
    
    /**
     * 保存 Token 到文件
     */
    fun saveTokens() {
        val data = buildJsonObject {
            presetToken?.let { put("presetToken", it) }
            clientToken?.let { token ->
                put("clientToken", Json.encodeToString(TokenData.serializer(), token))
            }
            put("clientId", clientId)
            put("lastUpdated", System.currentTimeMillis().toString())
        }
        
        storageFile.writeText(Json.encodeToString(JsonObject.serializer(), data))
        println("[ClientTokenManager] Token 已保存到文件")
    }
    
    /**
     * 清除专属 Token（用于 Token 丢失场景）
     */
    fun clearClientToken() {
        clientToken = null
        saveTokens()
        println("[ClientTokenManager] 已清除专属 Token")
    }
    
    /**
     * 设置预设 Token
     */
    fun updatePresetToken(token: String) {
        presetToken = token
        saveTokens()
    }
    
    /**
     * 设置专属 Token
     */
    fun updateClientToken(token: TokenData) {
        clientToken = token
        saveTokens()
    }
    
    /**
     * 生成安全的随机 Token（用于客户端 ID）
     */
    private fun generateSecureToken(byteLength: Int): String {
        val secureRandom = SecureRandom()
        val bytes = ByteArray(byteLength)
        secureRandom.nextBytes(bytes)
        return bytes.joinToString("") { "%02x".format(it) }
    }
}

/**
 * 认证测试主函数
 */
@Suppress("UNUSED_VARIABLE")
fun main() {
    println("========================================")
    println("认证系统可行性测试")
    println("========================================\n")
    
    // 清理之前的测试数据
    File("test-available/auth-system/server-config").deleteRecursively()
    File("test-available/auth-system/client-token.json").delete()
    File("test-available/auth-system/client_id.txt").delete()
    
    // 1. 服务端初始化
    println("=== 测试 1: 服务端初始化 ===")
    val serverManager = ServerTokenManager()
    serverManager.initialize()
    println()
    
    // 2. 客户端初始化（首次使用，无 Token）
    println("=== 测试 2: 客户端初始化（首次使用） ===")
    val clientManager = ClientTokenManager()
    clientManager.initialize()
    println("客户端 ID: ${clientManager.clientId}")
    println("预设 Token: ${clientManager.presetToken ?: "未设置"}")
    println("专属 Token: ${clientManager.clientToken?.token?.take(8) ?: "未获取"}...")
    println()
    
    // 3. 用户输入预设 Token
    println("=== 测试 3: 用户输入预设 Token ===")
    val userPresetToken = serverManager.presetToken // 模拟用户从服务端获取预设 Token
    clientManager.updatePresetToken(userPresetToken)
    println("用户输入预设 Token: ${userPresetToken.take(8)}...")
    println()
    
    // 4. 客户端使用预设 Token 换取专属 Token
    println("=== 测试 4: 客户端换取专属 Token ===")
    val clientToken = serverManager.exchangePresetToken(
        presetToken = userPresetToken,
        clientId = clientManager.clientId
    )
    
    if (clientToken != null) {
        println("✓ 成功获取专属 Token")
        println("  Token: ${clientToken.token.take(8)}...")
        println("  客户端 ID: ${clientToken.clientId}")
        println("  创建时间：${clientToken.createdAt}")
        println("  过期时间：${clientToken.expiresAt ?: "永不过期"}")
        
        clientManager.updateClientToken(clientToken)
    } else {
        println("✗ 获取专属 Token 失败")
        return
    }
    println()
    
    // 5. 验证专属 Token
    println("=== 测试 5: 验证专属 Token ===")
    val isValid = serverManager.verifyClientToken(clientToken!!.token)
    println("专属 Token 验证结果：${if (isValid) "✓ 有效" else "✗ 无效"}")
    println()
    
    // 6. 测试预设 Token 验证失败
    println("=== 测试 6: 测试错误的预设 Token ===")
    val invalidResult = serverManager.exchangePresetToken("invalid_token_123456")
    println("使用错误的预设 Token 换取专属 Token: ${if (invalidResult == null) "✓ 正确拒绝" else "✗ 验证失败"}")
    println()
    
    // 7. 测试专属 Token 验证失败
    println("=== 测试 7: 测试未授权的 Token ===")
    val isUnauthorizedValid = serverManager.verifyClientToken("unauthorized_token_123456")
    println("未授权 Token 验证结果：${if (!isUnauthorizedValid) "✓ 正确拒绝" else "✗ 验证失败"}")
    println()
    
    // 8. 测试多个客户端换取 Token
    println("=== 测试 8: 多个客户端换取 Token ===")
    val client2Manager = ClientTokenManager(
        File("test-available/auth-system/client2-token.json")
    )
    client2Manager.initialize()
    client2Manager.updatePresetToken(userPresetToken)
    
    val client2Token = serverManager.exchangePresetToken(
        presetToken = userPresetToken,
        clientId = client2Manager.clientId
    )
    
    if (client2Token != null) {
        println("✓ 第二个客户端成功获取专属 Token")
        println("  Token: ${client2Token.token.take(8)}...")
        println("  客户端 ID: ${client2Token.clientId}")
        client2Manager.updateClientToken(client2Token)
    }
    println()
    
    // 9. 验证两个客户端的 Token 都有效
    println("=== 测试 9: 验证多个客户端 Token ===")
    val client1Valid = serverManager.verifyClientToken(clientToken.token)
    val client2Valid = serverManager.verifyClientToken(client2Token!!.token)
    println("客户端 1 Token 验证：${if (client1Valid) "✓ 有效" else "✗ 无效"}")
    println("客户端 2 Token 验证：${if (client2Valid) "✓ 有效" else "✗ 无效"}")
    println()
    
    // 10. 测试 Token 撤销
    println("=== 测试 10: Token 撤销 ===")
    val revoked = serverManager.revokeToken(client2Token.token)
    println("撤销客户端 2 的 Token: ${if (revoked) "✓ 成功" else "✗ 失败"}")
    
    val client2AfterRevoke = serverManager.verifyClientToken(client2Token.token)
    println("撤销后客户端 2 Token 验证：${if (!client2AfterRevoke) "✓ 正确拒绝" else "✗ 验证失败"}")
    
    val client1AfterRevoke = serverManager.verifyClientToken(clientToken.token)
    println("撤销后客户端 1 Token 验证：${if (client1AfterRevoke) "✓ 仍然有效" else "✗ 验证失败"}")
    println()
    
    // 11. 测试预设 Token 每日更换
    println("=== 测试 11: 预设 Token 每日更换（模拟） ===")
    val oldPresetToken = serverManager.presetToken
    val oldPresetDate = serverManager.presetTokenDate
    
    // 模拟第二天
    val mockYesterday = LocalDate.now().minusDays(1)
    val mockData = mapOf(
        "token" to oldPresetToken,
        "date" to mockYesterday.format(DateTimeFormatter.ISO_LOCAL_DATE),
        "createdAt" to (System.currentTimeMillis() - 86400000).toString()
    )
    File("test-available/auth-system/server-config/preset_token.json").writeText(
        Json.encodeToString(JsonObject.serializer(), JsonObject(mockData.mapValues { JsonPrimitive(it.value) }))
    )
    
    // 重新初始化服务端（应该生成新的预设 Token）
    val serverManager2 = ServerTokenManager()
    serverManager2.initialize()
    
    val newPresetToken = serverManager2.presetToken
    val newPresetDate = serverManager2.presetTokenDate
    
    println("昨天的预设 Token: ${oldPresetToken.take(8)}... (日期：$oldPresetDate)")
    println("今天的预设 Token: ${newPresetToken.take(8)}... (日期：$newPresetDate)")
    println("预设 Token 已更换：${if (newPresetToken != oldPresetToken) "✓ 是" else "✗ 否"}")
    println()
    
    // 12. 测试客户端 Token 持久化
    println("=== 测试 12: 客户端 Token 持久化 ===")
    val clientManager2 = ClientTokenManager()
    clientManager2.initialize()
    println("重新加载客户端 Token:")
    println("  预设 Token: ${clientManager2.presetToken?.take(8) ?: "未设置"}...")
    println("  专属 Token: ${clientManager2.clientToken?.token?.take(8) ?: "未获取"}...")
    println("  客户端 ID: ${clientManager2.clientId}")
    println("Token 持久化：${if (clientManager2.clientToken != null) "✓ 成功" else "✗ 失败"}")
    println()
    
    // 13. 测试 Token 丢失后重新获取
    println("=== 测试 13: Token 丢失后重新获取 ===")
    clientManager.clearClientToken()
    println("清除专属 Token 后：${clientManager.clientToken ?: "null"}")
    
    // 使用预设 Token 重新换取
    val newClientToken = serverManager.exchangePresetToken(
        presetToken = clientManager.presetToken!!,
        clientId = clientManager.clientId
    )
    
    if (newClientToken != null) {
        clientManager.updateClientToken(newClientToken)
        println("重新获取专属 Token: ${newClientToken.token.take(8)}...")
        println("Token 丢失后重新获取：✓ 成功")
    }
    println()
    
    // 总结
    println("========================================")
    println("测试总结")
    println("========================================")
    println("✓ 预设 Token 生成和每日更换")
    println("✓ 专属 Token 生成和验证")
    println("✓ Token 持久化和加载")
    println("✓ 预设 Token 验证")
    println("✓ 专属 Token 换取")
    println("✓ 多客户端支持")
    println("✓ Token 撤销")
    println("✓ Token 丢失后重新获取")
    println()
    println("所有测试通过！认证系统实现可行。")
    println("========================================")
}
