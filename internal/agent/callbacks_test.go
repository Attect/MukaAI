package agent

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Attect/MukaAI/internal/model"
	"github.com/Attect/MukaAI/internal/state"
	"github.com/Attect/MukaAI/internal/tools"
)

// setupMinimalAgent 创建最小化的Agent实例，用于回调测试
func setupMinimalAgent(t *testing.T) (*Agent, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "agent-callback-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	stateManager, err := state.NewStateManager(filepath.Join(tmpDir, "state"), true)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("创建状态管理器失败: %v", err)
	}

	registry := tools.NewToolRegistry()
	server := mockAPIServer(nil)

	config := model.DefaultConfig()
	config.Endpoint = server.URL + "/"
	config.APIKey = "test-key"
	config.ModelName = "test-model"

	client, err := model.NewClient(config)
	if err != nil {
		server.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("创建模型客户端失败: %v", err)
	}

	agent, err := NewAgent(&Config{
		ModelClient:   client,
		ToolRegistry:  registry,
		StateManager:  stateManager,
		MaxIterations: 5,
	})
	if err != nil {
		server.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("创建Agent失败: %v", err)
	}

	cleanup := func() {
		server.Close()
		os.RemoveAll(tmpDir)
	}

	return agent, cleanup
}

// TestSetOnStreamChunk_基本功能 测试流式输出回调设置
func TestSetOnStreamChunk_基本功能(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange
	var called bool
	var received string
	fn := func(chunk string) {
		called = true
		received = chunk
	}

	// act
	agent.SetOnStreamChunk(fn)
	agent.onStreamChunk("test chunk")

	// assert
	if !called {
		t.Error("回调未被调用")
	}
	if received != "test chunk" {
		t.Errorf("期望收到 'test chunk', 实际收到 '%s'", received)
	}
}

// TestSetOnToolCall_基本功能 测试工具调用回调设置
func TestSetOnToolCall_基本功能(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange
	var called bool
	var receivedName, receivedArgs string
	fn := func(name, args string) {
		called = true
		receivedName = name
		receivedArgs = args
	}

	// act
	agent.SetOnToolCall(fn)
	agent.onToolCall("read_file", `{"path": "test.txt"}`)

	// assert
	if !called {
		t.Error("回调未被调用")
	}
	if receivedName != "read_file" {
		t.Errorf("期望工具名 'read_file', 实际 '%s'", receivedName)
	}
	if receivedArgs != `{"path": "test.txt"}` {
		t.Errorf("期望参数 '{\"path\": \"test.txt\"}', 实际 '%s'", receivedArgs)
	}
}

// TestSetOnIteration_基本功能 测试迭代回调设置
func TestSetOnIteration_基本功能(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange
	var received int
	fn := func(iteration int) {
		received = iteration
	}

	// act
	agent.SetOnIteration(fn)
	agent.onIteration(5)

	// assert
	if received != 5 {
		t.Errorf("期望迭代数 5, 实际 %d", received)
	}
}

// TestSetOnToolResult_基本功能 测试工具结果回调设置
func TestSetOnToolResult_基本功能(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange
	var receivedName, receivedResult string
	fn := func(name, resultJSON string) {
		receivedName = name
		receivedResult = resultJSON
	}

	// act
	agent.SetOnToolResult(fn)
	agent.onToolResult("write_file", `{"success": true}`)

	// assert
	if receivedName != "write_file" {
		t.Errorf("期望工具名 'write_file', 实际 '%s'", receivedName)
	}
	if receivedResult != `{"success": true}` {
		t.Errorf("期望结果 '{\"success\": true}', 实际 '%s'", receivedResult)
	}
}

// TestSetOnToolCallFull_基本功能 测试完整工具调用回调设置
func TestSetOnToolCallFull_基本功能(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange
	var receivedID, receivedName, receivedArgs string
	fn := func(toolCallID, name, args string) {
		receivedID = toolCallID
		receivedName = name
		receivedArgs = args
	}

	// act
	agent.SetOnToolCallFull(fn)
	agent.onToolCallFull("call-001", "read_file", `{"path":"a.go"}`)

	// assert
	if receivedID != "call-001" {
		t.Errorf("期望 ID 'call-001', 实际 '%s'", receivedID)
	}
	if receivedName != "read_file" {
		t.Errorf("期望工具名 'read_file', 实际 '%s'", receivedName)
	}
	if receivedArgs != `{"path":"a.go"}` {
		t.Errorf("期望参数 '{\"path\":\"a.go\"}', 实际 '%s'", receivedArgs)
	}
}

// TestSetOnReview_基本功能 测试审查结果回调设置
func TestSetOnReview_基本功能(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange
	var receivedStatus, receivedSummary string
	fn := func(status, summary string) {
		receivedStatus = status
		receivedSummary = summary
	}

	// act
	agent.SetOnReview(fn)
	agent.onReview("pass", "审查通过")

	// assert
	if receivedStatus != "pass" {
		t.Errorf("期望状态 'pass', 实际 '%s'", receivedStatus)
	}
	if receivedSummary != "审查通过" {
		t.Errorf("期望摘要 '审查通过', 实际 '%s'", receivedSummary)
	}
}

// TestSetOnVerify_基本功能 测试校验结果回调设置
func TestSetOnVerify_基本功能(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange
	var receivedStatus, receivedSummary string
	fn := func(status, summary string) {
		receivedStatus = status
		receivedSummary = summary
	}

	// act
	agent.SetOnVerify(fn)
	agent.onVerify("fail", "文件不存在")

	// assert
	if receivedStatus != "fail" {
		t.Errorf("期望状态 'fail', 实际 '%s'", receivedStatus)
	}
	if receivedSummary != "文件不存在" {
		t.Errorf("期望摘要 '文件不存在', 实际 '%s'", receivedSummary)
	}
}

