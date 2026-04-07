package tools

import (
	"context"
	"fmt"
)

// VerifyResult 校验结果（从agent包导入，避免循环依赖）
// 这里定义一个简化的结构，实际使用时会从agent包传递
type VerifyResult struct {
	Status  string        // 校验状态
	Issues  []VerifyIssue // 发现的问题列表
	Summary string        // 校验摘要
	Passed  int           // 通过的检查项数量
	Failed  int           // 失败的检查项数量
}

// VerifyIssue 校验问题
type VerifyIssue struct {
	Type        string // 问题类型
	Severity    string // 严重程度
	Description string // 问题描述
	Evidence    string // 证据
	Suggestion  string // 修正建议
	FilePath    string // 相关文件路径
}

// completeTaskTool 完成任务工具
type completeTaskTool struct {
	// verifier 校验回调函数
	// ctx: 上下文
	// taskGoal: 任务目标
	// workDir: 工作目录
	// 返回校验结果，如果为nil表示不进行校验
	verifier func(ctx context.Context, taskGoal string, workDir string) *VerifyResult
}

// NewCompleteTaskTool 创建完成任务工具
func NewCompleteTaskTool() *completeTaskTool {
	return &completeTaskTool{}
}

// NewCompleteTaskToolWithVerifier 创建带校验器的完成任务工具
func NewCompleteTaskToolWithVerifier(verifierFunc func(ctx context.Context, taskGoal string, workDir string) *VerifyResult) *completeTaskTool {
	return &completeTaskTool{
		verifier: verifierFunc,
	}
}

func (t *completeTaskTool) Name() string {
	return "complete_task"
}

func (t *completeTaskTool) Description() string {
	return "标记任务已完成。当所有任务目标都已达成时调用此工具。"
}

func (t *completeTaskTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"summary": map[string]interface{}{
				"type":        "string",
				"description": "任务完成总结",
			},
		},
		"required": []string{"summary"},
	}
}

func (t *completeTaskTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	summary, _ := params["summary"].(string)

	// 如果设置了校验器，执行校验
	if t.verifier != nil {
		// 从上下文中获取任务目标和工作目录（这些信息需要从外部注入）
		// 这里简化处理，实际使用时需要从上下文或参数中获取
		taskGoal, _ := ctx.Value("task_goal").(string)
		workDir, _ := ctx.Value("work_dir").(string)

		verifyResult := t.verifier(ctx, taskGoal, workDir)

		// 校验失败时返回失败结果
		if verifyResult != nil && verifyResult.Status == "fail" {
			// 构建失败信息
			failureMsg := fmt.Sprintf("任务完成校验失败: %s\n", verifyResult.Summary)
			if len(verifyResult.Issues) > 0 {
				failureMsg += "发现的问题:\n"
				for i, issue := range verifyResult.Issues {
					failureMsg += fmt.Sprintf("%d. [%s] %s\n", i+1, issue.Severity, issue.Description)
					if issue.Suggestion != "" {
						failureMsg += fmt.Sprintf("   建议: %s\n", issue.Suggestion)
					}
				}
			}

			return &ToolResult{
				Success: false,
				Error:   failureMsg,
				Data: map[string]interface{}{
					"status":        "verification_failed",
					"verify_result": verifyResult,
					"summary":       summary,
					"passed_checks": verifyResult.Passed,
					"failed_checks": verifyResult.Failed,
				},
			}, nil
		}
	}

	// 校验通过或未设置校验器，返回成功
	return &ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"status":  "completed",
			"summary": summary,
		},
	}, nil
}

type failTaskTool struct{}

func (t *failTaskTool) Name() string {
	return "fail_task"
}

func (t *failTaskTool) Description() string {
	return "标记任务失败。当遇到无法解决的问题时调用此工具。"
}

func (t *failTaskTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"reason": map[string]interface{}{
				"type":        "string",
				"description": "失败原因",
			},
		},
		"required": []string{"reason"},
	}
}

func (t *failTaskTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	reason, _ := params["reason"].(string)
	return &ToolResult{
		Success: false,
		Data: map[string]interface{}{
			"status": "failed",
			"reason": reason,
		},
	}, nil
}

type updateStateTool struct{}

func (t *updateStateTool) Name() string {
	return "update_state"
}

func (t *updateStateTool) Description() string {
	return "更新任务状态。记录完成的步骤或添加决策。"
}

func (t *updateStateTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"completed_step": map[string]interface{}{
				"type":        "string",
				"description": "已完成的步骤描述",
			},
			"decision": map[string]interface{}{
				"type":        "string",
				"description": "做出的决策",
			},
			"current_phase": map[string]interface{}{
				"type":        "string",
				"description": "当前阶段",
			},
		},
	}
}

func (t *updateStateTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	updates := make([]string, 0)

	if step, ok := params["completed_step"].(string); ok && step != "" {
		updates = append(updates, fmt.Sprintf("完成步骤: %s", step))
	}
	if decision, ok := params["decision"].(string); ok && decision != "" {
		updates = append(updates, fmt.Sprintf("决策: %s", decision))
	}
	if phase, ok := params["current_phase"].(string); ok && phase != "" {
		updates = append(updates, fmt.Sprintf("当前阶段: %s", phase))
	}

	return &ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"updates": updates,
		},
	}, nil
}

func RegisterStateTools(registry *ToolRegistry) {
	registry.MustRegisterTool(NewCompleteTaskTool())
	registry.MustRegisterTool(&failTaskTool{})
	registry.MustRegisterTool(&updateStateTool{})
}

// RegisterStateToolsWithVerifier 注册状态工具（带校验器）
// verifierFunc: 校验回调函数，用于在complete_task时验证任务完成情况
func RegisterStateToolsWithVerifier(registry *ToolRegistry, verifierFunc func(ctx context.Context, taskGoal string, workDir string) *VerifyResult) {
	registry.MustRegisterTool(NewCompleteTaskToolWithVerifier(verifierFunc))
	registry.MustRegisterTool(&failTaskTool{})
	registry.MustRegisterTool(&updateStateTool{})
}

func RegisterDefaultStateTools() {
	RegisterStateTools(defaultRegistry)
}
