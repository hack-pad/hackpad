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

func NewRemoteWasm(name, wasmURL string) (*Remote, error) {
	return NewRemote(name, wasmWorkerScript+"?wasm="+wasmURL)
}

func (r *Remote) Terminate() (err error) {
	defer common.CatchException(&err)
	r.worker.Call("terminate")
	return nil
}

func (r *Remote) PostMessage(message js.Value, transfers []js.Value) error {
	return r.port.PostMessage(message, transfers)
}

func (r *Remote) Listen(ctx context.Context, listener func(MessageEvent, error)) error {
	return r.port.Listen(ctx, listener)
}
