package kernel

import (
	"testing"
	"time"

	"nofx/market"
)

// ── THE FOLLOW-PLAN (round 17) — RECORDED ONLY. ONE clock per test (E12). ────
//
// Fixture: a SUPPORT at 100 (the fade episode came from ABOVE), session NY.
// Minute bars from t0; each helper appends one minute.

type fpTape struct {
	t0   time.Time
	bars []market.Kline
}

func (tp *fpTape) add(o, h, l, c float64) {
	s := tp.t0.Add(time.Duration(len(tp.bars)) * time.Minute)
	tp.bars = append(tp.bars, market.Kline{OpenTime: s.UnixMilli(), CloseTime: s.Add(time.Minute).UnixMilli() - 1, Open: o, High: h, Low: l, Close: c, Volume: 10})
}

func (tp *fpTape) flat(n int, px float64) {
	for i := 0; i < n; i++ {
		tp.add(px, px+0.5, px-0.5, px)
	}
}

func fpNow() time.Time { return time.Date(2026, 9, 11, 9, 0, 0, 0, CTLocation()) } // aligned to a 5m bucket

func fpInput(tp *fpTape, nowMs int64) FollowPlanInput {
	return FollowPlanInput{Level: 100, EntrySide: "above", OpenedAtMs: tp.t0.UnixMilli(), NowMs: nowMs,
		ScopeEndMs: tp.t0.Add(24 * time.Hour).UnixMilli(), Tick: 0.25, Bars: tp.bars, BucketMinutes: 5, Horizons: []int{10, 20}, FrictionPts: 2}
}

// E8 — break (closed bucket beyond) → role reversal → retest from the far side
// → would-be entry with its fill assumption → MAE/MFE at 10 and 20 buckets →
// bias_would_flip_to; the plan's live bias is not an input at all.
func TestFollowPlanE8BreakRetestExcursion(t *testing.T) {
	tp := &fpTape{t0: fpNow()}
	tp.flat(5, 101)              // bucket 0: sitting on the support
	tp.flat(5, 98)               // bucket 1: CLOSED below 100 → the break, direction down
	tp.flat(3, 97)               // drifting away below
	tp.add(98, 100.25, 97.5, 99) // the RETEST from below: range contains 100, trades THROUGH by a tick
	// 20 five-minute buckets after the retest bar: first leg goes to 96 (favourable for the short), then to 103 (adverse)
	tp.flat(50, 96)
	tp.flat(50, 103)
	now := tp.t0.Add(time.Duration(len(tp.bars)) * time.Minute).UnixMilli()
	fp := ComputeFollowPlan(fpInput(tp, now))

	if fp.BreakAtMs == nil || fp.BreakDir != "down" {
		t.Fatalf("break must be the first CLOSED bucket below the support: %+v", fp)
	}
	if want := tp.t0.Add(10 * time.Minute).UnixMilli(); *fp.BreakAtMs != want { // bucket 1's CLOSE
		t.Fatalf("break_at must be the closing instant of the breaking bucket: got %d want %d", *fp.BreakAtMs, want)
	}
	if fp.BiasWouldFlipTo != "short" || fp.Direction != -1 {
		t.Fatalf("a break below support implies short; role reversed → the follow is a SHORT at the level: %+v", fp)
	}
	if fp.RetestAtMs == nil || *fp.RetestAtMs != tp.t0.Add(13*time.Minute).UnixMilli() {
		t.Fatalf("retest must be the first touch from the far side after the break: %+v", fp)
	}
	if fp.EntryPx == nil || *fp.EntryPx != 100 || fp.EntryBasis != FollowEntryPassiveLimitThrough {
		t.Fatalf("would-be entry is the passive limit AT the level, filled because the bar traded through by a tick: %+v", fp)
	}
	if fp.MFE10 == nil || fp.MAE10 == nil || fp.Net10 == nil || fp.MFE20 == nil || fp.MAE20 == nil || fp.Net20 == nil {
		t.Fatalf("both horizons have enough closed buckets: %+v", fp)
	}
	// short from 100: first 10 buckets at 96 → MFE 4.5 (low 95.5), MAE 0 (the high 96.5 is still favourable); net at bucket 10 = (100-96)-2 = 2
	if *fp.MFE10 != 4.5 || *fp.MAE10 != 0 || *fp.Net10 != 2 {
		t.Fatalf("10-bucket excursion wrong: mfe=%v mae=%v net=%v", *fp.MFE10, *fp.MAE10, *fp.Net10)
	}
	// 20 buckets: adverse leg to 103.5 → MAE 3.5; net at bucket 20 = (100-103)-2 = -5
	if *fp.MFE20 != 4.5 || *fp.MAE20 != 3.5 || *fp.Net20 != -5 {
		t.Fatalf("20-bucket excursion wrong: mfe=%v mae=%v net=%v", *fp.MFE20, *fp.MAE20, *fp.Net20)
	}
	if fp.State != FollowStateComplete {
		t.Fatalf("state: %s", fp.State)
	}
}

