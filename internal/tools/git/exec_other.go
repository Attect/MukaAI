//go:build !windows

package git

import (
	"os/exec"
)

// configureHideWindow 非 Windows 平台无需配置
func configureHideWindow(cmd *exec.Cmd) {}
