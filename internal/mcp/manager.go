package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/Attect/MukaAI/internal/tools"
)

// MCPClientManager MCP客户端管理器
// 管理与多个MCP Server的连接，汇总工具并注册到ToolRegistry
type MCPClientManager struct {
	config   *MCPConfig
	registry *tools.ToolRegistry
	sessions map[string]*MCPSession // key: server id
	security *MCPSecurityChecker
	mu       sync.RWMutex
}

// NewMCPClientManager 创建MCP客户端管理器
func NewMCPClientManager(config *MCPConfig, registry *tools.ToolRegistry) *MCPClientManager {
	return &MCPClientManager{
		config:   config,
		registry: registry,
		sessions: make(map[string]*MCPSession),
		security: NewMCPSecurityChecker(&config.Security),
	}
}

// Initialize 初始化所有MCP连接
// 根据配置连接所有enabled的MCP Server，发现工具并注册
func (mgr *MCPClientManager) Initialize(ctx context.Context) error {
	if mgr.config == nil || !mgr.config.Enabled {
		return nil
	}

	enabledServers := mgr.config.GetEnabledServers()
	if len(enabledServers) == 0 {
		return nil
	}

	logMCP("开始初始化 %d 个MCP Server", len(enabledServers))

	var errs []error
	for _, serverCfg := range enabledServers {
		if err := mgr.connectServer(ctx, serverCfg); err != nil {
			logMCP("Server '%s' 连接失败: %v", serverCfg.ID, err)
			errs = append(errs, fmt.Errorf("server '%s': %w", serverCfg.ID, err))
			// 单个Server失败不影响其他Server
			continue
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d/%d 个MCP Server连接失败: %v", len(errs), len(enabledServers), errs)
	}

	logMCP("所有MCP Server初始化完成")
	return nil
}

// connectServer 连接单个Server
func (mgr *MCPClientManager) connectServer(ctx context.Context, cfg ServerConfig) error {
	// 解析环境变量引用
	if cfg.Env != nil {
		resolveMapEnvVars(cfg.Env)
	}
	if cfg.Headers != nil {
		resolveMapEnvVars(cfg.Headers)
	}

	// Prefix默认值为ID
	prefix := cfg.GetPrefix()

	// 创建Session（传入工具设置）
	session := NewMCPSession(cfg.ID, cfg)
	session.toolsSettings = cfg.ToolSettings

	// 连接
	logMCP("正在连接Server '%s' (transport: %s)...", cfg.ID, cfg.Transport)
	if err := session.Connect(ctx); err != nil {
		return err
	}

	// 发现工具
	mcpTools, err := session.DiscoverTools(ctx)
	if err != nil {
		// 发现工具失败时关闭连接
		_ = session.Close()
		return fmt.Errorf("发现工具失败: %w", err)
	}

	// 工具数量上限检查
	maxTools := mgr.config.Security.MaxTools
	if maxTools <= 0 {
		maxTools = 50
	}
	if len(mcpTools) > maxTools {
		logMCP("Server '%s' 工具数 %d 超过上限 %d，裁剪",
			cfg.ID, len(mcpTools), maxTools)
		mcpTools = mcpTools[:maxTools]
	}

	// 创建适配器并注册到ToolRegistry
	registered := 0
	for _, mcpTool := range mcpTools {
		// 检查工具设置：如果配置了禁用，跳过
		if cfg.ToolSettings != nil {
			if setting, ok := cfg.ToolSettings[mcpTool.Name]; ok && !setting.Enabled {
				logMCP("工具 '%s' 被配置禁用，跳过", mcpTool.Name)
				continue
			}
		}

		// 获取自定义描述
		customDesc := ""
		if cfg.ToolSettings != nil {
			if setting, ok := cfg.ToolSettings[mcpTool.Name]; ok {
				customDesc = setting.Description
			}
		}

		adapter := NewMCPToolAdapter(
			prefix,
			cfg.ID,
			mcpTool,
			session,
			cfg.GetTimeout(),
			mgr.security,
			cfg.ProjectPath,
			customDesc,
		)

		if err := mgr.registry.RegisterTool(adapter); err != nil {
			logMCP("注册工具 '%s' 失败: %v", adapter.Name(), err)
			continue
		}
		registered++
	}

	// 保存Session
	mgr.mu.Lock()
	mgr.sessions[cfg.ID] = session
	mgr.mu.Unlock()

	logMCP("Server '%s' 连接成功，注册了 %d/%d 个工具",
		cfg.ID, registered, len(mcpTools))

	return nil
}

// Shutdown 关闭所有MCP连接
// 优雅关闭所有Session，从ToolRegistry注销MCP工具
func (mgr *MCPClientManager) Shutdown() error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	var errs []error
	for id, session := range mgr.sessions {
		// 先注销该Server的所有工具
		mgr.unregisterServerTools(id)
		// 关闭连接
		if err := session.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭Server '%s' 失败: %w", id, err))
		}
		logMCP("Server '%s' 已关闭", id)
	}
	mgr.sessions = make(map[string]*MCPSession)

	if len(errs) > 0 {
		return fmt.Errorf("关闭MCP连接时出错: %v", errs)
	}
	return nil
}

