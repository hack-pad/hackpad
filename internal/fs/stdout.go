package fs

import (
	"bytes"
	"sync"
	"time"

	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpadfs"
)

var (
	stdout hackpadfs.File = &bufferedLogger{name: "dev/stdout", printFn: log.Print}
	stderr hackpadfs.File = &bufferedLogger{name: "dev/stderr", printFn: log.Error}
)

type bufferedLogger struct {
	unimplementedFile

	name      string
	printFn   func(...interface{}) int
	mu        sync.Mutex
	buf       bytes.Buffer
	timerOnce sync.Once
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

func (b *bufferedLogger) Print(s string) int {
	n, _ := b.Write([]byte(s))
	return n
}

func (b *bufferedLogger) Write(p []byte) (n int, err error) {
	b.timerOnce.Do(func() {
		const waitTime = time.Second / 2
		go func() {
			ticker := time.NewTicker(waitTime)
			for range ticker.C {
				b.flush()
			}
		}()
	})

	b.mu.Lock()
	_, _ = b.buf.Write(p) // at time of writing, bytes.Buffer.Write cannot return an error
	b.mu.Unlock()
	return len(p), nil
}

func (b *bufferedLogger) Name() string {
	return b.name
}

func (b *bufferedLogger) Close() error {
	// TODO prevent writes and return os.ErrClosed
	return nil
}
