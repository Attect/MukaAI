// Package agent 实现Agent核心循环和业务逻辑
package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"agentplus/internal/state"
)

// VerifyStatus 校验结果状态
type VerifyStatus string

const (
	// VerifyStatusPass 校验通过
	VerifyStatusPass VerifyStatus = "pass"
	// VerifyStatusWarning 警告，存在问题但不影响完成
	VerifyStatusWarning VerifyStatus = "warning"
	// VerifyStatusFail 校验失败，需要修正
	VerifyStatusFail VerifyStatus = "fail"
)

// VerifyIssueType 校验问题类型
type VerifyIssueType string

const (
	// VerifyIssueTypeFileNotFound 文件不存在
	VerifyIssueTypeFileNotFound VerifyIssueType = "file_not_found"
	// VerifyIssueTypeFileEmpty 文件内容为空
	VerifyIssueTypeFileEmpty VerifyIssueType = "file_empty"
	// VerifyIssueTypeContentMissing 内容缺失关键部分
	VerifyIssueTypeContentMissing VerifyIssueType = "content_missing"
	// VerifyIssueTypeKeywordNotFound 关键词未找到
	VerifyIssueTypeKeywordNotFound VerifyIssueType = "keyword_not_found"
	// VerifyIssueTypeCustomRuleFailed 自定义规则校验失败
	VerifyIssueTypeCustomRuleFailed VerifyIssueType = "custom_rule_failed"
	// VerifyIssueTypeInvalidPath 无效路径
	VerifyIssueTypeInvalidPath VerifyIssueType = "invalid_path"
	// VerifyIssueTypePermissionDenied 权限拒绝
	VerifyIssueTypePermissionDenied VerifyIssueType = "permission_denied"
	// VerifyIssueTypeJSSyntaxError JavaScript语法错误
	VerifyIssueTypeJSSyntaxError VerifyIssueType = "js_syntax_error"
	// VerifyIssueTypeHTMLStructureError HTML结构错误
	VerifyIssueTypeHTMLStructureError VerifyIssueType = "html_structure_error"
)

// VerifyIssue 校验发现的问题
type VerifyIssue struct {
	Type        VerifyIssueType `json:"type"`        // 问题类型
	Severity    string          `json:"severity"`    // 严重程度：low, medium, high, critical
	Description string          `json:"description"` // 问题描述
	Evidence    string          `json:"evidence"`    // 证据/示例
	Suggestion  string          `json:"suggestion"`  // 修正建议
	FilePath    string          `json:"file_path"`   // 相关文件路径（如果有）
	RuleName    string          `json:"rule_name"`   // 相关规则名称（如果是自定义规则）
	Timestamp   time.Time       `json:"timestamp"`   // 发现时间
}

// VerifyResult 校验结果
type VerifyResult struct {
	Status    VerifyStatus  `json:"status"`    // 校验状态
	Issues    []VerifyIssue `json:"issues"`    // 发现的问题列表
	Timestamp time.Time     `json:"timestamp"` // 校验时间
	Summary   string        `json:"summary"`   // 校验摘要
	Passed    int           `json:"passed"`    // 通过的检查项数量
	Failed    int           `json:"failed"`    // 失败的检查项数量
}

// IsFailed 检查是否失败
func (r *VerifyResult) IsFailed() bool {
	return r.Status == VerifyStatusFail
}

// HasWarnings 检查是否有警告
func (r *VerifyResult) HasWarnings() bool {
	return r.Status == VerifyStatusWarning || (len(r.Issues) > 0 && r.Status != VerifyStatusFail)
}

// GetCriticalIssues 获取严重级别的问题
func (r *VerifyResult) GetCriticalIssues() []VerifyIssue {
	var critical []VerifyIssue
	for _, issue := range r.Issues {
		if issue.Severity == "high" || issue.Severity == "critical" {
			critical = append(critical, issue)
		}
	}
	return critical
}

// VerifyConfig 校验器配置
type VerifyConfig struct {
	// 文件检查配置
	CheckFileExists    bool `json:"check_file_exists"`      // 是否检查文件存在
	CheckFileNonEmpty  bool `json:"check_file_non_empty"`   // 是否检查文件非空
	MaxFileSizeToCheck int  `json:"max_file_size_to_check"` // 最大检查文件大小（字节），0表示无限制

	// 内容检查配置
	CheckKeywords    bool     `json:"check_keywords"`     // 是否检查关键词
	RequiredKeywords []string `json:"required_keywords"`  // 必需的关键词列表
	KeywordMatchMode string   `json:"keyword_match_mode"` // 关键词匹配模式：any（任一匹配）, all（全部匹配）

	// 语法检查配置
	CheckJSSyntax      bool `json:"check_js_syntax"`      // 是否检查JavaScript语法
	CheckHTMLStructure bool `json:"check_html_structure"` // 是否检查HTML结构

	// 自定义规则配置
	EnableCustomRules bool `json:"enable_custom_rules"` // 是否启用自定义规则

	// 校验行为配置
	StopOnFirstFailure bool `json:"stop_on_first_failure"` // 遇到第一个失败是否停止
	MaxIssuesToReport  int  `json:"max_issues_to_report"`  // 最大报告问题数量
}

