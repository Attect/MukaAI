package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Attect/MukaAI/internal/model"
	"github.com/Attect/MukaAI/internal/state"
)

// CompressorConfig 压缩器配置
type CompressorConfig struct {
	// TriggerThreshold 触发压缩的上下文使用阈值（0.0-1.0）
	// 当上下文使用超过此阈值时触发压缩，默认0.8（80%）
	TriggerThreshold float64

	// MinMessagesToKeep 压缩后保留的最小消息数量
	// 确保压缩后仍有足够的上下文，默认10
	MinMessagesToKeep int

	// MaxMessagesToKeep 压缩后保留的最大消息数量
	// 防止压缩后上下文仍然过长，默认20
	MaxMessagesToKeep int

	// KeepSystemMessages 是否保留所有系统消息
	// 默认true，系统消息通常包含重要的指令
	KeepSystemMessages bool

	// KeepRecentToolCalls 保留最近N次工具调用及其结果
	// 默认3，保留最近的工具调用有助于理解当前状态
	KeepRecentToolCalls int

	// EnableProgressiveCompression 是否启用渐进式压缩
	// 启用后会先尝试轻度压缩，如果仍然超限再进行深度压缩
	EnableProgressiveCompression bool

	// SummaryMaxLength 摘要的最大长度（字符数）
	// 用于限制生成的摘要大小，默认2000
	SummaryMaxLength int

	// LLMSummaryTimeout LLM生成摘要的超时时间（秒）
	// 超时则回退到规则提取，默认10秒
	LLMSummaryTimeout int
}

// DefaultCompressorConfig 返回默认的压缩器配置
func DefaultCompressorConfig() *CompressorConfig {
	return &CompressorConfig{
		TriggerThreshold:             0.8,
		MinMessagesToKeep:            10,
		MaxMessagesToKeep:            20,
		KeepSystemMessages:           true,
		KeepRecentToolCalls:          3,
		EnableProgressiveCompression: true,
		SummaryMaxLength:             2000,
		LLMSummaryTimeout:            10,
	}
}

// Validate 验证配置有效性
func (c *CompressorConfig) Validate() error {
	if c.TriggerThreshold < 0 || c.TriggerThreshold > 1 {
		return fmt.Errorf("trigger threshold must be between 0 and 1, got %f", c.TriggerThreshold)
	}
	if c.MinMessagesToKeep < 1 {
		return fmt.Errorf("min messages to keep must be at least 1, got %d", c.MinMessagesToKeep)
	}
	if c.MaxMessagesToKeep < c.MinMessagesToKeep {
		return fmt.Errorf("max messages to keep (%d) must be >= min messages to keep (%d)",
			c.MaxMessagesToKeep, c.MinMessagesToKeep)
	}
	if c.KeepRecentToolCalls < 0 {
		return fmt.Errorf("keep recent tool calls must be >= 0, got %d", c.KeepRecentToolCalls)
	}
	if c.SummaryMaxLength < 100 {
		return fmt.Errorf("summary max length must be at least 100, got %d", c.SummaryMaxLength)
	}
	return nil
}

// CompressionResult 压缩结果
type CompressionResult struct {
	// CompressedMessages 压缩后的消息列表
	CompressedMessages []model.Message

	// OriginalCount 原始消息数量
	OriginalCount int

	// CompressedCount 压缩后消息数量
	CompressedCount int

	// OriginalTokens 原始token数量
	OriginalTokens int

	// CompressedTokens 压缩后token数量
	CompressedTokens int

	// CompressionRatio 压缩比率（0-1）
	CompressionRatio float64

	// Summary 生成的上下文摘要
	Summary string

	// WasCompressed 是否进行了压缩
	WasCompressed bool
}

// LLMSummaryFunc LLM摘要生成函数类型
// 接收消息列表和提示词，返回摘要文本
// 调用方负责超时控制，失败时返回空字符串
type LLMSummaryFunc func(ctx context.Context, messages []model.Message, prompt string) string

// Compressor 上下文压缩器
// 负责在上下文过长时压缩消息历史，保留关键信息
type Compressor struct {
	mu           sync.RWMutex
	modelClient  *model.Client
	config       *CompressorConfig
	llmSummarize LLMSummaryFunc // LLM摘要生成函数（可选，为nil则只使用规则提取）
}

