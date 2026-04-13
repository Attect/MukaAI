package context

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Indexer 索引器，协调文件扫描和关键词索引
// 线程安全，支持并发查询
type Indexer struct {
	workDir     string           // 工作目录（绝对路径）
	files       []FileEntry      // 所有已索引的文件条目
	keywordIdx  map[string][]int // 关键词 → 文件索引列表（倒排索引）
	symbolIdx   map[string][]int // 符号名 → 文件索引列表
	pathIdx     map[string]int   // 相对路径 → 文件索引（精确查找）
	mu          sync.RWMutex     // 保护并发访问
	ready       bool             // 索引是否构建完成
	scanned     bool             // 是否已完成扫描
	projectName string           // 项目名称（目录名）
}

// NewIndexer 创建新的索引器
// workDir: 工作目录路径
func NewIndexer(workDir string) *Indexer {
	absDir, _ := filepath.Abs(workDir)
	projectName := filepath.Base(absDir)
	return &Indexer{
		workDir:     absDir,
		projectName: projectName,
		keywordIdx:  make(map[string][]int),
		symbolIdx:   make(map[string][]int),
		pathIdx:     make(map[string]int),
	}
}

// Scan 扫描文件树并建立索引
// 返回索引的文件数量和错误
func (idx *Indexer) Scan() (int, error) {
	scanner := NewFileTreeScanner(idx.workDir)
	entries, err := scanner.Scan()
	if err != nil {
		return 0, fmt.Errorf("扫描文件树失败: %w", err)
	}

	// 为每个文件提取关键词和符号
	for i := range entries {
		content := readFileContentForIndexing(entries[i].AbsPath)
		if content != "" {
			entries[i].Keywords = ExtractKeywordsFromContent(content, entries[i].Language)
			entries[i].Symbols = ExtractSymbols(content, entries[i].Language)
		}
		// 从路径中补充关键词
		pathKeywords := ExtractKeywordsFromPath(entries[i].Path)
		entries[i].Keywords = mergeKeywords(entries[i].Keywords, pathKeywords)
	}

	// 加锁更新索引
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.files = entries
	idx.keywordIdx = buildKeywordIndex(entries)
	idx.symbolIdx = buildSymbolIndex(entries)
	idx.pathIdx = buildPathIndex(entries)
	idx.scanned = true
	idx.ready = true

	return len(entries), nil
}

// ScanAsync 异步扫描文件树并建立索引
// 在后台goroutine中执行，不阻塞调用者
// 返回一个channel，扫描完成后发送结果
func (idx *Indexer) ScanAsync() <-chan ScanResult {
	ch := make(chan ScanResult, 1)
	go func() {
		defer close(ch)
		count, err := idx.Scan()
		ch <- ScanResult{FileCount: count, Err: err}
	}()
	return ch
}

// ScanResult 扫描结果
type ScanResult struct {
	FileCount int   // 索引的文件数量
	Err       error // 错误信息
}

// Query 根据任务描述查询相关文件
// taskDesc: 任务描述文本
// topN: 返回的最大文件数量
func (idx *Indexer) Query(taskDesc string, topN int) []FileEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if !idx.ready || len(idx.files) == 0 {
		return nil
	}

	// 如果topN未指定或为负数，使用默认值10
	if topN <= 0 {
		topN = 10
	}

	return idx.queryLocked(taskDesc, topN)
}

// queryLocked 内部查询方法（调用者需持有读锁）
func (idx *Indexer) queryLocked(taskDesc string, topN int) []FileEntry {
	// 提取任务关键词
	taskKeywords := ExtractKeywordsFromTask(taskDesc)
	if len(taskKeywords) == 0 {
		// 无法提取关键词时，返回最近修改的文件
		return idx.getRecentFiles(topN)
	}

	// 计算每个文件的综合得分
	scores := make([]scoredFile, 0, len(idx.files))
	for i, file := range idx.files {
		score := idx.computeScore(file, i, taskKeywords, taskDesc)
		if score > 0 {
			scores = append(scores, scoredFile{FileEntry: file, score: score})
		}
	}

	// 按得分降序排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// 取TopN
	if len(scores) > topN {
		scores = scores[:topN]
	}

	// 提取文件条目
	result := make([]FileEntry, len(scores))
	for i, sf := range scores {
		result[i] = sf.FileEntry
	}
	return result
}

// computeScore 计算文件与任务的综合相关性得分
// 排序因子：关键词匹配(40%) + 最近修改时间(20%) + 路径相关性(20%) + 符号匹配(20%)
func (idx *Indexer) computeScore(file FileEntry, fileIdx int, taskKeywords []string, taskDesc string) float64 {
	// 1. 关键词匹配度（40%）
	keywordScore := idx.computeKeywordScore(file, fileIdx, taskKeywords)

	// 2. 文件最近修改时间（20%）
	timeScore := idx.computeTimeScore(file)

	// 3. 路径与任务的相关性（20%）
	pathScore := computePathRelevanceScore(file.Path, taskDesc)

	// 4. 符号匹配度（20%）
	symbolScore := idx.computeSymbolScore(file, fileIdx, taskKeywords)

	return keywordScore*0.4 + timeScore*0.2 + pathScore*0.2 + symbolScore*0.2
}