// DefaultVerifierConfig 返回默认校验器配置
func DefaultVerifierConfig() *VerifyConfig {
	return &VerifyConfig{
		CheckFileExists:    true,
		CheckFileNonEmpty:  true,
		MaxFileSizeToCheck: 10 * 1024 * 1024, // 10MB
		CheckKeywords:      true,
		RequiredKeywords:   []string{},
		KeywordMatchMode:   "any",
		CheckJSSyntax:      true, // 默认启用JavaScript语法检查
		CheckHTMLStructure: true, // 默认启用HTML结构检查
		EnableCustomRules:  true,
		StopOnFirstFailure: false,
		MaxIssuesToReport:  50,
	}
}

// VerifyRule 自定义校验规则接口
type VerifyRule interface {
	// Name 返回规则名称
	Name() string
	// Description 返回规则描述
	Description() string
	// Execute 执行校验规则
	// 返回nil表示通过，返回error表示失败
	Execute(ctx *VerifyContext) error
}

// VerifyContext 校验上下文
type VerifyContext struct {
	TaskState *state.TaskState       // 任务状态
	Files     []string               // 待校验的文件列表
	Content   string                 // 待校验的内容
	ExtraData map[string]interface{} // 额外数据
}

// Verifier 成果校验器
// 负责校验Agent的工作成果，确保任务完成质量
type Verifier struct {
	config *VerifyConfig

	// 自定义规则
	customRules []VerifyRule

	// 状态跟踪
	mu sync.RWMutex

	// 校验历史
	verifyHistory []VerifyResult
}

// NewVerifier 创建新的校验器
func NewVerifier(config *VerifyConfig) *Verifier {
	if config == nil {
		config = DefaultVerifierConfig()
	}

	return &Verifier{
		config:        config,
		customRules:   make([]VerifyRule, 0),
		verifyHistory: make([]VerifyResult, 0),
	}
}

// AddRule 添加自定义校验规则
func (v *Verifier) AddRule(rule VerifyRule) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.customRules = append(v.customRules, rule)
}

// RemoveRule 移除自定义校验规则
func (v *Verifier) RemoveRule(name string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	for i, rule := range v.customRules {
		if rule.Name() == name {
			v.customRules = append(v.customRules[:i], v.customRules[i+1:]...)
			return true
		}
	}
	return false
}

// ClearRules 清除所有自定义校验规则
func (v *Verifier) ClearRules() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.customRules = make([]VerifyRule, 0)
}

