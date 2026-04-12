package agent

import (
	"testing"
)

// TestThinkingTagProcessor_Basic 测试基本的思考标签处理
func TestThinkingTagProcessor_Basic(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 测试纯正文内容
	thinking, content := processor.Process("This is content")
	if thinking != "" {
		t.Errorf("Expected no thinking content, got '%s'", thinking)
	}
	if content != "This is content" {
		t.Errorf("Expected content 'This is content', got '%s'", content)
	}
}

// TestThinkingTagProcessor_ThinkingOnly 测试纯思考内容
func TestThinkingTagProcessor_ThinkingOnly(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 测试纯思考内容
	thinking, content := processor.Process("<thinking>This is thinking</thinking>")
	if thinking != "This is thinking" {
		t.Errorf("Expected thinking 'This is thinking', got '%s'", thinking)
	}
	if content != "" {
		t.Errorf("Expected no content, got '%s'", content)
	}
}

// TestThinkingTagProcessor_Mixed 测试混合内容
func TestThinkingTagProcessor_Mixed(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 测试混合内容
	thinking, content := processor.Process("<thinking>This is thinking</thinking>This is content")
	if thinking != "This is thinking" {
		t.Errorf("Expected thinking 'This is thinking', got '%s'", thinking)
	}
	if content != "This is content" {
		t.Errorf("Expected content 'This is content', got '%s'", content)
	}
}

// TestThinkingTagProcessor_MultipleThinking 测试多个思考块
func TestThinkingTagProcessor_MultipleThinking(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 测试多个思考块
	thinking1, content1 := processor.Process("<thinking>First thinking</thinking>Content 1")
	if thinking1 != "First thinking" {
		t.Errorf("Expected thinking 'First thinking', got '%s'", thinking1)
	}
	if content1 != "Content 1" {
		t.Errorf("Expected content 'Content 1', got '%s'", content1)
	}

	thinking2, content2 := processor.Process("<thinking>Second thinking</thinking>Content 2")
	if thinking2 != "Second thinking" {
		t.Errorf("Expected thinking 'Second thinking', got '%s'", thinking2)
	}
	if content2 != "Content 2" {
		t.Errorf("Expected content 'Content 2', got '%s'", content2)
	}
}

// TestThinkingTagProcessor_SplitTags 测试跨块的标签
func TestThinkingTagProcessor_SplitTags(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 测试开始标签跨块
	thinking1, content1 := processor.Process("<think")
	if thinking1 != "" {
		t.Errorf("Expected no thinking, got '%s'", thinking1)
	}
	if content1 != "" {
		t.Errorf("Expected no content, got '%s'", content1)
	}

	thinking2, content2 := processor.Process("ing>This is thinking</think")
	if thinking2 != "This is thinking" {
		t.Errorf("Expected thinking 'This is thinking', got '%s'", thinking2)
	}
	if content2 != "" {
		t.Errorf("Expected no content, got '%s'", content2)
	}

	thinking3, content3 := processor.Process("ing>Content")
	if thinking3 != "" {
		t.Errorf("Expected no thinking, got '%s'", thinking3)
	}
	if content3 != "Content" {
		t.Errorf("Expected content 'Content', got '%s'", content3)
	}
}

// TestThinkingTagProcessor_UnclosedTag 测试未闭合的标签
func TestThinkingTagProcessor_UnclosedTag(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 测试未闭合的思考标签
	thinking1, content1 := processor.Process("<thinking>This is thinking")
	if thinking1 != "This is thinking" {
		t.Errorf("Expected thinking 'This is thinking', got '%s'", thinking1)
	}
	if content1 != "" {
		t.Errorf("Expected no content, got '%s'", content1)
	}

	// 刷新缓冲区
	thinking2, content2 := processor.Flush()
	if thinking2 != "" {
		t.Errorf("Expected no thinking after flush, got '%s'", thinking2)
	}
	if content2 != "" {
		t.Errorf("Expected no content after flush, got '%s'", content2)
	}
}

