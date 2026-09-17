package kernel

import (
	"strings"
	"time"

	"nofx/market"
)

// S2 — TIMEFRAME-AWARE FRESHNESS (2026-09-16, structure-first planner wave).
//
// Today (W7/W11b) a level's freshness is the persisted 1m-touch ladder
// (A→B→C→done, store.LevelState). A 4h zone is "tested" when a single 1m bar
// trades into its band, which is why fresh HTF zones collapse to grade C after
// one touch and the 12-seat table stops seating them (CTO WHY block, 2026-09-16
// 16:31 CT: 244 HTF levels detected, 3 seated).
//
// This file grades the freshness of an HTF level on ITS OWN timeframe bars:
// a bar of the level's TF that traded into [Lo,Hi] after the level's origin
// date is one test. Grade by test count: 0 fresh · 1 tested-1 · 2 tested-2 ·
// >=3 stale. The knob is `day_plan.levels_fresh_by_tf` (default OFF) — with it
// off this file is never on the grading path and every score/prompt stays
// byte-identical (proved by the parity + golden tests).

// htfFreshTFSet is the set of timeframes graded on their own bars when the S2
// knob is ON. Outside this set a level keeps today's 1m-touch grading.
var htfFreshTFSet = map[string]bool{
	"1h": true, "2h": true, "4h": true, "6h": true, "8h": true, "12h": true,
	"1d": true, "3d": true, "1w": true,
}

// IsHTFFreshTF reports whether tf is one of the S2 own-timeframe grading TFs.
func IsHTFFreshTF(tf string) bool {
	return htfFreshTFSet[strings.ToLower(strings.TrimSpace(tf))]
}

// LevelFreshnessByTF grades an HTF level's freshness on its own timeframe
// bars. Caller routes: only levels with DetectedLevel.HTF==true and a TF in
// htfFreshTFSet may enter. Bars MUST be that same timeframe.
//
// RE-ENTRY semantics (S2 F1, CTO review 2026-09-17): a "test" is a RE-ENTRY,
// not a touch. The level's formation bars are its birth — they close inside the
// band and must never count. Counting starts only after the first own-TF bar
// that CLOSES fully outside [Lo,Hi] after origin (price has left the zone);
// each subsequent bar that trades back into the band while the previous bar was
// outside counts ONE test — consecutive in-band bars are one visit, not N.
//
// Origin: FormedAtMs first, OriginDate (midnight) only as fallback. Unknown
// origin → 0 tests (fresh) and originUsed="none"; originUsed is reported so the
// replay table states which source each level used.
//
// Returns the S2 display grade ("fresh" | "tested-1" | "tested-2" | "stale"),
// the test count, and which origin source was used ("formed_at" | "origin_date"
// | "none"). The display grade is what Research.Freshness carries; scoring maps
// it onto the unchanged freshMult/zoneFreshMult ladders via normalizeByTFGrade
// (levels_score.go).
func LevelFreshnessByTF(l DetectedLevel, now time.Time, bars []market.Kline) (string, int, string) {
	origin, source, ok := levelOriginTimeSource(l)
	if !ok || origin.IsZero() {
		return "fresh", 0, source
	}
	lo, hi := l.Lo, l.Hi
	if hi < lo {
		lo, hi = hi, lo
	}
	tests := 0
	left := false       // the first bar that closed fully outside: price has left
	prevOutside := true // formation bars are treated as one continuous inside visit
	for _, b := range bars {
		if b.OpenTime < origin.UnixMilli() {
			continue // before the level existed — cannot test it
		}
		if b.OpenTime > now.UnixMilli() {
			continue // future bar — not evidence
		}
		inBand := b.Low <= hi && b.High >= lo
		if !left {
			// Formation: count nothing until a bar CLOSES fully outside the band.
			if b.Close < lo || b.Close > hi {
				left = true
				prevOutside = true
			}
			continue
		}
		if inBand && prevOutside {
			tests++ // one re-entry visit
			prevOutside = false
		} else if !inBand {
			prevOutside = true
		}
	}
	switch tests {
	case 0:
		return "fresh", 0, source
	case 1:
		return "tested-1", 1, source
	case 2:
		return "tested-2", 2, source
	default:
		return "stale", tests, source
	}
}

// levelOriginTimeSource resolves a level's origin instant and names the source:
// FormedAtMs preferred, OriginDate (midnight) as fallback, "none" when both are
// absent.
func levelOriginTimeSource(l DetectedLevel) (time.Time, string, bool) {
	if l.FormedAtMs > 0 {
		return time.UnixMilli(l.FormedAtMs), "formed_at", true
	}
	if d := strings.TrimSpace(l.OriginDate); d != "" {
		if t, err := time.Parse("2006-01-02", d); err == nil {
			return t, "origin_date", true
		}
	}
	return time.Time{}, "none", false
}

// normalizeByTFGrade maps the S2 display vocabulary onto the canonical ladder
// strings the UNCHANGED freshMult/zoneFreshMult tables already understand
// (tested-1 → b · tested-2 → c · stale → done). Every pre-existing freshness
// string passes through unchanged, so with the knob OFF the multipliers are
// byte-identical to today.
func normalizeByTFGrade(f string) string {
	switch strings.ToLower(strings.TrimSpace(f)) {
	case "tested-1":
		return "b"
	case "tested-2":
		return "c"
	case "stale":
		return "done"
	default:
		return f
	}
}
