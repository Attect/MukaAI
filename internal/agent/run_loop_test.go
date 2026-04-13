package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Attect/MukaAI/internal/model"
	"github.com/Attect/MukaAI/internal/state"
	"github.com/Attect/MukaAI/internal/tools"
)

// setupAgentWithState 创建带有状态的Agent，用于run_loop子方法测试
func setupAgentWithState(t *testing.T, taskID string) (*Agent, *state.StateManager, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "agent-runloop-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	stateManager, err := state.NewStateManager(filepath.Join(tmpDir, "state"), true)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("创建状态管理器失败: %v", err)
	}

	// 创建任务状态
	_, err = stateManager.CreateTask(taskID, "test goal")
	if err != nil {
		stateManager.Load(taskID)
	}
	_ = stateManager.UpdateTaskStatus(taskID, "in_progress")

	registry := tools.NewToolRegistry()
	server := mockAPIServer(nil)

	config := model.DefaultConfig()
	config.Endpoint = server.URL + "/"
	config.APIKey = "test-key"
	config.ModelName = "test-model"

	client, err := model.NewClient(config)
	if err != nil {
		server.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("创建模型客户端失败: %v", err)
	}

	agent, err := NewAgent(&Config{
		ModelClient:   client,
		ToolRegistry:  registry,
		StateManager:  stateManager,
		MaxIterations: 10,
	})
	if err != nil {
		server.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("创建Agent失败: %v", err)
	}
	agent.taskID = taskID

	cleanup := func() {
		server.Close()
		os.RemoveAll(tmpDir)
	}

	return agent, stateManager, cleanup
}

// TestHandleMaxIterations_基本功能 测试达到最大迭代次数的处理
func TestHandleMaxIterations_基本功能(t *testing.T) {
	// arrange
	agent, _, cleanup := setupAgentWithState(t, "test-max-iter")
	defer cleanup()

	result := &RunResult{
		TaskID:    "test-max-iter",
		StartTime: time.Now().Add(-1 * time.Minute),
	}

	// act
	ret, err := agent.handleMaxIterations(result, 11, 10)

	// assert
	if err == nil {
		t.Error("期望返回错误")
	}
	if ret.Status != "max_iterations" {
		t.Errorf("期望状态 'max_iterations', 实际 '%s'", ret.Status)
	}
	if ret.Iterations != 10 {
		t.Errorf("期望迭代数 10, 实际 %d", ret.Iterations)
	}
	if ret.EndTime.IsZero() {
		t.Error("EndTime不应为零值")
	}
	if ret.Duration == 0 {
		t.Error("Duration不应为零值")
	}
}

// TestIsTaskComplete_各种完成标志 测试各种任务完成标志的识别
func TestIsTaskComplete_各种完成标志(t *testing.T) {
	agent, _, cleanup := setupAgentWithState(t, "test-complete")
	defer cleanup()

	tests := []struct {
		content  string
		expected bool
	}{
		{"任务已完成", true},
		{"task completed", true},
		{"任务完成", true},
		{"all done", true},
		{"已经完成", true},
		{"完成了", true},
		{"finished", true},
		{"done", true},
		{"任务结束", true},
		{"TASK COMPLETED", true}, // 大小写不敏感
		{"DONE", true},
		{"正在进行中", false},
		{"还需要继续", false},
		{"working on it", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("content=%q", tt.content), func(t *testing.T) {
			got := agent.isTaskComplete(tt.content)
			if got != tt.expected {
				t.Errorf("isTaskComplete(%q) = %v, want %v", tt.content, got, tt.expected)
			}
		})
	}
}

// TestFinalizeResult_正常时间 测试正常的时间差计算
func TestFinalizeResult_正常时间(t *testing.T) {
	agent, _, cleanup := setupAgentWithState(t, "test-finalize")
	defer cleanup()

	// arrange
	start := time.Now().Add(-5 * time.Second)
	end := time.Now()
	result := &RunResult{
		StartTime: start,
		EndTime:   end,
	}

	// act
	agent.finalizeResult(result)

	// assert
	if result.Duration == 0 {
		t.Error("Duration不应为零值")
	}
	if result.Duration < 4*time.Second || result.Duration > 10*time.Second {
		t.Errorf("Duration应在4-10秒之间, 实际 %v", result.Duration)
	}
}

