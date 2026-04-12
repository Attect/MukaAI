package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCommandSecurityChecker(t *testing.T) {
	workDir := "/workspace/test"
	baseAllow := []string{"go", "git"}
	checker := NewCommandSecurityChecker(workDir, baseAllow)

	// 验证白名单合并
	allowList := checker.GetExpandedAllowList()

	// 用户配置的应该在列表中
	if !containsStr(allowList, "go") {
		t.Error("用户白名单 'go' 应该在扩展列表中")
	}
	if !containsStr(allowList, "git") {
		t.Error("用户白名单 'git' 应该在扩展列表中")
	}

	// 常见构建命令也应该在列表中
	commonCmds := []string{"gcc", "cargo", "node", "python", "java", "npm", "pip", "cmake"}
	for _, cmd := range commonCmds {
		if !containsStr(allowList, cmd) {
			t.Errorf("常见命令 '%s' 应该在扩展白名单中", cmd)
		}
	}
}

func TestSecurityChecker_WhitelistedCommand(t *testing.T) {
	workDir := "/workspace/test"
	checker := NewCommandSecurityChecker(workDir, nil)

	// 白名单命令应该直接放行
	result := checker.Check("go", []string{"build", "./..."})
	if result.Verdict != SecurityAllow {
		t.Errorf("go build 应该被允许，实际: %s, 原因: %s", result.Verdict, result.Reason)
	}

	result = checker.Check("python", []string{"hello.py"})
	if result.Verdict != SecurityAllow {
		t.Errorf("python 应该被允许，实际: %s", result.Verdict)
	}

	result = checker.Check("gcc", []string{"hello.c", "-o", "hello"})
	if result.Verdict != SecurityAllow {
		t.Errorf("gcc 应该被允许，实际: %s", result.Verdict)
	}

	result = checker.Check("cargo", []string{"build"})
	if result.Verdict != SecurityAllow {
		t.Errorf("cargo 应该被允许，实际: %s", result.Verdict)
	}
}

func TestSecurityChecker_DangerousCommands(t *testing.T) {
	workDir := "/workspace/test"
	checker := NewCommandSecurityChecker(workDir, nil)

	// rm -rf 无明确目标 → 拒绝
	result := checker.Check("rm", []string{"-rf"})
	if result.Verdict != SecurityDeny {
		t.Errorf("rm -rf 无目标应该被拒绝，实际: %s", result.Verdict)
	}

	// rm -rf ~ → 拒绝（~是危险目标）
	result = checker.Check("rm", []string{"-rf", "~"})
	if result.Verdict != SecurityDeny {
		t.Errorf("rm -rf ~ 应该被拒绝，实际: %s", result.Verdict)
	}
}

func TestSecurityChecker_RmSafety(t *testing.T) {
	// 使用真实的临时目录确保跨平台兼容
	workDir := t.TempDir()
	checker := NewCommandSecurityChecker(workDir, nil)

	// rm 工作区内文件 → 放行
	result := checker.Check("rm", []string{"test.txt"})
	if result.Verdict != SecurityAllow {
		t.Errorf("rm 工作区内文件应该被允许，实际: %s", result.Verdict)
	}

	// rm 工作区外文件（使用真实的外部路径）→ 需确认
	outsidePath := filepath.Join(workDir, "..", "..", "etc", "hosts")
	result = checker.Check("rm", []string{outsidePath})
	if result.Verdict != SecurityConfirm {
		t.Errorf("rm 工作区外文件应该需要确认，实际: %s, 原因: %s", result.Verdict, result.Reason)
	}

	// rm -rf 无目标 → 拒绝
	result = checker.Check("rm", []string{"-rf"})
	if result.Verdict != SecurityDeny {
		t.Errorf("rm -rf 无目标应该被拒绝，实际: %s", result.Verdict)
	}
}

func TestSecurityChecker_NonWhitelistedCommand(t *testing.T) {
	workDir := "/workspace/test"
	checker := NewCommandSecurityChecker(workDir, nil)

	// 未知的非危险命令 → 需确认
	result := checker.Check("my-custom-tool", []string{"--help"})
	if result.Verdict != SecurityConfirm {
		t.Errorf("未知非危险命令应该需要确认，实际: %s", result.Verdict)
	}
}

func TestSecurityChecker_SensitiveFiles(t *testing.T) {
	workDir := "/workspace/test"
	checker := NewCommandSecurityChecker(workDir, nil)

	// 访问敏感文件 → 需确认
	result := checker.Check("cat", []string{"/etc/shadow"})
	if result.Verdict != SecurityConfirm {
		t.Errorf("访问敏感文件应该需要确认，实际: %s, 原因: %s", result.Verdict, result.Reason)
	}
}

