package kernel

import (
	"strings"
	"testing"

	"nofx/market"
)

// A3 (2026-08-26) — min-SL validation: pure verdict + level clearance + the
// full-gate interaction (refuse → widened retry → R:R honesty).

func TestMinSLVerdict(t *testing.T) {
	if blocked, msg := MinSLVerdict("open_long", 22.5, 38.1, 1.0); !blocked || !strings.HasPrefix(msg, "sl_too_tight") {
		t.Fatalf("22.5 < 1.0×38.1 must refuse, got blocked=%v msg=%q", blocked, msg)
	}
	if blocked, _ := MinSLVerdict("open_long", 40.0, 38.1, 1.0); blocked {
		t.Fatal("40.0 ≥ 1.0×38.1 must pass")
	}
	if blocked, _ := MinSLVerdict("open_short", 25.0, 40.0, 1.0); !blocked {
		t.Fatal("short twin must refuse identically")
	}
	if blocked, _ := MinSLVerdict("open_long", 22.5, 38.1, 0); blocked {
		t.Fatal("mult=0 (off) must pass")
	}
	if blocked, _ := MinSLVerdict("open_long", 22.5, 0, 1.0); blocked {
		t.Fatal("atr=0 must fail open")
	}
	if blocked, _ := MinSLVerdict("close_long", 22.5, 38.1, 1.0); blocked {
		t.Fatal("management actions never blocked")
	}
}

func TestMinSLAnchorForClearance(t *testing.T) {
	doc := PlanDoc{Levels: []PlanLevel{{Price: 29154.38, Label: "OB(bear)·4h", Grade: "A"}},
		Scenarios: []PlanScenario{{ID: "S1", Trigger: "reject at 29154.38", Condition: "reject", Direction: "short"}}}
	ap := &ActivePlan{Version: 1, Doc: doc, Session: "ASIA"}
	SetTraderPlanProviders("t1", TraderPlanProviders{ActivePlan: func(string) *ActivePlan { return ap }})
	defer SetTraderPlanProviders("t1", TraderPlanProviders{})
	ctx := &Context{TraderID: "t1"}
	d := &Decision{Symbol: "MNQ", Action: "open_short", CitedScenario: "S1"}
	anchor, ok := MinSLAnchorFor(ctx, d)
	if !ok || anchor != 29154.38 {
		t.Fatalf("anchor = %.2f ok=%v want 29154.38 true", anchor, ok)
	}
	if _, ok := MinSLAnchorFor(ctx, &Decision{Symbol: "MNQ", Action: "open_short"}); ok {
		t.Fatal("empty citation must fail open")
	}
	if _, ok := MinSLAnchorFor(ctx, &Decision{Symbol: "MNQ", Action: "open_short", CitedScenario: "S9"}); ok {
		t.Fatal("unknown scenario must fail open")
	}
	if _, ok := MinSLAnchorFor(&Context{TraderID: "nobody"}, d); ok {
		t.Fatal("no plan provider must fail open")
	}
}

// TestMinSLGateChainInteraction drives the REAL validateDecision: a tight stop
// is refused (min-SL), the retry with a widened stop passes min-SL, and a
// widened stop that breaks R:R is honestly refused by the R:R gate.
func TestMinSLGateChainInteraction(t *testing.T) {
	ctx := &Context{
		TraderID:  "t1",
		Structure: map[string]StructureState{"5m": {Atr: 38.1}},
		MarketDataMap: map[string]*market.Data{
			"MNQ": {CurrentPrice: 29200},
		},
		Account: AccountInfo{TotalEquity: 50000},
	}
	mk := func(sl, tp float64) *Decision {
		return &Decision{Symbol: "MNQ", Action: "open_long", Leverage: 5, PositionSizeUSD: 12000, Confidence: 70, StopLoss: sl, TakeProfit: tp}
	}
	if err := validateDecision(mk(29180, 29400), 50000, 5, 5, 5, 1, 3.0, 60, 20, ctx); err == nil || !strings.Contains(err.Error(), "sl_too_tight") {
		t.Fatalf("tight stop must hit the min-SL gate, got %v", err)
	}
	// 0B (2026-09-02): the floor moved 1.0 → 1.5×ATR5m, so "widened" now means
	// ≥ 1.5 × 38.1 = 57.15 pts. 29200−29160 = 40 pts no longer clears it; the
	// fixture widens to 29140 (60 pts) to keep testing what it always tested —
	// the gate ORDER (min-SL first, then R:R), not the floor's value.
	if err := validateDecision(mk(29140, 29220), 50000, 5, 5, 5, 1, 3.0, 60, 20, ctx); err == nil || !strings.Contains(err.Error(), "risk/reward") {
		t.Fatalf("widened-but-unviable stop must hit the R:R gate, got %v", err)
	}
	if err := validateDecision(mk(29140, 29500), 50000, 5, 5, 5, 1, 3.0, 60, 20, ctx); err != nil {
		t.Fatalf("viable widened stop must pass, got %v", err)
	}
	// The OLD floor's width (40 pts = 1.05×ATR) is now REFUSED — the pin for D1.
	if err := validateDecision(mk(29160, 29500), 50000, 5, 5, 5, 1, 3.0, 60, 20, ctx); err == nil || !strings.Contains(err.Error(), "sl_too_tight") {
		t.Fatalf("a 1.05×ATR stop passed the old 1.0 floor and must now be refused, got %v", err)
	}
	t.Setenv("MIN_SL_ATR_MULT", "0")
	defer t.Setenv("MIN_SL_ATR_MULT", "")
	if err := validateDecision(mk(29180, 29400), 50000, 5, 5, 5, 1, 3.0, 60, 20, ctx); err != nil {
		t.Fatalf("MIN_SL_ATR_MULT=0 must disable the gate, got %v", err)
	}
}
