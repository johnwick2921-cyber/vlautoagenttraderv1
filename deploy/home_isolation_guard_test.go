package deploy

// P0 test-isolation GUARD (TEST-WROTE-REAL-WORKER-ENV-2, owner order
// 2026-10-02): this package runs the REAL deploy scripts (install-updater-
// worker.sh, migrate-to-vl.sh, lock scripts) that read and write $HOME. The
// shared guard snapshots the real HOME's updater paths before the tests and
// fails the run if any changed — every subprocess run must keep its
// scratch-HOME override.

import (
	"os"
	"testing"

	"vl/internal/testhome"
)

func TestMain(m *testing.M) {
	os.Exit(testhome.Guard(m))
}
