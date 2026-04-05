// Package agent 实现Agent核心循环和业务逻辑
package agent

import (
	"fmt"
	"strings"
	"time"

	"agentplus/internal/model"
)

// FeedbackLevel 反馈级别
type FeedbackLevel string

const (
	// FeedbackLevelInfo 信息级别
	FeedbackLevelInfo FeedbackLevel = "info"
	// FeedbackLevelWarning 警告级别
	FeedbackLevelWarning FeedbackLevel = "warning"
	// FeedbackLevelError 错误级别
	FeedbackLevelError FeedbackLevel = "error"
	// FeedbackLevelCritical 严重错误级别
	FeedbackLevelCritical FeedbackLevel = "critical"
)

// FeedbackMessage 反馈消息
type FeedbackMessage struct {
	Level       FeedbackLevel `json:"level"`       // 反馈级别
	Title       string        `json:"title"`       // 反馈标题
	Content     string        `json:"content"`     // 反馈内容
	Suggestions []string      `json:"suggestions"` // 修正建议列表
	Timestamp   time.Time     `json:"timestamp"`   // 时间戳
}

// ToUserMessage 将反馈转换为用户消息
func (f *FeedbackMessage) ToUserMessage() model.Message {
	var sb strings.Builder

	// 标题
	sb.WriteString(fmt.Sprintf("## [%s] %s\n\n", strings.ToUpper(string(f.Level)), f.Title))

	// 内容
	sb.WriteString(f.Content)
	sb.WriteString("\n\n")

	// 建议
	if len(f.Suggestions) > 0 {
		sb.WriteString("### 修正建议\n")
		for i, suggestion := range f.Suggestions {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, suggestion))
		}
		sb.WriteString("\n")
	}

	// 结束提示
	sb.WriteString("请根据以上反馈调整你的行为，继续执行任务。")

	return model.NewUserMessage(sb.String())
}

// FeedbackInjector 审查反馈注入器
// 负责将审查结果转换为反馈消息并注入到对话中
type FeedbackInjector struct {
	// 配置
	maxFeedbackLength int  // 最大反馈长度
	includeEvidence   bool // 是否包含证据
	includeTimestamp  bool // 是否包含时间戳
}

// FeedbackInjectorConfig 反馈注入器配置
type FeedbackInjectorConfig struct {
	MaxFeedbackLength int  // 最大反馈长度（默认500）
	IncludeEvidence   bool // 是否包含证据（默认true）
	IncludeTimestamp  bool // 是否包含时间戳（默认false）
}

// NewFeedbackInjector 创建新的反馈注入器
func NewFeedbackInjector(config *FeedbackInjectorConfig) *FeedbackInjector {
	if config == nil {
		config = &FeedbackInjectorConfig{
			MaxFeedbackLength: 500,
			IncludeEvidence:   true,
			IncludeTimestamp:  false,
		}
	}

	if config.MaxFeedbackLength <= 0 {
		config.MaxFeedbackLength = 500
	}

	return &FeedbackInjector{
		maxFeedbackLength: config.MaxFeedbackLength,
		includeEvidence:   config.IncludeEvidence,
		includeTimestamp:  config.IncludeTimestamp,
	}
}

// InjectFeedback 根据审查结果生成反馈消息
// 返回用户消息，用于注入到对话历史中
func (fi *FeedbackInjector) InjectFeedback(result *ReviewResult) model.Message {
	if result == nil || len(result.Issues) == 0 {
		return model.Message{}
	}

	// 根据审查状态确定反馈级别
	level := fi.determineFeedbackLevel(result)

	// 生成反馈消息
	feedback := fi.buildFeedbackMessage(result, level)

	return feedback.ToUserMessage()
}

// InjectFeedbackForIssue 为单个问题生成反馈消息
func (fi *FeedbackInjector) InjectFeedbackForIssue(issue ReviewIssue) model.Message {
	level := fi.issueSeverityToFeedbackLevel(issue.Severity)

	feedback := &FeedbackMessage{
		Level:       level,
		Title:       fi.getIssueTitle(issue.Type),
		Content:     fi.buildIssueContent(issue),
		Suggestions: []string{issue.Suggestion},
		Timestamp:   time.Now(),
	}

	return feedback.ToUserMessage()
}