// NewCompressor 创建新的上下文压缩器
// 参数：
//   - modelClient: 模型客户端，用于计算token数量
//   - config: 压缩器配置，如果为nil则使用默认配置
//   - llmSummarize: LLM摘要生成函数（可选，为nil则只使用规则提取）
func NewCompressor(modelClient *model.Client, config *CompressorConfig, llmSummarize ...LLMSummaryFunc) (*Compressor, error) {
	if modelClient == nil {
		return nil, fmt.Errorf("model client cannot be nil")
	}

	if config == nil {
		config = DefaultCompressorConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid compressor config: %w", err)
	}

	// 设置LLM摘要超时默认值
	if config.LLMSummaryTimeout <= 0 {
		config.LLMSummaryTimeout = 10
	}

	var summarizeFn LLMSummaryFunc
	if len(llmSummarize) > 0 && llmSummarize[0] != nil {
		summarizeFn = llmSummarize[0]
	}

	return &Compressor{
		modelClient:  modelClient,
		config:       config,
		llmSummarize: summarizeFn,
	}, nil
}

// ShouldCompress 判断是否需要压缩
// 根据当前上下文使用情况判断是否需要触发压缩
// 参数：
//   - messages: 当前消息历史
//
// 返回：
//   - bool: 是否需要压缩
//   - float64: 当前上下文使用率（0-1）
func (c *Compressor) ShouldCompress(messages []model.Message) (bool, float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.shouldCompressInternal(messages)
}

// shouldCompressInternal 内部判断是否需要压缩（不加锁）
// 调用此方法的方法必须已经持有锁
func (c *Compressor) shouldCompressInternal(messages []model.Message) (bool, float64) {
	if len(messages) == 0 {
		return false, 0
	}

	// 计算当前token数量
	currentTokens := c.modelClient.CountTokens(messages)
	contextSize := c.modelClient.GetConfig().ContextSize

	if contextSize <= 0 {
		return false, 0
	}

	usageRatio := float64(currentTokens) / float64(contextSize)
	shouldCompress := usageRatio >= c.config.TriggerThreshold

	return shouldCompress, usageRatio
}

// Compress 压缩消息历史
// 根据任务状态和压缩策略压缩消息历史
// 参数：
//   - messages: 原始消息历史
//   - taskState: 当前任务状态（用于生成摘要）
//
// 返回：
//   - *CompressionResult: 压缩结果
//   - error: 错误信息
func (c *Compressor) Compress(messages []model.Message, taskState *state.TaskState) (*CompressionResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(messages) == 0 {
		return &CompressionResult{
			CompressedMessages: messages,
			WasCompressed:      false,
		}, nil
	}

	// 检查是否需要压缩（使用内部方法，避免死锁）
	shouldCompress, _ := c.shouldCompressInternal(messages)
	if !shouldCompress {
		return &CompressionResult{
			CompressedMessages: messages,
			OriginalCount:      len(messages),
			CompressedCount:    len(messages),
			WasCompressed:      false,
		}, nil
	}

	// 计算原始token数量
	originalTokens := c.modelClient.CountTokens(messages)

	// 执行压缩策略
	var compressed []model.Message
	var summary string
	var err error

	if c.config.EnableProgressiveCompression {
		// 渐进式压缩：先尝试轻度压缩
		compressed, summary, err = c.progressiveCompress(messages, taskState)
	} else {
		// 直接深度压缩
		compressed, summary, err = c.deepCompress(messages, taskState)
	}

	if err != nil {
		return nil, fmt.Errorf("compression failed: %w", err)
	}

	// 计算压缩后token数量
	compressedTokens := c.modelClient.CountTokens(compressed)

	// 计算压缩比率
	var compressionRatio float64
	if originalTokens > 0 {
		compressionRatio = float64(compressedTokens) / float64(originalTokens)
	}

	return &CompressionResult{
		CompressedMessages: compressed,
		OriginalCount:      len(messages),
		CompressedCount:    len(compressed),
		OriginalTokens:     originalTokens,
		CompressedTokens:   compressedTokens,
		CompressionRatio:   compressionRatio,
		Summary:            summary,
		WasCompressed:      true,
	}, nil
}

