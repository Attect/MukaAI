package supervisor

import (
	"context"
	"testing"
	"time"

	"github.com/Attect/MukaAI/internal/agent"
	"github.com/Attect/MukaAI/internal/model"
	"github.com/Attect/MukaAI/internal/state"
	"github.com/Attect/MukaAI/internal/tools"
)

// 创建测试用的监督器
func createTestSupervisor(t *testing.T) *Supervisor {
	// 创建模型客户端配置
	modelConfig := &model.Config{
		Endpoint:    "http://localhost:8080",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 4096,
	}

	// 创建模型客户端
	modelClient, err := model.NewClient(modelConfig)
	if err != nil {
		t.Fatalf("failed to create model client: %v", err)
	}

	// 创建工具注册中心
	toolRegistry := tools.NewToolRegistry()

	// 创建状态管理器
	stateManager, err := state.NewStateManager(t.TempDir(), false)
	if err != nil {
		t.Fatalf("failed to create state manager: %v", err)
	}

	// 创建审查器
	reviewer := agent.NewReviewer(agent.DefaultReviewConfig())

	// 创建监督器
	supervisor, err := NewSupervisor(modelClient, toolRegistry, stateManager, reviewer, DefaultSupervisorConfig())
	if err != nil {
		t.Fatalf("failed to create supervisor: %v", err)
	}

	return supervisor
}

// 创建测试用的任务状态
func createTestTaskState(t *testing.T, sm *state.StateManager) *state.TaskState {
	taskState, err := sm.CreateTask("test-task-1", "测试任务目标")
	if err != nil {
		t.Fatalf("failed to create task state: %v", err)
	}
	return taskState
}

// TestNewSupervisor 测试创建监督器
func TestNewSupervisor(t *testing.T) {
	supervisor := createTestSupervisor(t)
	if supervisor == nil {
		t.Fatal("supervisor should not be nil")
	}

	// 检查默认配置
	if !supervisor.config.EnableQualityCheck {
		t.Error("quality check should be enabled by default")
	}
	if !supervisor.config.EnableProgressCheck {
		t.Error("progress check should be enabled by default")
	}
	if !supervisor.config.AutoIntervene {
		t.Error("auto intervene should be enabled by default")
	}
}

// TestNewSupervisorWithNilConfig 测试使用nil配置创建监督器
func TestNewSupervisorWithNilConfig(t *testing.T) {
	modelConfig := &model.Config{
		Endpoint:    "http://localhost:8080",
		APIKey:      "test-key",
		ModelName:   "test-model",
		ContextSize: 4096,
	}
	modelClient, _ := model.NewClient(modelConfig)
	toolRegistry := tools.NewToolRegistry()
	stateManager, _ := state.NewStateManager(t.TempDir(), false)
	reviewer := agent.NewReviewer(nil)

	supervisor, err := NewSupervisor(modelClient, toolRegistry, stateManager, reviewer, nil)
	if err != nil {
		t.Fatalf("should not fail with nil config: %v", err)
	}

	if supervisor.config == nil {
		t.Error("config should be set to default")
	}
}

// TestMonitorWithEmptyOutput 测试监督空输出
func TestMonitorWithEmptyOutput(t *testing.T) {
	supervisor := createTestSupervisor(t)
	taskState := createTestTaskState(t, supervisor.stateManager)

	output := &AgentOutput{
		Content:   "",
		ToolCalls: nil,
		Timestamp: time.Now(),
		TaskID:    "test-task-1",
		AgentRole: "developer",
		Iteration: 1,
		Success:   true,
	}

	result := supervisor.Monitor(context.Background(), output, taskState)

	if result.Status != "warning" {
		t.Errorf("expected warning status for empty output, got: %s", result.Status)
	}

	if len(result.Issues) == 0 {
		t.Error("should find issues for empty output")
	}

	// 检查是否发现质量问题
	found := false
	for _, issue := range result.Issues {
		if issue.Type == IssueTypeQuality {
			found = true
			break
		}
	}
	if !found {
		t.Error("should find quality issue for empty output")
	}
}

