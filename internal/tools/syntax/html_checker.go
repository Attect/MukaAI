package syntax

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// 内联JS检查超时时间
const jsCheckTimeout = 5 * time.Second

// nodeAvailableCache 缓存node是否可用（进程级）
var (
	nodeAvailable     bool
	nodeAvailableOnce sync.Once
)

// isNodeAvailable 检查node命令是否可用
func isNodeAvailable() bool {
	nodeAvailableOnce.Do(func() {
		_, err := exec.LookPath("node")
		nodeAvailable = err == nil
	})
	return nodeAvailable
}

// scriptInfo 描述HTML中提取的<script>标签信息
type scriptInfo struct {
	// Index 是该script标签在文档中的顺序（0-based）
	Index int
	// Content 是script标签内的文本内容
	Content string
	// Type 是script标签的type属性值
	Type string
}

// HTMLChecker HTML语法检查器
// 使用golang.org/x/net/html构建DOM树进行语法验证
// 增强功能：提取<script>标签内的JavaScript代码并使用node --check进行检查
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
// 1. 先进行HTML结构检查
// 2. 提取<script>标签内的JavaScript代码
// 3. 使用node --check对内联JS进行语法检查
// 4. JS错误以"内联脚本"形式附加到Warnings中
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

	// 解析成功，检查HTML结构问题
	var warnings []SyntaxWarning
	c.checkHTMLIssues(doc, &warnings)

	// 提取并检查内联JavaScript
	scripts := extractScriptContents(doc)
	for _, script := range scripts {
		if !isJSScriptType(script.Type) {
			continue
		}
		jsWarnings := checkInlineJS(script.Content, script.Index)
		warnings = append(warnings, jsWarnings...)
	}

	result := newSuccessCheckResult("html", "native")
	if len(warnings) > 0 {
		result.Warnings = warnings
	}

	return result
}

// isJSScriptType 判断script标签的type属性是否为JavaScript类型
// 空type（默认）、text/javascript、module等为JS类型
// importmap、application/json、text/template等非JS类型跳过
func isJSScriptType(scriptType string) bool {
	if scriptType == "" {
		return true // 无type属性，默认为JavaScript
	}
	lower := strings.ToLower(strings.TrimSpace(scriptType))
	switch lower {
	case "text/javascript", "module", "application/javascript",
		"text/ecmascript", "application/ecmascript":
		return true
	default:
		return false
	}
}

// extractScriptContents 从HTML DOM树中提取所有<script>标签的文本内容和属性
func extractScriptContents(doc *html.Node) []scriptInfo {
	var scripts []scriptInfo

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n == nil {
			return
		}

		if n.Type == html.ElementNode && n.Data == "script" {
			// 获取type属性
			scriptType := ""
			for _, attr := range n.Attr {
				if attr.Key == "type" {
					scriptType = attr.Val
					break
				}
			}

			// 提取文本内容（递归收集所有文本节点）
			var content strings.Builder
			var extractText func(*html.Node)
			extractText = func(node *html.Node) {
				if node.Type == html.TextNode {
					content.WriteString(node.Data)
				}
				for c := node.FirstChild; c != nil; c = c.NextSibling {
					extractText(c)
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				extractText(c)
			}

			scripts = append(scripts, scriptInfo{
				Index:   len(scripts),
				Content: content.String(),
				Type:    scriptType,
			})
		}

		// 递归遍历子节点
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)
	return scripts
}

// checkInlineJS 使用node --check检查内联JavaScript代码
// 将JS内容写入临时文件后调用node检查，错误作为Warning返回
func checkInlineJS(jsContent string, scriptIndex int) []SyntaxWarning {
	if strings.TrimSpace(jsContent) == "" {
		return nil
	}

	if !isNodeAvailable() {
		return nil
	}

	// 写入临时文件
	tmpFile, err := os.CreateTemp("", "inline_js_check_*.js")
	if err != nil {
		return nil
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(jsContent)
	if err != nil {
		tmpFile.Close()
		return nil
	}
	tmpFile.Close()

	// 执行node --check
	ctx, cancel := context.WithTimeout(context.Background(), jsCheckTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", "--check", tmpFile.Name())
	// Windows 下隐藏控制台窗口（避免闪窗）
	configureHideWindow(cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err == nil {
		return nil // 无语法错误
	}

	// 解析node错误输出
	nodeErrs := parseNodeJSErrors(stderr.String(), "javascript")
	var warnings []SyntaxWarning
	for _, e := range nodeErrs {
		w := SyntaxWarning{
			Line:       e.Line,
			Column:     e.Column,
			Message:    fmt.Sprintf("内联脚本（第%d个<script>标签）: %s", scriptIndex+1, e.Message),
			Severity:   "warning",
			Suggestion: e.Suggestion,
		}
		warnings = append(warnings, w)
	}

	return warnings
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
