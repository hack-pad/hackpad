package promise

import (
	"syscall/js"
)

func Await(promise Promise) (js.Value, error) {
	errs := make(chan error, 1)
	results := make(chan js.Value, 1)
	promise.Then(func(value js.Value) interface{} {
		results <- value
		close(results)
		return nil
	}).Catch(func(rejectedReason js.Value) interface{} {
		err := js.Error{Value: rejectedReason}
		errs <- err
		close(errs)
		return nil
	})
	select {
	case err := <-errs:
		return js.Null(), err
	case result := <-results:
		return result, nil
	}
}

func AwaitGo(promise GoPromise) (interface{}, interface{}) {
	// TODO support error handling inside promise functions instead
	return <-promise.resolveChan, <-promise.rejectChan
}
