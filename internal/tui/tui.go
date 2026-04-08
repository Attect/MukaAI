// Package tui 提供基于 Bubble Tea 的终端用户界面
// 此文件用于确保 TUI 相关依赖被正确引入项目
package tui

import (
	// 导入 Bubble Tea v2 框架
	_ "charm.land/bubbletea/v2"

	// 导入 Bubbles UI 组件库
	_ "github.com/charmbracelet/bubbles/textinput"

	// 导入 Lipgloss 样式库
	_ "github.com/charmbracelet/lipgloss"
)

// TUI 模块将在后续任务中实现
// 当前仅用于依赖管理
