package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// 默认命令执行超时时间
const DefaultCommandTimeout = 60 * time.Second

// ==================== 命令白名单校验 ====================

// extractBaseCommand 从命令字符串中提取基础命令名称
// 例如: "go build -o app ./cmd" → "go"
// 例如: "git commit -m 'test'" → "git"
// 在Windows上会处理路径，提取文件名部分
func extractBaseCommand(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}

	// 取第一个空格前的部分作为命令
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}

	base := parts[0]

	// 处理路径情况：提取文件名
	// 例如 "/usr/bin/go" → "go", "C:\Tools\git.exe" → "git.exe"
	base = filepath.Base(base)

	// 在Windows上，去除扩展名进行比较
	if runtime.GOOS == "windows" {
		// 保留 .exe 后缀用于匹配，但也尝试不带后缀
		// 例如 "git.exe" 匹配 "git" 或 "git.exe"
		nameWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))
		if nameWithoutExt != "" {
			return nameWithoutExt
		}
	}

	return base
}

// splitShellSubCommands 将Shell命令按操作符分割为多个子命令
// 支持的操作符: |, ||, &&, ;
// 使用基础字符串分割，不引入Shell解析器，宁可误报不漏报
func splitShellSubCommands(command string) []string {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil
	}

	// 按优先级从低到高分隔符逐层分割
	// 先按 ; 分割，再按 && 和 || 分割，最后按 | 分割
	var segments []string

	// 第一步：按 ; 分割
	for _, seg := range strings.Split(command, ";") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		// 第二步：按 && 分割
		for _, seg2 := range strings.Split(seg, "&&") {
			seg2 = strings.TrimSpace(seg2)
			if seg2 == "" {
				continue
			}
			// 第三步：按 || 分割
			for _, seg3 := range strings.Split(seg2, "||") {
				seg3 = strings.TrimSpace(seg3)
				if seg3 == "" {
					continue
				}
				// 第四步：按 | 分割（管道）
				for _, seg4 := range strings.Split(seg3, "|") {
					seg4 = strings.TrimSpace(seg4)
					if seg4 == "" {
						continue
					}
					segments = append(segments, seg4)
				}
			}
		}
	}

	return segments
}