// TestThinkingTagProcessor_Flush 测试刷新缓冲区
func TestThinkingTagProcessor_Flush(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 处理一些内容
	thinking, content := processor.Process("Some content")
	if thinking != "" {
		t.Errorf("Expected no thinking, got '%s'", thinking)
	}
	if content != "Some content" {
		t.Errorf("Expected content 'Some content', got '%s'", content)
	}

	// 刷新缓冲区（应该为空）
	thinking2, content2 := processor.Flush()
	if thinking2 != "" {
		t.Errorf("Expected no thinking after flush, got '%s'", thinking2)
	}
	if content2 != "" {
		t.Errorf("Expected no content after flush, got '%s'", content2)
	}
}

// TestThinkingTagProcessor_Reset 测试重置处理器
func TestThinkingTagProcessor_Reset(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 处理一些内容
	processor.Process("<thinking>Thinking")

	// 重置
	processor.Reset()

	// 检查状态
	if processor.IsInThinking() {
		t.Error("Expected not to be in thinking after reset")
	}

	// 处理新内容
	thinking, content := processor.Process("New content")
	if thinking != "" {
		t.Errorf("Expected no thinking, got '%s'", thinking)
	}
	if content != "New content" {
		t.Errorf("Expected content 'New content', got '%s'", content)
	}
}

// TestThinkingTagProcessor_IsInThinking 测试状态检查
func TestThinkingTagProcessor_IsInThinking(t *testing.T) {
	processor := NewThinkingTagProcessor()

	if processor.IsInThinking() {
		t.Error("Expected not to be in thinking initially")
	}

	processor.Process("<thinking>")
	if !processor.IsInThinking() {
		t.Error("Expected to be in thinking after opening tag")
	}

	processor.Process("</thinking>")
	if processor.IsInThinking() {
		t.Error("Expected not to be in thinking after closing tag")
	}
}

// TestThinkingTagProcessor_EmptyTags 测试空标签
func TestThinkingTagProcessor_EmptyTags(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 测试空思考标签
	thinking, content := processor.Process("<thinking></thinking>Content")
	if thinking != "" {
		t.Errorf("Expected empty thinking, got '%s'", thinking)
	}
	if content != "Content" {
		t.Errorf("Expected content 'Content', got '%s'", content)
	}
}

// TestThinkingTagProcessor_NestedTags 测试嵌套标签（不应该出现，但需要处理）
func TestThinkingTagProcessor_NestedTags(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 测试嵌套标签（外层标签会被处理）
	thinking, _ := processor.Process("<thinking>Outer <thinking>Inner</thinking></thinking>Content")
	// 注意：这个测试可能需要根据实际需求调整
	// 当前的实现会处理第一个闭合标签
	if thinking == "" {
		t.Error("Expected some thinking content")
	}
}

// === 以下是新增的测试用例，提升覆盖率 ===

// TestThinkingTagProcessor_空字符串 测试空字符串输入
func TestThinkingTagProcessor_空字符串(t *testing.T) {
	processor := NewThinkingTagProcessor()

	thinking, content := processor.Process("")
	if thinking != "" {
		t.Errorf("期望空思考内容, 实际 '%s'", thinking)
	}
	if content != "" {
		t.Errorf("期望空正文内容, 实际 '%s'", content)
	}
}

// TestThinkingTagProcessor_纯文本无标签 测试不含任何标签的纯文本
func TestThinkingTagProcessor_纯文本无标签(t *testing.T) {
	processor := NewThinkingTagProcessor()

	thinking, content := processor.Process("这是一段普通的文本，没有任何标签。")
	if thinking != "" {
		t.Errorf("期望无思考内容, 实际 '%s'", thinking)
	}
	if content != "这是一段普通的文本，没有任何标签。" {
		t.Errorf("期望原始文本, 实际 '%s'", content)
	}
}

