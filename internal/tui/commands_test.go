package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSaveCommand 测试 /save 命令
func TestSaveCommand(t *testing.T) {
	// 创建测试对话
	conv := &Conversation{
		ID:        "test-conv-123",
		Title:     "Test Conversation",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages: []Message{
			{
				Role:      MessageRoleUser,
				Content:   "Hello",
				Timestamp: time.Now(),
			},
			{
				Role:      MessageRoleAssistant,
				Content:   "Hi there!",
				Timestamp: time.Now(),
			},
		},
		TokenUsage: 100,
	}

	// 测试保存到默认路径
	t.Run("SaveToDefaultPath", func(t *testing.T) {
		savedPath, err := SaveConversationToFile("", conv)
		if err != nil {
			t.Fatalf("保存对话失败: %v", err)
		}

		// 验证文件存在
		if _, err := os.Stat(savedPath); os.IsNotExist(err) {
			t.Fatalf("保存的文件不存在: %s", savedPath)
		}

		// 清理
		defer os.Remove(savedPath)

		// 验证文件内容
		loadedConv, err := LoadConversationFromFile(savedPath)
		if err != nil {
			t.Fatalf("加载对话失败: %v", err)
		}

		if loadedConv.ID != conv.ID {
			t.Errorf("对话 ID 不匹配: got %v, want %v", loadedConv.ID, conv.ID)
		}

		if loadedConv.Title != conv.Title {
			t.Errorf("对话标题不匹配: got %v, want %v", loadedConv.Title, conv.Title)
		}

		if len(loadedConv.Messages) != len(conv.Messages) {
			t.Errorf("消息数量不匹配: got %v, want %v", len(loadedConv.Messages), len(conv.Messages))
		}
	})

	// 测试保存到指定路径
	t.Run("SaveToSpecificPath", func(t *testing.T) {
		// 创建临时目录
		tmpDir, err := os.MkdirTemp("", "tui-test")
		if err != nil {
			t.Fatalf("创建临时目录失败: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// 指定文件路径
		filePath := filepath.Join(tmpDir, "test-conversation.json")
		savedPath, err := SaveConversationToFile(filePath, conv)
		if err != nil {
			t.Fatalf("保存对话失败: %v", err)
		}

		// 验证路径正确
		if savedPath != filePath {
			t.Errorf("保存路径不匹配: got %v, want %v", savedPath, filePath)
		}

		// 验证文件存在
		if _, err := os.Stat(savedPath); os.IsNotExist(err) {
			t.Fatalf("保存的文件不存在: %s", savedPath)
		}
	})

	// 测试保存空对话
	t.Run("SaveNilConversation", func(t *testing.T) {
		_, err := SaveConversationToFile("", nil)
		if err == nil {
			t.Error("保存空对话应该返回错误")
		}
	})
}

// TestExportConversation 测试对话导出功能
func TestExportConversation(t *testing.T) {
	conv := &Conversation{
		ID:        "export-test-123",
		Title:     "Export Test",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages: []Message{
			{
				Role:      MessageRoleUser,
				Content:   "Test message",
				Timestamp: time.Now(),
			},
		},
		TokenUsage: 50,
	}

	export, err := ExportConversation(conv)
	if err != nil {
		t.Fatalf("导出对话失败: %v", err)
	}

	// 验证导出数据
	if export.ID != conv.ID {
		t.Errorf("ID 不匹配: got %v, want %v", export.ID, conv.ID)
	}

	if export.Title != conv.Title {
		t.Errorf("标题不匹配: got %v, want %v", export.Title, conv.Title)
	}

	if export.Status != "active" {
		t.Errorf("状态不匹配: got %v, want %v", export.Status, "active")
	}

	if len(export.Messages) != len(conv.Messages) {
		t.Errorf("消息数量不匹配: got %v, want %v", len(export.Messages), len(conv.Messages))
	}
}

// TestLoadConversationFromFile 测试从文件加载对话
func TestLoadConversationFromFile(t *testing.T) {
	// 创建测试对话
	conv := &Conversation{
		ID:        "load-test-123",
		Title:     "Load Test",
		CreatedAt: time.Now(),
		Status:    ConvStatusFinished,
		Messages: []Message{
			{
				Role:      MessageRoleUser,
				Content:   "Load test message",
				Timestamp: time.Now(),
			},
		},
		TokenUsage: 75,
	}

	// 保存对话
	savedPath, err := SaveConversationToFile("", conv)
	if err != nil {
		t.Fatalf("保存对话失败: %v", err)
	}
	defer os.Remove(savedPath)

	// 加载对话
	loadedConv, err := LoadConversationFromFile(savedPath)
	if err != nil {
		t.Fatalf("加载对话失败: %v", err)
	}

	// 验证加载的数据
	if loadedConv.ID != conv.ID {
		t.Errorf("ID 不匹配: got %v, want %v", loadedConv.ID, conv.ID)
	}

	if loadedConv.Title != conv.Title {
		t.Errorf("标题不匹配: got %v, want %v", loadedConv.Title, conv.Title)
	}

	if loadedConv.Status != conv.Status {
		t.Errorf("状态不匹配: got %v, want %v", loadedConv.Status, conv.Status)
	}

	if len(loadedConv.Messages) != len(conv.Messages) {
		t.Errorf("消息数量不匹配: got %v, want %v", len(loadedConv.Messages), len(conv.Messages))
	}

	if loadedConv.TokenUsage != conv.TokenUsage {
		t.Errorf("Token 用量不匹配: got %v, want %v", loadedConv.TokenUsage, conv.TokenUsage)
	}
}

// TestCommandExecution 测试命令执行结果显示
func TestCommandExecution(t *testing.T) {
	// 创建 AppModel
	model := NewAppModel()

	// 创建活动对话
	conv := &Conversation{
		ID:        "cmd-test-123",
		Title:     "Command Test",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages:  make([]Message, 0),
	}
	model.activeConv = conv

	// 测试成功命令
	t.Run("SuccessCommand", func(t *testing.T) {
		initialMsgCount := len(model.activeConv.Messages)
		model.handleCommandExecuted("test", nil, "测试成功", nil)

		if len(model.activeConv.Messages) != initialMsgCount+1 {
			t.Errorf("消息数量应该增加 1: got %v, want %v", len(model.activeConv.Messages), initialMsgCount+1)
		}

		lastMsg := model.activeConv.Messages[len(model.activeConv.Messages)-1]
		if !strings.Contains(lastMsg.Content, "✓") {
			t.Errorf("成功消息应该包含 ✓: %v", lastMsg.Content)
		}
	})

	// 测试失败命令
	t.Run("FailureCommand", func(t *testing.T) {
		initialMsgCount := len(model.activeConv.Messages)
		model.handleCommandExecuted("test", nil, "", os.ErrNotExist)

		if len(model.activeConv.Messages) != initialMsgCount+1 {
			t.Errorf("消息数量应该增加 1: got %v, want %v", len(model.activeConv.Messages), initialMsgCount+1)
		}

		lastMsg := model.activeConv.Messages[len(model.activeConv.Messages)-1]
		if !strings.Contains(lastMsg.Content, "❌") {
			t.Errorf("失败消息应该包含 ❌: %v", lastMsg.Content)
		}
	})
}

// TestConversationListDisplay 测试对话列表显示
func TestConversationListDisplay(t *testing.T) {
	// 创建 AppModel
	model := NewAppModel()

	// 创建多个对话
	conv1 := &Conversation{
		ID:        "conv-1",
		Title:     "Conversation 1",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		Status:    ConvStatusFinished,
		Messages:  make([]Message, 0),
	}

	conv2 := &Conversation{
		ID:        "conv-2",
		Title:     "Conversation 2",
		CreatedAt: time.Now().Add(-1 * time.Hour),
		Status:    ConvStatusActive,
		Messages:  make([]Message, 0),
	}

	model.conversations = []*Conversation{conv1, conv2}
	model.activeConv = conv2

	// 测试更新对话列表
	model.updateDialogList()

	// 验证对话列表包含正确的对话
	dialogConvs := model.dialogList.GetConversations()
	if len(dialogConvs) != 2 {
		t.Errorf("对话列表应该包含 2 个对话: got %v", len(dialogConvs))
	}
}

// TestClearCommand 测试清空对话命令
func TestClearCommand(t *testing.T) {
	// 创建 AppModel
	model := NewAppModel()

	// 创建带有消息的对话
	conv := &Conversation{
		ID:        "clear-test-123",
		Title:     "Clear Test",
		CreatedAt: time.Now(),
		Status:    ConvStatusActive,
		Messages: []Message{
			{
				Role:      MessageRoleUser,
				Content:   "Test message 1",
				Timestamp: time.Now(),
			},
			{
				Role:      MessageRoleAssistant,
				Content:   "Test message 2",
				Timestamp: time.Now(),
			},
		},
		TokenUsage: 100,
	}
	model.activeConv = conv

	// 执行清空命令
	cmd := model.handleClearCommand()
	if cmd == nil {
		t.Error("清空命令应该返回一个命令")
	}

	// 验证对话已清空
	if len(model.activeConv.Messages) != 0 {
		t.Errorf("消息列表应该为空: got %v", len(model.activeConv.Messages))
	}

	if model.activeConv.TokenUsage != 0 {
		t.Errorf("Token 用量应该为 0: got %v", model.activeConv.TokenUsage)
	}
}

// TestHelpCommand 测试帮助命令
func TestHelpCommand(t *testing.T) {
	// 创建 AppModel
	model := NewAppModel()

	// 执行帮助命令
	cmd := model.handleHelpCommand()
	if cmd == nil {
		t.Error("帮助命令应该返回一个命令")
	}

	// 执行命令并获取结果
	msg := cmd()
	execMsg, ok := msg.(CommandExecutedMsg)
	if !ok {
		t.Fatal("命令应该返回 CommandExecutedMsg")
	}

	// 验证帮助文本包含所有命令
	helpText := execMsg.Result
	expectedCommands := []string{"/cd", "/conversations", "/clear", "/save", "/help", "/exit"}
	for _, cmd := range expectedCommands {
		if !strings.Contains(helpText, cmd) {
			t.Errorf("帮助文本应该包含命令 %s", cmd)
		}
	}
}
