package pubsub

import (
	"context"
	"testing"
)

func TestPubSub(t *testing.T) {
	t.Run("emit first", func(t *testing.T) {
		ps := New(context.Background())
		ps.Emit("hi")
		ps.Wait("hi")
	})

	t.Run("wait first", func(t *testing.T) {
		ps := New(context.Background())
		wait := make(chan struct{})
		completed := make(chan struct{})
		go func() {
			close(wait)
			ps.Wait("hi")
			close(completed)
		}()
		<-wait
		for _, key := range []string{"a", "b", "c", "hi"} {
			ps.Emit(key)
		}
		<-completed
	})
}
