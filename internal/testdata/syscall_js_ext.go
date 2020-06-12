// +build js,wasm

package syscall

const (
	LOCK_SH = 0x1
	LOCK_EX = 0x2
	LOCK_UN = 0x8
)

func Flock(fd, how int) error {
	if jsFS.Get("flock").IsUndefined() {
		// fs.flock is unavailable on Node.js and JS by default
		// typically it's included via node-fs-ext
		return ENOSYS
	}

	_, err := fsCall("flock", fd, how)
	return err
}