// TestFinalizeResult_零值时间 测试零值时间不崩溃
func TestFinalizeResult_零值时间(t *testing.T) {
	agent, _, cleanup := setupAgentWithState(t, "test-finalize-zero")
	defer cleanup()

	// arrange
	result := &RunResult{}

	// act - 不应panic
	agent.finalizeResult(result)

	// assert
	if result.Duration != 0 {
		t.Errorf("零值时间时Duration应为0, 实际 %v", result.Duration)
	}
}

// TestFinalizeResult_部分零值 测试StartTime为零但EndTime非零
func TestFinalizeResult_部分零值(t *testing.T) {
	agent, _, cleanup := setupAgentWithState(t, "test-finalize-partial")
	defer cleanup()

	// arrange
	result := &RunResult{
		EndTime: time.Now(),
	}

	// act - 不应panic
	agent.finalizeResult(result)

	// assert
	if result.Duration != 0 {
		t.Error("部分零值时Duration应为0")
	}
}

// TestInjectContinuePrompt_基本功能 测试注入继续执行的提示消息
func TestInjectContinuePrompt_基本功能(t *testing.T) {
	// arrange
	agent, _, cleanup := setupAgentWithState(t, "test-continue-prompt")
	defer cleanup()

	initialCount := agent.history.GetMessageCount()

	// act
	agent.injectContinuePrompt()

	// assert
	if agent.history.GetMessageCount() != initialCount+1 {
		t.Errorf("期望消息数增加1, 实际从%d变为%d", initialCount, agent.history.GetMessageCount())
	}

	msgs := agent.history.GetMessages()
	lastMsg := msgs[len(msgs)-1]
	if lastMsg.Role != model.RoleUser {
		t.Errorf("期望最后一条消息角色为user, 实际 '%s'", lastMsg.Role)
	}
	if lastMsg.Content == "" {
		t.Error("消息内容不应为空")
	}
}

// TestInjectContinuePrompt_回调触发 测试注入消息时触发回调
func TestInjectContinuePrompt_回调触发(t *testing.T) {
	// arrange
	agent, _, cleanup := setupAgentWithState(t, "test-continue-callback")
	defer cleanup()

	var callbackRole, callbackContent string
	agent.onHistoryAdd = func(role, content string) {
		callbackRole = role
		callbackContent = content
	}

	// act
	agent.injectContinuePrompt()

	// assert
	if callbackRole != "user" {
		t.Errorf("期望回调角色 'user', 实际 '%s'", callbackRole)
	}
	if callbackContent == "" {
		t.Error("回调内容不应为空")
	}
}

// TestCheckCancellation_正常上下文 测试未取消的上下文
func TestCheckCancellation_正常上下文(t *testing.T) {
	agent, _, cleanup := setupAgentWithState(t, "test-cancel-normal")
	defer cleanup()

	result := &RunResult{
		TaskID:    "test-cancel-normal",
		StartTime: time.Now(),
	}

	// act
	cancelled, ret, err := agent.checkCancellation(context.Background(), result, 1)

	// assert
	if cancelled {
		t.Error("正常上下文不应被取消")
	}
	if ret != nil {
		t.Error("未取消时应返回nil结果")
	}
	if err != nil {
		t.Error("未取消时不应返回错误")
	}
}

// TestCheckCancellation_已取消上下文 测试已取消的上下文
func TestCheckCancellation_已取消上下文(t *testing.T) {
	agent, _, cleanup := setupAgentWithState(t, "test-cancelled")
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	result := &RunResult{
		TaskID:    "test-cancelled",
		StartTime: time.Now(),
	}

	// act
	cancelled, ret, err := agent.checkCancellation(ctx, result, 5)

	// assert
	if !cancelled {
		t.Error("已取消的上下文应被识别")
	}
	if ret == nil {
		t.Fatal("取消时应返回非nil结果")
	}
	if ret.Status != "cancelled" {
		t.Errorf("期望状态 'cancelled', 实际 '%s'", ret.Status)
	}
	if err == nil {
		t.Error("取消时应返回错误")
	}
}

