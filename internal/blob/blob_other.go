// +build !js

package blob

type jsExtensions interface{}

type blob struct {
	bytes []byte
}

func NewFromBytes(buf []byte) Blob {
	return &blob{buf}
}

func (b *blob) Bytes() []byte {
	return b.bytes
}

func (b *blob) Len() int {
	return len(b.bytes)
}

func (b *blob) Slice(start, end int64) (_ Blob, err error) {
	return NewFromBytes(b.bytes[start:end]), nil
}

func (b *blob) Set(w Blob, off int64) (n int, err error) {
	n = copy(b.bytes[off:], w.Bytes())
	return n, nil
}

func (b *blob) Grow(off int64) error {
	b.bytes = append(b.bytes, make([]byte, off)...)
	return nil
}

func (b *blob) Truncate(size int64) {
	if int64(len(b.bytes)) < size {
		return
	}
	b.bytes = b.bytes[:size]
}
