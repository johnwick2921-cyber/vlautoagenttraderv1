package kernel

import (
	"testing"
	"time"
)

// ── W2 FADE PERMISSION — GATE-LEVEL FIXTURES, WRITTEN BEFORE THE PREDICATE ───
//
// Every one of these is RED until kernel/fade_permission.go exists. They pin
// the LABEL's behaviour, never a refusal: A31 and D3 say an excluded scenario
// is armed, placed and traded exactly as a permitted one.
//
// A28: each test owns ONE clock and states it. No test reads time.Now().

// sep3CT is the wave's single clock helper. 2026-09-03 is the day the book sold
// into a +483-point run; every replay below states its CT moment explicitly.
func sep3CT(t *testing.T, hhmm string) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("tz: %v", err)
	}
	got, err := time.ParseInLocation("2006-01-02 15:04", "2026-09-03 "+hhmm, loc)
	if err != nil {
		t.Fatalf("clock: %v", err)
	}
	return got
}

// The 2026-09-03 NY tape, measured from the store (C2/C5):
//
//	OR 5m      62.25 pts   = 0.77x the 13-session median 81.25
//	IB 08:30-09:30  high 29375.25  low 29199.25  size 176.00
//	first CLOSED 5m bucket beyond the IB high: 10:00 CT at 29436.75
const (
	sep3ORPts     = 62.25
	sep3ORMedian  = 81.25
	sep3IBHigh    = 29375.25
	sep3IBLow     = 29199.25
	orWideKDefaul = 1.28 // C5's p80 / median
)

// E1 — THE 09-03 PIN. After the IB break, a fade scenario carries `excluded`
// naming (b); before the IB exists, it does NOT — and the measured evidence is
// that the arm which actually filled (id 35, 09:02 CT) is uncovered by every
// exclusion. This test pins BOTH halves, because a label that quietly claimed
// coverage it lacks would be worse than no label.
func TestSep3FadeAfterIBBreakIsExcludedAndBeforeItIsNot(t *testing.T) {
	// the filled arm: 09:02 CT, IB not yet complete
	early := FadePermissionAt(sep3CT(t, "09:02"), FadeFacts{
		ORComplete: true, OR5mPts: sep3ORPts, ORMedianPts: sep3ORMedian,
		ORMedianN: 13, ORWideK: orWideKDefaul,
		IBComplete: false,
		Price:      29267.5,
	})
	if !early.Evaluated {
		t.Fatalf("09:02 must be EVALUATED (OR was complete), got not-evaluated")
	}
	if len(early.Exclusions) != 0 {
		t.Errorf("09:02 arm 35 is the one that FILLED and no exclusion covers it; got %v", early.Exclusions)
	}
	if !early.Permitted {
		t.Errorf("09:02 must read permitted — that is the finding, not a bug")
	}

	// arm 37's moment: 11:58 CT, price held closed buckets beyond the IB high
	late := FadePermissionAt(sep3CT(t, "11:58"), FadeFacts{
		ORComplete: true, OR5mPts: sep3ORPts, ORMedianPts: sep3ORMedian,
		ORMedianN: 13, ORWideK: orWideKDefaul,
		IBComplete: true, IBHigh: sep3IBHigh, IBLow: sep3IBLow,
		ClosedBucketBeyondIB: true,
		Price:                29490.0,
	})
	if late.Permitted {
		t.Errorf("11:58 is beyond a held IB break — must be excluded")
	}
	if !hasExclusion(late, "ib_held") {
		t.Errorf("11:58 must name ib_held; got %v", late.Exclusions)
	}
}

// E2 — KNOWABLE-AT-NOW. An exclusion whose input postdates `now` cannot fire,
// and the verdict is NOT-EVALUATED, never permitted (A24: no plausible zero).
func TestUncompletedOpeningRangeIsNotEvaluatedNotPermitted(t *testing.T) {
	v := FadePermissionAt(sep3CT(t, "08:33"), FadeFacts{ORComplete: false})
	if v.Evaluated {
		t.Fatalf("08:33: the OR bar has not closed — must be NOT evaluated")
	}
	if v.Permitted {
		t.Errorf("not-evaluated must never read permitted")
	}
	// THE OTHER HALF, so this cannot pass against a predicate that evaluates
	// nothing: once the OR HAS closed, the same call must evaluate. Without
	// this the test is satisfied by a stub returning zero values, which is the
	// vacuous pass A8 warns about.
	done := FadePermissionAt(sep3CT(t, "08:36"), FadeFacts{
		ORComplete: true, OR5mPts: sep3ORPts, ORMedianPts: sep3ORMedian,
		ORMedianN: 13, ORWideK: orWideKDefaul,
	})
	if !done.Evaluated {
		t.Errorf("08:36: the OR bar HAS closed — must be evaluated")
	}
}

