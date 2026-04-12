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

	// 日志记录器
	logger *AgentLogger // 运行日志记录器

	// 配置
	maxIterations int              // 最大迭代次数
	systemPrompt  string           // 系统提示词
	taskID        string           // 当前任务ID
	promptType    SystemPromptType // 提示词类型

	// 状态
	mu         sync.RWMutex
	running    bool
	cancelFunc context.CancelFunc

	// 校验状态
	verificationPassed     bool // 是否通过强制校验
	consecutiveNoToolCalls int  // 连续无工具调用计数

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

	// 校验和修正组件配置
	Reviewer        *Reviewer            // 审查器（可选，会自动创建）
	Verifier        *Verifier            // 校验器（可选，会自动创建）
	VerifierConfig  *VerifyConfig        // 校验器配置（可选）
	CorrectorConfig *SelfCorrectorConfig // 修正器配置（可选）

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

	// 初始化校验器
	verifier := config.Verifier
	if verifier == nil {
		verifier = NewVerifier(config.VerifierConfig)
	}

	// 初始化自我修正器
	corrector := NewSelfCorrector(config.CorrectorConfig)

	// 初始化日志记录器
	logger, err := NewAgentLogger(config.LogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &Agent{
		modelClient:   config.ModelClient,
		toolRegistry:  config.ToolRegistry,
		stateManager:  config.StateManager,
		executor:      NewToolExecutor(config.ToolRegistry),
		history:       NewHistoryManager(),
		reviewer:      reviewer,
		verifier:      verifier,
		corrector:     corrector,
		logger:        logger,
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
	a.verificationPassed = false
	a.consecutiveNoToolCalls = 0
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

	// 设置日志记录器的任务ID
	if a.logger != nil {
		a.logger.SetTaskID(a.taskID)
		a.logger.LogTaskStart(taskGoal)
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

	// 使用全局迭代计数器，避免内外层循环迭代计数不共享导致总调用次数远超maxIterations
	totalIterations := 0
	maxTotalIterations := a.maxIterations

	// 使用外层循环支持强制校验后的重试
	for {
		// 内层主循环
		for {
			// 递增全局迭代计数器
			totalIterations++
			if totalIterations > maxTotalIterations {
				// 达到最大迭代次数
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

			// 检查上下文是否已取消
			select {
			case <-runCtx.Done():
				result.Status = "cancelled"
				result.EndTime = time.Now()
				result.Iterations = totalIterations - 1
				if a.logger != nil {
					a.logger.LogTaskEnd("cancelled", result.Iterations, result.EndTime.Sub(result.StartTime))
				}
				a.finalizeResult(result)
				return result, runCtx.Err()
			default:
			}

			// 迭代回调
			if a.onIteration != nil {
				a.onIteration(totalIterations)
			}

			// 记录迭代日志
			if a.logger != nil {
				a.logger.LogIteration(totalIterations, "processing")
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
				if a.logger != nil {
					a.logger.LogError(fmt.Sprintf("模型调用失败: %s", err.Error()))
					a.logger.LogTaskEnd("failed", totalIterations, result.EndTime.Sub(result.StartTime))
				}
				a.finalizeResult(result)
				return result, fmt.Errorf("model call failed: %w", err)
			}

			// 添加助手消息到历史
			assistantMsg := model.Message{
				Role:      model.RoleAssistant,
				Content:   response.Content,
				ToolCalls: response.ToolCalls,
			}
			a.history.AddMessage(assistantMsg)

			// 消息历史添加回调
			if a.onHistoryAdd != nil {
				a.onHistoryAdd("assistant", response.Content)
			}

			// 记录助手消息
			if a.logger != nil {
				a.logger.LogMessage("assistant", response.Content)
			}

			// 审查模型输出（在工具调用前）
			taskState, _ := a.stateManager.GetState(a.taskID)
			reviewResult := a.reviewer.ReviewOutput(response.Content, response.ToolCalls, taskState)

			// 审查结果回调
			if a.onReview != nil {
				a.onReview(string(reviewResult.Status), reviewResult.Summary)
			}

			// 记录审查结果
			if a.logger != nil {
				a.logger.LogReview(reviewResult)
			}

			// 如果审查结果被阻断，生成修正指令并注入历史
			if reviewResult.IsBlocked() {
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
				continue
			}

			// 检查是否有工具调用
			if len(response.ToolCalls) > 0 {
				// 有工具调用，重置连续无工具调用计数器
				a.consecutiveNoToolCalls = 0

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
					return result, fmt.Errorf("tool execution failed: %w", err)
				}

				// 添加工具结果到历史
				a.history.AddMessages(toolResults)

				// 检查是否有end_exploration工具调用
				for _, tc := range response.ToolCalls {
					if tc.Function.Name == "end_exploration" {
						// 声明探索阶段结束
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
								result.Error = fmt.Sprintf("任务完成校验失败且重试次数耗尽: %s", verifyResult.Summary)
								result.EndTime = time.Now()
								a.stateManager.UpdateTaskStatus(a.taskID, "failed")
								if a.logger != nil {
									a.logger.LogError(result.Error)
									a.logger.LogTaskEnd("failed", totalIterations, result.EndTime.Sub(result.StartTime))
								}
								a.finalizeResult(result)
								return result, fmt.Errorf("verification failed and retries exhausted: %s", verifyResult.Summary)
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
							continue
						}

						// 校验通过，标记任务完成（但不立即返回）
						result.Status = "completed"
						result.EndTime = time.Now()
						result.Iterations = totalIterations
						// 不设置verificationPassed，让外层循环执行强制校验
						// 跳出内层循环，进入外层循环的强制校验阶段
						break
					}
					if tc.Function.Name == "fail_task" {
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
				}

				// 如果任务已完成，跳出内层循环
				if result.Status == "completed" {
					break
				}

				result.Iterations = totalIterations
				continue
			}

			// 没有工具调用，递增连续无工具调用计数器
			a.consecutiveNoToolCalls++

			// 无工具调用回调
			if a.onNoToolCall != nil {
				a.onNoToolCall(a.consecutiveNoToolCalls, response.Content)
			}

			// 检查是否完成（通过文本内容判断）
			if a.isTaskComplete(response.Content) {
				// 在完成任务前进行校验
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
						result.Error = fmt.Sprintf("任务完成校验失败且重试次数耗尽: %s", verifyResult.Summary)
						result.EndTime = time.Now()
						a.stateManager.UpdateTaskStatus(a.taskID, "failed")
						if a.logger != nil {
							a.logger.LogError(result.Error)
							a.logger.LogTaskEnd("failed", totalIterations, result.EndTime.Sub(result.StartTime))
						}
						a.finalizeResult(result)
						return result, fmt.Errorf("verification failed and retries exhausted: %s", verifyResult.Summary)
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
					continue
				}

				// 校验通过，标记任务完成（但不立即返回）
				result.Status = "completed"
				result.EndTime = time.Now()
				result.FinalResponse = response.Content
				result.Iterations = totalIterations
				// 不设置verificationPassed，让外层循环执行强制校验
				// 跳出内层循环，进入外层循环的强制校验阶段
				break
			}

			// 连续无工具调用超过阈值，视为纯对话，直接完成
			if a.consecutiveNoToolCalls >= 3 {
				result.Status = "completed"
				result.EndTime = time.Now()
				result.FinalResponse = response.Content
				result.Iterations = totalIterations
				a.finalizeResult(result)
				break
			}

			// 前几次尝试时，给模型一个温和的提示
			promptMsg := "请根据上述内容继续执行。如果已经完成，请调用complete_task工具。"
			if a.onHistoryAdd != nil {
				a.onHistoryAdd("user", promptMsg)
			}
			a.history.AddMessage(model.NewUserMessage(promptMsg))
			result.Iterations = totalIterations
		}

		// 检查任务状态
		if result.Status == "completed" {
			// 强制校验：即使之前通过了校验，也要再次确认
			if !a.verificationPassed {
				// 执行强制校验
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
						return result, fmt.Errorf("forced verification failed and retries exhausted: %s", verifyResult.Summary)
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
					continue
				}

				// 强制校验通过
				a.verificationPassed = true
			}

			// 强制校验通过，真正完成任务
			a.stateManager.UpdateTaskStatus(a.taskID, "completed")
			if a.logger != nil {
				a.logger.LogTaskEnd("completed", result.Iterations, result.EndTime.Sub(result.StartTime))
			}
			a.finalizeResult(result)
			return result, nil
		}

		// 如果任务失败或达到最大迭代次数，直接返回
		if result.Status == "failed" || result.Status == "cancelled" {
			a.finalizeResult(result)
			return result, nil
		}

		// 如果内层循环结束但任务未完成，继续外层循环
		if result.Status == "" || result.Status == "in_progress" {
			// 达到最大迭代次数
			result.Status = "max_iterations"
			result.EndTime = time.Now()
			a.stateManager.UpdateTaskStatus(a.taskID, "failed")
			if a.logger != nil {
				a.logger.LogError(fmt.Sprintf("达到最大迭代次数 (%d)", maxTotalIterations))
				a.logger.LogTaskEnd("max_iterations", result.Iterations, result.EndTime.Sub(result.StartTime))
			}
			a.finalizeResult(result)
			return result, fmt.Errorf("reached maximum iterations (%d)", maxTotalIterations)
		}
	}
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

	// 创建思考标签处理器
	thinkingProcessor := NewThinkingTagProcessor()

	// 获取流式处理器（线程安全）
	a.mu.RLock()
	handler := a.streamHandler
	a.mu.RUnlock()

	for event := range streamChan {
		if event.Error != nil {
			// 调用错误回调
			if handler != nil {
				handler.OnError(event.Error)
			}
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

		// 处理思考内容（通过 reasoning_content 字段，如 Qwen3.5）
		if choice.Delta.ReasoningContent != "" {
			if handler != nil {
				handler.OnThinking(choice.Delta.ReasoningContent)
			}
			// 通过回调传递思考内容
			if a.onThinking != nil {
				a.onThinking(choice.Delta.ReasoningContent)
			}
		}

		// 处理内容
		if choice.Delta.Content != "" {
			// 使用思考标签处理器处理内容（兼容 <thinking> 标签模式）
			thinking, content := thinkingProcessor.Process(choice.Delta.Content)

			// 只将非思考内容写入内容构建器
			// 这确保响应消息中不包含 <thinking> 标签
			if content != "" {
				contentBuilder.WriteString(content)
			}

			// 调用流式处理器回调
			if handler != nil {
				if thinking != "" {
					handler.OnThinking(thinking)
				}
				if content != "" {
					handler.OnContent(content)
				}
			}

			// 通过回调传递thinking标签中的思考内容
			if thinking != "" {
				if a.onThinking != nil {
					a.onThinking(thinking)
				}
			}

			// 兼容旧的回调
			if a.onStreamChunk != nil {
				a.onStreamChunk(content)
			}
		}

		// 处理工具调用
		if len(choice.Delta.ToolCalls) > 0 {
			for _, tc := range choice.Delta.ToolCalls {
				// 新的工具调用
				if tc.ID != "" {
					if currentToolCall != nil {
						toolCalls = append(toolCalls, *currentToolCall)
						// 调用工具调用回调（完成）
						if handler != nil {
							handler.OnToolCall(ConvertToolCall(*currentToolCall), true)
						}
					}
					currentToolCall = &model.ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
						Function: model.FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
					// 调用工具调用回调（开始）
					if handler != nil {
						handler.OnToolCall(ConvertToolCall(*currentToolCall), false)
					}
				} else if currentToolCall != nil {
					// 追加参数
					currentToolCall.Function.Arguments += tc.Function.Arguments
					// 调用工具调用回调（更新）
					if handler != nil {
						handler.OnToolCall(ConvertToolCall(*currentToolCall), false)
					}
				}
			}
		}
	}

	// 添加最后一个工具调用
	if currentToolCall != nil {
		toolCalls = append(toolCalls, *currentToolCall)
		// 调用工具调用回调（完成）
		if handler != nil {
			handler.OnToolCall(ConvertToolCall(*currentToolCall), true)
		}
	}

	// 刷新思考标签处理器的缓冲区
	thinking, content := thinkingProcessor.Flush()
	if thinking != "" {
		if handler != nil {
			handler.OnThinking(thinking)
		}
		if a.onThinking != nil {
			a.onThinking(thinking)
		}
	}
	if content != "" {
		contentBuilder.WriteString(content)
		if handler != nil {
			handler.OnContent(content)
		}
	}

	// 估算 token 用量
	// 简单估算：平均每4个字符约1个token
	totalContent := contentBuilder.String()
	for _, tc := range toolCalls {
		totalContent += tc.Function.Name + tc.Function.Arguments
	}
	usage := len(totalContent) / 4

	// 调用完成回调
	if handler != nil {
		handler.OnComplete(usage)
	}

	return &modelResponse{
		Content:   contentBuilder.String(),
		ToolCalls: toolCalls,
		Usage:     usage,
	}, nil
}

// modelResponse 模型响应
type modelResponse struct {
	Content   string
	ToolCalls []model.ToolCall
	Usage     int // token 用量（估算）
}

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

// SetOnToolResult 设置工具执行结果回调
func (a *Agent) SetOnToolResult(callback func(name, resultJSON string)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onToolResult = callback
}

// SetOnToolCallFull 设置工具调用完整回调
func (a *Agent) SetOnToolCallFull(callback func(toolCallID, name, args string)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onToolCallFull = callback
}

// SetOnReview 设置审查结果回调
func (a *Agent) SetOnReview(callback func(status, summary string)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onReview = callback
}

// SetOnVerify 设置校验结果回调
func (a *Agent) SetOnVerify(callback func(status, summary string)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onVerify = callback
}

// SetOnCorrection 设置修正指令回调
func (a *Agent) SetOnCorrection(callback func(instruction string)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onCorrection = callback
}

// SetOnNoToolCall 设置无工具调用回调
func (a *Agent) SetOnNoToolCall(callback func(count int, response string)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onNoToolCall = callback
}

// SetOnHistoryAdd 设置消息历史添加回调
func (a *Agent) SetOnHistoryAdd(callback func(role, content string)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onHistoryAdd = callback
}

// SetOnThinking 设置思考内容回调
func (a *Agent) SetOnThinking(callback func(thinking string)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onThinking = callback
}

// SetStreamHandler 设置流式消息处理器
func (a *Agent) SetStreamHandler(handler StreamHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.streamHandler = handler
}

// SendMessage 发送用户消息并启动推理
// 这是一个异步方法，会在后台启动推理过程
func (a *Agent) SendMessage(content string) error {
	// 检查是否已在运行（使用写锁消除TOCTOU竞态）
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return fmt.Errorf("agent is already running")
	}
	a.mu.Unlock()

	// 在后台启动推理
	go func() {
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
