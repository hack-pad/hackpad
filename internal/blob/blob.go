package blob

type Blob interface {
	jsExtensions
	Bytes() []byte
	Len() int
	Slice(start, end int64) (Blob, error)
	Set(w Blob, off int64) (n int, err error)
	Grow(off int64) error
	Truncate(size int64)
}
