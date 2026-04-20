//go:build windows

package tools

import (
	"os/exec"
	"syscall"
)

// configureHideWindow 在 exec.Cmd 上配置隐藏 Windows 控制台窗口
// 使用 CREATE_NO_WINDOW (0x08000000) 标志避免子进程创建可见窗口
func configureHideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