// progressiveCompress 渐进式压缩
// 先尝试轻度压缩，如果仍然超限再进行深度压缩
// 注意：此方法在Compress方法中被调用，调用时已持有锁
func (c *Compressor) progressiveCompress(messages []model.Message, taskState *state.TaskState) ([]model.Message, string, error) {
	// 第一阶段：轻度压缩（保留更多消息）
	lightCompressed, summary, err := c.lightCompress(messages, taskState)
	if err != nil {
		return nil, "", err
	}

	// 检查轻度压缩后是否仍然超限（使用内部方法，避免死锁）
	shouldCompress, _ := c.shouldCompressInternal(lightCompressed)
	if !shouldCompress {
		return lightCompressed, summary, nil
	}

	// 第二阶段：深度压缩
	return c.deepCompress(messages, taskState)
}

// lightCompress 轻度压缩
// 保留更多消息，只移除明显冗余的部分
func (c *Compressor) lightCompress(messages []model.Message, taskState *state.TaskState) ([]model.Message, string, error) {
	// 分离系统消息和其他消息
	var systemMessages []model.Message
	var otherMessages []model.Message

	for _, msg := range messages {
		if msg.Role == model.RoleSystem {
			systemMessages = append(systemMessages, msg)
		} else {
			otherMessages = append(otherMessages, msg)
		}
	}

	// 生成上下文摘要
	summary := c.generateContextSummary(messages, taskState)

	// 保留最近的工具调用和结果
	recentToolMessages := c.extractRecentToolMessages(otherMessages, c.config.KeepRecentToolCalls)

	// 保留最近的对话
	recentCount := c.config.MaxMessagesToKeep - len(systemMessages) - len(recentToolMessages)/2 // 工具调用和结果成对出现
	if recentCount < c.config.MinMessagesToKeep-len(systemMessages) {
		recentCount = c.config.MinMessagesToKeep - len(systemMessages)
	}

	var recentMessages []model.Message
	if len(otherMessages) > recentCount {
		// 跳过已经被提取的工具消息
		toolMsgIndices := make(map[int]bool)
		for _, tm := range recentToolMessages {
			for i, om := range otherMessages {
				if messagesEqual(om, tm) {
					toolMsgIndices[i] = true
					break
				}
			}
		}

		// 从最新的消息开始收集
		startIdx := len(otherMessages) - recentCount
		for i := startIdx; i < len(otherMessages); i++ {
			if !toolMsgIndices[i] {
				recentMessages = append(recentMessages, otherMessages[i])
			}
		}
	} else {
		recentMessages = otherMessages
	}

	// 合并消息
	result := make([]model.Message, 0, len(systemMessages)+len(recentMessages)+len(recentToolMessages))
	result = append(result, systemMessages...)

	// 如果有摘要，插入摘要作为用户消息
	if summary != "" {
		result = append(result, model.NewUserMessage(
			fmt.Sprintf("[上下文摘要]\n%s", summary),
		))
	}

	// 添加最近的消息和工具消息
	// 需要按原始顺序合并
	result = c.mergeMessagesInOrder(result, recentMessages, recentToolMessages)

	return result, summary, nil
}

// deepCompress 深度压缩
// 更激进的压缩策略，只保留最关键的信息
func (c *Compressor) deepCompress(messages []model.Message, taskState *state.TaskState) ([]model.Message, string, error) {
	// 分离系统消息和其他消息
	var systemMessages []model.Message
	var otherMessages []model.Message

	for _, msg := range messages {
		if msg.Role == model.RoleSystem {
			systemMessages = append(systemMessages, msg)
		} else {
			otherMessages = append(otherMessages, msg)
		}
	}

	// 生成详细的上下文摘要
	summary := c.generateDetailedSummary(messages, taskState)

	// 只保留最近的工具调用和结果
	recentToolMessages := c.extractRecentToolMessages(otherMessages, c.config.KeepRecentToolCalls)

	// 保留最后几条消息
	keepCount := c.config.MinMessagesToKeep - len(systemMessages) - len(recentToolMessages)/2
	if keepCount < 2 {
		keepCount = 2
	}

	var recentMessages []model.Message
	if len(otherMessages) > keepCount {
		// 找到工具消息的位置
		toolMsgIndices := make(map[int]bool)
		for _, tm := range recentToolMessages {
			for i, om := range otherMessages {
				if messagesEqual(om, tm) {
					toolMsgIndices[i] = true
					break
				}
			}
		}

		// 从最新的消息开始收集
		startIdx := len(otherMessages) - keepCount
		for i := startIdx; i < len(otherMessages); i++ {
			if !toolMsgIndices[i] {
				recentMessages = append(recentMessages, otherMessages[i])
			}
		}
	} else {
		recentMessages = otherMessages
	}

	// 合并消息
	result := make([]model.Message, 0, len(systemMessages)+1+len(recentMessages)+len(recentToolMessages))
	result = append(result, systemMessages...)

	// 插入详细摘要
	if summary != "" {
		result = append(result, model.NewUserMessage(
			fmt.Sprintf("[上下文摘要 - 深度压缩]\n%s", summary),
		))
	}

	// 合并消息
	result = c.mergeMessagesInOrder(result, recentMessages, recentToolMessages)

	return result, summary, nil
}

