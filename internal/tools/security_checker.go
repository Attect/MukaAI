package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SecurityVerdict 安全审查结果
type SecurityVerdict string

const (
	SecurityAllow   SecurityVerdict = "allow"   // 允许执行
	SecurityDeny    SecurityVerdict = "deny"    // 拒绝执行
	SecurityConfirm SecurityVerdict = "confirm" // 需要用户确认
)

// SecurityCheckResult 安全检查结果
type SecurityCheckResult struct {
	Verdict    SecurityVerdict `json:"verdict"`
	Reason     string          `json:"reason"`
	RiskLevel  string          `json:"risk_level"` // low, medium, high, critical
	Suggestion string          `json:"suggestion,omitempty"`
}

// CommandSecurityChecker 命令安全审查器
// 扩展白名单自动包含常见构建/运行/系统命令，
// 非白名单命令通过安全审查器检查后自动放行或拒绝
type CommandSecurityChecker struct {
	workDir         string                            // 工作目录
	expandedAllow   []string                          // 扩展白名单
	userApproveFunc func(command, reason string) bool // 用户确认回调（可选）
}

// NewCommandSecurityChecker 创建命令安全审查器
// workDir: 工作目录，用于路径安全检查
// baseAllowCommands: 用户配置的基础白名单
func NewCommandSecurityChecker(workDir string, baseAllowCommands []string) *CommandSecurityChecker {
	c := &CommandSecurityChecker{
		workDir: workDir,
	}
	c.expandedAllow = c.buildExpandedAllowList(baseAllowCommands)
	return c
}

// SetUserApproveFunc 设置用户确认回调
// 当命令需要确认时（SecurityConfirm），通过此回调询问用户
func (c *CommandSecurityChecker) SetUserApproveFunc(fn func(command, reason string) bool) {
	c.userApproveFunc = fn
}

// GetUserApproveFunc 获取用户确认回调
func (c *CommandSecurityChecker) GetUserApproveFunc() func(command, reason string) bool {
	return c.userApproveFunc
}

// buildExpandedAllowList 构建扩展白名单
// 包含：用户配置的命令 + 常见构建/运行/系统命令
func (c *CommandSecurityChecker) buildExpandedAllowList(baseAllow []string) []string {
	// 常见构建/运行命令
	commonBuild := []string{
		// Go
		"go",
		// C/C++
		"gcc", "g++", "cc", "c++", "clang", "clang++", "make", "cmake", "ninja",
		// Rust
		"cargo", "rustc",
		// Java/Kotlin
		"java", "javac", "kotlinc", "kotlin", "gradle", "mvn", "mvnw",
		// Node/JS/TS
		"node", "npm", "npx", "yarn", "pnpm", "tsc", "ts-node", "deno", "bun",
		// Python
		"python", "python3", "pip", "pip3", "poetry", "uv", "conda",
		// Ruby
		"ruby", "gem", "bundle",
		// C#
		"dotnet", "msbuild",
		// 系统工具
		"ls", "dir", "cat", "head", "tail", "wc", "sort", "uniq", "grep", "find",
		"echo", "pwd", "whoami", "hostname", "uname", "date", "env", "printenv",
		"which", "where", "type",
		"mkdir", "touch", "cp", "mv", "chmod", "chown",
		"tar", "zip", "unzip", "gzip", "gunzip",
		"curl", "wget",
		"git",
		"rm", // rm在安全检查中会验证路径
		"sed", "awk", "tr", "cut", "xargs",
		"diff", "patch",
		"docker", "podman",
		"test", "[",
		"sh", "bash", "zsh", "fish",
		"exit", "true", "false",
		// Windows特定
		"cmd", "powershell", "type", "copy", "del", "ren",
	}

	// 合并去重
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, cmd := range baseAllow {
		lower := strings.ToLower(cmd)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, lower)
		}
	}

	for _, cmd := range commonBuild {
		lower := strings.ToLower(cmd)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, lower)
		}
	}

	return result
}

