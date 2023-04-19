//go:build js && wasm
// +build js,wasm

package log

import (
	"fmt"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/global"
	"github.com/hack-pad/safejs"
)

var (
	console = safejs.MustGetGlobal("console")
)

const logLevelKey = "logLevel"

func init() {
	global.SetDefault(logLevelKey, LevelLog.String())
	global.SetDefault("setLogLevel", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return nil
		}
		level := args[0].String()
		SetLevel(parseLevel(level))
		return logLevel.String()
	}))
}

func SetLevel(level consoleType) {
	if level.Valid() {
		logLevel = level
		global.Set(logLevelKey, logLevel.String())
	}
}

func DebugJSValues(args ...interface{}) int {
	return logJSValues(LevelDebug, args...)
}

func PrintJSValues(args ...interface{}) int {
	return logJSValues(LevelLog, args...)
}

func WarnJSValues(args ...interface{}) int {
	return logJSValues(LevelWarn, args...)
}

func ErrorJSValues(args ...interface{}) int {
	return logJSValues(LevelError, args...)
}

func logJSValues(kind consoleType, args ...interface{}) int {
	if kind < logLevel {
		return 0
	}
	var jsArgs []interface{}
	for _, arg := range args {
		jsArg, err := safejs.ValueOf(arg)
		if err != nil {
			jsArg = safejs.Safe(js.ValueOf(fmt.Sprintf("LOGERR(%s: %T %+v)", err, arg, arg)))
		}
		jsArgs = append(jsArgs, jsArg)
	}
	_, _ = console.Call(kind.String(), jsArgs...)
	return 0
}

func writeLog(c consoleType, s string) {
	_, _ = console.Call(c.String(), s)
}
