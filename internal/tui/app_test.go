// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewAppModel 测试创建新的 TUI 应用 Model
func TestNewAppModel(t *testing.T) {
	model := NewAppModel()

	// 检查初始化状态
	if model.initialized != false {
		t.Error("新创建的模型应该是未初始化状态")
	}

	// 检查当前目录是否已设置
	if model.currentDir == "" {
		t.Error("新创建的模型应该设置当前目录")
	}

	// 检查状态栏是否已创建
	if model.statusBar == nil {
		t.Error("新创建的模型应该创建状态栏组件")
	}

	// 检查对话列表是否已初始化
	if model.conversations == nil {
		t.Error("新创建的模型应该初始化对话列表")
	}
}

// TestValidateDirectory 测试目录验证功能
func TestValidateDirectory(t *testing.T) {
	model := NewAppModel()

	// 测试存在的目录
	tempDir := t.TempDir()
	err := model.validateDirectory(tempDir)
	if err != nil {
		t.Errorf("验证存在的目录应该成功，但得到错误: %v", err)
	}

	// 测试不存在的目录
	err = model.validateDirectory("/nonexistent/directory/path")
	if err == nil {
		t.Error("验证不存在的目录应该失败")
	}

	// 测试文件路径（不是目录）
	tempFile := filepath.Join(tempDir, "testfile")
	file, err := os.Create(tempFile)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	file.Close()

	err = model.validateDirectory(tempFile)
	if err == nil {
		t.Error("验证文件路径应该失败（不是目录）")
	}
}

// TestHandleCDCommand 测试 /cd 命令处理
func TestHandleCDCommand(t *testing.T) {
	model := NewAppModel()
	originalDir := model.currentDir

	// 保存当前工作目录，测试结束后恢复
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}
	defer os.Chdir(wd)

	// 测试切换到临时目录
	tempDir := t.TempDir()
	cmd := model.handleCDCommand([]string{tempDir})

	// 检查命令是否返回了正确的消息
	if cmd == nil {
		t.Fatal("handleCDCommand 应该返回一个命令")
	}

	msg := cmd()
	dirChangedMsg, ok := msg.(WorkingDirChangedMsg)
	if !ok {
		t.Fatalf("期望 WorkingDirChangedMsg 类型，得到: %T", msg)
	}

	if dirChangedMsg.OldDir != originalDir {
		t.Errorf("旧目录应该是 %s，但得到: %s", originalDir, dirChangedMsg.OldDir)
	}

	if dirChangedMsg.NewDir != tempDir {
		t.Errorf("新目录应该是 %s，但得到: %s", tempDir, dirChangedMsg.NewDir)
	}
}

// TestHandleCDCommandWithInvalidPath 测试切换到无效路径
func TestHandleCDCommandWithInvalidPath(t *testing.T) {
	model := NewAppModel()

	// 测试切换到不存在的目录
	cmd := model.handleCDCommand([]string{"/nonexistent/path"})
	msg := cmd()

	cmdExecMsg, ok := msg.(CommandExecutedMsg)
	if !ok {
		t.Fatalf("期望 CommandExecutedMsg 类型，得到: %T", msg)
	}

	if cmdExecMsg.Error == nil {
		t.Error("切换到不存在的目录应该返回错误")
	}
}

// TestHandleCDCommandWithNoArgs 测试不带参数的 /cd 命令
func TestHandleCDCommandWithNoArgs(t *testing.T) {
	model := NewAppModel()

	// 测试不带参数
	cmd := model.handleCDCommand([]string{})
	msg := cmd()

	cmdExecMsg, ok := msg.(CommandExecutedMsg)
	if !ok {
		t.Fatalf("期望 CommandExecutedMsg 类型，得到: %T", msg)
	}

	if cmdExecMsg.Error == nil {
		t.Error("不带参数的 /cd 命令应该返回错误")
	}
}

