//go:build gui

// Package main 提供 MukaAI 的 GUI 模式入口
// 此文件仅在 gui 构建标签下编译，包含 Wails GUI 相关的初始化和启动逻辑。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	mukaai "github.com/Attect/MukaAI"
	"github.com/Attect/MukaAI/internal/agent"
	"github.com/Attect/MukaAI/internal/config"
	ctxpkg "github.com/Attect/MukaAI/internal/context"
	"github.com/Attect/MukaAI/internal/gui"
	"github.com/Attect/MukaAI/internal/state"
	"github.com/Attect/MukaAI/internal/supervisor"
	"github.com/Attect/MukaAI/internal/terminal"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

// GUIOptions GUI 模式的命令行参数
type GUIOptions struct {
	ConfigPath string
	WorkDir    string
	// 新增字段：初始任务支持
	InitialTask string // 初始任务描述
	LogPath     string // 日志文件路径
	AutoSend    bool   // 是否自动发送初始任务
}

// runGUICommand 运行 GUI 模式
// 加载配置、初始化 Agent 和工具、创建 Wails 应用并启动
func runGUICommand() {
	// 在 WebView2 初始化之前设置环境变量，启用无障碍支持
	// --force-renderer-accessibility: 强制WebView2渲染进程创建无障碍节点，暴露给Windows UI Automation
	enableWebView2Accessibility()

	// 解析 GUI 子命令的参数（os.Args[2:]为 gui 之后的参数）
	opts := parseGUIFlags()

	// 加载配置（与 CLI 模式相同的配置加载逻辑）
	cfg, err := config.LoadConfig(opts.ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败：%v\n", err)
		os.Exit(1)
	}

	// 命令行参数覆盖配置中的工作目录
	if opts.WorkDir != "" {
		cfg.Tools.WorkDir = opts.WorkDir
	}

	// 获取绝对工作目录
	workDir, err := cfg.GetAbsoluteWorkDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取工作目录失败：%v\n", err)
		os.Exit(1)
	}

	// 获取绝对状态目录
	stateDir, err := cfg.GetAbsoluteStateDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取状态目录失败：%v\n", err)
		os.Exit(1)
	}

	// 初始化模型客户端
	modelClient, err := initModelClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化模型客户端失败：%v\n", err)
		os.Exit(1)
	}

	// 初始化工具注册中心
	toolRegistry, err := initToolRegistry(workDir, cfg.Tools.AllowCommands, stateDir, modelClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化工具注册中心失败：%v\n", err)
		os.Exit(1)
	}

	// 初始化状态管理器（带自动清理功能）
	cleanupConfig := state.CleanupConfig{
		RetentionDays: cfg.State.CleanupDays,
		CheckInterval: 24 * time.Hour,
		Enabled:       cfg.State.CleanupEnable,
	}
	stateManager, err := state.NewStateManagerWithCleanup(stateDir, cfg.State.AutoSave, cleanupConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化状态管理器失败：%v\n", err)
		os.Exit(1)
	}

	// 创建 Agent 实例
	ag, err := agent.NewAgent(&agent.Config{
		ModelClient:   modelClient,
		ToolRegistry:  toolRegistry,
		StateManager:  stateManager,
		MaxIterations: cfg.Agent.MaxIterations,
		PromptType:    agent.PromptTypeOrchestrator,
		WorkDir:       workDir,
		PromptsPath:   workDir + "/prompts", // 提示词文件目录
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建 Agent 失败：%v\n", err)
		os.Exit(1)
	}

	// 创建 Supervisor 监督器（GUI 模式默认启用）
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
		ag.SetSupervisor(supInstance)
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
	injector := ctxpkg.NewInjectorFromContextSize(indexer, cfg.Model.ContextSize)
	ag.SetContextInjector(injector)

	// 创建 GUI App 实例
	app := gui.NewApp()
	app.SetAgent(ag)
	app.SetConfigPath(opts.ConfigPath)
	// 设置工具工作目录更新回调，确保切换工作目录时所有工具的workDir同步更新
	app.SetToolWorkDirUpdater(func(workDir string) {
		toolRegistry.UpdateAllToolWorkDirs(workDir)
	})

	// 设置命令行参数传递的初始任务和日志配置
	if opts.InitialTask != "" {
		log.Printf("Initial task configured: %s", opts.InitialTask)
	}
	if opts.LogPath != "" {
		log.Printf("Log path configured: %s", opts.LogPath)
	}
	if opts.AutoSend {
		log.Printf("Auto-send enabled")
	}

	// 创建终端管理器和 WebSocket 服务器
	terminalManager := terminal.NewTerminalManager("", workDir)
	wsServer := terminal.NewWebSocketServer(terminalManager)

	// 注册终端工具到工具注册中心
	if err := terminal.RegisterTerminalTools(toolRegistry, terminalManager); err != nil {
		log.Printf("Warning: failed to register terminal tools: %v", err)
	}

	// 创建 StreamBridge，将 Agent 流式事件桥接到 Wails 前端事件系统
	bridge := gui.NewStreamBridge(app)
	ag.SetStreamHandler(bridge)

	// 启动 Wails 应用
	if err := wails.Run(&options.App{
		Title:     "MukaAI",
		Width:     1024,
		Height:    768,
		MinWidth:  640,
		MinHeight: 480,
		AssetServer: &assetserver.Options{
			Assets: mukaai.FrontendAssets,
		},
		OnStartup: func(ctx context.Context) {
			// 先设置工作目录，再启动（确保对话存储使用正确路径）
			app.SetCurrentDir(workDir)
			app.Startup(ctx)
			bridge.SetContext(ctx)
			// 启动状态文件自动清理，传入应用上下文
			stateManager.StartCleanup(ctx)
			// 启动终端 WebSocket 服务器
			if err := wsServer.Start(); err != nil {
				log.Printf("Warning: failed to start terminal WebSocket server: %v", err)
			} else {
				app.SetTerminalWSUrl(wsServer.GetWSUrl())
			}

			// 如果配置了自动发送初始任务，在应用初始化后执行
			if opts.AutoSend && opts.InitialTask != "" {
				go func() {
					// 等待一小段时间确保 GUI 完全加载
					time.Sleep(2 * time.Second)

					// 发送初始任务
					if err := app.SendMessage(opts.InitialTask); err != nil {
						log.Printf("Failed to send initial task: %v", err)
						return
					}

					log.Printf("自动发送初始任务：%s", opts.InitialTask)
				}()
			}
		},
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	}); err != nil {
		// 即使出错也要停止清理 goroutine
		stateManager.StopCleanup()
		wsServer.Stop()
		terminalManager.Stop()
		fmt.Fprintf(os.Stderr, "启动 GUI 失败：%v\n", err)
		os.Exit(1)
	}

	// 正常退出时停止清理 goroutine
	stateManager.StopCleanup()
	wsServer.Stop()
	terminalManager.Stop()
}

