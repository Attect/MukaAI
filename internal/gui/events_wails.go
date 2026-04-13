//go:build gui

package gui

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Emit 发射Wails前端事件
// 仅在gui构建标签下编译，直接调用 runtime.EventsEmit
func (e *WailsEventEmitter) Emit(event string, data ...interface{}) {
	runtime.EventsEmit(e.ctx, event, data...)
}
