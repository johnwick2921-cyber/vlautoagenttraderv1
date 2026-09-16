package trader

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// g4Harness builds a minimal AutoTrader + a short-bias active plan for the
// transition state machine.
func g4Harness(t *testing.T, bias string) (*AutoTrader, *store.Store, func(*kernel.ActivePlan)) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	on := true
	at := &AutoTrader{
		id:       "t1",
		exchange: "ninjatrader",
		store:    st,
		config: AutoTraderConfig{
			NinjaTraderSymbol: "MNQ",
			StrategyConfig:    &store.StrategyConfig{Regime: &store.RegimeConfig{TransitionStanddown: &on}},
		},
	}
	birth := time.Date(2026, 8, 21, 4, 49, 0, 0, kernel.CTLocation()).UnixMilli()
	plan := &kernel.ActivePlan{
		Doc:     kernel.PlanDoc{Bias: kernel.PlanBias{Direction: bias}},
		Session: "LONDON",
		Version: 1,
		PlanID:  "p1",
		BirthMs: birth,
	}
	kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{ActivePlan: func(symbol string) *kernel.ActivePlan {
		return plan
	}})
	t.Cleanup(func() { kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{}) })
	return at, st, func(p *kernel.ActivePlan) { plan = p }
}

func g4ctx15m(events []kernel.StructureEvent) *kernel.Context {
	return &kernel.Context{Structure: map[string]kernel.StructureState{
		"15m": {Trend: "TRENDING_DOWN", Swing: &kernel.SwingRef{Kind: "LL", Price: 29220.25, TimeMs: 0},
			LastEvents: events},
	}}
}

func TestG4TransitionOpensOnCHoCH(t *testing.T) {
	at, _, _ := g4Harness(t, "short")
	ev := kernel.StructureEvent{Type: "CHoCH", Dir: "up", Price: 29470.25,
		TimeMs: time.Date(2026, 8, 21, 10, 45, 0, 0, kernel.CTLocation()).UnixMilli(), RefPrice: 29451.75}
	ctx := g4ctx15m([]kernel.StructureEvent{ev})
	at.observeTransitionStanddown(ctx)
	if !ctx.TransitionActive || ctx.TransitionDir != "short" {
		t.Fatalf("CHoCH-up vs short plan must open TRANSITION, got active=%v dir=%q", ctx.TransitionActive, ctx.TransitionDir)
	}
	if !strings.Contains(ctx.TransitionDetail, "CHoCH-up 15m") {
		t.Fatalf("detail must name the event: %q", ctx.TransitionDetail)
	}
	// Card chip mirror persisted.
	raw, _ := at.store.GetSystemConfig(store.TransitionKey("p1", 1))
	if !strings.Contains(raw, `"active":true`) {
		t.Fatalf("transition mirror not persisted: %q", raw)
	}
}

func TestG4TransitionClosesOnResumption(t *testing.T) {
	at, _, _ := g4Harness(t, "short")
	choch := kernel.StructureEvent{Type: "CHoCH", Dir: "up", Price: 29470.25,
		TimeMs: time.Date(2026, 8, 21, 10, 45, 0, 0, kernel.CTLocation()).UnixMilli()}
	at.observeTransitionStanddown(g4ctx15m([]kernel.StructureEvent{choch}))
	if !at.transition.Active {
		t.Fatalf("precondition: stand-down must be open")
	}
	// BOS resumption in the plan's direction after the trigger closes it.
	bos := kernel.StructureEvent{Type: "BOS", Dir: "down", Price: 29350,
		TimeMs: time.Date(2026, 8, 21, 11, 0, 0, 0, kernel.CTLocation()).UnixMilli()}
	ctx := g4ctx15m([]kernel.StructureEvent{choch, bos})
	at.observeTransitionStanddown(ctx)
	if ctx.TransitionActive || at.transition.Active {
		t.Fatalf("BOS resumption must close TRANSITION, got active=%v", ctx.TransitionActive)
	}
}

func TestG4TransitionClosesOnPlanReplacement(t *testing.T) {
	at, _, setPlan := g4Harness(t, "short")
	choch := kernel.StructureEvent{Type: "CHoCH", Dir: "up", Price: 29470.25,
		TimeMs: time.Date(2026, 8, 21, 10, 45, 0, 0, kernel.CTLocation()).UnixMilli()}
	at.observeTransitionStanddown(g4ctx15m([]kernel.StructureEvent{choch}))
	if !at.transition.Active {
		t.Fatalf("precondition: stand-down must be open")
	}
	// Flip confirmed → the planner wrote a new version (new identity).
	setPlan(&kernel.ActivePlan{Doc: kernel.PlanDoc{Bias: kernel.PlanBias{Direction: "long"}},
		Session: "LONDON", Version: 2, PlanID: "p1",
		BirthMs: time.Date(2026, 8, 21, 10, 50, 0, 0, kernel.CTLocation()).UnixMilli()})
	ctx := g4ctx15m([]kernel.StructureEvent{choch})
	at.observeTransitionStanddown(ctx)
	if ctx.TransitionActive || at.transition.Active {
		t.Fatalf("plan replacement (flip confirmed) must close TRANSITION")
	}
}

