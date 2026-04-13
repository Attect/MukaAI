// Package agent 实现Agent核心循环和业务逻辑
// fork.go 实现子代理Fork机制，支持Agent创建子代理执行特定任务
package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Attect/MukaAI/internal/model"
	"github.com/Attect/MukaAI/internal/state"
	"github.com/Attect/MukaAI/internal/tools"
)

// ForkManager 管理子代理的创建、执行和合并
// 提供Agent身份切换和上下文隔离机制
type ForkManager struct {
	// 核心组件
	modelClient  *model.Client       // 模型客户端
	toolRegistry *tools.ToolRegistry // 工具注册中心
	stateManager *state.StateManager // 状态管理器

	// 配置
	maxIterations int    // 子代理最大迭代次数
	workDir       string // 工作目录

	// 状态
	mu          sync.RWMutex
	activeForks map[string]*ForkedAgent // 活动的子代理
	forkCounter int                     // 子代理计数器

	// 回调
	onForkStart   func(role, task string)    // 子代理开始回调
	onForkEnd     func(role, summary string) // 子代理结束回调
	onStreamChunk func(chunk string)         // 流式输出回调
}

// ForkedAgent 表示一个被Fork出来的子代理
type ForkedAgent struct {
	ID           string    // 子代理唯一标识
	Role         string    // 子代理角色
	Task         string    // 子代理任务描述
	ParentTaskID string    // 父任务ID
	Agent        *Agent    // 子代理实例
	StartTime    time.Time // 开始时间
	EndTime      time.Time // 结束时间
	Summary      string    // 执行总结
	Status       string    // 状态：running, completed, failed
}

// ForkConfig Fork配置
type ForkConfig struct {
	ModelClient   *model.Client       // 模型客户端（必需）
	ToolRegistry  *tools.ToolRegistry // 工具注册中心（必需）
	StateManager  *state.StateManager // 状态管理器（必需）
	MaxIterations int                 // 子代理最大迭代次数（默认30）
	WorkDir       string              // 工作目录（可选）
}

// NewForkManager 创建新的Fork管理器
func NewForkManager(config *ForkConfig) (*ForkManager, error) {
	if config == nil {
		return nil, fmt.Errorf("fork config cannot be nil")
	}

	if config.ModelClient == nil {
		return nil, fmt.Errorf("model client is required")
	}

	if config.ToolRegistry == nil {
		return nil, fmt.Errorf("tool registry is required")
	}

	if config.StateManager == nil {
		return nil, fmt.Errorf("state manager is required")
	}

	maxIterations := config.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 30 // 子代理默认迭代次数较少
	}

	return &ForkManager{
		modelClient:   config.ModelClient,
		toolRegistry:  config.ToolRegistry,
		stateManager:  config.StateManager,
		maxIterations: maxIterations,
		workDir:       config.WorkDir,
		activeForks:   make(map[string]*ForkedAgent),
	}, nil
}

