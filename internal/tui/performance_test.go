// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"agentplus/internal/tui/components"
)

// PerformanceTestResult 性能测试结果
type PerformanceTestResult struct {
	Name           string
	Duration       time.Duration
	MemoryAllocMB  float64
	MemoryTotalMB  float64
	Operations     int
	AvgLatencyMs   float64
	P95LatencyMs   float64
	P99LatencyMs   float64
	Pass           bool
	Details        string
}

// TestStreamingPerformance 测试流式输出性能
// 要求: 延迟 < 100ms
func TestStreamingPerformance(t *testing.T) {
	// 创建测试模型
	model := NewAppModel()
	model.width = 80
	model.height = 24

	// 创建测试对话
	conv := &Conversation{
		ID:        "test-conv-1",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages:  make([]Message, 0),
	}
	model.activeConv = conv

	// 测试流式输出延迟
	iterations := 100
	latencies := make([]time.Duration, 0, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()

		// 模拟流式输出
		chunk := fmt.Sprintf("这是第%d次流式输出测试内容，包含一些文本用于测试性能。", i)
		model.streamManager.AddContent(chunk)

		// 强制刷新
		result := model.streamManager.ForceFlush()
		if result.HasData() {
			model.handleBatchUpdate(result)
		}

		latency := time.Since(start)
		latencies = append(latencies, latency)
	}

	// 计算统计数据
	var total time.Duration
	var maxLatency time.Duration
	for _, lat := range latencies {
		total += lat
		if lat > maxLatency {
			maxLatency = lat
		}
	}

	avgLatency := total / time.Duration(iterations)
	p95Latency := calculatePercentile(latencies, 95)
	p99Latency := calculatePercentile(latencies, 99)

	// 验证性能要求
	avgLatencyMs := float64(avgLatency.Microseconds()) / 1000.0
	p95LatencyMs := float64(p95Latency.Microseconds()) / 1000.0
	p99LatencyMs := float64(p99Latency.Microseconds()) / 1000.0

	passed := avgLatencyMs < 100.0

	t.Logf("流式输出性能测试结果:")
	t.Logf("  平均延迟: %.2f ms", avgLatencyMs)
	t.Logf("  P95延迟: %.2f ms", p95LatencyMs)
	t.Logf("  P99延迟: %.2f ms", p99LatencyMs)
	t.Logf("  最大延迟: %.2f ms", float64(maxLatency.Microseconds())/1000.0)
	t.Logf("  测试通过: %v", passed)

	if !passed {
		t.Errorf("流式输出平均延迟 %.2f ms 超过 100ms 要求", avgLatencyMs)
	}
}

// TestScrollPerformance 测试滚动性能
// 要求: 支持 1000+ 条消息，滚动流畅
func TestScrollPerformance(t *testing.T) {
	// 创建测试模型
	model := NewAppModel()
	model.width = 120
	model.height = 40

	// 创建测试对话
	conv := &Conversation{
		ID:        "test-conv-2",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages:  make([]Message, 0),
	}
	model.activeConv = conv

	// 生成大量消息
	messageCount := 1000
	t.Logf("生成 %d 条测试消息...", messageCount)

	start := time.Now()
	for i := 0; i < messageCount; i++ {
		msg := Message{
			Role:      MessageRoleUser,
			Content:   fmt.Sprintf("这是第 %d 条测试消息，包含一些文本内容用于测试滚动性能。消息内容长度适中，模拟真实使用场景。", i+1),
			Timestamp: time.Now(),
		}
		conv.Messages = append(conv.Messages, msg)
	}
	generateDuration := time.Since(start)

	// 测试渲染性能
	start = time.Now()
	content := model.renderConversation(conv)
	renderDuration := time.Since(start)

	// 测试滚动性能
	scrollIterations := 100
	scrollLatencies := make([]time.Duration, 0, scrollIterations)

	for i := 0; i < scrollIterations; i++ {
		start := time.Now()
		model.chatView.LineDown()
		scrollLatencies = append(scrollLatencies, time.Since(start))
	}

	var totalScrollLatency time.Duration
	for _, lat := range scrollLatencies {
		totalScrollLatency += lat
	}
	avgScrollLatency := totalScrollLatency / time.Duration(scrollIterations)

	// 计算内存占用
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memoryMB := float64(memStats.Alloc) / 1024 / 1024

	t.Logf("滚动性能测试结果:")
	t.Logf("  消息数量: %d", messageCount)
	t.Logf("  消息生成时间: %v", generateDuration)
	t.Logf("  渲染时间: %v", renderDuration)
	t.Logf("  平均滚动延迟: %v", avgScrollLatency)
	t.Logf("  内存占用: %.2f MB", memoryMB)
	t.Logf("  内容长度: %d 字符", len(content))

	// 验证性能要求
	renderMs := float64(renderDuration.Microseconds()) / 1000.0
	scrollMs := float64(avgScrollLatency.Microseconds()) / 1000.0

	// 渲染时间应该小于 50ms
	if renderMs > 50.0 {
		t.Errorf("渲染时间 %.2f ms 超过 50ms", renderMs)
	}

	// 滚动延迟应该小于 16ms (60fps)
	if scrollMs > 16.0 {
		t.Errorf("滚动延迟 %.2f ms 超过 16ms", scrollMs)
	}

	// 内存占用应该小于 100MB
	if memoryMB > 100.0 {
		t.Errorf("内存占用 %.2f MB 超过 100MB", memoryMB)
	}
}

