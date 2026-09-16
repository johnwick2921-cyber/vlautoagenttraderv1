package trader

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// ── CLASS 48 PIN TESTS ─────────────────────────────────────────────────────
// The three trades the owner pinned MUST be refused by the ONE entry gate:
//   587 — R:R 1.09 at the real fill (floor judged at execution price, not the
//         prompt snapshot that logged "R:R 2.03 → PASS").
//   589 — NY v3 S3 = long breakout_retest, a SHADOWED condition (0C).
//   590 — NY v5 S4 = long breakout_retest, SHADOWED. (Measured correction: at
//         the cited v5 the scenario direction was LONG, so the direction leg
//         does NOT fire for 590; the shadow leg is what refuses it. The
//         direction leg is still tested for the owner's stated class.)
// "Pass on current code, fail on the fix": the pass-on-current evidence is the
// live journal quoted in the Phase 1 report (`→ PASS` + `✓ MNQ open_long
// succeeded` for all three); these tests assert the fail-on-fix side.

func TestEntryGatePin587RRRefused(t *testing.T) {
	// 587: fill 29079.25, SL 29048.00, TP 29113.25 → risk 31.25 reward 34.00.
	reason, refused := EntryGate(EntryIntent{
		Path: "decision", Action: "open_long", Symbol: "MNQ",
		Entry: 29079.25, Stop: 29048.00, Target: 29113.25,
		MinRR: 2.0,
	})
	if !refused {
		t.Fatalf("pin 587 must be refused at the real fill; got allow")
	}
	if !strings.Contains(reason, "R:R 1.09") {
		t.Fatalf("pin 587 refusal should name R:R 1.09; got %q", reason)
	}
}

func TestEntryGatePin587PassesAtSnapshotPrice(t *testing.T) {
	// The SAME trade at the snapshot the old gate used (29069.50) — proving the
	// fix is the reference, not the trade.
	reason, refused := EntryGate(EntryIntent{
		Path: "decision", Action: "open_long", Symbol: "MNQ",
		Entry: 29069.50, Stop: 29048.00, Target: 29113.25,
		MinRR: 2.0,
	})
	if refused {
		t.Fatalf("pin 587 at the snapshot price should pass the R:R leg; got %q", reason)
	}
}

func TestEntryGatePin589ShadowRefused(t *testing.T) {
	// 589: NY v3 S3 = long breakout_retest — shadow list [breakout_retest, fvg_entry].
	reason, refused := EntryGate(EntryIntent{
		Path: "decision", Action: "open_long", Symbol: "MNQ",
		Entry: 29192.50, Stop: 29115.00, Target: 29317.25,
		MinRR:         2.0,
		CitedScenario: "S3",
		ScenarioDir:   "long",
		ScenarioCond:  "breakout_retest",
		ConditionShadowed: func(cond string) bool {
			return cond == "breakout_retest" || cond == "fvg_entry"
		},
	})
	if !refused {
		t.Fatalf("pin 589 (breakout_retest) must be refused as shadowed; got allow")
	}
	if !strings.Contains(reason, "SHADOW") {
		t.Fatalf("pin 589 refusal should name SHADOW; got %q", reason)
	}
}

func TestEntryGatePin590ShadowRefused(t *testing.T) {
	// 590: NY v5 S4 = long breakout_retest (direction long at the cited version —
	// the direction leg must NOT be the one that fires; the shadow leg does).
	reason, refused := EntryGate(EntryIntent{
		Path: "decision", Action: "open_long", Symbol: "MNQ",
		Entry: 29193.25, Stop: 29143.75, Target: 29317.25,
		MinRR:         2.0,
		CitedScenario: "S4",
		ScenarioDir:   "long", // measured: v5 S4 was LONG
		ScenarioCond:  "breakout_retest",
		ConditionShadowed: func(cond string) bool {
			return cond == "breakout_retest" || cond == "fvg_entry"
		},
	})
	if !refused {
		t.Fatalf("pin 590 (breakout_retest) must be refused as shadowed; got allow")
	}
	if !strings.Contains(reason, "SHADOW") {
		t.Fatalf("pin 590 refusal should name SHADOW; got %q", reason)
	}
}

