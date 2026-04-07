/**
 * TaskScheduler - 异步任务调度器
 * 
 * 支持功能：
 * - schedule(task, options): 调度任务
 * - cancel(taskId): 取消任务
 * - pause(taskId): 暂停任务
 * - resume(taskId): 恢复任务
 * - getStatus(taskId): 获取任务状态
 * 
 * 支持 4 种任务类型：立即任务、延迟任务、定时任务、周期任务
 * 支持 5 级优先级：CRITICAL > HIGH > NORMAL > LOW > IDLE
 * 支持任务重试机制（指数退避）
 */

// 任务类型枚举
const TaskType = {
  IMMEDIATE: 'immediate',   // 立即任务
  DELAYED: 'delayed',       // 延迟任务
  SCHEDULED: 'scheduled',   // 定时任务
  RECURRING: 'recurring'    // 周期任务
};

// 优先级枚举（数值越大优先级越高）
const Priority = {
  CRITICAL: { value: 'critical', level: 5 },
  HIGH: { value: 'high', level: 4 },
  NORMAL: { value: 'normal', level: 3 },
  LOW: { value: 'low', level: 2 },
  IDLE: { value: 'idle', level: 1 }
};

// 任务状态枚举
const TaskStatus = {
  PENDING: 'pending',      // 待执行
  RUNNING: 'running',      // 执行中
  PAUSED: 'paused',        // 已暂停
  COMPLETED: 'completed',   // 已完成
  CANCELLED: 'cancelled',   // 已取消
  FAILED: 'failed'          // 失败
};

// 重试配置
const RetryConfig = {
  DEFAULT_MAX_RETRIES: 3,
  DEFAULT_BASE_DELAY: 1000,      // 基础延迟（毫秒）
  DEFAULT_MAX_DELAY: 30000,       // 最大延迟（毫秒）
  DEFAULT_BACKOFF_MULTIPLIER: 2   // 退避乘数
};

/**
 * 任务类
 */
class Task {
  constructor(id, executor, options = {}) {
    this.id = id;
    this.executor = executor;
    this.type = options.type || TaskType.IMMEDIATE;
    this.priority = options.priority || Priority.NORMAL;
    this.delay = options.delay || 0;              // 延迟时间（毫秒）
    this.scheduleTime = options.scheduleTime || null;  // 计划执行时间
    this.interval = options.interval || null;     // 周期间隔（毫秒）
    this.maxRetries = options.maxRetries !== undefined ? 
      options.maxRetries : RetryConfig.DEFAULT_MAX_RETRIES;
    this.baseDelay = options.baseDelay !== undefined ? 
      options.baseDelay : RetryConfig.DEFAULT_BASE_DELAY;
    this.backoffMultiplier = options.backoffMultiplier !== undefined ? 
      options.backoffMultiplier : RetryConfig.DEFAULT_BACKOFF_MULTIPLIER;
    
    this.status = TaskStatus.PENDING;
    this.createdAt = Date.now();
    this.startedAt = null;
    this.completedAt = null;
    this.retryCount = 0;
    this.error = null;
    this.result = null;
    
    // 内部状态
    this._timerId = null;
    this._intervalId = null;
    this._pauseTimeoutId = null;
    this._resumePending = false;
  }

  toJSON() {
    return {
      id: this.id,
      type: this.type,
      priority: this.priority.value,
      status: this.status,
      retryCount: this.retryCount,
      maxRetries: this.maxRetries,
      createdAt: this.createdAt,
      startedAt: this.startedAt,
      completedAt: this.completedAt,
      error: this.error,
      result: this.result
    };
  }
}

/**
 * 任务调度器类
 */
class TaskScheduler {
  constructor() {
    // 任务存储：Map<taskId, Task>
    this._tasks = new Map();
    
    // 优先级队列：按优先级分组存储待执行任务
    this._priorityQueues = {
      [Priority.CRITICAL.level]: [],
      [Priority.HIGH.level]: [],
      [Priority.NORMAL.level]: [],
      [Priority.LOW.level]: [],
      [Priority.IDLE.level]: []
    };
    
    // 事件监听器
    this._eventListeners = new Map();
    
    // 调度器运行状态
    this._isRunning = true;
    
    // 当前正在执行的任务 ID 集合
    this._runningTasks = new Set();
  }

