//go:build !linux

package updaterworker

import (
	"os"
	"path/filepath"
	"syscall"
)

// openTrustAnchor (non-Linux fallback; production is Linux, see
// sshsig_open_linux.go): the file's directory must be a real directory
// (Lstat, never a symlink), the file is opened with O_NOFOLLOW, and the
// directory must still be that same real directory afterwards.
func openTrustAnchor(path string) (*os.File, error) {
	dir := filepath.Clean(filepath.Dir(path))
	before, err := os.Lstat(dir)
	if err != nil {
		return nil, anchorOpenError(dir, true, err)
	}
	if !before.IsDir() {
		return nil, anchorOpenError(dir, true, syscall.ENOTDIR)
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, anchorOpenError(path, false, err)
	}
	if after, err := os.Lstat(dir); err != nil || !after.IsDir() || !os.SameFile(before, after) {
		f.Close()
		return nil, anchorOpenError(dir, true, syscall.ENOTDIR)
	}
	return f, nil
}