// E4 — INDEPENDENCE. Two exclusions firing means BOTH are named; the label is
// a list, never the first match.
func TestEveryFiringExclusionIsNamed(t *testing.T) {
	v := FadePermissionAt(sep3CT(t, "10:30"), FadeFacts{
		ORComplete: true, OR5mPts: 200, ORMedianPts: sep3ORMedian,
		ORMedianN: 13, ORWideK: orWideKDefaul,
		IBComplete: true, IBHigh: sep3IBHigh, IBLow: sep3IBLow,
		ClosedBucketBeyondIB: true,
		Price:                29500,
	})
	if !hasExclusion(v, "or_wide") || !hasExclusion(v, "ib_held") {
		t.Fatalf("both or_wide and ib_held must be named; got %v", v.Exclusions)
	}
	if len(v.Exclusions) < 2 {
		t.Errorf("the label carries ALL that fired, got %d", len(v.Exclusions))
	}
}

// E5 — UNKNOWN NEVER EXCLUDES. No calendar slice means the T1 question could
// not be answered; that is not the same as answering "no blackout", and it is
// certainly not an exclusion.
func TestAbsentCalendarSliceIsUnknownNotExcluded(t *testing.T) {
	v := FadePermissionAt(sep3CT(t, "10:30"), FadeFacts{
		ORComplete: true, OR5mPts: sep3ORPts, ORMedianPts: sep3ORMedian,
		ORMedianN: 13, ORWideK: orWideKDefaul,
		CalendarHasSlice: false,
	})
	if hasExclusion(v, "t1_news") {
		t.Errorf("an unanswerable question must not fire an exclusion")
	}
	if !hasUnknown(v, "t1_news") {
		t.Errorf("t1_news must be reported UNKNOWN; got unknown=%v", v.Unknown)
	}
	if !v.Permitted {
		t.Errorf("UNKNOWN does not exclude, so this reads permitted")
	}
}

// E7 — THRESHOLD RESOLVER. k comes from the bound strategy; absent, the C5
// default carries its own provenance. A number with no provenance is a literal
// (A11/A24).
func TestThresholdCarriesItsResolvedProvenance(t *testing.T) {
	v := FadePermissionAt(sep3CT(t, "10:30"), FadeFacts{
		ORComplete: true, OR5mPts: 200, ORMedianPts: sep3ORMedian,
		ORMedianN: 13, ORWideK: orWideKDefaul,
	})
	ex := findExclusion(v, "or_wide")
	if ex == nil {
		t.Fatalf("or_wide must fire at 200 vs median 81.25")
	}
	if ex.Threshold != orWideKDefaul {
		t.Errorf("threshold = %v, want the resolved %v", ex.Threshold, orWideKDefaul)
	}
	if ex.Measured != 200 {
		t.Errorf("measured = %v, want the value actually read", ex.Measured)
	}
	if ex.SampleN != 13 {
		t.Errorf("the median's n must travel with it (A21), got %d", ex.SampleN)
	}
}

// DAY_TYPE IS NEVER AN INPUT. It is model-WORDED free text — ten distinct
// values in the corpus, including "trend-down extension / oversold reversal
// watch" — so an exclusion keyed on it would string-match an LLM's adjectives
// and call the result a measurement. This pin fails if FadeFacts ever grows a
// day_type field.
func TestFadeFactsCarriesNoDayType(t *testing.T) {
	for _, banned := range []string{"DayType", "Regime", "BiasLabel"} {
		if fadeFactsHasField(banned) {
			t.Errorf("FadeFacts must not read %s — it is model-worded prose, not a measurement", banned)
		}
	}
}
