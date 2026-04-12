package syntax

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// HTMLChecker HTML语法检查器
// 使用golang.org/x/net/html构建DOM树进行语法验证
type HTMLChecker struct{}

// NewHTMLChecker 创建HTML语法检查器
func NewHTMLChecker() *HTMLChecker {
	return &HTMLChecker{}
}

// SupportedExtensions 返回支持的文件扩展名
func (c *HTMLChecker) SupportedExtensions() []string {
	return []string{".html", ".htm"}
}

// Check 对HTML内容进行语法检查
func (c *HTMLChecker) Check(content string, filePath string) *SyntaxCheckResult {
	// 空内容视为合法
	if strings.TrimSpace(content) == "" {
		return newSuccessCheckResult("html", "native")
	}

	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		errs := parseHTMLError(err)
		return newErrorCheckResult("html", "native", errs)
	}

	// 解析成功，检查潜在问题
	var warnings []SyntaxWarning
	c.checkHTMLIssues(doc, &warnings)

	result := newSuccessCheckResult("html", "native")
	if len(warnings) > 0 {
		result.Warnings = warnings
	}

	return result
}

// parseHTMLError 解析HTML错误
func parseHTMLError(err error) []SyntaxError {
	errMsg := err.Error()

	syntaxErr := SyntaxError{
		Message:  errMsg,
		Severity: "error",
	}

	// 尝试提取行号信息（html.Parse的错误格式可能包含行号）
	if idx := strings.LastIndex(errMsg, "line "); idx >= 0 {
		// 尝试解析行号
		remaining := errMsg[idx+5:]
		var line int
		if _, err := fmt.Sscanf(remaining, "%d", &line); err == nil && line > 0 {
			syntaxErr.Line = line
		}
	}

	syntaxErr.Suggestion = htmlErrorSuggestion(errMsg)

	return []SyntaxError{syntaxErr}
}

// htmlErrorSuggestion 根据HTML错误信息生成修正建议
func htmlErrorSuggestion(errMsg string) string {
	msgLower := strings.ToLower(errMsg)
	switch {
	case strings.Contains(msgLower, "invalid character"):
		return "Ensure the file is valid UTF-8"
	case strings.Contains(msgLower, "expected"):
		return "Check for unclosed tags or malformed HTML structure"
	default:
		return "Check HTML structure for unclosed tags or invalid nesting"
	}
}

// checkHTMLIssues 检查HTML文档中的潜在问题
func (c *HTMLChecker) checkHTMLIssues(n *html.Node, warnings *[]SyntaxWarning) {
	if n == nil {
		return
	}

	// 检查自闭合标签是否被错误地使用
	if n.Type == html.ElementNode {
		// voidElements 是HTML中不应有闭合标签的元素
		voidElements := map[string]bool{
			"area": true, "base": true, "br": true, "col": true,
			"embed": true, "hr": true, "img": true, "input": true,
			"link": true, "meta": true, "param": true, "source": true,
			"track": true, "wbr": true,
		}

		if voidElements[n.Data] && n.FirstChild != nil {
			*warnings = append(*warnings, SyntaxWarning{
				Message:    fmt.Sprintf("void element <%s> should not have content", n.Data),
				Severity:   "warning",
				Suggestion: fmt.Sprintf("Remove content from <%s> or use a non-void element", n.Data),
			})
		}
	}

	// 递归检查子节点
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		c.checkHTMLIssues(child, warnings)
	}
}
