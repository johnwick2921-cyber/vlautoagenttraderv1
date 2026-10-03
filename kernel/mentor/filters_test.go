package mentor

import (
	"testing"

	"vl/market"
)

// b5 builds a 5m bucket bar at minute offsets from 17:00 CT.
func b5(minFrom1700 int, high, low, close float64) market.Kline {
	ms := int64(17*60+minFrom1700) * 60_000
	return market.Kline{OpenTime: ms, CloseTime: ms + 5*60_000 - 1, High: high, Low: low, Close: close}
}

// TestTriggerLineFirstBreakDrawsLine — §5.1 [D3.4 p1 @ 04:02, 08:04, 04:28]:
// the first 5m candle breaking the previous extreme draws the line at the
// BROKEN extreme, wick included, and fixes the direction.
func TestTriggerLineFirstBreakDrawsLine(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	bars := []market.Kline{
		b5(0, 100, 96, 97),    // 17:00 bucket, recorded (no previous to break)
		b5(5, 103, 97, 99),    // breaks the HIGH → buy line at 100
		b5(10, 104, 102, 103), // same-side break → ignored (FIRST break only [@ 19:48])
	}
	got := TriggerTick(TriggerLine{}, bars, 5, cfg)
	if got.Dir != SideLong || got.Price != 100 {
		t.Fatalf("line = %+v, want long @ 100 (the broken high, not the breaker's high)", got)
	}
	// the LAST bucket is the forming tail — it never commits (re-evaluated
	// every tick); the committed cursor is the second-to-last bucket.
	if got.LastBucket != bars[1].OpenTime {
		t.Fatalf("LastBucket = %d, want %d", got.LastBucket, bars[1].OpenTime)
	}
}

// TestTriggerFiresOnForming5mBucket — confirmation rule (a) [Buy-Sell Setup
// Trigger @11:54–12:34; deck slide 23 "Không cần đợi nến đóng"]: "PHÁ" (a
// break of a candle's extreme) fires the trigger line INTRABAR — no close
// needed. The evaluator feeds the still-forming 5m bucket and a break on it
// draws the line at once.
func TestTriggerFiresOnForming5mBucket(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	bars := []market.Kline{
		b5(0, 100, 96, 97),  // closed bucket (no previous to break)
		b5(5, 103, 99, 101), // the FORMING bucket breaks the previous high
	}
	got := TriggerTick(TriggerLine{}, bars, 5, cfg)
	if got.Dir != SideLong || got.Price != 100 {
		t.Fatalf("the forming bucket's break must fire the line intrabar: %+v, want long @ 100", got)
	}
}

// TestTriggerLineSellBreak — mirror: a candle breaking the previous LOW draws
// the sell line at that low.
func TestTriggerLineSellBreak(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	bars := []market.Kline{
		b5(0, 100, 96, 98),
		b5(5, 99, 94, 95), // breaks the LOW → sell line at 96
	}
	got := TriggerTick(TriggerLine{}, bars, 5, cfg)
	if got.Dir != SideShort || got.Price != 96 {
		t.Fatalf("line = %+v, want short @ 96", got)
	}
}

// TestTriggerTickIsIdempotentAndIncremental — B2: the persisted LastBucket
// means a re-tick on the same history changes nothing, and extending the
// history by one bucket processes exactly that bucket.
func TestTriggerTickIsIdempotentAndIncremental(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	bars := []market.Kline{
		b5(0, 100, 96, 97),
		b5(5, 103, 97, 99),
	}
	first := TriggerTick(TriggerLine{}, bars, 5, cfg)
	again := TriggerTick(first, bars, 5, cfg)
	if again != first {
		t.Fatalf("re-tick changed state: %+v → %+v", first, again)
	}
	bars = append(bars, b5(10, 105, 100, 104)) // same-direction break → no move, but processed
	third := TriggerTick(first, bars, 5, cfg)
	// the new tail (bars[2]) is forming — the committed cursor is bars[1].
	if third.LastBucket != bars[1].OpenTime {
		t.Fatalf("extended tick did not process only the new bucket: %+v", third)
	}
	if third.Dir != first.Dir || third.Price != first.Price {
		t.Fatalf("same-direction break moved the line: %+v → %+v", first, third)
	}
}

// TestTriggerLineMovesOnEveryReversalOnceEach — B3 (CTO ruling): each
// reversal moves the line once; later breaks in the SAME direction never move
// it; the NEXT reversal moves it again [D3.4 p1 @ 11:48–13:33].
func TestTriggerLineMovesOnEveryReversalOnceEach(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	tl := TriggerLine{Dir: SideLong, Price: 100, LastBucket: b5(0, 0, 0, 0).OpenTime, LastBar: b5(0, 102, 101, 102)}
	bars := []market.Kline{
		b5(5, 99, 97, 97),     // reversal: breaks the low → line moves to sell @ 101
		b5(10, 98, 96, 96),    // same-direction short break → NO move
		b5(15, 103, 100, 103), // next reversal: breaks the high → line moves to long @ 96
	}
	got := TriggerTick(tl, bars, 5, cfg)
	if got.Dir != SideLong || got.Price != 98 {
		t.Fatalf("after two reversals = %+v, want long @ 98 (the second reversal's broken high)", got)
	}
	if got.OldPrice != 101 || got.OldDir != SideShort {
		t.Fatalf("old line = %v/%q, want 101/short (the line before the LAST reversal)", got.OldPrice, got.OldDir)
	}
}

