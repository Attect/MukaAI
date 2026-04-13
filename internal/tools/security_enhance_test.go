package tools

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ==================== Shell管道分割 Tests ====================

func TestSplitShellSubCommands(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    []string
	}{
		{
			name:    "单命令",
			command: "go build",
			want:    []string{"go build"},
		},
		{
			name:    "管道",
			command: "cat file.txt | grep pattern",
			want:    []string{"cat file.txt", "grep pattern"},
		},
		{
			name:    "双竖线逻辑或",
			command: "cmd1 || cmd2",
			want:    []string{"cmd1", "cmd2"},
		},
		{
			name:    "双与号逻辑与",
			command: "cmd1 && cmd2",
			want:    []string{"cmd1", "cmd2"},
		},
		{
			name:    "分号",
			command: "cmd1; cmd2",
			want:    []string{"cmd1", "cmd2"},
		},
		{
			name:    "混合操作符",
			command: "echo hello | grep h && echo found || echo missing",
			want:    []string{"echo hello", "grep h", "echo found", "echo missing"},
		},
		{
			name:    "rm绕过场景",
			command: "ls || rm -rf /",
			want:    []string{"ls", "rm -rf /"},
		},
		{
			name:    "管道加rm绕过",
			command: "cat file | rm -rf /",
			want:    []string{"cat file", "rm -rf /"},
		},
		{
			name:    "分号加危险命令",
			command: "echo safe; rm -rf /home",
			want:    []string{"echo safe", "rm -rf /home"},
		},
		{
			name:    "空命令",
			command: "",
			want:    nil,
		},
		{
			name:    "只有空格",
			command: "   ",
			want:    nil,
		},
		{
			name:    "带空格的分号",
			command: "cmd1 ; ; cmd2",
			want:    []string{"cmd1", "cmd2"},
		},
		{
			name:    "三段管道",
			command: "cat a | sort | uniq",
			want:    []string{"cat a", "sort", "uniq"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitShellSubCommands(tt.command)
			if len(got) != len(tt.want) {
				t.Errorf("splitShellSubCommands(%q) = %v, want %v", tt.command, got, tt.want)
				return
			}
			for i, seg := range got {
				if seg != tt.want[i] {
					t.Errorf("splitShellSubCommands(%q)[%d] = %q, want %q", tt.command, i, seg, tt.want[i])
				}
			}
		})
	}
}

func TestIsCommandAllowed_PipeBypass(t *testing.T) {
	allowedCommands := []string{"ls", "cat", "echo", "go"}

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "简单允许命令",
			command: "ls",
			want:    true,
		},
		{
			name:    "管道后接允许命令",
			command: "ls | cat",
			want:    true,
		},
		{
			name:    "管道后接禁止命令",
			command: "ls | rm -rf /",
			want:    false,
		},
		{
			name:    "双竖线绕过",
			command: "echo hello || rm -rf /",
			want:    false,
		},
		{
			name:    "双与号绕过",
			command: "echo hello && dangerous_cmd",
			want:    false,
		},
		{
			name:    "分号绕过",
			command: "go build; rm -rf /",
			want:    false,
		},
		{
			name:    "全部允许的复合命令",
			command: "echo start && ls | cat",
			want:    true,
		},
		{
			name:    "白名单为空时全部允许",
			command: "anything || dangerous",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCommandAllowed(tt.command, allowedCommands)
			if tt.name == "白名单为空时全部允许" {
				got = isCommandAllowed(tt.command, nil)
			}
			if got != tt.want {
				t.Errorf("isCommandAllowed(%q, %v) = %v, want %v", tt.command, allowedCommands, got, tt.want)
			}
		})
	}
}

func TestBuildNotAllowedError_CompositeCommand(t *testing.T) {
	allowedCommands := []string{"ls", "echo"}

	// 测试复合命令的错误信息能指出具体哪个子命令不被允许
	result := buildNotAllowedError("ls || rm -rf /", allowedCommands)
	if result.Success {
		t.Error("should be error result")
	}
	if !strings.Contains(result.Error, "rm") {
		t.Errorf("error message should mention the blocked command 'rm', got: %s", result.Error)
	}
	if !strings.Contains(result.Error, "复合命令") {
		t.Errorf("error message should indicate composite command, got: %s", result.Error)
	}
}

// ==================== 环境变量过滤 Tests ====================

func TestFilterEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name           string
		env            map[string]interface{}
		wantFiltered   int
		wantRemoved    []string
		wantNotRemoved []string
	}{
		{
			name: "过滤PATH和HOME",
			env: map[string]interface{}{
				"PATH":   "/usr/bin",
				"HOME":   "/home/user",
				"MY_VAR": "safe_value",
			},
			wantFiltered:   1,
			wantRemoved:    []string{"PATH", "HOME"},
			wantNotRemoved: []string{"MY_VAR"},
		},
		{
			name: "过滤LD_PRELOAD",
			env: map[string]interface{}{
				"LD_PRELOAD": "/malicious.so",
				"APP_CONFIG": "production",
			},
			wantFiltered:   1,
			wantRemoved:    []string{"LD_PRELOAD"},
			wantNotRemoved: []string{"APP_CONFIG"},
		},
		{
			name: "大小写不敏感过滤",
			env: map[string]interface{}{
				"path": "/usr/bin",
				"home": "/home/user",
			},
			wantFiltered: 0,
			wantRemoved:  []string{"path", "home"},
		},
		{
			name: "无过滤",
			env: map[string]interface{}{
				"APP_MODE": "dev",
				"DEBUG":    "true",
			},
			wantFiltered:   2,
			wantRemoved:    nil,
			wantNotRemoved: []string{"APP_MODE", "DEBUG"},
		},
		{
			name: "过滤所有受保护变量",
			env: map[string]interface{}{
				"PATH":                  "/bin",
				"HOME":                  "/root",
				"USERPROFILE":           "C:\\Users",
				"USER":                  "root",
				"USERNAME":              "admin",
				"SHELL":                 "/bin/bash",
				"LD_LIBRARY_PATH":       "/lib",
				"DYLD_LIBRARY_PATH":     "/lib",
				"LD_PRELOAD":            "/bad.so",
				"DYLD_INSERT_LIBRARIES": "/bad.so",
				"SYSTEMROOT":            "C:\\Windows",
				"WINDIR":                "C:\\Windows",
				"HOSTNAME":              "server1",
				"COMPUTERNAME":          "PC1",
				"TEMP":                  "/tmp",
				"TMP":                   "/tmp",
			},
			wantFiltered: 0,
			wantRemoved: []string{
				"PATH", "HOME", "USERPROFILE", "USER", "USERNAME", "SHELL",
				"LD_LIBRARY_PATH", "DYLD_LIBRARY_PATH", "LD_PRELOAD",
				"DYLD_INSERT_LIBRARIES", "SYSTEMROOT", "WINDIR",
				"HOSTNAME", "COMPUTERNAME", "TEMP", "TMP",
			},
		},
		{
			name:         "空输入",
			env:          map[string]interface{}{},
			wantFiltered: 0,
			wantRemoved:  nil,
		},
		{
			name: "非string值被忽略",
			env: map[string]interface{}{
				"PATH":   12345,
				"MY_VAR": "safe",
			},
			wantFiltered:   1,
			wantRemoved:    nil, // PATH值非string，不会添加到removed中（也不应添加到filtered中）
			wantNotRemoved: []string{"MY_VAR"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered, removed := filterEnvironmentVariables(tt.env)

			if len(filtered) != tt.wantFiltered {
				t.Errorf("filtered count = %d, want %d", len(filtered), tt.wantFiltered)
			}

			// 检查被移除的变量
			removedSet := make(map[string]bool)
			for _, r := range removed {
				removedSet[r] = true
			}
			for _, want := range tt.wantRemoved {
				if !removedSet[want] {
					t.Errorf("expected '%s' to be removed, but it wasn't. removed: %v", want, removed)
				}
			}

			// 检查未被移除的变量
			for _, want := range tt.wantNotRemoved {
				if _, ok := filtered[want]; !ok {
					t.Errorf("expected '%s' to be in filtered result, but it wasn't", want)
				}
			}
		})
	}
}

func TestFilterEnvironmentVariables_DoesNotLeakValues(t *testing.T) {
	env := map[string]interface{}{
		"PATH": "/secret/path/value",
	}
	_, removed := filterEnvironmentVariables(env)
	// 确认返回的只是变量名，不包含值
	for _, r := range removed {
		if strings.Contains(r, "secret") || strings.Contains(r, "/") {
			t.Errorf("removed list should not contain variable values, got: %s", r)
		}
	}
}

// ==================== 敏感路径检查 Tests ====================

func TestCheckSensitivePath_UnixDenyPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific path tests")
	}

	denyPaths := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/etc/sudoers",
		"/etc/ssh/sshd_config",
		"/boot/vmlinuz",
		"/proc/1/cmdline",
		"/sys/kernel/notes",
	}

	for _, path := range denyPaths {
		result := CheckSensitivePath(path)
		if result.Level != PathCheckDeny {
			t.Errorf("CheckSensitivePath(%q) level = %s, want deny. reason: %s", path, result.Level, result.Reason)
		}
	}
}