// TestThinkingTagProcessor_跨Chunk开始标签 测试开始标签在两个chunk之间拆分
func TestThinkingTagProcessor_跨Chunk开始标签(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 第一个chunk: "<thinki" - 不完整的开始标签
	thinking1, content1 := processor.Process("<thinki")
	if thinking1 != "" {
		t.Errorf("第一个chunk应无思考内容, 实际 '%s'", thinking1)
	}
	if content1 != "" {
		t.Errorf("第一个chunk应无正文内容（缓冲等待完整标签）, 实际 '%s'", content1)
	}

	// 第二个chunk: "ng>hello</thinking>result"
	thinking2, content2 := processor.Process("ng>hello</thinking>result")
	if thinking2 != "hello" {
		t.Errorf("期望思考内容 'hello', 实际 '%s'", thinking2)
	}
	if content2 != "result" {
		t.Errorf("期望正文 'result', 实际 '%s'", content2)
	}
}

// TestThinkingTagProcessor_跨Chunk结束标签 测试结束标签在两个chunk之间拆分
func TestThinkingTagProcessor_跨Chunk结束标签(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 第一个chunk: 开始标签 + 思考内容 + 不完整的结束标签
	thinking1, content1 := processor.Process("<thinking>思考内容</thinki")
	if thinking1 != "思考内容" {
		t.Errorf("期望思考内容 '思考内容', 实际 '%s'", thinking1)
	}
	if content1 != "" {
		t.Errorf("期望无正文内容, 实际 '%s'", content1)
	}

	// 第二个chunk: 结束标签的剩余部分 + 正文
	thinking2, content2 := processor.Process("ng>正文内容")
	if thinking2 != "" {
		t.Errorf("期望无思考内容, 实际 '%s'", thinking2)
	}
	if content2 != "正文内容" {
		t.Errorf("期望正文 '正文内容', 实际 '%s'", content2)
	}
}

// TestThinkingTagProcessor_多次连续Process 测试多次连续处理
func TestThinkingTagProcessor_多次连续Process(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 连续发送多个普通文本chunk
	var allContent string
	chunks := []string{"Hello ", "World ", "测试", "内容"}
	for _, chunk := range chunks {
		_, content := processor.Process(chunk)
		allContent += content
	}

	if allContent != "Hello World 测试内容" {
		t.Errorf("期望拼接内容 'Hello World 测试内容', 实际 '%s'", allContent)
	}
}

// TestThinkingTagProcessor_Flush在思考状态 测试在思考状态时刷新
func TestThinkingTagProcessor_Flush在思考状态(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 进入思考状态但不关闭 - 此时Process会返回全部思考内容
	// 因为在思考状态下，如果没有找到结束标签，整个buffer作为thinking返回
	thinking1, _ := processor.Process("<thinking>未完成的思考内容")
	if thinking1 != "未完成的思考内容" {
		t.Errorf("Process应返回思考内容 '未完成的思考内容', 实际 '%s'", thinking1)
	}

	// 刷新应返回空（Process已经消费了buffer）
	thinking2, content2 := processor.Flush()
	if thinking2 != "" {
		t.Errorf("期望空思考内容（已被Process消费）, 实际 '%s'", thinking2)
	}
	if content2 != "" {
		t.Errorf("期望空正文内容, 实际 '%s'", content2)
	}
}

// TestThinkingTagProcessor_Flush在正文状态 测试在正文状态时刷新
func TestThinkingTagProcessor_Flush在正文状态(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 处理普通文本（不进入思考状态）
	processor.Process("正文内容")

	// 缓冲区已被消费，刷新返回空
	thinking, content := processor.Flush()
	if thinking != "" {
		t.Errorf("期望空思考内容, 实际 '%s'", thinking)
	}
	if content != "" {
		t.Errorf("期望空正文内容（已被Process返回）, 实际 '%s'", content)
	}
}

// TestThinkingTagProcessor_Reset清除状态 测试重置清除所有状态
func TestThinkingTagProcessor_Reset清除状态(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 进入思考状态
	processor.Process("<thinking>部分思考")
	if !processor.IsInThinking() {
		t.Error("应处于思考状态")
	}

	// 重置
	processor.Reset()
	if processor.IsInThinking() {
		t.Error("重置后不应处于思考状态")
	}

	// 缓冲区应被清空
	thinking, content := processor.Flush()
	if thinking != "" || content != "" {
		t.Errorf("重置后刷新应返回空, thinking='%s', content='%s'", thinking, content)
	}
}

