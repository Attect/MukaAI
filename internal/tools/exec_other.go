//go:build !windows

package tools

import (
	"os/exec"
)

// hideWindowSysProcAttr 非 Windows 平台不需要隐藏窗口
func hideWindowSysProcAttr() interface{} {
	return nil
}

// configureHideWindow 非 Windows 平台无需配置
func configureHideWindow(cmd *exec.Cmd) {}