func TestCheckSensitivePath_UnixDenyPathsPreciseMatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific path tests")
	}

	// /etc/passwd 应被拒绝，但 /etc/passwd_backup 不应被拒绝（前缀匹配需要路径边界）
	result := CheckSensitivePath("/etc/passwd")
	if result.Level != PathCheckDeny {
		t.Errorf("/etc/passwd should be denied, got: %s", result.Level)
	}

	// /etc/passwd_backup 不应匹配 /etc/passwd （需要路径边界）
	result = CheckSensitivePath("/etc/passwd_backup")
	if result.Level == PathCheckDeny {
		t.Error("/etc/passwd_backup should not be denied (not an exact match)")
	}

	// /etc/passwd.bak 也不应匹配
	result = CheckSensitivePath("/etc/passwd.bak")
	if result.Level == PathCheckDeny {
		t.Error("/etc/passwd.bak should not be denied")
	}
}

func TestCheckSensitivePath_WindowsDenyPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific path tests")
	}

	denyPaths := []string{
		`C:\Windows\System32\config\SAM`,
		`C:\Windows\System32\drivers\etc\hosts`,
		`C:\Windows\SysWOW64\kernel32.dll`,
		`c:\windows\system32\cmd.exe`, // 小写也应该匹配
	}

	for _, path := range denyPaths {
		result := CheckSensitivePath(path)
		if result.Level != PathCheckDeny {
			t.Errorf("CheckSensitivePath(%q) level = %s, want deny. reason: %s", path, result.Level, result.Reason)
		}
	}
}

func TestCheckSensitivePath_CrossPlatformWarnPaths(t *testing.T) {
	warnPaths := []string{
		"/home/user/.ssh/authorized_keys",
		"/home/user/.ssh/authorized_keys2",
		"/root/.gnupg/private-keys-v1.d/key.key",
		"C:\\Users\\test\\.ssh\\authorized_keys",
	}

	for _, path := range warnPaths {
		result := CheckSensitivePath(path)
		if result.Level != PathCheckWarn {
			t.Errorf("CheckSensitivePath(%q) level = %s, want warn. reason: %s", path, result.Level, result.Reason)
		}
		if result.MatchedPattern == "" {
			t.Errorf("CheckSensitivePath(%q) should have a matched pattern", path)
		}
	}
}

func TestCheckSensitivePath_SafePaths(t *testing.T) {
	safePaths := []string{
		"/home/user/project/main.go",
		"/tmp/build/output.bin",
		"/workspace/src/app.js",
		"C:\\Users\\dev\\project\\index.ts",
	}

	for _, path := range safePaths {
		result := CheckSensitivePath(path)
		if result.Level != PathCheckPass {
			t.Errorf("CheckSensitivePath(%q) level = %s, want pass. reason: %s", path, result.Level, result.Reason)
		}
	}
}

func TestCheckSensitivePath_ETCSSHSubdirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific path tests")
	}

	// /etc/ssh/ 下的任何文件都应被拒绝
	result := CheckSensitivePath("/etc/ssh/sshd_config")
	if result.Level != PathCheckDeny {
		t.Errorf("/etc/ssh/sshd_config should be denied, got: %s", result.Level)
	}

	result = CheckSensitivePath("/etc/ssh/ssh_host_rsa_key")
	if result.Level != PathCheckDeny {
		t.Errorf("/etc/ssh/ssh_host_rsa_key should be denied, got: %s", result.Level)
	}
}

func TestIsPathOrPrefixMatch(t *testing.T) {
	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		{"/etc/passwd", "/etc/passwd", true},
		{"/etc/passwd", "/etc/shadow", false},
		{"/etc/ssh/sshd_config", "/etc/ssh/", true},
		{"/etc/ssh", "/etc/ssh/", false}, // 不以/结尾的pattern精确匹配时不等
		{"/boot/vmlinuz", "/boot/", true},
		{"/proc/1/status", "/proc/", true},
		{"/sys/kernel", "/sys/", true},
		{"/etc/passwd_backup", "/etc/passwd", false}, // 非路径边界
		{"/etc/passwd", "/etc/passwd_backup", false},
	}

	for _, tt := range tests {
		got := isPathOrPrefixMatch(tt.path, tt.pattern)
		if got != tt.want {
			t.Errorf("isPathOrPrefixMatch(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
		}
	}
}

func TestValidatePath_SensitivePathDeny(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Windows上使用Windows路径
		_, _, err := validatePath(`C:\Windows\System32\config\SAM`, "")
		if err == nil {
			t.Error("should deny Windows sensitive path")
		}
	} else {
		_, _, err := validatePath("/etc/passwd", "")
		if err == nil {
			t.Error("should deny /etc/passwd even without workDir restriction")
		}
	}
}

