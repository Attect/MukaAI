package context

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"unicode"
)

// 停用词集合：常见英文介词、连词、代词等，从关键词中排除
var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "is": true, "are": true, "was": true,
	"were": true, "be": true, "been": true, "being": true, "have": true,
	"has": true, "had": true, "do": true, "does": true, "did": true,
	"will": true, "would": true, "could": true, "should": true,
	"may": true, "might": true, "can": true, "shall": true,
	"to": true, "of": true, "in": true, "for": true, "on": true,
	"with": true, "at": true, "by": true, "from": true, "as": true,
	"into": true, "through": true, "during": true, "before": true,
	"after": true, "above": true, "below": true, "between": true,
	"out": true, "off": true, "over": true, "under": true, "again": true,
	"further": true, "then": true, "once": true, "here": true,
	"there": true, "when": true, "where": true, "why": true, "how": true,
	"all": true, "each": true, "every": true, "both": true, "few": true,
	"more": true, "most": true, "other": true, "some": true, "such": true,
	"no": true, "not": true, "only": true, "own": true, "same": true,
	"so": true, "than": true, "too": true, "very": true,
	"just": true, "because": true, "but": true, "and": true, "or": true,
	"if": true, "while": true, "this": true, "that": true, "these": true,
	"those": true, "it": true, "its": true, "i": true, "me": true,
	"my": true, "we": true, "our": true, "you": true, "your": true,
	"he": true, "she": true, "they": true, "them": true, "what": true,
	"which": true, "who": true, "whom": true,
	// 中文停用词
	"的": true, "了": true, "在": true, "是": true, "我": true,
	"有": true, "和": true, "就": true, "不": true, "人": true,
	"都": true, "一": true, "一个": true, "上": true, "也": true,
	"很": true, "到": true, "说": true, "要": true, "去": true,
	"你": true, "会": true, "着": true, "没有": true, "看": true,
	"好": true, "自己": true, "这": true, "他": true, "她": true,
	"那": true, "个": true, "们": true, "把": true, "被": true,
	"让": true, "给": true, "对": true, "而": true, "但": true,
	"或": true, "与": true, "及": true, "等": true, "中": true,
	"它": true, "其": true, "如": true, "如果": true, "因为": true,
	"所以": true, "然后": true, "可以": true, "需要": true, "进行": true,
}

// 正则表达式：用于从不同语言的代码中提取标识符和符号

// Go语言的符号模式
var goPatterns = []*regexp.Regexp{
	regexp.MustCompile(`func\s+(?:\([^)]+\)\s+)?(\w+)`),            // func Name / func (r Type) Name
	regexp.MustCompile(`type\s+(\w+)\s+`),                          // type Name struct/interface
	regexp.MustCompile(`var\s+(\w+)\s+`),                           // var Name
	regexp.MustCompile(`const\s+(\w+)\s+`),                         // const Name
	regexp.MustCompile(`import\s+(?:"([^"]+)"|(\w+)\s+"([^"]+)")`), // import "pkg" / import alias "pkg"
}

// Java/Kotlin的符号模式
var javaPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?:public|private|protected)?\s*(?:static\s+)?(?:class|interface|enum)\s+(\w+)`),
	regexp.MustCompile(`(?:public|private|protected)?\s*(?:static\s+)?(?:\w+(?:<[^>]+>)?)\s+(\w+)\s*\(`),
	regexp.MustCompile(`import\s+(?:static\s+)?([\w.]+)`),
}

// TypeScript/JavaScript的符号模式
var tsPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?:export\s+)?(?:default\s+)?(?:function|const|let|var|class|interface|type|enum)\s+(\w+)`),
	regexp.MustCompile(`import\s+(?:\{([^}]+)\}|(\w+))\s+from`), // import { A, B } from / import X from
}

// Python的符号模式
var pythonPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?:def|class)\s+(\w+)`),
	regexp.MustCompile(`import\s+([\w.]+)`),
	regexp.MustCompile(`from\s+([\w.]+)\s+import`),
}

// Rust的符号模式
var rustPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?:pub\s+)?(?:fn|struct|enum|trait|impl|type)\s+(\w+)`),
	regexp.MustCompile(`use\s+([\w:]+)`),
}

// identifierPattern 通用标识符匹配（用于从文件名和路径中提取）
var identifierPattern = regexp.MustCompile(`[a-zA-Z_]\w*`)

