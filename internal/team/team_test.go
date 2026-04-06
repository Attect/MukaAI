package team

import (
	"testing"
)

// TestNewTeam 测试团队创建
func TestNewTeam(t *testing.T) {
	team := NewTeam("TestTeam", "测试团队")

	if team.Name != "TestTeam" {
		t.Errorf("Expected team name 'TestTeam', got '%s'", team.Name)
	}

	if team.Description != "测试团队" {
		t.Errorf("Expected description '测试团队', got '%s'", team.Description)
	}

	if team.Roles == nil {
		t.Error("Roles map should not be nil")
	}

	if len(team.Roles) != 0 {
		t.Errorf("New team should have no roles, got %d", len(team.Roles))
	}
}

// TestAddRole 测试添加角色
func TestAddRole(t *testing.T) {
	team := NewTeam("TestTeam", "测试团队")

	role := &AgentRole{
		Name:        "test_role",
		Description: "测试角色",
		Tools:       []string{"read_file"},
		CanFork:     false,
	}

	err := team.AddRole(role)
	if err != nil {
		t.Errorf("Failed to add role: %v", err)
	}

	if len(team.Roles) != 1 {
		t.Errorf("Expected 1 role, got %d", len(team.Roles))
	}

	// 测试添加重复角色
	err = team.AddRole(role)
	if err == nil {
		t.Error("Should fail when adding duplicate role")
	}

	// 测试添加nil角色
	err = team.AddRole(nil)
	if err == nil {
		t.Error("Should fail when adding nil role")
	}

	// 测试添加空名称角色
	err = team.AddRole(&AgentRole{Name: ""})
	if err == nil {
		t.Error("Should fail when adding role with empty name")
	}
}

// TestGetRole 测试获取角色
func TestGetRole(t *testing.T) {
	team := NewTeam("TestTeam", "测试团队")

	role := &AgentRole{
		Name:        "test_role",
		Description: "测试角色",
	}

	_ = team.AddRole(role)

	// 测试获取存在的角色
	gotRole, exists := team.GetRole("test_role")
	if !exists {
		t.Error("Role should exist")
	}
	if gotRole.Name != "test_role" {
		t.Errorf("Expected role name 'test_role', got '%s'", gotRole.Name)
	}

	// 测试获取不存在的角色
	_, exists = team.GetRole("non_existent")
	if exists {
		t.Error("Non-existent role should not exist")
	}
}

// TestRemoveRole 测试移除角色
func TestRemoveRole(t *testing.T) {
	team := NewTeam("TestTeam", "测试团队")

	role := &AgentRole{
		Name:        "test_role",
		Description: "测试角色",
	}

	_ = team.AddRole(role)

	// 测试移除存在的角色
	err := team.RemoveRole("test_role")
	if err != nil {
		t.Errorf("Failed to remove role: %v", err)
	}

	if len(team.Roles) != 0 {
		t.Errorf("Expected 0 roles after removal, got %d", len(team.Roles))
	}

	// 测试移除不存在的角色
	err = team.RemoveRole("non_existent")
	if err == nil {
		t.Error("Should fail when removing non-existent role")
	}
}

// TestSetDefaultRole 测试设置默认角色
func TestSetDefaultRole(t *testing.T) {
	team := NewTeam("TestTeam", "测试团队")

	role1 := &AgentRole{Name: "role1"}
	role2 := &AgentRole{Name: "role2"}

	_ = team.AddRole(role1)
	_ = team.AddRole(role2)

	// 第一个角色应该自动成为默认角色
	if team.DefaultRole != "role1" {
		t.Errorf("Expected default role 'role1', got '%s'", team.DefaultRole)
	}

	// 测试设置默认角色
	err := team.SetDefaultRole("role2")
	if err != nil {
		t.Errorf("Failed to set default role: %v", err)
	}

	if team.DefaultRole != "role2" {
		t.Errorf("Expected default role 'role2', got '%s'", team.DefaultRole)
	}

	// 测试设置不存在的角色为默认角色
	err = team.SetDefaultRole("non_existent")
	if err == nil {
		t.Error("Should fail when setting non-existent role as default")
	}
}

