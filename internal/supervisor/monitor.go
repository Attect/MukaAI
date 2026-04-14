// Package supervisor 实现Agent监督系统
// 提供实时监督、质量检查、干预机制等功能
package supervisor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Attect/MukaAI/internal/agent"
	"github.com/Attect/MukaAI/internal/model"
	"github.com/Attect/MukaAI/internal/state"
	"github.com/Attect/MukaAI/internal/tools"
)

// InterventionType 干预类型
type InterventionType string

const (
	// InterventionWarning 警告：记录问题但不中断
	InterventionWarning InterventionType = "warning"
	// InterventionPause 暂停：暂停当前Agent，等待人工确认
	InterventionPause InterventionType = "pause"
	// InterventionInterrupt 中断：中断当前操作，注入修正指令
	InterventionInterrupt InterventionType = "interrupt"
	// InterventionRollback 回滚：回滚到上一个稳定状态
	InterventionRollback InterventionType = "rollback"
)

// IssueType 监督问题类型
type IssueType string

const (
	// IssueTypeQuality 输出质量问题
	IssueTypeQuality IssueType = "quality"
	// IssueTypeProgress 任务进度问题
	IssueTypeProgress IssueType = "progress"
	// IssueTypeError 错误检测
	IssueTypeError IssueType = "error"
	// IssueTypeSecurity 安全问题
	IssueTypeSecurity IssueType = "security"
	// IssueTypeBehavior 行为异常
	IssueTypeBehavior IssueType = "behavior"
	// IssueTypeResource 资源使用问题
	IssueTypeResource IssueType = "resource"
)

// SupervisionIssue 监督发现的问题
type SupervisionIssue struct {
	// Type 问题类型
	Type IssueType `json:"type"`
	// Severity 严重程度：low, medium, high, critical
	Severity string `json:"severity"`
	// Description 问题描述
	Description string `json:"description"`
	// Evidence 证据/示例
	Evidence string `json:"evidence"`
	// Suggestion 修正建议
	Suggestion string `json:"suggestion"`
	// Timestamp 发现时间
	Timestamp time.Time `json:"timestamp"`
	// Context 上下文信息
	Context map[string]interface{} `json:"context,omitempty"`
}

// InterventionRecord 干预记录
type InterventionRecord struct {
	// ID 干预记录ID
	ID string `json:"id"`
	// Type 干预类型
	Type InterventionType `json:"type"`
	// Issue 相关问题
	Issue SupervisionIssue `json:"issue"`
	// Timestamp 干预时间
	Timestamp time.Time `json:"timestamp"`
	// Action 采取的行动
	Action string `json:"action"`
	// Result 干预结果
	Result string `json:"result"`
	// TaskID 相关任务ID
	TaskID string `json:"task_id"`
	// AgentRole 相关Agent角色
	AgentRole string `json:"agent_role"`
}

// SupervisionResult 监督结果
type SupervisionResult struct {
	// Status 监督状态：pass, warning, intervention
	Status string `json:"status"`
	// Issues 发现的问题列表
	Issues []SupervisionIssue `json:"issues"`
	// Timestamp 监督时间
	Timestamp time.Time `json:"timestamp"`
	// Summary 监督摘要
	Summary string `json:"summary"`
	// Intervention 需要的干预（如果有）
	Intervention *InterventionRecord `json:"intervention,omitempty"`
}

// IsInterventionNeeded 检查是否需要干预
func (r *SupervisionResult) IsInterventionNeeded() bool {
	return r.Intervention != nil
}

// GetCriticalIssues 获取严重问题
func (r *SupervisionResult) GetCriticalIssues() []SupervisionIssue {
	var critical []SupervisionIssue
	for _, issue := range r.Issues {
		if issue.Severity == "critical" || issue.Severity == "high" {
			critical = append(critical, issue)
		}
	}
	return critical
}