// extractRecentToolMessages 提取最近的工具调用和结果消息
func (c *Compressor) extractRecentToolMessages(messages []model.Message, count int) []model.Message {
	if count <= 0 {
		return nil
	}

	var toolMessages []model.Message
	toolCallCount := 0

	// 从后向前遍历，收集工具调用和结果
	for i := len(messages) - 1; i >= 0 && toolCallCount < count; i-- {
		msg := messages[i]
		// 检查是否是工具结果消息
		if msg.Role == model.RoleTool {
			toolMessages = append([]model.Message{msg}, toolMessages...)
			// 查找对应的工具调用消息
			toolCallID := msg.ToolCallID
			for j := i - 1; j >= 0; j-- {
				if messages[j].Role == model.RoleAssistant && len(messages[j].ToolCalls) > 0 {
					for _, tc := range messages[j].ToolCalls {
						if tc.ID == toolCallID {
							toolMessages = append([]model.Message{messages[j]}, toolMessages...)
							toolCallCount++
							break
						}
					}
					break
				}
			}
		}
	}

	return toolMessages
}

// mergeMessagesInOrder 按原始消息顺序合并recent和toolMessages
// 确保消息按照它们在原始对话中的出现顺序排列
func (c *Compressor) mergeMessagesInOrder(base, recent, toolMessages []model.Message) []model.Message {
	if len(recent) == 0 && len(toolMessages) == 0 {
		return base
	}

	// 使用消息内容哈希来标识消息，记录每条消息在原始otherMessages中的位置
	type indexedMsg struct {
		msg   model.Message
		order int // 基于内容的唯一标识来推断顺序
	}

	// 收集所有需要合并的消息
	var allMsgs []indexedMsg

	// 为recent消息分配顺序：使用消息在recent切片中的位置
	for i, msg := range recent {
		allMsgs = append(allMsgs, indexedMsg{msg: msg, order: i})
	}

	// 为toolMessages分配顺序：需要找到它们在recent中的相对位置
	// 工具调用消息应该在对应的assistant消息之后
	toolOffset := len(recent)
	for i, msg := range toolMessages {
		// 检查这条工具消息是否已经在recent中存在（避免重复）
		isDuplicate := false
		if msg.Role == model.RoleTool {
			for _, rMsg := range recent {
				if rMsg.Role == model.RoleTool && rMsg.ToolCallID == msg.ToolCallID {
					isDuplicate = true
					break
				}
			}
		} else if msg.Role == model.RoleAssistant && len(msg.ToolCalls) > 0 {
			// 检查assistant消息的工具调用是否已在recent中
			for _, rMsg := range recent {
				if rMsg.Role == model.RoleAssistant && len(rMsg.ToolCalls) > 0 {
					for _, tc := range msg.ToolCalls {
						for _, rtc := range rMsg.ToolCalls {
							if tc.ID == rtc.ID {
								isDuplicate = true
								break
							}
						}
						if isDuplicate {
							break
						}
					}
				}
				if isDuplicate {
					break
				}
			}
		}

		if !isDuplicate {
			allMsgs = append(allMsgs, indexedMsg{msg: msg, order: toolOffset + i})
		}
	}

	// 按order排序（稳定排序保持相同order的相对顺序）
	// 由于我们按顺序添加，已经是排序好的，直接追加到base即可
	result := make([]model.Message, len(base), len(base)+len(allMsgs))
	copy(result, base)
	for _, im := range allMsgs {
		result = append(result, im.msg)
	}

	return result
}