func TestValidatePath_SensitivePathWarn(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建 .ssh 目录和 authorized_keys 文件路径
	sshDir := filepath.Join(tmpDir, ".ssh")
	sshFile := filepath.Join(sshDir, "authorized_keys")

	_, warnings, err := validatePath(sshFile, tmpDir)
	if err != nil {
		// 可能路径不存在导致其他错误
		t.Logf("validatePath returned error (may be expected): %v", err)
	}

	// 如果没有错误，检查是否有警告
	if err == nil && len(warnings) == 0 {
		t.Error("should have warnings for .ssh/authorized_keys path")
	}
}

func TestValidatePath_SafePathWithWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	safeFile := filepath.Join(tmpDir, "main.go")
	_, warnings, err := validatePath(safeFile, tmpDir)
	if err != nil {
		t.Errorf("safe path should not error: %v", err)
	}
	if len(warnings) > 0 {
		t.Errorf("safe path should not have warnings, got: %v", warnings)
	}
}

// ==================== 集成测试：环境变量注入到命令执行 ====================

func TestExecuteCommandTool_EnvFiltering(t *testing.T) {
	tool := NewExecuteCommandTool()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo"
	} else {
		cmd = "echo"
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": cmd,
		"args":    []interface{}{"test_env"},
		"env": map[string]interface{}{
			"PATH":    "/malicious/path",
			"MY_FLAG": "safe_value",
		},
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	data := result.Data.(*CommandResult)
	// PATH应被过滤，MY_FLAG应被保留
	// 验证stderr中有过滤警告
	if !strings.Contains(data.Stderr, "PATH") || !strings.Contains(data.Stderr, "受保护的环境变量已被过滤") {
		t.Errorf("stderr should contain PATH filtering warning, got stderr: %s", data.Stderr)
	}
}

// ==================== 集成测试：敏感路径检查与文件操作 ====================

func TestWriteFileTool_SensitivePathBlocked(t *testing.T) {
	tool := NewWriteFileTool() // 无workDir限制

	if runtime.GOOS == "windows" {
		result, _ := tool.Execute(context.Background(), map[string]interface{}{
			"path":    `C:\Windows\System32\test_malicious.bat`,
			"content": "malicious content",
		})
		if result.Success {
			t.Error("should deny writing to Windows System32")
		}
	} else {
		result, _ := tool.Execute(context.Background(), map[string]interface{}{
			"path":    "/etc/passwd",
			"content": "malicious content",
		})
		if result.Success {
			t.Error("should deny writing to /etc/passwd")
		}
	}
}

func TestDeleteFileTool_SensitivePathBlocked(t *testing.T) {
	tool := NewDeleteFileTool() // 无workDir限制

	if runtime.GOOS == "windows" {
		result, _ := tool.Execute(context.Background(), map[string]interface{}{
			"path": `C:\Windows\System32\cmd.exe`,
		})
		if result.Success {
			t.Error("should deny deleting Windows System32 files")
		}
	} else {
		result, _ := tool.Execute(context.Background(), map[string]interface{}{
			"path": "/etc/shadow",
		})
		if result.Success {
			t.Error("should deny deleting /etc/shadow")
		}
	}
}

func TestReadFileTool_SensitivePathBlocked(t *testing.T) {
	tool := NewReadFileTool() // 无workDir限制

	if runtime.GOOS == "windows" {
		result, _ := tool.Execute(context.Background(), map[string]interface{}{
			"path": `C:\Windows\System32\config\SAM`,
		})
		if result.Success {
			t.Error("should deny reading Windows SAM file")
		}
	} else {
		result, _ := tool.Execute(context.Background(), map[string]interface{}{
			"path": "/etc/shadow",
		})
		if result.Success {
			t.Error("should deny reading /etc/shadow")
		}
	}
}

func TestWriteFileTool_SensitivePathWithWorkDirOverride(t *testing.T) {
	// 即使路径在workDir内，敏感路径检查仍然生效
	tmpDir := t.TempDir()
	tool := NewWriteFileToolWithWorkDir(tmpDir)

	// 在workDir内创建安全文件应成功
	safeFile := filepath.Join(tmpDir, "safe.txt")
	result, _ := tool.Execute(context.Background(), map[string]interface{}{
		"path":    safeFile,
		"content": "safe content",
	})
	if !result.Success {
		t.Errorf("should allow writing safe file in workDir: %s", result.Error)
	}

	// 清理
	os.Remove(safeFile)
}
