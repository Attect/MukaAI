# 任务列表

## 任务1：修改core.go添加强制校验

**状态**: pending

**内容**:
- 在Run方法返回前强制执行校验
- 校验失败时不返回completed状态
- 注入修正指令继续执行

## 任务2：分离重试计数

**状态**: pending

**内容**:
- 新增reviewRetryCount和verifyRetryCount
- 新增maxReviewRetries和maxVerifyRetries配置
- 修改ShouldRetry方法支持不同类型
- 新增RecordReviewFailure和RecordVerifyFailure方法

## 任务3：更新测试

**状态**: pending

**内容**:
- 更新selfcorrector_test.go
- 添加分离重试计数的测试用例
- 运行所有测试确保通过

## 任务4：集成测试

**状态**: pending

**内容**:
- 运行模糊需求测试
- 验证改进效果
- 更新验收清单
