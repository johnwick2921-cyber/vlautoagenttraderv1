package trader

import (
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
//	                           TCPTrader's three entry funcs, held across the
//	                           wire write); ok=false while held
//
// Unconfigured (MaintenanceDataDir()=="" — standalone fixtures) is today's
// behaviour: never held. main.go always configures it before traders load.
var maintenanceBarrier EntryBarrier

func resetMaintenanceBarrierForTest() { maintenanceBarrier.resetForTest() }

// maintenanceStateAfterReadHook is a TEST SEAM (nil in production): it runs
// between a reader's file read and its engage/release decision.
var maintenanceStateAfterReadHook func()

// maintenanceState reads the file and syncs the barrier to it.
func maintenanceState() (store.MaintenanceHoldState, bool) {
	dir := MaintenanceDataDir()
	if dir == "" {
		return store.MaintenanceHoldState{}, false
	}
	gen := maintenanceBarrier.Gen() // taken BEFORE the read (M2.1, review N1)
	st := store.ReadMaintenanceHold(dir)
	if maintenanceStateAfterReadHook != nil {
		maintenanceStateAfterReadHook()
	}
	if st.Held {
		maintenanceBarrier.Engage()
	} else if maintenanceBarrier.Held() {
		// Release only if nobody engaged since this read began: an "absent"
		// read that predates a hold is stale, not evidence the hold is gone.
		maintenanceBarrier.ReleaseIfGen(gen)
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

// maintenanceWireState is the hold as the wire carries it to the AddOn
// (site 7): (held, job id). A corrupt file is held with no job (fail-closed —
// the AddOn refuses entries; the gate then needs an ack for job "").
func maintenanceWireState() (bool, string) {
	st, configured := maintenanceState()
	if !configured || !st.Held {
		return false, ""
	}
	if st.Corrupt {
		return true, ""
	}
	return true, st.Hold.JobID
}

// MaintenanceInFlight is the number of entry sends holding a permit now.
func MaintenanceInFlight() int64 { return maintenanceBarrier.InFlight() }

// MaintenanceDrained reports held AND no entry send in flight.
func MaintenanceDrained() bool { return maintenanceBarrier.Drained() }

// maintenanceQueueHeld is the predicate the NT8 TCP server's queue consults
// before writing each queued entry (gap U2): MaintenanceHeld's boolean.
func maintenanceQueueHeld() bool {
	_, held := MaintenanceHeld()
	return held
}
