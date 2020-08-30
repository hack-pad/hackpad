package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/console"
	"github.com/pkg/errors"
)

var (
	document = js.Global().Get("document")
)

const (
	KeyEnter = "Enter"
)

func main() {
	if err := os.Chdir("playground"); err != nil {
		panic(err)
	}

	app := document.Call("createElement", "div")
	app.Call("setAttribute", "id", "command")
	document.Get("body").Call("insertBefore", app, nil)
	app.Set("innerHTML", `
<input type="text" spellcheck="false" placeholder="go version" />
<div class="console"></div>
`)

	commands := make(chan string)

	input := app.Call("querySelector", "input")
	input.Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		event := args[0]
		if event.Get("code").String() != KeyEnter {
			return nil
		}
		target := event.Get("target")
		commands <- target.Get("value").String()
		target.Set("value", "")
		return nil
	}))
	consoleElem := app.Call("querySelector", ".console")
	terminal := console.New(consoleElem)

	for {
		fmt.Fprint(terminal.Stdout(), "$ ")
		err := runCommand(terminal, <-commands)
		if err != nil {
			fmt.Fprintln(terminal.Stderr(), err.Error())
		}
	}
}

func runCommand(term console.Console, line string) error {
	tokens := strings.Split(line, " ")
	if len(tokens) == 0 {
		return nil
	}
	fmt.Fprintln(term.Stdout(), line)
	isBuiltin, err := runBuiltin(term, tokens[0], tokens[1:]...)
	if isBuiltin {
		return err
	}
	cmd := exec.Command(tokens[0], tokens[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = term.Stdout()
	cmd.Stderr = term.Stderr()
	return cmd.Run()
}

func runBuiltin(term console.Console, name string, args ...string) (ok bool, err error) {
	switch name {
	case "cd":
		err = cd(term, args...)
	case "ls":
		err = ls(term, args...)
	case "mkdir":
		err = mkdir(term, args...)
	case "cat":
		err = cat(term, args...)
	default:
		return false, errors.Errorf("Unknown builtin command: %s", name)
	}
	return true, errors.Wrap(err, name)
}
