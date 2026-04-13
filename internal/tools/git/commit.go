package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/Attect/MukaAI/internal/tools"
)

// ==================== git_commit 工具 ====================

// GitCommitResult git_commit工具的返回数据结构
type GitCommitResult struct {
	CommitHash   string `json:"commit_hash"`
	Branch       string `json:"branch"`
	FilesChanged int    `json:"files_changed"`
	Message      string `json:"message"`
}

// gitCommitTool git_commit工具实现
type gitCommitTool struct {
	workDir string
}

// NewGitCommitTool 创建git_commit工具
func NewGitCommitTool(workDir string) *gitCommitTool {
	return &gitCommitTool{workDir: workDir}
}

func (t *gitCommitTool) Name() string {
	return "git_commit"
}

func (t *gitCommitTool) Description() string {
	return "暂存并提交Git变更。推荐使用规范的提交消息格式: feat|fix|refactor|docs|test|chore: <描述>。" +
		"例如: 'feat: 添加用户登录功能'、'fix: 修复订单金额计算错误'。" +
		"默认暂存所有修改的文件(all=true)，也可指定具体的文件列表。"
}

func (t *gitCommitTool) Parameters() map[string]interface{} {
	return tools.BuildSchema(map[string]*tools.ToolParameter{
		"message": {
			Type:        "string",
			Description: "提交消息，推荐格式: feat|fix|refactor|docs|test|chore: <描述>",
		},
		"files": {
			Type:        "array",
			Description: "要提交的文件路径列表。不指定则暂存所有已修改文件",
			Items: &tools.ToolParameter{
				Type: "string",
			},
		},
		"all": {
			Type:        "boolean",
			Description: "是否暂存所有修改和删除的文件（git add -u），默认true。设为false时需配合files参数使用",
		},
	}, []string{"message"})
}

func (t *gitCommitTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	// 检查是否在Git仓库中
	if !isGitRepo(ctx, t.workDir) {
		return gitNotRepoError(), nil
	}

	// 解析必填参数: message
	messageVal, ok := params["message"]
	if !ok {
		return tools.NewErrorResult("缺少必填参数: message"), nil
	}
	message, ok := messageVal.(string)
	if !ok || strings.TrimSpace(message) == "" {
		return tools.NewErrorResult("参数 message 不能为空"), nil
	}

	// 解析可选参数: all (默认true)
	all := true
	if allVal, ok := params["all"]; ok {
		if b, ok := allVal.(bool); ok {
			all = b
		}
	}

	// 解析可选参数: files
	var files []string
	if filesVal, ok := params["files"]; ok {
		if arr, ok := filesVal.([]interface{}); ok {
			for _, f := range arr {
				if s, ok := f.(string); ok && s != "" {
					files = append(files, s)
				}
			}
		}
	}

	// 第一步: git add
	if len(files) > 0 {
		// 指定文件模式
		args := append([]string{"add", "--"}, files...)
		_, stderr, exitCode, err := runGitCommand(ctx, t.workDir, args...)
		if err != nil || exitCode != 0 {
			return tools.NewErrorResult(fmt.Sprintf("git add 失败: %s", stderr)), nil
		}
	} else if all {
		// 暂存所有已修改和删除的文件（不包括未跟踪文件）
		_, stderr, exitCode, err := runGitCommand(ctx, t.workDir, "add", "-u")
		if err != nil || exitCode != 0 {
			return tools.NewErrorResult(fmt.Sprintf("git add -u 失败: %s", stderr)), nil
		}
	} else {
		// all=false且未指定files，需要检查是否有已暂存的文件
		stdout, _, _, _ := runGitCommand(ctx, t.workDir, "diff", "--staged", "--name-only")
		if strings.TrimSpace(stdout) == "" {
			return tools.NewErrorResult("没有已暂存的文件可以提交。请设置 all=true 或指定 files 参数"), nil
		}
	}

	// 第二步: git commit
	_, stderr, exitCode, err := runGitCommand(ctx, t.workDir, "commit", "-m", message)
	if err != nil || exitCode != 0 {
		// 检查是否是"nothing to commit"
		if strings.Contains(stderr, "nothing to commit") {
			return tools.NewErrorResult("没有可提交的变更。工作区是干净的，或指定的文件没有修改"), nil
		}
		return tools.NewErrorResult(fmt.Sprintf("git commit 失败: %s", stderr)), nil
	}

	// 获取提交信息
	hash := ""
	stdout, _, _, _ := runGitCommand(ctx, t.workDir, "log", "-1", "--format=%h")
	hash = strings.TrimSpace(stdout)

	branch := ""
	stdout, _, _, _ = runGitCommand(ctx, t.workDir, "rev-parse", "--abbrev-ref", "HEAD")
	branch = strings.TrimSpace(stdout)

	// 获取变更文件数
	filesChanged := 0
	stdout, _, _, _ = runGitCommand(ctx, t.workDir, "diff", "--staged", "--name-only")
	if stdout != "" {
		filesChanged = len(strings.Split(strings.TrimSpace(stdout), "\n"))
	}
	// 如果是刚提交完，staged可能已经清空，用HEAD~1..HEAD
	stdout, _, _, _ = runGitCommand(ctx, t.workDir, "diff", "HEAD~1", "HEAD", "--name-only")
	if stdout != "" {
		filesChanged = len(strings.Split(strings.TrimSpace(stdout), "\n"))
	}

	return tools.NewSuccessResult(&GitCommitResult{
		CommitHash:   hash,
		Branch:       branch,
		FilesChanged: filesChanged,
		Message:      "commit created successfully",
	}), nil
}

