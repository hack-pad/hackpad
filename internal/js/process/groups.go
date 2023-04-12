//go:build js
// +build js

package process

import "syscall/js"

const (
	userID  = 0
	groupID = 0
)

func geteuid(args []js.Value) (interface{}, error) {
	return userID, nil
}

func getegid(args []js.Value) (interface{}, error) {
	return groupID, nil
}

func getgroups(args []js.Value) (interface{}, error) {
	return groupID, nil
}