// generateContextSummary 生成上下文摘要
// 先使用规则提取生成基础摘要，然后尝试用LLM生成语义摘要
// 如果LLM失败则回退到规则提取结果
func (c *Compressor) generateContextSummary(messages []model.Message, taskState *state.TaskState) string {
	// 先用规则提取方法生成基础摘要（作为降级方案）
	ruleSummary := c.generateRuleBasedSummary(messages, taskState)

	// 尝试调用LLM生成语义摘要
	if c.llmSummarize != nil {
		prompt := buildSummaryPrompt("简洁", messages, taskState)
		llmSummary := c.callLLMSummary(messages, prompt)
		if llmSummary != "" {
			// LLM成功，截断到最大长度
			if len(llmSummary) > c.config.SummaryMaxLength {
				llmSummary = llmSummary[:c.config.SummaryMaxLength] + "..."
			}
			return llmSummary
		}
	}

	// LLM失败或未配置，使用规则摘要
	if len(ruleSummary) > c.config.SummaryMaxLength {
		ruleSummary = ruleSummary[:c.config.SummaryMaxLength] + "..."
	}
	return ruleSummary
}

// generateDetailedSummary 生成详细摘要
// 用于深度压缩时提供更完整的上下文信息
// 先使用规则提取，然后尝试用LLM生成语义摘要
func (c *Compressor) generateDetailedSummary(messages []model.Message, taskState *state.TaskState) string {
	// 先用规则提取方法生成基础摘要（作为降级方案）
	ruleSummary := c.generateRuleBasedDetailedSummary(messages, taskState)

	// 尝试调用LLM生成语义摘要
	if c.llmSummarize != nil {
		prompt := buildSummaryPrompt("详细", messages, taskState)
		llmSummary := c.callLLMSummary(messages, prompt)
		if llmSummary != "" {
			// LLM成功，截断到最大长度
			if len(llmSummary) > c.config.SummaryMaxLength {
				llmSummary = llmSummary[:c.config.SummaryMaxLength] + "..."
			}
			return llmSummary
		}
	}

	// LLM失败或未配置，使用规则摘要
	if len(ruleSummary) > c.config.SummaryMaxLength {
		ruleSummary = ruleSummary[:c.config.SummaryMaxLength] + "..."
	}
	return ruleSummary
}