// Verify 执行成果校验
// files: 需要校验的文件列表
// taskState: 当前任务状态
func (v *Verifier) Verify(files []string, taskState *state.TaskState) *VerifyResult {
	result := &VerifyResult{
		Status:    VerifyStatusPass,
		Issues:    make([]VerifyIssue, 0),
		Timestamp: time.Now(),
		Passed:    0,
		Failed:    0,
	}

	// 1. 检查文件是否存在
	if v.config.CheckFileExists {
		for _, file := range files {
			if v.config.StopOnFirstFailure && len(result.Issues) > 0 {
				break
			}
			if issue := v.checkFileExists(file); issue != nil {
				result.Issues = append(result.Issues, *issue)
				result.Failed++
			} else {
				result.Passed++
			}
		}
	}

	// 2. 检查文件内容是否非空（只检查存在的文件）
	if v.config.CheckFileNonEmpty && !(v.config.StopOnFirstFailure && len(result.Issues) > 0) {
		for _, file := range files {
			if v.config.StopOnFirstFailure && len(result.Issues) > 0 {
				break
			}
			// 只检查存在的文件
			if _, err := os.Stat(file); os.IsNotExist(err) {
				continue
			}
			if issue := v.checkFileNonEmpty(file); issue != nil {
				result.Issues = append(result.Issues, *issue)
				result.Failed++
			} else {
				result.Passed++
			}
		}
	}

	// 3. 检查关键词匹配（只检查存在的文件）
	if v.config.CheckKeywords && len(v.config.RequiredKeywords) > 0 && !(v.config.StopOnFirstFailure && len(result.Issues) > 0) {
		for _, file := range files {
			if v.config.StopOnFirstFailure && len(result.Issues) > 0 {
				break
			}
			// 只检查存在的文件
			if _, err := os.Stat(file); os.IsNotExist(err) {
				continue
			}
			issues := v.checkKeywords(file)
			if len(issues) > 0 {
				result.Issues = append(result.Issues, issues...)
				result.Failed += len(issues)
			} else {
				result.Passed++
			}
		}
	}

	// 4. 检查JavaScript语法（只检查.js文件）
	if v.config.CheckJSSyntax && !(v.config.StopOnFirstFailure && len(result.Issues) > 0) {
		for _, file := range files {
			if v.config.StopOnFirstFailure && len(result.Issues) > 0 {
				break
			}
			// 只检查存在的.js文件
			if _, err := os.Stat(file); os.IsNotExist(err) {
				continue
			}
			if !strings.HasSuffix(strings.ToLower(file), ".js") {
				continue
			}
			// 读取文件内容并检查语法
			if content, err := os.ReadFile(file); err == nil {
				issues := v.checkJSSyntax(string(content), file)
				if len(issues) > 0 {
					result.Issues = append(result.Issues, issues...)
					result.Failed += len(issues)
				} else {
					result.Passed++
				}
			}
		}
	}

	// 5. 检查HTML结构（只检查.html文件）
	if v.config.CheckHTMLStructure && !(v.config.StopOnFirstFailure && len(result.Issues) > 0) {
		for _, file := range files {
			if v.config.StopOnFirstFailure && len(result.Issues) > 0 {
				break
			}
			// 只检查存在的.html文件
			if _, err := os.Stat(file); os.IsNotExist(err) {
				continue
			}
			if !strings.HasSuffix(strings.ToLower(file), ".html") {
				continue
			}
			// 读取文件内容并检查结构
			if content, err := os.ReadFile(file); err == nil {
				issues := v.checkHTMLStructure(string(content), file)
				if len(issues) > 0 {
					result.Issues = append(result.Issues, issues...)
					result.Failed += len(issues)
				} else {
					result.Passed++
				}
			}
		}
	}

	// 6. 执行自定义规则
	if v.config.EnableCustomRules && !(v.config.StopOnFirstFailure && len(result.Issues) > 0) {
		v.mu.RLock()
		rules := v.customRules
		v.mu.RUnlock()

		ctx := &VerifyContext{
			TaskState: taskState,
			Files:     files,
			ExtraData: make(map[string]interface{}),
		}

		for _, rule := range rules {
			if v.config.StopOnFirstFailure && len(result.Issues) > 0 {
				break
			}

			if err := rule.Execute(ctx); err != nil {
				issue := VerifyIssue{
					Type:        VerifyIssueTypeCustomRuleFailed,
					Severity:    "high",
					Description: fmt.Sprintf("自定义规则 '%s' 校验失败: %s", rule.Name(), err.Error()),
					Evidence:    err.Error(),
					Suggestion:  fmt.Sprintf("请检查规则 '%s' 的要求", rule.Name()),
					RuleName:    rule.Name(),
					Timestamp:   time.Now(),
				}
				result.Issues = append(result.Issues, issue)
				result.Failed++
			} else {
				result.Passed++
			}
		}
	}

	// 限制报告的问题数量（0表示不限制）
	if v.config.MaxIssuesToReport > 0 && len(result.Issues) > v.config.MaxIssuesToReport {
		result.Issues = result.Issues[:v.config.MaxIssuesToReport]
	}

	// 确定最终状态
	result.Status = v.determineStatus(result.Issues)
	result.Summary = v.generateSummary(result)

	// 记录校验历史
	v.recordResult(*result)

	return result
}

