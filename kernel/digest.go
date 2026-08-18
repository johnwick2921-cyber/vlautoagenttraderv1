package kernel

import (
	"fmt"
	"strings"
)

// P3.6-A — digest formatting + the tapered week chain. Pure + deterministic.
// The map remembers structure (level-state); the digest remembers the story.

// FormatSessionDigest renders a session's 3-line close digest.
func FormatSessionDigest(session, tradeDate, dayType string, entries int, realizedPnL float64) string {
	return fmt.Sprintf("%s %s — %s\n%d entries · realized %+.1f\noutcome: %s",
		session, tradeDate, orDash(dayType), entries, realizedPnL, pnlWord(realizedPnL))
}

// FormatDailyDigest renders the trade-date roll-up (3 lines).
func FormatDailyDigest(tradeDate, dayType string, sessions, entries int, realizedPnL float64) string {
	return fmt.Sprintf("%s (daily) — %s\n%d session(s), %d entries · realized %+.1f\noutcome: %s",
		tradeDate, orDash(dayType), sessions, entries, realizedPnL, pnlWord(realizedPnL))
}

// BuildDigestChain assembles the tapered week chain per the contract: the
// current trade-date's session digests (0–2 × 3 lines) + the last 3 daily digests
// FULL + days 4–7 as one-liners. `dailies` is newest-first.
func BuildDigestChain(sessionDigests, dailies []string) []string {
	var chain []string
	chain = append(chain, sessionDigests...)
	for i, d := range dailies {
		switch {
		case i < 3:
			chain = append(chain, d) // full 3-line
		case i < 7:
			chain = append(chain, firstLine(d)) // one-liner
		default:
			return chain
		}
	}
	return chain
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func pnlWord(p float64) string {
	switch {
	case p > 0:
		return "green"
	case p < 0:
		return "red"
	default:
		return "flat"
	}
}

// LearningTrade is one closed trade's learning snapshot for the digest line.
type LearningTrade struct {
	MAE, MFE    float64
	Grade       string // adherence grade A-F ("" = ungraded)
	PlanVersion int
}

// LearningLine renders the learning-loop digest line: average MAE/MFE and the
// adherence-grade distribution, with per-trade plan versions so the grades are
// linkable to the plan that governed them.
//
// P0-cleanup (2026-08-19) — before this, MAE/MFE and adherence were computed and
// stored but never reached the digest (the learning loop was half-blind).
func LearningLine(trades []LearningTrade) string {
	if len(trades) == 0 {
		return ""
	}
	var smae, smfe float64
	n := 0
	grades := map[string]int{}
	var versions []int
	for _, t := range trades {
		if t.MAE > 0 || t.MFE > 0 {
			smae += t.MAE
			smfe += t.MFE
			n++
		}
		if t.Grade != "" {
			grades[t.Grade]++
		}
		if t.PlanVersion > 0 {
			versions = append(versions, t.PlanVersion)
		}
	}
	if n == 0 {
		return ""
	}
	line := fmt.Sprintf("learning: avg MAE %.1f · avg MFE %.1f · adherence %v",
		smae/float64(n), smfe/float64(n), grades)
	if len(versions) > 0 {
		line += fmt.Sprintf(" · plan v%v", versions)
	}
	return line
}
