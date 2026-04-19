package gui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Attect/MukaAI/internal/agent"
)

// StreamBridge 将 agent.StreamHandler 接口桥接到 Wails 事件系统
// 实现 StreamHandler 接口，将所有流式事件转发为 Wails 前端事件
// 同时更新 App 中的对话状态，保证前端数据一致性
// 同时将对话内容写入日志文件，便于 CLI 风格监控和后续分析
type StreamBridge struct {
	app *App

	// 对话日志文件
	logFile     *os.File
	logMu       sync.Mutex
	logPath     string
	termStyle   bool  // 是否使用终端 ANSI 颜色码
	maxLogSize  int64 // 最大日志文件大小（字节），默认 10MB
	maxLogFiles int   // 保留的日志文件数量，默认 5
}

// NewStreamBridge 创建新的 StreamBridge 实例
func NewStreamBridge(app *App) *StreamBridge {
	return &StreamBridge{
		app:         app,
		maxLogSize:  10 * 1024 * 1024, // 10MB
		maxLogFiles: 5,
	}
}

// SetContext 设置 Wails 上下文
// 保留此方法以兼容外部调用（如 cmd/agentplus/gui.go 中的 OnStartup 回调）
// 实际事件发射现在通过 App 的 EventEmitter 进行，此方法仅更新 App 的上下文和发射器
func (b *StreamBridge) SetContext(ctx context.Context) {
	b.app.mu.Lock()
	defer b.app.mu.Unlock()
	b.app.ctx = ctx
	// 同时更新 EventEmitter 为新的 WailsEventEmitter
	b.app.eventEmitter = NewWailsEventEmitter(ctx)
}

// InitConversationLog 初始化对话日志文件
// logDir: 日志文件存放目录
// 启动新对话时调用，创建带时间戳的日志文件
func (b *StreamBridge) InitConversationLog(logDir string) error {
	b.logMu.Lock()
	defer b.logMu.Unlock()

	// 关闭之前的日志文件
	if b.logFile != nil {
		b.logFile.Close()
		b.logFile = nil
	}

	// 确保目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败：%w", err)
	}

	// 创建带时间戳的日志文件
	timestamp := time.Now().Format("20060102-150405")
	logPath := filepath.Join(logDir, fmt.Sprintf("conversation-%s.log", timestamp))

	f, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("创建日志文件失败：%w", err)
	}

	b.logFile = f
	b.logPath = logPath
	b.termStyle = true // 默认启用终端颜色码

	// 写入头部信息
	b.writeLog("%s", "\n╔══════════════════════════════════════════╗\n")
	b.writeLog("%s", "║  MukaAI 对话日志 - GUI 模式              ║\n")
	b.writeLog("%s", fmt.Sprintf("║  开始时间：%s           ║\n", time.Now().Format("2006-01-02 15:04:05")))
	b.writeLog("%s", "╚══════════════════════════════════════════╝\n\n")

	return nil
}

// CloseConversationLog 关闭对话日志文件
func (b *StreamBridge) CloseConversationLog() {
	b.logMu.Lock()
	defer b.logMu.Unlock()

	if b.logFile != nil {
		b.writeLog("%s", "\n╔══════════════════════════════════════════╗\n")
		b.writeLog("%s", fmt.Sprintf("║  结束时间：%s           ║\n", time.Now().Format("2006-01-02 15:04:05")))
		b.writeLog("%s", "╚══════════════════════════════════════════╝\n")
		b.logFile.Close()
		b.logFile = nil
	}
}

// GetLogPath 获取当前日志文件路径
func (b *StreamBridge) GetLogPath() string {
	b.logMu.Lock()
	defer b.logMu.Unlock()
	return b.logPath
}

// writeLog 写入日志（内部方法，调用方需持有 logMu 锁）
func (b *StreamBridge) writeLog(format string, args ...interface{}) {
	if b.logFile == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	b.logFile.WriteString(msg)
	b.logFile.Sync() // 实时刷盘，确保日志不丢失
}

