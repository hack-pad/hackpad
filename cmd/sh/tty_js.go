// +build js

package main

import "context"

func ttySetup() (context.CancelFunc, error) {
	return func() {}, nil
}
