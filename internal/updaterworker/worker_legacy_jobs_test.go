package updaterworker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vl/internal/updaterjob"
)

// TestStartSweepLoadsRealLegacyJobs (worker-self-update P0, canon 53): the
// owner's REAL job files from v2026.10.02.1–.5 (read-only copies, stripped
// nothing — they hold no secrets) must all load in the new worker's start
// sweep and the worker starts. Every released worker before worker_swapped
// wrote boot_verified/done → complete; history validation accepts that
// LEGACY edge for replay. Mutant (edge removed) → RED.
func TestStartSweepLoadsRealLegacyJobs(t *testing.T) {
	fixtures := filepath.Join("..", "updaterjob", "testdata", "legacy-jobs")
	ents, err := os.ReadDir(fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) == 0 {
		t.Fatal("no legacy fixtures")
	}
	data := t.TempDir()
	jobsDir := filepath.Join(data, "updater", "jobs")
	if err := os.MkdirAll(jobsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	terminal := 0
	var wantRecovery []string
	for _, e := range ents {
		b, err := os.ReadFile(filepath.Join(fixtures, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		if err := os.WriteFile(filepath.Join(jobsDir, e.Name()), b, 0o600); err != nil {
			t.Fatal(err)
		}
		// each fixture must validate through the PRODUCTION reader now
		j, err := updaterjob.Read(data, id)
		if err != nil {
			t.Fatalf("Read(%s): %v", e.Name(), err)
		}
		if j.State == updaterjob.StateRecoveryNeeded {
			wantRecovery = append(wantRecovery, id)
		} else {
			terminal++
		}
	}
	var logged []string
	w := &Worker{
		cfg: Config{
			Target: Target{DataDir: data},
			Logf:   func(f string, a ...any) { logged = append(logged, fmt.Sprintf(f, a...)) },
		},
		host: stubHost{rev: strings.Repeat("ab", 20)},
	}
	rep, err := w.sweep()
	if err != nil {
		t.Fatalf("sweep over the real legacy jobs: %v", err)
	}
	if rep.FinishedOthers != terminal {
		t.Fatalf("FinishedOthers = %d, want %d terminal jobs", rep.FinishedOthers, terminal)
	}
	if rep.Active != "" {
		t.Fatalf("sweep set an active job %q over finished history", rep.Active)
	}
	if len(rep.StaleAtStart) != 0 {
		t.Fatalf("sweep marked legacy history stale: %v", rep.StaleAtStart)
	}
	if len(rep.Recovery) != len(wantRecovery) {
		t.Fatalf("Recovery = %v, want %v", rep.Recovery, wantRecovery)
	}
	for _, id := range wantRecovery {
		found := false
		for _, r := range rep.Recovery {
			if r == id {
				found = true
			}
		}
		if !found {
			t.Fatalf("Recovery = %v, missing %s", rep.Recovery, id)
		}
	}
	for _, l := range logged {
		if strings.Contains(l, "no longer validatable") {
			t.Fatalf("a real legacy job was skipped as corrupt: %s", l)
		}
	}
}

// TestStartSweepSkipsUnvalidatableTerminalJob: a terminal job that cannot be
// validated is WARN-skipped (never fatal); a non-terminal corrupt job still
// stops the sweep and names the job.
func TestStartSweepSkipsUnvalidatableTerminalJob(t *testing.T) {
	data := t.TempDir()
	jobsDir := filepath.Join(data, "updater", "jobs")
	if err := os.MkdirAll(jobsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// a terminal job with a corrupt body (unknown state) → skipped
	badTerminal := filepath.Join(jobsDir, "0123456789abcdef0123456789abcdef.json")
	if err := os.WriteFile(badTerminal, []byte(`{"schema":1,"job_id":"0123456789abcdef0123456789abcdef","release_id":"v1.0.0","state":"complete","phase":"done"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var logged []string
	w := &Worker{
		cfg: Config{
			Target: Target{DataDir: data},
			Logf:   func(f string, a ...any) { logged = append(logged, fmt.Sprintf(f, a...)) },
		},
		host: stubHost{rev: strings.Repeat("ab", 20)},
	}
	rep, err := w.sweep()
	if err != nil {
		t.Fatalf("sweep over a corrupt TERMINAL job: %v", err)
	}
	if len(rep.Recovery) != 0 || rep.Active != "" {
		t.Fatalf("sweep = %+v", rep)
	}
	found := false
	for _, l := range logged {
		if strings.Contains(l, "0123456789abcdef0123456789abcdef") && strings.Contains(l, "skipped") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no WARN naming the skipped terminal job: %v", logged)
	}

	// a non-terminal corrupt job → the sweep errors and names it
	if err := os.WriteFile(filepath.Join(jobsDir, "1123456789abcdef0123456789abcdef.json"), []byte(`{"schema":1,"job_id":"1123456789abcdef0123456789abcdef","release_id":"v1.0.0","state":"downloaded","phase":"done"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := w.sweep(); err == nil || !strings.Contains(err.Error(), "1123456789abcdef0123456789abcdef") {
		t.Fatalf("non-terminal corrupt job did not stop the sweep with its id: %v", err)
	}
}
