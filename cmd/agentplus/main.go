// Package main 提供AgentPlus命令行入口
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"agentplus/internal/agent"
	"agentplus/internal/config"
	"agentplus/internal/model"
	"agentplus/internal/state"
	"agentplus/internal/tools"
)

// 版本信息
const (
	Version = "1.0.0"
	Name    = "AgentPlus"
)

// 命令行参数
type CLIOptions struct {
	ConfigPath    string
	TaskID        string
	WorkDir       string
	Verbose       bool
	NoSupervisor  bool
	MaxIterations int
	TaskGoal      string
}

func main() {
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

	toolRegistry, err := initToolRegistry(workDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing tool registry: %v\n", err)
		os.Exit(1)
	}

	stateManager, err := state.NewStateManager(stateDir, cfg.State.AutoSave)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing state manager: %v\n", err)
		os.Exit(1)
	}

	// 创建Agent
	ag, err := agent.NewAgent(&agent.Config{
		ModelClient:   modelClient,
		ToolRegistry:  toolRegistry,
		StateManager:  stateManager,
		MaxIterations: cfg.Agent.MaxIterations,
		PromptType:    agent.PromptTypeOrchestrator,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating agent: %v\n", err)
		os.Exit(1)
	}

	// 设置任务ID（如果指定）
	if opts.TaskID != "" {
		ag.SetTaskID(opts.TaskID)
	}

	// 设置回调
	setupCallbacks(ag, opts.Verbose)

	// 创建上下文，支持优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, stopping...")
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
	flag.BoolVar(&opts.NoSupervisor, "no-supervisor", false, "禁用监督")
	flag.IntVar(&opts.MaxIterations, "max-iterations", 0, "最大迭代次数")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s v%s - AI Agent CLI Tool\n\n", Name, Version)
		fmt.Fprintf(os.Stderr, "Usage: agentplus [options] <task>\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nInteractive Commands:\n")
		fmt.Fprintf(os.Stderr, "  /help     显示帮助信息\n")
		fmt.Fprintf(os.Stderr, "  /status   显示当前任务状态\n")
		fmt.Fprintf(os.Stderr, "  /pause    暂停任务\n")
		fmt.Fprintf(os.Stderr, "  /resume   恢复任务\n")
		fmt.Fprintf(os.Stderr, "  /quit     退出程序\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  agentplus \"创建一个Hello World程序\"\n")
		fmt.Fprintf(os.Stderr, "  agentplus -c ./config.yaml \"分析项目结构\"\n")
		fmt.Fprintf(os.Stderr, "  agentplus -t task-123 \"继续执行任务\"\n")
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
func initToolRegistry(workDir string) (*tools.ToolRegistry, error) {
	registry := tools.NewToolRegistry()

	// 注册文件系统工具
	if err := tools.RegisterFilesystemTools(registry); err != nil {
		return nil, fmt.Errorf("failed to register filesystem tools: %w", err)
	}

	// 注册命令执行工具
	if err := tools.RegisterCommandTools(registry); err != nil {
		return nil, fmt.Errorf("failed to register command tools: %w", err)
	}

	return registry, nil
}

// setupCallbacks 设置Agent回调
func setupCallbacks(ag *agent.Agent, verbose bool) {
	// 流式输出回调
	ag.SetOnStreamChunk(func(chunk string) {
		fmt.Print(chunk)
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
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
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
