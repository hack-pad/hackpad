// +build js

package fs

import (
	"os"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/pkg/errors"
)

func (s fileShim) fchmod(args []js.Value) ([]interface{}, error) {
	_, err := s.fchmodSync(args)
	return nil, err
}

func (s fileShim) fchmodSync(args []js.Value) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("Invalid number of args, expected 2: %v", args)
	}

	fid := common.FID(args[0].Int())
	mode := os.FileMode(args[1].Int())
	return nil, s.process.Files().Fchmod(fid, mode)
}
