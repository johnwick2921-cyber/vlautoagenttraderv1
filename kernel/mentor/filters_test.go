package mentor

import (
	"testing"

	"vl/market"
)

// TestTriggerLineFirstBreakDrawsLine — §5.1 [D3.4 p1 @ 04:02, 08:04, 04:28]:
// the first 5m candle breaking the previous extreme draws the line at the
// BROKEN extreme, wick included, and fixes the direction.
func TestTriggerLineFirstBreakDrawsLine(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	bars := []market.Kline{
		{High: 100, Low: 96},  // previous
		{High: 103, Low: 97},  // breaks the HIGH → buy line at 100 (the broken extreme)
		{High: 104, Low: 102}, // same-side break → ignored (FIRST break only [@ 19:48])
	}
	got := TriggerTick(TriggerLine{}, bars, cfg)
	if got.Dir != SideLong || got.Price != 100 {
		t.Fatalf("line = %+v, want long @ 100 (the broken high, not the breaker's high)", got)
	}
}

// TestTriggerLineSellBreak — mirror: a candle breaking the previous LOW draws
// the sell line at that low.
func TestTriggerLineSellBreak(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	bars := []market.Kline{
		{High: 100, Low: 96},
		{High: 99, Low: 94}, // breaks the LOW → sell line at 96
	}
	got := TriggerTick(TriggerLine{}, bars, cfg)
	if got.Dir != SideShort || got.Price != 96 {
		t.Fatalf("line = %+v, want short @ 96", got)
	}
}

// TestTriggerLineMovesOnceOnReversal — [D3.4 p1 @ 11:48–13:33]: "MOVE THE LINE
// ONLY ON A REVERSAL — MỘT LẦN MỘT THÔI". The first opposite break flips the
// line; every later break leaves it alone.
func TestTriggerLineMovesOnceOnReversal(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	tl := TriggerLine{Dir: SideLong, Price: 100}
	bars := []market.Kline{
		{High: 102, Low: 101},
		{High: 99, Low: 97}, // reversal: breaks the low → line moves to sell @ 101
	}
	got := TriggerTick(tl, bars, cfg)
	if got.Dir != SideShort || got.Price != 101 || !got.Moved {
		t.Fatalf("after reversal = %+v, want short @ 101 moved=true", got)
	}
	// a later break on either side does NOT move the line again
	more := []market.Kline{
		{High: 100, Low: 98},
		{High: 105, Low: 100}, // breaks the high — ignored (already moved)
	}
	got2 := TriggerTick(got, more, cfg)
	if got2.Dir != SideShort || got2.Price != 101 || got2.OldPrice != 100 {
		t.Fatalf("second move happened: %+v (must stay short @ 101, old line 100)", got2)
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
	tl := TriggerLine{Dir: SideShort, Price: 97, Moved: true, OldPrice: 100, OldDir: SideLong}
	if ok, _, reason := TriggerVerdict(tl, 98.5); ok || reason == "" {
		t.Fatalf("between two lines must be no-trade")
	}
	if ok, side, _ := TriggerVerdict(tl, 96); !ok || side != SideShort {
		t.Fatalf("below the new sell line: ok=%v side=%q", ok, side)
	}
}

// TestTriggerLineOnRecorded5mTape — canon 53: the recorded 2026-09-15 5m tape
// (84 bars incl. the pre-RTH lead-in) must produce a first break and the
// line never moves more than once.
func TestTriggerLineOnRecorded5mTape(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	bars := loadFixture(t, "mnq_5m_2026-09-15_rth", "5m")
	got := TriggerLine{}
	movedCount := 0
	prevMoved := false
	for i := 1; i < len(bars); i++ {
		got = TriggerTick(got, bars[i-1:i+1], cfg)
		if got.Moved && !prevMoved {
			movedCount++
		}
		prevMoved = got.Moved
	}
	if got.Dir == "" {
		t.Fatal("no trigger line was drawn on the recorded 5m tape")
	}
	if movedCount > 1 {
		t.Fatalf("line moved %d times — 'MỘT LẦN MỘT THÔI'", movedCount)
	}
	t.Logf("recorded 5m tape: first line %q @ %.2f, moves=%d", got.Dir, got.Price, movedCount)
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
