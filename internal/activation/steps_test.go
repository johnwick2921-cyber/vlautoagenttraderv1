package activation

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// withSystem swaps the machine seam for the duration of one test.
func withSystem(t *testing.T, s *system) {
	t.Helper()
	old := sys
	sys = s
	t.Cleanup(func() { sys = old })
}

// statLine builds a /proc/<pid>/stat line whose field 22 is ticks. The comm
// deliberately contains a space and parentheses.
func statLine(pid int, ticks uint64) string {
	mid := ""
	for i := 4; i <= 21; i++ {
		mid += " 0"
	}
	return fmt.Sprintf("%d (nofx bin (x)) S%s %d 0 0", pid, mid, ticks)
}

func tempDB(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "data.db")
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`create table t (a int); insert into t values (1),(2);`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return p
}

func TestBackupProducesACopyThatOpensAndPassesIntegrityCheck(t *testing.T) {
	src := tempDB(t)
	dst := filepath.Join(t.TempDir(), "sub", "copy.db")
	rc, err := Backup(src, dst)
	if err != nil {
		t.Fatalf("Backup: %v", err)
	}
	if !rc.OK || rc.Evidence["integrity_check"] != "ok" {
		t.Fatalf("receipt does not record the verification: %+v", rc.Evidence)
	}
	db, err := sql.Open("sqlite", dst)
	if err != nil {
		t.Fatalf("copy does not open: %v", err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow("select count(*) from t").Scan(&n); err != nil || n != 2 {
		t.Fatalf("copy has %d rows (err %v), want 2 — the backup did not carry the data", n, err)
	}
}

// A file that exists at the destination but is NOT a database must not be
// mistaken for an already-completed backup. That is the idempotence trap: the
// cheap check is "does the file exist", and the honest one is "does it open".
func TestBackupReplacesADestinationThatIsNotAValidDatabase(t *testing.T) {
	src := tempDB(t)
	dst := filepath.Join(t.TempDir(), "copy.db")
	if err := os.WriteFile(dst, []byte("this is not a database"), 0o644); err != nil {
		t.Fatal(err)
	}
	rc, err := Backup(src, dst)
	if err != nil {
		t.Fatalf("Backup refused to replace a junk destination: %v", err)
	}
	if rc.Evidence["already"] == "true" {
		t.Fatal("Backup treated a non-database file as an existing backup")
	}
	if err := verifyBackup(dst); err != nil {
		t.Fatalf("destination still not a valid backup: %v", err)
	}
}

func TestBackupIsIdempotentOnAGoodExistingCopy(t *testing.T) {
	src := tempDB(t)
	dst := filepath.Join(t.TempDir(), "copy.db")
	if _, err := Backup(src, dst); err != nil {
		t.Fatalf("first: %v", err)
	}
	rc, err := Backup(src, dst)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if rc.Evidence["already"] != "true" {
		t.Fatal("a second Backup over a verified copy must be a no-op with already=true")
	}
}

// THE REFUSAL THAT MATTERS: the pid we were told to replace is now a different
// process. Signalling it would kill an innocent bystander.
func TestActivateRefusesToSignalARecycledPID(t *testing.T) {
	killed := false
	withSystem(t, &system{
		// field 22 differs from the Identity we pass in => recycled.
		ReadStat: func(pid int) (string, error) { return statLine(pid, 999999), nil },
		Kill:     func(pid int) error { killed = true; return nil },
		MainPID:  func() (int, error) { return 4242, nil },
		Now:      time.Now,
		Sleep:    func(time.Duration) {},
	})
	rel, prev := twoReleases(t)
	_, rc, err := Activate(rel, prev, Identity{PID: 4242, StartTicks: 111111})
	if err == nil {
		t.Fatal("Activate signalled a pid it could not confirm")
	}
	if killed {
		t.Fatal("Activate KILLED a recycled pid — the guard did not hold")
	}
	if !strings.Contains(err.Error(), "no longer the process") {
		t.Fatalf("refusal does not name the cause: %v", err)
	}
	if rc.Evidence["installed"] != "binary,dist,RELEASE" {
		t.Fatal("files should already be in place before the kill is attempted")
	}
}

// Files first, then the kill — and all three halves, not just the binary.
func TestActivateInstallsAllThreeHalvesBeforeKilling(t *testing.T) {
	var orderKill bool
	var distAtKill string
	rel, prev := twoReleases(t)
	withSystem(t, &system{
		ReadStat: func(pid int) (string, error) { return statLine(pid, 111111), nil },
		Kill: func(pid int) error {
			orderKill = true
			b, _ := os.ReadFile(filepath.Join(prev.Dist, "index.html"))
			distAtKill = string(b)
			return nil
		},
		MainPID: func() (int, error) { return 4243, nil },
		Now:     time.Now,
		Sleep:   func(time.Duration) {},
	})
	next, rc, err := Activate(rel, prev, Identity{PID: 4242, StartTicks: 111111})
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if !orderKill {
		t.Fatal("never killed")
	}
	if distAtKill != "NEW" {
		t.Fatalf("dist was %q at kill time — the swap must complete BEFORE the kill", distAtKill)
	}
	got, _ := os.ReadFile(prev.ReleaseFile)
	if strings.TrimSpace(string(got)) != rel.SHA {
		t.Fatalf("RELEASE holds %q, want %s — the marker is a half nobody remembers", got, rel.SHA)
	}
	if next.PID != 4243 {
		t.Fatalf("returned identity pid %d, want the NEW pid 4243", next.PID)
	}
	if !rc.OK {
		t.Fatal("receipt not OK")
	}
}

// twoReleases builds a "new" release dir and a "current install" dir.
func twoReleases(t *testing.T) (rel, prev Release) {
	t.Helper()
	newDir, curDir := t.TempDir(), t.TempDir()
	mk := func(dir, binBody, distBody, releaseBody string) Release {
		if err := os.MkdirAll(filepath.Join(dir, "web", "dist"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "nofx-bin"), []byte(binBody), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "web", "dist", "index.html"), []byte(distBody), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "RELEASE"), []byte(releaseBody), 0o644); err != nil {
			t.Fatal(err)
		}
		return Release{
			Dir: dir, SHA: releaseBody,
			Binary:      filepath.Join(dir, "nofx-bin"),
			Dist:        filepath.Join(dir, "web", "dist"),
			ReleaseFile: filepath.Join(dir, "RELEASE"),
		}
	}
	rel = mk(newDir, "NEWBIN", "NEW", strings.Repeat("b", 40))
	prev = mk(curDir, "OLDBIN", "OLD", strings.Repeat("a", 40))
	return rel, prev
}

