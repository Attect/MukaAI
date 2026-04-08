package tui

import (
	"errors"
	"testing"
	"time"
)

func TestDefaultBatchUpdateConfig(t *testing.T) {
	config := DefaultBatchUpdateConfig()

	if config.BufferDuration <= 0 {
		t.Error("BufferDuration should be positive")
	}
	if config.MaxBufferSize <= 0 {
		t.Error("MaxBufferSize should be positive")
	}
	if config.MinUpdateInterval <= 0 {
		t.Error("MinUpdateInterval should be positive")
	}
}

func TestMessageBuffer_AddThinking(t *testing.T) {
	config := DefaultBatchUpdateConfig()
	buffer := NewMessageBuffer(config)

	// 添加思考内容
	buffer.AddThinking("思考内容1")
	buffer.AddThinking("思考内容2")

	// 刷新并检查结果
	result := buffer.Flush()
	if !result.HasThinking {
		t.Error("Should have thinking content")
	}
	if result.Thinking != "思考内容1思考内容2" {
		t.Errorf("Thinking content mismatch: got %s", result.Thinking)
	}
}

func TestMessageBuffer_AddContent(t *testing.T) {
	config := DefaultBatchUpdateConfig()
	buffer := NewMessageBuffer(config)

	// 添加正文内容
	buffer.AddContent("正文内容1")
	buffer.AddContent("正文内容2")

	// 刷新并检查结果
	result := buffer.Flush()
	if !result.HasContent {
		t.Error("Should have content")
	}
	if result.Content != "正文内容1正文内容2" {
		t.Errorf("Content mismatch: got %s", result.Content)
	}
}

