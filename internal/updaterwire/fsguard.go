package updaterwire

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

// Filesystem refusals shared by Dial and the worker's Listen. Each is a
// sentinel so a test can prove WHICH check refused.
var (
	ErrSymlink      = errors.New("updaterwire: path is a symlink")
	ErrNotSocket    = errors.New("updaterwire: path is not a unix socket")
	ErrNotDir       = errors.New("updaterwire: updater dir is not a directory")
	ErrForeignOwner = errors.New("updaterwire: not owned by this uid")
	ErrLoosePerms   = errors.New("updaterwire: permissions looser than owner-only")
	ErrBadPath      = errors.New("updaterwire: socket path must be absolute")
)

// fileOwner is the uid lookup seam (tests make one file read as another
// uid's without root). ok=false — no stat_t — is refused like a foreign owner.
var fileOwner = func(fi fs.FileInfo) (uint32, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return st.Uid, true
}

func checkOwner(fi fs.FileInfo, euid int) error {
	uid, ok := fileOwner(fi)
	if !ok || euid < 0 || uid != uint32(euid) {
		return ErrForeignOwner
	}
	return nil
}

// CheckPrivateDir refuses dir unless it is a real directory (not a symlink),
// owned by euid, with no group/other permission bits. The private dir is what
// closes the bind→chmod window on the socket: nobody else can reach inside.
func CheckPrivateDir(dir string, euid int) error {
	fi, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("updaterwire: updater dir: %w", err)
	}
	if fi.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("updater dir: %w", ErrSymlink)
	}
	if !fi.IsDir() {
		return ErrNotDir
	}
	if err := checkOwner(fi, euid); err != nil {
		return fmt.Errorf("updater dir: %w", err)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("updater dir %#o: %w", fi.Mode().Perm(), ErrLoosePerms)
	}
	return nil
}

// CheckSocketFile refuses path unless it is a unix socket (not a symlink, not
// a regular file), owned by euid, with no group/other permission bits. An
// absent path returns an error wrapping fs.ErrNotExist.
func CheckSocketFile(path string, euid int) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("updaterwire: socket: %w", err)
	}
	if fi.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("socket: %w", ErrSymlink)
	}
	if fi.Mode()&fs.ModeSocket == 0 {
		return ErrNotSocket
	}
	if err := checkOwner(fi, euid); err != nil {
		return fmt.Errorf("socket: %w", err)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("socket %#o: %w", fi.Mode().Perm(), ErrLoosePerms)
	}
	return nil
}