// SupervisionStats 监督统计
type SupervisionStats struct {
	// TotalChecks 总检查次数
	TotalChecks int `json:"total_checks"`
	// IssuesFound 发现的问题总数
	IssuesFound int `json:"issues_found"`
	// Interventions 干预次数
	Interventions int `json:"interventions"`
	// WarningsIssued 发出的警告次数
	WarningsIssued int `json:"warnings_issued"`
	// PausesTriggered 触发的暂停次数
	PausesTriggered int `json:"pauses_triggered"`
	// InterruptsTriggered 触发的中断次数
	InterruptsTriggered int `json:"interrupts_triggered"`
	// RollbacksTriggered 触发的回滚次数
	RollbacksTriggered int `json:"rollbacks_triggered"`
	// LastCheckTime 最后检查时间
	LastCheckTime time.Time `json:"last_check_time"`
	// StartTime 监督开始时间
	StartTime time.Time `json:"start_time"`
}

// SupervisorConfig 监督器配置
type SupervisorConfig struct {
	// EnableQualityCheck 启用输出质量检查
	EnableQualityCheck bool `json:"enable_quality_check"`
	// EnableProgressCheck 启用任务进度检查
	EnableProgressCheck bool `json:"enable_progress_check"`
	// EnableErrorDetection 启用错误检测
	EnableErrorDetection bool `json:"enable_error_detection"`
	// EnableSecurityCheck 启用安全检查
	EnableSecurityCheck bool `json:"enable_security_check"`
	// EnableBehaviorCheck 启用行为检查
	EnableBehaviorCheck bool `json:"enable_behavior_check"`
	// EnableResourceCheck 启用资源检查
	EnableResourceCheck bool `json:"enable_resource_check"`
	// MonitorInterval 监督频率（秒）
	MonitorInterval int `json:"monitor_interval"`
	// MaxWarnings 最大警告次数（超过后升级为干预）
	MaxWarnings int `json:"max_warnings"`
	// AutoIntervene 自动干预（否则仅记录）
	AutoIntervene bool `json:"auto_intervene"`
	// MaxConsecutiveErrors 最大连续错误次数
	MaxConsecutiveErrors int `json:"max_consecutive_errors"`
	// QualityThreshold 质量阈值（0-100）
	QualityThreshold int `json:"quality_threshold"`
	// ProgressTimeout 进度超时时间（秒）
	ProgressTimeout int `json:"progress_timeout"`
	// EnableParallelMonitor 启用并行监督
	EnableParallelMonitor bool `json:"enable_parallel_monitor"`
	// MaxParallelChecks 最大并行检查数
	MaxParallelChecks int `json:"max_parallel_checks"`
}

// DefaultSupervisorConfig 返回默认监督器配置
func DefaultSupervisorConfig() *SupervisorConfig {
	return &SupervisorConfig{
		EnableQualityCheck:    true,
		EnableProgressCheck:   true,
		EnableErrorDetection:  true,
		EnableSecurityCheck:   false, // 默认关闭，需要明确启用
		EnableBehaviorCheck:   true,
		EnableResourceCheck:   false,
		MonitorInterval:       5,
		MaxWarnings:           5,
		AutoIntervene:         true,
		MaxConsecutiveErrors:  5,
		QualityThreshold:      40,
		ProgressTimeout:       1800, // 30分钟
		EnableParallelMonitor: true,
		MaxParallelChecks:     5,
	}
}

// AgentOutput Agent输出
type AgentOutput struct {
	// Content 输出内容
	Content string `json:"content"`
	// ToolCalls 工具调用列表
	ToolCalls []model.ToolCall `json:"tool_calls"`
	// Timestamp 输出时间
	Timestamp time.Time `json:"timestamp"`
	// TaskID 任务ID
	TaskID string `json:"task_id"`
	// AgentRole Agent角色
	AgentRole string `json:"agent_role"`
	// Iteration 迭代次数
	Iteration int `json:"iteration"`
	// Success 是否成功
	Success bool `json:"success"`
	// Error 错误信息（如果有）
	Error string `json:"error,omitempty"`
}

