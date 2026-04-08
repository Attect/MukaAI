// Package components 提供 TUI 组件实现
package components

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestNewFormatter 测试格式化器创建
func TestNewFormatter(t *testing.T) {
	f := NewFormatter()
	if f == nil {
		t.Fatal("NewFormatter() returned nil")
	}
	if f.maxLineLength != 80 {
		t.Errorf("Expected maxLineLength 80, got %d", f.maxLineLength)
	}
	if f.maxContentLines != 10 {
		t.Errorf("Expected maxContentLines 10, got %d", f.maxContentLines)
	}
	if f.indent != "  " {
		t.Errorf("Expected indent '  ', got %q", f.indent)
	}
}

// TestNewFormatterWithConfig 测试带配置的格式化器创建
func TestNewFormatterWithConfig(t *testing.T) {
	f := NewFormatterWithConfig(100, 20, "    ")
	if f == nil {
		t.Fatal("NewFormatterWithConfig() returned nil")
	}
	if f.maxLineLength != 100 {
		t.Errorf("Expected maxLineLength 100, got %d", f.maxLineLength)
	}
	if f.maxContentLines != 20 {
		t.Errorf("Expected maxContentLines 20, got %d", f.maxContentLines)
	}
	if f.indent != "    " {
		t.Errorf("Expected indent '    ', got %q", f.indent)
	}
}

// TestFormatterFormatToolCall 测试工具调用格式化
func TestFormatterFormatToolCall(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name       string
		toolName   string
		arguments  string
		isComplete bool
		wantErr    bool
	}{
		{
			name:       "simple arguments",
			toolName:   "create_file",
			arguments:  `{"path": "/test/file.go", "content": "package main"}`,
			isComplete: true,
			wantErr:    false,
		},
		{
			name:       "nested arguments",
			toolName:   "execute_command",
			arguments:  `{"command": "ls", "args": ["-la", "/home"]}`,
			isComplete: true,
			wantErr:    false,
		},
		{
			name:       "streaming incomplete",
			toolName:   "read_file",
			arguments:  `{"path": "/test/file.go"`,
			isComplete: false,
			wantErr:    false,
		},
		{
			name:       "empty arguments",
			toolName:   "list_files",
			arguments:  `{}`,
			isComplete: true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.FormatToolCall(tt.toolName, tt.arguments, tt.isComplete)

			// 检查结果包含工具名称
			if !strings.Contains(result, tt.toolName) {
				t.Errorf("FormatToolCall() result does not contain tool name %q", tt.toolName)
			}

			// 检查结果非空
			if result == "" {
				t.Error("FormatToolCall() returned empty string")
			}

			// 检查格式化结构
			if !strings.Contains(result, "Tool:") {
				t.Error("FormatToolCall() result does not contain 'Tool:' header")
			}
		})
	}
}

// TestFormatterFormatToolResult 测试工具结果格式化
func TestFormatterFormatToolResult(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name    string
		result  string
		isError bool
	}{
		{
			name:    "success result",
			result:  "File created successfully",
			isError: false,
		},
		{
			name:    "error result",
			result:  "Failed to create file: permission denied",
			isError: true,
		},
		{
			name:    "multiline result",
			result:  "Line 1\nLine 2\nLine 3\nLine 4\nLine 5",
			isError: false,
		},
		{
			name:    "empty result",
			result:  "",
			isError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.FormatToolResult(tt.result, tt.isError)

			// 检查结果非空
			if result == "" {
				t.Error("FormatToolResult() returned empty string")
			}

			// 检查格式化结构
			if !strings.Contains(result, "Tool Result") {
				t.Error("FormatToolResult() result does not contain 'Tool Result' header")
			}
		})
	}
}

