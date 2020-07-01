package interop

import (
	"fmt"
	"runtime/debug"
	"strings"
	"syscall/js"

	"github.com/johnstarich/go-wasm/log"
	"github.com/pkg/errors"
)

type Func = func(args []js.Value) (interface{}, error)

type CallbackFunc = func(args []js.Value) ([]interface{}, error)

func SetFunc(val js.Value, name string, fn interface{}) js.Func {
	defer handlePanic(0)

	switch fn.(type) {
	case Func, CallbackFunc:
	default:
		panic(fmt.Sprintf("Invalid SetFunc type: %T", fn))
	}

	wrappedFn := func(_ js.Value, args []js.Value) interface{} {
		return setFuncHandler(name, fn, args)
	}
	jsWrappedFn := js.FuncOf(wrappedFn)
	val.Set(name, jsWrappedFn)
	return jsWrappedFn
}

func setFuncHandler(name string, fn interface{}, args []js.Value) (returnedVal interface{}) {
	logArgs := append([]js.Value{js.ValueOf("running op: " + name)}, args...)
	log.DebugJSValues(logArgs...)

	switch fn := fn.(type) {
	case Func:
		defer func() {
			log.DebugJSValues(js.ValueOf("completed sync op: "+name), js.ValueOf(returnedVal))
			handlePanic(0)
		}()

		ret, err := fn(args)
		if err != nil {
			log.Error(errors.Wrap(err, name).Error())
		}
		return ret
	case CallbackFunc:
		// callback style detected, so pop callback arg and call it with the return values
		// error always goes first
		callback := args[len(args)-1]
		args = args[:len(args)-1]
		go func() (returnedVal interface{}) {
			defer func() {
				log.DebugJSValues(js.ValueOf("completed op: "+name), js.ValueOf(returnedVal))
				handlePanic(0)
			}()
			ret, err := fn(args)
			err = WrapAsJSError(err, name)
			callbackArgs := append([]interface{}{err}, ret...)
			callback.Invoke(callbackArgs...)
			return callbackArgs
		}()
		return nil
	default:
		panic("impossible case") // handled above
	}
}

func handlePanic(skipPanicLines int) interface{} {
	r := recover()
	if r == nil {
		return nil
	}
	stack := string(debug.Stack())
	for iter := 0; iter < skipPanicLines; iter++ {
		ix := strings.IndexRune(stack, '\n')
		if ix == -1 {
			break
		}
		stack = stack[ix+1:]
	}
	switch r := r.(type) {
	case js.Value:
		log.ErrorJSValues(
			js.ValueOf("panic:"),
			r,
			js.ValueOf("\n\n"+stack),
		)
	default:
		log.Errorf("panic: (%T) %+v\n\n%s", r, r, stack)
	}
	// TODO need to find a way to just throw the error instead of crashing
	return r
}

func PanicLogger() {
	r := handlePanic(0)
	if r != nil {
		panic(r)
	}
}