// ==================== git_add 工具 ====================

// gitAddTool git_add工具实现
type gitAddTool struct {
	workDir string
}

// NewGitAddTool 创建git_add工具
func NewGitAddTool(workDir string) *gitAddTool {
	return &gitAddTool{workDir: workDir}
}

func (t *gitAddTool) Name() string {
	return "git_add"
}

func (t *gitAddTool) Description() string {
	return "将文件添加到Git暂存区。指定要暂存的文件路径列表。使用 '.' 可以暂存所有变更文件。"
}

func (t *gitAddTool) Parameters() map[string]interface{} {
	return tools.BuildSchema(map[string]*tools.ToolParameter{
		"files": {
			Type:        "array",
			Description: "要暂存的文件路径列表。使用 '.' 暂存所有文件",
			Items: &tools.ToolParameter{
				Type: "string",
			},
		},
	}, []string{"files"})
}

func (t *gitAddTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	// 检查是否在Git仓库中
	if !isGitRepo(ctx, t.workDir) {
		return gitNotRepoError(), nil
	}

	// 解析必填参数: files
	filesVal, ok := params["files"]
	if !ok {
		return tools.NewErrorResult("缺少必填参数: files"), nil
	}

	var files []string
	switch fv := filesVal.(type) {
	case []interface{}:
		for _, f := range fv {
			if s, ok := f.(string); ok && s != "" {
				files = append(files, s)
			}
		}
	case string:
		// 兼容直接传字符串的情况
		if fv != "" {
			files = append(files, fv)
		}
	}

	if len(files) == 0 {
		return tools.NewErrorResult("参数 files 不能为空，至少指定一个文件路径"), nil
	}

	// 执行 git add
	args := append([]string{"add", "--"}, files...)
	_, stderr, exitCode, err := runGitCommand(ctx, t.workDir, args...)
	if err != nil || exitCode != 0 {
		return tools.NewErrorResult(fmt.Sprintf("git add 失败: %s", stderr)), nil
	}

	return tools.NewSuccessResult(map[string]interface{}{
		"files_added": files,
		"message":     fmt.Sprintf("%d file(s) staged successfully", len(files)),
	}), nil
}
