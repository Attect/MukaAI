package syntax

import (
	"encoding/json"
	"strings"
)

// JSONChecker JSON语法检查器
// 使用encoding/json标准库进行语法验证
type JSONChecker struct{}

// NewJSONChecker 创建JSON语法检查器
func NewJSONChecker() *JSONChecker {
	return &JSONChecker{}
}

// SupportedExtensions 返回支持的文件扩展名
func (c *JSONChecker) SupportedExtensions() []string {
	return []string{".json"}
}

// Check 对JSON内容进行语法检查
func (c *JSONChecker) Check(content string, filePath string) *SyntaxCheckResult {
	// 空内容视为合法（空JSON文件）
	if strings.TrimSpace(content) == "" {
		return newSuccessCheckResult("json", "native")
	}

	var syntaxError *json.SyntaxError
	dec := json.NewDecoder(strings.NewReader(content))

	// 尝试解析JSON
	var val interface{}
	if err := dec.Decode(&val); err != nil {
		if se, ok := err.(*json.SyntaxError); ok {
			syntaxError = se
		} else {
			// 其他类型的错误（如unmarshal类型错误），不属于语法错误
			// json.Decoder会在遇到非JSON内容时返回SyntaxError
			// 如果是其他错误，尝试用Unmarshal验证
			if err2 := json.Unmarshal([]byte(content), &val); err2 != nil {
				if se2, ok := err2.(*json.SyntaxError); ok {
					syntaxError = se2
				} else {
					// 真正的非语法错误（理论上不应出现），按语法错误处理
					return newErrorCheckResult("json", "native", []SyntaxError{
						{
							Message:  err2.Error(),
							Severity: "error",
						},
					})
				}
			}
		}
	}

	if syntaxError == nil {
		return newSuccessCheckResult("json", "native")
	}

	// 从偏移量计算行号和列号
	line, column := offsetToLineCol(content, int(syntaxError.Offset))

	errs := []SyntaxError{
		{
			Line:     line,
			Column:   column,
			Message:  syntaxError.Error(),
			Severity: "error",
		},
	}

	// 根据常见JSON错误添加修正建议
	suggestion := jsonErrorSuggestion(syntaxError.Error())
	if suggestion != "" {
		errs[0].Suggestion = suggestion
	}

	return newErrorCheckResult("json", "native", errs)
}

// jsonErrorSuggestion 根据JSON错误信息生成修正建议
func jsonErrorSuggestion(errMsg string) string {
	msg := strings.ToLower(errMsg)
	switch {
	case strings.Contains(msg, "unexpected end"):
		return "Check for unclosed brackets [], braces {}, or quotes"
	case strings.Contains(msg, "invalid character"):
		return "Verify JSON syntax: strings must be in double quotes, no trailing commas"
	case strings.Contains(msg, "unexpected character"):
		return "Check for extra or missing characters, ensure proper JSON structure"
	default:
		return ""
	}
}

// offsetToLineCol 将字节偏移量转换为行号和列号（均为1-based）
func offsetToLineCol(content string, offset int) (int, int) {
	if offset <= 0 {
		return 1, 1
	}
	if offset > len(content) {
		offset = len(content)
	}

	line := 1
	col := 1
	for i := 0; i < offset && i < len(content); i++ {
		if content[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}
