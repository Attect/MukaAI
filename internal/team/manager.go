package team

import (
	"fmt"
	"sync"
	"time"
)

// RoleManager 角色管理器
// 提供角色查询、切换、工具权限管理等功能
type RoleManager struct {
	// team 团队配置
	team *Team

	// currentRole 当前活动角色
	currentRole string

	// roleHistory 角色切换历史
	roleHistory []RoleSwitchRecord

	// mu 保护并发访问
	mu sync.RWMutex
}

// RoleSwitchRecord 角色切换记录
type RoleSwitchRecord struct {
	// FromRole 切换前角色
	FromRole string `json:"from_role"`

	// ToRole 切换后角色
	ToRole string `json:"to_role"`

	// Reason 切换原因
	Reason string `json:"reason"`

	// Timestamp 切换时间戳
	Timestamp string `json:"timestamp"`
}

// NewRoleManager 创建新的角色管理器
func NewRoleManager(team *Team) (*RoleManager, error) {
	if team == nil {
		return nil, fmt.Errorf("team cannot be nil")
	}

	if err := team.Validate(); err != nil {
		return nil, fmt.Errorf("invalid team configuration: %w", err)
	}

	return &RoleManager{
		team:        team,
		currentRole: team.DefaultRole,
		roleHistory: make([]RoleSwitchRecord, 0),
	}, nil
}

// GetRole 获取角色定义
func (rm *RoleManager) GetRole(name string) (*AgentRole, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	role, exists := rm.team.GetRole(name)
	if !exists {
		return nil, fmt.Errorf("role '%s' not found", name)
	}

	return role, nil
}

// GetCurrentRole 获取当前活动角色
func (rm *RoleManager) GetCurrentRole() *AgentRole {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	role, exists := rm.team.GetRole(rm.currentRole)
	if !exists {
		// 如果当前角色不存在，返回默认角色
		role, _ = rm.team.GetRole(rm.team.DefaultRole)
	}
	return role
}

// GetCurrentRoleName 获取当前角色名称
func (rm *RoleManager) GetCurrentRoleName() string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.currentRole
}

// GetSystemPrompt 获取指定角色的系统提示词
func (rm *RoleManager) GetSystemPrompt(roleName string) (string, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	role, exists := rm.team.GetRole(roleName)
	if !exists {
		return "", fmt.Errorf("role '%s' not found", roleName)
	}

	return role.SystemPrompt, nil
}

// GetCurrentSystemPrompt 获取当前角色的系统提示词
func (rm *RoleManager) GetCurrentSystemPrompt() string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	role, exists := rm.team.GetRole(rm.currentRole)
	if !exists {
		role, _ = rm.team.GetRole(rm.team.DefaultRole)
	}
	return role.SystemPrompt
}

// GetAvailableTools 获取指定角色可用的工具列表
func (rm *RoleManager) GetAvailableTools(roleName string) ([]string, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	role, exists := rm.team.GetRole(roleName)
	if !exists {
		return nil, fmt.Errorf("role '%s' not found", roleName)
	}

	// 返回工具列表的副本
	tools := make([]string, len(role.Tools))
	copy(tools, role.Tools)
	return tools, nil
}

// GetCurrentAvailableTools 获取当前角色可用的工具列表
func (rm *RoleManager) GetCurrentAvailableTools() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	role, exists := rm.team.GetRole(rm.currentRole)
	if !exists {
		role, _ = rm.team.GetRole(rm.team.DefaultRole)
	}

	tools := make([]string, len(role.Tools))
	copy(tools, role.Tools)
	return tools
}

// HasToolPermission 检查指定角色是否有权限使用某个工具
func (rm *RoleManager) HasToolPermission(roleName, toolName string) (bool, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	role, exists := rm.team.GetRole(roleName)
	if !exists {
		return false, fmt.Errorf("role '%s' not found", roleName)
	}

	for _, tool := range role.Tools {
		if tool == toolName {
			return true, nil
		}
	}

	return false, nil
}

// CanFork 检查指定角色是否可以创建子代理
func (rm *RoleManager) CanFork(roleName string) (bool, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	role, exists := rm.team.GetRole(roleName)
	if !exists {
		return false, fmt.Errorf("role '%s' not found", roleName)
	}

	return role.CanFork, nil
}

// CanCurrentFork 检查当前角色是否可以创建子代理
func (rm *RoleManager) CanCurrentFork() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	role, exists := rm.team.GetRole(rm.currentRole)
	if !exists {
		role, _ = rm.team.GetRole(rm.team.DefaultRole)
	}
	return role.CanFork
}

