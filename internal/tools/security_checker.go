package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Attect/MukaAI/internal/model"
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

// SecurityEvaluator 安全评估器接口
// 用于评估非白名单命令的安全性，可由LLM或其他策略实现
type SecurityEvaluator interface {
	// Evaluate 评估命令是否安全执行
	// command: 命令名称或完整命令字符串
	// args: 命令参数列表
	// 返回评估结果，nil表示无法评估（调用方决定后续行为）
	Evaluate(command string, args []string) *SecurityCheckResult
}

// ModelCaller 模型调用接口，用于安全评估时调用LLM
// 提取接口使 SecurityAgentEvaluator 可测试，无需依赖具体的 *model.Client
type ModelCaller interface {
	ChatCompletion(ctx context.Context, messages []model.Message, tools []model.Tool) (*model.ChatCompletionResponse, error)
	ChatCompletionWithRetry(ctx context.Context, messages []model.Message, tools []model.Tool, retryConfig *model.RetryConfig) (*model.ChatCompletionResponse, error)
}

// SecurityAgentEvaluator 基于LLM的安全评估器
// 通过调用大语言模型判断命令安全性
type SecurityAgentEvaluator struct {
	modelClient ModelCaller                     // LLM客户端（接口类型，可mock）
	cache       map[string]*SecurityCheckResult // 评估结果缓存
	mu          sync.RWMutex                    // 缓存并发保护
}

// NewSecurityAgentEvaluator 创建基于LLM的安全评估器
// modelClient: LLM客户端（ModelCaller接口），传入nil则Evaluate始终返回nil（无法评估）
// *model.Client 天然满足 ModelCaller 接口，可直接传入
func NewSecurityAgentEvaluator(modelClient ModelCaller) *SecurityAgentEvaluator {
	return &SecurityAgentEvaluator{
		modelClient: modelClient,
		cache:       make(map[string]*SecurityCheckResult),
	}
}

// llmSecurityResponse LLM安全评估的JSON响应格式
type llmSecurityResponse struct {
	Verdict   string `json:"verdict"`    // "allow" 或 "deny"
	Reason    string `json:"reason"`     // 判断理由
	RiskLevel string `json:"risk_level"` // "low", "medium", "high"
}

