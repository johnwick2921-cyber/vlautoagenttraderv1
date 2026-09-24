package trader

import (
	"math"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
)

// W1b FOLD-7 — THE WIRE NOTE MEANS "ROUNDING MOVED A PRICE", NOTHING ELSE.
//
// entryGateWirePrices documents its note as "" when rounding moved nothing, so
// the refusal text stays byte-identical to the pre-E12 text. That held only for
// exact (power-of-two) ticks. On a non-power-of-two tick the float grid is not
// exact: RoundToTick(2000.3, 0.10) = 2000.3000000000002 ≠ 2000.3, so an M2K leg
// authored ON the 0.10 grid got a refusal claiming it was "judged on the
// wire-rounded prices" when rounding moved nothing a tick could see. Movement
// is now measured against the wire's own tick epsilon (the one WireStop uses).
//
// Production call sites: entryGateForArm (the arm seam's builder) and
// entryGateForDecisionAt (the decision path's builder).

const e12WireNote = "judged on the wire-rounded prices"

var e12PassClock = time.Date(2026, 9, 2, 15, 0, 0, 0, time.UTC)

func e12ArmFixture() (*kernel.ActivePlan, kernel.PlanScenario) {
	return &kernel.ActivePlan{PlanID: "2026-09-23:NY:e12", Version: 1, Session: "NY"},
		kernel.PlanScenario{ID: "S1", Direction: "long", Condition: "reclaim"}
}

// An M2K arm leg authored exactly on the 0.10 grid, refused by leg 5 on its
// own geometry (R:R 1.00 < 3.00): the refusal must NOT claim a wire rounding.
func TestEntryGateArmOnGridNonPow2TickReportsNoWireRounding(t *testing.T) {
	t.Setenv("MIN_SL_ATR_MULT", "0") // isolate leg 5
	at := &AutoTrader{id: "fold7-m2k-arm"}
	at.config.NinjaTraderSymbol = "M2KU6"
	if floor := at.armMinRRFor(nil); floor != 3 {
		t.Fatalf("fixture is built for an R:R floor of 3.00, got %.2f", floor)
	}
	plan, sc := e12ArmFixture()
	leg := kernel.PlanArmLeg{Entry: 2000.3, Stop: 1990.3, Target: 2010.3}
	reason, refused := at.entryGateForArm(plan, sc, leg, "long", "long", 0, e12PassClock)
	want := "entry_gate: R:R 1.00 below floor 3.00 at execution price 2000.3000 (SL 1990.3000 TP 2010.3000)"
	if !refused || reason != want {
		t.Fatalf("on-grid M2K leg (0.10 tick): want the pre-E12 refusal text byte-identical\n want %q\n  got refused=%v %q", want, refused, reason)
	}
}

// The same on the min-SL leg (leg 6), and on the short side (the stop rounds
// UP there): an on-grid stop is not a rounding.
func TestEntryGateArmOnGridNonPow2TickMinSLReportsNoWireRounding(t *testing.T) {
	t.Setenv("MIN_SL_ATR_MULT", "1.5")
	at := &AutoTrader{id: "fold7-m2k-minsl"}
	at.config.NinjaTraderSymbol = "M2KU6"
	plan, sc := e12ArmFixture()
	sc.Direction = "short"
	atr5m := 20.0 / 1.5 // floor 20.00; stop distance 10.00
	leg := kernel.PlanArmLeg{Entry: 2000.3, Stop: 2010.3, Target: 1900.1}
	reason, refused := at.entryGateForArm(plan, sc, leg, "short", "short", atr5m, e12PassClock)
	if !refused || armRefusalClass(reason) != "min_sl" {
		t.Fatalf("stop distance 10.00 < 20.00 — must be refused by the min-SL leg; got refused=%v %q", refused, reason)
	}
	if strings.Contains(reason, e12WireNote) {
		t.Fatalf("on-grid M2K short (0.10 tick): rounding moved nothing, yet the refusal claims it did: %q", reason)
	}
}

// The decision path shares the one EntryGate: an on-grid M2K market entry is
// refused without a spurious wire note.
func TestEntryGateDecisionOnGridNonPow2TickReportsNoWireRounding(t *testing.T) {
	t.Setenv("MIN_SL_ATR_MULT", "0")
	at := &AutoTrader{id: "fold7-m2k-d"}
	d := &kernel.Decision{Action: "open_short", Symbol: "M2KU6", StopLoss: 2010.3, TakeProfit: 1990.3}
	reason, refused := at.entryGateForDecisionAt(d, 2000.3, time.Now())
	if !refused || !strings.Contains(reason, "R:R 1.00 below floor") {
		t.Fatalf("decision path must refuse R:R 1.00; got refused=%v %q", refused, reason)
	}
	if strings.Contains(reason, e12WireNote) {
		t.Fatalf("on-grid M2K decision (0.10 tick): rounding moved nothing, yet the refusal claims it did: %q", reason)
	}
}

// The note still fires when rounding REALLY moves a price on the 0.10 grid:
// stop 1990.33 goes to the wire as 1990.30 (a long's stop rounds DOWN).
func TestEntryGateArmOffGridNonPow2TickStillReportsWireRounding(t *testing.T) {
	t.Setenv("MIN_SL_ATR_MULT", "0")
	at := &AutoTrader{id: "fold7-m2k-off"}
	at.config.NinjaTraderSymbol = "M2KU6"
	plan, sc := e12ArmFixture()
	leg := kernel.PlanArmLeg{Entry: 2000.3, Stop: 1990.33, Target: 2010.3}
	reason, refused := at.entryGateForArm(plan, sc, leg, "long", "long", 0, e12PassClock)
	if !refused || !strings.Contains(reason, e12WireNote) || !strings.Contains(reason, "SL 1990.3300") || !strings.Contains(reason, "(SL 1990.3000") {
		t.Fatalf("off-grid stop 1990.33 moves to 1990.30 on the wire — the refusal must say so and quote both; got refused=%v %q", refused, reason)
	}
}

// Live 0.25 (MNQ) is unaffected: on-grid prices come back bit-identical and
// the refusal text is the pre-E12 text, byte for byte.
func TestEntryGateArmOnGridQuarterTickByteIdentical(t *testing.T) {
	t.Setenv("MIN_SL_ATR_MULT", "0")
	at := &AutoTrader{id: "fold7-mnq"}
	plan, sc := e12ArmFixture()
	leg := kernel.PlanArmLeg{Entry: 29600.25, Stop: 29590.25, Target: 29610.25}
	reason, refused := at.entryGateForArm(plan, sc, leg, "long", "long", 0, e12PassClock)
	want := "entry_gate: R:R 1.00 below floor 3.00 at execution price 29600.2500 (SL 29590.2500 TP 29610.2500)"
	if !refused || reason != want {
		t.Fatalf("on-grid MNQ leg (0.25 tick): want the pre-E12 refusal text byte-identical\n want %q\n  got refused=%v %q", want, refused, reason)
	}
	e, s, tg, note := entryGateWirePrices("long", EntryIntent{Symbol: "MNQ", Entry: 29600.25, Stop: 29590.25, Target: 29610.25})
	if math.Float64bits(e) != math.Float64bits(29600.25) || math.Float64bits(s) != math.Float64bits(29590.25) ||
		math.Float64bits(tg) != math.Float64bits(29610.25) || note != "" {
		t.Fatalf("0.25 tick is exact in binary: on-grid prices must come back bit-identical with no note; got %v %v %v %q", e, s, tg, note)
	}
}
