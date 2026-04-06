// Package team 实现团队定义与角色管理
// 提供Agent角色定义、团队配置、角色切换等功能
package team

import (
	"fmt"
	"sync"
)

// AgentRole 定义Agent角色的属性和能力
type AgentRole struct {
	// Name 角色名称，唯一标识
	Name string `json:"name" yaml:"name"`

	// Description 角色描述，说明角色的职责和能力
	Description string `json:"description" yaml:"description"`

	// SystemPrompt 系统提示词，定义角色的行为规范
	SystemPrompt string `json:"system_prompt" yaml:"system_prompt"`

	// Tools 该角色可用的工具列表
	// 工具名称列表，对应tools包中注册的工具
	Tools []string `json:"tools" yaml:"tools"`

	// CanFork 是否可以创建子代理
	// 主控Agent通常具有此能力，可以派生子任务给其他Agent
	CanFork bool `json:"can_fork" yaml:"can_fork"`

	// MaxIterations 最大迭代次数
	// 限制该角色在单次任务中的最大循环次数
	MaxIterations int `json:"max_iterations" yaml:"max_iterations"`

	// Priority 角色优先级，用于任务分配
	// 数值越高优先级越高
	Priority int `json:"priority" yaml:"priority"`

	// Tags 角色标签，用于分类和筛选
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// Team 定义Agent团队结构
type Team struct {
	// Name 团队名称
	Name string `json:"name" yaml:"name"`

	// Description 团队描述
	Description string `json:"description" yaml:"description"`

	// Roles 团队中的角色定义
	// key为角色名称，value为角色定义
	Roles map[string]*AgentRole `json:"roles" yaml:"roles"`

	// DefaultRole 默认角色
	// 当未指定角色时使用的默认角色
	DefaultRole string `json:"default_role" yaml:"default_role"`

	// Workflow 工作流定义
	// 定义角色之间的协作流程
	Workflow []WorkflowStep `json:"workflow,omitempty" yaml:"workflow,omitempty"`

	// mu 保护并发访问
	mu sync.RWMutex
}

// WorkflowStep 定义工作流步骤
type WorkflowStep struct {
	// StepName 步骤名称
	StepName string `json:"step_name" yaml:"step_name"`

	// Role 执行该步骤的角色
	Role string `json:"role" yaml:"role"`

	// Description 步骤描述
	Description string `json:"description" yaml:"description"`

	// NextSteps 下一步骤的条件分支
	// key为条件，value为下一步骤名称
	NextSteps map[string]string `json:"next_steps,omitempty" yaml:"next_steps,omitempty"`
}

// NewTeam 创建新的团队实例
func NewTeam(name, description string) *Team {
	return &Team{
		Name:        name,
		Description: description,
		Roles:       make(map[string]*AgentRole),
		Workflow:    make([]WorkflowStep, 0),
	}
}

// AddRole 添加角色到团队
func (t *Team) AddRole(role *AgentRole) error {
	if role == nil {
		return fmt.Errorf("role cannot be nil")
	}

	if role.Name == "" {
		return fmt.Errorf("role name cannot be empty")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.Roles[role.Name]; exists {
		return fmt.Errorf("role '%s' already exists in team", role.Name)
	}

	t.Roles[role.Name] = role

	// 如果是第一个角色，设置为默认角色
	if len(t.Roles) == 1 {
		t.DefaultRole = role.Name
	}

	return nil
}

// GetRole 获取角色定义
func (t *Team) GetRole(name string) (*AgentRole, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	role, exists := t.Roles[name]
	return role, exists
}

// RemoveRole 从团队中移除角色
func (t *Team) RemoveRole(name string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.Roles[name]; !exists {
		return fmt.Errorf("role '%s' not found in team", name)
	}

	delete(t.Roles, name)

	// 如果移除的是默认角色，需要重新设置
	if t.DefaultRole == name {
		t.DefaultRole = ""
		// 选择第一个角色作为默认角色
		for roleName := range t.Roles {
			t.DefaultRole = roleName
			break
		}
	}

	return nil
}

// SetDefaultRole 设置默认角色
func (t *Team) SetDefaultRole(name string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.Roles[name]; !exists {
		return fmt.Errorf("role '%s' not found in team", name)
	}

	t.DefaultRole = name
	return nil
}

// ListRoles 列出所有角色名称
func (t *Team) ListRoles() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	names := make([]string, 0, len(t.Roles))
	for name := range t.Roles {
		names = append(names, name)
	}
	return names
}

// AddWorkflowStep 添加工作流步骤
func (t *Team) AddWorkflowStep(step WorkflowStep) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 验证角色是否存在
	if _, exists := t.Roles[step.Role]; !exists {
		return fmt.Errorf("role '%s' not found in team", step.Role)
	}

	t.Workflow = append(t.Workflow, step)
	return nil
}

