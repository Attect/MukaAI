// Package main 提供MukaAI命令行入口
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Attect/MukaAI/internal/agent"
	"github.com/Attect/MukaAI/internal/config"
	ctxpkg "github.com/Attect/MukaAI/internal/context"
	"github.com/Attect/MukaAI/internal/lsp"
	"github.com/Attect/MukaAI/internal/mcp"
	"github.com/Attect/MukaAI/internal/model"
	"github.com/Attect/MukaAI/internal/state"
	"github.com/Attect/MukaAI/internal/supervisor"
	"github.com/Attect/MukaAI/internal/tools"
	"github.com/Attect/MukaAI/internal/tools/git"
)

// 版本信息
const (
	Version = "1.0.0"
	Name    = "MukaAI"
)

// 命令行参数
type CLIOptions struct {
	ConfigPath    string
	TaskID        string
	WorkDir       string
	Verbose       bool
	Debug         bool
	NoSupervisor  bool
	MaxIterations int
	TaskGoal      string
}

func main() {
	// 检查是否为子命令模式
	if len(os.Args) > 1 {
		cmd := os.Args[1]

		// 检查是否为 gui 子命令
		if cmd == "gui" {
			runGUICommand()
			return
		}

		// 检查是否为 help 或 version
		if cmd == "help" || cmd == "--help" || cmd == "-h" {
			printUsage()
			return
		}
		if cmd == "version" || cmd == "--version" || cmd == "-v" {
			fmt.Printf("%s v%s\n", Name, Version)
			return
		}
	}

	// 默认运行 CLI 模式
	runCLICommand()
}