func TestEntryGateDirectionMismatchRefused(t *testing.T) {
	// The owner's stated 590 class ("long on a short scenario"): whenever the
	// cited scenario's direction opposes the action, refuse — even live,
	// non-shadowed conditions.
	reason, refused := EntryGate(EntryIntent{
		Path: "decision", Action: "open_long", Symbol: "MNQ",
		Entry: 29193.25, Stop: 29143.75, Target: 29317.25,
		MinRR:         2.0,
		CitedScenario: "S4",
		ScenarioDir:   "short",
		ScenarioCond:  "reject",
		ConditionShadowed: func(cond string) bool {
			return cond == "breakout_retest" || cond == "fvg_entry"
		},
	})
	if !refused {
		t.Fatalf("long citing a SHORT scenario must be refused as direction mismatch; got allow")
	}
	if !strings.Contains(reason, "direction mismatch") {
		t.Fatalf("direction refusal should name the mismatch; got %q", reason)
	}
}

func TestEntryGateOneLiveArmRefused(t *testing.T) {
	reason, refused := EntryGate(EntryIntent{
		Path: "decision", Action: "open_short", Symbol: "MNQ",
		Entry: 29100.00, Stop: 29150.00, Target: 29000.00,
		MinRR:            2.0,
		OpenPositionSide: "long",
	})
	if !refused {
		t.Fatalf("opposite-side entry while a position is open must be refused; got allow")
	}
	// SUPERSEDED WORDING (owner ruling 2026-09-03): the leg was one-live-ARM
	// and its message named the NETTING risk, because only an opposite-side
	// entry was refused. It is one-open-POSITION now — either side, any plan
	// version — so the message names what is open instead.
	for _, want := range []string{"one_open_position", "no adds, no flips", "long position is open"} {
		if !strings.Contains(reason, want) {
			t.Fatalf("refusal missing %q; got %q", want, reason)
		}
	}
}

func TestEntryGateMinSLRefused(t *testing.T) {
	// 589's stop 29115.00 vs entry 29192.50 = 77.5pt — passes min-SL; here a
	// too-close stop with R:R leg satisfied (floor 0.5) must still refuse.
	reason, refused := EntryGate(EntryIntent{
		Path: "decision", Action: "open_long", Symbol: "MNQ",
		Entry: 29192.50, Stop: 29180.00, Target: 29317.25,
		MinRR:     0.5,
		ATR5m:     35.89,
		MinSLMult: 1.5,
	})
	if !refused {
		t.Fatalf("stop inside 1.5×ATR5m must be refused; got allow")
	}
	if !strings.Contains(reason, "too close") {
		t.Fatalf("min-SL refusal should name the distance; got %q", reason)
	}
}

func TestEntryGateAdmitsCleanIntent(t *testing.T) {
	reason, refused := EntryGate(EntryIntent{
		Path: "decision", Action: "open_long", Symbol: "MNQ",
		Entry: 29150.00, Stop: 29100.00, Target: 29250.00,
		MinRR:     2.0,
		ATR5m:     20.0,
		MinSLMult: 1.5,
	})
	if refused {
		t.Fatalf("clean intent must be admitted; got %q", reason)
	}
}

func TestEntryGateArmSeamBuilderRefusesShadow(t *testing.T) {
	at := &AutoTrader{id: "pin-test"}
	plan := &kernel.ActivePlan{PlanID: "2026-09-02:NY:pin-test", Version: 5, Session: "NY"}
	sc := kernel.PlanScenario{ID: "S3", Direction: "long", Condition: "breakout_retest"}
	leg := kernel.PlanArmLeg{Entry: 29192.50, Stop: 29115.00, Target: 29317.25}
	// Breakout_retest is shadowed by env in this test process — resolve through
	// the real chain so the arm seam's builder exercises conditionShadowedFor.
	t.Setenv("SHADOW_CONDITIONS", "breakout_retest")
	reason, refused := at.entryGateForArm(plan, sc, leg, "long", "long", 0)
	if !refused {
		t.Fatalf("arm seam must refuse the shadowed arm through EntryGate; got allow")
	}
	if !strings.Contains(reason, "SHADOW") {
		t.Fatalf("arm seam refusal should name SHADOW; got %q", reason)
	}
}

func TestEntryGateDecisionBuilderRefusesRRAtLivePrice(t *testing.T) {
	at := &AutoTrader{id: "pin-test"}
	d := &kernel.Decision{
		Action:     "open_long",
		Symbol:     "MNQ",
		StopLoss:   29048.00,
		TakeProfit: 29113.25,
	}
	// live price = 587's real fill.
	reason, refused := at.entryGateForDecision(d, 29079.25)
	if !refused {
		t.Fatalf("decision builder must refuse 587's intent at the live fill; got allow")
	}
	if !strings.Contains(reason, "R:R 1.09") {
		t.Fatalf("decision builder refusal should name R:R 1.09; got %q", reason)
	}
}

