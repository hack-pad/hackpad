package fs

import (
	"os"
	"syscall/js"

	"github.com/pkg/errors"
)

func stat(args []js.Value) ([]interface{}, error) {
	info, err := statSync(args)
	return []interface{}{info}, err
}

func statSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	path := args[0].String()
	info, err := Stat(path)
	if err != nil {
		return nil, err
	}

	const blockSize = 4096 // TODO find useful value for blksize
	modTime := info.ModTime().UnixNano() / 1000
	return map[string]interface{}{
		"dev":     0,
		"ino":     0,
		"mode":    uint32(info.Mode()),
		"nlink":   1,
		"uid":     0, // TODO use real values for uid and gid
		"gid":     0,
		"rdev":    0,
		"size":    info.Size(),
		"blksize": blockSize,
		"blocks":  blockCount(info.Size(), blockSize),
		"atimeMs": modTime,
		"mtimeMs": modTime,
		"ctimeMs": modTime,
	}, nil
}

func Stat(path string) (os.FileInfo, error) {
	return filesystem.Stat(path)
}

func blockCount(size, blockSize int64) int64 {
	blocks := size / blockSize
	if size%blockSize > 0 {
		return blocks + 1
	}
	return blocks
}