// TestListRoles 测试列出角色
func TestListRoles(t *testing.T) {
	team := NewTeam("TestTeam", "测试团队")

	role1 := &AgentRole{Name: "role1"}
	role2 := &AgentRole{Name: "role2"}

	_ = team.AddRole(role1)
	_ = team.AddRole(role2)

	roles := team.ListRoles()
	if len(roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(roles))
	}
}

// TestValidate 测试团队验证
func TestValidate(t *testing.T) {
	// 测试空名称团队
	team := NewTeam("", "测试团队")
	err := team.Validate()
	if err == nil {
		t.Error("Team with empty name should fail validation")
	}

	// 测试无角色团队
	team = NewTeam("TestTeam", "测试团队")
	err = team.Validate()
	if err == nil {
		t.Error("Team with no roles should fail validation")
	}

	// 测试有效团队
	role := &AgentRole{Name: "test_role"}
	_ = team.AddRole(role)
	err = team.Validate()
	if err != nil {
		t.Errorf("Valid team should pass validation: %v", err)
	}
}

// TestClone 测试团队克隆
func TestClone(t *testing.T) {
	team := NewTeam("TestTeam", "测试团队")

	role := &AgentRole{
		Name:        "test_role",
		Description: "测试角色",
		Tools:       []string{"read_file", "write_file"},
		Tags:        []string{"core"},
	}

	_ = team.AddRole(role)

	clonedTeam := team.Clone()

	if clonedTeam.Name != team.Name {
		t.Errorf("Cloned team name should match original")
	}

	if len(clonedTeam.Roles) != len(team.Roles) {
		t.Errorf("Cloned team should have same number of roles")
	}

	// 验证是深拷贝
	clonedRole, _ := clonedTeam.GetRole("test_role")
	clonedRole.Description = "修改后的描述"

	originalRole, _ := team.GetRole("test_role")
	if originalRole.Description == "修改后的描述" {
		t.Error("Cloned team should be independent from original")
	}
}

// TestNewRoleManager 测试角色管理器创建
func TestNewRoleManager(t *testing.T) {
	team := NewTeam("TestTeam", "测试团队")
	role := &AgentRole{Name: "test_role"}
	_ = team.AddRole(role)

	// 测试有效团队
	rm, err := NewRoleManager(team)
	if err != nil {
		t.Errorf("Failed to create role manager: %v", err)
	}

	if rm.currentRole != "test_role" {
		t.Errorf("Expected current role 'test_role', got '%s'", rm.currentRole)
	}

	// 测试nil团队
	_, err = NewRoleManager(nil)
	if err == nil {
		t.Error("Should fail when creating role manager with nil team")
	}

	// 测试无效团队
	invalidTeam := NewTeam("", "测试团队")
	_, err = NewRoleManager(invalidTeam)
	if err == nil {
		t.Error("Should fail when creating role manager with invalid team")
	}
}

// TestGetRole 测试角色管理器获取角色
func TestRoleManager_GetRole(t *testing.T) {
	team := NewTeam("TestTeam", "测试团队")
	role := &AgentRole{
		Name:        "test_role",
		Description: "测试角色",
	}
	_ = team.AddRole(role)

	rm, _ := NewRoleManager(team)

	// 测试获取存在的角色
	gotRole, err := rm.GetRole("test_role")
	if err != nil {
		t.Errorf("Failed to get role: %v", err)
	}
	if gotRole.Name != "test_role" {
		t.Errorf("Expected role name 'test_role', got '%s'", gotRole.Name)
	}

	// 测试获取不存在的角色
	_, err = rm.GetRole("non_existent")
	if err == nil {
		t.Error("Should fail when getting non-existent role")
	}
}

