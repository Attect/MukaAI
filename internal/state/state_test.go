package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTaskState(t *testing.T) {
	id := "test-001"
	goal := "测试任务目标"

	state := NewTaskState(id, goal)

	if state.Task.ID != id {
		t.Errorf("Expected ID %s, got %s", id, state.Task.ID)
	}

	if state.Task.Goal != goal {
		t.Errorf("Expected Goal %s, got %s", goal, state.Task.Goal)
	}

	if state.Task.Status != "pending" {
		t.Errorf("Expected Status pending, got %s", state.Task.Status)
	}

	if state.Progress.CurrentPhase != "initialization" {
		t.Errorf("Expected CurrentPhase initialization, got %s", state.Progress.CurrentPhase)
	}

	if state.Agents.Active != "Orchestrator" {
		t.Errorf("Expected Active Agent Orchestrator, got %s", state.Agents.Active)
	}
}

func TestTaskStateMethods(t *testing.T) {
	state := NewTaskState("test-002", "测试方法")

	// 测试状态检查方法
	if state.IsCompleted() {
		t.Error("Task should not be completed initially")
	}

	if state.IsInProgress() {
		t.Error("Task should not be in progress initially")
	}

	if state.IsFailed() {
		t.Error("Task should not be failed initially")
	}

	// 测试更新状态
	state.UpdateStatus("in_progress")
	if !state.IsInProgress() {
		t.Error("Task should be in progress after update")
	}

	// 测试添加完成步骤
	state.AddCompletedStep("步骤1")
	if len(state.Progress.CompletedSteps) != 1 {
		t.Errorf("Expected 1 completed step, got %d", len(state.Progress.CompletedSteps))
	}

	// 测试添加决策
	state.AddDecision("选择Go语言实现")
	if len(state.Context.Decisions) != 1 {
		t.Errorf("Expected 1 decision, got %d", len(state.Context.Decisions))
	}

	// 测试添加约束
	state.AddConstraint("不使用外部依赖")
	if len(state.Context.Constraints) != 1 {
		t.Errorf("Expected 1 constraint, got %d", len(state.Context.Constraints))
	}

	// 测试添加文件
	state.AddFile("/path/to/file.go", "测试文件", "created")
	if len(state.Context.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(state.Context.Files))
	}

	// 测试添加Agent记录
	state.AddAgentRecord("Architect", "完成架构设计", "5m")
	if len(state.Agents.History) != 1 {
		t.Errorf("Expected 1 agent record, got %d", len(state.Agents.History))
	}

	// 测试设置活动Agent
	state.SetActiveAgent("Developer")
	if state.Agents.Active != "Developer" {
		t.Errorf("Expected Active Agent Developer, got %s", state.Agents.Active)
	}

	// 测试设置当前阶段
	state.SetCurrentPhase("implementation")
	if state.Progress.CurrentPhase != "implementation" {
		t.Errorf("Expected CurrentPhase implementation, got %s", state.Progress.CurrentPhase)
	}
}

func TestYAMLSerialization(t *testing.T) {
	state := NewTaskState("test-003", "YAML序列化测试")
	state.UpdateStatus("in_progress")
	state.AddCompletedStep("需求分析完成")
	state.AddDecision("使用YAML格式存储状态")
	state.AddConstraint("必须支持并发安全")
	state.SetActiveAgent("Developer")
	state.SetCurrentPhase("implementation")

	// 序列化为YAML
	yamlData, err := ToYAML(state)
	if err != nil {
		t.Fatalf("Failed to serialize to YAML: %v", err)
	}

	// 反序列化
	loadedState, err := ParseYAML(yamlData)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// 验证反序列化结果
	if loadedState.Task.ID != state.Task.ID {
		t.Errorf("Expected ID %s, got %s", state.Task.ID, loadedState.Task.ID)
	}

	if loadedState.Task.Goal != state.Task.Goal {
		t.Errorf("Expected Goal %s, got %s", state.Task.Goal, loadedState.Task.Goal)
	}

	if loadedState.Task.Status != state.Task.Status {
		t.Errorf("Expected Status %s, got %s", state.Task.Status, loadedState.Task.Status)
	}

	if len(loadedState.Progress.CompletedSteps) != len(state.Progress.CompletedSteps) {
		t.Errorf("Expected %d completed steps, got %d",
			len(state.Progress.CompletedSteps), len(loadedState.Progress.CompletedSteps))
	}

	if len(loadedState.Context.Decisions) != len(state.Context.Decisions) {
		t.Errorf("Expected %d decisions, got %d",
			len(state.Context.Decisions), len(loadedState.Context.Decisions))
	}
}