// Fork 创建并执行子代理
// parentAgent: 父Agent实例
// role: 子代理角色（如 "Worker", "Reviewer"）
// task: 子代理任务描述
// 返回子代理执行结果
func (fm *ForkManager) Fork(ctx context.Context, parentAgent *Agent, role, task string) (*ForkResult, error) {
	// 验证参数
	if parentAgent == nil {
		return nil, fmt.Errorf("parent agent cannot be nil")
	}
	if role == "" {
		return nil, fmt.Errorf("role cannot be empty")
	}
	if task == "" {
		return nil, fmt.Errorf("task cannot be empty")
	}

	// 生成子代理ID
	fm.mu.Lock()
	fm.forkCounter++
	forkID := fmt.Sprintf("fork-%d-%d", time.Now().UnixNano(), fm.forkCounter)
	fm.mu.Unlock()

	// 获取父任务ID
	parentTaskID := parentAgent.GetTaskID()

	// 创建子代理
	forkedAgent := &ForkedAgent{
		ID:           forkID,
		Role:         role,
		Task:         task,
		ParentTaskID: parentTaskID,
		StartTime:    time.Now(),
		Status:       "running",
	}

	// 注册到活动列表
	fm.mu.Lock()
	fm.activeForks[forkID] = forkedAgent
	fm.mu.Unlock()

	// 确保清理
	defer func() {
		fm.mu.Lock()
		delete(fm.activeForks, forkID)
		fm.mu.Unlock()
	}()

	// 回调
	if fm.onForkStart != nil {
		fm.onForkStart(role, task)
	}

	// 构建子代理系统提示词
	systemPrompt := fm.buildForkedAgentPrompt(role, task)

	// 创建子代理配置
	agentConfig := &Config{
		ModelClient:   fm.modelClient,
		ToolRegistry:  fm.toolRegistry,
		StateManager:  fm.stateManager,
		MaxIterations: fm.maxIterations,
		SystemPrompt:  systemPrompt,
		PromptType:    getPromptTypeByRole(role),
		WorkDir:       fm.workDir,
	}

	// 创建子代理实例
	agent, err := NewAgent(agentConfig)
	if err != nil {
		forkedAgent.Status = "failed"
		return nil, fmt.Errorf("failed to create forked agent: %w", err)
	}

	// 设置子代理的任务ID（使用父任务ID，共享状态）
	agent.SetTaskID(parentTaskID)

	// 设置流式输出回调
	if fm.onStreamChunk != nil {
		agent.SetOnStreamChunk(fm.onStreamChunk)
	}

	forkedAgent.Agent = agent

	// 构建子代理初始消息
	// 包含必要的上下文，但不复制完整历史
	forkMessages := fm.buildForkMessages(parentAgent, role, task)

	// 执行子代理
	result, err := fm.executeForkedAgent(ctx, agent, forkMessages)

	forkedAgent.EndTime = time.Now()

	if err != nil {
		forkedAgent.Status = "failed"
		return nil, fmt.Errorf("forked agent execution failed: %w", err)
	}

	forkedAgent.Status = "completed"
	forkedAgent.Summary = result.Summary

	// 回调
	if fm.onForkEnd != nil {
		fm.onForkEnd(role, result.Summary)
	}

	return result, nil
}

// ForkResult 子代理执行结果
type ForkResult struct {
	ForkID     string        // 子代理ID
	Role       string        // 角色
	Task       string        // 任务描述
	Summary    string        // 执行总结
	Status     string        // 状态
	Duration   time.Duration // 执行时长
	Iterations int           // 迭代次数
}

// buildForkedAgentPrompt 构建子代理的系统提示词
func (fm *ForkManager) buildForkedAgentPrompt(role, task string) string {
	// 根据角色获取基础提示词
	basePrompt := GetSystemPrompt(getPromptTypeByRole(role))

	// 添加身份切换提示
	identityPrompt := fmt.Sprintf(`

## 身份说明
你当前以【%s】身份执行任务。

## 任务说明
%s

## 执行要求
1. 专注于当前分配的任务，不要越界
2. 完成任务后，使用 complete_as_agent 工具提交总结
3. 如果遇到无法解决的问题，使用 fail_task 工具报告失败
4. 保持输出简洁，避免冗余

## 完成后
完成任务后，请使用 complete_as_agent 工具提交执行总结。
总结应简洁明了，说明完成了什么工作。
`, role, task)

	return basePrompt + identityPrompt
}

// buildForkMessages 构建子代理的初始消息
// 只复制必要的上下文，不复制完整历史
func (fm *ForkManager) buildForkMessages(parentAgent *Agent, role, task string) []model.Message {
	messages := make([]model.Message, 0)

	// 获取父任务的当前状态摘要
	stateSummary, err := fm.stateManager.GetYAMLSummary(parentAgent.GetTaskID())
	if err != nil {
		stateSummary = ""
	}

	// 构建任务提示
	taskPrompt := BuildForkTaskPrompt(role, task, stateSummary)

	// 添加用户消息
	messages = append(messages, model.NewUserMessage(taskPrompt))

	return messages
}

// executeForkedAgent 执行子代理
func (fm *ForkManager) executeForkedAgent(ctx context.Context, agent *Agent, initialMessages []model.Message) (*ForkResult, error) {
	// 将初始消息添加到子代理历史
	for _, msg := range initialMessages {
		agent.GetHistory().AddMessage(msg)
	}

	// 使用一个特殊的任务目标来运行子代理
	// 子代理会通过 complete_as_agent 工具来结束
	taskGoal := "执行分配的子任务，完成后使用 complete_as_agent 提交总结。"

	// 运行子代理
	runResult, err := agent.Run(ctx, taskGoal)
	if err != nil {
		return nil, err
	}

	// 构建结果
	result := &ForkResult{
		ForkID:     agent.taskID,
		Status:     runResult.Status,
		Duration:   runResult.EndTime.Sub(runResult.StartTime),
		Iterations: runResult.Iterations,
	}

	// 从历史中提取总结
	// 子代理应该通过 complete_as_agent 工具提交总结
	// 如果没有，则使用最终响应作为总结
	result.Summary = fm.extractSummaryFromHistory(agent)

	return result, nil
}

