// Package agent 实现Agent核心循环和业务逻辑
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogEntry 日志条目
// 每个日志条目对应一个JSON对象，使用JSON Lines格式存储
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`           // 时间戳
	Type      string                 `json:"type"`                // 类型：message, tool_call, tool_result, review, verification, error
	Content   string                 `json:"content"`             // 内容
	Metadata  map[string]interface{} `json:"metadata,omitempty"`  // 元数据
}

// AgentLogger Agent运行日志记录器
// 负责记录Agent运行过程中的所有关键事件，包括消息、工具调用、审查结果等
type AgentLogger struct {
	logFile   *os.File      // 日志文件
	logPath   string        // 日志文件路径
	mu        sync.Mutex    // 互斥锁，确保并发安全
	startTime time.Time     // 开始时间
	taskID    string        // 任务ID
}

// NewAgentLogger 创建新的日志记录器
// logPath: 日志文件路径，如果为空则不记录日志
func NewAgentLogger(logPath string) (*AgentLogger, error) {
	if logPath == "" {
		return nil, nil // 不记录日志
	}

	// 确保日志目录存在
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 打开或创建日志文件
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}

	logger := &AgentLogger{
		logFile:   logFile,
		logPath:   logPath,
		startTime: time.Now(),
	}

	// 写入会话开始标记
	logger.writeEntry(&LogEntry{
		Timestamp: time.Now(),
		Type:      "session_start",
		Content:   "Agent会话开始",
		Metadata: map[string]interface{}{
			"log_path": logPath,
		},
	})

	return logger, nil
}

// SetTaskID 设置任务ID
func (l *AgentLogger) SetTaskID(taskID string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.taskID = taskID
}

// LogMessage 记录消息
// role: 消息角色（system, user, assistant）
// content: 消息内容
func (l *AgentLogger) LogMessage(role, content string) {
	if l == nil {
		return
	}

	l.writeEntry(&LogEntry{
		Timestamp: time.Now(),
		Type:      "message",
		Content:   content,
		Metadata: map[string]interface{}{
			"role": role,
		},
	})
}

// LogToolCall 记录工具调用
// name: 工具名称
// args: 工具参数（JSON字符串）
func (l *AgentLogger) LogToolCall(name, args string) {
	if l == nil {
		return
	}

	l.writeEntry(&LogEntry{
		Timestamp: time.Now(),
		Type:      "tool_call",
		Content:   fmt.Sprintf("调用工具: %s", name),
		Metadata: map[string]interface{}{
			"tool_name": name,
			"arguments": args,
		},
	})
}

// LogToolResult 记录工具执行结果
// name: 工具名称
// result: 执行结果
// success: 是否成功
func (l *AgentLogger) LogToolResult(name, result string, success bool) {
	if l == nil {
		return
	}

	status := "成功"
	if !success {
		status = "失败"
	}

	l.writeEntry(&LogEntry{
		Timestamp: time.Now(),
		Type:      "tool_result",
		Content:   fmt.Sprintf("工具执行%s: %s", status, name),
		Metadata: map[string]interface{}{
			"tool_name": name,
			"result":    result,
			"success":   success,
		},
	})
}

// LogReview 记录审查结果
// result: 审查结果
func (l *AgentLogger) LogReview(result *ReviewResult) {
	if l == nil || result == nil {
		return
	}

	// 构建问题摘要
	issueCount := len(result.Issues)
	issues := make([]map[string]interface{}, 0, issueCount)
	for _, issue := range result.Issues {
		issues = append(issues, map[string]interface{}{
			"type":        string(issue.Type),
			"severity":    issue.Severity,
			"description": issue.Description,
		})
	}

	l.writeEntry(&LogEntry{
		Timestamp: time.Now(),
		Type:      "review",
		Content:   fmt.Sprintf("审查结果: %s - %s", result.Status, result.Summary),
		Metadata: map[string]interface{}{
			"status":      string(result.Status),
			"summary":     result.Summary,
			"issue_count": issueCount,
			"issues":      issues,
		},
	})
}

// LogVerification 记录校验结果
// result: 校验结果
func (l *AgentLogger) LogVerification(result *VerifyResult) {
	if l == nil || result == nil {
		return
	}

	// 构建问题摘要
	issueCount := len(result.Issues)
	issues := make([]map[string]interface{}, 0, issueCount)
	for _, issue := range result.Issues {
		issues = append(issues, map[string]interface{}{
			"type":        string(issue.Type),
			"severity":    issue.Severity,
			"description": issue.Description,
			"file_path":   issue.FilePath,
		})
	}

	l.writeEntry(&LogEntry{
		Timestamp: time.Now(),
		Type:      "verification",
		Content:   fmt.Sprintf("校验结果: %s - %s", result.Status, result.Summary),
		Metadata: map[string]interface{}{
			"status":      string(result.Status),
			"summary":     result.Summary,
			"passed":      result.Passed,
			"failed":      result.Failed,
			"issue_count": issueCount,
			"issues":      issues,
		},
	})
}

// LogError 记录错误
// err: 错误信息
func (l *AgentLogger) LogError(err string) {
	if l == nil {
		return
	}

	l.writeEntry(&LogEntry{
		Timestamp: time.Now(),
		Type:      "error",
		Content:   err,
	})
}

// LogIteration 记录迭代
// iteration: 迭代次数
// phase: 当前阶段
func (l *AgentLogger) LogIteration(iteration int, phase string) {
	if l == nil {
		return
	}

	l.writeEntry(&LogEntry{
		Timestamp: time.Now(),
		Type:      "iteration",
		Content:   fmt.Sprintf("迭代 #%d - %s", iteration, phase),
		Metadata: map[string]interface{}{
			"iteration": iteration,
			"phase":     phase,
		},
	})
}

// LogTaskStart 记录任务开始
// taskGoal: 任务目标
func (l *AgentLogger) LogTaskStart(taskGoal string) {
	if l == nil {
		return
	}

	l.writeEntry(&LogEntry{
		Timestamp: time.Now(),
		Type:      "task_start",
		Content:   "任务开始执行",
		Metadata: map[string]interface{}{
			"task_goal": taskGoal,
			"task_id":   l.taskID,
		},
	})
}

// LogTaskEnd 记录任务结束
// status: 任务状态
// iterations: 迭代次数
// duration: 执行时长
func (l *AgentLogger) LogTaskEnd(status string, iterations int, duration time.Duration) {
	if l == nil {
		return
	}

	l.writeEntry(&LogEntry{
		Timestamp: time.Now(),
		Type:      "task_end",
		Content:   fmt.Sprintf("任务结束: %s", status),
		Metadata: map[string]interface{}{
			"status":     status,
			"iterations": iterations,
			"duration":   duration.String(),
			"task_id":    l.taskID,
		},
	})
}

// LogCorrection 记录修正指令
// instruction: 修正指令内容
// reason: 修正原因
func (l *AgentLogger) LogCorrection(instruction, reason string) {
	if l == nil {
		return
	}

	l.writeEntry(&LogEntry{
		Timestamp: time.Now(),
		Type:      "correction",
		Content:   "注入修正指令",
		Metadata: map[string]interface{}{
			"instruction": instruction,
			"reason":      reason,
		},
	})
}

// writeEntry 写入日志条目
// 内部方法，确保线程安全
func (l *AgentLogger) writeEntry(entry *LogEntry) {
	if l == nil || l.logFile == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 序列化为JSON
	data, err := json.Marshal(entry)
	if err != nil {
		// 序列化失败，写入错误信息
		errorMsg := fmt.Sprintf(`{"timestamp":"%s","type":"error","content":"日志序列化失败: %s"}`,
			time.Now().Format(time.RFC3339), err.Error())
		l.logFile.WriteString(errorMsg + "\n")
		return
	}

	// 写入文件（每行一个JSON对象）
	l.logFile.WriteString(string(data) + "\n")

	// 刷新到磁盘
	l.logFile.Sync()
}

// Close 关闭日志文件
func (l *AgentLogger) Close() error {
	if l == nil || l.logFile == nil {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 写入会话结束标记
	duration := time.Since(l.startTime)
	endEntry := &LogEntry{
		Timestamp: time.Now(),
		Type:      "session_end",
		Content:   "Agent会话结束",
		Metadata: map[string]interface{}{
			"duration": duration.String(),
			"task_id":  l.taskID,
		},
	}

	data, _ := json.Marshal(endEntry)
	l.logFile.WriteString(string(data) + "\n")

	// 关闭文件
	err := l.logFile.Close()
	l.logFile = nil
	return err
}

// GetLogPath 获取日志文件路径
func (l *AgentLogger) GetLogPath() string {
	if l == nil {
		return ""
	}
	return l.logPath
}

// GetDuration 获取运行时长
func (l *AgentLogger) GetDuration() time.Duration {
	if l == nil {
		return 0
	}
	return time.Since(l.startTime)
}