// TestSetOnCorrection_基本功能 测试修正指令回调设置
func TestSetOnCorrection_基本功能(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange
	var received string
	fn := func(instruction string) {
		received = instruction
	}

	// act
	agent.SetOnCorrection(fn)
	agent.onCorrection("请修正文件内容")

	// assert
	if received != "请修正文件内容" {
		t.Errorf("期望 '请修正文件内容', 实际 '%s'", received)
	}
}

// TestSetOnNoToolCall_基本功能 测试无工具调用回调设置
func TestSetOnNoToolCall_基本功能(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange
	var receivedCount int
	var receivedResponse string
	fn := func(count int, response string) {
		receivedCount = count
		receivedResponse = response
	}

	// act
	agent.SetOnNoToolCall(fn)
	agent.onNoToolCall(3, "正在思考...")

	// assert
	if receivedCount != 3 {
		t.Errorf("期望次数 3, 实际 %d", receivedCount)
	}
	if receivedResponse != "正在思考..." {
		t.Errorf("期望 '正在思考...', 实际 '%s'", receivedResponse)
	}
}

// TestSetOnHistoryAdd_基本功能 测试消息历史添加回调设置
func TestSetOnHistoryAdd_基本功能(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange
	var receivedRole, receivedContent string
	fn := func(role, content string) {
		receivedRole = role
		receivedContent = content
	}

	// act
	agent.SetOnHistoryAdd(fn)
	agent.onHistoryAdd("user", "继续执行")

	// assert
	if receivedRole != "user" {
		t.Errorf("期望角色 'user', 实际 '%s'", receivedRole)
	}
	if receivedContent != "继续执行" {
		t.Errorf("期望 '继续执行', 实际 '%s'", receivedContent)
	}
}

// TestSetOnThinking_基本功能 测试思考内容回调设置
func TestSetOnThinking_基本功能(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange
	var received string
	fn := func(thinking string) {
		received = thinking
	}

	// act
	agent.SetOnThinking(fn)
	agent.onThinking("正在分析任务...")

	// assert
	if received != "正在分析任务..." {
		t.Errorf("期望 '正在分析任务...', 实际 '%s'", received)
	}
}

// TestSetStreamHandler_基本功能 测试流式消息处理器设置
func TestSetStreamHandler_基本功能(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange
	var contentReceived string
	handler := NewStreamHandlerFunc().
		OnContent(func(chunk string) {
			contentReceived = chunk
		}).
		Build()

	// act
	agent.SetStreamHandler(handler)
	handlerFromAgent := agent.GetStreamHandler()

	// assert
	if handlerFromAgent == nil {
		t.Fatal("获取到的handler不应为nil")
	}
	handlerFromAgent.OnContent("test")
	if contentReceived != "test" {
		t.Errorf("期望 'test', 实际 '%s'", contentReceived)
	}
}

// TestCallbacks_并发安全 测试回调的并发读写安全
func TestCallbacks_并发安全(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange
	var wg sync.WaitGroup
	callCount := 0
	var mu sync.Mutex

	fn := func(chunk string) {
		mu.Lock()
		callCount++
		mu.Unlock()
	}

	// act - 并发设置和读取回调
	for i := 0; i < 10; i++ {
		wg.Add(2)

		// 并发设置回调
		go func() {
			defer wg.Done()
			agent.SetOnStreamChunk(fn)
		}()

		// 并发读取并调用回调
		go func() {
			defer wg.Done()
			agent.SetOnStreamChunk(fn)
			if agent.onStreamChunk != nil {
				agent.onStreamChunk("test")
			}
		}()
	}

	wg.Wait()

	// assert - 不应panic
	mu.Lock()
	if callCount == 0 {
		t.Error("回调应该被调用至少一次")
	}
	mu.Unlock()
}

// TestCallbacks_覆盖设置 测试回调可以被覆盖
func TestCallbacks_覆盖设置(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange
	var firstCalled, secondCalled bool

	// act
	agent.SetOnStreamChunk(func(chunk string) { firstCalled = true })
	agent.SetOnStreamChunk(func(chunk string) { secondCalled = true })
	agent.onStreamChunk("test")

	// assert
	if firstCalled {
		t.Error("第一次设置的回调不应被调用")
	}
	if !secondCalled {
		t.Error("第二次设置的回调应被调用")
	}
}

// TestCallbacks_设置为Nil 测试回调可以设置为nil
func TestCallbacks_设置为Nil(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange & act
	agent.SetOnStreamChunk(func(chunk string) {})
	agent.SetOnStreamChunk(nil)

	// assert - 不应panic
	if agent.onStreamChunk != nil {
		t.Error("回调应被设为nil")
	}
}

// TestSetMultipleCallbacks_同时设置多个回调 测试同时设置多个回调互不干扰
func TestSetMultipleCallbacks_同时设置多个回调(t *testing.T) {
	agent, cleanup := setupMinimalAgent(t)
	defer cleanup()

	// arrange
	var streamChunkCalled, toolCallCalled, iterationCalled bool

	// act
	agent.SetOnStreamChunk(func(chunk string) { streamChunkCalled = true })
	agent.SetOnToolCall(func(name, args string) { toolCallCalled = true })
	agent.SetOnIteration(func(iteration int) { iterationCalled = true })

	agent.onToolCall("test", "{}")

	// assert
	if streamChunkCalled {
		t.Error("streamChunk回调不应被调用")
	}
	if !toolCallCalled {
		t.Error("toolCall回调应被调用")
	}
	if iterationCalled {
		t.Error("iteration回调不应被调用")
	}
}
