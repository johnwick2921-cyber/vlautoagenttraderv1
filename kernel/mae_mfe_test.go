package kernel

import (
	"testing"

	"nofx/market"
)

// synthetic 1m hold: entry at t0, exit at t3; a spike up to 15650 and dip to 15550.
func holdBars() []market.Kline {
	mk := func(i int, hi, lo float64) market.Kline {
		open := int64(i) * 60_000
		return market.Kline{OpenTime: open, High: hi, Low: lo, CloseTime: open + 60_000 - 1}
	}
	return []market.Kline{
		mk(0, 15610, 15595),
		mk(1, 15650, 15600), // favorable spike
		mk(2, 15605, 15550), // adverse dip
		mk(3, 15620, 15600),
	}
}

func TestComputeExcursionLong(t *testing.T) {
	bars := holdBars()
	ex := ComputeExcursion(15600, "LONG", bars, 0, 3*60_000)
	if ex.MFE != 50 { // 15650 - 15600
		t.Fatalf("long MFE = %v want 50", ex.MFE)
	}
	if ex.MAE != 50 { // 15600 - 15550
		t.Fatalf("long MAE = %v want 50", ex.MAE)
	}
}

func TestComputeExcursionShort(t *testing.T) {
	bars := holdBars()
	ex := ComputeExcursion(15600, "SHORT", bars, 0, 3*60_000)
	if ex.MFE != 50 { // 15600 - 15550 (down is favorable for a short)
		t.Fatalf("short MFE = %v want 50", ex.MFE)
	}
	if ex.MAE != 50 { // 15650 - 15600
		t.Fatalf("short MAE = %v want 50", ex.MAE)
	}
}

func TestComputeExcursionWindowBounds(t *testing.T) {
	bars := holdBars()
	// Exclude the adverse dip (bar 2) and the favorable... keep only bars 0-1.
	ex := ComputeExcursion(15600, "LONG", bars, 0, 1*60_000)
	if ex.MFE != 50 || ex.MAE != 5 { // bar0 low 15595 → MAE 5; bar1 high 15650 → MFE 50
		t.Fatalf("windowed = MAE %v MFE %v want 5/50", ex.MAE, ex.MFE)
	}
}

func TestComputeExcursionDegenerate(t *testing.T) {
	if ex := ComputeExcursion(0, "LONG", holdBars(), 0, 60_000); ex != (Excursion{}) {
		t.Fatalf("bad entry → zero excursion")
	}
	if ex := ComputeExcursion(15600, "LONG", holdBars(), 100, 50); ex != (Excursion{}) {
		t.Fatalf("exit before entry → zero excursion")
	}
}
