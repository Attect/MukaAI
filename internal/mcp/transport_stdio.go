package mcp

import (
	"fmt"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// createStdioTransport 创建stdio模式的Transport
// stdio模式通过启动子进程，使用stdin/stdout进行JSON-RPC通信
func (s *MCPSession) createStdioTransport() (mcp.Transport, error) {
	if s.config.Command == "" {
		return nil, fmt.Errorf("stdio模式必须提供command")
	}

	cmd := exec.Command(s.config.Command, s.config.Args...)

	// 设置环境变量
	if len(s.config.Env) > 0 {
		// 复制当前环境变量
		cmd.Env = cmd.Environ()
		for k, v := range s.config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// 使用SDK的CommandTransport
	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	return transport, nil
}
