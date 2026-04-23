// Package agent Run()主循环子方法
// 从 core.go 的 Run() 方法中提取的子方法，降低主循环复杂度
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Attect/MukaAI/internal/model"
)

// iterationResult 单次迭代处理结果
// 用于从子方法向Run()主循环传递处理状态
type iterationResult struct {
	// 操作类型
	action string // "continue" | "break" | "return"

	// 返回值（仅当action=="return"时有效）
	result *RunResult
	err    error
}

// handleMaxIterations 处理达到最大迭代次数的情况
func (a *Agent) handleMaxIterations(result *RunResult, totalIterations, maxTotalIterations int) (*RunResult, error) {
	result.Status = "max_iterations"
	result.EndTime = time.Now()
	result.Iterations = totalIterations - 1
	a.stateManager.UpdateTaskStatus(a.taskID, "failed")
	if a.logger != nil {
		a.logger.LogError(fmt.Sprintf("达到最大迭代次数 (%d)", maxTotalIterations))
		a.logger.LogTaskEnd("max_iterations", result.Iterations, result.EndTime.Sub(result.StartTime))
	}
	a.finalizeResult(result)
	return result, fmt.Errorf("reached maximum iterations (%d)", maxTotalIterations)
}

// handleReviewBlock 处理审查阻断的情况
// 返回 nil 表示已处理（应continue），非nil表示应直接返回
func (a *Agent) handleReviewBlock(reviewResult *ReviewResult, result *RunResult, totalIterations int) (*RunResult, error) {
	// 分析失败并生成修正指令
	correctionResult := a.corrector.AnalyzeFailure(nil, reviewResult)
	correctionInstruction := a.corrector.GenerateCorrectionInstruction(correctionResult)

	// 记录修正指令
	if a.logger != nil {
		a.logger.LogCorrection(correctionInstruction, reviewResult.Summary)
	}

	// 检查是否还有审查重试机会
	if !a.corrector.ShouldRetryReview() {
		// 重试次数耗尽，任务失败
		result.Status = "failed"
		result.Error = fmt.Sprintf("审查阻断且重试次数耗尽: %s", reviewResult.Summary)
		result.EndTime = time.Now()
		a.stateManager.UpdateTaskStatus(a.taskID, "failed")
		if a.logger != nil {
			a.logger.LogError(result.Error)
			a.logger.LogTaskEnd("failed", totalIterations, result.EndTime.Sub(result.StartTime))
		}
		a.finalizeResult(result)
		return result, fmt.Errorf("review blocked and retries exhausted: %s", reviewResult.Summary)
	}

	// 注入修正指令到历史
	if a.onCorrection != nil {
		a.onCorrection(correctionInstruction)
	}
	if a.onHistoryAdd != nil {
		a.onHistoryAdd("user", correctionInstruction)
	}
	a.history.AddMessage(model.NewUserMessage(correctionInstruction))
	result.Iterations = totalIterations
	return nil, nil // nil表示已处理，主循环应continue
}

// verifyAndCorrect 执行校验并处理失败情况
// 返回 (passed bool, result *RunResult, err error)
// passed=true: 校验通过
// result非nil: 应直接返回（失败）
func (a *Agent) verifyAndCorrect(runCtx context.Context, taskGoal string, result *RunResult, totalIterations int, errorPrefix string) (bool, *RunResult, error) {
	verifyResult := a.verifyTaskCompletion(runCtx, taskGoal)

	// 记录校验结果
	if a.logger != nil {
		a.logger.LogVerification(verifyResult)
	}

	// 校验结果回调
	if a.onVerify != nil {
		a.onVerify(string(verifyResult.Status), verifyResult.Summary)
	}

	if verifyResult != nil && verifyResult.IsFailed() {
		// 校验失败，记录失败并生成修正指令
		correctionResult := a.corrector.AnalyzeFailure(verifyResult, nil)
		correctionInstruction := a.corrector.GenerateCorrectionInstruction(correctionResult)

		// 记录修正指令
		if a.logger != nil {
			a.logger.LogCorrection(correctionInstruction, verifyResult.Summary)
		}

		// 检查是否还有校验重试机会
		if !a.corrector.ShouldRetryVerify() {
			// 重试次数耗尽，任务失败
			result.Status = "failed"
			result.Error = fmt.Sprintf("%s: %s", errorPrefix, verifyResult.Summary)
			result.EndTime = time.Now()
			a.stateManager.UpdateTaskStatus(a.taskID, "failed")
			if a.logger != nil {
				a.logger.LogError(result.Error)
				a.logger.LogTaskEnd("failed", totalIterations, result.EndTime.Sub(result.StartTime))
			}
			a.finalizeResult(result)
			return false, result, fmt.Errorf("%s: %s", errorPrefix, verifyResult.Summary)
		}

		// 注入修正指令到历史，继续执行
		if a.onCorrection != nil {
			a.onCorrection(correctionInstruction)
		}
		if a.onHistoryAdd != nil {
			a.onHistoryAdd("user", correctionInstruction)
		}
		a.history.AddMessage(model.NewUserMessage(correctionInstruction))
		result.Iterations = totalIterations
		return false, nil, nil // 校验失败但可重试
	}

	return true, nil, nil // 校验通过
}

