package agent

import (
	"strings"
)

// ThinkingTagProcessor 思考标签处理器
// 用于在流式输出中识别和处理 <thinking> 标签
type ThinkingTagProcessor struct {
	// inThinking 是否正在思考标签内
	inThinking bool

	// buffer 缓冲区，用于处理跨块的标签
	buffer string
}

// NewThinkingTagProcessor 创建新的思考标签处理器
func NewThinkingTagProcessor() *ThinkingTagProcessor {
	return &ThinkingTagProcessor{
		inThinking: false,
		buffer:     "",
	}
}

// Process 处理内容块
// 返回思考内容和正文内容
func (p *ThinkingTagProcessor) Process(chunk string) (thinking string, content string) {
	// 将新块添加到缓冲区
	p.buffer += chunk

	// 处理缓冲区中的内容
	for {
		if p.inThinking {
			// 在思考标签内，查找结束标签
			endIdx := strings.Index(p.buffer, "</thinking>")
			if endIdx == -1 {
				// 没有找到结束标签
				// 检查是否有可能是不完整的结束标签
				if partialLen := p.getPartialTagLen("</thinking"); partialLen > 0 {
					// 保留可能不完整的标签部分
					thinking = p.buffer[:len(p.buffer)-partialLen]
					p.buffer = p.buffer[len(p.buffer)-partialLen:]
					return
				}
				// 整个缓冲区都是思考内容
				thinking = p.buffer
				p.buffer = ""
				return
			}

			// 找到结束标签，提取思考内容
			thinking = p.buffer[:endIdx]
			p.buffer = p.buffer[endIdx+len("</thinking>"):]
			p.inThinking = false

			// 继续处理剩余的缓冲区
			continue
		}

		// 不在思考标签内，查找开始标签
		startIdx := strings.Index(p.buffer, "<thinking>")
		if startIdx == -1 {
			// 没有找到开始标签
			// 检查是否有可能是不完整的开始标签
			if partialLen := p.getPartialTagLen("<thinking"); partialLen > 0 {
				// 保留可能不完整的标签部分
				content = p.buffer[:len(p.buffer)-partialLen]
				p.buffer = p.buffer[len(p.buffer)-partialLen:]
				return
			}
			// 整个缓冲区都是正文内容
			content = p.buffer
			p.buffer = ""
			return
		}

		// 找到开始标签，提取正文内容
		if startIdx > 0 {
			content = p.buffer[:startIdx]
		}
		p.buffer = p.buffer[startIdx+len("<thinking>"):]
		p.inThinking = true

		// 继续处理剩余的缓冲区
		continue
	}
}

// hasPartialTag 检查缓冲区末尾是否有部分标签
func (p *ThinkingTagProcessor) hasPartialTag(tag string) bool {
	// 检查缓冲区末尾是否匹配标签的前缀
	for i := 1; i < len(tag); i++ {
		if strings.HasSuffix(p.buffer, tag[:i]) {
			return true
		}
	}
	return false
}

// getPartialTagLen 获取部分标签的长度
func (p *ThinkingTagProcessor) getPartialTagLen(tag string) int {
	// 检查缓冲区末尾是否匹配标签的前缀
	for i := len(tag) - 1; i >= 1; i-- {
		if strings.HasSuffix(p.buffer, tag[:i]) {
			return i
		}
	}
	return 0
}

// Flush 刷新缓冲区
// 返回剩余的内容（根据当前状态判断是思考还是正文）
func (p *ThinkingTagProcessor) Flush() (thinking string, content string) {
	if p.buffer == "" {
		return "", ""
	}

	if p.inThinking {
		thinking = p.buffer
	} else {
		content = p.buffer
	}

	p.buffer = ""
	return
}

// IsInThinking 是否正在思考标签内
func (p *ThinkingTagProcessor) IsInThinking() bool {
	return p.inThinking
}

// Reset 重置处理器状态
func (p *ThinkingTagProcessor) Reset() {
	p.inThinking = false
	p.buffer = ""
}