// runCLICommand 运行 CLI 模式
func runCLICommand() {
	// 解析命令行参数
	opts := parseFlags()

	// 加载配置
	cfg, err := config.LoadConfig(opts.ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// 命令行参数覆盖配置
	if opts.WorkDir != "" {
		cfg.Tools.WorkDir = opts.WorkDir
	}
	if opts.MaxIterations > 0 {
		cfg.Agent.MaxIterations = opts.MaxIterations
	}

	// 获取绝对工作目录
	workDir, err := cfg.GetAbsoluteWorkDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting work directory: %v\n", err)
		os.Exit(1)
	}

	// 获取绝对状态目录
	stateDir, err := cfg.GetAbsoluteStateDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting state directory: %v\n", err)
		os.Exit(1)
	}

	// 初始化组件
	modelClient, err := initModelClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing model client: %v\n", err)
		os.Exit(1)
	}

	toolRegistry, err := initToolRegistry(workDir, cfg.Tools.AllowCommands)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing tool registry: %v\n", err)
		os.Exit(1)
	}

	stateManager, err := state.NewStateManager(stateDir, cfg.State.AutoSave)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing state manager: %v\n", err)
		os.Exit(1)
	}

	// 启动状态文件自动清理
	cleanupConfig := state.CleanupConfig{
		RetentionDays: cfg.State.CleanupDays,
		CheckInterval: 24 * time.Hour,
		Enabled:       cfg.State.CleanupEnable,
	}
	stateManagerWithCleanup, err := state.NewStateManagerWithCleanup(stateDir, cfg.State.AutoSave, cleanupConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing state manager with cleanup: %v\n", err)
		os.Exit(1)
	}
	// 替换为带清理功能的实例
	stateManager = stateManagerWithCleanup

	// 初始化MCP客户端管理器（如果启用）
	var mcpManager *mcp.MCPClientManager
	if cfg.MCP.Enabled {
		mcpManager = initMCPManager(cfg, toolRegistry, workDir)
	}

	// 初始化LSP管理器并注册诊断工具（如果启用）
	var lspManager *lsp.LSPManager
	if cfg.LSP.Enabled {
		lspManager = initLSPManager(cfg, toolRegistry, workDir)
	}

	// 解析日志路径（相对于工作目录）
	logPath := cfg.Logging.LogPath
	if logPath != "" && !filepath.IsAbs(logPath) {
		logPath = filepath.Join(workDir, logPath)
	}

	// 创建Agent
	ag, err := agent.NewAgent(&agent.Config{
		ModelClient:   modelClient,
		ToolRegistry:  toolRegistry,
		StateManager:  stateManager,
		MaxIterations: cfg.Agent.MaxIterations,
		PromptType:    agent.PromptTypeOrchestrator,
		WorkDir:       workDir,
		LogPath:       logPath,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating agent: %v\n", err)
		os.Exit(1)
	}

	// 设置任务ID（如果指定）
	if opts.TaskID != "" {
		ag.SetTaskID(opts.TaskID)
	}

	// 创建代码上下文索引器（异步构建索引）
	indexer := ctxpkg.NewIndexer(workDir)
	scanCh := indexer.ScanAsync()
	go func() {
		result := <-scanCh
		if result.Err != nil {
			log.Printf("Warning: context indexing failed: %v", result.Err)
			return
		}
		log.Printf("Context index built: %d files indexed", result.FileCount)
	}()

	// 创建上下文注入器并设置到Agent
	injector := ctxpkg.NewInjectorFromContextSize(indexer, cfg.Model.ContextSize)
	ag.SetContextInjector(injector)

	// 创建Supervisor监督器（默认启用，--no-supervisor时跳过）
	var sup agent.Supervisor
	if !opts.NoSupervisor {
		supInstance, err := supervisor.NewSupervisor(
			modelClient,
			toolRegistry,
			stateManager,
			ag.GetReviewer(),
			nil, // 使用默认配置
		)
		if err != nil {
			log.Printf("Warning: failed to create supervisor, running without supervision: %v", err)
		} else {
			sup = supInstance
			ag.SetSupervisor(sup)

			// 设置监督结果回调（CLI模式简单输出）
			ag.SetOnSupervisor(func(result *agent.SupervisionResult) {
				if result.Status != "pass" {
					fmt.Printf("\n\033[33m[Supervisor]\033[0m %s\n", result.Summary)
					if result.InterventionType != "" && result.InterventionType != "warning" {
						fmt.Printf("\033[33m  Intervention: %s - %s\033[0m\n", result.InterventionType, result.InterventionAction)
					}
				}
			})
		}
	}

	// 设置回调
	if opts.Debug {
		setupDebugCallbacks(ag)
	} else {
		setupCallbacks(ag, opts.Verbose)
	}

	// 创建上下文，支持优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动状态文件自动清理
	stateManager.StartCleanup(ctx)
	defer stateManager.StopCleanup()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, stopping...")
		// 关闭MCP连接
		if mcpManager != nil {
			_ = mcpManager.Shutdown()
		}
		// 关闭LSP语言服务器
		if lspManager != nil {
			lspManager.ShutdownAll()
		}
		cancel()
	}()

	// 获取任务目标
	taskGoal := opts.TaskGoal
	if taskGoal == "" {
		// 交互式输入
		taskGoal, err = interactiveInput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}
		if taskGoal == "" {
			fmt.Println("No task provided, exiting.")
			os.Exit(0)
		}
	}

	// 执行任务
	fmt.Printf("\n%s v%s - Starting task...\n", Name, Version)
	fmt.Printf("Task: %s\n", taskGoal)
	fmt.Println(strings.Repeat("-", 60))

	result, err := ag.Run(ctx, taskGoal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		if result != nil {
			printResult(result)
		}
		os.Exit(1)
	}

	// 打印结果
	printResult(result)

	// 关闭MCP连接
	if mcpManager != nil {
		_ = mcpManager.Shutdown()
	}

	// 关闭LSP语言服务器
	if lspManager != nil {
		lspManager.ShutdownAll()
	}
}

