package main

import (
	"fmt"

	"github.com/johnstarich/go-wasm/internal/fs"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/process"
)

func main() {
	fmt.Println("polyfill!")
	process.Init()
	fs.Init()
	interop.SetInitialized("polyfill")
	select {}
}
