package trader

import (
	"strings"
	"testing"

	"nofx/kernel"
	"nofx/store"
)

// ── W-EXEC-TRUTH W5 (CTO 1790194913337) — the owner's re-read and reset treat
// a MACHINE plan as "no plan", like the scheduler: the re-read is the chain's
// first AI read (exactly one planner call, no replan budget spent) and the
// reset has no AI chain to abandon. Driven through ForceReread / ForceReset,
// the production entry points the API calls.

// freeCountingClient counts planner calls and never blocks.
func freeCountingClient() *countingPlanClient {
	c := &countingPlanClient{release: make(chan struct{})}
	close(c.release)
	return c
}

func TestOwnerRereadOnAMachineOnlyChainIsOneFreeRead(t *testing.T) {
	at, st := handOffTrader(t)
	schedulerTape(t)
	now := handOffNow()
	pinTraderNow(t, now)
	ev, _ := machinePlanV1(t, at, st, now)
	c := freeCountingClient()
	at.mcpClient = c
	cap := at.replanCapFor("NY")
	before := store.GetReplanBudget(st, at.id, handOffDate, "NY", cap)

	gate := at.CanForceReread(now)
	if !gate.Allowed || gate.ReplansLeft != cap {
		t.Fatalf("a machine-only chain is 'no plan': the re-read is allowed with the full budget, got %+v", gate)
	}
	got, err := at.ForceReread(now)
	defer drainReReads(t)
	if err != nil || !got.Allowed || got.Reason != "" {
		t.Fatalf("the re-read must run: %+v err=%v", got, err)
	}
	if n := c.calls.Load(); n != 1 {
		t.Fatalf("exactly one planner call, got %d", n)
	}
	v2, _ := st.Plan().GetLatestPlanForTraderSession(handOffDate, "NY", at.id)
	if v2 == nil || v2.Version != 2 || v2.TriggerReason != store.TriggerOwnerReread || store.IsMachinePlan(v2) {
		t.Fatalf("the owner's read lands the first AI version over the machine plan: %+v", v2)
	}
	after := store.GetReplanBudget(st, at.id, handOffDate, "NY", cap)
	if after.Used != before.Used {
		t.Fatalf("the first AI read over a machine plan spends nothing: used %d → %d", before.Used, after.Used)
	}
	// The live Picture scenario rides into the owner's v2 like any AI version.
	doc, _ := resolveActivePlanDoc(st, v2)
	if !docHasRef(doc.Scenarios, ev.OppKey) {
		t.Fatal("the live P1 must be re-appended to the owner's v2")
	}
}

// The gate AND the pre-read re-verify ignore the counter on a machine-only
// chain, exactly as on no plan (a counter at its cap cannot refuse the
// chain's first AI read, and it is not spent further).
func TestOwnerRereadOnAMachineOnlyChainIgnoresTheCounterLikeNoPlan(t *testing.T) {
	at, st := handOffTrader(t)
	schedulerTape(t)
	now := handOffNow()
	pinTraderNow(t, now)
	machinePlanV1(t, at, st, now)
	c := freeCountingClient()
	at.mcpClient = c
	cap := at.replanCapFor("NY")
	for i := 0; i < cap; i++ {
		if _, err := store.SpendReplan(st, at.id, handOffDate, "NY"); err != nil {
			t.Fatal(err)
		}
	}
	if g := at.CanForceReread(now); !g.Allowed {
		t.Fatalf("the gate must not refuse the first AI read on the counter: %+v", g)
	}
	got, err := at.ForceReread(now)
	defer drainReReads(t)
	if err != nil || got.Reason != "" || c.calls.Load() != 1 {
		t.Fatalf("the pre-read re-verify must not refuse it either: %+v err=%v calls=%d", got, err, c.calls.Load())
	}
	if b := store.GetReplanBudget(st, at.id, handOffDate, "NY", cap); b.Used != cap {
		t.Fatalf("nothing more is spent: used %d, want %d", b.Used, cap)
	}
}

// Control: over an AI plan the owner's re-read still spends exactly one.
func TestOwnerRereadOverAnAIPlanStillSpendsOne(t *testing.T) {
	at, st := handOffTrader(t)
	schedulerTape(t)
	now := handOffNow()
	pinTraderNow(t, now)
	seedAIPlan(t, st, "active")
	c := freeCountingClient()
	at.mcpClient = c
	cap := at.replanCapFor("NY")
	before := store.GetReplanBudget(st, at.id, handOffDate, "NY", cap).Used
	if _, err := at.ForceReread(now); err != nil {
		t.Fatal(err)
	}
	defer drainReReads(t)
	if after := store.GetReplanBudget(st, at.id, handOffDate, "NY", cap).Used; after != before+1 || c.calls.Load() != 1 {
		t.Fatalf("a re-read over an AI plan spends one: used %d → %d, calls %d", before, after, c.calls.Load())
	}
}

// The reset has no AI chain to abandon on a machine-only chain: refused like
// no plan — no baseline marker, no read.
func TestOwnerResetRefusesAMachineOnlyChainLikeNoPlan(t *testing.T) {
	at, st := handOffTrader(t)
	schedulerTape(t)
	now := handOffNow()
	pinTraderNow(t, now)
	machinePlanV1(t, at, st, now)
	c := freeCountingClient()
	at.mcpClient = c
	g := at.CanForceReset(now)
	if g.Allowed || !strings.Contains(g.Reason, "only a machine Picture plan") {
		t.Fatalf("reset on a machine-only chain must refuse like no plan, got %+v", g)
	}
	if _, err := at.ForceReset(now); err == nil {
		t.Fatal("ForceReset must refuse too")
	}
	if b := store.GetResetBaseline(st, at.id, handOffDate, "NY"); b != 1 {
		t.Fatalf("no reset marker may be written, baseline = %d", b)
	}
	if c.calls.Load() != 0 {
		t.Fatalf("no read runs, calls = %d", c.calls.Load())
	}
	if row, _ := st.Plan().GetLatestPlanForTraderSession(handOffDate, "NY", at.id); row.Version != 1 || !store.IsMachinePlan(row) || row.Lifecycle != "active" {
		t.Fatalf("the machine plan is untouched: %+v", row)
	}
}

func docHasRef(scs []kernel.PlanScenario, ref string) bool {
	for _, s := range scs {
		if s.Machine != nil && s.Machine.Ref == ref {
			return true
		}
	}
	return false
}
