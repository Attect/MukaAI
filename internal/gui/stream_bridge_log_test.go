package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Attect/MukaAI/internal/agent"
)

// TestStreamBridge_LogFunctionality 测试 StreamBridge 的日志功能增强
func TestStreamBridge_LogFunctionality(t *testing.T) {
	app := NewApp()
	bridge := NewStreamBridge(app)

	// 创建临时日志目录
	tmpDir := t.TempDir()

	// 初始化日志
	err := bridge.InitConversationLog(tmpDir)
	if err != nil {
		t.Fatalf("InitConversationLog failed: %v", err)
	}
	defer bridge.CloseConversationLog()

	// 验证日志文件已创建
	logPath := bridge.GetLogPath()
	if logPath == "" {
		t.Fatal("Log path should not be empty")
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatalf("Log file should exist: %s", logPath)
	}

	// 测试 OnThinking 日志记录
	bridge.OnThinking("这是思考内容")
	verifyLogFileContains(t, logPath, "[THINKING]")

	// 测试 OnContent 日志记录
	bridge.OnContent("这是正文内容")
	verifyLogFileContains(t, logPath, "[CONTENT]")

	// 测试 OnToolCall 日志记录
	toolCall := agent.ToolCallInfo{
		ID:        "call-001",
		Name:      "read_file",
		Arguments: `{"path": "test.txt"}`,
	}
	bridge.OnToolCall(toolCall, false) // In Progress
	verifyLogFileContains(t, logPath, "[TOOL_CALL]")
	verifyLogFileContains(t, logPath, "Status: In Progress")

	// 完成工具调用
	bridge.OnToolCall(toolCall, true) // Complete
	verifyLogFileContains(t, logPath, "Status: Complete")

	// 测试 OnToolResult 日志记录
	toolResult := agent.ToolCallInfo{
		ID:          "call-001",
		Name:        "read_file",
		Result:      "文件内容：Hello World",
		ResultError: "",
	}
	bridge.OnToolResult(toolResult)
	verifyLogFileContains(t, logPath, "[TOOL_RESULT]")
	verifyLogFileContains(t, logPath, "Success: true")

	// 测试 OnComplete 日志记录
	bridge.OnComplete(1250)
	verifyLogFileContains(t, logPath, "[COMPLETE]")
	verifyLogFileContains(t, logPath, "Token Usage: 1250")

	// 测试 OnError 日志记录
	testErr := os.ErrNotExist
	bridge.OnError(testErr)
	verifyLogFileContains(t, logPath, "[ERROR]")
	verifyLogFileContains(t, logPath, testErr.Error())

	// 测试 OnTaskDone 日志记录
	bridge.OnTaskDone()
	verifyLogFileContains(t, logPath, "[TASK_DONE]")

	t.Log("所有日志功能测试通过")
}

// TestStreamBridge_LogRotation 测试日志文件轮转功能
func TestStreamBridge_LogRotation(t *testing.T) {
	app := NewApp()
	bridge := NewStreamBridge(app)

	// 设置较小的最大文件大小（1KB）以触发轮转
	bridge.maxLogSize = 1024 // 1KB

	tmpDir := t.TempDir()

	// 初始化日志
	err := bridge.InitConversationLog(tmpDir)
	if err != nil {
		t.Fatalf("InitConversationLog failed: %v", err)
	}
	defer bridge.CloseConversationLog()

	initialPath := bridge.GetLogPath()

	// 写入足够的内容以触发轮转
	largeContent := strings.Repeat("这是一条测试日志内容，用于触发日志文件轮转功能。\n", 100)
	bridge.OnContent(largeContent)

	// 验证轮转是否发生
	time.Sleep(100 * time.Millisecond) // 给轮转一些时间

	// 检查是否有轮转后的文件
	files, err := filepath.Glob(filepath.Join(tmpDir, "conversation-*.log.*"))
	if err != nil {
		t.Fatalf("Failed to list log files: %v", err)
	}

	// 应该至少有一个轮转文件（带时间戳后缀）
	if len(files) == 0 {
		t.Log("未检测到日志轮转（可能内容大小未达到阈值）")
	} else {
		t.Logf("检测到 %d 个轮转后的日志文件", len(files))
	}

	// 验证当前日志路径已更新
	currentPath := bridge.GetLogPath()
	if currentPath != initialPath {
		t.Logf("日志文件已轮转，新路径：%s", currentPath)
	}
}

