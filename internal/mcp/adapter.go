package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Attect/MukaAI/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPToolAdapter 将MCP工具适配为内部Tool接口
// 实现tools.Tool接口，使MCP工具对上层Agent核心透明
type MCPToolAdapter struct {
	prefix            string // 工具名前缀（来自ServerConfig.Prefix）
	serverID          string // Server ID（用于标识来源）
	toolName          string // MCP原始工具名
	customDescription string // 自定义描述（来自ToolSettingConfig.Description）
	description       string // 最终使用的描述
	inputSchema       map[string]interface{}
	session           *MCPSession
	timeout           time.Duration
	security          *MCPSecurityChecker
	projectPath       string // 自动注入的项目路径
}

// NewMCPToolAdapter 创建MCP工具适配器
func NewMCPToolAdapter(prefix string, serverID string, mcpTool *mcp.Tool, session *MCPSession, timeout time.Duration, security *MCPSecurityChecker, projectPath string, customDescription string) *MCPToolAdapter {
	// 提取description：优先使用自定义描述
	desc := customDescription
	if desc == "" {
		desc = mcpTool.Description
		if desc == "" && mcpTool.Title != "" {
			desc = mcpTool.Title
		}
	}

	// 转换InputSchema
	var schema map[string]interface{}
	if mcpTool.InputSchema != nil {
		switch v := mcpTool.InputSchema.(type) {
		case map[string]interface{}:
			schema = v
		case json.RawMessage:
			_ = json.Unmarshal(v, &schema)
		default:
			// 尝试JSON序列化再反序列化
			data, err := json.Marshal(mcpTool.InputSchema)
			if err == nil {
				_ = json.Unmarshal(data, &schema)
			}
		}
	}
	if schema == nil {
		schema = map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}

	return &MCPToolAdapter{
		prefix:            prefix,
		serverID:          serverID,
		toolName:          mcpTool.Name,
		customDescription: customDescription,
		description:       desc,
		inputSchema:       schema,
		session:           session,
		timeout:           timeout,
		security:          security,
		projectPath:       projectPath,
	}
}

// Name 实现tools.Tool接口
// 返回带前缀的工具名: mcp_{prefix}_{tool_name}
func (a *MCPToolAdapter) Name() string {
	return fmt.Sprintf("mcp_%s_%s", a.prefix, a.toolName)
}

// Description 实现tools.Tool接口
// 返回带MCP标识的工具描述
func (a *MCPToolAdapter) Description() string {
	prefix := fmt.Sprintf("[MCP: %s]", a.serverID)
	if a.customDescription != "" {
		return fmt.Sprintf("%s %s", prefix, a.customDescription)
	}
	if a.description != "" {
		return fmt.Sprintf("%s %s", prefix, a.description)
	}
	return fmt.Sprintf("%s MCP tool: %s", prefix, a.toolName)
}

// Parameters 实现tools.Tool接口
// 返回工具的JSON Schema参数定义
func (a *MCPToolAdapter) Parameters() map[string]interface{} {
	return a.inputSchema
}

// Execute 实现tools.Tool接口
// 调用MCP Server上的工具并转换结果
func (a *MCPToolAdapter) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	// 1. 安全策略检查
	if a.security != nil {
		verdict := a.security.CheckTool(a.Name())
		switch verdict {
		case SecurityPolicyDeny:
			return tools.NewErrorResult(fmt.Sprintf("MCP工具 '%s' 被安全策略拒绝", a.Name())), nil
		case SecurityPolicyConfirm:
			// TODO: 集成用户确认机制（CLI模式询问，GUI模式弹窗）
			// 当前先放行，后续迭代实现
		}
	}

	// 1.5 自动注入projectPath（如果adapter配置了且params中没有）
	if a.projectPath != "" {
		if params == nil {
			params = make(map[string]interface{})
		}
		if _, exists := params["projectPath"]; !exists {
			params["projectPath"] = a.projectPath
		}
	}

	// 2. 设置超时
	timeoutCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	// 3. 调用MCP工具
	result, err := a.session.CallTool(timeoutCtx, a.toolName, params)
	if err != nil {
		// 区分超时错误和其他错误
		if ctx.Err() == context.DeadlineExceeded || timeoutCtx.Err() == context.DeadlineExceeded {
			return tools.NewErrorResult(fmt.Sprintf("[MCP: %s] 工具调用超时 (%v)", a.serverID, a.timeout)), nil
		}
		return tools.NewErrorResult(fmt.Sprintf("[MCP: %s] 工具调用失败: %v", a.serverID, err)), nil
	}

	// 4. 转换结果
	return convertMCPResult(result, a.serverID), nil
}

// ExtractServerID 从MCP工具名中提取Server ID（实际返回prefix）
// 输入格式: mcp_{prefix}_{tool_name}
// 如果不是MCP工具名，返回空字符串
func ExtractServerID(toolName string) string {
	if !strings.HasPrefix(toolName, "mcp_") {
		return ""
	}
	rest := toolName[4:] // 去掉 "mcp_"
	parts := strings.SplitN(rest, "_", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

// ExtractPrefix 从MCP工具名中提取前缀（与ExtractServerID等价，语义更清晰）
func ExtractPrefix(toolName string) string {
	return ExtractServerID(toolName)
}

// ExtractOriginalToolName 从MCP工具名中提取原始工具名
// 输入格式: mcp_{prefix}_{tool_name}
func ExtractOriginalToolName(toolName string) string {
	if !strings.HasPrefix(toolName, "mcp_") {
		return toolName
	}
	rest := toolName[4:]
	parts := strings.SplitN(rest, "_", 2)
	if len(parts) < 2 {
		return toolName
	}
	return parts[1]
}

// IsMCPToolName 检查工具名是否为MCP工具
func IsMCPToolName(toolName string) bool {
	return strings.HasPrefix(toolName, "mcp_") && strings.Count(toolName[4:], "_") >= 1
}
