package interop

import (
	"fmt"
	"runtime/debug"
	"strings"
	"syscall/js"

	"github.com/johnstarich/go-wasm/log"
	"github.com/pkg/errors"
)

var jsErr = js.Global().Get("Error")

type Func = func(args []js.Value) (interface{}, error)

type CallbackFunc = func(args []js.Value) ([]interface{}, error)

func SetFunc(val js.Value, name string, fn interface{}) js.Func {
	defer handlePanic(0)

	switch fn.(type) {
	case Func, CallbackFunc:
	default:
		panic(fmt.Sprintf("Invalid SetFunc type: %T", fn))
	}

	wrappedFn := func(_ js.Value, args []js.Value) (returnedVal interface{}) {
		log.Debug("running op: ", name)
		defer func() {
			log.DebugJSValues(js.ValueOf("completed op: "+name), js.ValueOf(returnedVal))
		}()

		const unhelpfulStackLines = 7
		defer handlePanic(unhelpfulStackLines)

		switch fn := fn.(type) {
		case Func:
			ret, err := fn(args)
			if err != nil {
				log.Error(errors.Wrap(err, name).Error())
				return nil
			}
			return ret
		case CallbackFunc:
			// callback style detected, so pop callback arg and call it with the return values
			// error always goes first
			callback := args[len(args)-1]
			args = args[:len(args)-1]
			ret, err := fn(args)
			if err != nil {
				err = errors.Wrap(err, name)
				err = js.Error{Value: jsErr.New(err.Error())}
			}
			callbackArgs := append([]interface{}{err}, ret...)
			go callback.Invoke(callbackArgs...)
			return nil
		default:
			panic("impossible case") // handled above
		}
	}
	jsWrappedFn := js.FuncOf(wrappedFn)
	val.Set(name, jsWrappedFn)
	return jsWrappedFn
}

func handlePanic(skipPanicLines int) {
	r := recover()
	if r == nil {
		return
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
	// TODO keep the panic? need to find a way to just throw the error instead of crashing
	//panic(r)
}