// TestMonitorWithValidOutput 测试监督有效输出
func TestMonitorWithValidOutput(t *testing.T) {
	supervisor := createTestSupervisor(t)
	taskState := createTestTaskState(t, supervisor.stateManager)

	output := &AgentOutput{
		Content:   "这是一个有效的输出内容，包含了足够的信息。",
		ToolCalls: nil,
		Timestamp: time.Now(),
		TaskID:    "test-task-1",
		AgentRole: "developer",
		Iteration: 1,
		Success:   true,
	}

	result := supervisor.Monitor(context.Background(), output, taskState)

	// 有效输出应该通过监督
	if result.Status == "intervention" {
		t.Errorf("should not need intervention for valid output, got: %s", result.Status)
	}
}

// TestMonitorWithError 测试监督错误输出
func TestMonitorWithError(t *testing.T) {
	supervisor := createTestSupervisor(t)
	taskState := createTestTaskState(t, supervisor.stateManager)

	output := &AgentOutput{
		Content:   "执行失败",
		ToolCalls: nil,
		Timestamp: time.Now(),
		TaskID:    "test-task-1",
		AgentRole: "developer",
		Iteration: 1,
		Success:   false,
		Error:     "工具执行失败: 文件不存在",
	}

	result := supervisor.Monitor(context.Background(), output, taskState)

	if result.Status == "pass" {
		t.Error("should not pass for error output")
	}

	// 检查是否发现错误问题
	found := false
	for _, issue := range result.Issues {
		if issue.Type == IssueTypeError {
			found = true
			break
		}
	}
	if !found {
		t.Error("should find error issue")
	}
}