// printUsage 打印使用说明
func printUsage() {
	fmt.Fprintf(os.Stderr, "%s v%s - AI Agent CLI Tool\n\n", Name, Version)
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  mukaai [options] <task>        Run in CLI mode (default)\n")
	fmt.Fprintf(os.Stderr, "  mukaai gui [options]           Run in GUI mode\n")
	fmt.Fprintf(os.Stderr, "  mukaai help                    Show this help message\n")
	fmt.Fprintf(os.Stderr, "  mukaai version                 Show version information\n")
	fmt.Fprintf(os.Stderr, "\nCLI Mode Options:\n")
	fmt.Fprintf(os.Stderr, "  -c, --config <file>               配置文件路径 (default: ./configs/config.yaml)\n")
	fmt.Fprintf(os.Stderr, "  -t, --task <id>                   继续已有任务ID\n")
	fmt.Fprintf(os.Stderr, "  -w, --workdir <dir>               工作目录\n")
	fmt.Fprintf(os.Stderr, "  -v, --verbose                     详细输出\n")
	fmt.Fprintf(os.Stderr, "  --debug                           调试模式（完整交互输出）\n")
	fmt.Fprintf(os.Stderr, "  --no-supervisor                   禁用监督\n")
	fmt.Fprintf(os.Stderr, "  --max-iterations <n>              最大迭代次数\n")
	fmt.Fprintf(os.Stderr, "\nGUI Mode Options:\n")
	fmt.Fprintf(os.Stderr, "  -c, --config <file>               配置文件路径 (default: ./configs/config.yaml)\n")
	fmt.Fprintf(os.Stderr, "  -w, --workdir <dir>               工作目录\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  # CLI 模式\n")
	fmt.Fprintf(os.Stderr, "  mukaai \"创建一个Hello World程序\"\n")
	fmt.Fprintf(os.Stderr, "  mukaai -c ./config.yaml \"分析项目结构\"\n")
	fmt.Fprintf(os.Stderr, "  mukaai -t task-123 \"继续执行任务\"\n")
	fmt.Fprintf(os.Stderr, "\n  # GUI 模式\n")
	fmt.Fprintf(os.Stderr, "  mukaai gui\n")
	fmt.Fprintf(os.Stderr, "\nBuild:\n")
	fmt.Fprintf(os.Stderr, "  # CLI 构建（默认，不需要前端资源）\n")
	fmt.Fprintf(os.Stderr, "  go build -o mukaai.exe ./cmd/agentplus\n")
	fmt.Fprintf(os.Stderr, "  # GUI 构建（需要 frontend/dist 目录）\n")
	fmt.Fprintf(os.Stderr, "  go build -tags gui -ldflags \"-w -s\" -o mukaai.exe ./cmd/agentplus\n")
}

// parseFlags 解析命令行参数
func parseFlags() *CLIOptions {
	opts := &CLIOptions{}

	flag.StringVar(&opts.ConfigPath, "c", "./configs/config.yaml", "配置文件路径")
	flag.StringVar(&opts.ConfigPath, "config", "./configs/config.yaml", "配置文件路径")
	flag.StringVar(&opts.TaskID, "t", "", "继续已有任务ID")
	flag.StringVar(&opts.TaskID, "task", "", "继续已有任务ID")
	flag.StringVar(&opts.WorkDir, "w", "", "工作目录")
	flag.StringVar(&opts.WorkDir, "workdir", "", "工作目录")
	flag.BoolVar(&opts.Verbose, "v", false, "详细输出")
	flag.BoolVar(&opts.Verbose, "verbose", false, "详细输出")
	flag.BoolVar(&opts.Debug, "debug", false, "调试模式（输出完整交互信息）")
	flag.BoolVar(&opts.NoSupervisor, "no-supervisor", false, "禁用监督")
	flag.IntVar(&opts.MaxIterations, "max-iterations", 0, "最大迭代次数")

	flag.Usage = func() {
		printUsage()
	}

	flag.Parse()

	// 获取非flag参数作为任务目标
	args := flag.Args()
	if len(args) > 0 {
		opts.TaskGoal = strings.Join(args, " ")
	}

	return opts
}

// initModelClient 初始化模型客户端
func initModelClient(cfg *config.Config) (*model.Client, error) {
	modelCfg := &model.Config{
		Endpoint:    cfg.Model.Endpoint,
		APIKey:      cfg.Model.APIKey,
		ModelName:   cfg.Model.ModelName,
		ContextSize: cfg.Model.ContextSize,
	}

	return model.NewClient(modelCfg)
}

