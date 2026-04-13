package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Model.Endpoint == "" {
		t.Error("Model.Endpoint should not be empty")
	}
	if cfg.Model.ModelName == "" {
		t.Error("Model.ModelName should not be empty")
	}
	if cfg.Model.ContextSize <= 0 {
		t.Error("Model.ContextSize should be positive")
	}
	if cfg.Agent.MaxIterations <= 0 {
		t.Error("Agent.MaxIterations should be positive")
	}
	if cfg.State.Dir == "" {
		t.Error("State.Dir should not be empty")
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
model:
  endpoint: "http://localhost:8080/v1/"
  api_key: "test-key"
  model_name: "test-model"
  context_size: 100000

agent:
  max_iterations: 50
  temperature: 0.5

state:
  dir: "./test-state"
  auto_save: false

tools:
  work_dir: "./test-work"
  allow_commands:
    - "go"
    - "git"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 加载配置
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// 验证配置值
	if cfg.Model.Endpoint != "http://localhost:8080/v1/" {
		t.Errorf("Model.Endpoint = %s, want http://localhost:8080/v1/", cfg.Model.Endpoint)
	}
	if cfg.Model.APIKey != "test-key" {
		t.Errorf("Model.APIKey = %s, want test-key", cfg.Model.APIKey)
	}
	if cfg.Model.ModelName != "test-model" {
		t.Errorf("Model.ModelName = %s, want test-model", cfg.Model.ModelName)
	}
	if cfg.Model.ContextSize != 100000 {
		t.Errorf("Model.ContextSize = %d, want 100000", cfg.Model.ContextSize)
	}
	if cfg.Agent.MaxIterations != 50 {
		t.Errorf("Agent.MaxIterations = %d, want 50", cfg.Agent.MaxIterations)
	}
	if cfg.Agent.Temperature != 0.5 {
		t.Errorf("Agent.Temperature = %f, want 0.5", cfg.Agent.Temperature)
	}
	if cfg.State.Dir != "./test-state" {
		t.Errorf("State.Dir = %s, want ./test-state", cfg.State.Dir)
	}
	if cfg.State.AutoSave {
		t.Error("State.AutoSave should be false")
	}
	if cfg.Tools.WorkDir != "./test-work" {
		t.Errorf("Tools.WorkDir = %s, want ./test-work", cfg.Tools.WorkDir)
	}
	if len(cfg.Tools.AllowCommands) != 2 {
		t.Errorf("len(Tools.AllowCommands) = %d, want 2", len(cfg.Tools.AllowCommands))
	}
}

