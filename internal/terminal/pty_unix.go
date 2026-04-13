//go:build !windows

package terminal

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// unixPTY Unix 平台（Linux/macOS）的 PTY 实现
// 使用 creack/pty 库提供 POSIX 伪终端支持
type unixPTY struct {
	ptmx    *os.File  // PTY master 端
	cmd     *exec.Cmd // 子进程
	running bool
}

// newPTY 创建新的 Unix PTY 实例
func newPTY() PTY {
	return &unixPTY{}
}

// Start 启动 PTY 并运行指定命令
func (p *unixPTY) Start(cmd string, args []string, rows, cols int) error {
	p.cmd = exec.Command(cmd, args...)

	// 设置终端尺寸
	ws := &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	}

	var err error
	p.ptmx, err = pty.StartWithSize(p.cmd, ws)
	if err != nil {
		return fmt.Errorf("failed to start pty: %w", err)
	}

	p.running = true
	return nil
}

// Read 从 PTY 读取输出
func (p *unixPTY) Read(b []byte) (int, error) {
	if p.ptmx == nil {
		return 0, fmt.Errorf("pty not started")
	}
	return p.ptmx.Read(b)
}

// Write 向 PTY 写入输入
func (p *unixPTY) Write(b []byte) (int, error) {
	if p.ptmx == nil {
		return 0, fmt.Errorf("pty not started")
	}
	return p.ptmx.Write(b)
}

// Resize 调整 PTY 终端尺寸
func (p *unixPTY) Resize(rows, cols int) error {
	if p.ptmx == nil {
		return fmt.Errorf("pty not started")
	}
	return pty.Setsize(p.ptmx, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
}

// Close 关闭 PTY
func (p *unixPTY) Close() error {
	if p.ptmx == nil {
		return nil
	}
	p.running = false
	return p.ptmx.Close()
}

// Wait 等待 PTY 进程退出
func (p *unixPTY) Wait() error {
	if p.cmd == nil {
		return fmt.Errorf("pty not started")
	}
	err := p.cmd.Wait()
	p.running = false
	return err
}

// getDefaultShell 获取 Unix 默认 Shell
// 检测顺序：/bin/bash → /bin/zsh → /bin/sh
func getDefaultShell() string {
	candidates := []string{"/bin/bash", "/bin/zsh", "/bin/sh"}
	for _, shell := range candidates {
		if info, err := os.Stat(shell); err == nil && info.Mode().IsRegular() {
			return shell
		}
	}
	// 使用 SHELL 环境变量
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/sh"
}
