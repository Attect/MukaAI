package git

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Attect/MukaAI/internal/tools"
)

// ==================== git_diff 工具 ====================

// DiffFile 单个文件的diff信息
type DiffFile struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Diff      string `json:"diff"`
}

// GitDiffResult git_diff工具的返回数据结构
type GitDiffResult struct {
	Files   []DiffFile `json:"files"`
	Summary string     `json:"summary"`
}

// gitDiffTool git_diff工具实现
type gitDiffTool struct {
	workDir string
}

// NewGitDiffTool 创建git_diff工具
func NewGitDiffTool(workDir string) *gitDiffTool {
	return &gitDiffTool{workDir: workDir}
}

func (t *gitDiffTool) Name() string {
	return "git_diff"
}

func (t *gitDiffTool) Description() string {
	return "查看Git文件的变更内容（diff）。支持查看工作区变更或暂存区变更，可指定单个文件路径。返回每个变更文件的路径、状态、新增/删除行数以及完整的unified diff内容。"
}

func (t *gitDiffTool) Parameters() map[string]interface{} {
	return tools.BuildSchema(map[string]*tools.ToolParameter{
		"path": {
			Type:        "string",
			Description: "要查看变更的文件路径，不指定则显示所有变更文件",
		},
		"staged": {
			Type:        "boolean",
			Description: "是否查看暂存区的变更（git diff --staged），默认false查看工作区变更",
		},
	}, nil)
}

func (t *gitDiffTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	// 检查是否在Git仓库中
	if !isGitRepo(ctx, t.workDir) {
		return gitNotRepoError(), nil
	}

	// 解析参数
	staged := false
	if stagedVal, ok := params["staged"]; ok {
		if b, ok := stagedVal.(bool); ok {
			staged = b
		}
	}

	path, _ := params["path"].(string)

	// 构建git diff命令参数
	args := []string{"diff"}
	if staged {
		args = append(args, "--staged")
	}
	// 添加统计信息参数
	args = append(args, "--numstat")
	if path != "" {
		args = append(args, "--", path)
	}

	// 先获取numstat用于统计
	numstatStdout, stderr, exitCode, err := runGitCommand(ctx, t.workDir, args...)
	if err != nil || exitCode != 0 {
		// 如果没有差异，返回空结果
		if strings.Contains(stderr, "no changes") || exitCode == 0 {
			return tools.NewSuccessResult(&GitDiffResult{
				Files:   []DiffFile{},
				Summary: "no changes",
			}), nil
		}
		return tools.NewErrorResult(fmt.Sprintf("执行git diff失败: %s", stderr)), nil
	}

	// 解析numstat获取每个文件的统计信息
	fileStats := parseNumstat(numstatStdout)

	// 获取完整diff内容
	diffArgs := []string{"diff"}
	if staged {
		diffArgs = append(diffArgs, "--staged")
	}
	if path != "" {
		diffArgs = append(diffArgs, "--", path)
	}

	diffStdout, _, diffExitCode, diffErr := runGitCommand(ctx, t.workDir, diffArgs...)
	if diffErr != nil && diffExitCode != 0 {
		return tools.NewErrorResult(fmt.Sprintf("执行git diff失败: %s", diffStdout)), nil
	}

	// 解析diff内容，按文件分割
	files := parseDiffOutput(diffStdout, fileStats)

	// 构建摘要
	totalAdd, totalDel := 0, 0
	for _, f := range files {
		totalAdd += f.Additions
		totalDel += f.Deletions
	}
	summary := fmt.Sprintf("%d file(s) changed, %d insertions(+), %d deletions(-)",
		len(files), totalAdd, totalDel)

	return tools.NewSuccessResult(&GitDiffResult{
		Files:   files,
		Summary: summary,
	}), nil
}

// numstatEntry numstat解析结果
type numstatEntry struct {
	additions int
	deletions int
	path      string
}

// parseNumstat 解析git diff --numstat输出
// 格式: <additions>\t<deletions>\t<path>
func parseNumstat(output string) map[string]*numstatEntry {
	stats := make(map[string]*numstatEntry)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		additions := 0
		deletions := 0
		// 二进制文件显示为 "-"
		if parts[0] != "-" {
			fmt.Sscanf(parts[0], "%d", &additions)
		}
		if parts[1] != "-" {
			fmt.Sscanf(parts[1], "%d", &deletions)
		}
		filePath := parts[2]
		// 处理重命名: old => new
		if idx := strings.Index(filePath, " => "); idx >= 0 {
			filePath = filePath[idx+4:] // 取新路径
		}
		// 去除花括号包围的重命名格式
		if strings.HasPrefix(filePath, "{") {
			if idx := strings.LastIndex(filePath, "}"); idx >= 0 {
				filePath = filePath[idx+1:]
			}
		}
		stats[filePath] = &numstatEntry{
			additions: additions,
			deletions: deletions,
			path:      filePath,
		}
	}
	return stats
}

// parseDiffOutput 解析完整diff输出，按文件拆分
func parseDiffOutput(output string, fileStats map[string]*numstatEntry) []DiffFile {
	var files []DiffFile

	if strings.TrimSpace(output) == "" {
		return files
	}

	// 按diff文件头分割
	// 匹配 diff --git a/path b/path 格式
	fileHeaderRe := regexp.MustCompile(`^diff --git a/(.+?) b/(.+?)$`)

	var currentFile *DiffFile
	var currentDiffLines []string

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := fileHeaderRe.FindStringSubmatch(line)
		if len(matches) >= 3 {
			// 保存前一个文件
			if currentFile != nil {
				currentFile.Diff = strings.Join(currentDiffLines, "\n")
				files = append(files, *currentFile)
			}

			// 开始新文件
			filePath := matches[2] // b/侧的路径
			// 去除可能的a/前缀
			filePath = strings.TrimPrefix(filePath, "b/")

			additions, deletions := 0, 0
			if stat, ok := fileStats[filePath]; ok {
				additions = stat.additions
				deletions = stat.deletions
			}

			currentFile = &DiffFile{
				Path:      filePath,
				Status:    "modified", // 默认为修改
				Additions: additions,
				Deletions: deletions,
			}
			currentDiffLines = []string{line}
			continue
		}

		if currentFile != nil {
			currentDiffLines = append(currentDiffLines, line)
			// 检测文件状态
			if strings.HasPrefix(line, "new file mode ") {
				currentFile.Status = "added"
			} else if strings.HasPrefix(line, "deleted file mode ") {
				currentFile.Status = "deleted"
			} else if strings.HasPrefix(line, "rename from ") {
				currentFile.Status = "renamed"
			}
		}
	}

	// 保存最后一个文件
	if currentFile != nil {
		currentFile.Diff = strings.Join(currentDiffLines, "\n")
		files = append(files, *currentFile)
	}

	return files
}
