//go:build gui

// Package agentplus 提供前端资源的嵌入支持
// 由于Go的embed指令要求路径相对于源文件目录且不允许..路径，
// 前端资源只能从项目根目录嵌入，因此在此处定义
//
// 此文件仅在 gui 构建标签下编译，需要 frontend/dist 目录存在。
// CLI构建时使用 frontend_assets_stub.go 替代。
package agentplus

import "embed"

// FrontendAssets 嵌入的前端构建产物
// 在开发模式下(wails dev)，Vite开发服务器提供资源，此嵌入不生效
// 在生产构建时(wails build)，前端资源被打包进二进制文件
//
//go:embed all:frontend/dist
var FrontendAssets embed.FS
