package interop

import "syscall/js"

var initObj = js.Global().Get("goWasmInitialized")

func SetInitialized(name string) {
	initObj.Set(name, true)
}

func IsInitialized(name string) bool {
	val := initObj.Get(name)
	switch val.Type() {
	case js.TypeBoolean:
		return val.Bool()
	default:
		return false
	}
}
