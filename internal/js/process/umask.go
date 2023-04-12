//go:build js
// +build js

package process

import "syscall/js"

var currentUMask = 0755

func umask(args []js.Value) (interface{}, error) {
	if len(args) == 0 {
		return currentUMask, nil
	}
	oldUMask := currentUMask
	currentUMask = args[0].Int()
	return oldUMask, nil
}
