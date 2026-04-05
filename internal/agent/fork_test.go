package agent

import (
	"context"
	"testing"
	"time"

	"agentplus/internal/model"
	"agentplus/internal/state"
	"agentplus/internal/tools"
)

// TestNewForkManager 测试创建Fork管理器
func TestNewForkManager(t *testing.T) {
	// 准备测试依赖
	modelClient := &model.Client{}
	toolRegistry := tools.NewToolRegistry()
	stateManager, _ := state.NewStateManager(t.TempDir(), false)

	tests := []struct {
		name    string
		config  *ForkConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "fork config cannot be nil",
		},
		{
			name: "nil model client",
			config: &ForkConfig{
				ModelClient:  nil,
				ToolRegistry: toolRegistry,
				StateManager: stateManager,
			},
			wantErr: true,
			errMsg:  "model client is required",
		},
		{
			name: "nil tool registry",
			config: &ForkConfig{
				ModelClient:  modelClient,
				ToolRegistry: nil,
				StateManager: stateManager,
			},
			wantErr: true,
			errMsg:  "tool registry is required",
		},
		{
			name: "nil state manager",
			config: &ForkConfig{
				ModelClient:  modelClient,
				ToolRegistry: toolRegistry,
				StateManager: nil,
			},
			wantErr: true,
			errMsg:  "state manager is required",
		},
		{
			name: "valid config",
			config: &ForkConfig{
				ModelClient:   modelClient,
				ToolRegistry:  toolRegistry,
				StateManager:  stateManager,
				MaxIterations: 20,
			},
			wantErr: false,
		},
		{
			name: "default max iterations",
			config: &ForkConfig{
				ModelClient:   modelClient,
				ToolRegistry:  toolRegistry,
				StateManager:  stateManager,
				MaxIterations: 0, // 应该使用默认值30
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, err := NewForkManager(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewForkManager() expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("NewForkManager() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("NewForkManager() unexpected error: %v", err)
				return
			}
			if fm == nil {
				t.Error("NewForkManager() returned nil")
				return
			}
			// 检查默认值
			if tt.config.MaxIterations <= 0 && fm.maxIterations != 30 {
				t.Errorf("NewForkManager() maxIterations = %v, want 30", fm.maxIterations)
			}
		})
	}
}

// TestGetPromptTypeByRole 测试角色到提示词类型的映射
func TestGetPromptTypeByRole(t *testing.T) {
	tests := []struct {
		role     string
		expected SystemPromptType
	}{
		{"Worker", PromptTypeWorker},
		{"worker", PromptTypeWorker},
		{"Reviewer", PromptTypeReviewer},
		{"reviewer", PromptTypeReviewer},
		{"Orchestrator", PromptTypeOrchestrator},
		{"Unknown", PromptTypeOrchestrator},
		{"", PromptTypeOrchestrator},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			result := getPromptTypeByRole(tt.role)
			if result != tt.expected {
				t.Errorf("getPromptTypeByRole(%s) = %v, want %v", tt.role, result, tt.expected)
			}
		})
	}
}

// TestBuildForkTaskPrompt 测试构建子代理任务提示
func TestBuildForkTaskPrompt(t *testing.T) {
	role := "Worker"
	task := "实现用户登录功能"
	stateSummary := "当前进度: 50%"

	prompt := BuildForkTaskPrompt(role, task, stateSummary)

	// 检查关键内容
	if !contains(prompt, "Worker") {
		t.Error("BuildForkTaskPrompt missing role")
	}
	if !contains(prompt, "实现用户登录功能") {
		t.Error("BuildForkTaskPrompt missing task")
	}
	if !contains(prompt, "当前进度") {
		t.Error("BuildForkTaskPrompt missing state summary")
	}
	if !contains(prompt, "complete_as_agent") {
		t.Error("BuildForkTaskPrompt missing complete_as_agent instruction")
	}
}

// TestBuildForkTaskPromptWithoutState 测试无状态摘要的任务提示
func TestBuildForkTaskPromptWithoutState(t *testing.T) {
	role := "Reviewer"
	task := "审查代码质量"

	prompt := BuildForkTaskPrompt(role, task, "")

	if !contains(prompt, "Reviewer") {
		t.Error("BuildForkTaskPrompt missing role")
	}
	if !contains(prompt, "审查代码质量") {
		t.Error("BuildForkTaskPrompt missing task")
	}
	// 空状态摘要不应该出现"当前任务状态"
	if contains(prompt, "当前任务状态") {
		t.Error("BuildForkTaskPrompt should not include state section when empty")
	}
}

