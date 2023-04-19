//go:build js
// +build js

package interop

import (
	"sync"
	"syscall/js"
)

type EventCallback func(event Event, args ...interface{})

type EventTarget interface {
	Listen(eventName string, callback EventCallback)
	Emit(event Event, args ...interface{})
}

func NewEventTarget() EventTarget {
	return &eventTarget{
		listeners: make(map[string][]EventCallback),
	}
}

type Event struct {
	Target js.Value
	Type   string
}

type eventTarget struct {
	mu        sync.Mutex
	listeners map[string][]EventCallback
}

func (e *eventTarget) Listen(eventName string, callback EventCallback) {
	e.mu.Lock()
	e.listeners[eventName] = append(e.listeners[eventName], callback)
	e.mu.Unlock()
}

func (e *eventTarget) Emit(event Event, args ...interface{}) {
	e.mu.Lock()
	listeners := e.listeners[event.Type]
	e.mu.Unlock()
	for _, l := range listeners {
		l(event, args...)
	}
}