func TestSecurityChecker_ScriptExecution(t *testing.T) {
	workDir := "/workspace/test"
	checker := NewCommandSecurityChecker(workDir, nil)

	// 执行工作区内脚本 → 通过sh白名单放行
	result := checker.Check("sh", []string{"build.sh"})
	if result.Verdict != SecurityAllow {
		t.Errorf("sh执行脚本应该被允许，实际: %s", result.Verdict)
	}

	result = checker.Check("bash", []string{"deploy.sh"})
	if result.Verdict != SecurityAllow {
		t.Errorf("bash执行脚本应该被允许，实际: %s", result.Verdict)
	}
}

func TestSecurityChecker_UserApproveFunc(t *testing.T) {
	workDir := "/workspace/test"
	checker := NewCommandSecurityChecker(workDir, nil)

	approved := false
	checker.SetUserApproveFunc(func(command, reason string) bool {
		approved = true
		return true
	})

	// 非白名单命令，应该需要确认
	result := checker.Check("custom-tool", []string{"arg1"})
	if result.Verdict != SecurityConfirm {
		t.Errorf("非白名单命令应该需要确认，实际: %s", result.Verdict)
	}

	// 确认函数可通过GetUserApproveFunc获取
	fn := checker.GetUserApproveFunc()
	if fn == nil {
		t.Error("确认函数不应为nil")
	}
	// 调用确认函数验证其工作
	fn("test", "test reason")
	if !approved {
		t.Error("确认函数应该被调用")
	}
}

func TestSecurityChecker_WindowsBuildCommands(t *testing.T) {
	workDir := "C:\\workspace\\test"
	checker := NewCommandSecurityChecker(workDir, nil)

	// Windows构建命令也应该放行
	result := checker.Check("dotnet", []string{"build"})
	if result.Verdict != SecurityAllow {
		t.Errorf("dotnet build 应该被允许，实际: %s", result.Verdict)
	}

	result = checker.Check("msbuild", []string{"project.sln"})
	if result.Verdict != SecurityAllow {
		t.Errorf("msbuild 应该被允许，实际: %s", result.Verdict)
	}
}

func TestSecurityChecker_RealWorldScenarios(t *testing.T) {
	// 获取当前目录作为工作目录
	workDir, _ := filepath.Abs(".")
	checker := NewCommandSecurityChecker(workDir, []string{"go"})

	// 场景1: go test → 放行
	result := checker.Check("go", []string{"test", "./..."})
	if result.Verdict != SecurityAllow {
		t.Errorf("go test 应该被允许: %s", result.Reason)
	}

	// 场景2: npm install → 放行
	result = checker.Check("npm", []string{"install"})
	if result.Verdict != SecurityAllow {
		t.Errorf("npm install 应该被允许: %s", result.Reason)
	}

	// 场景3: python -m pytest → 放行
	result = checker.Check("python", []string{"-m", "pytest"})
	if result.Verdict != SecurityAllow {
		t.Errorf("python -m pytest 应该被允许: %s", result.Reason)
	}

	// 场景4: curl下载脚本到系统目录 → 需确认或拒绝
	result = checker.Check("curl", []string{"-o", "/etc/script.sh", "http://example.com/script.sh"})
	if result.Verdict != SecurityConfirm && result.Verdict != SecurityDeny {
		t.Errorf("curl到系统目录应该需要确认: %s", result.Reason)
	}
}

func TestSecurityChecker_IsCommandInExpandedAllowList(t *testing.T) {
	workDir := "/workspace/test"
	checker := NewCommandSecurityChecker(workDir, []string{"go"})

	if !checker.IsCommandInExpandedAllowList("go") {
		t.Error("go 应该在扩展白名单中")
	}

	if !checker.IsCommandInExpandedAllowList("npm") {
		t.Error("npm 应该在扩展白名单中（自动包含）")
	}

	if checker.IsCommandInExpandedAllowList("malicious-tool") {
		t.Error("malicious-tool 不应该在扩展白名单中")
	}
}

func TestSecurityChecker_SystemDirAccess(t *testing.T) {
	workDir := "/workspace/test"
	checker := NewCommandSecurityChecker(workDir, nil)

	// 写入系统目录 → 需确认
	result := checker.Check("unknown-cmd", []string{"write", "/etc/config.conf"})
	if result.Verdict != SecurityConfirm {
		t.Errorf("写入系统目录应该需要确认，实际: %s, 原因: %s", result.Verdict, result.Reason)
	}

	// 访问Windows系统目录
	result = checker.Check("unknown-cmd", []string{"copy", "c:\\windows\\system32\\test"})
	if result.Verdict != SecurityConfirm {
		t.Errorf("访问Windows系统目录应该需要确认，实际: %s", result.Verdict)
	}
}

