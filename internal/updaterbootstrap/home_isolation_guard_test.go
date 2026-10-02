package updaterbootstrap

// P0 test-isolation GUARD: the package's tests write the worker env and the
// last-authz files under throwaway homes (attended(t) isolates HOME + the
// userHomeDir seam). This TestMain snapshots the PROCESS home's
// ~/.config/vl-updater before the tests and fails the run if ANY file in it
// changed (created, deleted, content, or mtime) — the live box's config dir
// must survive every test run byte-identically. A test that forgets its
// isolation trips this guard (mutant-pinned: remove the Setenv in attended
// and the guard fails).

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type fileStamp struct {
	sha256 string
	mtime  int64
}

// configDirSig maps a rel path under ~/.config/vl-updater to its stamp; a
// missing dir reads as an empty map.
func configDirSig() map[string]fileStamp {
	sig := map[string]fileStamp{}
	home, err := os.UserHomeDir()
	if err != nil {
		return sig
	}
	dir := filepath.Join(home, ".config", "vl-updater")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return sig
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		fi, err := os.Stat(p)
		if err != nil {
			sig[e.Name()] = fileStamp{sha256: "unreadable", mtime: -1}
			continue
		}
		b, err := os.ReadFile(p)
		sum := "unreadable"
		if err == nil {
			h := sha256.Sum256(b)
			sum = hex.EncodeToString(h[:])
		}
		sig[e.Name()] = fileStamp{sha256: sum, mtime: fi.ModTime().UnixNano()}
	}
	return sig
}

func TestMain(m *testing.M) {
	before := configDirSig()
	code := m.Run()
	after := configDirSig()

	var changed []string
	for name, b := range before {
		if a, ok := after[name]; !ok {
			changed = append(changed, name+" DELETED during the tests")
		} else if a.sha256 != b.sha256 || a.mtime != b.mtime {
			changed = append(changed, name+" CHANGED during the tests")
		}
	}
	for name := range after {
		if _, ok := before[name]; !ok {
			changed = append(changed, name+" CREATED during the tests")
		}
	}
	sort.Strings(changed)
	if len(changed) > 0 {
		fmt.Fprintf(os.Stderr, "TEST-ISOLATION GUARD FAILED — the real ~/.config/vl-updater changed during the package tests:\n  %s\nEvery test that reaches the worker-env or last-authz paths must call attended(t) (which isolates HOME) or set its own t.Setenv(\"HOME\", t.TempDir()).\n", strings.Join(changed, "\n  "))
		os.Exit(1)
	}
	os.Exit(code)
}