// camelCaseSplit 驼峰命名分割（camelCase -> camel, case）
var camelCaseSplit = regexp.MustCompile(`([a-z])([A-Z])`)

// ExtractKeywordsFromFilename 从文件名中提取关键词
// 支持驼峰、蛇形、短横线命名风格
func ExtractKeywordsFromFilename(filename string) []string {
	// 去掉扩展名
	name := filename
	if idx := strings.LastIndex(filename, "."); idx > 0 {
		name = filename[:idx]
	}
	if name == "" {
		return nil
	}

	// 短横线分割
	parts := strings.Split(name, "-")
	// 蛇形分割
	var expanded []string
	for _, part := range parts {
		subParts := strings.Split(part, "_")
		expanded = append(expanded, subParts...)
	}

	// 驼峰分割 + 转小写
	var keywords []string
	for _, part := range expanded {
		if part == "" {
			continue
		}
		// 在小写字母和大写字母之间插入空格，然后分割
		split := camelCaseSplit.ReplaceAllString(part, "$1 $2")
		for _, word := range strings.Split(split, " ") {
			word = strings.ToLower(strings.TrimSpace(word))
			if word != "" && !stopWords[word] && len(word) > 1 {
				keywords = append(keywords, word)
			}
		}
	}

	return keywords
}

// ExtractKeywordsFromPath 从文件路径中提取关键词
// 包括目录名和文件名中的关键词
func ExtractKeywordsFromPath(relPath string) []string {
	parts := strings.Split(relPath, "/")
	var keywords []string
	for _, part := range parts {
		kws := ExtractKeywordsFromFilename(part)
		keywords = append(keywords, kws...)
	}
	return keywords
}

// ExtractKeywordsFromContent 从代码内容中提取关键词
// 基于正则表达式提取标识符，过滤停用词
func ExtractKeywordsFromContent(content string, lang string) []string {
	var patterns []*regexp.Regexp
	switch lang {
	case "go":
		patterns = goPatterns
	case "java", "kotlin":
		patterns = javaPatterns
	case "javascript", "typescript":
		patterns = tsPatterns
	case "python":
		patterns = pythonPatterns
	case "rust":
		patterns = rustPatterns
	default:
		// 对于未知语言，使用通用标识符提取
		patterns = nil
	}

	keywordSet := make(map[string]bool)

	// 使用语言特定模式提取
	for _, pat := range patterns {
		matches := pat.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			for i := 1; i < len(match); i++ {
				if match[i] != "" {
					extracted := extractIdentifiers(match[i])
					for _, id := range extracted {
						id = strings.ToLower(id)
						if !stopWords[id] && len(id) > 1 {
							keywordSet[id] = true
						}
					}
				}
			}
		}
	}

	// 从注释中提取关键词（// 和 /* */ 和 #）
	extractCommentKeywords(content, keywordSet)

	// 转为切片
	keywords := make([]string, 0, len(keywordSet))
	for kw := range keywordSet {
		keywords = append(keywords, kw)
	}
	return keywords
}

// extractIdentifiers 从字符串中提取标识符
// 处理复合标识符（如包路径 a.b.c -> a, b, c）
func extractIdentifiers(s string) []string {
	s = strings.TrimSpace(s)
	// 处理包路径分隔符
	s = strings.ReplaceAll(s, ".", " ")
	s = strings.ReplaceAll(s, ":", " ")
	s = strings.ReplaceAll(s, "/", " ")
	s = strings.ReplaceAll(s, ",", " ")
	s = strings.ReplaceAll(s, "{", " ")
	s = strings.ReplaceAll(s, "}", " ")

	return identifierPattern.FindAllString(s, -1)
}

// commentPatterns 注释匹配的正则
var (
	singleLineComment = regexp.MustCompile(`//\s*(.+)`)
	blockComment      = regexp.MustCompile(`/\*\s*(.*?)\s*\*/`)
	pythonComment     = regexp.MustCompile(`#\s*(.+)`)
)

// extractCommentKeywords 从注释中提取自然语言关键词
func extractCommentKeywords(content string, keywordSet map[string]bool) {
	// 单行注释 //
	matches := singleLineComment.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		addWordsFromText(m[1], keywordSet)
	}

	// 块注释 /* */
	matches = blockComment.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		addWordsFromText(m[1], keywordSet)
	}

	// Python风格注释 #
	matches = pythonComment.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		addWordsFromText(m[1], keywordSet)
	}
}