// InjectBlockingFeedback 生成阻断级别的反馈
// 用于需要立即修正的情况
func (fi *FeedbackInjector) InjectBlockingFeedback(result *ReviewResult) model.Message {
	blockingIssues := result.GetBlockingIssues()
	if len(blockingIssues) == 0 {
		return model.Message{}
	}

	var sb strings.Builder
	sb.WriteString("## [CRITICAL] 执行被阻断\n\n")
	sb.WriteString("检测到严重问题，需要立即修正：\n\n")

	for i, issue := range blockingIssues {
		sb.WriteString(fmt.Sprintf("### 问题 %d: %s\n", i+1, fi.getIssueTitle(issue.Type)))
		sb.WriteString(fmt.Sprintf("**描述**: %s\n\n", issue.Description))

		if fi.includeEvidence && issue.Evidence != "" {
			sb.WriteString(fmt.Sprintf("**证据**: %s\n\n", issue.Evidence))
		}

		sb.WriteString(fmt.Sprintf("**建议**: %s\n\n", issue.Suggestion))
	}

	sb.WriteString("---\n")
	sb.WriteString("请立即修正以上问题后再继续执行任务。")

	return model.NewUserMessage(sb.String())
}

// InjectWarningFeedback 生成警告级别的反馈
// 用于提醒但不阻断执行
func (fi *FeedbackInjector) InjectWarningFeedback(result *ReviewResult) model.Message {
	if result.Status != ReviewStatusWarning {
		return model.Message{}
	}

	var sb strings.Builder
	sb.WriteString("## [WARNING] 执行警告\n\n")
	sb.WriteString("检测到以下问题，请注意：\n\n")

	for i, issue := range result.Issues {
		sb.WriteString(fmt.Sprintf("%d. **%s**: %s\n", i+1, fi.getIssueTitle(issue.Type), issue.Description))
		if issue.Suggestion != "" {
			sb.WriteString(fmt.Sprintf("   - 建议: %s\n", issue.Suggestion))
		}
	}

	sb.WriteString("\n请考虑以上建议，但可以继续执行任务。")

	return model.NewUserMessage(sb.String())
}

// InjectProgressFeedback 生成进度相关的反馈
func (fi *FeedbackInjector) InjectProgressFeedback(iterationCount int, maxIterations int) model.Message {
	var sb strings.Builder

	sb.WriteString("## [INFO] 进度提醒\n\n")
	sb.WriteString(fmt.Sprintf("当前已迭代 %d 次（最大 %d 次）。\n\n", iterationCount, maxIterations))

	if iterationCount > maxIterations/2 {
		sb.WriteString("**注意**: 已超过最大迭代次数的一半，请评估当前方法的有效性。\n\n")
		sb.WriteString("建议：\n")
		sb.WriteString("1. 检查是否需要调整策略\n")
		sb.WriteString("2. 考虑分解任务为更小的步骤\n")
		sb.WriteString("3. 如果任务已完成，请使用 complete_task 工具\n")
	}

	return model.NewUserMessage(sb.String())
}

// InjectLoopDetectedFeedback 生成循环检测反馈
func (fi *FeedbackInjector) InjectLoopDetectedFeedback(toolName, arguments string, repeatCount int) model.Message {
	var sb strings.Builder

	sb.WriteString("## [CRITICAL] 检测到无限循环\n\n")
	sb.WriteString(fmt.Sprintf("工具 `%s` 已重复调用 %d 次，使用相同参数。\n\n", toolName, repeatCount))
	sb.WriteString(fmt.Sprintf("**参数**: %s\n\n", arguments))
	sb.WriteString("**可能原因**:\n")
	sb.WriteString("1. 工具执行失败但未正确处理错误\n")
	sb.WriteString("2. 参数设置不正确\n")
	sb.WriteString("3. 期望的结果无法通过当前方法获得\n\n")
	sb.WriteString("**建议**:\n")
	sb.WriteString("1. 检查工具参数是否正确\n")
	sb.WriteString("2. 尝试不同的方法或工具\n")
	sb.WriteString("3. 如果确实需要重复操作，请修改参数\n")

	return model.NewUserMessage(sb.String())
}

// InjectFailureFeedback 生成失败反馈
func (fi *FeedbackInjector) InjectFailureFeedback(toolName string, consecutiveFailures int, lastError string) model.Message {
	var sb strings.Builder

	sb.WriteString("## [ERROR] 连续失败警告\n\n")
	sb.WriteString(fmt.Sprintf("工具 `%s` 已连续失败 %d 次。\n\n", toolName, consecutiveFailures))
	sb.WriteString(fmt.Sprintf("**最后错误**: %s\n\n", lastError))
	sb.WriteString("**建议**:\n")
	sb.WriteString("1. 检查工具参数是否正确\n")
	sb.WriteString("2. 确认前置条件是否满足\n")
	sb.WriteString("3. 考虑使用其他方法完成任务\n")
	sb.WriteString("4. 如果问题持续，请报告任务失败\n")

	return model.NewUserMessage(sb.String())
}

