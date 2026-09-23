package trader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/store"
)

// withMaintenanceDir points the process-wide hold at a temp data dir and
// resets the barrier; restores both on cleanup.
func withMaintenanceDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	prev := MaintenanceDataDir()
	SetMaintenanceDataDir(dir)
	resetMaintenanceBarrierForTest()
	resetPlannerHoldNotedForTest()
	t.Cleanup(func() {
		SetMaintenanceDataDir(prev)
		resetMaintenanceBarrierForTest()
		resetPlannerHoldNotedForTest() // review 3 F17: never carried between tests
	})
	return dir
}

func setHold(t *testing.T, dir, job string) {
	t.Helper()
	if err := store.WriteMaintenanceHold(dir, store.MaintenanceHold{Held: true, JobID: job, Since: time.Now().UTC().Format(time.RFC3339), Owner: "updater"}); err != nil {
		t.Fatal(err)
	}
}

// Unconfigured (no data dir) = today's behaviour: never held, permits granted.
func TestMaintenanceGateUnconfiguredIsNotHeld(t *testing.T) {
	prev := MaintenanceDataDir()
	SetMaintenanceDataDir("")
	resetMaintenanceBarrierForTest()
	t.Cleanup(func() { SetMaintenanceDataDir(prev); resetMaintenanceBarrierForTest() })
	if reason, held := MaintenanceHeld(); held || reason != "" {
		t.Fatalf("unconfigured must not hold: %q", reason)
	}
	rel, ok := MaintenanceEntryPermit()
	if !ok {
		t.Fatal("unconfigured must grant permits")
	}
	rel()
}

// Absent file = not held; the permit is granted and counted in flight.
func TestMaintenanceGateAbsentFileGrantsPermits(t *testing.T) {
	withMaintenanceDir(t)
	if _, held := MaintenanceHeld(); held {
		t.Fatal("absent file must not hold")
	}
	rel, ok := MaintenanceEntryPermit()
	if !ok || MaintenanceInFlight() != 1 {
		t.Fatalf("permit ok=%v inflight=%d", ok, MaintenanceInFlight())
	}
	rel()
}

// A present hold refuses permits, names the job, and engages the barrier so
// the status surface can report drained.
func TestMaintenanceGateHoldRefusesAndNamesTheJob(t *testing.T) {
	dir := withMaintenanceDir(t)
	setHold(t, dir, "job-42")
	reason, held := MaintenanceHeld()
	if !held || !strings.Contains(reason, "job-42") {
		t.Fatalf("held=%v reason=%q", held, reason)
	}
	if _, ok := MaintenanceEntryPermit(); ok {
		t.Fatal("held → permit refused")
	}
	if !maintenanceBarrier.Held() || !maintenanceBarrier.Drained() {
		t.Fatal("the barrier is engaged and (nothing in flight) drained")
	}
}

// Corrupt file = HELD, reason says unreadable.
func TestMaintenanceGateCorruptFileHolds(t *testing.T) {
	dir := withMaintenanceDir(t)
	setHold(t, dir, "x")
	if err := writeRaw(dir, "{garbage"); err != nil {
		t.Fatal(err)
	}
	reason, held := MaintenanceHeld()
	if !held || !strings.Contains(reason, "unreadable") {
		t.Fatalf("corrupt must hold with an 'unreadable' reason: held=%v %q", held, reason)
	}
}

// Clearing the file releases the barrier on the next read.
func TestMaintenanceGateClearReleases(t *testing.T) {
	dir := withMaintenanceDir(t)
	setHold(t, dir, "job-1")
	if _, held := MaintenanceHeld(); !held {
		t.Fatal("held")
	}
	if err := store.ClearMaintenanceHold(dir, "job-1"); err != nil {
		t.Fatal(err)
	}
	if _, held := MaintenanceHeld(); held {
		t.Fatal("cleared → not held")
	}
	rel, ok := MaintenanceEntryPermit()
	if !ok {
		t.Fatal("cleared → permits granted again")
	}
	rel()
}

// The race the barrier exists for, through the production permit: a send
// holding a permit when the hold lands keeps its permit; the hold engages at
// once; drained flips only after the send releases.
func TestMaintenanceGateHoldDuringAnInFlightSend(t *testing.T) {
	dir := withMaintenanceDir(t)
	rel, ok := MaintenanceEntryPermit()
	if !ok {
		t.Fatal("permit")
	}
	setHold(t, dir, "job-9")
	if _, held := MaintenanceHeld(); !held {
		t.Fatal("held")
	}
	if maintenanceBarrier.Drained() {
		t.Fatal("not drained while a send is in flight")
	}
	if _, ok := MaintenanceEntryPermit(); ok {
		t.Fatal("new permit refused while held")
	}
	rel()
	if !maintenanceBarrier.Drained() {
		t.Fatal("drained after the in-flight send released")
	}
}

// M2.1 (review 3 F12): the permit RE-READS the hold file itself. A hold written
// a moment ago — before any other reader has engaged the barrier — must refuse
// the very next permit (mutation: permit from the barrier alone).
func TestEntryPermitReReadsAHoldNoReaderHasSeenYet(t *testing.T) {
	dir := withMaintenanceDir(t)
	if maintenanceBarrier.Held() {
		t.Fatal("fixture: the barrier must start released")
	}
	setHold(t, dir, "job-fresh") // written; no MaintenanceHeld() call has run since
	if release, ok := MaintenanceEntryPermit(); ok {
		release()
		t.Fatal("a hold written before this permit must refuse it, even with the barrier not yet engaged")
	}
}

// writeRaw overwrites the hold file with raw bytes (a TEST helper: it lived in
// the production file maintenance_gate.go until the M2.1 writer scan — review 3
// F13 — found it; only the operator CLI may write the hold in production).
func writeRaw(dir, body string) error {
	if err := os.MkdirAll(filepath.Dir(store.MaintenanceHoldPath(dir)), 0o700); err != nil {
		return err
	}
	return os.WriteFile(store.MaintenanceHoldPath(dir), []byte(body), 0o600)
}

// M2.1 (review 2 N1 / review 3 F16): a reader whose read of the file PREDATES
// a hold must not release the barrier another reader has since engaged. The
// interleaving, deterministically: reader P reads "absent"; before P decides,
// the hold is written and reader Q engages; P then must NOT release.
func TestAStaleReaderNeverReleasesABarrierEngagedAfterItsRead(t *testing.T) {
	dir := withMaintenanceDir(t)
	maintenanceBarrier.Engage() // a previous hold's engagement, not yet released
	fired := false
	maintenanceStateAfterReadHook = func() {
		if fired {
			return
		}
		fired = true
		setHold(t, dir, "job-n1")                // the hold lands after P's read…
		if _, held := MaintenanceHeld(); !held { // …and reader Q engages
			t.Fatal("fixture: reader Q must see the hold")
		}
	}
	t.Cleanup(func() { maintenanceStateAfterReadHook = nil })
	_ = dir
	// Reader P: its read happened BEFORE the hold file existed.
	if err := store.ClearMaintenanceHold(dir, "none"); err != nil {
		t.Fatal(err)
	}
	MaintenanceHeld()
	if !maintenanceBarrier.Held() {
		t.Fatal("a stale 'absent' read released a barrier engaged after it — entries could pass during a hold")
	}
}
