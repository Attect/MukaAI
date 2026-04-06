# Task 2: 事件总线

## 元数据
- **标题**: 事件总线实现
- **作者**: 开发团队
- **日期**: 2026-04-06
- **版本**: 1.0.0

## 题目描述

实现一个基于发布-订阅模式的事件总线系统，支持同步和异步事件处理、事件过滤和路由、死信队列等功能。该系统应具备高性能、高可靠性的特点，能够处理大量的事件消息。

## 功能需求

### 1. 核心功能

#### 1.1 事件发布与订阅
- 支持发布者发布事件到事件总线
- 支持订阅者订阅特定类型的事件
- 支持多个订阅者订阅同一事件
- 支持订阅者取消订阅

#### 1.2 事件处理模式
- **同步处理**: 事件发布后，同步调用所有订阅者
- **异步处理**: 事件发布后，异步调用订阅者，不阻塞发布者
- 支持配置每个订阅者的处理模式

#### 1.3 事件过滤与路由
- 支持基于事件类型的订阅
- 支持基于事件内容的过滤（如属性值匹配）
- 支持自定义过滤规则
- 支持事件路由到特定的订阅者组

#### 1.4 死信队列（Dead Letter Queue）
- 处理失败的事件自动进入死信队列
- 支持查询死信队列中的事件
- 支持手动重试死信队列中的事件
- 支持配置死信队列的最大容量和过期时间

#### 1.5 事件追踪
- 支持记录事件的完整生命周期
- 支持查询事件的处理状态
- 支持统计事件处理的性能指标

### 2. 技术要求

#### 2.1 设计模式应用
- **观察者模式**: 事件发布-订阅的核心模式
- **生产者-消费者模式**: 异步事件处理
- **策略模式**: 不同的事件分发策略
- **装饰器模式**: 事件处理器的增强（如重试、日志）

#### 2.2 并发处理
- 使用线程池管理异步事件处理
- 确保线程安全，避免竞态条件
- 合理控制线程池大小，避免资源耗尽

#### 2.3 性能优化
- 支持批量事件发布
- 支持事件处理的异步批处理
- 提供性能监控指标

## API设计示例

```java
// 创建事件总线
EventBus eventBus = new EventBusBuilder()
    .threadPoolSize(10)
    .enableDeadLetterQueue(true)
    .enableEventTracing(true)
    .build();

// 定义事件
public class OrderCreatedEvent {
    private String orderId;
    private String customerId;
    private BigDecimal amount;
    // getters and setters
}

// 订阅事件 - 同步处理
@EventHandler(mode = HandleMode.SYNC)
public class OrderCreatedSyncHandler {
    @Subscribe
    public void handle(OrderCreatedEvent event) {
        // 同步处理逻辑
    }
}

// 订阅事件 - 异步处理
@EventHandler(mode = HandleMode.ASYNC)
public class OrderCreatedAsyncHandler {
    @Subscribe
    public void handle(OrderCreatedEvent event) {
        // 异步处理逻辑
    }
}

// 订阅事件 - 带过滤条件
@EventHandler(filter = @EventFilter(property = "amount", operator = ">", value = "1000"))
public class HighValueOrderHandler {
    @Subscribe
    public void handle(OrderCreatedEvent event) {
        // 处理高价值订单
    }
}

// 注册订阅者
eventBus.register(new OrderCreatedSyncHandler());
eventBus.register(new OrderCreatedAsyncHandler());
eventBus.register(new HighValueOrderHandler());

// 发布事件
OrderCreatedEvent event = new OrderCreatedEvent("ORD-001", "CUST-123", new BigDecimal("1500"));
eventBus.publish(event);

// 查询死信队列
List<DeadLetterEvent> deadLetters = eventBus.getDeadLetterQueue().list();

// 重试死信事件
eventBus.getDeadLetterQueue().retry(deadLetters.get(0).getEventId());

// 查询事件追踪信息
EventTrace trace = eventBus.getEventTrace(event.getEventId());
```

## 验收标准

### 功能验收

