package log

import (
	"fmt"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/global"
)

var (
	console  = js.Global().Get("console")
	logLevel = LevelLog
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

type consoleType int

const (
	LevelDebug consoleType = iota
	LevelLog
	LevelWarn
	LevelError
)

func (c consoleType) Valid() bool {
	switch c {
	case LevelDebug, LevelLog, LevelWarn, LevelError:
		return true
	default:
		return false
	}
}

func (c consoleType) String() string {
	switch c {
	case LevelDebug:
		return "debug"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "log"
	}
}

func parseLevel(level string) consoleType {
	switch level {
	case LevelDebug.String():
		return LevelDebug
	case LevelLog.String():
		return LevelLog
	case LevelWarn.String():
		return LevelWarn
	case LevelError.String():
		return LevelError
	default:
		return -1
	}
}

func SetLevel(level consoleType) {
	if level.Valid() {
		logLevel = level
		global.Set(logLevelKey, logLevel.String())
	}
}

func Debugf(format string, args ...interface{}) int {
	return logf(LevelDebug, format, args...)
}

func Printf(format string, args ...interface{}) int {
	return logf(LevelLog, format, args...)
}

func Warnf(format string, args ...interface{}) int {
	return logf(LevelWarn, format, args...)
}

func Errorf(format string, args ...interface{}) int {
	return logf(LevelError, format, args...)
}

func logf(kind consoleType, format string, args ...interface{}) int {
	if kind < logLevel {
		return 0
	}
	s := fmt.Sprintf(format, args...)
	console.Call(kind.String(), s)
	return len(s)
}

func Debug(args ...interface{}) int {
	return log(LevelDebug, args...)
}

func Print(args ...interface{}) int {
	return log(LevelLog, args...)
}

func Warn(args ...interface{}) int {
	return log(LevelWarn, args...)
}

func Error(args ...interface{}) int {
	return log(LevelError, args...)
}

func log(kind consoleType, args ...interface{}) int {
	if kind < logLevel {
		return 0
	}
	s := fmt.Sprint(args...)
	console.Call(kind.String(), s)
	return len(s)
}

func DebugJSValues(args ...js.Value) int {
	return logJSValues(LevelDebug, args...)
}

func PrintJSValues(args ...js.Value) int {
	return logJSValues(LevelLog, args...)
}

func WarnJSValues(args ...js.Value) int {
	return logJSValues(LevelWarn, args...)
}

func ErrorJSValues(args ...js.Value) int {
	return logJSValues(LevelError, args...)
}

func logJSValues(kind consoleType, args ...js.Value) int {
	if kind < logLevel {
		return 0
	}
	var intArgs []interface{}
	for _, arg := range args {
		intArgs = append(intArgs, arg)
	}
	console.Call(kind.String(), intArgs...)
	return 0
}
