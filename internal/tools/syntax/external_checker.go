package syntax

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// 外部工具检查超时时间
const externalCheckTimeout = 5 * time.Second

// externalToolDef 定义外部检查工具的配置
type externalToolDef struct {
	// language 语言名称
	language string
	// extensions 支持的文件扩展名
	extensions []string
	// command 外部工具命令名（用于可用性检测）
	command string
	// buildArgs 根据临时文件路径构建命令参数
	buildArgs func(tmpFile string) []string
	// parseErrors 解析外部工具的stderr输出为语法错误
	parseErrors func(stderr string, language string) []SyntaxError
}

// ExternalChecker 外部工具语法检查器
// 统一管理通过调用外部进程执行语法检查的逻辑
// 支持并发安全，内置工具可用性缓存
type ExternalChecker struct {
	// tools 外部工具定义列表
	tools []externalToolDef

	// extToTool 扩展名→工具定义映射
	extToTool map[string]*externalToolDef

	// availabilityCache 工具可用性缓存（进程级）
	// key: command name, value: true=可用, false=不可用
	availabilityCache map[string]bool

	// mu 保护可用性缓存
	mu sync.RWMutex
}

// NewExternalChecker 创建外部工具检查器
func NewExternalChecker() *ExternalChecker {
	c := &ExternalChecker{
		extToTool:         make(map[string]*externalToolDef),
		availabilityCache: make(map[string]bool),
	}
	c.initTools()
	return c
}

// initTools 初始化所有外部工具定义
func (c *ExternalChecker) initTools() {
	c.tools = []externalToolDef{
		// JavaScript (§9.3.3.2)
		{
			language:   "javascript",
			extensions: []string{".js", ".mjs"},
			command:    "node",
			buildArgs: func(tmpFile string) []string {
				return []string{"--check", tmpFile}
			},
			parseErrors: parseNodeJSErrors,
		},
		// TypeScript (§9.3.3.3)
		{
			language:   "typescript",
			extensions: []string{".ts", ".tsx"},
			command:    "tsc",
			buildArgs: func(tmpFile string) []string {
				return []string{"--noEmit", "--strict", tmpFile}
			},
			parseErrors: parseTSCErrors,
		},
		// Python (§9.3.3.4)
		{
			language:   "python",
			extensions: []string{".py", ".pyw"},
			command:    "python",
			buildArgs: func(tmpFile string) []string {
				return []string{"-m", "py_compile", tmpFile}
			},
			parseErrors: parsePythonErrors,
		},
		// Java (§9.3.3.5)
		{
			language:   "java",
			extensions: []string{".java"},
			command:    "javac",
			buildArgs: func(tmpFile string) []string {
				// -Xstdout /dev/null 仅语法级检查，-d 指定临时输出目录
				if runtime.GOOS == "windows" {
					return []string{"-Xstdout", "NUL", "-d", os.TempDir(), tmpFile}
				}
				return []string{"-Xstdout", "/dev/null", "-d", os.TempDir(), tmpFile}
			},
			parseErrors: parseJavaErrors,
		},
		// Kotlin (§9.3.3.6)
		{
			language:   "kotlin",
			extensions: []string{".kt", ".kts"},
			command:    "kotlinc",
			buildArgs: func(tmpFile string) []string {
				return []string{"-nowarn", "-script", tmpFile}
			},
			parseErrors: parseKotlinErrors,
		},
		// Rust (§9.3.3.7)
		{
			language:   "rust",
			extensions: []string{".rs"},
			command:    "rustc",
			buildArgs: func(tmpFile string) []string {
				return []string{"--crate-type", "lib", "--edition", "2024", tmpFile}
			},
			parseErrors: parseRustErrors,
		},
		// Shell (§9.3.3.8)
		{
			language:   "shell",
			extensions: []string{".sh", ".bash"},
			command:    "bash",
			buildArgs: func(tmpFile string) []string {
				return []string{"-n", tmpFile}
			},
			parseErrors: parseShellErrors,
		},
		// PowerShell (§9.3.3.9)
		{
			language:   "powershell",
			extensions: []string{".ps1", ".psm1"},
			command:    "pwsh",
			buildArgs: func(tmpFile string) []string {
				// 使用-File参数避免路径注入，配合-NoProfile和-NoLogo减少输出
				return []string{"-NoProfile", "-NoLogo", "-NoExec", "-Command",
					"$errors = $null; $tokens = $null; [System.Management.Automation.Language.Parser]::ParseFile('" +
						strings.ReplaceAll(tmpFile, "'", "''") +
						"', [ref]$tokens, [ref]$errors); if ($errors) { foreach ($e in $errors) { Write-Error $e.ToString() } }"}
			},
			parseErrors: parsePSErrors,
		},
		// BAT (§9.3.3.10) - 仅做基础结构检查，不执行文件
		// cmd.exe没有仅检查语法的选项，此处通过type命令验证文件可读性
		// 非Windows平台cmd不可用，自动降级
		{
			language:   "bat",
			extensions: []string{".bat", ".cmd"},
			command:    "cmd",
			buildArgs: func(tmpFile string) []string {
				return []string{"/c", "type", tmpFile}
			},
			parseErrors: parseBATErrors,
		},
		// Gradle (§9.3.3.11) - 同时支持 .gradle 和 .gradle.kts
		{
			language:   "gradle",
			extensions: []string{".gradle", ".gradle.kts"},
			command:    "gradle",
			buildArgs: func(tmpFile string) []string {
				return []string{"--dry-run", "-b", tmpFile}
			},
			parseErrors: parseGradleErrors,
		},
	}

	// 建立扩展名映射
	for i := range c.tools {
		tool := &c.tools[i]
		for _, ext := range tool.extensions {
			c.extToTool[ext] = tool
		}
	}
}

