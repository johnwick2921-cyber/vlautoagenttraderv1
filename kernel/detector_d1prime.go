package kernel

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"nofx/market"
)

// ── D1′ — THE CALIBRATED TOUCH DETECTOR (2026-09-03) ─────────────────────────
//
// Replaces two instruments that could not be wrong in the direction that
// mattered:
//   · touch_telemetry.go called a touch a "rejection" when the close was still
//     on its starting side — ≈0.69 on IID noise BY CONSTRUCTION.
//   · level_stats_calc.go:70 counted ANY ≥reactPts move away from the level as
//     a reaction, so a blast-through scored as a reaction.
// Every reaction rate ever published from them (84%, 70.3%, 75.1%) is an
// artifact of the predicate, not a property of the tape.
//
// D1′ is a gambler's-ruin design: barriers are anchored to the LEVEL at
// L ± k·Δ, so they are equidistant from the start point and a driftless walk
// is a coin flip. Calibrated on IID-shuffled tape at p(hold) = 0.5067
// (k=3, exit_on=close, H=12) — the report's stated conclusion and the operating
// point BOTH replays (1m and 1h) ran on, so the live instrument matches the
// baselines it will be compared against.
//
// OWNER RULING 2026-09-03 on chosen_k.json: that artifact says k=6/range. Its
// selection rule minimized |p−0.5| and IGNORED ambiguity share — k=6 discards
// 50.6% of episodes (n 56,922 → 21,948) to buy 0.0015, which is inside the
// confidence interval. The report's k=3 stands; k=3/range is computed as a
// SENSITIVITY variant and presented beside it, never used for decisions.
//
// Reference: ~/nofx-analysis/detector-redesign/detectors.py detect_symmetric_v2.

// TouchOutcome is one episode. Ambiguous outcomes are RECORDED and excluded
// from the rate — never dropped (that is how a rate lies about its own base).
type TouchOutcome struct {
	Ordinal    int
	OpenedAtMs int64
	ClosedAtMs int64
	Entry      string // "below" | "above" — the side price came FROM
	Exit       string // "below" | "above" | "" when ambiguous
	Outcome    string // hold | break | ambiguous_span | ambiguous_horizon
	BarsToExit int
	MFE        float64 // best excursion in the entry direction, pts
	MAE        float64 // worst excursion against it, pts (≤ 0)
}

// IsAmbiguous reports whether this episode is excluded from the hold rate.
func (o TouchOutcome) IsAmbiguous() bool { return strings.HasPrefix(o.Outcome, "ambiguous") }

// ── resolved knobs (A11: resolved values, never file defaults at the call) ───

// DetectorK is the barrier multiple. Default 3 (report conclusion).
func DetectorK() float64 { return envFloatDefault("DETECTOR_K", 3.0) }

// DetectorHorizonBars is the episode horizon in bars. Default 12.
func DetectorHorizonBars() int { return envIntDefault("DETECTOR_HORIZON_BARS", 12) }

// DetectorExitOn is "close" (decisions) or "range" (sensitivity only).
func DetectorExitOn() string {
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("DETECTOR_EXIT_ON"))); v == "range" || v == "close" {
		return v
	}
	return "close"
}

// DetectorDeltaDays is the trailing session-day window Δ is measured over.
func DetectorDeltaDays() int { return envIntDefault("DETECTOR_DELTA_DAYS", 5) }

func envFloatDefault(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil && f > 0 {
			return f
		}
	}
	return def
}

func envIntDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// MeanAbsIncrement is Δ: the mean |close-to-close increment| over the supplied
// bars. This is the tape's OWN scale, re-derived per period — never a constant.
// Fewer than 2 bars → 0, and a 0 Δ means the detector cannot run (no band).
func MeanAbsIncrement(bars []market.Kline) float64 {
	if len(bars) < 2 {
		return 0
	}
	sum, n := 0.0, 0
	for i := 1; i < len(bars); i++ {
		sum += math.Abs(bars[i].Close - bars[i-1].Close)
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// DetectTouchOutcomes is the ONE live detector. A port of detect_symmetric_v2,
// deliberately line-for-line so the parity fixture can be byte-equal.
//
// An episode OPENS on a genuine touch of the level — the bar's range contains
// L and the PREVIOUS bar's did not — not on entry to a band edge. The entry
// side is the side the previous bar closed on. It CLOSES when a close (or, in
// range mode, a bar's range) passes a barrier: exiting the entry side is a
// HOLD, the far side is a BREAK, both in one bar is ambiguous_span, and running
// out of horizon is ambiguous_horizon.
func DetectTouchOutcomes(bars []market.Kline, level, k, delta float64, horizon int, exitOn string) []TouchOutcome {
	if level <= 0 || k <= 0 || delta <= 0 || horizon <= 0 || len(bars) < 2 {
		return nil
	}
	up, lo := level+k*delta, level-k*delta
	var eps []TouchOutcome
	ordinal := 0
	n := len(bars)
	for i := 1; i < n; {
		b, pb := bars[i], bars[i-1]
		touch := pb.Low <= level && level <= pb.High
		cur := b.Low <= level && level <= b.High
		if !(cur && !touch) {
			i++
			continue
		}
		entry := "above"
		if pb.Close < level {
			entry = "below"
		}
		ordinal++
		outcome, exitSide := "", ""
		mfe, mae := 0.0, 0.0
		j := i + 1
		for ; j < n && (j-i) <= horizon; j++ {
			c := bars[j]
			if entry == "below" {
				mfe = math.Max(mfe, c.High-level)
				mae = math.Min(mae, c.Low-level)
			} else {
				mfe = math.Max(mfe, level-c.Low)
				mae = math.Min(mae, level-c.High)
			}
			var cu, cd bool
			if exitOn == "close" {
				cu, cd = c.Close > up, c.Close < lo
			} else {
				cu, cd = c.High > up, c.Low < lo
			}
			if cu && cd {
				outcome = "ambiguous_span"
				break
			}
			if cu {
				outcome, exitSide = breakOrHold(entry, "above"), "above"
				break
			}
			if cd {
				outcome, exitSide = breakOrHold(entry, "below"), "below"
				break
			}
		}
		if outcome == "" {
			outcome = "ambiguous_horizon"
		}
		closedAt := int64(0)
		if j < n {
			closedAt = bars[j].OpenTime
		}
		eps = append(eps, TouchOutcome{
			Ordinal: ordinal, OpenedAtMs: b.OpenTime, ClosedAtMs: closedAt,
			Entry: entry, Exit: exitSide, Outcome: outcome,
			BarsToExit: j - i, MFE: mfe, MAE: mae,
		})
		i = j + 1
	}
	return eps
}

// breakOrHold: exiting the side you came FROM is a hold; the far side a break.
func breakOrHold(entry, exit string) string {
	if entry == exit {
		return "hold"
	}
	return "break"
}

// HoldRate returns hold/(hold+break) with n, and the ambiguous count that was
// EXCLUDED. A rate without its n is not evidence (A24), so callers get both.
func HoldRate(eps []TouchOutcome) (p float64, n, ambiguous int) {
	hold, brk := 0, 0
	for _, e := range eps {
		switch {
		case e.IsAmbiguous():
			ambiguous++
		case e.Outcome == "hold":
			hold++
		case e.Outcome == "break":
			brk++
		}
	}
	n = hold + brk
	if n == 0 {
		return 0, 0, ambiguous
	}
	return float64(hold) / float64(n), n, ambiguous
}

// WilsonInterval is the 95% score interval for p over n. Returns (0,0) at n=0.
func WilsonInterval(p float64, n int) (lo, hi float64) {
	if n <= 0 {
		return 0, 0
	}
	const z = 1.959963984540054
	nf := float64(n)
	den := 1 + z*z/nf
	centre := (p + z*z/(2*nf)) / den
	half := z * math.Sqrt(p*(1-p)/nf+z*z/(4*nf*nf)) / den
	return centre - half, centre + half
}

// DetectorRateFloor is the n below which a rate is DESCRIPTIVE ONLY.
const DetectorRateFloor = 200

// FormatHoldRate renders a rate that can never be quoted without its base.
func FormatHoldRate(eps []TouchOutcome) string {
	p, n, amb := HoldRate(eps)
	if n == 0 {
		return "p(hold)=n/a (n=0" + ambSuffix(amb) + ")"
	}
	lo, hi := WilsonInterval(p, n)
	s := "p(hold)=" + strconv.FormatFloat(p, 'f', 3, 64) +
		" [" + strconv.FormatFloat(lo, 'f', 3, 64) + ", " + strconv.FormatFloat(hi, 'f', 3, 64) + "]" +
		" n=" + strconv.Itoa(n) + ambSuffix(amb)
	if n < DetectorRateFloor {
		s += " — n<" + strconv.Itoa(DetectorRateFloor) + ", DESCRIPTIVE ONLY"
	}
	return s
}

func ambSuffix(amb int) string {
	if amb == 0 {
		return ""
	}
	return " · ambiguous=" + strconv.Itoa(amb) + " excluded"
}

// MeanAbsIncrementOf is Δ from a bare increment series (the calibration path).
func MeanAbsIncrementOf(incs []float64) float64 {
	if len(incs) == 0 {
		return 0
	}
	sum := 0.0
	for _, d := range incs {
		sum += math.Abs(d)
	}
	return sum / float64(len(incs))
}

// DetectorBootLine reports the live detector and the two tables recording it.
// Every field READ from its resolver; the counts come from the tables, so an
// empty table says 0 rather than implying a measurement (A24).
// The retirement clause is RESTORED, and only because a grep earned it (owner
// ruling 2026-09-03: it stays out until a grep proves it). Checked across
// web/src/guide/content and the Go tree: no surface renders a RATE from the
// retired instruments. The single Guide hit — "first-touch reactions with
// confirmation" in the REACT ZONE card — is a play DESCRIPTION, not a measured
// rate, so it is not a retirement target. TestNoSurfaceRendersARetiredRate
// re-runs that grep on every build.
func DetectorBootLine(outcomes, pool int64) string {
	return fmt.Sprintf("detector: D1′ k=%.0f Δ=resolved-per-read band=k×Δ H=%d exit_on=%s · touch_outcomes=%d · candidate_pool=%d · legacy rates retired (grep-verified: no surface renders one)",
		DetectorK(), DetectorHorizonBars(), DetectorExitOn(), outcomes, pool)
}