// Evaluate 使用LLM评估命令安全性
// 返回评估结果；如果modelClient为nil、评估失败或超时，返回nil
func (e *SecurityAgentEvaluator) Evaluate(command string, args []string) *SecurityCheckResult {
	// 构建完整命令字符串用于缓存key
	fullCmd := command
	if len(args) > 0 {
		fullCmd = command + " " + strings.Join(args, " ")
	}
	cacheKey := strings.ToLower(fullCmd)

	// 检查缓存（即使modelClient为nil也可以命中缓存）
	e.mu.RLock()
	if cached, ok := e.cache[cacheKey]; ok {
		e.mu.RUnlock()
		return cached
	}
	e.mu.RUnlock()

	// 无模型客户端时无法评估
	if e.modelClient == nil {
		return nil
	}

	// 构建评估prompt
	fullCmdDisplay := command
	if len(args) > 0 {
		fullCmdDisplay = command + " " + strings.Join(args, " ")
	}

	systemPrompt := `You are a command security evaluator for an AI coding assistant.
Evaluate whether the following command is safe to execute automatically without human confirmation.

Rules:
1. Commands that only READ files or show system info are SAFE.
2. Commands that modify files WITHIN the user's workspace/project are SAFE.
3. Commands that install development dependencies (npm, pip, cargo, etc.) are SAFE.
4. Commands that run tests, build tools, or development servers are SAFE.
5. Commands that download from untrusted sources or execute remote code are UNSAFE.
6. Commands that modify system configuration or security settings are UNSAFE.
7. Commands that access or exfiltrate sensitive data (passwords, keys, tokens) are UNSAFE.
8. Commands that open network listeners or tunnels are UNSAFE.

Respond ONLY with valid JSON in this exact format (no markdown, no extra text):
{"verdict": "allow", "reason": "brief explanation", "risk_level": "low"}
{"verdict": "deny", "reason": "brief explanation", "risk_level": "high"}`

	userPrompt := fmt.Sprintf("Evaluate this command: %s", fullCmdDisplay)

	// 带超时和重试调用 LLM（60 秒超时，fail-open 避免阻塞）
	// 调整超时时间以适配本地模型服务的响应特性
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 使用带重试的请求
	retryConfig := &model.RetryConfig{
		MaxRetries:      3,
		InitialDelay:    2 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffFactor:   2.0,
		RetryableErrors: []string{"connection refused", "connection reset", "timeout", "deadline exceeded", "EOF"},
	}
	resp, err := e.modelClient.ChatCompletionWithRetry(ctx, []model.Message{
		model.NewSystemMessage(systemPrompt),
		model.NewUserMessage(userPrompt),
	}, nil, retryConfig)
	if err != nil {
		// LLM调用失败，记录日志，返回nil让调用方使用默认行为（fail-open）
		log.Printf("[SecurityEvaluator] LLM评估失败(fail-open): %v", err)
		return nil
	}

	if len(resp.Choices) == 0 {
		log.Printf("[SecurityEvaluator] LLM返回空响应(fail-open)")
		return nil
	}

	// 解析LLM响应
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	// 清理可能的markdown代码块包裹
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var llmResp llmSecurityResponse
	if err := json.Unmarshal([]byte(content), &llmResp); err != nil {
		// JSON解析失败，fail-open
		log.Printf("[SecurityEvaluator] LLM响应解析失败(fail-open): %s, raw: %s", err, content)
		return nil
	}

	// 构建结果
	var result *SecurityCheckResult
	switch strings.ToLower(llmResp.Verdict) {
	case "allow":
		result = &SecurityCheckResult{
			Verdict:   SecurityAllow,
			Reason:    fmt.Sprintf("LLM评估通过: %s", llmResp.Reason),
			RiskLevel: defaultRiskLevel(llmResp.RiskLevel, "low"),
		}
	case "deny":
		result = &SecurityCheckResult{
			Verdict:    SecurityDeny,
			Reason:     fmt.Sprintf("LLM评估拒绝: %s", llmResp.Reason),
			RiskLevel:  defaultRiskLevel(llmResp.RiskLevel, "high"),
			Suggestion: "如果此命令是必要的，请手动确认后执行",
		}
	default:
		// 无法识别的verdict，fail-open
		log.Printf("[SecurityEvaluator] LLM返回未知verdict(fail-open): %s", llmResp.Verdict)
		return nil
	}

	// 缓存结果
	e.mu.Lock()
	e.cache[cacheKey] = result
	e.mu.Unlock()

	return result
}

// defaultRiskLevel 返回有效的风险等级，无效时返回defaultValue
func defaultRiskLevel(level, defaultValue string) string {
	switch strings.ToLower(level) {
	case "low", "medium", "high", "critical":
		return strings.ToLower(level)
	default:
		return defaultValue
	}
}

// CommandSecurityChecker 命令安全审查器
// 扩展白名单自动包含常见构建/运行/系统命令，
// 非白名单命令通过动态白名单、危险模式检测、安全评估器逐级判断
type CommandSecurityChecker struct {
	workDir         string                            // 工作目录
	expandedAllow   []string                          // 静态白名单（扩展后）
	dynamicAllow    map[string]bool                   // 动态白名单（内存缓存）
	dynamicAllowMu  sync.RWMutex                      // 动态白名单并发保护
	stateDir        string                            // 持久化目录
	userApproveFunc func(command, reason string) bool // 用户确认回调（可选）
	evaluator       SecurityEvaluator                 // 安全评估器（可选）
}

// NewCommandSecurityChecker 创建命令安全审查器
// workDir: 工作目录，用于路径安全检查
// baseAllowCommands: 用户配置的基础白名单
func NewCommandSecurityChecker(workDir string, baseAllowCommands []string) *CommandSecurityChecker {
	c := &CommandSecurityChecker{
		workDir:      workDir,
		dynamicAllow: make(map[string]bool),
	}
	c.expandedAllow = c.buildExpandedAllowList(baseAllowCommands)
	return c
}

