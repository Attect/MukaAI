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

// Config 完整的应用配置
type Config struct {
	Model ModelConfig `yaml:"model"`
	Agent AgentConfig `yaml:"agent"`
	State StateConfig `yaml:"state"`
	Tools ToolsConfig `yaml:"tools"`
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

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Model: ModelConfig{
			Endpoint:    "http://127.0.0.1:11453/v1/",
			APIKey:      "no-key",
			ModelName:   "Huihui-Qwen3.5-27B-abliterated.Q4_K_M",
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
// 环境变量格式：AGENTPLUS_<SECTION>_<KEY>
// 例如：AGENTPLUS_MODEL_ENDPOINT=http://localhost:8080/v1/
func applyEnvOverrides(config *Config) {
	// Model配置
	if v := os.Getenv("AGENTPLUS_MODEL_ENDPOINT"); v != "" {
		config.Model.Endpoint = v
	}
	if v := os.Getenv("AGENTPLUS_MODEL_API_KEY"); v != "" {
		config.Model.APIKey = v
	}
	if v := os.Getenv("AGENTPLUS_MODEL_NAME"); v != "" {
		config.Model.ModelName = v
	}
	if v := os.Getenv("AGENTPLUS_MODEL_CONTEXT_SIZE"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			config.Model.ContextSize = i
		}
	}

	// Agent配置
	if v := os.Getenv("AGENTPLUS_AGENT_MAX_ITERATIONS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			config.Agent.MaxIterations = i
		}
	}
	if v := os.Getenv("AGENTPLUS_AGENT_TEMPERATURE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			config.Agent.Temperature = f
		}
	}

	// State配置
	if v := os.Getenv("AGENTPLUS_STATE_DIR"); v != "" {
		config.State.Dir = v
	}
	if v := os.Getenv("AGENTPLUS_STATE_AUTO_SAVE"); v != "" {
		config.State.AutoSave = strings.ToLower(v) == "true"
	}
	if v := os.Getenv("AGENTPLUS_STATE_CLEANUP_DAYS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			config.State.CleanupDays = i
		}
	}
	if v := os.Getenv("AGENTPLUS_STATE_CLEANUP_ENABLE"); v != "" {
		config.State.CleanupEnable = strings.ToLower(v) == "true"
	}

	// Tools配置
	if v := os.Getenv("AGENTPLUS_TOOLS_WORK_DIR"); v != "" {
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
