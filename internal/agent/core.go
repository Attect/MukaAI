package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Attect/MukaAI/internal/model"
	"github.com/Attect/MukaAI/internal/state"
	"github.com/Attect/MukaAI/internal/tools"
)

// ContextInjector 代码上下文注入器接口
// 用于在模型调用前自动注入与任务相关的文件上下文
type ContextInjector interface {
	// InjectContext 根据任务描述注入上下文到消息历史中
	// 返回注入后的消息列表和注入的文件路径列表
	InjectContext(taskDesc string, messages []model.Message) []model.Message
}

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

	// 重复检测组件
	repetitionDetector *RepetitionDetector // 模型输出重复检测器
	repetitionRetries  int                 // 当前重复重试次数（独立于主迭代计数）

	// 上下文压缩组件
	compressor *Compressor // 上下文压缩器（智能压缩替代粗暴截断）

	// 监督组件
	supervisor Supervisor // Agent监督器（可选，通过接口解耦）

	// 上下文感知组件
	contextInjector ContextInjector // 代码上下文注入器（可选，自动提取相关文件）

	// 日志记录器
	logger *AgentLogger // 运行日志记录器

	// 配置
	maxIterations int              // 最大迭代次数
	systemPrompt  string           // 系统提示词
	taskID        string           // 当前任务ID
	promptType    SystemPromptType // 提示词类型
	workDir       string           // 实际工作目录，用于注入到提示词

	// 状态
	mu         sync.RWMutex
	running    bool
	cancelFunc context.CancelFunc

	// Supervisor冷却
	lastCorrectionIteration int // 上次注入修正指令的迭代号，用于冷却期控制

	// 回调
	onStreamChunk  func(chunk string)                  // 流式输出回调
	onToolCall     func(name, args string)             // 工具调用回调（即将废弃）
	onToolResult   func(name, resultJSON string)       // 工具执行结果回调
	onToolCallFull func(toolCallID, name, args string) // 工具调用完整回调
	onReview       func(status string, summary string) // 审查结果回调
	onVerify       func(status string, summary string) // 校验结果回调
	onCorrection   func(instruction string)            // 修正指令回调
	onNoToolCall   func(count int, response string)    // 无工具调用回调
	onHistoryAdd   func(role, content string)          // 消息历史添加回调
	onIteration    func(iteration int)                 // 迭代回调
	onThinking     func(thinking string)               // 思考内容回调
	onSupervisor   func(result *SupervisionResult)     // 监督结果回调

	// 流式处理器
	streamHandler StreamHandler // 流式消息处理器
}

