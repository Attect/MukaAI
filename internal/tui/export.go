// Package tui 提供基于 Bubble Tea 的终端用户界面
package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ConversationExport 导出的对话数据结构
type ConversationExport struct {
	// ID 对话唯一标识
	ID string `json:"id"`
	// Title 对话标题
	Title string `json:"title"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
	// Status 对话状态
	Status string `json:"status"`
	// TokenUsage token 用量
	TokenUsage int `json:"token_usage"`
	// IsSubConversation 是否为子对话
	IsSubConversation bool `json:"is_sub_conversation"`
	// ParentID 父对话 ID
	ParentID string `json:"parent_id,omitempty"`
	// AgentRole Agent 角色
	AgentRole string `json:"agent_role,omitempty"`
	// Messages 消息列表
	Messages []MessageExport `json:"messages"`
	// ExportedAt 导出时间
	ExportedAt time.Time `json:"exported_at"`
}

// MessageExport 导出的消息数据结构
type MessageExport struct {
	// Role 消息角色
	Role string `json:"role"`
	// Content 正文内容
	Content string `json:"content"`
	// Thinking 思考内容
	Thinking string `json:"thinking,omitempty"`
	// ToolCalls 工具调用列表
	ToolCalls []ToolCallExport `json:"tool_calls,omitempty"`
	// TokenUsage token 用量
	TokenUsage int `json:"token_usage"`
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
}

// ToolCallExport 导出的工具调用数据结构
type ToolCallExport struct {
	// ID 工具调用唯一标识
	ID string `json:"id"`
	// Name 工具名称
	Name string `json:"name"`
	// Arguments 工具参数（JSON 格式）
	Arguments string `json:"arguments"`
	// Result 工具执行结果
	Result string `json:"result,omitempty"`
	// ResultError 工具执行错误
	ResultError string `json:"result_error,omitempty"`
}

// ExportConversation 导出对话为 JSON 格式
func ExportConversation(conv *Conversation) (*ConversationExport, error) {
	if conv == nil {
		return nil, fmt.Errorf("对话为空")
	}

	// 转换状态
	var status string
	switch conv.Status {
	case ConvStatusActive:
		status = "active"
	case ConvStatusWaiting:
		status = "waiting"
	case ConvStatusFinished:
		status = "finished"
	default:
		status = "unknown"
	}

	// 转换消息
	messages := make([]MessageExport, 0, len(conv.Messages))
	for _, msg := range conv.Messages {
		// 转换角色
		var role string
		switch msg.Role {
		case MessageRoleUser:
			role = "user"
		case MessageRoleAssistant:
			role = "assistant"
		case MessageRoleTool:
			role = "tool"
		default:
			role = "unknown"
		}

		// 转换工具调用
		toolCalls := make([]ToolCallExport, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			toolCalls = append(toolCalls, ToolCallExport{
				ID:          tc.ID,
				Name:        tc.Name,
				Arguments:   tc.Arguments,
				Result:      tc.Result,
				ResultError: tc.ResultError,
			})
		}

		messages = append(messages, MessageExport{
			Role:       role,
			Content:    msg.Content,
			Thinking:   msg.Thinking,
			ToolCalls:  toolCalls,
			TokenUsage: msg.TokenUsage,
			Timestamp:  msg.Timestamp,
		})
	}

	return &ConversationExport{
		ID:                conv.ID,
		Title:             conv.Title,
		CreatedAt:         conv.CreatedAt,
		Status:            status,
		TokenUsage:        conv.TokenUsage,
		IsSubConversation: conv.IsSubConversation,
		ParentID:          conv.ParentID,
		AgentRole:         conv.AgentRole,
		Messages:          messages,
		ExportedAt:        time.Now(),
	}, nil
}

// SaveConversationToFile 保存对话到文件
// filePath: 文件路径（可选，如果为空则使用默认路径）
// conv: 要保存的对话
// 返回保存的文件路径和错误信息
func SaveConversationToFile(filePath string, conv *Conversation) (string, error) {
	// 导出对话
	export, err := ExportConversation(conv)
	if err != nil {
		return "", fmt.Errorf("导出对话失败: %w", err)
	}

	// 如果没有指定文件路径，使用默认路径
	if filePath == "" {
		// 使用对话 ID 和时间戳作为文件名
		timestamp := time.Now().Format("20060102_150405")
		fileName := fmt.Sprintf("conversation_%s_%s.json", timestamp, conv.ID[:8])
		filePath = fileName
	}

	// 确保文件扩展名为 .json
	if !strings.HasSuffix(strings.ToLower(filePath), ".json") {
		filePath = filePath + ".json"
	}

	// 序列化为 JSON
	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化对话失败: %w", err)
	}

	// 创建目录（如果需要）
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("创建目录失败: %w", err)
		}
	}

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	// 获取绝对路径
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	return absPath, nil
}

// LoadConversationFromFile 从文件加载对话
// filePath: 文件路径
// 返回加载的对话和错误信息
func LoadConversationFromFile(filePath string) (*Conversation, error) {
	// 读取文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 反序列化 JSON
	var export ConversationExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	// 转换为 Conversation
	conv := &Conversation{
		ID:                export.ID,
		Title:             export.Title,
		CreatedAt:         export.CreatedAt,
		TokenUsage:        export.TokenUsage,
		IsSubConversation: export.IsSubConversation,
		ParentID:          export.ParentID,
		AgentRole:         export.AgentRole,
		Messages:          make([]Message, 0, len(export.Messages)),
	}

	// 转换状态
	switch export.Status {
	case "active":
		conv.Status = ConvStatusActive
	case "waiting":
		conv.Status = ConvStatusWaiting
	case "finished":
		conv.Status = ConvStatusFinished
	default:
		conv.Status = ConvStatusActive
	}

	// 转换消息
	for _, msgExport := range export.Messages {
		// 转换角色
		var role MessageRole
		switch msgExport.Role {
		case "user":
			role = MessageRoleUser
		case "assistant":
			role = MessageRoleAssistant
		case "tool":
			role = MessageRoleTool
		default:
			role = MessageRoleUser
		}

		// 转换工具调用
		toolCalls := make([]ToolCall, 0, len(msgExport.ToolCalls))
		for _, tcExport := range msgExport.ToolCalls {
			toolCalls = append(toolCalls, ToolCall{
				ID:          tcExport.ID,
				Name:        tcExport.Name,
				Arguments:   tcExport.Arguments,
				Result:      tcExport.Result,
				ResultError: tcExport.ResultError,
			})
		}

		conv.Messages = append(conv.Messages, Message{
			Role:       role,
			Content:    msgExport.Content,
			Thinking:   msgExport.Thinking,
			ToolCalls:  toolCalls,
			TokenUsage: msgExport.TokenUsage,
			Timestamp:  msgExport.Timestamp,
		})
	}

	return conv, nil
}
