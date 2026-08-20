package kernel

import (
	"strings"

	"nofx/market"
)

// P0.4 — SCENARIO-FACT EVALUATOR (the keystone).
//
// Pure, deterministic Go facts about a price LEVEL given recent bars. The
// planner proposes scenarios (judgment = AI); the executor decides ENTRIES by
// checking these FACTS (facts = Go) every cycle. The prompt tail carries them
// verbatim ("sweep=T/F, closes-beyond n, acceptance n/2"). NOTHING here calls an
// LLM — it is unit-tested on bar fixtures first, exactly as the spec demands.
//
// Bar convention (matches kernel/svp.go): bars are chronological (oldest→newest);
// a bar is CLOSED iff CloseTime < nowMs. OHLCV are tick-aligned float64.
// Direction convention: DirAbove (+1) = the side ABOVE the level; DirBelow (-1)
// = below. All comparisons are strict for "beyond", inclusive for "touched".

const (
	// DirAbove is the side above the level (bullish/breakout-up side).
	DirAbove = 1
	// DirBelow is the side below the level.
	DirBelow = -1
)

// LevelFacts is the full fact snapshot for one level, in the queried direction.
type LevelFacts struct {
	Level            float64 `json:"level"`
	LatestClose      float64 `json:"latest_close"`
	DistancePoints   float64 `json:"distance_points"` // latestClose - level (signed)
	ClosesBeyondUp   int     `json:"closes_beyond_up"`
	ClosesBeyondDown int     `json:"closes_beyond_down"`
	Swept            bool    `json:"swept"`     // in the queried dir
	Reclaimed        bool    `json:"reclaimed"` // in the queried dir
	Rejected         bool    `json:"rejected"`  // held as S/R in the queried dir
	Accepted         bool    `json:"accepted"`  // accepted through in dir per rule
	AcceptHave       int     `json:"accept_have"`
	AcceptNeed       int     `json:"accept_need"`
	StillValid       bool    `json:"still_valid"` // not (touched in-window AND accepted through)
}

// acceptanceNeed maps an acceptance rule to the number of consecutive closes
// beyond required. "15m-close" → 1; "2x5m"/"2×5m"/default → 2.
func acceptanceNeed(rule string) int {
	switch strings.ReplaceAll(strings.ToLower(strings.TrimSpace(rule)), "×", "x") {
	case "15m-close", "15m", "15mclose", "15m_close":
		return 1
	case "5m-close", "5m_close", "1x5m", "1x5m_close":
		return 1 // A2: one 5m close, as authored
	case "2x5m", "2x_5m", "2x5m_close", "":
		return 2
	default:
		return 2
	}
}

// acceptanceTFMinutes maps an acceptance rule to the BAR TIMEFRAME its N closes
// are meant to be counted in. "2x5m" → 5-minute bars; "15m-close" → 15.
//
// This exists because the rule names a timeframe and the counting function does
// not: ClosesBeyond counts *bars*, whatever length they happen to be. Feeding it
// the 1-minute SVP series silently turned "2 five-minute closes" (10 minutes of
// acceptance) into "2 one-minute closes" (2 minutes) — see PlanIsDeadSince.
func acceptanceTFMinutes(rule string) int {
	switch strings.ReplaceAll(strings.ToLower(strings.TrimSpace(rule)), "×", "x") {
	case "15m-close", "15m", "15mclose", "15m_close":
		return 15
	default:
		return 5 // "2x5m", "5m-close" and the default rule
	}
}

// AcceptanceIntervalMinutes returns the bar length (in minutes) the rule's N
// consecutive closes are meant to be counted on: 5 for "2x5m" (and the default),
// 15 for "15m-close". This is the single source of truth for the acceptance
// interval — no call site may pick its own bar length.
func AcceptanceIntervalMinutes(rule string) int {
	return acceptanceTFMinutes(rule)
}

// AcceptanceBars resolves the series acceptance facts must be counted on.
//
// The acceptance rule NAMES the timeframe — "2x5m" means two consecutive
// 5-MINUTE closes, "15m-close" means one 15-minute close — while the raw
// counters (ClosesBeyond / Acceptance / LevelStillValid) count BARS of whatever
// series they are handed. This is the single resolver: every acceptance /
// closes-beyond consumer must pass its bars through here (directly or via
// EvaluateLevelFacts / LevelStillValidOn) and never feed a raw series straight
// to the counters. Routing the 1-minute SVP cache through here is what keeps
// "two 5-minute closes" (10 minutes of acceptance) from being satisfied by two
// 1-minute closes — the executor prompt and the card used to announce acceptance
// ~5× early.
func AcceptanceBars(bars []market.Kline, rule string) []market.Kline {
	return aggregateToMinutes(bars, acceptanceTFMinutes(rule))
}

