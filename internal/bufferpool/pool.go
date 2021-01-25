package bufferpool

import (
	"go.uber.org/atomic"
)

type Pool struct {
	bufferCount atomic.Int64
	bufferSize  uint64
	buffers     chan *Buffer
}

type Buffer struct {
	Data []byte
	pool *Pool
}

func New(bufferSize, maxBuffers uint64) *Pool {
	if maxBuffers == 0 {
		maxBuffers = 1
	}
	p := &Pool{
		bufferSize: bufferSize,
		buffers:    make(chan *Buffer, maxBuffers),
	}
	p.addBuffer() // start with 1 buffer, ready to go
	return p
}

func (p *Pool) addBuffer() {
	for {
		count := p.bufferCount.Load()
		if int(count) == cap(p.buffers) {
			return // already at max buffers, no-op
		}
		if p.bufferCount.CAS(count, count+1) {
			break // successfully provisioned slot for new buffer
		}
	}
	buf := &Buffer{
		Data: make([]byte, p.bufferSize),
		pool: p,
	}
	p.buffers <- buf
}

func (p *Pool) Wait() *Buffer {
	select {
	case buf := <-p.buffers:
		return buf
	default:
		p.addBuffer()
		// may not always get the new buffer, but looping could allocate more buffers far too quickly
		return <-p.buffers
	}
}

func (b *Buffer) Done() {
	b.pool.buffers <- b
}
