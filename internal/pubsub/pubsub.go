package pubsub

import (
	"context"
	"sync"
)

type PubSub interface {
	// Emit signals all Waiters waiting on 'key' will unblock.
	// Can not be called concurrently.
	Emit(key string)
	// Wait waits for the 'key' to be emitted or for Close() to be called
	Wait(key string)
}

type pubsub struct {
	mu          sync.RWMutex
	subscribers map[string][]context.CancelFunc
	visited     map[string]bool
	ctx         context.Context
}

// New creates a new PubSub that unblocks all calls to Wait when ctx is canceled
func New(ctx context.Context) PubSub {
	return &pubsub{
		ctx:         ctx,
		subscribers: make(map[string][]context.CancelFunc),
		visited:     make(map[string]bool),
	}
}

func (ps *pubsub) Emit(key string) {
	ps.mu.RLock()
	visited := ps.visited[key]
	ps.mu.RUnlock()
	if visited {
		return
	}
	ps.mu.Lock()
	ps.visited[key] = true
	funcs := ps.subscribers[key]
	ps.subscribers[key] = nil
	ps.mu.Unlock()
	for _, cancel := range funcs {
		cancel()
	}
}

func (ps *pubsub) Wait(key string) {
	select {
	case <-ps.ctx.Done():
		return
	default:
	}

	ps.mu.Lock()
	if ps.visited[key] {
		ps.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(ps.ctx)
	ps.subscribers[key] = append(ps.subscribers[key], cancel)
	ps.mu.Unlock()

	select {
	case <-ps.ctx.Done():
	case <-ctx.Done():
	}
}
