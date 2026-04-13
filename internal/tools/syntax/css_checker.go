package syntax

import (
	"io"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// CSSChecker CSS语法检查器
// 使用github.com/tdewolff/parse/v2/css进行Token级别的语法验证
type CSSChecker struct{}

// NewCSSChecker 创建CSS语法检查器
func NewCSSChecker() *CSSChecker {
	return &CSSChecker{}
}

// SupportedExtensions 返回支持的文件扩展名
func (c *CSSChecker) SupportedExtensions() []string {
	return []string{".css"}
}

// Check 对CSS内容进行语法检查
func (c *CSSChecker) Check(content string, filePath string) *SyntaxCheckResult {
	// 空内容视为合法
	if strings.TrimSpace(content) == "" {
		return newSuccessCheckResult("css", "native")
	}

	// 使用css.NewLexer解析Token流
	input := parse.NewInputString(content)
	lexer := css.NewLexer(input)

	var errs []SyntaxError
	braceDepth := 0 // 追踪花括号嵌套深度
	line := 1

	for {
		tt, tokenVal := lexer.Next()
		if tt == css.ErrorToken {
			// 检查是否是EOF（正常结束）
			if lexer.Err() == io.EOF {
				break
			}
			// 解析错误
			errMsg := string(tokenVal)
			if errMsg == "" && lexer.Err() != nil {
				errMsg = lexer.Err().Error()
			}
			errs = append(errs, SyntaxError{
				Line:       line,
				Message:    errMsg,
				Severity:   "error",
				Suggestion: cssErrorSuggestion(errMsg),
			})
			break
		}

		// 统计换行以追踪行号
		line += strings.Count(string(tokenVal), "\n")

		// 追踪花括号匹配
		switch tt {
		case css.LeftBraceToken:
			braceDepth++
		case css.RightBraceToken:
			braceDepth--
			if braceDepth < 0 {
				errs = append(errs, SyntaxError{
					Line:       line,
					Message:    "unexpected closing brace '}'",
					Severity:   "error",
					Suggestion: "Check for extra closing braces or missing opening braces",
				})
				braceDepth = 0 // 重置以避免连续报错
			}
		}
	}

	// 检查未闭合的花括号
	if braceDepth > 0 {
		errs = append(errs, SyntaxError{
			Line:       line,
			Message:    "unclosed brace, missing '}' to close block",
			Severity:   "error",
			Suggestion: "Add missing closing brace(s) for CSS rule block(s)",
		})
	}

	if len(errs) > 0 {
		return newErrorCheckResult("css", "native", errs)
	}

	return newSuccessCheckResult("css", "native")
}

// cssErrorSuggestion 根据CSS错误信息生成修正建议
func cssErrorSuggestion(errMsg string) string {
	msgLower := strings.ToLower(errMsg)
	switch {
	case strings.Contains(msgLower, "unexpected"):
		return "Check for invalid CSS syntax or misplaced tokens"
	case strings.Contains(msgLower, "invalid"):
		return "Verify CSS property names and values are correct"
	default:
		return "Check CSS syntax for unclosed rules or invalid selectors"
	}
}
