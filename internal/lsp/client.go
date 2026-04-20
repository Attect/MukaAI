package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LSPClient 通用LSP客户端，通过stdin/stdout与语言服务器通信
// 使用JSON-RPC 2.0协议，支持基础LSP操作（initialize/didOpen/diagnostics）
type LSPClient struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	rootURI string

	// 请求-响应管理
	mu      sync.Mutex
	pending map[int]chan *JSONRPCResponse
	nextID  int

	// 诊断收集
	diagMu      sync.RWMutex
	diagnostics map[string][]Diagnostic // uri -> diagnostics

	// 生命周期状态
	initialized bool
	running     bool
	version     int // 文档版本计数器

	// 崩溃恢复
	crashCount int
}

// NewClient 创建新的LSP客户端
// command: 语言服务器可执行文件路径
// args: 启动参数
// rootURI: 项目根目录URI（file://格式）
func NewClient(command string, args []string, rootURI string) *LSPClient {
	return &LSPClient{
		cmd:         exec.Command(command, args...),
		rootURI:     rootURI,
		pending:     make(map[int]chan *JSONRPCResponse),
		diagnostics: make(map[string][]Diagnostic),
	}
}

// Start 启动语言服务器进程并开始读取响应
func (c *LSPClient) Start(ctx context.Context) error {
	var err error

	// 创建stdin/stdout管道
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// 启动进程
	// Windows 下隐藏控制台窗口（避免闪窗）
	configureHideWindow(c.cmd)
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start language server: %w", err)
	}

	c.running = true

	// 启动后台读取goroutine
	go c.readLoop(ctx)

	return nil
}

// Initialize 发送LSP initialize请求
func (c *LSPClient) Initialize(ctx context.Context) error {
	params := &InitializeParams{
		ProcessID: 0, // 0表示不跟踪进程
		RootURI:   c.rootURI,
		Capabilities: ClientCaps{
			TextDocument: TextDocumentClientCaps{
				PublishDiagnostics: PublishDiagnosticsCaps{
					RelatedInformation: false,
				},
			},
		},
	}

	resp, err := c.sendRequest(ctx, "initialize", params)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	// 发送initialized通知
	if err := c.sendNotification("initialized", map[string]interface{}{}); err != nil {
		return fmt.Errorf("initialized notification failed: %w", err)
	}

	c.initialized = true
	return nil
}

// DidOpen 通知服务器打开了一个文本文档
// uri: 文件URI（file://格式）
// language: 语言标识符（如"go"、"typescript"、"python"）
// content: 文件内容
func (c *LSPClient) DidOpen(ctx context.Context, uri, language, content string) error {
	c.mu.Lock()
	c.version++
	version := c.version
	c.mu.Unlock()

	params := &DidOpenParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: language,
			Version:    version,
			Text:       content,
		},
	}

	// 清除该文件之前的诊断
	c.diagMu.Lock()
	delete(c.diagnostics, uri)
	c.diagMu.Unlock()

	return c.sendNotification("textDocument/didOpen", params)
}

// DidChange 通知服务器文件内容发生了变更
func (c *LSPClient) DidChange(ctx context.Context, uri, content string) error {
	c.mu.Lock()
	c.version++
	version := c.version
	c.mu.Unlock()

	params := &DidChangeParams{
		TextDocument: VersionedTextDocumentIdentifier{
			URI:     uri,
			Version: version,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: content},
		},
	}

	return c.sendNotification("textDocument/didChange", params)
}

// Diagnostics 获取指定URI文件的诊断信息
// 返回当前缓存的诊断结果（通过publishDiagnostics通知收集）
func (c *LSPClient) Diagnostics(ctx context.Context, uri string) ([]Diagnostic, error) {
	c.diagMu.RLock()
	defer c.diagMu.RUnlock()
	diags := c.diagnostics[uri]
	// 返回副本避免外部修改
	result := make([]Diagnostic, len(diags))
	copy(result, diags)
	return result, nil
}

// Shutdown 发送shutdown请求并关闭客户端
func (c *LSPClient) Shutdown(ctx context.Context) error {
	if !c.running {
		return nil
	}

	// 发送shutdown请求
	_, _ = c.sendRequest(ctx, "shutdown", nil)

	// 发送exit通知
	_ = c.sendNotification("exit", nil)

	return c.Close()
}