// SupportedExtensions 返回所有支持的文件扩展名
func (c *ExternalChecker) SupportedExtensions() []string {
	exts := make([]string, 0, len(c.extToTool))
	for ext := range c.extToTool {
		exts = append(exts, ext)
	}
	return exts
}

// Check 对文件内容执行外部工具语法检查
func (c *ExternalChecker) Check(content string, filePath string) *SyntaxCheckResult {
	lowerPath := strings.ToLower(filePath)

	// 处理双扩展名（如 .gradle.kts）
	// filepath.Ext() 只返回最后一个扩展名（如 .kts），需要后缀匹配复合扩展名
	for ext, tool := range c.extToTool {
		if strings.Count(ext, ".") > 1 && strings.HasSuffix(lowerPath, ext) {
			return c.executeToolCheck(tool, content, ext)
		}
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	tool, ok := c.extToTool[ext]
	if !ok {
		return nil
	}

	return c.executeToolCheck(tool, content, ext)
}

// executeToolCheck 执行具体的外部工具检查
func (c *ExternalChecker) executeToolCheck(tool *externalToolDef, content string, ext string) *SyntaxCheckResult {
	// 检查工具可用性
	if !c.isToolAvailable(tool.command) {
		return newDegradedCheckResult(
			tool.language,
			fmt.Sprintf("external tool '%s' is not available", tool.command),
		)
	}

	// 创建临时文件
	tmpFile, err := c.createTempFile(content, ext)
	if err != nil {
		return newDegradedCheckResult(
			tool.language,
			fmt.Sprintf("failed to create temp file: %v", err),
		)
	}
	defer os.Remove(tmpFile)

	// 执行外部工具
	args := tool.buildArgs(tmpFile)
	stdout, stderr, exitCode, err := c.runCommand(tool.command, args)

	// 命令执行失败（非退出码问题）
	if err != nil && exitCode == -1 {
		return newDegradedCheckResult(
			tool.language,
			fmt.Sprintf("failed to execute '%s': %v", tool.command, err),
		)
	}

	// 退出码0表示语法正确
	if exitCode == 0 {
		return newSuccessCheckResult(tool.language, "external")
	}

	// 解析错误输出
	output := strings.TrimSpace(stderr)
	if output == "" {
		output = strings.TrimSpace(stdout)
	}

	errs := tool.parseErrors(output, tool.language)
	if len(errs) == 0 {
		// 无法解析出具体错误，返回原始输出
		errs = []SyntaxError{
			{
				Message:    truncateOutput(output, 500),
				Severity:   "error",
				Suggestion: fmt.Sprintf("Check %s syntax", tool.language),
			},
		}
	}

	return newErrorCheckResult(tool.language, "external", errs)
}

// isToolAvailable 检查外部工具是否可用（带缓存）
func (c *ExternalChecker) isToolAvailable(command string) bool {
	// 先尝试从缓存读取
	c.mu.RLock()
	available, cached := c.availabilityCache[command]
	c.mu.RUnlock()

	if cached {
		return available
	}

	// 首次检查，执行lookup
	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查（防止并发重复检测）
	if available, cached = c.availabilityCache[command]; cached {
		return available
	}

	// 执行实际检测
	_, err := exec.LookPath(command)
	available = err == nil
	c.availabilityCache[command] = available

	return available
}

// createTempFile 创建包含内容的临时文件
func (c *ExternalChecker) createTempFile(content string, ext string) (string, error) {
	tmpFile, err := os.CreateTemp("", "syntax_check_*"+ext)
	if err != nil {
		return "", err
	}

	_, err = tmpFile.WriteString(content)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}

	tmpFile.Close()
	return tmpFile.Name(), nil
}