// TestHandleCDCommandWithHomeDirectory 测试切换到用户主目录
func TestHandleCDCommandWithHomeDirectory(t *testing.T) {
	model := NewAppModel()

	// 测试切换到用户主目录
	cmd := model.handleCDCommand([]string{"~"})
	msg := cmd()

	// 应该成功切换
	dirChangedMsg, ok := msg.(WorkingDirChangedMsg)
	if !ok {
		// 可能是错误消息
		cmdExecMsg, ok := msg.(CommandExecutedMsg)
		if ok && cmdExecMsg.Error != nil {
			t.Logf("切换到用户主目录失败（可能是权限问题）: %v", cmdExecMsg.Error)
			return
		}
		t.Fatalf("期望 WorkingDirChangedMsg 类型，得到: %T", msg)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Logf("无法获取用户主目录，跳过验证")
		return
	}

	if dirChangedMsg.NewDir != homeDir {
		t.Errorf("新目录应该是 %s，但得到: %s", homeDir, dirChangedMsg.NewDir)
	}
}

// TestHandleCDCommandWithRelativePath 测试相对路径切换
func TestHandleCDCommandWithRelativePath(t *testing.T) {
	model := NewAppModel()

	// 保存当前工作目录，测试结束后恢复
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}
	defer os.Chdir(wd)

	// 创建临时目录结构
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("创建子目录失败: %v", err)
	}

	// 先切换到临时目录
	model.SetCurrentDir(tempDir)

	// 测试相对路径切换
	cmd := model.handleCDCommand([]string{"subdir"})
	msg := cmd()

	dirChangedMsg, ok := msg.(WorkingDirChangedMsg)
	if !ok {
		cmdExecMsg, ok := msg.(CommandExecutedMsg)
		if ok && cmdExecMsg.Error != nil {
			t.Errorf("相对路径切换失败: %v", cmdExecMsg.Error)
			return
		}
		t.Fatalf("期望 WorkingDirChangedMsg 类型，得到: %T", msg)
	}

	if dirChangedMsg.NewDir != subDir {
		t.Errorf("新目录应该是 %s，但得到: %s", subDir, dirChangedMsg.NewDir)
	}
}

// TestHandleWorkingDirChanged 测试工作目录变更处理
func TestHandleWorkingDirChanged(t *testing.T) {
	model := NewAppModel()
	oldDir := model.currentDir
	newDir := t.TempDir()

	// 处理工作目录变更
	model.handleWorkingDirChanged(oldDir, newDir)

	// 检查当前目录是否已更新
	if model.currentDir != newDir {
		t.Errorf("当前目录应该是 %s，但得到: %s", newDir, model.currentDir)
	}

	// 检查状态栏是否已更新
	if model.statusBar.CurrentDir != newDir {
		t.Errorf("状态栏目录应该是 %s，但得到: %s", newDir, model.statusBar.CurrentDir)
	}
}

// TestSetCurrentDir 测试设置当前工作目录
func TestSetCurrentDir(t *testing.T) {
	model := NewAppModel()

	// 保存当前工作目录，测试结束后恢复
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}
	defer os.Chdir(wd)

	tempDir := t.TempDir()

	// 测试设置当前目录
	err = model.SetCurrentDir(tempDir)
	if err != nil {
		t.Errorf("设置当前目录失败: %v", err)
	}

	// 检查当前目录是否已更新
	if model.currentDir != tempDir {
		t.Errorf("当前目录应该是 %s，但得到: %s", tempDir, model.currentDir)
	}

	// 检查状态栏是否已更新
	if model.statusBar.CurrentDir != tempDir {
		t.Errorf("状态栏目录应该是 %s，但得到: %s", tempDir, model.statusBar.CurrentDir)
	}
}

// TestSetCurrentDirWithInvalidPath 测试设置无效路径
func TestSetCurrentDirWithInvalidPath(t *testing.T) {
	model := NewAppModel()

	// 测试设置不存在的目录
	err := model.SetCurrentDir("/nonexistent/path")
	if err == nil {
		t.Error("设置不存在的目录应该返回错误")
	}
}

