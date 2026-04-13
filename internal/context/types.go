// Package context 实现代码库自动索引和上下文感知系统
// 三层架构：索引层（文件树+关键词+符号）→ 查询层（匹配+排序）→ 注入层（组装+Token预算）
package context

import (
	"path/filepath"
	"strings"
	"time"
)

// FileEntry 文件条目，表示索引中的一个文件
type FileEntry struct {
	Path     string    // 相对于工作目录的路径（使用正斜杠）
	AbsPath  string    // 绝对路径
	Language string    // 编程语言（通过扩展名检测）
	Size     int64     // 文件大小（字节）
	ModTime  time.Time // 最后修改时间
	Keywords []string  // 从文件中提取的关键词
	Symbols  []Symbol  // 从代码中提取的符号
	Lines    int       // 文件行数
}

// Symbol 代码符号（函数、类、接口、类型定义等）
type Symbol struct {
	Name string // 符号名称
	Kind string // 类型：function, class, interface, type, variable, method
	Line int    // 定义所在行号（从1开始）
}

// scoredFile 内部使用：带分数的文件条目，用于排序
type scoredFile struct {
	FileEntry
	score float64 // 综合得分
}

// extToLang 文件扩展名到编程语言的映射
var extToLang = map[string]string{
	".go":    "go",
	".java":  "java",
	".kt":    "kotlin",
	".kts":   "kotlin",
	".py":    "python",
	".js":    "javascript",
	".ts":    "typescript",
	".tsx":   "typescript",
	".jsx":   "javascript",
	".rs":    "rust",
	".html":  "html",
	".css":   "css",
	".scss":  "scss",
	".less":  "less",
	".json":  "json",
	".yaml":  "yaml",
	".yml":   "yaml",
	".xml":   "xml",
	".sql":   "sql",
	".sh":    "shell",
	".bash":  "shell",
	".zsh":   "shell",
	".bat":   "batch",
	".ps1":   "powershell",
	".c":     "c",
	".cpp":   "cpp",
	".h":     "c",
	".hpp":   "cpp",
	".cs":    "csharp",
	".rb":    "ruby",
	".php":   "php",
	".swift": "swift",
	".dart":  "dart",
	".lua":   "lua",
	".r":     "r",
	".toml":  "toml",
	".ini":   "ini",
	".cfg":   "ini",
	".md":    "markdown",
	".txt":   "text",
}

// binaryExtensions 二进制文件扩展名集合，扫描时跳过这些文件
var binaryExtensions = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".bin": true, ".dat": true, ".db": true, ".sqlite": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
	".ico": true, ".webp": true, ".svg": true,
	".mp3": true, ".mp4": true, ".wav": true, ".avi": true, ".mkv": true,
	".zip": true, ".tar": true, ".gz": true, ".rar": true, ".7z": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true,
	".class": true, ".jar": true, ".war": true,
	".o": true, ".a": true, ".lib": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
	".wasm": true,
}

// ignoreDirs 扫描时忽略的目录名集合
var ignoreDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "__pycache__": true,
	".idea": true, ".vscode": true, "dist": true, "build": true,
	"target": true, "bin": true, "obj": true, ".gradle": true,
	".mvn": true, "logs": true, "state": true, "ai": true,
	".next": true, ".nuxt": true, ".cache": true, ".tmp": true,
	"coverage": true, ".coverage": true, "__tests__": false,
}

// detectLanguage 通过文件扩展名检测编程语言
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if lang, ok := extToLang[ext]; ok {
		return lang
	}
	return "unknown"
}

// isBinaryFile 判断文件是否为二进制文件（通过扩展名）
func isBinaryFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return binaryExtensions[ext]
}

// shouldIgnoreDir 判断目录是否应被忽略
func shouldIgnoreDir(name string) bool {
	return ignoreDirs[name]
}

// maxFileSize 单个文件的最大大小（1MB），超过此大小的文件不索引
const maxFileSize int64 = 1 * 1024 * 1024

// maxFileLines 单个文件内容截断的最大行数
const maxFileLines = 500

// truncateHeadLines 截断时保留的前部行数
const truncateHeadLines = 200

// truncateTailLines 截断时保留的尾部行数
const truncateTailLines = 100
