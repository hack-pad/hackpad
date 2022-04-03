// +build js

package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/promise"
)

func (s domShim) installFunc(this js.Value, args []js.Value) interface{} {
	resolve, reject, prom := promise.New()
	go func() {
		err := s.install(args)
		if err != nil {
			reject(interop.WrapAsJSError(err, "Failed to install binary"))
			return
		}
		resolve(nil)
	}()
	return prom
}

func (s domShim) install(args []js.Value) error {
	if len(args) != 1 {
		return errors.New("Expected command name to install")
	}
	command := args[0].String()
	command = filepath.Base(command) // ensure no path chars are present

	if err := os.MkdirAll("/bin", 0644); err != nil {
		return err
	}

	body, err := httpGetFetch("wasm/" + command + ".wasm")
	if err != nil {
		return err
	}
	defer runtime.GC()
	file, err := os.OpenFile("/bin/"+command, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0750)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(body.Bytes()); err != nil {
		return err
	}
	log.Print("Install completed: ", command)
	return nil
}