// Supervisor 监督器
// 负责监督Agent的行为、检查输出质量、执行干预
type Supervisor struct {
	// 核心组件
	modelClient  *model.Client       // 模型客户端
	toolRegistry *tools.ToolRegistry // 工具注册中心
	stateManager *state.StateManager // 状态管理器
	reviewer     *agent.Reviewer     // 程序逻辑审查器
	config       *SupervisorConfig   // 配置

	// 状态跟踪
	mu                sync.RWMutex
	interventionLog   []InterventionRecord // 干预历史
	statistics        SupervisionStats     // 统计信息
	warningCount      int                  // 当前警告计数
	consecutiveErrors int                  // 连续错误计数
	lastProgressTime  time.Time            // 最后进度更新时间
	lastStableState   *state.TaskState     // 最后稳定状态（用于回滚）

	// 回调函数
	onIntervention func(record InterventionRecord) // 干预回调
	onWarning      func(issue SupervisionIssue)    // 警告回调
	onIssueFound   func(issue SupervisionIssue)    // 问题发现回调

	// 控制通道
	pauseChan  chan struct{} // 暂停通道
	resumeChan chan struct{} // 恢复通道
	stopChan   chan struct{} // 停止通道
}

// NewSupervisor 创建新的监督器
func NewSupervisor(
	modelClient *model.Client,
	toolRegistry *tools.ToolRegistry,
	stateManager *state.StateManager,
	reviewer *agent.Reviewer,
	config *SupervisorConfig,
) (*Supervisor, error) {
	if modelClient == nil {
		return nil, fmt.Errorf("model client cannot be nil")
	}
	if toolRegistry == nil {
		return nil, fmt.Errorf("tool registry cannot be nil")
	}
	if stateManager == nil {
		return nil, fmt.Errorf("state manager cannot be nil")
	}
	if reviewer == nil {
		return nil, fmt.Errorf("reviewer cannot be nil")
	}

	if config == nil {
		config = DefaultSupervisorConfig()
	}

	return &Supervisor{
		modelClient:     modelClient,
		toolRegistry:    toolRegistry,
		stateManager:    stateManager,
		reviewer:        reviewer,
		config:          config,
		interventionLog: make([]InterventionRecord, 0),
		statistics: SupervisionStats{
			StartTime: time.Now(),
		},
		pauseChan:  make(chan struct{}),
		resumeChan: make(chan struct{}),
		stopChan:   make(chan struct{}),
	}, nil
}

// Monitor 监督Agent输出
// ctx: 上下文
// agentOutput: Agent的输出
// taskState: 当前任务状态
// 返回监督结果
func (s *Supervisor) Monitor(
	ctx context.Context,
	agentOutput *AgentOutput,
	taskState *state.TaskState,
) *SupervisionResult {
	result := &SupervisionResult{
		Status:    "pass",
		Issues:    make([]SupervisionIssue, 0),
		Timestamp: time.Now(),
	}

	// 更新统计
	s.mu.Lock()
	s.statistics.TotalChecks++
	s.statistics.LastCheckTime = result.Timestamp
	s.mu.Unlock()

	// 1. 输出质量检查
	if s.config.EnableQualityCheck {
		if issues := s.checkQuality(agentOutput, taskState); len(issues) > 0 {
			result.Issues = append(result.Issues, issues...)
		}
	}

	// 2. 任务进度检查
	if s.config.EnableProgressCheck {
		if issues := s.checkProgress(agentOutput, taskState); len(issues) > 0 {
			result.Issues = append(result.Issues, issues...)
		}
	}

	// 3. 错误检测
	if s.config.EnableErrorDetection {
		if issues := s.checkErrors(agentOutput, taskState); len(issues) > 0 {
			result.Issues = append(result.Issues, issues...)
		}
	}

	// 4. 安全检查
	if s.config.EnableSecurityCheck {
		if issues := s.checkSecurity(agentOutput, taskState); len(issues) > 0 {
			result.Issues = append(result.Issues, issues...)
		}
	}

	// 5. 行为检查
	if s.config.EnableBehaviorCheck {
		if issues := s.checkBehavior(agentOutput, taskState); len(issues) > 0 {
			result.Issues = append(result.Issues, issues...)
		}
	}

	// 6. 资源检查
	if s.config.EnableResourceCheck {
		if issues := s.checkResource(agentOutput, taskState); len(issues) > 0 {
			result.Issues = append(result.Issues, issues...)
		}
	}

	// 更新问题统计
	if len(result.Issues) > 0 {
		s.mu.Lock()
		s.statistics.IssuesFound += len(result.Issues)
		s.mu.Unlock()
	}

	// 确定监督状态和干预
	result.Status = s.determineStatus(result.Issues)
	result.Summary = s.generateSummary(result)
	result.Intervention = s.determineIntervention(result, agentOutput)

	// 触发回调
	s.triggerCallbacks(result)

	return result
}

