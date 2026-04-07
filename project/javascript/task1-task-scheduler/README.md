# Task 1: 异步任务调度器

## 题目描述

实现一个功能完整的异步任务调度器，能够管理和执行各种类型的异步任务。

## 功能要求

### 1. 任务类型支持

#### 1.1 立即任务 (Immediate Task)
- 任务创建后立即执行
- 支持异步函数

#### 1.2 延迟任务 (Delayed Task)
- 在指定延迟时间后执行
- 支持取消操作

#### 1.3 定时任务 (Scheduled Task)
- 在指定时间点执行
- 支持绝对时间和相对时间

#### 1.4 周期任务 (Periodic Task)
- 按固定间隔重复执行
- 支持固定延迟和固定频率两种模式
- 支持最大执行次数限制

### 2. 任务优先级

- 支持5个优先级级别: `CRITICAL`, `HIGH`, `NORMAL`, `LOW`, `IDLE`
- 高优先级任务优先执行
- 同优先级任务按创建时间排序

### 3. 任务管理

#### 3.1 任务取消
- 支持取消未执行的任务
- 取消操作需要清理相关资源
- 提供取消回调机制

#### 3.2 任务重试
- 支持配置最大重试次数
- 支持自定义重试延迟策略
- 支持指数退避算法

#### 3.3 任务状态追踪
- 任务状态: `PENDING`, `RUNNING`, `COMPLETED`, `FAILED`, `CANCELLED`
- 提供状态变更事件

### 4. 并发控制

- 支持配置最大并发任务数
- 任务队列管理
- 支持任务暂停和恢复

## 接口设计

```javascript
// 任务调度器接口
interface TaskScheduler {
  // 提交任务
  submit(task: Task): TaskHandle;
  
  // 取消任务
  cancel(taskId: string): boolean;
  
  // 获取任务状态
  getStatus(taskId: string): TaskStatus;
  
  // 暂停调度器
  pause(): void;
  
  // 恢复调度器
  resume(): void;
  
  // 关闭调度器
  shutdown(): Promise<void>;
}

// 任务接口
interface Task {
  id: string;
  type: 'immediate' | 'delayed' | 'scheduled' | 'periodic';
  priority: Priority;
  execute: () => Promise<any>;
  
  // 延迟任务配置
  delay?: number;
  
  // 定时任务配置
  scheduledTime?: number;
  
  // 周期任务配置
  interval?: number;
  maxExecutions?: number;
  fixedRate?: boolean;
  
  // 重试配置
  retryConfig?: {
    maxRetries: number;
    delayStrategy: 'fixed' | 'exponential';
    baseDelay: number;
  };
  
  // 回调
  onComplete?: (result: any) => void;
  onError?: (error: Error) => void;
  onCancel?: () => void;
}

// 任务句柄
interface TaskHandle {
  id: string;
  promise: Promise<any>;
  cancel: () => boolean;
  getStatus: () => TaskStatus;
}

// 优先级枚举
enum Priority {
  CRITICAL = 0,
  HIGH = 1,
  NORMAL = 2,
  LOW = 3,
  IDLE = 4
}

// 任务状态枚举
enum TaskStatus {
  PENDING = 'PENDING',
  RUNNING = 'RUNNING',
  COMPLETED = 'COMPLETED',
  FAILED = 'FAILED',
  CANCELLED = 'CANCELLED'
}
```

## 验收标准

### 功能验收

1. **立即任务执行**
   - 任务创建后立即开始执行
   - 正确处理异步函数
   - 正确返回执行结果

2. **延迟任务执行**
   - 任务在指定延迟后执行
   - 延迟时间准确(误差<100ms)
   - 支持在延迟期间取消

3. **定时任务执行**
   - 任务在指定时间点执行
   - 支持绝对时间戳和相对时间
   - 时间准确(误差<100ms)

4. **周期任务执行**
   - 任务按间隔重复执行
   - 正确区分固定延迟和固定频率模式
   - 达到最大执行次数后停止

5. **优先级管理**
   - 高优先级任务优先执行
   - 同优先级按FIFO顺序
   - 优先级抢占正确工作