// Config Agent配置
type Config struct {
	ModelClient   *model.Client       // 模型客户端（必需）
	ToolRegistry  *tools.ToolRegistry // 工具注册中心（必需）
	StateManager  *state.StateManager // 状态管理器（必需）
	MaxIterations int                 // 最大迭代次数（默认50）
	SystemPrompt  string              // 自定义系统提示词（可选）
	PromptType    SystemPromptType    // 提示词类型（默认orchestrator）
	WorkDir       string              // 实际工作目录，用于注入到提示词（可选，为空时回退到os.Getwd()）

	// 校验和修正组件配置
	Reviewer        *Reviewer            // 审查器（可选，会自动创建）
	Verifier        *Verifier            // 校验器（可选，会自动创建）
	VerifierConfig  *VerifyConfig        // 校验器配置（可选）
	CorrectorConfig *SelfCorrectorConfig // 修正器配置（可选）

	// 监督组件配置
	Supervisor Supervisor // Agent监督器（可选，nil则不启用监督检查）

	// 上下文感知组件配置
	ContextInjector ContextInjector // 代码上下文注入器（可选，nil则不启用自动上下文注入）

	// 日志配置
	LogPath string // 日志文件路径（可选，为空则不记录日志）
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

	// 设置审查器的工作目录（用于文件存在性检查）
	if config.WorkDir != "" {
		reviewer.SetWorkDir(config.WorkDir)
	}

	// 初始化校验器
	verifier := config.Verifier
	if verifier == nil {
		verifier = NewVerifier(config.VerifierConfig)
	}

	// 初始化自我修正器
	corrector := NewSelfCorrector(config.CorrectorConfig)

	// 初始化重复检测器
	repetitionDetector := NewRepetitionDetector(DefaultRepetitionConfig())

	// 初始化上下文压缩器
	compressor, _ := NewCompressor(config.ModelClient, DefaultCompressorConfig())
	// 注意：NewCompressor失败时compressor为nil，preIteration会回退到Truncate

	// 初始化日志记录器
	logger, err := NewAgentLogger(config.LogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &Agent{
		modelClient:        config.ModelClient,
		toolRegistry:       config.ToolRegistry,
		stateManager:       config.StateManager,
		executor:           NewToolExecutor(config.ToolRegistry),
		history:            NewHistoryManager(),
		reviewer:           reviewer,
		verifier:           verifier,
		corrector:          corrector,
		supervisor:         config.Supervisor,
		contextInjector:    config.ContextInjector,
		logger:             logger,
		maxIterations:      maxIterations,
		systemPrompt:       systemPrompt,
		promptType:         promptType,
		workDir:            config.WorkDir,
		repetitionDetector: repetitionDetector,
		compressor:         compressor,
	}, nil
}

// Run 执行任务主循环
// ctx: 上下文，用于取消任务
// taskGoal: 任务目标描述
// 返回执行结果和错误
//
// 主循环结构：外层循环支持强制校验后的重试，内层循环执行迭代
// 每次迭代：调用模型 → 审查输出 → 执行工具/处理响应 → 校验完成
func (a *Agent) Run(ctx context.Context, taskGoal string) (*RunResult, error) {
	// 初始化运行上下文（状态锁、取消上下文、任务ID、日志、状态管理）
	runCtx, result, totalIterations, maxTotalIterations, cleanup, err := a.initRunContext(ctx, taskGoal)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// 校验状态局部变量（仅在Run内使用，避免并发安全隐患）
	verificationPassed := false
	consecutiveNoToolCalls := 0

	// 使用外层循环支持强制校验后的重试
	for {
		// 内层主循环
	innerLoop:
		for {
			totalIterations++

			// 前置检查：最大迭代次数、上下文取消
			if totalIterations > maxTotalIterations {
				return a.handleMaxIterations(result, totalIterations, maxTotalIterations)
			}
			if cancelled, ret, err := a.checkCancellation(runCtx, result, totalIterations); cancelled {
				return ret, err
			}

			// 迭代前处理：回调、日志、上下文截断
			a.preIteration(totalIterations)

			// 调用模型（含可重试错误的自动重试）
			response, err := a.callModelWithRetry(runCtx, totalIterations)
			if err != nil {
				return a.handleModelCallError(result, err, totalIterations)
			}

			// 重复检测处理：当模型输出被检测为重复时，不记录该响应到历史
			// 而是注入防重复提示并重试
			if response.RepetitionDetected {
				ir := a.handleRepetition(response, totalIterations)
				if ir != nil {
					switch ir.action {
					case "return":
						return ir.result, ir.err
					case "continue":
						continue
					}
				}
				// 重试失败兜底，跳过本次迭代
				continue
			}

			// 模型正常返回（无重复），重置重复重试计数
			a.repetitionRetries = 0

			// 跳过空响应：当模型只返回thinking而没有任何content和tool_calls时，
			// 不记录到对话历史，因为空的assistant消息会导致某些LLM服务端（如llama.cpp）
			// 返回400错误（Jinja模板校验失败）
			if response.Content == "" && len(response.ToolCalls) == 0 {
				if a.logger != nil {
					a.logger.LogMessage("system", "[模型返回空响应（仅thinking），跳过记录]")
				}
				continue
			}

			// 记录模型响应到历史
			a.recordModelResponse(response)

			// 审查模型输出，处理审查阻断
			reviewResult := a.reviewModelOutput(response)
			if reviewResult.IsBlocked() {
				ret, err := a.handleReviewBlock(reviewResult, result, totalIterations)
				if ret != nil || err != nil {
					return ret, err
				}
				continue // 审查阻断但可重试，继续内层循环
			}

			// 根据是否有工具调用，分别处理
			if len(response.ToolCalls) > 0 {
				consecutiveNoToolCalls = 0
				ir := a.handleToolCallsIteration(runCtx, response, result, taskGoal, totalIterations)
				if ir != nil {
					switch ir.action {
					case "return":
						return ir.result, ir.err
					case "break":
						break innerLoop // 跳出内层for循环
					case "continue":
						continue
					}
				}
				// 兜底：如果任务已完成，跳出内层循环
				if result.Status == "completed" {
					break innerLoop
				}
				continue
			}

			// 无工具调用处理
			consecutiveNoToolCalls++
			if a.onNoToolCall != nil {
				a.onNoToolCall(consecutiveNoToolCalls, response.Content)
			}

			ir := a.handleNoToolCallIteration(runCtx, response, result, taskGoal, totalIterations, consecutiveNoToolCalls)
			if ir != nil {
				switch ir.action {
				case "return":
					return ir.result, ir.err
				case "break":
					break innerLoop // 跳出内层for循环
				case "continue":
					continue
				}
			}
		}

		// 外层循环：处理任务完成后的强制校验
		if result.Status == "completed" {
			if !verificationPassed {
				ir := a.handleForcedVerification(runCtx, taskGoal, result)
				if ir != nil {
					if ir.action == "return" {
						return ir.result, ir.err
					}
					// action == "continue"：强制校验失败但可重试
					continue
				}
				verificationPassed = true
			}

			// 强制校验通过，真正完成任务
			a.stateManager.UpdateTaskStatus(a.taskID, "completed")
			if a.logger != nil {
				a.logger.LogTaskEnd("completed", result.Iterations, result.EndTime.Sub(result.StartTime))
			}
			a.finalizeResult(result)
			return result, nil
		}

		// 如果任务失败或取消，直接返回
		if result.Status == "failed" || result.Status == "cancelled" {
			a.finalizeResult(result)
			return result, nil
		}

		// 其他状态视为未完成，按最大迭代次数处理
		return a.handleMaxIterations(result, totalIterations+1, maxTotalIterations)
	}
}

// initRunContext 初始化运行上下文
// 返回 runCtx、result、totalIterations、maxTotalIterations、cleanup函数、error
func (a *Agent) initRunContext(ctx context.Context, taskGoal string) (context.Context, *RunResult, int, int, func(), error) {
	// 检查是否已在运行
	// 注意：SendMessage会预先设置running=true，此时cancelFunc为nil，
	// 可与"另一个Run正在执行"（cancelFunc非nil）区分
	a.mu.Lock()
	if a.running {
		if a.cancelFunc != nil {
			// cancelFunc已设置，说明另一个Run正在执行中
			a.mu.Unlock()
			return nil, nil, 0, 0, nil, fmt.Errorf("agent is already running")
		}
		// running=true但cancelFunc为nil：由SendMessage预设，允许通过
	} else {
		a.running = true
	}
	a.mu.Unlock()

	cleanup := func() {
		a.mu.Lock()
		a.running = false
		a.cancelFunc = nil
		a.mu.Unlock()
	}

	// 创建可取消的上下文
	runCtx, cancel := context.WithCancel(ctx)

	a.mu.Lock()
	a.cancelFunc = cancel
	a.mu.Unlock()

	// 生成任务ID
	if a.taskID == "" {
		a.taskID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}

	// 设置日志记录器的任务ID
	if a.logger != nil {
		a.logger.SetTaskID(a.taskID)
		a.logger.LogTaskStart(taskGoal)
	}

	// 创建任务状态
	_, err := a.stateManager.CreateTask(a.taskID, taskGoal)
	if err != nil {
		_, err = a.stateManager.Load(a.taskID)
		if err != nil {
			cancel()
			cleanup()
			return nil, nil, 0, 0, nil, fmt.Errorf("failed to create or load task: %w", err)
		}
	}

	// 更新任务状态为进行中
	if err := a.stateManager.UpdateTaskStatus(a.taskID, "in_progress"); err != nil {
		cancel()
		cleanup()
		return nil, nil, 0, 0, nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// 初始化消息历史
	a.history.Clear()
	a.history.AddMessage(model.NewSystemMessage(a.systemPrompt))

	// 自动注入代码上下文（在系统消息之后、用户消息之前）
	if a.contextInjector != nil {
		contextMessages := a.contextInjector.InjectContext(taskGoal, a.history.GetMessages())
		// 替换历史中的消息为注入后的消息
		a.history.Clear()
		a.history.AddMessages(contextMessages)

		if a.logger != nil {
			a.logger.LogMessage("system", "[自动上下文注入完成]")
		}
	}

	// 构建初始任务提示
	stateSummary, _ := a.stateManager.GetYAMLSummary(a.taskID)
	taskPrompt := BuildTaskPrompt(taskGoal, stateSummary, a.workDir)
	a.history.AddMessage(model.NewUserMessage(taskPrompt))

	result := &RunResult{
		TaskID:    a.taskID,
		StartTime: time.Now(),
	}

	return runCtx, result, 0, a.maxIterations, func() {
		cancel()
		cleanup()
	}, nil
}

// checkCancellation 检查上下文是否已取消
// 返回 (是否取消, 结果, 错误)
func (a *Agent) checkCancellation(runCtx context.Context, result *RunResult, totalIterations int) (bool, *RunResult, error) {
	select {
	case <-runCtx.Done():
		result.Status = "cancelled"
		result.EndTime = time.Now()
		result.Iterations = totalIterations - 1
		if a.logger != nil {
			a.logger.LogTaskEnd("cancelled", result.Iterations, result.EndTime.Sub(result.StartTime))
		}
		a.finalizeResult(result)
		return true, result, runCtx.Err()
	default:
		return false, nil, nil
	}
}

// preIteration 迭代前处理：回调、日志、上下文压缩
func (a *Agent) preIteration(totalIterations int) {
	if a.onIteration != nil {
		a.onIteration(totalIterations)
	}
	if a.logger != nil {
		a.logger.LogIteration(totalIterations, "processing")
	}

	// 上下文管理：优先使用智能压缩，回退到简单截断
	messages := a.history.GetMessagesRef()
	if len(messages) == 0 {
		return
	}

	if a.compressor != nil {
		// 使用Compressor进行智能压缩
		shouldCompress, usageRatio := a.compressor.ShouldCompress(messages)
		if shouldCompress {
			taskState, _ := a.stateManager.GetState(a.taskID)
			result, err := a.compressor.Compress(messages, taskState)
			if err == nil && result.WasCompressed {
				// 压缩成功，替换历史消息
				a.history.Clear()
				a.history.AddMessages(result.CompressedMessages)

				if a.logger != nil {
					a.logger.LogMessage("system", fmt.Sprintf(
						"[上下文压缩] 使用率: %.1f%%, 消息: %d→%d, token: %d→%d, 压缩比: %.1f%%, 摘要长度: %d",
						usageRatio*100,
						result.OriginalCount, result.CompressedCount,
						result.OriginalTokens, result.CompressedTokens,
						result.CompressionRatio*100,
						len(result.Summary),
					))
				}
			} else if err != nil {
				// 压缩失败，回退到Truncate
				if a.logger != nil {
					a.logger.LogError(fmt.Sprintf("[上下文压缩] 压缩失败，回退到截断: %s", err.Error()))
				}
				config := a.modelClient.GetConfig()
				a.history.Truncate(int(float64(config.ContextSize)*0.8), a.modelClient.CountTokens)
			}
		}
	} else {
		// 没有Compressor，使用简单截断（向后兼容）
		if a.modelClient.IsContextOverflow(messages) {
			config := a.modelClient.GetConfig()
			a.history.Truncate(int(float64(config.ContextSize)*0.8), a.modelClient.CountTokens)
		}
	}
}

// handleModelCallError 处理模型调用错误
func (a *Agent) handleModelCallError(result *RunResult, err error, totalIterations int) (*RunResult, error) {
	result.Status = "failed"
	result.Error = err.Error()
	result.EndTime = time.Now()
	a.stateManager.UpdateTaskStatus(a.taskID, "failed")
	if a.logger != nil {
		a.logger.LogError(fmt.Sprintf("模型调用失败: %s", err.Error()))
		a.logger.LogTaskEnd("failed", totalIterations, result.EndTime.Sub(result.StartTime))
	}
	a.finalizeResult(result)
	return result, fmt.Errorf("model call failed: %w", err)
}

// callModelWithRetry 调用模型，对可重试的错误自动重试
// 最多重试3次，递增间隔（5s, 15s, 30s）
// 不可重试的错误（4xx客户端错误等）直接返回
func (a *Agent) callModelWithRetry(ctx context.Context, iteration int) (*modelResponse, error) {
	maxRetries := 3
	retryDelays := []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		response, err := a.callModel(ctx)
		if err == nil {
			return response, nil
		}

		// 检查是否可重试
		if !a.isRetryableError(err) {
			return nil, err
		}

		// 已达到最大重试次数
		if attempt >= maxRetries {
			if a.logger != nil {
				a.logger.LogError(fmt.Sprintf("模型调用重试耗尽（%d次）: %s", maxRetries, err.Error()))
			}
			return nil, err
		}

		// 等待后重试
		delay := retryDelays[attempt]
		if a.logger != nil {
			a.logger.LogError(fmt.Sprintf("模型调用失败（可重试），%v后第%d次重试: %s", delay, attempt+1, err.Error()))
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// 继续重试
		}
	}

	return nil, fmt.Errorf("unreachable")
}

// isRetryableError 判断错误是否为可重试的临时性错误
// 可重试：网络断开、连接拒绝、5xx服务器错误、429限流
// 不可重试：400客户端错误、401认证失败、403禁止等
func (a *Agent) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// 网络和连接类错误（可重试）
	retryablePatterns := []string{
		"wsarecv:", // Windows socket recv error
		"wsasend:", // Windows socket send error
		"connection refused",
		"connection reset",
		"forcibly closed", // connection forcibly closed
		"broken pipe",
		"i/o timeout",
		"EOF",           // unexpected EOF
		"no such host",  // DNS failure
		"TLS handshake", // TLS issues
	}
	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	// API错误：检查状态码
	var apiErr *model.APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsRetryable()
	}

	// RequestError可能包装了可重试的底层错误，递归检查
	var reqErr *model.RequestError
	if errors.As(err, &reqErr) && reqErr.Err != nil {
		return a.isRetryableError(reqErr.Err)
	}

	return false
}

