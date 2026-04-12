// Package agent 模型调用逻辑
// 从 core.go 提取的模型调用和流式响应处理逻辑
package agent

import (
	"context"
	"strings"

	"agentplus/internal/model"
)

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