// ParallelMonitor 并行监督多个Agent输出
// ctx: 上下文
// outputs: Agent输出通道
// 返回监督结果通道
func (s *Supervisor) ParallelMonitor(
	ctx context.Context,
	outputs <-chan *AgentOutput,
) <-chan *SupervisionResult {
	resultChan := make(chan *SupervisionResult, 10)

	if !s.config.EnableParallelMonitor {
		// 串行处理
		go func() {
			defer close(resultChan)
			for output := range outputs {
				taskState, _ := s.stateManager.GetState(output.TaskID)
				result := s.Monitor(ctx, output, taskState)
				resultChan <- result
			}
		}()
		return resultChan
	}

	// 并行处理
	go func() {
		defer close(resultChan)

		// 使用工作池模式
		workerCount := s.config.MaxParallelChecks
		if workerCount <= 0 {
			workerCount = 5
		}

		var wg sync.WaitGroup

		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for output := range outputs {
					select {
					case <-ctx.Done():
						return
					default:
						taskState, _ := s.stateManager.GetState(output.TaskID)
						result := s.Monitor(ctx, output, taskState)
						select {
						case resultChan <- result:
						case <-ctx.Done():
							return
						}
					}
				}
			}()
		}

		wg.Wait()
	}()

	return resultChan
}

// Intervene 执行干预
// ctx: 上下文
// issue: 发现的问题
// 返回干预记录
func (s *Supervisor) Intervene(
	ctx context.Context,
	issue SupervisionIssue,
) *InterventionRecord {
	// 确定干预类型
	interventionType := s.determineInterventionType(issue)

	record := &InterventionRecord{
		ID:        fmt.Sprintf("intervention-%d", time.Now().UnixNano()),
		Type:      interventionType,
		Issue:     issue,
		Timestamp: time.Now(),
		Action:    s.generateInterventionAction(interventionType, issue),
	}

	// 更新统计
	s.mu.Lock()
	s.statistics.Interventions++
	switch interventionType {
	case InterventionWarning:
		s.statistics.WarningsIssued++
		s.warningCount++
	case InterventionPause:
		s.statistics.PausesTriggered++
	case InterventionInterrupt:
		s.statistics.InterruptsTriggered++
	case InterventionRollback:
		s.statistics.RollbacksTriggered++
	}
	s.interventionLog = append(s.interventionLog, *record)
	s.mu.Unlock()

	// 执行干预动作
	record.Result = s.executeIntervention(ctx, interventionType, issue)

	// 触发干预回调
	if s.onIntervention != nil {
		s.onIntervention(*record)
	}

	return record
}

// checkQuality 检查输出质量
func (s *Supervisor) checkQuality(
	output *AgentOutput,
	taskState *state.TaskState,
) []SupervisionIssue {
	issues := make([]SupervisionIssue, 0)

	// 检查输出是否为空
	if output.Content == "" && len(output.ToolCalls) == 0 {
		issues = append(issues, SupervisionIssue{
			Type:        IssueTypeQuality,
			Severity:    "high",
			Description: "Agent输出为空，没有任何内容或工具调用",
			Evidence:    "输出内容为空",
			Suggestion:  "检查Agent是否正常工作，或是否存在阻塞",
			Timestamp:   time.Now(),
		})
		return issues
	}

	// 检查输出长度是否合理（仅在无工具调用时检查，有工具调用时文本短是正常的）
	if len(output.Content) > 0 && len(output.Content) < 10 && len(output.ToolCalls) == 0 {
		issues = append(issues, SupervisionIssue{
			Type:        IssueTypeQuality,
			Severity:    "low",
			Description: "Agent输出内容过短且无工具调用，可能不完整",
			Evidence:    fmt.Sprintf("输出长度: %d 字符", len(output.Content)),
			Suggestion:  "检查是否需要更详细的输出",
			Timestamp:   time.Now(),
		})
	}

	// 使用审查器检查输出质量
	if taskState != nil {
		reviewResult := s.reviewer.ReviewOutput(output.Content, output.ToolCalls, taskState)
		if reviewResult.HasWarnings() || reviewResult.IsBlocked() {
			for _, reviewIssue := range reviewResult.Issues {
				issues = append(issues, SupervisionIssue{
					Type:        IssueTypeQuality,
					Severity:    reviewIssue.Severity,
					Description: reviewIssue.Description,
					Evidence:    reviewIssue.Evidence,
					Suggestion:  reviewIssue.Suggestion,
					Timestamp:   reviewIssue.Timestamp,
					Context: map[string]interface{}{
						"review_type": reviewIssue.Type,
					},
				})
			}
		}
	}

	return issues
}

