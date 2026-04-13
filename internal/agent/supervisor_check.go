// Package agent Supervisor检查集成
// 在Agent主循环每次迭代后执行监督检查，根据干预级别处理结果
package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Attect/MukaAI/internal/model"
)

// runSupervision 在迭代中执行监督检查
// 在工具执行后或模型输出后调用，检查Agent行为质量
// 返回nil表示无干预，非nil表示需要处理干预
func (a *Agent) runSupervision(
	runCtx context.Context,
	response *modelResponse,
	toolCallsResult []model.Message,
	taskGoal string,
	totalIterations int,
) *iterationResult {
	// 获取supervisor（线程安全）
	a.mu.RLock()
	sup := a.supervisor
	a.mu.RUnlock()

	if sup == nil {
		return nil // 未启用监督
	}

	// 构建AgentOutput
	output := a.buildAgentOutput(response, toolCallsResult, totalIterations)

	// 获取任务状态
	taskState, _ := a.stateManager.GetState(a.taskID)

	// 执行监督检查
	supResult := sup.Check(runCtx, output, taskState)

	// 记录监督结果
	if a.logger != nil {
		a.logger.LogMessage("supervisor", supResult.Summary)
	}

	// 触发监督结果回调
	if a.onSupervisor != nil {
		a.onSupervisor(supResult)
	}

	// 通过StreamHandler推送监督结果到GUI
	a.mu.RLock()
	handler := a.streamHandler
	a.mu.RUnlock()
	if handler != nil {
		if sh, ok := handler.(SupervisorResultHandler); ok {
			sh.OnSupervisorResult(supResult)
		}
	}

	// 根据干预级别处理
	if supResult.Status == "pass" {
		return nil // 无问题
	}

	log.Printf("[Supervisor] status=%s, intervention=%s, summary=%s",
		supResult.Status, supResult.InterventionType, supResult.Summary)

	switch supResult.InterventionType {
	case "interrupt", "rollback":
		// 严重干预：终止当前任务
		return a.handleSupervisorHalt(supResult, totalIterations)

	case "pause":
		// 中等干预：注入修正指令到对话历史
		return a.handleSupervisorCorrection(supResult, totalIterations)

	case "warning", "":
		// 轻微干预：仅记录警告日志
		log.Printf("[Supervisor] Warning: %s", supResult.Summary)
		for _, issue := range supResult.Issues {
			log.Printf("[Supervisor]   - [%s] %s: %s", issue.Severity, issue.Type, issue.Description)
		}
		return nil
	}

	return nil
}

// buildAgentOutput 从模型响应和工具结果构建AgentOutput
func (a *Agent) buildAgentOutput(
	response *modelResponse,
	toolCallsResult []model.Message,
	iteration int,
) *AgentOutput {
	output := &AgentOutput{
		Content:   response.Content,
		ToolCalls: response.ToolCalls,
		TaskID:    a.taskID,
		Iteration: iteration,
		Success:   true,
	}

	// 检查工具结果中是否有错误
	for _, msg := range toolCallsResult {
		if msg.Role == model.RoleTool {
			// 工具结果中的错误通常通过特定字段标识
			if msg.Content != "" && containsError(msg.Content) {
				output.Error = msg.Content
				output.Success = false
				break
			}
		}
	}

	return output
}

// handleSupervisorHalt 处理halt/interrupt/rollback级别干预
// 终止当前任务
func (a *Agent) handleSupervisorHalt(supResult *SupervisionResult, totalIterations int) *iterationResult {
	log.Printf("[Supervisor] HALT: 任务被监督器终止 - %s", supResult.Summary)

	// 构建详细的错误信息
	errMsg := fmt.Sprintf("Supervisor干预终止: %s", supResult.Summary)
	if len(supResult.Issues) > 0 {
		errMsg += "\n问题列表:"
		for _, issue := range supResult.Issues {
			errMsg += fmt.Sprintf("\n  - [%s] %s: %s", issue.Severity, issue.Type, issue.Description)
		}
	}

	result := &RunResult{
		TaskID:    a.taskID,
		Status:    "failed",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Error:     errMsg,
	}
	a.stateManager.UpdateTaskStatus(a.taskID, "failed")
	if a.logger != nil {
		a.logger.LogError(errMsg)
	}
	a.finalizeResult(result)

	return &iterationResult{
		action: "return",
		result: result,
		err:    fmt.Errorf("supervisor halt: %s", supResult.Summary),
	}
}

// handleSupervisorCorrection 处理correction级别干预
// 注入修正指令到对话历史，让模型自行修正
// 包含冷却期机制：至少间隔3次迭代才注入下一次修正指令，避免频繁干扰模型推理
func (a *Agent) handleSupervisorCorrection(supResult *SupervisionResult, totalIterations int) *iterationResult {
	// 冷却期检查：至少间隔3次迭代才注入下一次修正指令
	a.mu.RLock()
	lastIter := a.lastCorrectionIteration
	a.mu.RUnlock()

	if lastIter > 0 && totalIterations-lastIter < 3 {
		log.Printf("[Supervisor] 修正指令冷却中，跳过注入 (上次迭代: %d, 当前: %d)", lastIter, totalIterations)
		return nil
	}

	a.mu.Lock()
	a.lastCorrectionIteration = totalIterations
	a.mu.Unlock()

	log.Printf("[Supervisor] CORRECTION: 注入修正指令 - %s", supResult.Summary)

	// 构建修正指令
	correctionInstruction := a.buildSupervisorCorrectionInstruction(supResult)

	// 注入修正指令到对话历史
	if a.onCorrection != nil {
		a.onCorrection(correctionInstruction)
	}
	if a.onHistoryAdd != nil {
		a.onHistoryAdd("user", correctionInstruction)
	}
	a.history.AddMessage(model.NewUserMessage(correctionInstruction))

	return nil // 继续迭代，模型会根据修正指令调整
}

// buildSupervisorCorrectionInstruction 从监督结果构建修正指令
func (a *Agent) buildSupervisorCorrectionInstruction(supResult *SupervisionResult) string {
	instruction := "[系统监督修正]\n"
	instruction += supResult.Summary + "\n\n"
	instruction += "请根据以下反馈调整你的行为：\n"

	for _, issue := range supResult.Issues {
		instruction += fmt.Sprintf("- [%s] %s\n", issue.Severity, issue.Description)
		if issue.Suggestion != "" {
			instruction += fmt.Sprintf("  建议: %s\n", issue.Suggestion)
		}
	}

	instruction += "\n请继续执行任务，注意避免上述问题。"
	return instruction
}

// containsError 检查工具结果内容是否包含错误标识
// 排除工具正常返回的业务错误（success:true 时即使有 error 字段也不算错误）
func containsError(content string) bool {
	// 如果内容包含 "success":true，说明工具成功返回，不算错误
	successMarkers := []string{`"success":true`, `"success": true`}
	for _, marker := range successMarkers {
		if len(content) >= len(marker) {
			for i := 0; i <= len(content)-len(marker); i++ {
				if content[i:i+len(marker)] == marker {
					return false
				}
			}
		}
	}

	// 检查是否包含明确的错误字段
	// 收紧匹配范围，避免误判工具返回的JSON中偶然出现的error字符串
	errorMarkers := []string{`"error":"`, `"error": "`}
	for _, marker := range errorMarkers {
		if len(content) >= len(marker) {
			for i := 0; i <= len(content)-len(marker); i++ {
				if content[i:i+len(marker)] == marker {
					return true
				}
			}
		}
	}
	return false
}