// aggregateToMinutes groups bars into fixed wall-clock buckets of tfMinutes and
// returns one OHLC bar per NON-EMPTY bucket.
//
// It never invents a bucket: a closed market produces no bar, exactly as the
// source series does. Buckets are keyed on absolute epoch minutes so the result
// is independent of where the input window happens to start, and CloseTime is the
// bucket's true end so the "closed bar" tests downstream stay honest — a
// half-formed final bucket is correctly ignored until it completes.
//
// tfMinutes <= 1, or a source series already coarser than the bucket, makes this
// a pass-through in effect (one bar per bucket).
func aggregateToMinutes(bars []market.Kline, tfMinutes int) []market.Kline {
	if tfMinutes <= 1 || len(bars) == 0 {
		return bars
	}
	span := int64(tfMinutes) * 60_000
	out := make([]market.Kline, 0, len(bars)/tfMinutes+1)
	var curBucket int64 = -1
	for i := range bars {
		b := bars[i]
		bucket := b.OpenTime / span
		if bucket != curBucket {
			out = append(out, market.Kline{
				OpenTime: bucket * span,
				// -1 matches the repo's bar convention (CloseTime is the last
				// instant INSIDE the bar): a bucket counts as closed the moment
				// now reaches its end, and a still-forming final bucket does not.
				CloseTime: bucket*span + span - 1,
				Open:      b.Open, High: b.High, Low: b.Low, Close: b.Close,
				Volume: b.Volume,
			})
			curBucket = bucket
			continue
		}
		agg := &out[len(out)-1]
		if b.High > agg.High {
			agg.High = b.High
		}
		if b.Low < agg.Low {
			agg.Low = b.Low
		}
		agg.Close = b.Close
		agg.Volume += b.Volume
	}
	return out
}

// SignedDistancePoints returns price - level (positive = price above level).
func SignedDistancePoints(price, level float64) float64 { return price - level }

// DistanceTicks returns the signed distance in ticks; 0 if tick <= 0 (unknown
// instrument — callers must guard, mirroring market.FuturesTickSize's 0 return).
func DistanceTicks(price, level, tick float64) float64 {
	if tick <= 0 {
		return 0
	}
	return (price - level) / tick
}

// latestClosedClose returns the most recent CLOSED bar's close.
func latestClosedClose(bars []market.Kline, nowMs int64) (float64, bool) {
	for i := len(bars) - 1; i >= 0; i-- {
		if bars[i].CloseTime < nowMs {
			return bars[i].Close, true
		}
	}
	return 0, false
}

// ClosesBeyond counts the most-recent CONSECUTIVE closed bars whose close is
// strictly beyond level in dir. The run breaks at the first closed bar not
// beyond. Trailing not-yet-closed bars are skipped. This is the "closes-beyond n"
// tail fact and the basis of acceptance.
func ClosesBeyond(bars []market.Kline, level float64, dir int, nowMs int64) int {
	count := 0
	for i := len(bars) - 1; i >= 0; i-- {
		b := bars[i]
		if b.CloseTime >= nowMs {
			continue // not closed yet (only ever the newest bar(s))
		}
		beyond := (dir >= 0 && b.Close > level) || (dir < 0 && b.Close < level)
		if !beyond {
			break
		}
		count++
	}
	return count
}

// Acceptance reports whether price has ACCEPTED through the level in dir under
// the rule (need consecutive closes beyond). Returns (accepted, have, need).
func Acceptance(bars []market.Kline, level float64, dir int, rule string, nowMs int64) (bool, int, int) {
	need := acceptanceNeed(rule)
	have := ClosesBeyond(bars, level, dir, nowMs)
	return have >= need, have, need
}

// Swept reports a liquidity sweep within the last `lookback` closed bars: a bar
// whose WICK pierced beyond level in sweepDir but which CLOSED back on the other
// side (a stop-run + rejection). sweepDir=+1 → high pierced above, close ≤ level;
// sweepDir=-1 → low pierced below, close ≥ level.
func Swept(bars []market.Kline, level float64, sweepDir, lookback int, nowMs int64) bool {
	if lookback <= 0 {
		lookback = 3
	}
	checked := 0
	for i := len(bars) - 1; i >= 0 && checked < lookback; i-- {
		b := bars[i]
		if b.CloseTime >= nowMs {
			continue
		}
		checked++
		if sweepDir >= 0 {
			if b.High > level && b.Close <= level {
				return true
			}
		} else {
			if b.Low < level && b.Close >= level {
				return true
			}
		}
	}
	return false
}

