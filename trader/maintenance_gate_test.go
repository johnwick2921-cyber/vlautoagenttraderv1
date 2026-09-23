package trader

import (
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
	t.Cleanup(func() {
		SetMaintenanceDataDir(prev)
		resetMaintenanceBarrierForTest()
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
