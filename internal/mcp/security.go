package mcp

import (
	"strings"
)

// SecurityPolicy 安全策略类型
type SecurityPolicy string

const (
	SecurityPolicyAllow   SecurityPolicy = "allow"
	SecurityPolicyConfirm SecurityPolicy = "confirm"
	SecurityPolicyDeny    SecurityPolicy = "deny"
)

// MCPSecurityChecker MCP工具安全策略检查器
// 根据配置的策略决定工具调用的安全级别
type MCPSecurityChecker struct {
	defaultPolicy   SecurityPolicy
	denyPatterns    []string // 拒绝的工具名模式
	confirmPatterns []string // 需确认的工具名模式
	allowPatterns   []string // 允许的工具名模式
}

// NewMCPSecurityChecker 创建MCP安全检查器
func NewMCPSecurityChecker(config *MCPSecurityConfig) *MCPSecurityChecker {
	if config == nil {
		return &MCPSecurityChecker{
			defaultPolicy: SecurityPolicyAllow,
		}
	}

	policy := SecurityPolicy(config.DefaultPolicy)
	if policy == "" {
		policy = SecurityPolicyAllow
	}

	return &MCPSecurityChecker{
		defaultPolicy:   policy,
		denyPatterns:    config.DenyTools,
		confirmPatterns: config.ConfirmTools,
		allowPatterns:   config.AllowTools,
	}
}

// CheckTool 检查工具调用的安全策略
// 返回策略：allow（放行）、confirm（需确认）、deny（拒绝）
// 检查优先级：deny > allow > confirm > default
func (c *MCPSecurityChecker) CheckTool(toolName string) SecurityPolicy {
	// 1. 先检查deny列表（最高优先级，黑名单永远优先）
	for _, pattern := range c.denyPatterns {
		if matchToolPattern(pattern, toolName) {
			return SecurityPolicyDeny
		}
	}

	// 2. 检查allow列表（白名单优先于默认策略）
	for _, pattern := range c.allowPatterns {
		if matchToolPattern(pattern, toolName) {
			return SecurityPolicyAllow
		}
	}

	// 3. 检查confirm列表
	for _, pattern := range c.confirmPatterns {
		if matchToolPattern(pattern, toolName) {
			return SecurityPolicyConfirm
		}
	}

	// 4. 使用默认策略
	return c.defaultPolicy
}

// matchToolPattern 检查工具名是否匹配模式
// 支持通配符 *，例如：
//   - "mcp_*_delete_*" 匹配所有Server的delete相关工具
//   - "mcp_filesystem_*" 匹配filesystem Server的所有工具
//   - "mcp_*_exec_*" 匹配所有Server的exec相关工具
func matchToolPattern(pattern, toolName string) bool {
	// 精确匹配
	if pattern == toolName {
		return true
	}

	// 无通配符则不匹配
	if !strings.Contains(pattern, "*") {
		return false
	}

	// 简单通配符匹配：将 * 转换为任意字符串
	return simpleWildcardMatch(pattern, toolName)
}

// simpleWildcardMatch 简单的通配符匹配
// 只支持 * 通配符，匹配任意字符序列
func simpleWildcardMatch(pattern, text string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 0 {
		return pattern == text
	}

	idx := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		pos := strings.Index(text[idx:], part)
		if pos < 0 {
			return false
		}
		// 第一段必须从文本开头匹配
		if i == 0 && pos != 0 {
			return false
		}
		idx += pos + len(part)
	}

	// 最后一段必须匹配到末尾（除非最后是*）
	if !strings.HasSuffix(pattern, "*") {
		lastPart := parts[len(parts)-1]
		if lastPart != "" && !strings.HasSuffix(text, lastPart) {
			return false
		}
	}

	return true
}