// callLLMSummary 调用LLM生成摘要，带超时控制
// 返回空字符串表示失败（调用方应回退到规则提取）
func (c *Compressor) callLLMSummary(messages []model.Message, prompt string) string {
	timeout := time.Duration(c.config.LLMSummaryTimeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result := c.llmSummarize(ctx, messages, prompt)
	return result
}

// generateRuleBasedSummary 使用规则提取生成基础上下文摘要
// 这是LLM摘要的降级方案
func (c *Compressor) generateRuleBasedSummary(messages []model.Message, taskState *state.TaskState) string {
	var summary strings.Builder

	// 1. 从任务状态获取基本信息
	if taskState != nil {
		stateSummary, err := state.GetYAMLSummary(taskState)
		if err == nil && stateSummary != "" {
			summary.WriteString(stateSummary)
			summary.WriteString("\n")
		}
	}

	// 2. 提取关键决策点
	keyDecisions := c.extractKeyDecisions(messages)
	if len(keyDecisions) > 0 {
		summary.WriteString("关键决策:\n")
		for _, decision := range keyDecisions {
			summary.WriteString(fmt.Sprintf("  - %s\n", decision))
		}
	}

	// 3. 提取最近的操作
	recentActions := c.extractRecentActions(messages, 5)
	if len(recentActions) > 0 {
		summary.WriteString("最近操作:\n")
		for _, action := range recentActions {
			summary.WriteString(fmt.Sprintf("  - %s\n", action))
		}
	}

	return summary.String()
}

// generateRuleBasedDetailedSummary 使用规则提取生成详细摘要
// 这是LLM摘要的降级方案，用于深度压缩
func (c *Compressor) generateRuleBasedDetailedSummary(messages []model.Message, taskState *state.TaskState) string {
	var summary strings.Builder

	// 1. 任务目标和状态
	if taskState != nil {
		summary.WriteString(fmt.Sprintf("任务目标: %s\n", taskState.Task.Goal))
		summary.WriteString(fmt.Sprintf("当前状态: %s\n", taskState.Task.Status))
		summary.WriteString(fmt.Sprintf("当前阶段: %s\n", taskState.Progress.CurrentPhase))

		// 已完成步骤
		if len(taskState.Progress.CompletedSteps) > 0 {
			summary.WriteString("已完成步骤:\n")
			for _, step := range taskState.Progress.CompletedSteps {
				summary.WriteString(fmt.Sprintf("  - %s\n", step))
			}
		}

		// 待完成步骤
		if len(taskState.Progress.PendingSteps) > 0 {
			summary.WriteString("待完成步骤:\n")
			for _, step := range taskState.Progress.PendingSteps {
				summary.WriteString(fmt.Sprintf("  - %s\n", step))
			}
		}

		// 关键决策
		if len(taskState.Context.Decisions) > 0 {
			summary.WriteString("关键决策:\n")
			for _, decision := range taskState.Context.Decisions {
				summary.WriteString(fmt.Sprintf("  - %s\n", decision))
			}
		}

		// 相关文件
		if len(taskState.Context.Files) > 0 {
			summary.WriteString("相关文件:\n")
			for _, file := range taskState.Context.Files {
				summary.WriteString(fmt.Sprintf("  - %s (%s)\n", file.Path, file.Status))
			}
		}
	}

	// 2. 从消息中提取关键信息
	keyDecisions := c.extractKeyDecisions(messages)
	if len(keyDecisions) > 0 && taskState == nil {
		summary.WriteString("关键决策:\n")
		for _, decision := range keyDecisions {
			summary.WriteString(fmt.Sprintf("  - %s\n", decision))
		}
	}

	// 3. 提取工具调用历史
	toolHistory := c.extractToolHistory(messages)
	if len(toolHistory) > 0 {
		summary.WriteString("工具调用历史:\n")
		for _, th := range toolHistory {
			summary.WriteString(fmt.Sprintf("  - %s\n", th))
		}
	}

	return summary.String()
}

// buildSummaryPrompt 构建LLM摘要生成的提示词
// level: "简洁" 或 "详细"
func buildSummaryPrompt(level string, messages []model.Message, taskState *state.TaskState) string {
	var prompt strings.Builder

	if level == "简洁" {
		prompt.WriteString("请根据以下对话历史，生成一份简洁的中文摘要。摘要应包含：已完成的工作、当前状态、下一步计划、关键发现。\n\n")
	} else {
		prompt.WriteString("请根据以下对话历史，生成一份详细的中文摘要。摘要应包含：已完成的工作（列出具体步骤）、当前状态、下一步计划、关键发现、技术决策、遇到的问题及解决方案。\n\n")
	}

	// 添加任务状态信息
	if taskState != nil {
		prompt.WriteString(fmt.Sprintf("任务目标: %s\n", taskState.Task.Goal))
		prompt.WriteString(fmt.Sprintf("当前状态: %s\n", taskState.Task.Status))
		prompt.WriteString(fmt.Sprintf("当前阶段: %s\n", taskState.Progress.CurrentPhase))
		prompt.WriteString("\n")
	}

	// 添加消息摘要（只发送角色+内容前200字+工具调用名，避免消耗过多token）
	prompt.WriteString("对话历史摘要:\n")
	for i, msg := range messages {
		if i > 50 {
			// 最多50条消息摘要
			prompt.WriteString("...(省略更早的消息)\n")
			break
		}

		content := msg.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}

		switch msg.Role {
		case model.RoleSystem:
			prompt.WriteString(fmt.Sprintf("[系统] %s\n", content))
		case model.RoleUser:
			prompt.WriteString(fmt.Sprintf("[用户] %s\n", content))
		case model.RoleAssistant:
			prompt.WriteString(fmt.Sprintf("[助手] %s", content))
			if len(msg.ToolCalls) > 0 {
				toolNames := make([]string, len(msg.ToolCalls))
				for j, tc := range msg.ToolCalls {
					toolNames[j] = tc.Function.Name
				}
				prompt.WriteString(fmt.Sprintf(" [调用工具: %s]", strings.Join(toolNames, ", ")))
			}
			prompt.WriteString("\n")
		case model.RoleTool:
			resultSummary := content
			if len(resultSummary) > 100 {
				resultSummary = resultSummary[:100] + "..."
			}
			prompt.WriteString(fmt.Sprintf("[工具结果 %s] %s\n", msg.Name, resultSummary))
		}
	}

	prompt.WriteString("\n请直接输出摘要内容，不要包含额外的格式标记。")

	return prompt.String()
}

