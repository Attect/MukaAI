// Package agent 任务完成校验逻辑
// 从 core.go 提取的任务完成校验方法
package agent

import (
	"context"
	"fmt"
	"time"
)

// verifyTaskCompletion 验证任务完成情况
// 在任务标记完成前调用，确保任务真正完成
func (a *Agent) verifyTaskCompletion(ctx context.Context, taskGoal string) *VerifyResult {
	// 获取当前任务状态
	taskState, err := a.stateManager.GetState(a.taskID)
	if err != nil {
		return &VerifyResult{
			Status: VerifyStatusFail,
			Issues: []VerifyIssue{
				{
					Type:        VerifyIssueTypeContentMissing,
					Severity:    "high",
					Description: fmt.Sprintf("无法获取任务状态: %s", err.Error()),
					Timestamp:   time.Now(),
				},
			},
			Timestamp: time.Now(),
			Summary:   "无法获取任务状态",
		}
	}

	// 提取需要校验的文件列表
	files := make([]string, 0)
	if taskState != nil {
		for _, fileInfo := range taskState.Context.Files {
			if fileInfo.Status == "created" || fileInfo.Status == "modified" {
				files = append(files, fileInfo.Path)
			}
		}
	}

	// 执行任务完成校验（不检查任务状态，因为此时任务还未标记为completed）
	// 使用Verify而不是VerifyTaskCompletion，因为后者会检查任务状态
	return a.verifier.Verify(files, taskState)
}
