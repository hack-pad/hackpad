// +build js

package ide

import (
	"context"
	"syscall/js"
)

type ConsoleBuilder interface {
	New(elem js.Value, rawName, name string, args ...string) (Console, error)
}

type Console interface {
	Tabber
}

type ConsoleWaiter interface {
	Wait() error
}

type TaskConsoleBuilder interface {
	New(elem js.Value) TaskConsole
}

type TaskConsole interface {
	Tabber
	Start(rawName, name string, args ...string) (context.Context, error)
}
