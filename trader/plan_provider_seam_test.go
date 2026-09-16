package trader

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/logger"
	"nofx/store"
)

// ── CLEANUP BATCH 2, B3 — installActivePlanProvider HAS A CLOCK SEAM, AND ITS
// NIL PATH SAYS WHY ──────────────────────────────────────────────────────────
//
// Before: the provider closed over time.Now() and returned nil in silence
// when no session was live, so no arm-path fixture could be driven at a
// chosen moment (every one was pinned to whatever session happened to be
// open when the suite ran), and a MISSING PLAN produced no line at all — a
// refused arm logs why; a missing plan sent a lane to inspect the split
// logic, the one thing that was not wrong.

func captureTraderLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := logger.Log.Out
	logger.Log.SetOutput(&buf)
	t.Cleanup(func() { logger.Log.SetOutput(old) })
	return &buf
}

func planProviderFixture(t *testing.T) (*AutoTrader, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "pp.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	at := &AutoTrader{
		id: "pp-trader", exchange: "ninjatrader", store: st,
		config: AutoTraderConfig{NinjaTraderSymbol: "MNQ", StrategyConfig: &store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}},
	}
	planProviderNilMu.Lock()
	delete(planProviderNilReason, at.id)
	planProviderNilMu.Unlock()
	return at, st
}

// THE SEAM: the same provider, driven at two chosen moments, answers from the
// clock it was handed — inside NY with an active plan it serves the plan;
// in the 15:00–17:00 CT gap it serves nil — with no dependence on the hour
// the suite runs at.
func TestActivePlanProviderIsDrivenByTheClockItIsHanded(t *testing.T) {
	at, st := planProviderFixture(t)
	inNY := time.Date(2026, 9, 10, 10, 0, 0, 0, kernel.CTLocation())
	gap := time.Date(2026, 9, 10, 16, 0, 0, 0, kernel.CTLocation())
	tradeDate, _ := kernel.PlanChainTradeDate(&kernel.SessionDef{Name: kernel.SessionNY, WindowStartCT: "08:30", WindowEndCT: "14:45"}, inNY)
	if tradeDate == "" {
		tradeDate = plannerTradeDateCT(inNY)
	}
	if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: "pp-plan", StrategyID: at.id, TradeDate: tradeDate, Session: kernel.SessionNY, Lifecycle: "active",
		Doc: `{"bias":{"direction":"neutral"},"levels":[],"scenarios":[]}`, CreatedAt: inNY.Add(-time.Hour)}); err != nil {
		t.Fatal(err)
	}
	clock := inNY
	installActivePlanProviderAt(at, st, func() time.Time { return clock })
	t.Cleanup(func() { kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{}) })

	if p := kernel.ActivePlanFor(at.id, "MNQ"); p == nil || p.Session != kernel.SessionNY || p.PlanID != "pp-plan" {
		t.Fatalf("inside NY with an active plan the provider must serve it, got %+v", p)
	}
	clock = gap
	if p := kernel.ActivePlanFor(at.id, "MNQ"); p != nil {
		t.Fatalf("in the 15:00–17:00 CT gap the provider must serve nil, got %+v", p)
	}
	clock = inNY
	if p := kernel.ActivePlanFor(at.id, "MNQ"); p == nil {
		t.Fatal("back inside NY the provider must serve the plan again")
	}
}

// THE NIL PATH IS NOT SILENT: each reason prints ONCE when it becomes the
// reason, not once per executor cycle; recovery prints once.
func TestActivePlanProviderLogsWhyItReturnsNil_OncePerReason(t *testing.T) {
	at, st := planProviderFixture(t)
	buf := captureTraderLog(t)
	gap := time.Date(2026, 9, 10, 16, 0, 0, 0, kernel.CTLocation())
	inNY := time.Date(2026, 9, 10, 10, 0, 0, 0, kernel.CTLocation())
	clock := gap
	installActivePlanProviderAt(at, st, func() time.Time { return clock })
	t.Cleanup(func() { kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{}) })

	for i := 0; i < 5; i++ {
		_ = kernel.ActivePlanFor(at.id, "MNQ")
	}
	out := buf.String()
	if n := strings.Count(out, "active-plan provider returns NO PLAN"); n != 1 {
		t.Fatalf("no-session nil must log exactly once for five cycles, logged %d:\n%s", n, out)
	}
	if !strings.Contains(out, "no session is live in the registry") {
		t.Fatalf("the line must say WHY:\n%s", out)
	}
	// a different reason: inside NY with nothing authored
	clock = inNY
	buf.Reset()
	for i := 0; i < 3; i++ {
		_ = kernel.ActivePlanFor(at.id, "MNQ")
	}
	out = buf.String()
	if n := strings.Count(out, "active-plan provider returns NO PLAN"); n != 1 || !strings.Contains(out, "no plan row for NY") {
		t.Fatalf("a changed reason logs once, naming it (got %d):\n%s", n, out)
	}
	// recovery: a plan appears → served, one recovery line
	tradeDate, _ := kernel.PlanChainTradeDate(&kernel.SessionDef{Name: kernel.SessionNY, WindowStartCT: "08:30", WindowEndCT: "14:45"}, inNY)
	if tradeDate == "" {
		tradeDate = plannerTradeDateCT(inNY)
	}
	if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: "pp-plan2", StrategyID: at.id, TradeDate: tradeDate, Session: kernel.SessionNY, Lifecycle: "active",
		Doc: `{"bias":{"direction":"neutral"},"levels":[],"scenarios":[]}`, CreatedAt: inNY.Add(-time.Hour)}); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	for i := 0; i < 3; i++ {
		if kernel.ActivePlanFor(at.id, "MNQ") == nil {
			t.Fatal("plan must be served once authored")
		}
	}
	out = buf.String()
	if n := strings.Count(out, "a plan is served again"); n != 1 {
		t.Fatalf("recovery logs exactly once (got %d):\n%s", n, out)
	}
}

// THE ENTRY POINT IS A ONE-LINE DELEGATE (the class-60 lint reads
// clock-seams.list for this; here the row is asserted to exist so the seam
// cannot be dropped from the list without this file noticing).
func TestActivePlanProviderSeamIsRegistered(t *testing.T) {
	found := false
	for _, r := range loadSeamRules(t) {
		if r.entry == "installActivePlanProvider" && r.at == "installActivePlanProviderAt" {
			found = true
		}
	}
	if !found {
		t.Fatal("clock-seams.list lacks installActivePlanProvider:installActivePlanProviderAt")
	}
}
