package agent

import (
	"github.com/Attect/MukaAI/internal/model"
)

// StreamHandler 流式消息处理器接口
// 用于处理模型流式输出的各种事件
type StreamHandler interface {
	// OnThinking 处理思考内容块
	// 当模型输出思考内容时调用（如 <thinking> 标签内的内容）
	OnThinking(chunk string)

	// OnContent 处理正文内容块
	// 当模型输出正文内容时调用
	OnContent(chunk string)

	// OnToolCall 处理工具调用
	// call: 工具调用信息
	// isComplete: 表示工具调用是否已完成流式生成
	OnToolCall(call ToolCallInfo, isComplete bool)

	// OnToolResult 处理工具执行结果
	// 当工具执行完成并返回结果时调用
	OnToolResult(result ToolCallInfo)

	// OnComplete 处理推理完成
	// 当一次推理完成时调用
	// usage: 本次推理的 token 用量
	OnComplete(usage int)

	// OnError 处理错误
	// 当流式输出过程中发生错误时调用
	OnError(err error)

	// OnTaskDone 处理任务完成
	// 当整个任务（包括所有迭代）完成后调用，无论成功还是失败
	// 与 OnComplete 的区别：OnComplete 是单次推理完成，OnTaskDone 是整个任务完成
	OnTaskDone()

	// OnCompression 处理上下文压缩
	// 当上下文压缩发生时调用
	// originalCount: 原始消息数量
	// compressedCount: 压缩后消息数量
	// originalTokens: 原始 token 数量
	// compressedTokens: 压缩后 token 数量
	// summary: 压缩摘要（如果有）
	OnCompression(originalCount, compressedCount, originalTokens, compressedTokens int, summary string)
}

// SupervisorResultHandler 监督结果处理器接口
// StreamHandler实现者可选择同时实现此接口以接收监督结果事件
// 用于将Supervisor检查结果推送到GUI等前端
type SupervisorResultHandler interface {
	// OnSupervisorResult 处理监督结果
	// 当Supervisor完成一次检查后调用
	OnSupervisorResult(result *SupervisionResult)
}

// ToolCallInfo 工具调用信息
// 用于在流式处理中传递工具调用信息
type ToolCallInfo struct {
	// ID 工具调用唯一标识
	ID string

	// Name 工具名称
	Name string

	// Arguments 工具参数（JSON 格式）
	Arguments string

	// IsComplete 是否已完成流式生成
	IsComplete bool

	// Result 工具执行结果
	Result string

	// ResultError 工具执行错误
	ResultError string
}

// StreamHandlerFunc 流式处理器函数类型
// 用于将函数转换为 StreamHandler 接口
type StreamHandlerFunc struct {
	onThinking    func(chunk string)
	onContent     func(chunk string)
	onToolCall    func(call ToolCallInfo, isComplete bool)
	onToolResult  func(result ToolCallInfo)
	onComplete    func(usage int)
	onError       func(err error)
	onTaskDone    func()
	onCompression func(originalCount, compressedCount, originalTokens, compressedTokens int, summary string)
}

// NewStreamHandlerFunc 创建基于函数的流式处理器
func NewStreamHandlerFunc() *StreamHandlerFunc {
	return &StreamHandlerFunc{}
}

// OnThinking 设置思考内容处理函数
func (h *StreamHandlerFunc) OnThinking(fn func(chunk string)) *StreamHandlerFunc {
	h.onThinking = fn
	return h
}

// OnContent 设置正文内容处理函数
func (h *StreamHandlerFunc) OnContent(fn func(chunk string)) *StreamHandlerFunc {
	h.onContent = fn
	return h
}

// OnToolCall 设置工具调用处理函数
func (h *StreamHandlerFunc) OnToolCall(fn func(call ToolCallInfo, isComplete bool)) *StreamHandlerFunc {
	h.onToolCall = fn
	return h
}

// OnToolResult 设置工具结果处理函数
func (h *StreamHandlerFunc) OnToolResult(fn func(result ToolCallInfo)) *StreamHandlerFunc {
	h.onToolResult = fn
	return h
}

