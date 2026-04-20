//go:build windows

package syntax

import (
	"os/exec"
	"syscall"
)

// configureHideWindow 在 exec.Cmd 上配置隐藏 Windows 控制台窗口
func configureHideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
