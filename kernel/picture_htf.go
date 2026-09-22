package kernel

import (
	"sort"

	"nofx/market"
)

// W-PICTURE-HTF (2026-09-19) — the owner's two-picture method as deterministic
// detection over native NT8 bars. PURE functions over []market.Kline; the
// event-driven evaluator (trader layer) feeds them COMPLETED bars only (the
// C# AddOn marks final bars — see the wire work). Nothing here reads clocks,
// positions, or orders: every rule below is a pure verdict over the bars given.
//
// Rule defaults are EXPLICIT ENGINEERING DEFAULTS (owner-approved for SIM),
// not research-proven optimums: 120-candle 4H discovery window, 2+2 pivot
// confirmation, 24-candle 5m swing search, one-tick close rule, 10s entry
// window and 2s freshness limit.

// PictureHtfLevel is one 4H body pivot: support or resistance derived from
// COMPLETED candle bodies. Wicks are kept separately (they are excursions, not
// levels). A level is knowable only after both trailing candles close (no
// lookahead); a completed close through the far edge retires it.
type PictureHtfLevel struct {
	Role       string  // "support" | "resistance"
	SourceIdx  int     // index of the pivot candle in the discovery window
	SourceOpen int64   // pivot candle open time (ms)
	BodyTop    float64 // body top = max(open, close)
	BodyBottom float64 // body bottom = min(open, close)
	WickHigh   float64
	WickLow    float64
	KnowableAt int64 // open time of the candle two after the pivot (ms)
	Retired    bool
	RetiredAt  int64   // open time of the retiring candle, 0 when active
	Boundary   float64 // breakout/breakdown boundary: body top (res) / body bottom (sup)
	TargetEdge float64 // target zone near edge: body top (res) / body bottom (sup)
}

// PictureHtfSweep records the support-rejection shape from the owner's first
// picture: a completed 4H candle trades below a KNOWN support and closes back
// above it. Context only — never an entry prerequisite. Mirrored for resistance
// (resistanceReclaim).
type PictureHtfSweep struct {
	LevelIdx  int // index into the levels slice
	Role      string
	WickedTo  float64 // wick extreme reached
	Reclaimed bool
	BarOpen   int64 // the sweeping candle's open time
}

// BodyPivots4H discovers support/resistance pivots from completed 4H candle
// bodies over the latest `window` bars. STRICT comparisons: equal extremes do
// not form a pivot. A pivot at index i is usable only when i+2 exists (both
// trailing candles have closed). Retirements are applied to the SNAPSHOT:
// any later completed close through the far edge retires the level.
func BodyPivots4H(bars []market.Kline, window int) []PictureHtfLevel {
	if window <= 0 {
		window = 120
	}
	if len(bars) > window {
		bars = bars[len(bars)-window:]
	}
	var out []PictureHtfLevel
	for i := 1; i < len(bars)-1; i++ {
		bodyTop := maxOf(bars[i].Open, bars[i].Close)
		bodyBottom := minOf(bars[i].Open, bars[i].Close)
		if bars[i-1].Open == 0 || bars[i+1].Open == 0 {
			continue
		}
		res := true
		sup := true
		// All four confirming neighbors: i-2, i-1, i+1, i+2.
		for _, j := range []int{i - 2, i - 1, i + 1, i + 2} {
			if j < 0 || j >= len(bars) || bars[j].Open == 0 {
				continue
			}
			bt := maxOf(bars[j].Open, bars[j].Close)
			bb := minOf(bars[j].Open, bars[j].Close)
			if bt >= bodyTop {
				res = false
			}
			if bb <= bodyBottom {
				sup = false
			}
		}
		role := ""
		switch {
		case res && sup:
			continue // equal-body plateau: no pivot (ambiguous duplicate levels)
		case res:
			role = "resistance"
		case sup:
			role = "support"
		}
		if role == "" {
			continue
		}
		lvl := PictureHtfLevel{
			Role:       role,
			SourceIdx:  i,
			SourceOpen: bars[i].OpenTime,
			BodyTop:    bodyTop,
			BodyBottom: bodyBottom,
			WickHigh:   bars[i].High,
			WickLow:    bars[i].Low,
		}
		if i+2 < len(bars) {
			lvl.KnowableAt = bars[i+2].OpenTime
		}
		switch role {
		case "resistance":
			lvl.Boundary = bodyTop
			lvl.TargetEdge = bodyTop
		case "support":
			lvl.Boundary = bodyBottom
			lvl.TargetEdge = bodyBottom
		}
		out = append(out, lvl)
	}
	// Retire levels whose far edge was crossed by a LATER completed close.
	for li := range out {
		l := &out[li]
		for j := l.SourceIdx + 2; j < len(bars); j++ {
			if bars[j].Open == 0 {
				continue
			}
			if l.Role == "resistance" && bars[j].Close > l.BodyTop+0.0 {
				l.Retired = true
				l.RetiredAt = bars[j].OpenTime
			}
			if l.Role == "support" && bars[j].Close < l.BodyBottom-0.0 {
				l.Retired = true
				l.RetiredAt = bars[j].OpenTime
			}
		}
	}
	return out
}