// extractSummaryFromHistory 从子代理历史中提取总结
func (fm *ForkManager) extractSummaryFromHistory(agent *Agent) string {
	// 查找 complete_as_agent 工具调用
	history := agent.GetHistory()
	messages := history.GetMessages()

	// 倒序查找工具结果
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == model.RoleTool && msg.Name == "complete_as_agent" {
			// 解析工具结果
			return msg.Content
		}
	}

	// 如果没有找到工具结果，使用最后的助手消息
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == model.RoleAssistant && msg.Content != "" {
			// 截取前500字符作为总结
			if len(msg.Content) > 500 {
				return msg.Content[:500] + "..."
			}
			return msg.Content
		}
	}

	return "子代理执行完成，但未提供总结"
}

// Join 合并子代理结果到父Agent
// 将子代理的执行总结作为工具结果返回给父Agent
func (fm *ForkManager) Join(parentAgent *Agent, forkResult *ForkResult) (string, error) {
	if parentAgent == nil {
		return "", fmt.Errorf("parent agent cannot be nil")
	}
	if forkResult == nil {
		return "", fmt.Errorf("fork result cannot be nil")
	}

	// 构建合并提示
	joinPrompt := BuildJoinPrompt(forkResult.Role, forkResult.Task, forkResult.Summary)

	// 更新状态管理器中的Agent记录
	duration := forkResult.Duration.String()
	if err := fm.stateManager.SwitchAgent(
		parentAgent.GetTaskID(),
		"Orchestrator", // 返回主Agent
		fmt.Sprintf("[%s] %s", forkResult.Role, forkResult.Summary),
		duration,
	); err != nil {
		// 记录错误但不中断流程
		fmt.Printf("Warning: failed to update agent record: %v\n", err)
	}

	return joinPrompt, nil
}

// GetActiveForks 获取当前活动的子代理列表
func (fm *ForkManager) GetActiveForks() []*ForkedAgent {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	forks := make([]*ForkedAgent, 0, len(fm.activeForks))
	for _, fork := range fm.activeForks {
		forks = append(forks, fork)
	}
	return forks
}

// SetOnForkStart 设置子代理开始回调
func (fm *ForkManager) SetOnForkStart(callback func(role, task string)) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.onForkStart = callback
}

// SetOnForkEnd 设置子代理结束回调
func (fm *ForkManager) SetOnForkEnd(callback func(role, summary string)) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.onForkEnd = callback
}

// SetOnStreamChunk 设置流式输出回调
func (fm *ForkManager) SetOnStreamChunk(callback func(chunk string)) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.onStreamChunk = callback
}

// ==================== 辅助函数 ====================

// getPromptTypeByRole 根据角色获取提示词类型
func getPromptTypeByRole(role string) SystemPromptType {
	switch role {
	case "Worker", "worker":
		return PromptTypeWorker
	case "Reviewer", "reviewer":
		return PromptTypeReviewer
	default:
		return PromptTypeOrchestrator
	}
}

// BuildForkTaskPrompt 构建子代理任务提示
func BuildForkTaskPrompt(role, task, stateSummary string) string {
	prompt := fmt.Sprintf("## 身份切换\n\n接下来我转变身份为【%s】，需要执行以下任务：\n\n", role)
	prompt += fmt.Sprintf("## 任务内容\n\n%s\n\n", task)

	if stateSummary != "" {
		prompt += "## 当前任务状态\n\n" + stateSummary + "\n\n"
	}

	prompt += "请开始执行任务。完成后使用 complete_as_agent 工具提交执行总结。"
	return prompt
}

// BuildJoinPrompt 构建合并提示
func BuildJoinPrompt(role, task, summary string) string {
	return fmt.Sprintf("## 子代理执行完成\n\n我以【%s】身份完成了以下任务：\n\n**任务**: %s\n\n**执行总结**: %s\n\n现在继续主任务，请检查YAML状态并继续执行。", role, task, summary)
}

// ==================== 子代理工具定义 ====================

// SpawnAgentTool 创建子代理工具
type SpawnAgentTool struct {
	forkManager *ForkManager
	parentAgent *Agent
}

