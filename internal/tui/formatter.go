// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// Formatter 工具调用格式化器
// 负责将工具调用的 JSON 参数和结果格式化为易读的文本
type Formatter struct {
	// maxLineLength 单行最大长度（超过则折叠）
	maxLineLength int
	// maxContentLines 内容最大行数（超过则折叠）
	maxContentLines int
	// indent 缩进字符串
	indent string
	// styles 格式化样式
	styles FormatterStyles
}

// FormatterStyles 格式化器样式
type FormatterStyles struct {
	// Key JSON 键样式
	Key lipgloss.Style
	// Value JSON 值样式
	Value lipgloss.Style
	// String JSON 字符串样式
	String lipgloss.Style
	// Number JSON 数字样式
	Number lipgloss.Style
	// Boolean JSON 布尔样式
	Boolean lipgloss.Style
	// Null JSON null 样式
	Null lipgloss.Style
	// Bracket 括号样式
	Bracket lipgloss.Style
	// Folded 折叠提示样式
	Folded lipgloss.Style
	// Error 错误样式
	Error lipgloss.Style
	// Success 成功样式
	Success lipgloss.Style
}

// DefaultFormatterStyles 返回默认格式化器样式
func DefaultFormatterStyles() FormatterStyles {
	return FormatterStyles{
		Key: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true),
		Value: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB")),
		String: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")),
		Number: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3B82F6")),
		Boolean: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B5CF6")),
		Null: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")),
		Bracket: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")),
		Folded: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")),
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")),
	}
}

// NewFormatter 创建新的格式化器
func NewFormatter() *Formatter {
	return &Formatter{
		maxLineLength:   80,
		maxContentLines: 10,
		indent:          "  ",
		styles:          DefaultFormatterStyles(),
	}
}

// NewFormatterWithConfig 创建带配置的格式化器
func NewFormatterWithConfig(maxLineLength, maxContentLines int, indent string) *Formatter {
	return &Formatter{
		maxLineLength:   maxLineLength,
		maxContentLines: maxContentLines,
		indent:          indent,
		styles:          DefaultFormatterStyles(),
	}
}

// SetStyles 设置样式
func (f *Formatter) SetStyles(styles FormatterStyles) {
	f.styles = styles
}

// SetMaxLineLength 设置单行最大长度
func (f *Formatter) SetMaxLineLength(length int) {
	f.maxLineLength = length
}

// SetMaxContentLines 设置内容最大行数
func (f *Formatter) SetMaxContentLines(lines int) {
	f.maxContentLines = lines
}

// FormatToolCall 格式化工具调用
// name: 工具名称
// arguments: JSON 格式的参数字符串
// isComplete: 是否已完成流式生成
// 返回格式化后的字符串
func (f *Formatter) FormatToolCall(name, arguments string, isComplete bool) string {
	var builder strings.Builder

	// 标题
	title := fmt.Sprintf("┌─ Tool: %s ", name)
	titlePadding := strings.Repeat("─", max(0, 60-len(title)))
	title = title + titlePadding + "┐"
	builder.WriteString(title)
	builder.WriteString("\n")

	if isComplete {
		// 完成后，格式化显示参数
		builder.WriteString("│ Parameters:\n")
		formattedArgs := f.FormatJSON(arguments, 1)
		for _, line := range strings.Split(formattedArgs, "\n") {
			builder.WriteString("│ ")
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	} else {
		// 流式生成中，显示原文
		builder.WriteString("│ ")
		builder.WriteString(arguments)
		builder.WriteString("▌\n")
	}

	// 底部
	builder.WriteString("└")
	builder.WriteString(strings.Repeat("─", 60))
	builder.WriteString("┘")

	return builder.String()
}

// FormatToolResult 格式化工具结果
// result: 工具执行结果
// isError: 是否为错误结果
// 返回格式化后的字符串
func (f *Formatter) FormatToolResult(result string, isError bool) string {
	var builder strings.Builder

	// 标题
	title := "┌─ Tool Result "
	titlePadding := strings.Repeat("─", max(0, 60-len(title)))
	title = title + titlePadding + "┐"
	builder.WriteString(title)
	builder.WriteString("\n")

	// 内容
	if isError {
		// 错误结果
		content := f.styles.Error.Render(result)
		builder.WriteString("│ ")
		builder.WriteString(content)
		builder.WriteString("\n")
	} else {
		// 成功结果
		// 检查是否需要折叠
		lines := strings.Split(result, "\n")
		if len(lines) > f.maxContentLines {
			// 折叠显示
			for i := 0; i < f.maxContentLines; i++ {
				builder.WriteString("│ ")
				line := f.truncateLine(lines[i])
				builder.WriteString(line)
				builder.WriteString("\n")
			}
			remaining := len(lines) - f.maxContentLines
			foldedText := fmt.Sprintf("... (%d more lines)", remaining)
			builder.WriteString("│ ")
			builder.WriteString(f.styles.Folded.Render(foldedText))
			builder.WriteString("\n")
		} else {
			// 完整显示
			for _, line := range lines {
				builder.WriteString("│ ")
				line = f.truncateLine(line)
				if isError {
					line = f.styles.Error.Render(line)
				}
				builder.WriteString(line)
				builder.WriteString("\n")
			}
		}
	}

	// 底部
	builder.WriteString("└")
	builder.WriteString(strings.Repeat("─", 60))
	builder.WriteString("┘")

	return builder.String()
}

// FormatJSON 格式化 JSON 字符串
// jsonStr: JSON 字符串
// indentLevel: 缩进级别
// 返回格式化后的字符串
func (f *Formatter) FormatJSON(jsonStr string, indentLevel int) string {
	// 尝试解析 JSON
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		// 解析失败，返回原文
		return jsonStr
	}

	// 格式化 JSON
	return f.formatValue(data, indentLevel)
}

