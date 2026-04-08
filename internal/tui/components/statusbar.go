// Package components 提供 TUI 组件实现
package components

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
)

// StatusBarConfig 状态栏配置
type StatusBarConfig struct {
	// ShowDirectory 是否显示工作目录
	ShowDirectory bool
	// ShowTokens 是否显示 token 用量
	ShowTokens bool
	// ShowInferences 是否显示推理次数
	ShowInferences bool
	// MaxDirectoryLength 目录最大显示长度
	MaxDirectoryLength int
}

// DefaultStatusBarConfig 返回默认状态栏配置
func DefaultStatusBarConfig() StatusBarConfig {
	return StatusBarConfig{
		ShowDirectory:      true,
		ShowTokens:         true,
		ShowInferences:     true,
		MaxDirectoryLength: 40,
	}
}

// StatusBar 状态栏组件
type StatusBar struct {
	// Width 宽度
	Width int
	// CurrentDir 当前工作目录
	CurrentDir string
	// TotalTokens 总 token 用量
	TotalTokens int
	// InferenceCount 推理次数
	InferenceCount int
	// Config 配置
	Config StatusBarConfig
	// 样式
	styles statusBarStyles
}

// statusBarStyles 状态栏样式
type statusBarStyles struct {
	// container 容器样式
	container lipgloss.Style
	// directory 目录样式
	directory lipgloss.Style
	// tokens token 样式
	tokens lipgloss.Style
	// inferences 推理次数样式
	inferences lipgloss.Style
	// separator 分隔符样式
	separator lipgloss.Style
	// icon 图标样式
	icon lipgloss.Style
}

// NewStatusBar 创建新的状态栏组件
func NewStatusBar(config StatusBarConfig) *StatusBar {
	// 定义颜色
	colorBackground := lipgloss.Color("#1F2937") // 深灰色背景
	colorText := lipgloss.Color("#F3F4F6")       // 浅灰色文本
	colorAccent := lipgloss.Color("#7C3AED")     // 紫色强调
	colorMuted := lipgloss.Color("#9CA3AF")      // 浅灰色弱化

	// 初始化样式
	styles := statusBarStyles{
		container: lipgloss.NewStyle().
			Background(colorBackground).
			Foreground(colorText).
			Padding(0, 1).
			Height(1),
		directory: lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1).
			Bold(true),
		tokens: lipgloss.NewStyle().
			Foreground(colorAccent).
			Padding(0, 1).
			Bold(true),
		inferences: lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1),
		separator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#374151")).
			Padding(0, 1),
		icon: lipgloss.NewStyle().
			Foreground(colorAccent).
			Padding(0, 0, 0, 1),
	}

	return &StatusBar{
		Config: config,
		styles: styles,
	}
}

// SetWidth 设置宽度
func (s *StatusBar) SetWidth(width int) {
	s.Width = width
}

// SetDirectory 设置工作目录
func (s *StatusBar) SetDirectory(dir string) {
	s.CurrentDir = dir
}

// SetTokens 设置 token 用量
func (s *StatusBar) SetTokens(tokens int) {
	s.TotalTokens = tokens
}

// SetInferenceCount 设置推理次数
func (s *StatusBar) SetInferenceCount(count int) {
	s.InferenceCount = count
}

// UpdateTokens 更新 token 用量（增量）
func (s *StatusBar) UpdateTokens(delta int) {
	s.TotalTokens += delta
}

// UpdateInferenceCount 更新推理次数（增量）
func (s *StatusBar) UpdateInferenceCount(delta int) {
	s.InferenceCount += delta
}

// Render 渲染状态栏
func (s *StatusBar) Render() string {
	if s.Width <= 0 {
		s.Width = 80 // 默认宽度
	}

	var parts []string

	// 1. 工作目录部分
	if s.Config.ShowDirectory {
		dirDisplay := s.formatDirectory(s.CurrentDir)
		icon := s.styles.icon.Render("📁")
		dirPart := s.styles.directory.Render(dirDisplay)
		parts = append(parts, icon+dirPart)
	}

	// 2. Token 用量部分
	if s.Config.ShowTokens {
		tokenText := fmt.Sprintf("Tokens: %d", s.TotalTokens)
		tokenPart := s.styles.tokens.Render(tokenText)
		parts = append(parts, tokenPart)
	}

	// 3. 推理次数部分
	if s.Config.ShowInferences {
		inferenceText := fmt.Sprintf("Inferences: %d", s.InferenceCount)
		inferencePart := s.styles.inferences.Render(inferenceText)
		parts = append(parts, inferencePart)
	}

	// 使用分隔符连接各部分
	separator := s.styles.separator.Render("│")
	content := strings.Join(parts, separator)

	// 应用容器样式并填充到指定宽度
	rendered := s.styles.container.Render(content)

	// 计算需要填充的空格
	contentWidth := lipgloss.Width(rendered)
	if contentWidth < s.Width {
		padding := s.Width - contentWidth
		rendered += strings.Repeat(" ", padding)
	}

	return rendered
}

// formatDirectory 格式化目录显示
func (s *StatusBar) formatDirectory(dir string) string {
	if dir == "" {
		return "~"
	}

	// 如果目录长度超过最大长度，进行缩写
	if len(dir) > s.Config.MaxDirectoryLength {
		// 尝试缩写路径
		return s.abbreviatePath(dir, s.Config.MaxDirectoryLength)
	}

	return dir
}

// abbreviatePath 缩写路径
func (s *StatusBar) abbreviatePath(path string, maxLen int) string {
	// 获取绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	// 如果绝对路径仍然太长，进行缩写
	if len(absPath) > maxLen {
		// 保留最后两级目录
		dir := filepath.Dir(absPath)
		base := filepath.Base(absPath)

		// 尝试缩短父目录
		parentDir := filepath.Base(dir)
		if parentDir != "" && parentDir != "." && parentDir != ".." {
			abbreviated := filepath.Join("...", parentDir, base)
			if len(abbreviated) <= maxLen {
				return abbreviated
			}
		}

		// 如果还是太长，只保留文件名
		if len(base) <= maxLen {
			return base
		}

		// 最后手段：截断
		return "..." + base[len(base)-maxLen+3:]
	}

	return absPath
}

// String 实现 Stringer 接口
func (s *StatusBar) String() string {
	return s.Render()
}