// A FORMING bucket beyond the level is NOT a break (confirmation-truth).
func TestFollowPlanFormingBucketIsNotABreak(t *testing.T) {
	tp := &fpTape{t0: fpNow()}
	tp.flat(5, 101)
	tp.flat(3, 98) // three minutes of the next bucket below the level — the bucket has not closed
	now := tp.t0.Add(8 * time.Minute).UnixMilli()
	fp := ComputeFollowPlan(fpInput(tp, now))
	if fp.BreakAtMs != nil {
		t.Fatalf("a forming bucket must not read as a break: %+v", fp)
	}
	if fp.State != FollowStateOpen {
		t.Fatalf("state: %s", fp.State)
	}
	// Two minutes later the bucket closes below → now it is a break.
	tp.flat(2, 98)
	now = tp.t0.Add(10 * time.Minute).UnixMilli()
	if fp = ComputeFollowPlan(fpInput(tp, now)); fp.BreakAtMs == nil {
		t.Fatal("the closed bucket below the level is the break")
	}
}

// A level never broken: a plan with break_at NULL — a row, not a missing row.
// At the scope's end it is final: no_break.
func TestFollowPlanNeverBroken(t *testing.T) {
	tp := &fpTape{t0: fpNow()}
	tp.flat(30, 101)
	in := fpInput(tp, tp.t0.Add(30*time.Minute).UnixMilli())
	if fp := ComputeFollowPlan(in); fp.BreakAtMs != nil || fp.State != FollowStateOpen {
		t.Fatalf("unbroken and inside scope → open with break_at NULL: %+v", fp)
	}
	in.NowMs = in.ScopeEndMs
	if fp := ComputeFollowPlan(in); fp.BreakAtMs != nil || fp.State != FollowStateNoBreak {
		t.Fatalf("unbroken at scope end → no_break, break_at still NULL: %+v", fp)
	}
}

// A touch is not a fill: a retest bar that only reaches the level without
// trading through leaves the entry NULL with the assumption named.
func TestFollowPlanTouchIsNotAFill(t *testing.T) {
	tp := &fpTape{t0: fpNow()}
	tp.flat(5, 101)
	tp.flat(5, 98)
	tp.add(98, 100, 97.5, 99) // high == level exactly: touched, not through
	tp.flat(10, 97)
	now := tp.t0.Add(time.Duration(len(tp.bars)) * time.Minute).UnixMilli()
	fp := ComputeFollowPlan(fpInput(tp, now))
	if fp.RetestAtMs == nil {
		t.Fatal("the touch is a retest")
	}
	if fp.EntryPx != nil || fp.EntryBasis != FollowEntryTouchNotFill {
		t.Fatalf("a touch is not a fill: entry must be NULL with basis touch_not_fill: %+v", fp)
	}
	if fp.MFE10 != nil || fp.Net10 != nil {
		t.Fatal("no entry → no excursion")
	}
}

// Resistance mirror: entry from below, break UP, follow is a LONG at the level.
func TestFollowPlanResistanceMirror(t *testing.T) {
	tp := &fpTape{t0: fpNow()}
	tp.flat(5, 99)
	tp.flat(5, 102)                // closed above → break up
	tp.add(102, 102.5, 99.75, 101) // retest from above, through by a tick (low 99.75)
	tp.flat(50, 104)
	tp.flat(50, 98)
	now := tp.t0.Add(time.Duration(len(tp.bars)) * time.Minute).UnixMilli()
	in := fpInput(tp, now)
	in.EntrySide = "below"
	fp := ComputeFollowPlan(in)
	if fp.BreakDir != "up" || fp.BiasWouldFlipTo != "long" || fp.Direction != 1 || fp.EntryPx == nil || *fp.EntryPx != 100 {
		t.Fatalf("mirror wrong: %+v", fp)
	}
	if *fp.MFE10 != 4.5 || *fp.Net10 != 2 || *fp.MAE20 != 2.5 || *fp.Net20 != -4 {
		t.Fatalf("mirror excursions wrong: mfe10=%v net10=%v mae20=%v net20=%v", *fp.MFE10, *fp.Net10, *fp.MAE20, *fp.Net20)
	}
}
