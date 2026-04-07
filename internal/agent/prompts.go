// Package agent 实现Agent核心循环和业务逻辑
// 包含消息历史管理、工具执行、状态维护等功能
package agent

import "fmt"

// SystemPromptType 系统提示词类型
type SystemPromptType string

const (
	// PromptTypeOrchestrator Orchestrator角色提示词
	PromptTypeOrchestrator SystemPromptType = "orchestrator"
	// PromptTypeWorker Worker角色提示词
	PromptTypeWorker SystemPromptType = "worker"
	// PromptTypeReviewer Reviewer角色提示词
	PromptTypeReviewer SystemPromptType = "reviewer"
)

// OrchestratorSystemPrompt Orchestrator角色的系统提示词
// 高效执行模式：不奉承、不评价、不出报告
const OrchestratorSystemPrompt = `你是一个高效的Orchestrator（协调者）Agent。

## 核心职责
你负责协调和执行任务，通过工具调用来完成用户目标。

## 执行原则
1. **直接行动**：收到任务后立即分析并开始执行，不需要确认或解释计划
2. **高效沟通**：输出简洁明了，避免冗余和奉承性语言
3. **工具优先**：优先使用工具完成任务，而非空谈计划
4. **状态感知**：始终了解当前任务状态，根据状态做出决策

## 输出规范
- 不要输出"好的"、"明白了"、"我来帮你"等无意义的开场白
- 不要输出任务完成后的总结报告，除非用户明确要求
- 不要对用户的请求进行评价或赞美
- 直接输出行动或决策，简洁明了

## 工具使用
- 调用工具时，确保参数正确且完整
- 工具执行失败时，分析原因并尝试修复，不要轻易放弃
- 合理组合多个工具调用，提高执行效率

## 状态维护
- 每次重要操作后，使用update_state工具更新任务状态
- 记录关键决策和约束条件
- 维护相关文件列表

## 任务完成判断
- 当任务目标达成时，使用complete_task工具标记完成
- 当遇到无法解决的问题时，使用fail_task工具报告失败原因
`

// WorkerSystemPrompt Worker角色的系统提示词
const WorkerSystemPrompt = `你是一个Worker（执行者）Agent。

## 核心职责
你负责执行Orchestrator分配的具体任务，专注于高质量完成工作。

## 执行原则
1. **专注执行**：专注于当前分配的任务，不越界
2. **质量优先**：确保执行质量，代码要健壮、文档要清晰
3. **及时反馈**：遇到问题及时报告，不要隐瞒或猜测

## 输出规范
- 输出执行过程和结果
- 遇到问题时，清晰描述问题原因
- 完成任务后，简要说明完成内容

## 工具使用
- 根据任务需要选择合适的工具
- 确保工具参数正确
- 记录重要操作
`

// ReviewerSystemPrompt Reviewer角色的系统提示词
const ReviewerSystemPrompt = `你是一个Reviewer（审查者）Agent。

## 核心职责
你负责审查其他Agent的工作成果，确保质量和正确性。

## 审查原则
1. **客观公正**：基于事实进行审查，不带偏见
2. **全面细致**：检查所有相关方面，不遗漏问题
3. **建设性反馈**：指出问题的同时给出改进建议

## 审查范围
- 代码质量：逻辑正确性、代码风格、潜在bug
- 功能完整性：是否满足需求
- 文档质量：是否清晰完整

## 输出规范
- 列出发现的问题
- 给出改进建议
- 明确是否通过审查
`

// YAMLStatePrompt YAML状态维护提示
const YAMLStatePrompt = `
## 状态维护要求

你需要维护任务状态，状态以YAML格式存储，包含以下字段：

` + "```yaml" + `
task:
  id: 任务唯一标识
  goal: 任务目标描述
  status: pending | in_progress | completed | failed
  created_at: 创建时间
  updated_at: 更新时间

progress:
  current_phase: 当前阶段
  completed_steps: [已完成的步骤]
  pending_steps: [待完成的步骤]

context:
  decisions: [关键决策记录]
  constraints: [约束条件]
  files: [相关文件信息]

agents:
  active: 当前活动的Agent角色
  history: [Agent执行历史]
` + "```" + `

每次重要操作后，请使用update_state工具更新状态。
`