// handleToolCallsIteration 处理有工具调用的迭代
// 返回nil表示主循环应continue，非nil表示应返回或break
func (a *Agent) handleToolCallsIteration(runCtx context.Context, response *modelResponse, result *RunResult, taskGoal string, totalIterations int) *iterationResult {
	// 执行工具调用
	toolResults, err := a.executeTools(runCtx, response.ToolCalls)
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		result.EndTime = time.Now()
		a.stateManager.UpdateTaskStatus(a.taskID, "failed")
		if a.logger != nil {
			a.logger.LogError(fmt.Sprintf("工具执行失败: %s", err.Error()))
			a.logger.LogTaskEnd("failed", totalIterations, result.EndTime.Sub(result.StartTime))
		}
		a.finalizeResult(result)
		return &iterationResult{action: "return", result: result, err: fmt.Errorf("tool execution failed: %w", err)}
	}

	// 工具执行后：执行监督检查
	if ir := a.runSupervision(runCtx, response, toolResults, taskGoal, totalIterations); ir != nil {
		return ir
	}

	// 添加工具结果到历史
	a.history.AddMessages(toolResults)

	// 检查是否有end_exploration工具调用
	for _, tc := range response.ToolCalls {
		if tc.Function.Name == "end_exploration" {
			a.reviewer.EndExploration()
			if a.logger != nil {
				a.logger.LogMessage("system", "探索阶段已结束，开始严格监控任务进度")
			}
		}
	}

	// 检查是否有任务完成/失败的工具调用
	for _, tc := range response.ToolCalls {
		if tc.Function.Name == "complete_task" {
			// 在完成任务前进行校验
			passed, ret, err := a.verifyAndCorrect(runCtx, taskGoal, result, totalIterations, "任务完成校验失败且重试次数耗尽")
			if err != nil {
				return &iterationResult{action: "return", result: ret, err: err}
			}
			if !passed {
				// 校验失败但可重试
				return &iterationResult{action: "continue"}
			}

			// 校验通过，标记任务完成（但不立即返回）
			result.Status = "completed"
			result.EndTime = time.Now()
			result.Iterations = totalIterations
			// 不设置verificationPassed，让外层循环执行强制校验
			return &iterationResult{action: "break"}
		}
		if tc.Function.Name == "fail_task" {
			ret, err := a.failTask(result, totalIterations)
			return &iterationResult{action: "return", result: ret, err: err}
		}
	}

	// 普通工具调用完成，继续迭代
	result.Iterations = totalIterations
	return &iterationResult{action: "continue"}
}

// handleNoToolCallIteration 处理无工具调用的迭代
// 返回 nil 表示主循环应 continue，非 nil 表示应 break 或 return
func (a *Agent) handleNoToolCallIteration(runCtx context.Context, response *modelResponse, result *RunResult, taskGoal string, totalIterations int, consecutiveNoToolCalls int) *iterationResult {
	// 无工具调用时也执行监督检查
	if ir := a.runSupervision(runCtx, response, nil, taskGoal, totalIterations); ir != nil {
		return ir
	}

	// 连续无工具调用超过阈值，使用 LLM 代理检查任务是否完成
	if consecutiveNoToolCalls >= 5 {
		complete, finalResponse := a.checkTaskCompletionViaLLM(runCtx, taskGoal, response.Content)
		if complete {
			// LLM 判断任务已完成
			result.Status = "completed"
			result.EndTime = time.Now()
			result.FinalResponse = finalResponse
			result.Iterations = totalIterations
			a.finalizeResult(result)
			return &iterationResult{action: "break"}
		}

		// LLM 判断未完成，继续迭代
		promptMsg := "请根据上述内容继续执行。如果已经完成，请调用 complete_task 工具。"
		if a.onHistoryAdd != nil {
			a.onHistoryAdd("user", promptMsg)
		}
		a.history.AddMessage(model.NewUserMessage(promptMsg))
		result.Iterations = totalIterations
		return &iterationResult{action: "continue"}
	}

	// 前几次尝试时，给模型一个温和的提示
	promptMsg := "请根据上述内容继续执行。如果已经完成，请调用 complete_task 工具。"
	if a.onHistoryAdd != nil {
		a.onHistoryAdd("user", promptMsg)
	}
	a.history.AddMessage(model.NewUserMessage(promptMsg))
	result.Iterations = totalIterations
	return &iterationResult{action: "continue"}
}

