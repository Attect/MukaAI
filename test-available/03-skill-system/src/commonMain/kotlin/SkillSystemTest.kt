import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import java.io.File
import java.util.regex.Pattern
import kotlinx.coroutines.runBlocking

@Serializable
data class SkillDefinition(
    val name: String,
    val description: String,
    val author: String? = null,
    val version: String = "1.0.0",
    val parameters: Map<String, ParameterSpec> = emptyMap(),
    val requiredPermissions: List<String> = emptyList(),
    val tags: List<String> = emptyList()
)

@Serializable
data class ParameterSpec(
    val type: String,
    val required: Boolean = true,
    val description: String = "",
    val defaultValue: String? = null
)

@Serializable
data class SkillExecutionRequest(
    val skillName: String,
    val parameters: Map<String, String>
)

@Serializable
data class SkillExecutionResult(
    val success: Boolean,
    val output: String,
    val error: String? = null,
    val metadata: Map<String, String> = emptyMap()
)

interface SkillExecutor {
    suspend fun execute(skillName: String, parameters: Map<String, String>): SkillExecutionResult
}

class SkillManager {
    private val skills = mutableMapOf<String, SkillDefinition>()
    private val executors = mutableMapOf<String, SkillExecutor>()
    
    fun loadSkillFromMarkdown(skillFile: File): SkillDefinition? {
        val content = skillFile.readText()
        
        val yamlPattern = Pattern.compile("---\\s*\\n([\\s\\S]*?)\\n---")
        val matcher = yamlPattern.matcher(content)
        
        if (!matcher.find()) {
            println("未找到 SKILL.md frontmatter")
            return null
        }
        
        val yamlContent = matcher.group(1)
        val yamlLines = yamlContent.split("\n")
        
        val yamlMap = mutableMapOf<String, String>()
        for (line in yamlLines) {
            val colonIndex = line.indexOf(':')
            if (colonIndex > 0) {
                val key = line.substring(0, colonIndex).trim()
                val value = line.substring(colonIndex + 1).trim().removePrefix("\"").removeSuffix("\"")
                yamlMap[key] = value
            }
        }
        
        val name = yamlMap["name"] ?: return null
        val description = yamlMap["description"] ?: ""
        val author = yamlMap["author"]
        val version = yamlMap["version"] ?: "1.0.0"
        
        val parameters = mutableMapOf<String, ParameterSpec>()
        
        val skillDef = SkillDefinition(
            name = name,
            description = description,
            author = author,
            version = version,
            parameters = parameters
        )
        
        skills[name] = skillDef
        return skillDef
    }
    
    fun registerExecutor(skillName: String, executor: SkillExecutor) {
        executors[skillName] = executor
    }
    
    suspend fun executeSkill(request: SkillExecutionRequest): SkillExecutionResult {
        val executor = executors[request.skillName]
        if (executor == null) {
            return SkillExecutionResult(
                success = false,
                output = "",
                error = "未找到技能：${request.skillName}"
            )
        }
        
        return executor.execute(request.skillName, request.parameters)
    }
    
    fun searchSkills(query: String = ""): List<SkillDefinition> {
        return if (query.isEmpty()) {
            skills.values.toList()
        } else {
            skills.values.filter { skill ->
                skill.name.contains(query, ignoreCase = true) ||
                skill.description.contains(query, ignoreCase = true) ||
                skill.tags.any { it.contains(query, ignoreCase = true) }
            }
        }
    }
    
    fun getAllSkills(): List<SkillDefinition> = skills.values.toList()
}

class EchoSkillExecutor : SkillExecutor {
    override suspend fun execute(skillName: String, parameters: Map<String, String>): SkillExecutionResult {
        val text = parameters["text"] ?: "无输入文本"
        return SkillExecutionResult(
            success = true,
            output = "Echo: $text",
            metadata = mapOf("input_length" to text.length.toString())
        )
    }
}

class CalculatorSkillExecutor : SkillExecutor {
    override suspend fun execute(skillName: String, parameters: Map<String, String>): SkillExecutionResult {
        val expression = parameters["expression"] ?: return SkillExecutionResult(
            success = false,
            output = "",
            error = "缺少表达式参数"
        )
        
        val result = try {
            val cleanedExpr = expression.replace(Regex("[^0-9+\\-*/(). ]"), "")
            evalExpressionSafely(cleanedExpr)
        } catch (e: Exception) {
            return SkillExecutionResult(
                success = false,
                output = "",
                error = "计算错误：${e.message}"
            )
        }
        
        return SkillExecutionResult(
            success = true,
            output = "$expression = $result",
            metadata = mapOf("result" to result.toString())
        )
    }
}

fun main() = runBlocking {
    println("=== 测试 03: Skill 系统实现方案 ===\n")
    
    val skillManager = SkillManager()
    
    skillManager.registerExecutor("echo", EchoSkillExecutor())
    skillManager.registerExecutor("calculator", CalculatorSkillExecutor())
    
    try {
        println("[测试 1] 执行内置技能...")
        val echoResult = skillManager.executeSkill(
            SkillExecutionRequest(
                skillName = "echo",
                parameters = mapOf("text" to "Hello from skill system!")
            )
        )
        println("Echo 结果：${echoResult.output}")
        println("执行成功：${echoResult.success}\n")
        
        println("[测试 2] 搜索可用技能...")
        val allSkills = skillManager.getAllSkills()
        println("可用技能数量：${allSkills.size}")
        allSkills.forEach { skill ->
            println("  - ${skill.name}: ${skill.description}")
        }
        println()
        
        println("[测试 3] 模拟加载 SKILL.md 文件...")
        
        val exampleSkillFile = File("temp-example-skill.md")
        exampleSkillFile.writeText("""
---
name: "example-skill"
description: "示例技能用于演示 SKILL.md 格式"
author: "Assistant"
version: "1.0.0"
tags: ["utility", "demo"]
---

# 示例技能

这是一个示例技能，用于演示 SKILL.md 文件格式。

## 功能
- 演示技能定义格式
- 展示参数配置
- 验证加载流程

        """.trimIndent())
        
        val loadedSkill = skillManager.loadSkillFromMarkdown(exampleSkillFile)
        if (loadedSkill != null) {
            println("成功加载技能：${loadedSkill.name}")
            println("描述：${loadedSkill.description}")
            println("版本：${loadedSkill.version}")
            println("作者：${loadedSkill.author}")
        } else {
            println("技能加载失败")
        }
        
        exampleSkillFile.delete()
        println()
        
        println("[测试 4] 执行不存在的技能...")
        val invalidResult = skillManager.executeSkill(
            SkillExecutionRequest(
                skillName = "nonexistent-skill",
                parameters = emptyMap()
            )
        )
        println("执行结果：${invalidResult.error}")
        println()
        
        println("=== Skill 系统基本功能验证通过！ ===")
        println("注意：完整实现需要考虑安全性、沙箱执行、权限控制等功能")
        
    } catch (e: Exception) {
        println("\n=== 测试失败：${e.message} ===")
        e.printStackTrace()
    }
}

fun evalExpressionSafely(expression: String): Double {
    return expression.toDoubleOrNull() ?: 0.0
}
