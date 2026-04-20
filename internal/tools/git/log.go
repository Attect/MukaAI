package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Attect/MukaAI/internal/tools"
)

// ==================== git_log 工具 ====================

// CommitEntry 单条提交记录
type CommitEntry struct {
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Message string `json:"message"`
}

// GitLogResult git_log工具的返回数据结构
type GitLogResult struct {
	Commits []CommitEntry `json:"commits"`
	Total   int           `json:"total"`
	Message string        `json:"message"`
}

// gitLogTool git_log工具实现
type gitLogTool struct {
	workDir string
}

// NewGitLogTool 创建git_log工具
func NewGitLogTool(workDir string) *gitLogTool {
	return &gitLogTool{workDir: workDir}
}

// SetWorkDir 设置工作目录（用于Git操作路径校验）
func (t *gitLogTool) SetWorkDir(workDir string) {
	t.workDir = workDir
}

func (t *gitLogTool) Name() string {
	return "git_log"
}

func (t *gitLogTool) Description() string {
	return "查看Git提交日志。支持限制显示条数、按文件路径过滤。默认使用单行格式显示最近的提交记录。"
}

func (t *gitLogTool) Parameters() map[string]interface{} {
	return tools.BuildSchema(map[string]*tools.ToolParameter{
		"count": {
			Type:        "integer",
			Description: "显示的提交条数，默认20",
		},
		"path": {
			Type:        "string",
			Description: "按文件路径过滤提交日志",
		},
		"oneline": {
			Type:        "boolean",
			Description: "是否使用单行格式显示（默认true）。设为false时显示完整的提交信息（作者、日期、消息）",
		},
	}, nil)
}

func (t *gitLogTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	// 检查是否在Git仓库中
	if !isGitRepo(ctx, t.workDir) {
		return gitNotRepoError(), nil
	}

	// 解析参数
	count := 20
	if countVal, ok := params["count"]; ok {
		switch c := countVal.(type) {
		case float64:
			count = int(c)
		case int:
			count = c
		}
	}
	if count <= 0 {
		count = 20
	}
	if count > 100 {
		count = 100 // 限制最大100条，防止输出过长
	}

	path, _ := params["path"].(string)

	oneline := true
	if onelineVal, ok := params["oneline"]; ok {
		if b, ok := onelineVal.(bool); ok {
			oneline = b
		}
	}

	// 构建git log命令
	if oneline {
		return t.executeOneline(ctx, count, path)
	}
	return t.executeFull(ctx, count, path)
}

// executeOneline 执行单行格式的git log
func (t *gitLogTool) executeOneline(ctx context.Context, count int, path string) (*tools.ToolResult, error) {
	args := []string{"log", "--oneline", "-n", strconv.Itoa(count)}
	if path != "" {
		args = append(args, "--", path)
	}

	stdout, stderr, exitCode, err := runGitCommand(ctx, t.workDir, args...)
	if err != nil || exitCode != 0 {
		return tools.NewErrorResult(fmt.Sprintf("执行git log失败: %s", stderr)), nil
	}

	var commits []CommitEntry
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 格式: <hash> <message>
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 1 {
			continue
		}
		entry := CommitEntry{
			Hash: parts[0],
		}
		if len(parts) >= 2 {
			entry.Message = parts[1]
		}
		commits = append(commits, entry)
	}

	if commits == nil {
		commits = []CommitEntry{}
	}

	return tools.NewSuccessResult(&GitLogResult{
		Commits: commits,
		Total:   len(commits),
		Message: fmt.Sprintf("showing %d commits", len(commits)),
	}), nil
}

// executeFull 执行完整格式的git log
func (t *gitLogTool) executeFull(ctx context.Context, count int, path string) (*tools.ToolResult, error) {
	// 使用自定义格式，便于解析
	// 每个字段用特殊分隔符包裹
	args := []string{
		"log",
		fmt.Sprintf("-n%d", count),
		"--format=__COMMIT__%n%H%n%an%n%ai%n%s",
	}
	if path != "" {
		args = append(args, "--", path)
	}

	stdout, stderr, exitCode, err := runGitCommand(ctx, t.workDir, args...)
	if err != nil || exitCode != 0 {
		return tools.NewErrorResult(fmt.Sprintf("执行git log失败: %s", stderr)), nil
	}

	var commits []CommitEntry
	entries := strings.Split(stdout, "__COMMIT__\n")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		lines := strings.SplitN(entry, "\n", 4)
		if len(lines) < 3 {
			continue
		}
		commit := CommitEntry{
			Hash:    strings.TrimSpace(lines[0]),
			Author:  strings.TrimSpace(lines[1]),
			Date:    strings.TrimSpace(lines[2]),
			Message: "",
		}
		if len(lines) >= 4 {
			commit.Message = strings.TrimSpace(lines[3])
		}
		commits = append(commits, commit)
	}

	if commits == nil {
		commits = []CommitEntry{}
	}

	return tools.NewSuccessResult(&GitLogResult{
		Commits: commits,
		Total:   len(commits),
		Message: fmt.Sprintf("showing %d commits", len(commits)),
	}), nil
}