// checkProgress 检查任务进度
func (s *Supervisor) checkProgress(
	output *AgentOutput,
	taskState *state.TaskState,
) []SupervisionIssue {
	issues := make([]SupervisionIssue, 0)

	if taskState == nil {
		return issues
	}

	// 检查任务是否超时
	if taskState.Task.Status == "in_progress" {
		duration := time.Since(taskState.Task.UpdatedAt)
		timeout := time.Duration(s.config.ProgressTimeout) * time.Second

		if duration > timeout {
			issues = append(issues, SupervisionIssue{
				Type:        IssueTypeProgress,
				Severity:    "high",
				Description: fmt.Sprintf("任务执行超时，已运行 %.0f 秒", duration.Seconds()),
				Evidence:    fmt.Sprintf("超时阈值: %d 秒", s.config.ProgressTimeout),
				Suggestion:  "考虑中断任务或调整执行策略",
				Timestamp:   time.Now(),
				Context: map[string]interface{}{
					"duration": duration.Seconds(),
					"timeout":  timeout.Seconds(),
				},
			})
		}
	}

	// 检查进度是否停滞
	s.mu.Lock()
	lastProgress := s.lastProgressTime
	s.mu.Unlock()

	if !lastProgress.IsZero() {
		stagnantTime := time.Since(lastProgress)
		if stagnantTime > time.Duration(float64(s.config.ProgressTimeout)*0.75)*time.Second {
			issues = append(issues, SupervisionIssue{
				Type:        IssueTypeProgress,
				Severity:    "low",
				Description: fmt.Sprintf("任务进度停滞 %.0f 秒", stagnantTime.Seconds()),
				Evidence:    "没有新的完成步骤",
				Suggestion:  "检查是否有阻塞或需要调整策略",
				Timestamp:   time.Now(),
			})
		}
	}

	// 更新进度时间
	if taskState != nil && len(taskState.Progress.CompletedSteps) > 0 {
		s.mu.Lock()
		s.lastProgressTime = time.Now()
		s.mu.Unlock()
	}

	return issues
}

// checkErrors 检查错误
func (s *Supervisor) checkErrors(
	output *AgentOutput,
	taskState *state.TaskState,
) []SupervisionIssue {
	issues := make([]SupervisionIssue, 0)

	// 检查输出中的错误标记
	if output.Error != "" {
		// 区分系统错误和工具业务错误
		isSystemError := !isToolBusinessError(output.Error)

		severity := "medium"
		if isSystemError {
			severity = "high"
		}

		issues = append(issues, SupervisionIssue{
			Type:        IssueTypeError,
			Severity:    severity,
			Description: "Agent执行过程中发生错误",
			Evidence:    output.Error,
			Suggestion:  "检查错误原因并尝试修复",
			Timestamp:   time.Now(),
		})

		// 只有系统错误才递增连续错误计数
		if isSystemError {
			s.mu.Lock()
			s.consecutiveErrors++
			count := s.consecutiveErrors
			s.mu.Unlock()

			if count >= s.config.MaxConsecutiveErrors {
				issues = append(issues, SupervisionIssue{
					Type:        IssueTypeError,
					Severity:    "critical",
					Description: fmt.Sprintf("连续系统错误次数达到 %d 次", count),
					Evidence:    "系统可能存在严重问题",
					Suggestion:  "建议暂停任务并进行人工检查",
					Timestamp:   time.Now(),
				})
			}
		}
	} else {
		// 成功时重置错误计数
		s.mu.Lock()
		s.consecutiveErrors = 0
		s.mu.Unlock()
	}

	// 注意：工具调用的结果检查应在工具执行之后进行（由Agent主循环中的ReviewToolResult处理）
	// 此处不应在工具执行前就以success=false调用ReviewToolResult，
	// 否则每次迭代都会误报"连续失败"（Bug已修复）

	return issues
}