// NewSpawnAgentTool 创建子代理工具
func NewSpawnAgentTool(forkManager *ForkManager, parentAgent *Agent) *SpawnAgentTool {
	return &SpawnAgentTool{
		forkManager: forkManager,
		parentAgent: parentAgent,
	}
}

func (t *SpawnAgentTool) Name() string {
	return "spawn_agent"
}

func (t *SpawnAgentTool) Description() string {
	return `创建一个子代理来执行特定任务。子代理会以指定的角色身份执行任务，完成后返回执行总结。

使用场景：
- 需要以特定角色（如Worker、Reviewer）执行任务
- 需要隔离上下文执行独立任务
- 需要并行或嵌套执行任务

注意：子代理执行是同步的，主Agent会等待子代理完成。`
}

func (t *SpawnAgentTool) Parameters() map[string]interface{} {
	return tools.BuildSchema(map[string]*tools.ToolParameter{
		"role": {
			Type:        "string",
			Description: "子代理角色，如 'Worker'（执行者）、'Reviewer'（审查者）等",
			Required:    true,
			Enum:        []string{"Worker", "Reviewer", "Specialist"},
		},
		"task": {
			Type:        "string",
			Description: "子代理需要执行的任务描述，应清晰具体",
			Required:    true,
		},
	}, []string{"role", "task"})
}

func (t *SpawnAgentTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	role, _ := params["role"].(string)
	task, _ := params["task"].(string)

	if role == "" {
		return tools.NewErrorResult("missing required parameter: role"), nil
	}
	if task == "" {
		return tools.NewErrorResult("missing required parameter: task"), nil
	}

	// 创建并执行子代理
	result, err := t.forkManager.Fork(ctx, t.parentAgent, role, task)
	if err != nil {
		return tools.NewErrorResult(fmt.Sprintf("failed to spawn agent: %v", err)), nil
	}

	// 合并结果到父Agent
	joinPrompt, err := t.forkManager.Join(t.parentAgent, result)
	if err != nil {
		return tools.NewErrorResult(fmt.Sprintf("failed to join agent: %v", err)), nil
	}

	return tools.NewSuccessResult(map[string]interface{}{
		"fork_id":    result.ForkID,
		"role":       result.Role,
		"task":       result.Task,
		"summary":    result.Summary,
		"status":     result.Status,
		"duration":   result.Duration.String(),
		"iterations": result.Iterations,
		"message":    joinPrompt,
	}), nil
}

// CompleteAsAgentTool 子代理完成任务工具
type CompleteAsAgentTool struct{}

// NewCompleteAsAgentTool 创建子代理完成任务工具
func NewCompleteAsAgentTool() *CompleteAsAgentTool {
	return &CompleteAsAgentTool{}
}

func (t *CompleteAsAgentTool) Name() string {
	return "complete_as_agent"
}

func (t *CompleteAsAgentTool) Description() string {
	return `子代理完成任务后提交执行总结。此工具用于标记子代理任务完成并返回总结给主Agent。

使用时机：
- 子代理完成了分配的任务
- 需要将执行结果返回给主Agent

总结要求：
- 简洁明了，说明完成了什么工作
- 包含关键结果或发现
- 如有问题，说明遇到的问题`
}

func (t *CompleteAsAgentTool) Parameters() map[string]interface{} {
	return tools.BuildSchema(map[string]*tools.ToolParameter{
		"summary": {
			Type:        "string",
			Description: "执行总结，应简洁明了地说明完成的工作和结果",
			Required:    true,
		},
	}, []string{"summary"})
}

func (t *CompleteAsAgentTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	summary, _ := params["summary"].(string)

	if summary == "" {
		return tools.NewErrorResult("missing required parameter: summary"), nil
	}

	// 返回成功结果，总结会被提取并返回给主Agent
	return tools.NewSuccessResult(map[string]interface{}{
		"success": true,
		"summary": summary,
		"message": "子代理任务完成，总结已提交",
	}), nil
}

// RegisterForkTools 注册Fork相关工具到注册中心
func RegisterForkTools(registry *tools.ToolRegistry, forkManager *ForkManager, parentAgent *Agent) error {
	// 注册 spawn_agent 工具
	if err := registry.RegisterTool(NewSpawnAgentTool(forkManager, parentAgent)); err != nil {
		return fmt.Errorf("failed to register spawn_agent tool: %w", err)
	}

	// 注册 complete_as_agent 工具
	if err := registry.RegisterTool(NewCompleteAsAgentTool()); err != nil {
		return fmt.Errorf("failed to register complete_as_agent tool: %w", err)
	}

	return nil
}