// recordModelResponse 记录模型响应到历史
func (a *Agent) recordModelResponse(response *modelResponse) {
	assistantMsg := model.Message{
		Role:      model.RoleAssistant,
		Content:   response.Content,
		ToolCalls: response.ToolCalls,
	}
	a.history.AddMessage(assistantMsg)

	if a.onHistoryAdd != nil {
		a.onHistoryAdd("assistant", response.Content)
	}
	if a.logger != nil {
		a.logger.LogMessage("assistant", response.Content)
	}
}

// reviewModelOutput 审查模型输出并触发回调
func (a *Agent) reviewModelOutput(response *modelResponse) *ReviewResult {
	taskState, _ := a.stateManager.GetState(a.taskID)
	reviewResult := a.reviewer.ReviewOutput(response.Content, response.ToolCalls, taskState)

	if a.onReview != nil {
		a.onReview(string(reviewResult.Status), reviewResult.Summary)
	}
	if a.logger != nil {
		a.logger.LogReview(reviewResult)
	}

	return reviewResult
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

// finalizeResult 计算并填充RunResult的Duration字段
func (a *Agent) finalizeResult(result *RunResult) {
	if !result.EndTime.IsZero() && !result.StartTime.IsZero() {
		result.Duration = result.EndTime.Sub(result.StartTime)
	}
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
		"已经完成",
		"完成了",
		"finished",
		"done",
		"任务结束",
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

// SendMessage 发送用户消息并启动推理
// 这是一个异步方法，会在后台启动推理过程
func (a *Agent) SendMessage(content string) error {
	// 检查是否已在运行，并将running=true设在同一临界区内，消除TOCTOU竞态
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return fmt.Errorf("agent is already running")
	}
	a.running = true
	a.mu.Unlock()

	// 在后台启动推理
	go func() {
		// goroutine结束时重置running标志（持锁保护）
		defer func() {
			a.mu.Lock()
			a.running = false
			a.mu.Unlock()
		}()

		ctx := context.Background()
		_, err := a.Run(ctx, content)
		if err != nil {
			// 如果有流式处理器，发送错误消息
			if handler := a.GetStreamHandler(); handler != nil {
				handler.OnError(err)
			}
		}
		// 无论成功还是失败，都发送一个任务完成信号
		// 使用 OnTaskDone 通知 GUI 推理已完全结束
		if handler := a.GetStreamHandler(); handler != nil {
			handler.OnTaskDone()
		}
	}()

	return nil
}

// GetStreamHandler 获取流式消息处理器
func (a *Agent) GetStreamHandler() StreamHandler {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.streamHandler
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

// GetLogger 获取日志记录器
func (a *Agent) GetLogger() *AgentLogger {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.logger
}

// Close 关闭Agent，释放资源
func (a *Agent) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 关闭日志记录器
	if a.logger != nil {
		return a.logger.Close()
	}

	return nil
}

// handleRepetition 处理模型输出重复的情况
// 当callModel检测到重复输出时调用此方法进行重试决策。
// 重试时不记录重复的响应到对话历史，而是注入防重复提示。
// 返回nil表示应continue（继续迭代），非nil包含具体的处理结果。
func (a *Agent) handleRepetition(response *modelResponse, totalIterations int) *iterationResult {
	a.repetitionRetries++

	maxRetries := 3 // 默认值
	if a.repetitionDetector != nil {
		maxRetries = a.repetitionDetector.GetConfig().MaxRetries
	}

	if a.logger != nil {
		a.logger.LogError(fmt.Sprintf("检测到模型输出重复，模式: %q, 重试次数: %d/%d",
			response.RepetitionPattern, a.repetitionRetries, maxRetries))
	}

	// 超过最大重试次数，放弃重试
	if a.repetitionRetries > maxRetries {
		a.repetitionRetries = 0
		if a.logger != nil {
			a.logger.LogError("重复重试次数耗尽，跳过本次迭代")
		}
		return &iterationResult{action: "continue"}
	}

	// 不记录重复的响应到历史（callModel返回空Content，recordModelResponse不会被调用）
	// 注入防重复提示到对话历史，引导模型改变输出
	retryPrompt := a.repetitionDetector.BuildRetryPrompt(response.RepetitionPattern, a.repetitionRetries)
	a.history.AddMessage(model.NewUserMessage(retryPrompt))

	if a.onHistoryAdd != nil {
		a.onHistoryAdd("user", retryPrompt)
	}
	if a.onCorrection != nil {
		a.onCorrection(retryPrompt)
	}

	return &iterationResult{action: "continue"}
}