// TestSubConversationManagement 测试子对话管理性能
func TestSubConversationManagement(t *testing.T) {
	// 创建对话管理器
	manager := NewConversationManager()

	// 测试创建大量对话
	conversationCount := 100
	start := time.Now()

	// 创建主对话
	mainConv := manager.CreateConversation("主对话")
	convIDs := []string{mainConv.ID}

	// 创建子对话
	for i := 0; i < conversationCount-1; i++ {
		subConv, err := manager.CreateSubConversation(
			mainConv.ID,
			fmt.Sprintf("Agent-%d", i),
			fmt.Sprintf("子任务 %d", i),
		)
		if err != nil {
			t.Errorf("创建子对话失败: %v", err)
		}
		convIDs = append(convIDs, subConv.ID)
	}
	createDuration := time.Since(start)

	// 测试获取所有对话
	start = time.Now()
	_ = manager.GetAllConversations()
	getAllDuration := time.Since(start)

	// 测试切换对话
	switchIterations := 100
	switchLatencies := make([]time.Duration, 0, switchIterations)

	for i := 0; i < switchIterations; i++ {
		convID := convIDs[i%len(convIDs)]
		start := time.Now()
		manager.SwitchConversation(convID)
		switchLatencies = append(switchLatencies, time.Since(start))
	}

	var totalSwitchLatency time.Duration
	for _, lat := range switchLatencies {
		totalSwitchLatency += lat
	}
	avgSwitchLatency := totalSwitchLatency / time.Duration(switchIterations)

	// 测试获取统计信息
	start = time.Now()
	stats := manager.GetStatistics()
	statsDuration := time.Since(start)

	// 计算内存占用
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memoryMB := float64(memStats.Alloc) / 1024 / 1024

	t.Logf("子对话管理性能测试结果:")
	t.Logf("  对话数量: %d", conversationCount)
	t.Logf("  创建时间: %v", createDuration)
	t.Logf("  获取所有对话时间: %v", getAllDuration)
	t.Logf("  平均切换延迟: %v", avgSwitchLatency)
	t.Logf("  获取统计时间: %v", statsDuration)
	t.Logf("  内存占用: %.2f MB", memoryMB)
	t.Logf("  统计信息: %+v", stats)

	// 验证性能要求
	switchMs := float64(avgSwitchLatency.Microseconds()) / 1000.0
	if switchMs > 10.0 {
		t.Errorf("对话切换延迟 %.2f ms 超过 10ms", switchMs)
	}
}

// TestMemoryUsage 测试内存占用
// 要求: 正常使用场景 < 100MB
func TestMemoryUsage(t *testing.T) {
	// 重置内存统计
	runtime.GC()
	runtime.GC()

	var startMemStats runtime.MemStats
	runtime.ReadMemStats(&startMemStats)

	// 创建测试模型
	model := NewAppModel()
	model.width = 120
	model.height = 40

	// 创建多个对话
	for i := 0; i < 10; i++ {
		conv := &Conversation{
			ID:        fmt.Sprintf("test-conv-%d", i),
			CreatedAt: time.Now(),
			Status:    ConvStatusActive,
			Messages:  make([]Message, 0),
		}

		// 每个对话添加 100 条消息
		for j := 0; j < 100; j++ {
			msg := Message{
				Role:      MessageRoleUser,
				Content:   strings.Repeat("测试内容 ", 50), // 约 200 字符
				Thinking:  strings.Repeat("思考内容 ", 30), // 约 120 字符
				Timestamp: time.Now(),
			}
			conv.Messages = append(conv.Messages, msg)
		}

		model.conversations = append(model.conversations, conv)
	}

	// 设置活动对话
	model.activeConv = model.conversations[0]

	// 渲染对话
	_ = model.renderConversation(model.activeConv)

	// 计算内存占用
	var endMemStats runtime.MemStats
	runtime.ReadMemStats(&endMemStats)

	memoryDelta := endMemStats.Alloc - startMemStats.Alloc
	memoryMB := float64(memoryDelta) / 1024 / 1024

	t.Logf("内存占用测试结果:")
	t.Logf("  对话数量: 10")
	t.Logf("  每个对话消息数: 100")
	t.Logf("  总消息数: 1000")
	t.Logf("  内存增量: %.2f MB", memoryMB)
	t.Logf("  当前内存占用: %.2f MB", float64(endMemStats.Alloc)/1024/1024)

	// 验证内存占用
	if memoryMB > 100.0 {
		t.Errorf("内存占用 %.2f MB 超过 100MB", memoryMB)
	}
}

