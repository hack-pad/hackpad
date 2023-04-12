//go:build js
// +build js

package dom

import (
	"syscall/js"
	"time"

	"github.com/hack-pad/hackpad/internal/interop"
)

var (
	window = NewFromJS(js.Global())
)

func SetTimeout(fn func(args []js.Value), delay time.Duration, args ...js.Value) int {
	intArgs := append([]interface{}{
		interop.SingleUseFunc(func(_ js.Value, args []js.Value) interface{} {
			fn(args)
			return nil
		}),
		delay.Milliseconds(),
	}, interop.SliceFromJSValues(args)...)
	timeoutID := window.elem.Call("setTimeout", intArgs...)
	return timeoutID.Int()
}

func QueueMicrotask(fn func()) {
	queueMicrotask := window.GetProperty("queueMicrotask")
	if queueMicrotask.Truthy() {
		queueMicrotask.Invoke(interop.SingleUseFunc(func(this js.Value, args []js.Value) interface{} {
			fn()
			return nil
		}))
	} else {
		SetTimeout(func(args []js.Value) {
			fn()
		}, 0)
	}
}

func ViewportRect() *Rect {
	width, height := window.GetProperty("innerWidth").Float(), window.GetProperty("innerHeight").Float()
	return &Rect{
		Left:   0,
		Top:    0,
		Right:  width,
		Bottom: height,
		Width:  width,
		Height: height,
	}
}

func Alert(message string) {
	window.elem.Call("alert", message)
}

func Confirm(prompt string) bool {
	return window.elem.Call("confirm", prompt).Bool()
}

func Reload() {
	window.GetProperty("location").Call("reload")
}
