package api

import (
	"os"
	"path/filepath"
	"testing"
)

// L4: the knob is OFF by default and OFF must be byte-identical to before.
// This is the test that makes "additive" a fact rather than an intention.
func TestUnsetReleaseDirServesExactlyTheOldPath(t *testing.T) {
	t.Setenv("NOFX_RELEASE_DIR", "")
	resetReleaseDirForTest()
	if got := ResolvedDistDir(); got != UIDistDir {
		t.Fatalf("ResolvedDistDir() = %q with the knob unset, want the historical %q", got, UIDistDir)
	}
	if got := CurrentReleaseDir(); got != "" {
		t.Fatalf("CurrentReleaseDir() = %q with the knob unset, want empty", got)
	}
}

func TestSetReleaseDirServesFromTheActiveRelease(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NOFX_RELEASE_DIR", root)
	resetReleaseDirForTest()
	want := filepath.Join(root, "current", UIDistDir)
	if got := ResolvedDistDir(); got != want {
		t.Fatalf("ResolvedDistDir() = %q, want %q", got, want)
	}
}

// Resolved ONCE: a path that can change under a running process is a path two
// parts of the same boot will read differently, and then nothing agrees about
// what is being served.
func TestReleaseDirIsResolvedOnceAndDoesNotMoveUnderTheProcess(t *testing.T) {
	first := t.TempDir()
	t.Setenv("NOFX_RELEASE_DIR", first)
	resetReleaseDirForTest()
	before := ResolvedDistDir()

	// Change the environment the way a careless caller might.
	if err := os.Setenv("NOFX_RELEASE_DIR", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if after := ResolvedDistDir(); after != before {
		t.Fatalf("the resolved dist moved under the process: %q -> %q", before, after)
	}
}
