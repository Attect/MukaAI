# AI 并发控制和会话压缩配置示例

## 配置文件示例 (application.conf)

```hocon
app {
    aiService {
        # LM Studio 基础配置
        lmStudioUrl = "http://localhost:1234/v1"
        apiKey = "sk-test-key-123456"
        modelName = "qwen3.5-9b-uncensored-hauhaucs-aggressive"
        
        # 模型配置（可选：如果无法自动获取）
        # 如果未指定，系统会尝试从 LM Studio API 自动获取
        modelMaxTokens = 32768
        
        # ==================== 会话压缩配置 ====================
        sessionCompression {
            # 是否启用会话压缩
            enabled = true
            
            # 压缩触发阈值（百分比）
            # 当会话 token 使用量达到模型最大 token 的 80% 时触发压缩
            thresholdPercent = 0.8
            
            # 最小阈值（可选）
            # 即使百分比计算结果更低，也不会低于这个值
            minThresholdTokens = 10000
            
            # 最大阈值（可选）
            # 即使百分比计算结果更高，也不会超过这个值
            maxThresholdTokens = 30000
            
            # 压缩策略
            # - summarize: 使用 AI 摘要压缩（推荐）
            # - truncate: 截断最早的对话
            # - smart: 智能压缩（优先保留重要信息）
            strategy = "summarize"
            
            # 压缩时保留的 token 数量
            # 压缩后会话保留的 token 数量（相对于最大 token 的百分比）
            retainPercent = 0.5
        }
        
        # ==================== 并发控制配置 ====================
        concurrency {
            # 最大并发请求数
            # 对 LM Studio 同时发起的 AI 模型请求数量上限
            maxConcurrentRequests = 3
            
            # 是否启用队列
            # 超过最大并发数的请求将进入队列等待
            queueEnabled = true
            
            # 队列超时时间（毫秒）
            # 请求在队列中等待的最长时间，超时后抛出异常
            queueTimeout = 300000  # 5 分钟
            
            # 队列模式
            # - fifo: 先到先得（推荐）
            # - priority: 优先级队列（待实现）
            queueMode = "fifo"
            
            # 队列容量限制
            # - 0: 无限制（推荐）
            # - >0: 最大队列长度，超过后新请求被拒绝
            queueCapacity = 0
            
            # 队列监控
            # 启用队列状态日志
            enableQueueMonitoring = true
            
            # 队列等待警告阈值（毫秒）
            # 当请求等待时间超过此值时发出警告日志
            queueWarningThreshold = 10000  # 10 秒
        }
        
        # ==================== 重试配置 ====================
        retry {
            # 是否启用重试
            enabled = true
            
            # 最大重试次数
            maxRetries = 3
            
            # 重试延迟（毫秒）
            retryDelay = 1000
            
            # 重试延迟增长倍数
            # 第二次重试延迟 = retryDelay * multiplier
            # 第三次重试延迟 = retryDelay * multiplier^2
            multiplier = 2.0
            
            # 重试的 HTTP 状态码
            retryableStatusCodes = [429, 500, 502, 503, 504]
        }
        
        # ==================== 超时配置 ====================
        timeout {
            # 请求超时（毫秒）
            requestTimeout = 60000  # 1 分钟
            
            # 连接超时（毫秒）
            connectTimeout = 10000  # 10 秒
            
            # 队列等待超时（毫秒）
            # 优先级高于 queueTimeout
            queueWaitTimeout = 300000  # 5 分钟
        }
    }
}
```

## 配置项详解

### 会话压缩配置

#### thresholdPercent (推荐配置)
**类型**: Double (0.0 - 1.0)  
**默认值**: 0.8  
**说明**: 当会话 token 使用量达到模型最大 token 的此百分比时，触发自动压缩。

**示例**:
- 模型最大 token: 32768
- thresholdPercent: 0.8
- 触发阈值: 32768 × 0.8 = 26214 tokens

#### minThresholdTokens (可选)
**类型**: Int  
**默认值**: 无  
**说明**: 最小触发阈值，即使百分比计算结果更低，也不会低于此值。

**使用场景**: 防止小模型（如 4096 tokens）的阈值过低

**示例**:
- 模型最大 token: 4096
- thresholdPercent: 0.8 → 计算阈值: 3276
- minThresholdTokens: 5000
- 实际阈值: max(3276, 5000) = 5000

#### maxThresholdTokens (可选)
**类型**: Int  
**默认值**: 无  
**说明**: 最大触发阈值，即使百分比计算结果更高，也不会超过此值。

**使用场景**: 防止大模型的阈值过高，导致内存占用过大

#### strategy (压缩策略)
**类型**: String  
**可选值**: 
- `summarize`: 使用 AI 对早期对话进行摘要（推荐）
- `truncate`: 直接截断最早的对话
- `smart`: 智能压缩（保留重要信息，如代码、关键事实）