// TestMonitorWithSecurityIssue 测试监督安全问题
func TestSupervisor_checkSecurity(t *testing.T) {
	supervisor := createTestSupervisor(t)
	supervisor.config.EnableSecurityCheck = true
	taskState := createTestTaskState(t, supervisor.stateManager)

	output := &AgentOutput{
		Content: "执行危险命令",
		ToolCalls: []model.ToolCall{
			{
				ID:   "tc-1",
				Type: "function",
				Function: model.FunctionCall{
					Name:      "execute_command",
					Arguments: `{"command": "rm -rf /"}`,
				},
			},
		},
		Timestamp: time.Now(),
		TaskID:    "test-task-1",
		AgentRole: "developer",
		Iteration: 1,
		Success:   true,
	}

	result := supervisor.Monitor(context.Background(), output, taskState)

	// 应该发现安全问题
	found := false
	for _, issue := range result.Issues {
		if issue.Type == IssueTypeSecurity {
			found = true
			if issue.Severity != "critical" {
				t.Errorf("security issue should be critical, got: %s", issue.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("should find security issue for dangerous command")
	}
}

// TestIntervene 测试干预机制
func TestIntervene(t *testing.T) {
	supervisor := createTestSupervisor(t)

	issue := SupervisionIssue{
		Type:        IssueTypeError,
		Severity:    "high",
		Description: "测试问题",
		Evidence:    "测试证据",
		Suggestion:  "测试建议",
		Timestamp:   time.Now(),
	}

	record := supervisor.Intervene(context.Background(), issue)

	if record == nil {
		t.Fatal("intervention record should not be nil")
	}

	// 高严重度在警告计数未达上限时为warning而非pause
	if record.Type != InterventionWarning {
		t.Errorf("expected warning intervention for high severity with low warning count, got: %s", record.Type)
	}

	// 检查统计更新
	stats := supervisor.GetStatistics()
	if stats.Interventions != 1 {
		t.Errorf("expected 1 intervention, got: %d", stats.Interventions)
	}
}

// TestInterveneWithCriticalIssue 测试严重问题的干预
func TestInterveneWithCriticalIssue(t *testing.T) {
	supervisor := createTestSupervisor(t)

	issue := SupervisionIssue{
		Type:        IssueTypeSecurity,
		Severity:    "critical",
		Description: "严重安全问题",
		Evidence:    "执行危险命令",
		Suggestion:  "立即停止",
		Timestamp:   time.Now(),
	}

	record := supervisor.Intervene(context.Background(), issue)

	if record.Type != InterventionInterrupt {
		t.Errorf("expected interrupt for critical security issue, got: %s", record.Type)
	}
}

// TestParallelMonitor 测试并行监督
func TestParallelMonitor(t *testing.T) {
	supervisor := createTestSupervisor(t)
	_ = createTestTaskState(t, supervisor.stateManager)

	// 创建输出通道
	outputs := make(chan *AgentOutput, 5)

	// 发送测试输出
	go func() {
		for i := 0; i < 5; i++ {
			outputs <- &AgentOutput{
				Content:   "测试输出",
				ToolCalls: nil,
				Timestamp: time.Now(),
				TaskID:    "test-task-1",
				AgentRole: "developer",
				Iteration: i + 1,
				Success:   true,
			}
		}
		close(outputs)
	}()

	// 并行监督
	resultChan := supervisor.ParallelMonitor(context.Background(), outputs)

	// 收集结果
	count := 0
	for range resultChan {
		count++
	}

	if count != 5 {
		t.Errorf("expected 5 results, got: %d", count)
	}
}

// TestDetermineStatus 测试确定监督状态
func TestDetermineStatus(t *testing.T) {
	supervisor := createTestSupervisor(t)

	tests := []struct {
		name     string
		issues   []SupervisionIssue
		expected string
	}{
		{
			name:     "无问题",
			issues:   []SupervisionIssue{},
			expected: "pass",
		},
		{
			name: "低严重度问题",
			issues: []SupervisionIssue{
				{Severity: "low"},
			},
			expected: "warning",
		},
		{
			name: "中等严重度问题",
			issues: []SupervisionIssue{
				{Severity: "medium"},
			},
			expected: "warning",
		},
		{
			name: "高严重度问题",
			issues: []SupervisionIssue{
				{Severity: "high"},
			},
			expected: "warning",
		},
		{
			name: "严重问题",
			issues: []SupervisionIssue{
				{Severity: "critical"},
			},
			expected: "intervention",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := supervisor.determineStatus(tt.issues)
			if status != tt.expected {
				t.Errorf("expected status %s, got: %s", tt.expected, status)
			}
		})
	}
}

// TestDetermineInterventionType 测试确定干预类型
func TestDetermineInterventionType(t *testing.T) {
	supervisor := createTestSupervisor(t)

	tests := []struct {
		name     string
		issue    SupervisionIssue
		expected InterventionType
	}{
		{
			name: "严重安全问题",
			issue: SupervisionIssue{
				Severity: "critical",
				Type:     IssueTypeSecurity,
			},
			expected: InterventionInterrupt,
		},
		{
			name: "严重其他问题",
			issue: SupervisionIssue{
				Severity: "critical",
				Type:     IssueTypeError,
			},
			expected: InterventionRollback,
		},
		{
			name: "高严重度问题（警告计数未达上限）",
			issue: SupervisionIssue{
				Severity: "high",
				Type:     IssueTypeQuality,
			},
			expected: InterventionWarning, // 警告计数未达到MaxWarnings时只返回warning
		},
		{
			name: "中等严重度问题",
			issue: SupervisionIssue{
				Severity: "medium",
				Type:     IssueTypeProgress,
			},
			expected: InterventionWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interventionType := supervisor.determineInterventionType(tt.issue)
			if interventionType != tt.expected {
				t.Errorf("expected intervention type %s, got: %s", tt.expected, interventionType)
			}
		})
	}
}

// TestGetStatistics 测试获取统计信息
func TestGetStatistics(t *testing.T) {
	supervisor := createTestSupervisor(t)
	taskState := createTestTaskState(t, supervisor.stateManager)

	// 执行几次监督
	for i := 0; i < 3; i++ {
		output := &AgentOutput{
			Content:   "测试输出",
			Timestamp: time.Now(),
			TaskID:    "test-task-1",
			AgentRole: "developer",
			Iteration: i + 1,
			Success:   true,
		}
		supervisor.Monitor(context.Background(), output, taskState)
	}

	stats := supervisor.GetStatistics()

	if stats.TotalChecks != 3 {
		t.Errorf("expected 3 total checks, got: %d", stats.TotalChecks)
	}
}

// TestGetInterventionLog 测试获取干预日志
func TestGetInterventionLog(t *testing.T) {
	supervisor := createTestSupervisor(t)

	// 执行干预
	issue := SupervisionIssue{
		Type:        IssueTypeError,
		Severity:    "high",
		Description: "测试问题",
		Timestamp:   time.Now(),
	}
	supervisor.Intervene(context.Background(), issue)

	log := supervisor.GetInterventionLog()

	if len(log) != 1 {
		t.Errorf("expected 1 intervention log, got: %d", len(log))
	}

	if log[0].Type != InterventionWarning {
		t.Errorf("expected warning intervention, got: %s", log[0].Type)
	}
}

// TestReset 测试重置监督器
func TestReset(t *testing.T) {
	supervisor := createTestSupervisor(t)

	// 执行一些操作
	issue := SupervisionIssue{
		Type:        IssueTypeError,
		Severity:    "high",
		Description: "测试问题",
		Timestamp:   time.Now(),
	}
	supervisor.Intervene(context.Background(), issue)

	// 重置
	supervisor.Reset()

	// 检查状态
	stats := supervisor.GetStatistics()
	if stats.TotalChecks != 0 {
		t.Error("total checks should be 0 after reset")
	}

	log := supervisor.GetInterventionLog()
	if len(log) != 0 {
		t.Error("intervention log should be empty after reset")
	}
}

// TestSaveStableState 测试保存稳定状态
func TestSaveStableState(t *testing.T) {
	supervisor := createTestSupervisor(t)
	taskState := createTestTaskState(t, supervisor.stateManager)

	// 保存稳定状态
	supervisor.SaveStableState(taskState)

	// 检查是否保存
	supervisor.mu.RLock()
	lastStable := supervisor.lastStableState
	supervisor.mu.RUnlock()

	if lastStable == nil {
		t.Error("stable state should be saved")
	}

	if lastStable.Task.ID != taskState.Task.ID {
		t.Error("stable state should match original state")
	}
}

// TestCallbacks 测试回调函数
func TestCallbacks(t *testing.T) {
	supervisor := createTestSupervisor(t)
	taskState := createTestTaskState(t, supervisor.stateManager)

	// 设置回调
	_ = false // interventionCalled - not used in this test
	_ = false // warningCalled - not used in this test
	issueFoundCalled := false

	supervisor.SetOnIntervention(func(record InterventionRecord) {
		// interventionCalled = true
	})

	supervisor.SetOnWarning(func(issue SupervisionIssue) {
		// warningCalled = true
	})

	supervisor.SetOnIssueFound(func(issue SupervisionIssue) {
		issueFoundCalled = true
	})

	// 执行监督（触发问题）
	output := &AgentOutput{
		Content:   "", // 空输出会触发问题
		Timestamp: time.Now(),
		TaskID:    "test-task-1",
		AgentRole: "developer",
		Iteration: 1,
		Success:   true,
	}
	supervisor.Monitor(context.Background(), output, taskState)

	// 检查回调是否被调用
	if !issueFoundCalled {
		t.Error("issue found callback should be called")
	}

	// 测试干预回调
	interventionCalled := false
	supervisor.SetOnIntervention(func(record InterventionRecord) {
		interventionCalled = true
	})

	// 执行干预
	issue := SupervisionIssue{
		Type:        IssueTypeError,
		Severity:    "high",
		Description: "测试问题",
		Timestamp:   time.Now(),
	}
	supervisor.Intervene(context.Background(), issue)

	if !interventionCalled {
		t.Error("intervention callback should be called")
	}

	// 测试警告回调
	warningCalled := false
	supervisor.SetOnWarning(func(issue SupervisionIssue) {
		warningCalled = true
	})

	// 重置监督器状态
	supervisor.Reset()

	// 执行监督，触发警告级别的输出
	warningOutput := &AgentOutput{
		Content:   "短", // 过短的输出会触发警告
		Timestamp: time.Now(),
		TaskID:    "test-task-1",
		AgentRole: "developer",
		Iteration: 1,
		Success:   true,
	}
	warningResult := supervisor.Monitor(context.Background(), warningOutput, taskState)

	// 只有当状态为warning时才会触发警告回调
	if warningResult.Status == "warning" && !warningCalled {
		t.Error("warning callback should be called for warning status")
	}
}

// TestConsecutiveErrors 测试连续错误检测
func TestConsecutiveErrors(t *testing.T) {
	supervisor := createTestSupervisor(t)
	supervisor.config.MaxConsecutiveErrors = 3
	taskState := createTestTaskState(t, supervisor.stateManager)

	// 连续产生错误
	for i := 0; i < 4; i++ {
		output := &AgentOutput{
			Content:   "执行失败",
			Timestamp: time.Now(),
			TaskID:    "test-task-1",
			AgentRole: "developer",
			Iteration: i + 1,
			Success:   false,
			Error:     "工具执行失败",
		}
		result := supervisor.Monitor(context.Background(), output, taskState)

		// 第4次应该触发严重问题
		if i == 3 {
			found := false
			for _, issue := range result.Issues {
				if issue.Severity == "critical" && issue.Type == IssueTypeError {
					found = true
					break
				}
			}
			if !found {
				t.Error("should detect consecutive errors as critical issue")
			}
		}
	}
}

// TestSupervisionResultMethods 测试监督结果方法
func TestSupervisionResultMethods(t *testing.T) {
	result := &SupervisionResult{
		Status: "intervention",
		Issues: []SupervisionIssue{
			{Severity: "low", Type: IssueTypeQuality},
			{Severity: "high", Type: IssueTypeError},
			{Severity: "critical", Type: IssueTypeSecurity},
		},
		Intervention: &InterventionRecord{Type: InterventionInterrupt},
	}

	// 测试 IsInterventionNeeded
	if !result.IsInterventionNeeded() {
		t.Error("should need intervention")
	}

	// 测试 GetCriticalIssues
	critical := result.GetCriticalIssues()
	if len(critical) != 2 {
		t.Errorf("expected 2 critical issues, got: %d", len(critical))
	}
}

// TestDefaultSupervisorConfig 测试默认配置
func TestDefaultSupervisorConfig(t *testing.T) {
	config := DefaultSupervisorConfig()

	if !config.EnableQualityCheck {
		t.Error("quality check should be enabled by default")
	}
	if !config.EnableProgressCheck {
		t.Error("progress check should be enabled by default")
	}
	if !config.EnableErrorDetection {
		t.Error("error detection should be enabled by default")
	}
	if config.EnableSecurityCheck {
		t.Error("security check should be disabled by default")
	}
	if config.MonitorInterval <= 0 {
		t.Error("monitor interval should be positive")
	}
	if config.MaxWarnings <= 0 {
		t.Error("max warnings should be positive")
	}
}

// TestSupervisionIssueTypes 测试问题类型
func TestSupervisionIssueTypes(t *testing.T) {
	types := []IssueType{
		IssueTypeQuality,
		IssueTypeProgress,
		IssueTypeError,
		IssueTypeSecurity,
		IssueTypeBehavior,
		IssueTypeResource,
	}

	for _, issueType := range types {
		if string(issueType) == "" {
			t.Errorf("issue type %s should not be empty", issueType)
		}
	}
}

// TestInterventionTypes 测试干预类型
func TestInterventionTypes(t *testing.T) {
	types := []InterventionType{
		InterventionWarning,
		InterventionPause,
		InterventionInterrupt,
		InterventionRollback,
	}

	for _, interventionType := range types {
		if string(interventionType) == "" {
			t.Errorf("intervention type %s should not be empty", interventionType)
		}
	}
}

// TestMonitorContextCancellation 测试监督上下文取消
func TestMonitorContextCancellation(t *testing.T) {
	supervisor := createTestSupervisor(t)
	taskState := createTestTaskState(t, supervisor.stateManager)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	output := &AgentOutput{
		Content:   "测试输出",
		Timestamp: time.Now(),
		TaskID:    "test-task-1",
		AgentRole: "developer",
		Iteration: 1,
		Success:   true,
	}

	// 应该能够处理已取消的上下文
	result := supervisor.Monitor(ctx, output, taskState)
	if result == nil {
		t.Error("should return result even with cancelled context")
	}
}

// TestParallelMonitorContextCancellation 测试并行监督上下文取消
func TestParallelMonitorContextCancellation(t *testing.T) {
	supervisor := createTestSupervisor(t)
	_ = createTestTaskState(t, supervisor.stateManager)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	outputs := make(chan *AgentOutput, 1)
	outputs <- &AgentOutput{
		Content:   "测试输出",
		Timestamp: time.Now(),
		TaskID:    "test-task-1",
		AgentRole: "developer",
		Iteration: 1,
		Success:   true,
	}
	close(outputs)

	// 应该能够处理已取消的上下文
	resultChan := supervisor.ParallelMonitor(ctx, outputs)

	// 应该能够正常关闭
	for range resultChan {
	}
}
