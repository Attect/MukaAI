// Package agent 实现Agent核心循环和业务逻辑
package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewAgentLogger(t *testing.T) {
	// 测试创建日志记录器
	t.Run("创建日志记录器成功", func(t *testing.T) {
		// 创建临时目录
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test.log")

		logger, err := NewAgentLogger(logPath)
		if err != nil {
			t.Fatalf("创建日志记录器失败: %v", err)
		}
		if logger == nil {
			t.Fatal("日志记录器不应为nil")
		}
		defer logger.Close()

		// 检查日志文件是否创建
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Error("日志文件未创建")
		}
	})

	t.Run("空路径返回nil", func(t *testing.T) {
		logger, err := NewAgentLogger("")
		if err != nil {
			t.Errorf("空路径应该返回nil而不是错误: %v", err)
		}
		if logger != nil {
			t.Error("空路径应该返回nil")
		}
	})

	t.Run("自动创建日志目录", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "subdir", "deep", "test.log")

		logger, err := NewAgentLogger(logPath)
		if err != nil {
			t.Fatalf("创建日志记录器失败: %v", err)
		}
		if logger == nil {
			t.Fatal("日志记录器不应为nil")
		}
		defer logger.Close()

		// 检查日志文件是否创建
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Error("日志文件未创建")
		}
	})
}

func TestAgentLogger_LogMessage(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewAgentLogger(logPath)
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}
	defer logger.Close()

	// 记录消息
	logger.LogMessage("user", "测试消息")

	// 读取日志文件
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	// 验证日志内容
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 2 {
		t.Fatalf("日志行数不足，期望至少2行，实际%d行", len(lines))
	}

	// 解析消息日志
	var entry LogEntry
	if err := json.Unmarshal([]byte(lines[1]), &entry); err != nil {
		t.Fatalf("解析日志失败: %v", err)
	}

	if entry.Type != "message" {
		t.Errorf("日志类型错误，期望message，实际%s", entry.Type)
	}
	if entry.Content != "测试消息" {
		t.Errorf("日志内容错误，期望'测试消息'，实际'%s'", entry.Content)
	}
	if entry.Metadata["role"] != "user" {
		t.Errorf("元数据角色错误，期望user，实际%v", entry.Metadata["role"])
	}
}

func TestAgentLogger_LogToolCall(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewAgentLogger(logPath)
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}
	defer logger.Close()

	// 记录工具调用
	logger.LogToolCall("read_file", `{"file_path": "/test.txt"}`)

	// 读取日志文件
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	// 解析日志
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var entry LogEntry
	if err := json.Unmarshal([]byte(lines[1]), &entry); err != nil {
		t.Fatalf("解析日志失败: %v", err)
	}

	if entry.Type != "tool_call" {
		t.Errorf("日志类型错误，期望tool_call，实际%s", entry.Type)
	}
	if entry.Metadata["tool_name"] != "read_file" {
		t.Errorf("工具名称错误，期望read_file，实际%v", entry.Metadata["tool_name"])
	}
}

func TestAgentLogger_LogToolResult(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewAgentLogger(logPath)
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}
	defer logger.Close()

	// 记录工具结果
	logger.LogToolResult("read_file", "文件内容", true)

	// 读取日志文件
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	// 解析日志
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var entry LogEntry
	if err := json.Unmarshal([]byte(lines[1]), &entry); err != nil {
		t.Fatalf("解析日志失败: %v", err)
	}

	if entry.Type != "tool_result" {
		t.Errorf("日志类型错误，期望tool_result，实际%s", entry.Type)
	}
	if entry.Metadata["success"] != true {
		t.Errorf("成功标志错误，期望true，实际%v", entry.Metadata["success"])
	}
}

func TestAgentLogger_LogReview(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewAgentLogger(logPath)
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}
	defer logger.Close()

	// 创建审查结果
	reviewResult := &ReviewResult{
		Status:    ReviewStatusPass,
		Issues:    []ReviewIssue{},
		Timestamp: time.Now(),
		Summary:   "审查通过",
	}

	// 记录审查结果
	logger.LogReview(reviewResult)

	// 读取日志文件
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	// 解析日志
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var entry LogEntry
	if err := json.Unmarshal([]byte(lines[1]), &entry); err != nil {
		t.Fatalf("解析日志失败: %v", err)
	}

	if entry.Type != "review" {
		t.Errorf("日志类型错误，期望review，实际%s", entry.Type)
	}
	if entry.Metadata["status"] != "pass" {
		t.Errorf("审查状态错误，期望pass，实际%v", entry.Metadata["status"])
	}
}

func TestAgentLogger_LogVerification(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewAgentLogger(logPath)
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}
	defer logger.Close()

	// 创建校验结果
	verifyResult := &VerifyResult{
		Status:    VerifyStatusPass,
		Issues:    []VerifyIssue{},
		Timestamp: time.Now(),
		Summary:   "校验通过",
		Passed:    5,
		Failed:    0,
	}

	// 记录校验结果
	logger.LogVerification(verifyResult)

	// 读取日志文件
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	// 解析日志
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var entry LogEntry
	if err := json.Unmarshal([]byte(lines[1]), &entry); err != nil {
		t.Fatalf("解析日志失败: %v", err)
	}

	if entry.Type != "verification" {
		t.Errorf("日志类型错误，期望verification，实际%s", entry.Type)
	}
	if entry.Metadata["passed"].(float64) != 5 {
		t.Errorf("通过数量错误，期望5，实际%v", entry.Metadata["passed"])
	}
}