// checkSecurity 检查安全问题
func (s *Supervisor) checkSecurity(
	output *AgentOutput,
	taskState *state.TaskState,
) []SupervisionIssue {
	issues := make([]SupervisionIssue, 0)

	// 检查危险命令
	dangerousCommands := []string{
		"rm -rf /",
		"dd if=/dev/zero",
		"mkfs",
		":(){ :|:& };:",
		"chmod 777",
		"chown root",
	}

	for _, tc := range output.ToolCalls {
		if tc.Function.Name == "execute_command" || tc.Function.Name == "shell_execute" {
			for _, dangerous := range dangerousCommands {
				if contains(tc.Function.Arguments, dangerous) {
					issues = append(issues, SupervisionIssue{
						Type:        IssueTypeSecurity,
						Severity:    "critical",
						Description: "检测到危险命令",
						Evidence:    fmt.Sprintf("命令包含: %s", dangerous),
						Suggestion:  "阻止执行并检查Agent行为",
						Timestamp:   time.Now(),
						Context: map[string]interface{}{
							"tool_name": tc.Function.Name,
							"arguments": tc.Function.Arguments,
						},
					})
				}
			}
		}
	}

	// 检查敏感文件访问
	sensitivePaths := []string{
		"/etc/passwd",
		"/etc/shadow",
		"/root/.ssh",
		".env",
		"credentials",
	}

	for _, tc := range output.ToolCalls {
		if tc.Function.Name == "read_file" {
			for _, sensitive := range sensitivePaths {
				if contains(tc.Function.Arguments, sensitive) {
					issues = append(issues, SupervisionIssue{
						Type:        IssueTypeSecurity,
						Severity:    "high",
						Description: "尝试访问敏感文件",
						Evidence:    fmt.Sprintf("路径: %s", sensitive),
						Suggestion:  "确认是否有必要访问此文件",
						Timestamp:   time.Now(),
						Context: map[string]interface{}{
							"tool_name": tc.Function.Name,
							"file_path": sensitive,
						},
					})
				}
			}
		}
	}

	return issues
}

// checkBehavior 检查行为异常
func (s *Supervisor) checkBehavior(
	output *AgentOutput,
	taskState *state.TaskState,
) []SupervisionIssue {
	issues := make([]SupervisionIssue, 0)

	// 检查迭代次数是否过高
	if output.Iteration > 30 {
		issues = append(issues, SupervisionIssue{
			Type:        IssueTypeBehavior,
			Severity:    "medium",
			Description: fmt.Sprintf("迭代次数过高: %d", output.Iteration),
			Evidence:    "可能存在循环或效率问题",
			Suggestion:  "检查是否可以优化执行策略",
			Timestamp:   time.Now(),
		})
	}

	// 检查工具调用频率
	toolCallCount := len(output.ToolCalls)
	if toolCallCount > 10 {
		issues = append(issues, SupervisionIssue{
			Type:        IssueTypeBehavior,
			Severity:    "low",
			Description: fmt.Sprintf("单次输出工具调用过多: %d", toolCallCount),
			Evidence:    "可能影响执行效率",
			Suggestion:  "考虑合并或优化工具调用",
			Timestamp:   time.Now(),
		})
	}

	return issues
}

// checkResource 检查资源使用
func (s *Supervisor) checkResource(
	output *AgentOutput,
	taskState *state.TaskState,
) []SupervisionIssue {
	issues := make([]SupervisionIssue, 0)

	// 检查输出大小
	if len(output.Content) > 100000 { // 100KB
		issues = append(issues, SupervisionIssue{
			Type:        IssueTypeResource,
			Severity:    "medium",
			Description: fmt.Sprintf("输出内容过大: %d 字节", len(output.Content)),
			Evidence:    "可能消耗过多token",
			Suggestion:  "考虑压缩或分段输出",
			Timestamp:   time.Now(),
		})
	}

	return issues
}

