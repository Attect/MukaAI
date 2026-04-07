# Task 2: 流式数据处理器

## 题目描述

实现一个高性能的流式数据处理器，能够处理大量数据的转换、过滤、聚合等操作，并具备背压控制和错误恢复能力。

## 功能要求

### 1. 数据流操作

#### 1.1 Transform (转换)
- 支持同步和异步转换函数
- 支持一对多转换(一个输入产生多个输出)
- 支持过滤转换(返回null/undefined丢弃数据)

#### 1.2 Filter (过滤)
- 支持同步和异步过滤条件
- 支持多个过滤条件的组合(AND/OR)
- 支持过滤统计

#### 1.3 Batch (批处理)
- 支持按数量批次
- 支持按时间窗口批次
- 支持按大小限制批次
- 批次超时自动刷新

#### 1.4 Aggregate (聚合)
- 支持常见聚合函数: sum, avg, count, min, max
- 支持自定义聚合函数
- 支持分组聚合
- 支持滑动窗口聚合

#### 1.5 其他操作
- **Map**: 一对一映射
- **FlatMap**: 一对多映射并展平
- **Reduce**: 累积计算
- **Take**: 取前N个元素
- **Skip**: 跳过前N个元素
- **Distinct**: 去重
- **Sort**: 排序(需要缓冲区)

### 2. 背压控制

#### 2.1 基本背压
- 监控内部缓冲区大小
- 当缓冲区达到高水位时暂停读取
- 当缓冲区降至低水位时恢复读取
- 支持配置水位线

#### 2.2 流量控制
- 支持限速(Rate Limiting)
- 支持节流(Throttling)
- 支持防抖(Debouncing)

#### 2.3 内存管理
- 监控内存使用
- 内存压力时自动降级
- 支持配置最大内存使用

### 3. 错误处理

#### 3.1 错误恢复
- 支持配置错误处理策略
- 策略: `skip`, `retry`, `halt`
- 支持自定义错误处理器

#### 3.2 错误隔离
- 单个数据处理错误不影响其他数据
- 记录错误数据和错误信息
- 支持错误重放

#### 3.3 容错机制
- 支持检查点(Checkpoint)
- 支持从检查点恢复
- 支持事务性处理

### 4. 流控制

#### 4.1 暂停和恢复
- 支持暂停数据流
- 支持恢复数据流
- 暂停期间保持数据不丢失

#### 4.2 取消和清理
- 支持取消流处理
- 正确清理资源
- 提供取消回调

## 接口设计

```javascript
// 流处理器接口
interface StreamProcessor {
  // 添加数据源
  from(source: ReadableStream | Iterable | AsyncIterable): StreamProcessor;
  
  // 转换操作
  transform(fn: TransformFunction): StreamProcessor;
  
  // 过滤操作
  filter(predicate: FilterPredicate): StreamProcessor;
  
  // 批处理
  batch(options: BatchOptions): StreamProcessor;
  
  // 聚合
  aggregate(options: AggregateOptions): StreamProcessor;
  
  // 其他操作
  map(fn: MapFunction): StreamProcessor;
  flatMap(fn: FlatMapFunction): StreamProcessor;
  reduce(fn: ReduceFunction, initial: any): StreamProcessor;
  take(n: number): StreamProcessor;
  skip(n: number): StreamProcessor;
  distinct(keyFn?: KeyFunction): StreamProcessor;
  sort(compareFn?: CompareFunction): StreamProcessor;
  
  // 背压控制
  throttle(options: ThrottleOptions): StreamProcessor;
  rateLimit(rate: number): StreamProcessor;
  
  // 错误处理
  onError(handler: ErrorHandler): StreamProcessor;
  retry(options: RetryOptions): StreamProcessor;
  
  // 输出
  to(sink: WritableStream | Function): Promise<StreamResult>;
  
  // 流控制
  pause(): void;
  resume(): void;
  cancel(): void;
  
  // 事件
  on(event: string, listener: Function): StreamProcessor;
}

// 转换函数
type TransformFunction = (data: any) => any | Promise<any> | null | undefined;

// 过滤谓词
type FilterPredicate = (data: any) => boolean | Promise<boolean>;

// 批处理选项
interface BatchOptions {
  size?: number;           // 批次大小
  timeout?: number;        // 超时时间(ms)
  maxSize?: number;        // 最大字节大小
}

// 聚合选项
interface AggregateOptions {
  type: 'sum' | 'avg' | 'count' | 'min' | 'max' | 'custom';
  field?: string;          // 聚合字段
  groupBy?: string | Function;  // 分组字段或函数
  window?: {               // 滑动窗口
    size: number;
    slide: number;
  };
  custom?: AggregateFunction;   // 自定义聚合函数
  initial?: any;          // 初始值
}

// 节流选项
interface ThrottleOptions {
  interval: number;        // 时间间隔
  leading?: boolean;       // 是否在开始时立即执行
  trailing?: boolean;      // 是否在结束时执行
}

// 重试选项
interface RetryOptions {
  maxRetries: number;
  delay?: number;
  backoff?: 'fixed' | 'exponential';
}

// 错误处理器
type ErrorHandler = (error: Error, data: any) => 'skip' | 'retry' | 'halt' | void;

// 流处理结果
interface StreamResult {
  processed: number;       // 处理的数据数量
  errors: Error[];         // 错误列表
  duration: number;        // 处理时长
  memory: {                // 内存使用
    peak: number;
    average: number;
  };
}

// 背压配置
interface BackpressureConfig {
  highWaterMark: number;   // 高水位线
  lowWaterMark: number;    // 低水位线
  maxBufferSize: number;   // 最大缓冲区大小
}
```

