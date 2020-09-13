package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"

	"github.com/fatih/color"
	"github.com/johnstarich/go-wasm/internal/console"
	"github.com/pkg/errors"
)

func main() {
	reader := newRuneReader(os.Stdin)
	term := newTerminal()
	color.NoColor = false // override, since wasm isn't considered a "tty"

	term.ReadEvalPrintLoop(reader)
}

func runCommand(term console.Console, line string) error {
	tokens := tokenize(line)
	if len(tokens) == 0 {
		return nil
	}
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
	case "cat":
		err = cat(term, args...)
	case "cd":
		err = cd(term, args...)
	case "echo":
		fmt.Fprintln(term.Stdout(), strings.Join(args, " "))
	case "ls":
		err = ls(term, args...)
	case "mkdir":
		err = mkdir(term, args...)
	case "mv":
		err = mv(term, args...)
	case "pwd":
		path, pErr := os.Getwd()
		fmt.Fprintln(term.Stdout(), path)
		err = pErr
	case "rm":
		err = rm(term, args...)
	case "rmdir":
		err = rmdir(term, args...)
	case "touch":
		err = touch(term, args...)
	case "which":
		err = which(term, args...)
	default:
		return false, errors.Errorf("Unknown builtin command: %s", name)
	}
	return true, errors.Wrap(err, name)
}

func tokenize(line string) []string {
	var tokens []string
	var buf []rune
	for _, c := range line {
		if unicode.IsSpace(c) && len(buf) != 0 {
			tokens = append(tokens, string(buf))
			buf = nil
		} else {
			buf = append(buf, c)
		}
	}
	if len(buf) != 0 {
		tokens = append(tokens, string(buf))
	}
	return tokens
}