// determineStatus 确定监督状态
func (s *Supervisor) determineStatus(issues []SupervisionIssue) string {
	if len(issues) == 0 {
		return "pass"
	}

	// 检查是否有严重问题
	for _, issue := range issues {
		if issue.Severity == "critical" {
			return "intervention"
		}
		if issue.Severity == "high" {
			return "warning"
		}
	}

	return "warning"
}

// determineIntervention 确定是否需要干预
func (s *Supervisor) determineIntervention(
	result *SupervisionResult,
	output *AgentOutput,
) *InterventionRecord {
	if result.Status == "pass" {
		return nil
	}

	// 检查是否需要干预
	criticalIssues := result.GetCriticalIssues()
	if len(criticalIssues) == 0 {
		// 检查警告次数
		s.mu.RLock()
		warningCount := s.warningCount
		s.mu.RUnlock()

		if warningCount < s.config.MaxWarnings {
			return nil
		}
	}

	// 需要干预
	if s.config.AutoIntervene {
		// 选择最严重的问题
		var mostSevere SupervisionIssue
		for _, issue := range criticalIssues {
			if mostSevere.Severity == "" || issue.Severity > mostSevere.Severity {
				mostSevere = issue
			}
		}
		if mostSevere.Severity == "" && len(result.Issues) > 0 {
			mostSevere = result.Issues[0]
		}

		return s.Intervene(context.Background(), mostSevere)
	}

	return nil
}

// determineInterventionType 确定干预类型
func (s *Supervisor) determineInterventionType(issue SupervisionIssue) InterventionType {
	switch issue.Severity {
	case "critical":
		// 严重问题：中断或回滚
		if issue.Type == IssueTypeSecurity {
			return InterventionInterrupt
		}
		return InterventionRollback
	case "high":
		// 高严重度但不紧急：仅警告
		// 只有在连续多次high后才升级为pause
		s.mu.RLock()
		warningCount := s.warningCount
		s.mu.RUnlock()
		if warningCount >= s.config.MaxWarnings {
			return InterventionPause
		}
		return InterventionWarning
	default:
		// 其他：警告
		return InterventionWarning
	}
}

// generateInterventionAction 生成干预动作描述
func (s *Supervisor) generateInterventionAction(
	interventionType InterventionType,
	issue SupervisionIssue,
) string {
	switch interventionType {
	case InterventionWarning:
		return fmt.Sprintf("发出警告: %s", issue.Description)
	case InterventionPause:
		return fmt.Sprintf("暂停执行，等待人工确认: %s", issue.Description)
	case InterventionInterrupt:
		return fmt.Sprintf("中断当前操作: %s", issue.Description)
	case InterventionRollback:
		return fmt.Sprintf("回滚到上一个稳定状态: %s", issue.Description)
	default:
		return "未知干预类型"
	}
}

// executeIntervention 执行干预
func (s *Supervisor) executeIntervention(
	ctx context.Context,
	interventionType InterventionType,
	issue SupervisionIssue,
) string {
	switch interventionType {
	case InterventionWarning:
		// 警告：仅记录
		return "警告已记录"

	case InterventionPause:
		// 暂停：发送暂停信号
		select {
		case s.pauseChan <- struct{}{}:
			// 等待恢复信号
			select {
			case <-s.resumeChan:
				return "已恢复执行"
			case <-ctx.Done():
				return "暂停被取消"
			}
		default:
			return "暂停信号已发送"
		}

	case InterventionInterrupt:
		// 中断：停止当前操作
		select {
		case s.stopChan <- struct{}{}:
			return "已中断当前操作"
		default:
			return "中断信号已发送"
		}

	case InterventionRollback:
		// 回滚：恢复到上一个稳定状态
		s.mu.RLock()
		lastStable := s.lastStableState
		s.mu.RUnlock()

		if lastStable != nil {
			// 恢复状态
			if err := s.stateManager.Save(lastStable.Task.ID); err != nil {
				return fmt.Sprintf("回滚失败: %v", err)
			}
			return fmt.Sprintf("已回滚到状态: %s", lastStable.Task.ID)
		}
		return "无可用稳定状态，无法回滚"

	default:
		return "未知干预类型"
	}
}