## 验收标准

### 功能验收

1. **Transform操作**
   - 同步转换正确执行
   - 异步转换正确等待
   - 一对多转换正确展平
   - 返回null/undefined正确过滤

2. **Filter操作**
   - 过滤条件正确应用
   - 多条件组合正确工作
   - 过滤统计准确

3. **Batch操作**
   - 按数量批次正确分组
   - 按时间窗口批次正确触发
   - 按大小限制批次正确分割
   - 超时批次正确刷新

4. **Aggregate操作**
   - 基本聚合函数正确计算
   - 自定义聚合函数正确执行
   - 分组聚合正确工作
   - 滑动窗口聚合正确实现

5. **背压控制**
   - 高水位时正确暂停
   - 低水位时正确恢复
   - 限速功能正确工作
   - 内存监控准确

6. **错误处理**
   - skip策略正确跳过错误数据
   - retry策略正确重试
   - halt策略正确停止
   - 错误信息正确记录

7. **流控制**
   - 暂停时数据不丢失
   - 恢复后正确继续处理
   - 取消后资源正确释放

### 性能验收

1. **吞吐量**
   - 处理速度 > 10,000 条/秒
   - 支持流式处理无限数据
   - 内存使用稳定

2. **背压效果**
   - 背压触发时上游正确暂停
   - 内存使用不超过配置限制
   - 无数据丢失

3. **内存管理**
   - 处理100万条数据内存增长 < 50MB
   - 无内存泄漏
   - GC压力可控

4. **延迟**
   - 单条数据处理延迟 < 1ms
   - 批处理延迟符合配置
   - 背压响应延迟 < 10ms

### 代码质量验收

1. **代码结构**
   - 清晰的管道(Pipeline)设计
   - 操作符可组合
   - 良好的扩展性

2. **错误处理**
   - 完善的异常捕获
   - 清晰的错误信息
   - 合理的默认行为

3. **测试覆盖**
   - 单元测试覆盖率 > 80%
   - 包含大数据量测试
   - 包含背压场景测试
   - 包含错误场景测试

## 测试用例示例

