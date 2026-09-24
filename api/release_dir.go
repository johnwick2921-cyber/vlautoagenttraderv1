package api

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// NOFX_RELEASE_DIR turns the install into VERSIONED runtimes:
//
//	NOFX_RELEASE_DIR/<sha>/{nofx-bin,web/dist,RELEASE,manifest.json}
//	NOFX_RELEASE_DIR/current -> <sha>
//
// with `current` a symlink, so an activation or a rollback moves all three
// halves at once and can never leave a mixed install.
//
// It is resolved ONCE, at startup. A path that can change under a running
// process is a path that will be read differently by two parts of the same
// boot, and then nothing agrees about what is being served.
//
// UNSET is the default and means today's behaviour, byte for byte.
var (
	releaseDirOnce sync.Once
	releaseDirVal  string
)

// ReleaseDir returns the configured release root, or "" when unset.
func ReleaseDir() string {
	releaseDirOnce.Do(func() {
		releaseDirVal = strings.TrimSpace(os.Getenv("NOFX_RELEASE_DIR"))
	})
	return releaseDirVal
}

// CurrentReleaseDir is NOFX_RELEASE_DIR/current, or "" when unset.
func CurrentReleaseDir() string {
	root := ReleaseDir()
	if root == "" {
		return ""
	}
	return filepath.Join(root, "current")
}

// ResolvedDistDir is where the served bundle actually lives: under the active
// release when NOFX_RELEASE_DIR is set, else the working-directory-relative
// path this process has always used.
func ResolvedDistDir() string {
	cur := CurrentReleaseDir()
	if cur == "" {
		return UIDistDir
	}
	return filepath.Join(cur, UIDistDir)
}

// resetReleaseDirForTest lets a test vary the environment. Production never
// calls it: the whole point of the sync.Once is that the value cannot move
// under a running process.
func resetReleaseDirForTest() {
	releaseDirOnce = sync.Once{}
	releaseDirVal = ""
}