6. **任务取消**
   - 未执行任务可成功取消
   - 正在执行的任务收到取消信号
   - 取消后资源正确释放

7. **任务重试**
   - 失败任务自动重试
   - 重试次数限制正确
   - 指数退避算法正确实现

8. **并发控制**
   - 并发数不超过配置的最大值
   - 任务队列正确管理
   - 暂停和恢复功能正常

### 性能验收

1. **任务调度延迟**
   - 任务提交到开始执行的延迟 < 10ms
   - 高优先级任务响应时间 < 5ms

2. **内存管理**
   - 完成的任务资源及时释放
   - 长时间运行无内存泄漏
   - 支持1000+任务并发

3. **稳定性**
   - 连续运行24小时无崩溃
   - 异常任务不影响其他任务
   - 正确处理各种边界情况

### 代码质量验收

1. **代码结构**
   - 清晰的类和方法划分
   - 良好的封装性
   - 符合SOLID原则

2. **错误处理**
   - 完善的错误捕获
   - 清晰的错误信息
   - 合理的错误恢复

3. **测试覆盖**
   - 单元测试覆盖率 > 80%
   - 包含边界情况测试
   - 包含并发场景测试

## 测试用例示例

```javascript
// 测试1: 基本立即任务
const scheduler = new TaskScheduler({ maxConcurrent: 5 });
const task = {
  id: 'task-1',
  type: 'immediate',
  priority: Priority.NORMAL,
  execute: async () => {
    return 'result';
  }
};

const handle = scheduler.submit(task);
const result = await handle.promise;
assert(result === 'result');

// 测试2: 延迟任务
const delayedTask = {
  id: 'task-2',
  type: 'delayed',
  priority: Priority.HIGH,
  delay: 1000,
  execute: async () => {
    return 'delayed result';
  }
};

const start = Date.now();
const handle2 = scheduler.submit(delayedTask);
await handle2.promise;
const elapsed = Date.now() - start;
assert(elapsed >= 1000 && elapsed < 1100);

// 测试3: 优先级
const results = [];
const lowTask = {
  id: 'low',
  type: 'immediate',
  priority: Priority.LOW,
  execute: async () => { results.push('low'); }
};

const highTask = {
  id: 'high',
  type: 'immediate',
  priority: Priority.HIGH,
  execute: async () => { results.push('high'); }
};

scheduler.submit(lowTask);
scheduler.submit(highTask);
await Promise.all([...]);
assert(results[0] === 'high');

// 测试4: 任务取消
const cancelTask = {
  id: 'cancel',
  type: 'delayed',
  priority: Priority.NORMAL,
  delay: 5000,
  execute: async () => {
    return 'should not execute';
  },
  onCancel: () => {
    console.log('task cancelled');
  }
};

const handle3 = scheduler.submit(cancelTask);
setTimeout(() => {
  handle3.cancel();
}, 100);
await new Promise(resolve => setTimeout(resolve, 200));
assert(handle3.getStatus() === TaskStatus.CANCELLED);

// 测试5: 重试机制
let attempts = 0;
const retryTask = {
  id: 'retry',
  type: 'immediate',
  priority: Priority.NORMAL,
  retryConfig: {
    maxRetries: 3,
    delayStrategy: 'exponential',
    baseDelay: 100
  },
  execute: async () => {
    attempts++;
    if (attempts < 3) {
      throw new Error('temporary failure');
    }
    return 'success';
  }
};

const handle4 = scheduler.submit(retryTask);
const result = await handle4.promise;
assert(result === 'success');
assert(attempts === 3);
```

## 实现提示

1. **使用优先队列**管理待执行任务
2. **使用Promise和async/await**处理异步逻辑
3. **使用setTimeout和setInterval**实现定时功能
4. **使用AbortController**实现任务取消
5. **考虑使用Worker Threads**处理CPU密集型任务
6. **注意清理定时器和事件监听器**避免内存泄漏

## 参考资料

- [Node.js Event Loop](https://nodejs.org/en/docs/guides/event-loop-timers-and-nexttick/)
- [MDN - Promise](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise)
- [AbortController API](https://nodejs.org/api/globals.html#class-abortcontroller)
