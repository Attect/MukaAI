package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agentplus/internal/model"
	"agentplus/internal/state"
	"agentplus/internal/tools"
)

// TestParseToolCallArguments_有效JSON 测试解析有效的工具调用参数
func TestParseToolCallArguments_有效JSON(t *testing.T) {
	// arrange
	tc := model.ToolCall{
		ID:   "call-001",
		Type: "function",
		Function: model.FunctionCall{
			Name:      "update_state",
			Arguments: `{"phase": "coding", "decision": "use Go"}`,
		},
	}

	// act
	args, err := ParseToolCallArguments(tc)

	// assert
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if args["phase"] != "coding" {
		t.Errorf("期望 phase='coding', 实际 '%v'", args["phase"])
	}
	if args["decision"] != "use Go" {
		t.Errorf("期望 decision='use Go', 实际 '%v'", args["decision"])
	}
}

// TestParseToolCallArguments_无效JSON 测试解析无效的JSON参数
func TestParseToolCallArguments_无效JSON(t *testing.T) {
	// arrange
	tc := model.ToolCall{
		Function: model.FunctionCall{
			Name:      "test",
			Arguments: `{invalid json}`,
		},
	}

	// act
	_, err := ParseToolCallArguments(tc)

	// assert
	if err == nil {
		t.Error("期望解析失败，但成功")
	}
}

// TestParseToolCallArguments_空字符串 测试解析空字符串参数
func TestParseToolCallArguments_空字符串(t *testing.T) {
	// arrange
	tc := model.ToolCall{
		Function: model.FunctionCall{
			Name:      "test",
			Arguments: "",
		},
	}

	// act
	_, err := ParseToolCallArguments(tc)

	// assert
	if err == nil {
		t.Error("期望解析失败，但成功")
	}
}

// TestBuildToolResultMessage_基本功能 测试构建工具结果消息
func TestBuildToolResultMessage_基本功能(t *testing.T) {
	// arrange
	tc := model.ToolCall{
		ID:   "call-001",
		Type: "function",
		Function: model.FunctionCall{
			Name:      "read_file",
			Arguments: `{"path": "test.txt"}`,
		},
	}
	result := tools.NewSuccessResult("文件内容")

	// act
	msg := BuildToolResultMessage(tc, result)

	// assert
	if msg.Role != model.RoleTool {
		t.Errorf("期望角色为 tool, 实际 '%s'", msg.Role)
	}
	if msg.ToolCallID != "call-001" {
		t.Errorf("期望 ToolCallID='call-001', 实际 '%s'", msg.ToolCallID)
	}
}

// TestBuildToolErrorMessage_基本功能 测试构建工具错误消息
func TestBuildToolErrorMessage_基本功能(t *testing.T) {
	// arrange
	tc := model.ToolCall{
		ID:   "call-002",
		Type: "function",
		Function: model.FunctionCall{
			Name:      "write_file",
			Arguments: `{}`,
		},
	}

	// act
	msg := BuildToolErrorMessage(tc, "权限不足")

	// assert
	if msg.Role != model.RoleTool {
		t.Errorf("期望角色为 tool, 实际 '%s'", msg.Role)
	}
	if msg.ToolCallID != "call-002" {
		t.Errorf("期望 ToolCallID='call-002', 实际 '%s'", msg.ToolCallID)
	}
}