// TestGetCurrentRole 测试获取当前角色
func TestRoleManager_GetCurrentRole(t *testing.T) {
	team := NewTeam("TestTeam", "测试团队")
	role := &AgentRole{
		Name:        "test_role",
		Description: "测试角色",
	}
	_ = team.AddRole(role)

	rm, _ := NewRoleManager(team)

	currentRole := rm.GetCurrentRole()
	if currentRole.Name != "test_role" {
		t.Errorf("Expected current role 'test_role', got '%s'", currentRole.Name)
	}
}

// TestSwitchRole 测试角色切换
func TestRoleManager_SwitchRole(t *testing.T) {
	team := NewTeam("TestTeam", "测试团队")
	role1 := &AgentRole{Name: "role1", CanFork: true}
	role2 := &AgentRole{Name: "role2", CanFork: false}
	_ = team.AddRole(role1)
	_ = team.AddRole(role2)

	rm, _ := NewRoleManager(team)

	// 测试切换角色
	newRole, err := rm.SwitchRole("role2", "测试切换")
	if err != nil {
		t.Errorf("Failed to switch role: %v", err)
	}

	if newRole.Name != "role2" {
		t.Errorf("Expected new role 'role2', got '%s'", newRole.Name)
	}

	if rm.GetCurrentRoleName() != "role2" {
		t.Errorf("Expected current role 'role2', got '%s'", rm.GetCurrentRoleName())
	}

	// 测试切换到不存在的角色
	_, err = rm.SwitchRole("non_existent", "测试")
	if err == nil {
		t.Error("Should fail when switching to non-existent role")
	}

	// 验证切换历史
	history := rm.GetRoleHistory()
	if len(history) != 1 {
		t.Errorf("Expected 1 history record, got %d", len(history))
	}
}

// TestHasToolPermission 测试工具权限检查
func TestRoleManager_HasToolPermission(t *testing.T) {
	team := NewTeam("TestTeam", "测试团队")
	role := &AgentRole{
		Name:  "test_role",
		Tools: []string{"read_file", "write_file"},
	}
	_ = team.AddRole(role)

	rm, _ := NewRoleManager(team)

	// 测试有权限的工具
	hasPermission, err := rm.HasToolPermission("test_role", "read_file")
	if err != nil {
		t.Errorf("Failed to check permission: %v", err)
	}
	if !hasPermission {
		t.Error("Should have permission for 'read_file'")
	}

	// 测试无权限的工具
	hasPermission, err = rm.HasToolPermission("test_role", "delete_file")
	if err != nil {
		t.Errorf("Failed to check permission: %v", err)
	}
	if hasPermission {
		t.Error("Should not have permission for 'delete_file'")
	}
}

// TestCanFork 测试Fork能力检查
func TestRoleManager_CanFork(t *testing.T) {
	team := NewTeam("TestTeam", "测试团队")
	role1 := &AgentRole{Name: "role1", CanFork: true}
	role2 := &AgentRole{Name: "role2", CanFork: false}
	_ = team.AddRole(role1)
	_ = team.AddRole(role2)

	rm, _ := NewRoleManager(team)

	// 测试可以Fork的角色
	canFork, err := rm.CanFork("role1")
	if err != nil {
		t.Errorf("Failed to check fork ability: %v", err)
	}
	if !canFork {
		t.Error("role1 should be able to fork")
	}

	// 测试不能Fork的角色
	canFork, err = rm.CanFork("role2")
	if err != nil {
		t.Errorf("Failed to check fork ability: %v", err)
	}
	if canFork {
		t.Error("role2 should not be able to fork")
	}
}