// isCommandAllowed 检查命令是否在白名单中
// 对于包含管道/链式操作符的Shell命令，会分割后逐段检查
// allowedCommands 为空时表示允许所有命令（向后兼容）
func isCommandAllowed(command string, allowedCommands []string) bool {
	// 白名单为空时，不做限制（向后兼容）
	if len(allowedCommands) == 0 {
		return true
	}

	// 分割为子命令逐个检查，防止 "cmd1 || rm -rf /" 绕过
	subCommands := splitShellSubCommands(command)
	if len(subCommands) == 0 {
		baseCmd := extractBaseCommand(command)
		return baseCmd != ""
	}

	for _, subCmd := range subCommands {
		baseCmd := extractBaseCommand(subCmd)
		if baseCmd == "" {
			return false
		}

		// 检查白名单（不区分大小写）
		baseCmdLower := strings.ToLower(baseCmd)
		found := false
		for _, allowed := range allowedCommands {
			if strings.ToLower(allowed) == baseCmdLower {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// buildNotAllowedError 构建命令不被允许的错误信息
func buildNotAllowedError(command string, allowedCommands []string) *ToolResult {
	// 尝试找出哪个子命令不被允许，提供更精确的错误信息
	subCommands := splitShellSubCommands(command)
	if len(subCommands) > 1 {
		for _, subCmd := range subCommands {
			baseCmd := extractBaseCommand(subCmd)
			baseCmdLower := strings.ToLower(baseCmd)
			found := false
			for _, allowed := range allowedCommands {
				if strings.ToLower(allowed) == baseCmdLower {
					found = true
					break
				}
			}
			if !found {
				return NewErrorResult(fmt.Sprintf(
					"命令 '%s' 不在允许列表中（在复合命令中被发现）。允许的命令: [%s]",
					baseCmd,
					strings.Join(allowedCommands, ", "),
				))
			}
		}
	}

	baseCmd := extractBaseCommand(command)
	return NewErrorResult(fmt.Sprintf(
		"命令 '%s' 不在允许列表中。允许的命令: [%s]",
		baseCmd,
		strings.Join(allowedCommands, ", "),
	))
}

// ==================== 环境变量安全过滤 ====================

// protectedEnvVars 受保护的环境变量黑名单
// 这些变量被注入后可能影响系统安全性，因此必须过滤
var protectedEnvVars = []string{
	"PATH",
	"HOME",
	"USERPROFILE",
	"USER",
	"USERNAME",
	"SHELL",
	"LD_LIBRARY_PATH",
	"DYLD_LIBRARY_PATH",
	"LD_PRELOAD",
	"DYLD_INSERT_LIBRARIES",
	"SYSTEMROOT",
	"WINDIR",
	"HOSTNAME",
	"COMPUTERNAME",
	"TEMP",
	"TMP",
}

// filterEnvironmentVariables 过滤环境变量中的受保护变量
// 移除黑名单中的变量，但不阻止执行，返回被过滤的变量名列表
func filterEnvironmentVariables(env map[string]interface{}) (map[string]string, []string) {
	// 构建受保护变量集合（不区分大小写，因为环境变量在Windows上大小写不敏感）
	protected := make(map[string]bool, len(protectedEnvVars))
	for _, v := range protectedEnvVars {
		protected[strings.ToUpper(v)] = true
	}

	filtered := make(map[string]string, len(env))
	var removed []string

	for k, v := range env {
		keyUpper := strings.ToUpper(k)
		if protected[keyUpper] {
			if vStr, ok := v.(string); ok {
				// 记录被过滤的变量名（不记录值，避免泄露）
				removed = append(removed, k)
				_ = vStr // 不使用值
			}
			continue
		}
		// 安全变量，保留
		if vStr, ok := v.(string); ok {
			filtered[k] = vStr
		}
	}

	return filtered, removed
}

// ==================== ExecuteCommand Tool ====================

// CommandResult 命令执行结果
type CommandResult struct {
	// Command 执行的命令
	Command string `json:"command"`

	// Args 命令参数
	Args []string `json:"args,omitempty"`

	// WorkingDir 工作目录
	WorkingDir string `json:"working_dir,omitempty"`

	// Stdout 标准输出
	Stdout string `json:"stdout"`

	// Stderr 标准错误输出
	Stderr string `json:"stderr,omitempty"`

	// ExitCode 退出码
	ExitCode int `json:"exit_code"`

	// Success 是否成功（退出码为0）
	Success bool `json:"success"`

	// Duration 执行时长
	Duration string `json:"duration"`

	// TimedOut 是否超时
	TimedOut bool `json:"timed_out"`
}

// ExecuteCommandTool 命令执行工具
type ExecuteCommandTool struct {
	// timeout 默认超时时间
	timeout time.Duration
	// allowedCommands 允许执行的命令白名单，为空时不做限制
	allowedCommands []string
	// securityChecker 命令安全审查器（可选，优先于白名单检查）
	securityChecker *CommandSecurityChecker
}

// NewExecuteCommandTool 创建命令执行工具
func NewExecuteCommandTool() *ExecuteCommandTool {
	return &ExecuteCommandTool{
		timeout:         DefaultCommandTimeout,
		allowedCommands: nil,
	}
}

// NewExecuteCommandToolWithTimeout 创建带自定义超时的命令执行工具
func NewExecuteCommandToolWithTimeout(timeout time.Duration) *ExecuteCommandTool {
	return &ExecuteCommandTool{
		timeout:         timeout,
		allowedCommands: nil,
	}
}

// NewExecuteCommandToolWithAllowedCommands 创建带命令白名单的命令执行工具
func NewExecuteCommandToolWithAllowedCommands(allowedCommands []string) *ExecuteCommandTool {
	return &ExecuteCommandTool{
		timeout:         DefaultCommandTimeout,
		allowedCommands: allowedCommands,
	}
}

func (t *ExecuteCommandTool) Name() string {
	return "execute_command"
}

func (t *ExecuteCommandTool) Description() string {
	return "执行系统命令并返回结果。支持设置工作目录和超时时间。返回标准输出、标准错误、退出码等信息。仅允许执行配置中白名单内的命令。"
}

func (t *ExecuteCommandTool) Parameters() map[string]interface{} {
	return BuildSchema(map[string]*ToolParameter{
		"command": {
			Type:        "string",
			Description: "要执行的命令名称或路径",
			Required:    true,
		},
		"args": {
			Type:        "array",
			Description: "命令参数列表",
			Items: &ToolParameter{
				Type: "string",
			},
		},
		"working_dir": {
			Type:        "string",
			Description: "命令执行的工作目录，必须是绝对路径",
		},
		"timeout": {
			Type:        "integer",
			Description: "超时时间（秒），默认60秒",
			Minimum:     func() *float64 { v := float64(1); return &v }(),
			Maximum:     func() *float64 { v := float64(600); return &v }(),
		},
		"env": {
			Type:        "object",
			Description: "环境变量，键值对形式",
		},
	}, []string{"command"})
}

func (t *ExecuteCommandTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	// 获取命令
	commandVal, ok := params["command"]
	if !ok {
		return NewErrorResult("missing required parameter: command"), nil
	}

	command, ok := commandVal.(string)
	if !ok {
		return NewErrorResult("parameter 'command' must be a string"), nil
	}

	if strings.TrimSpace(command) == "" {
		return NewErrorResult("command cannot be empty"), nil
	}

	// 校验命令安全性
	// 获取命令参数用于安全检查
	var cmdArgsForCheck []string
	if argsVal, ok := params["args"]; ok {
		if args, ok := argsVal.([]interface{}); ok {
			for _, arg := range args {
				if argStr, ok := arg.(string); ok {
					cmdArgsForCheck = append(cmdArgsForCheck, argStr)
				}
			}
		}
	}
	if t.securityChecker != nil {
		checkResult := t.securityChecker.Check(command, cmdArgsForCheck)
		switch checkResult.Verdict {
		case SecurityAllow:
			// 放行
		case SecurityDeny:
			return NewErrorResult(fmt.Sprintf("命令被安全审查拒绝: %s", checkResult.Reason)), nil
		case SecurityConfirm:
			// 需要用户确认
			if t.securityChecker.GetUserApproveFunc() != nil {
				if !t.securityChecker.GetUserApproveFunc()(command, checkResult.Reason) {
					return NewErrorResult(fmt.Sprintf("用户拒绝执行: %s。原因: %s", command, checkResult.Reason)), nil
				}
			} else {
				return NewErrorResult(fmt.Sprintf("命令需要确认但无确认通道: %s。原因: %s", command, checkResult.Reason)), nil
			}
		}
	} else if !isCommandAllowed(command, t.allowedCommands) {
		return buildNotAllowedError(command, t.allowedCommands), nil
	}

	// 解析命令和参数
	var cmdName string
	var cmdArgs []string

	// 获取额外参数
	if argsVal, ok := params["args"]; ok {
		if args, ok := argsVal.([]interface{}); ok {
			for _, arg := range args {
				if argStr, ok := arg.(string); ok {
					cmdArgs = append(cmdArgs, argStr)
				}
			}
		}
	}

	// 根据操作系统处理命令
	if runtime.GOOS == "windows" {
		cmdName = "cmd"
		cmdArgs = append([]string{"/c", command}, cmdArgs...)
	} else {
		if len(cmdArgs) > 0 {
			cmdName = "sh"
			cmdArgs = []string{"-c", command + " " + strings.Join(cmdArgs, " ")}
		} else {
			parts := strings.Fields(command)
			if len(parts) > 0 {
				cmdName = parts[0]
				if len(parts) > 1 {
					cmdArgs = parts[1:]
				}
			}
		}
	}

	// 解析超时
	timeout := t.timeout
	if timeoutVal, ok := params["timeout"]; ok {
		if timeoutSec, ok := timeoutVal.(float64); ok {
			timeout = time.Duration(timeoutSec) * time.Second
		} else if timeoutSec, ok := timeoutVal.(int); ok {
			timeout = time.Duration(timeoutSec) * time.Second
		}
	}

	// 创建带超时的上下文（只创建一次 cmd，避免丢失设置）
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(timeoutCtx, cmdName, cmdArgs...)
	} else {
		if len(cmdArgs) > 0 {
			cmd = exec.CommandContext(timeoutCtx, cmdName, cmdArgs...)
		} else {
			cmd = exec.CommandContext(timeoutCtx, cmdName)
		}
	}

	// 设置工作目录
	if workDirVal, ok := params["working_dir"]; ok {
		if workDir, ok := workDirVal.(string); ok && workDir != "" {
			cmd.Dir = workDir
		}
	}

	// 设置环境变量（继承当前进程的环境变量，再追加自定义变量）
	// 安全过滤：移除受保护的环境变量（PATH/HOME等），防止环境注入攻击
	var envWarnings []string
	if envVal, ok := params["env"]; ok {
		if env, ok := envVal.(map[string]interface{}); ok {
			// 过滤受保护的环境变量
			safeEnv, removed := filterEnvironmentVariables(env)
			if len(removed) > 0 {
				envWarnings = removed
			}
			// 继承当前进程的所有环境变量
			cmd.Env = append(os.Environ())
			// 追加经过过滤的安全环境变量
			for k, v := range safeEnv {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}
		}
	}

	// 准备输出缓冲区
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 记录开始时间
	startTime := time.Now()

	// 执行命令
	err := cmd.Run()
	duration := time.Since(startTime)

	// 构建结果
	result := &CommandResult{
		Command:    command,
		Args:       cmdArgs,
		WorkingDir: cmd.Dir,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		Duration:   duration.String(),
		TimedOut:   timeoutCtx.Err() == context.DeadlineExceeded,
	}

	// 获取退出码
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else if result.TimedOut {
			result.ExitCode = -1
			result.Stderr = fmt.Sprintf("command timed out after %s", timeout)
		} else {
			result.ExitCode = -1
			result.Stderr = err.Error()
		}
		result.Success = false
	} else {
		result.ExitCode = 0
		result.Success = true
	}

	// 如果有环境变量被过滤，附加警告信息到stderr
	if len(envWarnings) > 0 {
		warningMsg := fmt.Sprintf("[安全警告] 以下受保护的环境变量已被过滤: %s", strings.Join(envWarnings, ", "))
		if result.Stderr != "" {
			result.Stderr += "\n" + warningMsg
		} else {
			result.Stderr = warningMsg
		}
	}

	return NewSuccessResult(result), nil
}

// ==================== ShellExecute Tool ====================

// ShellExecuteTool Shell命令执行工具
// 提供更简单的接口，直接执行shell命令字符串
type ShellExecuteTool struct {
	timeout         time.Duration
	allowedCommands []string
	securityChecker *CommandSecurityChecker
}

func NewShellExecuteTool() *ShellExecuteTool {
	return &ShellExecuteTool{
		timeout:         DefaultCommandTimeout,
		allowedCommands: nil,
	}
}

// NewShellExecuteToolWithAllowedCommands 创建带命令白名单的Shell命令执行工具
func NewShellExecuteToolWithAllowedCommands(allowedCommands []string) *ShellExecuteTool {
	return &ShellExecuteTool{
		timeout:         DefaultCommandTimeout,
		allowedCommands: allowedCommands,
	}
}

func (t *ShellExecuteTool) Name() string {
	return "shell_execute"
}

func (t *ShellExecuteTool) Description() string {
	return "执行Shell命令字符串。自动处理管道、重定向等Shell特性。适用于需要执行复杂Shell命令的场景。仅允许执行配置中白名单内的命令。"
}

func (t *ShellExecuteTool) Parameters() map[string]interface{} {
	return BuildSchema(map[string]*ToolParameter{
		"command": {
			Type:        "string",
			Description: "要执行的Shell命令",
			Required:    true,
		},
		"working_dir": {
			Type:        "string",
			Description: "命令执行的工作目录，必须是绝对路径",
		},
		"timeout": {
			Type:        "integer",
			Description: "超时时间（秒），默认60秒",
			Minimum:     func() *float64 { v := float64(1); return &v }(),
			Maximum:     func() *float64 { v := float64(600); return &v }(),
		},
	}, []string{"command"})
}

func (t *ShellExecuteTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	// 获取命令
	commandVal, ok := params["command"]
	if !ok {
		return NewErrorResult("missing required parameter: command"), nil
	}

	command, ok := commandVal.(string)
	if !ok {
		return NewErrorResult("parameter 'command' must be a string"), nil
	}

	if strings.TrimSpace(command) == "" {
		return NewErrorResult("command cannot be empty"), nil
	}

	// 校验命令安全性
	// 对于shell命令，提取第一个命令进行安全校验
	if t.securityChecker != nil {
		checkResult := t.securityChecker.Check(command, nil)
		switch checkResult.Verdict {
		case SecurityAllow:
			// 放行
		case SecurityDeny:
			return NewErrorResult(fmt.Sprintf("命令被安全审查拒绝: %s", checkResult.Reason)), nil
		case SecurityConfirm:
			// 需要用户确认
			if t.securityChecker.GetUserApproveFunc() != nil {
				if !t.securityChecker.GetUserApproveFunc()(command, checkResult.Reason) {
					return NewErrorResult(fmt.Sprintf("用户拒绝执行: %s。原因: %s", command, checkResult.Reason)), nil
				}
			} else {
				return NewErrorResult(fmt.Sprintf("命令需要确认但无确认通道: %s。原因: %s", command, checkResult.Reason)), nil
			}
		}
	} else if !isCommandAllowed(command, t.allowedCommands) {
		return buildNotAllowedError(command, t.allowedCommands), nil
	}

	// 设置超时
	timeout := t.timeout
	if timeoutVal, ok := params["timeout"]; ok {
		if timeoutSec, ok := timeoutVal.(float64); ok {
			timeout = time.Duration(timeoutSec) * time.Second
		} else if timeoutSec, ok := timeoutVal.(int); ok {
			timeout = time.Duration(timeoutSec) * time.Second
		}
	}

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 根据操作系统选择shell
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(timeoutCtx, "cmd", "/c", command)
	} else {
		cmd = exec.CommandContext(timeoutCtx, "sh", "-c", command)
	}

	// 设置工作目录
	if workDirVal, ok := params["working_dir"]; ok {
		if workDir, ok := workDirVal.(string); ok && workDir != "" {
			cmd.Dir = workDir
		}
	}

	// 准备输出缓冲区
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 记录开始时间
	startTime := time.Now()

	// 执行命令
	err := cmd.Run()
	duration := time.Since(startTime)

	// 构建结果
	result := &CommandResult{
		Command:    command,
		WorkingDir: cmd.Dir,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		Duration:   duration.String(),
		TimedOut:   timeoutCtx.Err() == context.DeadlineExceeded,
	}

	// 获取退出码
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else if result.TimedOut {
			result.ExitCode = -1
			result.Stderr = fmt.Sprintf("command timed out after %s", timeout)
		} else {
			result.ExitCode = -1
			result.Stderr = err.Error()
		}
		result.Success = false
	} else {
		result.ExitCode = 0
		result.Success = true
	}

	return NewSuccessResult(result), nil
}

