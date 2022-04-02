package jsworker

import (
	"context"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/common"
)

var (
	jsWorker = js.Global().Get("Worker")
)

const (
	wasmWorkerScript = "/wasmWorker.js"
)

type Remote struct {
	port   *MessagePort
	worker js.Value
}

func NewRemote(name, url string) (_ *Remote, err error) {
	defer common.CatchException(&err)
	val := jsWorker.New(url, map[string]interface{}{
		"name": name,
	})
	port, err := wrapMessagePort(val)
	return &Remote{
		port:   port,
		worker: val,
	}, err
}

func NewRemoteWasm(ctx context.Context, name, wasmURL string) (*Remote, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	l, err := NewRemote(name, wasmWorkerScript+"?wasm="+wasmURL)
	if err != nil {
		return nil, err
	}
	err = l.port.Listen(ctx, func(ev MessageEvent, err error) {
		if jsString(ev.Data) == "ready" {
			cancel()
		}
	})
	if err != nil {
		return nil, err
	}
	<-ctx.Done()
	return l, err
}

func (l *Remote) Terminate() (err error) {
	defer common.CatchException(&err)
	l.worker.Call("terminate")
	return nil
}
