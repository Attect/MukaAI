# Task 3: 有限状态机框架

## 题目描述

实现一个功能强大的有限状态机框架，支持状态定义、转换规则、动作触发，具备状态历史和并行状态处理能力。

## 功能要求

### 1. 状态定义

#### 1.1 基本状态
- 支持定义状态名称
- 支持状态进入/退出动作
- 支持状态数据(上下文)

#### 1.2 复合状态
- 支持嵌套状态(子状态机)
- 支持历史状态(浅历史/深历史)
- 支持并行状态(正交区域)

#### 1.3 特殊状态
- 初始状态(Initial State)
- 终止状态(Final State)
- 历史状态(History State)

### 2. 转换规则

#### 2.1 基本转换
- 支持事件触发转换
- 支持条件转换(Guard)
- 支持无条件转换(Always)
- 支持延迟转换(After)

#### 2.2 转换动作
- 支持转换前动作
- 支持转换后动作
- 支持转换中动作

#### 2.3 转换类型
- **外部转换**: 退出当前状态,进入目标状态
- **内部转换**: 不改变当前状态
- **自转换**: 退出并重新进入当前状态
- **局部转换**: 只进入子状态,不退出父状态

### 3. 动作系统

#### 3.1 动作类型
- **Entry动作**: 进入状态时执行
- **Exit动作**: 退出状态时执行
- **Transition动作**: 转换时执行
- **Activity**: 状态中的持续活动

#### 3.2 动作执行
- 支持同步和异步动作
- 支持动作队列
- 支持动作取消

#### 3.3 副作用
- 支持状态变更通知
- 支持上下文更新
- 支持外部服务调用

### 4. 事件系统

#### 4.1 事件类型
- 支持自定义事件
- 支持内置事件(如超时事件)
- 支持事件数据

#### 4.2 事件处理
- 支持事件队列
- 支持事件优先级
- 支持事件过滤

### 5. 状态历史

#### 5.1 历史记录
- 记录状态转换历史
- 记录事件历史
- 记录动作执行历史

#### 5.2 状态回滚
- 支持回滚到指定状态
- 支持撤销/重做
- 支持快照和恢复

#### 5.3 时间旅行
- 支持查看任意时刻的状态
- 支持重放事件序列
- 支持调试模式

### 6. 并行状态

#### 6.1 正交区域
- 支持多个并行状态区域
- 每个区域独立状态转换
- 区域间通信

#### 6.2 同步机制
- 支持区域同步点
- 支持等待所有区域完成
- 支持部分区域完成

## 接口设计

```javascript
// 状态机接口
interface StateMachine {
  // 启动状态机
  start(): Promise<void>;
  
  // 停止状态机
  stop(): Promise<void>;
  
  // 发送事件
  send(event: string | Event): Promise<State>;
  
  // 获取当前状态
  getState(): State;
  
  // 获取上下文
  getContext(): any;
  
  // 更新上下文
  updateContext(data: any): void;
  
  // 状态历史
  getHistory(): StateHistory;
  
  // 回滚
  rollback(steps: number): Promise<void>;
  
  // 快照
  snapshot(): StateSnapshot;
  
  // 恢复
  restore(snapshot: StateSnapshot): Promise<void>;
  
  // 事件监听
  on(event: string, listener: Function): StateMachine;
  off(event: string, listener: Function): StateMachine;
}

// 状态定义
interface StateConfig {
  id: string;
  initial?: boolean;
  final?: boolean;
  type?: 'atomic' | 'compound' | 'parallel' | 'history';
  
  // 子状态
  states?: StateConfig[];
  
  // 进入动作
  entry?: Action | Action[];
  
  // 退出动作
  exit?: Action | Action[];
  
  // 活动动作
  activities?: Activity[];
  
  // 转换规则
  on?: TransitionConfig[];
  
  // 延迟转换
  after?: DelayedTransitionConfig[];
  
  // 无条件转换
  always?: TransitionConfig[];
  
  // 历史状态配置
  history?: 'shallow' | 'deep';
  
  // 数据
  data?: any;
  
  // 并行区域
  regions?: RegionConfig[];
}

// 转换配置
interface TransitionConfig {
  event?: string;              // 触发事件
  target: string;              // 目标状态
  guard?: GuardFunction;       // 守卫条件
  actions?: Action | Action[]; // 转换动作
  internal?: boolean;          // 是否内部转换
}

// 延迟转换配置
interface DelayedTransitionConfig {
  delay: number;               // 延迟时间(ms)
  target: string;              // 目标状态
  guard?: GuardFunction;       // 守卫条件
  actions?: Action | Action[]; // 转换动作
}

// 动作定义
type Action = string | ActionFunction | ActionObject;

interface ActionObject {
  type: string;
  exec: ActionFunction;
  params?: any;
}

type ActionFunction = (context: any, event: Event) => void | Promise<void>;

// 守卫函数
type GuardFunction = (context: any, event: Event) => boolean;

// 事件定义
interface Event {
  type: string;
  data?: any;
  timestamp?: number;
}

// 状态快照
interface StateSnapshot {
  state: string;
  context: any;
  history: StateHistory;
  timestamp: number;
}

// 状态历史
interface StateHistory {
  transitions: TransitionRecord[];
  events: EventRecord[];
  actions: ActionRecord[];
}

interface TransitionRecord {
  from: string;
  to: string;
  event: string;
  timestamp: number;
}

interface EventRecord {
  event: Event;
  handled: boolean;
  timestamp: number;
}

interface ActionRecord {
  action: string;
  state: string;
  timestamp: number;
}

// 并行区域配置
interface RegionConfig {
  id: string;
  states: StateConfig[];
  initial: string;
}

// 活动定义
interface Activity {
  id: string;
  start: ActivityFunction;
  stop: () => void;
}

type ActivityFunction = (context: any) => void | Promise<void>;
```

