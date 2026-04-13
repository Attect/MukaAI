package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Attect/MukaAI/internal/tools"
)

// setupTestRepo 创建一个临时Git仓库用于测试
// 返回仓库路径和清理函数
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	// 检查git是否可用
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}

	tmpDir, err := os.MkdirTemp("", "git-tools-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	// 初始化Git仓库
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("git init 失败: %v", err)
	}

	// 配置用户信息（CI环境可能没有）
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = tmpDir
	cmd.Run()

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// createAndCommitFile 创建文件并提交
func createAndCommitFile(t *testing.T, repoDir, fileName, content, commitMsg string) {
	t.Helper()
	filePath := filepath.Join(repoDir, fileName)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	cmd := exec.Command("git", "add", fileName)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git add 失败: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", commitMsg)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit 失败: %v", err)
	}
}

// ==================== git_status 测试 ====================

func TestGitStatusTool_Name(t *testing.T) {
	tool := NewGitStatusTool(".")
	if tool.Name() != "git_status" {
		t.Errorf("expected name 'git_status', got '%s'", tool.Name())
	}
}

func TestGitStatusTool_NotGitRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "not-git-repo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tool := NewGitStatusTool(tmpDir)
	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for non-git repo")
	}
}

func TestGitStatusTool_CleanRepo(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	createAndCommitFile(t, repoDir, "initial.txt", "hello", "initial commit")

	tool := NewGitStatusTool(repoDir)
	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Data.(*GitStatusResult)
	if !ok {
		t.Fatal("unexpected data type")
	}
	if !data.IsClean {
		t.Error("expected clean repo")
	}
	if data.Branch == "" {
		t.Error("expected non-empty branch name")
	}
}

func TestGitStatusTool_WithModifications(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	createAndCommitFile(t, repoDir, "file1.txt", "initial content", "initial commit")

	// 修改文件
	if err := os.WriteFile(filepath.Join(repoDir, "file1.txt"), []byte("modified content"), 0644); err != nil {
		t.Fatal(err)
	}

	// 创建未跟踪文件
	if err := os.WriteFile(filepath.Join(repoDir, "untracked.txt"), []byte("new file"), 0644); err != nil {
		t.Fatal(err)
	}

	tool := NewGitStatusTool(repoDir)
	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Data.(*GitStatusResult)
	if !ok {
		t.Fatal("unexpected data type")
	}
	if data.IsClean {
		t.Error("expected dirty repo")
	}
	if len(data.Modified) == 0 {
		t.Error("expected modified files")
	}
	if len(data.Untracked) == 0 {
		t.Error("expected untracked files")
	}
}

// ==================== git_diff 测试 ====================

func TestGitDiffTool_Name(t *testing.T) {
	tool := NewGitDiffTool(".")
	if tool.Name() != "git_diff" {
		t.Errorf("expected name 'git_diff', got '%s'", tool.Name())
	}
}

func TestGitDiffTool_NoChanges(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	createAndCommitFile(t, repoDir, "file1.txt", "content", "initial commit")

	tool := NewGitDiffTool(repoDir)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"staged": false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Data.(*GitDiffResult)
	if !ok {
		t.Fatal("unexpected data type")
	}
	if len(data.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(data.Files))
	}
}

