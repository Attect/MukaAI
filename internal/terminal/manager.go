package terminal

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
)

// TerminalManager 终端管理器
// 管理 PTY 生命周期、输出缓冲、WebSocket 广播
// 线程安全，支持并发访问
type TerminalManager struct {
	mu sync.Mutex

	pty       PTY           // 当前 PTY 实例
	outputBuf *OutputBuffer // 输出缓冲区
	running   bool          // PTY 是否运行中
	started   bool          // PTY 是否已启动过
	exitCode  int           // 进程退出码
	done      chan struct{} // 进程退出信号

	// WebSocket 客户端管理
	wsMu      sync.RWMutex
	wsClients map[chan []byte]struct{} // WebSocket 输出广播通道集合

	// 输出通知（用于 terminal_exec 等待输出稳定）
	outputNotifyMu sync.Mutex
	outputNotify   *sync.Cond
	lastOutputTime time.Time // 最后一次收到输出的时间

	// 配置
	shell   string // 使用的 shell
	workDir string // 工作目录
}

// NewTerminalManager 创建新的终端管理器
// shell: 指定 shell 路径，为空时使用系统默认
// workDir: 工作目录，为空时使用当前目录
func NewTerminalManager(shell, workDir string) *TerminalManager {
	tm := &TerminalManager{
		outputBuf: NewOutputBuffer(MaxOutputBufferSize),
		wsClients: make(map[chan []byte]struct{}),
		shell:     shell,
		workDir:   workDir,
		done:      make(chan struct{}),
	}
	tm.outputNotify = sync.NewCond(&tm.outputNotifyMu)
	return tm
}

// Start 启动终端（创建 PTY 并运行 shell）
// 如果已经在运行，先停止再重启
func (tm *TerminalManager) Start() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 如果已经在运行，先停止
	if tm.running {
		tm.stopLocked()
	}

	// 确定要使用的 shell
	shell := tm.shell
	if shell == "" {
		shell = getDefaultShell()
	}

	// 创建 PTY
	pty := newPTY()

	// 启动 PTY
	if err := pty.Start(shell, nil, DefaultTerminalRows, DefaultTerminalCols); err != nil {
		return fmt.Errorf("failed to start pty with shell %s: %w", shell, err)
	}

	tm.pty = pty
	tm.running = true
	tm.started = true
	tm.exitCode = -1
	tm.done = make(chan struct{})

	// 如果有工作目录，发送 cd 命令
	if tm.workDir != "" {
		go func() {
			// 等待 shell 启动完成
			time.Sleep(500 * time.Millisecond)
			cdCmd := fmt.Sprintf("cd %s\n", tm.workDir)
			tm.pty.Write([]byte(cdCmd))
		}()
	}

	// 启动输出读取 goroutine
	go tm.readOutput()

	return nil
}

// Stop 停止终端
func (tm *TerminalManager) Stop() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.stopLocked()
}

// stopLocked 停止终端（调用方必须持有锁）
func (tm *TerminalManager) stopLocked() error {
	if !tm.running || tm.pty == nil {
		return nil
	}

	tm.running = false

	// 关闭 PTY
	err := tm.pty.Close()

	// 等待进程退出（带超时）
	select {
	case <-tm.done:
	case <-time.After(3 * time.Second):
		log.Printf("[Terminal] timeout waiting for pty to exit")
	}

	// 通知所有 WebSocket 客户端进程已退出
	tm.broadcastWS(TerminalMessage{
		Type: "exit",
		Code: tm.exitCode,
	})

	// 关闭所有 WebSocket 客户端通道
	tm.wsMu.Lock()
	for ch := range tm.wsClients {
		close(ch)
	}
	tm.wsClients = make(map[chan []byte]struct{})
	tm.wsMu.Unlock()

	return err
}

// readOutput 持续从 PTY 读取输出
// 将输出写入缓冲区并广播给 WebSocket 客户端
func (tm *TerminalManager) readOutput() {
	buf := make([]byte, 4096)
	for {
		n, err := tm.pty.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			// 写入输出缓冲区
			tm.outputBuf.Write(data)

			// 更新最后输出时间并通知等待者
			tm.outputNotifyMu.Lock()
			tm.lastOutputTime = time.Now()
			tm.outputNotifyMu.Unlock()
			tm.outputNotify.Broadcast()

			// 广播给 WebSocket 客户端
			msg := TerminalMessage{
				Type: "output",
				Data: string(data),
			}
			tm.broadcastWS(msg)
		}

		if err != nil {
			if err == io.EOF {
				// 进程正常退出
				tm.mu.Lock()
				tm.running = false
				tm.exitCode = 0
				tm.mu.Unlock()
			} else {
				// 读取错误（进程可能被杀死）
				tm.mu.Lock()
				tm.running = false
				tm.exitCode = -1
				tm.mu.Unlock()
			}

			// 通知所有等待者
			tm.outputNotify.Broadcast()

			// 发送退出消息
			tm.broadcastWS(TerminalMessage{
				Type: "exit",
				Code: tm.exitCode,
			})

			// 关闭 done 通道
			select {
			case <-tm.done:
				// 已关闭
			default:
				close(tm.done)
			}
			return
		}
	}
}

