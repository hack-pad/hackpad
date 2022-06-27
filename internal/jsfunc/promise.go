package jsfunc

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/jserror"
	"github.com/hack-pad/hackpad/internal/promise"
)

type ErrFunc = func(this js.Value, args []js.Value) (js.Wrapper, error)

func Promise(fn ErrFunc) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve, reject, prom := promise.New()
		go func() {
			value, err := fn(this, args)
			if err != nil {
				reject(jserror.Wrap(err, "Failed to install binary"))
				return
			}
			resolve(value)
		}()
		return prom
	})
}
