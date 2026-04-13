//go:build windows

package terminal

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/UserExistsError/conpty"
)

// windowsPTY Windows 平台的 PTY 实现
// 使用 Windows ConPty API（Windows 10 1809+ 原生伪终端支持）
type windowsPTY struct {
	cpty    *conpty.ConPty
	running bool
}

// newPTY 创建新的 Windows PTY 实例
func newPTY() PTY {
	return &windowsPTY{}
}

// Start 启动 ConPty 并运行指定命令
// cmd: 命令名称（如 cmd.exe、powershell.exe）
// args: 命令参数（Windows 下合并到命令行字符串中）
func (p *windowsPTY) Start(cmd string, args []string, rows, cols int) error {
	// 构建完整命令行
	cmdLine := cmd
	if len(args) > 0 {
		cmdLine = cmd + " " + strings.Join(args, " ")
	}

	// 创建 ConPty 并启动进程
	cpty, err := conpty.Start(cmdLine, conpty.ConPtyDimensions(cols, rows))
	if err != nil {
		return fmt.Errorf("failed to start conpty: %w", err)
	}

	p.cpty = cpty
	p.running = true
	return nil
}

// Read 从 ConPty 读取输出
func (p *windowsPTY) Read(b []byte) (int, error) {
	if p.cpty == nil {
		return 0, fmt.Errorf("pty not started")
	}
	n, err := p.cpty.Read(b)
	return n, err
}

// Write 向 ConPty 写入输入
func (p *windowsPTY) Write(b []byte) (int, error) {
	if p.cpty == nil {
		return 0, fmt.Errorf("pty not started")
	}
	n, err := p.cpty.Write(b)
	return n, err
}

// Resize 调整 ConPty 终端尺寸
func (p *windowsPTY) Resize(rows, cols int) error {
	if p.cpty == nil {
		return fmt.Errorf("pty not started")
	}
	return p.cpty.Resize(cols, rows)
}

// Close 关闭 ConPty
func (p *windowsPTY) Close() error {
	if p.cpty == nil {
		return nil
	}
	p.running = false
	return p.cpty.Close()
}

// Wait 等待 ConPty 进程退出
func (p *windowsPTY) Wait() error {
	if p.cpty == nil {
		return fmt.Errorf("pty not started")
	}
	ctx := context.Background()
	_, err := p.cpty.Wait(ctx)
	p.running = false
	return err
}

// getDefaultShell 获取 Windows 默认 Shell
// 检测顺序：PowerShell → cmd.exe
func getDefaultShell() string {
	// 优先使用 PowerShell（如果存在）
	psPath := os.Getenv("SystemRoot") + "\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"
	if _, err := os.Stat(psPath); err == nil {
		return psPath
	}

	// 检查 PowerShell Core (pwsh)
	if path, err := os.Stat("pwsh.exe"); err == nil && path != nil {
		return "pwsh.exe"
	}

	// 回退到 cmd.exe
	return "cmd.exe"
}
