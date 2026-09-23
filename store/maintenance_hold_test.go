package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// W-ONE-BUTTON M2 — the maintenance hold file. Absent = NOT held (today's
// behaviour, byte-identical). Present and well-formed = what it says.
// Present but unreadable or unparseable = HELD (fail closed).

func TestMaintenanceHoldAbsentFileIsNotHeld(t *testing.T) {
	dir := t.TempDir()
	st := ReadMaintenanceHold(dir)
	if st.Held || st.Present || st.Corrupt {
		t.Fatalf("absent file must read NOT held, not present, not corrupt: %+v", st)
	}
	if st.Path != filepath.Join(dir, "updater", "hold.json") {
		t.Fatalf("path = %q", st.Path)
	}
}

func TestMaintenanceHoldWrittenFileReadsBack(t *testing.T) {
	dir := t.TempDir()
	since := time.Date(2026, 9, 22, 22, 30, 0, 0, time.UTC)
	if err := WriteMaintenanceHold(dir, MaintenanceHold{Held: true, JobID: "job-1", Since: since.Format(time.RFC3339), Reason: "update", Owner: "updater"}); err != nil {
		t.Fatal(err)
	}
	st := ReadMaintenanceHold(dir)
	if !st.Held || !st.Present || st.Corrupt || st.Hold.JobID != "job-1" || st.Hold.Owner != "updater" {
		t.Fatalf("written hold must read back held: %+v", st)
	}
	// no temp files left behind by the atomic write
	ents, _ := os.ReadDir(filepath.Join(dir, "updater"))
	if len(ents) != 1 {
		t.Fatalf("atomic write left %d entries in updater/, want exactly hold.json", len(ents))
	}
}

func TestMaintenanceHoldCorruptFileIsHeld(t *testing.T) {
	for name, body := range map[string]string{
		"garbage":          "{not json",
		"empty":            "",
		"held-without-job": `{"held":true,"since":"2026-09-22T22:30:00Z","owner":"updater"}`,
		"held-bad-since":   `{"held":true,"job_id":"j","since":"yesterday","owner":"updater"}`,
		"held-not-a-bool":  `{"held":"yes","job_id":"j","since":"2026-09-22T22:30:00Z"}`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "updater"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "updater", "hold.json"), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			st := ReadMaintenanceHold(dir)
			if !st.Held || !st.Corrupt || !st.Present || st.Err == "" {
				t.Fatalf("%s: a present but unusable hold file must read HELD+corrupt with a reason: %+v", name, st)
			}
		})
	}
}

func TestMaintenanceHoldUnreadableFileIsHeld(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads 0000 files")
	}
	dir := t.TempDir()
	if err := WriteMaintenanceHold(dir, MaintenanceHold{Held: true, JobID: "j", Since: "2026-09-22T22:30:00Z", Owner: "updater"}); err != nil {
		t.Fatal(err)
	}
	p := MaintenanceHoldPath(dir)
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(p, 0o600)
	resetMaintenanceHoldCacheForTest()
	if st := ReadMaintenanceHold(dir); !st.Held || !st.Corrupt {
		t.Fatalf("an unreadable hold file must read HELD: %+v", st)
	}
}

// An explicit, well-formed release ("held":false) is not held. Only a
// well-formed file can release; anything unusable holds.
func TestMaintenanceHoldExplicitReleaseIsNotHeld(t *testing.T) {
	dir := t.TempDir()
	if err := WriteMaintenanceHold(dir, MaintenanceHold{Held: false, JobID: "job-1", Since: "2026-09-22T22:30:00Z", Owner: "updater"}); err != nil {
		t.Fatal(err)
	}
	if st := ReadMaintenanceHold(dir); st.Held || st.Corrupt || !st.Present {
		t.Fatalf("an explicit held=false file is present and not held: %+v", st)
	}
}

// The mtime/size cache must never serve a stale verdict after the file changes.
func TestMaintenanceHoldCacheSeesEveryChange(t *testing.T) {
	dir := t.TempDir()
	if st := ReadMaintenanceHold(dir); st.Held {
		t.Fatal("absent must not be held")
	}
	if err := WriteMaintenanceHold(dir, MaintenanceHold{Held: true, JobID: "a", Since: "2026-09-22T22:30:00Z", Owner: "updater"}); err != nil {
		t.Fatal(err)
	}
	if st := ReadMaintenanceHold(dir); !st.Held || st.Hold.JobID != "a" {
		t.Fatalf("after write: %+v", st)
	}
	if err := WriteMaintenanceHold(dir, MaintenanceHold{Held: true, JobID: "b", Since: "2026-09-22T22:31:00Z", Owner: "updater"}); err != nil {
		t.Fatal(err)
	}
	if st := ReadMaintenanceHold(dir); st.Hold.JobID != "b" {
		t.Fatalf("a rewrite must be seen (cache keyed on inode+mtime+size): %+v", st)
	}
	if err := ClearMaintenanceHold(dir, "b"); err != nil {
		t.Fatal(err)
	}
	if st := ReadMaintenanceHold(dir); st.Held || st.Present {
		t.Fatalf("after clear: %+v", st)
	}
}

// Clearing is scoped to the job that holds: a mismatched job id must not
// release someone else's hold (the updater owns its hold; nothing else clears it).
func TestMaintenanceHoldClearRequiresTheHoldingJob(t *testing.T) {
	dir := t.TempDir()
	if err := WriteMaintenanceHold(dir, MaintenanceHold{Held: true, JobID: "job-1", Since: "2026-09-22T22:30:00Z", Owner: "updater"}); err != nil {
		t.Fatal(err)
	}
	if err := ClearMaintenanceHold(dir, "job-2"); err == nil {
		t.Fatal("clearing with the wrong job id must fail")
	}
	if st := ReadMaintenanceHold(dir); !st.Held {
		t.Fatal("a failed clear must leave the hold in place")
	}
	if err := ClearMaintenanceHold(dir, ""); err == nil {
		t.Fatal("clearing with an empty job id must fail")
	}
}