func TestLoadConfigNonExistent(t *testing.T) {
	// 加载不存在的配置文件应该返回默认配置
	cfg, err := LoadConfig("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig should not fail for non-existent file: %v", err)
	}

	// 应该是默认配置
	defaultCfg := DefaultConfig()
	if cfg.Model.Endpoint != defaultCfg.Model.Endpoint {
		t.Error("Should return default config for non-existent file")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	// 创建无效的YAML文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `
model:
  endpoint: [invalid
  api_key: "test"
`
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 加载配置应该失败
	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig should fail for invalid YAML")
	}
}

func TestEnvOverrides(t *testing.T) {
	// 设置环境变量（使用代码中实际的前缀 MUKAAI_）
	os.Setenv("MUKAAI_MODEL_ENDPOINT", "http://env-test:9090/v1/")
	os.Setenv("MUKAAI_MODEL_API_KEY", "env-key")
	os.Setenv("MUKAAI_MODEL_NAME", "env-model")
	os.Setenv("MUKAAI_MODEL_CONTEXT_SIZE", "50000")
	os.Setenv("MUKAAI_AGENT_MAX_ITERATIONS", "200")
	os.Setenv("MUKAAI_AGENT_TEMPERATURE", "0.9")
	os.Setenv("MUKAAI_STATE_DIR", "/env/state")
	os.Setenv("MUKAAI_STATE_AUTO_SAVE", "false")
	os.Setenv("MUKAAI_TOOLS_WORK_DIR", "/env/work")

	defer func() {
		os.Unsetenv("MUKAAI_MODEL_ENDPOINT")
		os.Unsetenv("MUKAAI_MODEL_API_KEY")
		os.Unsetenv("MUKAAI_MODEL_NAME")
		os.Unsetenv("MUKAAI_MODEL_CONTEXT_SIZE")
		os.Unsetenv("MUKAAI_AGENT_MAX_ITERATIONS")
		os.Unsetenv("MUKAAI_AGENT_TEMPERATURE")
		os.Unsetenv("MUKAAI_STATE_DIR")
		os.Unsetenv("MUKAAI_STATE_AUTO_SAVE")
		os.Unsetenv("MUKAAI_TOOLS_WORK_DIR")
	}()

	// 加载配置（使用不存在的文件，这样会使用默认值+环境变量）
	cfg, err := LoadConfig("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// 验证环境变量覆盖
	if cfg.Model.Endpoint != "http://env-test:9090/v1/" {
		t.Errorf("Model.Endpoint = %s, want http://env-test:9090/v1/", cfg.Model.Endpoint)
	}
	if cfg.Model.APIKey != "env-key" {
		t.Errorf("Model.APIKey = %s, want env-key", cfg.Model.APIKey)
	}
	if cfg.Model.ModelName != "env-model" {
		t.Errorf("Model.ModelName = %s, want env-model", cfg.Model.ModelName)
	}
	if cfg.Model.ContextSize != 50000 {
		t.Errorf("Model.ContextSize = %d, want 50000", cfg.Model.ContextSize)
	}
	if cfg.Agent.MaxIterations != 200 {
		t.Errorf("Agent.MaxIterations = %d, want 200", cfg.Agent.MaxIterations)
	}
	if cfg.Agent.Temperature != 0.9 {
		t.Errorf("Agent.Temperature = %f, want 0.9", cfg.Agent.Temperature)
	}
	if cfg.State.Dir != "/env/state" {
		t.Errorf("State.Dir = %s, want /env/state", cfg.State.Dir)
	}
	if cfg.State.AutoSave {
		t.Error("State.AutoSave should be false")
	}
	if cfg.Tools.WorkDir != "/env/work" {
		t.Errorf("Tools.WorkDir = %s, want /env/work", cfg.Tools.WorkDir)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "empty endpoint",
			config: &Config{
				Model: ModelConfig{
					Endpoint:    "",
					ModelName:   "test",
					ContextSize: 1000,
				},
				Agent: AgentConfig{
					MaxIterations: 10,
					Temperature:   0.7,
				},
				State: StateConfig{Dir: "."},
			},
			wantErr: true,
		},
		{
			name: "empty model name",
			config: &Config{
				Model: ModelConfig{
					Endpoint:    "http://localhost",
					ModelName:   "",
					ContextSize: 1000,
				},
				Agent: AgentConfig{
					MaxIterations: 10,
					Temperature:   0.7,
				},
				State: StateConfig{Dir: "."},
			},
			wantErr: true,
		},
		{
			name: "invalid context size",
			config: &Config{
				Model: ModelConfig{
					Endpoint:    "http://localhost",
					ModelName:   "test",
					ContextSize: 0,
				},
				Agent: AgentConfig{
					MaxIterations: 10,
					Temperature:   0.7,
				},
				State: StateConfig{Dir: "."},
			},
			wantErr: true,
		},
		{
			name: "invalid max iterations",
			config: &Config{
				Model: ModelConfig{
					Endpoint:    "http://localhost",
					ModelName:   "test",
					ContextSize: 1000,
				},
				Agent: AgentConfig{
					MaxIterations: 0,
					Temperature:   0.7,
				},
				State: StateConfig{Dir: "."},
			},
			wantErr: true,
		},
		{
			name: "invalid temperature",
			config: &Config{
				Model: ModelConfig{
					Endpoint:    "http://localhost",
					ModelName:   "test",
					ContextSize: 1000,
				},
				Agent: AgentConfig{
					MaxIterations: 10,
					Temperature:   3.0,
				},
				State: StateConfig{Dir: "."},
			},
			wantErr: true,
		},
		{
			name: "empty state dir",
			config: &Config{
				Model: ModelConfig{
					Endpoint:    "http://localhost",
					ModelName:   "test",
					ContextSize: 1000,
				},
				Agent: AgentConfig{
					MaxIterations: 10,
					Temperature:   0.7,
				},
				State: StateConfig{Dir: ""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetAbsoluteWorkDir(t *testing.T) {
	cfg := &Config{
		Tools: ToolsConfig{WorkDir: "."},
	}

	absPath, err := cfg.GetAbsoluteWorkDir()
	if err != nil {
		t.Fatalf("GetAbsoluteWorkDir failed: %v", err)
	}

	if !filepath.IsAbs(absPath) {
		t.Errorf("GetAbsoluteWorkDir should return absolute path, got %s", absPath)
	}
}

func TestGetAbsoluteStateDir(t *testing.T) {
	cfg := &Config{
		State: StateConfig{Dir: "./state"},
	}

	absPath, err := cfg.GetAbsoluteStateDir()
	if err != nil {
		t.Fatalf("GetAbsoluteStateDir failed: %v", err)
	}

	if !filepath.IsAbs(absPath) {
		t.Errorf("GetAbsoluteStateDir should return absolute path, got %s", absPath)
	}
}
