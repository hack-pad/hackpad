// +build js

package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/log"
)

func (s domShim) installFunc(this js.Value, args []js.Value) (js.Wrapper, error) {
	return nil, s.install(args)
}

func (s domShim) install(args []js.Value) error {
	if len(args) != 1 {
		return errors.New("Expected command name to install")
	}
	command := args[0].String()
	return s.Install(command)
}

func (s domShim) Install(command string) error {
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
