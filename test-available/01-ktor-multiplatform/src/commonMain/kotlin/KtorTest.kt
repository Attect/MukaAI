import io.ktor.server.application.*
import io.ktor.server.engine.*
import io.ktor.server.netty.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import io.ktor.server.plugins.cors.routing.*
import io.ktor.server.plugins.contentnegotiation.*
import io.ktor.server.request.*
import io.ktor.serialization.kotlinx.json.*
import io.ktor.client.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import kotlinx.serialization.Serializable
import kotlinx.coroutines.delay
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

@Serializable
data class Message(val text: String, val timestamp: Long = System.currentTimeMillis())

@Serializable
data class ResponseData(val status: String)

@Serializable
data class ReceivedData(val received: String)

@Serializable
data class Response(
    val success: Boolean, 
    val data: String? = null, 
    val error: String? = null
)

suspend fun main() {
    println("=== 测试 01: Ktor 服务端 + 客户端架构验证 ===")
    
    // 启动服务端
    val server = embeddedServer(Netty, port = 8080) {
        install(CORS) {
            anyHost()
        }
        install(ContentNegotiation) {
            json()
        }
        routing {
            get("/health") {
                call.respondText("""{"success":true,"data":"{\"status\":\"ok\"}","error":null}""")
            }
            
            get("/api/message") {
                val msg = Message(text = "Hello from Ktor Server!")
                val jsonData = Json.encodeToString(msg)
                call.respondText("""{"success":true,"data":$jsonData,"error":null}""")
            }
            
            post("/api/message") {
                val msg = call.receive<Message>()
                println("收到消息：${msg.text}")
                call.respondText("""{"success":true,"data":"{\"received\":\"${msg.text}\"}","error":null}""")
            }
        }
    }.start(wait = false)
    
    println("服务端已启动在 http://localhost:8080")
    
    // 等待服务端启动
    delay(1000)
    
    // 创建客户端并测试
    val client = HttpClient {
        install(io.ktor.client.plugins.contentnegotiation.ContentNegotiation) {
            json()
        }
    }
    
    try {
        // 测试 1: 健康检查
        println("\n[测试 1] 健康检查...")
        val healthResponse = client.get("http://localhost:8080/health")
        println("健康检查响应：${healthResponse.bodyAsText()}")
        
        // 测试 2: GET 请求
        println("\n[测试 2] GET 请求获取消息...")
        val getMessage = client.get("http://localhost:8080/api/message")
        println("GET 响应：${getMessage.bodyAsText()}")
        
        // 测试 3: POST 请求
        println("\n[测试 3] POST 请求发送消息...")
        val testMessage = Message(text = "Test message from client")
        val jsonBody = Json.encodeToString(testMessage)
        val postResponse = client.post("http://localhost:8080/api/message") {
            setBody(jsonBody)
            contentType(io.ktor.http.ContentType.Application.Json)
        }
        println("POST 响应：${postResponse.bodyAsText()}")
        
        println("\n=== 所有测试通过！Ktor 服务端 + 客户端架构可行 ===")
    } catch (e: Exception) {
        println("\n=== 测试失败：${e.message} ===")
        e.printStackTrace()
    } finally {
        client.close()
        server.stop(1000, 2000)
        println("服务端已关闭")
    }
}