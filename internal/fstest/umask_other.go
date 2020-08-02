// +build plan9 windows

package fstest

func setUmask(mask int) (oldmask int) {
	return 0
}
