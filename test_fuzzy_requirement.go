package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"agentplus/internal/agent"
	"agentplus/internal/config"
	"agentplus/internal/model"
	"agentplus/internal/state"
	"agentplus/internal/tools"
)

func main() {
	fmt.Println("=== Agent Plus 模糊需求测试 - HTML5工具箱（带校验机制）===")
	fmt.Println()

	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	modelClient, err := model.NewClient(&model.Config{
		Endpoint:    cfg.Model.Endpoint,
		APIKey:      cfg.Model.APIKey,
		ModelName:   cfg.Model.ModelName,
		ContextSize: cfg.Model.ContextSize,
	})
	if err != nil {
		fmt.Printf("创建模型客户端失败: %v\n", err)
		os.Exit(1)
	}

	toolRegistry := tools.NewToolRegistry()
	tools.RegisterFilesystemTools(toolRegistry)
	tools.RegisterCommandTools(toolRegistry)

	stateManager, err := state.NewStateManager("./state", true)
	if err != nil {
		fmt.Printf("创建状态管理器失败: %v\n", err)
		os.Exit(1)
	}

	workDir := filepath.Join("project", "html-tools")
	taskGoal := `当前处于内网环境，缺少一些简单的在线工具,使用html5+js实现： 
1、Base64的加解密 
2、JSON结构完整性检测和格式化、对比 
3、Unix时间戳与当前时区时间的互相转换 
 
我希望页面设计是一个工作台样式，有一个首页，左侧是工具已打开的工具列表,默认第一项是首页（不可关闭），右侧一开始是首页，有各项工具的卡片，点击后，切换至对应工具的页面（不跳转到新页面，在原页面动态切换），左侧列表出现打开的工具箱，每一项的右侧有一个关闭按钮，点击后回到首页模块并移除工具项。 
要求：双击html文件使用浏览器打开，通过file协议，注意浏览器的安全限制。

工作目录: ` + workDir + `

完成后调用complete_task工具标记任务完成。`

	verifierConfig := &agent.VerifyConfig{
		CheckFileExists:    true,
		CheckFileNonEmpty:  true,
		CheckKeywords:      true,
		RequiredKeywords:   []string{"Base64", "JSON", "时间戳", "工作台"},
		KeywordMatchMode:   "all",
		MaxIssuesToReport:  10,
		CheckJSSyntax:      true, // 启用JavaScript语法检查
		CheckHTMLStructure: true, // 启用HTML结构检查
	}

	correctorConfig := &agent.SelfCorrectorConfig{
		MaxRetries:           10,
		ExponentialBackoff:   true,
		MaxFailureHistory:    50,
		FailurePatternWindow: 5,
		MaxReviewRetries:     10,
		MaxVerifyRetries:     10,
	}

	// 创建日志目录
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("创建日志目录失败: %v\n", err)
		os.Exit(1)
	}

	// 生成日志文件路径（使用时间戳）
	logPath := filepath.Join(logDir, fmt.Sprintf("agent_%s.log", time.Now().Format("20060102_150405")))

	ag, err := agent.NewAgent(&agent.Config{
		ModelClient:     modelClient,
		ToolRegistry:    toolRegistry,
		StateManager:    stateManager,
		MaxIterations:   100,
		SystemPrompt:    agent.GetSystemPrompt(agent.PromptTypeOrchestrator) + agent.VerificationPrompt,
		VerifierConfig:  verifierConfig,
		CorrectorConfig: correctorConfig,
		LogPath:         logPath,
	})
	if err != nil {
		fmt.Printf("创建Agent失败: %v\n", err)
		os.Exit(1)
	}
	defer ag.Close() // 确保关闭Agent，释放资源

	verifier := ag.GetVerifier()
	if verifier != nil {
		verifierFunc := func(ctx context.Context, taskGoal string, workDir string) *tools.VerifyResult {
			files := []string{filepath.Join(workDir, "index.html")}
			taskState, _ := stateManager.GetState(ag.GetTaskID())
			result := verifier.VerifyTaskCompletion(files, taskState)
			return convertVerifyResult(result)
		}
		tools.RegisterStateToolsWithVerifier(toolRegistry, verifierFunc)
	}

	fmt.Println()
	fmt.Println("=== 开始执行任务 ===")
	fmt.Println()
	fmt.Printf("任务: %s\n\n", taskGoal)
	fmt.Println("校验配置:")
	fmt.Printf("  - 必需关键词: %v\n", verifierConfig.RequiredKeywords)
	fmt.Printf("  - 最大重试次数: %d\n", correctorConfig.MaxRetries)
	fmt.Printf("  - 日志文件: %s\n", logPath)
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Hour)
	defer cancel()

	result, err := ag.Run(ctx, taskGoal)
	if err != nil {
		fmt.Printf("执行任务失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("=== 任务执行结果 ===")
	fmt.Printf("状态: %s\n", result.Status)
	fmt.Printf("迭代次数: %d\n", result.Iterations)
	fmt.Printf("耗时: %v\n", result.Duration)

	corrector := ag.GetCorrector()
	if corrector != nil {
		stats := corrector.GetStatistics()
		fmt.Println()
		fmt.Println("=== 修正统计 ===")
		fmt.Printf("总失败次数: %d\n", stats.TotalFailures)
		fmt.Printf("已修正失败: %d\n", stats.CorrectedFailures)
		fmt.Printf("总修正次数: %d\n", stats.TotalCorrections)
		fmt.Printf("修正成功率: %.1f%%\n", stats.CorrectionSuccessRate*100)
		fmt.Printf("剩余重试次数: %d\n", stats.RemainingRetries)
	}

	logger := ag.GetLogger()
	if logger != nil {
		fmt.Println()
		fmt.Printf("日志文件: %s\n", logger.GetLogPath())
	}

	if result.Status == "completed" {
		fmt.Println()
		fmt.Println("=== 测试通过 ===")
	} else {
		fmt.Println()
		fmt.Println("=== 测试失败 ===")
		os.Exit(1)
	}
}

func convertVerifyResult(ar *agent.VerifyResult) *tools.VerifyResult {
	if ar == nil {
		return nil
	}
	issues := make([]tools.VerifyIssue, len(ar.Issues))
	for i, issue := range ar.Issues {
		issues[i] = tools.VerifyIssue{
			Type:        string(issue.Type),
			Severity:    issue.Severity,
			Description: issue.Description,
			Evidence:    issue.Evidence,
			Suggestion:  issue.Suggestion,
		}
	}
	return &tools.VerifyResult{
		Status:  string(ar.Status),
		Issues:  issues,
		Summary: ar.Summary,
		Passed:  ar.Passed,
		Failed:  ar.Failed,
	}
}