// ── NO-TRADE-RIDER (2026-09-03) — one gate, ONE ATR5m resolver ─────────────

func TestEntryGatePin36640MinSLPassesAtArmSeamATR(t *testing.T) {
	// Decision record 36640 (09-02 18:48:49): open_short SL 29231 TP 29149,
	// refused by the old wiring with "25.25 < 450.56 = 1.5×ATR5m" (dATR 300.4).
	// At the arm seam's ATR5m=12.78 from the same minutes (journal 18:45-18:47),
	// the floor is 19.17 and the 25.25pt stop PASSES.
	base := EntryIntent{
		Path: "decision", Action: "open_short", Symbol: "MNQ",
		Entry: 29205.75, Stop: 29231.00, Target: 29149.00,
		MinRR: 2.0, MinSLMult: 1.5,
	}
	pass := base
	pass.ATR5m = 12.78 // the arm seam's measured value that evening
	if reason, refused := EntryGate(pass); refused {
		t.Fatalf("36640 intent must PASS min-SL at ATR5m=12.78 (floor 19.17 < 25.25); got %q", reason)
	}
	old := base
	old.ATR5m = 300.4 // the DAILY ATR the old wiring read (→ floor 450.60)
	reason, refused := EntryGate(old)
	if !refused || !strings.Contains(reason, "450.60") {
		t.Fatalf("36640 intent at dATR 300.4 must be refused with the 450.60 floor (the current-code failure); got refused=%v %q", refused, reason)
	}
}

func TestArmSeamATR5mIsTheOneResolver(t *testing.T) {
	// Fixture provider: the SAME 1m bars both seams resolve against.
	fixture := make([]market.Kline, 120)
	now := time.Now().Add(-2 * time.Hour)
	px := 29200.0
	for i := range fixture {
		px += math.Sin(float64(i)/5) * 3.5
		fixture[i] = market.Kline{OpenTime: now.Add(time.Duration(i) * time.Minute).UnixMilli(),
			Open: px, High: px + 4, Low: px - 4, Close: px, Volume: 10}
	}
	orig := market.FuturesBarsProvider
	market.FuturesBarsProvider = func(symbol string, interval string, count int) []market.Kline {
		return fixture
	}
	t.Cleanup(func() { market.FuturesBarsProvider = orig })

	got := armSeamATR5m("MNQ")
	want := armSeamATR5mFromBars(fixture)
	if got == 0 || math.Abs(got-want) > 1e-9 {
		t.Fatalf("armSeamATR5m must BE the arm-seam expression: got %.4f want %.4f", got, want)
	}
	// Same tick, both seams: an intent whose stop is just inside the floor must
	// be refused with the SAME floor by the decision path builder that the
	// pure gate fed from the resolver would quote.
	at := &AutoTrader{id: "pin-test"}
	if got == 0 {
		t.Skip("ATR fixture degenerate; skipping floor-equality half")
	}
	floor := 1.5 * got
	live := 29200.0
	wantDist := 0.95 * floor // sub-floor → min-SL leg must fire
	d := &kernel.Decision{Action: "open_short", Symbol: "MNQ",
		StopLoss: live + wantDist, TakeProfit: live - 3*wantDist}
	reason, refused := at.entryGateForDecision(d, live)
	if !refused {
		t.Fatalf("decision path must refuse the sub-floor stop; got allow")
	}
	if !strings.Contains(reason, fmt.Sprintf("%.2f", floor)) {
		t.Fatalf("decision-path refusal must quote the same floor %.2f; got %q", floor, reason)
	}
}

func TestEntryGateDecisionTelemetryNoDoublePrefix(t *testing.T) {
	at := &AutoTrader{id: "pin-test"}
	ar := &store.DecisionAction{}
	entryGateDecisionTelemetry(at, ar, "entry_gate: R:R 1.42 below floor 2.00")
	if ar.Error != "entry_gate: R:R 1.42 below floor 2.00" {
		t.Fatalf("prefixed reason must pass through once; got %q", ar.Error)
	}
	entryGateDecisionTelemetry(at, ar, "R:R 1.42 below floor 2.00")
	if ar.Error != "entry_gate: R:R 1.42 below floor 2.00" {
		t.Fatalf("unprefixed reason must gain exactly one prefix; got %q", ar.Error)
	}
}