// InjectDirectionFeedback 生成方向偏离反馈
func (fi *FeedbackInjector) InjectDirectionFeedback(goal string, output string) model.Message {
	var sb strings.Builder

	sb.WriteString("## [WARNING] 可能偏离任务目标\n\n")
	sb.WriteString(fmt.Sprintf("**任务目标**: %s\n\n", goal))
	sb.WriteString(fmt.Sprintf("**当前输出**: %s\n\n", truncateString(output, 200)))
	sb.WriteString("**建议**:\n")
	sb.WriteString("1. 重新审视任务目标\n")
	sb.WriteString("2. 确保当前操作与目标相关\n")
	sb.WriteString("3. 避免执行无关的任务\n")

	return model.NewUserMessage(sb.String())
}

// determineFeedbackLevel 根据审查结果确定反馈级别
func (fi *FeedbackInjector) determineFeedbackLevel(result *ReviewResult) FeedbackLevel {
	switch result.Status {
	case ReviewStatusBlock:
		return FeedbackLevelCritical
	case ReviewStatusWarning:
		return FeedbackLevelWarning
	default:
		return FeedbackLevelInfo
	}
}

// buildFeedbackMessage 构建反馈消息
func (fi *FeedbackInjector) buildFeedbackMessage(result *ReviewResult, level FeedbackLevel) *FeedbackMessage {
	// 收集所有建议
	suggestions := make([]string, 0)
	for _, issue := range result.Issues {
		if issue.Suggestion != "" {
			suggestions = append(suggestions, issue.Suggestion)
		}
	}

	// 构建内容
	content := fi.buildContent(result)

	return &FeedbackMessage{
		Level:       level,
		Title:       fi.getTitle(result),
		Content:     content,
		Suggestions: suggestions,
		Timestamp:   time.Now(),
	}
}

// buildContent 构建反馈内容
func (fi *FeedbackInjector) buildContent(result *ReviewResult) string {
	var sb strings.Builder

	for _, issue := range result.Issues {
		sb.WriteString(fmt.Sprintf("- **%s**: %s", fi.getIssueTitle(issue.Type), issue.Description))

		if fi.includeEvidence && issue.Evidence != "" {
			sb.WriteString(fmt.Sprintf("\n  证据: %s", issue.Evidence))
		}

		sb.WriteString("\n")
	}

	// 限制长度
	content := sb.String()
	if len(content) > fi.maxFeedbackLength {
		content = content[:fi.maxFeedbackLength] + "..."
	}

	return content
}

// getTitle 获取反馈标题
func (fi *FeedbackInjector) getTitle(result *ReviewResult) string {
	switch result.Status {
	case ReviewStatusBlock:
		return "执行被阻断，需要修正"
	case ReviewStatusWarning:
		return "检测到潜在问题"
	default:
		return "审查反馈"
	}
}

// getIssueTitle 获取问题标题
func (fi *FeedbackInjector) getIssueTitle(issueType IssueType) string {
	titles := map[IssueType]string{
		IssueTypeDirection:       "方向偏离",
		IssueTypeInfiniteLoop:    "无限循环",
		IssueTypeInvalidToolCall: "无效工具调用",
		IssueTypeRepeatedFailure: "重复失败",
		IssueTypeFabrication:     "编造内容",
		IssueTypeNoProgress:      "无进度",
	}

	if title, ok := titles[issueType]; ok {
		return title
	}
	return "未知问题"
}

// buildIssueContent 构建单个问题的内容
func (fi *FeedbackInjector) buildIssueContent(issue ReviewIssue) string {
	var sb strings.Builder

	sb.WriteString(issue.Description)
	sb.WriteString("\n\n")

	if fi.includeEvidence && issue.Evidence != "" {
		sb.WriteString("**证据**: ")
		sb.WriteString(issue.Evidence)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// issueSeverityToFeedbackLevel 将问题严重度转换为反馈级别
func (fi *FeedbackInjector) issueSeverityToFeedbackLevel(severity string) FeedbackLevel {
	switch severity {
	case "critical":
		return FeedbackLevelCritical
	case "high":
		return FeedbackLevelError
	case "medium":
		return FeedbackLevelWarning
	default:
		return FeedbackLevelInfo
	}
}

// BatchInjectFeedback 批量生成反馈消息
// 用于处理多个审查结果
func (fi *FeedbackInjector) BatchInjectFeedback(results []*ReviewResult) []model.Message {
	messages := make([]model.Message, 0)

	for _, result := range results {
		if result == nil || len(result.Issues) == 0 {
			continue
		}

		msg := fi.InjectFeedback(result)
		if msg.Content != "" {
			messages = append(messages, msg)
		}
	}

	return messages
}

// FormatFeedbackForLog 格式化反馈用于日志记录
func FormatFeedbackForLog(result *ReviewResult) string {
	if result == nil {
		return "无审查结果"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s] %s\n", result.Status, result.Summary))

	for _, issue := range result.Issues {
		sb.WriteString(fmt.Sprintf("  - %s (%s): %s\n", issue.Type, issue.Severity, issue.Description))
	}

	return sb.String()
}
