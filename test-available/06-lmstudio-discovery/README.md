# 可行性测试 06: LM Studio 模型自动发现

## 测试目的
验证自动发现和使用 LM Studio 提供的 AI 模型的能力。

## 测试内容
1. LM Studio REST API 连接测试
2. 模型列表获取
3. 模型信息解析
4. 模型自动切换
5. 聊天补全测试
6. 多轮对话测试

## LM Studio API 配置
- 基础地址：`http://127.0.0.1:11452`
- API Key: `sk-lm-ocCyCxvA:LFkKFpG7abUUr3kwp1ek`
- 模型列表端点：`GET /api/v1/models`
- 聊天补全端点：`POST /v1/chat/completions`
- 使用 Bearer Token 认证
- 注意：LM Studio 的模型列表返回格式与标准 OpenAI API 略有不同

## 项目结构
```
06-lmstudio-discovery/
├── src/
│   └── commonMain/
│       └── kotlin/
│           └── LMStudioTest.kt      # 测试源代码
├── build.gradle.kts                  # Gradle 构建配置
├── settings.gradle.kts               # Gradle 设置
├── gradle/                           # Gradle Wrapper
├── gradlew                           # Gradle Wrapper (Unix)
├── gradlew.bat                       # Gradle Wrapper (Windows)
└── README.md                         # 本说明文件
```

## 构建和运行

### 前置要求
- JDK 17 或更高版本
- LM Studio 服务已启动并加载模型

### 运行测试

#### Windows:
```bash
gradlew.bat run
```

#### Linux/macOS:
```bash
./gradlew run
```

### 预期输出
```
=== 测试 06: LM Studio 模型自动发现 ===

API 地址：http://127.0.0.1:11452/api/v1
认证方式：Bearer Token

[测试 1] 检查 LM Studio 服务可用性...
✓ LM Studio 服务可用

[测试 2] 获取可用模型列表...
发现 X 个模型:
  1. model-name-1
  2. model-name-2

[测试 3] 使用模型 'model-name-1' 进行聊天测试...

响应详情:
  响应 ID: xxx
  使用模型：model-name-1
  助手回复：你好！我是...
  Token 使用:
    提示词：XX
    完成：XX
    总计：XX

[测试 4] 多轮对话测试...
多轮对话回复：...

=== ✓ 所有测试通过！LM Studio 模型自动发现功能验证成功 ===
```

## 预期结果
- ✓ 能够自动发现本地运行的 LM Studio 服务
- ✓ 获取可用模型列表
- ✓ 使用模型进行聊天对话
- ✓ 支持多轮对话

## 自定义配置
如需修改 API 地址或密钥，请编辑 `src/commonMain/kotlin/LMStudioTest.kt` 中的 `LMStudioClient` 类:

```kotlin
class LMStudioClient(
    private val baseUrl: String = "http://127.0.0.1:11452/api/v1",
    private val apiKey: String = "sk-lm-ocCyCxvA:LFkKFpG7abUUr3kwp1ek"
)
```

## 参考
- [LM Studio REST API](../../references/lm-studio/lm-studio-rest-api.md)
- [Ktor Client Documentation](../../references/ktor/ktor-client.md)
