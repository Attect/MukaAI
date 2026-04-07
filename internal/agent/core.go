package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"agentplus/internal/model"
	"agentplus/internal/state"
	"agentplus/internal/tools"
)

// Agent 核心Agent结构体
// 负责协调模型、工具和状态管理，实现任务执行主循环
type Agent struct {
	// 核心组件
	modelClient  *model.Client       // 模型客户端
	toolRegistry *tools.ToolRegistry // 工具注册中心
	stateManager *state.StateManager // 状态管理器
	executor     *ToolExecutor       // 工具执行器
	history      *HistoryManager     // 消息历史

	// 校验和修正组件
	reviewer  *Reviewer      // 程序逻辑审查器
	verifier  *Verifier      // 成果校验器
	corrector *SelfCorrector // 自我修正器

	// 配置
	maxIterations int              // 最大迭代次数
	systemPrompt  string           // 系统提示词
	taskID        string           // 当前任务ID
	promptType    SystemPromptType // 提示词类型

	// 状态
	mu         sync.RWMutex
	running    bool
	cancelFunc context.CancelFunc

	// 回调
	onStreamChunk func(chunk string)      // 流式输出回调
	onToolCall    func(name, args string) // 工具调用回调
	onIteration   func(iteration int)     // 迭代回调
}

// Config Agent配置
type Config struct {
	ModelClient   *model.Client       // 模型客户端（必需）
	ToolRegistry  *tools.ToolRegistry // 工具注册中心（必需）
	StateManager  *state.StateManager // 状态管理器（必需）
	MaxIterations int                 // 最大迭代次数（默认50）
	SystemPrompt  string              // 自定义系统提示词（可选）
	PromptType    SystemPromptType    // 提示词类型（默认orchestrator）

	// 校验和修正组件配置
	Reviewer        *Reviewer            // 审查器（可选，会自动创建）
	Verifier        *Verifier            // 校验器（可选，会自动创建）
	VerifierConfig  *VerifyConfig        // 校验器配置（可选）
	CorrectorConfig *SelfCorrectorConfig // 修正器配置（可选）
}

// NewAgent 创建新的Agent实例
func NewAgent(config *Config) (*Agent, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
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

	// 设置默认值
	maxIterations := config.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 50
	}

	promptType := config.PromptType
	if promptType == "" {
		promptType = PromptTypeOrchestrator
	}

	// 获取系统提示词
	systemPrompt := config.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = GetSystemPrompt(promptType)
	}

	// 初始化审查器
	reviewer := config.Reviewer
	if reviewer == nil {
		reviewer = NewReviewer(nil) // 使用默认配置
	}

	// 初始化校验器
	verifier := config.Verifier
	if verifier == nil {
		verifier = NewVerifier(config.VerifierConfig)
	}

	// 初始化自我修正器
	corrector := NewSelfCorrector(config.CorrectorConfig)

	return &Agent{
		modelClient:   config.ModelClient,
		toolRegistry:  config.ToolRegistry,
		stateManager:  config.StateManager,
		executor:      NewToolExecutor(config.ToolRegistry),
		history:       NewHistoryManager(),
		reviewer:      reviewer,
		verifier:      verifier,
		corrector:     corrector,
		maxIterations: maxIterations,
		systemPrompt:  systemPrompt,
		promptType:    promptType,
	}, nil
}