  /**
   * 调度任务
   * @param {Function} executor - 任务执行函数
   * @param {Object} options - 任务配置选项
   * @param {string} options.id - 任务 ID（可选，自动生成）
   * @param {string} options.type - 任务类型：immediate|delayed|scheduled|recurring
   * @param {string} options.priority - 优先级：critical|high|normal|low|idle
   * @param {number} options.delay - 延迟时间（毫秒），用于延迟任务
   * @param {Date} options.scheduleTime - 计划执行时间，用于定时任务
   * @param {number} options.interval - 周期间隔（毫秒），用于周期任务
   * @param {number} options.maxRetries - 最大重试次数
   * @param {number} options.baseDelay - 基础延迟（毫秒）
   * @param {number} options.backoffMultiplier - 退避乘数
   * @returns {string} 任务 ID
   */
  schedule(executor, options = {}) {
    const taskId = options.id || `task-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    
    // 创建任务实例
    const task = new Task(taskId, executor, options);
    this._tasks.set(taskId, task);
    
    // 将任务添加到优先级队列
    this._addToPriorityQueue(task);
    
    // 根据任务类型调度执行
    switch (task.type) {
      case TaskType.IMMEDIATE:
        this._scheduleImmediate(task);
        break;
      case TaskType.DELAYED:
        this._scheduleDelayed(task);
        break;
      case TaskType.SCHEDULED:
        this._scheduleScheduled(task);
        break;
      case TaskType.RECURRING:
        this._scheduleRecurring(task);
        break;
    }
    
    this._emitEvent('task:scheduled', { taskId, task });
    
    return taskId;
  }

  /**
   * 取消任务
   * @param {string} taskId - 任务 ID
   * @returns {boolean} 是否成功取消
   */
  cancel(taskId) {
    const task = this._tasks.get(taskId);
    if (!task) {
      return false;
    }

    // 如果任务正在运行，清除定时器
    if (task.status === TaskStatus.RUNNING || task.status === TaskStatus.PAUSED) {
      clearTimeout(task._timerId);
      clearInterval(task._intervalId);
      clearTimeout(task._pauseTimeoutId);
    }

    // 更新任务状态
    task.status = TaskStatus.CANCELLED;
    task.completedAt = Date.now();
    
    // 从运行集合中移除
    this._runningTasks.delete(taskId);
    
    // 从优先级队列中移除
    this._removeFromPriorityQueue(task);
    
    this._emitEvent('task:cancelled', { taskId, task });
    
    return true;
  }

  /**
   * 暂停任务
   * @param {string} taskId - 任务 ID
   * @returns {boolean} 是否成功暂停
   */
  pause(taskId) {
    const task = this._tasks.get(taskId);
    if (!task) {
      return false;
    }

    // 只有运行中的任务可以暂停
    if (task.status !== TaskStatus.RUNNING && task.status !== TaskStatus.PENDING) {
      return false;
    }

    // 清除当前定时器
    clearTimeout(task._timerId);
    clearInterval(task._intervalId);
    
    // 更新状态为暂停
    task.status = TaskStatus.PAUSED;
    
    this._emitEvent('task:paused', { taskId, task });
    
    return true;
  }

  /**
   * 恢复任务
   * @param {string} taskId - 任务 ID
   * @returns {boolean} 是否成功恢复
   */
  resume(taskId) {
    const task = this._tasks.get(taskId);
    if (!task) {
      return false;
    }

    // 只有暂停的任务可以恢复
    if (task.status !== TaskStatus.PAUSED) {
      return false;
    }

    // 更新状态为运行中
    task.status = TaskStatus.RUNNING;
    
    // 重新调度任务执行
    switch (task.type) {
      case TaskType.IMMEDIATE:
      case TaskType.DELAYED:
        this._executeTask(task);
        break;
      case TaskType.SCHEDULED:
        this._scheduleScheduled(task);
        break;
      case TaskType.RECURRING:
        this._scheduleRecurring(task);
        break;
    }
    
    this._emitEvent('task:resumed', { taskId, task });
    
    return true;
  }

  /**
   * 获取任务状态
   * @param {string} taskId - 任务 ID
   * @returns {Object|null} 任务状态信息，如果任务不存在返回 null
   */
  getStatus(taskId) {
    const task = this._tasks.get(taskId);
    if (!task) {
      return null;
    }

    return task.toJSON();
  }

  /**
   * 获取所有任务列表
   * @returns {Array} 任务列表
   */
  getAllTasks() {
    return Array.from(this._tasks.values()).map(task => task.toJSON());
  }

  /**
   * 注册事件监听器
   * @param {string} event - 事件名称
   * @param {Function} listener - 监听函数
   */
  on(event, listener) {
    if (!this._eventListeners.has(event)) {
      this._eventListeners.set(event, []);
    }
    this._eventListeners.get(event).push(listener);
  }

  /**
   * 停止调度器
   */
  stop() {
    this._isRunning = false;
    
    // 取消所有任务
    for (const [taskId] of this._tasks) {
      this.cancel(taskId);
    }
    
    this._tasks.clear();
  }

  /**
   * 内部方法：将任务添加到优先级队列
   */
  _addToPriorityQueue(task) {
    const queue = this._priorityQueues[task.priority.level];
    queue.push(task);
    
    // 按创建时间排序（FIFO）
    queue.sort((a, b) => a.createdAt - b.createdAt);
  }

  /**
   * 内部方法：从优先级队列移除任务
   */
  _removeFromPriorityQueue(task) {
    const queue = this._priorityQueues[task.priority.level];
    const index = queue.findIndex(t => t.id === task.id);
    if (index !== -1) {
      queue.splice(index, 1);
    }
  }

  /**
   * 内部方法：调度立即任务
   */
  _scheduleImmediate(task) {
    // 立即任务直接加入优先级队列并执行
    this._executeTask(task);
  }

  /**
   * 内部方法：调度延迟任务
   */
  _scheduleDelayed(task) {
    task._timerId = setTimeout(() => {
      if (task.status === TaskStatus.PENDING) {
        this._executeTask(task);
      }
    }, task.delay);
  }

  /**
   * 内部方法：调度定时任务
   */
  _scheduleScheduled(task) {
    const now = Date.now();
    const delay = task.scheduleTime.getTime() - now;
    
    if (delay <= 0) {
      // 如果计划时间已过，立即执行
      this._executeTask(task);
    } else {
      task._timerId = setTimeout(() => {
        if (task.status === TaskStatus.PENDING) {
          this._executeTask(task);
        }
      }, delay);
    }
  }

  /**
   * 内部方法：调度周期任务
   */
  _scheduleRecurring(task) {
    // 立即执行一次
    this._executeTask(task);
    
    // 设置周期性执行
    task._intervalId = setInterval(() => {
      if (task.status !== TaskStatus.PAUSED) {
        this._executeTask(task);
      }
    }, task.interval);
  }

  /**
   * 内部方法：执行任务（支持重试和指数退避）
   */
  _executeTask(task) {
    if (this._runningTasks.has(task.id)) {
      // 如果任务已经在运行中，跳过
      return;
    }

    this._runningTasks.add(task.id);
    task.status = TaskStatus.RUNNING;
    task.startedAt = Date.now();
    
    this._emitEvent('task:started', { taskId: task.id, task });

    // 执行任务（支持 Promise）
    const executeWithRetry = async () => {
      try {
        const result = await task.executor();
        
        // 任务执行成功
        task.result = result;
        task.status = TaskStatus.COMPLETED;
        task.completedAt = Date.now();
        
        this._emitEvent('task:completed', { taskId: task.id, task, result });
        
      } catch (error) {
        // 检查是否需要重试
        if (task.retryCount < task.maxRetries) {
          task.retryCount++;
          
          // 计算指数退避延迟
          const delay = Math.min(
            task.baseDelay * Math.pow(task.backoffMultiplier, task.retryCount - 1),
            RetryConfig.DEFAULT_MAX_DELAY
          );
          
          this._emitEvent('task:retrying', { 
            taskId: task.id, 
            task, 
            retryCount: task.retryCount, 
            delay 
          });
          
          // 延迟后重试
          setTimeout(() => {
            if (task.status !== TaskStatus.CANCELLED) {
              executeWithRetry();
            }
          }, delay);
        } else {
          // 重试次数用尽，任务失败
          task.error = error;
          task.status = TaskStatus.FAILED;
          task.completedAt = Date.now();
          
          this._emitEvent('task:failed', { taskId: task.id, task, error });
        }
      } finally {
        this._runningTasks.delete(task.id);
      }
    };

    executeWithRetry();
  }

  /**
   * 内部方法：发射事件
   */
  _emitEvent(event, data) {
    const listeners = this._eventListeners.get(event);
    if (listeners) {
      for (const listener of listeners) {
        try {
          listener(data);
        } catch (error) {
          console.error(`Error in event listener for ${event}:`, error);
        }
      }
    }
  }
}

// 导出模块
module.exports = {
  TaskScheduler,
  TaskType,
  Priority,
  TaskStatus,
  RetryConfig
};
