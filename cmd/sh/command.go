package main

import (
	"os"
	"os/exec"
	"unicode"

	"github.com/johnstarich/go-wasm/internal/console"
	"github.com/pkg/errors"
)

func runCommand(term console.Console, line string) error {
	tokens := tokenize(line)
	if len(tokens) == 0 {
		return nil
	}
	commandName := tokens[0]
	args := tokens[1:]

	builtin, ok := builtins[commandName]
	if ok {
		err := builtin(term, args...)
		return errors.Wrap(err, commandName)
	}
	cmd := exec.Command(commandName, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = term.Stdout()
	cmd.Stderr = term.Stderr()
	return cmd.Run()
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
