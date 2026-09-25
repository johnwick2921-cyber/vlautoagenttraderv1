package deploy

// W-PR-B fold [9] — a rollback step must prove the binary OPENED the database
// on THIS boot, and any FATAL fails the step. Before the fix, steps 2 and 3
// passed on artifacts inherited from step 1: the db file exists, the old regex
// misses sqlite's "duplicate column name", and the 5-table count was satisfied
// by step 1's own schema — a binary that never opened the database (or died
// with an unrecognized migration FATAL) was stamped tested:true.
//
// Class-250 probe: THIS TEST IS THE CALLER. It extracts the REAL migrate_with
// function out of deploy/release/db-compat.sh and executes those exact lines
// against stub binaries. The mutations that prove it:
//   (a) a boot with NO "✅ Database initialized" marker must fail — the old
//       code passed it on inherited artifacts;
//   (b) a boot that FATALs with "duplicate column name" (absent from the old
//       regex) must fail — the old code passed it.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// extractMigrateWith returns the literal migrate_with function from
// db-compat.sh — the production lines, not a re-implementation (canon 53).
func extractMigrateWith(t *testing.T, src string) string {
	t.Helper()
	lines := strings.Split(src, "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "migrate_with() {") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("migrate_with not found in db-compat.sh")
	}
	for i := start + 1; i < len(lines); i++ {
		if lines[i] == "}" {
			return strings.Join(lines[start:i+1], "\n")
		}
	}
	t.Fatalf("migrate_with has no closing brace")
	return ""
}

func makeFiveTableDB(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(dir, "data", "data.db")
	for i := 1; i <= 5; i++ {
		cmd := exec.Command("sqlite3", db, fmt.Sprintf("CREATE TABLE t%d (id INTEGER PRIMARY KEY);", i))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("sqlite3 create t%d: %v\n%s", i, err, out)
		}
	}
	return db
}

func runMigrateWith(t *testing.T, fn string, bin, dir, label string) (string, int) {
	t.Helper()
	work := t.TempDir()
	harness := fmt.Sprintf("#!/usr/bin/env bash\nset -uo pipefail\nWORK=%q; NETNS=\"\"; RSA_PRIVATE_KEY=\"\"; DATA_ENCRYPTION_KEY=\"\"; JWT_SECRET=\"\"; BOOT_SECS=5\n%s\nmigrate_with %q %q %q\n",
		work, fn, bin, dir, label)
	hp := filepath.Join(work, "h.sh")
	if err := os.WriteFile(hp, []byte(harness), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", hp)
	out, err := cmd.CombinedOutput()
	rc := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			rc = ee.ExitCode()
		} else {
			t.Fatalf("harness: %v\n%s", err, out)
		}
	}
	return string(out), rc
}

func TestDbcompatStepRequiresTheDatabaseToOpen(t *testing.T) {
	fn := extractMigrateWith(t, repoFile(t, "deploy/release/db-compat.sh"))
	stubDir := t.TempDir()

	writeStub := func(name, body string) string {
		t.Helper()
		p := filepath.Join(stubDir, name)
		if err := os.WriteFile(p, []byte("#!/usr/bin/env bash\n"+body), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	// (a) a binary that never opens the database must fail the step.
	dirA := t.TempDir()
	makeFiveTableDB(t, dirA)
	out, rc := runMigrateWith(t, fn, "/bin/true", dirA, "probe-never-opens")
	if rc == 0 {
		t.Fatalf("a boot that never opens the database passed the step on inherited artifacts:\n%s", out)
	}
	if !strings.Contains(out, "never opened the database") {
		t.Fatalf("the refusal must name the missing open proof; got:\n%s", out)
	}

	// (b) a FATAL the old regex misses ("duplicate column name") must fail.
	dirB := t.TempDir()
	makeFiveTableDB(t, dirB)
	fatalBin := writeStub("fatal-before-open.sh", `printf '%s\n' "[FATA] store: duplicate column name: foo" >&2; exit 1`)
	out, rc = runMigrateWith(t, fn, fatalBin, dirB, "probe-unmatched-fatal")
	if rc == 0 {
		t.Fatalf("a boot that FATALs before opening the database passed the step:\n%s", out)
	}
	if !strings.Contains(out, "FATAL") {
		t.Fatalf("the refusal must name the FATAL; got:\n%s", out)
	}

	// (c) the honest minimal proof: the binary's own post-store-init boot line
	//     plus the existing db/table checks ⇒ ok.
	dirC := t.TempDir()
	makeFiveTableDB(t, dirC)
	openBin := writeStub("opens-then-exits.sh", "printf '%s\\n' '✅ Database initialized (GORM, SQLite)'; exit 0")
	out, rc = runMigrateWith(t, fn, openBin, dirC, "probe-opens")
	if rc != 0 {
		t.Fatalf("a boot that logs the database-open line must pass; rc=%d\n%s", rc, out)
	}
	if !strings.Contains(out, "ok (") {
		t.Fatalf("expected the ok verdict; got:\n%s", out)
	}
}
