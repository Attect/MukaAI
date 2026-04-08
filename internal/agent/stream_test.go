package agent

import (
	"errors"
	"testing"

	"agentplus/internal/model"
)

// TestStreamHandlerFunc 测试函数式流式处理器
func TestStreamHandlerFunc(t *testing.T) {
	// 记录调用情况
	var thinkingCalled bool
	var contentCalled bool
	var toolCallCalled bool
	var toolResultCalled bool
	var completeCalled bool
	var errorCalled bool

	// 创建流式处理器
	handler := NewStreamHandlerFunc().
		OnThinking(func(chunk string) {
			thinkingCalled = true
			if chunk != "thinking content" {
				t.Errorf("Expected thinking content 'thinking content', got '%s'", chunk)
			}
		}).
		OnContent(func(chunk string) {
			contentCalled = true
			if chunk != "content" {
				t.Errorf("Expected content 'content', got '%s'", chunk)
			}
		}).
		OnToolCall(func(call ToolCallInfo, isComplete bool) {
			toolCallCalled = true
			if call.Name != "test_tool" {
				t.Errorf("Expected tool name 'test_tool', got '%s'", call.Name)
			}
		}).
		OnToolResult(func(result ToolCallInfo) {
			toolResultCalled = true
			if result.Name != "test_tool" {
				t.Errorf("Expected tool name 'test_tool', got '%s'", result.Name)
			}
		}).
		OnComplete(func(usage int) {
			completeCalled = true
			if usage != 100 {
				t.Errorf("Expected usage 100, got %d", usage)
			}
		}).
		OnError(func(err error) {
			errorCalled = true
			if err.Error() != "test error" {
				t.Errorf("Expected error 'test error', got '%s'", err.Error())
			}
		}).
		Build()

	// 测试各个方法
	handler.OnThinking("thinking content")
	if !thinkingCalled {
		t.Error("OnThinking was not called")
	}

	handler.OnContent("content")
	if !contentCalled {
		t.Error("OnContent was not called")
	}

	handler.OnToolCall(ToolCallInfo{Name: "test_tool"}, true)
	if !toolCallCalled {
		t.Error("OnToolCall was not called")
	}

	handler.OnToolResult(ToolCallInfo{Name: "test_tool"})
	if !toolResultCalled {
		t.Error("OnToolResult was not called")
	}

	handler.OnComplete(100)
	if !completeCalled {
		t.Error("OnComplete was not called")
	}

	handler.OnError(errors.New("test error"))
	if !errorCalled {
		t.Error("OnError was not called")
	}
}

// TestStreamHandlerFuncWithNilFunctions 测试空函数的流式处理器
func TestStreamHandlerFuncWithNilFunctions(t *testing.T) {
	// 创建没有设置任何函数的流式处理器
	handler := NewStreamHandlerFunc().Build()

	// 测试各个方法不会 panic
	handler.OnThinking("thinking")
	handler.OnContent("content")
	handler.OnToolCall(ToolCallInfo{Name: "test"}, true)
	handler.OnToolResult(ToolCallInfo{Name: "test"})
	handler.OnComplete(100)
	handler.OnError(errors.New("error"))
}

// TestConvertToolCall 测试工具调用转换
func TestConvertToolCall(t *testing.T) {
	tc := model.ToolCall{
		ID:   "call-123",
		Type: "function",
		Function: model.FunctionCall{
			Name:      "test_tool",
			Arguments: `{"arg1": "value1"}`,
		},
	}

	info := ConvertToolCall(tc)

	if info.ID != "call-123" {
		t.Errorf("Expected ID 'call-123', got '%s'", info.ID)
	}
	if info.Name != "test_tool" {
		t.Errorf("Expected Name 'test_tool', got '%s'", info.Name)
	}
	if info.Arguments != `{"arg1": "value1"}` {
		t.Errorf("Expected Arguments '{\"arg1\": \"value1\"}', got '%s'", info.Arguments)
	}
	if !info.IsComplete {
		t.Error("Expected IsComplete to be true")
	}
}

// TestConvertToolCallWithResult 测试带结果的工具调用转换
func TestConvertToolCallWithResult(t *testing.T) {
	tc := model.ToolCall{
		ID:   "call-123",
		Type: "function",
		Function: model.FunctionCall{
			Name:      "test_tool",
			Arguments: `{"arg1": "value1"}`,
		},
	}

	info := ConvertToolCallWithResult(tc, "success result", "")

	if info.ID != "call-123" {
		t.Errorf("Expected ID 'call-123', got '%s'", info.ID)
	}
	if info.Name != "test_tool" {
		t.Errorf("Expected Name 'test_tool', got '%s'", info.Name)
	}
	if info.Result != "success result" {
		t.Errorf("Expected Result 'success result', got '%s'", info.Result)
	}
	if info.ResultError != "" {
		t.Errorf("Expected ResultError '', got '%s'", info.ResultError)
	}

	// 测试错误结果
	infoError := ConvertToolCallWithResult(tc, "", "error occurred")
	if infoError.Result != "" {
		t.Errorf("Expected Result '', got '%s'", infoError.Result)
	}
	if infoError.ResultError != "error occurred" {
		t.Errorf("Expected ResultError 'error occurred', got '%s'", infoError.ResultError)
	}
}
