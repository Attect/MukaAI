// Package agent 实现Agent核心循环和业务逻辑
package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// CorrectionStatus 修正状态
type CorrectionStatus string

const (
	// CorrectionStatusNeeded 需要修正
	CorrectionStatusNeeded CorrectionStatus = "needed"
	// CorrectionStatusRetryable 可重试
	CorrectionStatusRetryable CorrectionStatus = "retryable"
	// CorrectionStatusExhausted 重试次数耗尽
	CorrectionStatusExhausted CorrectionStatus = "exhausted"
	// CorrectionStatusSuccess 修正成功
	CorrectionStatusSuccess CorrectionStatus = "success"
)

// FailureType 失败类型
type FailureType string

const (
	// FailureTypeVerify 校验失败
	FailureTypeVerify FailureType = "verify"
	// FailureTypeReview 审查失败
	FailureTypeReview FailureType = "review"
	// FailureTypeTool 工具执行失败
	FailureTypeTool FailureType = "tool"
	// FailureTypeSystem 系统错误
	FailureTypeSystem FailureType = "system"
)

// FailureRecord 失败记录
type FailureRecord struct {
	Type        FailureType `json:"type"`         // 失败类型
	Timestamp   time.Time   `json:"timestamp"`    // 失败时间
	Summary     string      `json:"summary"`      // 失败摘要
	Details     string      `json:"details"`      // 详细信息
	Issues      []string    `json:"issues"`       // 问题列表
	RetryCount  int         `json:"retry_count"`  // 重试次数
	Corrected   bool        `json:"corrected"`    // 是否已修正
}

// CorrectionResult 修正结果
type CorrectionResult struct {
	Status           CorrectionStatus `json:"status"`             // 修正状态
	NeedsCorrection  bool             `json:"needs_correction"`   // 是否需要修正
	Instruction      string           `json:"instruction"`        // 修正指令内容
	RemainingRetries int              `json:"remaining_retries"`  // 剩余重试次数
	FailureSummary   string           `json:"failure_summary"`    // 失败原因摘要
	Suggestions      []string         `json:"suggestions"`        // 修正建议列表
	Priority         string           `json:"priority"`           // 优先级：low, medium, high, critical
	Timestamp        time.Time        `json:"timestamp"`          // 时间戳
}

// SelfCorrectorConfig 自我修正器配置
type SelfCorrectorConfig struct {
	// 重试配置
	MaxRetries          int `json:"max_retries"`           // 最大重试次数
	RetryDelayMs       int `json:"retry_delay_ms"`        // 重试延迟（毫秒）
	ExponentialBackoff bool `json:"exponential_backoff"`   // 是否启用指数退避

	// 失败分析配置
	MaxFailureHistory  int `json:"max_failure_history"`   // 最大失败历史记录数
	FailurePatternWindow int `json:"failure_pattern_window"` // 失败模式检测窗口大小

	// 修正指令配置
	MaxInstructionLength int `json:"max_instruction_length"` // 修正指令最大长度
	IncludeEvidence      bool `json:"include_evidence"`       // 是否包含证据
	IncludeSuggestions   bool `json:"include_suggestions"`    // 是否包含建议
}

// DefaultSelfCorrectorConfig 返回默认自我修正器配置
func DefaultSelfCorrectorConfig() *SelfCorrectorConfig {
	return &SelfCorrectorConfig{
		MaxRetries:           3,
		RetryDelayMs:        1000,
		ExponentialBackoff:  true,
		MaxFailureHistory:   50,
		FailurePatternWindow: 5,
		MaxInstructionLength: 2000,
		IncludeEvidence:     true,
		IncludeSuggestions:  true,
	}
}

// SelfCorrector 自我修正器
// 负责分析失败原因、生成修正指令、管理重试逻辑
type SelfCorrector struct {
	config *SelfCorrectorConfig

	// 状态跟踪
	mu sync.RWMutex

	// 重试计数
	currentRetryCount int

	// 失败历史记录
	failureHistory []FailureRecord

	// 修正历史
	correctionHistory []CorrectionResult
}

