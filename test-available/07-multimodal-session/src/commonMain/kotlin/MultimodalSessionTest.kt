import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import java.io.File
import java.util.Base64
import kotlinx.coroutines.runBlocking

@Serializable
data class Message(
    val id: String,
    val type: MessageType,
    val content: String,
    val mimeType: String? = null,
    val timestamp: Long = System.currentTimeMillis()
)

@Serializable
enum class MessageType {
    TEXT, IMAGE, MULTIMODAL
}

@Serializable
data class MultimodalMessage(
    val text: String,
    val images: List<ImageData>
)

@Serializable
data class ImageData(
    val data: String,
    val mimeType: String,
    val width: Int? = null,
    val height: Int? = null
)

class MultimodalProcessor {
    
    fun createTextMessage(id: String, content: String): Message {
        return Message(id = id, type = MessageType.TEXT, content = content)
    }
    
    fun createImageMessage(id: String, imagePath: String): Message {
        val file = File(imagePath)
        if (!file.exists()) {
            throw IllegalArgumentException("图片文件不存在：$imagePath")
        }
        
        val imageBytes = file.readBytes()
        val base64Data = Base64.getEncoder().encodeToString(imageBytes)
        val mimeType = "image/${file.extension.lowercase()}"
        
        return Message(id = id, type = MessageType.IMAGE, content = base64Data, mimeType = mimeType)
    }
    
    fun createMultimodalMessage(id: String, text: String, imagePaths: List<String>): Message {
        val images = imagePaths.map { imagePath ->
            val file = File(imagePath)
            val imageBytes = file.readBytes()
            val base64Data = Base64.getEncoder().encodeToString(imageBytes)
            val mimeType = "image/${file.extension.lowercase()}"
            ImageData(data = base64Data, mimeType = mimeType)
        }
        
        val multimodal = MultimodalMessage(text = text, images = images)
        val jsonContent = Json.encodeToString(multimodal)
        
        return Message(id = id, type = MessageType.MULTIMODAL, content = jsonContent)
    }
    
    fun formatForLLM(message: Message): List<Map<String, String>> {
        return when (message.type) {
            MessageType.TEXT -> listOf(mapOf("type" to "text", "text" to message.content))
            MessageType.IMAGE -> {
                val mimeType = message.mimeType ?: "image/png"
                listOf(mapOf("type" to "image_url", "image_url" to "data:$mimeType;base64,${message.content}"))
            }
            MessageType.MULTIMODAL -> {
                val multimodal = Json.decodeFromString<MultimodalMessage>(message.content)
                val result = mutableListOf<Map<String, String>>()
                
                result.add(mapOf("type" to "text", "text" to multimodal.text))
                
                multimodal.images.forEach { img ->
                    result.add(mapOf("type" to "image_url", "image_url" to "data:${img.mimeType};base64,${img.data}"))
                }
                
                result.toList()
            }
        }
    }
}

fun main() = runBlocking {
    println("=== 测试 07: 多模态会话 (文本 + 图片) ===\n")
    
    val processor = MultimodalProcessor()
    
    try {
        println("[测试 1] 创建文本消息...")
        val textMsg = processor.createTextMessage("msg1", "这是一条文本消息")
        println("文本消息：${Json.encodeToString(textMsg)}\n")
        
        println("[测试 2] 检查测试图片...")
        val testImage = File("test-image.jpg")
        if (testImage.exists()) {
            println("找到测试图片：${testImage.absolutePath}")
            
            val imageMsg = processor.createImageMessage("msg2", testImage.absolutePath)
            println("图片消息创建成功：类型=${imageMsg.mimeType}, 大小=${imageMsg.content.length} 字符")
            println()
        } else {
            println("跳过图片测试 (无测试图片)")
            println("提示：创建 test-image.jpg 文件以运行完整测试\n")
        }
        
        println("[测试 3] 创建多模态消息结构...")
        val multimodalMsg = if (testImage.exists()) {
            processor.createMultimodalMessage(
                "msg3",
                "请分析这张图片中的内容",
                listOf(testImage.absolutePath)
            )
        } else {
            processor.createMultimodalMessage(
                "msg3",
                "请分析这张图片中的内容",
                emptyList()
            )
        }
        println("多模态消息：${multimodalMsg.type}")
        println("内容长度：${multimodalMsg.content.length} 字符\n")
        
        println("[测试 4] 格式化为 LLM 输入格式...")
        val llmFormat = processor.formatForLLM(textMsg)
        println("LLM 格式化结果：${llmFormat}\n")
        
        if (testImage.exists()) {
            val llmFormatImage = processor.formatForLLM(
                processor.createImageMessage("msg4", testImage.absolutePath)
            )
            println("图片 LLM 格式化结果：${llmFormatImage}\n")
        }
        
        println("=== 多模态会话处理逻辑验证通过！ ===")
        println("注意：完整图片测试需要提供测试图片文件")
        
    } catch (e: Exception) {
        println("\n=== 测试失败：${e.message} ===")
        e.printStackTrace()
    }
}