func TestMessageBuffer_AddToolCall(t *testing.T) {
	config := DefaultBatchUpdateConfig()
	buffer := NewMessageBuffer(config)

	// 添加工具调用
	call1 := ToolCall{
		ID:        "call1",
		Name:      "test_tool",
		Arguments: `{"arg": "value"}`,
	}
	buffer.AddToolCall(call1, false)

	// 刷新并检查结果
	result := buffer.Flush()
	if !result.HasToolCalls {
		t.Error("Should have tool calls")
	}
	if len(result.ToolCalls) != 1 {
		t.Errorf("Tool calls count mismatch: got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].ID != "call1" {
		t.Errorf("Tool call ID mismatch: got %s", result.ToolCalls[0].ID)
	}
}

func TestMessageBuffer_AddToolResult(t *testing.T) {
	config := DefaultBatchUpdateConfig()
	buffer := NewMessageBuffer(config)

	// 添加工具结果
	result1 := ToolCall{
		ID:     "call1",
		Result: "执行成功",
	}
	buffer.AddToolResult(result1)

	// 刷新并检查结果
	result := buffer.Flush()
	if !result.HasToolResult {
		t.Error("Should have tool result")
	}
	if len(result.ToolResults) != 1 {
		t.Errorf("Tool results count mismatch: got %d", len(result.ToolResults))
	}
	if result.ToolResults[0].Result != "执行成功" {
		t.Errorf("Tool result mismatch: got %s", result.ToolResults[0].Result)
	}
}

func TestMessageBuffer_AddComplete(t *testing.T) {
	config := DefaultBatchUpdateConfig()
	buffer := NewMessageBuffer(config)

	// 添加完成消息
	buffer.AddComplete(100)

	// 刷新并检查结果
	result := buffer.Flush()
	if len(result.Messages) != 1 {
		t.Errorf("Messages count mismatch: got %d", len(result.Messages))
	}
	if result.Messages[0].Type != "complete" {
		t.Errorf("Message type mismatch: got %s", result.Messages[0].Type)
	}
	if result.Messages[0].Usage != 100 {
		t.Errorf("Usage mismatch: got %d", result.Messages[0].Usage)
	}
}

func TestMessageBuffer_AddError(t *testing.T) {
	config := DefaultBatchUpdateConfig()
	buffer := NewMessageBuffer(config)

	// 添加错误消息
	testErr := errors.New("test error")
	buffer.AddError(testErr)

	// 刷新并检查结果
	result := buffer.Flush()
	if len(result.Messages) != 1 {
		t.Errorf("Messages count mismatch: got %d", len(result.Messages))
	}
	if result.Messages[0].Type != "error" {
		t.Errorf("Message type mismatch: got %s", result.Messages[0].Type)
	}
	if result.Messages[0].Error != testErr {
		t.Errorf("Error mismatch: got %v", result.Messages[0].Error)
	}
}

func TestMessageBuffer_ShouldFlush(t *testing.T) {
	// 使用较短的缓冲时间进行测试
	config := BatchUpdateConfig{
		BufferDuration:    10, // 10ms
		MaxBufferSize:     10,
		EnableBatching:    true,
		MinUpdateInterval: 5,
	}
	buffer := NewMessageBuffer(config)

	// 初始状态不应该刷新
	if buffer.ShouldFlush() {
		t.Error("Should not flush initially")
	}

	// 添加数据
	buffer.AddContent("test")

	// 等待缓冲时间
	time.Sleep(15 * time.Millisecond)

	// 应该刷新
	if !buffer.ShouldFlush() {
		t.Error("Should flush after buffer duration")
	}
}

func TestMessageBuffer_Clear(t *testing.T) {
	config := DefaultBatchUpdateConfig()
	buffer := NewMessageBuffer(config)

	// 添加数据
	buffer.AddThinking("思考")
	buffer.AddContent("内容")
	buffer.AddToolCall(ToolCall{ID: "1", Name: "tool"}, false)

	// 清空
	buffer.Clear()

	// 刷新应该没有数据
	result := buffer.Flush()
	if result.HasData() {
		t.Error("Should have no data after clear")
	}
}

func TestMessageBuffer_MultipleToolCalls(t *testing.T) {
	config := DefaultBatchUpdateConfig()
	buffer := NewMessageBuffer(config)

	// 添加多个工具调用
	call1 := ToolCall{ID: "call1", Name: "tool1"}
	call2 := ToolCall{ID: "call2", Name: "tool2"}
	call3 := ToolCall{ID: "call3", Name: "tool3"}

	buffer.AddToolCall(call1, false)
	buffer.AddToolCall(call2, false)
	buffer.AddToolCall(call3, true)

	// 刷新并检查结果
	result := buffer.Flush()
	if len(result.ToolCalls) != 3 {
		t.Errorf("Tool calls count mismatch: got %d", len(result.ToolCalls))
	}

	// 检查工具调用是否都在
	found := make(map[string]bool)
	for _, tc := range result.ToolCalls {
		found[tc.ID] = true
	}
	if !found["call1"] || !found["call2"] || !found["call3"] {
		t.Error("Missing tool calls")
	}
}

func TestMessageBuffer_UpdateToolCall(t *testing.T) {
	config := DefaultBatchUpdateConfig()
	buffer := NewMessageBuffer(config)

	// 添加工具调用（未完成）
	call1 := ToolCall{ID: "call1", Name: "tool1", Arguments: `{"arg": "value"}`}
	buffer.AddToolCall(call1, false)

	// 更新工具调用（已完成）
	call1Updated := ToolCall{ID: "call1", Name: "tool1", Arguments: `{"arg": "updated"}`}
	buffer.AddToolCall(call1Updated, true)

	// 刷新并检查结果
	result := buffer.Flush()
	if len(result.ToolCalls) != 1 {
		t.Errorf("Tool calls count mismatch: got %d", len(result.ToolCalls))
	}
	if !result.ToolCalls[0].IsComplete {
		t.Error("Tool call should be complete")
	}
	if result.ToolCalls[0].Arguments != `{"arg": "updated"}` {
		t.Errorf("Arguments mismatch: got %s", result.ToolCalls[0].Arguments)
	}
}

func TestFlushResult_HasData(t *testing.T) {
	// 空结果
	result := &FlushResult{}
	if result.HasData() {
		t.Error("Empty result should not have data")
	}

	// 有思考内容
	result = &FlushResult{HasThinking: true, Thinking: "思考"}
	if !result.HasData() {
		t.Error("Should have data with thinking")
	}

	// 有正文内容
	result = &FlushResult{HasContent: true, Content: "内容"}
	if !result.HasData() {
		t.Error("Should have data with content")
	}

	// 有工具调用
	result = &FlushResult{HasToolCalls: true, ToolCalls: []ToolCall{{ID: "1"}}}
	if !result.HasData() {
		t.Error("Should have data with tool calls")
	}

	// 有工具结果
	result = &FlushResult{HasToolResult: true, ToolResults: []ToolCall{{ID: "1"}}}
	if !result.HasData() {
		t.Error("Should have data with tool result")
	}

	// 有消息
	result = &FlushResult{Messages: []BufferedMessage{{Type: "complete"}}}
	if !result.HasData() {
		t.Error("Should have data with messages")
	}
}

func TestStreamUpdateManager_StartStop(t *testing.T) {
	config := DefaultBatchUpdateConfig()
	manager := NewStreamUpdateManager(config)

	// 启动
	manager.Start()
	if !manager.IsRunning() {
		t.Error("Manager should be running")
	}

	// 再次启动应该无效
	manager.Start()
	if !manager.IsRunning() {
		t.Error("Manager should still be running")
	}

	// 停止
	manager.Stop()
	if manager.IsRunning() {
		t.Error("Manager should not be running")
	}
}

func TestStreamUpdateManager_OnUpdate(t *testing.T) {
	config := BatchUpdateConfig{
		BufferDuration:    10, // 10ms
		MaxBufferSize:     10,
		EnableBatching:    true,
		MinUpdateInterval: 5,
	}
	manager := NewStreamUpdateManager(config)

	// 记录更新次数
	updateCount := 0
	var lastResult *FlushResult

	manager.SetOnUpdate(func(result *FlushResult) {
		updateCount++
		lastResult = result
	})

	// 启动
	manager.Start()
	defer manager.Stop()

	// 添加数据
	manager.AddContent("测试内容")

	// 等待更新
	time.Sleep(20 * time.Millisecond)

	// 应该触发了更新
	if updateCount == 0 {
		t.Error("Should have triggered update")
	}
	if lastResult == nil || lastResult.Content != "测试内容" {
		t.Errorf("Last result content mismatch: got %v", lastResult)
	}
}

func TestStreamUpdateManager_ForceFlush(t *testing.T) {
	config := DefaultBatchUpdateConfig()
	manager := NewStreamUpdateManager(config)

	// 记录更新
	updateCount := 0
	manager.SetOnUpdate(func(result *FlushResult) {
		updateCount++
	})

	// 添加数据（不启动管理器）
	manager.AddContent("测试内容")

	// 强制刷新
	result := manager.ForceFlush()

	// 应该触发了更新
	if updateCount != 1 {
		t.Errorf("Update count mismatch: got %d", updateCount)
	}
	if result == nil || result.Content != "测试内容" {
		t.Errorf("Result content mismatch: got %v", result)
	}
}

func TestStreamUpdateManager_ConcurrentAccess(t *testing.T) {
	config := BatchUpdateConfig{
		BufferDuration:    10,
		MaxBufferSize:     100,
		EnableBatching:    true,
		MinUpdateInterval: 5,
	}
	manager := NewStreamUpdateManager(config)

	// 启动
	manager.Start()
	defer manager.Stop()

	// 并发添加数据
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				manager.AddContent("content")
				manager.AddThinking("thinking")
				manager.AddToolCall(ToolCall{
					ID:   string(rune(id*10 + j)),
					Name: "tool",
				}, false)
			}
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 等待更新完成
	time.Sleep(50 * time.Millisecond)

	// 如果没有 panic 或死锁，测试通过
}