// TestBatchUpdatePerformance 测试批量更新性能
func TestBatchUpdatePerformance(t *testing.T) {
	// 创建测试模型
	model := NewAppModel()
	model.width = 80
	model.height = 24

	// 创建测试对话
	conv := &Conversation{
		ID:        "test-conv-3",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages:  make([]Message, 0),
	}
	model.activeConv = conv

	// 测试批量更新性能
	iterations := 1000
	latencies := make([]time.Duration, 0, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()

		// 添加多条消息到缓冲区
		for j := 0; j < 5; j++ {
			model.streamManager.AddContent(fmt.Sprintf("内容块 %d-%d", i, j))
			model.streamManager.AddThinking(fmt.Sprintf("思考块 %d-%d", i, j))
		}

		// 强制刷新
		result := model.streamManager.ForceFlush()
		if result.HasData() {
			model.handleBatchUpdate(result)
		}

		latency := time.Since(start)
		latencies = append(latencies, latency)
	}

	// 计算统计数据
	var total time.Duration
	for _, lat := range latencies {
		total += lat
	}

	avgLatency := total / time.Duration(iterations)
	p95Latency := calculatePercentile(latencies, 95)
	p99Latency := calculatePercentile(latencies, 99)

	t.Logf("批量更新性能测试结果:")
	t.Logf("  迭代次数: %d", iterations)
	t.Logf("  平均延迟: %v", avgLatency)
	t.Logf("  P95延迟: %v", p95Latency)
	t.Logf("  P99延迟: %v", p99Latency)
}

// TestRenderPerformance 测试渲染性能
func TestRenderPerformance(t *testing.T) {
	// 创建测试模型
	model := NewAppModel()
	model.width = 120
	model.height = 40

	// 创建测试对话
	conv := &Conversation{
		ID:        "test-conv-4",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages:  make([]Message, 0),
	}
	model.activeConv = conv

	// 添加不同类型的消息
	for i := 0; i < 100; i++ {
		// 用户消息
		userMsg := Message{
			Role:      MessageRoleUser,
			Content:   fmt.Sprintf("用户消息 %d: 这是一个测试消息，包含一些文本内容。", i),
			Timestamp: time.Now(),
		}
		conv.Messages = append(conv.Messages, userMsg)

		// 助手消息（包含思考、正文、工具调用）
		assistantMsg := Message{
			Role:      MessageRoleAssistant,
			Thinking:  fmt.Sprintf("思考内容 %d: 这是思考过程...", i),
			Content:   fmt.Sprintf("正文内容 %d: 这是助手的回复内容，包含一些详细说明。", i),
			ToolCalls: []ToolCall{
				{
					ID:        fmt.Sprintf("tool-%d", i),
					Name:      "create_file",
					Arguments: `{"path": "/test/file.go", "content": "package main\n\nfunc main() {}"}`,
					IsComplete: true,
					Result:     "文件创建成功",
				},
			},
			TokenUsage: 150,
			Timestamp:  time.Now(),
		}
		conv.Messages = append(conv.Messages, assistantMsg)
	}

	// 测试渲染性能
	iterations := 100
	latencies := make([]time.Duration, 0, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_ = model.renderConversation(conv)
		latency := time.Since(start)
		latencies = append(latencies, latency)
	}

	// 计算统计数据
	var total time.Duration
	for _, lat := range latencies {
		total += lat
	}

	avgLatency := total / time.Duration(iterations)
	p95Latency := calculatePercentile(latencies, 95)
	p99Latency := calculatePercentile(latencies, 99)

	t.Logf("渲染性能测试结果:")
	t.Logf("  消息数量: %d", len(conv.Messages))
	t.Logf("  迭代次数: %d", iterations)
	t.Logf("  平均延迟: %v", avgLatency)
	t.Logf("  P95延迟: %v", p95Latency)
	t.Logf("  P99延迟: %v", p99Latency)

	// 验证性能要求
	avgMs := float64(avgLatency.Microseconds()) / 1000.0
	if avgMs > 30.0 {
		t.Errorf("渲染平均延迟 %.2f ms 超过 30ms", avgMs)
	}
}