// TestTriggerVerdictSideAndNoTradeZone — [D3.4 p1 @ 10:21, 06:22, 09:30, 16:38]:
// entries only on the trigger side; between two opposing lines → no trade.
func TestTriggerVerdictSideAndNoTradeZone(t *testing.T) {
	// long line at 100: above allowed, below refused
	if ok, side, _ := TriggerVerdict(TriggerLine{Dir: SideLong, Price: 100}, 101); !ok || side != SideLong {
		t.Fatalf("above the buy line: ok=%v side=%q", ok, side)
	}
	if ok, _, reason := TriggerVerdict(TriggerLine{Dir: SideLong, Price: 100}, 99); ok || reason == "" {
		t.Fatalf("below the buy line must be refused with a reason")
	}
	// after a reversal the zone between old (100) and new (97) lines is no-trade
	tl := TriggerLine{Dir: SideShort, Price: 97, OldPrice: 100, OldDir: SideLong}
	if ok, _, reason := TriggerVerdict(tl, 98.5); ok || reason == "" {
		t.Fatalf("between two lines must be no-trade")
	}
	// R4: the zone INCLUDES the lines themselves — price sitting exactly ON
	// either line is still no-trade [D3.4 p1 @ 16:56–17:17].
	for _, p := range []float64{100, 97} {
		if ok, _, reason := TriggerVerdict(tl, p); ok || reason == "" {
			t.Fatalf("price exactly on a zone line (%v) must be no-trade", p)
		}
	}
	if ok, side, _ := TriggerVerdict(tl, 96); !ok || side != SideShort {
		t.Fatalf("below the new sell line: ok=%v side=%q", ok, side)
	}
	// above the old buy line is the WRONG side of the current (sell) line —
	// still refused, but for the wrong-side reason, not the zone.
	if ok, _, reason := TriggerVerdict(tl, 101); ok || reason == "" {
		t.Fatalf("above the current sell line must be refused")
	}
}

// TestTriggerLineOnRecorded5mTape — canon 53: the recorded 2026-09-15 5m tape
// (84 buckets incl. the pre-RTH lead-in) must draw a line and process every
// bucket exactly once.
func TestTriggerLineOnRecorded5mTape(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	bars := loadFixture(t, "mnq_5m_2026-09-15_rth", "5m")
	got := TriggerTick(TriggerLine{}, bars, 5, cfg)
	if got.Dir == "" {
		t.Fatal("no trigger line was drawn on the recorded 5m tape")
	}
	// the last bucket is the forming tail — it never commits.
	if got.LastBucket != bars[len(bars)-2].OpenTime {
		t.Fatalf("LastBucket = %d, want the second-to-last bucket %d", got.LastBucket, bars[len(bars)-2].OpenTime)
	}
	// re-tick idempotence on the real tape (B2)
	again := TriggerTick(got, bars, 5, cfg)
	if again != got {
		t.Fatal("re-tick on the recorded tape changed state (B2)")
	}
	t.Logf("recorded 5m tape: line %q @ %.2f", got.Dir, got.Price)
}

// TestISBConflictVerdict — §5.3 / §12 [D4.2 p1 @ 13:59, 14:35]: a live 5m ISB
// and a live 15m churn in OPPOSITE directions → no trade at all.
func TestISBConflictVerdict(t *testing.T) {
	// 15m churn: bars 0-2 inside bar 0's range, bar 0 direction long.
	// 5m ISB: bar 3 inside bar 2, bar 3 direction short → conflict.
	bars := []market.Kline{
		{Open: 100, Close: 110, High: 112, Low: 98}, // 15m first (long)
		{Open: 104, Close: 108, High: 111, Low: 100},
		{Open: 105, Close: 109, High: 110, Low: 101}, // 15m ends
		{Open: 109, Close: 106, High: 110, Low: 102}, // 5m ISB (short) inside bar 2
	}
	if !ISBConflictVerdict(bars) {
		t.Fatal("opposite 15m/5m ISB directions must conflict")
	}
	// same direction → no conflict
	bars[3] = market.Kline{Open: 106, Close: 109.5, High: 110, Low: 102} // long
	if ISBConflictVerdict(bars) {
		t.Fatal("same-direction ISBs must not conflict")
	}
	// no 5m ISB → no conflict even with a churn
	bars[3] = market.Kline{Open: 102, Close: 113, High: 114, Low: 101} // body escapes
	if ISBConflictVerdict(bars) {
		t.Fatal("no live 5m ISB must never conflict")
	}
}

// TestMidRangeISBOnly — §12 [D3.2 p2 @ 08:34]: between two levels, PHL/PLH is
// banned and only the ISB survives.
func TestMidRangeISBOnly(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.RangeGapPts = 100
	levels := []Level{
		{Key: "up", Price: 105},
		{Key: "down", Price: 95},
	}
	if !MidRange(levels, 100, cfg) {
		t.Fatal("price between two levels within the gap must be mid-range")
	}
	if ok, _ := SetupPermittedVerdict("PHL", levels, 100, cfg); ok {
		t.Fatal("PHL/PLH must be banned mid-range")
	}
	if ok, reason := SetupPermittedVerdict("ISB", levels, 100, cfg); !ok || reason != "" {
		t.Fatalf("ISB must survive mid-range: ok=%v reason=%q", ok, reason)
	}
	// one-sided (no level below within the gap) → not mid-range
	if MidRange([]Level{{Key: "up", Price: 105}}, 100, cfg) {
		t.Fatal("one-sided level set is not mid-range")
	}
	// disabled (RangeGapPts 0) → filter off (L4)
	if MidRange(levels, 100, DefaultConfig()) {
		t.Fatal("disabled mid-range filter fired")
	}
}
