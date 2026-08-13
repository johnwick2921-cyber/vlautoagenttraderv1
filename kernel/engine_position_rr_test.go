// F1 — REAL risk:reward. These tests prove the R:R gate uses the ACTUAL entry
// reference (current-price snapshot, or an explicit limit price) instead of the old
// synthetic 20%-of-range entry that pinned every R:R at 4.0. Hand-computed cases are
// asserted side-by-side with the code's verdict.
package kernel

import (
	"testing"

	"nofx/market"
	"nofx/telemetry"
)

// mdCtx builds a Context whose MarketDataMap gives `sym` the entry-reference price.
func mdCtx(sym string, price float64) *Context {
	return &Context{MarketDataMap: map[string]*market.Data{sym: {Symbol: sym, CurrentPrice: price}}}
}

// mnqRR builds a valid MNQ open (notional/leverage/confidence all fine) with the
// given side + stop/target, so validateDecision reaches the R:R block.
func mnqRR(action string, sl, tp float64) *Decision {
	return &Decision{
		Symbol: "MNQ", Action: action, Leverage: 1,
		PositionSizeUSD: 60000, StopLoss: sl, TakeProfit: tp, Confidence: 90,
	}
}

// TestF1_RealRR_HandComputedMatrix walks 6 hand-computed cases (long + short, pass +
// fail, exact-threshold, and a zero/negative-risk reject) and asserts the code's
// accept/reject matches the math. entry is supplied via the ctx snapshot.
func TestF1_RealRR_HandComputedMatrix(t *testing.T) {
	eq, b, a, br, ar := gateArgs()

	cases := []struct {
		name       string
		action     string
		entry      float64
		sl, tp     float64
		minRR      float64
		wantRR     float64 // hand-computed (for documentation; 0 for the reject case)
		wantReject bool
	}{
		// LONG: risk = entry-SL, reward = TP-entry.
		{"long 3.0 exact-threshold PASS", "open_long", 100, 90, 130, 3.0, 3.0, false}, // 30/10=3.0 ≥ 3.0
		{"long 2.5 below-min FAIL", "open_long", 100, 90, 125, 3.0, 2.5, true},         // 25/10=2.5 < 3.0
		{"long 2.0 above-min PASS", "open_long", 100, 80, 140, 1.5, 2.0, false},        // 40/20=2.0 ≥ 1.5
		// SHORT: risk = SL-entry, reward = entry-TP.
		{"short 3.0 exact-threshold PASS", "open_short", 100, 110, 70, 3.0, 3.0, false}, // 30/10=3.0 ≥ 3.0
		{"short 1.5 below-min FAIL", "open_short", 100, 110, 85, 3.0, 1.5, true},         // 15/10=1.5 < 3.0
		// Zero/negative risk: stop ABOVE entry for a long (SL<TP still holds, so the
		// SL-vs-TP check passes; the entry-vs-SL wrong side is validateDecision's).
		{"long stop-through-entry REJECT", "open_long", 100, 105, 130, 3.0, 0, true}, // risk = -5 ≤ 0
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := mnqRR(c.action, c.sl, c.tp)
			err := validateDecision(d, eq, b, a, br, ar, c.minRR, 0, 20, mdCtx("MNQ", c.entry))
			if c.wantReject && err == nil {
				t.Fatalf("expected REJECT (hand R:R=%.2f, min=%.2f) but passed", c.wantRR, c.minRR)
			}
			if !c.wantReject && err != nil {
				t.Fatalf("expected PASS (hand R:R=%.2f, min=%.2f) but rejected: %v", c.wantRR, c.minRR, err)
			}
		})
	}
}

// TestF1_LimitEntryWins: an explicit AI limit price (d.Price) is the entry reference,
// overriding the current-price snapshot.
func TestF1_LimitEntryWins(t *testing.T) {
	eq, b, a, br, ar := gateArgs()
	// Snapshot says 999 (would give a wildly different R:R); the AI limit price 100
	// must win → risk 10, reward 30, R:R 3.0 ≥ 3.0 → PASS.
	d := mnqRR("open_long", 90, 130)
	d.Price = 100.0
	if err := validateDecision(d, eq, b, a, br, ar, 3.0, 0, 20, mdCtx("MNQ", 999)); err != nil {
		t.Fatalf("explicit limit entry 100 should give R:R 3.0 and PASS, got: %v", err)
	}
	// Same limit but min 3.5 → 3.0 < 3.5 → FAIL (proves the limit price drove it).
	d2 := mnqRR("open_long", 90, 130)
	d2.Price = 100.0
	if err := validateDecision(d2, eq, b, a, br, ar, 3.5, 0, 20, mdCtx("MNQ", 999)); err == nil {
		t.Fatal("limit-entry R:R 3.0 should be REJECTED at min 3.5")
	}
}

// TestF1_NilGuards: no ctx / no market data → the R:R gate fails OPEN (skips), never
// panics, so a data gap can't wedge the validator.
func TestF1_NilGuards(t *testing.T) {
	eq, b, a, br, ar := gateArgs()
	// Would-be R:R = 0.1 (terrible), but nil ctx → no entry ref → gate skipped → PASS.
	if err := validateDecision(mnqRR("open_long", 90, 91), eq, b, a, br, ar, 3.0, 0, 20, nil); err != nil {
		t.Fatalf("nil ctx must skip the R:R gate (fail-open), got: %v", err)
	}
	// ctx present but the symbol absent → entryRef 0 → skipped → PASS.
	if err := validateDecision(mnqRR("open_long", 90, 91), eq, b, a, br, ar, 3.0, 0, 20, mdCtx("ES", 100)); err != nil {
		t.Fatalf("missing symbol in ctx must skip the R:R gate, got: %v", err)
	}
}

// TestF1_RRGateCounter: a rejection increments the B6 "rr_gate" counter for the
// deciding trader.
func TestF1_RRGateCounter(t *testing.T) {
	telemetry.RolloverGateBlocks(1) // adopt a session-day
	_, before := telemetry.GateBlockSnapshot()
	start := before["TID"]["rr_gate"]

	eq, b, a, br, ar := gateArgs()
	ctx := mdCtx("MNQ", 100)
	ctx.TraderID = "TID"
	// R:R 2.5 < 3.0 → reject → counter++.
	_ = validateDecision(mnqRR("open_long", 90, 125), eq, b, a, br, ar, 3.0, 0, 20, ctx)

	_, after := telemetry.GateBlockSnapshot()
	if after["TID"]["rr_gate"] != start+1 {
		t.Fatalf("rr_gate counter = %d, want %d", after["TID"]["rr_gate"], start+1)
	}
}
