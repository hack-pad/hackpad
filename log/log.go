package log

import "fmt"

type consoleType int

const (
	LevelDebug consoleType = iota
	LevelLog
	LevelWarn
	LevelError
)

var logLevel = LevelLog

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
	writeLog(kind, s)
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
	writeLog(kind, s)
	return len(s)
}