// VerifyTaskCompletion 专门用于complete_task前的校验
// 执行更严格的校验，确保任务真正完成
func (v *Verifier) VerifyTaskCompletion(files []string, taskState *state.TaskState) *VerifyResult {
	result := &VerifyResult{
		Status:    VerifyStatusPass,
		Issues:    make([]VerifyIssue, 0),
		Timestamp: time.Now(),
		Passed:    0,
		Failed:    0,
	}

	// 注意：在complete_task工具调用时，任务状态还是in_progress
	// 所以这里不检查任务状态，只检查文件和内容
	// 任务状态的检查由Agent核心的强制校验负责

	// 1. 检查所有必需文件是否存在且非空
	for _, file := range files {
		// 检查文件存在
		if issue := v.checkFileExists(file); issue != nil {
			issue.Severity = "critical" // 任务完成校验时，文件不存在是严重问题
			result.Issues = append(result.Issues, *issue)
			result.Failed++
			continue
		}
		result.Passed++

		// 检查文件非空
		if issue := v.checkFileNonEmpty(file); issue != nil {
			issue.Severity = "high" // 任务完成校验时，文件为空是高严重度问题
			result.Issues = append(result.Issues, *issue)
			result.Failed++
			continue
		}
		result.Passed++
	}

	// 3. 检查关键词（如果配置了）
	if v.config.CheckKeywords && len(v.config.RequiredKeywords) > 0 {
		for _, file := range files {
			if issues := v.checkKeywords(file); len(issues) > 0 {
				// 提高严重度
				for i := range issues {
					issues[i].Severity = "high"
				}
				result.Issues = append(result.Issues, issues...)
				result.Failed += len(issues)
			} else {
				result.Passed++
			}
		}
	}

	// 4. JavaScript语法检查（如果启用）
	if v.config.CheckJSSyntax {
		for _, file := range files {
			// 只检查.html和.js文件
			ext := strings.ToLower(filepath.Ext(file))
			if ext == ".html" || ext == ".htm" || ext == ".js" {
				content, err := os.ReadFile(file)
				if err != nil {
					continue
				}
				if issues := v.checkJSSyntax(string(content), file); len(issues) > 0 {
					result.Issues = append(result.Issues, issues...)
					result.Failed += len(issues)
				} else {
					result.Passed++
				}
			}
		}
	}

	// 5. HTML结构检查（如果启用）
	if v.config.CheckHTMLStructure {
		for _, file := range files {
			// 只检查.html文件
			ext := strings.ToLower(filepath.Ext(file))
			if ext == ".html" || ext == ".htm" {
				content, err := os.ReadFile(file)
				if err != nil {
					continue
				}
				if issues := v.checkHTMLStructure(string(content), file); len(issues) > 0 {
					result.Issues = append(result.Issues, issues...)
					result.Failed += len(issues)
				} else {
					result.Passed++
				}
			}
		}
	}

	// 4. 执行自定义规则（更严格）
	if v.config.EnableCustomRules {
		v.mu.RLock()
		rules := v.customRules
		v.mu.RUnlock()

		ctx := &VerifyContext{
			TaskState: taskState,
			Files:     files,
			ExtraData: make(map[string]interface{}),
		}

		for _, rule := range rules {
			if err := rule.Execute(ctx); err != nil {
				issue := VerifyIssue{
					Type:        VerifyIssueTypeCustomRuleFailed,
					Severity:    "high", // 任务完成校验时，自定义规则失败是高严重度问题
					Description: fmt.Sprintf("自定义规则 '%s' 校验失败: %s", rule.Name(), err.Error()),
					Evidence:    err.Error(),
					Suggestion:  fmt.Sprintf("请确保满足规则 '%s' 的要求", rule.Name()),
					RuleName:    rule.Name(),
					Timestamp:   time.Now(),
				}
				result.Issues = append(result.Issues, issue)
				result.Failed++
			} else {
				result.Passed++
			}
		}
	}

	// 确定最终状态
	result.Status = v.determineStatus(result.Issues)
	result.Summary = v.generateSummary(result)

	// 记录校验历史
	v.recordResult(*result)

	return result
}

// VerifyFiles 批量校验文件
func (v *Verifier) VerifyFiles(files []string) *VerifyResult {
	return v.Verify(files, nil)
}

// VerifyContent 校验内容（不涉及文件）
func (v *Verifier) VerifyContent(content string, taskState *state.TaskState) *VerifyResult {
	result := &VerifyResult{
		Status:    VerifyStatusPass,
		Issues:    make([]VerifyIssue, 0),
		Timestamp: time.Now(),
		Passed:    0,
		Failed:    0,
	}

	// 检查关键词
	if v.config.CheckKeywords && len(v.config.RequiredKeywords) > 0 {
		if issues := v.checkContentKeywords(content); len(issues) > 0 {
			result.Issues = append(result.Issues, issues...)
			result.Failed += len(issues)
		} else {
			result.Passed++
		}
	}

	// 执行自定义规则
	if v.config.EnableCustomRules {
		v.mu.RLock()
		rules := v.customRules
		v.mu.RUnlock()

		ctx := &VerifyContext{
			TaskState: taskState,
			Content:   content,
			ExtraData: make(map[string]interface{}),
		}

		for _, rule := range rules {
			if err := rule.Execute(ctx); err != nil {
				issue := VerifyIssue{
					Type:        VerifyIssueTypeCustomRuleFailed,
					Severity:    "high",
					Description: fmt.Sprintf("自定义规则 '%s' 校验失败: %s", rule.Name(), err.Error()),
					Evidence:    err.Error(),
					Suggestion:  fmt.Sprintf("请检查规则 '%s' 的要求", rule.Name()),
					RuleName:    rule.Name(),
					Timestamp:   time.Now(),
				}
				result.Issues = append(result.Issues, issue)
				result.Failed++
			} else {
				result.Passed++
			}
		}
	}

	result.Status = v.determineStatus(result.Issues)
	result.Summary = v.generateSummary(result)

	v.recordResult(*result)

	return result
}