// runCommand 执行外部命令并返回输出
func (c *ExternalChecker) runCommand(command string, args []string) (string, string, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), externalCheckTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	// Windows 下隐藏控制台窗口（避免闪窗）
	configureHideWindow(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			return "", "command timed out", -1, fmt.Errorf("command timed out after %s", externalCheckTimeout)
		} else {
			return "", err.Error(), -1, err
		}
	}

	return stdout.String(), stderr.String(), exitCode, nil
}

// truncateOutput 截断输出以避免过长的错误信息
func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ==================== 各语言错误解析器 ====================

// parseNodeJSErrors 解析Node.js语法错误输出
// 格式示例: "file.js:3:5 SyntaxError: Unexpected token '}'"
var nodeJSErrorPattern = regexp.MustCompile(`:(\d+):(\d+)\s+(.+)`)

func parseNodeJSErrors(stderr string, language string) []SyntaxError {
	var errs []SyntaxError
	lines := strings.Split(stderr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if matches := nodeJSErrorPattern.FindStringSubmatch(line); len(matches) >= 4 {
			var lineNum, colNum int
			fmt.Sscanf(matches[1], "%d", &lineNum)
			fmt.Sscanf(matches[2], "%d", &colNum)
			msg := strings.TrimSpace(matches[3])
			suggestion := ""
			if strings.Contains(strings.ToLower(msg), "unexpected") {
				suggestion = "Check for missing operators, brackets, or invalid syntax before this token"
			}
			errs = append(errs, SyntaxError{
				Line:       lineNum,
				Column:     colNum,
				Message:    msg,
				Severity:   "error",
				Suggestion: suggestion,
			})
		}
	}
	if len(errs) == 0 && stderr != "" {
		errs = append(errs, SyntaxError{
			Message:    truncateOutput(stderr, 500),
			Severity:   "error",
			Suggestion: "Check JavaScript syntax",
		})
	}
	return errs
}

// parseTSCErrors 解析TypeScript编译器错误输出
// 格式示例: "file.ts(3,5): error TS1005: ';' expected."
var tscErrorPattern = regexp.MustCompile(`\((\d+),(\d+)\):\s+(error|warning)\s+(TS\d+):\s+(.+)`)

func parseTSCErrors(stderr string, language string) []SyntaxError {
	var errs []SyntaxError
	lines := strings.Split(stderr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if matches := tscErrorPattern.FindStringSubmatch(line); len(matches) >= 6 {
			var lineNum, colNum int
			fmt.Sscanf(matches[1], "%d", &lineNum)
			fmt.Sscanf(matches[2], "%d", &colNum)
			errs = append(errs, SyntaxError{
				Line:       lineNum,
				Column:     colNum,
				Message:    fmt.Sprintf("%s: %s", matches[4], matches[5]),
				Severity:   "error",
				Suggestion: tscSuggestion(matches[4]),
			})
		}
	}
	if len(errs) == 0 && stderr != "" {
		errs = append(errs, SyntaxError{
			Message:    truncateOutput(stderr, 500),
			Severity:   "error",
			Suggestion: "Check TypeScript syntax",
		})
	}
	return errs
}

func tscSuggestion(code string) string {
	switch code {
	case "TS1005":
		return "Check for missing punctuation or keywords"
	case "TS1109":
		return "Expression expected, check for misplaced operators"
	case "TS1128":
		return "Declaration or statement expected"
	default:
		return ""
	}
}

// parsePythonErrors 解析Python语法错误输出
// 格式示例: '  File "file.py", line 3\n    print("hello\n          ^\nSyntaxError: EOL while scanning string literal'
var pythonErrorPattern = regexp.MustCompile(`File\s+"[^"]*",\s*line\s+(\d+)`)
var pythonErrorMsgPattern = regexp.MustCompile(`(SyntaxError|IndentationError|TabError):\s*(.+)`)

