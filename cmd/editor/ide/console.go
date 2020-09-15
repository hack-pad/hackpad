package ide

import (
	"context"
	"syscall/js"
)

type ConsoleBuilder interface {
	New(elem js.Value, rawName, name string, args ...string) (Console, error)
}

type Console interface {
	Wait() error
	Titles() <-chan string
}

type TaskConsoleBuilder interface {
	New(elem js.Value) TaskConsole
}

type TaskConsole interface {
	Console
	Start(rawName, name string, args ...string) (context.Context, error)
}
