# 规范：校验机制改进

## Why

当前校验机制存在两个问题：

1. **校验时机问题**：校验仅在`complete_task`工具执行时触发，如果Agent绕过该工具直接结束任务，校验机制将不会生效。

2. **重试机制问题**：审查阻断和校验失败共享同一个重试计数器，可能导致审查阻断消耗了大部分重试次数，真正的校验失败没有足够机会修正。

## 目标

1. 在Agent任务结束时强制执行校验，确保所有任务都经过校验
2. 分离审查重试和校验重试的计数，为不同类型失败设置独立配额

## 技术方案

### 改进1：强制校验

修改`internal/agent/core.go`的Run方法：

1. 在任务完成检测时（`isTaskComplete`）加入校验逻辑
2. 在Run方法返回前，如果状态为completed，强制执行一次校验
3. 校验失败时，不返回completed状态，而是注入修正指令继续执行

### 改进2：分离重试计数

修改`internal/agent/selfcorrector.go`：

1. 新增`reviewRetryCount`和`verifyRetryCount`分别跟踪
2. 新增`maxReviewRetries`和`maxVerifyRetries`配置
3. 修改`ShouldRetry`方法，根据失败类型使用不同计数器
4. 新增`RecordReviewFailure`和`RecordVerifyFailure`方法

## 实现范围

### 文件修改

1. `internal/agent/core.go` - 添加强制校验逻辑
2. `internal/agent/selfcorrector.go` - 分离重试计数
3. `internal/agent/selfcorrector_test.go` - 更新测试

## 验收标准

1. 任务结束时强制执行校验
2. 校验失败不返回completed状态
3. 审查和校验有独立的重试配额
4. 所有测试通过