// SetLLMSummarize 设置LLM摘要生成函数
// 用于在创建Compressor后注入LLM能力
func (c *Compressor) SetLLMSummarize(fn LLMSummaryFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.llmSummarize = fn
}

// ExtractKeyInfo 提取关键信息
// 公开方法，允许外部调用提取消息中的关键信息
func (c *Compressor) ExtractKeyInfo(messages []model.Message) *KeyInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	info := &KeyInfo{
		Decisions:     c.extractKeyDecisions(messages),
		RecentActions: c.extractRecentActions(messages, 10),
		ToolHistory:   c.extractToolHistory(messages),
	}

	return info
}

// KeyInfo 关键信息结构
type KeyInfo struct {
	Decisions     []string // 关键决策
	RecentActions []string // 最近操作
	ToolHistory   []string // 工具调用历史
}

// extractKeyDecisions 从消息中提取关键决策
// 通过分析assistant消息中的决策性语句
func (c *Compressor) extractKeyDecisions(messages []model.Message) []string {
	var decisions []string

	keywords := []string{
		"决定", "选择", "采用", "使用", "方案", "策略",
		"decision", "choose", "select", "use", "decided", "will use", "approach", "strategy",
		"fix", "fixed", "implement", "implemented", "resolve", "resolved",
	}

	for _, msg := range messages {
		if msg.Role != model.RoleAssistant {
			continue
		}

		content := strings.ToLower(msg.Content)
		for _, keyword := range keywords {
			if strings.Contains(content, keyword) {
				// 提取包含关键词的句子
				sentences := c.extractSentencesWithKeyword(msg.Content, keyword)
				decisions = append(decisions, sentences...)
				break
			}
		}
	}

	// 去重并限制数量
	return c.deduplicateAndLimit(decisions, 5)
}

// extractRecentActions 提取最近的操作
func (c *Compressor) extractRecentActions(messages []model.Message, count int) []string {
	var actions []string

	// 从后向前遍历
	for i := len(messages) - 1; i >= 0 && len(actions) < count; i-- {
		msg := messages[i]

		// 提取工具调用
		if msg.Role == model.RoleAssistant && len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				action := fmt.Sprintf("调用工具: %s", tc.Function.Name)
				actions = append(actions, action)
			}
		}

		// 提取工具结果摘要
		if msg.Role == model.RoleTool {
			resultSummary := msg.Content
			if len(resultSummary) > 100 {
				resultSummary = resultSummary[:100] + "..."
			}
			action := fmt.Sprintf("工具结果 [%s]: %s", msg.Name, resultSummary)
			actions = append(actions, action)
		}
	}

	// 反转顺序，使最新的在最后
	for i, j := 0, len(actions)-1; i < j; i, j = i+1, j-1 {
		actions[i], actions[j] = actions[j], actions[i]
	}

	return actions
}

// extractToolHistory 提取工具调用历史
// 保留最近N次调用（按时间倒序），最多20条记录
// 格式：工具名(参数摘要) [第N次调用]
func (c *Compressor) extractToolHistory(messages []model.Message) []string {
	const maxEntries = 20

	// 收集所有工具调用，记录每个工具名的调用次数（从后向前计数）
	type toolEntry struct {
		text      string
		callIndex int // 该工具名的第几次调用（从后向前计数）
	}
	var entries []toolEntry
	toolCallCounts := make(map[string]int) // 工具名 -> 从后向前的累计调用次数

	// 从后向前遍历，保留最近的调用
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == model.RoleAssistant && len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				toolCallCounts[tc.Function.Name]++
				argsSummary := c.summarizeArguments(tc.Function.Arguments)
				entry := toolEntry{
					text:      fmt.Sprintf("%s(%s) [第%d次调用]", tc.Function.Name, argsSummary, toolCallCounts[tc.Function.Name]),
					callIndex: toolCallCounts[tc.Function.Name],
				}
				entries = append(entries, entry)
			}
		}
	}

	// entries 是按时间倒序的，反转为正序
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	// 限制最多 maxEntries 条
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}

	result := make([]string, len(entries))
	for i, e := range entries {
		result[i] = e.text
	}

	return result
}

