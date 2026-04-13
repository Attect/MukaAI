package gui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// --- 注意事项 ---
// gui 包通过 EventEmitter 接口抽象了 runtime.EventsEmit 调用。
// 本文件测试不涉及 EventsEmit 的纯逻辑方法和错误路径。
// 涉及事件发射的完整路径测试在 integration_test.go 中覆盖。
// 测试中使用 noopEventEmitter 避免触发真实的事件发射。

// --- App 基础测试（不涉及 EventsEmit） ---

func TestNewApp(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("NewApp should return non-nil App")
	}
	if app.currentDir == "" {
		t.Error("NewApp should set currentDir")
	}
}

func TestApp_SetAgent(t *testing.T) {
	app := NewApp()
	app.SetAgent(nil)
}

func TestApp_SetCurrentDir(t *testing.T) {
	app := NewApp()
	tmpDir := t.TempDir()
	app.SetCurrentDir(tmpDir)
	if app.currentDir != tmpDir {
		t.Errorf("expected currentDir=%s, got %s", tmpDir, app.currentDir)
	}
}

func TestApp_GetWorkDir(t *testing.T) {
	app := NewApp()
	tmpDir := t.TempDir()
	app.SetCurrentDir(tmpDir)
	if got := app.GetWorkDir(); got != tmpDir {
		t.Errorf("GetWorkDir() = %s, want %s", got, tmpDir)
	}
}

func TestApp_GetTokenStats(t *testing.T) {
	app := NewApp()
	stats := app.GetTokenStats()
	if stats.TotalTokens != 0 {
		t.Errorf("expected TotalTokens=0, got %d", stats.TotalTokens)
	}
	if stats.InferenceCount != 0 {
		t.Errorf("expected InferenceCount=0, got %d", stats.InferenceCount)
	}
}

func TestApp_SetConfigPath(t *testing.T) {
	app := NewApp()
	app.SetConfigPath("/some/path/config.yaml")
	if app.configPath != "/some/path/config.yaml" {
		t.Errorf("expected configPath=/some/path/config.yaml, got %s", app.configPath)
	}
}

func TestApp_GetTerminalWSUrl(t *testing.T) {
	app := NewApp()
	if url := app.GetTerminalWSUrl(); url != "" {
		t.Errorf("expected empty url, got %s", url)
	}
	app.SetTerminalWSUrl("ws://localhost:8080")
	if url := app.GetTerminalWSUrl(); url != "ws://localhost:8080" {
		t.Errorf("expected ws://localhost:8080, got %s", url)
	}
}

func TestApp_Shutdown_NilStore(t *testing.T) {
	app := NewApp()
	app.Shutdown() // 不应panic
}

// --- Settings 测试（不涉及 EventsEmit） ---

func TestApp_GetSettingsEmptyPath(t *testing.T) {
	app := NewApp()
	settings := app.GetSettings()
	if len(settings) != 0 {
		t.Errorf("expected empty settings, got %v", settings)
	}
}

func TestApp_GetSettingsWithFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
model:
  endpoint: "http://localhost:8080/v1/"
  api_key: "test-key"
  model_name: "test-model"
  context_size: 100000
agent:
  temperature: 0.7
  max_iterations: 50
tools:
  work_dir: "./work"
  allow_commands:
    - "go"
    - "git"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	app := NewApp()
	app.SetConfigPath(configPath)

	settings := app.GetSettings()
	if settings["endpoint"] != "http://localhost:8080/v1/" {
		t.Errorf("expected endpoint=http://localhost:8080/v1/, got %v", settings["endpoint"])
	}
	if settings["model_name"] != "test-model" {
		t.Errorf("expected model_name=test-model, got %v", settings["model_name"])
	}
	if settings["temperature"] != 0.7 {
		t.Errorf("expected temperature=0.7, got %v", settings["temperature"])
	}
	if settings["work_dir"] != "./work" {
		t.Errorf("expected work_dir=./work, got %v", settings["work_dir"])
	}
}

