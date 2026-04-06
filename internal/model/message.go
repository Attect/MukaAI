package model

import "encoding/json"

// Role 消息角色类型
type Role string

const (
	RoleSystem    Role = "system"    // 系统消息，用于设置Agent行为
	RoleUser      Role = "user"      // 用户消息
	RoleAssistant Role = "assistant" // 助手/模型回复
	RoleTool      Role = "tool"      // 工具调用结果
)

// Message 聊天消息结构
// 兼容OpenAI Chat Completion API的消息格式
type Message struct {
	Role       Role       `json:"role"`                   // 消息角色
	Content    string     `json:"content,omitempty"`      // 消息内容
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // 工具调用请求（assistant消息）
	ToolCallID string     `json:"tool_call_id,omitempty"` // 工具调用ID（tool消息）
	Name       string     `json:"name,omitempty"`         // 工具名称（tool消息）
}

// ToolCall 工具调用请求
// 当模型决定调用工具时，会返回此结构
type ToolCall struct {
	ID       string       `json:"id"`       // 工具调用唯一标识
	Type     string       `json:"type"`     // 类型，通常为"function"
	Function FunctionCall `json:"function"` // 函数调用详情
}

// FunctionCall 函数调用详情
type FunctionCall struct {
	Name      string `json:"name"`      // 函数名称
	Arguments string `json:"arguments"` // 函数参数（JSON字符串）
}

// Tool 工具定义
// 用于向模型描述可用的工具
type Tool struct {
	Type     string      `json:"type"`     // 工具类型，通常为"function"
	Function FunctionDef `json:"function"` // 函数定义
}

// FunctionDef 函数定义
// 描述函数的名称、描述和参数schema
type FunctionDef struct {
	Name        string                 `json:"name"`                  // 函数名称
	Description string                 `json:"description,omitempty"` // 函数描述
	Parameters  map[string]interface{} `json:"parameters,omitempty"`  // 参数JSON Schema
}

// ChatCompletionRequest 聊天补全请求
// 兼容OpenAI Chat Completion API
type ChatCompletionRequest struct {
	Model       string    `json:"model"`                 // 模型名称
	Messages    []Message `json:"messages"`              // 消息历史
	Temperature float64   `json:"temperature,omitempty"` // 温度参数（0-2）
	MaxTokens   int       `json:"max_tokens,omitempty"`  // 最大生成token数
	Tools       []Tool    `json:"tools,omitempty"`       // 可用工具列表
	Stream      bool      `json:"stream,omitempty"`      // 是否启用流式响应
}

// ChatCompletionResponse 聊天补全响应
type ChatCompletionResponse struct {
	ID      string   `json:"id"`      // 响应ID
	Object  string   `json:"object"`  // 对象类型
	Created int64    `json:"created"` // 创建时间戳
	Model   string   `json:"model"`   // 使用的模型
	Choices []Choice `json:"choices"` // 响应选项
	Usage   Usage    `json:"usage"`   // token使用统计
}

// Choice 响应选项
type Choice struct {
	Index        int     `json:"index"`           // 选项索引
	Message      Message `json:"message"`         // 消息内容
	Delta        *Delta  `json:"delta,omitempty"` // 增量内容（流式响应）
	FinishReason string  `json:"finish_reason"`   // 结束原因
}

// Delta 流式响应增量内容
type Delta struct {
	Role      string     `json:"role,omitempty"`       // 角色
	Content   string     `json:"content,omitempty"`    // 内容片段
	ToolCalls []ToolCall `json:"tool_calls,omitempty"` // 工具调用
}

// Usage token使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`     // 输入token数
	CompletionTokens int `json:"completion_tokens"` // 输出token数
	TotalTokens      int `json:"total_tokens"`      // 总token数
}

// StreamResponse 流式响应
// 每个SSE事件的数据格式
type StreamResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

// ToJSON 将消息序列化为JSON字节
// 用于调试和日志记录
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// ParseToolCallArguments 解析工具调用参数
// 将Arguments JSON字符串解析为map
func (tc *ToolCall) ParseToolCallArguments() (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return nil, err
	}
	return args, nil
}

// NewSystemMessage 创建系统消息
func NewSystemMessage(content string) Message {
	return Message{
		Role:    RoleSystem,
		Content: content,
	}
}

// NewUserMessage 创建用户消息
func NewUserMessage(content string) Message {
	return Message{
		Role:    RoleUser,
		Content: content,
	}
}

// NewAssistantMessage 创建助手消息
func NewAssistantMessage(content string) Message {
	return Message{
		Role:    RoleAssistant,
		Content: content,
	}
}

// NewAssistantMessageWithToolCalls 创建带工具调用的助手消息
func NewAssistantMessageWithToolCalls(content string, toolCalls []ToolCall) Message {
	return Message{
		Role:      RoleAssistant,
		Content:   content,
		ToolCalls: toolCalls,
	}
}

// NewToolResultMessage 创建工具结果消息
func NewToolResultMessage(toolCallID, name, content string) Message {
	return Message{
		Role:       RoleTool,
		ToolCallID: toolCallID,
		Name:       name,
		Content:    content,
	}
}