// TestFormatJSON 测试 JSON 格式化
func TestFormatJSON(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name     string
		jsonStr  string
		wantErr  bool
		contains string
	}{
		{
			name:     "simple object",
			jsonStr:  `{"key": "value"}`,
			wantErr:  false,
			contains: "key",
		},
		{
			name:     "nested object",
			jsonStr:  `{"outer": {"inner": "value"}}`,
			wantErr:  false,
			contains: "outer",
		},
		{
			name:     "array",
			jsonStr:  `{"items": [1, 2, 3]}`,
			wantErr:  false,
			contains: "items",
		},
		{
			name:     "mixed types",
			jsonStr:  `{"string": "text", "number": 42, "bool": true, "null": null}`,
			wantErr:  false,
			contains: "string",
		},
		{
			name:     "invalid json",
			jsonStr:  `{invalid}`,
			wantErr:  false, // 应该返回原文
			contains: "{invalid}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.FormatJSON(tt.jsonStr, 0)

			// 检查结果非空
			if result == "" {
				t.Error("FormatJSON() returned empty string")
			}

			// 检查包含预期内容
			if !strings.Contains(result, tt.contains) {
				t.Errorf("FormatJSON() result does not contain %q", tt.contains)
			}
		})
	}
}

// TestFormatToolCallCompact 测试紧凑格式化
func TestFormatToolCallCompact(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name      string
		toolName  string
		arguments string
		wantErr   bool
	}{
		{
			name:      "simple arguments",
			toolName:  "create_file",
			arguments: `{"path": "/test/file.go", "content": "package main"}`,
			wantErr:   false,
		},
		{
			name:      "many arguments",
			toolName:  "execute",
			arguments: `{"a": 1, "b": 2, "c": 3, "d": 4}`,
			wantErr:   false,
		},
		{
			name:      "invalid json",
			toolName:  "test",
			arguments: `{invalid}`,
			wantErr:   false, // 应该返回简单格式
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.FormatToolCallCompact(tt.toolName, tt.arguments)

			// 检查结果非空
			if result == "" {
				t.Error("FormatToolCallCompact() returned empty string")
			}

			// 检查包含工具名称
			if !strings.Contains(result, tt.toolName) {
				t.Errorf("FormatToolCallCompact() result does not contain tool name %q", tt.toolName)
			}
		})
	}
}

// TestFormatToolResultCompact 测试紧凑结果格式化
func TestFormatToolResultCompact(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name    string
		result  string
		isError bool
	}{
		{
			name:    "success",
			result:  "File created successfully",
			isError: false,
		},
		{
			name:    "error",
			result:  "Permission denied",
			isError: true,
		},
		{
			name:    "long result",
			result:  strings.Repeat("a", 150),
			isError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.FormatToolResultCompact(tt.result, tt.isError)

			// 检查结果非空
			if result == "" {
				t.Error("FormatToolResultCompact() returned empty string")
			}

			// 检查长度限制（考虑样式字符）
			if len(result) > 150 {
				t.Errorf("FormatToolResultCompact() result too long: %d", len(result))
			}
		})
	}
}

// TestParseToolCallArguments 测试参数解析
func TestParseToolCallArguments(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name      string
		arguments string
		wantErr   bool
		wantKeys  []string
	}{
		{
			name:      "valid json",
			arguments: `{"path": "/test/file.go", "content": "package main"}`,
			wantErr:   false,
			wantKeys:  []string{"path", "content"},
		},
		{
			name:      "empty json",
			arguments: `{}`,
			wantErr:   false,
			wantKeys:  []string{},
		},
		{
			name:      "invalid json",
			arguments: `{invalid}`,
			wantErr:   true,
			wantKeys:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := f.ParseToolCallArguments(tt.arguments)

			if tt.wantErr {
				if err == nil {
					t.Error("ParseToolCallArguments() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseToolCallArguments() unexpected error: %v", err)
				return
			}

			// 检查键
			for _, key := range tt.wantKeys {
				if _, ok := params[key]; !ok {
					t.Errorf("ParseToolCallArguments() missing key %q", key)
				}
			}
		})
	}
}