// Reclaimed reports whether price reclaimed the level in dir: the latest closed
// bar closes beyond in dir AND within `lookback` closed bars there was a close
// on the OPPOSITE side (price crossed back through and held).
func Reclaimed(bars []market.Kline, level float64, dir, lookback int, nowMs int64) bool {
	if lookback <= 0 {
		lookback = 3
	}
	latest, ok := latestClosedClose(bars, nowMs)
	if !ok {
		return false
	}
	if dir >= 0 && !(latest > level) {
		return false
	}
	if dir < 0 && !(latest < level) {
		return false
	}
	checked := 0
	for i := len(bars) - 1; i >= 0 && checked < lookback; i-- {
		b := bars[i]
		if b.CloseTime >= nowMs {
			continue
		}
		checked++
		if dir >= 0 && b.Close < level {
			return true
		}
		if dir < 0 && b.Close > level {
			return true
		}
	}
	return false
}

// Rejected reports whether the level HELD as support/resistance within the last
// `lookback` closed bars: a bar touched/pierced the level but the latest closed
// bar closed back on the dir side. dir=+1 → support held (a low reached ≤ level,
// now closing above); dir=-1 → resistance held (a high reached ≥ level, now
// closing below).
func Rejected(bars []market.Kline, level float64, dir, lookback int, nowMs int64) bool {
	if lookback <= 0 {
		lookback = 3
	}
	latest, ok := latestClosedClose(bars, nowMs)
	if !ok {
		return false
	}
	if dir >= 0 && !(latest > level) {
		return false
	}
	if dir < 0 && !(latest < level) {
		return false
	}
	checked := 0
	for i := len(bars) - 1; i >= 0 && checked < lookback; i-- {
		b := bars[i]
		if b.CloseTime >= nowMs {
			continue
		}
		checked++
		if dir >= 0 && b.Low <= level {
			return true
		}
		if dir < 0 && b.High >= level {
			return true
		}
	}
	return false
}

// LevelStillValid reports whether the level has NOT been accepted through on
// either side (per rule). Once `need` consecutive closes sit beyond on one side,
// the level flipped roles → consumed → no longer a fresh barrier.
func LevelStillValid(bars []market.Kline, level float64, rule string, nowMs int64) bool {
	need := acceptanceNeed(rule)
	return ClosesBeyond(bars, level, DirAbove, nowMs) < need &&
		ClosesBeyond(bars, level, DirBelow, nowMs) < need
}

// LevelStillValidOn reports validity after resolving the rule timeframe from the
// RAW series, so callers can hand it the 1-minute SVP cache unmodified.
func LevelStillValidOn(bars []market.Kline, level float64, rule string, nowMs int64) bool {
	return LevelStillValid(AcceptanceBars(bars, rule), level, rule, nowMs)
}

// EvaluateLevelFacts computes the full fact snapshot for a level in one pass —
// the block the executor prompt tail and the card consume per scenario. dir is
// the scenario's expected direction; rule is the acceptance rule; lookback bounds
// sweep/reclaim/reject scans (0 → default 3).
func EvaluateLevelFacts(bars []market.Kline, level float64, dir int, rule string, lookback int, nowMs int64) LevelFacts {
	lc, _ := latestClosedClose(bars, nowMs)
	// Acceptance, closes-beyond and validity are counted on the timeframe the
	// RULE names (2x5m → 5-minute bars; 15m-close → 15-minute bars), never on
	// the source series' own length. Sweep/reclaim/reject stay on the source
	// series: their lookback is a fixed 3-bar event window, not an acceptance
	// horizon, and resolving them to rule-length bars would silently widen it.
	judge := AcceptanceBars(bars, rule)
	accepted, have, need := Acceptance(judge, level, dir, rule, nowMs)
	// P1c — validity is touch-gated: a level that price merely SITS beyond
	// (support below price all session, resistance above) is not consumed.
	// Consumption = touched in-window AND accepted through on rule-TF closes;
	// a wick alone never consumes and sitting-beyond alone never consumes.
	// This keeps the card / PLAN STATUS block aligned with ConsumedSince and
	// PlanIsDeadSince, which already require the touch.
	touched := levelTouched(judge, level, nowMs)
	return LevelFacts{
		Level:            level,
		LatestClose:      lc,
		DistancePoints:   lc - level,
		ClosesBeyondUp:   ClosesBeyond(judge, level, DirAbove, nowMs),
		ClosesBeyondDown: ClosesBeyond(judge, level, DirBelow, nowMs),
		Swept:            Swept(bars, level, dir, lookback, nowMs),
		Reclaimed:        Reclaimed(bars, level, dir, lookback, nowMs),
		Rejected:         Rejected(bars, level, dir, lookback, nowMs),
		Accepted:         accepted,
		AcceptHave:       have,
		AcceptNeed:       need,
		StillValid:       LevelStillValid(judge, level, rule, nowMs) || !touched,
	}
}