// NewSelfCorrector 创建新的自我修正器
func NewSelfCorrector(config *SelfCorrectorConfig) *SelfCorrector {
	if config == nil {
		config = DefaultSelfCorrectorConfig()
	}

	return &SelfCorrector{
		config:            config,
		currentRetryCount: 0,
		failureHistory:    make([]FailureRecord, 0),
		correctionHistory: make([]CorrectionResult, 0),
	}
}

// AnalyzeFailure 分析校验失败结果
// 提取关键问题，生成问题摘要
func (sc *SelfCorrector) AnalyzeFailure(verifyResult *VerifyResult, reviewResult *ReviewResult) *CorrectionResult {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	result := &CorrectionResult{
		Status:           CorrectionStatusNeeded,
		NeedsCorrection:  true,
		RemainingRetries: sc.config.MaxRetries - sc.currentRetryCount,
		Suggestions:      make([]string, 0),
		Timestamp:        time.Now(),
	}

	// 收集所有问题
	issues := make([]string, 0)
	details := make([]string, 0)
	priority := "low"

	// 分析校验失败
	if verifyResult != nil && verifyResult.IsFailed() {
		result.Status = CorrectionStatusNeeded
		result.NeedsCorrection = true

		// 提取校验问题
		for _, issue := range verifyResult.Issues {
			issueStr := fmt.Sprintf("[%s] %s", issue.Severity, issue.Description)
			issues = append(issues, issueStr)

			// 更新优先级
			if issue.Severity == "critical" {
				priority = "critical"
			} else if issue.Severity == "high" && priority != "critical" {
				priority = "high"
			} else if issue.Severity == "medium" && priority != "critical" && priority != "high" {
				priority = "medium"
			}

			// 添加建议
			if issue.Suggestion != "" && sc.config.IncludeSuggestions {
				result.Suggestions = append(result.Suggestions, issue.Suggestion)
			}

			// 添加证据
			if issue.Evidence != "" && sc.config.IncludeEvidence {
				details = append(details, fmt.Sprintf("证据: %s", issue.Evidence))
			}
		}
	}

	// 分析审查失败
	if reviewResult != nil && reviewResult.IsBlocked() {
		result.Status = CorrectionStatusNeeded
		result.NeedsCorrection = true

		// 提取审查问题
		for _, issue := range reviewResult.Issues {
			issueStr := fmt.Sprintf("[%s] %s", issue.Severity, issue.Description)
			issues = append(issues, issueStr)

			// 更新优先级
			if issue.Severity == "critical" {
				priority = "critical"
			} else if issue.Severity == "high" && priority != "critical" {
				priority = "high"
			} else if issue.Severity == "medium" && priority != "critical" && priority != "high" {
				priority = "medium"
			}

			// 添加建议
			if issue.Suggestion != "" && sc.config.IncludeSuggestions {
				result.Suggestions = append(result.Suggestions, issue.Suggestion)
			}

			// 添加证据
			if issue.Evidence != "" && sc.config.IncludeEvidence {
				details = append(details, fmt.Sprintf("证据: %s", issue.Evidence))
			}
		}
	}

	// 生成失败摘要
	result.FailureSummary = sc.generateFailureSummary(issues)
	result.Priority = priority

	// 记录失败
	failureRecord := FailureRecord{
		Type:       sc.determineFailureType(verifyResult, reviewResult),
		Timestamp:  time.Now(),
		Summary:    result.FailureSummary,
		Details:    strings.Join(details, "\n"),
		Issues:     issues,
		RetryCount: sc.currentRetryCount,
		Corrected:  false,
	}
	sc.failureHistory = append(sc.failureHistory, failureRecord)

	// 保持历史记录在合理范围内
	if len(sc.failureHistory) > sc.config.MaxFailureHistory {
		sc.failureHistory = sc.failureHistory[len(sc.failureHistory)-sc.config.MaxFailureHistory:]
	}

	return result
}

