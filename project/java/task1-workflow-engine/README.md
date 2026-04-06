# Task 1: 工作流引擎

## 元数据
- **标题**: 工作流引擎实现
- **作者**: 开发团队
- **日期**: 2026-04-06
- **版本**: 1.0.0

## 题目描述

实现一个可配置的工作流引擎，用于管理和执行复杂的业务流程。该引擎需要支持多种任务节点类型、条件分支、并行执行，并能够持久化工作流状态以便在系统重启后恢复执行。

## 功能需求

### 1. 核心功能

#### 1.1 工作流定义
- 支持通过代码API或配置文件定义工作流
- 工作流包含多个节点，节点之间通过连线形成有向图
- 支持工作流版本管理

#### 1.2 节点类型
- **任务节点（Task Node）**: 执行具体的业务逻辑
- **条件节点（Condition Node）**: 根据条件判断选择执行路径
- **并行节点（Parallel Node）**: 并发执行多个分支
- **汇聚节点（Join Node）**: 等待多个分支完成后再继续
- **开始节点（Start Node）**: 工作流入口
- **结束节点（End Node）**: 工作流出口

#### 1.3 执行控制
- 支持同步和异步执行
- 支持任务超时设置（超时后自动重试或失败）
- 支持任务重试机制（可配置重试次数和间隔）
- 支持手动暂停和恢复工作流

#### 1.4 状态管理
- 支持工作流实例的状态持久化
- 支持从持久化存储中恢复工作流执行
- 支持查询工作流执行历史

### 2. 技术要求

#### 2.1 设计模式应用
- **状态模式**: 管理工作流和节点的不同状态
- **策略模式**: 不同节点类型的执行策略
- **责任链模式**: 节点之间的执行流转
- **命令模式**: 任务执行的封装

#### 2.2 并发处理
- 使用Java并发包（java.util.concurrent）
- 合理使用线程池管理并行任务
- 确保线程安全

#### 2.3 持久化
- 提供持久化接口（PersistenceService）
- 默认实现基于文件系统或内存存储
- 支持扩展为数据库存储

## API设计示例

```java
// 工作流定义
Workflow workflow = WorkflowBuilder.create("order-process")
    .startNode("start")
    .taskNode("validate-order", new ValidateOrderTask())
        .timeout(5000, TimeUnit.MILLISECONDS)
        .retry(3, 1000)
    .conditionNode("check-inventory")
        .when("available", "ship-order")
        .when("unavailable", "backorder")
    .parallelNode("ship-order")
        .branch("send-email", new SendEmailTask())
        .branch("update-inventory", new UpdateInventoryTask())
    .joinNode("join-shipping")
    .taskNode("complete-order", new CompleteOrderTask())
    .endNode("end")
    .build();

// 执行工作流
WorkflowEngine engine = new WorkflowEngine();
WorkflowInstance instance = engine.start(workflow, context);

// 查询状态
WorkflowStatus status = instance.getStatus();
List<NodeExecution> history = instance.getExecutionHistory();

// 恢复执行
engine.resume(instance.getInstanceId());
```

## 验收标准

### 功能验收

| 序号 | 验收项 | 验收标准 | 优先级 |
|------|--------|----------|--------|
| 1 | 工作流定义 | 能够通过API正确定义包含多种节点的工作流 | P0 |
| 2 | 任务节点执行 | 任务节点能够正确执行并返回结果 | P0 |
| 3 | 条件分支 | 条件节点能够根据条件正确选择执行路径 | P0 |
| 4 | 并行执行 | 并行节点能够并发执行多个分支 | P0 |
| 5 | 汇聚等待 | 汇聚节点能够正确等待所有分支完成 | P0 |
| 6 | 状态持久化 | 工作流状态能够持久化到存储 | P0 |
| 7 | 状态恢复 | 能够从持久化存储恢复工作流继续执行 | P0 |
| 8 | 任务超时 | 任务超时后能够正确处理（重试或失败） | P1 |
| 9 | 任务重试 | 任务失败后能够按配置重试 | P1 |
| 10 | 执行历史 | 能够查询工作流执行历史记录 | P1 |

### 质量验收

| 序号 | 验收项 | 验收标准 |
|------|--------|----------|
| 1 | 单元测试 | 测试覆盖率 ≥ 80% |
| 2 | 并发安全 | 并行执行无死锁、无竞态条件 |
| 3 | 性能要求 | 单节点执行延迟 < 10ms |
| 4 | 代码规范 | 通过阿里巴巴Java开发规范检查 |
| 5 | 文档完整 | 包含设计文档、API文档、使用示例 |

## 测试用例

### 测试场景1: 简单顺序工作流
```
开始 -> 任务A -> 任务B -> 结束
```
验证任务按顺序正确执行。

### 测试场景2: 条件分支工作流
```
开始 -> 条件判断 -> [分支A / 分支B] -> 结束
```
验证条件分支正确选择路径。

### 测试场景3: 并行执行工作流
```
开始 -> 并行节点 -> [任务A, 任务B, 任务C] -> 汇聚节点 -> 结束
```
验证并行任务正确执行并汇聚。

### 测试场景4: 持久化和恢复
```
1. 启动工作流执行到任务A
2. 持久化状态
3. 停止工作流
4. 从持久化存储恢复
5. 继续执行任务B
```
验证状态持久化和恢复功能。

### 测试场景5: 超时和重试
```
任务A设置超时5秒，重试3次
模拟任务A执行时间超过5秒
```
验证超时和重试机制。

## 项目结构

```
task1-workflow-engine/
├── src/
│   ├── main/
│   │   └── java/
│   │       └── com/workflow/
│   │           ├── core/           # 核心类
│   │           │   ├── Workflow.java
│   │           │   ├── WorkflowEngine.java
│   │           │   ├── WorkflowInstance.java
│   │           │   └── WorkflowContext.java
│   │           ├── node/           # 节点类型
│   │           │   ├── Node.java
│   │           │   ├── TaskNode.java
│   │           │   ├── ConditionNode.java
│   │           │   ├── ParallelNode.java
│   │           │   └── JoinNode.java
│   │           ├── state/          # 状态管理
│   │           │   ├── WorkflowState.java
│   │           │   └── NodeState.java
│   │           ├── persistence/    # 持久化
│   │           │   ├── PersistenceService.java
│   │           │   └── FilePersistenceService.java
│   │           └── builder/        # 构建器
│   │               └── WorkflowBuilder.java
│   └── test/
│       └── java/
│           └── com/workflow/
│               ├── WorkflowEngineTest.java
│               ├── NodeExecutionTest.java
│               ├── PersistenceTest.java
│               └── ParallelExecutionTest.java
├── README.md
└── pom.xml
```

## 提交清单

- [ ] 完整的源代码实现
- [ ] 单元测试代码（覆盖率 ≥ 80%）
- [ ] 设计文档（包含类图、时序图）
- [ ] API使用示例
- [ ] 性能测试报告
- [ ] 代码规范检查报告

## 参考资料

- [工作流模式](http://www.workflowpatterns.com/)
- [Activiti工作流引擎源码](https://github.com/Activiti/Activiti)
- [Java并发编程实战](https://jcip.net/)
