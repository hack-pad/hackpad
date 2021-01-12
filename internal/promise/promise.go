package promise

type Promise interface {
	Then(fn func(value interface{}) interface{}) Promise
	Catch(fn func(value interface{}) interface{}) Promise
	Await() (interface{}, error)
}
