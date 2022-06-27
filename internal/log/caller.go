package log

import (
	"fmt"
	"runtime"
	"strings"
)

const (
	hackpadCommonPrefix = "github.com/hack-pad/"
)

func getCaller(skip int) string {
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return ""
	}
	file = strings.TrimPrefix(file, hackpadCommonPrefix)
	fn := runtime.FuncForPC(pc).Name()
	fn = fn[strings.LastIndexAny(fn, "./")+1:]
	return fmt.Sprintf("%s:%d:%s()", file, line, fn)
}
