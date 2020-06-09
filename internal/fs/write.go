package fs

import (
	"bytes"
	"io"
	"sync"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/log"
	"github.com/pkg/errors"
)

func writeSync(args []js.Value) (interface{}, error) {
	ret, err := write(args)
	if len(ret) > 1 {
		return ret[0], err
	}
	return ret, err
}

func write(args []js.Value) ([]interface{}, error) {
	// args: fd, buffer, offset, length, position
	if len(args) < 2 {
		return nil, errors.Errorf("missing required args, expected fd and buffer: %+v", args)
	}
	fd := uint64(args[0].Int())
	jsBuffer := args[1]
	offset := 0
	if len(args) >= 3 {
		offset = args[2].Int()
	}
	length := jsBuffer.Length()
	if len(args) >= 4 {
		length = args[3].Int()
	}
	var position *int64
	if len(args) >= 5 && args[4].Type() == js.TypeNumber {
		position = new(int64)
		*position = int64(args[4].Int())
	}

	buffer := make([]byte, length)
	js.CopyBytesToGo(buffer, jsBuffer)
	if fd < minFD {
		var n int
		switch fd {
		case 2:
			n = stderr.Print(string(buffer))
		default:
			n = stdout.Print(string(buffer))
		}
		return []interface{}{n}, nil
	}
	n, err := Write(fd, buffer, offset, length, position)
	js.CopyBytesToJS(jsBuffer, buffer)
	return []interface{}{n, jsBuffer}, err
}

func Write(fd uint64, buffer []byte, offset, length int, position *int64) (n int, err error) {
	fileDescriptor := fileDescriptorIDs[fd]
	if fileDescriptor == nil {
		return 0, errors.Errorf("unknown fd %d", fd)
	}
	// 'offset' in Node.js's read is the offset in the buffer to start writing at,
	// and 'position' is where to begin reading from in the file.
	if position != nil {
		_, err := fileDescriptor.file.Seek(*position, io.SeekStart)
		if err != nil {
			return 0, err
		}
	}
	n, err = fileDescriptor.file.Write(buffer[offset : offset+length])
	if err == io.EOF {
		err = nil
	}
	return
}

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
