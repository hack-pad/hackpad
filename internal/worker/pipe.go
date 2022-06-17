package worker

import (
	"context"
	"io"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/jsworker"
	"github.com/hack-pad/hackpadfs/indexeddb/idbblob"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
)

func optionalPipe(v js.Value) *jsworker.MessagePort {
	if v.Type() != js.TypeObject {
		return nil
	}
	port, err := jsworker.WrapMessagePort(v)
	if err != nil {
		panic(err)
	}
	return port
}

type portPipe struct {
	port              *jsworker.MessagePort
	receivedData      <-chan portPipeMessage
	remainingReadData []byte
	cancel            context.CancelFunc
}

type portPipeMessage struct {
	Data []byte
	Err  error
}

func portToReadWriteCloser(port *jsworker.MessagePort) (io.ReadWriteCloser, error) {
	ctx, cancel := context.WithCancel(context.Background())
	receivedData := make(chan portPipeMessage)
	err := port.Listen(ctx, func(me jsworker.MessageEvent, err error) {
		var buf []byte
		if err == nil {
			var bl blob.Blob
			bl, err = idbblob.New(me.Data)
			if err == nil {
				buf = bl.Bytes()
			}
		}
		receivedData <- portPipeMessage{Data: buf, Err: err}
	})
	if err != nil {
		return nil, err
	}
	return &portPipe{
		port:         port,
		receivedData: receivedData,
		cancel:       cancel,
	}, nil
}

func (p *portPipe) Close() error {
	p.cancel()
	return p.port.Close()
}

func (p *portPipe) Read(b []byte) (n int, err error) {
	if len(p.remainingReadData) == 0 {
		message := <-p.receivedData
		p.remainingReadData = message.Data
		err = message.Err
	}
	n = copy(b, p.remainingReadData)
	p.remainingReadData = p.remainingReadData[n:]
	return
}

func (p *portPipe) Write(b []byte) (n int, err error) {
	bl := idbblob.FromBlob(blob.NewBytes(b)).JSValue()
	err = p.port.PostMessage(bl, []js.Value{bl.Get("buffer")})
	if err == nil {
		n = len(b)
	}
	return
}
