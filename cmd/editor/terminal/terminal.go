// +build js

package terminal

import (
	"io"
	"os/exec"
	"syscall/js"

	"github.com/johnstarich/go-wasm/cmd/editor/element"
	"github.com/johnstarich/go-wasm/cmd/editor/ide"
	"github.com/johnstarich/go-wasm/internal/blob"
	"github.com/johnstarich/go-wasm/log"
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
	cmd       *exec.Cmd
	titleChan chan string
}

func (b *terminalBuilder) New(elem *element.Element, rawName, name string, args ...string) (ide.Console, error) {
	term := &terminal{
		xterm:     b.newXTermFunc.Invoke(elem),
		titleChan: make(chan string, 1),
	}
	return term, term.start(rawName, name, args...)
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

	err = t.cmd.Start()
	if err != nil {
		return err
	}

	f := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		chunk := []byte(args[0].String())
		_, err := stdin.Write(chunk)
		if err != nil {
			log.Error("Failed to write to terminal:", err)
		}
		return nil
	})
	go func() {
		_ = t.cmd.Wait()
		f.Release()
	}()
	t.xterm.Call("onData", f)
	go readOutputPipes(t.xterm, stdout)
	go readOutputPipes(t.xterm, stderr)
	return nil
}

func (t *terminal) Wait() error {
	return t.cmd.Wait()
}

func readOutputPipes(term js.Value, r io.Reader) {
	buf := make([]byte, 1)
	for {
		_, err := r.Read(buf)
		if err != nil {
			log.Error("Failed to write to terminal:", err)
		} else {
			term.Call("write", blob.NewFromBytes(buf))
		}
	}
}

func (t *terminal) Titles() <-chan string {
	return t.titleChan
}
