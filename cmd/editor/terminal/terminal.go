//go:build js
// +build js

package terminal

import (
	"io"
	"os/exec"
	"syscall/js"

	"github.com/hack-pad/hackpad/cmd/editor/dom"
	"github.com/hack-pad/hackpad/cmd/editor/ide"
	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpadfs/indexeddb/idbblob"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
)

type terminalBuilder struct {
	newXTermFunc js.Value
}

func New(xtermFunc js.Value) ide.ConsoleBuilder {
	return &terminalBuilder{
		newXTermFunc: xtermFunc,
	}
}

type terminal struct {
	xterm     js.Value
	closables []func() error
	cmd       *exec.Cmd
	titleChan chan string
	closed    bool
}

func (b *terminalBuilder) New(elem *dom.Element, rawName, name string, args ...string) (ide.Console, error) {
	term := &terminal{
		xterm:     b.newXTermFunc.Invoke(elem.JSValue()),
		titleChan: make(chan string, 1),
	}
	go func() {
		err := term.start(rawName, name, args...)
		if err != nil {
			log.Error("Failed to start terminal:", err)
		}
	}()
	return term, nil
}

func (t *terminal) start(rawName, name string, args ...string) error {
	t.titleChan <- "Terminal"

	if rawName == "" {
		rawName = name
	}
	t.cmd = exec.Command(name, args...)
	t.cmd.Path = rawName
	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := t.cmd.StderrPipe()
	if err != nil {
		return err
	}
	t.closables = append(t.closables, stdin.Close, stdout.Close, stderr.Close)

	err = t.cmd.Start()
	if err != nil {
		return err
	}

	f := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		chunk := []byte(args[0].String())
		_, err := stdin.Write(chunk)
		if err == io.EOF {
			err = t.Close()
		}
		if err != nil {
			log.Error("Failed to write to terminal:", err)
		}
		return nil
	})
	go func() {
		_ = t.cmd.Wait()
		f.Release()
	}()
	dataListener := t.xterm.Call("onData", f)
	t.closables = append(t.closables, func() (err error) {
		defer common.CatchException(&err)
		dataListener.Call("dispose")
		log.Print("disposed of data listener")
		return nil
	})

	go t.readOutputPipes(stdout)
	go t.readOutputPipes(stderr)
	return nil
}

func (t *terminal) Wait() error {
	return t.cmd.Wait()
}

func (t *terminal) Close() error {
	if t.closed {
		return nil
	}
	t.closed = true
	const colorRed = "\033[1;31m"
	t.xterm.Call("write", idbblob.FromBlob(blob.NewBytes([]byte("\n\r"+colorRed+"[exited]\n\r"))).JSValue())
	var err error
	for _, closer := range t.closables {
		cErr := closer()
		if cErr != nil {
			err = cErr
		}
	}
	close(t.titleChan)
	return err
}

func (t *terminal) readOutputPipes(r io.Reader) {
	buf := make([]byte, 1)
	for {
		_, err := r.Read(buf)
		switch err {
		case nil:
			t.xterm.Call("write", idbblob.FromBlob(blob.NewBytes(buf)).JSValue())
		case io.EOF:
			t.Close()
			return
		default:
			log.Error("Failed to write to terminal:", err)
		}
	}
}

func (t *terminal) Titles() <-chan string {
	return t.titleChan
}
