// +build js,wasm

package log

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/global"
)

var (
	console = js.Global().Get("console")
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
	return logJSValues(LevelDebug, 1, args...)
}

func PrintJSValues(args ...interface{}) int {
	return logJSValues(LevelLog, 1, args...)
}

func WarnJSValues(args ...interface{}) int {
	return logJSValues(LevelWarn, 1, args...)
}

func ErrorJSValues(args ...interface{}) int {
	return logJSValues(LevelError, 1, args...)
}

func logJSValues(kind consoleType, skip int, args ...interface{}) int {
	if kind < logLevel {
		return 0
	}
	caller := getCaller(skip + 1)
	args = append([]interface{}{caller}, args...)
	console.Call(kind.String(), args...)
	return 0
}

func writeLog(c consoleType, s string) {
	console.Call(c.String(), s)
}
