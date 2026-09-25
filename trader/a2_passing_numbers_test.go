package trader

// WAVE PLANNER Lane A, item A2 — feasibility refusals carry the numbers that
// would pass.
//
// Evidence rows (planner_rejected_prompts, evidence DB 09-24 copy, sample-id
// law):
//   342/343  geometry: net_nonpositive (gain=2.0000 cost=2.0000 net=0.0000)
//   347      geometry: rr (gain=6.5000 risk=4.5000 rr=1.444444 min=2.000000)
//   368/369  market_in_zone at the far edge … R:R below arm min …
//   370      geometry: rr (gain=2.5000 risk=4.5000 rr=0.555556 min=2.000000)
//
// Today those refusals name what FAILED but not what would PASS. A2: the
// compose refusal states min_target (tick-rounded outward); the arm-gate R:R
// refusal states the minimum target; the min-SL floor refusals state the
// passing stop. Class-250 probe: the tests call the PRODUCTION producers with
// the cited rows' numbers and assert the new fields.
//
// The mutation that proves it: dropping any of the appended fields fails the
// corresponding assertion (and the pre-edit tree fails all of them).

import (
	"strings"
	"testing"

	"nofx/kernel"
	"nofx/levelidentity"
	"nofx/store"
)

func fp(v float64) *float64 { return &v }
func int64p(v int64) *int64 { return &v }

// strictID computes the canonical strict identity id the production gate
// compares against (the identity must carry the SAME inputs or LevelByID
// fails at the identity gate, not at the guard under test).
func strictID(t *testing.T, symbol, kind, origin, tf string, lo, hi float64, formed int64) string {
	t.Helper()
	in := levelidentity.Inputs{Symbol: symbol, Kind: kind, Lo: &lo, Hi: &hi, OriginDate: origin, TF: tf, FormedCloseMs: &formed}
	id, why := levelidentity.ID(in)
	if id == nil {
		t.Fatalf("id: %s", why)
	}
	return *id
}

func TestA2ComposeRefusalsCarryThePassingNumbers(t *testing.T) {
	// Row 347 numbers through the production compose: long reject, entry 100,
	// stop 95.5 (risk 4.5), target 106.5 (gain 6.5) → rr=1.4444 < 2.
	doc := &kernel.PlanDoc{
		Zones: &kernel.LevelZoneMap{Zones: []kernel.LevelZone{
			{Anchor: 100, Lo: fp(99), Hi: fp(100), Sources: []kernel.ZoneSource{{Price: 100, Label: "PDH", Kind: "PDH", TF: "1d"}}},
			{Anchor: 106.5, Lo: fp(106.5), Hi: fp(107), Sources: []kernel.ZoneSource{{Price: 106.5, Label: "RTH-H", Kind: "RTH-H", TF: "1d"}}},
		}},
		IdentityLevels: []kernel.PlanLevel{
			{ID: sp(strictID(t, "MNQ", "PDH", "2026-09-24", "1d", 99, 100, 1727136000000)), Symbol: sp("MNQ"), Kind: sp("PDH"), Lo: fp(99), Hi: fp(100), Price: 100, Label: "PDH", TF: sp("1d"), OriginDate: sp("2026-09-24"), FormedCloseMs: int64p(1727136000000)},
		},
		Scenarios: []kernel.PlanScenario{{
			ID: "S1", LevelID: sp(strictID(t, "MNQ", "PDH", "2026-09-24", "1d", 99, 100, 1727136000000)), Condition: "reject", Direction: "long",
			Arm: &kernel.PlanArmSpec{Enabled: true, Entry: 100, Stop: 95.5, Target: 106.5},
		}},
	}
	policy := store.StructuralStopPolicy{BufferPoints: 4.5, BufferKnown: true, CostPoints: 2, CostKnown: true, MinRR: 2}
	r := ComposeLevelFadeGeometryWith(doc, doc.Scenarios[0], kernel.PlanArmLeg{Entry: 100}, policy, 20, 0.25, 2, true)
	if r.Reason != "rr" || !strings.Contains(r.Detail, "min_target=") {
		t.Fatalf("row 347: the rr refusal must state min_target; got %s (%s)", r.Reason, r.Detail)
	}

	// Row 342/343 numbers: gain == cost (2.0 == 2.0) → net=0.
	doc2 := &kernel.PlanDoc{
		Zones: &kernel.LevelZoneMap{Zones: []kernel.LevelZone{
			{Anchor: 100, Lo: fp(99), Hi: fp(100), Sources: []kernel.ZoneSource{{Price: 100, Label: "PDH", Kind: "PDH", TF: "1d"}}},
			{Anchor: 102, Lo: fp(102), Hi: fp(102.5), Sources: []kernel.ZoneSource{{Price: 102, Label: "RTH-H", Kind: "RTH-H", TF: "1d"}}},
		}},
		IdentityLevels: []kernel.PlanLevel{
			{ID: sp(strictID(t, "MNQ", "PDH", "2026-09-24", "1d", 99, 100, 1727136000000)), Symbol: sp("MNQ"), Kind: sp("PDH"), Lo: fp(99), Hi: fp(100), Price: 100, Label: "PDH", TF: sp("1d"), OriginDate: sp("2026-09-24"), FormedCloseMs: int64p(1727136000000)},
		},
		Scenarios: []kernel.PlanScenario{{
			ID: "S1", LevelID: sp(strictID(t, "MNQ", "PDH", "2026-09-24", "1d", 99, 100, 1727136000000)), Condition: "reject", Direction: "long",
			Arm: &kernel.PlanArmSpec{Enabled: true, Entry: 100, Stop: 95.5, Target: 102},
		}},
	}
	r2 := ComposeLevelFadeGeometryWith(doc2, doc2.Scenarios[0], kernel.PlanArmLeg{Entry: 100}, policy, 20, 0.25, 2, true)
	if r2.Reason != "net_nonpositive" || !strings.Contains(r2.Detail, "min_target=") {
		t.Fatalf("rows 342/343: the net refusal must state min_target; got %s (%s)", r2.Reason, r2.Detail)
	}
}

func TestA2GateRefusalsCarryThePassingNumbers(t *testing.T) {
	at := feasPlannerTrader(t, nil)
	cfg := at.config.StrategyConfig

	// Row 369 through the production arm gate: R:R below arm min.
	sc := kernel.PlanScenario{ID: "S1", Direction: "long", Quality: "A", Condition: "reject",
		Arm: &kernel.PlanArmSpec{Enabled: true, Entry: 100, Stop: 95.5, Target: 106.5}}
	v := at.armGateVerdictFor(sc, kernel.PlanArmLeg{Entry: 100, Stop: 95.5, Target: 106.5}, "long", nil, 8, "", cfg, "NY")
	if !strings.Contains(v, "minimum target") {
		t.Fatalf("row 369: the arm R:R refusal must state the minimum target; got %q", v)
	}

	// min-SL floor: stop 99.0 with ATR5m 8 → dist 1 < 12.
	v2 := at.armGateVerdictFor(sc, kernel.PlanArmLeg{Entry: 100, Stop: 99, Target: 120}, "long", nil, 8, "", cfg, "NY")
	if !strings.Contains(v2, "passing stop") {
		t.Fatalf("min-SL refusal must state the passing stop; got %q", v2)
	}
}
