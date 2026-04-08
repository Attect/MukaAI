# 任务列表

## 任务1：修改core.go添加强制校验

**状态**: completed

**内容**:
- [x] 在Run方法返回前强制执行校验
- [x] 校验失败时不返回completed状态
- [x] 注入修正指令继续执行

## 任务2：分离重试计数

**状态**: completed

**内容**:
- [x] 新增reviewRetryCount和verifyRetryCount
- [x] 新增maxReviewRetries和maxVerifyRetries配置
- [x] 修改ShouldRetry方法支持不同类型
- [x] 新增RecordReviewFailure和RecordVerifyFailure方法

## 任务3：更新测试

**状态**: completed

**内容**:
- [x] 更新selfcorrector_test.go
- [x] 添加分离重试计数的测试用例
- [x] 运行所有测试确保通过

## 任务4：集成测试

**状态**: completed

**内容**:
- [x] 运行模糊需求测试
- [x] 验证改进效果
- [x] 更新验收清单

# 执行日志

## 实施记录

### 2026-04-07 实施

1. **修改core.go**
   - 添加`verificationPassed`字段跟踪校验状态
   - 修改Run方法使用双层循环结构
   - 任务完成时不设置`verificationPassed`，让外层循环执行强制校验
   - 强制校验通过后才设置`verificationPassed = true`

2. **修改selfcorrector.go**
   - 添加`MaxReviewRetries`和`MaxVerifyRetries`配置
   - 添加`reviewRetryCount`和`verifyRetryCount`计数器
   - 新增`ShouldRetryReview()`和`ShouldRetryVerify()`方法
   - 新增`RecordReviewFailure()`和`RecordVerifyFailure()`方法

3. **修复逻辑问题**
   - 发现`verificationPassed`在内层循环被设置为true，导致强制校验不执行
   - 修复：移除内层循环的设置，只在强制校验通过后设置

4. **调整测试配置**
   - 增加审查重试次数到10次
   - 增加校验重试次数到10次

### 测试结果

- **状态**: completed
- **迭代次数**: 9次
- **修正次数**: 0次
- **关键词验证**: 全部通过

## 结论

校验机制改进已成功实施。系统现在能够：
1. 在任务完成时强制执行校验
2. 审查和校验使用独立的重试配额
3. 确保只有通过校验的任务才能标记完成