// Run 执行任务主循环
// ctx: 上下文，用于取消任务
// taskGoal: 任务目标描述
// 返回执行结果和错误
func (a *Agent) Run(ctx context.Context, taskGoal string) (*RunResult, error) {
	// 检查是否已在运行
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return nil, fmt.Errorf("agent is already running")
	}
	a.running = true
	a.mu.Unlock()

	// 确保运行状态被重置
	defer func() {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
	}()

	// 创建可取消的上下文
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	a.mu.Lock()
	a.cancelFunc = cancel
	a.mu.Unlock()

	// 生成任务ID
	if a.taskID == "" {
		a.taskID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}

	// 创建任务状态
	_, err := a.stateManager.CreateTask(a.taskID, taskGoal)
	if err != nil {
		// 如果任务已存在，尝试加载
		_, err = a.stateManager.Load(a.taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to create or load task: %w", err)
		}
	}

	// 更新任务状态为进行中
	if err := a.stateManager.UpdateTaskStatus(a.taskID, "in_progress"); err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// 初始化消息历史
	a.history.Clear()
	a.history.AddMessage(model.NewSystemMessage(a.systemPrompt))

	// 构建初始任务提示
	stateSummary, _ := a.stateManager.GetYAMLSummary(a.taskID)
	taskPrompt := BuildTaskPrompt(taskGoal, stateSummary)
	a.history.AddMessage(model.NewUserMessage(taskPrompt))

	// 主循环
	result := &RunResult{
		TaskID:     a.taskID,
		StartTime:  time.Now(),
		Iterations: 0,
	}

	for iteration := 1; iteration <= a.maxIterations; iteration++ {
		// 检查上下文是否已取消
		select {
		case <-runCtx.Done():
			result.Status = "cancelled"
			result.EndTime = time.Now()
			return result, runCtx.Err()
		default:
		}

		// 迭代回调
		if a.onIteration != nil {
			a.onIteration(iteration)
		}

		// 检查上下文是否超出限制
		if a.modelClient.IsContextOverflow(a.history.GetMessagesRef()) {
			// 截断历史
			config := a.modelClient.GetConfig()
			a.history.Truncate(int(float64(config.ContextSize)*0.8), a.modelClient.CountTokens)
		}

		// 发送消息给模型（使用流式响应）
		response, err := a.callModel(runCtx)
		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			result.EndTime = time.Now()
			a.stateManager.UpdateTaskStatus(a.taskID, "failed")
			return result, fmt.Errorf("model call failed: %w", err)
		}

		// 添加助手消息到历史
		assistantMsg := model.Message{
			Role:      model.RoleAssistant,
			Content:   response.Content,
			ToolCalls: response.ToolCalls,
		}
		a.history.AddMessage(assistantMsg)

		// 审查模型输出（在工具调用前）
		taskState, _ := a.stateManager.GetState(a.taskID)
		reviewResult := a.reviewer.ReviewOutput(response.Content, response.ToolCalls, taskState)

		// 如果审查结果被阻断，生成修正指令并注入历史
		if reviewResult.IsBlocked() {
			// 分析失败并生成修正指令
			correctionResult := a.corrector.AnalyzeFailure(nil, reviewResult)
			correctionInstruction := a.corrector.GenerateCorrectionInstruction(correctionResult)

			// 检查是否还有重试机会
			if !a.corrector.ShouldRetry() {
				// 重试次数耗尽，任务失败
				result.Status = "failed"
				result.Error = fmt.Sprintf("审查阻断且重试次数耗尽: %s", reviewResult.Summary)
				result.EndTime = time.Now()
				a.stateManager.UpdateTaskStatus(a.taskID, "failed")
				return result, fmt.Errorf("review blocked and retries exhausted: %s", reviewResult.Summary)
			}

			// 注入修正指令到历史
			a.history.AddMessage(model.NewUserMessage(correctionInstruction))
			result.Iterations = iteration
			continue
		}

		// 检查是否有工具调用
		if len(response.ToolCalls) > 0 {
			// 执行工具调用
			toolResults, err := a.executeTools(runCtx, response.ToolCalls)
			if err != nil {
				result.Status = "failed"
				result.Error = err.Error()
				result.EndTime = time.Now()
				a.stateManager.UpdateTaskStatus(a.taskID, "failed")
				return result, fmt.Errorf("tool execution failed: %w", err)
			}

			// 添加工具结果到历史
			a.history.AddMessages(toolResults)

			// 检查是否有任务完成/失败的工具调用
			for _, tc := range response.ToolCalls {
				if tc.Function.Name == "complete_task" {
					// 在完成任务前进行校验
					verifyResult := a.verifyTaskCompletion(runCtx, taskGoal)

					if verifyResult != nil && verifyResult.IsFailed() {
						// 校验失败，记录失败并生成修正指令
						correctionResult := a.corrector.AnalyzeFailure(verifyResult, nil)
						correctionInstruction := a.corrector.GenerateCorrectionInstruction(correctionResult)

						// 检查是否还有重试机会
						if !a.corrector.ShouldRetry() {
							// 重试次数耗尽，任务失败
							result.Status = "failed"
							result.Error = fmt.Sprintf("任务完成校验失败且重试次数耗尽: %s", verifyResult.Summary)
							result.EndTime = time.Now()
							a.stateManager.UpdateTaskStatus(a.taskID, "failed")
							return result, fmt.Errorf("verification failed and retries exhausted: %s", verifyResult.Summary)
						}

						// 注入修正指令到历史，继续执行
						a.history.AddMessage(model.NewUserMessage(correctionInstruction))
						result.Iterations = iteration
						continue
					}

					// 校验通过，标记任务完成
					result.Status = "completed"
					result.EndTime = time.Now()
					a.stateManager.UpdateTaskStatus(a.taskID, "completed")
					return result, nil
				}
				if tc.Function.Name == "fail_task" {
					result.Status = "failed"
					result.EndTime = time.Now()
					a.stateManager.UpdateTaskStatus(a.taskID, "failed")
					return result, nil
				}
			}

			result.Iterations = iteration
			continue
		}

		// 没有工具调用，检查是否完成
		// 如果响应中包含完成标志，则结束
		if a.isTaskComplete(response.Content) {
			// 在完成任务前进行校验
			verifyResult := a.verifyTaskCompletion(runCtx, taskGoal)

			if verifyResult != nil && verifyResult.IsFailed() {
				// 校验失败，记录失败并生成修正指令
				correctionResult := a.corrector.AnalyzeFailure(verifyResult, nil)
				correctionInstruction := a.corrector.GenerateCorrectionInstruction(correctionResult)

				// 检查是否还有重试机会
				if !a.corrector.ShouldRetry() {
					// 重试次数耗尽，任务失败
					result.Status = "failed"
					result.Error = fmt.Sprintf("任务完成校验失败且重试次数耗尽: %s", verifyResult.Summary)
					result.EndTime = time.Now()
					a.stateManager.UpdateTaskStatus(a.taskID, "failed")
					return result, fmt.Errorf("verification failed and retries exhausted: %s", verifyResult.Summary)
				}

				// 注入修正指令到历史，继续执行
				a.history.AddMessage(model.NewUserMessage(correctionInstruction))
				result.Iterations = iteration
				continue
			}

			result.Status = "completed"
			result.EndTime = time.Now()
			result.FinalResponse = response.Content
			a.stateManager.UpdateTaskStatus(a.taskID, "completed")
			return result, nil
		}

		// 如果没有工具调用且没有完成标志，可能是模型在等待更多输入
		// 添加一个提示让模型继续
		a.history.AddMessage(model.NewUserMessage("请继续执行任务。如果任务已完成，请使用complete_task工具标记完成。"))

		result.Iterations = iteration
	}

	// 达到最大迭代次数
	result.Status = "max_iterations"
	result.EndTime = time.Now()
	a.stateManager.UpdateTaskStatus(a.taskID, "failed")
	return result, fmt.Errorf("reached maximum iterations (%d)", a.maxIterations)
}

