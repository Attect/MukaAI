package syntax

import (
	"fmt"
	"go/parser"
	"go/token"
	"strings"
)

// GoChecker Go语言语法检查器
// 使用go/parser + go/token标准库进行语法验证
type GoChecker struct{}

// NewGoChecker 创建Go语言语法检查器
func NewGoChecker() *GoChecker {
	return &GoChecker{}
}

// SupportedExtensions 返回支持的文件扩展名
func (c *GoChecker) SupportedExtensions() []string {
	return []string{".go"}
}

// Check 对Go源码进行语法检查
func (c *GoChecker) Check(content string, filePath string) *SyntaxCheckResult {
	// 空内容视为合法
	if strings.TrimSpace(content) == "" {
		return newSuccessCheckResult("go", "native")
	}

	fset := token.NewFileSet()
	// 仅做语法级检查，不导入外部包
	// parser.ParseFile 的 AllErrors 模式会收集所有语法错误
	_, err := parser.ParseFile(fset, filePath, strings.NewReader(content), parser.AllErrors)
	if err == nil {
		return newSuccessCheckResult("go", "native")
	}

	errs := parseGoErrors(err, fset)
	return newErrorCheckResult("go", "native", errs)
}

// parseGoErrors 解析Go解析器返回的错误
// go/parser 返回的是 scanner.ErrorList 类型
func parseGoErrors(err error, fset *token.FileSet) []SyntaxError {
	var errs []SyntaxError

	// 尝试作为scanner.ErrorList处理
	if errList, ok := err.(interface{ Error() string }); ok {
		errMsg := errList.Error()

		// scanner.ErrorList 的输出格式为 "file:line:col: message\n..."
		// 尝试逐行解析
		lines := strings.Split(errMsg, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			syntaxErr := parseGoErrorLine(line)
			if syntaxErr != nil {
				syntaxErr.Severity = "error"
				syntaxErr.Suggestion = goErrorSuggestion(syntaxErr.Message)
				errs = append(errs, *syntaxErr)
			}
		}
	}

	// 如果没有解析出任何错误，使用原始错误信息
	if len(errs) == 0 {
		errs = append(errs, SyntaxError{
			Message:    err.Error(),
			Severity:   "error",
			Suggestion: "Check Go syntax for missing brackets, braces, or keywords",
		})
	}

	return errs
}

// parseGoErrorLine 解析单行Go错误 "file:line:col: message"
func parseGoErrorLine(line string) *SyntaxError {
	// 格式: "/path/to/file.go:10:5: expected statement"
	parts := strings.SplitN(line, ":", 4)
	if len(parts) < 4 {
		return &SyntaxError{Message: line}
	}

	var lineNum, colNum int
	fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &lineNum)
	fmt.Sscanf(strings.TrimSpace(parts[2]), "%d", &colNum)
	msg := strings.TrimSpace(parts[3])

	return &SyntaxError{
		Line:    lineNum,
		Column:  colNum,
		Message: msg,
	}
}

// goErrorSuggestion 根据Go错误信息生成修正建议
func goErrorSuggestion(msg string) string {
	msgLower := strings.ToLower(msg)
	switch {
	case strings.Contains(msgLower, "expected declaration"):
		return "Check for misplaced statements outside of functions or incorrect package structure"
	case strings.Contains(msgLower, "expected statement"):
		return "Check for incomplete statements or syntax errors in expressions"
	case strings.Contains(msgLower, "unexpected"):
		return "Check for extra or misplaced tokens, missing operators or delimiters"
	case strings.Contains(msgLower, "expected '}'"):
		return "Check for unclosed braces, ensure all blocks are properly closed"
	case strings.Contains(msgLower, "expected ')'"):
		return "Check for unclosed parentheses in function calls or expressions"
	case strings.Contains(msgLower, "expected ']'"):
		return "Check for unclosed brackets in slice/array/map literals"
	case strings.Contains(msgLower, "non-declaration statement"):
		return "Ensure statements are inside a function body"
	case strings.Contains(msgLower, "import"):
		return "Check import syntax: import \"path\" or import ( \"path1\" \"path2\" )"
	default:
		return ""
	}
}