// formatValue 格式化 JSON 值
func (f *Formatter) formatValue(value interface{}, indentLevel int) string {
	switch v := value.(type) {
	case map[string]interface{}:
		return f.formatObject(v, indentLevel)
	case []interface{}:
		return f.formatArray(v, indentLevel)
	case string:
		return f.styles.String.Render(fmt.Sprintf("%q", v))
	case float64:
		return f.styles.Number.Render(fmt.Sprintf("%v", v))
	case bool:
		return f.styles.Boolean.Render(fmt.Sprintf("%v", v))
	case nil:
		return f.styles.Null.Render("null")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatObject 格式化 JSON 对象
func (f *Formatter) formatObject(obj map[string]interface{}, indentLevel int) string {
	if len(obj) == 0 {
		return f.styles.Bracket.Render("{}")
	}

	var builder strings.Builder
	indent := strings.Repeat(f.indent, indentLevel)
	nextIndent := strings.Repeat(f.indent, indentLevel+1)

	builder.WriteString(f.styles.Bracket.Render("{"))
	builder.WriteString("\n")

	// 遍历键值对
	first := true
	for key, value := range obj {
		if !first {
			builder.WriteString(",\n")
		}
		first = false

		// 键
		builder.WriteString(nextIndent)
		builder.WriteString(f.styles.Key.Render(fmt.Sprintf("%q", key)))
		builder.WriteString(": ")

		// 值
		formattedValue := f.formatValue(value, indentLevel+1)
		// 如果值是多行的，需要调整缩进
		if strings.Contains(formattedValue, "\n") {
			lines := strings.Split(formattedValue, "\n")
			for i, line := range lines {
				if i > 0 {
					builder.WriteString("\n")
					builder.WriteString(nextIndent)
				}
				builder.WriteString(line)
			}
		} else {
			builder.WriteString(formattedValue)
		}
	}

	builder.WriteString("\n")
	builder.WriteString(indent)
	builder.WriteString(f.styles.Bracket.Render("}"))

	return builder.String()
}

// formatArray 格式化 JSON 数组
func (f *Formatter) formatArray(arr []interface{}, indentLevel int) string {
	if len(arr) == 0 {
		return f.styles.Bracket.Render("[]")
	}

	var builder strings.Builder
	indent := strings.Repeat(f.indent, indentLevel)
	nextIndent := strings.Repeat(f.indent, indentLevel+1)

	builder.WriteString(f.styles.Bracket.Render("["))
	builder.WriteString("\n")

	// 遍历数组元素
	for i, value := range arr {
		if i > 0 {
			builder.WriteString(",\n")
		}

		builder.WriteString(nextIndent)
		formattedValue := f.formatValue(value, indentLevel+1)
		// 如果值是多行的，需要调整缩进
		if strings.Contains(formattedValue, "\n") {
			lines := strings.Split(formattedValue, "\n")
			for j, line := range lines {
				if j > 0 {
					builder.WriteString("\n")
					builder.WriteString(nextIndent)
				}
				builder.WriteString(line)
			}
		} else {
			builder.WriteString(formattedValue)
		}
	}

	builder.WriteString("\n")
	builder.WriteString(indent)
	builder.WriteString(f.styles.Bracket.Render("]"))

	return builder.String()
}

// truncateLine 截断过长的行
func (f *Formatter) truncateLine(line string) string {
	if len(line) <= f.maxLineLength {
		return line
	}
	return line[:f.maxLineLength-3] + "..."
}

// FormatToolCallCompact 格式化工具调用（紧凑格式）
// 用于在有限空间内显示工具调用信息
func (f *Formatter) FormatToolCallCompact(name, arguments string) string {
	// 解析参数
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &params); err != nil {
		// 解析失败，返回简单格式
		return fmt.Sprintf("%s(%s)", name, arguments)
	}

	// 提取主要参数
	var parts []string
	for key, value := range params {
		// 只显示前3个参数
		if len(parts) >= 3 {
			parts = append(parts, "...")
			break
		}
		// 格式化值
		var valueStr string
		switch v := value.(type) {
		case string:
			if len(v) > 20 {
				valueStr = fmt.Sprintf("%q...", v[:17])
			} else {
				valueStr = fmt.Sprintf("%q", v)
			}
		case float64:
			valueStr = fmt.Sprintf("%v", v)
		case bool:
			valueStr = fmt.Sprintf("%v", v)
		case nil:
			valueStr = "null"
		default:
			valueStr = "..."
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, valueStr))
	}

	return fmt.Sprintf("%s(%s)", name, strings.Join(parts, ", "))
}

