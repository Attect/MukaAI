package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Attect/MukaAI/internal/model"
	"github.com/Attect/MukaAI/internal/state"
)

// TestNewReviewer 测试创建审查器
func TestNewReviewer(t *testing.T) {
	// 使用默认配置
	reviewer := NewReviewer(nil)
	if reviewer == nil {
		t.Fatal("NewReviewer returned nil")
	}

	if reviewer.config == nil {
		t.Error("reviewer config should not be nil")
	}

	// 使用自定义配置
	config := &ReviewConfig{
		EnableDirectionCheck:   false,
		MaxRepeatedActions:     5,
		MaxConsecutiveFailures: 2,
	}
	reviewer = NewReviewer(config)
	if reviewer.config.MaxRepeatedActions != 5 {
		t.Errorf("expected MaxRepeatedActions=5, got %d", reviewer.config.MaxRepeatedActions)
	}
}

// TestDefaultReviewConfig 测试默认配置
func TestDefaultReviewConfig(t *testing.T) {
	config := DefaultReviewConfig()

	if !config.EnableDirectionCheck {
		t.Error("EnableDirectionCheck should be true by default")
	}
	if config.MaxRepeatedActions != 4 {
		t.Errorf("expected MaxRepeatedActions=4, got %d", config.MaxRepeatedActions)
	}
	if config.MaxConsecutiveFailures != 3 {
		t.Errorf("expected MaxConsecutiveFailures=3, got %d", config.MaxConsecutiveFailures)
	}
}

// TestReviewOutput 测试输出审查
func TestReviewOutput(t *testing.T) {
	reviewer := NewReviewer(nil)

	// 测试正常输出
	taskState := state.NewTaskState("test-1", "实现用户登录功能")
	output := "我将开始实现用户登录功能，首先创建登录表单"
	toolCalls := []model.ToolCall{
		{
			ID:   "tc-1",
			Type: "function",
			Function: model.FunctionCall{
				Name:      "write_file",
				Arguments: `{"file_path": "login.go", "content": "package main"}`,
			},
		},
	}

	result := reviewer.ReviewOutput(output, toolCalls, taskState)

	if result == nil {
		t.Fatal("ReviewOutput returned nil")
	}

	// 检查结果结构
	if result.Timestamp.IsZero() {
		t.Error("timestamp should be set")
	}
	if result.Summary == "" {
		t.Error("summary should not be empty")
	}
}

// TestReviewOutputWithInvalidToolCall 测试无效工具调用检测
func TestReviewOutputWithInvalidToolCall(t *testing.T) {
	reviewer := NewReviewer(nil)

	// 测试空工具名称
	toolCalls := []model.ToolCall{
		{
			ID:   "tc-1",
			Type: "function",
			Function: model.FunctionCall{
				Name:      "",
				Arguments: `{}`,
			},
		},
	}

	result := reviewer.ReviewOutput("", toolCalls, nil)

	if result.Status != ReviewStatusBlock {
		t.Errorf("expected status block, got %s", result.Status)
	}

	if len(result.Issues) == 0 {
		t.Error("should detect invalid tool call")
	}

	if result.Issues[0].Type != IssueTypeInvalidToolCall {
		t.Errorf("expected issue type invalid_tool_call, got %s", result.Issues[0].Type)
	}
}

// TestReviewOutputWithInvalidJSON 测试无效JSON参数检测
func TestReviewOutputWithInvalidJSON(t *testing.T) {
	reviewer := NewReviewer(nil)

	toolCalls := []model.ToolCall{
		{
			ID:   "tc-1",
			Type: "function",
			Function: model.FunctionCall{
				Name:      "write_file",
				Arguments: `{invalid json}`,
			},
		},
	}

	result := reviewer.ReviewOutput("", toolCalls, nil)

	if result.Status != ReviewStatusBlock {
		t.Errorf("expected status block, got %s", result.Status)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Type == IssueTypeInvalidToolCall {
			found = true
			break
		}
	}

	if !found {
		t.Error("should detect invalid JSON arguments")
	}
}

