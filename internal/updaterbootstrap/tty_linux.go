//go:build linux

package updaterbootstrap

import (
	"io"
	"os"
	"syscall"
	"unsafe"
)

// stdinIsTerminal reports whether r is an *os.File open on a terminal
// (TCGETS succeeds). A pipe, a regular file or any non-file reader is not.
func stdinIsTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok || f == nil {
		return false
	}
	var t syscall.Termios
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TCGETS, uintptr(unsafe.Pointer(&t)))
	return e == 0
}
