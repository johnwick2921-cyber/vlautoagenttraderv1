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
	// no temp files left behind by the atomic write (the .hold.lock file is
	// the writers' flock, MUST-1, and is expected to persist)
	ents, _ := os.ReadDir(filepath.Join(dir, "updater"))
	for _, e := range ents {
		if n := e.Name(); n != "hold.json" && n != ".hold.lock" {
			t.Fatalf("atomic write left %q behind in updater/", n)
		}
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

// MUST-1 (CTO review of cb513079): clear is read-then-remove. Without a lock
// held across both, an operator's `clear --job X` that has read X can remove
// the hold job Y wrote in between — entries allowed during Y's install. The
// hook pauses the clearer between its read and its remove; a writer runs in
// that window. Invariant: the wrong job never removes the right job's file.
func TestMaintenanceHoldClearCannotRemoveAHoldWrittenDuringIt(t *testing.T) {
	dir := t.TempDir()
	if err := WriteMaintenanceHold(dir, MaintenanceHold{Held: true, JobID: "X", Since: "2026-09-22T22:30:00Z", Owner: "cli"}); err != nil {
		t.Fatal(err)
	}
	wrote := make(chan error, 1)
	holdClearAfterReadHook = func() {
		// the clearer has read X and is about to remove; job Y writes now
		go func() {
			wrote <- WriteMaintenanceHold(dir, MaintenanceHold{Held: true, JobID: "Y", Since: "2026-09-22T22:31:00Z", Owner: "updater"})
		}()
		select {
		case err := <-wrote:
			wrote <- err // the writer finished while we were paused (no lock)
		case <-time.After(200 * time.Millisecond):
			// the writer is blocked (lock held across read+remove) — go on
		}
	}
	defer func() { holdClearAfterReadHook = nil }()
	clearErr := ClearMaintenanceHold(dir, "X")
	select {
	case err := <-wrote:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the writer never finished (bounded wait)")
	}
	_ = clearErr
	st := ReadMaintenanceHold(dir)
	if !st.Held || st.Hold.JobID != "Y" {
		t.Fatalf("job Y's hold was lost to job X's clear: %+v", st)
	}
}

// The same invariant under real concurrency, -race: many rounds of
// write(X) → {clear(X) ∥ write(Y)}; the final state is always Y held.
func TestMaintenanceHoldWriterAndClearerRaceNeverLosesTheRightJob(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 200; i++ {
		if err := WriteMaintenanceHold(dir, MaintenanceHold{Held: true, JobID: "X", Since: "2026-09-22T22:30:00Z", Owner: "cli"}); err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{}, 2)
		go func() { _ = ClearMaintenanceHold(dir, "X"); done <- struct{}{} }()
		go func() {
			_ = WriteMaintenanceHold(dir, MaintenanceHold{Held: true, JobID: "Y", Since: "2026-09-22T22:31:00Z", Owner: "updater"})
			done <- struct{}{}
		}()
		for k := 0; k < 2; k++ {
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("round did not finish (bounded wait)")
			}
		}
		if st := ReadMaintenanceHold(dir); !st.Held || st.Hold.JobID != "Y" {
			t.Fatalf("round %d: the right job's hold was lost: %+v", i, st)
		}
	}
}