// checkAndRotateLog 检查并执行日志文件轮转（内部方法）
// 当日志文件大小超过 maxLogSize 时触发轮转
func (b *StreamBridge) checkAndRotateLog() error {
	if b.logFile == nil {
		return nil
	}

	fileInfo, err := b.logFile.Stat()
	if err != nil {
		return fmt.Errorf("获取日志文件信息失败：%w", err)
	}

	// 如果文件超过最大大小，进行轮转
	if fileInfo.Size() >= b.maxLogSize {
		return b.rotateLog()
	}

	return nil
}

// rotateLog 执行日志文件轮转（内部方法）
func (b *StreamBridge) rotateLog() error {
	// 关闭当前日志文件
	if b.logFile != nil {
		b.logFile.Close()
	}

	// 重命名当前日志文件（添加时间戳）
	oldPath := b.logPath
	newPath := fmt.Sprintf("%s.%s", oldPath, time.Now().Format("20060102150405"))

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("重命名日志文件失败：%w", err)
	}

	b.logPath = newPath

	// 清理旧的日志文件，保留最近的文件
	b.cleanupOldLogs()

	// 创建新的日志文件
	return b.createLogFile()
}

// createLogFile 创建新的日志文件（内部方法）
func (b *StreamBridge) createLogFile() error {
	timestamp := time.Now().Format("20060102-150405")
	logPath := filepath.Join(filepath.Dir(b.logPath), fmt.Sprintf("conversation-%s.log", timestamp))

	f, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("创建新日志文件失败：%w", err)
	}

	b.logFile = f
	b.logPath = logPath

	// 写入新的头部信息
	b.writeLog("%s", "\n╔══════════════════════════════════════════╗\n")
	b.writeLog("%s", "║  MukaAI 对话日志 - GUI 模式（轮转）      ║\n")
	b.writeLog("%s", fmt.Sprintf("║  开始时间：%s           ║\n", time.Now().Format("2006-01-02 15:04:05")))
	b.writeLog("%s", "╚══════════════════════════════════════════╝\n\n")

	return nil
}

// cleanupOldLogs 清理旧的日志文件（内部方法）
func (b *StreamBridge) cleanupOldLogs() {
	logDir := filepath.Dir(b.logPath)
	prefix := "conversation-"

	files, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	var logFiles []os.DirEntry
	for _, f := range files {
		if !f.IsDir() && len(f.Name()) > len(prefix) && f.Name()[:len(prefix)] == prefix {
			logFiles = append(logFiles, f)
		}
	}

	// 如果文件数量超过限制，删除最旧的文件
	if len(logFiles) > b.maxLogFiles {
		// 按名称排序（时间戳在文件名中，字母顺序即时间顺序）
		for i := 0; i < len(logFiles)-b.maxLogFiles; i++ {
			oldFile := filepath.Join(logDir, logFiles[i].Name())
			os.Remove(oldFile)
		}
	}
}

// OnThinking 处理思考内容块
// 将思考内容追加到当前消息的 thinking 字段，并发射 stream:thinking 事件
func (b *StreamBridge) OnThinking(chunk string) {
	b.app.mu.Lock()
	conv := b.app.getActiveConversation()
	if conv != nil {
		if conv.currentMessage == nil {
			conv.currentMessage = &message{
				role:      "assistant",
				timestamp: time.Now(),
			}
		}
		conv.currentMessage.thinking += chunk
		conv.currentMessage.isStreaming = true
		conv.currentMessage.streamingType = "thinking"
	}
	b.app.mu.Unlock()
	b.app.emit("stream:thinking", chunk)
	b.app.emit("conversation:updated", b.app.GetConversationData())

	// 写入日志
	b.logMu.Lock()
	defer b.logMu.Unlock()
	logEntry := fmt.Sprintf("[THINKING] %s", chunk)
	b.writeLog("%s", logEntry)
}

