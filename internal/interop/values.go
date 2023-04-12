//go:build js
// +build js

package interop

import "syscall/js"

var jsObject = js.Global().Get("Object")

func SliceFromStrings(strs []string) js.Value {
	var values []interface{}
	for _, s := range strs {
		values = append(values, s)
	}
	return js.ValueOf(values)
}

func StringsFromJSValue(value js.Value) []string {
	var strs []string
	length := value.Length()
	for i := 0; i < length; i++ {
		strs = append(strs, value.Index(i).String())
	}
	return strs
}

func SliceFromJSValue(value js.Value) []js.Value {
	var values []js.Value
	length := value.Length()
	for i := 0; i < length; i++ {
		values = append(values, value.Index(i))
	}
	return values
}

func SliceFromJSValues(args []js.Value) []interface{} {
	var values []interface{}
	for _, arg := range args {
		values = append(values, arg)
	}
	return values
}

func Keys(value js.Value) []string {
	jsKeys := jsObject.Call("keys", value)
	length := jsKeys.Length()
	var keys []string
	for i := 0; i < length; i++ {
		keys = append(keys, jsKeys.Index(i).String())
	}
	return keys
}

func Entries(value js.Value) map[string]js.Value {
	entries := make(map[string]js.Value)
	for _, key := range Keys(value) {
		entries[key] = value.Get(key)
	}
	return entries
}

func StringMap(m map[string]string) js.Value {
	jsValue := make(map[string]interface{})
	for key, value := range m {
		jsValue[key] = value
	}
	return js.ValueOf(jsValue)
}
