package model

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client OpenAI API兼容客户端
// 实现与OpenAI Chat Completion API的通信
type Client struct {
	config     *Config
	httpClient *http.Client
}

// NewClient 创建新的模型客户端
// config: 模型配置，如果为nil则使用默认配置
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 300 * time.Minute, // 长时间推理的超时设置，适配慢速本地模型
		},
	}, nil
} 

// ChatCompletion 发送聊天补全请求
// ctx: 上下文，用于取消请求
// messages: 消息历史
// tools: 可用工具列表（可选）
func (c *Client) ChatCompletion(ctx context.Context, messages []Message, tools []Tool) (*ChatCompletionResponse, error) {
	req := ChatCompletionRequest{
		Model:    c.config.ModelName,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
	}

	return c.doChatCompletion(ctx, &req)
}

// ChatCompletionWithTemperature 发送带温度参数的聊天补全请求
func (c *Client) ChatCompletionWithTemperature(ctx context.Context, messages []Message, tools []Tool, temperature float64) (*ChatCompletionResponse, error) {
	req := ChatCompletionRequest{
		Model:       c.config.ModelName,
		Messages:    messages,
		Tools:       tools,
		Temperature: temperature,
		Stream:      false,
	}

	return c.doChatCompletion(ctx, &req)
}

// doChatCompletion 执行聊天补全请求
func (c *Client) doChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	// 序列化请求
	body, err := json.Marshal(req)
	if err != nil {
		return nil, &RequestError{Operation: "marshal", Err: err}
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.Endpoint+"chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, &RequestError{Operation: "create_request", Err: err}
	}

	// 设置请求头
	c.setHeaders(httpReq)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, &RequestError{Operation: "do_request", Err: err}
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	// 解析响应
	var result ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, &RequestError{Operation: "decode_response", Err: err}
	}

	// 处理思考标签
	c.processThinkingTags(&result)

	return &result, nil
}

// StreamChatCompletion 流式聊天补全
// 返回一个通道，每次收到流式数据时发送到通道
func (c *Client) StreamChatCompletion(ctx context.Context, messages []Message, tools []Tool) (<-chan StreamEvent, error) {
	req := ChatCompletionRequest{
		Model:    c.config.ModelName,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	}

	return c.doStreamChatCompletion(ctx, &req)
}

// StreamEvent 流式事件
type StreamEvent struct {
	Response *StreamResponse // 响应数据
	Error    error           // 错误（如果有）
	Done     bool            // 是否完成
}

// doStreamChatCompletion 执行流式聊天补全请求
func (c *Client) doStreamChatCompletion(ctx context.Context, req *ChatCompletionRequest) (<-chan StreamEvent, error) {
	// 序列化请求
	body, err := json.Marshal(req)
	if err != nil {
		return nil, &RequestError{Operation: "marshal", Err: err}
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.Endpoint+"chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, &RequestError{Operation: "create_request", Err: err}
	}

	// 设置请求头
	c.setHeaders(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, &RequestError{Operation: "do_request", Err: err}
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	// 创建事件通道
	eventChan := make(chan StreamEvent, 100)

	// 启动goroutine处理SSE流
	go c.processSSEStream(resp, eventChan)

	return eventChan, nil
}

// processSSEStream 处理SSE流
// 解析Server-Sent Events并发送到通道
func (c *Client) processSSEStream(resp *http.Response, eventChan chan<- StreamEvent) {
	defer close(eventChan)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	var buffer bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行
		if line == "" {
			continue
		}

		// 处理SSE数据行
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// 检查是否为结束标记
			if data == "[DONE]" {
				eventChan <- StreamEvent{Done: true}
				return
			}

			// 解析JSON数据
			var streamResp StreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				eventChan <- StreamEvent{Error: &RequestError{Operation: "parse_sse", Err: err}}
				continue
			}

			// 处理思考标签
			c.processStreamThinkingTags(&streamResp)

			// 发送事件
			eventChan <- StreamEvent{Response: &streamResp}
		} else {
			// 累积非data行（某些实现可能需要）
			buffer.WriteString(line)
			buffer.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		eventChan <- StreamEvent{Error: &RequestError{Operation: "read_stream", Err: err}}
	}
}

// setHeaders 设置HTTP请求头
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
}

// processThinkingTags 处理思考标签
// 某些模型（如Qwen）会在输出中包含<thinking>标签
// 此函数提取思考内容并清理标签
func (c *Client) processThinkingTags(resp *ChatCompletionResponse) {
	for i := range resp.Choices {
		content := resp.Choices[i].Message.Content
		resp.Choices[i].Message.Content = c.extractAndCleanThinking(content)
	}
}

// processStreamThinkingTags 处理流式响应中的思考标签
func (c *Client) processStreamThinkingTags(resp *StreamResponse) {
	for i := range resp.Choices {
		if resp.Choices[i].Delta != nil {
			content := resp.Choices[i].Delta.Content
			resp.Choices[i].Delta.Content = c.extractAndCleanThinking(content)
		}
	}
}

// extractAndCleanThinking 提取并清理思考标签
// 返回清理后的内容
// 注意：此函数保留思考内容但移除标签，如需完全移除思考内容请修改此函数
func (c *Client) extractAndCleanThinking(content string) string {
	// 移除<thinking>和</thinking>标签，保留内容
	// 如果需要完全移除思考内容，可以使用正则表达式替换
	content = strings.ReplaceAll(content, "<thinking>", "")
	content = strings.ReplaceAll(content, "</thinking>", "")
	return content
}

// GetConfig 获取客户端配置
func (c *Client) GetConfig() *Config {
	return c.config
}

// CountTokens 估算消息的token数量
// 这是一个粗略估算，实际token数需要使用tokenizer
func (c *Client) CountTokens(messages []Message) int {
	// 简单估算：平均每4个字符约1个token
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content)
		for _, tc := range msg.ToolCalls {
			totalChars += len(tc.Function.Name) + len(tc.Function.Arguments)
		}
	}
	return totalChars / 4
}

// IsContextOverflow 检查是否超出上下文限制
func (c *Client) IsContextOverflow(messages []Message) bool {
	return c.CountTokens(messages) > c.config.ContextSize
}

// RequestError 请求错误
type RequestError struct {
	Operation string // 操作名称
	Err       error  // 原始错误
}

// Error 实现error接口
func (e *RequestError) Error() string {
	return fmt.Sprintf("请求错误 [%s]: %v", e.Operation, e.Err)
}

// Unwrap 实现errors.Unwrap
func (e *RequestError) Unwrap() error {
	return e.Err
}

// APIError API错误
type APIError struct {
	StatusCode int    // HTTP状态码
	Message    string // 错误消息
}

// Error 实现error接口
func (e *APIError) Error() string {
	return fmt.Sprintf("API错误 [状态码: %d]: %s", e.StatusCode, e.Message)
}
