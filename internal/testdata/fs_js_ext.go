// +build js,wasm

package syscall

func Pipe(fd []int) error {
	if jsFS.Get("pipe").IsUndefined() {
		// fs.pipe is unavailable on Node.js and JS
		// no known JS implementation exists, but is needed to run Go
		return ENOSYS
	}

	jsFD, err := fsCall("pipe")
	if err != nil {
		return err
	}
	fd[0] = jsFD.Index(0).Int()
	fd[1] = jsFD.Index(1).Int()
	return nil
}
