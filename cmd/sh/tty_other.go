// +build !js

package main

import (
	"context"
	"fmt"
	"os"

	gotty "github.com/mattn/go-tty"
)

func ttySetup() (context.CancelFunc, error) {
	tty, err := gotty.Open()
	if err != nil {
		return nil, err
	}
	cancel, err := tty.Raw()
	if err != nil {
		return nil, err
	}
	os.Stdin = tty.Input()
	return func() {
		err := cancel()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to restore tty:", err)
		}
		tty.Close()
	}, nil
}
