package syntax

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// XMLChecker XML语法检查器
// 使用encoding/xml标准库进行语法验证
type XMLChecker struct{}

// NewXMLChecker 创建XML语法检查器
func NewXMLChecker() *XMLChecker {
	return &XMLChecker{}
}

// SupportedExtensions 返回支持的文件扩展名
func (c *XMLChecker) SupportedExtensions() []string {
	return []string{".xml", ".xsd", ".xsl", ".xslt", ".svg", ".pom"}
}

// Check 对XML内容进行语法检查
func (c *XMLChecker) Check(content string, filePath string) *SyntaxCheckResult {
	// 空内容视为合法
	if strings.TrimSpace(content) == "" {
		return newSuccessCheckResult("xml", "native")
	}

	decoder := xml.NewDecoder(strings.NewReader(content))
	decoder.Strict = true
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity

	var errs []SyntaxError
	for {
		_, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			// 解析XML错误
			syntaxErr := parseXMLError(err)
			errs = append(errs, syntaxErr)
			// 继续尝试解析以收集更多错误，但限制最多10个
			if len(errs) >= 10 {
				break
			}
			// 严重错误时停止
			if strings.Contains(err.Error(), "invalid character") &&
				!strings.Contains(err.Error(), "entity") {
				break
			}
		}
	}

	if len(errs) == 0 {
		return newSuccessCheckResult("xml", "native")
	}

	return newErrorCheckResult("xml", "native", errs)
}

// parseXMLError 解析XML错误，提取行号和信息
func parseXMLError(err error) SyntaxError {
	errMsg := err.Error()

	syntaxErr := SyntaxError{
		Message:  errMsg,
		Severity: "error",
	}

	// 尝试从xml.SyntaxError提取行号
	if xmlSE, ok := err.(*xml.SyntaxError); ok {
		syntaxErr.Line = xmlSE.Line
		syntaxErr.Message = xmlSE.Error()
	}

	// 添加修正建议
	syntaxErr.Suggestion = xmlErrorSuggestion(errMsg)

	return syntaxErr
}

// xmlErrorSuggestion 根据XML错误信息生成修正建议
func xmlErrorSuggestion(errMsg string) string {
	msgLower := strings.ToLower(errMsg)
	switch {
	case strings.Contains(msgLower, "invalid character entity"):
		return "Use valid XML entities: &amp; &lt; &gt; &quot; &apos;"
	case strings.Contains(msgLower, "expected element name"):
		return "Check for unclosed tags or malformed element syntax"
	case strings.Contains(msgLower, "invalid character"):
		return "Ensure the file is valid UTF-8 and contains only legal XML characters"
	case strings.Contains(msgLower, "unclosed"):
		return fmt.Sprintf("Check for unclosed tags or attributes: %s", errMsg)
	case strings.Contains(msgLower, "xml name"):
		return "Element and attribute names must start with a letter or underscore"
	case strings.Contains(msgLower, "expected"):
		return "Check for missing closing tags, quotes, or proper XML structure"
	default:
		return ""
	}
}
