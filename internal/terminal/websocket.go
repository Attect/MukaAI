package terminal

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketServer 终端 WebSocket 服务器
// 提供独立的 HTTP 服务器用于 WebSocket 连接
// 与 Wails 前端面板通信，传递终端输入/输出
type WebSocketServer struct {
	manager  *TerminalManager
	server   *http.Server
	upgrader websocket.Upgrader
	port     int
	mu       sync.Mutex
	started  bool
}

// NewWebSocketServer 创建新的 WebSocket 服务器
func NewWebSocketServer(manager *TerminalManager) *WebSocketServer {
	return &WebSocketServer{
		manager: manager,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// 允许所有来源（本地应用，不存在安全风险）
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// Start 启动 WebSocket 服务器
// 使用随机可用端口绑定到 127.0.0.1
func (s *WebSocketServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws/terminal", s.handleTerminal)

	s.server = &http.Server{
		Handler: mux,
	}

	// 监听随机端口
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	s.port = listener.Addr().(*net.TCPAddr).Port
	s.started = true

	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("[TerminalWS] server error: %v", err)
		}
	}()

	log.Printf("[TerminalWS] WebSocket server started on 127.0.0.1:%d", s.port)
	return nil
}

// Stop 停止 WebSocket 服务器
func (s *WebSocketServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started || s.server == nil {
		return nil
	}

	s.started = false
	return s.server.Close()
}

// GetPort 返回 WebSocket 服务器端口
func (s *WebSocketServer) GetPort() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

// GetWSUrl 返回 WebSocket 连接 URL
func (s *WebSocketServer) GetWSUrl() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return ""
	}
	return fmt.Sprintf("ws://127.0.0.1:%d/ws/terminal", s.port)
}

// handleTerminal 处理 WebSocket 连接
// 将 WebSocket 消息转发给 TerminalManager，并将终端输出回传
func (s *WebSocketServer) handleTerminal(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[TerminalWS] upgrade error: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("[TerminalWS] new client connected from %s", conn.RemoteAddr())

	// 确保终端已启动
	if !s.manager.IsRunning() {
		if err := s.manager.Start(); err != nil {
			s.sendError(conn, "Failed to start terminal: "+err.Error())
			return
		}
	}

	// 订阅终端输出
	outputCh := s.manager.Subscribe()
	defer s.manager.Unsubscribe(outputCh)

	// 发送历史输出
	history := s.manager.ReadOutput()
	if history != "" {
		msg := TerminalMessage{Type: "output", Data: history}
		data, _ := json.Marshal(msg)
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("[TerminalWS] failed to send history: %v", err)
			return
		}
	}

	// goroutine: 将终端输出转发给 WebSocket 客户端
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case data, ok := <-outputCh:
				if !ok {
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
					log.Printf("[TerminalWS] write error: %v", err)
					return
				}
			}
		}
	}()

	// 主循环: 读取 WebSocket 消息并转发给终端
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[TerminalWS] read error: %v", err)
			}
			break
		}

		var msg TerminalMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			s.sendError(conn, "Invalid message format")
			continue
		}

		switch msg.Type {
		case "input":
			// 转发用户输入到 PTY
			if err := s.manager.Write([]byte(msg.Data)); err != nil {
				s.sendError(conn, err.Error())
			}

		case "resize":
			// 调整终端尺寸
			if msg.Rows > 0 && msg.Cols > 0 {
				if err := s.manager.Resize(msg.Rows, msg.Cols); err != nil {
					log.Printf("[TerminalWS] resize error: %v", err)
				}
			}

		case "signal":
			// 发送信号
			if err := s.manager.SendSignal(msg.Signal); err != nil {
				s.sendError(conn, err.Error())
			}

		default:
			s.sendError(conn, "Unknown message type: "+msg.Type)
		}
	}

	// 等待输出 goroutine 结束
	select {
	case <-done:
	case <-time.After(time.Second):
	}
}

// sendError 通过 WebSocket 发送错误消息
func (s *WebSocketServer) sendError(conn *websocket.Conn, message string) {
	msg := TerminalMessage{Type: "error", Message: message}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	conn.WriteMessage(websocket.TextMessage, data)
}
