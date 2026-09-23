package trader

import (
	"os"
	"path/filepath"

	"nofx/store"
)

// ── W-ONE-BUTTON M2 — THE MAINTENANCE GATE (process-wide) ──────────────────
//
// One predicate every entry site calls, one barrier every entry SEND passes.
// The hold file (store.ReadMaintenanceHold, stat-cached) is the source of
// truth; reading it ENGAGES or RELEASES the process-wide barrier, so the
// barrier can never disagree with the file for longer than one read.
//
//	MaintenanceHeld()        → (reason, held): the gate predicate (sites 1–3, 5)
//	MaintenanceEntryPermit() → (release, ok): the send-side permit (site 4,
//	                           TCPTrader's four entry funcs, held across the
//	                           wire write); ok=false while held
//
// Unconfigured (MaintenanceDataDir()=="" — standalone fixtures) is today's
// behaviour: never held. main.go always configures it before traders load.
var maintenanceBarrier EntryBarrier

func resetMaintenanceBarrierForTest() { maintenanceBarrier = EntryBarrier{} }

// maintenanceState reads the file and syncs the barrier to it.
func maintenanceState() (store.MaintenanceHoldState, bool) {
	dir := MaintenanceDataDir()
	if dir == "" {
		return store.MaintenanceHoldState{}, false
	}
	st := store.ReadMaintenanceHold(dir)
	if st.Held {
		maintenanceBarrier.Engage()
	} else if maintenanceBarrier.Held() {
		maintenanceBarrier.Release()
	}
	return st, true
}

// MaintenanceHeld is the gate predicate: (reason, true) while the
// installation is under maintenance. The reason names the job, or says the
// hold file is unreadable (fail-closed).
func MaintenanceHeld() (string, bool) {
	st, configured := maintenanceState()
	if !configured || !st.Held {
		return "", false
	}
	return maintenanceReason(st), true
}

func maintenanceReason(st store.MaintenanceHoldState) string {
	if st.Corrupt {
		return "hold file unreadable (" + st.Err + ") — fail-closed"
	}
	r := "hold job=" + st.Hold.JobID + " since=" + st.Hold.Since
	if st.Hold.Owner != "" {
		r += " owner=" + st.Hold.Owner
	}
	return r
}

// MaintenanceEntryPermit is the send-side permit for one entry send. It
// re-reads the hold first (so a hold written a microsecond ago is honoured),
// then asks the barrier. The caller holds the release across the wire write.
func MaintenanceEntryPermit() (release func(), ok bool) {
	if _, held := MaintenanceHeld(); held {
		return nil, false
	}
	return maintenanceBarrier.Permit()
}

// MaintenanceInFlight is the number of entry sends holding a permit now.
func MaintenanceInFlight() int64 { return maintenanceBarrier.InFlight() }

// MaintenanceDrained reports held AND no entry send in flight.
func MaintenanceDrained() bool { return maintenanceBarrier.Drained() }

// writeRaw is a test helper: overwrite the hold file with raw bytes.
func writeRaw(dir, body string) error {
	return os.WriteFile(filepath.Join(dir, "updater", "hold.json"), []byte(body), 0o600)
}

// maintenanceQueueHeld is the predicate the NT8 TCP server's queue consults
// before writing each queued entry (gap U2): MaintenanceHeld's boolean.
func maintenanceQueueHeld() bool {
	_, held := MaintenanceHeld()
	return held
}
