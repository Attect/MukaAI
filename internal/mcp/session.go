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
	id      string
	config  ServerConfig
	client  *mcp.Client
	session *mcp.ClientSession
	tools   []*mcp.Tool // 发现的工具列表
	mu      sync.RWMutex
	status  SessionStatus
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
}

// NewMCPSession 创建新的MCP Session
func NewMCPSession(id string, config ServerConfig) *MCPSession {
	return &MCPSession{
		id:     id,
		config: config,
		status: StatusDisconnected,
	}
}

// Connect 建立连接并初始化
func (s *MCPSession) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status = StatusConnecting

	// 创建MCP Client
	s.client = mcp.NewClient(
		&mcp.Implementation{
			Name:    "MukaAI",
			Version: "1.0.0",
		},
		nil, // 使用默认ClientOptions
	)

	// 根据Transport类型创建Transport
	transport, err := s.createTransport()
	if err != nil {
		s.status = StatusError
		return fmt.Errorf("创建transport失败: %w", err)
	}

	// 连接并初始化（SDK的Connect方法会自动完成initialize/initialized握手）
	session, err := s.client.Connect(ctx, transport, nil)
	if err != nil {
		s.status = StatusError
		return fmt.Errorf("连接MCP Server失败: %w", err)
	}

	s.session = session
	s.status = StatusConnected

	return nil
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

// CallTool 调用指定工具
func (s *MCPSession) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	s.mu.RLock()
	if s.session == nil {
		s.mu.RUnlock()
		return nil, fmt.Errorf("session未连接")
	}
	sess := s.session
	s.mu.RUnlock()

	result, err := sess.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		return nil, fmt.Errorf("调用MCP工具 '%s' 失败: %w", toolName, err)
	}

	return result, nil
}

// Close 关闭连接
func (s *MCPSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session != nil {
		err := s.session.Close()
		s.session = nil
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
