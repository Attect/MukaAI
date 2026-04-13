package syntax

import (
	"strings"
)

// PropertiesChecker Properties格式检查器
// 使用Go标准库逐行解析，验证转义序列和行延续符
type PropertiesChecker struct{}

// NewPropertiesChecker 创建Properties格式检查器
func NewPropertiesChecker() *PropertiesChecker {
	return &PropertiesChecker{}
}

// SupportedExtensions 返回支持的文件扩展名
func (c *PropertiesChecker) SupportedExtensions() []string {
	return []string{".properties"}
}

// Check 对Properties文件内容进行语法检查
func (c *PropertiesChecker) Check(content string, filePath string) *SyntaxCheckResult {
	// 空内容视为合法
	if strings.TrimSpace(content) == "" {
		return newSuccessCheckResult("properties", "native")
	}

	var errs []SyntaxError
	var warnings []SyntaxWarning

	lines := strings.Split(content, "\n")
	lineNum := 0
	continuation := false // 上一行是否以\结尾（行延续）

	for _, rawLine := range lines {
		lineNum++
		line := rawLine

		// 处理行延续：如果上一行以\结尾，本行是上一行的续行
		if continuation {
			continuation = false
			// 检查本行是否也以\结尾
			if hasContinuation(line) {
				continuation = true
			}
			continue
		}

		// 去除行首空白（但保留原始行用于行延续检查）
		trimmed := strings.TrimLeft(line, " \t")

		// 空行
		if trimmed == "" {
			continue
		}

		// 注释行（以#或!开头）
		if trimmed[0] == '#' || trimmed[0] == '!' {
			continue
		}

		// 检查行延续符
		if hasContinuation(line) {
			continuation = true
		}

		// 验证转义序列（在值部分）
		valueStart := findValueStart(trimmed)
		if valueStart >= 0 {
			value := trimmed[valueStart:]
			validateEscapes(value, lineNum, &errs, &warnings)
		}

		// 检查键分隔符（=或:）
		sepIdx := findKeySeparator(trimmed)
		if sepIdx < 0 && !continuation {
			// 没有分隔符的行，不一定是错误（可能是空值键）
			// 但如果整行不是注释也不是空行，给出警告
			// 注意：Properties格式允许 key=value, key:value, key value（空格分隔）
			// 所以只要有非空白字符就算合法
		}
	}

	// 如果文件末尾仍在行延续中，这是不完整的
	if continuation {
		errs = append(errs, SyntaxError{
			Line:       lineNum,
			Message:    "file ends with a continuation line (trailing backslash)",
			Severity:   "error",
			Suggestion: "Remove the trailing backslash or add the continuation content",
		})
	}

	if len(errs) > 0 {
		return newErrorCheckResult("properties", "native", errs)
	}

	result := newSuccessCheckResult("properties", "native")
	if len(warnings) > 0 {
		result.Warnings = warnings
	}

	return result
}

// hasContinuation 检查行是否以延续符结尾
// 延续符是行末的反斜杠（\），但不是转义的（\\）
func hasContinuation(line string) bool {
	// 从行末向前找，跳过空白
	idx := len(line) - 1
	for idx >= 0 && (line[idx] == ' ' || line[idx] == '\t' || line[idx] == '\r') {
		idx--
	}
	if idx < 0 {
		return false
	}
	if line[idx] != '\\' {
		return false
	}
	// 检查反斜杠数量，偶数个表示都是转义，奇数个表示最后一个\是延续符
	count := 0
	for idx >= 0 && line[idx] == '\\' {
		count++
		idx--
	}
	return count%2 == 1
}

// findValueStart 找到值部分的起始位置
// 在键分隔符（=或:或首个空白）之后
func findValueStart(line string) int {
	i := 0
	// 跳过键（到分隔符为止）
	for i < len(line) {
		ch := line[i]
		if ch == '\\' && i+1 < len(line) {
			// 转义字符，跳过两个字符
			i += 2
			continue
		}
		if ch == '=' || ch == ':' {
			i++
			// 跳过分隔符后的空白
			for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
				i++
			}
			return i
		}
		if ch == ' ' || ch == '\t' {
			// 空格分隔符
			for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
				i++
			}
			// 如果后面是=或:，继续跳过
			if i < len(line) && (line[i] == '=' || line[i] == ':') {
				i++
				for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
					i++
				}
			}
			return i
		}
		i++
	}
	return -1
}

// findKeySeparator 查找键分隔符位置
func findKeySeparator(line string) int {
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '\\' && i+1 < len(line) {
			i++ // 跳过转义
			continue
		}
		if ch == '=' || ch == ':' {
			return i
		}
		if ch == ' ' || ch == '\t' {
			return i
		}
	}
	return -1
}

// validateEscapes 验证转义序列的有效性
func validateEscapes(value string, lineNum int, errs *[]SyntaxError, warnings *[]SyntaxWarning) {
	i := 0
	for i < len(value) {
		if value[i] != '\\' {
			i++
			continue
		}
		i++ // 跳过反斜杠
		if i >= len(value) {
			// 行末的反斜杠是行延续符，由上层处理
			return
		}

		ch := value[i]
		switch ch {
		// 合法转义序列
		case 'n', 't', 'r', '\\', '=', ':', '#', '!', ' ':
			// 这些都是合法的
			i++
		case 'u':
			// Unicode转义 \uXXXX
			i++
			if i+4 > len(value) {
				*errs = append(*errs, SyntaxError{
					Line:       lineNum,
					Message:    "incomplete Unicode escape sequence, expected \\uXXXX",
					Severity:   "error",
					Suggestion: "Provide 4 hexadecimal digits after \\u, e.g. \\u0041",
				})
				return
			}
			for j := 0; j < 4; j++ {
				hex := value[i+j]
				if !isHexDigit(hex) {
					*errs = append(*errs, SyntaxError{
						Line:       lineNum,
						Message:    "invalid Unicode escape sequence, \\u must be followed by 4 hex digits",
						Severity:   "error",
						Suggestion: "Use valid hexadecimal digits (0-9, A-F, a-f) in \\uXXXX",
					})
					return
				}
			}
			i += 4
		default:
			// 未知转义序列，发出警告而非错误
			// Properties规范中，未知转义序列的行为未严格定义
			*warnings = append(*warnings, SyntaxWarning{
				Line:       lineNum,
				Message:    "unknown escape sequence '\\" + string(ch) + "'",
				Severity:   "warning",
				Suggestion: "Use valid escape sequences: \\n \\t \\r \\\\ \\= \\: \\uXXXX",
			})
			i++
		}
	}
}

// isHexDigit 检查是否为十六进制数字
func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}
