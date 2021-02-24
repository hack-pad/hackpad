package main

import (
	"flag"
	"io"
	"os"
	"strings"
)

func main() {
	os.Exit(run())
}

func run() int {
	cancel, err := ttySetup()
	if err != nil {
		panic(err)
	}
	defer cancel()

	command := flag.String("c", "", "Read and execute commands from the given string value.")
	flag.Parse()

	var reader io.RuneReader
	if *command != "" {
		reader = newRuneReader(strings.NewReader(*command))
	} else {
		reader = newRuneReader(os.Stdin)
	}
	os.Stdout, err = newCarriageReturnWriter(os.Stdout)
	if err != nil {
		panic(err)
	}
	os.Stderr, err = newCarriageReturnWriter(os.Stderr)
	if err != nil {
		panic(err)
	}
	term := newTerminal()

	return term.ReadEvalPrintLoop(reader)
}