// ActiveLevels returns the non-retired levels knowable at or before nowMs.
func ActiveLevels(levels []PictureHtfLevel, nowMs int64) []PictureHtfLevel {
	var out []PictureHtfLevel
	for _, l := range levels {
		if !l.Retired && (l.KnowableAt == 0 || l.KnowableAt <= nowMs) {
			out = append(out, l)
		}
	}
	return out
}

// SweepReclaim4H recognizes the support-rejection shape over COMPLETED 4H bars:
// a candle trades below a known (non-retired, knowable) support boundary and
// closes back above it. Mirrored for resistance. Returns one record per
// (level, candle) event, ascending by bar time.
func SweepReclaim4H(levels []PictureHtfLevel, bars []market.Kline) []PictureHtfSweep {
	var out []PictureHtfSweep
	for li, l := range levels {
		if l.Retired {
			continue
		}
		for _, b := range bars {
			if b.OpenTime <= l.SourceOpen {
				continue
			}
			switch l.Role {
			case "support":
				if b.Low < l.Boundary && b.Close > l.Boundary {
					out = append(out, PictureHtfSweep{LevelIdx: li, Role: "support", WickedTo: b.Low, Reclaimed: true, BarOpen: b.OpenTime})
				}
			case "resistance":
				if b.High > l.Boundary && b.Close < l.Boundary {
					out = append(out, PictureHtfSweep{LevelIdx: li, Role: "resistance", WickedTo: b.High, Reclaimed: true, BarOpen: b.OpenTime})
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].BarOpen < out[j].BarOpen })
	return out
}

// H1CloseBreakResult is the deterministic H1 breakout verdict. Only the CLOSE
// must cross by at least one tick; the body need not sit entirely beyond.
type H1CloseBreakResult struct {
	Fired     bool
	Direction string // "long" | "short"
	LevelIdx  int    // the chosen extreme level among the crossed ones
	Crossed   []int  // all crossed eligible levels, ascending by extremity
	Boundary  float64
	PrevClose float64
	NewClose  float64
}

// H1CloseBreak evaluates the H1 confirmation over completed H1 candles:
// the level was knowable before the confirming candle opened, the previous
// completed close was at/below (long) or at/above (short) the boundary, and
// the new completed close is at least one tick beyond it. With several levels
// crossed, the highest resistance (long) / lowest support (short) wins; the
// others ride along as context.
func H1CloseBreak(levels []PictureHtfLevel, prevH1, newH1 market.Kline, tickSize float64) H1CloseBreakResult {
	res := H1CloseBreakResult{PrevClose: prevH1.Close, NewClose: newH1.Close}
	if tickSize <= 0 {
		tickSize = 0.25
	}
	if prevH1.OpenTime == 0 || newH1.OpenTime == 0 {
		return res
	}
	// Long: cross ABOVE a resistance. Short: cross BELOW a support.
	var longCrossed, shortCrossed []int
	for li, l := range levels {
		if l.Retired || (l.KnowableAt != 0 && l.KnowableAt > newH1.OpenTime) {
			continue
		}
		switch l.Role {
		case "resistance":
			if prevH1.Close <= l.Boundary && newH1.Close >= l.Boundary+tickSize {
				longCrossed = append(longCrossed, li)
			}
		case "support":
			if prevH1.Close >= l.Boundary && newH1.Close <= l.Boundary-tickSize {
				shortCrossed = append(shortCrossed, li)
			}
		}
	}
	if len(longCrossed) > 0 {
		res.Fired = true
		res.Direction = "long"
		res.Crossed = longCrossed
		sort.Slice(res.Crossed, func(a, b int) bool { return levels[res.Crossed[a]].Boundary > levels[res.Crossed[b]].Boundary })
		res.LevelIdx = res.Crossed[0]
		res.Boundary = levels[res.LevelIdx].Boundary
		return res
	}
	if len(shortCrossed) > 0 {
		res.Fired = true
		res.Direction = "short"
		res.Crossed = shortCrossed
		sort.Slice(res.Crossed, func(a, b int) bool { return levels[res.Crossed[a]].Boundary < levels[res.Crossed[b]].Boundary })
		res.LevelIdx = res.Crossed[0]
		res.Boundary = levels[res.LevelIdx].Boundary
		return res
	}
	return res
}

// StructuralSwing5M finds the stop anchor per the plan: the most recent 5m
// swing confirmed by two completed candles on EACH side (STRICT comparisons —
// equal extremes do not form a swing), searched over the preceding `lookback`
// completed candles, using wick extremes. Only swings completed (confirming
// candles closed) by confirmDeadline are eligible. direction selects the swing
// side: long → swing LOW below; short → swing HIGH above.
func StructuralSwing5M(bars []market.Kline, direction string, lookback int, confirmDeadline int64) (price float64, ok bool) {
	if lookback <= 0 {
		lookback = 24
	}
	if len(bars) > lookback+2 {
		bars = bars[len(bars)-(lookback+2):]
	}
	for i := 2; i < len(bars)-2; i++ {
		if bars[i].Open == 0 || bars[i+2].Open == 0 || bars[i+2].OpenTime > confirmDeadline {
			continue
		}
		if direction == "long" {
			if bars[i].Low < bars[i-1].Low && bars[i].Low < bars[i-2].Low &&
				bars[i].Low < bars[i+1].Low && bars[i].Low < bars[i+2].Low {
				price = bars[i].Low
				ok = true
			}
		} else {
			if bars[i].High > bars[i-1].High && bars[i].High > bars[i-2].High &&
				bars[i].High > bars[i+1].High && bars[i].High > bars[i+2].High {
				price = bars[i].High
				ok = true
			}
		}
	}
	return price, ok
}

// NearestOpposingZone selects the target: for a long, the nearest ACTIVE
// resistance zone whose near edge is above the entry; for a short, the nearest
// ACTIVE support zone below. Returns (edge, ok). Retired or not-yet-knowable
// zones are ineligible — and a nearer eligible zone is NEVER skipped.
func NearestOpposingZone(levels []PictureHtfLevel, entry float64, direction string, nowMs int64) (float64, bool) {
	best := 0.0
	found := false
	better := func(cand float64) bool {
		if direction == "long" {
			return cand > entry && (!found || cand < best)
		}
		return cand < entry && (!found || cand > best)
	}
	for _, l := range levels {
		if l.Retired || (l.KnowableAt != 0 && l.KnowableAt > nowMs) {
			continue
		}
		if direction == "long" && l.Role != "resistance" {
			continue
		}
		if direction == "short" && l.Role != "support" {
			continue
		}
		if better(l.TargetEdge) {
			best = l.TargetEdge
			found = true
		}
	}
	return best, found
}

// MomentumStall is the advisory weakening-close observation (plan §G): after
// two consecutive rising H1 close-to-close moves, a non-rising close is a
// long-momentum stall; mirrored for shorts. Advisory ONLY — it never trades.
type MomentumStall struct {
	Fired     bool
	Direction string // "long" (rises stalled) | "short" (falls stalled)
	Prev2     float64
	Prev1     float64
	Current   float64
	Change    float64 // current - prev1
}

// H1MomentumStall computes the stall verdict from the last three completed H1
// closes (oldest first).
func H1MomentumStall(closes []float64) MomentumStall {
	if len(closes) < 3 {
		return MomentumStall{}
	}
	c0, c1, c2 := closes[len(closes)-3], closes[len(closes)-2], closes[len(closes)-1]
	out := MomentumStall{Prev2: c0, Prev1: c1, Current: c2, Change: c2 - c1}
	if c1 > c0 && c2 > c1 {
		// rising streak intact — no stall yet
		return out
	}
	if c1 < c0 && c2 < c1 {
		return out
	}
	if c1 > c0 {
		out.Fired = true
		out.Direction = "long"
	}
	if c1 < c0 {
		out.Fired = true
		out.Direction = "short"
	}
	return out
}

func maxOf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minOf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