// GenerateCorrectionInstruction 根据失败原因生成修正指令
// 指令应该清晰指导Agent如何修复问题
func (sc *SelfCorrector) GenerateCorrectionInstruction(correctionResult *CorrectionResult) string {
	if correctionResult == nil || !correctionResult.NeedsCorrection {
		return ""
	}

	var instruction strings.Builder

	// 构建修正指令
	instruction.WriteString("## 任务执行失败，需要进行修正\n\n")
	instruction.WriteString(fmt.Sprintf("**失败原因**: %s\n\n", correctionResult.FailureSummary))
	instruction.WriteString(fmt.Sprintf("**优先级**: %s\n\n", correctionResult.Priority))
	instruction.WriteString(fmt.Sprintf("**剩余重试次数**: %d\n\n", correctionResult.RemainingRetries))

	// 添加修正建议
	if len(correctionResult.Suggestions) > 0 {
		instruction.WriteString("### 修正建议\n\n")
		for i, suggestion := range correctionResult.Suggestions {
			instruction.WriteString(fmt.Sprintf("%d. %s\n", i+1, suggestion))
		}
		instruction.WriteString("\n")
	}

	// 根据失败类型添加具体指导
	instruction.WriteString("### 下一步操作\n\n")
	instruction.WriteString(sc.generateActionGuidance(correctionResult))

	// 限制指令长度
	instructionStr := instruction.String()
	if len(instructionStr) > sc.config.MaxInstructionLength {
		instructionStr = instructionStr[:sc.config.MaxInstructionLength] + "...\n（指令已截断）"
	}

	return instructionStr
}

// ShouldRetry 检查是否还有重试机会
// 更新重试计数
func (sc *SelfCorrector) ShouldRetry() bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// 检查是否还有重试机会
	if sc.currentRetryCount >= sc.config.MaxRetries {
		return false
	}

	// 增加重试计数
	sc.currentRetryCount++

	return true
}

// RecordFailure 记录失败历史
// 用于分析失败模式
func (sc *SelfCorrector) RecordFailure(failureType FailureType, summary string, details string, issues []string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	record := FailureRecord{
		Type:       failureType,
		Timestamp:  time.Now(),
		Summary:    summary,
		Details:    details,
		Issues:     issues,
		RetryCount: sc.currentRetryCount,
		Corrected:  false,
	}

	sc.failureHistory = append(sc.failureHistory, record)

	// 保持历史记录在合理范围内
	if len(sc.failureHistory) > sc.config.MaxFailureHistory {
		sc.failureHistory = sc.failureHistory[len(sc.failureHistory)-sc.config.MaxFailureHistory:]
	}
}

// RecordCorrection 记录修正结果
func (sc *SelfCorrector) RecordCorrection(result *CorrectionResult) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if result == nil {
		return
	}

	// 标记最近的失败记录为已修正
	if len(sc.failureHistory) > 0 {
		sc.failureHistory[len(sc.failureHistory)-1].Corrected = true
	}

	// 记录修正结果
	sc.correctionHistory = append(sc.correctionHistory, *result)

	// 如果修正成功，重置重试计数
	if result.Status == CorrectionStatusSuccess {
		sc.currentRetryCount = 0
	}
}

// Reset 重置修正器状态
func (sc *SelfCorrector) Reset() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.currentRetryCount = 0
	sc.failureHistory = make([]FailureRecord, 0)
	sc.correctionHistory = make([]CorrectionResult, 0)
}

// GetRetryCount 获取当前重试次数
func (sc *SelfCorrector) GetRetryCount() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.currentRetryCount
}

// GetRemainingRetries 获取剩余重试次数
func (sc *SelfCorrector) GetRemainingRetries() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.config.MaxRetries - sc.currentRetryCount
}

// GetFailureHistory 获取失败历史记录
func (sc *SelfCorrector) GetFailureHistory() []FailureRecord {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	history := make([]FailureRecord, len(sc.failureHistory))
	copy(history, sc.failureHistory)
	return history
}

// GetCorrectionHistory 获取修正历史记录
func (sc *SelfCorrector) GetCorrectionHistory() []CorrectionResult {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	history := make([]CorrectionResult, len(sc.correctionHistory))
	copy(history, sc.correctionHistory)
	return history
}