// NewCommandSecurityCheckerWithState 创建带持久化的命令安全审查器
// workDir: 工作目录
// baseAllowCommands: 用户配置的基础白名单
// stateDir: 持久化目录，动态白名单JSON文件存储于此
func NewCommandSecurityCheckerWithState(workDir string, baseAllowCommands []string, stateDir string) *CommandSecurityChecker {
	c := &CommandSecurityChecker{
		workDir:      workDir,
		dynamicAllow: make(map[string]bool),
		stateDir:     stateDir,
	}
	c.expandedAllow = c.buildExpandedAllowList(baseAllowCommands)
	c.loadDynamicAllowList()
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

// SetEvaluator 设置安全评估器
// 注入后，非白名单且非危险的命令将使用评估器判断安全性
func (c *CommandSecurityChecker) SetEvaluator(evaluator SecurityEvaluator) {
	c.evaluator = evaluator
}

// GetEvaluator 获取当前安全评估器
func (c *CommandSecurityChecker) GetEvaluator() SecurityEvaluator {
	return c.evaluator
}

// SetWorkDir 设置工作目录（用于路径安全检查）
// 当用户切换工作目录时调用，更新审查器的路径校验基准
func (c *CommandSecurityChecker) SetWorkDir(workDir string) {
	c.workDir = workDir
}

// === 动态白名单管理 ===

// AddToDynamicAllowList 添加命令到动态白名单并持久化
func (c *CommandSecurityChecker) AddToDynamicAllowList(command string) {
	key := strings.ToLower(extractBaseCommand(command))
	c.dynamicAllowMu.Lock()
	c.dynamicAllow[key] = true
	c.dynamicAllowMu.Unlock()
	c.saveDynamicAllowList()
}

// RemoveFromDynamicAllowList 从动态白名单移除命令并持久化
func (c *CommandSecurityChecker) RemoveFromDynamicAllowList(command string) {
	key := strings.ToLower(extractBaseCommand(command))
	c.dynamicAllowMu.Lock()
	delete(c.dynamicAllow, key)
	c.dynamicAllowMu.Unlock()
	c.saveDynamicAllowList()
}

// GetDynamicAllowList 获取当前动态白名单（返回排序后的副本）
func (c *CommandSecurityChecker) GetDynamicAllowList() []string {
	c.dynamicAllowMu.RLock()
	defer c.dynamicAllowMu.RUnlock()

	result := make([]string, 0, len(c.dynamicAllow))
	for cmd := range c.dynamicAllow {
		result = append(result, cmd)
	}
	return result
}

// isInDynamicAllowList 检查命令是否在动态白名单中
func (c *CommandSecurityChecker) isInDynamicAllowList(baseCmd string) bool {
	c.dynamicAllowMu.RLock()
	defer c.dynamicAllowMu.RUnlock()
	return c.dynamicAllow[baseCmd]
}

// loadDynamicAllowList 从文件加载动态白名单
func (c *CommandSecurityChecker) loadDynamicAllowList() {
	if c.stateDir == "" {
		return
	}

	filePath := filepath.Join(c.stateDir, "dynamic_allowlist.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		// 文件不存在或读取失败，使用空白名单
		return
	}

	var commands []string
	if err := json.Unmarshal(data, &commands); err != nil {
		log.Printf("[SecurityChecker] 动态白名单文件解析失败: %v", err)
		return
	}

	c.dynamicAllowMu.Lock()
	for _, cmd := range commands {
		c.dynamicAllow[strings.ToLower(strings.TrimSpace(cmd))] = true
	}
	c.dynamicAllowMu.Unlock()

	log.Printf("[SecurityChecker] 已加载 %d 条动态白名单记录", len(commands))
}

// saveDynamicAllowList 持久化动态白名单到文件
func (c *CommandSecurityChecker) saveDynamicAllowList() {
	if c.stateDir == "" {
		return
	}

	c.dynamicAllowMu.RLock()
	commands := make([]string, 0, len(c.dynamicAllow))
	for cmd := range c.dynamicAllow {
		commands = append(commands, cmd)
	}
	c.dynamicAllowMu.RUnlock()

	// 确保目录存在
	if err := os.MkdirAll(c.stateDir, 0755); err != nil {
		log.Printf("[SecurityChecker] 创建持久化目录失败: %v", err)
		return
	}

	data, err := json.MarshalIndent(commands, "", "  ")
	if err != nil {
		log.Printf("[SecurityChecker] 序列化动态白名单失败: %v", err)
		return
	}

	filePath := filepath.Join(c.stateDir, "dynamic_allowlist.json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		log.Printf("[SecurityChecker] 写入动态白名单文件失败: %v", err)
	}
}

// === 核心检查流程 ===

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
// 执行流程：静态白名单 → 动态白名单 → 危险模式检测 → 安全评估器(LLM)
// command: 命令名称或完整命令字符串
// args: 命令参数列表
func (c *CommandSecurityChecker) Check(command string, args []string) *SecurityCheckResult {
	baseCmd := strings.ToLower(extractBaseCommand(command))

	// 1. 静态白名单 → 直接放行（快速通道）
	if c.isInAllowList(baseCmd) {
		return c.checkWhitelistedCommand(baseCmd, command, args)
	}

	// 2. 动态白名单 → 直接放行（仍做基本路径检查）
	if c.isInDynamicAllowList(baseCmd) {
		return c.checkWhitelistedCommand(baseCmd, command, args)
	}

	// 3. 快速危险模式检测 → 拒绝
	if result := c.checkDangerousPatterns(command, args); result != nil {
		return result
	}

	// 4. 安全评估器（LLM）评估
	return c.evaluateWithSecurityAgent(command, args)
}

// isInAllowList 检查是否在静态扩展白名单中
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

// checkDangerousPatterns 快速危险模式检测
// 从checkNonWhitelistedCommand中提取的纯危险模式匹配逻辑
// 命中危险模式 → 返回 deny 结果
// 未命中 → 返回 nil（继续后续评估）
func (c *CommandSecurityChecker) checkDangerousPatterns(command string, args []string) *SecurityCheckResult {
	fullCmd := command
	if len(args) > 0 {
		fullCmd = command + " " + strings.Join(args, " ")
	}
	fullCmdLower := strings.ToLower(fullCmd)

	// === 危险模式检测 ===

	// 1. 危险命令模式
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

	// 未命中任何危险模式
	return nil
}

// evaluateWithSecurityAgent 使用安全评估器评估命令
// 有评估器 → 使用LLM评估，通过后自动加入动态白名单
// 无评估器 → 默认放行（向后兼容 default_policy=allow 的行为）
func (c *CommandSecurityChecker) evaluateWithSecurityAgent(command string, args []string) *SecurityCheckResult {
	// 有安全评估器时，使用LLM评估
	if c.evaluator != nil {
		result := c.evaluator.Evaluate(command, args)
		if result != nil {
			// LLM评估通过，自动加入动态白名单
			if result.Verdict == SecurityAllow {
				baseCmd := strings.ToLower(extractBaseCommand(command))
				c.AddToDynamicAllowList(baseCmd)
			}
			return result
		}
		// LLM评估失败（超时/错误），fail-open：默认放行
		return &SecurityCheckResult{
			Verdict:   SecurityAllow,
			Reason:    "安全评估器无法评估，默认放行",
			RiskLevel: "medium",
		}
	}

	// 无评估器时，默认放行（向后兼容）
	return &SecurityCheckResult{
		Verdict:   SecurityAllow,
		Reason:    fmt.Sprintf("命令 '%s' 不在白名单中，但无安全评估器，默认放行", extractBaseCommand(command)),
		RiskLevel: "medium",
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

// checkNonWhitelistedCommand 非白名单命令的安全审查（保留向后兼容）
// 内部使用 checkDangerousPatterns 实现
func (c *CommandSecurityChecker) checkNonWhitelistedCommand(command string, args []string) *SecurityCheckResult {
	// 危险模式检测
	if result := c.checkDangerousPatterns(command, args); result != nil {
		return result
	}

	// 非危险命令：走评估器流程
	return c.evaluateWithSecurityAgent(command, args)
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