func TestSecurityChecker_SimplePatternMatch(t *testing.T) {
	tests := []struct {
		pattern string
		text    string
		want    bool
	}{
		{"rm -rf /", "rm -rf /", true},
		{"curl.*secret", "curl -d secret=value", true},
		{"nc -l", "nc -l 8080", true},
		{"curl.*secret", "wget --safe", false},
		{"iptables -f", "iptables -f", true},
	}

	for _, tt := range tests {
		got := simplePatternMatch(tt.pattern, tt.text)
		if got != tt.want {
			t.Errorf("simplePatternMatch(%q, %q) = %v, want %v", tt.pattern, tt.text, got, tt.want)
		}
	}
}

func TestSecurityChecker_Deduplication(t *testing.T) {
	workDir := "/workspace/test"
	// 包含重复项
	baseAllow := []string{"go", "GO", "Go", "npm", "NPM"}
	checker := NewCommandSecurityChecker(workDir, baseAllow)

	allowList := checker.GetExpandedAllowList()

	// 统计go出现的次数
	goCount := 0
	for _, cmd := range allowList {
		if cmd == "go" {
			goCount++
		}
	}
	if goCount != 1 {
		t.Errorf("go 应该只出现一次（去重），实际出现 %d 次", goCount)
	}
}

func TestSecurityChecker_RmInWorkDirSubdirectory(t *testing.T) {
	// 使用真实路径测试
	tmpDir := t.TempDir()
	checker := NewCommandSecurityChecker(tmpDir, nil)

	// 删除工作目录内的子目录文件 → 放行
	result := checker.Check("rm", []string{filepath.Join("subdir", "test.txt")})
	if result.Verdict != SecurityAllow {
		t.Errorf("rm 工作区内子目录文件应该被允许，实际: %s, 原因: %s", result.Verdict, result.Reason)
	}

	// 使用绝对路径删除工作区内的文件 → 放行
	result = checker.Check("rm", []string{filepath.Join(tmpDir, "test.txt")})
	if result.Verdict != SecurityAllow {
		t.Errorf("rm 工作区内绝对路径文件应该被允许，实际: %s, 原因: %s", result.Verdict, result.Reason)
	}

	// 删除工作区外的文件 → 需确认
	result = checker.Check("rm", []string{filepath.Join(tmpDir, "..", "outside.txt")})
	if result.Verdict != SecurityConfirm {
		t.Errorf("rm 工作区外文件应该需要确认，实际: %s, 原因: %s", result.Verdict, result.Reason)
	}
}

func TestSecurityChecker_EmptyWorkDir(t *testing.T) {
	checker := NewCommandSecurityChecker("", nil)

	// workDir为空时，rm路径检查不应崩溃
	result := checker.Check("rm", []string{"test.txt"})
	// 不会panic即可
	_ = result
}

func containsStr(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func TestSecurityChecker_EnvironmentVariableCommands(t *testing.T) {
	workDir := "/workspace/test"
	checker := NewCommandSecurityChecker(workDir, nil)

	// 环境查看命令应该放行
	result := checker.Check("env", nil)
	if result.Verdict != SecurityAllow {
		t.Errorf("env 应该被允许，实际: %s", result.Verdict)
	}

	result = checker.Check("printenv", nil)
	if result.Verdict != SecurityAllow {
		t.Errorf("printenv 应该被允许，实际: %s", result.Verdict)
	}

	result = checker.Check("which", []string{"go"})
	if result.Verdict != SecurityAllow {
		t.Errorf("which 应该被允许，实际: %s", result.Verdict)
	}
}

func TestSecurityChecker_DockerCommands(t *testing.T) {
	workDir := "/workspace/test"
	checker := NewCommandSecurityChecker(workDir, nil)

	// docker命令应该放行
	result := checker.Check("docker", []string{"build", "-t", "myapp", "."})
	if result.Verdict != SecurityAllow {
		t.Errorf("docker build 应该被允许，实际: %s", result.Verdict)
	}

	result = checker.Check("docker", []string{"ps"})
	if result.Verdict != SecurityAllow {
		t.Errorf("docker ps 应该被允许，实际: %s", result.Verdict)
	}

	// podman命令也应该放行
	result = checker.Check("podman", []string{"images"})
	if result.Verdict != SecurityAllow {
		t.Errorf("podman images 应该被允许，实际: %s", result.Verdict)
	}
}

func TestSecurityChecker_PathWithSpaces(t *testing.T) {
	// 测试包含空格的Windows路径
	if os.PathSeparator == '\\' {
		workDir := "C:\\Users\\Test User\\project"
		checker := NewCommandSecurityChecker(workDir, nil)

		result := checker.Check("rm", []string{"test.txt"})
		if result.Verdict != SecurityAllow {
			t.Errorf("rm 工作区内文件应该被允许（路径含空格），实际: %s", result.Verdict)
		}
	}
}
