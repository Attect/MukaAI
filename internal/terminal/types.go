// Package terminal 提供跨平台交互式终端功能
// 支持 Windows(cmd/powershell)、Linux(bash/zsh)、macOS(bash/zsh)
// 通过 PTY 抽象层屏蔽操作系统差异，提供统一的终端管理接口
package terminal

import "sync"

// PTY 伪终端接口，定义跨平台的 PTY 操作
// Windows 实现使用 ConPty API，Unix 实现使用 POSIX PTY
type PTY interface {
	// Start 启动 PTY 并运行指定的 shell 进程
	// rows/cols 为初始终端尺寸
	Start(cmd string, args []string, rows, cols int) error

	// Read 从 PTY 读取输出（终端显示的内容）
	Read(p []byte) (n int, err error)

	// Write 向 PTY 写入输入（模拟键盘输入）
	Write(p []byte) (n int, err error)

	// Resize 调整终端尺寸
	Resize(rows, cols int) error

	// Close 关闭 PTY 及相关资源
	Close() error

	// Wait 等待 PTY 进程退出
	Wait() error
}

// TerminalMessage WebSocket 通信消息格式
// 前端与后端通过此格式交换终端数据
type TerminalMessage struct {
	// Type 消息类型
	// 客户端→服务端: input, resize, signal
	// 服务端→客户端: output, exit, error
	Type string `json:"type"`

	// Data 文本数据（input/output 消息使用）
	Data string `json:"data,omitempty"`

	// Rows 终端行数（resize 消息使用）
	Rows int `json:"rows,omitempty"`

	// Cols 终端列数（resize 消息使用）
	Cols int `json:"cols,omitempty"`

	// Signal 信号名称（signal 消息使用，如 SIGINT）
	Signal string `json:"signal,omitempty"`

	// Code 退出码（exit 消息使用）
	Code int `json:"code,omitempty"`

	// Message 错误消息（error 消息使用）
	Message string `json:"message,omitempty"`
}

// OutputBuffer 线程安全的输出缓冲区
// 用于缓存 PTY 输出，供 Agent 工具和 WebSocket 读取
type OutputBuffer struct {
	mu     sync.Mutex
	buf    []byte
	maxLen int // 最大缓冲区大小
}

// NewOutputBuffer 创建新的输出缓冲区
func NewOutputBuffer(maxLen int) *OutputBuffer {
	return &OutputBuffer{
		buf:    make([]byte, 0, 4096),
		maxLen: maxLen,
	}
}

// Write 向缓冲区写入数据（实现 io.Writer 接口）
func (b *OutputBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.buf = append(b.buf, p...)

	// 如果超过最大长度，保留后半部分
	if len(b.buf) > b.maxLen {
		half := b.maxLen / 2
		b.buf = b.buf[len(b.buf)-half:]
	}

	return len(p), nil
}

// ReadAll 读取缓冲区全部内容（返回副本）
func (b *OutputBuffer) ReadAll() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

// Len 返回缓冲区当前长度
func (b *OutputBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.buf)
}

// Clear 清空缓冲区
func (b *OutputBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = b.buf[:0]
}

// ReadFrom 从缓冲区指定位置开始读取
// 返回从 startPos 位置到末尾的内容
func (b *OutputBuffer) ReadFrom(startPos int) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if startPos >= len(b.buf) {
		return ""
	}
	return string(b.buf[startPos:])
}

const (
	// DefaultTerminalRows 默认终端行数
	DefaultTerminalRows = 24

	// DefaultTerminalCols 默认终端列数
	DefaultTerminalCols = 80

	// MaxOutputBufferSize 输出缓冲区最大大小 (1MB)
	MaxOutputBufferSize = 1024 * 1024

	// DefaultExecTimeout terminal_exec 默认超时时间（秒）
	DefaultExecTimeout = 30

	// OutputStableInterval 输出稳定判定间隔（毫秒）
	// 在此时间内无新输出，则认为输出已稳定
	OutputStableInterval = 500
)