// TestReviewOutputWithDirectionDeviation 测试方向偏离检测
func TestReviewOutputWithDirectionDeviation(t *testing.T) {
	config := &ReviewConfig{
		EnableDirectionCheck:   true,
		EnableFabricationCheck: false,
	}
	reviewer := NewReviewer(config)

	// 任务目标是实现登录功能，但输出完全不相关
	// 使用英文测试，因为中文分词需要更复杂的处理
	taskState := state.NewTaskState("test-1", "implement user login authentication feature")
	output := "Today is a sunny day, I went to the park for a walk"
	toolCalls := []model.ToolCall{}

	result := reviewer.ReviewOutput(output, toolCalls, taskState)

	// 应该检测到方向偏离
	found := false
	for _, issue := range result.Issues {
		if issue.Type == IssueTypeDirection {
			found = true
			break
		}
	}

	if !found {
		t.Error("should detect direction deviation")
	}
}

// TestReviewOutputWithInfiniteLoop 测试无限循环检测
func TestReviewOutputWithInfiniteLoop(t *testing.T) {
	config := &ReviewConfig{
		MaxRepeatedActions: 2,
		LoopWindowSize:     2, // 设置为2，这样在第3次调用时就能检测到循环
	}
	reviewer := NewReviewer(config)

	// 模拟重复操作（使用execute_command而非read_file，
	// 因为read_file已被checkInfiniteLoop豁免）
	toolCall := model.ToolCall{
		ID:   "tc-1",
		Type: "function",
		Function: model.FunctionCall{
			Name:      "execute_command",
			Arguments: `{"command": "echo hello"}`,
		},
	}

	// 第一次调用
	result := reviewer.ReviewOutput("", []model.ToolCall{toolCall}, nil)
	if result.Status == ReviewStatusBlock {
		t.Error("first call should not be blocked")
	}

	// 第二次调用
	result = reviewer.ReviewOutput("", []model.ToolCall{toolCall}, nil)
	if result.Status == ReviewStatusBlock {
		t.Error("second call should not be blocked")
	}

	// 第三次调用 - 应该检测到循环
	result = reviewer.ReviewOutput("", []model.ToolCall{toolCall}, nil)

	found := false
	for _, issue := range result.Issues {
		if issue.Type == IssueTypeInfiniteLoop {
			found = true
			break
		}
	}

	if !found {
		t.Error("should detect infinite loop after repeated actions")
	}
}

// TestReviewToolResult 测试工具结果审查
func TestReviewToolResult(t *testing.T) {
	reviewer := NewReviewer(nil)

	// 测试成功结果
	result := reviewer.ReviewToolResult("write_file", `{"file_path": "test.txt"}`, `{"success": true}`, true)
	if result.Status != ReviewStatusPass {
		t.Errorf("expected pass for successful result, got %s", result.Status)
	}

	// 测试失败结果
	result = reviewer.ReviewToolResult("write_file", `{"file_path": "test.txt"}`, `{"success": false, "error": "permission denied"}`, false)
	if result.Status != ReviewStatusPass {
		t.Errorf("first failure should not block, got %s", result.Status)
	}
}

// TestReviewToolResultWithRepeatedFailure 测试重复失败检测
func TestReviewToolResultWithRepeatedFailure(t *testing.T) {
	config := &ReviewConfig{
		MaxConsecutiveFailures: 2,
	}
	reviewer := NewReviewer(config)

	// 连续失败
	for i := 0; i < 3; i++ {
		result := reviewer.ReviewToolResult(
			"write_file",
			`{"file_path": "test.txt"}`,
			`{"success": false, "error": "permission denied"}`,
			false,
		)

		if i >= 1 { // 第二次失败后应该检测到问题
			if result.Status != ReviewStatusBlock {
				t.Errorf("iteration %d: expected block status, got %s", i, result.Status)
			}
		}
	}
}