// RunResult 运行结果
type RunResult struct {
	TaskID        string        // 任务ID
	Status        string        // 状态：completed, failed, cancelled, max_iterations
	StartTime     time.Time     // 开始时间
	EndTime       time.Time     // 结束时间
	Duration      time.Duration // 执行时长
	Iterations    int           // 迭代次数
	FinalResponse string        // 最终响应内容
	Error         string        // 错误信息
}

// callModel 调用模型
func (a *Agent) callModel(ctx context.Context) (*modelResponse, error) {
	// 获取工具Schema
	toolSchemas := a.executor.GetToolSchemas()

	// 获取消息历史
	messages := a.history.GetMessagesRef()

	// 使用流式响应
	streamChan, err := a.modelClient.StreamChatCompletion(ctx, messages, toolSchemas)
	if err != nil {
		return nil, err
	}

	// 收集流式响应
	var contentBuilder strings.Builder
	var toolCalls []model.ToolCall
	var currentToolCall *model.ToolCall

	for event := range streamChan {
		if event.Error != nil {
			return nil, event.Error
		}

		if event.Done {
			break
		}

		if event.Response == nil || len(event.Response.Choices) == 0 {
			continue
		}

		choice := event.Response.Choices[0]
		if choice.Delta == nil {
			continue
		}

		// 处理内容
		if choice.Delta.Content != "" {
			contentBuilder.WriteString(choice.Delta.Content)

			// 流式输出回调
			if a.onStreamChunk != nil {
				a.onStreamChunk(choice.Delta.Content)
			}
		}

		// 处理工具调用
		if len(choice.Delta.ToolCalls) > 0 {
			for _, tc := range choice.Delta.ToolCalls {
				// 新的工具调用
				if tc.ID != "" {
					if currentToolCall != nil {
						toolCalls = append(toolCalls, *currentToolCall)
					}
					currentToolCall = &model.ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
						Function: model.FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				} else if currentToolCall != nil {
					// 追加参数
					currentToolCall.Function.Arguments += tc.Function.Arguments
				}
			}
		}
	}

	// 添加最后一个工具调用
	if currentToolCall != nil {
		toolCalls = append(toolCalls, *currentToolCall)
	}

	return &modelResponse{
		Content:   contentBuilder.String(),
		ToolCalls: toolCalls,
	}, nil
}

