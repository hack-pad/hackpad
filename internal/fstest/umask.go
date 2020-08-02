// +build !plan9,!windows

package fstest

import "syscall"

func setUmask(mask int) (oldmask int) {
	return syscall.Umask(mask)
}
