package mentor

import (
	"vl/market"
)

// Filters is the direction-and-context gate set (PLAN v1 §4). Every filter is
// pure: bars/levels in, a verdict out.

// ── 5-minute trigger line [§5.1, D3.4 p1 @ 04:02–13:33] ─────────────────────
// BUY trigger = a 5m candle breaks the previous candle's HIGH; SELL = breaks
// the previous LOW [@ 04:02, 08:04]. A line is drawn at the BROKEN extreme,
// WICK INCLUDED [@ 04:28]. Entries only on the trigger side [@ 10:21]. The
// line moves ONLY on a reversal, once [@ 11:48–13:33]. Between two opposing
// lines → no trade [@ 16:38].

type TriggerLine struct {
	Dir   Side    // allowed direction ("" before the first break)
	Price float64 // the broken extreme (wick included)
	Moved bool    // the line has already moved on a reversal — no more moves

	// OldPrice/oldDir are the line from BEFORE the reversal move; the zone
	// strictly between the two lines is the no-trade zone [@ 16:38].
	OldPrice float64
	OldDir   Side
}

// TriggerTick advances the trigger state over the closed 5m bars since the
// last tick (at least 2 bars are needed for a break). The trigger candle need
// not close [@ 08:19]; on closed bars the break is read from the candle's
// extremes, which is the same test once the candle has closed.
func TriggerTick(prev TriggerLine, bars5m []market.Kline, cfg Config) TriggerLine {
	if !cfg.Enabled || len(bars5m) < 2 {
		return prev
	}
	next := prev
	for i := 1; i < len(bars5m); i++ {
		cur, before := bars5m[i], bars5m[i-1]
		brokeHigh := cur.High > before.High
		brokeLow := cur.Low < before.Low
		switch {
		case brokeHigh && brokeLow:
			// broke BOTH: draw both lines, keep the one price is NOW beyond
			// [D3.4 p2 @ 18:49].
			if cur.Close > before.High {
				brokeLow = false
			} else {
				brokeHigh = false
			}
		}
		if next.Dir == "" {
			switch {
			case brokeHigh:
				next.Dir, next.Price = SideLong, before.High
			case brokeLow:
				next.Dir, next.Price = SideShort, before.Low
			}
			continue
		}
		if next.Moved {
			continue // "MỘT LẦN MỘT THÔI" — the line moves once [@ 11:48–13:33]
		}
		reversal := next.Dir == SideLong && brokeLow || next.Dir == SideShort && brokeHigh
		if !reversal {
			continue
		}
		next.OldDir, next.OldPrice = next.Dir, next.Price
		if brokeLow {
			next.Dir, next.Price = SideShort, before.Low
		} else {
			next.Dir, next.Price = SideLong, before.High
		}
		next.Moved = true
	}
	return next
}

// TriggerVerdict filters an entry by the trigger line: allowed side only,
// nothing between two opposing lines [@ 16:38].
func TriggerVerdict(t TriggerLine, price float64) (ok bool, side Side, reason string) {
	if t.Dir == "" {
		return true, "", ""
	}
	between := t.OldPrice != 0 &&
		(price > t.Price && price < t.OldPrice || price < t.Price && price > t.OldPrice)
	if between {
		return false, "", "price between two opposing trigger lines — no trade [D3.4 p1 @ 16:38]"
	}
	if t.Dir == SideLong && price < t.Price {
		return false, "", "below the buy trigger line — do nothing [D3.4 p1 @ 06:22]"
	}
	if t.Dir == SideShort && price > t.Price {
		return false, "", "above the sell trigger line — do nothing [D3.4 p1 @ 06:22]"
	}
	return true, t.Dir, ""
}

// ── 15m/5m ISB conflict [§5.3, D4.2 p1 @ 13:59; §12] ────────────────────────
// 2–3 adjacent 5m candles churning and going nowhere = a 15m inside bar
// [@ 00:00]. Same direction → trade; OPPOSITE directions → DO NOT TRADE AT ALL
// [@ 14:35].

// Is5mISB reports a 5m inside bar (the 1m body rule applied to 5m bars).
func Is5mISB(prev, cur market.Kline) bool {
	return IsISB(prev, cur)
}

// Is15mChurn is the 15m-ISB proxy (DS-108 §2.1): 3 consecutive 5m bars each
// inside the FIRST one's range.
func Is15mChurn(bars5m []market.Kline) bool {
	if len(bars5m) < 3 {
		return false
	}
	first := bars5m[len(bars5m)-3]
	last := bars5m[len(bars5m)-1]
	for _, b := range bars5m[len(bars5m)-2:] {
		if b.High > first.High || b.Low < first.Low {
			return false
		}
	}
	_ = last
	return true
}

// barDir is a candle's own direction (body up/down) — the "inside bar's
// direction" read in §5.3.
func barDir(b market.Kline) Side {
	if b.Close > b.Open {
		return SideLong
	}
	return SideShort
}

// ISBConflictVerdict: a live 5m ISB and a live 15m churn with OPPOSITE
// directions → no trade at all [D4.2 p1 @ 14:35]. The 5m ISB is the most
// recent pair; the 15m read is the last 3 5m bars.
func ISBConflictVerdict(bars5m []market.Kline) (conflict bool) {
	if len(bars5m) < 3 {
		return false
	}
	last, prev := bars5m[len(bars5m)-1], bars5m[len(bars5m)-2]
	if !Is5mISB(prev, last) || !Is15mChurn(bars5m) {
		return false
	}
	return barDir(last) != barDir(bars5m[len(bars5m)-3])
}

// ── mid-range [§12, D3.2 p2 @ 08:34; D3.4 p3 @ 01:44] ───────────────────────
// The middle of any range: between two levels → PHL/PLH banned, ISB only.

// MidRange reports price sitting between two levels, each within
// cfg.RangeGapPts of price (RangeGapPts 0 disables the filter).
func MidRange(levels []Level, price float64, cfg Config) bool {
	if cfg.RangeGapPts <= 0 {
		return false
	}
	above, below := false, false
	for _, l := range levels {
		d := l.Price - price
		switch {
		case d > 0 && d <= cfg.RangeGapPts:
			above = true
		case d < 0 && -d <= cfg.RangeGapPts:
			below = true
		}
	}
	return above && below
}

// SetupPermittedVerdict gates a setup kind by the mid-range filter: in the
// middle of a range only the ISB survives [D3.2 p2 @ 08:34].
func SetupPermittedVerdict(kind string, levels []Level, price float64, cfg Config) (ok bool, reason string) {
	if !MidRange(levels, price, cfg) {
		return true, ""
	}
	if kind == "ISB" {
		return true, ""
	}
	return false, "mid-range: PHL/PLH banned, inside bar only [D3.2 p2 @ 08:34]"
}
