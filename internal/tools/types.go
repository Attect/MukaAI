// Package tools 提供Agent可调用的工具集
// 工具系统采用注册机制，支持动态扩展
package tools

import (
	"context"
	"encoding/json"
)

// Tool 定义工具接口，所有工具必须实现此接口
// 工具通过JSON Schema描述参数，供模型调用时参考
type Tool interface {
	// Name 返回工具名称，用于模型调用时的标识
	Name() string

	// Description 返回工具描述，帮助模型理解工具用途
	Description() string

	// Parameters 返回工具参数的JSON Schema
	// Schema格式遵循OpenAI Function Calling规范
	Parameters() map[string]interface{}

	// Execute 执行工具，返回结果
	// ctx: 上下文，用于取消操作
	// params: 工具参数，已从JSON解析为map
	Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
}

// ToolResult 工具执行结果
// 统一的结果格式，便于模型理解和处理
type ToolResult struct {
	// Success 是否执行成功
	Success bool `json:"success"`

	// Data 执行结果数据
	// 不同工具返回不同结构的数据
	Data interface{} `json:"data,omitempty"`

	// Error 错误信息，仅在失败时有值
	Error string `json:"error,omitempty"`

	// Metadata 额外元数据，如执行时间、文件大小等
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToJSON 将结果转换为JSON字符串
// 用于返回给模型或记录日志
func (r *ToolResult) ToJSON() string {
	bytes, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return `{"success":false,"error":"failed to marshal result"}`
	}
	return string(bytes)
}

// ToolParameter 定义工具参数的Schema结构
// 用于构建OpenAI Function Calling兼容的参数描述
type ToolParameter struct {
	// Type 参数类型：string, number, integer, boolean, array, object
	Type string `json:"type"`

	// Description 参数描述
	Description string `json:"description,omitempty"`

	// Enum 枚举值，限制参数可选范围
	Enum []string `json:"enum,omitempty"`

	// Required 是否必需
	Required bool `json:"-"`

	// Default 默认值
	Default interface{} `json:"default,omitempty"`

	// Properties 对象类型的属性定义
	Properties map[string]*ToolParameter `json:"properties,omitempty"`

	// Items 数组元素的Schema
	Items *ToolParameter `json:"items,omitempty"`

	// MinLength 字符串最小长度
	MinLength *int `json:"minLength,omitempty"`

	// MaxLength 字符串最大长度
	MaxLength *int `json:"maxLength,omitempty"`

	// Minimum 数值最小值
	Minimum *float64 `json:"minimum,omitempty"`

	// Maximum 数值最大值
	Maximum *float64 `json:"maximum,omitempty"`
}

// ToMap 将ToolParameter转换为map[string]interface{}
// 用于构建完整的JSON Schema
func (p *ToolParameter) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type": p.Type,
	}

	if p.Description != "" {
		result["description"] = p.Description
	}

	if len(p.Enum) > 0 {
		result["enum"] = p.Enum
	}

	if p.Default != nil {
		result["default"] = p.Default
	}

	if p.Properties != nil {
		props := make(map[string]interface{})
		for k, v := range p.Properties {
			props[k] = v.ToMap()
		}
		result["properties"] = props
	}

	if p.Items != nil {
		result["items"] = p.Items.ToMap()
	}

	if p.MinLength != nil {
		result["minLength"] = *p.MinLength
	}

	if p.MaxLength != nil {
		result["maxLength"] = *p.MaxLength
	}

	if p.Minimum != nil {
		result["minimum"] = *p.Minimum
	}

	if p.Maximum != nil {
		result["maximum"] = *p.Maximum
	}

	return result
}

// BuildSchema 构建完整的工具参数Schema
// 返回OpenAI Function Calling兼容的参数定义
func BuildSchema(properties map[string]*ToolParameter, required []string) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
	}

	props := schema["properties"].(map[string]interface{})
	for name, param := range properties {
		props[name] = param.ToMap()
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// NewSuccessResult 创建成功结果
func NewSuccessResult(data interface{}) *ToolResult {
	return &ToolResult{
		Success: true,
		Data:    data,
	}
}

// NewErrorResult 创建错误结果
func NewErrorResult(err string) *ToolResult {
	return &ToolResult{
		Success: false,
		Error:   err,
	}
}

// NewErrorResultWithError 创建带原始错误的错误结果
func NewErrorResultWithError(err string, originalErr error) *ToolResult {
	result := &ToolResult{
		Success: false,
		Error:   err,
	}
	if originalErr != nil {
		result.Metadata = map[string]interface{}{
			"original_error": originalErr.Error(),
		}
	}
	return result
}