// TestBuildJoinPrompt 测试构建合并提示
func TestBuildJoinPrompt(t *testing.T) {
	role := "Worker"
	task := "实现用户登录功能"
	summary := "已完成登录API开发，包含JWT认证"

	prompt := BuildJoinPrompt(role, task, summary)

	// 检查关键内容
	if !contains(prompt, "Worker") {
		t.Error("BuildJoinPrompt missing role")
	}
	if !contains(prompt, "实现用户登录功能") {
		t.Error("BuildJoinPrompt missing task")
	}
	if !contains(prompt, "已完成登录API开发") {
		t.Error("BuildJoinPrompt missing summary")
	}
	if !contains(prompt, "继续主任务") {
		t.Error("BuildJoinPrompt missing continue instruction")
	}
}

// TestSpawnAgentTool 测试创建子代理工具
func TestSpawnAgentTool(t *testing.T) {
	tool := NewSpawnAgentTool(nil, nil)

	// 测试工具名称
	if tool.Name() != "spawn_agent" {
		t.Errorf("SpawnAgentTool.Name() = %v, want spawn_agent", tool.Name())
	}

	// 测试工具描述
	desc := tool.Description()
	if desc == "" {
		t.Error("SpawnAgentTool.Description() returned empty string")
	}

	// 测试参数定义
	params := tool.Parameters()
	if params == nil {
		t.Error("SpawnAgentTool.Parameters() returned nil")
		return
	}

	// 检查必需参数
	required, ok := params["required"].([]string)
	if !ok {
		t.Error("SpawnAgentTool.Parameters() missing required fields")
		return
	}
	if len(required) != 2 {
		t.Errorf("SpawnAgentTool.Parameters() has %d required fields, want 2", len(required))
	}
}

// TestCompleteAsAgentTool 测试子代理完成任务工具
func TestCompleteAsAgentTool(t *testing.T) {
	tool := NewCompleteAsAgentTool()

	// 测试工具名称
	if tool.Name() != "complete_as_agent" {
		t.Errorf("CompleteAsAgentTool.Name() = %v, want complete_as_agent", tool.Name())
	}

	// 测试工具描述
	desc := tool.Description()
	if desc == "" {
		t.Error("CompleteAsAgentTool.Description() returned empty string")
	}

	// 测试参数定义
	params := tool.Parameters()
	if params == nil {
		t.Error("CompleteAsAgentTool.Parameters() returned nil")
		return
	}

	// 检查必需参数
	required, ok := params["required"].([]string)
	if !ok {
		t.Error("CompleteAsAgentTool.Parameters() missing required fields")
		return
	}
	if len(required) != 1 || required[0] != "summary" {
		t.Errorf("CompleteAsAgentTool.Parameters() required = %v, want [summary]", required)
	}
}

// TestCompleteAsAgentToolExecute 测试CompleteAsAgent工具执行
func TestCompleteAsAgentToolExecute(t *testing.T) {
	tool := NewCompleteAsAgentTool()
	ctx := context.Background()

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name:    "missing summary",
			params:  map[string]interface{}{},
			wantErr: false, // 工具执行不返回错误，返回错误结果
		},
		{
			name: "empty summary",
			params: map[string]interface{}{
				"summary": "",
			},
			wantErr: false,
		},
		{
			name: "valid summary",
			params: map[string]interface{}{
				"summary": "任务完成，已实现所有功能",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, tt.params)
			if err != nil {
				t.Errorf("CompleteAsAgentTool.Execute() error = %v", err)
				return
			}
			if result == nil {
				t.Error("CompleteAsAgentTool.Execute() returned nil result")
				return
			}

			// 检查结果
			summary, _ := tt.params["summary"].(string)
			if summary != "" && !result.Success {
				t.Error("CompleteAsAgentTool.Execute() should succeed with valid summary")
			}
			if summary == "" && result.Success {
				t.Error("CompleteAsAgentTool.Execute() should fail with empty summary")
			}
		})
	}
}

// TestForkManagerCallbacks 测试Fork管理器回调
func TestForkManagerCallbacks(t *testing.T) {
	// 准备测试依赖
	modelClient := &model.Client{}
	toolRegistry := tools.NewToolRegistry()
	stateManager, _ := state.NewStateManager(t.TempDir(), false)

	fm, err := NewForkManager(&ForkConfig{
		ModelClient:   modelClient,
		ToolRegistry:  toolRegistry,
		StateManager:  stateManager,
		MaxIterations: 10,
	})
	if err != nil {
		t.Fatalf("NewForkManager() error = %v", err)
	}

	// 测试设置回调
	var forkStartCalled bool
	var forkEndCalled bool
	var streamCalled bool

	fm.SetOnForkStart(func(role, task string) {
		forkStartCalled = true
	})
	fm.SetOnForkEnd(func(role, summary string) {
		forkEndCalled = true
	})
	fm.SetOnStreamChunk(func(chunk string) {
		streamCalled = true
	})

	// 验证回调已设置
	if fm.onForkStart == nil {
		t.Error("SetOnForkStart did not set callback")
	}
	if fm.onForkEnd == nil {
		t.Error("SetOnForkEnd did not set callback")
	}
	if fm.onStreamChunk == nil {
		t.Error("SetOnStreamChunk did not set callback")
	}

	// 调用回调验证
	fm.onForkStart("Worker", "test task")
	if !forkStartCalled {
		t.Error("onForkStart callback was not called")
	}

	fm.onForkEnd("Worker", "test summary")
	if !forkEndCalled {
		t.Error("onForkEnd callback was not called")
	}

	fm.onStreamChunk("test chunk")
	if !streamCalled {
		t.Error("onStreamChunk callback was not called")
	}
}