// FormatToolResultCompact 格式化工具结果（紧凑格式）
func (f *Formatter) FormatToolResultCompact(result string, isError bool) string {
	// 截断过长的结果
	maxLen := 100
	if len(result) > maxLen {
		result = result[:maxLen-3] + "..."
	}

	if isError {
		return f.styles.Error.Render("✗ " + result)
	}
	return f.styles.Success.Render("✓ " + result)
}

// ParseToolCallArguments 解析工具调用参数
// 返回解析后的参数映射和解析错误（如果有）
func (f *Formatter) ParseToolCallArguments(arguments string) (map[string]interface{}, error) {
	var params map[string]interface{}
	err := json.Unmarshal([]byte(arguments), &params)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tool call arguments: %w", err)
	}
	return params, nil
}

// FormatParameterList 格式化参数列表
// params: 参数映射
// 返回格式化后的参数列表字符串
func (f *Formatter) FormatParameterList(params map[string]interface{}) string {
	var builder strings.Builder
	builder.WriteString("Parameters:\n")

	if len(params) == 0 {
		builder.WriteString(f.indent)
		builder.WriteString("(none)")
		builder.WriteString("\n")
		return builder.String()
	}

	for key, value := range params {
		builder.WriteString(f.indent)
		builder.WriteString(f.styles.Key.Render(key))
		builder.WriteString(": ")

		// 格式化值
		formattedValue := f.formatValue(value, 1)
		// 如果值是多行的，需要调整缩进
		if strings.Contains(formattedValue, "\n") {
			lines := strings.Split(formattedValue, "\n")
			for i, line := range lines {
				if i > 0 {
					builder.WriteString("\n")
					builder.WriteString(f.indent)
					builder.WriteString(f.indent)
				}
				builder.WriteString(line)
			}
		} else {
			builder.WriteString(formattedValue)
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// FormatFoldedContent 格式化折叠内容
// content: 内容字符串
// folded: 是否折叠
// 返回格式化后的字符串
func (f *Formatter) FormatFoldedContent(content string, folded bool) string {
	if !folded {
		return content
	}

	lines := strings.Split(content, "\n")
	if len(lines) <= f.maxContentLines {
		return content
	}

	// 显示前几行
	var builder strings.Builder
	for i := 0; i < f.maxContentLines; i++ {
		builder.WriteString(lines[i])
		builder.WriteString("\n")
	}

	// 折叠提示
	remaining := len(lines) - f.maxContentLines
	foldedText := fmt.Sprintf("... (%d more lines, press Enter to expand)", remaining)
	builder.WriteString(f.styles.Folded.Render(foldedText))

	return builder.String()
}

// max 返回两个整数中的最大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
