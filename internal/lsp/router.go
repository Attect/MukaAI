package lsp

import (
	"path/filepath"
	"strings"
)

// extToLanguage 文件扩展名到语言标识符的映射
var extToLanguage = map[string]string{
	".go":  "go",
	".ts":  "typescript",
	".tsx": "typescript",
	".js":  "typescript",
	".jsx": "typescript",
	".py":  "python",
	".pyw": "python",
	".pyi": "python",
}

// defaultLanguageServers 默认语言服务器配置
var defaultLanguageServers = map[string]LanguageServerConfig{
	"go": {
		Language: "go",
		Command:  "gopls",
		Args:     []string{},
	},
	"typescript": {
		Language: "typescript",
		Command:  "typescript-language-server",
		Args:     []string{"--stdio"},
	},
	"python": {
		Language: "python",
		Command:  "pylsp",
		Args:     []string{},
	},
}

// LanguageFromPath 根据文件路径推断语言类型
// 通过文件扩展名匹配，返回语言标识符（如"go"、"typescript"、"python"）
// 无法识别的扩展名返回空字符串
func LanguageFromPath(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	if lang, ok := extToLanguage[ext]; ok {
		return lang
	}
	return ""
}

// ServerForLanguage 根据语言标识符获取对应的服务器配置
// 返回配置和是否找到
func ServerForLanguage(language string) (LanguageServerConfig, bool) {
	cfg, ok := defaultLanguageServers[language]
	return cfg, ok
}

// SupportedExtensions 返回所有支持的文件扩展名列表
func SupportedExtensions() []string {
	exts := make([]string, 0, len(extToLanguage))
	for ext := range extToLanguage {
		exts = append(exts, ext)
	}
	return exts
}

// SupportedLanguages 返回所有支持的语言列表
func SupportedLanguages() []string {
	seen := make(map[string]bool)
	langs := make([]string, 0)
	for _, lang := range extToLanguage {
		if !seen[lang] {
			seen[lang] = true
			langs = append(langs, lang)
		}
	}
	return langs
}

// IsSupportedFile 判断文件是否受LSP支持
func IsSupportedFile(filePath string) bool {
	return LanguageFromPath(filePath) != ""
}

// LanguageDisplayName 返回语言的可读名称
func LanguageDisplayName(language string) string {
	switch language {
	case "go":
		return "Go"
	case "typescript":
		return "TypeScript/JavaScript"
	case "python":
		return "Python"
	default:
		return language
	}
}

// DefaultConfigs 返回默认语言服务器配置（用于初始化Manager）
func DefaultConfigs() map[string]LanguageServerConfig {
	configs := make(map[string]LanguageServerConfig)
	for k, v := range defaultLanguageServers {
		configs[k] = v
	}
	return configs
}