// generateSummary 生成监督摘要
func (s *Supervisor) generateSummary(result *SupervisionResult) string {
	if len(result.Issues) == 0 {
		return "监督通过，未发现问题"
	}

	summary := fmt.Sprintf("发现 %d 个问题: ", len(result.Issues))
	typeCounts := make(map[IssueType]int)
	for _, issue := range result.Issues {
		typeCounts[issue.Type]++
	}

	parts := make([]string, 0)
	for t, c := range typeCounts {
		parts = append(parts, fmt.Sprintf("%s(%d)", t, c))
	}
	summary += joinStrings(parts, ", ")

	if result.Intervention != nil {
		summary += fmt.Sprintf(" | 已执行干预: %s", result.Intervention.Type)
	}

	return summary
}

// triggerCallbacks 触发回调函数
func (s *Supervisor) triggerCallbacks(result *SupervisionResult) {
	// 触发问题发现回调
	if s.onIssueFound != nil {
		for _, issue := range result.Issues {
			s.onIssueFound(issue)
		}
	}

	// 触发警告回调
	if s.onWarning != nil && result.Status == "warning" {
		for _, issue := range result.Issues {
			if issue.Severity != "critical" {
				s.onWarning(issue)
			}
		}
	}
}

// SetOnIntervention 设置干预回调
func (s *Supervisor) SetOnIntervention(callback func(record InterventionRecord)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onIntervention = callback
}

// SetOnWarning 设置警告回调
func (s *Supervisor) SetOnWarning(callback func(issue SupervisionIssue)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onWarning = callback
}

// SetOnIssueFound 设置问题发现回调
func (s *Supervisor) SetOnIssueFound(callback func(issue SupervisionIssue)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onIssueFound = callback
}

// GetInterventionLog 获取干预历史
func (s *Supervisor) GetInterventionLog() []InterventionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log := make([]InterventionRecord, len(s.interventionLog))
	copy(log, s.interventionLog)
	return log
}

// GetStatistics 获取监督统计
func (s *Supervisor) GetStatistics() SupervisionStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statistics
}

// Reset 重置监督器状态
func (s *Supervisor) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.interventionLog = make([]InterventionRecord, 0)
	s.statistics = SupervisionStats{
		StartTime: time.Now(),
	}
	s.warningCount = 0
	s.consecutiveErrors = 0
	s.lastProgressTime = time.Time{}
	s.lastStableState = nil
}

// SaveStableState 保存稳定状态（用于回滚）
func (s *Supervisor) SaveStableState(taskState *state.TaskState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 克隆状态
	if taskState != nil {
		stateCopy := *taskState
		s.lastStableState = &stateCopy
	}
}

// Resume 恢复执行（用于暂停后）
func (s *Supervisor) Resume() {
	select {
	case s.resumeChan <- struct{}{}:
	default:
	}
}

// Stop 停止监督
func (s *Supervisor) Stop() {
	select {
	case s.stopChan <- struct{}{}:
	default:
	}
}

// GetPauseChan 获取暂停通道
func (s *Supervisor) GetPauseChan() <-chan struct{} {
	return s.pauseChan
}

// GetStopChan 获取停止通道
func (s *Supervisor) GetStopChan() <-chan struct{} {
	return s.stopChan
}

// 辅助函数

// isToolBusinessError 判断是否为工具返回的业务错误（非系统错误）
// 工具返回业务错误的特征：包含 "success":false 或 "isError":true 等
// 这类错误是工具正常返回，只是业务结果为失败，不应计入系统连续错误
func isToolBusinessError(errorContent string) bool {
	businessErrorMarkers := []string{
		`"success":false`,
		`"success": false`,
		`"isError":true`,
		`"isError": true`,
	}
	for _, marker := range businessErrorMarkers {
		if strings.Contains(errorContent, marker) {
			return true
		}
	}
	return false
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

// containsMiddle 检查字符串中间是否包含子串
func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// joinStrings 连接字符串切片
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
