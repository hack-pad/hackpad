//go:build js
// +build js

package fs

import (
	"os"
	"syscall"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/process"
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
	p := process.Current()
	info, err := p.Files().Stat(path)
	return jsStat(info), err
}

var (
	funcTrue = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return true
	})
	funcFalse = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return false
	})
)

func jsStat(info os.FileInfo) js.Value {
	if info == nil {
		return js.Null()
	}
	const blockSize = 4096 // TODO find useful value for blksize
	modTime := info.ModTime().UnixNano() / 1e6
	return js.ValueOf(map[string]interface{}{
		"dev":     0,
		"ino":     0,
		"mode":    jsMode(info.Mode()),
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

		"isBlockDevice":     funcFalse,
		"isCharacterDevice": funcFalse,
		"isDirectory":       jsBoolFunc(info.IsDir()),
		"isFIFO":            funcFalse,
		"isFile":            jsBoolFunc(info.Mode().IsRegular()),
		"isSocket":          funcFalse,
		"isSymbolicLink":    jsBoolFunc(info.Mode()&os.ModeSymlink == os.ModeSymlink),
	})
}

var modeBitTranslation = map[os.FileMode]uint32{
	os.ModeDir:        syscall.S_IFDIR,
	os.ModeCharDevice: syscall.S_IFCHR,
	os.ModeNamedPipe:  syscall.S_IFIFO,
	os.ModeSymlink:    syscall.S_IFLNK,
	os.ModeSocket:     syscall.S_IFSOCK,
}

func jsMode(mode os.FileMode) uint32 {
	for goBit, jsBit := range modeBitTranslation {
		if mode&goBit == goBit {
			mode = mode & ^goBit | os.FileMode(jsBit)
		}
	}
	return uint32(mode)
}

func blockCount(size, blockSize int64) int64 {
	blocks := size / blockSize
	if size%blockSize > 0 {
		return blocks + 1
	}
	return blocks
}

func jsBoolFunc(b bool) js.Func {
	if b {
		return funcTrue
	}
	return funcFalse
}