// checkFileExists 检查文件是否存在
func (v *Verifier) checkFileExists(filePath string) *VerifyIssue {
	// 规范化路径
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return &VerifyIssue{
			Type:        VerifyIssueTypeInvalidPath,
			Severity:    "high",
			Description: fmt.Sprintf("无效的文件路径: %s", filePath),
			Evidence:    err.Error(),
			Suggestion:  "请检查文件路径是否正确",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		}
	}

	// 检查文件是否存在
	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return &VerifyIssue{
			Type:        VerifyIssueTypeFileNotFound,
			Severity:    "high",
			Description: fmt.Sprintf("文件不存在: %s", filePath),
			Evidence:    fmt.Sprintf("路径: %s", absPath),
			Suggestion:  "请确保文件已创建",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		}
	}

	if err != nil {
		// 检查是否是权限问题
		if os.IsPermission(err) {
			return &VerifyIssue{
				Type:        VerifyIssueTypePermissionDenied,
				Severity:    "high",
				Description: fmt.Sprintf("无权限访问文件: %s", filePath),
				Evidence:    err.Error(),
				Suggestion:  "请检查文件权限",
				FilePath:    filePath,
				Timestamp:   time.Now(),
			}
		}

		return &VerifyIssue{
			Type:        VerifyIssueTypeFileNotFound,
			Severity:    "medium",
			Description: fmt.Sprintf("无法访问文件: %s", filePath),
			Evidence:    err.Error(),
			Suggestion:  "请检查文件状态",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		}
	}

	// 检查是否是目录
	if info.IsDir() {
		return &VerifyIssue{
			Type:        VerifyIssueTypeInvalidPath,
			Severity:    "medium",
			Description: fmt.Sprintf("路径是目录而非文件: %s", filePath),
			Evidence:    fmt.Sprintf("路径: %s", absPath),
			Suggestion:  "请指定具体的文件路径",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		}
	}

	return nil
}

// checkFileNonEmpty 检查文件内容是否非空
func (v *Verifier) checkFileNonEmpty(filePath string) *VerifyIssue {
	// 规范化路径
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil // 路径错误已在checkFileExists中处理
	}

	// 检查文件大小
	info, err := os.Stat(absPath)
	if err != nil {
		return nil // 文件不存在已在checkFileExists中处理
	}

	// 检查文件大小限制
	if v.config.MaxFileSizeToCheck > 0 && info.Size() > int64(v.config.MaxFileSizeToCheck) {
		// 文件过大，跳过内容检查
		return nil
	}

	// 检查文件是否为空
	if info.Size() == 0 {
		return &VerifyIssue{
			Type:        VerifyIssueTypeFileEmpty,
			Severity:    "medium",
			Description: fmt.Sprintf("文件内容为空: %s", filePath),
			Evidence:    fmt.Sprintf("文件大小: %d 字节", info.Size()),
			Suggestion:  "请确保文件包含必要的内容",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		}
	}

	// 读取文件内容检查是否只有空白字符
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}

	// 检查内容是否只有空白
	if len(strings.TrimSpace(string(content))) == 0 {
		return &VerifyIssue{
			Type:        VerifyIssueTypeFileEmpty,
			Severity:    "medium",
			Description: fmt.Sprintf("文件只包含空白字符: %s", filePath),
			Evidence:    fmt.Sprintf("文件大小: %d 字节，但内容全是空白", info.Size()),
			Suggestion:  "请确保文件包含有效内容",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		}
	}

	return nil
}

// checkKeywords 检查文件中是否包含必需的关键词
func (v *Verifier) checkKeywords(filePath string) []VerifyIssue {
	issues := make([]VerifyIssue, 0)

	// 规范化路径
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return issues
	}

	// 读取文件内容
	content, err := os.ReadFile(absPath)
	if err != nil {
		return issues
	}

	contentStr := strings.ToLower(string(content))
	foundKeywords := make([]string, 0)
	missingKeywords := make([]string, 0)

	for _, keyword := range v.config.RequiredKeywords {
		if strings.Contains(contentStr, strings.ToLower(keyword)) {
			foundKeywords = append(foundKeywords, keyword)
		} else {
			missingKeywords = append(missingKeywords, keyword)
		}
	}

	// 根据匹配模式判断是否通过
	if v.config.KeywordMatchMode == "all" && len(missingKeywords) > 0 {
		issue := VerifyIssue{
			Type:        VerifyIssueTypeKeywordNotFound,
			Severity:    "high",
			Description: fmt.Sprintf("文件缺少必需的关键词: %s", filePath),
			Evidence:    fmt.Sprintf("缺少的关键词: %v", missingKeywords),
			Suggestion:  "请确保文件包含所有必需的关键词",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		}
		issues = append(issues, issue)
	} else if v.config.KeywordMatchMode == "any" && len(foundKeywords) == 0 && len(v.config.RequiredKeywords) > 0 {
		issue := VerifyIssue{
			Type:        VerifyIssueTypeKeywordNotFound,
			Severity:    "high",
			Description: fmt.Sprintf("文件未包含任何必需的关键词: %s", filePath),
			Evidence:    fmt.Sprintf("需要的关键词: %v", v.config.RequiredKeywords),
			Suggestion:  "请确保文件至少包含一个必需的关键词",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		}
		issues = append(issues, issue)
	}

	return issues
}

