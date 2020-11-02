// +build js

package fs

import (
	"os"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/common"
	"github.com/johnstarich/go-wasm/internal/process"
	"github.com/pkg/errors"
)

func fchmod(args []js.Value) ([]interface{}, error) {
	_, err := fchmodSync(args)
	return nil, err
}

func fchmodSync(args []js.Value) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("Invalid number of args, expected 2: %v", args)
	}

	fid := common.FID(args[0].Int())
	mode := os.FileMode(args[1].Int())
	p := process.Current()
	return nil, p.Files().Fchmod(fid, mode)
}
