// Package agent 实现Agent核心循环和业务逻辑
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"agentplus/internal/model"
	"agentplus/internal/state"
)

// ReviewStatus 审查结果状态
type ReviewStatus string

const (
	// ReviewStatusPass 审查通过，继续执行
	ReviewStatusPass ReviewStatus = "pass"
	// ReviewStatusWarning 警告，记录但继续执行
	ReviewStatusWarning ReviewStatus = "warning"
	// ReviewStatusBlock 阻断，需要修正
	ReviewStatusBlock ReviewStatus = "block"
)

// IssueType 问题类型
type IssueType string

const (
	// IssueTypeDirection 方向偏离
	IssueTypeDirection IssueType = "direction"
	// IssueTypeInfiniteLoop 无限循环
	IssueTypeInfiniteLoop IssueType = "infinite_loop"
	// IssueTypeInvalidToolCall 无效工具调用
	IssueTypeInvalidToolCall IssueType = "invalid_tool_call"
	// IssueTypeRepeatedFailure 重复失败
	IssueTypeRepeatedFailure IssueType = "repeated_failure"
	// IssueTypeFabrication 编造内容
	IssueTypeFabrication IssueType = "fabrication"
	// IssueTypeNoProgress 无进度
	IssueTypeNoProgress IssueType = "no_progress"
)

// ReviewIssue 审查发现的问题
type ReviewIssue struct {
	Type        IssueType `json:"type"`         // 问题类型
	Severity    string    `json:"severity"`     // 严重程度：low, medium, high, critical
	Description string    `json:"description"`  // 问题描述
	Evidence    string    `json:"evidence"`     // 证据/示例
	Suggestion  string    `json:"suggestion"`   // 修正建议
	ToolName    string    `json:"tool_name"`    // 相关工具名称（如果有）
	Timestamp   time.Time `json:"timestamp"`    // 发现时间
}

// ReviewResult 审查结果
type ReviewResult struct {
	Status    ReviewStatus  `json:"status"`     // 审查状态
	Issues    []ReviewIssue `json:"issues"`     // 发现的问题列表
	Timestamp time.Time     `json:"timestamp"`  // 审查时间
	Summary   string        `json:"summary"`    // 审查摘要
}

// IsBlocked 检查是否被阻断
func (r *ReviewResult) IsBlocked() bool {
	return r.Status == ReviewStatusBlock
}

// HasWarnings 检查是否有警告
func (r *ReviewResult) HasWarnings() bool {
	return r.Status == ReviewStatusWarning || len(r.Issues) > 0
}

// GetBlockingIssues 获取阻断级别的问题
func (r *ReviewResult) GetBlockingIssues() []ReviewIssue {
	var blocking []ReviewIssue
	for _, issue := range r.Issues {
		if issue.Severity == "high" || issue.Severity == "critical" {
			blocking = append(blocking, issue)
		}
	}
	return blocking
}

// ReviewConfig 审查器配置
type ReviewConfig struct {
	// 方向偏离检测
	EnableDirectionCheck bool `json:"enable_direction_check"` // 是否启用方向偏离检测

	// 循环检测
	MaxRepeatedActions int `json:"max_repeated_actions"` // 相同操作最大重复次数
	LoopWindowSize     int `json:"loop_window_size"`     // 循环检测窗口大小

	// 失败检测
	MaxConsecutiveFailures int `json:"max_consecutive_failures"` // 最大连续失败次数
	FailureResetInterval   int `json:"failure_reset_interval"`   // 失败计数重置间隔（秒）

	// 编造检测
	EnableFabricationCheck bool `json:"enable_fabrication_check"` // 是否启用编造检测

	// 进度检测
	MaxIterationsWithoutProgress int `json:"max_iterations_without_progress"` // 无进度最大迭代次数
	ProgressCheckWindow          int `json:"progress_check_window"`           // 进度检查窗口

	// 文件验证
	MaxFileChecksPerReview int `json:"max_file_checks_per_review"` // 每次审查最大文件检查数
}

