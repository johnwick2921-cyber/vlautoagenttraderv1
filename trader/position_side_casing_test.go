package trader

import (
	"os"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	ntwire "nofx/provider/ninjatrader"
	ntTrader "nofx/trader/ninjatrader"
)

// ── W-EXEC-TRUTH W0 (Q10/Q12, checklist canon 28) — POSITION SIDES ARE READ
// CANONICALLY WHERE THEY ENTER ──────────────────────────────────────────────
//
// Two dead safety legs, one cause: the value was compared before it was
// canonicalized.
//   - ntHeldPosition required positionAmt > 0, but NT8 (like the crypto
//     brokers) signs a SHORT negative, so a held short read as FLAT and
//     reconcileBeforeOpenNT let an entry net onto it.
//   - EntryGate leg 7 (one open position) is fed by builders that filtered
//     p.Side == "long" || "short", while every writer stores "LONG"/"SHORT",
//     so leg 7 never saw a position; the same-side guards in executeOpen*
//     compared pos["side"] == "long" against NT8's "LONG" and never fired.

func TestNtHeldPositionSeesBothSidesOnTheWire(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	nt := ntTrader.NewTCPTrader(s, "MNQ", "Sim101")
	at := &AutoTrader{trader: nt, exchange: "ninjatrader"}
	for _, side := range []string{"long", "short"} {
		s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{{Symbol: "MNQ", Side: side, Quantity: 1, AvgPrice: 30000}})
		if got := at.ntHeldPosition("MNQ"); got != side {
			pos, _ := nt.GetPositions()
			t.Fatalf("NT8 holds a %s (GetPositions %+v) but ntHeldPosition = %q", side, pos, got)
		}
	}
	s.SeedPositionsForTest("Sim101", nil)
	if got := at.ntHeldPosition("MNQ"); got != "" {
		t.Fatalf("flat NT8 must read flat, got %q", got)
	}
}

// Leg 7 on the DECISION path sees a position written by the production writer.
func TestDecisionLegSevenSeesAStoredPosition(t *testing.T) {
	w := newDropWire(t)
	w.at.recordPositionChange("sig-open", "MNQ", "LONG", "open_long", 1, 29000, 1, 0, 0, 70)
	reason, refused := w.at.entryGateForDecisionAt(&kernel.Decision{Action: "open_long", Symbol: "MNQ", StopLoss: 28900, TakeProfit: 29300}, 29000, time.Now())
	if !refused || !strings.Contains(reason, "one_open_position") {
		t.Fatalf("an OPEN LONG row must refuse a second entry (leg 7), got refused=%v %q", refused, reason)
	}
}

// Leg 7 on the ARM path sees the same stored position.
func TestArmLegSevenSeesAStoredPosition(t *testing.T) {
	w := newDropWire(t)
	w.at.recordPositionChange("sig-open", "MNQ", "SHORT", "open_short", 1, 29000, 1, 0, 0, 70)
	plan := &kernel.ActivePlan{PlanID: "2026-09-23:NY", Session: "NY", Version: 1}
	sc := kernel.PlanScenario{ID: "S1", Direction: "long"}
	reason, refused := w.at.entryGateForArm(plan, sc, kernel.PlanArmLeg{}, "long", "long", 0)
	if !refused || !strings.Contains(reason, "one_open_position") {
		t.Fatalf("an OPEN SHORT row must refuse an arm (leg 7), got refused=%v %q", refused, reason)
	}
}

// The canonicalizer: NT8 maps (UPPERCASE side, signed amount) and crypto maps
// (lowercase side, signed amount) read the same; a missing side falls back to
// the sign.
func TestBrokerPositionSideIsCanonical(t *testing.T) {
	for _, tc := range []struct {
		pos  map[string]interface{}
		want string
	}{
		{map[string]interface{}{"side": "LONG", "positionAmt": 1.0}, "long"},
		{map[string]interface{}{"side": "SHORT", "positionAmt": -1.0}, "short"},
		{map[string]interface{}{"side": "long", "positionAmt": 2.0}, "long"},
		{map[string]interface{}{"side": " short ", "positionAmt": -2.0}, "short"},
		{map[string]interface{}{"positionAmt": -1.0}, "short"},
		{map[string]interface{}{"positionAmt": 1.0}, "long"},
		{map[string]interface{}{"side": "Buy", "positionAmt": 0.0}, ""},
	} {
		if got := brokerPositionSide(tc.pos); got != tc.want {
			t.Errorf("brokerPositionSide(%v) = %q, want %q", tc.pos, got, tc.want)
		}
	}
}

// No raw lowercase side comparison survives in the order paths: every broker
// side read goes through the canonicalizer.
func TestOrderPathsReadBrokerSidesCanonically(t *testing.T) {
	for _, f := range []string{"auto_trader_orders.go", "entry_gate.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src := string(b)
		for _, bad := range []string{`pos["side"] == "long"`, `pos["side"] == "short"`, `p.Side == "long" || p.Side == "short"`} {
			if strings.Contains(src, bad) {
				t.Errorf("%s still compares a raw side (%s) — read it through positionSide/brokerPositionSide (canon 28)", f, bad)
			}
		}
	}
}