// TestCheckFabrication 测试编造内容检测
func TestCheckFabrication(t *testing.T) {
	// 创建临时文件用于测试
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	config := &ReviewConfig{
		EnableFabricationCheck: true,
	}
	reviewer := NewReviewer(config)

	// 测试输出中提到不存在的文件
	output := "我已经读取了文件 /nonexistent/path/file.go 的内容"
	toolCalls := []model.ToolCall{
		{
			ID:   "tc-1",
			Type: "function",
			Function: model.FunctionCall{
				Name:      "read_file",
				Arguments: `{"file_path": "/nonexistent/path/file.go"}`,
			},
		},
	}

	result := reviewer.ReviewOutput(output, toolCalls, nil)

	// 应该检测到编造
	found := false
	for _, issue := range result.Issues {
		if issue.Type == IssueTypeFabrication {
			found = true
			break
		}
	}

	if !found {
		t.Error("should detect fabrication for nonexistent file")
	}
}

// TestCheckProgress 测试进度检查
func TestCheckProgress(t *testing.T) {
	config := &ReviewConfig{
		MaxIterationsWithoutProgress: 3,
	}
	reviewer := NewReviewer(config)

	taskState := state.NewTaskState("test-1", "测试任务")

	// 多次迭代但无进度
	for i := 0; i < 4; i++ {
		result := reviewer.ReviewOutput("继续执行", nil, taskState)

		if i >= 2 { // 第三次迭代后应该检测到无进度
			found := false
			for _, issue := range result.Issues {
				if issue.Type == IssueTypeNoProgress {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("iteration %d: should detect no progress", i)
			}
		}
	}

	// 添加完成步骤，应该重置计数
	taskState.AddCompletedStep("step1")
	_ = reviewer.ReviewOutput("继续执行", nil, taskState)

	// 进度计数应该重置
	reviewer.mu.RLock()
	iterationCount := reviewer.iterationCount
	reviewer.mu.RUnlock()

	if iterationCount != 0 {
		t.Errorf("iteration count should be reset to 0, got %d", iterationCount)
	}
}

// TestReviewResultMethods 测试ReviewResult的方法
func TestReviewResultMethods(t *testing.T) {
	// 测试IsBlocked
	result := &ReviewResult{Status: ReviewStatusBlock}
	if !result.IsBlocked() {
		t.Error("IsBlocked should return true for block status")
	}

	result = &ReviewResult{Status: ReviewStatusPass}
	if result.IsBlocked() {
		t.Error("IsBlocked should return false for pass status")
	}

	// 测试HasWarnings
	result = &ReviewResult{Status: ReviewStatusWarning}
	if !result.HasWarnings() {
		t.Error("HasWarnings should return true for warning status")
	}

	result = &ReviewResult{
		Status: ReviewStatusPass,
		Issues: []ReviewIssue{{Type: IssueTypeDirection}},
	}
	if !result.HasWarnings() {
		t.Error("HasWarnings should return true when issues exist")
	}

	// 测试GetBlockingIssues
	result = &ReviewResult{
		Issues: []ReviewIssue{
			{Type: IssueTypeDirection, Severity: "low"},
			{Type: IssueTypeInfiniteLoop, Severity: "critical"},
			{Type: IssueTypeInvalidToolCall, Severity: "high"},
		},
	}
	blocking := result.GetBlockingIssues()
	if len(blocking) != 2 {
		t.Errorf("expected 2 blocking issues, got %d", len(blocking))
	}
}

// TestReviewerReset 测试审查器重置
func TestReviewerReset(t *testing.T) {
	reviewer := NewReviewer(nil)

	// 添加一些状态
	reviewer.failureCount = 5
	reviewer.iterationCount = 10
	reviewer.actionHistory = append(reviewer.actionHistory, ActionRecord{
		ToolName:  "test",
		Arguments: "{}",
		Timestamp: time.Now(),
	})

	// 重置
	reviewer.Reset()

	if reviewer.failureCount != 0 {
		t.Errorf("failureCount should be 0, got %d", reviewer.failureCount)
	}
	if reviewer.iterationCount != 0 {
		t.Errorf("iterationCount should be 0, got %d", reviewer.iterationCount)
	}
	if len(reviewer.actionHistory) != 0 {
		t.Errorf("actionHistory should be empty, got %d items", len(reviewer.actionHistory))
	}
}

// TestGetActionHistory 测试获取操作历史
func TestGetActionHistory(t *testing.T) {
	reviewer := NewReviewer(nil)

	// 初始应该为空
	history := reviewer.GetActionHistory()
	if len(history) != 0 {
		t.Error("initial history should be empty")
	}

	// 添加操作
	toolCall := model.ToolCall{
		ID:   "tc-1",
		Type: "function",
		Function: model.FunctionCall{
			Name:      "test_tool",
			Arguments: `{"arg": "value"}`,
		},
	}
	reviewer.ReviewOutput("", []model.ToolCall{toolCall}, nil)

	history = reviewer.GetActionHistory()
	if len(history) != 1 {
		t.Errorf("expected 1 history item, got %d", len(history))
	}

	if history[0].ToolName != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got '%s'", history[0].ToolName)
	}
}

