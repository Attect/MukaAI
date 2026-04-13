// Package agent Supervisor集成接口
// 定义Supervisor与Agent主循环交互所需的接口和数据类型
// 通过接口解耦，supervisor包依赖agent包（已有依赖方向），避免循环导入
package agent

import (
	"context"

	"github.com/Attect/MukaAI/internal/model"
	"github.com/Attect/MukaAI/internal/state"
)

// SupervisionIssue 监督发现的问题
type SupervisionIssue struct {
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Evidence    string                 `json:"evidence"`
	Suggestion  string                 `json:"suggestion"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// SupervisionResult 监督检查结果
// 包含监督检查的完整结果，用于Agent主循环决策
type SupervisionResult struct {
	// Status 监督状态：pass, warning, intervention
	Status string `json:"status"`
	// Issues 发现的问题列表
	Issues []SupervisionIssue `json:"issues"`
	// Summary 监督摘要
	Summary string `json:"summary"`
	// InterventionType 干预类型：warning, pause, interrupt, rollback
	InterventionType string `json:"intervention_type,omitempty"`
	// InterventionAction 干预动作描述
	InterventionAction string `json:"intervention_action,omitempty"`
}

// IsInterventionNeeded 检查是否需要干预
// 当干预类型为pause/interrupt/rollback时返回true
func (r *SupervisionResult) IsInterventionNeeded() bool {
	return r.InterventionType != "" && r.InterventionType != "warning"
}

// HasCriticalIssues 是否存在严重问题
func (r *SupervisionResult) HasCriticalIssues() bool {
	for _, issue := range r.Issues {
		if issue.Severity == "critical" {
			return true
		}
	}
	return false
}

// AgentOutput Agent输出信息，供Supervisor检查
type AgentOutput struct {
	// Content 输出内容
	Content string
	// ToolCalls 工具调用列表
	ToolCalls []model.ToolCall
	// TaskID 任务ID
	TaskID string
	// Iteration 当前迭代次数
	Iteration int
	// Success 是否成功
	Success bool
	// Error 错误信息
	Error string
}

// Supervisor Agent监督接口
// 由supervisor包中的Supervisor结构体通过适配器实现
// Agent主循环通过此接口调用监督检查，无需直接导入supervisor包
type Supervisor interface {
	// Check 执行监督检查
	// ctx: 上下文
	// output: Agent的输出信息
	// taskState: 当前任务状态
	// 返回监督检查结果
	Check(ctx context.Context, output *AgentOutput, taskState *state.TaskState) *SupervisionResult
}