// TestFormatParameterList 测试参数列表格式化
func TestFormatParameterList(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{
			name: "simple params",
			params: map[string]interface{}{
				"path":    "/test/file.go",
				"content": "package main",
			},
		},
		{
			name: "nested params",
			params: map[string]interface{}{
				"config": map[string]interface{}{
					"debug": true,
					"port":  8080,
				},
			},
		},
		{
			name:   "empty params",
			params: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.FormatParameterList(tt.params)

			// 检查结果非空
			if result == "" {
				t.Error("FormatParameterList() returned empty string")
			}

			// 检查包含 "Parameters:"
			if !strings.Contains(result, "Parameters:") {
				t.Error("FormatParameterList() result does not contain 'Parameters:'")
			}
		})
	}
}

// TestFormatFoldedContent 测试折叠内容格式化
func TestFormatFoldedContent(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name    string
		content string
		folded  bool
		wantLen int
	}{
		{
			name:    "short content folded",
			content: "Line 1\nLine 2\nLine 3",
			folded:  true,
			wantLen: 3, // 3行，不折叠
		},
		{
			name:    "long content folded",
			content: strings.Repeat("Line\n", 20),
			folded:  true,
			wantLen: 11, // 10行 + 折叠提示
		},
		{
			name:    "long content not folded",
			content: strings.Repeat("Line\n", 20),
			folded:  false,
			wantLen: 21, // 20行内容 + 1个空行
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.FormatFoldedContent(tt.content, tt.folded)

			// 检查结果非空
			if result == "" {
				t.Error("FormatFoldedContent() returned empty string")
			}

			// 检查行数
			lines := strings.Split(result, "\n")
			if len(lines) != tt.wantLen {
				t.Errorf("FormatFoldedContent() got %d lines, want %d", len(lines), tt.wantLen)
			}
		})
	}
}