// OnComplete 设置完成处理函数
func (h *StreamHandlerFunc) OnComplete(fn func(usage int)) *StreamHandlerFunc {
	h.onComplete = fn
	return h
}

// OnError 设置错误处理函数
func (h *StreamHandlerFunc) OnError(fn func(err error)) *StreamHandlerFunc {
	h.onError = fn
	return h
}

// OnTaskDone 设置任务完成处理函数
func (h *StreamHandlerFunc) OnTaskDone(fn func()) *StreamHandlerFunc {
	h.onTaskDone = fn
	return h
}

// OnCompression 设置压缩处理函数
func (h *StreamHandlerFunc) OnCompression(fn func(originalCount, compressedCount, originalTokens, compressedTokens int, summary string)) *StreamHandlerFunc {
	h.onCompression = fn
	return h
}

// Build 构建 StreamHandler 接口
func (h *StreamHandlerFunc) Build() StreamHandler {
	return &streamHandlerFuncImpl{
		onThinking:    h.onThinking,
		onContent:     h.onContent,
		onToolCall:    h.onToolCall,
		onToolResult:  h.onToolResult,
		onComplete:    h.onComplete,
		onError:       h.onError,
		onTaskDone:    h.onTaskDone,
		onCompression: h.onCompression,
	}
}

// streamHandlerFuncImpl 函数式流式处理器实现
type streamHandlerFuncImpl struct {
	onThinking    func(chunk string)
	onContent     func(chunk string)
	onToolCall    func(call ToolCallInfo, isComplete bool)
	onToolResult  func(result ToolCallInfo)
	onComplete    func(usage int)
	onError       func(err error)
	onTaskDone    func()
	onCompression func(originalCount, compressedCount, originalTokens, compressedTokens int, summary string)
}

func (h *streamHandlerFuncImpl) OnThinking(chunk string) {
	if h.onThinking != nil {
		h.onThinking(chunk)
	}
}

func (h *streamHandlerFuncImpl) OnContent(chunk string) {
	if h.onContent != nil {
		h.onContent(chunk)
	}
}

func (h *streamHandlerFuncImpl) OnToolCall(call ToolCallInfo, isComplete bool) {
	if h.onToolCall != nil {
		h.onToolCall(call, isComplete)
	}
}

func (h *streamHandlerFuncImpl) OnToolResult(result ToolCallInfo) {
	if h.onToolResult != nil {
		h.onToolResult(result)
	}
}

func (h *streamHandlerFuncImpl) OnComplete(usage int) {
	if h.onComplete != nil {
		h.onComplete(usage)
	}
}

func (h *streamHandlerFuncImpl) OnError(err error) {
	if h.onError != nil {
		h.onError(err)
	}
}

func (h *streamHandlerFuncImpl) OnTaskDone() {
	if h.onTaskDone != nil {
		h.onTaskDone()
	}
}

func (h *streamHandlerFuncImpl) OnCompression(originalCount, compressedCount, originalTokens, compressedTokens int, summary string) {
	if h.onCompression != nil {
		h.onCompression(originalCount, compressedCount, originalTokens, compressedTokens, summary)
	}
}

// ConvertToolCall 将 model.ToolCall 转换为 ToolCallInfo
func ConvertToolCall(tc model.ToolCall) ToolCallInfo {
	return ToolCallInfo{
		ID:         tc.ID,
		Name:       tc.Function.Name,
		Arguments:  tc.Function.Arguments,
		IsComplete: true,
	}
}

// ConvertToolCallWithResult 将 model.ToolCall 转换为带结果的 ToolCallInfo
func ConvertToolCallWithResult(tc model.ToolCall, result string, resultError string) ToolCallInfo {
	return ToolCallInfo{
		ID:          tc.ID,
		Name:        tc.Function.Name,
		Arguments:   tc.Function.Arguments,
		IsComplete:  true,
		Result:      result,
		ResultError: resultError,
	}
}
