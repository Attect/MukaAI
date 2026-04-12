//go:build !gui

// Package agentplus 提供前端资源的嵌入支持
//
// 此文件为CLI构建模式下的桩实现，不嵌入任何前端资源。
// 当使用 gui 构建标签时，frontend_assets.go 中的真实实现会替代此文件。
package agentplus

import "embed"

// FrontendAssets 前端资源的空实现
// 在CLI构建模式下为零值embed.FS，不会被使用。
// GUI构建模式下由 frontend_assets.go 提供真实的嵌入资源。
var FrontendAssets embed.FS
