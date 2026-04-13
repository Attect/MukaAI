package terminal

import (
	"strings"
	"testing"
)

// TestNewOutputBuffer 测试输出缓冲区创建
func TestNewOutputBuffer(t *testing.T) {
	buf := NewOutputBuffer(1024)
	if buf == nil {
		t.Fatal("NewOutputBuffer returned nil")
	}
	if buf.Len() != 0 {
		t.Fatalf("expected initial length 0, got %d", buf.Len())
	}
}

// TestOutputBufferWriteAndRead 测试写入和读取
func TestOutputBufferWriteAndRead(t *testing.T) {
	buf := NewOutputBuffer(1024)

	data := []byte("hello world")
	n, err := buf.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Fatalf("expected %d bytes written, got %d", len(data), n)
	}

	content := buf.ReadAll()
	if content != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", content)
	}
}

// TestOutputBufferMultipleWrites 测试多次写入
func TestOutputBufferMultipleWrites(t *testing.T) {
	buf := NewOutputBuffer(1024)

	buf.Write([]byte("hello "))
	buf.Write([]byte("world"))

	content := buf.ReadAll()
	if content != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", content)
	}
}

// TestOutputBufferOverflow 测试缓冲区溢出截断
func TestOutputBufferOverflow(t *testing.T) {
	maxLen := 100
	buf := NewOutputBuffer(maxLen)

	// 写入超过最大长度的数据
	largeData := strings.Repeat("a", 200)
	buf.Write([]byte(largeData))

	// 缓冲区应该被截断
	if buf.Len() > maxLen {
		t.Fatalf("buffer length %d exceeds max %d", buf.Len(), maxLen)
	}
}

// TestOutputBufferReadFrom 测试从指定位置读取
func TestOutputBufferReadFrom(t *testing.T) {
	buf := NewOutputBuffer(1024)

	buf.Write([]byte("hello world"))

	// 从位置 6 开始读取
	content := buf.ReadFrom(6)
	if content != "world" {
		t.Fatalf("expected 'world', got '%s'", content)
	}

	// 从超出长度的位置读取
	content = buf.ReadFrom(100)
	if content != "" {
		t.Fatalf("expected empty string, got '%s'", content)
	}
}

// TestOutputBufferClear 测试清空
func TestOutputBufferClear(t *testing.T) {
	buf := NewOutputBuffer(1024)

	buf.Write([]byte("hello world"))
	buf.Clear()

	if buf.Len() != 0 {
		t.Fatalf("expected length 0 after clear, got %d", buf.Len())
	}

	content := buf.ReadAll()
	if content != "" {
		t.Fatalf("expected empty string after clear, got '%s'", content)
	}
}

// TestTerminalMessage 测试消息序列化
func TestTerminalMessage(t *testing.T) {
	msg := TerminalMessage{
		Type: "output",
		Data: "hello world\n",
	}

	// 验证字段值
	if msg.Type != "output" {
		t.Fatalf("expected type 'output', got '%s'", msg.Type)
	}
	if msg.Data != "hello world\n" {
		t.Fatalf("expected data 'hello world\\n', got '%s'", msg.Data)
	}
}

// TestTerminalMessageTypes 测试不同消息类型
func TestTerminalMessageTypes(t *testing.T) {
	messages := []TerminalMessage{
		{Type: "input", Data: "ls -la\n"},
		{Type: "resize", Rows: 40, Cols: 120},
		{Type: "signal", Signal: "SIGINT"},
		{Type: "output", Data: "file1.txt\n"},
		{Type: "exit", Code: 0},
		{Type: "error", Message: "something went wrong"},
	}

	for _, msg := range messages {
		if msg.Type == "" {
			t.Fatal("message type should not be empty")
		}
	}
}

// TestConstants 测试常量值
func TestConstants(t *testing.T) {
	if DefaultTerminalRows != 24 {
		t.Fatalf("expected DefaultTerminalRows=24, got %d", DefaultTerminalRows)
	}
	if DefaultTerminalCols != 80 {
		t.Fatalf("expected DefaultTerminalCols=80, got %d", DefaultTerminalCols)
	}
	if DefaultExecTimeout != 30 {
		t.Fatalf("expected DefaultExecTimeout=30, got %d", DefaultExecTimeout)
	}
	if MaxOutputBufferSize != 1024*1024 {
		t.Fatalf("expected MaxOutputBufferSize=1048576, got %d", MaxOutputBufferSize)
	}
}

// TestNewTerminalManager 测试终端管理器创建
func TestNewTerminalManager(t *testing.T) {
	tm := NewTerminalManager("", "")
	if tm == nil {
		t.Fatal("NewTerminalManager returned nil")
	}
	if tm.IsRunning() {
		t.Fatal("new manager should not be running")
	}
	if tm.IsStarted() {
		t.Fatal("new manager should not be started")
	}
}

// TestTerminalManagerReadOutput 测试未启动时读取输出
func TestTerminalManagerReadOutput(t *testing.T) {
	tm := NewTerminalManager("", "")

	// 未启动时输出应为空
	output := tm.ReadOutput()
	if output != "" {
		t.Fatalf("expected empty output, got '%s'", output)
	}
}

// TestTerminalManagerWriteNotRunning 测试未启动时写入
func TestTerminalManagerWriteNotRunning(t *testing.T) {
	tm := NewTerminalManager("", "")

	err := tm.Write([]byte("test"))
	if err == nil {
		t.Fatal("expected error when writing to stopped terminal")
	}
}

// TestTerminalManagerResizeNotRunning 测试未启动时调整大小
func TestTerminalManagerResizeNotRunning(t *testing.T) {
	tm := NewTerminalManager("", "")

	err := tm.Resize(40, 120)
	if err == nil {
		t.Fatal("expected error when resizing stopped terminal")
	}
}