// GetWorkflow 获取工作流定义
func (t *Team) GetWorkflow() []WorkflowStep {
	t.mu.RLock()
	defer t.mu.RUnlock()

	workflow := make([]WorkflowStep, len(t.Workflow))
	copy(workflow, t.Workflow)
	return workflow
}

// DefaultTeam 创建默认团队配置
// 包含标准的Agent角色定义
func DefaultTeam() *Team {
	team := NewTeam("DefaultTeam", "默认Agent团队，包含完整的开发流程角色")

	// 添加所有预定义角色
	roles := GetPredefinedRoles()
	for _, role := range roles {
		_ = team.AddRole(role) // 忽略错误，因为这是初始化
	}

	// 设置默认角色
	_ = team.SetDefaultRole("orchestrator")

	// 定义默认工作流
	_ = team.AddWorkflowStep(WorkflowStep{
		StepName:    "task_analysis",
		Role:        "orchestrator",
		Description: "分析任务，制定执行计划",
		NextSteps: map[string]string{
			"need_design": "architecture_design",
			"direct_execute": "implementation",
		},
	})

	_ = team.AddWorkflowStep(WorkflowStep{
		StepName:    "architecture_design",
		Role:        "architect",
		Description: "进行架构设计和技术选型",
		NextSteps: map[string]string{
			"design_complete": "implementation",
		},
	})

	_ = team.AddWorkflowStep(WorkflowStep{
		StepName:    "implementation",
		Role:        "developer",
		Description: "实现功能代码",
		NextSteps: map[string]string{
			"code_complete": "testing",
		},
	})

	_ = team.AddWorkflowStep(WorkflowStep{
		StepName:    "testing",
		Role:        "tester",
		Description: "编写和执行测试",
		NextSteps: map[string]string{
			"test_pass": "code_review",
			"test_fail": "implementation",
		},
	})

	_ = team.AddWorkflowStep(WorkflowStep{
		StepName:    "code_review",
		Role:        "reviewer",
		Description: "代码审查",
		NextSteps: map[string]string{
			"review_pass": "task_complete",
			"review_fail": "implementation",
		},
	})

	_ = team.AddWorkflowStep(WorkflowStep{
		StepName:    "task_complete",
		Role:        "orchestrator",
		Description: "任务完成，整理输出",
	})

	return team
}

// Validate 验证团队配置的有效性
func (t *Team) Validate() error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.Name == "" {
		return fmt.Errorf("team name cannot be empty")
	}

	if len(t.Roles) == 0 {
		return fmt.Errorf("team must have at least one role")
	}

	if t.DefaultRole == "" {
		return fmt.Errorf("team must have a default role")
	}

	if _, exists := t.Roles[t.DefaultRole]; !exists {
		return fmt.Errorf("default role '%s' not found in team", t.DefaultRole)
	}

	// 验证工作流中的角色引用
	for i, step := range t.Workflow {
		if _, exists := t.Roles[step.Role]; !exists {
			return fmt.Errorf("workflow step %d: role '%s' not found", i, step.Role)
		}
	}

	return nil
}

// Clone 克隆团队配置
func (t *Team) Clone() *Team {
	t.mu.RLock()
	defer t.mu.RUnlock()

	newTeam := &Team{
		Name:         t.Name,
		Description:  t.Description,
		Roles:        make(map[string]*AgentRole),
		DefaultRole:  t.DefaultRole,
		Workflow:     make([]WorkflowStep, len(t.Workflow)),
	}

	// 复制角色
	for name, role := range t.Roles {
		roleCopy := *role
		roleCopy.Tools = make([]string, len(role.Tools))
		copy(roleCopy.Tools, role.Tools)
		roleCopy.Tags = make([]string, len(role.Tags))
		copy(roleCopy.Tags, role.Tags)
		newTeam.Roles[name] = &roleCopy
	}

	// 复制工作流
	copy(newTeam.Workflow, t.Workflow)

	return newTeam
}
