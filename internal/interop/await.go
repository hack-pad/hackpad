package interop

import (
	"errors"
	"syscall/js"

	"github.com/johnstarich/go-wasm/log"
)

func Await(promise js.Value) (js.Value, error) {
	errs := make(chan error, 1)
	results := make(chan js.Value, 1)
	fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var result js.Value
		if len(args) > 0 {
			result = args[0]
		}
		results <- result
		close(results)
		return nil
	})
	defer fn.Release()
	catch := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var err error
		if len(args) > 0 {
			err = js.Error{Value: args[0]}
		} else {
			err = errors.New("error arg missing")
		}
		log.Errorf("Promise rejected: %s", err.Error())
		errs <- err
		close(errs)
		return nil
	})
	defer catch.Release()
	promise.Call("then", fn).Call("catch", catch)
	select {
	case err := <-errs:
		return js.Null(), err
	case result := <-results:
		return result, nil
	}
}
