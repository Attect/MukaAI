// Package syntax 提供文件语法检查功能
// 支持多种编程语言和配置文件格式的语法验证
package syntax

// SyntaxCheckResult 语法检查结果
type SyntaxCheckResult struct {
	// Language 检查的语言类型
	Language string `json:"language"`

	// HasErrors 是否存在语法错误
	HasErrors bool `json:"has_errors"`

	// Errors 语法错误列表
	Errors []SyntaxError `json:"errors,omitempty"`

	// Warnings 语法警告列表
	Warnings []SyntaxWarning `json:"warnings,omitempty"`

	// CheckMethod 检查方法：native(Go原生) / external(外部工具) / skipped(跳过)
	CheckMethod string `json:"check_method"`

	// Degraded 是否降级（外部工具不可用）
	Degraded bool `json:"degraded,omitempty"`

	// DegradedReason 降级原因
	DegradedReason string `json:"degraded_reason,omitempty"`
}

// SyntaxError 语法错误
type SyntaxError struct {
	// Line 行号（1-based）
	Line int `json:"line,omitempty"`

	// Column 列号（1-based，可选）
	Column int `json:"column,omitempty"`

	// Message 错误描述
	Message string `json:"message"`

	// Severity 严重程度：error / warning
	Severity string `json:"severity"`

	// Suggestion 修正建议（可选）
	Suggestion string `json:"suggestion,omitempty"`
}

// SyntaxWarning 语法警告（复用SyntaxError结构）
type SyntaxWarning = SyntaxError

// newSuccessCheckResult 创建一个无错误的检查结果
func newSuccessCheckResult(language, method string) *SyntaxCheckResult {
	return &SyntaxCheckResult{
		Language:    language,
		HasErrors:   false,
		Errors:      []SyntaxError{},
		Warnings:    []SyntaxWarning{},
		CheckMethod: method,
		Degraded:    false,
	}
}

// newDegradedCheckResult 创建一个降级的检查结果
func newDegradedCheckResult(language, reason string) *SyntaxCheckResult {
	return &SyntaxCheckResult{
		Language:       language,
		HasErrors:      false,
		Errors:         []SyntaxError{},
		Warnings:       []SyntaxWarning{},
		CheckMethod:    "skipped",
		Degraded:       true,
		DegradedReason: reason,
	}
}

// newErrorCheckResult 创建一个包含错误的检查结果
func newErrorCheckResult(language, method string, errors []SyntaxError) *SyntaxCheckResult {
	return &SyntaxCheckResult{
		Language:    language,
		HasErrors:   true,
		Errors:      errors,
		Warnings:    []SyntaxWarning{},
		CheckMethod: method,
		Degraded:    false,
	}
}
