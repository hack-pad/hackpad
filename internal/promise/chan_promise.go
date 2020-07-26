package promise

type GoPromise struct {
	resolveChan, rejectChan <-chan interface{}
}

func NewGoPromise() (resolve, reject Resolver, promise GoPromise) {
	resolveChan, rejectChan := make(chan interface{}, 1), make(chan interface{}, 1)
	promise.resolveChan, promise.rejectChan = resolveChan, rejectChan

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
	return
}

func (p GoPromise) Then(fn func(value interface{}) interface{}) GoPromise {
	// TODO support failing a Then call
	resolve, _, prom := NewGoPromise()
	go func() {
		value, ok := <-p.resolveChan
		if ok {
			newValue := fn(value)
			resolve(newValue)
		}
	}()
	return prom
}

func (p GoPromise) Catch(fn func(rejectedReason interface{}) interface{}) GoPromise {
	_, reject, prom := NewGoPromise()
	go func() {
		reason, ok := <-p.rejectChan
		if ok {
			newReason := fn(reason)
			reject(newReason)
		}
	}()
	return prom
}