// DefaultReviewConfig 返回默认审查配置
func DefaultReviewConfig() *ReviewConfig {
	return &ReviewConfig{
		EnableDirectionCheck:         true,
		MaxRepeatedActions:           3,
		LoopWindowSize:               5,
		MaxConsecutiveFailures:       3,
		FailureResetInterval:         60,
		EnableFabricationCheck:       true,
		MaxIterationsWithoutProgress: 5,
		ProgressCheckWindow:          3,
		MaxFileChecksPerReview:       10,
	}
}

// Reviewer 程序逻辑审查器
// 负责审查Agent的输出和行为，检测问题并提供修正建议
type Reviewer struct {
	config *ReviewConfig

	// 状态跟踪
	mu sync.RWMutex

	// 历史记录（用于循环检测）
	actionHistory []ActionRecord
	failureCount  int
	lastFailure   time.Time

	// 进度跟踪
	iterationCount   int
	lastProgressStep string
}

// ActionRecord 操作记录
type ActionRecord struct {
	ToolName  string    `json:"tool_name"`
	Arguments string    `json:"arguments"`
	Result    string    `json:"result"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
}

// NewReviewer 创建新的审查器
func NewReviewer(config *ReviewConfig) *Reviewer {
	if config == nil {
		config = DefaultReviewConfig()
	}

	return &Reviewer{
		config:        config,
		actionHistory: make([]ActionRecord, 0),
	}
}

// ReviewOutput 审查模型输出
// output: 模型的响应内容
// toolCalls: 工具调用列表
// state: 当前任务状态
func (r *Reviewer) ReviewOutput(output string, toolCalls []model.ToolCall, taskState *state.TaskState) *ReviewResult {
	result := &ReviewResult{
		Status:    ReviewStatusPass,
		Issues:    make([]ReviewIssue, 0),
		Timestamp: time.Now(),
	}

	// 1. 方向偏离检测
	if r.config.EnableDirectionCheck && taskState != nil {
		if issue := r.checkDirection(output, taskState); issue != nil {
			result.Issues = append(result.Issues, *issue)
		}
	}

	// 2. 工具调用审查
	for _, tc := range toolCalls {
		// 2.1 无效工具调用检测
		if issue := r.checkInvalidToolCall(tc); issue != nil {
			result.Issues = append(result.Issues, *issue)
		}

		// 2.2 无限循环检测
		if issue := r.checkInfiniteLoop(tc); issue != nil {
			result.Issues = append(result.Issues, *issue)
		}

		// 记录操作历史
		r.recordAction(tc, "")
	}

	// 3. 编造内容检测
	if r.config.EnableFabricationCheck {
		if issues := r.checkFabrication(output, toolCalls); len(issues) > 0 {
			result.Issues = append(result.Issues, issues...)
		}
	}

	// 4. 进度验证
	if taskState != nil {
		if issue := r.checkProgress(taskState); issue != nil {
			result.Issues = append(result.Issues, *issue)
		}
	}

	// 确定最终状态
	result.Status = r.determineStatus(result.Issues)
	result.Summary = r.generateSummary(result)

	return result
}

// ReviewToolResult 审查工具执行结果
// 在工具执行后调用，用于检测失败模式
func (r *Reviewer) ReviewToolResult(toolName string, arguments string, result string, success bool) *ReviewResult {
	resultObj := &ReviewResult{
		Status:    ReviewStatusPass,
		Issues:    make([]ReviewIssue, 0),
		Timestamp: time.Now(),
	}

	// 更新操作记录
	r.updateActionResult(toolName, arguments, result, success)

	// 检测重复失败
	if !success {
		r.mu.Lock()
		r.failureCount++
		r.lastFailure = time.Now()
		failCount := r.failureCount
		r.mu.Unlock()

		if failCount >= r.config.MaxConsecutiveFailures {
			issue := ReviewIssue{
				Type:        IssueTypeRepeatedFailure,
				Severity:    "high",
				Description: fmt.Sprintf("连续失败次数达到%d次，可能存在系统性问题", failCount),
				Evidence:    fmt.Sprintf("工具 %s 执行失败: %s", toolName, result),
				Suggestion:  "请检查工具参数是否正确，或考虑使用其他方法完成任务",
				ToolName:    toolName,
				Timestamp:   time.Now(),
			}
			resultObj.Issues = append(resultObj.Issues, issue)
		}
	} else {
		// 成功时重置失败计数
		r.mu.Lock()
		if time.Since(r.lastFailure).Seconds() > float64(r.config.FailureResetInterval) {
			r.failureCount = 0
		}
		r.mu.Unlock()
	}

	resultObj.Status = r.determineStatus(resultObj.Issues)
	resultObj.Summary = r.generateSummary(resultObj)

	return resultObj
}

// checkDirection 检查方向偏离
// 验证输出是否与任务目标一致
func (r *Reviewer) checkDirection(output string, taskState *state.TaskState) *ReviewIssue {
	if taskState == nil || taskState.Task.Goal == "" {
		return nil
	}

	goal := strings.ToLower(taskState.Task.Goal)
	outputLower := strings.ToLower(output)

	// 检查是否包含与目标相关的关键词
	// 提取目标中的关键词（简化实现）
	keywords := extractKeywords(goal)
	matchedCount := 0
	for _, kw := range keywords {
		if strings.Contains(outputLower, kw) {
			matchedCount++
		}
	}

	// 如果关键词匹配率过低，可能偏离方向
	if len(keywords) > 0 && matchedCount < len(keywords)/3 {
		return &ReviewIssue{
			Type:        IssueTypeDirection,
			Severity:    "medium",
			Description: "输出内容可能与任务目标偏离",
			Evidence:    fmt.Sprintf("目标: %s\n输出: %s", taskState.Task.Goal, truncateString(output, 200)),
			Suggestion:  "请确保输出内容与任务目标相关，专注于完成当前任务",
			Timestamp:   time.Now(),
		}
	}

	return nil
}

// checkInvalidToolCall 检查无效工具调用
func (r *Reviewer) checkInvalidToolCall(tc model.ToolCall) *ReviewIssue {
	// 检查工具名称是否为空
	if tc.Function.Name == "" {
		return &ReviewIssue{
			Type:        IssueTypeInvalidToolCall,
			Severity:    "high",
			Description: "工具调用缺少工具名称",
			Evidence:    fmt.Sprintf("ToolCall ID: %s", tc.ID),
			Suggestion:  "请确保工具调用包含有效的工具名称",
			ToolName:    tc.Function.Name,
			Timestamp:   time.Now(),
		}
	}

	// 检查参数是否为有效JSON
	if tc.Function.Arguments != "" {
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return &ReviewIssue{
				Type:        IssueTypeInvalidToolCall,
				Severity:    "high",
				Description: "工具调用参数不是有效的JSON格式",
				Evidence:    fmt.Sprintf("工具: %s, 参数: %s, 错误: %v", tc.Function.Name, tc.Function.Arguments, err),
				Suggestion:  "请确保工具参数是有效的JSON格式",
				ToolName:    tc.Function.Name,
				Timestamp:   time.Now(),
			}
		}
	}

	return nil
}

// checkInfiniteLoop 检查无限循环
// 检测重复相同操作
func (r *Reviewer) checkInfiniteLoop(tc model.ToolCall) *ReviewIssue {
	r.mu.RLock()
	history := r.actionHistory
	r.mu.RUnlock()

	if len(history) < r.config.LoopWindowSize {
		return nil
	}

	// 检查最近的操作是否重复
	recentActions := history
	if len(recentActions) > r.config.LoopWindowSize {
		recentActions = recentActions[len(recentActions)-r.config.LoopWindowSize:]
	}

	// 统计相同操作次数
	sameCount := 0
	for _, action := range recentActions {
		if action.ToolName == tc.Function.Name && action.Arguments == tc.Function.Arguments {
			sameCount++
		}
	}

	if sameCount >= r.config.MaxRepeatedActions {
		return &ReviewIssue{
			Type:        IssueTypeInfiniteLoop,
			Severity:    "critical",
			Description: fmt.Sprintf("检测到可能的无限循环：相同操作重复%d次", sameCount),
			Evidence:    fmt.Sprintf("工具: %s, 参数: %s", tc.Function.Name, tc.Function.Arguments),
			Suggestion:  "请尝试不同的方法或参数，避免重复相同操作",
			ToolName:    tc.Function.Name,
			Timestamp:   time.Now(),
		}
	}

	return nil
}

// checkFabrication 检查编造内容
// 验证声称存在的文件或函数是否真实存在
func (r *Reviewer) checkFabrication(output string, toolCalls []model.ToolCall) []ReviewIssue {
	issues := make([]ReviewIssue, 0)

	// 从输出中提取声称存在的文件路径
	filePaths := extractFilePaths(output)

	// 限制检查数量
	checkCount := 0
	for _, path := range filePaths {
		if checkCount >= r.config.MaxFileChecksPerReview {
			break
		}

		// 检查文件是否存在
		if _, err := os.Stat(path); os.IsNotExist(err) {
			issues = append(issues, ReviewIssue{
				Type:        IssueTypeFabrication,
				Severity:    "high",
				Description: fmt.Sprintf("声称存在的文件不存在: %s", path),
				Evidence:    fmt.Sprintf("输出中提到的文件路径: %s", path),
				Suggestion:  "请确保文件路径正确，或先创建文件再引用",
				Timestamp:   time.Now(),
			})
		}
		checkCount++
	}

	// 检查工具调用中声称的文件操作
	for _, tc := range toolCalls {
		if tc.Function.Name == "read_file" || tc.Function.Name == "write_file" {
			args, err := parseToolArgs(tc.Function.Arguments)
			if err != nil {
				continue
			}

			if filePath, ok := args["file_path"].(string); ok {
				// 对于读取操作，检查文件是否存在
				if tc.Function.Name == "read_file" {
					if _, err := os.Stat(filePath); os.IsNotExist(err) {
						issues = append(issues, ReviewIssue{
							Type:        IssueTypeFabrication,
							Severity:    "medium",
							Description: fmt.Sprintf("尝试读取不存在的文件: %s", filePath),
							Evidence:    fmt.Sprintf("工具调用: %s, 文件路径: %s", tc.Function.Name, filePath),
							Suggestion:  "请确认文件路径正确，或先创建文件",
							ToolName:    tc.Function.Name,
							Timestamp:   time.Now(),
						})
					}
				}
			}
		}
	}

	return issues
}

// checkProgress 检查进度
// 验证是否真正推进任务进度
func (r *Reviewer) checkProgress(taskState *state.TaskState) *ReviewIssue {
	r.mu.Lock()
	r.iterationCount++
	currentIteration := r.iterationCount
	r.mu.Unlock()

	// 检查是否有新的完成步骤
	currentStep := ""
	if len(taskState.Progress.CompletedSteps) > 0 {
		currentStep = taskState.Progress.CompletedSteps[len(taskState.Progress.CompletedSteps)-1]
	}

	// 检查进度是否停滞
	if currentStep == r.lastProgressStep {
		if currentIteration >= r.config.MaxIterationsWithoutProgress {
			return &ReviewIssue{
				Type:        IssueTypeNoProgress,
				Severity:    "medium",
				Description: fmt.Sprintf("已迭代%d次但任务进度未推进", currentIteration),
				Evidence:    fmt.Sprintf("当前阶段: %s, 已完成步骤: %v", taskState.Progress.CurrentPhase, taskState.Progress.CompletedSteps),
				Suggestion:  "请检查当前方法是否有效，考虑调整策略或分解任务",
				Timestamp:   time.Now(),
			}
		}
	} else {
		// 有进度，重置计数
		r.mu.Lock()
		r.iterationCount = 0
		r.lastProgressStep = currentStep
		r.mu.Unlock()
	}

	return nil
}

// recordAction 记录操作
func (r *Reviewer) recordAction(tc model.ToolCall, result string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record := ActionRecord{
		ToolName:  tc.Function.Name,
		Arguments: tc.Function.Arguments,
		Result:    result,
		Timestamp: time.Now(),
		Success:   true, // 初始假设成功，后续更新
	}

	r.actionHistory = append(r.actionHistory, record)

	// 保持历史记录在合理范围内
	maxHistory := r.config.LoopWindowSize * 3
	if len(r.actionHistory) > maxHistory {
		r.actionHistory = r.actionHistory[len(r.actionHistory)-maxHistory:]
	}
}

// updateActionResult 更新操作结果
func (r *Reviewer) updateActionResult(toolName string, arguments string, result string, success bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 找到最近的匹配操作并更新
	for i := len(r.actionHistory) - 1; i >= 0; i-- {
		if r.actionHistory[i].ToolName == toolName && r.actionHistory[i].Arguments == arguments {
			r.actionHistory[i].Result = result
			r.actionHistory[i].Success = success
			break
		}
	}
}

// determineStatus 根据问题确定审查状态
func (r *Reviewer) determineStatus(issues []ReviewIssue) ReviewStatus {
	if len(issues) == 0 {
		return ReviewStatusPass
	}

	// 检查是否有高严重度问题
	for _, issue := range issues {
		if issue.Severity == "critical" || issue.Severity == "high" {
			return ReviewStatusBlock
		}
	}

	// 有问题但严重度较低
	return ReviewStatusWarning
}

// generateSummary 生成审查摘要
func (r *Reviewer) generateSummary(result *ReviewResult) string {
	if len(result.Issues) == 0 {
		return "审查通过，未发现问题"
	}

	summary := fmt.Sprintf("发现%d个问题: ", len(result.Issues))
	typeCounts := make(map[IssueType]int)
	for _, issue := range result.Issues {
		typeCounts[issue.Type]++
	}

	parts := make([]string, 0)
	for t, c := range typeCounts {
		parts = append(parts, fmt.Sprintf("%s(%d)", t, c))
	}
	summary += strings.Join(parts, ", ")

	return summary
}

// GetActionHistory 获取操作历史
func (r *Reviewer) GetActionHistory() []ActionRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	history := make([]ActionRecord, len(r.actionHistory))
	copy(history, r.actionHistory)
	return history
}

// Reset 重置审查器状态
func (r *Reviewer) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.actionHistory = make([]ActionRecord, 0)
	r.failureCount = 0
	r.iterationCount = 0
	r.lastProgressStep = ""
}

// GetFailureCount 获取当前失败计数
func (r *Reviewer) GetFailureCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.failureCount
}

// 辅助函数

// extractKeywords 从文本中提取关键词
func extractKeywords(text string) []string {
	// 简化实现：提取长度大于2的单词
	words := strings.Fields(text)
	keywords := make([]string, 0)

	// 过滤常见停用词
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "shall": true, "can": true,
		"need": true, "dare": true, "ought": true, "used": true, "to": true,
		"of": true, "in": true, "for": true, "on": true, "with": true,
		"at": true, "by": true, "from": true, "as": true, "into": true,
		"through": true, "during": true, "before": true, "after": true,
		"above": true, "below": true, "between": true, "under": true,
		"again": true, "further": true, "then": true, "once": true,
		"and": true, "but": true, "or": true, "nor": true, "so": true,
		"yet": true, "both": true, "either": true, "neither": true,
		"not": true, "only": true, "own": true, "same": true, "than": true,
		"too": true, "very": true, "just": true, "also": true,
		// 中文停用词
		"的": true, "是": true, "在": true, "了": true, "和": true,
		"与": true, "或": true, "但": true, "这": true, "那": true,
		"有": true, "为": true, "对": true, "以": true, "及": true,
	}

	for _, word := range words {
		word = strings.ToLower(strings.TrimSpace(word))
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	// 去重
	seen := make(map[string]bool)
	unique := make([]string, 0)
	for _, kw := range keywords {
		if !seen[kw] {
			seen[kw] = true
			unique = append(unique, kw)
		}
	}

	return unique
}

// extractFilePaths 从文本中提取文件路径
func extractFilePaths(text string) []string {
	paths := make([]string, 0)

	// 匹配常见文件路径模式
	patterns := []string{
		`[a-zA-Z]:\\[\\\w\s\-\.]+`,           // Windows路径
		`/[\w/\-\.]+`,                         // Unix路径
		`[\w\-\.]+\.(go|py|js|ts|java|md|txt)`, // 文件名
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(text, -1)
		paths = append(paths, matches...)
	}

	// 去重
	seen := make(map[string]bool)
	unique := make([]string, 0)
	for _, p := range paths {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}

	return unique
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// parseToolArgs 解析工具参数
func parseToolArgs(argsJSON string) (map[string]interface{}, error) {
	var args map[string]interface{}
	if argsJSON == "" {
		return args, nil
	}
	err := json.Unmarshal([]byte(argsJSON), &args)
	return args, err
}