// TestThinkingTagProcessor_思考标签前后有正文 测试思考标签前后都有正文
func TestThinkingTagProcessor_思考标签前后有正文(t *testing.T) {
	processor := NewThinkingTagProcessor()

	thinking, content := processor.Process("前置正文<thinking>思考内容</thinking>后置正文")
	if thinking != "思考内容" {
		t.Errorf("期望思考 '思考内容', 实际 '%s'", thinking)
	}
	// 注意：当前实现中Process单次只返回content或thinking
	// 前置正文和后置正文可能在同一次Process中一起返回
	if content == "" {
		t.Error("期望有正文内容")
	}
}

// TestThinkingTagProcessor_连续两个思考块 测试连续两个思考块
func TestThinkingTagProcessor_连续两个思考块(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 第一个思考块
	thinking1, content1 := processor.Process("<thinking>第一段思考</thinking>")
	if thinking1 != "第一段思考" {
		t.Errorf("期望 '第一段思考', 实际 '%s'", thinking1)
	}
	if content1 != "" {
		t.Errorf("期望无正文, 实际 '%s'", content1)
	}

	// 第二个思考块
	thinking2, content2 := processor.Process("<thinking>第二段思考</thinking>")
	if thinking2 != "第二段思考" {
		t.Errorf("期望 '第二段思考', 实际 '%s'", thinking2)
	}
	if content2 != "" {
		t.Errorf("期望无正文, 实际 '%s'", content2)
	}
}

// TestThinkingTagProcessor_缓冲区等于标签前缀 测试缓冲区恰好等于不完整标签
func TestThinkingTagProcessor_缓冲区等于标签前缀(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 缓冲区积累为 "<thinking" (不完整开始标签)
	thinking1, content1 := processor.Process("<")
	if thinking1 != "" || content1 != "" {
		t.Errorf("单字符 '<' 不应产生输出, thinking='%s', content='%s'", thinking1, content1)
	}

	thinking2, content2 := processor.Process("thinking")
	if thinking2 != "" || content2 != "" {
		t.Errorf("不完整的开始标签 '<thinking' 不应产生输出, thinking='%s', content='%s'", thinking2, content2)
	}

	// 补完开始标签
	thinking3, content3 := processor.Process(">思考内容</thinking>正文")
	if thinking3 != "思考内容" {
		t.Errorf("期望思考 '思考内容', 实际 '%s'", thinking3)
	}
	if content3 != "正文" {
		t.Errorf("期望正文 '正文', 实际 '%s'", content3)
	}
}

// TestNewThinkingTagProcessor_初始状态 测试新建处理器的初始状态
func TestNewThinkingTagProcessor_初始状态(t *testing.T) {
	processor := NewThinkingTagProcessor()

	if processor.IsInThinking() {
		t.Error("新建处理器不应处于思考状态")
	}
}

// TestThinkingTagProcessor_仅开始标签 测试仅有开始标签
func TestThinkingTagProcessor_仅开始标签(t *testing.T) {
	processor := NewThinkingTagProcessor()

	thinking, content := processor.Process("<thinking>")
	if thinking != "" {
		t.Errorf("仅有开始标签时应无思考内容, 实际 '%s'", thinking)
	}
	if content != "" {
		t.Errorf("仅有开始标签时应无正文内容, 实际 '%s'", content)
	}
	if !processor.IsInThinking() {
		t.Error("应处于思考状态")
	}
}

// TestThinkingTagProcessor_仅结束标签 测试未匹配的结束标签
func TestThinkingTagProcessor_仅结束标签(t *testing.T) {
	processor := NewThinkingTagProcessor()

	// 不在思考状态下遇到结束标签，应作为普通文本处理
	thinking, content := processor.Process("</thinking>")
	if thinking != "" {
		t.Errorf("未进入思考状态时不应有思考输出, 实际 '%s'", thinking)
	}
	if content != "</thinking>" {
		t.Errorf("应作为普通文本输出, 实际 '%s'", content)
	}
}
