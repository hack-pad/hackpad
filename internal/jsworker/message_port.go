package jsworker

import (
	"context"
	"errors"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/jsfunc"
	"github.com/hack-pad/hackpad/internal/log"
)

type MessagePort struct {
	jsMessagePort js.Value
}

var jsMessageChannel = js.Global().Get("MessageChannel")

func NewChannel() (port1, port2 *MessagePort, err error) {
	defer common.CatchException(&err)
	channel := jsMessageChannel.New()
	port1, err = wrapMessagePort(channel.Get("port1"))
	if err != nil {
		return
	}
	port2, err = wrapMessagePort(channel.Get("port2"))
	return
}

func wrapMessagePort(v js.Value) (*MessagePort, error) {
	if !v.Get("postMessage").Truthy() {
		return nil, errors.New("invalid MessagePort value: postMessage is not a function")
	}
	return &MessagePort{v}, nil
}

func (p *MessagePort) PostMessage(message js.Value, transfers []js.Value) (err error) {
	defer common.CatchException(&err)
	args := append([]interface{}{message}, interop.SliceFromJSValues(transfers)...)
	log.PrintJSValues("Post message:", message)
	p.jsMessagePort.Call("postMessage", args...)
	return nil
}

func (p *MessagePort) Listen(ctx context.Context, listener func(MessageEvent, error)) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer common.CatchExceptionHandler(func(e error) {
		err = e
		cancel()
	})

	messageHandler := jsfunc.NonBlocking(func(this js.Value, args []js.Value) interface{} {
		ev, err := parseMessageEvent(args[0])
		listener(ev, err)
		return nil
	})
	errorHandler := jsfunc.NonBlocking(func(this js.Value, args []js.Value) interface{} {
		ev, err := parseMessageEvent(args[0])
		if err == nil {
			err = MessageEventErr{ev}
		}
		listener(MessageEvent{}, err)
		return nil
	})

	go func() {
		<-ctx.Done()
		defer messageHandler.Release()
		defer errorHandler.Release()
		p.jsMessagePort.Call("removeEventListener", "message", messageHandler)
		p.jsMessagePort.Call("removeEventListener", "messageerror", errorHandler)
	}()
	p.jsMessagePort.Call("addEventListener", "message", messageHandler)
	p.jsMessagePort.Call("addEventListener", "messageerror", errorHandler)
	if p.jsMessagePort.Get("start").Truthy() {
		p.jsMessagePort.Call("start")
	}
	return nil
}

func (p *MessagePort) Close() (err error) {
	defer common.CatchException(&err)
	p.jsMessagePort.Call("close")
	return nil
}