func parsePythonErrors(stderr string, language string) []SyntaxError {
	var errs []SyntaxError

	var lineNum int
	var errMsg string

	// 提取行号
	if matches := pythonErrorPattern.FindStringSubmatch(stderr); len(matches) >= 2 {
		fmt.Sscanf(matches[1], "%d", &lineNum)
	}

	// 提取错误消息
	if matches := pythonErrorMsgPattern.FindStringSubmatch(stderr); len(matches) >= 3 {
		errMsg = matches[2]
	}

	if lineNum > 0 || errMsg != "" {
		syntaxErr := SyntaxError{
			Line:     lineNum,
			Message:  errMsg,
			Severity: "error",
		}
		if errMsg == "" {
			syntaxErr.Message = truncateOutput(stderr, 500)
		}
		syntaxErr.Suggestion = pythonErrorSuggestion(errMsg)
		errs = append(errs, syntaxErr)
	}

	if len(errs) == 0 && stderr != "" {
		errs = append(errs, SyntaxError{
			Message:    truncateOutput(stderr, 500),
			Severity:   "error",
			Suggestion: "Check Python syntax",
		})
	}
	return errs
}

func pythonErrorSuggestion(msg string) string {
	msgLower := strings.ToLower(msg)
	switch {
	case strings.Contains(msgLower, "eol while scanning"):
		return "Check for unclosed string literals (missing closing quote)"
	case strings.Contains(msgLower, "unexpected eof"):
		return "Check for unclosed parentheses, brackets, or braces"
	case strings.Contains(msgLower, "invalid syntax"):
		return "Check for missing colons, incorrect indentation, or invalid tokens"
	case strings.Contains(msgLower, "indent"):
		return "Use consistent indentation (4 spaces per level, avoid mixing tabs and spaces)"
	default:
		return ""
	}
}

// parseJavaErrors 解析javac错误输出
// 格式示例: "file.java:3: error: ';' expected"
var javaErrorPattern = regexp.MustCompile(`\.java:(\d+):\s*(error|warning):\s*(.+)`)

func parseJavaErrors(stderr string, language string) []SyntaxError {
	var errs []SyntaxError
	lines := strings.Split(stderr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if matches := javaErrorPattern.FindStringSubmatch(line); len(matches) >= 4 {
			var lineNum int
			fmt.Sscanf(matches[1], "%d", &lineNum)
			severity := "error"
			if matches[2] == "warning" {
				severity = "warning"
			}
			errs = append(errs, SyntaxError{
				Line:       lineNum,
				Message:    strings.TrimSpace(matches[3]),
				Severity:   severity,
				Suggestion: javaErrorSuggestion(matches[3]),
			})
		}
	}
	if len(errs) == 0 && stderr != "" {
		errs = append(errs, SyntaxError{
			Message:    truncateOutput(stderr, 500),
			Severity:   "error",
			Suggestion: "Check Java syntax",
		})
	}
	return errs
}

func javaErrorSuggestion(msg string) string {
	msgLower := strings.ToLower(msg)
	switch {
	case strings.Contains(msgLower, "';' expected"):
		return "Add a semicolon at the end of the statement"
	case strings.Contains(msgLower, "class, interface, or enum expected"):
		return "Check for extra closing braces or code outside of class body"
	case strings.Contains(msgLower, "cannot find symbol"):
		return "Verify import statements and variable/method names"
	case strings.Contains(msgLower, "reached end of file"):
		return "Check for missing closing braces"
	default:
		return ""
	}
}

// parseKotlinErrors 解析kotlinc错误输出
// 格式示例: "e: file.kt: (3, 5): expecting ')'"
var kotlinErrorPattern = regexp.MustCompile(`[ew]:\s*\S+\.kts?:\s*\((\d+),\s*(\d+)\):\s*(.+)`)

func parseKotlinErrors(stderr string, language string) []SyntaxError {
	var errs []SyntaxError
	lines := strings.Split(stderr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if matches := kotlinErrorPattern.FindStringSubmatch(line); len(matches) >= 4 {
			var lineNum, colNum int
			fmt.Sscanf(matches[1], "%d", &lineNum)
			fmt.Sscanf(matches[2], "%d", &colNum)
			errs = append(errs, SyntaxError{
				Line:       lineNum,
				Column:     colNum,
				Message:    strings.TrimSpace(matches[3]),
				Severity:   "error",
				Suggestion: "Check Kotlin syntax for missing or misplaced tokens",
			})
		}
	}
	if len(errs) == 0 && stderr != "" {
		errs = append(errs, SyntaxError{
			Message:    truncateOutput(stderr, 500),
			Severity:   "error",
			Suggestion: "Check Kotlin syntax",
		})
	}
	return errs
}