// extractSentencesWithKeyword 提取包含关键词的句子
func (c *Compressor) extractSentencesWithKeyword(text, keyword string) []string {
	var sentences []string

	// 简单的句子分割
	parts := strings.Split(text, "\n")
	for _, part := range parts {
		if strings.Contains(strings.ToLower(part), strings.ToLower(keyword)) {
			// 清理并添加
			cleaned := strings.TrimSpace(part)
			if len(cleaned) > 10 && len(cleaned) < 200 {
				sentences = append(sentences, cleaned)
			}
		}
	}

	return sentences
}

// summarizeArguments 生成参数摘要
func (c *Compressor) summarizeArguments(argsJSON string) string {
	// 简单处理：提取参数名
	argsJSON = strings.TrimSpace(argsJSON)
	if argsJSON == "" || argsJSON == "{}" {
		return ""
	}

	// 移除花括号
	argsJSON = strings.Trim(argsJSON, "{}")

	// 提取参数名
	var params []string
	parts := strings.Split(argsJSON, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) >= 1 {
			key := strings.TrimSpace(strings.Trim(kv[0], "\""))
			if key != "" {
				params = append(params, key)
			}
		}
	}

	if len(params) == 0 {
		return ""
	}

	if len(params) <= 3 {
		return strings.Join(params, ", ")
	}

	return fmt.Sprintf("%s, ... (%d params)", params[0], len(params))
}

// deduplicateAndLimit 去重并限制数量
func (c *Compressor) deduplicateAndLimit(items []string, limit int) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
			if len(result) >= limit {
				break
			}
		}
	}

	return result
}

// GetConfig 获取压缩器配置
func (c *Compressor) GetConfig() *CompressorConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// UpdateConfig 更新压缩器配置
func (c *Compressor) UpdateConfig(config *CompressorConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = config
	return nil
}

// GetCompressionStats 获取压缩统计信息
// 返回最近一次压缩的统计（如果有的话）
func (c *Compressor) GetCompressionStats(messages []model.Message) *CompressionStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(messages) == 0 {
		return &CompressionStats{}
	}

	tokens := c.modelClient.CountTokens(messages)
	contextSize := c.modelClient.GetConfig().ContextSize

	var usageRatio float64
	if contextSize > 0 {
		usageRatio = float64(tokens) / float64(contextSize)
	}

	return &CompressionStats{
		MessageCount:     len(messages),
		TokenCount:       tokens,
		ContextSize:      contextSize,
		UsageRatio:       usageRatio,
		ShouldCompress:   usageRatio >= c.config.TriggerThreshold,
		TriggerThreshold: c.config.TriggerThreshold,
	}
}

// CompressionStats 压缩统计信息
type CompressionStats struct {
	MessageCount     int     // 消息数量
	TokenCount       int     // Token数量
	ContextSize      int     // 上下文大小
	UsageRatio       float64 // 使用率
	ShouldCompress   bool    // 是否应该压缩
	TriggerThreshold float64 // 触发阈值
}

// messagesEqual 比较两条消息是否相等
// 由于Message结构体包含slice，无法直接使用==比较
func messagesEqual(a, b model.Message) bool {
	if a.Role != b.Role {
		return false
	}
	if a.Content != b.Content {
		return false
	}
	if a.ToolCallID != b.ToolCallID {
		return false
	}
	if a.Name != b.Name {
		return false
	}
	// 比较ToolCalls长度
	if len(a.ToolCalls) != len(b.ToolCalls) {
		return false
	}
	// 逐个比较ToolCalls
	for i := range a.ToolCalls {
		if a.ToolCalls[i].ID != b.ToolCalls[i].ID {
			return false
		}
		if a.ToolCalls[i].Type != b.ToolCalls[i].Type {
			return false
		}
		if a.ToolCalls[i].Function.Name != b.ToolCalls[i].Function.Name {
			return false
		}
		if a.ToolCalls[i].Function.Arguments != b.ToolCalls[i].Function.Arguments {
			return false
		}
	}
	return true
}
