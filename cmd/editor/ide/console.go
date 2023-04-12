//go:build js
// +build js

package ide

import (
	"context"

	"github.com/hack-pad/hackpad/cmd/editor/dom"
)

type ConsoleBuilder interface {
	New(elem *dom.Element, rawName, name string, args ...string) (Console, error)
}

type Console interface {
	Tabber
}

type ConsoleWaiter interface {
	Wait() error
}

type TaskConsoleBuilder interface {
	New(elem *dom.Element) TaskConsole
}

type TaskConsole interface {
	Tabber
	Start(rawName, name string, args ...string) (context.Context, error)
}
