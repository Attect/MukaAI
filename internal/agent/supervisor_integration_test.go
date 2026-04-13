// Package agent Supervisor集成测试
// 测试Supervisor与Agent主循环的集成行为
package agent

import (
	"context"
	"sync"
	"testing"

	"github.com/Attect/MukaAI/internal/model"
	"github.com/Attect/MukaAI/internal/state"
	"github.com/Attect/MukaAI/internal/tools"
)

// mockSupervisor 模拟Supervisor，用于集成测试
type mockSupervisor struct {
	mu             sync.Mutex
	checkCount     int
	resultToReturn *SupervisionResult
	lastOutput     *AgentOutput
}

func (m *mockSupervisor) Check(ctx context.Context, output *AgentOutput, taskState *state.TaskState) *SupervisionResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkCount++
	m.lastOutput = output
	if m.resultToReturn != nil {
		return m.resultToReturn
	}
	return &SupervisionResult{Status: "pass", Summary: "mock pass"}
}

func (m *mockSupervisor) getCheckCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.checkCount
}

// TestSupervisorNilDoesNotPanic 测试Supervisor为nil时不会panic
func TestSupervisorNilDoesNotPanic(t *testing.T) {
	ag := createTestAgent(t)
	// 不设置Supervisor，默认为nil

	response := &modelResponse{
		Content:   "test content",
		ToolCalls: nil,
	}

	// 应该不会panic
	result := ag.runSupervision(context.Background(), response, nil, "test goal", 1)
	if result != nil {
		t.Error("expected nil result when supervisor is nil")
	}
}

// TestSupervisorPassStatus 测试监督检查通过时正常继续
func TestSupervisorPassStatus(t *testing.T) {
	ag := createTestAgent(t)
	mock := &mockSupervisor{
		resultToReturn: &SupervisionResult{
			Status:  "pass",
			Summary: "all checks passed",
		},
	}
	ag.SetSupervisor(mock)

	response := &modelResponse{
		Content:   "normal output",
		ToolCalls: nil,
	}

	result := ag.runSupervision(context.Background(), response, nil, "test goal", 1)
	if result != nil {
		t.Errorf("expected nil result for pass status, got action=%s", result.action)
	}
	if mock.getCheckCount() != 1 {
		t.Errorf("expected 1 check, got %d", mock.getCheckCount())
	}
}

// TestSupervisorHaltIntervention 测试interrupt级别干预终止任务
func TestSupervisorHaltIntervention(t *testing.T) {
	ag := createTestAgent(t)
	mock := &mockSupervisor{
		resultToReturn: &SupervisionResult{
			Status:             "intervention",
			InterventionType:   "interrupt",
			InterventionAction: "halting task",
			Summary:            "critical issue detected",
			Issues: []SupervisionIssue{
				{Type: "security", Severity: "critical", Description: "dangerous command"},
			},
		},
	}
	ag.SetSupervisor(mock)

	response := &modelResponse{
		Content: "rm -rf /",
	}

	result := ag.runSupervision(context.Background(), response, nil, "test goal", 1)
	if result == nil {
		t.Fatal("expected non-nil result for halt intervention")
	}
	if result.action != "return" {
		t.Errorf("expected action=return, got %s", result.action)
	}
	if result.result.Status != "failed" {
		t.Errorf("expected status=failed, got %s", result.result.Status)
	}
}

// TestSupervisorWarningIntervention 测试warning级别干预仅记录
func TestSupervisorWarningIntervention(t *testing.T) {
	ag := createTestAgent(t)
	mock := &mockSupervisor{
		resultToReturn: &SupervisionResult{
			Status:             "warning",
			InterventionType:   "warning",
			InterventionAction: "issuing warning",
			Summary:            "quality issue detected",
			Issues: []SupervisionIssue{
				{Type: "quality", Severity: "medium", Description: "output too short"},
			},
		},
	}
	ag.SetSupervisor(mock)

	response := &modelResponse{
		Content: "short",
	}

	result := ag.runSupervision(context.Background(), response, nil, "test goal", 1)
	// Warning should not stop execution
	if result != nil {
		t.Errorf("expected nil result for warning intervention, got action=%s", result.action)
	}
}

// TestSupervisorCorrectionIntervention 测试pause级别干预注入修正指令
func TestSupervisorCorrectionIntervention(t *testing.T) {
	ag := createTestAgent(t)
	mock := &mockSupervisor{
		resultToReturn: &SupervisionResult{
			Status:             "warning",
			InterventionType:   "pause",
			InterventionAction: "injecting correction",
			Summary:            "progress stalled",
			Issues: []SupervisionIssue{
				{Type: "progress", Severity: "high", Description: "no progress for 5 minutes"},
			},
		},
	}
	ag.SetSupervisor(mock)

	response := &modelResponse{
		Content: "stalled output",
	}

	// 捕获修正指令
	var capturedCorrection string
	ag.SetOnCorrection(func(instruction string) {
		capturedCorrection = instruction
	})

	result := ag.runSupervision(context.Background(), response, nil, "test goal", 1)
	// Correction should not stop execution
	if result != nil {
		t.Errorf("expected nil result for correction intervention, got action=%s", result.action)
	}
	if capturedCorrection == "" {
		t.Error("expected correction instruction to be injected")
	}
}

