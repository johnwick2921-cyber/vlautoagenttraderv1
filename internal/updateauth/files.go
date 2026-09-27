package updateauth

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// geteuid is a seam so the wrong-owner refusal is testable unprivileged.
var geteuid = os.Geteuid

// ErrUnsafe wraps every refusal of a file or directory whose type, mode or
// owner is not exactly what enrollment writes.
var ErrUnsafe = errors.New("updateauth: unsafe file")

func unsafeErr(p, why string) error { return fmt.Errorf("%w: %s: %s", ErrUnsafe, p, why) }

// checkPrivateDir refuses a directory that is a symlink, not a directory,
// group/other-accessible (looser than 0700) or owned by another uid.
func checkPrivateDir(dir string) error {
	fi, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return unsafeErr(dir, "is a symlink")
	}
	if !fi.IsDir() {
		return unsafeErr(dir, "is not a directory")
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return unsafeErr(dir, fmt.Sprintf("mode %04o is looser than 0700", fi.Mode().Perm()))
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return unsafeErr(dir, "owner unknown")
	}
	if int(st.Uid) != geteuid() {
		return unsafeErr(dir, fmt.Sprintf("owned by uid %d, not %d", st.Uid, geteuid()))
	}
	return nil
}

// readPrivateFile reads p only when its directory passes checkPrivateDir and
// p itself is a regular file (never a symlink: O_NOFOLLOW; never a FIFO or
// device: checked on the OPENED descriptor, so a swap between check and open
// cannot slip past), mode no looser than 0600, owned by the effective uid,
// and at most max bytes.
func readPrivateFile(p string, max int64) ([]byte, error) {
	if err := checkPrivateDir(filepath.Dir(p)); err != nil {
		return nil, err
	}
	// O_NONBLOCK: opening a FIFO for reading would otherwise block until a
	// writer appears; the fstat below then refuses it.
	f, err := os.OpenFile(p, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) {
			return nil, unsafeErr(p, "is a symlink")
		}
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, unsafeErr(p, "is not a regular file")
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return nil, unsafeErr(p, fmt.Sprintf("mode %04o is looser than 0600", fi.Mode().Perm()))
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, unsafeErr(p, "owner unknown")
	}
	if int(st.Uid) != geteuid() {
		return nil, unsafeErr(p, fmt.Sprintf("owned by uid %d, not %d", st.Uid, geteuid()))
	}
	if fi.Size() > max {
		return nil, unsafeErr(p, "too large")
	}
	b, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, unsafeErr(p, "too large")
	}
	return b, nil
}

// ensurePrivateDir creates <dataDir>/updater at 0700 when absent and then
// requires it to pass checkPrivateDir (an existing looser/foreign/symlinked
// dir is refused, never "repaired").
func ensurePrivateDir(dataDir string) (string, error) {
	dir := Dir(dataDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := checkPrivateDir(dir); err != nil {
		return "", err
	}
	return dir, nil
}

// lockFile takes an exclusive flock on p (created 0600, never through a
// symlink) and returns its unlock.
func lockFile(p string) (func(), error) {
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("updateauth: lock: %w", err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}

// writeAtomic is the store/maintenance_hold.go write sequence: temp file in
// the same dir (CreateTemp is 0600 from birth), write, fsync, close, chmod
// 0600, rename over the target, fsync the dir. A reader sees the old file or
// the new file, never a partial one; rename replaces a symlink at the target
// rather than writing through it.
func writeAtomic(dir, target string, b []byte) error {
	f, err := os.CreateTemp(dir, ".updateauth-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp) // no-op after a successful rename
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		return err
	}
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		d.Close()
	}
	return nil
}