// Check 检查命令是否安全
// command: 命令名称或完整命令字符串
// args: 命令参数列表
func (c *CommandSecurityChecker) Check(command string, args []string) *SecurityCheckResult {
	baseCmd := strings.ToLower(extractBaseCommand(command))

	// 1. 白名单命令：直接放行，但仍做路径检查
	if c.isInAllowList(baseCmd) {
		return c.checkWhitelistedCommand(baseCmd, command, args)
	}

	// 2. 非白名单命令：安全审查
	return c.checkNonWhitelistedCommand(command, args)
}

// isInAllowList 检查是否在扩展白名单中
func (c *CommandSecurityChecker) isInAllowList(cmd string) bool {
	for _, allowed := range c.expandedAllow {
		if allowed == cmd {
			return true
		}
	}
	return false
}

// checkWhitelistedCommand 白名单命令的额外安全检查
// 主要是rm等危险命令的路径检查，以及curl/wget下载到系统目录的检查
func (c *CommandSecurityChecker) checkWhitelistedCommand(baseCmd, command string, args []string) *SecurityCheckResult {
	// rm命令：检查是否删除工作区外的文件
	if baseCmd == "rm" {
		return c.checkRmSafety(command, args)
	}

	// curl/wget命令：检查是否下载到系统目录
	if baseCmd == "curl" || baseCmd == "wget" {
		return c.checkDownloadSafety(baseCmd, command, args)
	}

	// cat命令：检查是否访问敏感文件
	if baseCmd == "cat" || baseCmd == "head" || baseCmd == "tail" {
		return c.readFileSafety(command, args)
	}

	// 其他白名单命令直接放行
	return &SecurityCheckResult{
		Verdict:   SecurityAllow,
		Reason:    "命令在白名单中",
		RiskLevel: "low",
	}
}

// checkRmSafety 检查rm命令安全性
func (c *CommandSecurityChecker) checkRmSafety(command string, args []string) *SecurityCheckResult {
	hasRecursive := false
	hasForce := false
	targets := make([]string, 0)

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			if strings.Contains(arg, "r") || strings.Contains(arg, "R") {
				hasRecursive = true
			}
			if strings.Contains(arg, "f") {
				hasForce = true
			}
		} else {
			targets = append(targets, arg)
		}
	}

	// rm -rf / 或 rm -rf ~ 等全局删除 → 直接拒绝
	// 使用字符串匹配而非路径操作，以跨平台兼容
	for _, target := range targets {
		// 检测危险目标路径（跨平台）
		dangerousTargets := []string{"/", "~", "\\"}
		for _, dt := range dangerousTargets {
			cleanTarget := strings.TrimRight(target, "/")
			cleanTarget = strings.TrimRight(cleanTarget, "\\")
			if cleanTarget == dt {
				return &SecurityCheckResult{
					Verdict:    SecurityDeny,
					Reason:     fmt.Sprintf("rm 目标 '%s' 极其危险，已拒绝", target),
					RiskLevel:  "critical",
					Suggestion: "请勿删除根目录或用户主目录",
				}
			}
		}
	}

	// 检查目标路径是否在工作区外
	for _, target := range targets {
		absTarget := target

		// 判断是否为绝对路径
		if filepath.IsAbs(target) {
			absTarget = target
		} else if c.workDir != "" {
			absTarget = filepath.Join(c.workDir, target)
		} else {
			// workDir为空，跳过路径检查
			continue
		}

		// 如果workDir为空，跳过路径检查
		if c.workDir == "" {
			continue
		}

		// 检查是否在工作区外
		// 使用卷匹配和..路径检查双重验证
		rel, err := filepath.Rel(c.workDir, absTarget)
		if err != nil || strings.HasPrefix(rel, "..") {
			return &SecurityCheckResult{
				Verdict:    SecurityConfirm,
				Reason:     fmt.Sprintf("rm命令目标 '%s' 在工作区外，需要确认", target),
				RiskLevel:  "medium",
				Suggestion: "建议只删除工作区内的文件",
			}
		}
	}

	// rm -rf 无明确目标 → 拒绝
	if hasRecursive && hasForce && len(targets) == 0 {
		return &SecurityCheckResult{
			Verdict:   SecurityDeny,
			Reason:    "rm -rf 无明确目标，极其危险",
			RiskLevel: "critical",
		}
	}

	return &SecurityCheckResult{
		Verdict:   SecurityAllow,
		Reason:    "rm命令目标在工作区内",
		RiskLevel: "low",
	}
}

