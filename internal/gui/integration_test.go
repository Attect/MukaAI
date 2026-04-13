package gui

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Attect/MukaAI/internal/agent"
)

// --- MockEventEmitter ---

// EmittedEvent 记录一次事件发射
type EmittedEvent struct {
	Event string
	Data  []interface{}
}

// MockEventEmitter 测试用的事件发射器Mock
// 线程安全地记录所有Emit调用，供测试断言使用
type MockEventEmitter struct {
	mu     sync.RWMutex
	events []EmittedEvent
}

// Emit 记录事件调用
func (m *MockEventEmitter) Emit(event string, data ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, EmittedEvent{
		Event: event,
		Data:  data,
	})
}

// GetEvents 获取所有已发射事件的快照
func (m *MockEventEmitter) GetEvents() []EmittedEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]EmittedEvent, len(m.events))
	copy(result, m.events)
	return result
}

// FindEvents 查找指定事件名的所有发射记录
func (m *MockEventEmitter) FindEvents(eventName string) []EmittedEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []EmittedEvent
	for _, e := range m.events {
		if e.Event == eventName {
			result = append(result, e)
		}
	}
	return result
}

// Reset 清空已记录的事件
func (m *MockEventEmitter) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = nil
}

// Count 获取已发射事件总数
func (m *MockEventEmitter) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.events)
}

// --- 测试辅助 ---

// newTestAppWithMock 创建带有MockEventEmitter的测试App
func newTestAppWithMock() (*App, *MockEventEmitter) {
	app := NewApp()
	app.ctx = context.Background()
	mock := &MockEventEmitter{}
	app.eventEmitter = mock
	return app, mock
}

// newTestAppWithMockAndConversations 创建带有MockEventEmitter和预设对话的测试App
func newTestAppWithMockAndConversations() (*App, *MockEventEmitter) {
	app, mock := newTestAppWithMock()
	app.conversations = []*conversation{
		{
			id:        "conv-1",
			title:     "对话1",
			createdAt: time.Now().Add(-2 * time.Hour),
			status:    "active",
			messages: []*message{
				{role: "user", content: "hello", timestamp: time.Now()},
				{role: "assistant", content: "hi", timestamp: time.Now()},
			},
		},
		{
			id:        "conv-2",
			title:     "对话2",
			createdAt: time.Now().Add(-1 * time.Hour),
			status:    "active",
			messages: []*message{
				{role: "user", content: "world", timestamp: time.Now()},
			},
		},
	}
	app.activeConvID = "conv-1"
	return app, mock
}

// --- StreamBridge 集成测试 ---

// TestStreamBridge_OnThinking_EmitsEvent 验证OnThinking发射正确的事件
func TestStreamBridge_OnThinking_EmitsEvent(t *testing.T) {
	app, mock := newTestAppWithMock()
	// 创建活跃对话
	app.mu.Lock()
	app.conversations = append(app.conversations, &conversation{
		id: "conv-test", title: "测试", createdAt: time.Now(), status: "active",
	})
	app.activeConvID = "conv-test"
	app.mu.Unlock()

	bridge := NewStreamBridge(app)
	bridge.OnThinking("思考内容块")

	events := mock.GetEvents()
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}

	// 验证stream:thinking事件
	if events[0].Event != "stream:thinking" {
		t.Errorf("expected first event 'stream:thinking', got %q", events[0].Event)
	}
	if len(events[0].Data) < 1 || events[0].Data[0] != "思考内容块" {
		t.Errorf("expected thinking chunk data, got %v", events[0].Data)
	}

	// 验证conversation:updated事件
	if events[1].Event != "conversation:updated" {
		t.Errorf("expected second event 'conversation:updated', got %q", events[1].Event)
	}

	// 验证内部状态更新
	app.mu.RLock()
	conv := app.getActiveConversation()
	app.mu.RUnlock()
	if conv == nil || conv.currentMessage == nil {
		t.Fatal("expected currentMessage to be created")
	}
	if conv.currentMessage.thinking != "思考内容块" {
		t.Errorf("expected thinking='思考内容块', got %q", conv.currentMessage.thinking)
	}
	if !conv.currentMessage.isStreaming {
		t.Error("expected isStreaming=true")
	}
	if conv.currentMessage.streamingType != "thinking" {
		t.Errorf("expected streamingType='thinking', got %q", conv.currentMessage.streamingType)
	}
}

