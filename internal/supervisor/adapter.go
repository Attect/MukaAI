// Package supervisor Agent集成适配器
// 提供Supervisor结构体到agent.Supervisor接口的适配实现
// 将supervisor内部类型转换为agent包定义的接口类型
package supervisor

import (
	"context"
	"time"

	"github.com/Attect/MukaAI/internal/agent"
	"github.com/Attect/MukaAI/internal/state"
)

// Check 执行监督检查，实现agent.Supervisor接口
// 将agent.AgentOutput转换为supervisor内部的AgentOutput格式，
// 调用内部Monitor方法，再将结果转换回agent.SupervisionResult
func (s *Supervisor) Check(
	ctx context.Context,
	output *agent.AgentOutput,
	taskState *state.TaskState,
) *agent.SupervisionResult {
	// 转换为supervisor内部的AgentOutput
	internalOutput := &AgentOutput{
		Content:   output.Content,
		ToolCalls: output.ToolCalls,
		TaskID:    output.TaskID,
		Iteration: output.Iteration,
		Success:   output.Success,
		Error:     output.Error,
		Timestamp: time.Now(),
	}

	// 调用内部Monitor方法
	internalResult := s.Monitor(ctx, internalOutput, taskState)

	// 转换结果
	result := &agent.SupervisionResult{
		Status:  internalResult.Status,
		Summary: internalResult.Summary,
	}

	// 转换Issues
	for _, issue := range internalResult.Issues {
		result.Issues = append(result.Issues, agent.SupervisionIssue{
			Type:        string(issue.Type),
			Severity:    issue.Severity,
			Description: issue.Description,
			Evidence:    issue.Evidence,
			Suggestion:  issue.Suggestion,
			Context:     issue.Context,
		})
	}

	// 转换Intervention信息
	if internalResult.Intervention != nil {
		result.InterventionType = string(internalResult.Intervention.Type)
		result.InterventionAction = internalResult.Intervention.Action
	}

	return result
}
