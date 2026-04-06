package team_test

import (
	"agentplus/internal/team"
	"fmt"
)

// ExampleDefaultTeam 演示如何使用默认团队配置
func ExampleDefaultTeam() {
	// 创建默认团队
	defaultTeam := team.DefaultTeam()

	// 获取默认角色
	fmt.Printf("默认角色: %s\n", defaultTeam.DefaultRole)

	// 获取角色数量
	fmt.Printf("角色总数: %d\n", len(defaultTeam.ListRoles()))

	// 验证团队配置
	err := defaultTeam.Validate()
	fmt.Printf("团队配置有效: %v\n", err == nil)

	// Output:
	// 默认角色: orchestrator
	// 角色总数: 6
	// 团队配置有效: true
}

// ExampleRoleManager 演示如何使用角色管理器
func ExampleRoleManager() {
	// 创建默认团队
	defaultTeam := team.DefaultTeam()

	// 创建角色管理器
	roleManager, err := team.NewRoleManager(defaultTeam)
	if err != nil {
		fmt.Printf("创建角色管理器失败: %v\n", err)
		return
	}

	// 获取当前角色
	currentRole := roleManager.GetCurrentRole()
	fmt.Printf("当前角色: %s\n", currentRole.Name)
	fmt.Printf("角色描述: %s\n", currentRole.Description)

	// 获取当前角色的可用工具
	tools := roleManager.GetCurrentAvailableTools()
	fmt.Printf("\n可用工具数量: %d\n", len(tools))
	fmt.Printf("部分工具: %v\n", tools[:3])

	// 检查是否可以Fork
	canFork := roleManager.CanCurrentFork()
	fmt.Printf("\n当前角色是否可以Fork: %v\n", canFork)

	// 切换角色
	newRole, err := roleManager.SwitchRole(team.RoleDeveloper, "需要实现代码")
	if err != nil {
		fmt.Printf("切换角色失败: %v\n", err)
		return
	}
	fmt.Printf("\n切换到角色: %s\n", newRole.Name)

	// 检查新角色是否可以Fork
	canFork, _ = roleManager.CanFork(team.RoleDeveloper)
	fmt.Printf("Developer角色是否可以Fork: %v\n", canFork)

	// 查看角色切换历史
	history := roleManager.GetRoleHistory()
	fmt.Printf("\n角色切换历史: %d 条记录\n", len(history))
	for i, record := range history {
		fmt.Printf("%d. %s -> %s: %s\n", i+1, record.FromRole, record.ToRole, record.Reason)
	}

	// Output:
	// 当前角色: orchestrator
	// 角色描述: 主控Agent，负责协调任务流程、分配子任务、维护全局状态。作为团队的核心协调者，确保任务按计划执行。
	//
	// 可用工具数量: 10
	// 部分工具: [read_file write_file list_directory]
	//
	// 当前角色是否可以Fork: true
	//
	// 切换到角色: developer
	// Developer角色是否可以Fork: false
	//
	// 角色切换历史: 1 条记录
	// 1. orchestrator -> developer: 需要实现代码
}

// Example_customTeam 演示如何创建自定义团队
func Example_customTeam() {
	// 创建自定义团队
	customTeam := team.NewTeam("CustomTeam", "自定义团队配置")

	// 添加自定义角色
	customRole := &team.AgentRole{
		Name:          "custom_role",
		Description:   "自定义角色示例",
		SystemPrompt:  "你是一个自定义角色...",
		Tools:         []string{"read_file", "write_file"},
		CanFork:       false,
		MaxIterations: 20,
		Priority:      50,
		Tags:          []string{"custom", "example"},
	}

	err := customTeam.AddRole(customRole)
	if err != nil {
		fmt.Printf("添加角色失败: %v\n", err)
		return
	}

	// 验证团队配置
	err = customTeam.Validate()
	if err != nil {
		fmt.Printf("团队配置无效: %v\n", err)
		return
	}

	fmt.Printf("团队名称: %s\n", customTeam.Name)
	fmt.Printf("团队描述: %s\n", customTeam.Description)
	fmt.Printf("角色数量: %d\n", len(customTeam.ListRoles()))

	// Output:
	// 团队名称: CustomTeam
	// 团队描述: 自定义团队配置
	// 角色数量: 1
}

// Example_workflow 演示如何使用工作流
func Example_workflow() {
	// 创建默认团队
	defaultTeam := team.DefaultTeam()

	// 获取工作流定义
	workflow := defaultTeam.GetWorkflow()

	fmt.Println("默认工作流步骤：")
	for i, step := range workflow {
		fmt.Printf("%d. %s (%s): %s\n", i+1, step.StepName, step.Role, step.Description)
		if len(step.NextSteps) > 0 {
			fmt.Println("   下一步骤：")
			for condition, nextStep := range step.NextSteps {
				fmt.Printf("   - %s -> %s\n", condition, nextStep)
			}
		}
	}

	// Output:
	// 默认工作流步骤：
	// 1. task_analysis (orchestrator): 分析任务，制定执行计划
	//    下一步骤：
	//    - need_design -> architecture_design
	//    - direct_execute -> implementation
	// 2. architecture_design (architect): 进行架构设计和技术选型
	//    下一步骤：
	//    - design_complete -> implementation
	// 3. implementation (developer): 实现功能代码
	//    下一步骤：
	//    - code_complete -> testing
	// 4. testing (tester): 编写和执行测试
	//    下一步骤：
	//    - test_pass -> code_review
	//    - test_fail -> implementation
	// 5. code_review (reviewer): 代码审查
	//    下一步骤：
	//    - review_pass -> task_complete
	//    - review_fail -> implementation
	// 6. task_complete (orchestrator): 任务完成，整理输出
}
