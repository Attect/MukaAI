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