// OnContent 处理正文内容块
// 将正文内容追加到当前消息的 content 字段，并发射 stream:content 事件
func (b *StreamBridge) OnContent(chunk string) {
	b.app.mu.Lock()
	conv := b.app.getActiveConversation()
	if conv != nil {
		if conv.currentMessage == nil {
			conv.currentMessage = &message{
				role:      "assistant",
				timestamp: time.Now(),
			}
		}
		conv.currentMessage.content += chunk
		conv.currentMessage.isStreaming = true
		conv.currentMessage.streamingType = "content"
	}
	b.app.mu.Unlock()
	b.app.emit("stream:content", chunk)
	b.app.emit("conversation:updated", b.app.GetConversationData())

	// 写入日志
	b.logMu.Lock()
	defer b.logMu.Unlock()
	logEntry := fmt.Sprintf("[CONTENT] %s", chunk)
	b.writeLog("%s", logEntry)
}

// OnToolCall 处理工具调用
// 将工具调用信息更新到当前消息的 toolCalls 列表，并发射 stream:toolcall 事件
// 如果同一 ID 的工具调用已存在，则更新其内容（流式参数拼接场景）
func (b *StreamBridge) OnToolCall(call agent.ToolCallInfo, isComplete bool) {
	b.app.mu.Lock()
	conv := b.app.getActiveConversation()
	if conv != nil {
		if conv.currentMessage == nil {
			conv.currentMessage = &message{
				role:      "assistant",
				timestamp: time.Now(),
			}
		}
		tc := ToolCall{
			ID:          call.ID,
			Name:        call.Name,
			Arguments:   call.Arguments,
			IsComplete:  isComplete,
			Result:      call.Result,
			ResultError: call.ResultError,
		}
		// 查找是否已存在同 ID 的工具调用，存在则更新（流式参数拼接）
		found := false
		for i, existing := range conv.currentMessage.toolCalls {
			if existing.ID == call.ID {
				conv.currentMessage.toolCalls[i] = tc
				found = true
				break
			}
		}
		if !found {
			conv.currentMessage.toolCalls = append(conv.currentMessage.toolCalls, tc)
		}
		conv.currentMessage.isStreaming = true
		conv.currentMessage.streamingType = "tool"
	}
	b.app.mu.Unlock()

	eventData := map[string]interface{}{
		"id":          call.ID,
		"name":        call.Name,
		"arguments":   call.Arguments,
		"isComplete":  isComplete,
		"result":      call.Result,
		"resultError": call.ResultError,
	}
	b.app.emit("stream:toolcall", eventData)
	b.app.emit("conversation:updated", b.app.GetConversationData())

	// 写入日志
	b.logMu.Lock()
	defer b.logMu.Unlock()

	logEntry := fmt.Sprintf("\n[TOOL_CALL] %s\n", call.Name)
	if call.ID != "" {
		logEntry += fmt.Sprintf("  ID: %s\n", call.ID)
	}
	if call.Arguments != "" {
		logEntry += fmt.Sprintf("  Arguments: %s\n", call.Arguments)
	}
	if isComplete {
		logEntry += "  Status: Complete\n"
	} else {
		logEntry += "  Status: In Progress\n"
	}

	b.writeLog("%s", logEntry)

	// 检查是否需要轮转日志文件
	b.checkAndRotateLog()
}

// OnToolResult 处理工具执行结果
// 更新当前消息中对应工具调用的结果，并发射 stream:toolresult 事件
func (b *StreamBridge) OnToolResult(result agent.ToolCallInfo) {
	b.app.mu.Lock()
	conv := b.app.getActiveConversation()
	if conv != nil && conv.currentMessage != nil {
		for i, tc := range conv.currentMessage.toolCalls {
			if tc.ID == result.ID {
				conv.currentMessage.toolCalls[i].Result = result.Result
				conv.currentMessage.toolCalls[i].ResultError = result.ResultError
				break
			}
		}
	}
	b.app.mu.Unlock()

	eventData := map[string]interface{}{
		"id":          result.ID,
		"name":        result.Name,
		"result":      result.Result,
		"resultError": result.ResultError,
	}
	b.app.emit("stream:toolresult", eventData)
	b.app.emit("conversation:updated", b.app.GetConversationData())

	// 写入日志
	b.logMu.Lock()
	defer b.logMu.Unlock()

	logEntry := fmt.Sprintf("\n[TOOL_RESULT] %s\n", result.Name)
	if result.ID != "" {
		logEntry += fmt.Sprintf("  ID: %s\n", result.ID)
	}
	logEntry += fmt.Sprintf("  Success: %v\n", result.ResultError == "")

	if result.Result != "" {
		output := result.Result
		// 对于长输出，添加截断标记
		if len(output) > 1000 {
			output = output[:1000] + "\n... [output truncated]"
		}
		logEntry += fmt.Sprintf("  Output:\n%s\n", output)
	}

	if result.ResultError != "" {
		logEntry += fmt.Sprintf("  Error: %s\n", result.ResultError)
	}

	b.writeLog("%s", logEntry)
}

