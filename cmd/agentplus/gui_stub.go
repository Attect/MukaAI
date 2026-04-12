//go:build !gui

// Package main 提供AgentPlus的GUI模式桩实现
// 此文件在非 gui 构建标签下编译，当用户尝试运行 gui 子命令时给出提示。
package main

import (
	"fmt"
	"os"
)

// runGUICommand GUI模式桩实现
// 当前构建不包含GUI支持，提示用户使用 gui 构建标签重新编译。
func runGUICommand() {
	fmt.Fprintf(os.Stderr, "错误：当前构建不包含GUI模式支持。\n")
	fmt.Fprintf(os.Stderr, "请使用以下命令重新编译以启用GUI模式：\n")
	fmt.Fprintf(os.Stderr, "  go build -tags gui ./cmd/agentplus\n")
	fmt.Fprintf(os.Stderr, "或使用 wails build 进行完整构建。\n")
	os.Exit(1)
}
