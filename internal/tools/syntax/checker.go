package syntax

import (
	"path/filepath"
	"strings"
	"sync"
)

// SyntaxChecker 语法检查器接口
// 所有语言检查器必须实现此接口
type SyntaxChecker interface {
	// SupportedExtensions 返回支持的文件扩展名列表（含.号，如 ".json"）
	SupportedExtensions() []string

	// Check 对内容进行语法检查
	// content: 文件内容
	// filePath: 文件路径（用于错误提示和确定语言）
	Check(content string, filePath string) *SyntaxCheckResult
}

// Dispatcher 语法检查调度器
// 根据文件扩展名选择合适的检查器并执行检查
type Dispatcher struct {
	// checkers 扩展名→检查器映射
	checkers map[string]SyntaxChecker

	// mu 保护checkers map的读写锁
	mu sync.RWMutex

	// maxFileSize 文件大小上限（字节），超过此大小跳过检查
	maxFileSize int64
}

// NewDispatcher 创建新的语法检查调度器
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		checkers:    make(map[string]SyntaxChecker),
		maxFileSize: 10 * 1024 * 1024, // 10MB
	}
}

// RegisterChecker 注册检查器
// 检查器通过SupportedExtensions()声明支持的扩展名，调度器建立映射
func (d *Dispatcher) RegisterChecker(checker SyntaxChecker) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, ext := range checker.SupportedExtensions() {
		d.checkers[strings.ToLower(ext)] = checker
	}
}

// Check 对文件内容执行语法检查
// 根据文件扩展名选择合适的检查器
// content: 文件内容
// filePath: 文件路径，用于确定语言和错误提示
// fileSize: 文件大小（字节），超过限制时跳过
func (d *Dispatcher) Check(content string, filePath string, fileSize int64) *SyntaxCheckResult {
	// 文件大小检查（读锁保护）
	d.mu.RLock()
	maxSize := d.maxFileSize
	d.mu.RUnlock()
	if fileSize > maxSize {
		return newDegradedCheckResult(
			detectLanguage(filePath),
			"file size exceeds 10MB limit, syntax check skipped",
		)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		// 无扩展名，跳过检查
		return nil
	}

	d.mu.RLock()
	checker, ok := d.checkers[ext]
	d.mu.RUnlock()

	if !ok {
		// 未知文件类型，跳过检查（返回nil表示不包含syntax_check字段）
		return nil
	}

	// 使用recover捕获检查器可能的panic，防止工具崩溃
	var result *SyntaxCheckResult
	func() {
		defer func() {
			if r := recover(); r != nil {
				// 检查器panic，返回降级结果
				result = newDegradedCheckResult(
					detectLanguage(filePath),
					"syntax checker panicked during execution",
				)
			}
		}()
		result = checker.Check(content, filePath)
	}()

	return result
}

// MaxFileSize 返回文件大小限制
func (d *Dispatcher) MaxFileSize() int64 {
	return d.maxFileSize
}

// SetMaxFileSize 设置文件大小限制
func (d *Dispatcher) SetMaxFileSize(size int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.maxFileSize = size
}

// RegisteredExtensions 返回所有已注册的文件扩展名
func (d *Dispatcher) RegisteredExtensions() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	exts := make([]string, 0, len(d.checkers))
	for ext := range d.checkers {
		exts = append(exts, ext)
	}
	return exts
}

// detectLanguage 根据文件扩展名检测语言名称
func detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".xml", ".xsd", ".xsl", ".xslt", ".svg", ".pom":
		return "xml"
	case ".html", ".htm":
		return "html"
	case ".go":
		return "go"
	case ".js", ".mjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py", ".pyw":
		return "python"
	case ".java":
		return "java"
	case ".kt", ".kts":
		// .gradle.kts 应识别为 gradle 而非 kotlin
		if strings.HasSuffix(strings.ToLower(filePath), ".gradle.kts") {
			return "gradle"
		}
		return "kotlin"
	case ".rs":
		return "rust"
	case ".sh", ".bash":
		return "shell"
	case ".ps1", ".psm1":
		return "powershell"
	case ".bat", ".cmd":
		return "batch"
	case ".gradle":
		return "gradle"
	case ".toml":
		return "toml"
	case ".css":
		return "css"
	case ".sql":
		return "sql"
	case ".properties":
		return "properties"
	default:
		return strings.TrimPrefix(ext, ".")
	}
}