## 验收标准

### 功能验收

1. **基本状态转换**
   - 事件触发转换正确执行
   - 状态进入/退出动作正确调用
   - 转换动作正确执行
   - 上下文正确传递

2. **条件转换**
   - Guard函数正确判断
   - 条件不满足时转换不执行
   - 多个条件正确评估

3. **延迟转换**
   - 延迟时间准确
   - 延迟期间可取消
   - 状态改变时正确清理

4. **嵌套状态**
   - 子状态正确初始化
   - 状态转换正确冒泡
   - 进入/退出动作正确顺序

5. **历史状态**
   - 浅历史正确恢复
   - 深历史正确恢复
   - 历史状态正确更新

6. **并行状态**
   - 多个区域独立运行
   - 区域间正确通信
   - 同步点正确工作

7. **状态历史和回滚**
   - 历史记录完整
   - 回滚正确执行
   - 快照和恢复正确工作

8. **错误处理**
   - 无效转换正确处理
   - 动作错误正确捕获
   - 状态机保持一致性

### 性能验收

1. **转换性能**
   - 单次状态转换 < 1ms
   - 复杂嵌套状态转换 < 5ms
   - 支持1000+状态定义

2. **内存管理**
   - 历史记录可配置限制
   - 无内存泄漏
   - 快照大小可控

3. **并发性能**
   - 支持并行处理多个事件
   - 事件队列正确管理
   - 无竞态条件

### 代码质量验收

1. **代码结构**
   - 清晰的状态机模型
   - 可扩展的架构设计
   - 符合状态图规范

2. **错误处理**
   - 完善的错误检查
   - 清晰的错误信息
   - 合理的错误恢复

3. **测试覆盖**
   - 单元测试覆盖率 > 80%
   - 包含复杂场景测试
   - 包含边界情况测试

## 测试用例示例

