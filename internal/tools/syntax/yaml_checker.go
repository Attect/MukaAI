package syntax

import (
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLChecker YAML语法检查器
// 使用gopkg.in/yaml.v3进行语法验证
type YAMLChecker struct{}

// NewYAMLChecker 创建YAML语法检查器
func NewYAMLChecker() *YAMLChecker {
	return &YAMLChecker{}
}

// SupportedExtensions 返回支持的文件扩展名
func (c *YAMLChecker) SupportedExtensions() []string {
	return []string{".yaml", ".yml"}
}

// Check 对YAML内容进行语法检查
func (c *YAMLChecker) Check(content string, filePath string) *SyntaxCheckResult {
	// 空内容视为合法
	if strings.TrimSpace(content) == "" {
		return newSuccessCheckResult("yaml", "native")
	}

	var val interface{}
	if err := yaml.Unmarshal([]byte(content), &val); err != nil {
		errs := parseYAMLError(err.Error())
		return newErrorCheckResult("yaml", "native", errs)
	}

	return newSuccessCheckResult("yaml", "native")
}

// yamlErrorPattern 匹配yaml错误输出中的行号信息
// 例如: "yaml: line 3: could not find expected ':'"
var yamlLinePattern = regexp.MustCompile(`yaml:\s*line\s+(\d+):\s*(.+)`)

// yamlErrorPattern2 匹配其他格式的yaml错误
// 例如: "yaml: unmarshal errors:\n  line 3: cannot unmarshal ..."
var yamlLinePattern2 = regexp.MustCompile(`line\s+(\d+):\s*(.+)`)

// parseYAMLError 解析YAML错误信息，提取行号和描述
func parseYAMLError(errMsg string) []SyntaxError {
	var errs []SyntaxError

	// 尝试匹配 "yaml: line N: message" 格式
	if matches := yamlLinePattern.FindStringSubmatch(errMsg); len(matches) >= 3 {
		line, _ := strconv.Atoi(matches[1])
		msg := matches[2]
		syntaxErr := SyntaxError{
			Line:     line,
			Message:  msg,
			Severity: "error",
		}
		syntaxErr.Suggestion = yamlErrorSuggestion(msg)
		errs = append(errs, syntaxErr)
		return errs
	}

	// 尝试匹配多行错误 "line N: message"
	lines := strings.Split(errMsg, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if matches := yamlLinePattern2.FindStringSubmatch(line); len(matches) >= 3 {
			lineNum, _ := strconv.Atoi(matches[1])
			msg := matches[2]
			syntaxErr := SyntaxError{
				Line:     lineNum,
				Message:  msg,
				Severity: "error",
			}
			syntaxErr.Suggestion = yamlErrorSuggestion(msg)
			errs = append(errs, syntaxErr)
		}
	}

	// 如果未能解析出行号，返回原始错误
	if len(errs) == 0 {
		errs = append(errs, SyntaxError{
			Message:  errMsg,
			Severity: "error",
		})
	}

	return errs
}

// yamlErrorSuggestion 根据YAML错误信息生成修正建议
func yamlErrorSuggestion(msg string) string {
	msgLower := strings.ToLower(msg)
	switch {
	case strings.Contains(msgLower, "indent"):
		return "Check YAML indentation, use consistent spaces (not tabs)"
	case strings.Contains(msgLower, "could not find expected"),
		strings.Contains(msgLower, "expected"):
		return "Check for missing colons, incorrect nesting, or unclosed quotes"
	case strings.Contains(msgLower, "control character"):
		return "Remove or escape control characters in string values"
	case strings.Contains(msgLower, "invalid escape"):
		return "Use valid escape sequences or wrap strings in quotes"
	case strings.Contains(msgLower, "found character that cannot start any token"):
		return "Check for special characters that need quoting"
	default:
		return ""
	}
}
