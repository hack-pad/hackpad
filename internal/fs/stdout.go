package fs

import (
	"bytes"
	"sync"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/log"
)

var (
	stdout = bufferedLogger{printFn: log.Print}
	stderr = bufferedLogger{printFn: log.Error}
)

type bufferedLogger struct {
	printFn   func(...interface{}) int
	mu        sync.Mutex
	buf       bytes.Buffer
	timerOnce sync.Once
	timerID   js.Value
}

func (b *bufferedLogger) flush() {
	if b.buf.Len() == 0 {
		return
	}

	const maxBufLen = 4096

	b.mu.Lock()
	i := bytes.LastIndexByte(b.buf.Bytes(), '\n')
	var buf []byte
	if i == -1 || b.buf.Len() > maxBufLen {
		buf = b.buf.Bytes()
		b.buf.Reset()
	} else {
		i++ // include newline char if present
		buf = make([]byte, i)
		n, _ := b.buf.Read(buf) // at time of writing, only io.EOF can be returned -- which we don't need
		buf = buf[:n]
	}
	b.mu.Unlock()
	if len(buf) != 0 {
		b.printFn(string(buf))
	}
}

func (b *bufferedLogger) jsFlush(this js.Value, args []js.Value) interface{} {
	b.flush()
	return nil
}

func (b *bufferedLogger) Print(s string) int {
	b.timerOnce.Do(func() {
		const waitTime = time.Second / 2
		b.timerID = js.Global().Call("setInterval", js.FuncOf(b.jsFlush), waitTime.Milliseconds())
	})

	b.mu.Lock()
	_, _ = b.buf.WriteString(s) // at time of writing, bytes.Buffer.WriteString cannot return an error
	b.mu.Unlock()
	return len(s)
}