// OnComplete 处理单次推理完成
// 固化当前消息到 messages 列表，并创建新的 currentMessage 用于下一次迭代
// 每次模型推理成为一个独立的消息块
func (b *StreamBridge) OnComplete(usage int) {
	b.app.mu.Lock()
	conv := b.app.getActiveConversation()
	if conv != nil && conv.currentMessage != nil {
		// 标记当前消息流式结束
		conv.currentMessage.isStreaming = false
		conv.currentMessage.tokenUsage = usage
		conv.currentMessage.timestamp = time.Now()

		// 只有当消息有实际内容时才固化，避免保存空消息
		if conv.currentMessage.content != "" || conv.currentMessage.thinking != "" || len(conv.currentMessage.toolCalls) > 0 {
			conv.messages = append(conv.messages, conv.currentMessage)
			b.app.saveConv(conv) // 持久化：保存固化后的消息
		}

		// 更新 token 统计
		conv.tokenUsage += usage
		b.app.totalTokens += usage
		b.app.inferenceCount++

		// 创建新的 currentMessage 用于下一次迭代
		// 但保持 isStreaming = true，直到整个任务完成
		conv.currentMessage = &message{
			role:      "assistant",
			timestamp: time.Now(),
		}
	}
	b.app.mu.Unlock()

	b.app.emit("stream:complete", map[string]interface{}{
		"usage": usage,
	})
	b.app.emit("tokenstats:updated", b.app.GetTokenStats())
	b.app.emit("conversation:updated", b.app.GetConversationData())

	// 写入日志
	b.logMu.Lock()
	defer b.logMu.Unlock()
	logEntry := fmt.Sprintf("\n[COMPLETE] Token Usage: %d\n", usage)
	b.writeLog("%s", logEntry)
}

// OnError 处理错误
// 当发生实际错误时，将当前流式消息（如果有内容）固化并发射 stream:error 事件
// 注意：err 为 nil 时不发射 error 事件，仅更新对话状态
func (b *StreamBridge) OnError(err error) {
	b.app.mu.Lock()
	if err != nil {
		if conv := b.app.getActiveConversation(); conv != nil && conv.currentMessage != nil {
			conv.currentMessage.isStreaming = false
			// 只有当消息有实际内容时才保存，避免保存空消息
			if conv.currentMessage.content != "" || conv.currentMessage.thinking != "" || len(conv.currentMessage.toolCalls) > 0 {
				conv.messages = append(conv.messages, conv.currentMessage)
				b.app.saveConv(conv) // 持久化：保存错误时固化的消息
			}
			conv.currentMessage = nil
		}
	}
	b.app.isStreaming = false
	b.app.mu.Unlock()

	if err != nil {
		b.app.emit("stream:error", err.Error())
	}
	b.app.emit("conversation:updated", b.app.GetConversationData())

	// 写入日志
	b.logMu.Lock()
	defer b.logMu.Unlock()
	logEntry := fmt.Sprintf("\n[ERROR] %v\n", err)
	logEntry += fmt.Sprintf("  Time: %s\n", time.Now().Format(time.RFC3339))
	b.writeLog("%s", logEntry)
}

