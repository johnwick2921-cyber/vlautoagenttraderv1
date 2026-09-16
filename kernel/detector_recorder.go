package kernel

import (
	"fmt"
	"math"
)

// ── 1B WRITERS — the detector's output reaches the tables ────────────────────
//
// D1 shipped a calibrated detector that nothing called. These are the two pure
// functions the production paths hand their data to, kept in the kernel so they
// can be tested without a store and without touching seating logic (a hard
// stop-line: this wave LOGS the pool and MEASURES touches, it changes no
// decision).

// CandidateRecord is one level the constructor produced, seated or cut, with
// the propensity that decided it.
type CandidateRecord struct {
	Price      float64
	Kind       string
	Label      string
	Rank       int // 1-based within the seated set; 0 when cut
	Seated     bool
	CutReason  string
	Score      float64
	Threshold  float64
	Grade      string
	Components string // JSON; "{}" = not computed, never "" (A24)
}

// BuildCandidatePool classifies every produced level against the seated set.
// It DERIVES the cut reason from the same inputs the constructor used — it does
// not re-run or alter the seating decision, so the record can never disagree
// with the plan by making its own judgement.
func BuildCandidatePool(all []DetectedLevel, seated []ScoredLevel, price, dATR, proximityK float64, maxLevels int) []CandidateRecord {
	if len(all) == 0 {
		return nil
	}
	if proximityK <= 0 {
		proximityK = ActivationWindowK
	}
	if maxLevels <= 0 {
		maxLevels = DefaultMaxLevels
	}
	band := proximityK * dATR
	rankOf := map[string]int{}
	scoreOf := map[string]float64{}
	gradeOf := map[string]string{}
	key := func(p float64, kind, label string) string { return fmt.Sprintf("%.4f|%s|%s", p, kind, label) }
	for i, s := range seated {
		k := key(s.Price, string(s.Kind), s.Label)
		rankOf[k] = i + 1
		scoreOf[k] = s.Score
		gradeOf[k] = s.Grade
	}
	// The seating threshold, observed rather than assumed: the weakest score
	// that made it in. With nothing seated there is no threshold to report.
	threshold := 0.0
	if len(seated) > 0 {
		threshold = seated[0].Score
		for _, s := range seated {
			if s.Score < threshold {
				threshold = s.Score
			}
		}
	}
	out := make([]CandidateRecord, 0, len(all))
	for _, l := range all {
		k := key(l.Price, string(l.Kind), l.Label)
		rec := CandidateRecord{
			Price: l.Price, Kind: string(l.Kind), Label: l.Label,
			Threshold: threshold, Components: researchComponents(l.Research),
		}
		if l.Research != nil {
			if l.Research.Score != nil {
				rec.Score = *l.Research.Score
			}
			if l.Research.Grade != nil {
				rec.Grade = *l.Research.Grade
			}
			if l.Research.Exclusion != nil {
				rec.CutReason = *l.Research.Exclusion
			}
		}
		if r, ok := rankOf[k]; ok {
			rec.Rank, rec.Seated, rec.Score, rec.Grade = r, true, scoreOf[k], gradeOf[k]
			out = append(out, rec)
			continue
		}
		if rec.CutReason != "" {
			out = append(out, rec)
			continue
		}
		// Cut. The reason is derived from the constructor's own predicates, in
		// the order it applies them, so the record explains the decision rather
		// than guessing at it.
		switch {
		case dATR > 0 && math.Abs(l.Price-price) > band:
			rec.CutReason = fmt.Sprintf("proximity: %.2f pts from price > %.1f×dATR (%.2f)", math.Abs(l.Price-price), proximityK, band)
		case len(seated) >= maxLevels:
			rec.CutReason = fmt.Sprintf("max_levels: %d seated, cap %d", len(seated), maxLevels)
		default:
			rec.CutReason = "not seated (cluster collapse or grade floor)"
		}
		out = append(out, rec)
	}
	return out
}

// NewEpisodesSince filters a detector run to the episodes not yet recorded, so
// a per-read hook is safe to call repeatedly: only episodes opening AFTER the
// last stored one are new. Restart-safe, because the watermark comes from the
// store rather than memory.
func NewEpisodesSince(eps []TouchOutcome, lastOpenedMs int64) []TouchOutcome {
	out := make([]TouchOutcome, 0, len(eps))
	for _, e := range eps {
		if e.OpenedAtMs > lastOpenedMs && e.ClosedAtMs != 0 {
			out = append(out, e)
		}
	}
	return out
}
