package deploy

import (
	"strings"
	"testing"
)

// TRADER-TEST-HANG (owner order 2026-10-02): vl/trader is the repo's slowest
// package — 1,417.6s on the live box at load 8.78 [A] — and local `go test`
// runs WITHOUT -timeout silently use the 10-minute default, so the package
// dies at the deadline and whichever test started last is blamed (it showed
// TestIdentityE6LegacyAndBootReads, which passes alone in seconds). CI pins
// -timeout 30m and is green. This pin keeps the repo's OWN runners (Makefile
// test targets) at CI parity, so the local recipe cannot regress to the
// default. The production call site is the Makefile on disk.
func TestMakefileTestTargetsCarryTheCITimeout(t *testing.T) {
	body := repoFile(t, "Makefile")
	found := 0
	for _, line := range strings.Split(body, "\n") {
		if !strings.Contains(line, "go test") {
			continue
		}
		found++
		if !strings.Contains(line, "-timeout 30m") {
			t.Fatalf("every Makefile go test line must carry -timeout 30m (CI parity): %q", line)
		}
	}
	if found < 2 {
		t.Fatalf("expected at least 2 go test lines in the Makefile, saw %d", found)
	}
}
