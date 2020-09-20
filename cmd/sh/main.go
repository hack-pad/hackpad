package main

import (
	"flag"
	"io"
	"os"
	"strings"
)

func main() {
	command := flag.String("c", "", "Read and execute commands from the given string value.")
	flag.Parse()

	var reader io.RuneReader
	if *command != "" {
		reader = newRuneReader(strings.NewReader(*command))
	} else {
		reader = newRuneReader(os.Stdin)
	}
	term := newTerminal()

	term.ReadEvalPrintLoop(reader)
}