// TestHandleSpecialTools_updateState 测试update_state特殊工具处理
func TestHandleSpecialTools_updateState(t *testing.T) {
	// arrange
	tmpDir, err := os.MkdirTemp("", "agent-exec-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stateManager, _ := state.NewStateManager(filepath.Join(tmpDir, "state"), true)
	_, _ = stateManager.CreateTask("test-task-001", "test goal")
	_ = stateManager.UpdateTaskStatus("test-task-001", "in_progress")

	registry := tools.NewToolRegistry()
	executor := NewToolExecutor(registry)

	agent := &Agent{
		stateManager: stateManager,
		executor:     executor,
		history:      NewHistoryManager(),
		taskID:       "test-task-001",
	}

	// act - 测试更新阶段
	args := map[string]interface{}{
		"phase":          "testing",
		"decision":       "使用单元测试",
		"completed_step": "编码完成",
	}
	argsJSON, _ := json.Marshal(args)
	tc := model.ToolCall{
		Function: model.FunctionCall{
			Name:      "update_state",
			Arguments: string(argsJSON),
		},
	}
	err = agent.handleSpecialTools(context.Background(), tc)

	// assert
	if err != nil {
		t.Fatalf("handleSpecialTools失败: %v", err)
	}

	// 验证状态已更新
	taskState, _ := stateManager.GetState("test-task-001")
	if taskState == nil {
		t.Fatal("任务状态不应为nil")
	}
	if taskState.Progress.CurrentPhase != "testing" {
		t.Errorf("期望 phase='testing', 实际 '%s'", taskState.Progress.CurrentPhase)
	}
}

// TestHandleSpecialTools_addFile 测试add_file特殊工具处理
func TestHandleSpecialTools_addFile(t *testing.T) {
	// arrange
	tmpDir, err := os.MkdirTemp("", "agent-exec-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stateManager, _ := state.NewStateManager(filepath.Join(tmpDir, "state"), true)
	_, _ = stateManager.CreateTask("test-task-002", "test goal")
	_ = stateManager.UpdateTaskStatus("test-task-002", "in_progress")

	registry := tools.NewToolRegistry()
	executor := NewToolExecutor(registry)

	agent := &Agent{
		stateManager: stateManager,
		executor:     executor,
		history:      NewHistoryManager(),
		taskID:       "test-task-002",
	}

	// act
	args := map[string]interface{}{
		"path":        "/tmp/test.go",
		"description": "测试文件",
		"status":      "created",
	}
	argsJSON, _ := json.Marshal(args)
	tc := model.ToolCall{
		Function: model.FunctionCall{
			Name:      "add_file",
			Arguments: string(argsJSON),
		},
	}
	err = agent.handleSpecialTools(context.Background(), tc)

	// assert
	if err != nil {
		t.Fatalf("handleSpecialTools失败: %v", err)
	}

	taskState, _ := stateManager.GetState("test-task-002")
	if taskState == nil {
		t.Fatal("任务状态不应为nil")
	}
	found := false
	for _, f := range taskState.Context.Files {
		if f.Path == "/tmp/test.go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("文件未被添加到任务状态中")
	}
}

// TestHandleSpecialTools_未知工具 测试不存在的特殊工具
func TestHandleSpecialTools_未知工具(t *testing.T) {
	// arrange
	tmpDir, _ := os.MkdirTemp("", "agent-exec-test-*")
	defer os.RemoveAll(tmpDir)

	stateManager, _ := state.NewStateManager(filepath.Join(tmpDir, "state"), true)
	registry := tools.NewToolRegistry()

	agent := &Agent{
		stateManager: stateManager,
		executor:     NewToolExecutor(registry),
		history:      NewHistoryManager(),
		taskID:       "test-task",
	}

	// act
	tc := model.ToolCall{
		Function: model.FunctionCall{
			Name:      "unknown_special_tool",
			Arguments: `{}`,
		},
	}
	err := agent.handleSpecialTools(context.Background(), tc)

	// assert
	if err != nil {
		t.Errorf("未知特殊工具不应返回错误, 实际: %v", err)
	}
}

// TestHandleSpecialTools_无效参数JSON 测试无效的参数JSON
func TestHandleSpecialTools_无效参数JSON(t *testing.T) {
	// arrange
	tmpDir, _ := os.MkdirTemp("", "agent-exec-test-*")
	defer os.RemoveAll(tmpDir)

	stateManager, _ := state.NewStateManager(filepath.Join(tmpDir, "state"), true)
	registry := tools.NewToolRegistry()

	agent := &Agent{
		stateManager: stateManager,
		executor:     NewToolExecutor(registry),
		history:      NewHistoryManager(),
		taskID:       "test-task",
	}

	// act
	tc := model.ToolCall{
		Function: model.FunctionCall{
			Name:      "update_state",
			Arguments: `{invalid}`,
		},
	}
	err := agent.handleSpecialTools(context.Background(), tc)

	// assert
	if err == nil {
		t.Error("期望参数解析失败")
	}
}

// TestFailTask_基本功能 测试任务失败处理
func TestFailTask_基本功能(t *testing.T) {
	// arrange
	tmpDir, _ := os.MkdirTemp("", "agent-exec-test-*")
	defer os.RemoveAll(tmpDir)

	stateManager, _ := state.NewStateManager(filepath.Join(tmpDir, "state"), true)
	_, _ = stateManager.CreateTask("test-task-fail", "test goal")
	_ = stateManager.UpdateTaskStatus("test-task-fail", "in_progress")

	registry := tools.NewToolRegistry()
	agent := &Agent{
		stateManager: stateManager,
		executor:     NewToolExecutor(registry),
		history:      NewHistoryManager(),
		taskID:       "test-task-fail",
	}

	result := &RunResult{
		TaskID:    "test-task-fail",
		StartTime: time.Now().Add(-1 * time.Minute),
	}

	// act
	ret, err := agent.failTask(result, 5)

	// assert
	if err != nil {
		t.Fatalf("failTask不应返回错误: %v", err)
	}
	if ret.Status != "failed" {
		t.Errorf("期望状态 'failed', 实际 '%s'", ret.Status)
	}
	if ret.Iterations != 5 {
		t.Errorf("期望迭代数 5, 实际 %d", ret.Iterations)
	}
	if ret.EndTime.IsZero() {
		t.Error("EndTime不应为零值")
	}
}

// TestExecuteTools_空工具调用 测试空工具调用列表
func TestExecuteTools_空工具调用(t *testing.T) {
	// arrange
	tmpDir, _ := os.MkdirTemp("", "agent-exec-test-*")
	defer os.RemoveAll(tmpDir)

	stateManager, _ := state.NewStateManager(filepath.Join(tmpDir, "state"), true)
	registry := tools.NewToolRegistry()

	agent := &Agent{
		stateManager: stateManager,
		executor:     NewToolExecutor(registry),
		history:      NewHistoryManager(),
		taskID:       "test-task",
	}

	// act
	results, err := agent.executeTools(context.Background(), []model.ToolCall{})

	// assert
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("期望0个结果, 实际 %d", len(results))
	}
}

// TestExecuteTools_工具不存在 测试工具不存在的情况
func TestExecuteTools_工具不存在(t *testing.T) {
	// arrange
	tmpDir, _ := os.MkdirTemp("", "agent-exec-test-*")
	defer os.RemoveAll(tmpDir)

	stateManager, _ := state.NewStateManager(filepath.Join(tmpDir, "state"), true)
	registry := tools.NewToolRegistry()

	agent := &Agent{
		stateManager: stateManager,
		executor:     NewToolExecutor(registry),
		history:      NewHistoryManager(),
		taskID:       "test-task",
	}

	// act - 调用不存在的工具
	toolCalls := []model.ToolCall{
		{
			ID:   "call-001",
			Type: "function",
			Function: model.FunctionCall{
				Name:      "nonexistent_tool",
				Arguments: `{}`,
			},
		},
	}
	results, err := agent.executeTools(context.Background(), toolCalls)

	// assert - 工具不存在时executor返回成功但包含错误信息
	if err != nil {
		t.Fatalf("工具不存在不应返回错误: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("期望1个结果, 实际 %d", len(results))
	}
	// 结果应包含工具不存在的错误信息
	var resultData map[string]interface{}
	json.Unmarshal([]byte(results[0].Content), &resultData)
	if success, ok := resultData["success"].(bool); ok && success {
		t.Error("不存在的工具调用结果不应为成功")
	}
}
