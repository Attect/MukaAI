package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
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

// TestCleanupConfig 测试清理配置
func TestCleanupConfig(t *testing.T) {
	// 测试默认清理配置
	cfg := DefaultCleanupConfig()
	if cfg.RetentionDays != 30 {
		t.Errorf("默认保留天数应为30，实际为 %d", cfg.RetentionDays)
	}
	if cfg.CheckInterval != 24*time.Hour {
		t.Errorf("默认检查间隔应为24小时")
	}
	if !cfg.Enabled {
		t.Error("默认应启用自动清理")
	}
}

// TestNewStateManagerWithCleanup 测试带清理配置的状态管理器创建
func TestNewStateManagerWithCleanup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cleanup_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 测试自定义配置
	cfg := CleanupConfig{
		RetentionDays: 7,
		CheckInterval: time.Hour,
		Enabled:       true,
	}
	manager, err := NewStateManagerWithCleanup(tempDir, true, cfg)
	if err != nil {
		t.Fatalf("创建状态管理器失败: %v", err)
	}

	if manager.cleanupConfig.RetentionDays != 7 {
		t.Errorf("保留天数应为7，实际为 %d", manager.cleanupConfig.RetentionDays)
	}
	if manager.cleanupConfig.CheckInterval != time.Hour {
		t.Error("检查间隔应为1小时")
	}
	if !manager.cleanupConfig.Enabled {
		t.Error("应启用自动清理")
	}
}

// TestNewStateManagerWithCleanup_InvalidConfig 测试无效配置会使用默认值
func TestNewStateManagerWithCleanup_InvalidConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cleanup_invalid_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 无效的配置值
	cfg := CleanupConfig{
		RetentionDays: -1,
		CheckInterval: -1,
		Enabled:       true,
	}
	manager, err := NewStateManagerWithCleanup(tempDir, true, cfg)
	if err != nil {
		t.Fatalf("创建状态管理器失败: %v", err)
	}

	// 应该被修正为默认值
	if manager.cleanupConfig.RetentionDays != 30 {
		t.Errorf("无效保留天数应被修正为30，实际为 %d", manager.cleanupConfig.RetentionDays)
	}
	if manager.cleanupConfig.CheckInterval != 24*time.Hour {
		t.Error("无效检查间隔应被修正为24小时")
	}
}

// TestCleanupNow 测试立即清理功能
func TestCleanupNow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cleanup_now_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := CleanupConfig{
		RetentionDays: 30,
		CheckInterval: 24 * time.Hour,
		Enabled:       false, // 禁用自动清理，仅测试手动清理
	}
	manager, err := NewStateManagerWithCleanup(tempDir, true, cfg)
	if err != nil {
		t.Fatalf("创建状态管理器失败: %v", err)
	}

	// 创建一个过期的已完成任务（更新时间设为31天前）
	expiredState := NewTaskState("expired-001", "过期任务")
	expiredState.UpdateStatus("completed")
	expiredState.Task.UpdatedAt = time.Now().AddDate(0, 0, -31)
	err = SaveYAML(expiredState, filepath.Join(tempDir, "task-expired-001.yaml"))
	if err != nil {
		t.Fatalf("保存过期状态文件失败: %v", err)
	}

	// 创建一个未过期的已完成任务
	recentState := NewTaskState("recent-001", "近期任务")
	recentState.UpdateStatus("completed")
	recentState.Task.UpdatedAt = time.Now().AddDate(0, 0, -10)
	err = SaveYAML(recentState, filepath.Join(tempDir, "task-recent-001.yaml"))
	if err != nil {
		t.Fatalf("保存近期状态文件失败: %v", err)
	}

	// 创建一个进行中的任务（即使过期也不应被清理）
	inProgressState := NewTaskState("progress-001", "进行中任务")
	inProgressState.UpdateStatus("in_progress")
	inProgressState.Task.UpdatedAt = time.Now().AddDate(0, 0, -60)
	err = SaveYAML(inProgressState, filepath.Join(tempDir, "task-progress-001.yaml"))
	if err != nil {
		t.Fatalf("保存进行中状态文件失败: %v", err)
	}

	// 创建一个过期的失败任务
	failedState := NewTaskState("failed-001", "失败任务")
	failedState.UpdateStatus("failed")
	failedState.Task.UpdatedAt = time.Now().AddDate(0, 0, -40)
	err = SaveYAML(failedState, filepath.Join(tempDir, "task-failed-001.yaml"))
	if err != nil {
		t.Fatalf("保存失败状态文件失败: %v", err)
	}

	// 执行清理
	cleaned, err := manager.CleanupNow()
	if err != nil {
		t.Fatalf("清理执行失败: %v", err)
	}

	// 应该清理2个文件：expired-001 和 failed-001
	if cleaned != 2 {
		t.Errorf("应清理2个文件，实际清理了 %d 个", cleaned)
	}

	// 验证文件状态
	if _, err := os.Stat(filepath.Join(tempDir, "task-expired-001.yaml")); !os.IsNotExist(err) {
		t.Error("过期已完成任务文件应已被删除")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "task-recent-001.yaml")); os.IsNotExist(err) {
		t.Error("近期已完成任务文件不应被删除")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "task-progress-001.yaml")); os.IsNotExist(err) {
		t.Error("进行中任务文件不应被删除，即使已过期")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "task-failed-001.yaml")); !os.IsNotExist(err) {
		t.Error("过期失败任务文件应已被删除")
	}
}

