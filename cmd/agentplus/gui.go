//go:build gui

// Package main 提供MukaAI的GUI模式入口
// 此文件仅在 gui 构建标签下编译，包含Wails GUI相关的初始化和启动逻辑。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Attect/MukaAI"
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

// GUIOptions GUI模式的命令行参数
type GUIOptions struct {
	ConfigPath string
	WorkDir    string
}

// runGUICommand 运行 GUI 模式
// 加载配置、初始化Agent和工具、创建Wails应用并启动
func runGUICommand() {
	// 解析GUI子命令的参数（os.Args[2:]为gui之后的参数）
	opts := parseGUIFlags()

	// 加载配置（与CLI模式相同的配置加载逻辑）
	cfg, err := config.LoadConfig(opts.ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 命令行参数覆盖配置中的工作目录
	if opts.WorkDir != "" {
		cfg.Tools.WorkDir = opts.WorkDir
	}

	// 获取绝对工作目录
	workDir, err := cfg.GetAbsoluteWorkDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取工作目录失败: %v\n", err)
		os.Exit(1)
	}

	// 获取绝对状态目录
	stateDir, err := cfg.GetAbsoluteStateDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取状态目录失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化模型客户端
	modelClient, err := initModelClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化模型客户端失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化工具注册中心
	toolRegistry, err := initToolRegistry(workDir, cfg.Tools.AllowCommands, stateDir, modelClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化工具注册中心失败: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "初始化状态管理器失败: %v\n", err)
		os.Exit(1)
	}

	// 创建Agent实例
	ag, err := agent.NewAgent(&agent.Config{
		ModelClient:   modelClient,
		ToolRegistry:  toolRegistry,
		StateManager:  stateManager,
		MaxIterations: cfg.Agent.MaxIterations,
		PromptType:    agent.PromptTypeOrchestrator,
		WorkDir:       workDir,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建Agent失败: %v\n", err)
		os.Exit(1)
	}

	// 创建Supervisor监督器（GUI模式默认启用）
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

	// 创建GUI App实例
	app := gui.NewApp()
	app.SetAgent(ag)
	app.SetConfigPath(opts.ConfigPath)

	// 创建终端管理器和 WebSocket 服务器
	terminalManager := terminal.NewTerminalManager("", workDir)
	wsServer := terminal.NewWebSocketServer(terminalManager)

	// 注册终端工具到工具注册中心
	if err := terminal.RegisterTerminalTools(toolRegistry, terminalManager); err != nil {
		log.Printf("Warning: failed to register terminal tools: %v", err)
	}

	// 创建StreamBridge，将Agent流式事件桥接到Wails前端事件系统
	bridge := gui.NewStreamBridge(app)
	ag.SetStreamHandler(bridge)

	// 启动Wails应用
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
			app.Startup(ctx)
			bridge.SetContext(ctx)
			app.SetCurrentDir(workDir)
			// 启动状态文件自动清理，传入应用上下文
			stateManager.StartCleanup(ctx)
			// 启动终端 WebSocket 服务器
			if err := wsServer.Start(); err != nil {
				log.Printf("Warning: failed to start terminal WebSocket server: %v", err)
			} else {
				app.SetTerminalWSUrl(wsServer.GetWSUrl())
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
		// 即使出错也要停止清理goroutine
		stateManager.StopCleanup()
		wsServer.Stop()
		terminalManager.Stop()
		fmt.Fprintf(os.Stderr, "启动GUI失败: %v\n", err)
		os.Exit(1)
	}

	// 正常退出时停止清理goroutine
	stateManager.StopCleanup()
	wsServer.Stop()
	terminalManager.Stop()
}

// parseGUIFlags 解析GUI子命令的命令行参数
// 从os.Args[2:]中解析，因为os.Args[1]是"gui"子命令
func parseGUIFlags() *GUIOptions {
	opts := &GUIOptions{}

	guiFlagSet := flag.NewFlagSet("gui", flag.ExitOnError)
	guiFlagSet.StringVar(&opts.ConfigPath, "c", "./configs/config.yaml", "配置文件路径")
	guiFlagSet.StringVar(&opts.ConfigPath, "config", "./configs/config.yaml", "配置文件路径")
	guiFlagSet.StringVar(&opts.WorkDir, "w", "", "工作目录")
	guiFlagSet.StringVar(&opts.WorkDir, "workdir", "", "工作目录")

	guiFlagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "MukaAI GUI Mode\n\n")
		fmt.Fprintf(os.Stderr, "Usage: mukaai gui [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		guiFlagSet.PrintDefaults()
	}

	guiFlagSet.Parse(os.Args[2:])

	return opts
}
