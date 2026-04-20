// Package git 提供原生Git操作工具集
// 封装Git操作最佳实践，使Agent无需了解Git命令细节即可完成版本控制操作
package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/Attect/MukaAI/internal/tools"
)

// gitAvailable Git是否可用的缓存状态
var (
	gitAvailable     bool
	gitAvailableOnce sync.Once
)

// checkGitAvailable 检测git命令是否可用
// 结果会被缓存，只检测一次
func checkGitAvailable() bool {
	gitAvailableOnce.Do(func() {
		cmd := exec.Command("git", "--version")
		// Windows 下隐藏控制台窗口（避免闪窗）
		configureHideWindow(cmd)
		if err := cmd.Run(); err != nil {
			gitAvailable = false
			return
		}
		gitAvailable = true
	})
	return gitAvailable
}

// runGitCommand 执行Git命令并返回输出
// 所有Git工具共用此函数执行底层git命令
func runGitCommand(ctx context.Context, workDir string, args ...string) (stdout, stderr string, exitCode int, err error) {
	// 首次使用时检测Git可用性
	if !checkGitAvailable() {
		return "", "git command is not available on this system", -1, fmt.Errorf("git命令不可用，请确保系统已安装Git")
	}

	// 设置30秒超时，防止长时间阻塞
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "git", args...)
	cmd.Dir = workDir

	// Windows 下隐藏控制台窗口（避免闪窗）
	configureHideWindow(cmd)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
		return stdout, stderr, exitCode, runErr
	}

	return stdout, stderr, 0, nil
}

// isGitRepo 检查工作目录是否在Git仓库中
func isGitRepo(ctx context.Context, workDir string) bool {
	_, _, exitCode, _ := runGitCommand(ctx, workDir, "rev-parse", "--is-inside-work-tree")
	return exitCode == 0
}

// gitNotRepoError 返回不在Git仓库中的错误结果
func gitNotRepoError() *tools.ToolResult {
	return tools.NewErrorResult("当前目录不在Git仓库中，请先初始化Git仓库（git init）或切换到Git仓库目录")
}

// RegisterGitTools 注册所有Git工具到工具注册中心
func RegisterGitTools(registry *tools.ToolRegistry, workDir string) error {
	toolList := []tools.Tool{
		NewGitStatusTool(workDir),
		NewGitDiffTool(workDir),
		NewGitCommitTool(workDir),
		NewGitLogTool(workDir),
		NewGitAddTool(workDir),
	}

	for _, tool := range toolList {
		if err := registry.RegisterTool(tool); err != nil {
			return fmt.Errorf("failed to register git tool %s: %w", tool.Name(), err)
		}
	}
	return nil
}