// Write 向终端写入输入（模拟键盘输入）
func (tm *TerminalManager) Write(data []byte) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if !tm.running || tm.pty == nil {
		return fmt.Errorf("terminal is not running")
	}

	_, err := tm.pty.Write(data)
	return err
}

// Resize 调整终端尺寸
func (tm *TerminalManager) Resize(rows, cols int) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if !tm.running || tm.pty == nil {
		return fmt.Errorf("terminal is not running")
	}

	return tm.pty.Resize(rows, cols)
}

// ReadOutput 读取终端输出缓冲区全部内容
func (tm *TerminalManager) ReadOutput() string {
	return tm.outputBuf.ReadAll()
}

// ReadOutputFrom 从指定位置读取终端输出
func (tm *TerminalManager) ReadOutputFrom(pos int) string {
	return tm.outputBuf.ReadFrom(pos)
}

// OutputLen 返回输出缓冲区当前长度
func (tm *TerminalManager) OutputLen() int {
	return tm.outputBuf.Len()
}

// IsRunning 返回终端是否正在运行
func (tm *TerminalManager) IsRunning() bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.running
}

// GetExitCode 返回进程退出码
func (tm *TerminalManager) GetExitCode() int {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.exitCode
}

// WaitForOutputStable 等待输出稳定（无新输出超过指定间隔）
// 返回等待期间收集到的输出和是否超时
func (tm *TerminalManager) WaitForOutputStable(stableInterval, timeout time.Duration, startPos int) (string, bool) {
	deadline := time.Now().Add(timeout)
	stableDeadline := time.Now().Add(stableInterval)

	for {
		tm.outputNotifyMu.Lock()
		// 等待新输出或超时
		remaining := deadline.Sub(time.Now())
		if remaining <= 0 {
			tm.outputNotifyMu.Unlock()
			break
		}

		// 检查是否已稳定
		if time.Now().After(stableDeadline) {
			tm.outputNotifyMu.Unlock()
			break
		}

		// 等待输出通知，最多等 100ms 再检查
		go func() {
			time.Sleep(100 * time.Millisecond)
			tm.outputNotify.Broadcast()
		}()
		tm.outputNotify.Wait()
		tm.outputNotifyMu.Unlock()

		// 更新稳定截止时间
		tm.outputNotifyMu.Lock()
		stableDeadline = tm.lastOutputTime.Add(stableInterval)
		tm.outputNotifyMu.Unlock()

		// 检查总超时
		if time.Now().After(deadline) {
			break
		}
	}

	return tm.outputBuf.ReadFrom(startPos), time.Now().After(deadline)
}

// SendSignal 向终端发送信号
// 注意：Windows 平台仅支持有限的信号处理
func (tm *TerminalManager) SendSignal(signal string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if !tm.running || tm.pty == nil {
		return fmt.Errorf("terminal is not running")
	}

	// 发送 Ctrl+C（通过写入 \x03）
	switch signal {
	case "SIGINT", "INT":
		_, err := tm.pty.Write([]byte{0x03})
		return err
	case "SIGQUIT", "QUIT":
		_, err := tm.pty.Write([]byte{0x1c}) // Ctrl+\
		return err
	case "SIGTERM", "TERM":
		// Unix: 可以发送信号；Windows: 使用 Ctrl+C
		_, err := tm.pty.Write([]byte{0x03})
		return err
	default:
		return fmt.Errorf("unsupported signal: %s", signal)
	}
}

// Subscribe 订阅终端输出广播
// 返回一个通道，当有新的终端输出时会发送到该通道
// 调用者应在不需要时调用 Unsubscribe 取消订阅
func (tm *TerminalManager) Subscribe() chan []byte {
	ch := make(chan []byte, 256)
	tm.wsMu.Lock()
	tm.wsClients[ch] = struct{}{}
	tm.wsMu.Unlock()
	return ch
}

// Unsubscribe 取消订阅终端输出
func (tm *TerminalManager) Unsubscribe(ch chan []byte) {
	tm.wsMu.Lock()
	delete(tm.wsClients, ch)
	tm.wsMu.Unlock()
}

// broadcastWS 广播消息给所有 WebSocket 客户端
func (tm *TerminalManager) broadcastWS(msg TerminalMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[Terminal] failed to marshal broadcast message: %v", err)
		return
	}

	tm.wsMu.RLock()
	defer tm.wsMu.RUnlock()

	for ch := range tm.wsClients {
		select {
		case ch <- data:
		default:
			// 通道已满，丢弃旧消息
			log.Printf("[Terminal] ws client channel full, dropping message")
		}
	}
}

// IsStarted 返回终端是否已启动过
func (tm *TerminalManager) IsStarted() bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.started
}