// Watch must not accept a boot line written BEFORE the restart: that is the
// previous boot's line and it says nothing about the process running now.
func TestWatchIsRedOnAPreKillBootLineAndGreenOnAPostKillOne(t *testing.T) {
	sha := strings.Repeat("c", 40)
	dir := t.TempDir()
	logPath := filepath.Join(dir, "nofx.log")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"release":%q}`, sha)
	}))
	defer srv.Close()

	// One boot line, timestamped an hour in the PAST, naming the right sha.
	past := time.Now().Add(-time.Hour)
	old := fmt.Sprintf("%s [INFO] booted %s\n", past.Format("01-02 15:04:05"), sha)
	if err := os.WriteFile(logPath, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}

	withSystem(t, &system{
		ReadStat: func(pid int) (string, error) { return statLine(pid, 1), nil },
		Kill:     func(int) error { return nil },
		MainPID:  func() (int, error) { return 1, nil },
		Now:      time.Now,
		Sleep:    func(time.Duration) {},
	})

	// RED: health agrees, but the only boot line predates the kill.
	if _, err := Watch(Release{SHA: sha}, Identity{PID: 1}, logPath, srv.URL, 50*time.Millisecond); err == nil {
		t.Fatal("Watch accepted a boot line that predates the restart")
	} else if !strings.Contains(err.Error(), "no boot line") {
		t.Fatalf("refusal must name the failing leg, got: %v", err)
	}

	// GREEN: append a line written NOW.
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(f, "%s [INFO] booted %s\n", time.Now().Add(time.Second).Format("01-02 15:04:05"), sha)
	f.Close()

	rc, err := Watch(Release{SHA: sha}, Identity{PID: 1}, logPath, srv.URL, 3*time.Second)
	if err != nil {
		t.Fatalf("Watch refused a genuine post-restart boot: %v", err)
	}
	if rc.Evidence["boot_line"] == "" || rc.Evidence["health_sha"] != sha {
		t.Fatalf("receipt must record BOTH signals: %+v", rc.Evidence)
	}
}

// Health alone is not proof: a process that never restarted still answers.
func TestWatchRefusesWhenHealthAgreesButTheLogNeverShowsARestart(t *testing.T) {
	sha := strings.Repeat("d", 40)
	logPath := filepath.Join(t.TempDir(), "nofx.log")
	if err := os.WriteFile(logPath, []byte("no boot lines here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"release":%q}`, sha)
	}))
	defer srv.Close()
	withSystem(t, &system{Now: time.Now, Sleep: func(time.Duration) {}})
	if _, err := Watch(Release{SHA: sha}, Identity{PID: 1}, logPath, srv.URL, 30*time.Millisecond); err == nil {
		t.Fatal("Watch was satisfied by /api/health alone")
	}
}
