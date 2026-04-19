package model

// Config 模型服务配置
// 定义了连接模型服务所需的所有参数
type Config struct {
	Endpoint    string `yaml:"endpoint"`     // API端点地址，如 http://127.0.0.1:11453/v1/
	APIKey      string `yaml:"api_key"`      // API密钥，本地部署可能为no-key
	ModelName   string `yaml:"model_name"`   // 模型名称，如 Huihui-Qwen3.5-27B-abliterated.Q4_K_M
	ContextSize int    `yaml:"context_size"` // 上下文大小（token数），如200000
}

// DefaultConfig 返回默认配置
// 用于在配置文件缺失时提供合理的默认值
func DefaultConfig() *Config {
	return &Config{
		Endpoint:    "http://127.0.0.1:11453/v1/",
		APIKey:      "no-key",
		ModelName:   "mradermacher/Huihui-Qwen3.5-27B-abliterated-GGUF/Huihui-Qwen3.5-27B-abliterated.Q4_K_M",
		ContextSize: 200000,
	}
}

// Clone 返回配置的深拷贝
// 用于在读取配置时创建快照，避免并发读写竞态
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	return &Config{
		Endpoint:    c.Endpoint,
		APIKey:      c.APIKey,
		ModelName:   c.ModelName,
		ContextSize: c.ContextSize,
	}
}

// Validate 验证配置的有效性
// 确保必要的配置项不为空
func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return &ConfigError{Field: "endpoint", Message: "endpoint不能为空"}
	}
	if c.ModelName == "" {
		return &ConfigError{Field: "model_name", Message: "model_name不能为空"}
	}
	if c.ContextSize <= 0 {
		return &ConfigError{Field: "context_size", Message: "context_size必须大于0"}
	}
	return nil
}

// ConfigError 配置错误
// 用于表示配置验证失败的具体原因
type ConfigError struct {
	Field   string // 错误的字段名
	Message string // 错误消息
}

// Error 实现error接口
func (e *ConfigError) Error() string {
	return "配置错误 [" + e.Field + "]: " + e.Message
}
