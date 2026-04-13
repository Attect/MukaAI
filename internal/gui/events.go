package gui

import "context"

// EventEmitter 事件发射器抽象接口
// 将 runtime.EventsEmit 的直接调用解耦为可注入的接口，
// 使集成测试能通过 MockEventEmitter 覆盖完整路径。
type EventEmitter interface {
	// Emit 发射事件到前端
	// event: 事件名称
	// data: 可选的事件数据
	Emit(event string, data ...interface{})
}

// WailsEventEmitter 生产环境的Wails事件发射器
// 包装 runtime.EventsEmit 调用，Emit方法实现由构建标签文件提供：
//   - events_wails.go (//go:build gui): 调用 runtime.EventsEmit
//   - events_stub.go (//go:build !gui): 空实现或日志输出
type WailsEventEmitter struct {
	ctx context.Context
}

// NewWailsEventEmitter 创建Wails事件发射器
func NewWailsEventEmitter(ctx context.Context) *WailsEventEmitter {
	return &WailsEventEmitter{ctx: ctx}
}

// noopEventEmitter 空操作事件发射器
// 用于测试中不需要验证事件时的默认注入
type noopEventEmitter struct{}

func (e *noopEventEmitter) Emit(event string, data ...interface{}) {
	// 空操作，不发射任何事件
}