// initToolRegistry 初始化工具注册中心
// workDir: 工作目录
// allowCommands: 允许执行的命令白名单，为空时不做限制
func initToolRegistry(workDir string, allowCommands []string) (*tools.ToolRegistry, error) {
	registry := tools.NewToolRegistry()

	// 注册文件系统工具（带工作目录限制）
	if err := tools.RegisterFilesystemToolsWithWorkDir(registry, workDir); err != nil {
		return nil, fmt.Errorf("failed to register filesystem tools: %w", err)
	}

	// 注册命令执行工具（带安全审查系统）
	if err := tools.RegisterCommandToolsWithSecurity(registry, allowCommands, workDir, func(command, reason string) bool {
		fmt.Printf("\n\033[33m⚠ 命令安全确认\033[0m\n")
		fmt.Printf("  命令: %s\n", command)
		fmt.Printf("  原因: %s\n", reason)
		fmt.Printf("  是否允许执行? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		return input == "y" || input == "yes"
	}); err != nil {
		return nil, fmt.Errorf("failed to register command tools: %w", err)
	}

	// 注册状态工具（complete_task, update_state, fail_task, end_exploration）
	// 注意：状态工具需要校验器回调，这里暂时使用一个简单的校验器
	verifierFunc := func(ctx context.Context, taskGoal string, workDir string) *tools.VerifyResult {
		// 任务完成校验由Agent核心的verifyTaskCompletion负责
		// 工具层校验器只做基础检查（当前为pass-through）
		// 两层校验的分工：
		//   - 工具层：简单校验（目前pass-through，可扩展）
		//   - Agent核心层：真正的文件存在性、内容完整性校验
		return &tools.VerifyResult{
			Status:  "pass",
			Summary: "工具层基础校验通过，详细校验由Agent核心执行",
		}
	}
	tools.RegisterStateToolsWithVerifier(registry, verifierFunc)

	// 注册Git工具（git_status, git_diff, git_commit, git_log, git_add）
	if err := git.RegisterGitTools(registry, workDir); err != nil {
		return nil, fmt.Errorf("failed to register git tools: %w", err)
	}

	return registry, nil
}

// setupCallbacks 设置Agent回调
func setupCallbacks(ag *agent.Agent, verbose bool) {
	// 流式输出回调（正文内容）
	ag.SetOnStreamChunk(func(chunk string) {
		fmt.Print(chunk)
	})

	// 思考内容回调（reasoning_content 和 <thinking> 标签中的内容，灰色显示）
	ag.SetOnThinking(func(thinking string) {
		fmt.Printf("\033[90m%s\033[0m", thinking)
	})

	// 工具调用回调
	ag.SetOnToolCall(func(name, args string) {
		if verbose {
			fmt.Printf("\n[Tool] %s(%s)\n", name, truncateString(args, 100))
		} else {
			fmt.Printf("\n[Tool] %s\n", name)
		}
	})

	// 迭代回调
	ag.SetOnIteration(func(iteration int) {
		if verbose {
			fmt.Printf("\n[Iteration %d]\n", iteration)
		}
	})
}

// setupDebugCallbacks 设置调试模式回调（输出完整交互信息）
func setupDebugCallbacks(ag *agent.Agent) {
	// 流式输出
	ag.SetOnStreamChunk(func(chunk string) {
		fmt.Print(chunk)
	})

	// 完整工具调用（参数完整输出）
	ag.SetOnToolCallFull(func(toolCallID, name, args string) {
		fmt.Printf("\n\033[36m─── Tool Call ───\033[0m\n")
		fmt.Printf("\033[36m  ID:   %s\033[0m\n", toolCallID)
		fmt.Printf("\033[36m  Name: %s\033[0m\n", name)
		// 格式化JSON参数
		var prettyArgs bytes.Buffer
		if json.Indent(&prettyArgs, []byte(args), "  ", "  ") == nil {
			fmt.Printf("\033[36m  Args:\033[0m\n  %s\n", prettyArgs.String())
		} else {
			fmt.Printf("\033[36m  Args: %s\033[0m\n", args)
		}
	})

	// 旧回调也保留（debug模式下不输出，避免重复）
	ag.SetOnToolCall(func(name, args string) {})

	// 工具执行结果
	ag.SetOnToolResult(func(name, resultJSON string) {
		fmt.Printf("\n\033[32m─── Tool Result: %s ───\033[0m\n", name)
		var prettyResult bytes.Buffer
		if json.Indent(&prettyResult, []byte(resultJSON), "  ", "  ") == nil {
			// 限制输出长度
			output := prettyResult.String()
			runes := []rune(output)
			if len(runes) > 500 {
				output = string(runes[:500]) + "\n  ... (truncated)"
			}
			fmt.Printf("\033[32m  %s\033[0m\n", output)
		} else {
			fmt.Printf("\033[32m  %s\033[0m\n", resultJSON)
		}
	})

	// 审查结果
	ag.SetOnReview(func(status, summary string) {
		color := "\033[33m" // yellow
		icon := "⚠"
		switch status {
		case "pass":
			color = "\033[32m" // green
			icon = "✓"
		case "block":
			color = "\033[31m" // red
			icon = "✗"
		}
		fmt.Printf("\n%s─── Review %s ───%s\n", color, icon, "\033[0m")
		fmt.Printf("%s  Status:  %s%s\n", color, status, "\033[0m")
		fmt.Printf("%s  Summary: %s%s\n", color, summary, "\033[0m")
	})

	// 校验结果
	ag.SetOnVerify(func(status, summary string) {
		color := "\033[32m" // green
		icon := "✓"
		if status == "fail" {
			color = "\033[31m" // red
			icon = "✗"
		}
		fmt.Printf("\n%s─── Verify %s ───%s\n", color, icon, "\033[0m")
		fmt.Printf("%s  Status:  %s%s\n", color, status, "\033[0m")
		fmt.Printf("%s  Summary: %s%s\n", color, summary, "\033[0m")
	})

	// 修正指令
	ag.SetOnCorrection(func(instruction string) {
		fmt.Printf("\n\033[35m─── Correction ───\033[0m\n")
		runes := []rune(instruction)
		if len(runes) > 300 {
			fmt.Printf("\033[35m  %s...\033[0m\n", string(runes[:300]))
		} else {
			fmt.Printf("\033[35m  %s\033[0m\n", instruction)
		}
	})

	// 无工具调用
	ag.SetOnNoToolCall(func(count int, response string) {
		fmt.Printf("\n\033[33m─── No Tool Call (consecutive: %d) ───\033[0m\n", count)
	})

	// 消息历史添加
	ag.SetOnHistoryAdd(func(role, content string) {
		// 只在debug模式下输出简要信息
		runes := []rune(content)
		preview := content
		if len(runes) > 80 {
			preview = string(runes[:80]) + "..."
		}
		preview = strings.ReplaceAll(preview, "\n", " ")
		fmt.Printf("\n\033[90m─── History [%s]: %s ───\033[0m\n", role, preview)
	})

	// 迭代
	ag.SetOnIteration(func(iteration int) {
		fmt.Printf("\n\033[34m═══════════ Iteration %d ═══════════\033[0m\n", iteration)
	})

	// 思考内容回调（reasoning_content 和 <thinking> 标签中的内容）
	// 使用缓冲区累积thinking内容，以完整块输出避免颜色码闪烁
	thinkingStarted := false
	ag.SetOnThinking(func(thinking string) {
		if !thinkingStarted {
			fmt.Print("\033[90m") // 进入灰色
			thinkingStarted = true
		}
		fmt.Print(thinking)
		// 检测thinking内容是否结束（遇到换行后紧跟非空白内容的模式）
		// 简单策略：如果thinking内容以换行结尾，可能结束了
		if strings.HasSuffix(thinking, "\n\n") || thinking == "" {
			fmt.Print("\033[0m") // 退出灰色
			thinkingStarted = false
		}
	})
}

// interactiveInput 交互式输入任务
func interactiveInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s v%s - AI Agent CLI Tool\n", Name, Version)
	fmt.Println("Enter your task (type /help for commands, empty line to submit):")
	fmt.Println(strings.Repeat("-", 60))

	var lines []string
	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		line = strings.TrimSpace(line)

		// 处理特殊命令
		if strings.HasPrefix(line, "/") {
			cmd := strings.ToLower(line)
			switch cmd {
			case "/help":
				printInteractiveHelp()
				continue
			case "/quit", "/exit":
				return "", nil
			case "/status":
				fmt.Println("Status: No active task")
				continue
			case "/clear":
				lines = nil
				fmt.Println("Input cleared")
				continue
			default:
				fmt.Printf("Unknown command: %s (type /help for available commands)\n", cmd)
				continue
			}
		}

		// 空行表示输入结束
		if line == "" {
			break
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n"), nil
}

// printInteractiveHelp 打印交互式帮助
func printInteractiveHelp() {
	fmt.Print(`
Interactive Commands:
  /help     显示此帮助信息
  /quit     退出程序
  /status   显示当前任务状态
  /clear    清除当前输入

Input Tips:
  - 输入多行任务描述，以空行结束
  - 任务描述应清晰明确，包含具体目标
  - 可以指定文件路径、技术要求等细节

Examples:
  > 创建一个Go语言的HTTP服务器
  > 监听8080端口
  > 提供/hello端点返回Hello World
  >
  (空行提交任务)

`)
}

// printResult 打印执行结果
func printResult(result *agent.RunResult) {
	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Println("Task Execution Result:")
	fmt.Printf("  Task ID:    %s\n", result.TaskID)
	fmt.Printf("  Status:     %s\n", result.Status)
	fmt.Printf("  Iterations: %d\n", result.Iterations)
	fmt.Printf("  Duration:   %v\n", result.EndTime.Sub(result.StartTime).Round(time.Millisecond))

	if result.Error != "" {
		fmt.Printf("  Error:      %s\n", result.Error)
	}

	if result.FinalResponse != "" {
		fmt.Println("\nFinal Response:")
		fmt.Println(result.FinalResponse)
	}

	fmt.Println(strings.Repeat("-", 60))
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// getWorkDir 获取工作目录
func getWorkDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}

// init 初始化
func init() {
	// 确保工作目录正确
	if err := os.Chdir(getWorkDir()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to change to work directory: %v\n", err)
	}
}

// initMCPManager 初始化MCP客户端管理器
func initMCPManager(cfg *config.Config, registry *tools.ToolRegistry, workDir string) *mcp.MCPClientManager {
	// 将config包的MCP配置转换为mcp包的配置
	mcpConfig := &mcp.MCPConfig{
		Enabled: cfg.MCP.Enabled,
		Security: mcp.MCPSecurityConfig{
			DefaultPolicy: cfg.MCP.Security.DefaultPolicy,
			DenyTools:     cfg.MCP.Security.DenyTools,
			ConfirmTools:  cfg.MCP.Security.ConfirmTools,
			AllowTools:    cfg.MCP.Security.AllowTools,
			MaxTools:      cfg.MCP.Security.MaxTools,
		},
	}

	// 转换Server配置
	for _, s := range cfg.MCP.Servers {
		projectPath := s.ProjectPath
		if projectPath == "" {
			projectPath = workDir // 使用workDir作为默认projectPath
		}
		mcpConfig.Servers = append(mcpConfig.Servers, mcp.ServerConfig{
			ID:          s.ID,
			Enabled:     s.Enabled,
			Transport:   s.Transport,
			Command:     s.Command,
			Args:        s.Args,
			Env:         s.Env,
			URL:         s.URL,
			Headers:     s.Headers,
			Timeout:     s.Timeout,
			ProjectPath: projectPath,
		})
	}

	// 验证配置
	if err := mcpConfig.Validate(); err != nil {
		log.Printf("[MCP] 配置验证失败: %v", err)
		return nil
	}

	manager := mcp.NewMCPClientManager(mcpConfig, registry)

	// 异步初始化MCP连接，不阻塞应用启动
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := manager.Initialize(ctx); err != nil {
			log.Printf("[MCP] 初始化出错: %v", err)
		}
	}()

	return manager
}

// initLSPManager 初始化LSP管理器并注册诊断工具
func initLSPManager(cfg *config.Config, registry *tools.ToolRegistry, workDir string) *lsp.LSPManager {
	manager := lsp.NewManager(workDir)

	// 配置语言服务器
	if len(cfg.LSP.Servers) > 0 {
		configs := make(map[string]lsp.LanguageServerConfig)
		for lang, entry := range cfg.LSP.Servers {
			configs[lang] = lsp.LanguageServerConfig{
				Language: lang,
				Command:  entry.Command,
				Args:     entry.Args,
			}
		}
		manager.ConfigureFromMap(configs)
	} else {
		// 使用默认配置
		manager.ConfigureFromMap(lsp.DefaultConfigs())
	}

	// 注册LSP诊断工具到工具注册中心
	if err := lsp.RegisterLSPTools(registry, manager, workDir); err != nil {
		log.Printf("[LSP] 注册诊断工具失败: %v", err)
		return nil
	}

	log.Printf("[LSP] 已启用，支持Go/TypeScript/Python代码诊断")
	return manager
}
