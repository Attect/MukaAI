package state

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadYAML 从文件加载YAML并解析为TaskState
// 参数：
//   - filePath: YAML文件路径
//
// 返回：
//   - *TaskState: 解析后的任务状态
//   - error: 错误信息
func LoadYAML(filePath string) (*TaskState, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file %s: %w", filePath, err)
	}

	return ParseYAML(data)
}

// SaveYAML 将TaskState序列化为YAML并保存到文件
// 参数：
//   - state: 任务状态实例
//   - filePath: 目标文件路径
//
// 返回：
//   - error: 错误信息
func SaveYAML(state *TaskState, filePath string) error {
	data, err := ToYAML(state)
	if err != nil {
		return fmt.Errorf("failed to serialize state to YAML: %w", err)
	}

	// 使用0644权限创建文件，所有者可读写，其他用户只读
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write YAML file %s: %w", filePath, err)
	}

	return nil
}

// ParseYAML 解析YAML字节数据为TaskState
// 参数：
//   - data: YAML字节数据
//
// 返回：
//   - *TaskState: 解析后的任务状态
//   - error: 错误信息
func ParseYAML(data []byte) (*TaskState, error) {
	var state TaskState
	err := yaml.Unmarshal(data, &state)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &state, nil
}

// ToYAML 将TaskState序列化为YAML字节数据
// 参数：
//   - state: 任务状态实例
//
// 返回：
//   - []byte: YAML字节数据
//   - error: 错误信息
func ToYAML(state *TaskState) ([]byte, error) {
	if state == nil {
		return nil, fmt.Errorf("state is nil")
	}

	data, err := yaml.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state to YAML: %w", err)
	}

	return data, nil
}

// ToYAMLString 将TaskState序列化为YAML字符串
// 参数：
//   - state: 任务状态实例
//
// 返回：
//   - string: YAML字符串
//   - error: 错误信息
func ToYAMLString(state *TaskState) (string, error) {
	data, err := ToYAML(state)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetYAMLSummary 生成任务状态的YAML摘要，用于上下文压缩
// 该方法提取关键信息，生成简洁的摘要版本
// 参数：
//   - state: 任务状态实例
//
// 返回：
//   - string: 摘要字符串
//   - error: 错误信息
func GetYAMLSummary(state *TaskState) (string, error) {
	if state == nil {
		return "", fmt.Errorf("state is nil")
	}

	// 创建摘要结构，只包含关键信息
	summary := fmt.Sprintf("Task ID: %s\n", state.Task.ID)
	summary += fmt.Sprintf("Goal: %s\n", state.Task.Goal)
	summary += fmt.Sprintf("Status: %s\n", state.Task.Status)
	summary += fmt.Sprintf("Current Phase: %s\n", state.Progress.CurrentPhase)

	// 已完成步骤数量
	summary += fmt.Sprintf("Completed Steps: %d\n", len(state.Progress.CompletedSteps))
	if len(state.Progress.CompletedSteps) > 0 {
		// 只显示最后3个完成的步骤
		start := 0
		if len(state.Progress.CompletedSteps) > 3 {
			start = len(state.Progress.CompletedSteps) - 3
		}
		for i := start; i < len(state.Progress.CompletedSteps); i++ {
			summary += fmt.Sprintf("  - %s\n", state.Progress.CompletedSteps[i])
		}
	}

	// 待完成步骤
	summary += fmt.Sprintf("Pending Steps: %d\n", len(state.Progress.PendingSteps))
	if len(state.Progress.PendingSteps) > 0 {
		// 只显示前5个待完成步骤
		limit := 5
		if len(state.Progress.PendingSteps) < limit {
			limit = len(state.Progress.PendingSteps)
		}
		for i := 0; i < limit; i++ {
			summary += fmt.Sprintf("  - %s\n", state.Progress.PendingSteps[i])
		}
	}

	// 关键决策
	summary += fmt.Sprintf("Key Decisions: %d\n", len(state.Context.Decisions))
	if len(state.Context.Decisions) > 0 {
		// 只显示最后3个决策
		start := 0
		if len(state.Context.Decisions) > 3 {
			start = len(state.Context.Decisions) - 3
		}
		for i := start; i < len(state.Context.Decisions); i++ {
			summary += fmt.Sprintf("  - %s\n", state.Context.Decisions[i])
		}
	}

	// 当前活动Agent
	summary += fmt.Sprintf("Active Agent: %s\n", state.Agents.Active)

	// Agent历史记录数量
	summary += fmt.Sprintf("Agent History: %d records\n", len(state.Agents.History))

	return summary, nil
}
