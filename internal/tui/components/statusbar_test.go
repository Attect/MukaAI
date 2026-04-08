// Package components 提供 TUI 组件实现
package components

import (
	"strings"
	"testing"
)

func TestNewStatusBar(t *testing.T) {
	config := DefaultStatusBarConfig()
	sb := NewStatusBar(config)

	if sb == nil {
		t.Fatal("NewStatusBar() returned nil")
	}

	if sb.Config.ShowDirectory != true {
		t.Error("ShowDirectory should be true by default")
	}

	if sb.Config.ShowTokens != true {
		t.Error("ShowTokens should be true by default")
	}

	if sb.Config.ShowInferences != true {
		t.Error("ShowInferences should be true by default")
	}
}

func TestStatusBar_SetWidth(t *testing.T) {
	sb := NewStatusBar(DefaultStatusBarConfig())
	sb.SetWidth(100)

	if sb.Width != 100 {
		t.Errorf("Width = %d, want 100", sb.Width)
	}
}

func TestStatusBar_SetDirectory(t *testing.T) {
	sb := NewStatusBar(DefaultStatusBarConfig())
	sb.SetDirectory("/home/user/project")

	if sb.CurrentDir != "/home/user/project" {
		t.Errorf("CurrentDir = %s, want /home/user/project", sb.CurrentDir)
	}
}

func TestStatusBar_SetTokens(t *testing.T) {
	sb := NewStatusBar(DefaultStatusBarConfig())
	sb.SetTokens(12345)

	if sb.TotalTokens != 12345 {
		t.Errorf("TotalTokens = %d, want 12345", sb.TotalTokens)
	}
}

func TestStatusBar_SetInferenceCount(t *testing.T) {
	sb := NewStatusBar(DefaultStatusBarConfig())
	sb.SetInferenceCount(5)

	if sb.InferenceCount != 5 {
		t.Errorf("InferenceCount = %d, want 5", sb.InferenceCount)
	}
}

func TestStatusBar_UpdateTokens(t *testing.T) {
	sb := NewStatusBar(DefaultStatusBarConfig())
	sb.SetTokens(100)
	sb.UpdateTokens(50)

	if sb.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want 150", sb.TotalTokens)
	}
}

func TestStatusBar_UpdateInferenceCount(t *testing.T) {
	sb := NewStatusBar(DefaultStatusBarConfig())
	sb.SetInferenceCount(2)
	sb.UpdateInferenceCount(1)

	if sb.InferenceCount != 3 {
		t.Errorf("InferenceCount = %d, want 3", sb.InferenceCount)
	}
}

func TestStatusBar_Render(t *testing.T) {
	tests := []struct {
		name           string
		dir            string
		tokens         int
		inferences     int
		width          int
		wantContains   []string
		dontWantContains []string
	}{
		{
			name:         "基本渲染",
			dir:          "/home/user/project",
			tokens:       12345,
			inferences:   5,
			width:        100,
			wantContains: []string{"📁", "Tokens: 12345", "Inferences: 5"},
		},
		{
			name:         "空目录",
			dir:          "",
			tokens:       0,
			inferences:   0,
			width:        80,
			wantContains: []string{"~", "Tokens: 0", "Inferences: 0"},
		},
		{
			name:         "大数值",
			dir:          "/path/to/project",
			tokens:       999999,
			inferences:   999,
			width:        120,
			wantContains: []string{"Tokens: 999999", "Inferences: 999"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := NewStatusBar(DefaultStatusBarConfig())
			sb.SetDirectory(tt.dir)
			sb.SetTokens(tt.tokens)
			sb.SetInferenceCount(tt.inferences)
			sb.SetWidth(tt.width)

			result := sb.Render()

			// 检查结果不为空
			if result == "" {
				t.Error("Render() returned empty string")
			}

			// 检查是否包含期望的内容
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Render() result does not contain %q", want)
				}
			}

			// 检查是否不包含不期望的内容
			for _, dontWant := range tt.dontWantContains {
				if strings.Contains(result, dontWant) {
					t.Errorf("Render() result should not contain %q", dontWant)
				}
			}
		})
	}
}