// TestCleanupNow_CancelledTask 测试已取消任务的清理
func TestCleanupNow_CancelledTask(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cleanup_cancelled_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := CleanupConfig{
		RetentionDays: 30,
		CheckInterval: 24 * time.Hour,
		Enabled:       false,
	}
	manager, err := NewStateManagerWithCleanup(tempDir, true, cfg)
	if err != nil {
		t.Fatalf("创建状态管理器失败: %v", err)
	}

	// 创建一个过期的已取消任务
	cancelledState := NewTaskState("cancelled-001", "已取消任务")
	cancelledState.UpdateStatus("cancelled")
	cancelledState.Task.UpdatedAt = time.Now().AddDate(0, 0, -35)
	err = SaveYAML(cancelledState, filepath.Join(tempDir, "task-cancelled-001.yaml"))
	if err != nil {
		t.Fatalf("保存已取消状态文件失败: %v", err)
	}

	cleaned, err := manager.CleanupNow()
	if err != nil {
		t.Fatalf("清理执行失败: %v", err)
	}
	if cleaned != 1 {
		t.Errorf("应清理1个已取消任务文件，实际清理了 %d 个", cleaned)
	}
}

// TestCleanupNow_EmptyDir 测试空目录的清理
func TestCleanupNow_EmptyDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cleanup_empty_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := CleanupConfig{RetentionDays: 30, Enabled: false}
	manager, err := NewStateManagerWithCleanup(tempDir, true, cfg)
	if err != nil {
		t.Fatalf("创建状态管理器失败: %v", err)
	}

	cleaned, err := manager.CleanupNow()
	if err != nil {
		t.Fatalf("清理空目录不应报错: %v", err)
	}
	if cleaned != 0 {
		t.Errorf("空目录应清理0个文件，实际清理了 %d 个", cleaned)
	}
}

