// ── W2 — FADE PERMISSION: IS THE BOOK ALLOWED TO FADE RIGHT NOW? ─────────────
//
// A LABEL AND A COUNTER. NEVER A GATE. Nothing in this file refuses, sizes,
// cancels or authorizes anything; an excluded scenario is armed, placed and
// traded exactly as a permitted one. E3 decides whether any of it earns a rule.
//
// WHY IT IS ONLY A LABEL. Round 11 §1: a reliable early range-vs-trend
// classifier for MNQ is NOT established, and the published claim it names
// should not be adopted. So this ships a few PRE-DECLARED exclusions instead,
// and measures them.
//
// WHAT THE EVIDENCE SAYS ABOUT THEIR COVERAGE — stated here because a label
// implying protection it lacks is worse than no label. On 2026-09-03, the day
// the book sold into a +483-point run, the NY session authorized exactly three
// arms (ids 35, 36, 37). The one that FILLED — id 35, short at 29285.00, 09:02
// CT — is covered by NONE of these exclusions:
//
//	(a) the opening range was 62.25 pts = 0.77x the 13-session median 81.25;
//	    below median, so "too wide" cannot fire at any k >= 1.0
//	(b) the IB does not exist until 09:30; arm 35 fired at 09:02
//	(c) price never cleared the authored map — the planner re-seated levels
//	    ahead of price all morning (v4 max 29375.25 -> v5 29539.38 -> v6/v7
//	    29619.50), so "beyond the map" cannot catch a map that moves
//
// The honest headline: ON THE ONE DAY WE HAVE, THIS LABEL WOULD HAVE PERMITTED
// THE DAMAGING TRADE. That is E3's null, stated in advance.
//
// DAY_TYPE IS NEVER AN INPUT. It is model-WORDED free text — ten distinct
// values in the corpus, including "trend-down extension / oversold reversal
// watch" — so an exclusion keyed on it would string-match an LLM's adjectives
// and call the result a measurement. TestFadeFactsCarriesNoDayType pins it.
package kernel

import (
	"reflect"
	"sort"
	"time"
)

// Exclusion names. One definition; the strip, the Guide and the boot line all
// read these constants rather than retyping the strings (class 97).
const (
	FadeExOpeningRangeWide = "or_wide"
	FadeExIBBrokenHeld     = "ib_held"
	FadeExBeyondMap        = "beyond_map"
	FadeExT1News           = "t1_news"
	FadeExFirstNMinutes    = "first_n"
)

// FadeExclusionOrder is the declared list, in the order the boot line prints
// them. Pre-declared means declared HERE, before the wave shipped.
var FadeExclusionOrder = []string{
	FadeExOpeningRangeWide, FadeExIBBrokenHeld, FadeExBeyondMap,
	FadeExT1News, FadeExFirstNMinutes,
}

// FadeExclusionCoverageNote is the help text D5 requires: each exclusion states
// what it would NOT have caught, so the label cannot imply coverage it lacks.
var FadeExclusionCoverageNote = map[string]string{
	FadeExOpeningRangeWide: "would not have fired on 2026-09-03 (OR 0.77x median)",
	FadeExIBBrokenHeld:     "would not have fired on the 09-03 arm that filled (09:02, before the IB existed)",
	FadeExBeyondMap:        "structurally blind to a planner that re-seats ahead of price (09-03: map moved with the run)",
	FadeExT1News:           "UNKNOWN when the calendar has no slice; UNKNOWN never excludes",
	FadeExFirstNMinutes:    "reuses the existing no-trade band's N; it is a clock window, not a state reading",
}

// FadeExclusion is one exclusion's verdict, carrying what it measured and what
// it compared against. A threshold without its measured value is unauditable.
type FadeExclusion struct {
	Name      string
	Measured  float64
	Threshold float64
	SampleN   int    // n behind a resolved threshold (A21); 0 when not sample-derived
	Basis     string // "[I]" declared, or "[T] n=<n>" when a tape number backs it
}

// FadeFacts carries ONLY inputs knowable at `now` (C4). There is deliberately
// no day_type, no regime, no bias label, and no completed-session anything:
// round 11 forbids labelling a day from its finished profile and claiming the
// label was available at the open.
type FadeFacts struct {
	// (a) opening range — knowable once the 08:30-08:35 bar has CLOSED.
	ORComplete  bool
	OR5mPts     float64
	ORMedianPts float64
	ORMedianN   int
	ORWideK     float64

	// (b) IB broken and held — knowable from 09:30, then CONTINUOUSLY
	// (owner ruling 2026-09-10: the dispatch's "by 09:30" deadline was blind
	// to 09-03, whose break came at 10:00).
	IBComplete           bool
	IBHigh, IBLow        float64
	ClosedBucketBeyondIB bool

	// (c) beyond the map — knowable at any tick, from the seated references.
	SeatedRefsKnown bool
	MaxSeated       float64
	MinSeated       float64
	Direction       string // the scenario's own direction: "long" | "short"

	Price float64

	// (d) T1 blackout — UNKNOWN when the calendar has no slice for the day.
	CalendarHasSlice bool
	InT1Blackout     bool

	// (e) first N minutes — reuses the existing no-trade band's N, never a
	// retyped literal.
	SessionOpen   time.Time
	FirstNMinutes int
}

