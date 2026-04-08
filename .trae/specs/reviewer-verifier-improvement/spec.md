# 规范：审查器和校验器改进

## Why

当前审查器的direction检查存在以下问题：
1. 基于简单关键词匹配，无法理解语义
2. 可能误报（模型输出是工具调用时不包含任务关键词但仍然正确）

当前校验器的JS/HTML检查已实现但未启用。

## 目标

1. 改进direction检查：只在特定场景下启用（输出过短或无工具调用时）
2. 启用JavaScript语法检查
3. 启用HTML结构检查

## 技术方案

### 改进1：智能direction检查

修改`checkDirection`方法：
- 如果输出包含工具调用，跳过direction检查
- 如果输出长度过短（<100字符），启用direction检查
- 如果输出长度适中但有足够内容，降低检查频率

### 改进2：启用JS/HTML校验

修改`VerifyTaskCompletion`方法：
- 在文件内容检查后，执行JavaScript语法检查
- 执行HTML结构检查
- 将检查结果添加到校验结果中

### 改进3：配置化

在VerifyConfig中添加：
- `EnableJSSyntaxCheck bool` - 启用JS语法检查
- `EnableHTMLStructureCheck bool` - 启用HTML结构检查

## 实现范围

### 文件修改

1. `internal/agent/reviewer.go` - 改进direction检查
2. `internal/agent/verifier.go` - 启用JS/HTML检查

## 验收标准

1. direction检查不再误报
2. JS语法错误能被检测
3. HTML结构错误能被检测
4. 所有测试通过
