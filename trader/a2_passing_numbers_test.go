package trader

// WAVE PLANNER Lane A, item A2 — feasibility refusals carry the numbers that
// would pass, pinned on their VALUES, with the honesty property the CTO's fold
// demanded: each stated number fed BACK through the same producer passes with
// NO refusal (a number that only clears the named check would be refused again
// at the next one — the loop A2 exists to end).
//
// Evidence rows (planner_rejected_prompts, evidence DB 09-24 copy, sample-id
// law):
//   342/343  geometry: net_nonpositive (gain=2.0000 cost=2.0000 net=0.0000)
//   347      geometry: rr (gain=6.5000 risk=4.5000 rr=1.444444 min=2.000000)
//   368/369  market_in_zone at the far edge … R:R below arm min …
//   370      geometry: rr (gain=2.5000 risk=4.5000 rr=0.555556 min=2.000000)
//
// All cited rows share entry=100, risk=4.5, cost=2, min_rr=2 → the farther of
// (cost floor 102.25, rr floor 109.00) is 109.00. A mutant that changes the
// number while keeping the words FAILS here; the pre-fold tree fails the exact
// pins.

import (
	"fmt"
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

// a2Doc builds the production compose fixture for one target-zone price.
// entry 100, stop 95.5 (risk 4.5), cost 2, min_rr 2 — the cited rows' numbers.
func a2Doc(t *testing.T, targetPx float64) *kernel.PlanDoc {
	t.Helper()
	id := strictID(t, "MNQ", "PDH", "2026-09-24", "1d", 99, 100, 1727136000000)
	return &kernel.PlanDoc{
		Zones: &kernel.LevelZoneMap{Zones: []kernel.LevelZone{
			// A line level AT the entry: the production compose draws the stop
			// from the zone edge (Lo − buffer = 100 − 4.5 = 95.5), so the risk
			// is exactly the cited rows' 4.5 and the stated floors are the
			// CTO's exact values.
			{Anchor: 100, Lo: fp(100), Hi: fp(100), Sources: []kernel.ZoneSource{{Price: 100, Label: "PDH", Kind: "PDH", TF: "1d"}}},
			{Anchor: targetPx, Lo: fp(targetPx), Hi: fp(targetPx + 0.5), Sources: []kernel.ZoneSource{{Price: targetPx, Label: "RTH-H", Kind: "RTH-H", TF: "1d"}}},
		}},
		IdentityLevels: []kernel.PlanLevel{
			{ID: sp(id), Symbol: sp("MNQ"), Kind: sp("PDH"), Lo: fp(99), Hi: fp(100), Price: 100, Label: "PDH", TF: sp("1d"), OriginDate: sp("2026-09-24"), FormedCloseMs: int64p(1727136000000)},
		},
		Scenarios: []kernel.PlanScenario{{
			ID: "S1", LevelID: sp(id), Condition: "reject", Direction: "long",
			Arm: &kernel.PlanArmSpec{Enabled: true, Entry: 100, Stop: 95.5, Target: targetPx},
		}},
	}
}

func TestA2ComposeRefusalsCarryThePassingNumbersExact(t *testing.T) {
	policy := store.StructuralStopPolicy{BufferPoints: 4.5, BufferKnown: true, CostPoints: 2, CostKnown: true, MinRR: 2}

	// Row 347: target 106.5 → rr=1.4444 < 2. The passing target is the rr
	// floor: 100 + 4.5×2 = 109.00, tick-exact.
	doc347 := a2Doc(t, 106.5)
	r := ComposeLevelFadeGeometryWith(doc347, doc347.Scenarios[0], kernel.PlanArmLeg{Entry: 100}, policy, 20, 0.25, 2, true)
	if r.Reason != "rr" || !strings.Contains(r.Detail, "min_target=109.00") {
		t.Fatalf("row 347: rr refusal must state min_target=109.00; got %s (%s)", r.Reason, r.Detail)
	}

	// Rows 342/343: target 102 → net=0. The cost-only floor is 102.25, but a
	// target there is refused at rr — the stated number must be the FARTHER of
	// (cost floor, rr floor) = 109.00.
	doc343 := a2Doc(t, 102)
	r2 := ComposeLevelFadeGeometryWith(doc343, doc343.Scenarios[0], kernel.PlanArmLeg{Entry: 100}, policy, 20, 0.25, 2, true)
	if r2.Reason != "net_nonpositive" || !strings.Contains(r2.Detail, "min_target=109.00") {
		t.Fatalf("rows 342/343: net refusal must state the farther floor min_target=109.00; got %s (%s)", r2.Reason, r2.Detail)
	}

	// Row 370: target 102.5 → rr=0.5556. Same rr floor: 109.00.
	doc370 := a2Doc(t, 102.5)
	r3 := ComposeLevelFadeGeometryWith(doc370, doc370.Scenarios[0], kernel.PlanArmLeg{Entry: 100}, policy, 20, 0.25, 2, true)
	if r3.Reason != "rr" || !strings.Contains(r3.Detail, "min_target=109.00") {
		t.Fatalf("row 370: rr refusal must state min_target=109.00; got %s (%s)", r3.Reason, r3.Detail)
	}

	// The honesty property: the SAME producer, fed the stated number, must
	// NOT refuse (a refusal here is the loop A2 exists to end).
	docPass := a2Doc(t, 109)
	if pass := ComposeLevelFadeGeometryWith(docPass, docPass.Scenarios[0], kernel.PlanArmLeg{Entry: 100}, policy, 20, 0.25, 2, true); pass.Reason != "" {
		t.Fatalf("compose fed its own min_target=109.00 must pass; got %s (%s)", pass.Reason, pass.Detail)
	}
}

func TestA2GateRefusalsCarryThePassingNumbersExact(t *testing.T) {
	at := feasPlannerTrader(t, nil)
	cfg := at.config.StrategyConfig
	floor := resolvedMinRR(cfg) // the fixture's RESOLVED arm floor (3.0 fallback)

	// Row 369 through the production arm gate: R:R below arm min. ATR5m 3 keeps
	// the min-SL floor (4.5) satisfied by the 4.5-risk leg, so the rr refusal is
	// what fires; the stated minimum target must be the exact rr floor at the
	// resolved minimum: 100 + 4.5×floor.
	sc := kernel.PlanScenario{ID: "S1", Direction: "long", Quality: "A", Condition: "reject",
		Arm: &kernel.PlanArmSpec{Enabled: true, Entry: 100, Stop: 95.5, Target: 106.5}}
	minTarget := 100 + 4.5*floor
	wantMin := fmt.Sprintf("minimum target %.2f", minTarget)
	v := at.armGateVerdictFor(sc, kernel.PlanArmLeg{Entry: 100, Stop: 95.5, Target: 106.5}, "long", nil, 3, "", cfg, "NY")
	if !strings.Contains(v, wantMin) {
		t.Fatalf("row 369: the arm R:R refusal must state the exact %s; got %q", wantMin, v)
	}
	if pass := at.armGateVerdictFor(sc, kernel.PlanArmLeg{Entry: 100, Stop: 95.5, Target: minTarget}, "long", nil, 3, "", cfg, "NY"); pass != "" {
		t.Fatalf("arm gate fed its own minimum target %.2f must pass; got %q", minTarget, pass)
	}

	// min-SL floor: ATR5m 8 → floor 12 → the passing stop is 88.00.
	v2 := at.armGateVerdictFor(sc, kernel.PlanArmLeg{Entry: 100, Stop: 99, Target: 120}, "long", nil, 8, "", cfg, "NY")
	if !strings.Contains(v2, "passing stop 88.00 or farther") {
		t.Fatalf("arm min-SL refusal must state the exact passing stop 88.00; got %q", v2)
	}
	// Fed back, the passing stop must clear BOTH remaining checks: dist 12
	// passes the floor, and the rr floor needs target ≥ 100 + 12×floor.
	passTarget := 100 + 12*floor
	if pass := at.armGateVerdictFor(sc, kernel.PlanArmLeg{Entry: 100, Stop: 88, Target: passTarget}, "long", nil, 8, "", cfg, "NY"); pass != "" {
		t.Fatalf("arm gate fed its own passing stop 88.00 must pass; got %q", pass)
	}

	// The entry gate's min-SL leg states the same passing stop, at the SAME
	// production call site (EntryGate, both paths' shared function).
	eg, _ := EntryGate(EntryIntent{Path: "arm", Action: "open_long", Symbol: "MNQ",
		Entry: 100, Stop: 99, Target: 130, ATR5m: 8, MinRR: 2, MinSLMult: kernel.MinSLATRMult()})
	if !strings.Contains(eg, "passing stop 88.00 or farther") {
		t.Fatalf("entry_gate min-SL refusal must state the exact passing stop 88.00; got %q", eg)
	}
	if _, refused := EntryGate(EntryIntent{Path: "arm", Action: "open_long", Symbol: "MNQ",
		Entry: 100, Stop: 88, Target: 130, ATR5m: 8, MinRR: 2, MinSLMult: kernel.MinSLATRMult()}); refused {
		t.Fatal("entry_gate fed its own passing stop 88.00 must not refuse")
	}
}