// TestExtractKeywords 测试关键词提取
func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected int // 期望的关键词数量下限
	}{
		// 中文文本没有空格分隔，整个句子被当作一个词，所以关键词数量为1
		{"实现用户登录功能", 1},
		{"Create a new file for user authentication", 3},
		{"the quick brown fox jumps over the lazy dog", 3}, // 停用词应被过滤
		{"", 0},
	}

	for _, test := range tests {
		keywords := extractKeywords(test.input)
		if len(keywords) < test.expected {
			t.Errorf("input '%s': expected at least %d keywords, got %d: %v",
				test.input, test.expected, len(keywords), keywords)
		}
	}
}

// TestExtractFilePaths 测试文件路径提取
func TestExtractFilePaths(t *testing.T) {
	tests := []struct {
		input    string
		expected int // 期望的路径数量下限
	}{
		{"读取文件 C:\\Users\\test\\file.go 的内容", 1},
		{"文件 /home/user/project/main.py 不存在", 1},
		// config.yaml 不匹配正则表达式，因为正则要求有扩展名前缀
		{"创建了 config.yaml 和 README.md 文件", 1},
		{"没有文件路径", 0},
	}

	for _, test := range tests {
		paths := extractFilePaths(test.input)
		if len(paths) < test.expected {
			t.Errorf("input '%s': expected at least %d paths, got %d: %v",
				test.input, test.expected, len(paths), paths)
		}
	}
}

// TestTruncateString 测试字符串截断
func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a long string", 10, "this is a ..."},
		{"exact", 5, "exact"},
	}

	for _, test := range tests {
		result := truncateString(test.input, test.maxLen)
		if result != test.expected {
			t.Errorf("truncateString(%q, %d) = %q, expected %q",
				test.input, test.maxLen, result, test.expected)
		}
	}
}

// TestParseToolArgs 测试工具参数解析
func TestParseToolArgs(t *testing.T) {
	// 有效JSON
	args, err := parseToolArgs(`{"key": "value", "number": 123}`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if args["key"] != "value" {
		t.Errorf("expected key=value, got %v", args["key"])
	}

	// 空字符串
	args, err = parseToolArgs("")
	if err != nil {
		t.Errorf("unexpected error for empty string: %v", err)
	}

	// 无效JSON
	_, err = parseToolArgs(`{invalid}`)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// TestConcurrentReview 测试并发审查
func TestConcurrentReview(t *testing.T) {
	reviewer := NewReviewer(nil)

	done := make(chan bool)

	// 并发执行审查
	for i := 0; i < 10; i++ {
		go func(id int) {
			toolCall := model.ToolCall{
				ID:   "tc-" + string(rune(id)),
				Type: "function",
				Function: model.FunctionCall{
					Name:      "test_tool",
					Arguments: `{"id": ` + string(rune('0'+id)) + `}`,
				},
			}
			reviewer.ReviewOutput("", []model.ToolCall{toolCall}, nil)
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 检查历史记录
	history := reviewer.GetActionHistory()
	if len(history) != 10 {
		t.Errorf("expected 10 history items, got %d", len(history))
	}
}
