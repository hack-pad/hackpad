// +build js

package interop

import "github.com/johnstarich/go-wasm/internal/global"

func SetInitialized() {
	global.Set("ready", true)
}
