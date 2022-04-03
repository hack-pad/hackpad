// +build js

package process

import "syscall/js"

const (
	userID  = 0
	groupID = 0
)

func (s processShim) geteuid(args []js.Value) (interface{}, error) {
	return userID, nil
}

func (s processShim) getegid(args []js.Value) (interface{}, error) {
	return groupID, nil
}

func (s processShim) getgroups(args []js.Value) (interface{}, error) {
	return groupID, nil
}
