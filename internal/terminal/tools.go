package terminal

import (
	"context"
	"fmt"
	"time"

	"github.com/Attect/MukaAI/internal/tools"
)

// ==================== terminal_exec 工具 ====================

// TerminalExecTool 终端执行命令工具
// Agent 通过此工具在交互式终端中执行命令
// 命令在 PTY 中执行，支持需要交互式输入的命令
type TerminalExecTool struct {
	manager *TerminalManager
}

// NewTerminalExecTool 创建终端执行工具
func NewTerminalExecTool(manager *TerminalManager) *TerminalExecTool {
	return &TerminalExecTool{manager: manager}
}

func (t *TerminalExecTool) Name() string {
	return "terminal_exec"
}

func (t *TerminalExecTool) Description() string {
	return "在交互式终端中执行命令并获取输出。适用于需要交互式终端的命令（如 top、vim 等），或需要处理交互式提示的命令。命令在伪终端中执行，支持实时输出。长时间运行的命令不会阻塞，返回 is_running: true。"
}

func (t *TerminalExecTool) Parameters() map[string]interface{} {
	return tools.BuildSchema(map[string]*tools.ToolParameter{
		"command": {
			Type:        "string",
			Description: "要执行的命令",
			Required:    true,
		},
		"timeout": {
			Type:        "integer",
			Description: "等待输出的超时时间（秒），默认30秒。长时间运行的命令可设置较短超时以快速返回。",
			Default:     DefaultExecTimeout,
		},
		"wait_for_output": {
			Type:        "boolean",
			Description: "是否等待输出稳定后再返回（默认true）。设为false时立即返回，适合后台任务。",
			Default:     true,
		},
	}, []string{"command"})
}

func (t *TerminalExecTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	// 获取命令
	commandVal, ok := params["command"]
	if !ok {
		return tools.NewErrorResult("missing required parameter: command"), nil
	}
	command, ok := commandVal.(string)
	if !ok {
		return tools.NewErrorResult("parameter 'command' must be a string"), nil
	}
	if command == "" {
		return tools.NewErrorResult("command cannot be empty"), nil
	}

	// 解析超时时间
	timeout := time.Duration(DefaultExecTimeout) * time.Second
	if timeoutVal, ok := params["timeout"]; ok {
		if timeoutSec, ok := timeoutVal.(float64); ok {
			timeout = time.Duration(timeoutSec) * time.Second
		}
	}

	// 解析是否等待输出
	waitForOutput := true
	if waitVal, ok := params["wait_for_output"]; ok {
		if wait, ok := waitVal.(bool); ok {
			waitForOutput = wait
		}
	}

	// 确保终端已启动
	if !t.manager.IsRunning() {
		if err := t.manager.Start(); err != nil {
			return tools.NewErrorResult(fmt.Sprintf("failed to start terminal: %s", err.Error())), nil
		}
		// 等待终端初始化
		time.Sleep(300 * time.Millisecond)
	}

	// 记录当前输出位置
	startPos := t.manager.OutputLen()

	// 发送命令到终端
	if err := t.manager.Write([]byte(command + "\n")); err != nil {
		return tools.NewErrorResult(fmt.Sprintf("failed to send command: %s", err.Error())), nil
	}

	if !waitForOutput {
		// 不等待输出，立即返回
		return tools.NewSuccessResult(map[string]interface{}{
			"output":     "",
			"exit_code":  nil,
			"is_running": true,
			"message":    "command executed in terminal (not waiting for output)",
		}), nil
	}

	// 等待输出稳定
	stableInterval := time.Duration(OutputStableInterval) * time.Millisecond
	output, timedOut := t.manager.WaitForOutputStable(stableInterval, timeout, startPos)

	// 判断命令是否还在运行
	isRunning := t.manager.IsRunning()
	var exitCode interface{}
	if !isRunning {
		exitCode = t.manager.GetExitCode()
	}

	result := map[string]interface{}{
		"output":     output,
		"exit_code":  exitCode,
		"is_running": isRunning,
		"message":    "command executed in terminal",
	}

	if timedOut {
		result["message"] = fmt.Sprintf("output collection timed out after %s", timeout)
	}

	return tools.NewSuccessResult(result), nil
}

