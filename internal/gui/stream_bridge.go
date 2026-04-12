package gui

import (
	"context"
	"time"

	"agentplus/internal/agent"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// StreamBridge 将agent.StreamHandler接口桥接到Wails事件系统
// 实现StreamHandler接口，将所有流式事件转发为Wails前端事件
// 同时更新App中的对话状态，保证前端数据一致性
type StreamBridge struct {
	ctx context.Context
	app *App
}

// NewStreamBridge 创建新的StreamBridge实例
func NewStreamBridge(app *App) *StreamBridge {
	return &StreamBridge{app: app}
}

// SetContext 设置Wails上下文
// 必须在Wails应用启动后调用，否则事件无法发送到前端
func (b *StreamBridge) SetContext(ctx context.Context) {
	b.ctx = ctx
}

// OnThinking 处理思考内容块
// 将思考内容追加到当前消息的thinking字段，并发射stream:thinking事件
func (b *StreamBridge) OnThinking(chunk string) {
	b.app.mu.Lock()
	conv := b.app.getActiveConversation()
	if conv != nil {
		if conv.currentMessage == nil {
			conv.currentMessage = &message{
				role:      "assistant",
				timestamp: time.Now(),
			}
		}
		conv.currentMessage.thinking += chunk
		conv.currentMessage.isStreaming = true
		conv.currentMessage.streamingType = "thinking"
	}
	b.app.mu.Unlock()
	runtime.EventsEmit(b.ctx, "stream:thinking", chunk)
	runtime.EventsEmit(b.ctx, "conversation:updated", b.app.GetConversationData())
}

// OnContent 处理正文内容块
// 将正文内容追加到当前消息的content字段，并发射stream:content事件
func (b *StreamBridge) OnContent(chunk string) {
	b.app.mu.Lock()
	conv := b.app.getActiveConversation()
	if conv != nil {
		if conv.currentMessage == nil {
			conv.currentMessage = &message{
				role:      "assistant",
				timestamp: time.Now(),
			}
		}
		conv.currentMessage.content += chunk
		conv.currentMessage.isStreaming = true
		conv.currentMessage.streamingType = "content"
	}
	b.app.mu.Unlock()
	runtime.EventsEmit(b.ctx, "stream:content", chunk)
	runtime.EventsEmit(b.ctx, "conversation:updated", b.app.GetConversationData())
}

// OnToolCall 处理工具调用
// 将工具调用信息更新到当前消息的toolCalls列表，并发射stream:toolcall事件
// 如果同一ID的工具调用已存在，则更新其内容（流式参数拼接场景）
func (b *StreamBridge) OnToolCall(call agent.ToolCallInfo, isComplete bool) {
	b.app.mu.Lock()
	conv := b.app.getActiveConversation()
	if conv != nil {
		if conv.currentMessage == nil {
			conv.currentMessage = &message{
				role:      "assistant",
				timestamp: time.Now(),
			}
		}
		tc := ToolCall{
			ID:          call.ID,
			Name:        call.Name,
			Arguments:   call.Arguments,
			IsComplete:  isComplete,
			Result:      call.Result,
			ResultError: call.ResultError,
		}
		// 查找是否已存在同ID的工具调用，存在则更新（流式参数拼接）
		found := false
		for i, existing := range conv.currentMessage.toolCalls {
			if existing.ID == call.ID {
				conv.currentMessage.toolCalls[i] = tc
				found = true
				break
			}
		}
		if !found {
			conv.currentMessage.toolCalls = append(conv.currentMessage.toolCalls, tc)
		}
		conv.currentMessage.isStreaming = true
		conv.currentMessage.streamingType = "tool"
	}
	b.app.mu.Unlock()

	eventData := map[string]interface{}{
		"id":          call.ID,
		"name":        call.Name,
		"arguments":   call.Arguments,
		"isComplete":  isComplete,
		"result":      call.Result,
		"resultError": call.ResultError,
	}
	runtime.EventsEmit(b.ctx, "stream:toolcall", eventData)
	runtime.EventsEmit(b.ctx, "conversation:updated", b.app.GetConversationData())
}

// OnToolResult 处理工具执行结果
// 更新当前消息中对应工具调用的结果，并发射stream:toolresult事件
func (b *StreamBridge) OnToolResult(result agent.ToolCallInfo) {
	b.app.mu.Lock()
	conv := b.app.getActiveConversation()
	if conv != nil && conv.currentMessage != nil {
		for i, tc := range conv.currentMessage.toolCalls {
			if tc.ID == result.ID {
				conv.currentMessage.toolCalls[i].Result = result.Result
				conv.currentMessage.toolCalls[i].ResultError = result.ResultError
				break
			}
		}
	}
	b.app.mu.Unlock()

	eventData := map[string]interface{}{
		"id":          result.ID,
		"name":        result.Name,
		"result":      result.Result,
		"resultError": result.ResultError,
	}
	runtime.EventsEmit(b.ctx, "stream:toolresult", eventData)
	runtime.EventsEmit(b.ctx, "conversation:updated", b.app.GetConversationData())
}

// OnComplete 处理单次推理完成
// 仅更新token统计和状态，不再固化消息到messages列表
// 消息的固化改由OnTaskDone统一处理，避免多次迭代产生多条assistant消息
func (b *StreamBridge) OnComplete(usage int) {
	b.app.mu.Lock()
	conv := b.app.getActiveConversation()
	if conv != nil && conv.currentMessage != nil {
		conv.currentMessage.isStreaming = false
		conv.currentMessage.tokenUsage = usage
		conv.currentMessage.timestamp = time.Now()
		// 不再固化消息到messages列表，由OnTaskDone处理
		// 更新token统计
		conv.tokenUsage += usage
		b.app.totalTokens += usage
		b.app.inferenceCount++
	}
	b.app.mu.Unlock()

	runtime.EventsEmit(b.ctx, "stream:complete", map[string]interface{}{
		"usage": usage,
	})
	runtime.EventsEmit(b.ctx, "tokenstats:updated", b.app.GetTokenStats())
	// 不再在这里发射conversation:updated，因为消息还没固化
}

// OnError 处理错误
// 当发生实际错误时，将当前流式消息（如果有内容）固化并发射stream:error事件
// 注意：err为nil时不发射error事件，仅更新对话状态
func (b *StreamBridge) OnError(err error) {
	b.app.mu.Lock()
	if err != nil {
		if conv := b.app.getActiveConversation(); conv != nil && conv.currentMessage != nil {
			conv.currentMessage.isStreaming = false
			// 只有当消息有实际内容时才保存，避免保存空消息
			if conv.currentMessage.content != "" || conv.currentMessage.thinking != "" || len(conv.currentMessage.toolCalls) > 0 {
				conv.messages = append(conv.messages, conv.currentMessage)
			}
			conv.currentMessage = nil
		}
	}
	b.app.isStreaming = false
	b.app.mu.Unlock()

	if err != nil {
		runtime.EventsEmit(b.ctx, "stream:error", err.Error())
	}
	runtime.EventsEmit(b.ctx, "conversation:updated", b.app.GetConversationData())
}

// OnTaskDone 处理任务完成
// 当整个Agent任务（包括所有迭代）完成后调用
// 固化当前消息到消息列表，重置流式状态，并发射stream:done事件
func (b *StreamBridge) OnTaskDone() {
	b.app.mu.Lock()
	// 固化当前消息到消息列表
	conv := b.app.getActiveConversation()
	if conv != nil && conv.currentMessage != nil {
		conv.currentMessage.isStreaming = false
		conv.messages = append(conv.messages, conv.currentMessage)
		conv.currentMessage = nil
	}
	b.app.isStreaming = false
	b.app.mu.Unlock()

	runtime.EventsEmit(b.ctx, "stream:done")
	runtime.EventsEmit(b.ctx, "conversation:updated", b.app.GetConversationData())
}
