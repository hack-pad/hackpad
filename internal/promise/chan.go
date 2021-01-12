package promise

import (
	"github.com/pkg/errors"
)

type Go struct {
	resolveChan, rejectChan <-chan interface{}
}

func NewGo() (resolve, reject Resolver, promise Go) {
	resolveChan, rejectChan := make(chan interface{}, 1), make(chan interface{}, 1)
	var p Go
	p.resolveChan, p.rejectChan = resolveChan, rejectChan

	resolve = func(result interface{}) {
		resolveChan <- result
		close(resolveChan)
		close(rejectChan)
	}
	reject = func(result interface{}) {
		rejectChan <- result
		close(resolveChan)
		close(rejectChan)
	}
	return resolve, reject, p
}

func (p Go) Then(fn func(value interface{}) interface{}) Promise {
	// TODO support failing a Then call
	resolve, _, prom := NewGo()
	go func() {
		value, ok := <-p.resolveChan
		if ok {
			newValue := fn(value)
			resolve(newValue)
		}
	}()
	return prom
}

func (p Go) Catch(fn func(rejectedReason interface{}) interface{}) Promise {
	_, reject, prom := NewGo()
	go func() {
		reason, ok := <-p.rejectChan
		if ok {
			newReason := fn(reason)
			reject(newReason)
		}
	}()
	return prom
}

func (p Go) Await() (interface{}, error) {
	// TODO support error handling inside promise functions instead
	value := <-p.resolveChan
	switch err := (<-p.rejectChan).(type) {
	case nil:
		return value, nil
	case error:
		return value, err
	default:
		return value, errors.Errorf("%v", err)
	}
}