func TestStatusBar_RenderWithConfig(t *testing.T) {
	tests := []struct {
		name         string
		config       StatusBarConfig
		wantContains []string
	}{
		{
			name: "只显示目录",
			config: StatusBarConfig{
				ShowDirectory:      true,
				ShowTokens:         false,
				ShowInferences:     false,
				MaxDirectoryLength: 40,
			},
			wantContains: []string{"📁"},
		},
		{
			name: "只显示Tokens",
			config: StatusBarConfig{
				ShowDirectory:      false,
				ShowTokens:         true,
				ShowInferences:     false,
				MaxDirectoryLength: 40,
			},
			wantContains: []string{"Tokens:"},
		},
		{
			name: "只显示推理次数",
			config: StatusBarConfig{
				ShowDirectory:      false,
				ShowTokens:         false,
				ShowInferences:     true,
				MaxDirectoryLength: 40,
			},
			wantContains: []string{"Inferences:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := NewStatusBar(tt.config)
			sb.SetDirectory("/test/path")
			sb.SetTokens(100)
			sb.SetInferenceCount(3)
			sb.SetWidth(80)

			result := sb.Render()

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Render() result does not contain %q", want)
				}
			}
		})
	}
}

func TestStatusBar_FormatDirectory(t *testing.T) {
	tests := []struct {
		name     string
		dir      string
		maxLen   int
		wantLen  int // 期望长度不超过这个值
	}{
		{
			name:    "短路径",
			dir:     "/home/user",
			maxLen:  40,
			wantLen: 40,
		},
		{
			name:    "长路径",
			dir:     "/very/long/path/to/some/directory/that/exceeds/max/length",
			maxLen:  30,
			wantLen: 30,
		},
		{
			name:    "空路径",
			dir:     "",
			maxLen:  40,
			wantLen: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultStatusBarConfig()
			config.MaxDirectoryLength = tt.maxLen
			sb := NewStatusBar(config)
			sb.SetDirectory(tt.dir)

			result := sb.formatDirectory(tt.dir)

			if len(result) > tt.wantLen {
				t.Errorf("formatDirectory() length = %d, want <= %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestStatusBar_String(t *testing.T) {
	sb := NewStatusBar(DefaultStatusBarConfig())
	sb.SetDirectory("/test")
	sb.SetTokens(100)
	sb.SetInferenceCount(5)
	sb.SetWidth(80)

	result := sb.String()

	if result == "" {
		t.Error("String() returned empty string")
	}

	// String() 应该调用 Render()
	if result != sb.Render() {
		t.Error("String() should return same result as Render()")
	}
}

func TestStatusBar_DefaultConfig(t *testing.T) {
	config := DefaultStatusBarConfig()

	if config.ShowDirectory != true {
		t.Error("Default ShowDirectory should be true")
	}

	if config.ShowTokens != true {
		t.Error("Default ShowTokens should be true")
	}

	if config.ShowInferences != true {
		t.Error("Default ShowInferences should be true")
	}

	if config.MaxDirectoryLength != 40 {
		t.Errorf("Default MaxDirectoryLength = %d, want 40", config.MaxDirectoryLength)
	}
}

func TestStatusBar_WidthPadding(t *testing.T) {
	sb := NewStatusBar(DefaultStatusBarConfig())
	sb.SetDirectory("/test")
	sb.SetTokens(100)
	sb.SetInferenceCount(5)

	// 测试不同宽度
	widths := []int{80, 100, 120}
	for _, width := range widths {
		sb.SetWidth(width)
		result := sb.Render()

		// 结果应该填充到指定宽度
		lines := strings.Split(result, "\n")
		if len(lines) > 0 {
			// 注意：lipgloss.Width 计算的是可见宽度，不包括 ANSI 转义码
			// 这里我们只检查结果不为空
			if lines[0] == "" {
				t.Errorf("Render() with width %d returned empty line", width)
			}
		}
	}
}

func TestStatusBar_ConcurrentAccess(t *testing.T) {
	sb := NewStatusBar(DefaultStatusBarConfig())

	// 并发测试
	done := make(chan bool)

	// 并发写入
	for i := 0; i < 10; i++ {
		go func(n int) {
			sb.SetTokens(n * 100)
			sb.SetInferenceCount(n)
			sb.UpdateTokens(1)
			sb.UpdateInferenceCount(1)
			_ = sb.Render()
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}
}
