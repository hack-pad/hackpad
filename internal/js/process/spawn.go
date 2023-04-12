//go:build js
// +build js

package process

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

func spawn(args []js.Value) (interface{}, error) {
	if len(args) == 0 {
		return nil, errors.Errorf("Invalid number of args, expected command name: %v", args)
	}

	command := args[0].String()
	argv := []string{command}
	if len(args) >= 2 {
		if args[1].Type() != js.TypeObject || args[1].Get("length").IsUndefined() {
			return nil, errors.New("Second arg must be an array of arguments")
		}
		length := args[1].Length()
		for i := 0; i < length; i++ {
			argv = append(argv, args[1].Index(i).String())
		}
	}

	procAttr := &process.ProcAttr{}
	if len(args) >= 3 {
		argv[0], procAttr = parseProcAttr(command, args[2])
	}
	return Spawn(command, argv, procAttr)
}

type jsWrapper interface {
	JSValue() js.Value
}

func Spawn(command string, args []string, attr *process.ProcAttr) (js.Value, error) {
	p, err := process.New(command, args, attr)
	if err != nil {
		return js.Value{}, err
	}
	return p.(jsWrapper).JSValue(), p.Start()
}

func parseProcAttr(defaultCommand string, value js.Value) (argv0 string, attr *process.ProcAttr) {
	argv0 = defaultCommand
	attr = &process.ProcAttr{}
	if dir := value.Get("cwd"); dir.Truthy() {
		attr.Dir = dir.String()
	}
	if env := value.Get("env"); env.Truthy() {
		attr.Env = make(map[string]string)
		for name, prop := range interop.Entries(env) {
			attr.Env[name] = prop.String()
		}
	}

	if stdio := value.Get("stdio"); stdio.Truthy() {
		length := stdio.Length()
		for i := 0; i < length; i++ {
			file := stdio.Index(i)
			switch file.Type() {
			case js.TypeNumber:
				fd := fs.FID(file.Int())
				attr.Files = append(attr.Files, fs.Attr{FID: fd})
			case js.TypeString:
				switch file.String() {
				case "ignore":
					attr.Files = append(attr.Files, fs.Attr{Ignore: true})
				case "inherit":
					attr.Files = append(attr.Files, fs.Attr{FID: fs.FID(i)})
				case "pipe":
					attr.Files = append(attr.Files, fs.Attr{Pipe: true})
				}
			}
		}
	}

	if jsArgv0 := value.Get("argv0"); jsArgv0.Truthy() {
		argv0 = jsArgv0.String()
	}

	return
}
