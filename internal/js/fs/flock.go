//go:build js
// +build js

package fs

import (
	"syscall"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

func flock(args []js.Value) ([]interface{}, error) {
	_, err := flockSync(args)
	return nil, err
}

func flockSync(args []js.Value) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("Invalid number of args, expected 2: %v", args)
	}
	fid := common.FID(args[0].Int())
	flag := args[1].Int()
	var action fs.LockAction
	shouldLock := true
	switch flag {
	case syscall.LOCK_EX:
		action = fs.LockExclusive
	case syscall.LOCK_SH:
		action = fs.LockShared
	case syscall.LOCK_UN:
		action = fs.Unlock
	}

	return nil, Flock(fid, action, shouldLock)
}

func Flock(fid common.FID, action fs.LockAction, shouldLock bool) error {
	p := process.Current()
	return p.Files().Flock(fid, action)
}
