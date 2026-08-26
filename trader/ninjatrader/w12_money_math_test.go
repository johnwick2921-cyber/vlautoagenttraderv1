package ninjatrader

import (
	"math"
	"testing"

	"nofx/market"
)

// W12 — MNQ money-math oracle. The realized-P&L formula (close_sync.go:135-148) is
// audited against THREE REAL closed SIM trades (from trader_positions, all qty1
// fee0) plus a synthetic qty>1 row. MNQ point value = $2.00 (market/futures_symbol.go).
// pnlRef is an INDEPENDENT reimplementation of the production formula (oracle A).
func pnlRef(side string, entry, exit, qty, pv float64) float64 {
	if entry <= 0 {
		return 0
	}
	if side == "LONG" {
		return (exit - entry) * qty * pv
	}
	return (entry - exit) * qty * pv
}

func TestW12MNQPointValueAndPnL(t *testing.T) {
	pv := market.FuturesPointValue("MNQ")
	if pv != 2.0 {
		t.Fatalf("MNQ point value must be $2.00/pt, got %v", pv)
	}

	// The three REAL closed trades — the independent formula must reproduce the
	// stored realized_pnl TO THE CENT.
	cases := []struct {
		name             string
		side             string
		entry, exit, qty float64
		wantPnL          float64
	}{
		{"id515 LONG winner", "LONG", 29901.25, 30025.00, 1, 247.5},
		{"id519 LONG loser", "LONG", 30199.50, 30179.25, 1, -40.5},
		{"id512 SHORT loser", "SHORT", 29875.25, 29877.00, 1, -3.5},
		// synthetic: 3 contracts scales linearly.
		{"synthetic qty3 LONG", "LONG", 30000.00, 30010.00, 3, 60.0}, // 10pts × $2 × 3
	}
	for _, c := range cases {
		got := pnlRef(c.side, c.entry, c.exit, c.qty, pv)
		if math.Abs(got-c.wantPnL) > 1e-9 {
			t.Fatalf("%s: pnl = %.4f, want %.4f", c.name, got, c.wantPnL)
		}
	}

	// property: LONG and SHORT are sign-symmetric for the same move.
	if pnlRef("LONG", 100, 105, 1, pv) != -pnlRef("SHORT", 100, 105, 1, pv) {
		t.Fatal("LONG/SHORT must be sign-symmetric for the same price move")
	}
	// property: zero/absent entry → zero P&L (never fabricated).
	if pnlRef("LONG", 0, 105, 1, pv) != 0 {
		t.Fatal("no entry reference → zero P&L")
	}
}

// W12 — tick rounding oracle: every price reaching an ORDER must be MNQ-tick (0.25)
// aligned. RoundToTick = round-half-away-from-zero of price/tick, ×tick.
func TestW12TickRounding(t *testing.T) {
	if ts := InstrumentTickSize("MNQ"); ts != 0.25 {
		t.Fatalf("MNQ tick must be 0.25, got %v", ts)
	}
	cases := []struct{ in, want float64 }{
		{29901.30, 29901.25}, // .30 → nearest .25
		{29901.37, 29901.25}, // .37 closer to .25 than .50
		{29901.38, 29901.50}, // .38 closer to .50
		{29901.13, 29901.25}, // .13 → .25
		{29901.125, 29901.25}, // exact half → away from zero → .25
		{29901.25, 29901.25},  // already aligned (idempotent)
	}
	for _, c := range cases {
		got := RoundToTick(c.in, 0.25)
		if math.Abs(got-c.want) > 1e-9 {
			t.Fatalf("RoundToTick(%.4f) = %.4f, want %.4f", c.in, got, c.want)
		}
		// property: rounding is idempotent (tick-aligned prices are fixed points).
		if r2 := RoundToTick(got, 0.25); math.Abs(r2-got) > 1e-9 {
			t.Fatalf("RoundToTick not idempotent at %.4f", got)
		}
		// property: result is a multiple of the tick.
		if q := got / 0.25; math.Abs(q-math.Round(q)) > 1e-9 {
			t.Fatalf("%.4f is not tick-aligned", got)
		}
	}
	// tick<=0 → passthrough (never divide by zero).
	if RoundToTick(123.456, 0) != 123.456 {
		t.Fatal("tick<=0 must pass the price through unchanged")
	}
}
