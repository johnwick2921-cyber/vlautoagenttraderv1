package updatersource

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReleaseRepoMatchesWorkflowContract (fold B1): the ONE
// build-time release-repo constant must equal release.yml's RELEASE_REPO —
// a box checks releases for the repo the workflow publishes under, never
// another. The workflow file is read relative to the repo root (the module
// root in tests), like the existing workflow contract tests do.
func TestReleaseRepoMatchesWorkflowContract(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	const envLine = "  RELEASE_REPO: "
	idx := strings.Index(string(b), envLine)
	if idx < 0 {
		t.Fatalf("release.yml has no %q line", strings.TrimSpace(envLine))
	}
	rest := string(b)[idx+len(envLine):]
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		rest = rest[:nl]
	}
	got := strings.TrimSpace(rest)
	if got != ReleaseRepo {
		t.Fatalf("release.yml RELEASE_REPO = %q, ReleaseRepo = %q", got, ReleaseRepo)
	}
}

// repoRoot walks up from the test's working directory to the module root
// (go.mod).
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test working directory")
		}
		dir = parent
	}
}