// computeKeywordScore 计算关键词匹配得分
func (idx *Indexer) computeKeywordScore(file FileEntry, fileIdx int, taskKeywords []string) float64 {
	// 构建文件的关键词集合（用于快速查找）
	fileKeywordSet := make(map[string]bool)
	for _, kw := range file.Keywords {
		fileKeywordSet[kw] = true
	}

	matchCount := 0
	exactMatchCount := 0
	for _, tk := range taskKeywords {
		if fileKeywordSet[tk] {
			exactMatchCount++
			matchCount++
		} else {
			// 前缀匹配（如任务关键词"agent"匹配"agentcore"）
			for fk := range fileKeywordSet {
				if len(tk) >= 3 && (strings.HasPrefix(fk, tk) || strings.HasPrefix(tk, fk)) {
					matchCount++
					break
				}
			}
		}
	}

	if matchCount == 0 {
		return 0
	}

	// 归一化：匹配比例
	ratio := float64(matchCount) / float64(len(taskKeywords))
	exactBonus := float64(exactMatchCount) / float64(len(taskKeywords))

	return 0.5*ratio + 0.5*exactBonus
}

// computeTimeScore 计算时间得分（最近修改的文件得分更高）
func (idx *Indexer) computeTimeScore(file FileEntry) float64 {
	if len(idx.files) == 0 {
		return 0
	}

	// 找到最新和最早的修改时间
	var newest, oldest time.Time
	for _, f := range idx.files {
		if newest.IsZero() || f.ModTime.After(newest) {
			newest = f.ModTime
		}
		if oldest.IsZero() || f.ModTime.Before(oldest) {
			oldest = f.ModTime
		}
	}

	if newest.Equal(oldest) {
		return 0.5
	}

	// 线性插值：最新=1.0，最早=0.0
	elapsed := file.ModTime.Sub(oldest).Seconds()
	total := newest.Sub(oldest).Seconds()
	if total == 0 {
		return 0.5
	}
	return elapsed / total
}

// computeSymbolScore 计算符号匹配得分
func (idx *Indexer) computeSymbolScore(file FileEntry, fileIdx int, taskKeywords []string) float64 {
	if len(file.Symbols) == 0 {
		return 0
	}

	matchCount := 0
	for _, tk := range taskKeywords {
		for _, sym := range file.Symbols {
			if strings.EqualFold(sym.Name, tk) {
				matchCount++
				break
			} else if len(tk) >= 3 && (strings.HasPrefix(strings.ToLower(sym.Name), tk) ||
				strings.HasPrefix(tk, strings.ToLower(sym.Name))) {
				matchCount++
				break
			}
		}
	}

	if matchCount == 0 {
		return 0
	}

	return float64(matchCount) / float64(len(taskKeywords))
}

// computePathRelevanceScore 计算路径与任务描述的相关性得分
func computePathRelevanceScore(path string, taskDesc string) float64 {
	taskLower := strings.ToLower(taskDesc)

	score := 0.0

	// 检查路径中的文件名是否直接出现在任务描述中
	fileName := filepath.Base(path)
	fileNameNoExt := fileName
	if idx := strings.LastIndex(fileName, "."); idx > 0 {
		fileNameNoExt = fileName[:idx]
	}

	if strings.Contains(taskLower, strings.ToLower(fileName)) {
		score += 0.5 // 文件名完全出现在任务描述中
	}
	if strings.Contains(taskLower, strings.ToLower(fileNameNoExt)) {
		score += 0.3 // 文件名（无扩展名）出现在任务描述中
	}

	// 检查路径中的目录名是否出现在任务描述中
	parts := strings.Split(path, "/")
	for _, part := range parts[:len(parts)-1] { // 排除文件名本身
		partNoExt := part
		if idx := strings.LastIndex(part, "."); idx > 0 {
			partNoExt = part[:idx]
		}
		if len(partNoExt) > 2 && strings.Contains(taskLower, strings.ToLower(partNoExt)) {
			score += 0.2
		}
	}

	if score > 1.0 {
		score = 1.0
	}
	return score
}

// getRecentFiles 返回最近修改的文件（作为无关键词匹配时的备选）
func (idx *Indexer) getRecentFiles(n int) []FileEntry {
	sorted := make([]FileEntry, len(idx.files))
	copy(sorted, idx.files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ModTime.After(sorted[j].ModTime)
	})
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}

// IsReady 检查索引是否已构建完成
func (idx *Indexer) IsReady() bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.ready
}

// GetFileCount 返回已索引的文件数量
func (idx *Indexer) GetFileCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.files)
}

// GetProjectName 返回项目名称
func (idx *Indexer) GetProjectName() string {
	return idx.projectName
}

// GetWorkDir 返回工作目录
func (idx *Indexer) GetWorkDir() string {
	return idx.workDir
}

// GetFileByPath 通过相对路径精确查找文件
func (idx *Indexer) GetFileByPath(relPath string) (FileEntry, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	i, ok := idx.pathIdx[relPath]
	if !ok {
		return FileEntry{}, false
	}
	return idx.files[i], true
}

// buildKeywordIndex 构建关键词倒排索引
func buildKeywordIndex(entries []FileEntry) map[string][]int {
	idx := make(map[string][]int)
	for i, entry := range entries {
		for _, kw := range entry.Keywords {
			idx[kw] = append(idx[kw], i)
		}
	}
	return idx
}

// buildSymbolIndex 构建符号索引
func buildSymbolIndex(entries []FileEntry) map[string][]int {
	idx := make(map[string][]int)
	for i, entry := range entries {
		for _, sym := range entry.Symbols {
			lower := strings.ToLower(sym.Name)
			idx[lower] = append(idx[lower], i)
		}
	}
	return idx
}

// buildPathIndex 构建路径索引
func buildPathIndex(entries []FileEntry) map[string]int {
	idx := make(map[string]int)
	for i, entry := range entries {
		idx[entry.Path] = i
	}
	return idx
}

// mergeKeywords 合并关键词列表（去重）
func mergeKeywords(existing []string, additional []string) []string {
	seen := make(map[string]bool)
	for _, kw := range existing {
		seen[kw] = true
	}
	for _, kw := range additional {
		if !seen[kw] {
			existing = append(existing, kw)
			seen[kw] = true
		}
	}
	return existing
}
