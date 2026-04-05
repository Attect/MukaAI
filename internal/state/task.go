package state

import (
	"time"
)

// TaskState 表示完整的任务状态文档
// 这是YAML状态文档的根结构，包含任务的所有状态信息
type TaskState struct {
	Task     TaskInfo     `yaml:"task"`
	Progress ProgressInfo `yaml:"progress"`
	Context  ContextInfo  `yaml:"context"`
	Agents   AgentsInfo   `yaml:"agents"`
}

// TaskInfo 包含任务的基本信息
type TaskInfo struct {
	ID        string    `yaml:"id"`         // 任务唯一标识符
	Goal      string    `yaml:"goal"`       // 任务目标描述
	Status    string    `yaml:"status"`     // 任务状态：pending, in_progress, completed, failed
	CreatedAt time.Time `yaml:"created_at"` // 创建时间
	UpdatedAt time.Time `yaml:"updated_at"` // 最后更新时间
}

// ProgressInfo 记录任务的进度信息
type ProgressInfo struct {
	CurrentPhase   string   `yaml:"current_phase"`   // 当前阶段名称
	CompletedSteps []string `yaml:"completed_steps"` // 已完成的步骤列表
	PendingSteps   []string `yaml:"pending_steps"`   // 待完成的步骤列表
}

// ContextInfo 存储任务的上下文信息
type ContextInfo struct {
	Decisions   []string   `yaml:"decisions"`           // 决策记录
	Constraints []string   `yaml:"constraints"`         // 约束条件
	Files       []FileInfo `yaml:"files,omitempty"`     // 相关文件信息（可选）
}

// FileInfo 描述任务相关的文件信息
type FileInfo struct {
	Path        string `yaml:"path"`        // 文件路径
	Description string `yaml:"description"` // 文件描述
	Status      string `yaml:"status"`      // 文件状态：created, modified, deleted
}

// AgentsInfo 管理Agent团队的状态
type AgentsInfo struct {
	Active  string        `yaml:"active"`           // 当前活动的Agent角色
	History []AgentRecord `yaml:"history"`          // Agent执行历史记录
}

// AgentRecord 记录单个Agent的执行历史
type AgentRecord struct {
	Role     string `yaml:"role"`     // Agent角色
	Summary  string `yaml:"summary"`  // 执行摘要
	Duration string `yaml:"duration"` // 执行时长
}

// NewTaskState 创建一个新的任务状态实例
// 参数：
//   - id: 任务唯一标识符
//   - goal: 任务目标描述
// 返回：
//   - 初始化后的TaskState实例
func NewTaskState(id, goal string) *TaskState {
	now := time.Now()
	return &TaskState{
		Task: TaskInfo{
			ID:        id,
			Goal:      goal,
			Status:    "pending",
			CreatedAt: now,
			UpdatedAt: now,
		},
		Progress: ProgressInfo{
			CurrentPhase:   "initialization",
			CompletedSteps: []string{},
			PendingSteps:   []string{},
		},
		Context: ContextInfo{
			Decisions:   []string{},
			Constraints: []string{},
			Files:       []FileInfo{},
		},
		Agents: AgentsInfo{
			Active:  "Orchestrator",
			History: []AgentRecord{},
		},
	}
}

// IsCompleted 检查任务是否已完成
func (ts *TaskState) IsCompleted() bool {
	return ts.Task.Status == "completed"
}

// IsInProgress 检查任务是否正在进行中
func (ts *TaskState) IsInProgress() bool {
	return ts.Task.Status == "in_progress"
}

// IsFailed 检查任务是否失败
func (ts *TaskState) IsFailed() bool {
	return ts.Task.Status == "failed"
}

// UpdateStatus 更新任务状态并更新时间戳
func (ts *TaskState) UpdateStatus(status string) {
	ts.Task.Status = status
	ts.Task.UpdatedAt = time.Now()
}

// AddCompletedStep 添加已完成的步骤
func (ts *TaskState) AddCompletedStep(step string) {
	ts.Progress.CompletedSteps = append(ts.Progress.CompletedSteps, step)
	ts.Task.UpdatedAt = time.Now()
}

// RemovePendingStep 从待办步骤中移除指定步骤
func (ts *TaskState) RemovePendingStep(step string) {
	for i, s := range ts.Progress.PendingSteps {
		if s == step {
			ts.Progress.PendingSteps = append(
				ts.Progress.PendingSteps[:i],
				ts.Progress.PendingSteps[i+1:]...,
			)
			break
		}
	}
	ts.Task.UpdatedAt = time.Now()
}

// AddDecision 添加决策记录
func (ts *TaskState) AddDecision(decision string) {
	ts.Context.Decisions = append(ts.Context.Decisions, decision)
	ts.Task.UpdatedAt = time.Now()
}

// AddConstraint 添加约束条件
func (ts *TaskState) AddConstraint(constraint string) {
	ts.Context.Constraints = append(ts.Context.Constraints, constraint)
	ts.Task.UpdatedAt = time.Now()
}

// AddFile 添加相关文件信息
func (ts *TaskState) AddFile(path, description, status string) {
	ts.Context.Files = append(ts.Context.Files, FileInfo{
		Path:        path,
		Description: description,
		Status:      status,
	})
	ts.Task.UpdatedAt = time.Now()
}

// AddAgentRecord 添加Agent执行记录
func (ts *TaskState) AddAgentRecord(role, summary, duration string) {
	ts.Agents.History = append(ts.Agents.History, AgentRecord{
		Role:     role,
		Summary:  summary,
		Duration: duration,
	})
	ts.Task.UpdatedAt = time.Now()
}

// SetActiveAgent 设置当前活动的Agent
func (ts *TaskState) SetActiveAgent(role string) {
	ts.Agents.Active = role
	ts.Task.UpdatedAt = time.Now()
}

// SetCurrentPhase 设置当前阶段
func (ts *TaskState) SetCurrentPhase(phase string) {
	ts.Progress.CurrentPhase = phase
	ts.Task.UpdatedAt = time.Now()
}