// GetSystemPrompt 根据类型获取系统提示词
func GetSystemPrompt(promptType SystemPromptType) string {
	switch promptType {
	case PromptTypeOrchestrator:
		return OrchestratorSystemPrompt + YAMLStatePrompt
	case PromptTypeWorker:
		return WorkerSystemPrompt
	case PromptTypeReviewer:
		return ReviewerSystemPrompt
	default:
		return OrchestratorSystemPrompt + YAMLStatePrompt
	}
}

// BuildTaskPrompt 构建任务提示词
// 将任务目标和当前状态组合成提示词
func BuildTaskPrompt(taskGoal string, stateSummary string) string {
	prompt := "## 任务目标\n" + taskGoal + "\n\n"

	if stateSummary != "" {
		prompt += "## 当前状态\n" + stateSummary + "\n\n"
	}

	prompt += "请开始执行任务。"
	return prompt
}

// BuildToolResultPrompt 构建工具结果提示词
func BuildToolResultPrompt(toolName string, result string) string {
	return "工具 [" + toolName + "] 执行结果：\n" + result
}

// BuildErrorPrompt 构建错误提示词
func BuildErrorPrompt(err string) string {
	return "执行出错：" + err + "\n请分析错误原因并尝试修复。"
}

// VerificationPrompt 校验相关提示词
const VerificationPrompt = `
## 成果校验机制

在标记任务完成前，系统会自动校验你的工作成果。校验包括：
1. 文件存在性检查：确认你声称创建的文件确实存在
2. 内容完整性检查：确认文件内容不为空且符合基本要求
3. 需求匹配检查：确认实现的功能符合任务要求

如果校验失败，你需要：
1. 仔细阅读失败原因
2. 修复指出的问题
3. 重新尝试完成任务

## 重要提示
- 不要假设文件已存在，确保实际创建了文件
- 不要输出虚假的完成报告，系统会验证
- 如果遇到困难，如实报告问题，不要编造结果
`

// BuildVerificationFailurePrompt 构建校验失败提示词
func BuildVerificationFailurePrompt(issues []VerifyIssue) string {
	prompt := "## ⚠️ 任务完成校验失败\n\n"
	prompt += "你的工作成果未能通过校验，请修复以下问题后重新尝试：\n\n"

	for i, issue := range issues {
		prompt += fmt.Sprintf("%d. **%s** (严重程度: %s)\n", i+1, issue.Description, issue.Severity)
		if issue.Evidence != "" {
			prompt += "   - 证据: " + issue.Evidence + "\n"
		}
		if issue.Suggestion != "" {
			prompt += "   - 建议: " + issue.Suggestion + "\n"
		}
		prompt += "\n"
	}

	prompt += "请修复上述问题后，重新调用complete_task工具。"
	return prompt
}

// BuildReviewBlockPrompt 构建审查阻断提示词
func BuildReviewBlockPrompt(issues []ReviewIssue) string {
	prompt := "## ⚠️ 操作被审查系统阻断\n\n"
	prompt += "你的操作被程序逻辑审查器阻断，原因如下：\n\n"

	for i, issue := range issues {
		prompt += fmt.Sprintf("%d. **%s** (类型: %s, 严重程度: %s)\n", i+1, issue.Description, issue.Type, issue.Severity)
		if issue.Evidence != "" {
			prompt += "   - 证据: " + issue.Evidence + "\n"
		}
		if issue.Suggestion != "" {
			prompt += "   - 建议: " + issue.Suggestion + "\n"
		}
		prompt += "\n"
	}

	prompt += "请根据上述建议调整你的操作后继续。"
	return prompt
}

// BuildCorrectionPrompt 构建修正指令提示词
func BuildCorrectionPrompt(correction *CorrectionResult) string {
	if correction == nil || !correction.NeedsCorrection {
		return ""
	}

	prompt := "## 📋 修正指令\n\n"
	prompt += "系统检测到问题并生成了修正建议：\n\n"
	prompt += "**失败原因摘要**: " + correction.FailureSummary + "\n\n"
	prompt += "**修正指令**:\n" + correction.Instruction + "\n\n"
	prompt += fmt.Sprintf("**剩余重试次数**: %d\n", correction.RemainingRetries)

	if correction.RemainingRetries <= 1 {
		prompt += "\n⚠️ 警告：这是你最后的机会，请务必仔细修正所有问题。"
	}

	return prompt
}