// GetConfig 获取修正器配置
func (sc *SelfCorrector) GetConfig() *SelfCorrectorConfig {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.config
}

// UpdateConfig 更新修正器配置
func (sc *SelfCorrector) UpdateConfig(config *SelfCorrectorConfig) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.config = config
	return nil
}

// DetectFailurePattern 检测失败模式
// 分析最近的失败记录，识别重复出现的失败模式
func (sc *SelfCorrector) DetectFailurePattern() *FailurePattern {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if len(sc.failureHistory) < sc.config.FailurePatternWindow {
		return nil
	}

	// 获取最近的失败记录
	recentFailures := sc.failureHistory
	if len(recentFailures) > sc.config.FailurePatternWindow {
		recentFailures = recentFailures[len(recentFailures)-sc.config.FailurePatternWindow:]
	}

	// 统计失败类型
	typeCount := make(map[FailureType]int)
	issueCount := make(map[string]int)

	for _, failure := range recentFailures {
		typeCount[failure.Type]++
		for _, issue := range failure.Issues {
			issueCount[issue]++
		}
	}

	// 检测重复模式
	pattern := &FailurePattern{
		DetectedAt:      time.Now(),
		FailureTypes:    typeCount,
		RecurringIssues: make([]string, 0),
		Severity:        "low",
	}

	// 找出重复出现的问题（出现2次以上）
	for issue, count := range issueCount {
		if count >= 2 {
			pattern.RecurringIssues = append(pattern.RecurringIssues, issue)
		}
	}

	// 确定严重程度
	if len(pattern.RecurringIssues) > 0 {
		pattern.Severity = "high"
		pattern.Description = fmt.Sprintf("检测到重复失败模式：最近%d次失败中，有%d个问题重复出现",
			len(recentFailures), len(pattern.RecurringIssues))
	} else {
		pattern.Description = "未检测到明显的失败模式"
	}

	return pattern
}

// FailurePattern 失败模式
type FailurePattern struct {
	DetectedAt      time.Time               `json:"detected_at"`       // 检测时间
	FailureTypes    map[FailureType]int     `json:"failure_types"`     // 失败类型统计
	RecurringIssues []string                `json:"recurring_issues"`  // 重复出现的问题
	Severity        string                  `json:"severity"`          // 严重程度
	Description     string                  `json:"description"`       // 模式描述
}

// 辅助方法

// generateFailureSummary 生成失败摘要
func (sc *SelfCorrector) generateFailureSummary(issues []string) string {
	if len(issues) == 0 {
		return "未知失败原因"
	}

	if len(issues) == 1 {
		return issues[0]
	}

	// 多个问题时，生成摘要
	summary := fmt.Sprintf("发现%d个问题: ", len(issues))
	if len(issues) <= 3 {
		summary += strings.Join(issues, "; ")
	} else {
		summary += strings.Join(issues[:3], "; ") + fmt.Sprintf(" 等%d个问题", len(issues))
	}

	return summary
}

// determineFailureType 确定失败类型
func (sc *SelfCorrector) determineFailureType(verifyResult *VerifyResult, reviewResult *ReviewResult) FailureType {
	// 优先判断审查失败
	if reviewResult != nil && reviewResult.IsBlocked() {
		return FailureTypeReview
	}

	// 其次判断校验失败
	if verifyResult != nil && verifyResult.IsFailed() {
		return FailureTypeVerify
	}

	// 默认为系统错误
	return FailureTypeSystem
}

