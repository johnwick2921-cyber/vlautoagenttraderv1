package mentor

import (
	"time"

	"vl/kernel"
	"vl/market"
)

// §4.1 — FTGH / FTGL boxes [D3.2 p1 @ 01:21–04:25; D3.2 p2 @ 03:45–09:14].
//
// Construction, verbatim rules implemented here:
//   - "A ZONE made of 2 OR MORE tops (or bottoms). Two is enough."
//     [D3.2 p1 @ 03:21]
//   - "FTGH: from the HIGHEST WICK of top 1 down to the NEAREST BODY of
//     top 2. FTGL: from the LOWEST WICK of bottom 1 up to the NEAREST BODY
//     of bottom 2." [@ 03:21–04:25]
//   - "Always EXTEND RIGHT. The box is reused indefinitely." [D3.2 p2
//     @ 06:21]
//   - "Tops need not be exactly level — 'mình DU DI'." [D4.1 p1 @ 23:26]
//     → BoxCfg.TolPts.
//   - "Draw it on the timeframe you TRADE — 1-minute trader on the
//     1-minute." [D4.1 p1 @ 22:50] → boxes form from the trade TF (the
//     swing slice is 5m/15m; BoxesBuild keeps the 5m swings).
//   - "BOX — vẽ ra được rồi là KHÔNG CẦN TEST GÌ NỮA HẾT": a box is valid
//     by construction, no 3rd-touch TEST is needed for the box itself
//     [D3.2 p2 @ 03:45]. The 3rd-touch rule belongs to the TRADE: "The
//     trade: price returns → stop order away from the box, with the
//     REJECTING candle as the reference. Taken on the THIRD touch"
//     [@ 04:29; D3.4 p3 @ 07:02] — exposed here as Box.Touches so the
//     entry layer can require the third.
//   - "NEVER trade inside the box — 'hoàn toàn không'" [D3.2 p1 @ 06:59]
//     → InsideAnyBox.
//   - "ESCAPE = the candle's BODY outside (whole candle better)" [@ 19:05]
//     and "Delete it once price escapes it" [D3.4 p2 @ 12:11] → Dead.
//   - "A drawn box is never redrawn; it stays to the end of the day"
//     [D4.1 p2 @ 02:39–03:31] → BoxesBuild is a pure scan of the SAME
//     history: identical bars produce the identical box set every tick
//     (no redraw), and boxes are scoped to the current trading day.
//   - "Draw TWO zones, never a third." [D3.2 p2 @ 09:14] → BoxCfg.MaxBoxes.

// BoxKind is which failure the zone names.
type BoxKind string

const (
	// FTGH — failure to go higher ("nó không thể nào lên được nữa"
	// [D3.2 p1 @ 02:45]).
	FTGH BoxKind = "ftgh"
	// FTGL — failure to go lower.
	FTGL BoxKind = "ftgl"
)

// Level kinds for DS-103's location gate (box edges are one of the four
// setup locations).
const (
	KindFTGHEdge LevelKind = "ftgh_edge"
	KindFTGLEdge LevelKind = "ftgl_edge"
)

// Box is one drawn FTGH/FTGL zone. Never redrawn: once built, the edges do
// not move. Escaped boxes are deleted (dropped from the set).
type Box struct {
	Kind   BoxKind
	Top    float64 // FTGH: highest wick of top 1 · FTGL: nearest body of bottom 2
	Bottom float64 // FTGH: nearest body of top 2 · FTGL: lowest wick of bottom 1
	Key    string  // stable id "ftgh:<top>:<bottom>"
	// Touches counts closed candles whose wick reached an edge since the
	// box formed — the entry layer requires the THIRD touch [@ 04:29].
	Touches int
}

// BoxCfg holds the §4.1 knobs.
type BoxCfg struct {
	// TolPts is the "mình DU DI" tolerance: two tops (bottoms) this far
	// apart or closer still form a zone [D4.1 p1 @ 23:26]. The method gives
	// no number; default 15 [C].
	TolPts float64
	// MaxBoxes — "Draw TWO zones, never a third" [D3.2 p2 @ 09:14].
	MaxBoxes int
	// TouchBandPts: a wick within this of an edge counts as a touch;
	// default 0.25 (the bot's swing-price match tolerance).
	TouchBandPts float64
}

// DefaultBoxCfg returns the §4.1 defaults.
func DefaultBoxCfg() BoxCfg {
	return BoxCfg{TolPts: 15, MaxBoxes: 2, TouchBandPts: 0.25}
}

// swingPairAt is one detected swing point with its defining bar index.
type swingPairAt struct {
	kind  kernel.LevelKind
	price float64
	idx   int
}

