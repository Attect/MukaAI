package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// 默认命令执行超时时间
const DefaultCommandTimeout = 60 * time.Second

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
}

// NewExecuteCommandTool 创建命令执行工具
func NewExecuteCommandTool() *ExecuteCommandTool {
	return &ExecuteCommandTool{
		timeout: DefaultCommandTimeout,
	}
}

// NewExecuteCommandToolWithTimeout 创建带自定义超时的命令执行工具
func NewExecuteCommandToolWithTimeout(timeout time.Duration) *ExecuteCommandTool {
	return &ExecuteCommandTool{
		timeout: timeout,
	}
}

func (t *ExecuteCommandTool) Name() string {
	return "execute_command"
}

func (t *ExecuteCommandTool) Description() string {
	return "执行系统命令并返回结果。支持设置工作目录和超时时间。返回标准输出、标准错误、退出码等信息。"
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

	// 解析命令和参数
	// 在Windows上，需要特殊处理shell命令
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
		// Windows下使用cmd /c执行命令
		cmdName = "cmd"
		cmdArgs = append([]string{"/c", command}, cmdArgs...)
	} else {
		// Unix系统使用sh -c执行
		// 如果有额外参数，将命令和参数组合
		if len(cmdArgs) > 0 {
			cmdName = "sh"
			cmdArgs = append([]string{"-c", command + " " + strings.Join(cmdArgs, " ")})
		} else {
			// 解析命令字符串
			parts := strings.Fields(command)
			if len(parts) > 0 {
				cmdName = parts[0]
				if len(parts) > 1 {
					cmdArgs = parts[1:]
				}
			}
		}
	}

	// 创建命令
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, cmdName, cmdArgs...)
	} else {
		if len(cmdArgs) > 0 {
			cmd = exec.CommandContext(ctx, cmdName, cmdArgs...)
		} else {
			cmd = exec.CommandContext(ctx, cmdName)
		}
	}

	// 设置工作目录
	if workDirVal, ok := params["working_dir"]; ok {
		if workDir, ok := workDirVal.(string); ok && workDir != "" {
			cmd.Dir = workDir
		}
	}

	// 设置环境变量
	if envVal, ok := params["env"]; ok {
		if env, ok := envVal.(map[string]interface{}); ok {
			// 继承当前环境变量
			cmd.Env = append(cmd.Env, cmd.Env...)
			for k, v := range env {
				if vStr, ok := v.(string); ok {
					cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, vStr))
				}
			}
		}
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
	cmd = exec.CommandContext(timeoutCtx, cmdName, cmdArgs...)
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

	return NewSuccessResult(result), nil
}

// ==================== ShellExecute Tool ====================

// ShellExecuteTool Shell命令执行工具
// 提供更简单的接口，直接执行shell命令字符串
type ShellExecuteTool struct {
	timeout time.Duration
}

func NewShellExecuteTool() *ShellExecuteTool {
	return &ShellExecuteTool{
		timeout: DefaultCommandTimeout,
	}
}

func (t *ShellExecuteTool) Name() string {
	return "shell_execute"
}

func (t *ShellExecuteTool) Description() string {
	return "执行Shell命令字符串。自动处理管道、重定向等Shell特性。适用于需要执行复杂Shell命令的场景。"
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
func RegisterCommandTools(registry *ToolRegistry) error {
	tools := []Tool{
		NewExecuteCommandTool(),
		NewShellExecuteTool(),
	}

	for _, tool := range tools {
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
