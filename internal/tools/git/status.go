package git

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Attect/MukaAI/internal/tools"
)

// ==================== git_status 工具 ====================

// GitStatusResult git_status工具的返回数据结构
type GitStatusResult struct {
	Branch    string   `json:"branch"`
	IsClean   bool     `json:"is_clean"`
	Modified  []string `json:"modified"`
	Added     []string `json:"added"`
	Deleted   []string `json:"deleted"`
	Untracked []string `json:"untracked"`
	Staged    []string `json:"staged"`
	Ahead     int      `json:"ahead"`
	Behind    int      `json:"behind"`
	Message   string   `json:"message"`
}

// gitStatusTool git_status工具实现
type gitStatusTool struct {
	workDir string
}

// NewGitStatusTool 创建git_status工具
func NewGitStatusTool(workDir string) *gitStatusTool {
	return &gitStatusTool{workDir: workDir}
}

func (t *gitStatusTool) Name() string {
	return "git_status"
}

func (t *gitStatusTool) Description() string {
	return "查看当前Git仓库的状态，包括分支、修改文件、暂存区文件、未跟踪文件、与远程的偏差等。无需参数，自动检测当前工作目录的Git状态。"
}

func (t *gitStatusTool) Parameters() map[string]interface{} {
	// 无参数
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *gitStatusTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	// 检查是否在Git仓库中
	if !isGitRepo(ctx, t.workDir) {
		return gitNotRepoError(), nil
	}

	// 执行 git status --porcelain=v2 --branch
	// porcelain=v2 提供机器可解析的格式，--branch包含分支信息
	stdout, stderr, exitCode, err := runGitCommand(ctx, t.workDir, "status", "--porcelain=v2", "--branch")
	if err != nil || exitCode != 0 {
		return tools.NewErrorResult(fmt.Sprintf("执行git status失败: %s", stderr)), nil
	}

	result := &GitStatusResult{
		Modified:  make([]string, 0),
		Added:     make([]string, 0),
		Deleted:   make([]string, 0),
		Untracked: make([]string, 0),
		Staged:    make([]string, 0),
	}

	// 解析porcelain=v2输出
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch {
		// # branch.oid <hash>
		// # branch.head <branch>
		// # branch.upstream <remote/branch>
		// # branch.ab +N -M
		case strings.HasPrefix(line, "# branch.head "):
			result.Branch = strings.TrimPrefix(line, "# branch.head ")
			if result.Branch == "(detached)" {
				result.Branch = "(detached HEAD)"
			}

		case strings.HasPrefix(line, "# branch.ab "):
			abStr := strings.TrimPrefix(line, "# branch.ab ")
			// 格式: +N -M
			parts := strings.Fields(abStr)
			for _, p := range parts {
				if strings.HasPrefix(p, "+") {
					result.Ahead, _ = strconv.Atoi(p[1:])
				} else if strings.HasPrefix(p, "-") {
					result.Behind, _ = strconv.Atoi(p[1:])
				}
			}

		// 1 <xy> <sub> <mH> <mI> <mW> <hH> <hI> <path>
		// 已跟踪文件的变更行，xy为两字符的状态码
		case strings.HasPrefix(line, "1 "):
			fields := strings.Fields(line)
			if len(fields) < 9 {
				continue
			}
			xy := fields[1]
			// path在fields[8]之后，因为路径可能包含空格
			pathStart := strings.Index(line[len("1 "):], fields[7]) + len(fields[7]) + len("1 ")
			filePath := strings.TrimSpace(line[pathStart:])
			if filePath == "" && len(fields) > 8 {
				filePath = fields[8]
			}
			if filePath == "" {
				continue
			}

			x := string(xy[0]) // 暂存区状态
			y := string(xy[1]) // 工作区状态

			// 暂存区状态
			if x == "A" || x == "M" || x == "D" || x == "R" || x == "C" {
				result.Staged = append(result.Staged, filePath)
			}

			// 工作区状态
			switch y {
			case "M":
				result.Modified = append(result.Modified, filePath)
			case "D":
				result.Deleted = append(result.Deleted, filePath)
			case "A":
				result.Added = append(result.Added, filePath)
			}

			// 如果暂存区也是修改/删除/新增，也记录到对应列表
			switch x {
			case "M":
				// 暂存区修改（如果工作区没有进一步修改）
				if y == "N" || y == "." {
					result.Modified = append(result.Modified, filePath)
				}
			case "A":
				if y == "N" || y == "." {
					result.Added = append(result.Added, filePath)
				}
			case "D":
				if y == "N" || y == "." {
					result.Deleted = append(result.Deleted, filePath)
				}
			}

		// 2 <xy> <sub> <mH> <mI> <mW> <hH> <hI> <X><num> <path>
		// 重命名/复制文件
		case strings.HasPrefix(line, "2 "):
			fields := strings.Fields(line)
			if len(fields) < 9 {
				continue
			}
			xy := fields[1]
			// 路径格式: <orig path> -> <new path>，在R/C操作中
			// 简化处理：提取路径部分
			pathParts := extractRenamedPath(line)
			if pathParts != "" {
				if string(xy[0]) == "R" || string(xy[0]) == "C" {
					result.Staged = append(result.Staged, pathParts)
				}
			}

		// ? <path>
		// 未跟踪文件
		case strings.HasPrefix(line, "? "):
			filePath := strings.TrimPrefix(line, "? ")
			result.Untracked = append(result.Untracked, filePath)

		// ! <path>
		// 忽略文件，跳过
		case strings.HasPrefix(line, "! "):
			// 忽略
		}
	}

	// 判断是否clean
	result.IsClean = len(result.Modified) == 0 && len(result.Added) == 0 &&
		len(result.Deleted) == 0 && len(result.Untracked) == 0 && len(result.Staged) == 0

	// 构建消息
	result.Message = buildStatusMessage(result)

	return tools.NewSuccessResult(result), nil
}

// extractRenamedPath 从porcelain=v2的第2类行中提取路径信息
func extractRenamedPath(line string) string {
	// 2 <xy> <sub> <mH> <mI> <mW> <hH> <hI> <X><num> <path> sep <orig path>
	// 使用正则匹配最后的路径部分
	re := regexp.MustCompile(`^2\s+\S+\s+\S+\s+\S+\s+\S+\s+\S+\s+\S+\s+\S+\s+\S+\d+\s+(.+)$`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// buildStatusMessage 构建状态摘要消息
func buildStatusMessage(r *GitStatusResult) string {
	if r.IsClean {
		return fmt.Sprintf("On branch %s, working tree clean", r.Branch)
	}

	parts := []string{fmt.Sprintf("On branch %s", r.Branch)}
	totalChanges := len(r.Modified) + len(r.Added) + len(r.Deleted) + len(r.Untracked) + len(r.Staged)
	parts = append(parts, fmt.Sprintf("%d file(s) changed", totalChanges))

	if len(r.Staged) > 0 {
		parts = append(parts, fmt.Sprintf("%d staged", len(r.Staged)))
	}
	if len(r.Modified) > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", len(r.Modified)))
	}
	if len(r.Untracked) > 0 {
		parts = append(parts, fmt.Sprintf("%d untracked", len(r.Untracked)))
	}
	if r.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("%d ahead", r.Ahead))
	}
	if r.Behind > 0 {
		parts = append(parts, fmt.Sprintf("%d behind", r.Behind))
	}

	return strings.Join(parts, ", ")
}
