package ninjatrader

import "testing"

// P0 pnl-record-integrity (2026-08-20) — the #526 class: a manual account
// flatten's frame qty must never be attributed to the bot's row. Pure math
// pins for each error class (V2):
func TestPnLAttributionMath(t *testing.T) {
	pv := 2.0 // MNQ $/pt
	entry, exit := 29626.25, 29660.964286
	rowQty, frameQty := 1.0, 21.0

	// E6/E4 — the live bug: frame qty applied to a 1-lot short.
	wrong := (entry - exit) * frameQty * pv
	right := (entry - exit) * rowQty * pv
	if wrong > -1457 || wrong < -1459 {
		t.Fatalf("sanity: the wrong math should reproduce −1458 (got %.2f)", wrong)
	}
	if right < -70 || right > -69 {
		t.Fatalf("attributed math must give the true −69.43 (got %.2f)", right)
	}

	// E2 — units: a ticks mixup would be 4× (pv=8/pt-equivalent); pinned.
	if (entry-exit)*rowQty*8 == right {
		t.Fatal("ticks-vs-points must differ")
	}
	// E3 — sign: a long-side formula on a short flips the sign.
	if (exit-entry)*rowQty*pv == right {
		t.Fatal("side formulas must differ")
	}
}
