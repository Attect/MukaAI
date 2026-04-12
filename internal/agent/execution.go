// Package agent 工具执行调度逻辑
// 从 core.go 提取的工具执行和特殊工具处理方法
package agent

import (
	"context"
	"time"

	"agentplus/internal/model"
)

// executeTools 执行工具调用
func (a *Agent) executeTools(ctx context.Context, toolCalls []model.ToolCall) ([]model.Message, error) {
	results := make([]model.Message, 0, len(toolCalls))

	// 获取流式处理器（线程安全）
	a.mu.RLock()
	handler := a.streamHandler
	a.mu.RUnlock()

	for _, tc := range toolCalls {
		// 记录工具调用
		if a.logger != nil {
			a.logger.LogToolCall(tc.Function.Name, tc.Function.Arguments)
		}

		// 工具调用完整回调
		if a.onToolCallFull != nil {
			a.onToolCallFull(tc.ID, tc.Function.Name, tc.Function.Arguments)
		}

		// 工具调用回调
		if a.onToolCall != nil {
			a.onToolCall(tc.Function.Name, tc.Function.Arguments)
		}

		// 执行工具
		result, err := a.executor.ExecuteToolCalls(ctx, []model.ToolCall{tc})
		if err != nil {
			// 记录工具执行失败
			if a.logger != nil {
				a.logger.LogToolResult(tc.Function.Name, err.Error(), false)
			}

			// 调用工具结果回调（失败）
			if handler != nil {
				handler.OnToolResult(ConvertToolCallWithResult(tc, "", err.Error()))
			}

			return nil, err
		}

		// 记录工具执行成功
		if a.logger != nil && len(result) > 0 {
			resultContent := ""
			if len(result) > 0 {
				resultContent = result[0].Content
				if len(resultContent) > 500 {
					resultContent = resultContent[:500] + "..."
				}
			}
			a.logger.LogToolResult(tc.Function.Name, resultContent, true)
		}

		// 调用工具结果回调（成功）
		if handler != nil && len(result) > 0 {
			resultContent := ""
			if len(result) > 0 {
				resultContent = result[0].Content
			}
			handler.OnToolResult(ConvertToolCallWithResult(tc, resultContent, ""))
		}

		// 工具结果回调（新回调）
		if a.onToolResult != nil && len(result) > 0 {
			a.onToolResult(tc.Function.Name, result[0].Content)
		}

		results = append(results, result...)

		// 处理特殊工具
		if err := a.handleSpecialTools(ctx, tc); err != nil {
			return nil, err
		}
	}

	return results, nil
}

// handleSpecialTools 处理特殊工具（如状态更新工具）
func (a *Agent) handleSpecialTools(ctx context.Context, tc model.ToolCall) error {
	switch tc.Function.Name {
	case "update_state":
		// 解析参数并更新状态
		args, err := ParseToolCallArguments(tc)
		if err != nil {
			return err
		}

		if phase, ok := args["phase"].(string); ok {
			a.stateManager.UpdateProgress(a.taskID, phase, "")
		}
		if decision, ok := args["decision"].(string); ok {
			a.stateManager.AddDecision(a.taskID, decision)
		}
		if step, ok := args["completed_step"].(string); ok {
			a.stateManager.CompleteStep(a.taskID, step)
		}

	case "add_file":
		args, err := ParseToolCallArguments(tc)
		if err != nil {
			return err
		}

		path, _ := args["path"].(string)
		description, _ := args["description"].(string)
		status, _ := args["status"].(string)
		if path != "" {
			a.stateManager.AddFile(a.taskID, path, description, status)
		}
	}

	return nil
}

// failTask 将任务标记为失败并返回结果
// 用于Run()主循环中处理fail_task工具调用
func (a *Agent) failTask(result *RunResult, totalIterations int) (*RunResult, error) {
	result.Status = "failed"
	result.EndTime = time.Now()
	result.Iterations = totalIterations
	a.stateManager.UpdateTaskStatus(a.taskID, "failed")
	if a.logger != nil {
		a.logger.LogTaskEnd("failed", result.Iterations, result.EndTime.Sub(result.StartTime))
	}
	a.finalizeResult(result)
	return result, nil
}