// TestCleanupNow_RemovesFromMemoryCache 测试清理时同步移除内存缓存
func TestCleanupNow_RemovesFromMemoryCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cleanup_cache_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := CleanupConfig{RetentionDays: 30, Enabled: false}
	manager, err := NewStateManagerWithCleanup(tempDir, true, cfg)
	if err != nil {
		t.Fatalf("创建状态管理器失败: %v", err)
	}

	// 通过manager创建任务（加入内存缓存）
	state1, err := manager.CreateTask("cache-test-001", "缓存测试任务")
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}

	// 修改状态为已完成并设置更新时间为31天前
	state1.UpdateStatus("completed")
	state1.Task.UpdatedAt = time.Now().AddDate(0, 0, -31)
	err = manager.Save("cache-test-001")
	if err != nil {
		t.Fatalf("保存任务失败: %v", err)
	}

	// 确认内存缓存中存在
	_, err = manager.GetState("cache-test-001")
	if err != nil {
		t.Fatalf("任务应在内存缓存中: %v", err)
	}

	// 执行清理
	cleaned, err := manager.CleanupNow()
	if err != nil {
		t.Fatalf("清理失败: %v", err)
	}
	if cleaned != 1 {
		t.Errorf("应清理1个文件，实际清理了 %d 个", cleaned)
	}

	// 内存缓存中应该已被移除
	_, err = manager.GetState("cache-test-001")
	if err == nil {
		t.Error("清理后内存缓存应已移除该任务")
	}
}

// TestListTasksFromDisk 测试从磁盘扫描任务列表
func TestListTasksFromDisk(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "list_disk_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewStateManager(tempDir, true)
	if err != nil {
		t.Fatalf("创建状态管理器失败: %v", err)
	}

	// 创建几个任务文件
	_, err = manager.CreateTask("disk-001", "磁盘任务1")
	if err != nil {
		t.Fatalf("创建任务1失败: %v", err)
	}
	_, err = manager.CreateTask("disk-002", "磁盘任务2")
	if err != nil {
		t.Fatalf("创建任务2失败: %v", err)
	}
	_, err = manager.CreateTask("disk-003", "磁盘任务3")
	if err != nil {
		t.Fatalf("创建任务3失败: %v", err)
	}

	// 创建一个新的manager实例（无内存缓存）来测试磁盘扫描
	manager2, err := NewStateManager(tempDir, false)
	if err != nil {
		t.Fatalf("创建第二个状态管理器失败: %v", err)
	}

	// ListTasks只查内存，应为空
	memoryIDs, err := manager2.ListTasks()
	if err != nil {
		t.Fatalf("ListTasks失败: %v", err)
	}
	if len(memoryIDs) != 0 {
		t.Errorf("新管理器的内存缓存应为空，实际有 %d 个任务", len(memoryIDs))
	}

	// ListTasksFromDisk应能扫描到3个文件
	diskIDs, err := manager2.ListTasksFromDisk()
	if err != nil {
		t.Fatalf("ListTasksFromDisk失败: %v", err)
	}
	if len(diskIDs) != 3 {
		t.Errorf("磁盘应扫描到3个任务，实际扫描到 %d 个", len(diskIDs))
	}

	// 验证包含预期的任务ID
	found := make(map[string]bool)
	for _, id := range diskIDs {
		found[id] = true
	}
	for _, expected := range []string{"disk-001", "disk-002", "disk-003"} {
		if !found[expected] {
			t.Errorf("磁盘扫描结果应包含任务ID %s", expected)
		}
	}
}