// BenchmarkStreamingOutput 流式输出基准测试
func BenchmarkStreamingOutput(b *testing.B) {
	model := NewAppModel()
	model.width = 80
	model.height = 24

	conv := &Conversation{
		ID:        "bench-conv",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages:  make([]Message, 0),
	}
	model.activeConv = conv

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.streamManager.AddContent("测试内容块")
		result := model.streamManager.ForceFlush()
		if result.HasData() {
			model.handleBatchUpdate(result)
		}
	}
}

// BenchmarkScrolling 滚动基准测试
func BenchmarkScrolling(b *testing.B) {
	model := NewAppModel()
	model.width = 120
	model.height = 40

	conv := &Conversation{
		ID:        "bench-conv",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages:  make([]Message, 0),
	}

	// 生成大量消息
	for i := 0; i < 1000; i++ {
		msg := Message{
			Role:      MessageRoleUser,
			Content:   fmt.Sprintf("消息 %d", i),
			Timestamp: time.Now(),
		}
		conv.Messages = append(conv.Messages, msg)
	}

	model.activeConv = conv
	_ = model.renderConversation(conv)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.chatView.LineDown()
	}
}

// BenchmarkConversationSwitching 对话切换基准测试
func BenchmarkConversationSwitching(b *testing.B) {
	manager := NewConversationManager()

	// 创建多个对话
	convIDs := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		conv := manager.CreateConversation(fmt.Sprintf("对话 %d", i))
		convIDs = append(convIDs, conv.ID)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		convID := convIDs[i%len(convIDs)]
		manager.SwitchConversation(convID)
	}
}

// calculatePercentile 计算百分位数
func calculatePercentile(latencies []time.Duration, percentile int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// 复制切片以避免修改原始数据
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// 简单排序
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// 计算百分位索引
	index := (len(sorted) * percentile) / 100
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// TestComponentPerformance 测试组件性能
func TestComponentPerformance(t *testing.T) {
	// 测试 ChatView 组件性能
	t.Run("ChatView", func(t *testing.T) {
		chatView := components.NewChatView(120, 40)

		// 生成大量消息
		messages := make([]components.MessageData, 0, 1000)
		for i := 0; i < 1000; i++ {
			msg := components.MessageData{
				Role:    "user",
				Content: fmt.Sprintf("消息 %d: 这是一个测试消息，包含一些文本内容用于测试性能。", i),
			}
			messages = append(messages, msg)
		}

		// 测试渲染性能
		start := time.Now()
		rendered := chatView.RenderMessages(messages)
		duration := time.Since(start)

		t.Logf("ChatView 渲染 1000 条消息耗时: %v", duration)
		t.Logf("渲染后内容长度: %d 字符", len(rendered))

		if duration > 50*time.Millisecond {
			t.Errorf("ChatView 渲染时间 %v 超过 50ms", duration)
		}
	})

	// 测试 InputComponent 性能
	t.Run("InputComponent", func(t *testing.T) {
		config := components.DefaultInputComponentConfig()
		input := components.NewInputComponent(config)

		// 测试输入处理性能
		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			input.SetValue(fmt.Sprintf("测试输入 %d", i))
			_ = input.GetValue()
		}

		duration := time.Since(start)
		avgLatency := duration / time.Duration(iterations)

		t.Logf("InputComponent 处理 %d 次输入耗时: %v", iterations, duration)
		t.Logf("平均延迟: %v", avgLatency)

		if avgLatency > 100*time.Microsecond {
			t.Errorf("InputComponent 平均延迟 %v 超过 100μs", avgLatency)
		}
	})

	// 测试 DialogList 性能
	t.Run("DialogList", func(t *testing.T) {
		dialogList := components.NewDialogList()

		// 生成大量对话
		conversations := make([]*components.Conversation, 0, 100)
		for i := 0; i < 100; i++ {
			conv := &components.Conversation{
				ID:           fmt.Sprintf("conv-%d", i),
				CreatedAt:    time.Now(),
				Status:       components.ConvStatusActive,
				Title:        fmt.Sprintf("对话 %d", i),
				MessageCount: i * 10,
				TokenUsage:   i * 100,
			}
			conversations = append(conversations, conv)
		}

		// 测试设置对话列表性能
		start := time.Now()
		dialogList.SetConversations(conversations)
		duration := time.Since(start)

		t.Logf("DialogList 设置 100 个对话耗时: %v", duration)

		if duration > 10*time.Millisecond {
			t.Errorf("DialogList 设置时间 %v 超过 10ms", duration)
		}
	})
}