func TestAgentLogger_LogError(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewAgentLogger(logPath)
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}
	defer logger.Close()

	// 记录错误
	logger.LogError("测试错误信息")

	// 读取日志文件
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	// 解析日志
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var entry LogEntry
	if err := json.Unmarshal([]byte(lines[1]), &entry); err != nil {
		t.Fatalf("解析日志失败: %v", err)
	}

	if entry.Type != "error" {
		t.Errorf("日志类型错误，期望error，实际%s", entry.Type)
	}
	if entry.Content != "测试错误信息" {
		t.Errorf("错误内容错误，期望'测试错误信息'，实际'%s'", entry.Content)
	}
}

func TestAgentLogger_LogIteration(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewAgentLogger(logPath)
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}
	defer logger.Close()

	// 记录迭代
	logger.LogIteration(1, "processing")

	// 读取日志文件
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	// 解析日志
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var entry LogEntry
	if err := json.Unmarshal([]byte(lines[1]), &entry); err != nil {
		t.Fatalf("解析日志失败: %v", err)
	}

	if entry.Type != "iteration" {
		t.Errorf("日志类型错误，期望iteration，实际%s", entry.Type)
	}
	if entry.Metadata["iteration"].(float64) != 1 {
		t.Errorf("迭代次数错误，期望1，实际%v", entry.Metadata["iteration"])
	}
}

func TestAgentLogger_LogTaskStart(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewAgentLogger(logPath)
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}
	defer logger.Close()

	// 设置任务ID
	logger.SetTaskID("test-task-123")

	// 记录任务开始
	logger.LogTaskStart("测试任务目标")

	// 读取日志文件
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	// 解析日志
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 2 {
		t.Fatalf("日志行数不足，期望至少2行，实际%d行", len(lines))
	}

	// task_start日志应该在第二行（索引1）
	var entry LogEntry
	if err := json.Unmarshal([]byte(lines[1]), &entry); err != nil {
		t.Fatalf("解析日志失败: %v", err)
	}

	if entry.Type != "task_start" {
		t.Errorf("日志类型错误，期望task_start，实际%s", entry.Type)
	}
	if entry.Metadata["task_goal"] != "测试任务目标" {
		t.Errorf("任务目标错误，期望'测试任务目标'，实际%v", entry.Metadata["task_goal"])
	}
}

func TestAgentLogger_LogTaskEnd(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewAgentLogger(logPath)
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}
	defer logger.Close()

	// 记录任务结束
	logger.LogTaskEnd("completed", 10, 5*time.Minute)

	// 读取日志文件
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	// 解析日志
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var entry LogEntry
	if err := json.Unmarshal([]byte(lines[1]), &entry); err != nil {
		t.Fatalf("解析日志失败: %v", err)
	}

	if entry.Type != "task_end" {
		t.Errorf("日志类型错误，期望task_end，实际%s", entry.Type)
	}
	if entry.Metadata["status"] != "completed" {
		t.Errorf("任务状态错误，期望completed，实际%v", entry.Metadata["status"])
	}
	if entry.Metadata["iterations"].(float64) != 10 {
		t.Errorf("迭代次数错误，期望10，实际%v", entry.Metadata["iterations"])
	}
}

func TestAgentLogger_Close(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewAgentLogger(logPath)
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}

	// 关闭日志记录器
	if err := logger.Close(); err != nil {
		t.Errorf("关闭日志记录器失败: %v", err)
	}

	// 读取日志文件
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	// 验证会话结束日志
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 2 {
		t.Fatalf("日志行数不足，期望至少2行，实际%d行", len(lines))
	}

	var entry LogEntry
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &entry); err != nil {
		t.Fatalf("解析日志失败: %v", err)
	}

	if entry.Type != "session_end" {
		t.Errorf("最后的日志类型应该是session_end，实际%s", entry.Type)
	}
}

func TestAgentLogger_GetLogPath(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewAgentLogger(logPath)
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}
	defer logger.Close()

	if logger.GetLogPath() != logPath {
		t.Errorf("日志路径错误，期望%s，实际%s", logPath, logger.GetLogPath())
	}
}

func TestAgentLogger_NilLogger(t *testing.T) {
	// 测试nil日志记录器不会panic
	var logger *AgentLogger

	// 所有方法都不应该panic
	logger.SetTaskID("test")
	logger.LogMessage("user", "test")
	logger.LogToolCall("test", "{}")
	logger.LogToolResult("test", "result", true)
	logger.LogReview(nil)
	logger.LogVerification(nil)
	logger.LogError("test")
	logger.LogIteration(1, "test")
	logger.LogTaskStart("test")
	logger.LogTaskEnd("completed", 1, time.Minute)
	logger.LogCorrection("test", "test")

	if logger.GetLogPath() != "" {
		t.Error("nil日志记录器的路径应该是空字符串")
	}
	if logger.GetDuration() != 0 {
		t.Error("nil日志记录器的时长应该是0")
	}
	if err := logger.Close(); err != nil {
		t.Errorf("nil日志记录器关闭不应该返回错误: %v", err)
	}
}

func TestAgentLogger_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewAgentLogger(logPath)
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}
	defer logger.Close()

	// 并发写入日志
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				logger.LogMessage("user", "并发测试")
				logger.LogToolCall("test", "{}")
				logger.LogError("并发错误")
			}
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 读取日志文件验证
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	// 验证日志行数（至少应该有会话开始 + 3000条日志）
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 3000 {
		t.Errorf("日志行数不足，期望至少3000行，实际%d行", len(lines))
	}
}
