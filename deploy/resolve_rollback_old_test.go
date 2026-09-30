package deploy

// UPDATER-USABLE-V1 (item C) — release.yml must build and ship the updater
// binaries, and dbcompat's OLD must come from the one resolver: the running
// RELEASE passed as rollback_from, else the highest-version v* tag that is an
// ANCESTOR of the new commit — never a sideline tag and never an ancient
// v1.0-* tag. The resolver is exercised at its production call site (the
// real script, run with bash against a real temp repository).

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseWorkflowBuildsAndShipsTheUpdaterBinaries(t *testing.T) {
	y := repoFile(t, ".github/workflows/release.yml")
	for _, want := range []string{
		"go build -trimpath -o \"$OUT/vl-updater\" ./cmd/vl-updater",
		"go build -trimpath -o \"$OUT/vl-updater-bootstrap\" ./cmd/vl-updater-bootstrap",
		"cp \"$OUT/vl-updater\" \"$OUT/vl-updater-bootstrap\" updater/",
		"vcs.modified=false",
		"resolve-rollback-old.sh",
		"v1.0-*",
	} {
		if !strings.Contains(y, want) {
			t.Fatalf("release.yml must carry %q", want)
		}
	}
	// The build must precede the staging, or the binaries would not ship.
	// (The copy INTO updater/ is what ships them; the build writes outside
	// the tree — see TestReleaseWorkflowGoBuildsWriteOutsideTheTree.)
	cp := `cp "$OUT/vl-updater" "$OUT/vl-updater-bootstrap" updater/`
	if strings.Index(y, cp) == -1 ||
		strings.Index(y, cp) > strings.Index(y, "Stage ONLY the allow-list") {
		t.Fatalf("the updater build step must come before the staging step")
	}
}

// gitRuns runs git in dir; the args come from a variadic so each call site
// stays a single slice (this box's toolchain refuses two-literals+spread).
func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// resolveRuns the resolver in dir with the given positional args.
func resolveRun(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()
	// Absolute: bash resolves the script path against cmd.Dir (the temp
	// repo), not against this test's own cwd.
	script, err := filepath.Abs(filepath.Join("..", "deploy", "release", "resolve-rollback-old.sh"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", append([]string{script}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	rc := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			rc = ee.ExitCode()
		} else {
			t.Fatalf("running resolver: %v", err)
		}
	}
	return strings.TrimSpace(string(out)), rc
}

func makeHistoryRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q", "-b", "main")
	gitRun(t, dir, "config", "user.email", "t@t")
	gitRun(t, dir, "config", "user.name", "t")
	commit := func(msg string) string {
		if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte(msg), 0o644); err != nil {
			t.Fatal(err)
		}
		gitRun(t, dir, "add", "f.txt")
		gitRun(t, dir, "commit", "-q", "-m", msg)
		return gitRun(t, dir, "rev-parse", "HEAD")
	}
	c1 := commit("one")
	gitRun(t, dir, "tag", "v1.0.0", c1)
	c2 := commit("two")
	gitRun(t, dir, "tag", "v2.0.1", c2)
	c3 := commit("three")
	gitRun(t, dir, "tag", "-a", "v3.0.0", "-m", "release 3", c3) // annotated: the resolver must peel
	c4 := commit("four")
	gitRun(t, dir, "tag", "v4.0.0", c4) // the current tag

	// A sideline tag with a HIGHER version than anything on the line: it is
	// not an ancestor of c4, so it must never be chosen.
	gitRun(t, dir, "checkout", "-q", c1)
	commitSide := commit("sideline")
	gitRun(t, dir, "tag", "v9.9.9", commitSide)
	gitRun(t, dir, "checkout", "-q", "main")

	// Tuck the fixture's pieces into the repo's own layout for the test.
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte(c4), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestResolveRollbackOld(t *testing.T) {
	t.Run("picks the highest-version ancestor, skipping the sideline and v1.0", func(t *testing.T) {
		dir := makeHistoryRepo(t)
		newSHA := gitRun(t, dir, "rev-parse", "v4.0.0^{commit}")
		want := gitRun(t, dir, "rev-parse", "v3.0.0^{commit}")
		got, rc := resolveRun(t, dir, newSHA, "v4.0.0", "")
		if rc != 0 || got != want {
			t.Fatalf("rc=%d got=%q want=%q", rc, got, want)
		}
	})

	t.Run("rollback_from wins verbatim when it is 40 hex", func(t *testing.T) {
		dir := makeHistoryRepo(t)
		newSHA := gitRun(t, dir, "rev-parse", "v4.0.0^{commit}")
		from := gitRun(t, dir, "rev-parse", "v2.0.1^{commit}")
		got, rc := resolveRun(t, dir, newSHA, "v4.0.0", from)
		if rc != 0 || got != from {
			t.Fatalf("rc=%d got=%q want=%q", rc, got, from)
		}
	})

	t.Run("a non-hex rollback_from is ignored, not trusted", func(t *testing.T) {
		dir := makeHistoryRepo(t)
		newSHA := gitRun(t, dir, "rev-parse", "v4.0.0^{commit}")
		want := gitRun(t, dir, "rev-parse", "v3.0.0^{commit}")
		got, rc := resolveRun(t, dir, newSHA, "v4.0.0", "not-a-sha")
		if rc != 0 || got != want {
			t.Fatalf("rc=%d got=%q want=%q", rc, got, want)
		}
	})

	t.Run("only v1.0-* ancestors ⇒ nothing — the pair is absent, never fabricated", func(t *testing.T) {
		dir := t.TempDir()
		gitRun(t, dir, "init", "-q", "-b", "main")
		gitRun(t, dir, "config", "user.email", "t@t")
		gitRun(t, dir, "config", "user.name", "t")
		commit := func(msg string) string {
			if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte(msg), 0o644); err != nil {
				t.Fatal(err)
			}
			gitRun(t, dir, "add", "f.txt")
			gitRun(t, dir, "commit", "-q", "-m", msg)
			return gitRun(t, dir, "rev-parse", "HEAD")
		}
		gitRun(t, dir, "tag", "v1.0.5", commit("old"))
		newSHA := commit("new")
		gitRun(t, dir, "tag", "v1.1.0", newSHA)
		got, rc := resolveRun(t, dir, newSHA, "v1.1.0", "")
		if rc != 0 || got != "" {
			t.Fatalf("rc=%d got=%q want empty", rc, got)
		}
	})

	t.Run("a non-hex new sha is a usage refusal", func(t *testing.T) {
		got, rc := resolveRun(t, t.TempDir(), "deadbeef", "v1.0.0", "")
		if rc != 2 || !strings.Contains(got, "40 hex") {
			t.Fatalf("rc=%d got=%q", rc, got)
		}
	})
}
