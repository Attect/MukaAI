package syntax

import (
	"strings"
)

// SQLChecker SQL基础语法检查器
// 使用Go标准库逐Token扫描，检查字符串闭合、括号匹配和注释闭合
type SQLChecker struct{}

// NewSQLChecker 创建SQL语法检查器
func NewSQLChecker() *SQLChecker {
	return &SQLChecker{}
}

// SupportedExtensions 返回支持的文件扩展名
func (c *SQLChecker) SupportedExtensions() []string {
	return []string{".sql"}
}

// Check 对SQL内容进行语法检查
func (c *SQLChecker) Check(content string, filePath string) *SyntaxCheckResult {
	// 空内容视为合法
	if strings.TrimSpace(content) == "" {
		return newSuccessCheckResult("sql", "native")
	}

	var errs []SyntaxError

	// 使用逐字符扫描方式检查基本结构完整性
	sqlErrs := scanSQLContent(content)
	errs = append(errs, sqlErrs...)

	if len(errs) > 0 {
		return newErrorCheckResult("sql", "native", errs)
	}

	return newSuccessCheckResult("sql", "native")
}

// sqlScanState SQL扫描器状态
type sqlScanState struct {
	line      int  // 当前行号
	col       int  // 当前列号
	inString  bool // 是否在单引号字符串内
	inIdent   bool // 是否在双引号标识符内
	quoteChar byte // 当前引号字符
}

// scanSQLContent 逐字符扫描SQL内容，检查基本结构完整性
func scanSQLContent(content string) []SyntaxError {
	var errs []SyntaxError
	state := sqlScanState{line: 1, col: 1}

	// 括号栈：追踪 ( 和 [ 的匹配
	var parenStack []parenInfo

	runes := []rune(content)
	i := 0

	for i < len(runes) {
		ch := runes[i]

		switch {
		// 换行
		case ch == '\n':
			state.line++
			state.col = 1
			i++
			continue

		// 单行注释 --
		case !state.inString && !state.inIdent && ch == '-' && i+1 < len(runes) && runes[i+1] == '-':
			// 跳过到行末
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
			continue

		// 块注释 /* */
		case !state.inString && !state.inIdent && ch == '/' && i+1 < len(runes) && runes[i+1] == '*':
			commentStartLine := state.line
			i += 2
			state.col += 2
			closed := false
			for i < len(runes) {
				if runes[i] == '\n' {
					state.line++
					state.col = 1
				} else {
					state.col++
				}
				if runes[i] == '*' && i+1 < len(runes) && runes[i+1] == '/' {
					i += 2
					closed = true
					break
				}
				i++
			}
			if !closed {
				errs = append(errs, SyntaxError{
					Line:       commentStartLine,
					Message:    "unclosed block comment, missing '*/'",
					Severity:   "error",
					Suggestion: "Close the block comment with '*/'",
				})
			}
			continue

		// 字符串（单引号）
		case !state.inIdent && ch == '\'':
			if !state.inString {
				state.inString = true
				state.quoteChar = '\''
				state.col++
				i++
				// 读取字符串内容
				for i < len(runes) {
					if runes[i] == '\n' {
						state.line++
						state.col = 1
						i++
						continue
					}
					if runes[i] == '\'' {
						// 检查转义：SQL用 '' 表示字面单引号
						if i+1 < len(runes) && runes[i+1] == '\'' {
							i += 2
							state.col += 2
							continue
						}
						// 也检查反斜杠转义（MySQL风格）
						state.inString = false
						i++
						state.col++
						break
					}
					// 反斜杠转义
					if runes[i] == '\\' && i+1 < len(runes) {
						i += 2
						state.col += 2
						continue
					}
					state.col++
					i++
				}
				if state.inString {
					errs = append(errs, SyntaxError{
						Line:       state.line,
						Message:    "unclosed string literal, missing closing single quote",
						Severity:   "error",
						Suggestion: "Add a closing single quote to complete the string literal",
					})
				}
			}
			continue

		// 标识符引号（双引号）
		case !state.inString && ch == '"':
			if !state.inIdent {
				state.inIdent = true
				state.col++
				i++
				for i < len(runes) {
					if runes[i] == '\n' {
						state.line++
						state.col = 1
						i++
						continue
					}
					if runes[i] == '"' {
						if i+1 < len(runes) && runes[i+1] == '"' {
							i += 2
							state.col += 2
							continue
						}
						state.inIdent = false
						i++
						state.col++
						break
					}
					state.col++
					i++
				}
				if state.inIdent {
					errs = append(errs, SyntaxError{
						Line:       state.line,
						Message:    "unclosed quoted identifier, missing closing double quote",
						Severity:   "error",
						Suggestion: "Add a closing double quote to complete the identifier",
					})
				}
			}
			continue

		// 左括号
		case !state.inString && !state.inIdent && ch == '(':
			parenStack = append(parenStack, parenInfo{
				char: '(',
				line: state.line,
				col:  state.col,
			})
			state.col++
			i++

		// 右括号
		case !state.inString && !state.inIdent && ch == ')':
			if len(parenStack) > 0 && parenStack[len(parenStack)-1].char == '(' {
				parenStack = parenStack[:len(parenStack)-1]
			} else {
				errs = append(errs, SyntaxError{
					Line:       state.line,
					Message:    "unexpected ')', no matching '('",
					Severity:   "error",
					Suggestion: "Check for extra closing parenthesis or missing opening parenthesis",
				})
			}
			state.col++
			i++

		// 左方括号
		case !state.inString && !state.inIdent && ch == '[':
			parenStack = append(parenStack, parenInfo{
				char: '[',
				line: state.line,
				col:  state.col,
			})
			state.col++
			i++

		// 右方括号
		case !state.inString && !state.inIdent && ch == ']':
			if len(parenStack) > 0 && parenStack[len(parenStack)-1].char == '[' {
				parenStack = parenStack[:len(parenStack)-1]
			} else {
				errs = append(errs, SyntaxError{
					Line:       state.line,
					Message:    "unexpected ']', no matching '['",
					Severity:   "error",
					Suggestion: "Check for extra closing bracket or missing opening bracket",
				})
			}
			state.col++
			i++

		default:
			state.col++
			i++
		}
	}

	// 检查未闭合的括号
	for _, p := range parenStack {
		closeChar := ")"
		if p.char == '[' {
			closeChar = "]"
		}
		errs = append(errs, SyntaxError{
			Line:       p.line,
			Message:    "unclosed '" + string(p.char) + "', missing '" + closeChar + "'",
			Severity:   "error",
			Suggestion: "Add closing '" + closeChar + "' to match the opening '" + string(p.char) + "'",
		})
	}

	return errs
}

// parenInfo 括号位置信息
type parenInfo struct {
	char rune
	line int
	col  int
}