```javascript
// 测试1: 基本状态转换
const machine = new StateMachine({
  id: 'traffic-light',
  initial: 'green',
  states: [
    {
      id: 'green',
      on: [
        { event: 'NEXT', target: 'yellow' }
      ],
      entry: () => console.log('进入绿灯'),
      exit: () => console.log('退出绿灯')
    },
    {
      id: 'yellow',
      on: [
        { event: 'NEXT', target: 'red' }
      ]
    },
    {
      id: 'red',
      on: [
        { event: 'NEXT', target: 'green' }
      ]
    }
  ]
});

await machine.start();
assert(machine.getState().id === 'green');

await machine.send('NEXT');
assert(machine.getState().id === 'yellow');

await machine.send('NEXT');
assert(machine.getState().id === 'red');

// 测试2: 条件转换
const machine2 = new StateMachine({
  id: 'door',
  initial: 'closed',
  context: { locked: false },
  states: [
    {
      id: 'closed',
      on: [
        {
          event: 'OPEN',
          target: 'open',
          guard: (ctx) => !ctx.locked
        }
      ]
    },
    {
      id: 'open',
      on: [
        { event: 'CLOSE', target: 'closed' }
      ]
    }
  ]
});

await machine2.start();
machine2.updateContext({ locked: true });
await machine2.send('OPEN');
assert(machine2.getState().id === 'closed'); // 应该还在closed

machine2.updateContext({ locked: false });
await machine2.send('OPEN');
assert(machine2.getState().id === 'open'); // 应该转换到open

// 测试3: 延迟转换
const machine3 = new StateMachine({
  id: 'timer',
  initial: 'idle',
  states: [
    {
      id: 'idle',
      on: [
        { event: 'START', target: 'running' }
      ]
    },
    {
      id: 'running',
      after: [
        { delay: 1000, target: 'finished' }
      ]
    },
    {
      id: 'finished',
      final: true
    }
  ]
});

await machine3.start();
await machine3.send('START');
assert(machine3.getState().id === 'running');

await sleep(1100);
assert(machine3.getState().id === 'finished');

// 测试4: 嵌套状态
const machine4 = new StateMachine({
  id: 'player',
  initial: 'stopped',
  states: [
    {
      id: 'stopped',
      on: [
        { event: 'PLAY', target: 'playing' }
      ]
    },
    {
      id: 'playing',
      initial: 'forward',
      states: [
        {
          id: 'forward',
          on: [
            { event: 'REWIND', target: 'rewinding' }
          ]
        },
        {
          id: 'rewinding',
          on: [
            { event: 'PLAY', target: 'forward' }
          ]
        }
      ],
      on: [
        { event: 'STOP', target: 'stopped' }
      ]
    }
  ]
});

await machine4.start();
await machine4.send('PLAY');
assert(machine4.getState().value === 'playing.forward');

await machine4.send('REWIND');
assert(machine4.getState().value === 'playing.rewinding');

await machine4.send('STOP');
assert(machine4.getState().id === 'stopped');

// 测试5: 历史状态
const machine5 = new StateMachine({
  id: 'cd-player',
  initial: 'stopped',
  states: [
    {
      id: 'stopped',
      on: [
        { event: 'PLAY', target: 'playing' }
      ]
    },
    {
      id: 'playing',
      initial: 'track1',
      history: 'deep',
      states: [
        { id: 'track1', on: [{ event: 'NEXT', target: 'track2' }] },
        { id: 'track2', on: [{ event: 'NEXT', target: 'track3' }] },
        { id: 'track3' }
      ],
      on: [
        { event: 'STOP', target: 'stopped' }
      ]
    }
  ]
});

await machine5.start();
await machine5.send('PLAY');
await machine5.send('NEXT');
assert(machine5.getState().value === 'playing.track2');

await machine5.send('STOP');
assert(machine5.getState().id === 'stopped');

await machine5.send('PLAY');
assert(machine5.getState().value === 'playing.track2'); // 恢复到历史状态

// 测试6: 并行状态
const machine6 = new StateMachine({
  id: 'system',
  type: 'parallel',
  regions: [
    {
      id: 'power',
      initial: 'off',
      states: [
        { id: 'off', on: [{ event: 'POWER_ON', target: 'on' }] },
        { id: 'on', on: [{ event: 'POWER_OFF', target: 'off' }] }
      ]
    },
    {
      id: 'network',
      initial: 'disconnected',
      states: [
        { id: 'disconnected', on: [{ event: 'CONNECT', target: 'connected' }] },
        { id: 'connected', on: [{ event: 'DISCONNECT', target: 'disconnected' }] }
      ]
    }
  ]
});

await machine6.start();
assert(machine6.getState().value === {
  power: 'off',
  network: 'disconnected'
});

await machine6.send('POWER_ON');
assert(machine6.getState().value === {
  power: 'on',
  network: 'disconnected'
});

await machine6.send('CONNECT');
assert(machine6.getState().value === {
  power: 'on',
  network: 'connected'
});

// 测试7: 状态历史和回滚
const machine7 = new StateMachine({
  id: 'wizard',
  initial: 'step1',
  states: [
    { id: 'step1', on: [{ event: 'NEXT', target: 'step2' }] },
    { id: 'step2', on: [{ event: 'NEXT', target: 'step3' }] },
    { id: 'step3', on: [{ event: 'NEXT', target: 'step4' }] },
    { id: 'step4' }
  ]
});

await machine7.start();
await machine7.send('NEXT');
await machine7.send('NEXT');
assert(machine7.getState().id === 'step3');

// 回滚一步
await machine7.rollback(1);
assert(machine7.getState().id === 'step2');

// 快照和恢复
const snapshot = machine7.snapshot();
await machine7.send('NEXT');
assert(machine7.getState().id === 'step3');

await machine7.restore(snapshot);
assert(machine7.getState().id === 'step2');

// 测试8: 动作执行顺序
const actions = [];
const machine8 = new StateMachine({
  id: 'test',
  initial: 'a',
  states: [
    {
      id: 'a',
      entry: () => actions.push('enter a'),
      exit: () => actions.push('exit a'),
      on: [
        {
          event: 'GO',
          target: 'b',
          actions: () => actions.push('transition a->b')
        }
      ]
    },
    {
      id: 'b',
      entry: () => actions.push('enter b'),
      exit: () => actions.push('exit b')
    }
  ]
});

await machine8.start();
assert.deepEqual(actions, ['enter a']);

actions.length = 0;
await machine8.send('GO');
assert.deepEqual(actions, [
  'exit a',
  'transition a->b',
  'enter b'
]);

// 辅助函数
function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}
```

## 实现提示

1. **使用状态图(Statechart)规范**作为设计参考
2. **实现状态树结构**支持嵌套和并行
3. **使用事件队列**处理事件
4. **实现事务机制**确保状态一致性
5. **考虑使用观察者模式**实现事件通知
6. **使用命令模式**实现动作队列
7. **考虑使用备忘录模式**实现快照功能

## 架构建议

```
StateMachine
├── StateNode (状态节点树)
│   ├── AtomicState
│   ├── CompoundState
│   └── ParallelState
├── EventQueue (事件队列)
├── ActionExecutor (动作执行器)
├── HistoryManager (历史管理器)
└── SnapshotManager (快照管理器)
```

## 参考资料

- [Statecharts](https://statecharts.github.io/)
- [XState](https://xstate.js.org/docs/)
- [UML State Machine](https://www.uml-diagrams.org/state-machine-diagrams.html)
- [Finite-state machine](https://en.wikipedia.org/wiki/Finite-state_machine)