// Close 关闭客户端连接和进程
func (c *LSPClient) Close() error {
	c.running = false

	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.stdout != nil {
		_ = c.stdout.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_ = c.cmd.Wait()
	}
	return nil
}

// IsRunning 返回客户端是否正在运行
func (c *LSPClient) IsRunning() bool {
	return c.running && c.initialized
}

// CrashCount 返回崩溃次数
func (c *LSPClient) CrashCount() int {
	return c.crashCount
}

// IncrementCrash 增加崩溃计数
func (c *LSPClient) IncrementCrash() {
	c.crashCount++
}

// ==================== 内部方法 ====================

// sendRequest 发送JSON-RPC请求并等待响应
func (c *LSPClient) sendRequest(ctx context.Context, method string, params interface{}) (*JSONRPCResponse, error) {
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan *JSONRPCResponse, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.writeMessage(req); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// 等待响应或超时
	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("request timeout: method=%s id=%d", method, id)
	}
}

// sendNotification 发送JSON-RPC通知（无需响应）
func (c *LSPClient) sendNotification(method string, params interface{}) error {
	notif := &JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return c.writeMessage(notif)
}

// writeMessage 写入JSON-RPC消息（带Content-Length头）
func (c *LSPClient) writeMessage(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// LSP使用Content-Length头格式
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))

	if _, err := c.stdin.Write([]byte(header)); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	if _, err := c.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write body: %w", err)
	}

	return nil
}

// readLoop 后台读取响应循环
func (c *LSPClient) readLoop(ctx context.Context) {
	reader := bufio.NewReader(c.stdout)

	for c.running {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// 读取Content-Length头
		var contentLength int
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF && c.running {
					log.Printf("[LSP] Error reading header: %v", err)
				}
				return
			}

			line = strings.TrimSpace(line)

			// 空行表示头部结束
			if line == "" {
				break
			}

			// 解析Content-Length
			if strings.HasPrefix(line, "Content-Length:") {
				lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
				contentLength, _ = strconv.Atoi(lengthStr)
			}
		}

		if contentLength <= 0 {
			continue
		}

		// 读取消息体
		body := make([]byte, contentLength)
		if _, err := io.ReadFull(reader, body); err != nil {
			if c.running {
				log.Printf("[LSP] Error reading body: %v", err)
			}
			return
		}

		// 解析消息
		c.handleMessage(body)
	}
}

// handleMessage 处理接收到的JSON-RPC消息
func (c *LSPClient) handleMessage(data []byte) {
	// 先判断是响应还是通知
	var raw struct {
		ID     int    `json:"id"`
		Method string `json:"method"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Printf("[LSP] Failed to parse message: %v", err)
		return
	}

	// 如果有method字段，是通知
	if raw.Method != "" {
		c.handleNotification(data)
		return
	}

	// 否则是响应
	c.handleResponse(data)
}

// handleResponse 处理JSON-RPC响应
func (c *LSPClient) handleResponse(data []byte) {
	var resp JSONRPCResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		log.Printf("[LSP] Failed to parse response: %v", err)
		return
	}

	c.mu.Lock()
	ch, ok := c.pending[resp.ID]
	c.mu.Unlock()

	if ok {
		select {
		case ch <- &resp:
		default:
			log.Printf("[LSP] Response channel full for id %d", resp.ID)
		}
	}
}

// handleNotification 处理JSON-RPC通知
func (c *LSPClient) handleNotification(data []byte) {
	var notif JSONRPCNotification
	if err := json.Unmarshal(data, &notif); err != nil {
		log.Printf("[LSP] Failed to parse notification: %v", err)
		return
	}

	switch notif.Method {
	case "textDocument/publishDiagnostics":
		c.handleDiagnostics(notif.Params)
	case "window/logMessage":
		// 日志消息，可选处理
	default:
		// 忽略其他通知
	}
}

// handleDiagnostics 处理诊断通知
func (c *LSPClient) handleDiagnostics(params interface{}) {
	// 重新序列化再反序列化，因为params是interface{}
	data, err := json.Marshal(params)
	if err != nil {
		return
	}

	var diagParams PublishDiagnosticsParams
	if err := json.Unmarshal(data, &diagParams); err != nil {
		log.Printf("[LSP] Failed to parse diagnostics: %v", err)
		return
	}

	c.diagMu.Lock()
	c.diagnostics[diagParams.URI] = diagParams.Diagnostics
	c.diagMu.Unlock()
}
