package store

import "fmt"

// ── CONTINUOUS SERIES (HISTORY IMPORT, wave 101) ─────────────────────────────
//
// THE SEAM RULE: a bar is never read across a roll silently. Raw access is
// per-contract (BarsBetweenOn / LastNBarsOn / WindowContract), and the current-
// contract filter covers live. A backtest that wants ONE series spanning rolls
// must build it DELIBERATELY — and the result is LABELLED as adjusted so it
// can never be mistaken for raw tape.
//
// BuildContinuous is that builder, and it is a PURE function: it returns rows,
// it never writes. It requires the caller to supply, per seam, the measured
// basis in POINTS as an explicit argument — never inferred from anything. If a
// caller cannot measure the basis it must not build a continuous series.
//
// The returned rows carry Contract=ContractMixed and
// Source=BarSourceContinuous, so if anyone hands them to a writer the store
// REFUSES them (InsertBars rejects continuous source; ImportBars rejects any
// source but historical_import) — an adjusted series cannot be persisted by
// accident, and a reader can always tell what it is holding.

// ContinuousSegment is one contract's slice of the series, in chronological
// order. Segments must not overlap in time; the caller owns that.
type ContinuousSegment struct {
	Contract string         // e.g. "MNQ 09-23"
	Bars     []BarHistoryDB // ascending by OpenTimeMs, all on this contract
}

// BuildContinuous stitches segments into one price scale: every segment after
// the first is shifted by the running sum of the preceding seams' basis
// (positive basis = the newer contract trades that many points ABOVE the older
// one, so its prices are shifted DOWN to the first contract's scale). A basis
// must be supplied per seam (len(segs)-1 values); a missing or empty basis is
// an error — a seam without a measured basis is a refusal, not a guess (A24).
func BuildContinuous(segs []ContinuousSegment, basisPts []float64) ([]BarHistoryDB, error) {
	if len(segs) == 0 {
		return nil, nil
	}
	if len(basisPts) != len(segs)-1 {
		return nil, fmt.Errorf("continuous: %d segments need %d explicit basis values, got %d — a seam without a measured basis is a refusal (wave 101)", len(segs), len(segs)-1, len(basisPts))
	}
	for i, b := range basisPts {
		if b == 0 {
			return nil, fmt.Errorf("continuous: seam %d (after %s) carries a zero basis — a zero basis is a claim to be proved, not a default", i+1, segs[i].Contract)
		}
	}
	var out []BarHistoryDB
	running := 0.0
	for i, seg := range segs {
		if i > 0 {
			running += basisPts[i-1]
		}
		if seg.Contract == "" {
			return nil, fmt.Errorf("continuous: segment %d has no contract name — an unlabelled segment is an accidental join (wave 101)", i)
		}
		for _, b := range seg.Bars {
			// Adjusted series are labelled, never raw: a reader that sees
			// Source=continuous:adjusted knows the prices are shifted.
			b.Contract = ContractMixed
			b.Source = BarSourceContinuous
			if i > 0 {
				b.O -= running
				b.H -= running
				b.L -= running
				b.C -= running
			}
			out = append(out, b)
		}
	}
	return out, nil
}