// FadeVerdict is the label. Evaluated=false means NOT EVALUATED, which is not
// "permitted" — A24 forbids a plausible zero standing in for a reading.
type FadeVerdict struct {
	Evaluated  bool
	Permitted  bool
	Exclusions []FadeExclusion
	Unknown    []string
}

// FadePermissionAt is PURE: it reads no clock, no store and no globals. `now`
// arrives as an argument (A28/class 60) and every field of f is something a
// caller could know at that instant.
//
// Each exclusion is evaluated INDEPENDENTLY and all that fire are named — the
// label is a list, never the first match (D1).
func FadePermissionAt(now time.Time, f FadeFacts) FadeVerdict {
	v := FadeVerdict{}

	// (e) FIRST N MINUTES. A clock window; evaluable whenever the session
	// open is known.
	if !f.SessionOpen.IsZero() && f.FirstNMinutes > 0 {
		v.Evaluated = true
		mins := now.Sub(f.SessionOpen).Minutes()
		if mins >= 0 && mins < float64(f.FirstNMinutes) {
			v.Exclusions = append(v.Exclusions, FadeExclusion{
				Name: FadeExFirstNMinutes, Measured: mins,
				Threshold: float64(f.FirstNMinutes), Basis: "[I]",
			})
		}
	}

	// (a) OPENING RANGE TOO WIDE. Nothing to say until the OR bar has CLOSED.
	if f.ORComplete && f.ORMedianPts > 0 && f.ORWideK > 0 {
		v.Evaluated = true
		ratio := f.OR5mPts / f.ORMedianPts
		if ratio > f.ORWideK {
			v.Exclusions = append(v.Exclusions, FadeExclusion{
				Name: FadeExOpeningRangeWide, Measured: f.OR5mPts,
				Threshold: f.ORWideK, SampleN: f.ORMedianN, Basis: "[I]",
			})
		}
	}

	// (b) IB BROKEN AND HELD, evaluated continuously. A CLOSED bucket beyond
	// the IB, per confirmation-truth — an intrabar poke is not acceptance.
	if f.IBComplete && f.IBHigh > 0 && f.IBLow > 0 {
		v.Evaluated = true
		beyond := f.Price > f.IBHigh || f.Price < f.IBLow
		if beyond && f.ClosedBucketBeyondIB {
			ref := f.IBHigh
			if f.Price < f.IBLow {
				ref = f.IBLow
			}
			v.Exclusions = append(v.Exclusions, FadeExclusion{
				Name: FadeExIBBrokenHeld, Measured: f.Price,
				Threshold: ref, Basis: "[I]",
			})
		}
	}

	// (c) PRICE BEYOND THE MAP, in the scenario's own direction.
	if f.SeatedRefsKnown && f.Price > 0 {
		v.Evaluated = true
		switch f.Direction {
		case "long":
			if f.MinSeated > 0 && f.Price < f.MinSeated {
				v.Exclusions = append(v.Exclusions, FadeExclusion{
					Name: FadeExBeyondMap, Measured: f.Price,
					Threshold: f.MinSeated, Basis: "[I]",
				})
			}
		case "short":
			if f.MaxSeated > 0 && f.Price > f.MaxSeated {
				v.Exclusions = append(v.Exclusions, FadeExclusion{
					Name: FadeExBeyondMap, Measured: f.Price,
					Threshold: f.MaxSeated, Basis: "[I]",
				})
			}
		}
	}

	// (d) T1 NEWS. An unanswerable question is UNKNOWN, and UNKNOWN NEVER
	// EXCLUDES — the two must stay distinguishable (E5).
	if !f.CalendarHasSlice {
		v.Unknown = append(v.Unknown, FadeExT1News)
	} else {
		v.Evaluated = true
		if f.InT1Blackout {
			v.Exclusions = append(v.Exclusions, FadeExclusion{
				Name: FadeExT1News, Measured: 1, Threshold: 1, Basis: "[I]",
			})
		}
	}

	sort.Slice(v.Exclusions, func(i, j int) bool { return v.Exclusions[i].Name < v.Exclusions[j].Name })
	v.Permitted = v.Evaluated && len(v.Exclusions) == 0
	return v
}

func hasExclusion(v FadeVerdict, name string) bool { return findExclusion(v, name) != nil }

func hasUnknown(v FadeVerdict, name string) bool {
	for _, u := range v.Unknown {
		if u == name {
			return true
		}
	}
	return false
}

func findExclusion(v FadeVerdict, name string) *FadeExclusion {
	for i := range v.Exclusions {
		if v.Exclusions[i].Name == name {
			return &v.Exclusions[i]
		}
	}
	return nil
}

// fadeFactsHasField reflects over the REAL struct, so the day_type pin cannot
// pass by a helper that always says no.
func fadeFactsHasField(name string) bool {
	_, ok := reflect.TypeOf(FadeFacts{}).FieldByName(name)
	return ok
}