// TestStreamBridge_OnContent_EmitsEvent 验证OnContent发射正确的事件
func TestStreamBridge_OnContent_EmitsEvent(t *testing.T) {
	app, mock := newTestAppWithMock()
	app.mu.Lock()
	app.conversations = append(app.conversations, &conversation{
		id: "conv-test", title: "测试", createdAt: time.Now(), status: "active",
	})
	app.activeConvID = "conv-test"
	app.mu.Unlock()

	bridge := NewStreamBridge(app)
	bridge.OnContent("正文内容块")

	events := mock.GetEvents()
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}

	if events[0].Event != "stream:content" {
		t.Errorf("expected first event 'stream:content', got %q", events[0].Event)
	}
	if len(events[0].Data) < 1 || events[0].Data[0] != "正文内容块" {
		t.Errorf("expected content chunk data, got %v", events[0].Data)
	}

	if events[1].Event != "conversation:updated" {
		t.Errorf("expected second event 'conversation:updated', got %q", events[1].Event)
	}

	// 验证内部状态
	app.mu.RLock()
	conv := app.getActiveConversation()
	app.mu.RUnlock()
	if conv == nil || conv.currentMessage == nil {
		t.Fatal("expected currentMessage to be created")
	}
	if conv.currentMessage.content != "正文内容块" {
		t.Errorf("expected content='正文内容块', got %q", conv.currentMessage.content)
	}
	if conv.currentMessage.streamingType != "content" {
		t.Errorf("expected streamingType='content', got %q", conv.currentMessage.streamingType)
	}
}

// TestStreamBridge_OnToolCall_EmitsEvent 验证OnToolCall发射正确的事件
func TestStreamBridge_OnToolCall_EmitsEvent(t *testing.T) {
	app, mock := newTestAppWithMock()
	app.mu.Lock()
	app.conversations = append(app.conversations, &conversation{
		id: "conv-test", title: "测试", createdAt: time.Now(), status: "active",
	})
	app.activeConvID = "conv-test"
	app.mu.Unlock()

	bridge := NewStreamBridge(app)
	call := agent.ToolCallInfo{
		ID:        "call-1",
		Name:      "read_file",
		Arguments: `{"path":"test.go"}`,
		Result:    "file content",
	}
	bridge.OnToolCall(call, true)

	events := mock.GetEvents()
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}

	// 验证stream:toolcall事件
	if events[0].Event != "stream:toolcall" {
		t.Errorf("expected first event 'stream:toolcall', got %q", events[0].Event)
	}
	if len(events[0].Data) < 1 {
		t.Fatal("expected toolcall event data")
	}
	eventData, ok := events[0].Data[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected toolcall data to be map[string]interface{}")
	}
	if eventData["id"] != "call-1" {
		t.Errorf("expected id='call-1', got %v", eventData["id"])
	}
	if eventData["name"] != "read_file" {
		t.Errorf("expected name='read_file', got %v", eventData["name"])
	}
	if eventData["isComplete"] != true {
		t.Errorf("expected isComplete=true, got %v", eventData["isComplete"])
	}

	// 验证conversation:updated事件
	if events[1].Event != "conversation:updated" {
		t.Errorf("expected second event 'conversation:updated', got %q", events[1].Event)
	}
}

