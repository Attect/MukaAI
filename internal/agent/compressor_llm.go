package agent

import (
	"context"

	"github.com/Attect/MukaAI/internal/model"
)

// newLLMSummaryFunc 创建基于model.Client的LLM摘要生成函数
// 使用ChatCompletion API生成摘要，失败时返回空字符串
func newLLMSummaryFunc(client *model.Client) LLMSummaryFunc {
	return func(ctx context.Context, messages []model.Message, prompt string) string {
		// 构建摘要请求的消息列表
		summaryMessages := []model.Message{
			model.NewSystemMessage("你是一个专业的对话摘要助手。请根据提供的对话历史，生成准确的中文摘要。摘要应该简洁明了，包含关键信息和决策。"),
			model.NewUserMessage(prompt),
		}

		// 调用ChatCompletion（非流式）
		resp, err := client.ChatCompletion(ctx, summaryMessages, nil)
		if err != nil {
			return "" // 失败时返回空字符串，调用方会回退到规则提取
		}

		if len(resp.Choices) == 0 {
			return ""
		}

		content := resp.Choices[0].Message.Content
		if content == "" {
			return ""
		}

		return content
	}
}

// summarizeWithLLM 使用LLM生成上下文摘要（公开方法，用于测试）
// ctx: 上下文，用于超时控制
// messages: 待摘要的消息列表
// prompt: 摘要生成的提示词
// 返回：摘要文本，如果LLM调用失败则返回空字符串（回退到规则提取）
func (c *Compressor) summarizeWithLLM(ctx context.Context, messages []model.Message, prompt string) string {
	c.mu.RLock()
	fn := c.llmSummarize
	c.mu.RUnlock()

	if fn == nil {
		return ""
	}

	return fn(ctx, messages, prompt)
}
