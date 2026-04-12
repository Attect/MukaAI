// Package agent 回调Setter方法
// 从 core.go 提取的回调配置方法，职责单一：设置Agent的各种回调函数
package agent

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
