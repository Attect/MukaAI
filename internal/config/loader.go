// Package config 提供配置加载和管理功能
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// CompressorConfig 上下文压缩器配置
type CompressorConfig struct {
	TriggerThreshold    float64 `yaml:"trigger_threshold"`      // 触发压缩的上下文使用阈值（0.0-1.0），默认0.8
	MinMessagesToKeep   int     `yaml:"min_messages_to_keep"`   // 压缩后保留的最小消息数量，默认10
	MaxMessagesToKeep   int     `yaml:"max_messages_to_keep"`   // 压缩后保留的最大消息数量，默认20
	KeepRecentToolCalls int     `yaml:"keep_recent_tool_calls"` // 保留最近N次工具调用，默认3
	SummaryMaxLength    int     `yaml:"summary_max_length"`     // 摘要的最大长度（字符数），默认2000
	LLMSummaryTimeout   int     `yaml:"llm_summary_timeout"`    // LLM摘要超时时间（秒），默认10
}

// DefaultCompressorConfig 返回默认的压缩器配置
func DefaultCompressorConfig() CompressorConfig {
	return CompressorConfig{
		TriggerThreshold:    0.8,
		MinMessagesToKeep:   10,
		MaxMessagesToKeep:   20,
		KeepRecentToolCalls: 3,
		SummaryMaxLength:    2000,
		LLMSummaryTimeout:   10,
	}
}

// Config 完整的应用配置
type Config struct {
	Model      ModelConfig      `yaml:"model"`
	Agent      AgentConfig      `yaml:"agent"`
	State      StateConfig      `yaml:"state"`
	Tools      ToolsConfig      `yaml:"tools"`
	MCP        MCPConfig        `yaml:"mcp"`
	LSP        LSPConfig        `yaml:"lsp"`
	Logging    LogConfig        `yaml:"logging"`
	Compressor CompressorConfig `yaml:"compressor"`
}

// LogConfig 日志配置
type LogConfig struct {
	LogPath string `yaml:"log_path"` // 对话日志文件路径，为空则不记录
}

// MCPConfig MCP（Model Context Protocol）配置
type MCPConfig struct {
	Enabled  bool              `yaml:"enabled"`
	Servers  []MCPServerConfig `yaml:"servers"`
	Security MCPSecurityConfig `yaml:"security"`
}

// MCPServerConfig 单个MCP Server配置
type MCPServerConfig struct {
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
	Timeout      int                             `yaml:"timeout"`      // 秒
	ProjectPath  string                          `yaml:"project_path"` // 项目路径，自动注入到MCP工具参数
	Prefix       string                          `yaml:"prefix"`       // 工具名前缀，默认使用ID
	ToolSettings map[string]MCPToolSettingConfig `yaml:"tools"`        // 每个工具的独立配置
}

// MCPToolSettingConfig 单个MCP工具的独立配置
type MCPToolSettingConfig struct {
	Enabled     bool   `yaml:"enabled"`     // 是否启用该工具，默认true
	Description string `yaml:"description"` // 自定义工具描述/提示词，覆盖MCP返回的默认描述
}

// MCPSecurityConfig MCP安全策略配置
type MCPSecurityConfig struct {
	DefaultPolicy string   `yaml:"default_policy"` // allow | confirm | deny
	DenyTools     []string `yaml:"deny_tools"`
	ConfirmTools  []string `yaml:"confirm_tools"`
	AllowTools    []string `yaml:"allow_tools"`
	MaxTools      int      `yaml:"max_tools"` // 单个Server最大工具数，默认50
}

// ModelConfig 模型服务配置
type ModelConfig struct {
	Endpoint    string `yaml:"endpoint"`
	APIKey      string `yaml:"api_key"`
	ModelName   string `yaml:"model_name"`
	ContextSize int    `yaml:"context_size"`
}

// AgentConfig Agent行为配置
type AgentConfig struct {
	MaxIterations int     `yaml:"max_iterations"`
	Temperature   float64 `yaml:"temperature"`
}