// ==================== terminal_input 工具 ====================

// TerminalInputTool 终端输入工具
// Agent 通过此工具向交互式终端发送输入
// 适用于需要响应交互式提示的场景（如密码输入、确认对话框等）
type TerminalInputTool struct {
	manager *TerminalManager
}

// NewTerminalInputTool 创建终端输入工具
func NewTerminalInputTool(manager *TerminalManager) *TerminalInputTool {
	return &TerminalInputTool{manager: manager}
}

func (t *TerminalInputTool) Name() string {
	return "terminal_input"
}

func (t *TerminalInputTool) Description() string {
	return "向交互式终端发送输入文本。用于响应需要用户交互的命令提示（如密码输入、确认对话框、选择菜单等）。默认会在输入后自动按回车键。"
}

func (t *TerminalInputTool) Parameters() map[string]interface{} {
	return tools.BuildSchema(map[string]*tools.ToolParameter{
		"text": {
			Type:        "string",
			Description: "要输入的文本内容",
			Required:    true,
		},
		"press_enter": {
			Type:        "boolean",
			Description: "是否在输入后自动按回车键（默认true）",
			Default:     true,
		},
	}, []string{"text"})
}

func (t *TerminalInputTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	textVal, ok := params["text"]
	if !ok {
		return tools.NewErrorResult("missing required parameter: text"), nil
	}
	text, ok := textVal.(string)
	if !ok {
		return tools.NewErrorResult("parameter 'text' must be a string"), nil
	}

	// 是否按回车
	pressEnter := true
	if pressVal, ok := params["press_enter"]; ok {
		if press, ok := pressVal.(bool); ok {
			pressEnter = press
		}
	}

	if !t.manager.IsRunning() {
		return tools.NewErrorResult("terminal is not running"), nil
	}

	// 构建输入
	input := text
	if pressEnter {
		input += "\n"
	}

	if err := t.manager.Write([]byte(input)); err != nil {
		return tools.NewErrorResult(fmt.Sprintf("failed to send input: %s", err.Error())), nil
	}

	return tools.NewSuccessResult(map[string]interface{}{
		"message": "input sent to terminal",
	}), nil
}

// ==================== terminal_read 工具 ====================

// TerminalReadTool 终端读取工具
// Agent 通过此工具读取终端当前的输出内容
// 用于检查命令执行结果、查看交互式命令的状态等
type TerminalReadTool struct {
	manager *TerminalManager
}

// NewTerminalReadTool 创建终端读取工具
func NewTerminalReadTool(manager *TerminalManager) *TerminalReadTool {
	return &TerminalReadTool{manager: manager}
}

func (t *TerminalReadTool) Name() string {
	return "terminal_read"
}

func (t *TerminalReadTool) Description() string {
	return "读取终端当前的输出内容。返回终端缓冲区中的所有输出。可用于检查命令执行结果、查看交互式命令的当前状态等。"
}

func (t *TerminalReadTool) Parameters() map[string]interface{} {
	return tools.BuildSchema(map[string]*tools.ToolParameter{}, []string{})
}

func (t *TerminalReadTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	output := t.manager.ReadOutput()
	isRunning := t.manager.IsRunning()

	var exitCode interface{}
	if !isRunning {
		exitCode = t.manager.GetExitCode()
	}

	return tools.NewSuccessResult(map[string]interface{}{
		"output":     output,
		"is_running": isRunning,
		"exit_code":  exitCode,
	}), nil
}

// ==================== 工具注册函数 ====================

// RegisterTerminalTools 注册所有终端工具到指定的工具注册中心
// manager: 终端管理器实例
func RegisterTerminalTools(registry *tools.ToolRegistry, manager *TerminalManager) error {
	toolList := []tools.Tool{
		NewTerminalExecTool(manager),
		NewTerminalInputTool(manager),
		NewTerminalReadTool(manager),
	}

	for _, tool := range toolList {
		if err := registry.RegisterTool(tool); err != nil {
			return fmt.Errorf("failed to register tool %s: %w", tool.Name(), err)
		}
	}
	return nil
}
