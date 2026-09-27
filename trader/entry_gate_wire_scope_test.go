package trader

import (
	"strings"
	"testing"
	"time"

	"nofx/kernel"
)

// W1b E12 verifier defects 1 and 4 — WHICH PRICES LEGS 5/6 JUDGE, BY VENUE.
//
// (1) The NT8 tick grid is a CME-futures fact. A non-CME symbol (a crypto
// trader still reaches EntryGate through admitEntry → entryGateForDecisionAt)
// fell through to the 0.25 index default: DOGEUSDT entry 0.12 rounded to 0,
// legs 5/6 skipped it, and an R:R 0.10 open that base refused was ALLOWED.
// Any non-CME venue is judged on the authored prices, exactly as at base.
//
// (4) The gate's tick must come from the same root resolver the wire uses, so
// a contract-code symbol (M2KU6) is judged on the M2K tick (0.10) the wire
// sends, not the 0.25 default.

func TestEntryGateNonCMESymbolJudgedOnAuthoredPrices(t *testing.T) {
	t.Setenv("MIN_SL_ATR_MULT", "0")
	at := &AutoTrader{id: "e12-doge"}
	if floor := at.armMinRRFor(nil); floor <= 0.10 {
		t.Fatalf("fixture needs an R:R floor above 0.10, got %.2f", floor)
	}
	d := &kernel.Decision{Action: "open_long", Symbol: "DOGEUSDT", StopLoss: 0.11, TakeProfit: 0.121}
	reason, refused := at.entryGateForDecisionAt(d, 0.12, time.Now())
	if !refused || !strings.Contains(reason, "R:R 0.10 below floor") {
		t.Fatalf("DOGEUSDT entry 0.12 SL 0.11 TP 0.121 (R:R 0.10) must be refused on the authored prices as at base; got refused=%v %q", refused, reason)
	}
	if strings.Contains(reason, "wire-rounded") {
		t.Fatalf("a non-CME venue must not be judged on the NT8 tick grid; got %q", reason)
	}
}

func TestEntryGateContractCodeSymbolJudgedOnWireTick(t *testing.T) {
	t.Setenv("MIN_SL_ATR_MULT", "0") // isolate leg 5
	at := &AutoTrader{id: "e12-m2k"}
	at.config.NinjaTraderSymbol = "M2KU6"
	if floor := at.armMinRRFor(nil); floor != 3 {
		t.Fatalf("fixture is built for an R:R floor of 3.00, got %.2f", floor)
	}
	plan := &kernel.ActivePlan{PlanID: "2026-09-23:NY:e12", Version: 1, Session: "NY"}
	sc := kernel.PlanScenario{ID: "S1", Direction: "long", Condition: "reclaim"}
	// On the M2K 0.10 grid nothing moves: R:R 29.80/9.80 = 3.04 ≥ 3.00, and the
	// wire sends exactly these prices (TestWireTickResolvesContractCodeRoot).
	// On the 0.25 default: stop 1990.00, target 2029.75 → R:R 2.975 < 3.00.
	leg := kernel.PlanArmLeg{Entry: 2000.00, Stop: 1990.20, Target: 2029.80}
	if reason, refused := at.entryGateForArm(plan, sc, leg, "long", "long", 0, time.Date(2026, 9, 2, 15, 0, 0, 0, time.UTC)); refused {
		t.Fatalf("M2KU6 leg judged on a tick the wire does not use (wire tick 0.10 sends R:R 3.04); got %q", reason)
	}
}