// enableWebView2Accessibility 配置WebView2的无障碍支持
// 必须在Wails运行前调用，因为WebView2在首次创建时读取此环境变量
// 启用后：Windows UI Automation可以访问WebView2内部的DOM无障碍树
// 注意：WebView2嵌入模式下不支持--remote-debugging-port，该标志会被忽略
func enableWebView2Accessibility() {
	existing := os.Getenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS")
	args := []string{}

	// 保留已有的浏览器参数
	if existing != "" {
		args = append(args, existing)
	}

	// 强制渲染进程创建无障碍节点，使Windows UI Automation能访问WebView2内部内容
	// 这是使自动化工具（如Windows MCP）能够识别和操作WebView2内部元素的关键
	args = append(args, "--force-renderer-accessibility")

	newArgs := strings.Join(args, " ")
	os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", newArgs)
	log.Printf("[GUI] WebView2 browser arguments: %s", newArgs)
}

// runDefaultCommand GUI构建默认运行GUI模式
func runDefaultCommand() {
	runGUICommand()
}

// parseGUIFlags 解析 GUI 子命令的命令行参数
// 支持两种调用方式：
// 1. mukaai-gui.exe [options] - 直接运行GUI
// 2. mukaai gui [options] - 通过gui子命令运行
func parseGUIFlags() *GUIOptions {
	opts := &GUIOptions{}

	guiFlagSet := flag.NewFlagSet("gui", flag.ExitOnError)
	guiFlagSet.StringVar(&opts.ConfigPath, "c", "./configs/config.yaml", "配置文件路径")
	guiFlagSet.StringVar(&opts.ConfigPath, "config", "./configs/config.yaml", "配置文件路径")
	guiFlagSet.StringVar(&opts.WorkDir, "w", "", "工作目录")
	guiFlagSet.StringVar(&opts.WorkDir, "workdir", "", "工作目录")

	// 新增：初始任务相关参数
	guiFlagSet.StringVar(&opts.InitialTask, "task", "", "初始任务描述")
	guiFlagSet.StringVar(&opts.LogPath, "log-path", "", "日志文件路径（默认：ai/logs/gui-log-时间戳.log）")
	guiFlagSet.BoolVar(&opts.AutoSend, "auto-send", false, "启动后自动发送初始任务")

	guiFlagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "MukaAI GUI Mode\n\n")
		fmt.Fprintf(os.Stderr, "Usage: mukaai-gui [options]\n")
		fmt.Fprintf(os.Stderr, "       mukaai gui [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		guiFlagSet.PrintDefaults()
	}

	// 判断调用方式：如果第一个参数是"gui"子命令，则从os.Args[2:]解析
	// 否则从os.Args[1:]解析
	var argsToParse []string
	if len(os.Args) > 1 && os.Args[1] == "gui" {
		argsToParse = os.Args[2:]
	} else {
		argsToParse = os.Args[1:]
	}
	guiFlagSet.Parse(argsToParse)

	return opts
}