// TestStreamBridge_OnToolResult_EmitsEvent 验证OnToolResult发射正确的事件
func TestStreamBridge_OnToolResult_EmitsEvent(t *testing.T) {
	app, mock := newTestAppWithMock()
	app.mu.Lock()
	app.conversations = append(app.conversations, &conversation{
		id: "conv-test", title: "测试", createdAt: time.Now(), status: "active",
		currentMessage: &message{
			role: "assistant",
			toolCalls: []ToolCall{
				{ID: "call-1", Name: "read_file"},
			},
			timestamp: time.Now(),
		},
	})
	app.activeConvID = "conv-test"
	app.mu.Unlock()

	bridge := NewStreamBridge(app)
	result := agent.ToolCallInfo{
		ID:     "call-1",
		Name:   "read_file",
		Result: "文件内容",
	}
	bridge.OnToolResult(result)

	events := mock.GetEvents()
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}

	// 验证stream:toolresult事件
	if events[0].Event != "stream:toolresult" {
		t.Errorf("expected first event 'stream:toolresult', got %q", events[0].Event)
	}
	eventData, ok := events[0].Data[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected toolresult data to be map[string]interface{}")
	}
	if eventData["id"] != "call-1" {
		t.Errorf("expected id='call-1', got %v", eventData["id"])
	}
	if eventData["result"] != "文件内容" {
		t.Errorf("expected result='文件内容', got %v", eventData["result"])
	}

	// 验证conversation:updated事件
	if events[1].Event != "conversation:updated" {
		t.Errorf("expected second event 'conversation:updated', got %q", events[1].Event)
	}
}

// TestStreamBridge_OnComplete_EmitsEvent 验证OnComplete发射正确的事件
func TestStreamBridge_OnComplete_EmitsEvent(t *testing.T) {
	app, mock := newTestAppWithMock()
	app.mu.Lock()
	app.conversations = append(app.conversations, &conversation{
		id: "conv-test", title: "测试", createdAt: time.Now(), status: "active",
		currentMessage: &message{
			role:      "assistant",
			content:   "回复内容",
			timestamp: time.Now(),
		},
	})
	app.activeConvID = "conv-test"
	app.mu.Unlock()

	bridge := NewStreamBridge(app)
	bridge.OnComplete(150)

	events := mock.GetEvents()
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}

	// 验证stream:complete事件
	if events[0].Event != "stream:complete" {
		t.Errorf("expected first event 'stream:complete', got %q", events[0].Event)
	}
	completeData, ok := events[0].Data[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected complete data to be map[string]interface{}")
	}
	if completeData["usage"] != 150 {
		t.Errorf("expected usage=150, got %v", completeData["usage"])
	}

	// 验证tokenstats:updated事件
	if events[1].Event != "tokenstats:updated" {
		t.Errorf("expected second event 'tokenstats:updated', got %q", events[1].Event)
	}

	// 验证token统计已更新
	stats := app.GetTokenStats()
	if stats.TotalTokens != 150 {
		t.Errorf("expected TotalTokens=150, got %d", stats.TotalTokens)
	}
	if stats.InferenceCount != 1 {
		t.Errorf("expected InferenceCount=1, got %d", stats.InferenceCount)
	}
}

// TestStreamBridge_OnError_EmitsEvent 验证OnError发射正确的事件
func TestStreamBridge_OnError_EmitsEvent(t *testing.T) {
	app, mock := newTestAppWithMock()
	app.mu.Lock()
	app.conversations = append(app.conversations, &conversation{
		id: "conv-test", title: "测试", createdAt: time.Now(), status: "active",
		currentMessage: &message{
			role:      "assistant",
			content:   "部分内容",
			timestamp: time.Now(),
		},
	})
	app.activeConvID = "conv-test"
	app.isStreaming = true
	app.mu.Unlock()

	bridge := NewStreamBridge(app)
	bridge.OnError(fmt.Errorf("模型调用失败"))

	events := mock.GetEvents()
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}

	// 验证stream:error事件
	if events[0].Event != "stream:error" {
		t.Errorf("expected first event 'stream:error', got %q", events[0].Event)
	}
	if len(events[0].Data) < 1 || events[0].Data[0] != "模型调用失败" {
		t.Errorf("expected error message data, got %v", events[0].Data)
	}

	// 验证conversation:updated事件
	if events[1].Event != "conversation:updated" {
		t.Errorf("expected second event 'conversation:updated', got %q", events[1].Event)
	}

	// 验证isStreaming已重置
	app.mu.RLock()
	streaming := app.isStreaming
	app.mu.RUnlock()
	if streaming {
		t.Error("expected isStreaming=false after error")
	}
}

