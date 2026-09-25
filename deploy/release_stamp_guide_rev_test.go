package deploy

// W-PR-B fold [27] — web/scripts/stamp-guide-rev.sh seds a literal that no
// longer exists (types.ts is `resolveGuideBuiltRev()` since the build-time
// stamp landed), so it changes NOTHING while printing "→ <rev>" — a stamp that
// reports success without changing a byte. It also stamps a 12-char
// /api/health revision the build gate refuses (the gate demands 40-hex
// VITE_GUIDE_BUILT_REV at build time).
//
// The honest resolution: DELETE the dead script and every operator-facing
// reference to it, and pin that no reference can come back. The successor
// mechanism is the build-time input (VITE_GUIDE_BUILT_REV at npm run build),
// already census-covered.
//
// Class-250 probe: the test greps the real tree. The mutation that proves it:
// recreating the script or re-adding any operator-facing reference to it fails
// the assertions below. (Historical reports under docs/superpowers/reports/ are
// archives and excluded.)

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStampGuideRevIsDeletedAndNeverRerouted(t *testing.T) {
	root := repoRoot(t)

	if _, err := os.Stat(filepath.Join(root, "web/scripts/stamp-guide-rev.sh")); err == nil {
		t.Fatal("web/scripts/stamp-guide-rev.sh still exists — it reports a stamp while changing nothing (sed target gone)")
	}

	// zero references anywhere outside the historical reports archive
	out, err := exec.Command("grep", "-rIn", "stamp-guide-rev", root).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
			t.Fatalf("grep failed: %v", err)
		}
		out = nil // exit 1 = no matches, which is exactly what we require
	}
	allowedPrefix := filepath.Join(root, "docs", "superpowers", "reports") + string(os.PathSeparator)
	for _, ln := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if ln == "" {
			continue
		}
		if strings.HasPrefix(ln, allowedPrefix) {
			continue // historical archive, not an operator route
		}
		t.Fatalf("a live reference to the deleted stamp script remains:\n  %s", ln)
	}

	// the successor mechanism must be what the docs point at instead
	readme, err := os.ReadFile(filepath.Join(root, "deploy", "release", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "VITE_GUIDE_BUILT_REV=<sha> npm run build") {
		t.Fatalf("the README must document the build-time stamp (VITE_GUIDE_BUILT_REV=<sha> npm run build)")
	}
}
