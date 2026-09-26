//go:build linux

package updaterworker

import (
	"os"
	"path/filepath"
	"syscall"
)

// openTrustAnchor opens the allowed-signers file at path for reading, never
// through a symlink: its DIRECTORY is opened first with O_NOFOLLOW|O_DIRECTORY
// (a symlinked <install>/deploy fails — U4F defect 3, probe P4) and the file is
// then opened RELATIVE to that directory with O_NOFOLLOW (openat), so a swap
// of either path element after the checks changes nothing that is read.
// O_NONBLOCK: a FIFO cannot block the read.
func openTrustAnchor(path string) (*os.File, error) {
	dir, name := filepath.Split(path)
	dir = filepath.Clean(dir)
	var dfd int
	err := retryEINTR(func() (err error) {
		dfd, err = syscall.Open(dir, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		return err
	})
	if err != nil {
		return nil, anchorOpenError(dir, true, err)
	}
	defer syscall.Close(dfd)
	var fd int
	err = retryEINTR(func() (err error) {
		fd, err = syscall.Openat(dfd, name, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
		return err
	})
	if err != nil {
		return nil, anchorOpenError(path, false, err)
	}
	return os.NewFile(uintptr(fd), path), nil
}

func retryEINTR(f func() error) error {
	for {
		if err := f(); err != syscall.EINTR {
			return err
		}
	}
}
