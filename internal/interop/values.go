package interop

import "syscall/js"

func SliceFromStrings(strs []string) js.Value {
	var values []interface{}
	for _, s := range strs {
		values = append(values, s)
	}
	return js.ValueOf(values)
}
