//go:build !gui

package gui

import "fmt"

// Emit 桩实现，非gui构建时使用
// 不调用Wails runtime，避免在CLI模式和测试中引入依赖
func (e *WailsEventEmitter) Emit(event string, data ...interface{}) {
	fmt.Printf("[EventEmitter] stub emit: %s %v\n", event, data)
}