func TestYAMLFileOperations(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "state_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	state := NewTaskState("test-004", "文件操作测试")
	state.AddCompletedStep("步骤1")
	state.AddDecision("决策1")

	// 保存到文件
	filePath := filepath.Join(tempDir, "task-test-004.yaml")
	err = SaveYAML(state, filePath)
	if err != nil {
		t.Fatalf("Failed to save YAML: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("File %s does not exist", filePath)
	}

	// 从文件加载
	loadedState, err := LoadYAML(filePath)
	if err != nil {
		t.Fatalf("Failed to load YAML: %v", err)
	}

	// 验证加载结果
	if loadedState.Task.ID != state.Task.ID {
		t.Errorf("Expected ID %s, got %s", state.Task.ID, loadedState.Task.ID)
	}
}

func TestGetYAMLSummary(t *testing.T) {
	state := NewTaskState("test-005", "摘要测试")
	state.UpdateStatus("in_progress")
	state.AddCompletedStep("步骤1")
	state.AddCompletedStep("步骤2")
	state.AddDecision("决策1")
	state.AddDecision("决策2")
	state.SetActiveAgent("Developer")

	summary, err := GetYAMLSummary(state)
	if err != nil {
		t.Fatalf("Failed to get YAML summary: %v", err)
	}

	// 验证摘要包含关键信息
	if summary == "" {
		t.Error("Summary should not be empty")
	}

	// 摘要应该包含任务ID
	if !contains(summary, "test-005") {
		t.Error("Summary should contain task ID")
	}

	// 摘要应该包含任务目标
	if !contains(summary, "摘要测试") {
		t.Error("Summary should contain task goal")
	}

	// 摘要应该包含状态
	if !contains(summary, "in_progress") {
		t.Error("Summary should contain task status")
	}
}

func TestStateManager(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "state_manager_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建状态管理器（启用自动保存）
	manager, err := NewStateManager(tempDir, true)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// 测试创建任务
	state, err := manager.CreateTask("test-006", "状态管理器测试")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	if state.Task.ID != "test-006" {
		t.Errorf("Expected ID test-006, got %s", state.Task.ID)
	}

	// 验证文件已创建
	filePath := filepath.Join(tempDir, "task-test-006.yaml")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("State file should be created")
	}

	// 测试更新进度
	err = manager.UpdateProgress("test-006", "implementation", "步骤1完成")
	if err != nil {
		t.Fatalf("Failed to update progress: %v", err)
	}

	// 验证进度已更新
	loadedState, err := manager.GetState("test-006")
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}

	if loadedState.Progress.CurrentPhase != "implementation" {
		t.Errorf("Expected phase implementation, got %s", loadedState.Progress.CurrentPhase)
	}

	// 测试添加决策
	err = manager.AddDecision("test-006", "使用标准库")
	if err != nil {
		t.Fatalf("Failed to add decision: %v", err)
	}

	// 测试切换Agent
	err = manager.SwitchAgent("test-006", "Architect", "完成开发", "10m")
	if err != nil {
		t.Fatalf("Failed to switch agent: %v", err)
	}

	// 验证Agent已切换
	loadedState, _ = manager.GetState("test-006")
	if loadedState.Agents.Active != "Architect" {
		t.Errorf("Expected active agent Architect, got %s", loadedState.Agents.Active)
	}

	// 测试删除任务
	err = manager.DeleteTask("test-006")
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// 验证文件已删除
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("State file should be deleted")
	}
}

func TestStateManagerConcurrency(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "state_concurrent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewStateManager(tempDir, false) // 禁用自动保存以提高性能
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// 创建任务
	_, err = manager.CreateTask("test-007", "并发测试")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// 并发测试：多个goroutine同时更新状态
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(index int) {
			for j := 0; j < 100; j++ {
				decision := "决策" + string(rune('A'+index))
				manager.AddDecision("test-007", decision)
			}
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证最终状态
	state, err := manager.GetState("test-007")
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}

	// 应该有1000个决策（10个goroutine * 100次）
	if len(state.Context.Decisions) != 1000 {
		t.Errorf("Expected 1000 decisions, got %d", len(state.Context.Decisions))
	}
}

func TestStateManagerLoadFromFile(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "state_load_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 先创建一个状态文件
	originalState := NewTaskState("test-008", "从文件加载测试")
	originalState.UpdateStatus("in_progress")
	originalState.AddCompletedStep("步骤1")
	filePath := filepath.Join(tempDir, "task-test-008.yaml")
	err = SaveYAML(originalState, filePath)
	if err != nil {
		t.Fatalf("Failed to save original state: %v", err)
	}

	// 创建新的状态管理器
	manager, err := NewStateManager(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// 从文件加载
	loadedState, err := manager.Load("test-008")
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// 验证加载的状态
	if loadedState.Task.ID != "test-008" {
		t.Errorf("Expected ID test-008, got %s", loadedState.Task.ID)
	}

	if loadedState.Task.Goal != "从文件加载测试" {
		t.Errorf("Expected goal '从文件加载测试', got %s", loadedState.Task.Goal)
	}

	if loadedState.Task.Status != "in_progress" {
		t.Errorf("Expected status in_progress, got %s", loadedState.Task.Status)
	}
}

// 辅助函数：检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestTimeSerialization(t *testing.T) {
	// 测试时间字段的序列化和反序列化
	state := NewTaskState("test-time", "时间测试")
	originalTime := state.Task.CreatedAt

	// 序列化
	yamlData, err := ToYAML(state)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// 反序列化
	loadedState, err := ParseYAML(yamlData)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// 验证时间字段能够正确序列化和反序列化
	if !loadedState.Task.CreatedAt.Equal(originalTime) {
		t.Errorf("Time mismatch: expected %v, got %v", originalTime, loadedState.Task.CreatedAt)
	}

	// 验证UpdatedAt也能正确处理
	if !loadedState.Task.UpdatedAt.Equal(state.Task.UpdatedAt) {
		t.Errorf("UpdatedAt mismatch: expected %v, got %v", state.Task.UpdatedAt, loadedState.Task.UpdatedAt)
	}
}