func TestApp_SaveSettings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `model:
  endpoint: "http://old:8080/v1/"
  api_key: "old-key"
agent:
  temperature: 0.5
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	app := NewApp()
	app.SetConfigPath(configPath)

	settings := map[string]interface{}{
		"endpoint":   "http://new:9090/v1/",
		"model_name": "new-model",
	}

	if err := app.SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if string(data) == "" {
		t.Error("config file should not be empty after save")
	}
	// SaveSettings使用yaml.Marshal写入，验证非空且包含更新
	s := string(data)
	if !containsSubstring(s, "http://new:9090/v1/") {
		t.Errorf("expected updated endpoint in config, got:\n%s", s)
	}
	if !containsSubstring(s, "new-model") {
		t.Errorf("expected new-model in config, got:\n%s", s)
	}
}

func TestApp_SaveSettingsNoPath(t *testing.T) {
	app := NewApp()
	err := app.SaveSettings(map[string]interface{}{"key": "value"})
	if err == nil {
		t.Error("expected error when configPath is empty")
	}
}

func TestApp_GetSettingsInvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte("not-yaml: [invalid"), 0644)

	app := NewApp()
	app.SetConfigPath(configPath)
	settings := app.GetSettings()
	if len(settings) != 0 {
		t.Errorf("expected empty settings for invalid YAML, got %v", settings)
	}
}

// --- Conversation 读取测试（不触发 EventsEmit） ---

func newTestAppWithConversations() *App {
	app := NewApp()
	app.ctx = context.Background()
	app.eventEmitter = &noopEventEmitter{}
	app.conversations = []*conversation{
		{
			id:        "conv-1",
			title:     "对话1",
			createdAt: time.Now().Add(-2 * time.Hour),
			status:    "active",
			messages: []*message{
				{role: "user", content: "hello", timestamp: time.Now()},
				{role: "assistant", content: "hi", timestamp: time.Now()},
			},
		},
		{
			id:        "conv-2",
			title:     "对话2",
			createdAt: time.Now().Add(-1 * time.Hour),
			status:    "active",
			messages: []*message{
				{role: "user", content: "world", timestamp: time.Now()},
			},
		},
	}
	app.activeConvID = "conv-1"
	return app
}

func TestApp_GetConversations(t *testing.T) {
	app := newTestAppWithConversations()
	convs := app.GetConversations()
	if len(convs) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(convs))
	}
	if convs[0].ID != "conv-1" {
		t.Errorf("expected first conv ID=conv-1, got %s", convs[0].ID)
	}
	if convs[0].MessageCount != 2 {
		t.Errorf("expected first conv MessageCount=2, got %d", convs[0].MessageCount)
	}
	if convs[1].MessageCount != 1 {
		t.Errorf("expected second conv MessageCount=1, got %d", convs[1].MessageCount)
	}
}

func TestApp_GetConversationData(t *testing.T) {
	app := newTestAppWithConversations()
	data := app.GetConversationData()

	if data["id"] != "conv-1" {
		t.Errorf("expected id=conv-1, got %v", data["id"])
	}

	messages, ok := data["messages"].([]Message)
	if !ok {
		t.Fatal("expected messages to be []Message")
	}
	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
	// 验证消息字段映射
	if messages[0].Role != "user" {
		t.Errorf("expected role=user, got %s", messages[0].Role)
	}
	if messages[0].Content != "hello" {
		t.Errorf("expected content=hello, got %s", messages[0].Content)
	}
}

func TestApp_GetConversationDataNoActive(t *testing.T) {
	app := NewApp()
	app.ctx = context.Background()
	data := app.GetConversationData()

	messages, ok := data["messages"].([]Message)
	if !ok {
		t.Fatal("expected messages to be []Message")
	}
	if len(messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(messages))
	}
}

func TestApp_GetConversationDataWithToolCalls(t *testing.T) {
	app := NewApp()
	app.ctx = context.Background()
	app.conversations = []*conversation{
		{
			id:        "conv-1",
			title:     "带工具调用的对话",
			createdAt: time.Now(),
			status:    "active",
			messages: []*message{
				{
					role:      "assistant",
					content:   "读取文件中...",
					timestamp: time.Now(),
					toolCalls: []ToolCall{
						{ID: "call-1", Name: "read_file", Arguments: "{}", IsComplete: true, Result: "file content"},
					},
				},
			},
		},
	}
	app.activeConvID = "conv-1"

	data := app.GetConversationData()
	messages := data["messages"].([]Message)
	if len(messages[0].ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(messages[0].ToolCalls))
	}
	if messages[0].ToolCalls[0].Name != "read_file" {
		t.Errorf("expected tool name=read_file, got %s", messages[0].ToolCalls[0].Name)
	}
}

// --- 错误路径测试（不触发 EventsEmit 的路径） ---

func TestApp_SwitchConversationNotFound(t *testing.T) {
	app := newTestAppWithConversations()
	err := app.SwitchConversation("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent conversation")
	}
}

func TestApp_SwitchConversationWhileStreaming(t *testing.T) {
	app := newTestAppWithConversations()
	app.isStreaming = true
	err := app.SwitchConversation("conv-2")
	if err == nil {
		t.Error("expected error when switching during streaming")
	}
}

func TestApp_DeleteConversationNotFound(t *testing.T) {
	app := newTestAppWithConversations()
	err := app.DeleteConversation("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent conversation")
	}
}

func TestApp_DeleteConversationWhileStreaming(t *testing.T) {
	app := newTestAppWithConversations()
	app.isStreaming = true
	err := app.DeleteConversation("conv-1")
	if err == nil {
		t.Error("expected error when deleting during streaming")
	}
}

func TestApp_SendMessageNoAgent(t *testing.T) {
	app := NewApp()
	err := app.SendMessage("test")
	if err == nil {
		t.Error("expected error when agent is nil")
	}
}

func TestApp_SendMessageWhileStreaming(t *testing.T) {
	app := NewApp()
	app.isStreaming = true
	err := app.SendMessage("test")
	if err == nil {
		t.Error("expected error when already streaming")
	}
}

func TestApp_SetWorkDirNonExistent(t *testing.T) {
	app := NewApp()
	err := app.SetWorkDir("/nonexistent/path/xyz")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestApp_SetWorkDirIsFile(t *testing.T) {
	app := NewApp()
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	os.WriteFile(tmpFile, []byte("test"), 0644)
	err := app.SetWorkDir(tmpFile)
	if err == nil {
		t.Error("expected error when path is a file")
	}
}

func TestApp_UpdateConversationTitleNotFound(t *testing.T) {
	app := newTestAppWithConversations()
	err := app.UpdateConversationTitle("nonexistent", "title")
	if err == nil {
		t.Error("expected error for nonexistent conversation")
	}
}

func TestApp_ExportConversationNotFound(t *testing.T) {
	app := newTestAppWithConversations()
	err := app.ExportConversation("nonexistent", "")
	if err == nil {
		t.Error("expected error for nonexistent conversation")
	}
}

// --- 内部方法测试 ---

func TestApp_getOrCreateActiveConversation_CreatesNew(t *testing.T) {
	app := NewApp()
	app.mu.Lock()
	conv := app.getOrCreateActiveConversation()
	app.mu.Unlock()

	if conv == nil {
		t.Fatal("expected non-nil conversation")
	}
	if conv.id == "" {
		t.Error("expected non-empty conversation ID")
	}
	if conv.title != "新对话" {
		t.Errorf("expected title=新对话, got %s", conv.title)
	}
	if len(app.conversations) != 1 {
		t.Errorf("expected 1 conversation, got %d", len(app.conversations))
	}
}

func TestApp_getOrCreateActiveConversation_ReturnsExisting(t *testing.T) {
	app := newTestAppWithConversations()
	app.mu.Lock()
	conv := app.getOrCreateActiveConversation()
	app.mu.Unlock()

	if conv.id != "conv-1" {
		t.Errorf("expected conv-1, got %s", conv.id)
	}
}

func TestApp_getActiveConversation(t *testing.T) {
	app := newTestAppWithConversations()
	conv := app.getActiveConversation()
	if conv == nil || conv.id != "conv-1" {
		t.Errorf("expected conv-1, got %v", conv)
	}
}

func TestApp_getActiveConversation_NoMatch(t *testing.T) {
	app := NewApp()
	app.conversations = []*conversation{
		{id: "conv-1"},
	}
	app.activeConvID = "nonexistent"
	conv := app.getActiveConversation()
	if conv != nil {
		t.Errorf("expected nil for no match, got %v", conv)
	}
}

// --- ConversationStore 测试（不涉及 EventsEmit） ---

func TestConversationStore_CreateAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConversationStore(tmpDir)
	if err != nil {
		t.Fatalf("NewConversationStore failed: %v", err)
	}
	defer store.Close()

	conv := &conversation{
		id:         "test-conv-1",
		title:      "测试对话",
		createdAt:  time.Now(),
		status:     "active",
		tokenUsage: 100,
		messages: []*message{
			{role: "user", content: "你好", timestamp: time.Now()},
			{role: "assistant", content: "你好！", timestamp: time.Now()},
		},
	}

	store.SaveConversation(conv)
	time.Sleep(600 * time.Millisecond)

	store2, err := NewConversationStore(tmpDir)
	if err != nil {
		t.Fatalf("NewConversationStore (reload) failed: %v", err)
	}
	defer store2.Close()

	convs, err := store2.LoadAllConversations()
	if err != nil {
		t.Fatalf("LoadAllConversations failed: %v", err)
	}
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(convs))
	}
	if convs[0].id != "test-conv-1" {
		t.Errorf("expected id=test-conv-1, got %s", convs[0].id)
	}
	if convs[0].title != "测试对话" {
		t.Errorf("expected title=测试对话, got %s", convs[0].title)
	}
	if convs[0].tokenUsage != 100 {
		t.Errorf("expected tokenUsage=100, got %d", convs[0].tokenUsage)
	}
	if len(convs[0].messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(convs[0].messages))
	}
	if convs[0].messages[0].role != "user" {
		t.Errorf("expected first message role=user, got %s", convs[0].messages[0].role)
	}
}

func TestConversationStore_CreateDir(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "nested", "dir")
	store, err := NewConversationStore(tmpDir)
	if err != nil {
		t.Fatalf("NewConversationStore should create nested dirs: %v", err)
	}
	store.Close()
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("expected directory to be created")
	}
}

func TestConversationStore_SaveWithToolCalls(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConversationStore(tmpDir)
	if err != nil {
		t.Fatalf("NewConversationStore failed: %v", err)
	}
	defer store.Close()

	conv := &conversation{
		id:        "conv-tools",
		title:     "工具调用对话",
		createdAt: time.Now(),
		status:    "active",
		messages: []*message{
			{
				role:      "assistant",
				content:   "使用工具",
				timestamp: time.Now(),
				toolCalls: []ToolCall{
					{ID: "call-1", Name: "read_file", Arguments: `{"path":"test"}`, IsComplete: true, Result: "content"},
					{ID: "call-2", Name: "write_file", Arguments: `{}`, IsComplete: false, ResultError: "permission denied"},
				},
			},
		},
	}

	store.SaveConversation(conv)
	time.Sleep(600 * time.Millisecond)

	store2, _ := NewConversationStore(tmpDir)
	defer store2.Close()

	convs, _ := store2.LoadAllConversations()
	if len(convs) != 1 || len(convs[0].messages) != 1 {
		t.Fatal("expected 1 conversation with 1 message")
	}
	if len(convs[0].messages[0].toolCalls) != 2 {
		t.Errorf("expected 2 toolCalls, got %d", len(convs[0].messages[0].toolCalls))
	}
	if convs[0].messages[0].toolCalls[0].Name != "read_file" {
		t.Errorf("expected tool name=read_file, got %s", convs[0].messages[0].toolCalls[0].Name)
	}
	if convs[0].messages[0].toolCalls[1].ResultError != "permission denied" {
		t.Errorf("expected ResultError=permission denied, got %s", convs[0].messages[0].toolCalls[1].ResultError)
	}
}

func TestConversationStore_DeleteConversation(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewConversationStore(tmpDir)
	if err != nil {
		t.Fatalf("NewConversationStore failed: %v", err)
	}
	defer store.Close()

	conv := &conversation{id: "to-delete", title: "待删除", createdAt: time.Now(), status: "active"}
	store.SaveConversation(conv)
	time.Sleep(600 * time.Millisecond)

	if err := store.DeleteConversation("to-delete"); err != nil {
		t.Fatalf("DeleteConversation failed: %v", err)
	}

	convs, _ := store.LoadAllConversations()
	if len(convs) != 0 {
		t.Errorf("expected 0 conversations after delete, got %d", len(convs))
	}
}

func TestConversationStore_DeleteNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewConversationStore(tmpDir)
	defer store.Close()

	err := store.DeleteConversation("nonexistent")
	if err != nil {
		t.Errorf("DeleteConversation should not fail for nonexistent: %v", err)
	}
}

func TestConversationStore_SaveNil(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewConversationStore(tmpDir)
	defer store.Close()

	store.SaveConversation(nil)
	store.SaveConversation(&conversation{})
	// 不应panic
}

func TestConversationStore_LoadEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewConversationStore(tmpDir)
	defer store.Close()

	convs, err := store.LoadAllConversations()
	if err != nil {
		t.Fatalf("LoadAllConversations failed: %v", err)
	}
	if len(convs) != 0 {
		t.Errorf("expected 0 conversations, got %d", len(convs))
	}
}

func TestConversationStore_CloseFlushesPending(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewConversationStore(tmpDir)

	conv := &conversation{
		id: "flush-test", title: "刷新测试", createdAt: time.Now(), status: "active",
		messages: []*message{{role: "user", content: "test", timestamp: time.Now()}},
	}

	store.SaveConversation(conv)
	store.Close() // 不等待防抖

	data, err := os.ReadFile(filepath.Join(tmpDir, "flush-test.json"))
	if err != nil {
		t.Fatalf("expected file to exist after Close: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty file content")
	}
}

func TestConversationStore_LoadSorted(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewConversationStore(tmpDir)

	now := time.Now()
	convs := []*conversation{
		{id: "c-newest", title: "最新", createdAt: now.Add(2 * time.Hour), status: "active"},
		{id: "c-oldest", title: "最旧", createdAt: now, status: "active"},
		{id: "c-middle", title: "中间", createdAt: now.Add(1 * time.Hour), status: "active"},
	}

	for _, c := range convs {
		store.SaveConversation(c)
	}
	store.Close()

	store2, _ := NewConversationStore(tmpDir)
	defer store2.Close()

	loaded, _ := store2.LoadAllConversations()
	if len(loaded) != 3 {
		t.Fatalf("expected 3 conversations, got %d", len(loaded))
	}
	if !sort.SliceIsSorted(loaded, func(i, j int) bool {
		return loaded[i].createdAt.Before(loaded[j].createdAt)
	}) {
		t.Error("conversations should be sorted by createdAt ascending")
	}
}

func TestConversationStore_BadVersion(t *testing.T) {
	tmpDir := t.TempDir()
	badData := `{"version":999,"id":"bad","title":"bad","createdAt":"2024-01-01T00:00:00Z","status":"active","tokenUsage":0,"messages":[]}`
	os.WriteFile(filepath.Join(tmpDir, "bad.json"), []byte(badData), 0644)

	store, _ := NewConversationStore(tmpDir)
	defer store.Close()

	convs, err := store.LoadAllConversations()
	if err != nil {
		t.Fatalf("LoadAllConversations should not fail: %v", err)
	}
	if len(convs) != 0 {
		t.Errorf("expected 0 conversations (bad version skipped), got %d", len(convs))
	}
}

func TestConversationStore_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "invalid.json"), []byte("not-json"), 0644)

	store, _ := NewConversationStore(tmpDir)
	defer store.Close()

	convs, err := store.LoadAllConversations()
	if err != nil {
		t.Fatalf("LoadAllConversations should not fail: %v", err)
	}
	if len(convs) != 0 {
		t.Errorf("expected 0 conversations (invalid JSON skipped), got %d", len(convs))
	}
}

func TestConversationStore_SkipsDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "subdir.json"), 0755) // 名为.json的目录
	store, _ := NewConversationStore(tmpDir)
	defer store.Close()

	convs, err := store.LoadAllConversations()
	if err != nil {
		t.Fatalf("LoadAllConversations failed: %v", err)
	}
	if len(convs) != 0 {
		t.Errorf("expected 0 conversations (directories skipped), got %d", len(convs))
	}
}

func TestConversationStore_Debounce(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewConversationStore(tmpDir)
	defer store.Close()

	conv := &conversation{id: "debounce-test", title: "初始", createdAt: time.Now(), status: "active"}

	// 快速多次保存
	for i := 0; i < 5; i++ {
		conv.title = fmt.Sprintf("标题%d", i)
		store.SaveConversation(conv)
	}
	time.Sleep(600 * time.Millisecond)

	store2, _ := NewConversationStore(tmpDir)
	defer store2.Close()

	convs, _ := store2.LoadAllConversations()
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation (debounced), got %d", len(convs))
	}
	if convs[0].title != "标题4" {
		t.Errorf("expected last title '标题4', got %s", convs[0].title)
	}
}

// --- StreamBridge 构造测试 ---

func TestNewStreamBridge(t *testing.T) {
	app := NewApp()
	bridge := NewStreamBridge(app)
	if bridge == nil {
		t.Fatal("NewStreamBridge should return non-nil")
	}
}

func TestStreamBridge_SetContext(t *testing.T) {
	app := NewApp()
	bridge := NewStreamBridge(app)
	ctx := context.Background()
	bridge.SetContext(ctx)
	// 验证App的上下文已更新
	if app.ctx != ctx {
		t.Error("SetContext should set app.ctx")
	}
	// 验证App的EventEmitter已创建
	app.mu.RLock()
	emitter := app.eventEmitter
	app.mu.RUnlock()
	if emitter == nil {
		t.Error("SetContext should create EventEmitter")
	}
}

// --- StreamBridge 内部状态更新测试 ---
// 注意：这些测试使用Wails context，会触发EventsEmit。
// 由于Wails runtime在非Wails环境下调用log.Fatalf会导致进程退出，
// 我们使用一个子进程来运行这些测试。
// 只有当app.ctx中没有events时，才会crash。
// 解决方案：在这些测试中不设置app.ctx（保持nil），
// 让StreamBridge操作app内部状态时不触发EventsEmit。
// 但stream_bridge.go的每个方法最后都调用了runtime.EventsEmit...
// 所以这些测试只能在Wails E2E环境中运行。

// --- 辅助函数 ---

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
