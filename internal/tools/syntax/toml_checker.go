package syntax

import (
	"strings"

	"github.com/BurntSushi/toml"
)

// TOMLChecker TOML语法检查器
// 使用github.com/BurntSushi/toml进行AST级别的语法验证
type TOMLChecker struct{}

// NewTOMLChecker 创建TOML语法检查器
func NewTOMLChecker() *TOMLChecker {
	return &TOMLChecker{}
}

// SupportedExtensions 返回支持的文件扩展名
func (c *TOMLChecker) SupportedExtensions() []string {
	return []string{".toml"}
}

// Check 对TOML内容进行语法检查
func (c *TOMLChecker) Check(content string, filePath string) *SyntaxCheckResult {
	// 空内容视为合法
	if strings.TrimSpace(content) == "" {
		return newSuccessCheckResult("toml", "native")
	}

	var val interface{}
	meta, err := toml.Decode(content, &val)
	// 检查是否有未解码的键（可能是重复键或拼写错误），这里只关注语法错误
	_ = meta

	if err != nil {
		errs := parseTOMLError(err)
		return newErrorCheckResult("toml", "native", errs)
	}

	return newSuccessCheckResult("toml", "native")
}

// parseTOMLError 解析TOML错误，提取行号和描述
func parseTOMLError(err error) []SyntaxError {
	errMsg := err.Error()

	syntaxErr := SyntaxError{
		Message:  errMsg,
		Severity: "error",
	}

	// BurntSushi/toml的ParseError包含行号和位置信息
	if pe, ok := err.(*toml.ParseError); ok {
		// 从Position字段获取行号
		syntaxErr.Line = pe.Position.Line
		// 使用Error()获取标准化的错误消息
		syntaxErr.Message = pe.Error()
	}

	// 如果没有从ParseError获取到行号，尝试从错误字符串解析
	if syntaxErr.Line == 0 {
		syntaxErr.Line = extractLineNumber(errMsg)
	}

	syntaxErr.Suggestion = tomlErrorSuggestion(errMsg)

	return []SyntaxError{syntaxErr}
}

// extractLineNumber 从错误消息字符串中提取行号
func extractLineNumber(errMsg string) int {
	// 尝试匹配 "line N" 或 "(N," 等常见行号格式
	for _, prefix := range []string{"line ", "Line "} {
		if idx := strings.Index(errMsg, prefix); idx >= 0 {
			remaining := errMsg[idx+len(prefix):]
			var lineNum int
			for i := 0; i < len(remaining); i++ {
				if remaining[i] >= '0' && remaining[i] <= '9' {
					lineNum = lineNum*10 + int(remaining[i]-'0')
				} else {
					break
				}
			}
			if lineNum > 0 {
				return lineNum
			}
		}
	}
	return 0
}

// tomlErrorSuggestion 根据TOML错误信息生成修正建议
func tomlErrorSuggestion(errMsg string) string {
	msgLower := strings.ToLower(errMsg)
	switch {
	case strings.Contains(msgLower, "duplicate"):
		return "Remove duplicate keys in the same table"
	case strings.Contains(msgLower, "invalid date") || strings.Contains(msgLower, "invalid datetime"):
		return "Use TOML date format: YYYY-MM-DD or YYYY-MM-DDTHH:MM:SSZ"
	case strings.Contains(msgLower, "invalid escape"):
		return "Use valid escape sequences: \\n, \\t, \\\", \\\\, \\b, \\f, \\r, \\uXXXX, \\UXXXXXXXX"
	case strings.Contains(msgLower, "bare key"):
		return "Use quotes for keys containing special characters, or use only A-Za-z0-9_-"
	case strings.Contains(msgLower, "expected"):
		return "Check for missing equals sign, bracket, or proper TOML syntax"
	case strings.Contains(msgLower, "newline"):
		return "Check for missing newline or incorrect multiline string syntax"
	case strings.Contains(msgLower, "unexpected"):
		return "Check for invalid characters or incorrect TOML syntax"
	default:
		return ""
	}
}
