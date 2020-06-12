package global

import "syscall/js"

const globalKey = "goWasm"

var globals js.Value

func init() {
	global := js.Global()
	if global.Get(globalKey).Truthy() {
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
