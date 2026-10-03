package main

// P0 test-isolation GUARD (TEST-WROTE-REAL-WORKER-ENV-2, owner order
// 2026-10-02): this package's tests run the REAL enrollment CLI
// (updaterbootstrap.Run … enroll …), which writes the worker credential to
// ~/.config/vl-updater/env under the process HOME. The shared guard snapshots
// the real HOME's updater paths before the tests and fails the run if any
// changed — a test that forgets t.Setenv("HOME", t.TempDir()) trips it
// (mutant-pinned: remove the HOME seam from
// TestUpdaterBootstrapEnrollsWhereTheBotReads and the guard fails).

import (
	"os"
	"testing"

	"vl/internal/testhome"
)

func TestMain(m *testing.M) {
	os.Exit(testhome.Guard(m))
}
