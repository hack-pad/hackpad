//go:build js
// +build js

package interop

import "github.com/hack-pad/hackpad/internal/global"

func SetInitialized() {
	global.Set("ready", true)
}
