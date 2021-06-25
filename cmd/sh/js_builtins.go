// +build js,wasm

package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"syscall/js"

	"github.com/fatih/color"
	"github.com/johnstarich/go-wasm/internal/console"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/pkg/errors"
)

var (
	jsFunction = js.Global().Get("Function")
	goWasm     = js.Global().Get("goWasm")
)

func init() {
	builtins["jseval"] = jseval
	builtins["wpk"] = wpk
	builtins["jsdownload"] = jsdownload
	color.NoColor = false // override, since wasm isn't considered a "tty"
}

func jsEval(funcStr string, args ...interface{}) js.Value {
	f := jsFunction.Invoke(`"use strict";` + funcStr)
	return f.Invoke(args...)
}

func jseval(term console.Console, args ...string) error {
	if len(args) < 1 {
		return errors.New("Must provide a string to run as a function")
	}
	result := jsEval(args[0], strings.Join(args[1:], " "))
	fmt.Fprintln(term.Stdout(), result)
	return nil
}

func wpk(term console.Console, args ...string) error {
	if len(args) < 2 {
		return errors.New(strings.TrimSpace(`
Usage: wpk add <pkg>

Installs a remote package by the name of 'pkg'.
`))
	}
	switch args[0] {
	case "add":
		prom := promise.From(goWasm.Call("install", args[1]))
		_, err := prom.Await()
		return err
	default:
		return errors.Errorf("Invalid command: %q", args[0])
	}
}

func jsdownload(term console.Console, args ...string) error {
	if len(args) < 1 {
		return errors.New("Must provide a file to download")
	}
	filePath := args[0]
	fileContents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Wrap(err, "Error reading file for download")
	}
	interop.StartDownload("", filePath, fileContents)
	return nil
}
