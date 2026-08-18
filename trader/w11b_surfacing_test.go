package trader

import (
	"path/filepath"
	"testing"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// W11b(1) — installLevelStateProvider maps a DetectedLevel to its persisted
// cross-session state via the SAME identity W7's writer uses (type-from-label +
// price-bin): consumed → "done", decayed → its grade, unknown → "" (fresh).
func TestW11bLevelStateProviderReadsStore(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	defer func() { kernel.LevelStateProvider = nil }() // don't leak the global

	const px = 30050.0
	label := "PDH"
	key := store.MakeLevelKey("t1", "MNQ", kernel.LevelTypeFromLabel(label), "", kernel.LevelBinIndex(px))

	// seed a consumed level.
	if err := st.LevelState().EnsureLevel(&store.LevelStateDB{
		TraderID: "t1", Symbol: "MNQ", LevelType: kernel.LevelTypeFromLabel(label), BinIndex: kernel.LevelBinIndex(px), Price: px,
	}); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if err := st.LevelState().MarkConsumed(key); err != nil {
		t.Fatalf("consume: %v", err)
	}

	installLevelStateProvider(nil, st)

	if got := kernel.LevelStateProvider("t1", "MNQ", kernel.DetectedLevel{Price: px, Label: label}); got != "done" {
		t.Fatalf("consumed level must read done, got %q", got)
	}
	// an unknown level → fresh ("").
	if got := kernel.LevelStateProvider("t1", "MNQ", kernel.DetectedLevel{Price: 12345, Label: "RN"}); got != "" {
		t.Fatalf("unknown level must read fresh (\"\"), got %q", got)
	}
}

// W11b(2) — overnight-gap inputs derive from the last two daily bars (prior close +
// session open); <2 bars → (0,0) so the regime gap stays inert (honest).
func TestW11bOvernightGapInputs(t *testing.T) {
	// not enough data.
	if pc, so := kernel.PriorCloseSessionOpen(nil); pc != 0 || so != 0 {
		t.Fatal("no bars → (0,0)")
	}
	daily := []market.Kline{
		{Open: 100, Close: 110}, // prior day
		{Open: 115, Close: 120}, // current session (gap up +5 vs prior close 110)
	}
	pc, so := kernel.PriorCloseSessionOpen(daily)
	if pc != 110 || so != 115 {
		t.Fatalf("want priorClose=110 sessionOpen=115, got %.0f/%.0f", pc, so)
	}
	// and it flows into a real gap in ComputeRegime.
	r := kernel.ComputeRegime(kernel.RegimeInputs{
		Price: 118, DailyBars: makeATRBars(120), PriorClose: pc, SessionOpen: so,
	})
	if !r.HasGap || r.OvernightGapATR == 0 {
		t.Fatalf("gap must be computed, got HasGap=%v gapATR=%.3f", r.HasGap, r.OvernightGapATR)
	}
}

// makeATRBars builds >=15 daily bars so ComputeRegime's ATR14 (>0) enables the gap.
func makeATRBars(n int) []market.Kline {
	var bars []market.Kline
	px := 100.0
	for i := 0; i < n; i++ {
		bars = append(bars, market.Kline{Open: px, High: px + 10, Low: px - 10, Close: px + 2})
		px += 1
	}
	return bars
}
