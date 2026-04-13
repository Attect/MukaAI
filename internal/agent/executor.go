package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Attect/MukaAI/internal/model"
	"github.com/Attect/MukaAI/internal/tools"
)

// ToolExecutor 执行工具调用
// 负责将模型的工具调用请求转换为实际执行并返回结果
type ToolExecutor struct {
	registry *tools.ToolRegistry
}

// NewToolExecutor 创建新的工具执行器
func NewToolExecutor(registry *tools.ToolRegistry) *ToolExecutor {
	return &ToolExecutor{
		registry: registry,
	}
}

// ExecuteToolCalls 执行工具调用列表
// 返回工具结果消息列表
// 并行执行多个工具调用以提高效率
func (e *ToolExecutor) ExecuteToolCalls(ctx context.Context, toolCalls []model.ToolCall) ([]model.Message, error) {
	if len(toolCalls) == 0 {
		return nil, nil
	}

	// 如果只有一个工具调用，直接执行
	if len(toolCalls) == 1 {
		result, err := e.executeToolCall(ctx, toolCalls[0])
		if err != nil {
			return nil, err
		}
		return []model.Message{result}, nil
	}

	// 并行执行多个工具调用
	results := make([]model.Message, len(toolCalls))
	errors := make([]error, len(toolCalls))
	var wg sync.WaitGroup

	for i, tc := range toolCalls {
		wg.Add(1)
		go func(index int, toolCall model.ToolCall) {
			defer wg.Done()
			result, err := e.executeToolCall(ctx, toolCall)
			results[index] = result
			errors[index] = err
		}(i, tc)
	}

	wg.Wait()

	// 检查是否有错误
	for _, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("tool execution failed: %w", err)
		}
	}

	return results, nil
}

// executeToolCall 执行单个工具调用
func (e *ToolExecutor) executeToolCall(ctx context.Context, tc model.ToolCall) (model.Message, error) {
	// 检查工具是否存在
	_, exists := e.registry.GetTool(tc.Function.Name)
	if !exists {
		// 工具不存在，返回错误结果
		return model.NewToolResultMessage(
			tc.ID,
			tc.Function.Name,
			fmt.Sprintf(`{"success":false,"error":"tool '%s' not found"}`, tc.Function.Name),
		), nil
	}

	// 执行工具
	result, err := e.registry.ExecuteTool(ctx, tc.Function.Name, tc.Function.Arguments)
	if err != nil {
		// 执行出错，返回错误结果
		return model.NewToolResultMessage(
			tc.ID,
			tc.Function.Name,
			fmt.Sprintf(`{"success":false,"error":"%s"}`, err.Error()),
		), nil
	}

	// 将结果转换为JSON
	resultJSON := result.ToJSON()

	// 返回工具结果消息
	return model.NewToolResultMessage(tc.ID, tc.Function.Name, resultJSON), nil
}

// ExecuteToolCallWithTimeout 执行工具调用（带超时）
func (e *ToolExecutor) ExecuteToolCallWithTimeout(ctx context.Context, tc model.ToolCall, timeout time.Duration) (model.Message, error) {
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 使用通道接收结果
	resultChan := make(chan model.Message, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := e.executeToolCall(timeoutCtx, tc)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return model.Message{}, err
	case <-timeoutCtx.Done():
		// 超时
		return model.NewToolResultMessage(
			tc.ID,
			tc.Function.Name,
			fmt.Sprintf(`{"success":false,"error":"tool execution timeout after %v"}`, timeout),
		), nil
	}
}

// ExecuteToolCallSequential 顺序执行工具调用
// 某些场景需要顺序执行（如后续工具依赖前一个工具的结果）
func (e *ToolExecutor) ExecuteToolCallSequential(ctx context.Context, toolCalls []model.ToolCall) ([]model.Message, error) {
	if len(toolCalls) == 0 {
		return nil, nil
	}

	results := make([]model.Message, 0, len(toolCalls))
	for _, tc := range toolCalls {
		result, err := e.executeToolCall(ctx, tc)
		if err != nil {
			return nil, fmt.Errorf("tool '%s' execution failed: %w", tc.Function.Name, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// ToolExecutionResult 工具执行结果（包含详细信息）
type ToolExecutionResult struct {
	ToolCallID string          // 工具调用ID
	ToolName   string          // 工具名称
	Arguments  string          // 原始参数
	Result     *tools.ToolResult // 执行结果
	Duration   time.Duration   // 执行时长
	Error      error           // 错误（如果有）
}

// ExecuteToolCallsWithDetails 执行工具调用并返回详细信息
func (e *ToolExecutor) ExecuteToolCallsWithDetails(ctx context.Context, toolCalls []model.ToolCall) ([]ToolExecutionResult, error) {
	if len(toolCalls) == 0 {
		return nil, nil
	}

	results := make([]ToolExecutionResult, len(toolCalls))
	var wg sync.WaitGroup

	for i, tc := range toolCalls {
		wg.Add(1)
		go func(index int, toolCall model.ToolCall) {
			defer wg.Done()

			start := time.Now()
			result := ToolExecutionResult{
				ToolCallID: toolCall.ID,
				ToolName:   toolCall.Function.Name,
				Arguments:  toolCall.Function.Arguments,
			}

			// 执行工具
			toolResult, err := e.registry.ExecuteTool(ctx, toolCall.Function.Name, toolCall.Function.Arguments)
			result.Duration = time.Since(start)
			result.Result = toolResult
			result.Error = err

			results[index] = result
		}(i, tc)
	}

	wg.Wait()

	return results, nil
}

// GetAvailableTools 获取可用工具列表
func (e *ToolExecutor) GetAvailableTools() []tools.Tool {
	return e.registry.GetAllTools()
}

// GetToolSchemas 获取工具Schema列表（用于发送给模型）
func (e *ToolExecutor) GetToolSchemas() []model.Tool {
	schemas := e.registry.GetAllToolSchemas()
	result := make([]model.Tool, len(schemas))

	for i, schema := range schemas {
		result[i] = model.Tool{
			Type: schema.Type,
			Function: model.FunctionDef{
				Name:        schema.Function.Name,
				Description: schema.Function.Description,
				Parameters:  schema.Function.Parameters,
			},
		}
	}

	return result
}

// HasTool 检查工具是否存在
func (e *ToolExecutor) HasTool(name string) bool {
	_, exists := e.registry.GetTool(name)
	return exists
}

// ParseToolCallArguments 解析工具调用参数
func ParseToolCallArguments(tc model.ToolCall) (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}
	return args, nil
}

// BuildToolResultMessage 构建工具结果消息
func BuildToolResultMessage(tc model.ToolCall, result *tools.ToolResult) model.Message {
	return model.NewToolResultMessage(tc.ID, tc.Function.Name, result.ToJSON())
}

// BuildToolErrorMessage 构建工具错误消息
func BuildToolErrorMessage(tc model.ToolCall, errMsg string) model.Message {
	return model.NewToolResultMessage(
		tc.ID,
		tc.Function.Name,
		fmt.Sprintf(`{"success":false,"error":"%s"}`, errMsg),
	)
}