// TestCheckCancellation_取消时迭代计数 测试取消时迭代数正确
func TestCheckCancellation_取消时迭代计数(t *testing.T) {
	agent, _, cleanup := setupAgentWithState(t, "test-cancel-iter")
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := &RunResult{
		TaskID:    "test-cancel-iter",
		StartTime: time.Now(),
	}

	// act - totalIterations=5, 应记录为4
	cancelled, ret, _ := agent.checkCancellation(ctx, result, 5)

	// assert
	if !cancelled {
		t.Fatal("应被取消")
	}
	if ret.Iterations != 4 {
		t.Errorf("期望迭代数 4 (5-1), 实际 %d", ret.Iterations)
	}
}

// TestIterationResult_结构体 测试iterationResult结构体
func TestIterationResult_结构体(t *testing.T) {
	// arrange & act
	result := &RunResult{
		TaskID:    "test",
		Status:    "completed",
		StartTime: time.Now(),
		EndTime:   time.Now(),
	}

	ir := &iterationResult{
		action: "return",
		result: result,
		err:    fmt.Errorf("test error"),
	}

	// assert
	if ir.action != "return" {
		t.Errorf("期望 action='return', 实际 '%s'", ir.action)
	}
	if ir.result.Status != "completed" {
		t.Errorf("期望 result.Status='completed', 实际 '%s'", ir.result.Status)
	}
	if ir.err == nil {
		t.Error("期望有错误")
	}
}

// TestRecordModelResponse_基本功能 测试记录模型响应
func TestRecordModelResponse_基本功能(t *testing.T) {
	agent, _, cleanup := setupAgentWithState(t, "test-record-response")
	defer cleanup()

	// arrange
	response := &modelResponse{
		Content: "测试响应内容",
		ToolCalls: []model.ToolCall{
			{
				ID:   "call-001",
				Type: "function",
				Function: model.FunctionCall{
					Name:      "read_file",
					Arguments: `{}`,
				},
			},
		},
	}

	// act
	agent.recordModelResponse(response)

	// assert
	msgs := agent.history.GetMessages()
	if len(msgs) == 0 {
		t.Fatal("历史消息不应为空")
	}
	lastMsg := msgs[len(msgs)-1]
	if lastMsg.Role != model.RoleAssistant {
		t.Errorf("期望角色 'assistant', 实际 '%s'", lastMsg.Role)
	}
	if lastMsg.Content != "测试响应内容" {
		t.Errorf("期望内容 '测试响应内容', 实际 '%s'", lastMsg.Content)
	}
}

// TestRecordModelResponse_回调触发 测试记录模型响应时回调触发
func TestRecordModelResponse_回调触发(t *testing.T) {
	agent, _, cleanup := setupAgentWithState(t, "test-record-callback")
	defer cleanup()

	var callbackRole, callbackContent string
	agent.onHistoryAdd = func(role, content string) {
		callbackRole = role
		callbackContent = content
	}

	response := &modelResponse{
		Content: "回调测试内容",
	}

	// act
	agent.recordModelResponse(response)

	// assert
	if callbackRole != "assistant" {
		t.Errorf("期望回调角色 'assistant', 实际 '%s'", callbackRole)
	}
	if callbackContent != "回调测试内容" {
		t.Errorf("期望回调内容 '回调测试内容', 实际 '%s'", callbackContent)
	}
}

// TestPreIteration_回调触发 测试迭代前处理触发回调
func TestPreIteration_回调触发(t *testing.T) {
	agent, _, cleanup := setupAgentWithState(t, "test-pre-iter")
	defer cleanup()

	var receivedIteration int
	agent.onIteration = func(iteration int) {
		receivedIteration = iteration
	}

	// act
	agent.preIteration(7)

	// assert
	if receivedIteration != 7 {
		t.Errorf("期望迭代数 7, 实际 %d", receivedIteration)
	}
}