```javascript
// 测试1: 基本转换
const processor = new StreamProcessor();
const result = await processor
  .from([1, 2, 3, 4, 5])
  .map(x => x * 2)
  .to(array => array);
assert.deepEqual(result.processed, [2, 4, 6, 8, 10]);

// 测试2: 过滤和批处理
const batches = [];
await new StreamProcessor()
  .from([1, 2, 3, 4, 5, 6, 7, 8, 9, 10])
  .filter(x => x % 2 === 0)
  .batch({ size: 3 })
  .to(batch => {
    batches.push(batch);
  });
assert.deepEqual(batches, [[2, 4, 6], [8, 10]]);

// 测试3: 聚合
const result = await new StreamProcessor()
  .from([
    { category: 'A', value: 10 },
    { category: 'B', value: 20 },
    { category: 'A', value: 30 }
  ])
  .aggregate({
    type: 'sum',
    field: 'value',
    groupBy: 'category'
  })
  .to(result => result);
assert.deepEqual(result, {
  A: 40,
  B: 20
});

// 测试4: 背压控制
let processed = 0;
let paused = false;
const processor = new StreamProcessor({
  backpressure: {
    highWaterMark: 100,
    lowWaterMark: 50
  }
});

processor
  .from(largeDataGenerator()) // 生成大量数据
  .throttle({ interval: 100 })
  .map(async x => {
    processed++;
    await sleep(10); // 模拟慢速处理
    return x;
  })
  .on('pause', () => { paused = true; })
  .on('resume', () => { paused = false; })
  .to(() => {});

// 验证背压触发
await sleep(1000);
assert(processed < 1000); // 由于背压，处理数量应该受限

// 测试5: 错误处理
const errors = [];
const result = await new StreamProcessor()
  .from([1, 2, 'invalid', 4, 5])
  .map(x => {
    if (typeof x !== 'number') {
      throw new Error('Invalid type');
    }
    return x * 2;
  })
  .onError((error, data) => {
    errors.push({ error, data });
    return 'skip';
  })
  .to(array => array);
assert.deepEqual(result.processed, [2, 4, 8, 10]);
assert(errors.length === 1);

// 测试6: 滑动窗口聚合
const windows = [];
await new StreamProcessor()
  .from([1, 2, 3, 4, 5, 6, 7, 8, 9, 10])
  .aggregate({
    type: 'sum',
    window: {
      size: 3,
      slide: 2
    }
  })
  .to(result => {
    windows.push(result);
  });
assert.deepEqual(windows, [
  [1, 2, 3],      // 窗口1
  [3, 4, 5],      // 窗口2
  [5, 6, 7],      // 窗口3
  [7, 8, 9],      // 窗口4
  [9, 10]         // 窗口5
].map(w => w.reduce((a, b) => a + b, 0)));

// 测试7: 大数据量处理
const count = 1000000;
let processedCount = 0;
const startMemory = process.memoryUsage().heapUsed;

await new StreamProcessor()
  .from(generateNumbers(count))
  .filter(x => x % 2 === 0)
  .map(x => x * 2)
  .to(() => {
    processedCount++;
  });

const endMemory = process.memoryUsage().heapUsed;
const memoryGrowth = endMemory - startMemory;
assert(processedCount === count / 2);
assert(memoryGrowth < 50 * 1024 * 1024); // 内存增长 < 50MB

// 辅助函数
function* generateNumbers(count) {
  for (let i = 0; i < count; i++) {
    yield i;
  }
}

async function* largeDataGenerator() {
  let i = 0;
  while (true) {
    yield i++;
    await sleep(1);
  }
}

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}
```

## 实现提示

1. **使用Node.js Stream API**作为基础
2. **实现背压机制**使用`highWaterMark`和`drain`事件
3. **使用异步迭代器**处理异步数据源
4. **使用生成器函数**实现惰性求值
5. **考虑使用对象池**减少GC压力
6. **使用Buffer**处理二进制数据
7. **注意错误边界**确保异常不会中断整个流

## 性能优化建议

1. **避免不必要的数据拷贝**
2. **使用对象池复用对象**
3. **批量处理减少函数调用开销**
4. **合理设置缓冲区大小**
5. **使用快速路径优化常见操作**
6. **考虑使用Worker Threads并行处理**

## 参考资料

- [Node.js Stream API](https://nodejs.org/api/stream.html)
- [Backpressure in Node.js Streams](https://nodejs.org/en/docs/guides/backpressuring-in-streams/)
- [Async Iterators](https://javascript.info/async-iterators-generators)
- [ReactiveX](http://reactivex.io/)
