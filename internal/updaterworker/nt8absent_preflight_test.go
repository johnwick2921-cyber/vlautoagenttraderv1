package updaterworker

import (
	"strings"
	"testing"

	"vl/internal/updaterjob"
)

// TestNT8ClosedPreflightUsesAbsentLegs is the P-D ruling item 3 end-to-end pin,
// at the production call site (the real runner + loopback app, harness test).
// Tonight's job 66383c7c refused at preflight with trader_cutover
// "working_orders: snapshot stale 18256s" while NT8 was CLOSED — preflight ran
// the NT8-present leg set unconditionally and never consulted the gate's
// nt8_absent verdict. The absent verdict is the only NT8-closed install path:
// the AddOn cannot answer the census, ack or cutover legs when its socket is
// gone.
//
// Setup mirrors production: the AddOn socket has been down ≥ nt8AbsentMinLinkDown,
// the ledger is flat, no planner read is claimed — and the cutover legs fail with
// the stale-snapshot shape (the harness knob models the production failure).
// The job must pass preflight, drain via drained_acked with drain_path=nt8_absent,
// and reach gate_ok.
func TestNT8ClosedPreflightUsesAbsentLegs(t *testing.T) {
	r := newRig(t, withNT8Down(absentDown), func(b *box) { b.cutoverStale = true })
	r.w.crash = func(q string) {
		if q == "gate_ok/done" {
			panic(crashPanic{q})
		}
	}
	if !r.runCrashing(t) {
		t.Fatalf("NT8 closed ≥60s with a flat ledger must run preflight → drained_acked → gate_ok; job=%s error=%q", r.job().State, r.job().Error)
	}
	j := r.job()
	if j.State != updaterjob.StateGateOK {
		t.Fatalf("state=%s, want gate_ok", j.State)
	}
	// The drain took the absent path and stamped it.
	rc := mustReceipt(t, j, "drain")
	if rc.Evidence["drain_path"] != drainPathNT8Absent {
		t.Fatalf("drain_path=%q, want %q", rc.Evidence["drain_path"], drainPathNT8Absent)
	}
	// The preflight receipt must pass — never a trader_cutover refusal.
	pf := mustReceipt(t, j, "preflight")
	if !pf.OK || strings.Contains(pf.Err, "trader_cutover") {
		t.Fatalf("preflight must use the absent legs when the absent verdict is eligible: ok=%v err=%q", pf.OK, pf.Err)
	}
}