// modelResponse 模型响应
type modelResponse struct {
	Content   string
	ToolCalls []model.ToolCall
}

// executeTools 执行工具调用
func (a *Agent) executeTools(ctx context.Context, toolCalls []model.ToolCall) ([]model.Message, error) {
	results := make([]model.Message, 0, len(toolCalls))

	for _, tc := range toolCalls {
		// 工具调用回调
		if a.onToolCall != nil {
			a.onToolCall(tc.Function.Name, tc.Function.Arguments)
		}

		// 执行工具
		result, err := a.executor.ExecuteToolCalls(ctx, []model.ToolCall{tc})
		if err != nil {
			return nil, err
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

// isTaskComplete 检查任务是否完成
func (a *Agent) isTaskComplete(content string) bool {
	// 检查是否包含完成标志
	lowerContent := strings.ToLower(content)
	completeMarkers := []string{
		"任务已完成",
		"task completed",
		"任务完成",
		"all done",
	}

	for _, marker := range completeMarkers {
		if strings.Contains(lowerContent, marker) {
			return true
		}
	}

	return false
}

// Stop 停止Agent运行
func (a *Agent) Stop() {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.cancelFunc != nil {
		a.cancelFunc()
	}
}

// IsRunning 检查Agent是否正在运行
func (a *Agent) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

// SetTaskID 设置任务ID
func (a *Agent) SetTaskID(taskID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.taskID = taskID
}

// GetTaskID 获取当前任务ID
func (a *Agent) GetTaskID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.taskID
}

// SetOnStreamChunk 设置流式输出回调
func (a *Agent) SetOnStreamChunk(callback func(chunk string)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onStreamChunk = callback
}

// SetOnToolCall 设置工具调用回调
func (a *Agent) SetOnToolCall(callback func(name, args string)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onToolCall = callback
}

// SetOnIteration 设置迭代回调
func (a *Agent) SetOnIteration(callback func(iteration int)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onIteration = callback
}

// GetHistory 获取消息历史管理器
func (a *Agent) GetHistory() *HistoryManager {
	return a.history
}

// GetState 获取当前任务状态
func (a *Agent) GetState() (*state.TaskState, error) {
	return a.stateManager.GetState(a.taskID)
}

// RunSync 同步执行任务（简化接口）
func (a *Agent) RunSync(ctx context.Context, taskGoal string) error {
	result, err := a.Run(ctx, taskGoal)
	if err != nil {
		return err
	}

	if result.Status != "completed" {
		if result.Error != "" {
			return fmt.Errorf("task failed: %s", result.Error)
		}
		return fmt.Errorf("task ended with status: %s", result.Status)
	}

	return nil
}

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

// SetReviewer 设置审查器
func (a *Agent) SetReviewer(r *Reviewer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.reviewer = r
}

// SetVerifier 设置校验器
func (a *Agent) SetVerifier(v *Verifier) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.verifier = v
}

// SetCorrector 设置自我修正器
func (a *Agent) SetCorrector(c *SelfCorrector) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.corrector = c
}

// GetReviewer 获取审查器
func (a *Agent) GetReviewer() *Reviewer {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.reviewer
}

// GetVerifier 获取校验器
func (a *Agent) GetVerifier() *Verifier {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.verifier
}

// GetCorrector 获取自我修正器
func (a *Agent) GetCorrector() *SelfCorrector {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.corrector
}