// TestSupervisorCallback 测试监督结果回调
func TestSupervisorCallback(t *testing.T) {
	ag := createTestAgent(t)
	mock := &mockSupervisor{
		resultToReturn: &SupervisionResult{
			Status:  "warning",
			Summary: "test warning",
		},
	}
	ag.SetSupervisor(mock)

	var capturedResult *SupervisionResult
	ag.SetOnSupervisor(func(result *SupervisionResult) {
		capturedResult = result
	})

	response := &modelResponse{Content: "test"}
	ag.runSupervision(context.Background(), response, nil, "test goal", 1)

	if capturedResult == nil {
		t.Fatal("expected supervisor callback to be called")
	}
	if capturedResult.Summary != "test warning" {
		t.Errorf("expected summary='test warning', got '%s'", capturedResult.Summary)
	}
}

// TestSupervisorRollbackIntervention 测试rollback级别干预终止任务
func TestSupervisorRollbackIntervention(t *testing.T) {
	ag := createTestAgent(t)
	mock := &mockSupervisor{
		resultToReturn: &SupervisionResult{
			Status:             "intervention",
			InterventionType:   "rollback",
			InterventionAction: "rolling back",
			Summary:            "critical error detected",
		},
	}
	ag.SetSupervisor(mock)

	response := &modelResponse{Content: "error state"}
	result := ag.runSupervision(context.Background(), response, nil, "test goal", 1)

	if result == nil {
		t.Fatal("expected non-nil result for rollback intervention")
	}
	if result.action != "return" {
		t.Errorf("expected action=return, got %s", result.action)
	}
}

// TestBuildAgentOutput 测试AgentOutput构建
func TestBuildAgentOutput(t *testing.T) {
	ag := createTestAgent(t)

	response := &modelResponse{
		Content: "test content",
		ToolCalls: []model.ToolCall{
			{ID: "tc1", Function: model.FunctionCall{Name: "read_file", Arguments: "{}"}},
		},
	}

	output := ag.buildAgentOutput(response, nil, 5)

	if output.Content != "test content" {
		t.Errorf("expected content='test content', got '%s'", output.Content)
	}
	if len(output.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(output.ToolCalls))
	}
	if output.Iteration != 5 {
		t.Errorf("expected iteration=5, got %d", output.Iteration)
	}
	if !output.Success {
		t.Error("expected success=true")
	}
}

// TestSupervisorResultIsInterventionNeeded 测试SupervisionResult.IsInterventionNeeded
func TestSupervisorResultIsInterventionNeeded(t *testing.T) {
	tests := []struct {
		name             string
		interventionType string
		expected         bool
	}{
		{"empty", "", false},
		{"warning", "warning", false},
		{"pause", "pause", true},
		{"interrupt", "interrupt", true},
		{"rollback", "rollback", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &SupervisionResult{InterventionType: tt.interventionType}
			if r.IsInterventionNeeded() != tt.expected {
				t.Errorf("IsInterventionNeeded() for %q = %v, want %v", tt.interventionType, !tt.expected, tt.expected)
			}
		})
	}
}

// TestSupervisorResultHasCriticalIssues 测试HasCriticalIssues
func TestSupervisorResultHasCriticalIssues(t *testing.T) {
	r := &SupervisionResult{
		Issues: []SupervisionIssue{
			{Severity: "low"},
			{Severity: "medium"},
		},
	}
	if r.HasCriticalIssues() {
		t.Error("expected no critical issues")
	}

	r.Issues = append(r.Issues, SupervisionIssue{Severity: "critical"})
	if !r.HasCriticalIssues() {
		t.Error("expected critical issues")
	}
}

// createTestAgent 创建用于测试的Agent实例
func createTestAgent(t *testing.T) *Agent {
	t.Helper()

	modelClient, err := model.NewClient(&model.Config{
		Endpoint:    "http://localhost:11434",
		APIKey:      "test",
		ModelName:   "test",
		ContextSize: 4096,
	})
	if err != nil {
		t.Fatalf("failed to create model client: %v", err)
	}

	registry := tools.NewToolRegistry()
	stateMgr, err := state.NewStateManager(t.TempDir(), false)
	if err != nil {
		t.Fatalf("failed to create state manager: %v", err)
	}

	ag, err := NewAgent(&Config{
		ModelClient:   modelClient,
		ToolRegistry:  registry,
		StateManager:  stateMgr,
		MaxIterations: 10,
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	return ag
}
