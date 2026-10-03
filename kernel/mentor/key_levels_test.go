package mentor

import (
	"testing"

	"vl/market"
)

// TestKeyLevelsPruneKeepsTheMoreRecent — §4.3 step 5 [D5.3 p1 @ 12:55–17:30]:
// two levels < 20 pts apart → delete one, KEEP THE MORE RECENT.
func TestKeyLevelsPruneKeepsTheMoreRecent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.KeyLevelRTHOnly = false
	cfg.KeyLevelPrunePts = 20
	// alternate colour every bar; opens walk up 10 pts at a time →
	// of every pair < 20 pts apart only the more recent survives: the kept
	// set is {110, 130, 150, 170} (every pair >= 20 apart, newest-first wins).
	bars := []market.Kline{}
	for i := 0; i < 8; i++ {
		b := market.Kline{OpenTime: int64(i) * 60_000, Open: 100 + float64(i)*10}
		if i%2 == 0 {
			b.Close = b.Open + 1 // green
		} else {
			b.Close = b.Open - 1 // red
		}
		bars = append(bars, b)
	}
	got := KeyLevels(bars, cfg)
	if len(got) != 4 {
		t.Fatalf("levels = %d, want 4 (pairs <20 apart collapse newest-first): %+v", len(got), got)
	}
	for _, want := range []float64{110, 130, 150, 170} {
		found := false
		for _, l := range got {
			if l.Price == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("kept set %+v missing %v (every pair < 20 keeps the more recent)", got, want)
		}
	}
	for i, a := range got {
		for j := i + 1; j < len(got); j++ {
			if abs(a.Price-got[j].Price) < cfg.KeyLevelPrunePts {
				t.Fatalf("kept pair %v/%v closer than prune: %+v", a.Price, got[j].Price, got)
			}
		}
	}
}

// TestKeyLevelsLineAtSecondCandleOpen — §4.3 step 4 [D5.3 p1 @ 12:55]:
// at a colour change the line is the OPEN of the SECOND candle
// (red→green: open of the green; green→red: open of the red).
func TestKeyLevelsLineAtSecondCandleOpen(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.KeyLevelRTHOnly = false
	cfg.KeyLevelPrunePts = 0.5 // no pruning — keep every change
	bars := []market.Kline{
		{OpenTime: 1, Open: 100, Close: 101}, // green
		{OpenTime: 2, Open: 103, Close: 102}, // red (change) → level at 103
		{OpenTime: 3, Open: 104, Close: 105}, // green (change) → level at 104
		{OpenTime: 4, Open: 106, Close: 107}, // green (no change)
	}
	got := KeyLevels(bars, cfg)
	if len(got) != 2 {
		t.Fatalf("levels = %d, want 2: %+v", len(got), got)
	}
	if got[0].Price != 103 || got[1].Price != 104 {
		t.Fatalf("levels = %v, want [103 104] (opens of the SECOND candle of each change)", got)
	}
}

// TestKeyLevelsRTHOnly — §4.3 step 1 [D5.3 p1 @ 12:55]: the walk runs on the
// REGULAR TRADING HOURS session. Bars outside 08:30–15:00 CT never produce a
// level even where a colour change exists.
func TestKeyLevelsRTHOnly(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.KeyLevelPrunePts = 0.5
	ms := int64(60_000)
	// 2026-09-15 08:28 CT is outside RTH: epoch-minutes since 1970-01-01 CT.
	// Use ctOf-consistent minutes directly: 08:28 CT and 08:31 CT.
	mk := func(hh, mm int, open, close float64) market.Kline {
		return market.Kline{OpenTime: (int64(19645)*24*60 + int64(hh*60+mm)) * ms, Open: open, Close: close}
	}
	bars := []market.Kline{
		mk(8, 28, 100, 101), // green, OUTSIDE RTH
		mk(8, 29, 103, 102), // red — a colour change, but both bars outside RTH
		mk(8, 31, 104, 105), // green — inside RTH, change vs the 08:29 bar
		mk(8, 32, 106, 107), // green — no change
	}
	got := KeyLevels(bars, cfg)
	if len(got) != 0 {
		t.Fatalf("RTH-only walk produced levels from outside-RTH bars: %+v", got)
	}
}

// TestKeyLevelsOnRecordedRTHDay — §4.3 on the bot's own tape (canon 53):
// the recorded 2026-09-15 RTH day must yield a sane pruned set (DS-108 §2.1
// measured ~200 raw colour changes/day, pruned to ~10–20) and every kept
// pair must respect the 20-pt prune.
func TestKeyLevelsOnRecordedRTHDay(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	bars := loadFixture(t, "mnq_1m_2026-09-15_rth", "1m")
	raw := countColourChanges(bars)
	got := KeyLevels(bars, cfg)
	if len(got) < 5 || len(got) > 60 {
		t.Fatalf("pruned levels = %d (raw changes %d), want ~10-20 per DS-108: %+v", len(got), raw, got)
	}
	for i := 1; i < len(got); i++ {
		if abs(got[i].Price-got[i-1].Price) < cfg.KeyLevelPrunePts {
			t.Fatalf("levels %d/%d are %.2f apart (< %v prune): %+v", i-1, i, abs(got[i].Price-got[i-1].Price), cfg.KeyLevelPrunePts, got)
		}
	}
}

// countColourChanges counts raw colour transitions on closed bars (§4.3 step 2).
func countColourChanges(bars []market.Kline) int {
	n := 0
	for i := 1; i < len(bars); i++ {
		if candleColour(bars[i-1]) != candleColour(bars[i]) {
			n++
		}
	}
	return n
}

// TestKeyLevelsDisabledIsNil — L4: with mentor_mode OFF the evaluator emits
// nothing (byte-identical bot).
func TestKeyLevelsDisabledIsNil(t *testing.T) {
	bars := loadFixture(t, "mnq_1m_2026-09-15_rth", "1m")
	if got := KeyLevels(bars, Config{}); got != nil {
		t.Fatalf("disabled config produced levels: %+v", got)
	}
	if got := EMALevels(bars, Config{}); got != nil {
		t.Fatalf("disabled config produced EMA levels: %+v", got)
	}
}

// TestEMA34LevelTracksCloses — EMA 34 computed on the recorded 1m day; the
// level is a single price equal to the standard EMA seeded at the first close
// (§11; DS-106: ema_periods knob).
func TestEMA34LevelTracksCloses(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.EMATFMinutes = 1
	bars := loadFixture(t, "mnq_1m_2026-09-15_rth", "1m")
	got := EMALevels(bars, cfg)
	if len(got) != 2 {
		t.Fatalf("EMA levels = %d, want 2 (34 and 9): %+v", len(got), got)
	}
	if got[0].Kind != KindEMA34 || got[1].Kind != KindEMA9 {
		t.Fatalf("kinds = %v/%v, want ema34/ema9", got[0].Kind, got[1].Kind)
	}
	if got[0].Price != emaValue(bars, 34) || got[1].Price != emaValue(bars, 9) {
		t.Fatalf("EMA values mismatch the standard EMA computation")
	}
}
