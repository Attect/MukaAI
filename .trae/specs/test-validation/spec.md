# Agent Plus 测试验证规格文档

## Why

Agent Plus系统已完成基础开发，现在需要通过执行复杂的开发题目来验证系统的实际工作效果。测试过程中需要：
- 实时监督Agent行为
- 发现问题即时打断调优
- 迭代改进直至效果完美

## What Changes

### 测试题目执行
- 执行Kotlin/Java/JavaScript测试题目
- 监督运行过程
- 记录问题和解决方案
- 迭代调优

### 调优内容
- 提示词优化
- 程序逻辑审查规则优化
- 工具调用策略优化

## Impact

- Affected specs: agent-plus
- Affected code:
  - `internal/agent/prompts.go` - 提示词
  - `internal/agent/reviewer.go` - 审查规则
  - `internal/agent/core.go` - 主循环逻辑

## ADDED Requirements

### Requirement: 测试执行器

系统应提供测试执行器来运行测试题目。

#### Scenario: 执行Kotlin测试
- **WHEN** 选择Kotlin测试题目
- **THEN** 系统创建工作目录，执行任务，收集结果

#### Scenario: 执行Java测试
- **WHEN** 选择Java测试题目
- **THEN** 系统创建工作目录，执行任务，收集结果

#### Scenario: 执行JavaScript测试
- **WHEN** 选择JavaScript测试题目
- **THEN** 系统创建工作目录，执行任务，收集结果

### Requirement: 监督测试过程

系统应在测试执行过程中实时监督Agent行为。

#### Scenario: 方向偏离检测
- **WHEN** Agent偏离任务目标
- **THEN** 监督系统检测并注入纠正提示

#### Scenario: 错误模式检测
- **WHEN** Agent陷入循环或重复失败
- **THEN** 监督系统检测并干预

#### Scenario: 编造内容检测
- **WHEN** Agent声称完成但实际未完成
- **THEN** 监督系统验证并要求实际执行

### Requirement: 问题修复与迭代

系统应能根据测试结果修复问题并迭代优化。

#### Scenario: 提示词优化
- **WHEN** 发现提示词不够清晰或引导错误
- **THEN** 更新提示词并重新测试

#### Scenario: 审查规则优化
- **WHEN** 发现审查规则误报或漏报
- **THEN** 更新审查规则并重新测试

#### Scenario: 工具策略优化
- **WHEN** 发现工具调用策略不合理
- **THEN** 优化工具调用逻辑并重新测试

### Requirement: 测试验收标准

测试题目应满足以下验收标准。

#### Scenario: Kotlin LRU缓存测试
- **WHEN** 执行Kotlin Task 1
- **THEN** 
  - 正确实现LRU缓存接口
  - 线程安全验证通过
  - 性能测试通过（10000次操作<100ms）
  - 代码质量符合规范

#### Scenario: Java工作流引擎测试
- **WHEN** 执行Java Task 1
- **THEN**
  - 正确实现工作流引擎接口
  - 条件分支正确判断
  - 并行任务正确执行
  - 状态持久化正常工作

#### Scenario: JavaScript任务调度器测试
- **WHEN** 执行JavaScript Task 1
- **THEN**
  - 任务按调度正确执行
  - 优先级正确生效
  - 任务取消正常工作
  - 重试机制正确实现

## 测试流程

```
1. 选择测试题目
2. 创建测试工作目录
3. 启动Agent Plus执行任务
4. 监督系统实时监控
   ├─ 方向偏离检测
   ├─ 错误模式检测
   └─ 编造内容检测
5. 发现问题时：
   ├─ 打断当前执行
   ├─ 分析问题原因
   ├─ 优化提示词/规则/逻辑
   └─ 重新测试
6. 验收测试结果
7. 记录问题和解决方案
```

## 调优策略

### 提示词调优
- 增加任务目标明确性
- 添加验收标准提示
- 强调工具使用规范

### 审查规则调优
- 调整检测阈值
- 增加特定错误模式
- 优化干预策略

### 工具调用调优
- 优化工具描述
- 增加工具使用示例
- 改进参数验证
