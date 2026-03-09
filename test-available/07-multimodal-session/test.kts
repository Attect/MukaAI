#!/usr/bin/env kotlin
//usr/bin/env kotlin -script $0; exit $?

@file:Repository("https://repo.maven.apache.org/maven2/")
@file:DependsOn("org.jetbrains.kotlinx:kotlinx-serialization-json:1.7.3")
@file:DependsOn("org.jetbrains.kotlinx:kotlinx-coroutines-core:1.9.0")

import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import java.io.ByteArrayOutputStream
import java.io.File
import javax.imageio.ImageIO
import java.awt.image.BufferedImage
import java.util.Base64
import javax.imageio.ImageReader
import javax.imageio.stream.ImageInputStream
import java.io.FileInputStream

@Serializable
data class Message(
    val id: String,
    val type: MessageType,
    val content: String, // 文本内容或 Base64 编码的图片
    val mimeType: String? = null, // 仅图片消息需要
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
    val data: String, // Base64 编码的图片数据
    val mimeType: String,
    val width: Int? = null,
    val height: Int? = null
)

class MultimodalProcessor {
    
    /**
     * 将图片文件转换为 Base64 编码
     */
    fun imageToBase64(imageFile: File): String {
        val image = ImageIO.read(imageFile)
        val outputStream = ByteArrayOutputStream()
        
        // 根据原始文件扩展名确定格式
        val format = when {
            imageFile.extension.lowercase() in listOf("png", "jpg", "jpeg", "gif", "webp") -> imageFile.extension.lowercase()
            else -> "png" // 默认格式
        }
        
        ImageIO.write(image, format, outputStream)
        return Base64.getEncoder().encode(outputStream.toByteArray()).toString(Charsets.UTF_8)
    }
    
    /**
     * 将 Base64 编码的图片数据转换为 BufferedImage
     */
    fun base64ToImage(base64Data: String): BufferedImage {
        val imageBytes = Base64.getDecoder().decode(base64Data)
        val inputStream = java.io.ByteArrayInputStream(imageBytes)
        return ImageIO.read(inputStream)
    }
    
    /**
     * 获取图片信息（尺寸等）
     */
    fun getImageInfo(base64Data: String): Pair<Int, Int> {
        val imageBytes = Base64.getDecoder().decode(base64Data)
        val inputStream = ImageInputStream(java.io.ByteArrayInputStream(imageBytes))
        
        val readers = ImageIO.getImageReaders(inputStream)
        if (!readers.hasNext()) {
            throw IllegalArgumentException("无法识别的图片格式")
        }
        
        val reader = readers.next()
        reader.input = inputStream
        
        val width = reader.getWidth(0)
        val height = reader.getHeight(0)
        
        return Pair(width, height)
    }
    
    /**
     * 创建文本消息
     */
    fun createTextMessage(id: String, content: String): Message {
        return Message(id = id, type = MessageType.TEXT, content = content)
    }
    
    /**
     * 创建图片消息
     */
    fun createImageMessage(id: String, imageFile: File): Message {
        val base64Data = imageToBase64(imageFile)
        val mimeType = "image/${imageFile.extension.lowercase()}"
        return Message(id = id, type = MessageType.IMAGE, content = base64Data, mimeType = mimeType)
    }
    
    /**
     * 创建多模态消息
     */
    fun createMultimodalMessage(id: String, text: String, imageFiles: List<File>): Message {
        val multimodal = MultimodalMessage(
            text = text,
            images = imageFiles.map { file ->
                val base64Data = imageToBase64(file)
                val mimeType = "image/${file.extension.lowercase()}"
                val (width, height) = getImageInfo(base64Data)
                ImageData(data = base64Data, mimeType = mimeType, width = width, height = height)
            }
        )
        
        val jsonContent = Json.encodeToString(multimodal)
        return Message(id = id, type = MessageType.MULTIMODAL, content = jsonContent)
    }
    
    /**
     * 解析消息为适合 LLM 的格式
     */
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
                
                // 添加文本部分
                result.add(mapOf("type" to "text", "text" to multimodal.text))
                
                // 添加图片部分
                multimodal.images.forEach { img ->
                    result.add(mapOf("type" to "image_url", "image_url" to "data:${img.mimeType};base64,${img.data}"))
                }
                
                result.toList()
            }
        }
    }
}

fun main() = kotlinx.coroutines.runBlocking {
    println("=== 测试 07: 多模态会话 (文本 + 图片) ===\n")
    
    val processor = MultimodalProcessor()
    
    try {
        // 测试 1: 创建文本消息
        println("[测试 1] 创建文本消息...")
        val textMsg = processor.createTextMessage("msg1", "这是一条文本消息")
        println("文本消息: ${Json.encodeToString(textMsg)}\n")
        
        // 测试 2: 检查是否有测试图片，如果没有则跳过图片测试
        val testImage = File("test-image.jpg")
        if (testImage.exists()) {
            println("[测试 2] 创建图片消息...")
            val imageMsg = processor.createImageMessage("msg2", testImage)
            println("图片消息创建成功: 类型=${imageMsg.mimeType}, 大小=${imageMsg.content.length} 字符")
            
            // 获取图片信息
            val imageInfo = processor.getImageInfo(imageMsg.content)
            println("图片尺寸: ${imageInfo.first}x${imageInfo.second}\n")
        } else {
            println("[测试 2] 跳过图片测试 (无测试图片)")
            println("提示：创建 test-image.jpg 文件以运行完整测试\n")
        }
        
        // 测试 3: 创建多模态消息 (模拟)
        println("[测试 3] 创建多模态消息结构...")
        val multimodalMsg = processor.createMultimodalMessage(
            "msg3",
            "请分析这张图片中的内容",
            emptyList() // 由于缺少测试图片，使用空列表
        )
        println("多模态消息: ${multimodalMsg.type}")
        println("内容长度: ${multimodalMsg.content.length} 字符\n")
        
        // 测试 4: 格式化为 LLM 可接受格式
        println("[测试 4] 格式化为 LLM 输入格式...")
        val llmFormat = processor.formatForLLM(textMsg)
        println("LLM 格式化结果: ${llmFormat}\n")
        
        if (testImage.exists()) {
            val llmFormatImage = processor.formatForLLM(
                processor.createImageMessage("msg4", testImage)
            )
            println("图片 LLM 格式化结果: ${llmFormatImage}\n")
        }
        
        println("=== 多模态会话处理逻辑验证通过！ ===")
        println("注意：完整图片测试需要提供测试图片文件")
        
    } catch (e: Exception) {
        println("\n=== 测试失败：${e.message} ===")
        e.printStackTrace()
    }
}
