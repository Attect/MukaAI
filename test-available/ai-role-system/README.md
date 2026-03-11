# AI 角色系统可行性测试

> 测试目的：验证 AI 角色和会话系统关键技术的可行性

**创建时间**: 2026-03-10  
**状态**: 进行中

## 测试目标

本测试目录用于验证以下关键技术：

1. **文件系统隔离**: 验证工作区目录结构的可行性
2. **疲劳值算法**: 验证时间衰减和增长逻辑
3. **实时打断**: 验证协程取消、工具中断、进程终止
4. **技能黑白名单**: 验证白名单优先、黑名单过滤
5. **模型验证**: 验证 100K 上下文、多模态、工具调用检测
6. **mem0 隔离/共享**: 验证不同角色的记忆隔离和群聊记忆共享

## 测试目录结构

```
test-available/ai-role-system/
├── README.md                    # 本文件
├── build.gradle.kts             # Gradle 构建配置
├── settings.gradle.kts          # Gradle 设置
├── gradle/
│   └── libs.versions.toml       # 版本目录
└── src/
    └── test/
        ├── FileSystemIsolationTest.kt    # 文件系统隔离测试
        ├── FatigueAlgorithmTest.kt       # 疲劳值算法测试
        ├── InterruptionMechanismTest.kt  # 实时打断测试
        ├── SkillWhitelistBlacklistTest.kt # 技能黑白名单测试
        └── ModelValidationTest.kt        # 模型验证测试
```

## 构建和运行

### 前置要求

- JDK 17 或更高版本
- Gradle 8.0 或更高版本
- Kotlin 2.3.10

### 构建项目

```bash
cd test-available/ai-role-system
./gradlew build
```

### 运行测试

```bash
# 运行所有测试
./gradlew test

# 运行单个测试类
./gradlew test --tests FileSystemIsolationTest
./gradlew test --tests FatigueAlgorithmTest
./gradlew test --tests InterruptionMechanismTest
./gradlew test --tests SkillWhitelistBlacklistTest
./gradlew test --tests ModelValidationTest
```

## 测试详情

### 1. 文件系统隔离测试

**目的**: 验证每个 AI 角色有独立的工作区目录

**测试内容**:
- 创建工作区目录结构
- 验证配置文件存储
- 验证技能目录隔离
- 验证会话历史存储

**预期结果**: ✅ 文件系统隔离可行

### 2. 疲劳值算法测试

**目的**: 验证疲劳值的时间衰减和增长逻辑

**测试内容**:
- 疲劳值增长计算
- 时间衰减计算
- 疲劳等级划分
- 持久化存储

**预期结果**: ✅ 疲劳值算法可行

### 3. 实时打断测试

**目的**: 验证协程取消、工具中断、进程终止

**测试内容**:
- 协程 Job 取消
- 工具调用中断
- 子进程终止
- 消息撤回机制

**预期结果**: ✅ 实时打断可行

### 4. 技能黑白名单测试

**目的**: 验证白名单优先、黑名单过滤机制

**测试内容**:
- 白名单逻辑
- 黑名单逻辑
- 白名单 + 黑名单组合
- 技能调用拦截

**预期结果**: ✅ 技能黑白名单可行

### 5. 模型验证测试

**目的**: 验证模型选择限制（100K 上下文、多模态、工具调用）

**测试内容**:
- LM Studio API 查询
- 模型能力验证
- 过滤不满足要求的模型

**预期结果**: ✅ 模型验证可行

## 测试结果

详见 [verification-report.md](./verification-report.md)

## 参考资料

- [设计需求文档](../../docs/design_by_user_say.md)
- [用户口述需求](../../docs/user_say.md)
- [Ktor 官方文档](https://ktor.io/)
- [Kotlin 协程文档](https://kotlinlang.org/docs/coroutines-overview.html)
