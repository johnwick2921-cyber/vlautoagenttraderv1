package kernel

import (
	"math"
	"strings"

	"nofx/market"
)

// ── THE FOLLOW-PLAN (round 17) — RECORDED ONLY, NEVER ARMED ─────────────────
//
// "A real trader knows one level, two plans: hold it and the trend confirms,
// break it and the bias flips." Round 17 measured the second half and found
// nothing on MNQ: role reversal is UNTESTED on index futures and the naive
// follow LOSES (Mesfin: 80.7% pullback stop-out; the sweep loses both ways).
// So the follow side is built as a RECORD: for every level a scenario is
// authored on, beside the fade-plan, the follow-plan is computed and stored —
// its break, its would-be entry, its MAE/MFE — and nothing here reaches the
// wire (E9). The record decides whether it ever goes live: a cell (approach
// direction × level timeframe, ~385 episodes) where broken levels reverse
// role with a 95% lower bound above 50% AND the follow nets positive after
// 2-pt friction.
//
// This file is PURE. Bars, the clock and the scope arrive as arguments; the
// caller (trader/follow_plan_wiring.go) reads the store and writes the row.
// The BREAK is judged by the confirmation-truth evaluator — a CLOSED bucket
// beyond the level, never a forming one. A touch is not a fill.

// FollowPlanInput is one fade episode's level, as the recorder sees it.
type FollowPlanInput struct {
	Level      float64
	EntrySide  string // "above" | "below" — the side price came FROM at the fade episode's open
	OpenedAtMs int64  // the fade episode's open — the scan starts here
	NowMs      int64  // the clock (A28); a bucket closing after it is FORMING
	ScopeEndMs int64  // the session-day's end — past it, an unbroken level is final
	Tick       float64
	Bars       []market.Kline // 1m, ONE contract, ascending
	// BucketMinutes is the confirmation bucket (5). Horizons are in buckets.
	BucketMinutes int
	Horizons      []int
	FrictionPts   float64
}

// FollowPlan is the recorded plan. Every pointer is NULL until its event
// occurs; a never-broken level is a plan with BreakAtMs nil.
type FollowPlan struct {
	BreakAtMs       *int64
	BreakDir        string // "up" | "down"
	RoleReversed    bool   // set from the instant of the break (recorded, never on the map)
	RetestAtMs      *int64
	EntryPx         *float64
	EntryBasis      string
	Direction       int // +1 long / −1 short — the follow's side after reversal; 0 until broken
	MAE10, MFE10    *float64
	Net10           *float64
	MAE20, MFE20    *float64
	Net20           *float64
	BiasWouldFlipTo string // "long" | "short"
	State           string
}

// Follow-plan states. Open ones are re-scanned by the recorder; final ones
// are never touched again.
const (
	FollowStateOpen       = "open"       // no break yet, scope still running
	FollowStateBroken     = "broken"     // broken, no retest yet
	FollowStateRetested   = "retested"   // retested, horizons still filling
	FollowStateComplete   = "complete"   // every horizon filled (final)
	FollowStateNoBreak    = "no_break"   // scope ended unbroken (final) — a zero-trade outcome
	FollowStateNoRetest   = "no_retest"  // scope ended broken, never retested (final)
	FollowStateIncomplete = "incomplete" // scope ended before the horizons filled (final)
	FollowStateNoFill     = "no_fill"    // retested, touch not a fill (final)
)

// Would-be entry bases. The level price is the LIMIT's price, which is what a
// resting order would carry; the assumption is the fill rule, named.
const (
	FollowEntryPassiveLimitThrough = "follow:passive_limit_at_level:through_1_tick"
	FollowEntryTouchNotFill        = "follow:touch_not_fill"
)

// FollowIsFinal reports whether a state needs no further scanning.
func FollowIsFinal(state string) bool {
	switch state {
	case FollowStateComplete, FollowStateNoBreak, FollowStateNoRetest, FollowStateIncomplete, FollowStateNoFill:
		return true
	}
	return false
}