**推荐**: 使用 `summarize` 策略，保持对话连贯性

### 并发控制配置

#### maxConcurrentRequests (核心配置)
**类型**: Int (≥ 1)  
**默认值**: 3  
**说明**: 对 LM Studio 同时发起的最大 AI 模型请求数量。

**推荐值**:
- 本地开发：2-3
- 生产环境（单用户）：3-5
- 生产环境（多用户）：5-10（根据服务器性能）

**影响**:
- 值越大：并发越高，但可能导致 LM Studio 响应变慢或失败
- 值越小：并发越低，但系统更稳定

#### queueEnabled
**类型**: Boolean  
**默认值**: true  
**说明**: 是否启用请求队列。超过最大并发数的请求进入队列等待。

**建议**: 始终启用，避免请求被直接拒绝

#### queueTimeout
**类型**: Long (毫秒)  
**默认值**: 300000 (5 分钟)  
**说明**: 请求在队列中等待的最长时间。

**示例**:
- 请求进入队列
- 等待 5 分钟后仍未获得执行机会
- 抛出 `QueueTimeoutException` 异常

#### queueMode
**类型**: String  
**可选值**: 
- `fifo`: First In First Out（先到先得）
- `priority`: 优先级队列（高优先级请求优先）

**推荐**: 使用 `fifo`，保证公平性

#### queueCapacity
**类型**: Int (≥ 0)  
**默认值**: 0 (无限制)  
**说明**: 队列最大容量。

**示例**:
- 0: 无限制（推荐）
- 100: 队列最多容纳 100 个请求，超过后新请求被拒绝

**建议**: 使用无限制队列，配合 queueTimeout 控制等待时间

#### enableQueueMonitoring
**类型**: Boolean  
**默认值**: true  
**说明**: 启用队列状态监控日志。

**日志示例**:
```
[Queue Monitor] 当前队列长度：5
[Queue Monitor] 平均等待时间：3.2s
[Queue Monitor] 最大等待时间：12.5s
```

### 重试配置

#### maxRetries
**类型**: Int (≥ 0)  
**默认值**: 3  
**说明**: 请求失败时的最大重试次数。

**推荐**: 2-3 次，避免过度重试

#### retryableStatusCodes
**类型**: List<Int>  
**默认值**: [429, 500, 502, 503, 504]  
**说明**: 触发重试的 HTTP 状态码。

**说明**:
- 429: Too Many Requests（请求过多）
- 500: Internal Server Error（服务器内部错误）
- 502: Bad Gateway（网关错误）
- 503: Service Unavailable（服务不可用）
- 504: Gateway Timeout（网关超时）

## 使用示例

### 示例 1: 开发环境配置
```hocon
aiService {
    # 开发环境：低并发，快速失败
    concurrency {
        maxConcurrentRequests = 2
        queueTimeout = 60000  # 1 分钟
    }
    
    sessionCompression {
        thresholdPercent = 0.7  # 更早触发压缩
    }
}
```

### 示例 2: 生产环境配置（多用户）
```hocon
aiService {
    # 生产环境：高并发，长等待
    concurrency {
        maxConcurrentRequests = 10
        queueTimeout = 600000  # 10 分钟
        queueCapacity = 100
    }
    
    sessionCompression {
        thresholdPercent = 0.85  # 稍晚触发压缩
        strategy = "smart"
    }
}
```

### 示例 3: 保守配置（稳定性优先）
```hocon
aiService {
    # 保守配置：低并发，严格超时
    concurrency {
        maxConcurrentRequests = 1  # 串行请求
        queueTimeout = 300000
        queueWarningThreshold = 5000  # 5 秒警告
    }
    
    retry {
        maxRetries = 5  # 更多重试次数
        retryDelay = 2000
    }
}
```

## 配置验证

### 最小可用配置
```hocon
aiService {
    # 仅配置必需项，其他使用默认值
    concurrency {
        maxConcurrentRequests = 3
    }
    
    sessionCompression {
        thresholdPercent = 0.8
    }
}
```

### 完整配置
见上文完整示例

## 注意事项

1. **模型最大 token 获取**:
   - 优先从 LM Studio API 自动获取
   - 如果获取失败，使用配置文件中的 `modelMaxTokens`
   - 如果都未配置，使用保守默认值（8192）

2. **并发数选择**:
   - 根据 LM Studio 服务器性能调整
   - 监控请求失败率和响应时间
   - 逐步增加并发数直到找到最佳值

3. **压缩策略选择**:
   - `summarize`: 保持对话连贯性，但需要额外 AI 调用
   - `truncate`: 快速简单，但可能丢失重要信息
   - `smart`: 平衡效果和性能（推荐）

4. **队列超时设置**:
   - 设置合理的超时时间，避免用户等待过久
   - 配合重试机制，提高成功率
   - 监控队列等待时间，及时告警
