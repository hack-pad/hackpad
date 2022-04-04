package jsworker

import (
	"context"
	"syscall/js"
)

type Local struct {
	port *MessagePort
}

var self *Local

func init() {
	jsSelf := js.Global().Get("self")
	if !jsSelf.Truthy() {
		return
	}
	port, err := wrapMessagePort(jsSelf)
	if err != nil {
		panic(err)
	}
	self = &Local{
		port: port,
	}
}

func GetLocal() *Local {
	return self
}

func (l *Local) PostMessage(message js.Value, transfers []js.Value) error {
	return l.port.PostMessage(message, transfers)
}

func (l *Local) Listen(ctx context.Context, listener func(MessageEvent, error)) error {
	return l.port.Listen(ctx, listener)
}

func (l *Local) Close() error {
	return l.port.Close()
}