// generateActionGuidance 生成行动指导
func (sc *SelfCorrector) generateActionGuidance(correctionResult *CorrectionResult) string {
	var guidance strings.Builder

	// 根据优先级给出不同的指导
	switch correctionResult.Priority {
	case "critical":
		guidance.WriteString("⚠️ **严重问题**: 请立即停止当前操作，仔细分析失败原因。\n")
		guidance.WriteString("建议：\n")
		guidance.WriteString("1. 回顾任务目标和当前状态\n")
		guidance.WriteString("2. 检查是否存在根本性的理解错误\n")
		guidance.WriteString("3. 考虑是否需要调整整体策略\n")
	case "high":
		guidance.WriteString("🔴 **高优先级问题**: 需要立即修正。\n")
		guidance.WriteString("建议：\n")
		guidance.WriteString("1. 根据上述建议修正问题\n")
		guidance.WriteString("2. 验证修正后的结果\n")
		guidance.WriteString("3. 如果问题持续，考虑替代方案\n")
	case "medium":
		guidance.WriteString("🟡 **中等优先级问题**: 建议修正后继续。\n")
		guidance.WriteString("建议：\n")
		guidance.WriteString("1. 根据建议进行修正\n")
		guidance.WriteString("2. 继续推进任务\n")
	default:
		guidance.WriteString("🟢 **低优先级问题**: 可以继续执行，但建议修正。\n")
		guidance.WriteString("建议：根据建议进行优化。\n")
	}

	// 添加重试相关信息
	if correctionResult.RemainingRetries > 0 {
		guidance.WriteString(fmt.Sprintf("\n还有 %d 次重试机会，请根据建议修正后继续执行。\n", correctionResult.RemainingRetries))
	} else {
		guidance.WriteString("\n⚠️ **重试次数已耗尽**，请考虑：\n")
		guidance.WriteString("1. 是否需要人工介入\n")
		guidance.WriteString("2. 是否需要调整任务目标\n")
		guidance.WriteString("3. 是否需要采用完全不同的方法\n")
	}

	return guidance.String()
}

// GetRetryDelay 获取重试延迟
// 如果启用了指数退避，延迟会随重试次数增加
func (sc *SelfCorrector) GetRetryDelay() time.Duration {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	baseDelay := time.Duration(sc.config.RetryDelayMs) * time.Millisecond

	if sc.config.ExponentialBackoff {
		// 指数退避：每次重试延迟翻倍
		multiplier := 1 << sc.currentRetryCount
		delay := baseDelay * time.Duration(multiplier)
		// 最大延迟30秒
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}
		return delay
	}

	return baseDelay
}

// MarkSuccess 标记修正成功
// 用于在修正成功后重置重试计数
func (sc *SelfCorrector) MarkSuccess() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.currentRetryCount = 0

	// 标记最近的失败记录为已修正
	if len(sc.failureHistory) > 0 {
		sc.failureHistory[len(sc.failureHistory)-1].Corrected = true
	}
}

// GetStatistics 获取统计信息
func (sc *SelfCorrector) GetStatistics() *CorrectionStatistics {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	stats := &CorrectionStatistics{
		CurrentRetryCount:  sc.currentRetryCount,
		MaxRetries:         sc.config.MaxRetries,
		RemainingRetries:   sc.config.MaxRetries - sc.currentRetryCount,
		TotalFailures:      len(sc.failureHistory),
		TotalCorrections:   len(sc.correctionHistory),
	}

	// 统计已修正的失败数
	for _, failure := range sc.failureHistory {
		if failure.Corrected {
			stats.CorrectedFailures++
		}
	}

	// 计算修正成功率
	if stats.TotalFailures > 0 {
		stats.CorrectionSuccessRate = float64(stats.CorrectedFailures) / float64(stats.TotalFailures)
	}

	return stats
}

// CorrectionStatistics 修正统计信息
type CorrectionStatistics struct {
	CurrentRetryCount    int     `json:"current_retry_count"`    // 当前重试次数
	MaxRetries           int     `json:"max_retries"`            // 最大重试次数
	RemainingRetries     int     `json:"remaining_retries"`      // 剩余重试次数
	TotalFailures        int     `json:"total_failures"`         // 总失败次数
	CorrectedFailures    int     `json:"corrected_failures"`     // 已修正的失败次数
	TotalCorrections     int     `json:"total_corrections"`      // 总修正次数
	CorrectionSuccessRate float64 `json:"correction_success_rate"` // 修正成功率
}