func TestGitDiffTool_WithChanges(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	createAndCommitFile(t, repoDir, "file1.txt", "line1\nline2\nline3\n", "initial commit")

	// 修改文件
	if err := os.WriteFile(filepath.Join(repoDir, "file1.txt"), []byte("line1\nmodified\nline3\nnew_line\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tool := NewGitDiffTool(repoDir)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"staged": false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Data.(*GitDiffResult)
	if !ok {
		t.Fatal("unexpected data type")
	}
	if len(data.Files) == 0 {
		t.Error("expected at least one changed file")
	}
	if data.Files[0].Additions == 0 && data.Files[0].Deletions == 0 {
		t.Error("expected some additions or deletions")
	}
	if data.Files[0].Diff == "" {
		t.Error("expected non-empty diff content")
	}
}

func TestGitDiffTool_StagedChanges(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	createAndCommitFile(t, repoDir, "file1.txt", "original\n", "initial commit")

	// 修改并暂存
	if err := os.WriteFile(filepath.Join(repoDir, "file1.txt"), []byte("modified\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", "file1.txt")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	tool := NewGitDiffTool(repoDir)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"staged": true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Data.(*GitDiffResult)
	if !ok {
		t.Fatal("unexpected data type")
	}
	if len(data.Files) == 0 {
		t.Error("expected staged changes")
	}
}

// ==================== git_commit 测试 ====================

func TestGitCommitTool_Name(t *testing.T) {
	tool := NewGitCommitTool(".")
	if tool.Name() != "git_commit" {
		t.Errorf("expected name 'git_commit', got '%s'", tool.Name())
	}
}

func TestGitCommitTool_MissingMessage(t *testing.T) {
	tool := NewGitCommitTool(".")
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for missing message")
	}
}

func TestGitCommitTool_Success(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// 创建文件
	if err := os.WriteFile(filepath.Join(repoDir, "newfile.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	tool := NewGitCommitTool(repoDir)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"message": "test: initial commit",
		"all":     false,
		"files":   []interface{}{"newfile.txt"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Data.(*GitCommitResult)
	if !ok {
		t.Fatal("unexpected data type")
	}
	if data.CommitHash == "" {
		t.Error("expected non-empty commit hash")
	}
}

func TestGitCommitTool_AllFlag(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	createAndCommitFile(t, repoDir, "file1.txt", "initial", "initial commit")

	// 修改文件
	if err := os.WriteFile(filepath.Join(repoDir, "file1.txt"), []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	tool := NewGitCommitTool(repoDir)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"message": "fix: update file",
		"all":     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
}

// ==================== git_log 测试 ====================

func TestGitLogTool_Name(t *testing.T) {
	tool := NewGitLogTool(".")
	if tool.Name() != "git_log" {
		t.Errorf("expected name 'git_log', got '%s'", tool.Name())
	}
}

func TestGitLogTool_WithCommits(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// 创建几个提交
	for i := 0; i < 3; i++ {
		fileName := filepath.Join(repoDir, "file"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(fileName, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("git", "add", ".")
		cmd.Dir = repoDir
		cmd.Run()
		cmd = exec.Command("git", "commit", "-m", "commit "+string(rune('0'+i)))
		cmd.Dir = repoDir
		cmd.Run()
	}

	tool := NewGitLogTool(repoDir)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"count": 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Data.(*GitLogResult)
	if !ok {
		t.Fatal("unexpected data type")
	}
	if data.Total != 3 {
		t.Errorf("expected 3 commits, got %d", data.Total)
	}
}

func TestGitLogTool_FullFormat(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	createAndCommitFile(t, repoDir, "file.txt", "content", "test commit")

	tool := NewGitLogTool(repoDir)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"oneline": false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Data.(*GitLogResult)
	if !ok {
		t.Fatal("unexpected data type")
	}
	if len(data.Commits) == 0 {
		t.Fatal("expected at least one commit")
	}
	if data.Commits[0].Author == "" {
		t.Error("expected non-empty author in full format")
	}
	if data.Commits[0].Date == "" {
		t.Error("expected non-empty date in full format")
	}
}

// ==================== git_add 测试 ====================

func TestGitAddTool_Name(t *testing.T) {
	tool := NewGitAddTool(".")
	if tool.Name() != "git_add" {
		t.Errorf("expected name 'git_add', got '%s'", tool.Name())
	}
}

func TestGitAddTool_MissingFiles(t *testing.T) {
	tool := NewGitAddTool(".")
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for missing files")
	}
}

func TestGitAddTool_Success(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// 创建文件
	if err := os.WriteFile(filepath.Join(repoDir, "test.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	tool := NewGitAddTool(repoDir)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"files": []interface{}{"test.txt"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
}

// ==================== RegisterGitTools 测试 ====================

func TestRegisterGitTools(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := RegisterGitTools(registry, ".")
	if err != nil {
		t.Fatalf("RegisterGitTools failed: %v", err)
	}

	expectedTools := []string{"git_status", "git_diff", "git_commit", "git_log", "git_add"}
	for _, name := range expectedTools {
		tool, exists := registry.GetTool(name)
		if !exists {
			t.Errorf("expected tool '%s' to be registered", name)
			continue
		}
		if tool.Name() != name {
			t.Errorf("expected tool name '%s', got '%s'", name, tool.Name())
		}
	}
}

func TestRegisterGitTools_DuplicateRegistration(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := RegisterGitTools(registry, ".")
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// 第二次注册应该失败（名称重复）
	err = RegisterGitTools(registry, ".")
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

// ==================== 辅助函数测试 ====================

func TestCheckGitAvailable(t *testing.T) {
	// 重置缓存状态
	gitAvailableOnce = sync.Once{}
	gitAvailable = false

	available := checkGitAvailable()
	if _, err := exec.LookPath("git"); err == nil && !available {
		t.Error("expected git to be available")
	}
}

func TestRunGitCommand_InvalidDir(t *testing.T) {
	// 在不存在的目录执行git命令
	_, _, exitCode, _ := runGitCommand(context.Background(), "/nonexistent/path", "status")
	if exitCode == 0 {
		t.Error("expected non-zero exit code for invalid directory")
	}
}
