# 用户口述功能需求

## 2026-03-09: 初始需求描述

### 用户需求概述
用户希望创建一个 Kotlin Multiplatform 项目，具有以下特征:

### 核心功能需求
1. **原生服务端**: 使用 Kotlin native 开发的服务端
2. **多平台客户端**: 配备 desktop/android/ios/web(wasm) 的交互界面客户端
3. **参考项目**: 参考 nanobot 和 openclaw 实现一个多工作区的 AI 助手工具
4. **Skill 系统**: 支持安装或创建 skill
5. **命令行能力**: 具备 shell/cmd 命令行执行能力
6. **浏览器控制**: 具备浏览器控制能力
7. **多模态会话**: 会话支持文本和图片
8. **AI 模型集成**: 自动发现和使用 LM Studio 提供的 AI 模型

### 技术约束
- 使用 Kotlin Multiplatform 技术栈
- 服务端使用 Kotlin native
- 使用 Ktor 作为网络框架
- 使用 Jetpack Compose 进行 UI 开发 (桌面端)

### 可行性测试要求
- 在开始项目开发前，先进行可行性测试
- 测试存放于 `test-available/` 目录
- 每个测试包含 README.md 说明和 test.kts 测试脚本

### 相关资源
- 参考项目已在 `references/` 目录:
  - `references/nanobot/` - nanobot 项目源码和分析
  - `references/openclaw/` - openclaw 项目源码和文档
- Gradle 配置已在 `gradle/` 目录

### 记录时间
2026-03-09

### 状态
✅ 已完成可行性测试
- 7 项关键技术测试全部通过
- 详见 `test-available/index.md`

## 2026-03-10: AI 记忆系统集成需求

### 用户需求概述
用户希望项目使用 mem0 作为 AI 会话的记忆处理系统。由于 mem0 没有 Kotlin/JVM 实现，需要将它构建为独立的可执行程序，通过 REST API 进行调用。

### 技术需求
1. **Mem0 部署方式**: 构建为独立可执行文件（非 Docker）
2. **运行方式**: 跟随当前项目启动后运行
3. **通信方式**: 使用 REST API 进行调用
4. **LLM 集成**: 使用 LM Studio 提供的 `qwen3.5-9b-uncensored-hauhaucs-aggressive` 模型
5. **Embedding**: 使用 LM Studio 提供的 `nomic-embed-text-v1.5` 模型

### 技术约束
- 禁止使用 Docker 部署
- 禁止安装 ollama（与 LM Studio 重复）
- 使用 mem0 官方的 LM Studio 支持
- 可执行文件应独立运行，无需 Python 环境

### 可行性测试
- 测试位置：`test-available/mem0-integration/`
- 测试内容：
  - Mem0 本地部署验证
  - LM Studio 集成验证
  - REST API 功能测试
  - PyInstaller 打包验证

### 状态
✅ 已完成可行性验证
- Mem0 成功本地部署
- LM Studio 集成成功（LLM + Embedding）
- REST API 测试通过（创建、获取、搜索记忆）
- PyInstaller 打包成功（生成独立 exe，约 50MB）
- 详见 `test-available/mem0-integration/verification-report.md`

### 记录时间
2026-03-10