// TestStreamBridge_OnTaskDone_EmitsEvent 验证OnTaskDone发射正确的事件
func TestStreamBridge_OnTaskDone_EmitsEvent(t *testing.T) {
	app, mock := newTestAppWithMock()
	app.mu.Lock()
	app.conversations = append(app.conversations, &conversation{
		id: "conv-test", title: "测试", createdAt: time.Now(), status: "active",
		currentMessage: &message{
			role:      "assistant",
			content:   "最终回复",
			timestamp: time.Now(),
		},
	})
	app.activeConvID = "conv-test"
	app.isStreaming = true
	app.mu.Unlock()

	bridge := NewStreamBridge(app)
	bridge.OnTaskDone()

	events := mock.GetEvents()
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}

	// 验证stream:done事件
	if events[0].Event != "stream:done" {
		t.Errorf("expected first event 'stream:done', got %q", events[0].Event)
	}
	if len(events[0].Data) != 0 {
		t.Errorf("expected no data for stream:done, got %v", events[0].Data)
	}

	// 验证conversation:updated事件
	if events[1].Event != "conversation:updated" {
		t.Errorf("expected second event 'conversation:updated', got %q", events[1].Event)
	}

	// 验证消息已固化到messages列表
	app.mu.RLock()
	conv := app.getActiveConversation()
	app.mu.RUnlock()
	if conv == nil {
		t.Fatal("expected conversation")
	}
	if len(conv.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(conv.messages))
	}
	if conv.messages[0].content != "最终回复" {
		t.Errorf("expected content='最终回复', got %q", conv.messages[0].content)
	}
	if conv.currentMessage != nil {
		t.Error("expected currentMessage to be nil after task done")
	}

	// 验证isStreaming已重置
	app.mu.RLock()
	streaming := app.isStreaming
	app.mu.RUnlock()
	if streaming {
		t.Error("expected isStreaming=false after task done")
	}
}

// --- App 方法集成测试 ---

// TestApp_SwitchConversation_EmitsEvent 验证SwitchConversation发射conversation:updated事件
func TestApp_SwitchConversation_EmitsEvent(t *testing.T) {
	app, mock := newTestAppWithMockAndConversations()

	err := app.SwitchConversation("conv-2")
	if err != nil {
		t.Fatalf("SwitchConversation failed: %v", err)
	}

	events := mock.FindEvents("conversation:updated")
	if len(events) != 1 {
		t.Fatalf("expected 1 conversation:updated event, got %d", len(events))
	}

	// 验证事件数据包含切换后的对话
	if len(events[0].Data) < 1 {
		t.Fatal("expected event data")
	}
	data, ok := events[0].Data[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be map[string]interface{}")
	}
	if data["id"] != "conv-2" {
		t.Errorf("expected active conv id=conv-2, got %v", data["id"])
	}
}

// TestApp_DeleteConversation_EmitsEvent 验证DeleteConversation发射conversation:updated事件
func TestApp_DeleteConversation_EmitsEvent(t *testing.T) {
	app, mock := newTestAppWithMockAndConversations()

	err := app.DeleteConversation("conv-1")
	if err != nil {
		t.Fatalf("DeleteConversation failed: %v", err)
	}

	events := mock.FindEvents("conversation:updated")
	if len(events) != 1 {
		t.Fatalf("expected 1 conversation:updated event, got %d", len(events))
	}

	// 验证对话已从列表移除
	convs := app.GetConversations()
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation after delete, got %d", len(convs))
	}
	if convs[0].ID != "conv-2" {
		t.Errorf("expected remaining conv id=conv-2, got %s", convs[0].ID)
	}
}

