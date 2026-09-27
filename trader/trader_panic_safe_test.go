package trader

import (
	"testing"
	"time"

	"nofx/discipline"
	"nofx/telemetry"
)

// TestTraderLoopPanicFreezesOnlyThatTrader is the P1-F pin (audit 2026-09-26):
// a panic in one trader's loop beat must NOT kill the process — it recovers,
// logs ERROR + stack, records trader_panic on the errors API, FREEZES only that
// trader (admitChain's frozen leg refuses new entries; closes keep working),
// and every other trader is untouched.
func TestTraderLoopPanicFreezesOnlyThatTrader(t *testing.T) {
	discipline.ResetFreezeForTest()
	t.Cleanup(discipline.ResetFreezeForTest)
	panicInTickOnce.Store(true)
	t.Cleanup(func() { panicInTickOnce.Store(false) })

	at1 := plannerTestTrader(t)
	at1.id = "panic-a"
	at2 := plannerTestTrader(t)
	at2.id = "survivor-b"

	// The production beat wrapper: the seam panics inside tickOnce, the
	// recovery catches it.
	at1.runBeatSafely("run loop", func() { at1.tickOnce(false) })

	if _, f1 := discipline.IsFrozen(at1.id); !f1 {
		t.Fatalf("the panicking trader must FREEZE (no new entries)")
	}
	if _, f2 := discipline.IsFrozen(at2.id); f2 {
		t.Fatalf("a second trader must NOT freeze — a panic is one trader's problem")
	}
	found := false
	for _, row := range telemetry.ErrorSummary(at1.id) {
		if row.Type == "trader_panic" && row.Count >= 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("the panic must be RECORDED (trader_panic) on the errors API")
	}

	// The panicking trader refuses entries at the admission chain's frozen leg
	// (entry_admission.go:215) — pinned here through the same call the paths use.
	refusal, refused := at1.admitChain(admitIntent{Path: admitDecision, Symbol: "MNQ", Action: "open_long"}, "MNQ", "open_long", time.Now())
	if !refused || refusal == "" {
		t.Fatalf("the frozen trader must refuse entries, got refusal=%q refused=%v", refusal, refused)
	}
}