// StateConfig 状态管理配置
type StateConfig struct {
	Dir           string `yaml:"dir"`
	AutoSave      bool   `yaml:"auto_save"`
	CleanupDays   int    `yaml:"cleanup_days"`   // 过期清理保留天数，0表示使用默认值30
	CleanupEnable bool   `yaml:"cleanup_enable"` // 是否启用自动清理，默认true
}

// ToolsConfig 工具配置
type ToolsConfig struct {
	WorkDir       string   `yaml:"work_dir"`
	AllowCommands []string `yaml:"allow_commands"`
}

// LSPConfig LSP（Language Server Protocol）配置
type LSPConfig struct {
	Enabled     bool                      `yaml:"enabled"`
	IdleTimeout int                       `yaml:"idle_timeout"` // 空闲超时（秒），默认600
	Servers     map[string]LSPServerEntry `yaml:"servers"`
}

// LSPServerEntry 单个语言服务器配置
type LSPServerEntry struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Model: ModelConfig{
			Endpoint:    "http://127.0.0.1:11453/v1/",
			APIKey:      "no-key",
			ModelName:   "mradermacher/Huihui-Qwen3.5-27B-abliterated-GGUF/Huihui-Qwen3.5-27B-abliterated.Q4_K_M",
			ContextSize: 200000,
		},
		Agent: AgentConfig{
			MaxIterations: 100,
			Temperature:   0.7,
		},
		State: StateConfig{
			Dir:           "./state",
			AutoSave:      true,
			CleanupDays:   30,
			CleanupEnable: true,
		},
		Tools: ToolsConfig{
			WorkDir:       ".",
			AllowCommands: []string{},
		},
		MCP: MCPConfig{
			Enabled: false,
			Security: MCPSecurityConfig{
				DefaultPolicy: "allow",
				MaxTools:      50,
			},
		},
		LSP: LSPConfig{
			Enabled:     false,
			IdleTimeout: 600,
			Servers: map[string]LSPServerEntry{
				"go":         {Command: "gopls"},
				"typescript": {Command: "typescript-language-server", Args: []string{"--stdio"}},
				"python":     {Command: "pylsp"},
			},
		},
		Compressor: DefaultCompressorConfig(),
	}
}

// LoadConfig 从文件加载配置
// path: 配置文件路径
// 返回加载的配置和错误信息
func LoadConfig(path string) (*Config, error) {
	// 从默认配置开始
	config := DefaultConfig()

	// 如果配置文件存在，读取并解析
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to access config file: %w", err)
	}

	// 应用环境变量覆盖
	applyEnvOverrides(config)

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// applyEnvOverrides 应用环境变量覆盖配置
// 环境变量格式：MUKAAI_<SECTION>_<KEY>
// 例如：MUKAAI_MODEL_ENDPOINT=http://localhost:8080/v1/
func applyEnvOverrides(config *Config) {
	// Model配置
	if v := os.Getenv("MUKAAI_MODEL_ENDPOINT"); v != "" {
		config.Model.Endpoint = v
	}
	if v := os.Getenv("MUKAAI_MODEL_API_KEY"); v != "" {
		config.Model.APIKey = v
	}
	if v := os.Getenv("MUKAAI_MODEL_NAME"); v != "" {
		config.Model.ModelName = v
	}
	if v := os.Getenv("MUKAAI_MODEL_CONTEXT_SIZE"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			config.Model.ContextSize = i
		}
	}

	// Agent配置
	if v := os.Getenv("MUKAAI_AGENT_MAX_ITERATIONS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			config.Agent.MaxIterations = i
		}
	}
	if v := os.Getenv("MUKAAI_AGENT_TEMPERATURE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			config.Agent.Temperature = f
		}
	}

	// State配置
	if v := os.Getenv("MUKAAI_STATE_DIR"); v != "" {
		config.State.Dir = v
	}
	if v := os.Getenv("MUKAAI_STATE_AUTO_SAVE"); v != "" {
		config.State.AutoSave = strings.ToLower(v) == "true"
	}
	if v := os.Getenv("MUKAAI_STATE_CLEANUP_DAYS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			config.State.CleanupDays = i
		}
	}
	if v := os.Getenv("MUKAAI_STATE_CLEANUP_ENABLE"); v != "" {
		config.State.CleanupEnable = strings.ToLower(v) == "true"
	}

	// Tools配置
	if v := os.Getenv("MUKAAI_TOOLS_WORK_DIR"); v != "" {
		config.Tools.WorkDir = v
	}
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	// 验证Model配置
	if c.Model.Endpoint == "" {
		return fmt.Errorf("model.endpoint cannot be empty")
	}
	if c.Model.ModelName == "" {
		return fmt.Errorf("model.model_name cannot be empty")
	}
	if c.Model.ContextSize <= 0 {
		return fmt.Errorf("model.context_size must be greater than 0")
	}

	// 验证Agent配置
	if c.Agent.MaxIterations <= 0 {
		return fmt.Errorf("agent.max_iterations must be greater than 0")
	}
	if c.Agent.Temperature < 0 || c.Agent.Temperature > 2 {
		return fmt.Errorf("agent.temperature must be between 0 and 2")
	}

	// 验证State配置
	if c.State.Dir == "" {
		return fmt.Errorf("state.dir cannot be empty")
	}

	// 验证Tools配置
	if c.Tools.WorkDir == "" {
		c.Tools.WorkDir = "."
	}

	// 验证MCP配置
	if c.MCP.Enabled {
		if err := c.validateMCPConfig(); err != nil {
			return err
		}
	}

	return nil
}

