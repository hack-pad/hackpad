//go:build js
// +build js

package global

import "syscall/js"

const globalKey = "hackpad"

var globals js.Value

func init() {
	global := js.Global()
	if globals.IsUndefined() {
		globals = global.Get(globalKey)
	}
	if !globals.IsUndefined() {
		return
	}
	global.Set(globalKey, map[string]interface{}{})
	globals = global.Get(globalKey)
}

func SetDefault(key string, value interface{}) {
	if globals.Get(key).IsUndefined() {
		globals.Set(key, value)
	}
}

func Set(key string, value interface{}) {
	globals.Set(key, value)
}

func Get(key string) js.Value {
	return globals.Get(key)
}
