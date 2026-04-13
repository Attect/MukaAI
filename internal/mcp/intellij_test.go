//go:build manual

package mcp

import (
	"context"
	"testing"
	"time"
)

// TestIntelliJStreamableHTTP 测试通过Streamable HTTP模式连接IntelliJ IDEA MCP Server
// 需要IntelliJ IDEA正在运行且MCP Server端口64342可用
// 运行方式: go test -tags manual -run TestIntelliJStreamableHTTP ./internal/mcp/ -v -timeout 60s
func TestIntelliJStreamableHTTP(t *testing.T) {
	cfg := ServerConfig{
		ID:        "idea",
		Enabled:   true,
		Transport: "http",
		URL:       "http://127.0.0.1:64342/stream",
		Timeout:   60,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := NewMCPSession("idea", cfg)
	if err := session.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer session.Close()

	tools, err := session.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("发现工具失败: %v", err)
	}

	t.Logf("发现 %d 个工具:", len(tools))
	for _, tool := range tools {
		t.Logf("  - %s: %s", tool.Name, tool.Description)
	}

	if len(tools) == 0 {
		t.Error("期望发现至少一个工具，但没有发现任何工具")
	}
}

// TestIntelliJSSE 测试通过SSE模式连接IntelliJ IDEA MCP Server
// 需要IntelliJ IDEA正在运行且MCP Server端口64342可用
// 运行方式: go test -tags manual -run TestIntelliJSSE ./internal/mcp/ -v -timeout 60s
func TestIntelliJSSE(t *testing.T) {
	cfg := ServerConfig{
		ID:        "idea_sse",
		Enabled:   true,
		Transport: "sse",
		URL:       "http://127.0.0.1:64342/sse",
		Timeout:   60,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := NewMCPSession("idea_sse", cfg)
	if err := session.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer session.Close()

	tools, err := session.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("发现工具失败: %v", err)
	}

	t.Logf("发现 %d 个工具:", len(tools))
	for _, tool := range tools {
		t.Logf("  - %s: %s", tool.Name, tool.Description)
	}

	if len(tools) == 0 {
		t.Error("期望发现至少一个工具，但没有发现任何工具")
	}
}
