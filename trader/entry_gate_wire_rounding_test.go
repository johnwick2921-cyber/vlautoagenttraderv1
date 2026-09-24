package trader

import (
	"math"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
)

// W1b E12(a) — LEGS 5 AND 6 JUDGE THE PRICES THE WIRE SENDS.
//
// The wire rounds entry and target to the nearest tick and the stop AWAY from
// the entry. A gate that judges the continuous (authored) prices can pass an
// order the broker then receives with a worse R:R (the wider stop) or a
// shorter stop distance (an off-grid entry that rounds toward the stop). The
// gate must judge the wire-rounded values and REFUSE when rounding drops them
// under the floor — never send them.
//
// Production call sites: entryGateForArm (the arm seam's builder) and
// entryGateForDecisionAt (the decision path's builder), both of which run the
// one EntryGate.

// rrFixture returns a long whose continuous R:R clears the floor and whose
// wire-rounded R:R (stop 29575.90 → 29575.75) does not.
func rrFixture(t *testing.T, floor float64) (entry, stop, target float64) {
	t.Helper()
	entry, stop = 29600.00, 29575.90 // risk 24.10 continuous, 24.25 on the wire
	reward := math.Ceil(floor*24.10/0.25) * 0.25
	target = entry + reward // on grid: the target does not move on the wire
	if reward/24.10+1e-9 < floor || reward/24.25 >= floor {
		t.Fatalf("fixture needs floor > 1.67 so one tick decides it; floor=%.2f reward=%.2f", floor, reward)
	}
	return
}

func TestEntryGateArmRRJudgedOnWireRoundedStop(t *testing.T) {
	t.Setenv("MIN_SL_ATR_MULT", "0") // isolate leg 5
	at := &AutoTrader{id: "e12-rr"}
	floor := at.armMinRRFor(nil)
	entry, stop, target := rrFixture(t, floor)
	plan := &kernel.ActivePlan{PlanID: "2026-09-23:NY:e12", Version: 1, Session: "NY"}
	sc := kernel.PlanScenario{ID: "S1", Direction: "long", Condition: "reclaim"}
	leg := kernel.PlanArmLeg{Entry: entry, Stop: stop, Target: target}
	reason, refused := at.entryGateForArm(plan, sc, leg, "long", "long", 0)
	if !refused {
		t.Fatalf("arm leg entry %.2f stop %.2f target %.2f: the wire sends stop 29575.75 (R:R %.4f < floor %.2f) — must be REFUSED, got allow",
			entry, stop, target, (target-entry)/24.25, floor)
	}
	if !strings.Contains(reason, "R:R") || !strings.Contains(reason, "below floor") {
		t.Fatalf("refusal must be the R:R leg; got %q", reason)
	}
	if !strings.Contains(reason, "29575.75") {
		t.Fatalf("refusal must quote the wire stop it judged (29575.75); got %q", reason)
	}
}

func TestEntryGateDecisionRRJudgedOnWireRoundedStop(t *testing.T) {
	t.Setenv("MIN_SL_ATR_MULT", "0")
	at := &AutoTrader{id: "e12-rr-d"}
	floor := at.armMinRRFor(nil)
	entry, stop, target := rrFixture(t, floor)
	d := &kernel.Decision{Action: "open_long", Symbol: "MNQ", StopLoss: stop, TakeProfit: target}
	reason, refused := at.entryGateForDecisionAt(d, entry, time.Now())
	if !refused || !strings.Contains(reason, "R:R") {
		t.Fatalf("decision path must refuse the R:R the wire would send; got refused=%v %q", refused, reason)
	}
}

// An off-grid limit entry rounds to the nearest tick on the wire. When that
// moves it TOWARD the stop, the broker's stop distance is shorter than the one
// judged on the continuous entry. Entry 29600.12 → 29600.00 on the wire, stop
// 29576.00 (on grid): continuous distance 24.12 ≥ floor 24.10, wire 24.00.
func TestEntryGateArmMinSLJudgedOnWireRoundedEntry(t *testing.T) {
	t.Setenv("MIN_SL_ATR_MULT", "1.5")
	at := &AutoTrader{id: "e12-minsl"}
	plan := &kernel.ActivePlan{PlanID: "2026-09-23:NY:e12", Version: 1, Session: "NY"}
	sc := kernel.PlanScenario{ID: "S1", Direction: "long", Condition: "reclaim"}
	atr5m := 24.10 / 1.5
	leg := kernel.PlanArmLeg{Entry: 29600.12, Stop: 29576.00, Target: 29800.00}
	reason, refused := at.entryGateForArm(plan, sc, leg, "long", "long", atr5m)
	if !refused {
		t.Fatalf("wire entry 29600.00 / stop 29576.00 = 24.00 < floor 24.10 — must be REFUSED, got allow")
	}
	if armRefusalClass(reason) != "min_sl" {
		t.Fatalf("refusal must be the min-SL leg; got %q", reason)
	}
}

// A stop that rounds AWAY can only widen the distance: the floor case of the
// #191 frame (composed 29575.90 on a 24.10 floor) is ADMITTED, and admitted on
// the value the broker receives.
func TestEntryGateArmMinSLFloorStopAdmittedOnWire(t *testing.T) {
	t.Setenv("MIN_SL_ATR_MULT", "1.5")
	at := &AutoTrader{id: "e12-minsl-ok"}
	plan := &kernel.ActivePlan{PlanID: "2026-09-23:NY:e12", Version: 1, Session: "NY"}
	sc := kernel.PlanScenario{ID: "S1", Direction: "long", Condition: "reclaim"}
	atr5m := 24.10 / 1.5
	leg := kernel.PlanArmLeg{Entry: 29600.00, Stop: 29575.90, Target: 29800.00}
	if reason, refused := at.entryGateForArm(plan, sc, leg, "long", "long", atr5m); refused {
		t.Fatalf("stop on the floor rounds away (24.25 ≥ 24.10) — must be admitted; got %q", reason)
	}
}