// validateMCPConfig 验证MCP配置
func (c *Config) validateMCPConfig() error {
	seen := make(map[string]bool)
	for i, s := range c.MCP.Servers {
		if !s.Enabled {
			continue
		}
		if s.ID == "" {
			return fmt.Errorf("mcp.servers[%d]: id不能为空", i)
		}
		if seen[s.ID] {
			return fmt.Errorf("mcp.servers[%d]: id '%s' 重复", i, s.ID)
		}
		seen[s.ID] = true

		switch s.Transport {
		case "stdio":
			if s.Command == "" {
				return fmt.Errorf("mcp.servers[%d] '%s': stdio模式必须提供command", i, s.ID)
			}
		case "http":
			if s.URL == "" {
				return fmt.Errorf("mcp.servers[%d] '%s': http模式必须提供url", i, s.ID)
			}
		case "sse":
			if s.URL == "" {
				return fmt.Errorf("mcp.servers[%d] '%s': sse模式必须提供url", i, s.ID)
			}
		case "":
			return fmt.Errorf("mcp.servers[%d] '%s': 必须指定transport类型", i, s.ID)
		default:
			return fmt.Errorf("mcp.servers[%d] '%s': 不支持的transport类型 '%s'", i, s.ID, s.Transport)
		}

		if s.Timeout != 0 && (s.Timeout < 5 || s.Timeout > 300) {
			return fmt.Errorf("mcp.servers[%d] '%s': timeout必须在5-300秒之间", i, s.ID)
		}
	}

	// 验证安全策略
	switch c.MCP.Security.DefaultPolicy {
	case "", "allow", "confirm", "deny":
		// 合法（空值使用默认allow）
	default:
		return fmt.Errorf("mcp.security.default_policy 必须是 allow、confirm 或 deny")
	}

	return nil
}

// GetAbsoluteWorkDir 获取绝对工作目录
func (c *Config) GetAbsoluteWorkDir() (string, error) {
	if filepath.IsAbs(c.Tools.WorkDir) {
		return c.Tools.WorkDir, nil
	}
	return filepath.Abs(c.Tools.WorkDir)
}

// GetAbsoluteStateDir 获取绝对状态目录
func (c *Config) GetAbsoluteStateDir() (string, error) {
	if filepath.IsAbs(c.State.Dir) {
		return c.State.Dir, nil
	}
	return filepath.Abs(c.State.Dir)
}

// ToModelConfig 转换为model包的Config
func (c *Config) ToModelConfig() *ModelConfig {
	return &c.Model
}