func TestStreamUpdateManager_Clear(t *testing.T) {
	config := DefaultBatchUpdateConfig()
	manager := NewStreamUpdateManager(config)

	// 添加数据
	manager.AddContent("测试内容")
	manager.AddThinking("思考")

	// 清空
	manager.Clear()

	// 强制刷新应该没有数据
	result := manager.ForceFlush()
	if result.HasData() {
		t.Error("Should have no data after clear")
	}
}

func TestMessageBuffer_GetBufferSize(t *testing.T) {
	config := DefaultBatchUpdateConfig()
	buffer := NewMessageBuffer(config)

	// 初始大小应该为 0
	if buffer.GetBufferSize() != 0 {
		t.Errorf("Initial buffer size should be 0, got %d", buffer.GetBufferSize())
	}

	// 添加消息
	buffer.AddComplete(100)
	buffer.AddError(errors.New("test"))

	// 检查大小
	if buffer.GetBufferSize() != 2 {
		t.Errorf("Buffer size should be 2, got %d", buffer.GetBufferSize())
	}
}

func TestMessageBuffer_HasPendingData(t *testing.T) {
	config := DefaultBatchUpdateConfig()
	buffer := NewMessageBuffer(config)

	// 初始状态不应该有待处理数据
	if buffer.HasPendingData() {
		t.Error("Should not have pending data initially")
	}

	// 添加数据
	buffer.AddContent("test")

	// 应该有待处理数据
	if !buffer.HasPendingData() {
		t.Error("Should have pending data after adding content")
	}

	// 刷新
	buffer.Flush()

	// 应该没有待处理数据
	if buffer.HasPendingData() {
		t.Error("Should not have pending data after flush")
	}
}

// 基准测试
func BenchmarkMessageBuffer_AddContent(b *testing.B) {
	config := DefaultBatchUpdateConfig()
	buffer := NewMessageBuffer(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer.AddContent("test content")
	}
}

func BenchmarkMessageBuffer_Flush(b *testing.B) {
	config := DefaultBatchUpdateConfig()
	buffer := NewMessageBuffer(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer.AddContent("test content")
		buffer.Flush()
	}
}

func BenchmarkStreamUpdateManager_AddContent(b *testing.B) {
	config := DefaultBatchUpdateConfig()
	manager := NewStreamUpdateManager(config)
	manager.Start()
	defer manager.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.AddContent("test content")
	}
}