func TestG4TransitionTimerExpiry(t *testing.T) {
	t.Setenv("TRANSITION_MAX_MIN", "0")
	at, _, _ := g4Harness(t, "short")
	choch := kernel.StructureEvent{Type: "CHoCH", Dir: "up", Price: 29470.25,
		TimeMs: time.Now().Add(-10 * time.Minute).UnixMilli()}
	// First cycle: opens on the fresh event (as it must).
	at.observeTransitionStanddown(g4ctx15m([]kernel.StructureEvent{choch}))
	if !at.transition.Active {
		t.Fatalf("precondition: fresh CHoCH must open the stand-down")
	}
	// Second cycle: the 0-min cap closes it and the same event must NOT reopen.
	ctx := g4ctx15m([]kernel.StructureEvent{choch})
	at.observeTransitionStanddown(ctx)
	if ctx.TransitionActive || at.transition.Active {
		t.Fatalf("0-min cap must close TRANSITION and the closed trigger must not reopen it")
	}
}

func TestG4TransitionFailOpenWithoutPlanOrSnapshot(t *testing.T) {
	on := true
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	at := &AutoTrader{id: "t1", exchange: "ninjatrader", store: st,
		config: AutoTraderConfig{StrategyConfig: &store.StrategyConfig{Regime: &store.RegimeConfig{TransitionStanddown: &on}}}}
	kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{ActivePlan: func(symbol string) *kernel.ActivePlan {
		return nil
	}})
	t.Cleanup(func() { kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{}) })
	ctx := &kernel.Context{Structure: nil}
	at.observeTransitionStanddown(ctx)
	if ctx.TransitionActive {
		t.Fatalf("no plan → stand-down must stay dormant (fail-open)")
	}
	// Off-switch: toggle off → state clears.
	off := false
	at.config.StrategyConfig.Regime.TransitionStanddown = &off
	at.observeTransitionStanddown(ctx)
	if ctx.TransitionActive || at.transition.Active {
		t.Fatalf("transition_standdown=off must restore pre-wave behavior")
	}
}

// TestG4Replay_NoFalseStanddownOnRealBars is the honest replay guard: on the
// real 2026-08-21 15m series no CHoCH/MSS close-through ever occurred, so the
// stand-down must NOT open on 539/544 — the mechanism is pinned by the
// synthetic fixtures above; the day's truth stays fail-open.
func TestG4Replay_NoFalseStanddownOnRealBars(t *testing.T) {
	rows := []struct {
		h, m         int
		o, hi, lo, c float64
	}{
		{6, 0, 29479.75, 29539.75, 29452.75, 29499.25},
		{6, 15, 29499.50, 29533.75, 29472.00, 29511.50},
		{6, 30, 29511.75, 29516.25, 29257.25, 29303.75},
		{6, 45, 29303.50, 29335.75, 29220.25, 29331.75},
		{7, 0, 29332.00, 29488.50, 29326.25, 29443.75},
		{7, 15, 29443.25, 29454.50, 29380.50, 29412.25},
		{7, 30, 29412.00, 29423.75, 29375.00, 29383.50},
		{7, 45, 29383.25, 29405.50, 29353.75, 29400.25},
		{8, 0, 29399.75, 29433.25, 29380.00, 29411.50},
		{9, 0, 29303.50, 29335.75, 29220.25, 29331.75},
		{10, 0, 29332.00, 29368.75, 29326.25, 29352.25},
		{10, 15, 29353.00, 29371.25, 29330.75, 29368.50},
		{10, 30, 29368.00, 29471.75, 29364.25, 29451.75},
		{10, 45, 29451.75, 29488.50, 29425.50, 29443.75},
		{11, 0, 29443.25, 29447.75, 29389.25, 29394.25},
		{11, 15, 29393.75, 29454.50, 29386.00, 29410.00},
		{11, 30, 29409.75, 29425.75, 29393.75, 29417.50},
		{11, 45, 29417.50, 29417.50, 29380.50, 29412.25},
		{12, 0, 29412.00, 29423.75, 29398.25, 29408.50},
		{12, 15, 29408.00, 29410.50, 29375.00, 29395.25},
		{12, 30, 29396.00, 29404.75, 29382.75, 29403.75},
		{12, 45, 29404.50, 29410.75, 29381.25, 29383.50},
		{13, 0, 29383.25, 29405.50, 29371.25, 29374.25},
		{13, 15, 29374.50, 29392.00, 29362.00, 29363.00},
		{13, 30, 29363.25, 29372.25, 29353.75, 29366.50},
		{13, 45, 29366.50, 29403.50, 29363.50, 29400.25},
		{14, 0, 29399.75, 29414.75, 29384.25, 29388.50},
		{14, 15, 29388.50, 29405.00, 29380.00, 29393.25},
		{14, 30, 29393.50, 29412.00, 29393.50, 29410.75},
		{14, 45, 29411.00, 29433.25, 29402.00, 29411.50},
	}
	kb := make([]market.KlineBar, len(rows))
	for i, r := range rows {
		kb[i] = market.KlineBar{Time: time.Date(2026, 8, 21, r.h, r.m, 0, 0, kernel.CTLocation()).UnixMilli(),
			Open: r.o, High: r.hi, Low: r.lo, Close: r.c}
	}
	st := kernel.ComputeStructureState(kb, 15, 0, time.Date(2026, 8, 21, 15, 0, 0, 0, kernel.CTLocation()).UnixMilli())
	at, _, _ := g4Harness(t, "short")
	ctx := &kernel.Context{Structure: map[string]kernel.StructureState{"15m": st}}
	at.observeTransitionStanddown(ctx)
	if ctx.TransitionActive || at.transition.Active {
		t.Fatalf("real 08-21 15m bars hold no CHoCH/MSS close-through — stand-down must stay closed, got events %+v", st.LastEvents)
	}
}
