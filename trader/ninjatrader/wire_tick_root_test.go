package ninjatrader

import (
	"math"
	"testing"
)

// W1b E12 verifier defect 4 — THE WIRE'S TICK COMES FROM THE INSTRUMENT ROOT.
//
// InstrumentTickSize keyed its table on the raw symbol, so a contract-code
// symbol (M2KU6) or a qualified one (M2K 12-26) fell through to the 0.25 index
// default while the M2K tick is 0.10. EntryGate legs 5/6 judge through the SAME
// function (trader/entry_gate.go entryGateWirePrices), so the root resolution
// lives there once, for both sides.
//
// Production call site: PlaceLimitEntry over the real loopback, trader symbol
// in contract-code form.
func TestWireTickResolvesContractCodeRoot(t *testing.T) {
	s, _, _, frames := stopEntryServer(t)
	tr := NewTCPTrader(s, "M2KU6", "Sim101")
	// 1990.20 / 2029.80 are on the 0.10 grid; the 0.25 grid would send 1990.00 / 2029.75.
	if _, err := tr.PlaceLimitEntry("M2KU6", "long", 1, 2000.00, 1990.20, 2029.80); err != nil {
		t.Fatalf("limit entry refused: %v", err)
	}
	p := awaitFrame(t, frames, "M2KU6 limit")
	if math.Abs(p.StopLoss-1990.20) > 1e-9 || math.Abs(p.TakeProfit-2029.80) > 1e-9 { // n×0.1 is inexact in binary
		t.Fatalf("M2KU6 wire sent SL %.2f TP %.2f, want 1990.20 / 2029.80 (M2K tick 0.10, not the 0.25 default)", p.StopLoss, p.TakeProfit)
	}
}

func TestInstrumentTickSizeResolvesRoot(t *testing.T) {
	for sym, want := range map[string]float64{
		"M2KU6": 0.10, "M2K 12-26": 0.10, "RTY.c.0": 0.10, "MYMZ6": 1.0,
		"MNQU6": 0.25, "MNQ 06-26": 0.25, "NQ.c.0": 0.25, "mnq": 0.25,
		"DOGEUSDT": 0.25, // not a CME root: the unchanged table default
	} {
		if got := InstrumentTickSize(sym); got != want {
			t.Errorf("InstrumentTickSize(%q) = %v, want %v", sym, got, want)
		}
	}
}
