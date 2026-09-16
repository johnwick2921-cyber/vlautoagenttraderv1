package api

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/store"
)

// ITEM 2 (2026-08-17) — ASKING MUST WORK WITH NO ACTIVE PLAN.
//
// handlePlanAsk used to 400 on night/disabled ("no active plan to ask about")
// and 404 when no row existed. That locked the thread at exactly the moment the
// owner most wants it: "why did the plan die?", "why no levels tonight?" — both
// only askable once the plan is gone.

func askTestServer(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "ask.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return &Server{store: st}, st
}

// Nothing stored at all → a usable NO-PLAN context, not an error.
func TestAskContextNoPlanStillOpens(t *testing.T) {
	s, _ := askTestServer(t)
	ctx := s.resolveAskContext("t1", "MNQ", time.Date(2026, 8, 16, 4, 0, 0, 0, time.UTC))

	if ctx.kind != askContextNoPlan {
		t.Fatalf("kind = %q, want %q — an empty store must still allow a question", ctx.kind, askContextNoPlan)
	}
	if !strings.Contains(ctx.label, "NO PLAN") {
		t.Errorf("label %q must say NO PLAN so the answer cannot be read as live", ctx.label)
	}
	if !strings.Contains(ctx.planBlock, "nothing to patch") {
		t.Errorf("the no-plan context must tell the model there is nothing to patch:\n%s", ctx.planBlock)
	}
	if ctx.planID == "" {
		t.Error("a plan id is still required so the thread has somewhere to live")
	}
}

// A dead plan is the MOST useful thing to ask about, and must arrive labelled
// with the reason it stopped being the plan.
func TestAskContextFallsBackToTheMostRecentDeadPlan(t *testing.T) {
	s, st := askTestServer(t)
	if _, err := st.Plan().AppendPlan(&store.PlanDB{
		PlanID: store.MakePlanID("2026-08-16", "ASIA"), StrategyID: "t1",
		TradeDate: "2026-08-16", Session: "ASIA",
		Lifecycle: "no_trade", TriggerReason: "replans_exhausted",
		Doc: `{"reasoning":"FAIL-CLOSED: re-plans exhausted after death condition","bias":{"direction":"neutral","conviction":"low","flip_condition":"n/a"},"levels":[],"scenarios":[],"no_trade":[],"death_condition":"already dead (fail-closed)"}`,
	}); err != nil {
		t.Fatal(err)
	}

	// 04:00 UTC = 23:00 CT — night, no active session.
	ctx := s.resolveAskContext("t1", "MNQ", time.Date(2026, 8, 16, 4, 0, 0, 0, time.UTC))

	if ctx.kind != askContextHistorical {
		t.Fatalf("kind = %q, want %q — the stored dead plan is the context", ctx.kind, askContextHistorical)
	}
	if !strings.Contains(ctx.label, "HISTORICAL") || !strings.Contains(ctx.label, "NOT the live plan") {
		t.Errorf("label %q must mark itself historical", ctx.label)
	}
	for _, want := range []string{
		"THIS IS NOT A LIVE PLAN",
		"replans_exhausted",               // why the version was written
		"FAIL-CLOSED: re-plans exhausted", // the plan's own reasoning
		"already dead (fail-closed)",      // its stated death condition
		"do NOT propose changes",
	} {
		if !strings.Contains(ctx.planBlock, want) {
			t.Errorf("the historical context block is missing %q:\n%s", want, ctx.planBlock)
		}
	}
	if ctx.row == nil || ctx.row.Lifecycle != "no_trade" {
		t.Error("the resolved row must be the dead plan itself, so the reply can cite it")
	}
}

// A stored plan whose lifecycle is NOT active must never be presented as active,
// even when it belongs to the live session — otherwise a NO-TRADE row would
// re-enable the patch path.
func TestAskContextTreatsADeadRowForTheLiveSessionAsHistorical(t *testing.T) {
	s, st := askTestServer(t)
	for _, lifecycle := range []string{"active", "no_trade"} {
		if _, err := st.Plan().AppendPlan(&store.PlanDB{
			PlanID: store.MakePlanID("2026-08-16", "NY"), StrategyID: "t1",
			TradeDate: "2026-08-16", Session: "NY", Lifecycle: lifecycle,
			Doc: `{"reasoning":"r","bias":{"direction":"long","conviction":"low","flip_condition":"n/a"},"levels":[],"scenarios":[],"no_trade":[],"death_condition":"x"}`,
		}); err != nil {
			t.Fatal(err)
		}
	}
	// Whatever the clock says, the newest NY row is no_trade → never "active".
	ctx := s.resolveAskContext("t1", "MNQ", time.Date(2026, 8, 16, 4, 0, 0, 0, time.UTC))
	if ctx.kind == askContextActive {
		t.Fatal("a no_trade row must never resolve to an ACTIVE context — that would re-open the patch path")
	}
}

// The stored context values are read back by the UI and pooled into the
// sycophancy KPI; renaming one silently re-labels history.
func TestAskContextValuesAreStable(t *testing.T) {
	if askContextActive != "active" || askContextHistorical != "historical" || askContextNoPlan != "no-plan" {
		t.Errorf("ask context values changed (%q/%q/%q) — stored plan_qa rows would be mislabelled",
			askContextActive, askContextHistorical, askContextNoPlan)
	}
}
