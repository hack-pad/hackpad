package interop

import (
	"fmt"
	"io"
	"os"
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

	wrappedFn := func(_ js.Value, args []js.Value) (returnedVal interface{}) {
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
				return nil
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
				err = wrapAsJSError(err, name)
				callbackArgs := append([]interface{}{err}, ret...)
				callback.Invoke(callbackArgs...)
				return callbackArgs
			}()
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

func wrapAsJSError(err error, message string) error {
	if err == nil {
		return nil
	}

	val := js.ValueOf(map[string]interface{}{
		"message": js.ValueOf(errors.Wrap(err, message).Error()),
		"code":    js.ValueOf(mapToErrNo(err)),
	})
	return js.Error{Value: val}
}

// errno names pulled from syscall/tables_js.go
func mapToErrNo(err error) string {
	if err, ok := err.(Error); ok {
		return err.Code()
	}
	switch err {
	case io.EOF, os.ErrNotExist:
		return "ENOENT"
	case os.ErrExist:
		return "EEXIST"
	case os.ErrPermission:
		return "EPERM"
	}
	switch {
	case os.IsNotExist(err):
		return "ENOENT"
	case os.IsExist(err):
		return "EEXIST"
	default:
		log.Errorf("Unknown error type: (%T) %+v", err, err)
		return "EPERM"
	}
}