// TestApp_InterruptInference_EmitsEvent 验证InterruptInference发射stream:interrupted和conversation:updated事件
func TestApp_InterruptInference_EmitsEvent(t *testing.T) {
	app, mock := newTestAppWithMock()
	app.mu.Lock()
	app.conversations = append(app.conversations, &conversation{
		id: "conv-test", title: "测试", createdAt: time.Now(), status: "active",
		currentMessage: &message{
			role:      "assistant",
			content:   "正在生成的内容",
			timestamp: time.Now(),
		},
	})
	app.activeConvID = "conv-test"
	app.isStreaming = true
	app.mu.Unlock()

	app.InterruptInference()

	// 验证stream:interrupted事件
	interruptedEvents := mock.FindEvents("stream:interrupted")
	if len(interruptedEvents) != 1 {
		t.Fatalf("expected 1 stream:interrupted event, got %d", len(interruptedEvents))
	}

	// 验证conversation:updated事件
	updatedEvents := mock.FindEvents("conversation:updated")
	if len(updatedEvents) != 1 {
		t.Fatalf("expected 1 conversation:updated event, got %d", len(updatedEvents))
	}

	// 验证消息包含打断标记
	app.mu.RLock()
	conv := app.getActiveConversation()
	app.mu.RUnlock()
	if conv == nil || len(conv.messages) == 0 {
		t.Fatal("expected message to be finalized with interrupt marker")
	}
	lastMsg := conv.messages[len(conv.messages)-1]
	if lastMsg.content != "正在生成的内容\n\n[用户打断]" {
		t.Errorf("expected content with interrupt marker, got %q", lastMsg.content)
	}

	// 验证isStreaming已重置
	app.mu.RLock()
	streaming := app.isStreaming
	app.mu.RUnlock()
	if streaming {
		t.Error("expected isStreaming=false after interrupt")
	}
}

// TestApp_ClearConversation_EmitsEvent 验证ClearConversation发射conversation:updated事件
func TestApp_ClearConversation_EmitsEvent(t *testing.T) {
	app, mock := newTestAppWithMockAndConversations()

	app.ClearConversation()

	events := mock.FindEvents("conversation:updated")
	if len(events) != 1 {
		t.Fatalf("expected 1 conversation:updated event, got %d", len(events))
	}
}

// TestApp_UpdateConversationTitle_EmitsEvent 验证UpdateConversationTitle发射conversation:updated事件
func TestApp_UpdateConversationTitle_EmitsEvent(t *testing.T) {
	app, mock := newTestAppWithMockAndConversations()

	err := app.UpdateConversationTitle("conv-1", "新标题")
	if err != nil {
		t.Fatalf("UpdateConversationTitle failed: %v", err)
	}

	events := mock.FindEvents("conversation:updated")
	if len(events) != 1 {
		t.Fatalf("expected 1 conversation:updated event, got %d", len(events))
	}

	// 验证标题已更新
	app.mu.RLock()
	conv := app.getActiveConversation()
	app.mu.RUnlock()
	if conv.title != "新标题" {
		t.Errorf("expected title='新标题', got %q", conv.title)
	}
}

// TestApp_SetWorkDir_EmitsEvent 验证SetWorkDir发射workdir:changed事件
func TestApp_SetWorkDir_EmitsEvent(t *testing.T) {
	app, mock := newTestAppWithMock()
	tmpDir := t.TempDir()

	err := app.SetWorkDir(tmpDir)
	if err != nil {
		t.Fatalf("SetWorkDir failed: %v", err)
	}

	events := mock.FindEvents("workdir:changed")
	if len(events) != 1 {
		t.Fatalf("expected 1 workdir:changed event, got %d", len(events))
	}
	if len(events[0].Data) < 1 {
		t.Fatal("expected event data")
	}
	if events[0].Data[0] != tmpDir {
		t.Errorf("expected workdir=%s, got %v", tmpDir, events[0].Data[0])
	}
}

// TestApp_SetEventEmitter 验证SetEventEmitter方法
func TestApp_SetEventEmitter(t *testing.T) {
	app := NewApp()
	mock := &MockEventEmitter{}
	app.SetEventEmitter(mock)

	app.mu.RLock()
	emitter := app.eventEmitter
	app.mu.RUnlock()
	if emitter != mock {
		t.Error("expected eventEmitter to be the mock")
	}
}

// TestStreamBridge_OnError_NilErr_NoErrorEvent 验证OnError传入nil error时不发射stream:error事件
func TestStreamBridge_OnError_NilErr_NoErrorEvent(t *testing.T) {
	app, mock := newTestAppWithMock()
	app.mu.Lock()
	app.conversations = append(app.conversations, &conversation{
		id: "conv-test", title: "测试", createdAt: time.Now(), status: "active",
	})
	app.activeConvID = "conv-test"
	app.isStreaming = true
	app.mu.Unlock()

	bridge := NewStreamBridge(app)
	bridge.OnError(nil)

	// 不应发射stream:error事件
	errorEvents := mock.FindEvents("stream:error")
	if len(errorEvents) != 0 {
		t.Errorf("expected 0 stream:error events for nil error, got %d", len(errorEvents))
	}

	// 应发射conversation:updated事件
	updatedEvents := mock.FindEvents("conversation:updated")
	if len(updatedEvents) != 1 {
		t.Errorf("expected 1 conversation:updated event, got %d", len(updatedEvents))
	}
}