// GetMaxIterations 获取指定角色的最大迭代次数
func (rm *RoleManager) GetMaxIterations(roleName string) (int, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	role, exists := rm.team.GetRole(roleName)
	if !exists {
		return 0, fmt.Errorf("role '%s' not found", roleName)
	}

	return role.MaxIterations, nil
}

// GetCurrentMaxIterations 获取当前角色的最大迭代次数
func (rm *RoleManager) GetCurrentMaxIterations() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	role, exists := rm.team.GetRole(rm.currentRole)
	if !exists {
		role, _ = rm.team.GetRole(rm.team.DefaultRole)
	}
	return role.MaxIterations
}

// SwitchRole 切换角色
// 返回切换后的角色定义
func (rm *RoleManager) SwitchRole(newRole, reason string) (*AgentRole, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 验证新角色是否存在
	role, exists := rm.team.GetRole(newRole)
	if !exists {
		return nil, fmt.Errorf("role '%s' not found", newRole)
	}

	// 记录切换历史
	record := RoleSwitchRecord{
		FromRole:  rm.currentRole,
		ToRole:    newRole,
		Reason:    reason,
		Timestamp: currentTime(),
	}
	rm.roleHistory = append(rm.roleHistory, record)

	// 更新当前角色
	rm.currentRole = newRole

	return role, nil
}

// SwitchRoleWithValidation 切换角色并进行权限验证
// 检查当前角色是否有权限切换到目标角色
func (rm *RoleManager) SwitchRoleWithValidation(newRole, reason string) (*AgentRole, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 获取当前角色
	currentRole, exists := rm.team.GetRole(rm.currentRole)
	if !exists {
		return nil, fmt.Errorf("current role '%s' not found", rm.currentRole)
	}

	// 验证新角色是否存在
	targetRole, exists := rm.team.GetRole(newRole)
	if !exists {
		return nil, fmt.Errorf("target role '%s' not found", newRole)
	}

	// 只有可以Fork的角色才能切换角色
	if !currentRole.CanFork {
		return nil, fmt.Errorf("current role '%s' cannot switch to other roles", rm.currentRole)
	}

	// 记录切换历史
	record := RoleSwitchRecord{
		FromRole:  rm.currentRole,
		ToRole:    newRole,
		Reason:    reason,
		Timestamp: currentTime(),
	}
	rm.roleHistory = append(rm.roleHistory, record)

	// 更新当前角色
	rm.currentRole = newRole

	return targetRole, nil
}

// GetRoleHistory 获取角色切换历史
func (rm *RoleManager) GetRoleHistory() []RoleSwitchRecord {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	history := make([]RoleSwitchRecord, len(rm.roleHistory))
	copy(history, rm.roleHistory)
	return history
}

// ListAllRoles 列出所有角色名称
func (rm *RoleManager) ListAllRoles() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.team.ListRoles()
}

// GetTeam 获取团队配置
func (rm *RoleManager) GetTeam() *Team {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.team
}

// ResetToDefault 重置到默认角色
func (rm *RoleManager) ResetToDefault() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	record := RoleSwitchRecord{
		FromRole:  rm.currentRole,
		ToRole:    rm.team.DefaultRole,
		Reason:    "reset to default role",
		Timestamp: currentTime(),
	}
	rm.roleHistory = append(rm.roleHistory, record)

	rm.currentRole = rm.team.DefaultRole
	return nil
}

// GetRolePriority 获取指定角色的优先级
func (rm *RoleManager) GetRolePriority(roleName string) (int, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	role, exists := rm.team.GetRole(roleName)
	if !exists {
		return 0, fmt.Errorf("role '%s' not found", roleName)
	}

	return role.Priority, nil
}

// GetRoleTags 获取指定角色的标签
func (rm *RoleManager) GetRoleTags(roleName string) ([]string, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	role, exists := rm.team.GetRole(roleName)
	if !exists {
		return nil, fmt.Errorf("role '%s' not found", roleName)
	}

	tags := make([]string, len(role.Tags))
	copy(tags, role.Tags)
	return tags, nil
}

// FindRolesByTag 根据标签查找角色
func (rm *RoleManager) FindRolesByTag(tag string) []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var matchedRoles []string
	for name, role := range rm.team.Roles {
		for _, roleTag := range role.Tags {
			if roleTag == tag {
				matchedRoles = append(matchedRoles, name)
				break
			}
		}
	}

	return matchedRoles
}

// GetWorkflow 获取工作流定义
func (rm *RoleManager) GetWorkflow() []WorkflowStep {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.team.GetWorkflow()
}

// currentTime 返回当前时间的字符串表示
// 这是一个辅助函数，便于测试时mock
func currentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