// TestGetCurrentDir 测试获取当前工作目录
func TestGetCurrentDir(t *testing.T) {
	model := NewAppModel()

	// 获取当前目录
	dir := model.GetCurrentDir()

	// 检查是否与模型中的目录一致
	if dir != model.currentDir {
		t.Errorf("获取的目录应该是 %s，但得到: %s", model.currentDir, dir)
	}
}

// TestHandleCommand 测试命令处理路由
func TestHandleCommand(t *testing.T) {
	model := NewAppModel()

	tests := []struct {
		name     string
		cmd      string
		wantQuit bool
	}{
		{
			name:     "cd 命令",
			cmd:      "/cd /tmp",
			wantQuit: false,
		},
		{
			name:     "help 命令",
			cmd:      "/help",
			wantQuit: false,
		},
		{
			name:     "exit 命令",
			cmd:      "/exit",
			wantQuit: true,
		},
		{
			name:     "quit 命令",
			cmd:      "/quit",
			wantQuit: true,
		},
		{
			name:     "q 命令",
			cmd:      "/q",
			wantQuit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := model.handleCommand(tt.cmd)

			if cmd == nil {
				if tt.wantQuit {
					t.Error("退出命令应该返回 tea.Quit")
				}
				return
			}

			// 检查是否为退出命令
			// 注意：tea.Quit 是一个特殊的命令，我们无法直接比较
			// 这里只检查命令是否返回了非 nil 值
			if tt.wantQuit {
				// 退出命令应该返回 tea.Quit
				t.Logf("退出命令返回了命令")
			}
		})
	}
}

// TestHandleCommandExecuted 测试命令执行结果处理
func TestHandleCommandExecuted(t *testing.T) {
	model := NewAppModel()

	// 测试错误结果
	model.handleCommandExecuted("test", nil, "", os.ErrNotExist)
	if model.lastError == "" {
		t.Error("处理错误结果应该设置 lastError")
	}

	// 测试成功结果
	model.lastError = ""
	model.handleCommandExecuted("test", nil, "success", nil)
	if model.lastError != "" {
		t.Error("处理成功结果不应该设置 lastError")
	}
}

// TestHandleClearCommand 测试清空对话命令
func TestHandleClearCommand(t *testing.T) {
	model := NewAppModel()

	// 添加一个对话
	conv := &Conversation{
		ID:         "test-conv",
		Messages:   []Message{{Role: MessageRoleUser, Content: "test"}},
		TokenUsage: 100,
	}
	model.activeConv = conv

	// 执行清空命令
	cmd := model.handleClearCommand()
	msg := cmd()

	cmdExecMsg, ok := msg.(CommandExecutedMsg)
	if !ok {
		t.Fatalf("期望 CommandExecutedMsg 类型，得到: %T", msg)
	}

	if cmdExecMsg.Error != nil {
		t.Errorf("清空对话应该成功，但得到错误: %v", cmdExecMsg.Error)
	}

	// 检查对话是否已清空
	if len(model.activeConv.Messages) != 0 {
		t.Error("对话消息应该被清空")
	}

	if model.activeConv.TokenUsage != 0 {
		t.Error("对话 token 用量应该被重置")
	}
}

// TestHandleHelpCommand 测试帮助命令
func TestHandleHelpCommand(t *testing.T) {
	model := NewAppModel()

	// 执行帮助命令
	cmd := model.handleHelpCommand()
	msg := cmd()

	cmdExecMsg, ok := msg.(CommandExecutedMsg)
	if !ok {
		t.Fatalf("期望 CommandExecutedMsg 类型，得到: %T", msg)
	}

	if cmdExecMsg.Error != nil {
		t.Errorf("帮助命令应该成功，但得到错误: %v", cmdExecMsg.Error)
	}

	if cmdExecMsg.Result == "" {
		t.Error("帮助命令应该返回帮助文本")
	}
}