// unregisterServerTools 从ToolRegistry注销指定Server的所有工具
func (mgr *MCPClientManager) unregisterServerTools(serverID string) {
	// ToolRegistry目前没有Unregister方法，这里记录日志
	// 工具会在下次启动时重新注册
	logMCP("Server '%s' 的工具将在下次启动时重新注册", serverID)
}

// GetServerStatus 获取所有Server的状态
func (mgr *MCPClientManager) GetServerStatus() []ServerStatus {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	statuses := make([]ServerStatus, 0, len(mgr.sessions))
	for _, session := range mgr.sessions {
		statuses = append(statuses, session.GetStatus())
	}
	return statuses
}

// GetMCPTools 获取所有MCP工具（已适配为内部Tool接口）
func (mgr *MCPClientManager) GetMCPTools() []tools.Tool {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	var result []tools.Tool
	for serverID, session := range mgr.sessions {
		mcpTools := session.GetTools()
		prefix := session.config.GetPrefix()
		toolSettings := session.GetToolSettings()
		for _, mcpTool := range mcpTools {
			// 检查工具设置：如果配置了禁用，跳过
			if toolSettings != nil {
				if setting, ok := toolSettings[mcpTool.Name]; ok && !setting.Enabled {
					continue
				}
			}

			// 获取自定义描述
			customDesc := ""
			if toolSettings != nil {
				if setting, ok := toolSettings[mcpTool.Name]; ok {
					customDesc = setting.Description
				}
			}

			adapter := NewMCPToolAdapter(
				prefix,
				serverID,
				mcpTool,
				session,
				session.config.GetTimeout(),
				mgr.security,
				session.config.ProjectPath,
				customDesc,
			)
			result = append(result, adapter)
		}
	}
	return result
}

// GetSessionCount 获取已连接的Session数量
func (mgr *MCPClientManager) GetSessionCount() int {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return len(mgr.sessions)
}

// ReloadServer 重新加载指定Server（关闭后重新连接）
func (mgr *MCPClientManager) ReloadServer(ctx context.Context, serverID string) error {
	mgr.mu.Lock()
	if session, ok := mgr.sessions[serverID]; ok {
		_ = session.Close()
		delete(mgr.sessions, serverID)
	}
	mgr.mu.Unlock()

	// 查找Server配置
	for _, cfg := range mgr.config.Servers {
		if cfg.ID == serverID {
			if !cfg.Enabled {
				return fmt.Errorf("Server '%s' 已禁用", serverID)
			}
			return mgr.connectServer(ctx, cfg)
		}
	}

	return fmt.Errorf("未找到Server '%s' 的配置", serverID)
}

// ensure MCPClientManager implements expected interface
var _ = []interface{}{(*MCPClientManager)(nil).Initialize, (*MCPClientManager)(nil).Shutdown}
