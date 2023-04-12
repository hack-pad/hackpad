//go:build !js
// +build !js

package log

import (
	"fmt"
	"os"
)

func writeLog(c consoleType, s string) {
	if os.Getenv("DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "%s: %s\n", c.String(), s)
	}
}