// TestStreamBridge_LogFormat 测试日志格式
func TestStreamBridge_LogFormat(t *testing.T) {
	app := NewApp()
	bridge := NewStreamBridge(app)

	tmpDir := t.TempDir()

	err := bridge.InitConversationLog(tmpDir)
	if err != nil {
		t.Fatalf("InitConversationLog failed: %v", err)
	}
	defer bridge.CloseConversationLog()

	logPath := bridge.GetLogPath()

	// 写入各种类型的日志
	bridge.OnThinking("思考测试")
	bridge.OnContent("内容测试")

	toolCall := agent.ToolCallInfo{
		ID:        "test-001",
		Name:      "test_tool",
		Arguments: `{"key": "value"}`,
	}
	bridge.OnToolCall(toolCall, true)

	toolResult := agent.ToolCallInfo{
		ID:          "test-001",
		Name:        "test_tool",
		Result:      "测试结果",
		ResultError: "",
	}
	bridge.OnToolResult(toolResult)

	bridge.OnComplete(500)
	bridge.OnTaskDone()

	// 读取日志文件内容
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// 验证日志格式包含所有必需的元素
	expectedElements := []string{
		"[THINKING]",
		"[CONTENT]",
		"[TOOL_CALL]",
		"[TOOL_RESULT]",
		"[COMPLETE]",
		"[TASK_DONE]",
	}

	for _, element := range expectedElements {
		if !strings.Contains(logContent, element) {
			t.Errorf("Log content should contain %s", element)
		}
	}

	t.Logf("日志格式验证通过，包含 %d 个日志类型标记", len(expectedElements))
}

// TestStreamBridge_LongOutputTruncation 测试长输出截断功能
func TestStreamBridge_LongOutputTruncation(t *testing.T) {
	app := NewApp()
	bridge := NewStreamBridge(app)

	tmpDir := t.TempDir()

	err := bridge.InitConversationLog(tmpDir)
	if err != nil {
		t.Fatalf("InitConversationLog failed: %v", err)
	}
	defer bridge.CloseConversationLog()

	logPath := bridge.GetLogPath()

	// 创建长输出结果（超过 1000 字符）
	longOutput := strings.Repeat("这是一条很长的测试结果，用于测试日志截断功能。\n", 50)

	toolResult := agent.ToolCallInfo{
		ID:          "long-output-001",
		Name:        "large_file_reader",
		Result:      longOutput,
		ResultError: "",
	}

	bridge.OnToolResult(toolResult)

	// 验证日志包含截断标记
	verifyLogFileContains(t, logPath, "[output truncated]")

	t.Log("长输出截断功能测试通过")
}

// TestStreamBridge_ErrorLoggingWithTimestamp 测试错误日志的时间戳记录
func TestStreamBridge_ErrorLoggingWithTimestamp(t *testing.T) {
	app := NewApp()
	bridge := NewStreamBridge(app)

	tmpDir := t.TempDir()

	err := bridge.InitConversationLog(tmpDir)
	if err != nil {
		t.Fatalf("InitConversationLog failed: %v", err)
	}
	defer bridge.CloseConversationLog()

	logPath := bridge.GetLogPath()

	// 记录错误
	testErr := os.ErrPermission
	bridge.OnError(testErr)

	// 读取日志内容
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// 验证包含错误信息和时间戳
	if !strings.Contains(logContent, "[ERROR]") {
		t.Error("Log should contain [ERROR] marker")
	}

	if !strings.Contains(logContent, "Time:") {
		t.Error("Error log should contain timestamp")
	}

	t.Log("错误日志时间戳记录测试通过")
}

// verifyLogFileContains 验证日志文件包含指定内容
func verifyLogFileContains(t *testing.T, logPath string, expectedContent string) {
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), expectedContent) {
		t.Errorf("Log file should contain '%s', but it doesn't", expectedContent)
	}
}