// checkTaskCompletionViaLLM 通过 LLM 代理检查任务是否完成
// 返回 (complete bool, finalResponse string)
func (a *Agent) checkTaskCompletionViaLLM(ctx context.Context, taskGoal string, lastResponse string) (bool, string) {
	// 获取当前消息历史
	messages := a.history.GetMessagesRef()

	// 构建检查 prompt
	checkPrompt := fmt.Sprintf(`你是一个任务完成检查代理。请分析以下对话内容，判断用户请求的任务是否已经完成。

任务目标：%s

最近一次助手回复：
%s

请只返回 JSON 格式的判断结果，不要包含其他内容。JSON 格式如下：
{
  "complete": true/false,
  "reason": "简要说明判断理由"
}

判断标准：
1. 如果任务已经完成（包括明确回复完成、或内容已满足用户需求），complete 为 true
2. 如果任务尚未完成（如模型还在思考、需要继续执行等），complete 为 false
3. 不要误判，宁可多一次迭代也不提前结束`, taskGoal, lastResponse)

	// 构建检查请求的消息历史
	checkMessages := append([]model.Message{
		model.NewSystemMessage("你是一个任务完成检查代理，负责判断用户请求的任务是否已经完成。请严格按照 JSON 格式返回判断结果。"),
		model.NewUserMessage(checkPrompt),
	}, messages...)

	// 调用模型进行检查
	retryConfig := model.DefaultRetryConfig()
	streamChan, err := a.modelClient.StreamChatCompletionWithRetry(ctx, checkMessages, nil, retryConfig)
	if err != nil {
		fmt.Printf("[checkTaskCompletionViaLLM] 模型调用失败: %v\n", err)
		return false, ""
	}

	// 收集响应
	var responseBuilder strings.Builder
	for event := range streamChan {
		if event.Error != nil {
			fmt.Printf("[checkTaskCompletionViaLLM] 流式响应错误: %v\n", event.Error)
			return false, ""
		}
		if event.Done {
			break
		}
		if event.Response == nil || len(event.Response.Choices) == 0 {
			continue
		}
		choice := event.Response.Choices[0]
		if choice.Delta != nil && choice.Delta.Content != "" {
			responseBuilder.WriteString(choice.Delta.Content)
		}
	}

	checkResult := responseBuilder.String()

	// 解析 JSON 结果
	var checkResp struct {
		Complete bool   `json:"complete"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(checkResult), &checkResp); err != nil {
		fmt.Printf("[checkTaskCompletionViaLLM] JSON 解析失败: %v, 原始内容: %s\n", err, checkResult)
		return false, ""
	}

	fmt.Printf("[checkTaskCompletionViaLLM] LLM 判断: complete=%v, reason=%s\n", checkResp.Complete, checkResp.Reason)
	return checkResp.Complete, lastResponse
}

// handleForcedVerification 处理外层循环的强制校验
// 返回nil表示校验通过（主循环应完成任务），非nil表示应返回或continue
func (a *Agent) handleForcedVerification(runCtx context.Context, taskGoal string, result *RunResult) *iterationResult {
	verifyResult := a.verifyTaskCompletion(runCtx, taskGoal)

	// 记录强制校验结果
	if a.logger != nil {
		a.logger.LogVerification(verifyResult)
	}

	// 校验结果回调
	if a.onVerify != nil {
		a.onVerify(string(verifyResult.Status), verifyResult.Summary)
	}

	if verifyResult != nil && verifyResult.IsFailed() {
		// 强制校验失败，注入修正指令，继续循环
		correctionResult := a.corrector.AnalyzeFailure(verifyResult, nil)
		correctionInstruction := a.corrector.GenerateCorrectionInstruction(correctionResult)

		// 记录修正指令
		if a.logger != nil {
			a.logger.LogCorrection(correctionInstruction, "强制校验失败: "+verifyResult.Summary)
		}

		// 检查是否还有校验重试机会
		if !a.corrector.ShouldRetryVerify() {
			// 重试次数耗尽，任务失败
			result.Status = "failed"
			result.Error = fmt.Sprintf("强制校验失败且重试次数耗尽: %s", verifyResult.Summary)
			result.EndTime = time.Now()
			a.stateManager.UpdateTaskStatus(a.taskID, "failed")
			if a.logger != nil {
				a.logger.LogError(result.Error)
				a.logger.LogTaskEnd("failed", result.Iterations, result.EndTime.Sub(result.StartTime))
			}
			a.finalizeResult(result)
			return &iterationResult{action: "return", result: result, err: fmt.Errorf("forced verification failed and retries exhausted: %s", verifyResult.Summary)}
		}

		// 重置状态为进行中，注入修正指令，继续外层循环
		result.Status = "in_progress"
		a.stateManager.UpdateTaskStatus(a.taskID, "in_progress")
		if a.onCorrection != nil {
			a.onCorrection(correctionInstruction)
		}
		if a.onHistoryAdd != nil {
			a.onHistoryAdd("user", correctionInstruction)
		}
		a.history.AddMessage(model.NewUserMessage(correctionInstruction))
		return &iterationResult{action: "continue"} // 外层循环continue
	}

	return nil // 校验通过
}

// injectContinuePrompt 注入继续执行的提示消息
func (a *Agent) injectContinuePrompt() {
	promptMsg := "请根据上述内容继续执行。如果已经完成，请调用complete_task工具。"
	if a.onHistoryAdd != nil {
		a.onHistoryAdd("user", promptMsg)
	}
	a.history.AddMessage(model.NewUserMessage(promptMsg))
}
