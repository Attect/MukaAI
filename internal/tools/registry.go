package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// ToolRegistry 工具注册中心
// 管理所有可用工具，提供注册、查询、执行等功能
// 线程安全，支持并发访问
type ToolRegistry struct {
	// tools 存储已注册的工具，key为工具名称
	tools map[string]Tool

	// mu 保护tools map的读写锁
	mu sync.RWMutex
}

// NewToolRegistry 创建新的工具注册中心
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// RegisterTool 注册工具到注册中心
// 如果工具名称已存在，返回错误
func (r *ToolRegistry) RegisterTool(tool Tool) error {
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool '%s' already registered", name)
	}

	r.tools[name] = tool
	return nil
}

// MustRegisterTool 注册工具，如果失败则panic
// 用于初始化时注册工具
func (r *ToolRegistry) MustRegisterTool(tool Tool) {
	if err := r.RegisterTool(tool); err != nil {
		panic(fmt.Sprintf("failed to register tool: %v", err))
	}
}

// GetTool 根据名称获取工具
// 如果工具不存在，返回nil和false
func (r *ToolRegistry) GetTool(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// GetAllTools 返回所有已注册的工具
func (r *ToolRegistry) GetAllTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ToolSchema 工具的Schema定义，供模型调用
// 遵循OpenAI Function Calling格式
type ToolSchema struct {
	// Type 固定为"function"
	Type string `json:"type"`

	// Function 函数定义
	Function FunctionSchema `json:"function"`
}

// FunctionSchema 函数Schema定义
type FunctionSchema struct {
	// Name 函数名称
	Name string `json:"name"`

	// Description 函数描述
	Description string `json:"description"`

	// Parameters 参数Schema
	Parameters map[string]interface{} `json:"parameters"`
}

// GetAllToolSchemas 返回所有工具的Schema定义
// 用于提供给模型进行Function Calling
func (r *ToolRegistry) GetAllToolSchemas() []ToolSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make([]ToolSchema, 0, len(r.tools))
	for _, tool := range r.tools {
		schemas = append(schemas, ToolSchema{
			Type: "function",
			Function: FunctionSchema{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.Parameters(),
			},
		})
	}
	return schemas
}

// GetAllToolSchemasJSON 返回所有工具Schema的JSON格式
// 便于直接传递给模型或记录日志
func (r *ToolRegistry) GetAllToolSchemasJSON() (string, error) {
	schemas := r.GetAllToolSchemas()
	bytes, err := json.MarshalIndent(schemas, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal tool schemas: %w", err)
	}
	return string(bytes), nil
}

// ExecuteTool 执行指定工具
// name: 工具名称
// params: 工具参数，JSON字符串或已解析的map
func (r *ToolRegistry) ExecuteTool(ctx context.Context, name string, params interface{}) (*ToolResult, error) {
	tool, exists := r.GetTool(name)
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}

	// 解析参数
	var paramMap map[string]interface{}
	switch p := params.(type) {
	case string:
		if err := json.Unmarshal([]byte(p), &paramMap); err != nil {
			return nil, fmt.Errorf("failed to parse params: %w", err)
		}
	case map[string]interface{}:
		paramMap = p
	case nil:
		paramMap = make(map[string]interface{})
	default:
		// 尝试JSON序列化后再解析
		bytes, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		if err := json.Unmarshal(bytes, &paramMap); err != nil {
			return nil, fmt.Errorf("failed to parse params: %w", err)
		}
	}

	// 执行工具
	return tool.Execute(ctx, paramMap)
}

// ToolCount 返回已注册工具数量
func (r *ToolRegistry) ToolCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// ListToolNames 返回所有工具名称列表
func (r *ToolRegistry) ListToolNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// 全局默认注册中心
var defaultRegistry = NewToolRegistry()

// DefaultRegistry 返回默认的全局注册中心
func DefaultRegistry() *ToolRegistry {
	return defaultRegistry
}

// RegisterTool 全局注册工具到默认注册中心
func RegisterTool(tool Tool) error {
	return defaultRegistry.RegisterTool(tool)
}

// MustRegisterTool 全局注册工具到默认注册中心，失败则panic
func MustRegisterTool(tool Tool) {
	defaultRegistry.MustRegisterTool(tool)
}

// GetTool 全局获取工具
func GetTool(name string) (Tool, bool) {
	return defaultRegistry.GetTool(name)
}

// ExecuteTool 全局执行工具
func ExecuteTool(ctx context.Context, name string, params interface{}) (*ToolResult, error) {
	return defaultRegistry.ExecuteTool(ctx, name, params)
}

// GetAllToolSchemas 全局获取所有工具Schema
func GetAllToolSchemas() []ToolSchema {
	return defaultRegistry.GetAllToolSchemas()
}
