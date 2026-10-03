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

	// OldPrice/oldDir are the line from BEFORE the last reversal move; the
	// zone strictly between the two lines is the no-trade zone [@ 16:38].
	OldPrice float64
	OldDir   Side

	// B2: persistence — the last PROCESSED bucket (clock-aligned open) and
	// its bar. TriggerTick processes only newer buckets; the forming bucket
	// may fire a break once and is never re-applied.
	LastBucket int64
	LastBar    market.Kline
}

// TriggerTick advances the trigger state over the closed 5m buckets newer
// than LastBucket (B2). The trigger candle need not close [@ 08:19]; a bucket
// that exists (even forming) may fire a break, is committed once, and is never
// re-applied. Buckets are compared only with the immediately PREVIOUS bucket
// (a gap skips the test — law 2 compares adjacent candles).
func TriggerTick(prev TriggerLine, bars5m []market.Kline, cfg Config) TriggerLine {
	if !cfg.Enabled || len(bars5m) == 0 {
		return prev
	}
	next := prev
	ms := int64(5) * 60_000
	for _, b := range bars5m {
		if b.OpenTime <= next.LastBucket {
			continue
		}
		last := next.LastBar
		if last.OpenTime > 0 && b.OpenTime == last.OpenTime+ms {
			next = applyBreak(next, last, b)
		}
		// the first bucket ever, or a gap: record and move on
		next.LastBar = b
		next.LastBucket = b.OpenTime
	}
	return next
}

// applyBreak runs the §5.1 break logic for one bucket against the previous
// adjacent bucket (B3: every reversal moves the line ONCE — later breaks in
// the SAME direction never move it; the NEXT reversal moves it again).
func applyBreak(next TriggerLine, before, cur market.Kline) TriggerLine {
	brokeHigh := cur.High > before.High
	brokeLow := cur.Low < before.Low
	if brokeHigh && brokeLow {
		// broke BOTH: draw both lines, keep the one price is NOW beyond
		// [D3.4 p2 @ 18:49].
		if cur.Close > before.High {
			brokeLow = false
		} else {
			brokeHigh = false
		}
	}
	switch {
	case next.Dir == "": // the FIRST break draws the line [@ 19:48]
		if brokeHigh {
			next.Dir, next.Price = SideLong, before.High
		} else if brokeLow {
			next.Dir, next.Price = SideShort, before.Low
		}
	case next.Dir == SideLong && brokeLow: // reversal: move the line once
		next.OldDir, next.OldPrice = next.Dir, next.Price
		next.Dir, next.Price = SideShort, before.Low
	case next.Dir == SideShort && brokeHigh: // reversal: move the line once
		next.OldDir, next.OldPrice = next.Dir, next.Price
		next.Dir, next.Price = SideLong, before.High
	}
	// same-direction breaks (including repeats) never move the line
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
