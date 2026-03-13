# Unicode-Escaped Path 可行性测试

## 测试目的

验证使用 Unicode-escaped 编码方案来解决 AI 模型在输出中文和数字混合路径时插入额外空格的问题。

## 背景

AI 模型（特别是 qwen3.5-plus 和 Qwen3.5-397B-A17B）在处理中文和数字混合的路径时，会插入错误的空格，导致：
- 命令行路径错误
- 工具调用失败
- 技能调用失败

## 技术方案

参考 qwen-code 项目 PR #2300 的实现：
1. **编码策略**：使用 Unicode-escaped 格式（如 `\u4e2d\u6587`）来表示中文字符
2. **工具输出**：在工具返回结果时，暴露 unicode 映射关系
3. **调度器处理**：在工具调用调度器中，接收到参数后先进行 Unicode 解码，再进行验证和执行

## 测试内容

### 测试 1: Unicode 编码/解码工具函数
- 测试 Kotlin 中字符串与 Unicode-escaped 格式的互相转换
- 验证编码和解码的正确性
- 测试包含中文、数字、特殊字符的混合路径

### 测试 2: 工具调用协议验证
- 模拟 AI 模型输出包含 Unicode-escaped 路径的工具调用
- 验证调度器能正确解码并执行
- 验证路径解码后能正确访问文件系统

### 测试 3: AI 模型理解能力验证
- 测试 AI 模型是否能理解并正确使用 Unicode-escaped 格式
- 验证提示词设计是否有效

## 构建和运行

```bash
# 使用 Kotlin 脚本运行测试
kotlinc -script test.kts
```

## 预期结果

1. ✅ Unicode 编码/解码函数工作正常
2. ✅ 包含中文的路径能正确编码和解码
3. ✅ 工具调用协议能处理 Unicode-escaped 参数
4. ✅ 解码后的路径能正确访问文件系统

## 参考

- qwen-code PR #2300: https://github.com/QwenLM/qwen-code/pull/2300