// parseRustErrors 解析rustc错误输出
// 格式示例: "error: expected `;`, found `}`\n --> file.rs:3:5"
var rustErrorPattern = regexp.MustCompile(`error(?:\[\w+\])?:\s*(.+)\n\s*-->\s*\S+:(\d+):(\d+)`)

func parseRustErrors(stderr string, language string) []SyntaxError {
	var errs []SyntaxError

	// 尝试多行匹配
	if matches := rustErrorPattern.FindStringSubmatch(stderr); len(matches) >= 4 {
		var lineNum, colNum int
		fmt.Sscanf(matches[2], "%d", &lineNum)
		fmt.Sscanf(matches[3], "%d", &colNum)
		errs = append(errs, SyntaxError{
			Line:       lineNum,
			Column:     colNum,
			Message:    strings.TrimSpace(matches[1]),
			Severity:   "error",
			Suggestion: rustErrorSuggestion(matches[1]),
		})
	}

	if len(errs) == 0 && stderr != "" {
		errs = append(errs, SyntaxError{
			Message:    truncateOutput(stderr, 500),
			Severity:   "error",
			Suggestion: "Check Rust syntax",
		})
	}
	return errs
}

func rustErrorSuggestion(msg string) string {
	msgLower := strings.ToLower(msg)
	switch {
	case strings.Contains(msgLower, "expected"):
		return "Check for missing or incorrect tokens at the indicated position"
	case strings.Contains(msgLower, "mismatched types"):
		return "Verify type annotations and expressions"
	default:
		return ""
	}
}

// parseShellErrors 解析Shell语法错误输出
// 格式示例: "file.sh: line 3: syntax error near unexpected token `}'"
var shellErrorPattern = regexp.MustCompile(`line\s+(\d+):\s*(.+)`)

func parseShellErrors(stderr string, language string) []SyntaxError {
	var errs []SyntaxError
	lines := strings.Split(stderr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if matches := shellErrorPattern.FindStringSubmatch(line); len(matches) >= 3 {
			var lineNum int
			fmt.Sscanf(matches[1], "%d", &lineNum)
			msg := strings.TrimSpace(matches[2])
			errs = append(errs, SyntaxError{
				Line:       lineNum,
				Message:    msg,
				Severity:   "error",
				Suggestion: shellErrorSuggestion(msg),
			})
		}
	}
	if len(errs) == 0 && stderr != "" {
		errs = append(errs, SyntaxError{
			Message:    truncateOutput(stderr, 500),
			Severity:   "error",
			Suggestion: "Check shell script syntax",
		})
	}
	return errs
}

func shellErrorSuggestion(msg string) string {
	msgLower := strings.ToLower(msg)
	switch {
	case strings.Contains(msgLower, "unexpected token"):
		return "Check for missing operators, quotes, or incorrect syntax near the indicated token"
	case strings.Contains(msgLower, "unexpected end of file"):
		return "Check for unclosed if/fi, case/esac, do/done, or missing quotes"
	default:
		return ""
	}
}

// parsePSErrors 解析PowerShell错误输出
func parsePSErrors(stderr string, language string) []SyntaxError {
	if stderr == "" {
		return nil
	}
	return []SyntaxError{
		{
			Message:    truncateOutput(stderr, 500),
			Severity:   "error",
			Suggestion: "Check PowerShell syntax",
		},
	}
}

// parseBATErrors 解析BAT检查输出
// cmd /c type 只验证文件可读性，错误输出较为简单
func parseBATErrors(stderr string, language string) []SyntaxError {
	if stderr == "" {
		return nil
	}
	return []SyntaxError{
		{
			Message:    truncateOutput(stderr, 500),
			Severity:   "error",
			Suggestion: "Check batch file syntax",
		},
	}
}

// parseGradleErrors 解析Gradle错误输出
func parseGradleErrors(stderr string, language string) []SyntaxError {
	var errs []SyntaxError
	lines := strings.Split(stderr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Gradle的Build错误通常包含行号信息
		if strings.Contains(strings.ToLower(line), "error") {
			errs = append(errs, SyntaxError{
				Message:    truncateOutput(line, 300),
				Severity:   "error",
				Suggestion: "Check Gradle build script syntax",
			})
		}
	}
	if len(errs) == 0 && stderr != "" {
		errs = append(errs, SyntaxError{
			Message:    truncateOutput(stderr, 500),
			Severity:   "error",
			Suggestion: "Check Gradle build script syntax",
		})
	}
	return errs
}
