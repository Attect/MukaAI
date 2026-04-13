// Package mcp 提供MCP（Model Context Protocol）客户端支持
// 使MukaAI能连接外部MCP Server获取工具
package mcp

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// MCPConfig MCP配置
type MCPConfig struct {
	Enabled  bool              `yaml:"enabled"`
	Servers  []ServerConfig    `yaml:"servers"`
	Security MCPSecurityConfig `yaml:"security"`
}

// ServerConfig 单个MCP Server配置
type ServerConfig struct {
	ID        string `yaml:"id"`
	Enabled   bool   `yaml:"enabled"`
	Transport string `yaml:"transport"` // "stdio" | "http" | "sse"
	// stdio模式配置
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args"`
	Env     map[string]string `yaml:"env"`
	// http模式配置
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
	// 通用配置
	Timeout     int    `yaml:"timeout"`      // 秒
	ProjectPath string `yaml:"project_path"` // 项目路径，自动注入到工具参数
}

// MCPSecurityConfig MCP安全策略配置
type MCPSecurityConfig struct {
	DefaultPolicy string   `yaml:"default_policy"` // allow | confirm | deny
	DenyTools     []string `yaml:"deny_tools"`
	ConfirmTools  []string `yaml:"confirm_tools"`
	AllowTools    []string `yaml:"allow_tools"`
	MaxTools      int      `yaml:"max_tools"` // 单个Server最大工具数，默认50
}

// DefaultMCPConfig 返回默认MCP配置
func DefaultMCPConfig() *MCPConfig {
	return &MCPConfig{
		Enabled: false,
		Security: MCPSecurityConfig{
			DefaultPolicy: "allow",
			MaxTools:      50,
		},
	}
}

// GetTimeout 获取超时时间（带默认值）
func (c *ServerConfig) GetTimeout() time.Duration {
	if c.Timeout <= 0 {
		return 30 * time.Second
	}
	if c.Timeout > 300 {
		return 300 * time.Second
	}
	return time.Duration(c.Timeout) * time.Second
}

// serverIDPattern Server ID只允许字母、数字、下划线
var serverIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_]{1,32}$`)

// Validate 验证MCP配置有效性
func (c *MCPConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	// 验证Server ID唯一性
	seen := make(map[string]bool)
	for i, s := range c.Servers {
		if !s.Enabled {
			continue
		}
		// ID格式校验
		if !serverIDPattern.MatchString(s.ID) {
			return fmt.Errorf("mcp server[%d]: id '%s' 不合法，只允许字母、数字、下划线，最长32字符", i, s.ID)
		}
		if seen[s.ID] {
			return fmt.Errorf("mcp server[%d]: id '%s' 重复", i, s.ID)
		}
		seen[s.ID] = true

		// Transport校验
		switch s.Transport {
		case "stdio":
			if s.Command == "" {
				return fmt.Errorf("mcp server[%d] '%s': stdio模式必须提供command", i, s.ID)
			}
		case "http":
			if s.URL == "" {
				return fmt.Errorf("mcp server[%d] '%s': http模式必须提供url", i, s.ID)
			}
		case "sse":
			if s.URL == "" {
				return fmt.Errorf("mcp server[%d] '%s': sse模式必须提供url", i, s.ID)
			}
		default:
			return fmt.Errorf("mcp server[%d] '%s': 不支持的transport类型 '%s'，仅支持 stdio、http 和 sse", i, s.ID, s.Transport)
		}

		// 超时范围校验
		if s.Timeout != 0 && (s.Timeout < 5 || s.Timeout > 300) {
			return fmt.Errorf("mcp server[%d] '%s': timeout必须在5-300秒之间", i, s.ID)
		}
	}

	// 安全策略校验
	switch c.Security.DefaultPolicy {
	case "allow", "confirm", "deny":
		// 合法
	default:
		return fmt.Errorf("mcp.security.default_policy 必须是 allow、confirm 或 deny")
	}

	return nil
}

// ResolveEnvVars 解析配置值中的 ${VAR} 环境变量引用
// 将 ${VAR_NAME} 替换为系统环境变量值，未定义时替换为空字符串
func ResolveEnvVars(value string) string {
	return os.ExpandEnv(strings.ReplaceAll(value, "${", "${"))
}

// resolveMapEnvVars 解析map中所有值的环境变量引用
func resolveMapEnvVars(m map[string]string) {
	for k, v := range m {
		m[k] = os.ExpandEnv(v)
	}
}

// GetEnabledServers 获取所有已启用的Server配置
func (c *MCPConfig) GetEnabledServers() []ServerConfig {
	var result []ServerConfig
	for _, s := range c.Servers {
		if s.Enabled {
			result = append(result, s)
		}
	}
	return result
}