// TestTruncateLine 测试行截断
func TestTruncateLine(t *testing.T) {
	f := NewFormatter()
	f.maxLineLength = 80

	tests := []struct {
		name    string
		line    string
		wantLen int
	}{
		{
			name:    "short line",
			line:    "short line",
			wantLen: 10,
		},
		{
			name:    "exact length",
			line:    strings.Repeat("a", 80),
			wantLen: 80,
		},
		{
			name:    "long line",
			line:    strings.Repeat("a", 100),
			wantLen: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.truncateLine(tt.line)

			if len(result) != tt.wantLen {
				t.Errorf("truncateLine() got length %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

// TestFormatValue 测试值格式化
func TestFormatValue(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name  string
		value interface{}
	}{
		{
			name:  "string",
			value: "test string",
		},
		{
			name:  "number",
			value: 42.0,
		},
		{
			name:  "boolean true",
			value: true,
		},
		{
			name:  "boolean false",
			value: false,
		},
		{
			name:  "null",
			value: nil,
		},
		{
			name:  "object",
			value: map[string]interface{}{"key": "value"},
		},
		{
			name:  "array",
			value: []interface{}{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.formatValue(tt.value, 0)

			// 检查结果非空
			if result == "" {
				t.Error("formatValue() returned empty string")
			}
		})
	}
}

// TestFormatObject 测试对象格式化
func TestFormatObject(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name   string
		object map[string]interface{}
	}{
		{
			name:   "empty object",
			object: map[string]interface{}{},
		},
		{
			name: "simple object",
			object: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name: "nested object",
			object: map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.formatObject(tt.object, 0)

			// 检查结果非空
			if result == "" {
				t.Error("formatObject() returned empty string")
			}
		})
	}
}

// TestFormatArray 测试数组格式化
func TestFormatArray(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name  string
		array []interface{}
	}{
		{
			name:  "empty array",
			array: []interface{}{},
		},
		{
			name:  "simple array",
			array: []interface{}{1, 2, 3},
		},
		{
			name:  "mixed array",
			array: []interface{}{"string", 42, true, nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.formatArray(tt.array, 0)

			// 检查结果非空
			if result == "" {
				t.Error("formatArray() returned empty string")
			}
		})
	}
}

// TestSetStyles 测试样式设置
func TestFormatterSetStyles(t *testing.T) {
	f := NewFormatter()
	styles := DefaultFormatterStyles()
	f.SetStyles(styles)

	// 检查样式已设置
	if f.styles.Key.GetForeground() == nil {
		t.Error("SetStyles() did not set Key style")
	}
}

// TestSetMaxLineLength 测试设置最大行长度
func TestSetMaxLineLength(t *testing.T) {
	f := NewFormatter()
	f.SetMaxLineLength(100)

	if f.maxLineLength != 100 {
		t.Errorf("SetMaxLineLength() failed, got %d", f.maxLineLength)
	}
}

// TestSetMaxContentLines 测试设置最大内容行数
func TestSetMaxContentLines(t *testing.T) {
	f := NewFormatter()
	f.SetMaxContentLines(20)

	if f.maxContentLines != 20 {
		t.Errorf("SetMaxContentLines() failed, got %d", f.maxContentLines)
	}
}

// TestIntegration 测试集成场景
func TestIntegration(t *testing.T) {
	f := NewFormatter()

	// 模拟完整的工具调用流程
	toolName := "create_file"
	arguments := `{"path": "/test/file.go", "content": "package main\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}"}`

	// 1. 流式生成中
	streamingResult := f.FormatToolCall(toolName, arguments, false)
	if !strings.Contains(streamingResult, "▌") {
		t.Error("Streaming result should contain cursor")
	}

	// 2. 完成后
	completeResult := f.FormatToolCall(toolName, arguments, true)
	if strings.Contains(completeResult, "▌") {
		t.Error("Complete result should not contain cursor")
	}

	// 3. 解析参数
	params, err := f.ParseToolCallArguments(arguments)
	if err != nil {
		t.Errorf("ParseToolCallArguments() failed: %v", err)
	}

	// 4. 格式化参数列表
	paramList := f.FormatParameterList(params)
	if !strings.Contains(paramList, "path") {
		t.Error("Parameter list should contain 'path'")
	}

	// 5. 工具结果
	result := "File created successfully"
	resultFormatted := f.FormatToolResult(result, false)
	if !strings.Contains(resultFormatted, result) {
		t.Error("Result should contain original text")
	}

	// 6. 错误结果
	errorResult := "Permission denied"
	errorFormatted := f.FormatToolResult(errorResult, true)
	if !strings.Contains(errorFormatted, errorResult) {
		t.Error("Error result should contain original text")
	}
}

// TestComplexJSON 测试复杂 JSON 格式化
func TestComplexJSON(t *testing.T) {
	f := NewFormatter()

	// 复杂的嵌套 JSON
	complexJSON := `{
		"tool": "execute_command",
		"params": {
			"command": "npm",
			"args": ["install", "--save-dev", "typescript"],
			"options": {
				"cwd": "/project",
				"env": {
					"NODE_ENV": "development"
				}
			}
		},
		"timeout": 60000,
		"retry": 3
	}`

	result := f.FormatJSON(complexJSON, 0)

	// 检查格式化结果
	if !strings.Contains(result, "tool") {
		t.Error("Result should contain 'tool'")
	}
	if !strings.Contains(result, "params") {
		t.Error("Result should contain 'params'")
	}
	if !strings.Contains(result, "command") {
		t.Error("Result should contain 'command'")
	}
}

// TestEdgeCases 测试边界情况
func TestEdgeCases(t *testing.T) {
	f := NewFormatter()

	// 空字符串
	emptyResult := f.FormatToolCall("test", "", true)
	if emptyResult == "" {
		t.Error("FormatToolCall with empty arguments should not return empty string")
	}

	// 超长参数
	longArgs := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		longArgs[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}
	longArgsJSON, _ := json.Marshal(longArgs)
	longResult := f.FormatToolCall("test", string(longArgsJSON), true)
	if longResult == "" {
		t.Error("FormatToolCall with many arguments should not return empty string")
	}

	// 特殊字符
	specialArgs := `{"path": "/test/file with spaces.go", "content": "package main\n\n// 中文注释\nfunc main() {}"}`
	specialResult := f.FormatToolCall("test", specialArgs, true)
	if specialResult == "" {
		t.Error("FormatToolCall with special characters should not return empty string")
	}
}
