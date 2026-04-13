package context

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FileTreeScanner 文件树扫描器
// 扫描工作目录，构建文件条目列表
type FileTreeScanner struct {
	workDir string // 工作目录（绝对路径）
}

// NewFileTreeScanner 创建新的文件树扫描器
func NewFileTreeScanner(workDir string) *FileTreeScanner {
	absDir, _ := filepath.Abs(workDir)
	return &FileTreeScanner{workDir: absDir}
}

// Scan 扫描工作目录，返回文件条目列表
// 遵循忽略规则：忽略指定目录、二进制文件、超大文件
func (s *FileTreeScanner) Scan() ([]FileEntry, error) {
	var entries []FileEntry

	err := filepath.WalkDir(s.workDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// 无法访问的路径直接跳过，不中断整体扫描
			return nil
		}

		absPath, _ := filepath.Abs(path)

		// 跳过工作目录本身
		if absPath == s.workDir {
			return nil
		}

		// 目录处理：忽略指定目录
		if d.IsDir() {
			if shouldIgnoreDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}

		// 文件处理：跳过二进制文件
		if isBinaryFile(d.Name()) {
			return nil
		}

		// 获取文件信息
		info, err := d.Info()
		if err != nil {
			return nil
		}

		// 跳过超大文件
		if info.Size() > maxFileSize {
			return nil
		}

		// 计算相对路径（统一使用正斜杠）
		relPath, err := filepath.Rel(s.workDir, absPath)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		// 检测编程语言
		lang := detectLanguage(absPath)
		if lang == "unknown" {
			// 对于未知语言的文件，只索引常见的文本文件
			ext := strings.ToLower(filepath.Ext(absPath))
			if ext != ".txt" && ext != ".md" && ext != "" {
				// 跳过无法识别的文件类型（避免索引.lock、.sum等杂项文件）
				if ext == ".lock" || ext == ".sum" || ext == ".mod" || ext == ".map" {
					return nil
				}
			}
		}

		// 统计行数（只对文本文件统计）
		lines := countLines(absPath)

		entry := FileEntry{
			Path:     relPath,
			AbsPath:  absPath,
			Language: lang,
			Size:     info.Size(),
			ModTime:  info.ModTime(),
			Lines:    lines,
		}

		entries = append(entries, entry)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("扫描文件树失败: %w", err)
	}

	return entries, nil
}

// ReadFileContent 读取文件内容
// 返回原始内容和截断后的内容
func ReadFileContent(absPath string) (full string, truncated string, err error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", "", fmt.Errorf("读取文件失败: %w", err)
	}

	full = string(data)

	// 统计行数
	lines := strings.Split(full, "\n")
	if len(lines) <= maxFileLines {
		return full, full, nil
	}

	// 截断策略：保留前200行 + ... truncated ... + 后100行
	head := lines[:truncateHeadLines]
	tail := lines[len(lines)-truncateTailLines:]
	truncated = strings.Join(head, "\n") +
		"\n... truncated (" + fmt.Sprintf("%d lines omitted", len(lines)-truncateHeadLines-truncateTailLines) + ") ...\n" +
		strings.Join(tail, "\n")

	return full, truncated, nil
}

// countLines 快速统计文件行数
func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	// 设置较大的缓冲区以支持长行
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		count++
	}
	return count
}