| 序号 | 验收项 | 验收标准 | 优先级 |
|------|--------|----------|--------|
| 1 | 事件发布 | 能够成功发布事件到事件总线 | P0 |
| 2 | 事件订阅 | 订阅者能够接收到订阅的事件 | P0 |
| 3 | 同步处理 | 同步模式下，发布者等待所有订阅者处理完成 | P0 |
| 4 | 异步处理 | 异步模式下，发布者不等待订阅者处理完成 | P0 |
| 5 | 事件过滤 | 过滤规则能够正确过滤事件 | P0 |
| 6 | 死信队列 | 处理失败的事件能够进入死信队列 | P0 |
| 7 | 死信重试 | 能够从死信队列重试事件 | P1 |
| 8 | 取消订阅 | 取消订阅后不再接收事件 | P1 |
| 9 | 事件追踪 | 能够查询事件的处理状态和历史 | P1 |
| 10 | 性能统计 | 能够统计事件处理的性能指标 | P2 |

### 质量验收

| 序号 | 验收项 | 验收标准 |
|------|--------|----------|
| 1 | 单元测试 | 测试覆盖率 ≥ 80% |
| 2 | 并发安全 | 多线程环境下无死锁、无竞态条件 |
| 3 | 性能要求 | 同步处理吞吐量 ≥ 10,000 events/sec |
| 4 | 性能要求 | 异步处理吞吐量 ≥ 50,000 events/sec |
| 5 | 代码规范 | 通过阿里巴巴Java开发规范检查 |
| 6 | 文档完整 | 包含设计文档、API文档、使用示例 |

## 测试用例

### 测试场景1: 基本发布订阅
```
1. 创建事件总线
2. 注册订阅者A、B、C
3. 发布事件E
4. 验证A、B、C都接收到事件E
```

### 测试场景2: 同步与异步处理
```
1. 注册同步订阅者A（处理时间100ms）
2. 注册异步订阅者B（处理时间200ms）
3. 发布事件并记录时间
4. 验证发布耗时约100ms（同步等待A）
5. 验证B异步处理完成
```

### 测试场景3: 事件过滤
```
1. 注册订阅者A，过滤条件：amount > 1000
2. 发布事件E1（amount=500）
3. 发布事件E2（amount=1500）
4. 验证A只接收到E2
```

### 测试场景4: 死信队列
```
1. 注册订阅者A，处理时抛出异常
2. 发布事件E
3. 验证E进入死信队列
4. 重试E
5. 验证E重新处理
```

### 测试场景5: 高并发测试
```
1. 创建事件总线（线程池大小10）
2. 注册100个订阅者
3. 并发发布10,000个事件
4. 验证所有事件都被正确处理
5. 验证无事件丢失
```

## 项目结构

```
task2-event-bus/
├── src/
│   ├── main/
│   │   └── java/
│   │       └── com/eventbus/
│   │           ├── core/              # 核心类
│   │           │   ├── EventBus.java
│   │           │   ├── Event.java
│   │           │   └── EventBusBuilder.java
│   │           ├── subscriber/        # 订阅者管理
│   │           │   ├── Subscriber.java
│   │           │   ├── SubscriberRegistry.java
│   │           │   └── SubscriberInvoker.java
│   │           ├── dispatcher/        # 事件分发
│   │           │   ├── EventDispatcher.java
│   │           │   ├── SyncDispatcher.java
│   │           │   └── AsyncDispatcher.java
│   │           ├── filter/            # 事件过滤
│   │           │   ├── EventFilter.java
│   │           │   └── FilterRule.java
│   │           ├── deadletter/        # 死信队列
│   │           │   ├── DeadLetterQueue.java
│   │           │   └── DeadLetterEvent.java
│   │           ├── trace/             # 事件追踪
│   │           │   ├── EventTracer.java
│   │           │   └── EventTrace.java
│   │           └── annotation/        # 注解
│   │               ├── EventHandler.java
│   │               └── Subscribe.java
│   └── test/
│       └── java/
│           └── com/eventbus/
│               ├── EventBusTest.java
│               ├── SyncAsyncTest.java
│               ├── FilterTest.java
│               ├── DeadLetterQueueTest.java
│               └── PerformanceTest.java
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

- [Guava EventBus源码](https://github.com/google/guava/tree/master/guava/src/com/google/common/eventbus)
- [Spring Event机制](https://docs.spring.io/spring-framework/docs/current/reference/html/core.html#context-functionality-events)
- [观察者模式详解](https://refactoring.guru/design-patterns/observer)
