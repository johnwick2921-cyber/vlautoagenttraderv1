// Package testhome is the shared P0 test-isolation guard for every package
// whose tests can reach the REAL updater worker state under the process HOME
// (~/.config/vl-updater/*, ~/bin/vl-updater). A package installs it in its
// TestMain:
//
//	func TestMain(m *testing.M) { os.Exit(testhome.Guard(m)) }
//
// Guard snapshots the real paths before the tests and fails the run if ANY
// of them changed (created, deleted, content, or mtime) — the live box's
// worker credential and binary must survive every test run byte-identically.
// A test that forgets its HOME isolation trips the guard: every test that can
// reach these paths must set t.Setenv("HOME", t.TempDir()) (or run through a
// package seam that does).
//
// Born from TEST-WROTE-REAL-WORKER-ENV-2 (owner P0 2026-10-02): a root
// package test ran the real enroll CLI and rewrote the live
// ~/.config/vl-updater/env with a freshly minted test token. The earlier
// guard covered internal/updaterbootstrap only; this helper is the widened
// version any package can mount.
package testhome

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

// watchedPaths are the real-HOME paths the guard snapshots: every file under
// ~/.config/vl-updater, and the installed worker binary ~/bin/vl-updater.
// Each rel path is resolved against the process HOME at run time.
var watchedPaths = []string{
	".config/vl-updater", // directory — every file inside is stamped
	"bin/vl-updater",     // file — stamped if it exists
}

// Sig maps a rel path under the real HOME to its stamp. A missing file or
// dir reads as absent (no entry), never fabricated.
func Sig() map[string]fileStamp {
	sig := map[string]fileStamp{}
	home, err := os.UserHomeDir()
	if err != nil {
		return sig
	}
	var stamp func(rel string)
	stamp = func(rel string) {
		p := filepath.Join(home, filepath.FromSlash(rel))
		fi, err := os.Stat(p)
		if err != nil {
			return // absent: no entry
		}
		if fi.IsDir() {
			entries, err := os.ReadDir(p)
			if err != nil {
				return
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				stamp(rel + "/" + e.Name())
			}
			return
		}
		b, err := os.ReadFile(p)
		sum := "unreadable"
		if err == nil {
			h := sha256.Sum256(b)
			sum = hex.EncodeToString(h[:])
		}
		sig[rel] = fileStamp{sha256: sum, mtime: fi.ModTime().UnixNano()}
	}
	for _, rel := range watchedPaths {
		stamp(rel)
	}
	return sig
}

// Guard runs the package tests with a before/after signature over the real
// HOME's watched paths and fails the run (rc 1) when anything changed.
func Guard(m *testing.M) int {
	before := Sig()
	code := m.Run()
	after := Sig()

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
		fmt.Fprintf(os.Stderr, "TEST-ISOLATION GUARD FAILED — the real HOME's updater paths changed during the package tests:\n  %s\nEvery test that reaches the worker env, last-authz, or worker binary paths must isolate HOME (t.Setenv(\"HOME\", t.TempDir()) or the package's own seam).\n", strings.Join(changed, "\n  "))
		return 1
	}
	return code
}
