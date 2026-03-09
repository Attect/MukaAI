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
