package kernel

import (
	"math"
	"testing"
	"time"

	"nofx/market"
)

// A2 (mega-research 2026-08-26) — eVWAP re-anchor: 15:00 CT cash close makes
// eVWAP a REAL distinct object. The old 16:00 CT anchor produced a window
// byte-identical to session VWAP (no bars 16:00–17:00 CT) — a degenerate
// duplicate burning a seat.
func TestEVWAPDistinctFromSessionVWAP(t *testing.T) {
	loc := CTLocation()
	now := time.Date(2026, 8, 26, 18, 30, 0, 0, loc) // 18:30 CT, session-day 08-26
	var bars []market.Kline
	add := func(h, m int, price float64) {
		ct := time.Date(2026, 8, 26, h, m, 0, 0, loc)
		bars = append(bars, market.Kline{
			OpenTime: ct.UnixMilli(), CloseTime: ct.UnixMilli() + 59_999,
			Open: price, High: price + 0.5, Low: price - 0.5, Close: price, Volume: 100,
		})
	}
	// Cash-close hour (15:00–15:59) trades FAR below: these bars exist in the
	// eVWAP window but NOT in the session VWAP window (17:00 anchor).
	for i := 0; i < 4; i++ {
		add(15, i*10, 29000+float64(i))
	}
	// Post-reopen (17:00+) trades much higher.
	for i := 0; i < 8; i++ {
		add(17, i*10, 29400+float64(i))
	}

	ev := EVWAPLevels(bars, now)
	sv := SessionVWAPLevels(bars, now)
	if len(ev) != 1 || len(sv) == 0 {
		t.Fatalf("both VWAPs must render (ev=%d sv=%d)", len(ev), len(sv))
	}
	if math.Abs(ev[0].Price-sv[0].Price) < 1.0 {
		t.Fatalf("eVWAP must be DISTINCT from session VWAP after the 15:00 re-anchor (ev=%.2f sv=%.2f)", ev[0].Price, sv[0].Price)
	}
	// eVWAP must sit between the low cash-close prices and the high post-reopen
	// prices — i.e. it actually spans the close hour.
	if ev[0].Price > 29350 || ev[0].Price < 29050 {
		t.Fatalf("eVWAP %.2f must reflect the close-hour volume (in 29050..29350)", ev[0].Price)
	}

	// Before 15:00 CT the anchor rolls to the PRIOR session-day's 15:00.
	early := time.Date(2026, 8, 26, 10, 0, 0, 0, loc)
	// Same bars + a prior-day 15:00 close-hour block far below.
	var bars2 []market.Kline
	prev := func(h, m int, price float64) {
		ct := time.Date(2026, 8, 25, h, m, 0, 0, loc)
		bars2 = append(bars2, market.Kline{
			OpenTime: ct.UnixMilli(), CloseTime: ct.UnixMilli() + 59_999,
			Open: price, High: price + 0.5, Low: price - 0.5, Close: price, Volume: 100,
		})
	}
	for i := 0; i < 4; i++ {
		prev(15, i*10, 28700+float64(i))
	}
	overnight := []market.Kline{}
	for i := 0; i < 8; i++ {
		prev(19, i*10, 28800+float64(i))
		overnight = append(overnight, bars2[len(bars2)-1])
	}
	ev2 := EVWAPLevels(bars2, early)
	if len(ev2) != 1 {
		t.Fatalf("pre-15:00 roll must yield one eVWAP, got %d", len(ev2))
	}
	if ev2[0].Price > 28830 || ev2[0].Price < 28730 {
		t.Fatalf("pre-15:00 eVWAP %.2f must span the prior-day close hour", ev2[0].Price)
	}
}
