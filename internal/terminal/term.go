//go:build js
// +build js

package terminal

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/process"
	"github.com/hack-pad/hackpadfs/indexeddb/idbblob"
	"github.com/pkg/errors"
)

func SpawnTerminal(this js.Value, args []js.Value) interface{} {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("Recovered from panic:", r)
			}
		}()
		err := Open(args)
		if err != nil {
			log.Error(err)
		}
	}()
	return nil
}

func Open(args []js.Value) error {
	if len(args) != 2 {
		return errors.New("Invalid number of args for spawnTerminal. Expected 2: term, options")
	}
	term := args[0]
	options := args[1]
	if options.Type() != js.TypeObject {
		return errors.Errorf("Invalid type for options: %s", options.Type())
	}
	var procArgs []string
	if args := options.Get("args"); args.Truthy() {
		procArgs = interop.StringsFromJSValue(args)
	}
	if len(procArgs) < 1 {
		return errors.New("options.args must have at least one argument")
	}

	workingDirectory := ""
	if wd := options.Get("cwd"); wd.Truthy() {
		workingDirectory = wd.String()
	}

	files := process.Current().Files()
	stdinR, stdinW := pipe(files)
	stdoutR, stdoutW := pipe(files)
	stderrR, stderrW := pipe(files)

	proc, err := process.New(procArgs[0], procArgs, &process.ProcAttr{
		Dir: workingDirectory,
		Files: []fs.Attr{
			{FID: stdinR},
			{FID: stdoutW},
			{FID: stderrW},
		},
	})
	if err != nil {
		return err
	}
	err = proc.Start()
	if err != nil {
		return err
	}

	f := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		chunk, err := idbblob.New(args[0])
		if err != nil {
			log.Error("blob: Failed to write to terminal:", err)
			return nil
		}
		_, err = files.Write(stdinW, chunk, 0, chunk.Len(), nil)
		if err != nil {
			log.Error("write: Failed to write to terminal:", err)
		}
		return nil
	})
	go func() {
		_, _ = proc.Wait()
		f.Release()
	}()
	term.Call("onData", f)
	go readOutputPipes(term, files, stdoutR)
	go readOutputPipes(term, files, stderrR)
	return nil
}

func pipe(files *fs.FileDescriptors) (r, w fs.FID) {
	p := files.Pipe()
	return p[0], p[1]
}

func readOutputPipes(term js.Value, files *fs.FileDescriptors, output fs.FID) {
	buf, err := idbblob.NewLength(1)
	if err != nil {
		panic(err)
	}
	for {
		_, err := files.Read(output, buf, 0, buf.Len(), nil)
		if err != nil {
			log.Error("Failed to write to terminal:", err)
		} else {
			term.Call("write", buf)
		}
	}
}
