package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Attect/MukaAI/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPSession 表示与单个MCP Server的连接会话
// 管理连接生命周期、工具发现和工具调用
type MCPSession struct {
	id            string
	config        ServerConfig
	toolsSettings map[string]ToolSettingConfig // 从配置传入的工具设置
	client        *mcp.Client
	session       *mcp.ClientSession
	tools         []*mcp.Tool // 发现的工具列表
	mu            sync.RWMutex
	status        SessionStatus

	// 可用性增强字段
	retryCount       int           // 重连次数
	maxRetries       int           // 最大重连次数
	retryInterval    time.Duration // 重连间隔
	lastError        error         // 最后一次错误
	lastConnectedAt  time.Time     // 最后连接时间
	healthCheckTimer *time.Timer   // 健康检查定时器
}

// SessionStatus Session运行状态
type SessionStatus string

const (
	StatusConnecting   SessionStatus = "connecting"
	StatusConnected    SessionStatus = "connected"
	StatusDisconnected SessionStatus = "disconnected"
	StatusError        SessionStatus = "error"
)

// ServerStatus Server运行状态（用于外部查询）
type ServerStatus struct {
	ID          string        `json:"id"`
	Status      SessionStatus `json:"status"`
	Tools       int           `json:"tools"`
	Error       string        `json:"error,omitempty"`
	ConnectedAt *time.Time    `json:"connected_at,omitempty"`
	Uptime      time.Duration `json:"uptime,omitempty"` // 连接持续时间
}

// NewMCPSession 创建新的MCP Session
func NewMCPSession(id string, config ServerConfig) *MCPSession {
	// 设置可用性增强参数的默认值
	maxRetries := config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	retryIntervalSec := config.RetryIntervalSec
	if retryIntervalSec <= 0 {
		retryIntervalSec = 2
	}

	// 如果禁用了自动重连，则将最大重试次数设为 0
	autoReconnect := config.AutoReconnect
	if !autoReconnect {
		maxRetries = 0
	}

	return &MCPSession{
		id:            id,
		config:        config,
		status:        StatusDisconnected,
		maxRetries:    maxRetries,
		retryInterval: time.Duration(retryIntervalSec) * time.Second,
	}
}

// Connect 建立连接并初始化（带自动重试）
func (s *MCPSession) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 尝试连接，最多重试 maxRetries 次
	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		if attempt > 0 {
			logMCP("Server '%s' 第 %d 次重连尝试...", s.id, attempt)
			time.Sleep(s.retryInterval)
		}

		s.status = StatusConnecting

		// 创建MCP Client
		s.client = mcp.NewClient(
			&mcp.Implementation{
				Name:    "MukaAI",
				Version: "1.0.0",
			},
			nil, // 使用默认 ClientOptions
		)

		// 根据 Transport 类型创建 Transport
		transport, err := s.createTransport()
		if err != nil {
			s.status = StatusError
			s.lastError = fmt.Errorf("创建 transport 失败: %w", err)
			continue
		}

		// 连接并初始化（SDK 的 Connect 方法会自动完成 initialize/initialized 握手）
		session, err := s.client.Connect(ctx, transport, nil)
		if err != nil {
			s.status = StatusError
			s.lastError = fmt.Errorf("连接 MCP Server 失败: %w", err)
			logMCP("Server '%s' 连接失败 (尝试 %d/%d): %v", s.id, attempt+1, s.maxRetries+1, err)

			// 创建新的 client（旧的不需要显式关闭）
			s.client = nil
			continue
		}

		s.session = session
		s.status = StatusConnected
		s.lastError = nil
		s.lastConnectedAt = time.Now()
		s.retryCount = 0

		logMCP("Server '%s' 连接成功 (尝试 %d/%d)", s.id, attempt+1, s.maxRetries+1)

		// 启动后台健康检查（仅对 stdio 模式）
		if s.config.Transport == "stdio" {
			s.startHealthCheck()
		}

		return nil
	}

	logMCP("Server '%s' 连接失败，已达最大重试次数 %d", s.id, s.maxRetries)
	return fmt.Errorf("连接 MCP Server '%s' 失败: %w (已重试 %d 次)", s.id, s.lastError, s.maxRetries)
}

// startHealthCheck 启动后台健康检查（仅对 stdio 模式）
func (s *MCPSession) startHealthCheck() {
	s.healthCheckTimer = time.NewTimer(30 * time.Second)
	go func() {
		for {
			select {
			case <-s.healthCheckTimer.C:
				s.checkHealth()
				s.healthCheckTimer.Reset(30 * time.Second)
			}
		}
	}()
}

// checkHealth 检查 MCP Server 健康状态
func (s *MCPSession) checkHealth() {
	s.mu.RLock()
	if s.session == nil || s.status != StatusConnected {
		s.mu.RUnlock()
		return
	}
	sess := s.session
	s.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 发送一个简单的 ping 请求（通过 ListTools 检查连接）
	_, err := sess.ListTools(ctx, nil)
	if err != nil {
		logMCP("Server '%s' 健康检查失败: %v，尝试重连...", s.id, err)
		go s.reconnect()
	} else {
		s.mu.Lock()
		s.lastConnectedAt = time.Now()
		s.mu.Unlock()
	}
}

// reconnect 重新连接 MCP Server
func (s *MCPSession) reconnect() {
	s.mu.Lock()
	s.status = StatusConnecting
	s.session = nil
	s.client = nil
	s.mu.Unlock()

	// 等待一段时间后重连
	time.Sleep(s.retryInterval)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.Connect(ctx); err != nil {
		logMCP("Server '%s' 重连失败: %v", s.id, err)
	} else {
		logMCP("Server '%s' 重连成功", s.id)
	}
}

