package main

import (
	"os"

	"github.com/fatih/color"
)

func main() {
	color.NoColor = false // override, since wasm isn't considered a "tty"
	reader := newRuneReader(os.Stdin)
	term := newTerminal()

	term.ReadEvalPrintLoop(reader)
}