// checkContentKeywords 检查内容中是否包含必需的关键词
func (v *Verifier) checkContentKeywords(content string) []VerifyIssue {
	issues := make([]VerifyIssue, 0)

	contentLower := strings.ToLower(content)
	foundKeywords := make([]string, 0)
	missingKeywords := make([]string, 0)

	for _, keyword := range v.config.RequiredKeywords {
		if strings.Contains(contentLower, strings.ToLower(keyword)) {
			foundKeywords = append(foundKeywords, keyword)
		} else {
			missingKeywords = append(missingKeywords, keyword)
		}
	}

	// 根据匹配模式判断是否通过
	if v.config.KeywordMatchMode == "all" && len(missingKeywords) > 0 {
		issue := VerifyIssue{
			Type:        VerifyIssueTypeKeywordNotFound,
			Severity:    "high",
			Description: "内容缺少必需的关键词",
			Evidence:    fmt.Sprintf("缺少的关键词: %v", missingKeywords),
			Suggestion:  "请确保内容包含所有必需的关键词",
			Timestamp:   time.Now(),
		}
		issues = append(issues, issue)
	} else if v.config.KeywordMatchMode == "any" && len(foundKeywords) == 0 && len(v.config.RequiredKeywords) > 0 {
		issue := VerifyIssue{
			Type:        VerifyIssueTypeKeywordNotFound,
			Severity:    "high",
			Description: "内容未包含任何必需的关键词",
			Evidence:    fmt.Sprintf("需要的关键词: %v", v.config.RequiredKeywords),
			Suggestion:  "请确保内容至少包含一个必需的关键词",
			Timestamp:   time.Now(),
		}
		issues = append(issues, issue)
	}

	return issues
}

// determineStatus 根据问题确定校验状态
func (v *Verifier) determineStatus(issues []VerifyIssue) VerifyStatus {
	if len(issues) == 0 {
		return VerifyStatusPass
	}

	// 检查是否有高严重度问题
	for _, issue := range issues {
		if issue.Severity == "critical" || issue.Severity == "high" {
			return VerifyStatusFail
		}
	}

	// 有问题但严重度较低
	return VerifyStatusWarning
}

// generateSummary 生成校验摘要
func (v *Verifier) generateSummary(result *VerifyResult) string {
	if len(result.Issues) == 0 {
		return fmt.Sprintf("校验通过，共检查%d项", result.Passed)
	}

	summary := fmt.Sprintf("发现%d个问题（通过%d项，失败%d项）: ", len(result.Issues), result.Passed, result.Failed)
	typeCounts := make(map[VerifyIssueType]int)
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

// recordResult 记录校验结果
func (v *Verifier) recordResult(result VerifyResult) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.verifyHistory = append(v.verifyHistory, result)

	// 保持历史记录在合理范围内
	maxHistory := 100
	if len(v.verifyHistory) > maxHistory {
		v.verifyHistory = v.verifyHistory[len(v.verifyHistory)-maxHistory:]
	}
}

// GetVerifyHistory 获取校验历史
func (v *Verifier) GetVerifyHistory() []VerifyResult {
	v.mu.RLock()
	defer v.mu.RUnlock()

	history := make([]VerifyResult, len(v.verifyHistory))
	copy(history, v.verifyHistory)
	return history
}

// GetLastResult 获取最后一次校验结果
func (v *Verifier) GetLastResult() *VerifyResult {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if len(v.verifyHistory) == 0 {
		return nil
	}

	result := v.verifyHistory[len(v.verifyHistory)-1]
	return &result
}

// Reset 重置校验器状态
func (v *Verifier) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.verifyHistory = make([]VerifyResult, 0)
}

// GetConfig 获取校验器配置
func (v *Verifier) GetConfig() *VerifyConfig {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.config
}

// UpdateConfig 更新校验器配置
func (v *Verifier) UpdateConfig(config *VerifyConfig) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	v.config = config
	return nil
}

// SetRequiredKeywords 设置必需的关键词
func (v *Verifier) SetRequiredKeywords(keywords []string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.config.RequiredKeywords = keywords
}

// GetCustomRules 获取所有自定义规则
func (v *Verifier) GetCustomRules() []VerifyRule {
	v.mu.RLock()
	defer v.mu.RUnlock()

	rules := make([]VerifyRule, len(v.customRules))
	copy(rules, v.customRules)
	return rules
}

// checkJSSyntax 检查JavaScript语法
func (v *Verifier) checkJSSyntax(content string, filePath string) []VerifyIssue {
	issues := make([]VerifyIssue, 0)

	// 检查未闭合的字符串
	issues = append(issues, v.checkUnclosedStrings(content, filePath)...)

	// 检查未闭合的括号
	issues = append(issues, v.checkUnclosedBrackets(content, filePath)...)

	// 检查模板字符串中的对象字面量问题
	issues = append(issues, v.checkTemplateLiteralObjects(content, filePath)...)

	// 检查常见语法错误
	issues = append(issues, v.checkCommonJSErrors(content, filePath)...)

	return issues
}

