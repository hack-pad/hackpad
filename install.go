package main

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/johnstarich/go-wasm/log"
)

func installFunc(this js.Value, args []js.Value) interface{} {
	resolve, reject, prom := promise.New()
	go func() {
		err := install(args)
		if err != nil {
			reject(interop.WrapAsJSError(err, "Failed to install binary"))
			return
		}
		resolve(nil)
	}()
	return prom
}

func install(args []js.Value) error {
	if len(args) != 1 {
		return errors.New("Expected command name to install")
	}
	command := args[0].String()
	command = filepath.Base(command) // ensure no path chars are present

	if err := os.MkdirAll("/bin", 0644); err != nil {
		return err
	}

	resp, err := http.Get("/" + command + ".wasm")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.Create("/bin/" + command)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	if err := f.Chmod(0750); err != nil {
		return err
	}

	log.Print("Install completed: ", command)
	return nil
}
