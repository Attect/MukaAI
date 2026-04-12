//go:build gui

// Package main 提供AgentPlus的GUI模式入口
// 此文件仅在 gui 构建标签下编译，包含Wails GUI相关的初始化和启动逻辑。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"agentplus"
	"agentplus/internal/agent"
	"agentplus/internal/config"
	"agentplus/internal/gui"
	"agentplus/internal/state"

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
	toolRegistry, err := initToolRegistry(workDir, cfg.Tools.AllowCommands)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化工具注册中心失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化状态管理器
	stateManager, err := state.NewStateManager(stateDir, cfg.State.AutoSave)
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

	// 创建GUI App实例
	app := gui.NewApp()
	app.SetAgent(ag)

	// 创建StreamBridge，将Agent流式事件桥接到Wails前端事件系统
	bridge := gui.NewStreamBridge(app)
	ag.SetStreamHandler(bridge)

	// 启动Wails应用
	if err := wails.Run(&options.App{
		Title:     "AgentPlus",
		Width:     1024,
		Height:    768,
		MinWidth:  640,
		MinHeight: 480,
		AssetServer: &assetserver.Options{
			Assets: agentplus.FrontendAssets,
		},
		OnStartup: func(ctx context.Context) {
			app.Startup(ctx)
			bridge.SetContext(ctx)
			app.SetCurrentDir(workDir)
		},
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	}); err != nil {
		fmt.Fprintf(os.Stderr, "启动GUI失败: %v\n", err)
		os.Exit(1)
	}
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
		fmt.Fprintf(os.Stderr, "AgentPlus GUI Mode\n\n")
		fmt.Fprintf(os.Stderr, "Usage: agentplus gui [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		guiFlagSet.PrintDefaults()
	}

	guiFlagSet.Parse(os.Args[2:])

	return opts
}