// TestStreamBridge_OnSupervisorResult_EmitsEvent 验证OnSupervisorResult发射正确事件
func TestStreamBridge_OnSupervisorResult_EmitsEvent(t *testing.T) {
	app, mock := newTestAppWithMock()
	bridge := NewStreamBridge(app)

	result := &agent.SupervisionResult{
		Status:           "warning",
		Summary:          "发现潜在问题",
		InterventionType: "warning",
		Issues: []agent.SupervisionIssue{
			{Type: "code_quality", Severity: "medium", Description: "缺少错误处理"},
		},
	}
	bridge.OnSupervisorResult(result)

	events := mock.FindEvents("supervisor:result")
	if len(events) != 1 {
		t.Fatalf("expected 1 supervisor:result event, got %d", len(events))
	}

	if len(events[0].Data) < 1 {
		t.Fatal("expected event data")
	}
	data, ok := events[0].Data[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be map[string]interface{}")
	}
	if data["status"] != "warning" {
		t.Errorf("expected status=warning, got %v", data["status"])
	}
	if data["summary"] != "发现潜在问题" {
		t.Errorf("expected summary='发现潜在问题', got %v", data["summary"])
	}
	if data["issues_count"] != 1 {
		t.Errorf("expected issues_count=1, got %v", data["issues_count"])
	}
}

// TestStreamBridge_OnSupervisorResult_NilEmitter_NoEvent 验证无EventEmitter时不发射事件
func TestStreamBridge_OnSupervisorResult_NilEmitter_NoEvent(t *testing.T) {
	app := NewApp()
	app.ctx = context.Background()
	// 不设置eventEmitter，保持为nil
	bridge := NewStreamBridge(app)

	result := &agent.SupervisionResult{
		Status:  "pass",
		Summary: "通过",
	}
	bridge.OnSupervisorResult(result) // 不应panic
}

// TestStreamBridge_MultipleCallbacks_EventOrder 验证多个回调的事件发射顺序
func TestStreamBridge_MultipleCallbacks_EventOrder(t *testing.T) {
	app, mock := newTestAppWithMock()
	app.mu.Lock()
	app.conversations = append(app.conversations, &conversation{
		id: "conv-test", title: "测试", createdAt: time.Now(), status: "active",
	})
	app.activeConvID = "conv-test"
	app.mu.Unlock()

	bridge := NewStreamBridge(app)

	// 模拟一次完整的流式对话流程
	bridge.OnThinking("正在思考...")
	bridge.OnContent("回复内容")
	bridge.OnComplete(100)
	bridge.OnTaskDone()

	// 验证关键事件的顺序
	events := mock.GetEvents()

	// 找到stream:thinking, stream:content, stream:complete, stream:done
	var eventNames []string
	for _, e := range events {
		eventNames = append(eventNames, e.Event)
	}

	// 验证stream:thinking在stream:content之前
	thinkingIdx := indexOf(eventNames, "stream:thinking")
	contentIdx := indexOf(eventNames, "stream:content")
	completeIdx := indexOf(eventNames, "stream:complete")
	doneIdx := indexOf(eventNames, "stream:done")

	if thinkingIdx == -1 {
		t.Error("expected stream:thinking event")
	}
	if contentIdx == -1 {
		t.Error("expected stream:content event")
	}
	if completeIdx == -1 {
		t.Error("expected stream:complete event")
	}
	if doneIdx == -1 {
		t.Error("expected stream:done event")
	}

	if thinkingIdx > contentIdx {
		t.Error("expected stream:thinking before stream:content")
	}
	if contentIdx > completeIdx {
		t.Error("expected stream:content before stream:complete")
	}
	if completeIdx > doneIdx {
		t.Error("expected stream:complete before stream:done")
	}
}

// --- 辅助函数 ---

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}