// TestStartStopCleanup 测试清理goroutine的启动和停止
func TestStartStopCleanup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "start_stop_cleanup_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := CleanupConfig{
		RetentionDays: 30,
		CheckInterval: 1 * time.Second, // 短间隔用于测试
		Enabled:       true,
	}
	manager, err := NewStateManagerWithCleanup(tempDir, true, cfg)
	if err != nil {
		t.Fatalf("创建状态管理器失败: %v", err)
	}

	// 启动清理goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.StartCleanup(ctx)

	// 创建一个过期的已完成任务
	expiredState := NewTaskState("auto-clean-001", "自动清理测试")
	expiredState.UpdateStatus("completed")
	expiredState.Task.UpdatedAt = time.Now().AddDate(0, 0, -31)
	err = SaveYAML(expiredState, filepath.Join(tempDir, "task-auto-clean-001.yaml"))
	if err != nil {
		t.Fatalf("保存状态文件失败: %v", err)
	}

	// 等待初始清理完成
	time.Sleep(100 * time.Millisecond)

	// 过期文件应已被初始清理删除
	if _, err := os.Stat(filepath.Join(tempDir, "task-auto-clean-001.yaml")); !os.IsNotExist(err) {
		t.Error("启动清理后过期文件应已被删除")
	}

	// 停止清理goroutine
	manager.StopCleanup()

	// 创建另一个过期文件，停止后不应被清理
	expiredState2 := NewTaskState("auto-clean-002", "停止后测试")
	expiredState2.UpdateStatus("completed")
	expiredState2.Task.UpdatedAt = time.Now().AddDate(0, 0, -31)
	err = SaveYAML(expiredState2, filepath.Join(tempDir, "task-auto-clean-002.yaml"))
	if err != nil {
		t.Fatalf("保存状态文件失败: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// 停止后文件应仍然存在
	if _, err := os.Stat(filepath.Join(tempDir, "task-auto-clean-002.yaml")); os.IsNotExist(err) {
		t.Error("停止清理后文件不应被删除")
	}
}

// TestStartCleanup_Disabled 测试禁用清理时不启动goroutine
func TestStartCleanup_Disabled(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cleanup_disabled_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := CleanupConfig{
		RetentionDays: 30,
		CheckInterval: time.Second,
		Enabled:       false, // 禁用
	}
	manager, err := NewStateManagerWithCleanup(tempDir, true, cfg)
	if err != nil {
		t.Fatalf("创建状态管理器失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动清理（应无效果因为已禁用）
	manager.StartCleanup(ctx)

	// 创建过期任务
	expiredState := NewTaskState("disabled-001", "禁用测试")
	expiredState.UpdateStatus("completed")
	expiredState.Task.UpdatedAt = time.Now().AddDate(0, 0, -31)
	err = SaveYAML(expiredState, filepath.Join(tempDir, "task-disabled-001.yaml"))
	if err != nil {
		t.Fatalf("保存状态文件失败: %v", err)
	}

	// 等待确认不会被清理
	time.Sleep(200 * time.Millisecond)

	if _, err := os.Stat(filepath.Join(tempDir, "task-disabled-001.yaml")); os.IsNotExist(err) {
		t.Error("禁用清理时文件不应被删除")
	}

	// StopCleanup在未启动时应安全调用
	manager.StopCleanup()
}

// TestStopCleanup_Idempotent 测试多次调用StopCleanup的安全性
func TestStopCleanup_Idempotent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "stop_idempotent_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewStateManager(tempDir, false)
	if err != nil {
		t.Fatalf("创建状态管理器失败: %v", err)
	}

	// 多次调用StopCleanup不应panic
	manager.StopCleanup()
	manager.StopCleanup()
	manager.StopCleanup()
}

// TestCleanupNow_PendingTaskNotCleaned 测试pending状态的任务不被清理
func TestCleanupNow_PendingTaskNotCleaned(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cleanup_pending_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := CleanupConfig{RetentionDays: 30, Enabled: false}
	manager, err := NewStateManagerWithCleanup(tempDir, true, cfg)
	if err != nil {
		t.Fatalf("创建状态管理器失败: %v", err)
	}

	// 创建一个过期的pending任务
	pendingState := NewTaskState("pending-001", "待处理过期任务")
	pendingState.Task.Status = "pending"
	pendingState.Task.UpdatedAt = time.Now().AddDate(0, 0, -60)
	err = SaveYAML(pendingState, filepath.Join(tempDir, "task-pending-001.yaml"))
	if err != nil {
		t.Fatalf("保存状态文件失败: %v", err)
	}

	cleaned, err := manager.CleanupNow()
	if err != nil {
		t.Fatalf("清理失败: %v", err)
	}
	if cleaned != 0 {
		t.Errorf("pending状态的任务不应被清理，实际清理了 %d 个", cleaned)
	}

	// 文件应仍然存在
	if _, err := os.Stat(filepath.Join(tempDir, "task-pending-001.yaml")); os.IsNotExist(err) {
		t.Error("pending状态的任务文件不应被删除")
	}
}