// ==================== 工具注册函数 ====================

// RegisterCommandTools 注册所有命令执行工具到指定注册中心
// 不带白名单限制，为了向后兼容保留
func RegisterCommandTools(registry *ToolRegistry) error {
	toolList := []Tool{
		NewExecuteCommandTool(),
		NewShellExecuteTool(),
	}

	for _, tool := range toolList {
		if err := registry.RegisterTool(tool); err != nil {
			return fmt.Errorf("failed to register tool %s: %w", tool.Name(), err)
		}
	}
	return nil
}

// RegisterCommandToolsWithAllowedCommands 注册带白名单限制的命令执行工具
// allowedCommands: 允许执行的命令名称列表，为空时不做限制
func RegisterCommandToolsWithAllowedCommands(registry *ToolRegistry, allowedCommands []string) error {
	toolList := []Tool{
		NewExecuteCommandToolWithAllowedCommands(allowedCommands),
		NewShellExecuteToolWithAllowedCommands(allowedCommands),
	}

	for _, tool := range toolList {
		if err := registry.RegisterTool(tool); err != nil {
			return fmt.Errorf("failed to register tool %s: %w", tool.Name(), err)
		}
	}
	return nil
}

// RegisterDefaultCommandTools 注册命令执行工具到默认注册中心
func RegisterDefaultCommandTools() error {
	return RegisterCommandTools(defaultRegistry)
}

// RegisterCommandToolsWithSecurity 注册带安全审查的命令执行工具
// 使用安全审查器替代原有的白名单机制：
//   - 扩展白名单自动包含常见构建/运行/系统命令
//   - 非白名单命令通过安全审查器检查
//   - CLI模式下通过userApproveFunc进行终端交互确认
//   - GUI模式下userApproveFunc为nil时，SecurityConfirm命令会被拒绝
func RegisterCommandToolsWithSecurity(registry *ToolRegistry, allowedCommands []string, workDir string, userApproveFunc func(command, reason string) bool) error {
	checker := NewCommandSecurityChecker(workDir, allowedCommands)
	if userApproveFunc != nil {
		checker.SetUserApproveFunc(userApproveFunc)
	}

	execTool := NewExecuteCommandToolWithAllowedCommands(allowedCommands)
	execTool.securityChecker = checker

	shellTool := NewShellExecuteToolWithAllowedCommands(allowedCommands)
	shellTool.securityChecker = checker

	for _, tool := range []Tool{execTool, shellTool} {
		if err := registry.RegisterTool(tool); err != nil {
			return fmt.Errorf("failed to register tool %s: %w", tool.Name(), err)
		}
	}
	return nil
}
