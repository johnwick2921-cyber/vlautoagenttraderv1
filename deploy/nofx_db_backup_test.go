package deploy

// P2-5 backup-script contract tests (CTO blocker 2026-09-26):
//   1. the DEFAULT run touches ONLY data.db — research is opt-in
//      (mutation: flip the script's default to ON and this test fails);
//   2. the OPT-IN run (NOFX_BACKUP_RESEARCH=1) backs research up;
//   3. the disk precheck REFUSES when free < 2.5 × source size;
//   4. the disk precheck REFUSES when free-after-backup < NOFX_BACKUP_MIN_FREE_GB.
//
// The script is driven with real small SQLite DBs (python3 stdlib) and a PATH
// shim that fakes `df`, so no real disk geometry is ever consulted.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const backupScript = "nofx-db-backup.sh"

// mkDB creates a small real SQLite database at path via python3's stdlib.
func mkTestSQLiteDB(t *testing.T, path string) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skipf("python3 not found — the backup script cannot run: %v", err)
	}
	out, err := exec.Command("python3", "-c",
		`import sqlite3,sys
c=sqlite3.connect(sys.argv[1])
c.execute('CREATE TABLE t(x INTEGER)')
c.executemany('INSERT INTO t VALUES (?)', [(i,) for i in range(100)])
c.commit(); c.close()`, path).CombinedOutput()
	if err != nil {
		t.Fatalf("python3 mkdb: %v\n%s", err, out)
	}
}

// runScript runs deploy/nofx-db-backup.sh in a fresh temp dir containing a real
// main DB and a real research DB, with a PATH shim whose `df` prints the
// fakeAvailBytes count as the backup volume's free space. It returns the
// backup root so callers can inspect what the run actually wrote.
func runBackupScript(t *testing.T, fakeAvailBytes int64, extraEnv map[string]string) (backupRoot, stdout, stderr string, rc int) {
	t.Helper()
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	mainDB := filepath.Join(dir, "main.db")
	researchDB := filepath.Join(dir, "research.db")
	mkTestSQLiteDB(t, mainDB)
	mkTestSQLiteDB(t, researchDB)

	shim := filepath.Join(dir, "shim")
	if err := os.MkdirAll(shim, 0o755); err != nil {
		t.Fatal(err)
	}
	dfShim := filepath.Join(shim, "df")
	if err := os.WriteFile(dfShim, []byte(fmt.Sprintf(`#!/bin/sh
echo "Filesystem 1B-blocks Used Available Capacity Mounted on"
echo "fake 0 0 %d 0%% /fake"
`, fakeAvailBytes)), 0o755); err != nil {
		t.Fatal(err)
	}

	backupRoot = filepath.Join(dir, "backups")
	env := []string{
		"HOME=" + home,
		"PATH=" + shim + ":" + os.Getenv("PATH"),
		"NOFX_DB=" + mainDB,
		"NOFX_DB_RESEARCH=" + researchDB,
		"NOFX_BACKUP_DIR=" + backupRoot,
	}
	for k, v := range extraEnv {
		env = append(env, k+"="+v)
	}

	cmd := exec.Command("bash", backupScript)
	cmd.Env = env
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	rc = 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			rc = ee.ExitCode()
		} else {
			t.Fatalf("running %s: %v", backupScript, err)
		}
	}
	return backupRoot, outBuf.String(), errBuf.String(), rc
}

// listBackupFiles returns every regular file under the backup root, relative.
func listBackupFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			rel, _ := filepath.Rel(root, p)
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestBackupDefaultRunTouchesOnlyMainDB(t *testing.T) {
	// A huge fake free space so the precheck can never be the refusal reason.
	root, _, stderr, rc := runBackupScript(t, 1<<40, nil)
	if rc != 0 {
		t.Fatalf("default run rc=%d stderr=%q", rc, stderr)
	}
	files := listBackupFiles(t, root)
	if len(files) == 0 {
		t.Fatal("default run wrote no backup files")
	}
	for _, f := range files {
		if strings.Contains(f, "research") {
			t.Fatalf("default run touched the research DB: %s (opt-in is NOFX_BACKUP_RESEARCH=1)", f)
		}
		if strings.Contains(f, ".partial") {
			t.Fatalf("default run left a partial file behind: %s", f)
		}
	}
	// The main DB must actually have been backed up, or the whole test is vacuous.
	mainFiles := 0
	for _, f := range files {
		if strings.HasPrefix(filepath.Base(f), "nofx-") && strings.HasSuffix(f, ".db.gz") {
			mainFiles++
		}
	}
	if mainFiles == 0 {
		t.Fatalf("default run produced no nofx-*.db.gz: %v", files)
	}
}

func TestBackupOptInBacksUpResearch(t *testing.T) {
	root, _, stderr, rc := runBackupScript(t, 1<<40, map[string]string{"NOFX_BACKUP_RESEARCH": "1"})
	if rc != 0 {
		t.Fatalf("opt-in run rc=%d stderr=%q", rc, stderr)
	}
	found := false
	for _, f := range listBackupFiles(t, root) {
		if strings.HasPrefix(filepath.Base(f), "research-") && strings.HasSuffix(f, ".db.gz") {
			found = true
		}
	}
	if !found {
		t.Fatalf("opt-in run wrote no research-*.db.gz; files=%v", listBackupFiles(t, root))
	}
}

func TestBackupRefusesWhenFreeBelowTwoPointFiveX(t *testing.T) {
	// 2,000 free bytes vs a real (small) SQLite source → free < 2.5 × source.
	root, _, stderr, rc := runBackupScript(t, 2000, nil)
	if rc == 0 {
		t.Fatalf("low-space run exited 0 — precheck did not refuse; stdout/stderr=%q", stderr)
	}
	if !strings.Contains(stderr, "REFUSED") {
		t.Fatalf("low-space refusal missing from stderr: %q", stderr)
	}
	if files := listBackupFiles(t, root); len(files) != 0 {
		t.Fatalf("refused run still wrote files: %v", files)
	}
}

func TestBackupRefusesWhenPostBackupFreeBelowFloor(t *testing.T) {
	// free-after-backup = 40 GB < the 50 GB floor → must refuse even though
	// 40 GB ≫ 2.5 × the tiny source.
	root, _, stderr, rc := runBackupScript(t, 40*1024*1024*1024, nil)
	if rc == 0 {
		t.Fatalf("floor run exited 0 — precheck did not refuse; stderr=%q", stderr)
	}
	if !strings.Contains(stderr, "REFUSED") {
		t.Fatalf("floor refusal missing from stderr: %q", stderr)
	}
	if files := listBackupFiles(t, root); len(files) != 0 {
		t.Fatalf("refused run still wrote files: %v", files)
	}
}