// ComputeFollowPlan runs the plan forward over the bars at NowMs.
func ComputeFollowPlan(in FollowPlanInput) FollowPlan {
	fp := FollowPlan{State: FollowStateOpen}
	if in.Level <= 0 || len(in.Bars) == 0 || in.BucketMinutes <= 0 {
		return fp
	}
	side := strings.ToLower(strings.TrimSpace(in.EntrySide))
	if side != "above" && side != "below" {
		return fp
	}
	// The fade side: price came from ABOVE → the level was SUPPORT, a break is
	// a close BELOW it; from BELOW → RESISTANCE, a break is a close ABOVE it.
	support := side == "above"
	beyond := func(close float64) bool {
		if support {
			return close < in.Level
		}
		return close > in.Level
	}

	// (1) BREAK — the first CLOSED bucket beyond the level, from the episode's
	// open. closedConfirmationBuckets drops a forming bucket by construction.
	closed, _ := closedConfirmationBuckets(in.Bars, in.OpenedAtMs, in.NowMs, in.BucketMinutes)
	breakCloseMs := int64(0)
	for _, b := range closed {
		if beyond(b.Close) {
			e := EvaluateBucketClose(b.OpenTime, in.BucketMinutes, in.NowMs)
			breakCloseMs = e.CloseMs
			break
		}
	}
	if breakCloseMs == 0 {
		if in.ScopeEndMs > 0 && in.NowMs >= in.ScopeEndMs {
			fp.State = FollowStateNoBreak
		}
		return fp
	}
	fp.BreakAtMs = &breakCloseMs
	fp.RoleReversed = true
	if support {
		fp.BreakDir, fp.BiasWouldFlipTo, fp.Direction = "down", "short", -1
	} else {
		fp.BreakDir, fp.BiasWouldFlipTo, fp.Direction = "up", "long", 1
	}
	fp.State = FollowStateBroken

	// (2) RETEST — the first 1m bar after the break whose range contains the
	// level and whose PREVIOUS close is on the far (new) side: price coming
	// back to the reversed level from where the break took it.
	tick := in.Tick
	if tick <= 0 {
		tick = 0.25
	}
	retestIdx := -1
	for i := 1; i < len(in.Bars); i++ {
		b := in.Bars[i]
		if b.OpenTime < breakCloseMs || b.OpenTime >= in.NowMs {
			continue
		}
		if !(b.Low <= in.Level && in.Level <= b.High) {
			continue
		}
		if !beyond(in.Bars[i-1].Close) {
			continue
		}
		retestIdx = i
		break
	}
	if retestIdx < 0 {
		if in.ScopeEndMs > 0 && in.NowMs >= in.ScopeEndMs {
			fp.State = FollowStateNoRetest
		}
		return fp
	}
	rb := in.Bars[retestIdx]
	rt := rb.OpenTime
	fp.RetestAtMs = &rt
	// The would-be entry: a passive limit AT the level in the follow direction.
	// It fills only if the bar traded THROUGH the level by a tick toward the
	// original side — a touch is not a fill.
	through := false
	if fp.Direction < 0 {
		through = rb.High >= in.Level+tick
	} else {
		through = rb.Low <= in.Level-tick
	}
	if !through {
		fp.EntryBasis = FollowEntryTouchNotFill
		fp.State = FollowStateNoFill
		return fp
	}
	entry := in.Level
	fp.EntryPx = &entry
	fp.EntryBasis = FollowEntryPassiveLimitThrough
	fp.State = FollowStateRetested

	// (3) OUTCOME — MAE/MFE over the next H CLOSED buckets after the retest bar.
	after := in.Bars[retestIdx+1:]
	buckets, _ := closedConfirmationBuckets(after, rb.OpenTime+1, in.NowMs, in.BucketMinutes)
	filled := 0
	for _, h := range in.Horizons {
		if len(buckets) < h {
			continue
		}
		mfe, mae := 0.0, 0.0
		for _, b := range buckets[:h] {
			var fav, adv float64
			if fp.Direction < 0 {
				fav, adv = entry-b.Low, b.High-entry
			} else {
				fav, adv = b.High-entry, entry-b.Low
			}
			mfe, mae = math.Max(mfe, fav), math.Max(mae, adv)
		}
		net := (buckets[h-1].Close-entry)*float64(fp.Direction) - in.FrictionPts
		m, a, n := mfe, mae, net
		switch h {
		case 10:
			fp.MFE10, fp.MAE10, fp.Net10 = &m, &a, &n
		case 20:
			fp.MFE20, fp.MAE20, fp.Net20 = &m, &a, &n
		}
		filled++
	}
	if filled == len(in.Horizons) {
		fp.State = FollowStateComplete
	} else if in.ScopeEndMs > 0 && in.NowMs >= in.ScopeEndMs {
		fp.State = FollowStateIncomplete
	}
	return fp
}
