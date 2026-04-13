package mcp

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// createHTTPTransport 创建Streamable HTTP模式的Transport
// HTTP模式连接远程MCP Server，使用HTTP POST/GET进行通信
func (s *MCPSession) createHTTPTransport() (mcp.Transport, error) {
	if s.config.URL == "" {
		return nil, fmt.Errorf("http模式必须提供url")
	}

	// 构建自定义HTTP Client以支持自定义Headers
	var httpClient *http.Client
	if len(s.config.Headers) > 0 {
		httpClient = &http.Client{
			Transport: &headerTransport{
				base:    http.DefaultTransport,
				headers: s.config.Headers,
			},
		}
	}

	transport := &mcp.StreamableClientTransport{
		Endpoint:   s.config.URL,
		HTTPClient: httpClient,
	}

	return transport, nil
}

// createSSETransport 创建SSE（Server-Sent Events）模式的Transport
// SSE模式连接远程MCP Server，通过SSE接收服务端消息，通过HTTP POST发送客户端消息
// 适用于旧版MCP规范（2024-11-05）的SSE transport
func (s *MCPSession) createSSETransport() (mcp.Transport, error) {
	if s.config.URL == "" {
		return nil, fmt.Errorf("sse模式必须提供url")
	}

	// 构建自定义HTTP Client以支持自定义Headers
	var httpClient *http.Client
	if len(s.config.Headers) > 0 {
		httpClient = &http.Client{
			Transport: &headerTransport{
				base:    http.DefaultTransport,
				headers: s.config.Headers,
			},
		}
	}

	transport := &mcp.SSEClientTransport{
		Endpoint:   s.config.URL,
		HTTPClient: httpClient,
	}

	return transport, nil
}

// headerTransport 自定义HTTP Transport，在请求中添加自定义Headers
type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

// RoundTrip 实现http.RoundTripper接口
func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range t.headers {
		// 跳过可能被HTTP库管理的headers
		if strings.EqualFold(k, "Host") {
			continue
		}
		req.Header.Set(k, v)
	}
	return t.base.RoundTrip(req)
}