// checkDownloadSafety 检查curl/wget命令下载目标路径
func (c *CommandSecurityChecker) checkDownloadSafety(baseCmd, command string, args []string) *SecurityCheckResult {
	fullCmd := command
	if len(args) > 0 {
		fullCmd = command + " " + strings.Join(args, " ")
	}
	fullCmdLower := strings.ToLower(fullCmd)

	// 检查下载到系统目录
	systemDirs := []string{"/etc/", "/usr/", "/boot/", "c:\\windows\\", "c:\\system32\\"}
	for _, sysDir := range systemDirs {
		if strings.Contains(fullCmdLower, sysDir) {
			return &SecurityCheckResult{
				Verdict:    SecurityConfirm,
				Reason:     fmt.Sprintf("%s 下载目标涉及系统目录 %s", baseCmd, sysDir),
				RiskLevel:  "high",
				Suggestion: "如果是在安装开发环境依赖，请确认后执行",
			}
		}
	}

	// 检查通过POST传输敏感信息
	dangerousPatterns := []struct {
		pattern string
		reason  string
	}{
		{"secret", "可能传输密钥到远程"},
		{"token", "可能传输令牌到远程"},
		{"password", "可能传输密码到远程"},
	}
	for _, dp := range dangerousPatterns {
		if strings.Contains(fullCmdLower, "-d") && strings.Contains(fullCmdLower, dp.pattern) {
			return &SecurityCheckResult{
				Verdict:    SecurityDeny,
				Reason:     dp.reason,
				RiskLevel:  "critical",
				Suggestion: "此操作存在安全风险，已被阻止",
			}
		}
	}

	return &SecurityCheckResult{
		Verdict:   SecurityAllow,
		Reason:    fmt.Sprintf("%s 命令在白名单中", baseCmd),
		RiskLevel: "low",
	}
}

// readFileSafety 检查文件读取命令是否访问敏感文件
func (c *CommandSecurityChecker) readFileSafety(command string, args []string) *SecurityCheckResult {
	sensitiveFiles := []string{"/etc/shadow", "/etc/passwd", ".ssh/", ".gnupg/", "id_rsa", "id_ed25519"}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		argLower := strings.ToLower(arg)
		for _, sf := range sensitiveFiles {
			if strings.Contains(argLower, sf) {
				return &SecurityCheckResult{
					Verdict:    SecurityConfirm,
					Reason:     fmt.Sprintf("命令涉及敏感文件/目录: %s", sf),
					RiskLevel:  "high",
					Suggestion: "请确认此操作的必要性",
				}
			}
		}
	}

	return &SecurityCheckResult{
		Verdict:   SecurityAllow,
		Reason:    "命令在白名单中",
		RiskLevel: "low",
	}
}