// OnTaskDone 处理任务完成
// 当整个 Agent 任务（包括所有迭代）完成后调用
// 固化最后一个消息（如果有内容），重置流式状态，并发射 stream:done 事件
func (b *StreamBridge) OnTaskDone() {
	b.app.mu.Lock()
	conv := b.app.getActiveConversation()
	if conv != nil && conv.currentMessage != nil {
		// 固化最后一个消息（如果有内容）
		if conv.currentMessage.content != "" || conv.currentMessage.thinking != "" || len(conv.currentMessage.toolCalls) > 0 {
			conv.currentMessage.isStreaming = false
			conv.messages = append(conv.messages, conv.currentMessage)
			b.app.saveConv(conv) // 持久化：保存最后的 assistant 消息
		}
		conv.currentMessage = nil
	}
	b.app.isStreaming = false
	b.app.mu.Unlock()

	b.app.emit("stream:done")
	b.app.emit("conversation:updated", b.app.GetConversationData())

	// 写入日志
	b.logMu.Lock()
	defer b.logMu.Unlock()
	logEntry := fmt.Sprintf("\n[TASK_DONE] 任务完成\n")
	logEntry += fmt.Sprintf("  Time: %s\n", time.Now().Format(time.RFC3339))
	b.writeLog("%s", logEntry)
}

// OnSupervisorResult 处理监督结果
// 实现 agent.SupervisorResultHandler 接口
// 将 Supervisor 检查结果作为系统事件推送到前端
func (b *StreamBridge) OnSupervisorResult(result *agent.SupervisionResult) {
	b.app.mu.RLock()
	emitter := b.app.eventEmitter
	b.app.mu.RUnlock()
	if emitter == nil {
		return
	}

	eventData := map[string]interface{}{
		"status":            result.Status,
		"summary":           result.Summary,
		"intervention_type": result.InterventionType,
		"issues_count":      len(result.Issues),
	}

	// 简化 issues 数据，避免传输过大
	issues := make([]map[string]interface{}, 0, len(result.Issues))
	for _, issue := range result.Issues {
		issues = append(issues, map[string]interface{}{
			"type":        issue.Type,
			"severity":    issue.Severity,
			"description": issue.Description,
		})
	}
	eventData["issues"] = issues

	b.app.emit("supervisor:result", eventData)

	// 写入日志
	b.logMu.Lock()
	defer b.logMu.Unlock()
	logEntry := fmt.Sprintf("\n[SUPERVISOR] Status: %s\n", result.Status)
	logEntry += fmt.Sprintf("  Summary: %s\n", result.Summary)
	if result.InterventionType != "" {
		logEntry += fmt.Sprintf("  Intervention: %s\n", result.InterventionType)
	}
	logEntry += fmt.Sprintf("  Issues Count: %d\n", len(result.Issues))
	b.writeLog("%s", logEntry)
}

// OnCompression 处理上下文压缩
// 实现 agent.StreamHandler 接口
// 将压缩信息推送到前端显示
func (b *StreamBridge) OnCompression(originalCount, compressedCount, originalTokens, compressedTokens int, summary string) {
	// 计算压缩比
	var compressionRatio float64
	if originalTokens > 0 {
		compressionRatio = float64(compressedTokens) / float64(originalTokens) * 100
	}

	// 发送事件到前端
	eventData := map[string]interface{}{
		"originalCount":    originalCount,
		"compressedCount":  compressedCount,
		"originalTokens":   originalTokens,
		"compressedTokens": compressedTokens,
		"compressionRatio": compressionRatio,
		"summary":          summary,
		"timestamp":        time.Now().Format(time.RFC3339),
	}
	b.app.emit("stream:compression", eventData)

	// 写入日志
	b.logMu.Lock()
	defer b.logMu.Unlock()
	logEntry := fmt.Sprintf("\n[COMPRESSION] 上下文压缩\n")
	logEntry += fmt.Sprintf("  消息数量: %d -> %d\n", originalCount, compressedCount)
	logEntry += fmt.Sprintf("  Token数量: %d -> %d\n", originalTokens, compressedTokens)
	logEntry += fmt.Sprintf("  压缩比: %.1f%%\n", compressionRatio)
	if summary != "" {
		// 限制摘要长度
		if len(summary) > 500 {
			summary = summary[:500] + "..."
		}
		logEntry += fmt.Sprintf("  摘要: %s\n", summary)
	}
	b.writeLog("%s", logEntry)
}
