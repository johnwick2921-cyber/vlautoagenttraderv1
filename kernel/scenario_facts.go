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
	StillValid       bool    `json:"still_valid"` // not yet accepted through either side
}

// acceptanceNeed maps an acceptance rule to the number of consecutive closes
// beyond required. "15m-close" → 1; "2x5m"/"2×5m"/default → 2.
func acceptanceNeed(rule string) int {
	switch strings.ReplaceAll(strings.ToLower(strings.TrimSpace(rule)), "×", "x") {
	case "15m-close", "15m", "15mclose", "15m_close":
		return 1
	case "2x5m", "2x_5m", "":
		return 2
	default:
		return 2
	}
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

// EvaluateLevelFacts computes the full fact snapshot for a level in one pass —
// the block the executor prompt tail consumes per scenario. dir is the scenario's
// expected direction; rule is the acceptance rule; lookback bounds sweep/reclaim/
// reject scans (0 → default 3).
func EvaluateLevelFacts(bars []market.Kline, level float64, dir int, rule string, lookback int, nowMs int64) LevelFacts {
	lc, _ := latestClosedClose(bars, nowMs)
	accepted, have, need := Acceptance(bars, level, dir, rule, nowMs)
	return LevelFacts{
		Level:            level,
		LatestClose:      lc,
		DistancePoints:   lc - level,
		ClosesBeyondUp:   ClosesBeyond(bars, level, DirAbove, nowMs),
		ClosesBeyondDown: ClosesBeyond(bars, level, DirBelow, nowMs),
		Swept:            Swept(bars, level, dir, lookback, nowMs),
		Reclaimed:        Reclaimed(bars, level, dir, lookback, nowMs),
		Rejected:         Rejected(bars, level, dir, lookback, nowMs),
		Accepted:         accepted,
		AcceptHave:       have,
		AcceptNeed:       need,
		StillValid:       LevelStillValid(bars, level, rule, nowMs),
	}
}
