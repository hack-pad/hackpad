// +build js

package fs

import (
	"syscall"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/pkg/errors"
)

func (s fileShim) flock(args []js.Value) ([]interface{}, error) {
	_, err := s.flockSync(args)
	return nil, err
}

func (s fileShim) flockSync(args []js.Value) (interface{}, error) {
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

	return nil, s.Flock(fid, action, shouldLock)
}

func (s fileShim) Flock(fid common.FID, action fs.LockAction, shouldLock bool) error {
	return s.process.Files().Flock(fid, action)
}
