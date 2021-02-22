// +build js

package ide

import (
	"context"

	"github.com/johnstarich/go-wasm/cmd/editor/element"
)

type ConsoleBuilder interface {
	New(elem *element.Element, rawName, name string, args ...string) (Console, error)
}

type Console interface {
	Tabber
}

type ConsoleWaiter interface {
	Wait() error
}

type TaskConsoleBuilder interface {
	New(elem *element.Element) TaskConsole
}

type TaskConsole interface {
	Tabber
	Start(rawName, name string, args ...string) (context.Context, error)
}