// checkUnclosedStrings 检查未闭合的字符串
func (v *Verifier) checkUnclosedStrings(content string, filePath string) []VerifyIssue {
	issues := make([]VerifyIssue, 0)

	// 简单检查：统计引号数量
	singleQuotes := strings.Count(content, "'")
	doubleQuotes := strings.Count(content, "\"")
	backticks := strings.Count(content, "`")

	// 检查转义引号
	escapedSingle := strings.Count(content, "\\'")
	escapedDouble := strings.Count(content, "\\\"")

	effectiveSingle := singleQuotes - escapedSingle
	effectiveDouble := doubleQuotes - escapedDouble

	if effectiveSingle%2 != 0 {
		issues = append(issues, VerifyIssue{
			Type:        VerifyIssueTypeJSSyntaxError,
			Severity:    "high",
			Description: "检测到未闭合的单引号字符串",
			Evidence:    fmt.Sprintf("文件 '%s' 中单引号数量为奇数(%d)", filePath, effectiveSingle),
			Suggestion:  "请检查并确保所有单引号字符串正确闭合",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		})
	}

	if effectiveDouble%2 != 0 {
		issues = append(issues, VerifyIssue{
			Type:        VerifyIssueTypeJSSyntaxError,
			Severity:    "high",
			Description: "检测到未闭合的双引号字符串",
			Evidence:    fmt.Sprintf("文件 '%s' 中双引号数量为奇数(%d)", filePath, effectiveDouble),
			Suggestion:  "请检查并确保所有双引号字符串正确闭合",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		})
	}

	if backticks%2 != 0 {
		issues = append(issues, VerifyIssue{
			Type:        VerifyIssueTypeJSSyntaxError,
			Severity:    "high",
			Description: "检测到未闭合的模板字符串",
			Evidence:    fmt.Sprintf("文件 '%s' 中反引号数量为奇数(%d)", filePath, backticks),
			Suggestion:  "请检查并确保所有模板字符串正确闭合",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		})
	}

	return issues
}

// checkUnclosedBrackets 检查未闭合的括号
func (v *Verifier) checkUnclosedBrackets(content string, filePath string) []VerifyIssue {
	issues := make([]VerifyIssue, 0)

	// 移除字符串内容后再检查括号
	cleanContent := v.removeStringContent(content)

	parenCount := 0
	bracketCount := 0
	braceCount := 0

	for _, ch := range cleanContent {
		switch ch {
		case '(':
			parenCount++
		case ')':
			parenCount--
		case '[':
			bracketCount++
		case ']':
			bracketCount--
		case '{':
			braceCount++
		case '}':
			braceCount--
		}
	}

	if parenCount != 0 {
		issues = append(issues, VerifyIssue{
			Type:        VerifyIssueTypeJSSyntaxError,
			Severity:    "high",
			Description: "检测到未闭合的圆括号",
			Evidence:    fmt.Sprintf("文件 '%s' 中圆括号不匹配(差%d)", filePath, parenCount),
			Suggestion:  "请检查并确保所有圆括号正确闭合",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		})
	}

	if bracketCount != 0 {
		issues = append(issues, VerifyIssue{
			Type:        VerifyIssueTypeJSSyntaxError,
			Severity:    "high",
			Description: "检测到未闭合的方括号",
			Evidence:    fmt.Sprintf("文件 '%s' 中方括号不匹配(差%d)", filePath, bracketCount),
			Suggestion:  "请检查并确保所有方括号正确闭合",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		})
	}

	if braceCount != 0 {
		issues = append(issues, VerifyIssue{
			Type:        VerifyIssueTypeJSSyntaxError,
			Severity:    "high",
			Description: "检测到未闭合的花括号",
			Evidence:    fmt.Sprintf("文件 '%s' 中花括号不匹配(差%d)", filePath, braceCount),
			Suggestion:  "请检查并确保所有花括号正确闭合",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		})
	}

	return issues
}

// checkTemplateLiteralObjects 检查模板字符串中的对象字面量问题
func (v *Verifier) checkTemplateLiteralObjects(content string, filePath string) []VerifyIssue {
	issues := make([]VerifyIssue, 0)

	// 检查模板字符串中直接包含多行对象字面量的模式
	// 模式：模板字符串内 ${...} 中包含换行的对象
	re := regexp.MustCompile("`[^`]*\\$\\{[^}]*\\n[^}]*\\}[^`]*`")
	matches := re.FindAllString(content, -1)

	for _, match := range matches {
		issues = append(issues, VerifyIssue{
			Type:        VerifyIssueTypeJSSyntaxError,
			Severity:    "medium",
			Description: "模板字符串中包含多行对象字面量可能导致格式问题",
			Evidence:    fmt.Sprintf("文件 '%s' 中发现: %s", filePath, v.truncateString(match, 100)),
			Suggestion:  "建议将对象字面量提取到变量中，然后在模板字符串中引用",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		})
	}

	return issues
}

