package kernel

import (
	"path/filepath"
	"testing"
)

// Unset must consult exactly the historical path, and nothing else: that is
// what makes the goldens byte-identical.
func TestReleaseMarkerPathsUnsetIsExactlyTheHistoricalPath(t *testing.T) {
	t.Setenv("NOFX_RELEASE_DIR", "")
	got := releaseMarkerPaths()
	if len(got) != 1 || got[0] != "deploy/RELEASE" {
		t.Fatalf("unset must resolve to exactly [deploy/RELEASE], got %v", got)
	}
}

// Set, the ACTIVE release's marker is consulted FIRST: deploy/RELEASE in the
// working directory belongs to whatever tree the process was launched from and
// would answer for a different build.
func TestReleaseMarkerPathsPrefersTheActiveRelease(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NOFX_RELEASE_DIR", root)
	got := releaseMarkerPaths()
	if len(got) != 2 {
		t.Fatalf("want the active marker and the fallback, got %v", got)
	}
	if want := filepath.Join(root, "current", "RELEASE"); got[0] != want {
		t.Fatalf("first path = %q, want the ACTIVE release's marker %q", got[0], want)
	}
	if got[1] != "deploy/RELEASE" {
		t.Fatalf("fallback = %q, want deploy/RELEASE", got[1])
	}
}
