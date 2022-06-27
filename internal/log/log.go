package log

import (
	"fmt"
)

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
	return logf(LevelDebug, 1, format, args...)
}

func Printf(format string, args ...interface{}) int {
	return logf(LevelLog, 1, format, args...)
}

func Warnf(format string, args ...interface{}) int {
	return logf(LevelWarn, 1, format, args...)
}

func Errorf(format string, args ...interface{}) int {
	return logf(LevelError, 1, format, args...)
}

func logf(kind consoleType, skip int, format string, args ...interface{}) int {
	if kind < logLevel {
		return 0
	}
	s := fmt.Sprintf(format, args...)
	if caller := getCaller(skip + 1); caller != "" {
		s = caller + " - " + s
	}
	writeLog(kind, s)
	return len(s)
}

func Debug(args ...interface{}) int {
	return log(LevelDebug, 1, args...)
}

func Print(args ...interface{}) int {
	return log(LevelLog, 1, args...)
}

func Warn(args ...interface{}) int {
	return log(LevelWarn, 1, args...)
}

func Error(args ...interface{}) int {
	return log(LevelError, 1, args...)
}

func log(kind consoleType, skip int, args ...interface{}) int {
	if kind < logLevel {
		return 0
	}
	s := fmt.Sprint(args...)
	if caller := getCaller(skip + 1); caller != "" {
		s = caller + " - " + s
	}
	writeLog(kind, s)
	return len(s)
}