// addWordsFromText 从文本中提取有意义的词并加入集合
func addWordsFromText(text string, keywordSet map[string]bool) {
	// 按空白分割
	words := strings.Fields(text)
	for _, word := range words {
		// 去除标点
		word = strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		word = strings.ToLower(word)
		if len(word) > 2 && !stopWords[word] {
			keywordSet[word] = true
		}
	}
}

// ExtractSymbols 从代码内容中提取符号定义
// 返回符号列表（函数名、类名、接口名等）
func ExtractSymbols(content string, lang string) []Symbol {
	var symbols []Symbol
	var patterns []*regexp.Regexp
	var kindMap []string // 对应的符号类型

	switch lang {
	case "go":
		// Go: func, type
		pat := regexp.MustCompile(`func\s+(?:\([^)]+\)\s+)?(\w+)`)
		for _, m := range pat.FindAllStringSubmatchIndex(content, -1) {
			name := content[m[2]:m[3]]
			line := strings.Count(content[:m[0]], "\n") + 1
			symbols = append(symbols, Symbol{Name: name, Kind: "function", Line: line})
		}
		typePat := regexp.MustCompile(`type\s+(\w+)\s+(struct|interface|func\s+|map\[|[a-z])`)
		for _, m := range typePat.FindAllStringSubmatchIndex(content, -1) {
			name := content[m[2]:m[3]]
			line := strings.Count(content[:m[0]], "\n") + 1
			kind := "type"
			typeSuffix := content[m[4]:m[5]]
			if typeSuffix == "struct" || typeSuffix == "interface" {
				kind = typeSuffix
			}
			symbols = append(symbols, Symbol{Name: name, Kind: kind, Line: line})
		}
		return symbols
	case "java", "kotlin":
		patterns = javaPatterns[:1] // class/interface/enum
		kindMap = []string{"class"}
	case "javascript", "typescript":
		patterns = tsPatterns[:1]
		kindMap = []string{"function"}
	case "python":
		pat := regexp.MustCompile(`(def|class)\s+(\w+)`)
		for _, m := range pat.FindAllStringSubmatchIndex(content, -1) {
			kind := "function"
			if content[m[2]:m[3]] == "class" {
				kind = "class"
			}
			name := content[m[4]:m[5]]
			line := strings.Count(content[:m[0]], "\n") + 1
			symbols = append(symbols, Symbol{Name: name, Kind: kind, Line: line})
		}
		return symbols
	case "rust":
		patterns = rustPatterns[:1]
		kindMap = []string{"function"}
	}

	for i, pat := range patterns {
		for _, m := range pat.FindAllStringSubmatchIndex(content, -1) {
			if len(m) >= 4 {
				name := content[m[2]:m[3]]
				line := strings.Count(content[:m[0]], "\n") + 1
				kind := "function"
				if i < len(kindMap) {
					kind = kindMap[i]
				}
				symbols = append(symbols, Symbol{Name: name, Kind: kind, Line: line})
			}
		}
	}

	return symbols
}

// ExtractKeywordsFromTask 从任务描述中提取查询关键词
// 分词后去除停用词，保留有意义的词
func ExtractKeywordsFromTask(taskDesc string) []string {
	// 按空白和标点分割
	words := strings.FieldsFunc(taskDesc, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	})

	keywordSet := make(map[string]bool)
	for _, word := range words {
		word = strings.ToLower(strings.TrimSpace(word))
		if len(word) > 1 && !stopWords[word] {
			keywordSet[word] = true
		}

		// 对驼峰命名进行分割
		split := camelCaseSplit.ReplaceAllString(word, "$1 $2")
		for _, part := range strings.Split(split, " ") {
			part = strings.ToLower(strings.TrimSpace(part))
			if len(part) > 1 && !stopWords[part] {
				keywordSet[part] = true
			}
		}
	}

	keywords := make([]string, 0, len(keywordSet))
	for kw := range keywordSet {
		keywords = append(keywords, kw)
	}
	return keywords
}

// readFileContentForIndexing 读取文件内容用于索引构建
// 限制读取前1000行以提高性能
func readFileContentForIndexing(absPath string) string {
	f, err := os.Open(absPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineCount := 0
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		lineCount++
		if lineCount >= 1000 {
			break
		}
	}

	return strings.Join(lines, "\n")
}
