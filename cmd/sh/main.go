package main

import (
	"io"
	"log"
	"os"

	"github.com/hack-pad/hush"
)

func main() {
	log.SetOutput(io.Discard)
	exitCode := hush.Run()
	os.Exit(exitCode)
}