// checkNonWhitelistedCommand 非白名单命令的安全审查
func (c *CommandSecurityChecker) checkNonWhitelistedCommand(command string, args []string) *SecurityCheckResult {
	fullCmd := command
	if len(args) > 0 {
		fullCmd = command + " " + strings.Join(args, " ")
	}
	fullCmdLower := strings.ToLower(fullCmd)

	// === 危险模式检测 ===

	// 1. 检查是否有删除工作区外文件的行为
	dangerousPatterns := []struct {
		pattern  string
		reason   string
		severity string
	}{
		// 删除危险操作
		{`rm -rf /`, "尝试删除根目录", "critical"},
		{`rm -rf ~`, "尝试删除用户主目录", "critical"},
		{`del /s c:\`, "尝试删除C盘文件", "critical"},
		{`format `, "尝试格式化磁盘", "critical"},

		// 密钥/凭证泄露
		{`curl.*-d.*secret`, "可能传输密钥到远程", "critical"},
		{`curl.*-d.*token`, "可能传输令牌到远程", "critical"},
		{`curl.*-d.*password`, "可能传输密码到远程", "critical"},
		{`wget.*--post-data.*key`, "可能通过POST传输密钥", "critical"},

		// 远程操控
		{`nc -l`, "启动netcat监听，可能开放后门", "critical"},
		{`ncat -l`, "启动ncat监听，可能开放后门", "critical"},
		{`socat tcp-listen`, "启动TCP监听，可能开放后门", "critical"},

		// 安全防护降低
		{`iptables -f`, "清空防火墙规则", "high"},
		{`setenforce 0`, "关闭SELinux", "high"},
		{`ufw disable`, "关闭UFW防火墙", "high"},
		{`systemctl stop firewall`, "停止防火墙服务", "high"},

		// 可疑的网络行为
		{`curl -o /etc/`, "下载文件到系统目录", "high"},
		{`wget -o /etc/`, "下载文件到系统目录", "high"},

		// SSH相关
		{`ssh -r`, "SSH反向隧道，可能暴露内网", "high"},
	}

	for _, dp := range dangerousPatterns {
		if strings.Contains(fullCmdLower, dp.pattern) ||
			simplePatternMatch(dp.pattern, fullCmdLower) {
			return &SecurityCheckResult{
				Verdict:    SecurityDeny,
				Reason:     dp.reason,
				RiskLevel:  dp.severity,
				Suggestion: "此操作存在安全风险，已被阻止",
			}
		}
	}

	// 2. 检查是否尝试写入系统目录
	systemDirs := []string{"/etc/", "/usr/", "/boot/", "c:\\windows\\", "c:\\system32\\"}
	for _, sysDir := range systemDirs {
		if strings.Contains(fullCmdLower, sysDir) {
			return &SecurityCheckResult{
				Verdict:    SecurityConfirm,
				Reason:     fmt.Sprintf("命令涉及系统目录 %s", sysDir),
				RiskLevel:  "high",
				Suggestion: "如果是在安装开发环境依赖，请确认后执行",
			}
		}
	}

	// 3. 检查是否尝试访问敏感文件
	sensitiveFiles := []string{"/etc/shadow", "/etc/passwd", ".ssh/", ".gnupg/", ".env", "id_rsa", "id_ed25519"}
	for _, sf := range sensitiveFiles {
		if strings.Contains(fullCmdLower, sf) {
			return &SecurityCheckResult{
				Verdict:    SecurityConfirm,
				Reason:     fmt.Sprintf("命令涉及敏感文件/目录: %s", sf),
				RiskLevel:  "high",
				Suggestion: "请确认此操作的必要性",
			}
		}
	}

	// 4. 默认：需要用户确认（低风险）
	return &SecurityCheckResult{
		Verdict:    SecurityConfirm,
		Reason:     fmt.Sprintf("命令 '%s' 不在白名单中，需要确认", extractBaseCommand(command)),
		RiskLevel:  "medium",
		Suggestion: "如果是编译构建或运行命令，建议添加到白名单配置中",
	}
}

// simplePatternMatch 简单的模式匹配（支持 .* 通配符）
func simplePatternMatch(pattern, text string) bool {
	parts := strings.Split(pattern, ".*")
	if len(parts) == 1 {
		return strings.Contains(text, pattern)
	}

	idx := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		pos := strings.Index(text[idx:], part)
		if pos == -1 {
			return false
		}
		// 第一部分必须在文本开头附近
		if i == 0 && pos > 10 {
			return false
		}
		idx += pos + len(part)
	}
	return true
}

// IsCommandInExpandedAllowList 检查命令是否在扩展白名单中（公开方法）
func (c *CommandSecurityChecker) IsCommandInExpandedAllowList(command string) bool {
	baseCmd := strings.ToLower(extractBaseCommand(command))
	return c.isInAllowList(baseCmd)
}

// GetExpandedAllowList 获取扩展白名单列表
func (c *CommandSecurityChecker) GetExpandedAllowList() []string {
	return c.expandedAllow
}
