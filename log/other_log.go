// +build !js

package log

import (
	"fmt"
	"os"
)

func writeLog(c consoleType, s string) {
	if os.Getenv("DEBUG") == "true" {
		fmt.Printf("%s: %s\n", c.String(), s)
	}
}