// TestDefaultTeam 测试默认团队创建
func TestDefaultTeam(t *testing.T) {
	team := DefaultTeam()

	if team.Name != "DefaultTeam" {
		t.Errorf("Expected team name 'DefaultTeam', got '%s'", team.Name)
	}

	// 验证所有预定义角色都存在
	expectedRoles := []string{
		RoleOrchestrator,
		RoleArchitect,
		RoleDeveloper,
		RoleTester,
		RoleReviewer,
		RoleSupervisor,
	}

	for _, roleName := range expectedRoles {
		role, exists := team.GetRole(roleName)
		if !exists {
			t.Errorf("Expected role '%s' not found", roleName)
			continue
		}

		if role.Name == "" {
			t.Errorf("Role '%s' has empty name", roleName)
		}

		if role.Description == "" {
			t.Errorf("Role '%s' has empty description", roleName)
		}

		if role.SystemPrompt == "" {
			t.Errorf("Role '%s' has empty system prompt", roleName)
		}

		if len(role.Tools) == 0 {
			t.Errorf("Role '%s' has no tools", roleName)
		}
	}

	// 验证默认角色
	if team.DefaultRole != RoleOrchestrator {
		t.Errorf("Expected default role 'orchestrator', got '%s'", team.DefaultRole)
	}

	// 验证团队配置有效
	err := team.Validate()
	if err != nil {
		t.Errorf("Default team should be valid: %v", err)
	}

	// 验证工作流
	workflow := team.GetWorkflow()
	if len(workflow) == 0 {
		t.Error("Default team should have workflow defined")
	}
}

// TestGetPredefinedRoles 测试获取预定义角色
func TestGetPredefinedRoles(t *testing.T) {
	roles := GetPredefinedRoles()

	if len(roles) != 6 {
		t.Errorf("Expected 6 predefined roles, got %d", len(roles))
	}

	// 验证每个角色都有必要的字段
	for _, role := range roles {
		if role.Name == "" {
			t.Error("Role has empty name")
		}

		if role.Description == "" {
			t.Errorf("Role '%s' has empty description", role.Name)
		}

		if role.SystemPrompt == "" {
			t.Errorf("Role '%s' has empty system prompt", role.Name)
		}

		if len(role.Tools) == 0 {
			t.Errorf("Role '%s' has no tools", role.Name)
		}

		if role.MaxIterations <= 0 {
			t.Errorf("Role '%s' has invalid MaxIterations: %d", role.Name, role.MaxIterations)
		}

		if role.Priority <= 0 {
			t.Errorf("Role '%s' has invalid Priority: %d", role.Name, role.Priority)
		}
	}
}

// TestRoleManager_FindRolesByTag 测试根据标签查找角色
func TestRoleManager_FindRolesByTag(t *testing.T) {
	team := NewTeam("TestTeam", "测试团队")
	role1 := &AgentRole{
		Name: "role1",
		Tags: []string{"core", "manager"},
	}
	role2 := &AgentRole{
		Name: "role2",
		Tags: []string{"worker"},
	}
	_ = team.AddRole(role1)
	_ = team.AddRole(role2)

	rm, _ := NewRoleManager(team)

	// 测试查找存在的标签
	coreRoles := rm.FindRolesByTag("core")
	if len(coreRoles) != 1 {
		t.Errorf("Expected 1 role with 'core' tag, got %d", len(coreRoles))
	}

	// 测试查找不存在的标签
	nonExistent := rm.FindRolesByTag("non_existent")
	if len(nonExistent) != 0 {
		t.Errorf("Expected 0 roles with 'non_existent' tag, got %d", len(nonExistent))
	}
}

// TestRoleManager_ResetToDefault 测试重置到默认角色
func TestRoleManager_ResetToDefault(t *testing.T) {
	team := NewTeam("TestTeam", "测试团队")
	role1 := &AgentRole{Name: "role1", CanFork: true}
	role2 := &AgentRole{Name: "role2", CanFork: false}
	_ = team.AddRole(role1)
	_ = team.AddRole(role2)
	_ = team.SetDefaultRole("role1")

	rm, _ := NewRoleManager(team)

	// 切换角色
	_, _ = rm.SwitchRole("role2", "测试")

	// 重置到默认角色
	err := rm.ResetToDefault()
	if err != nil {
		t.Errorf("Failed to reset to default: %v", err)
	}

	if rm.GetCurrentRoleName() != "role1" {
		t.Errorf("Expected current role 'role1' after reset, got '%s'", rm.GetCurrentRoleName())
	}
}
