package kernel

import (
	"path/filepath"
	"testing"

	"nofx/internal/censuswalk"
)

// test-srctree-race (2) — a file that VANISHES between the census walk and the
// parse (a concurrent test writing into the source tree, or an external
// editor) is skipped and LOGGED — never a whole-scan failure. Production call
// site: parseAcceptanceFile, the per-file step acceptanceGuardOffenders runs.
func TestAcceptanceScannerToleratesVanishedFile(t *testing.T) {
	var logged int
	offs, vanished, err := parseAcceptanceFile(censuswalk.File{
		Path: filepath.Join(t.TempDir(), "api", "ghost.go"), // never created
		Rel:  "api/ghost.go",
	}, func(format string, args ...any) { logged++ })
	if err != nil {
		t.Fatalf("a vanished file must not fail the scan: %v", err)
	}
	if !vanished {
		t.Fatal("a vanished file must report vanished=true (skipped), not an error")
	}
	if len(offs) != 0 {
		t.Fatalf("a vanished file produced offenders: %v", offs)
	}
	if logged != 1 {
		t.Fatalf("the skip must be logged exactly once, got %d", logged)
	}
}