// checkCommonJSErrors 检查常见JavaScript语法错误
func (v *Verifier) checkCommonJSErrors(content string, filePath string) []VerifyIssue {
	issues := make([]VerifyIssue, 0)

	// 检查缺少分号的函数声明（可选，但有助于代码质量）
	// 检查未定义的变量使用（简化检查）
	// 检查可能的保留字误用

	// 检查file协议兼容性问题
	if strings.Contains(content, "navigator.clipboard") {
		issues = append(issues, VerifyIssue{
			Type:        VerifyIssueTypeJSSyntaxError,
			Severity:    "medium",
			Description: "使用navigator.clipboard在file协议下可能受限",
			Evidence:    fmt.Sprintf("文件 '%s' 中使用了navigator.clipboard", filePath),
			Suggestion:  "建议添加降级方案，使用document.execCommand('copy')作为备选",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		})
	}

	// 检查localStorage/sessionStorage在file协议下的兼容性
	if strings.Contains(content, "localStorage.") || strings.Contains(content, "sessionStorage.") {
		issues = append(issues, VerifyIssue{
			Type:        VerifyIssueTypeJSSyntaxError,
			Severity:    "low",
			Description: "localStorage/sessionStorage在file协议下可能受限",
			Evidence:    fmt.Sprintf("文件 '%s' 中使用了localStorage或sessionStorage", filePath),
			Suggestion:  "建议添加try-catch处理可能的异常",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		})
	}

	return issues
}

// checkHTMLStructure 检查HTML结构
func (v *Verifier) checkHTMLStructure(content string, filePath string) []VerifyIssue {
	issues := make([]VerifyIssue, 0)

	// 检查DOCTYPE声明
	if !strings.Contains(strings.ToUpper(content), "<!DOCTYPE") {
		issues = append(issues, VerifyIssue{
			Type:        VerifyIssueTypeHTMLStructureError,
			Severity:    "medium",
			Description: "HTML文档缺少DOCTYPE声明",
			Evidence:    fmt.Sprintf("文件 '%s' 未找到DOCTYPE声明", filePath),
			Suggestion:  "建议在文档开头添加 <!DOCTYPE html>",
			FilePath:    filePath,
			Timestamp:   time.Now(),
		})
	}

	// 检查基本HTML结构
	requiredTags := []string{"<html", "</html>", "<head>", "</head>", "<body>", "</body>"}
	for _, tag := range requiredTags {
		if !strings.Contains(strings.ToLower(content), strings.ToLower(tag)) {
			issues = append(issues, VerifyIssue{
				Type:        VerifyIssueTypeHTMLStructureError,
				Severity:    "medium",
				Description: fmt.Sprintf("HTML文档缺少必需标签: %s", tag),
				Evidence:    fmt.Sprintf("文件 '%s' 未找到标签 %s", filePath, tag),
				Suggestion:  "请确保HTML文档结构完整",
				FilePath:    filePath,
				Timestamp:   time.Now(),
			})
		}
	}

	// 检查常见未闭合标签
	issues = append(issues, v.checkUnclosedTags(content, filePath)...)

	return issues
}

// checkUnclosedTags 检查未闭合的HTML标签
func (v *Verifier) checkUnclosedTags(content string, filePath string) []VerifyIssue {
	issues := make([]VerifyIssue, 0)

	// 自闭合标签列表
	selfClosingTags := map[string]bool{
		"br": true, "hr": true, "img": true, "input": true,
		"meta": true, "link": true, "area": true, "base": true,
		"col": true, "embed": true, "param": true, "source": true,
		"track": true, "wbr": true,
	}

	// 提取所有标签
	tagRegex := regexp.MustCompile(`<(/?)(\w+)[^>]*>`)
	matches := tagRegex.FindAllStringSubmatch(content, -1)

	openTags := make(map[string]int)

	for _, match := range matches {
		isClosing := match[1] == "/"
		tagName := strings.ToLower(match[2])

		if selfClosingTags[tagName] {
			continue
		}

		if isClosing {
			if count, exists := openTags[tagName]; exists {
				if count > 0 {
					openTags[tagName]--
				}
			}
		} else {
			openTags[tagName]++
		}
	}

	// 检查未闭合的标签
	for tagName, count := range openTags {
		if count > 0 {
			issues = append(issues, VerifyIssue{
				Type:        VerifyIssueTypeHTMLStructureError,
				Severity:    "high",
				Description: fmt.Sprintf("检测到未闭合的HTML标签: <%s>", tagName),
				Evidence:    fmt.Sprintf("文件 '%s' 中 <%s> 标签有 %d 个未闭合", filePath, tagName, count),
				Suggestion:  fmt.Sprintf("请检查并确保所有 <%s> 标签正确闭合", tagName),
				FilePath:    filePath,
				Timestamp:   time.Now(),
			})
		}
	}

	return issues
}

// removeStringContent 移除字符串内容以便检查括号
func (v *Verifier) removeStringContent(content string) string {
	// 移除单引号字符串
	result := regexp.MustCompile(`'[^']*'`).ReplaceAllString(content, "''")
	// 移除双引号字符串
	result = regexp.MustCompile(`"[^"]*"`).ReplaceAllString(result, "\"\"")
	// 移除模板字符串
	result = regexp.MustCompile("`[^`]*`").ReplaceAllString(result, "``")
	// 移除注释
	result = regexp.MustCompile(`//[^\n]*`).ReplaceAllString(result, "")
	result = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(result, "")
	return result
}

// truncateString 截断字符串
func (v *Verifier) truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