// BoxesBuild is the pure, deterministic box scan: swing highs/lows on the
// trade TF (reused kernel.SwingPointLevels, TF-filtered to 5m) pair up into
// zones in time order; escaped boxes are dropped; only boxes of the current
// trading day survive. The same history always rebuilds the same set — a
// drawn box is never redrawn.
func BoxesBuild(bars []market.Kline, cfg BoxCfg, now time.Time) []Box {
	if cfg.MaxBoxes <= 0 || cfg.TolPts <= 0 {
		cfg = DefaultBoxCfg()
	}
	swings := kernel.SwingPointLevels(bars, now)
	var seq []swingPairAt
	for _, d := range swings {
		if d.TF != "5m" { // boxes draw on the trade TF; the swing slice is 5m/15m
			continue
		}
		if d.Kind != kernel.KindSWGH && d.Kind != kernel.KindSWGL {
			continue
		}
		idx := barIndexByPrice(bars, d.Price)
		if idx < 0 {
			continue
		}
		seq = append(seq, swingPairAt{kind: d.Kind, price: d.Price, idx: idx})
	}
	today := tradingDayKey(now.In(ctime()))
	var out []Box
	for i := 0; i+1 < len(seq) && len(out) < cfg.MaxBoxes; i++ {
		s1, s2 := seq[i], seq[i+1]
		if s1.kind != s2.kind {
			continue
		}
		if abs(s1.price-s2.price) > cfg.TolPts {
			continue
		}
		b := boxFromPair(bars, s1, s2)
		if b == nil {
			continue
		}
		if tradingDayKey(time.UnixMilli(bars[s1.idx].OpenTime).In(ctime())) != today {
			continue // the box does not outlive its day
		}
		b.Touches = countBoxTouches(bars, *b, s1.idx, cfg)
		if escaped(bars, *b, s1.idx) {
			continue // price escaped → deleted [D3.4 p2 @ 12:11]
		}
		b.Key = "ftgh:" + fnum(b.Top) + ":" + fnum(b.Bottom)
		if b.Kind == FTGL {
			b.Key = "ftgl:" + fnum(b.Top) + ":" + fnum(b.Bottom)
		}
		out = append(out, *b)
	}
	return out
}

// boxFromPair builds the zone from two adjacent same-kind swings:
// FTGH: [nearest body of top 2, highest wick of top 1];
// FTGL: [lowest wick of bottom 1, nearest body of bottom 2].
// nil when the zone is degenerate (body outside the wick edge).
func boxFromPair(bars []market.Kline, s1, s2 swingPairAt) *Box {
	b1, b2 := bars[s1.idx], bars[s2.idx]
	if s1.kind == kernel.KindSWGH {
		top := b1.High // highest wick of top 1
		body := b2.Open
		if b2.Close > body {
			body = b2.Close // nearest body of top 2 (the higher edge)
		}
		if top <= body {
			return nil
		}
		return &Box{Kind: FTGH, Top: top, Bottom: body}
	}
	bottom := b1.Low // lowest wick of bottom 1
	body := b2.Open
	if b2.Close < body {
		body = b2.Close // nearest body of bottom 2 (the lower edge)
	}
	if body <= bottom {
		return nil
	}
	return &Box{Kind: FTGL, Top: body, Bottom: bottom}
}

// barIndexByPrice finds the last bar whose high/low equals the swing price
// within the touch tolerance (the swing's defining bar).
func barIndexByPrice(bars []market.Kline, price float64) int {
	for i := len(bars) - 1; i >= 0; i-- {
		if abs(bars[i].High-price) < 0.26 || abs(bars[i].Low-price) < 0.26 {
			return i
		}
	}
	return -1
}

// escaped reports whether a closed candle AFTER formation had its whole
// body outside the zone — "ESCAPE = the candle's BODY outside (whole candle
// better)" [D3.2 p1 @ 19:05].
func escaped(bars []market.Kline, b Box, formedAt int) bool {
	for _, c := range bars[formedAt+1:] {
		if c.CloseTime == 0 {
			continue
		}
		switch b.Kind {
		case FTGH:
			if c.Open > b.Top && c.Close > b.Top {
				return true
			}
		case FTGL:
			if c.Open < b.Bottom && c.Close < b.Bottom {
				return true
			}
		}
	}
	return false
}

// countBoxTouches counts closed candles after formation whose wick reached
// an edge — the third touch is the trade [D3.2 p1 @ 04:29].
func countBoxTouches(bars []market.Kline, b Box, formedAt int, cfg BoxCfg) int {
	n := 0
	for _, c := range bars[formedAt+1:] {
		if c.CloseTime == 0 {
			continue
		}
		if touchesEdge(b, c, cfg) {
			n++
		}
	}
	return n
}

// touchesEdge reports a wick touch of either edge within TouchBandPts.
func touchesEdge(b Box, c market.Kline, cfg BoxCfg) bool {
	switch b.Kind {
	case FTGH:
		if abs(c.High-b.Top) <= cfg.TouchBandPts {
			return true
		}
		// a candle straddling the zone from below also touches the lower edge
		return c.High >= b.Bottom && c.Low <= b.Bottom+cfg.TouchBandPts
	case FTGL:
		if abs(c.Low-b.Bottom) <= cfg.TouchBandPts {
			return true
		}
		return c.Low <= b.Top && c.High >= b.Top-cfg.TouchBandPts
	}
	return false
}

// BoxEdgeLocations exports each live box's edges as locations for DS-103's
// location gate (the box edge is one of the four setup locations).
func BoxEdgeLocations(boxes []Box) []Level {
	var out []Level
	for _, b := range boxes {
		out = append(out,
			Level{Key: b.Key + ":top", Kind: KindFTGHEdge, Price: b.Top},
			Level{Key: b.Key + ":bottom", Kind: KindFTGLEdge, Price: b.Bottom},
		)
	}
	return out
}

// InsideAnyBox is the §12 ban: "NEVER trade inside the box — 'hoàn toàn
// không' — neither the candle nor your entry point" [D3.2 p1 @ 06:59].
func InsideAnyBox(boxes []Box, price float64) bool {
	for _, b := range boxes {
		if price > b.Bottom && price < b.Top {
			return true
		}
	}
	return false
}