// TestForkManagerGetActiveForks 测试获取活动子代理
func TestForkManagerGetActiveForks(t *testing.T) {
	// 准备测试依赖
	modelClient := &model.Client{}
	toolRegistry := tools.NewToolRegistry()
	stateManager, _ := state.NewStateManager(t.TempDir(), false)

	fm, err := NewForkManager(&ForkConfig{
		ModelClient:   modelClient,
		ToolRegistry:  toolRegistry,
		StateManager:  stateManager,
		MaxIterations: 10,
	})
	if err != nil {
		t.Fatalf("NewForkManager() error = %v", err)
	}

	// 初始应该为空
	forks := fm.GetActiveForks()
	if len(forks) != 0 {
		t.Errorf("GetActiveForks() = %v, want empty", forks)
	}

	// 手动添加一个活动的fork用于测试
	fm.mu.Lock()
	fm.activeForks["test-fork-1"] = &ForkedAgent{
		ID:     "test-fork-1",
		Role:   "Worker",
		Task:   "test task",
		Status: "running",
	}
	fm.mu.Unlock()

	// 应该有一个活动fork
	forks = fm.GetActiveForks()
	if len(forks) != 1 {
		t.Errorf("GetActiveForks() = %v, want 1 fork", forks)
	}
	if forks[0].ID != "test-fork-1" {
		t.Errorf("GetActiveForks()[0].ID = %v, want test-fork-1", forks[0].ID)
	}
}

// TestRegisterForkTools 测试注册Fork工具
func TestRegisterForkTools(t *testing.T) {
	registry := tools.NewToolRegistry()

	// 准备测试依赖
	modelClient := &model.Client{}
	stateManager, _ := state.NewStateManager(t.TempDir(), false)

	fm, err := NewForkManager(&ForkConfig{
		ModelClient:   modelClient,
		ToolRegistry:  registry,
		StateManager:  stateManager,
		MaxIterations: 10,
	})
	if err != nil {
		t.Fatalf("NewForkManager() error = %v", err)
	}

	// 创建一个简单的Agent用于测试
	parentAgent := &Agent{
		taskID: "test-task",
	}

	// 注册工具
	err = RegisterForkTools(registry, fm, parentAgent)
	if err != nil {
		t.Errorf("RegisterForkTools() error = %v", err)
		return
	}

	// 验证工具已注册
	_, exists := registry.GetTool("spawn_agent")
	if !exists {
		t.Error("spawn_agent tool not registered")
	}

	_, exists = registry.GetTool("complete_as_agent")
	if !exists {
		t.Error("complete_as_agent tool not registered")
	}
}

// TestForkResult 测试ForkResult结构
func TestForkResult(t *testing.T) {
	result := &ForkResult{
		ForkID:     "fork-123",
		Role:       "Worker",
		Task:       "实现登录功能",
		Summary:    "已完成",
		Status:     "completed",
		Duration:   time.Minute * 5,
		Iterations: 10,
	}

	// 验证字段
	if result.ForkID != "fork-123" {
		t.Errorf("ForkResult.ForkID = %v, want fork-123", result.ForkID)
	}
	if result.Role != "Worker" {
		t.Errorf("ForkResult.Role = %v, want Worker", result.Role)
	}
	if result.Status != "completed" {
		t.Errorf("ForkResult.Status = %v, want completed", result.Status)
	}
}

// TestForkedAgent 测试ForkedAgent结构
func TestForkedAgent(t *testing.T) {
	now := time.Now()
	agent := &ForkedAgent{
		ID:           "fork-456",
		Role:         "Reviewer",
		Task:         "审查代码",
		ParentTaskID: "parent-123",
		StartTime:    now,
		EndTime:      now.Add(time.Minute),
		Summary:      "审查完成",
		Status:       "completed",
	}

	// 验证字段
	if agent.ID != "fork-456" {
		t.Errorf("ForkedAgent.ID = %v, want fork-456", agent.ID)
	}
	if agent.ParentTaskID != "parent-123" {
		t.Errorf("ForkedAgent.ParentTaskID = %v, want parent-123", agent.ParentTaskID)
	}
	if agent.Status != "completed" {
		t.Errorf("ForkedAgent.Status = %v, want completed", agent.Status)
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
