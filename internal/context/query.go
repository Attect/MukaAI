package context

import (
	"strings"
	"unicode/utf8"
)

// QueryEngine 查询和排序引擎
// 封装查询逻辑，提供更灵活的查询能力
type QueryEngine struct {
	indexer *Indexer
}

// NewQueryEngine 创建新的查询引擎
func NewQueryEngine(indexer *Indexer) *QueryEngine {
	return &QueryEngine{indexer: indexer}
}

// Query 执行查询，返回相关文件
func (qe *QueryEngine) Query(taskDesc string, topN int) []FileEntry {
	return qe.indexer.Query(taskDesc, topN)
}

// QueryWithFileHints 带文件提示的查询
// 当任务描述中明确提到文件路径时，优先返回这些文件
func (qe *QueryEngine) QueryWithFileHints(taskDesc string, topN int) []FileEntry {
	if !qe.indexer.IsReady() {
		return nil
	}

	// 先提取任务中可能出现的文件路径
	hintedFiles := qe.extractFileHints(taskDesc)

	// 正常查询
	normalResults := qe.indexer.Query(taskDesc, topN)

	// 合并：提示文件优先，避免重复
	result := make([]FileEntry, 0, topN)
	seen := make(map[string]bool)

	// 先添加提示文件
	for _, f := range hintedFiles {
		if !seen[f.Path] {
			result = append(result, f)
			seen[f.Path] = true
		}
	}

	// 再添加正常查询结果
	for _, f := range normalResults {
		if !seen[f.Path] {
			result = append(result, f)
			seen[f.Path] = true
		}
	}

	if len(result) > topN {
		result = result[:topN]
	}

	return result
}

// extractFileHints 从任务描述中提取可能的文件路径引用
func (qe *QueryEngine) extractFileHints(taskDesc string) []FileEntry {
	var results []FileEntry
	words := strings.FieldsFunc(taskDesc, func(r rune) bool {
		return r == ' ' || r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t'
	})

	for _, word := range words {
		// 检查是否像文件路径（包含/或\或.且有扩展名）
		if strings.Contains(word, ".") || strings.Contains(word, "/") || strings.Contains(word, "\\") {
			// 清理路径
			cleanPath := strings.Trim(word, "`'\"()[]{}")
			cleanPath = strings.ReplaceAll(cleanPath, "\\", "/")

			// 尝试直接匹配
			if f, ok := qe.indexer.GetFileByPath(cleanPath); ok {
				results = append(results, f)
				continue
			}

			// 尝试匹配文件名
			for _, f := range qe.queryByFileName(cleanPath) {
				results = append(results, f)
			}
		}
	}

	return results
}

// queryByFileName 通过文件名（不含路径）查找文件
func (qe *QueryEngine) queryByFileName(name string) []FileEntry {
	// 去除可能的路径前缀
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if idx := strings.LastIndex(name, "\\"); idx >= 0 {
		name = name[idx+1:]
	}

	var results []FileEntry
	for _, f := range qe.indexer.Query(name, 20) {
		fileName := f.Path
		if idx := strings.LastIndex(fileName, "/"); idx >= 0 {
			fileName = fileName[idx+1:]
		}
		if strings.EqualFold(fileName, name) {
			results = append(results, f)
		}
	}
	return results
}

// EstimateTokenCount 估算文本的token数量
// 使用UTF-8字符数进行粗略估算：平均每4个字符约1个token
func EstimateTokenCount(text string) int {
	return utf8.RuneCountInString(text) / 4
}

// TaskKeywords 获取任务描述中提取的关键词（用于调试）
func TaskKeywords(taskDesc string) []string {
	return ExtractKeywordsFromTask(taskDesc)
}