// createTransport 根据配置创建Transport
func (s *MCPSession) createTransport() (mcp.Transport, error) {
	switch s.config.Transport {
	case "stdio":
		return s.createStdioTransport()
	case "http":
		return s.createHTTPTransport()
	case "sse":
		return s.createSSETransport()
	default:
		return nil, fmt.Errorf("不支持的transport类型: %s", s.config.Transport)
	}
}

// DiscoverTools 发现并缓存工具列表
func (s *MCPSession) DiscoverTools(ctx context.Context) ([]*mcp.Tool, error) {
	s.mu.RLock()
	if s.session == nil {
		s.mu.RUnlock()
		return nil, fmt.Errorf("session未连接")
	}
	sess := s.session
	s.mu.RUnlock()

	result, err := sess.ListTools(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("发现工具失败: %w", err)
	}

	s.mu.Lock()
	s.tools = result.Tools
	s.mu.Unlock()

	return result.Tools, nil
}

// CallTool 调用指定工具（带超时和重试）
func (s *MCPSession) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	s.mu.RLock()
	if s.session == nil {
		s.mu.RUnlock()
		return nil, fmt.Errorf("session未连接")
	}
	sess := s.session
	s.mu.RUnlock()

	// 设置工具调用超时（默认 60 秒）
	callCtx, cancel := context.WithTimeout(ctx, s.config.GetTimeout())
	defer cancel()

	// 尝试调用，最多重试 2 次
	for attempt := 0; attempt <= 2; attempt++ {
		if attempt > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		result, err := sess.CallTool(callCtx, &mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		})
		if err != nil {
			logMCP("工具 '%s' 调用失败 (尝试 %d/2): %v", toolName, attempt+1, err)

			// 如果是超时错误，尝试重连
			if isTimeoutError(err) {
				go s.reconnect()
			}
			continue
		}

		return result, nil
	}

	return nil, fmt.Errorf("调用 MCP 工具 '%s' 失败: %w (已重试 2 次)", toolName, ctx.Err())
}

// isTimeoutError 检查错误是否为超时错误
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return len(errStr) > 0 && (errStr[len(errStr)-5:] == "timeout" || errStr[len(errStr)-12:] == "context deadline exceeded")
}

// Close 关闭连接
func (s *MCPSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 停止健康检查定时器
	if s.healthCheckTimer != nil {
		s.healthCheckTimer.Stop()
		s.healthCheckTimer = nil
	}

	if s.session != nil {
		err := s.session.Close()
		s.session = nil
		s.client = nil
		s.status = StatusDisconnected
		return err
	}
	return nil
}

// GetStatus 获取Session状态
func (s *MCPSession) GetStatus() ServerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := ServerStatus{
		ID:     s.id,
		Status: s.status,
		Tools:  len(s.tools),
	}

	if s.status == StatusConnected {
		now := time.Now()
		status.ConnectedAt = &now
		status.Uptime = now.Sub(s.lastConnectedAt)
	}

	if s.lastError != nil {
		status.Error = s.lastError.Error()
	}

	return status
}

// GetTools 获取已发现的工具列表
func (s *MCPSession) GetTools() []*mcp.Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*mcp.Tool, len(s.tools))
	copy(result, s.tools)
	return result
}

// GetToolSettings 获取工具设置配置
func (s *MCPSession) GetToolSettings() map[string]ToolSettingConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.toolsSettings == nil {
		return make(map[string]ToolSettingConfig)
	}
	result := make(map[string]ToolSettingConfig, len(s.toolsSettings))
	for k, v := range s.toolsSettings {
		result[k] = v
	}
	return result
}

// MCPToolInfo MCP工具信息缓存
type MCPToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// convertMCPResult 将MCP CallToolResult转换为内部ToolResult
func convertMCPResult(result *mcp.CallToolResult, serverID string) *tools.ToolResult {
	if result == nil {
		return tools.NewErrorResult(fmt.Sprintf("[MCP: %s] 收到空结果", serverID))
	}

	if result.IsError {
		// 工具执行错误
		errMsg := extractTextContent(result.Content)
		return tools.NewErrorResult(fmt.Sprintf("[MCP: %s] %s", serverID, errMsg))
	}

	// 提取内容
	textContent := extractTextContent(result.Content)

	// 如果有结构化内容，优先使用
	if result.StructuredContent != nil {
		return tools.NewSuccessResult(map[string]interface{}{
			"content":    textContent,
			"structured": result.StructuredContent,
			"server_id":  serverID,
		})
	}

	return tools.NewSuccessResult(map[string]interface{}{
		"content":   textContent,
		"server_id": serverID,
	})
}

// extractTextContent 从MCP Content列表中提取文本内容
func extractTextContent(contents []mcp.Content) string {
	var parts []string
	for _, c := range contents {
		// 类型断言获取TextContent
		switch v := c.(type) {
		case *mcp.TextContent:
			parts = append(parts, v.Text)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return joinTextParts(parts)
}

// joinTextParts 连接文本片段
func joinTextParts(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "\n"
		}
		result += p
	}
	return result
}

// logMCP 记录MCP日志（简化版，使用标准log）
func logMCP(format string, args ...interface{}) {
	msg := fmt.Sprintf("[MCP] "+format, args...)
	log.Println(msg)
}
