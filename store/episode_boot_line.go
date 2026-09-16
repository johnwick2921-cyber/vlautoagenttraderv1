// W1 / D6 — the episode boot line.
//
// NAMES THE RESOLVER, NEVER THE VALUE, for anything re-derived per read. Δ is
// the tape's own scale, recomputed from bars every period (kernel's
// MeanAbsIncrement), so printing a number for it at boot would be a literal
// wearing a resolver's clothes — and the next reader would quote it as the
// constant. k and H ARE resolved once from the environment, so those render
// their resolved values and carry the [I] label that says they are invented
// parameters rather than validated ones.
//
// The form matches the detector's own line so the two read as one family:
//   detector: D1′ k=%.0f Δ=resolved-per-read band=k×Δ H=%d exit_on=%s …

package store

import "fmt"

// EpisodeBootCounts are READ at boot. BackfillRan distinguishes "ran and found
// nothing" from "has not run" — a zero and an unknown must never render alike.
type EpisodeBootCounts struct {
	Open        int64
	ClosedToday int64

	NeverReached      int64
	ReachedDeclined   int64
	ConfirmedNotArmed int64
	ArmedNotFilled    int64
	Filled            int64

	BackfillRan            bool
	BackfillRecomputed     int64
	BackfillUnrecomputable int64
}

// episodeDetectorScope is the seam the boot line reads its detector parameters
// through. It is a variable so a test can move the resolver and prove the line
// moves with it; production never reassigns it.
var episodeDetectorScope = func() (k float64, horizon int) {
	return detectorKForBoot(), detectorHorizonForBoot()
}

// EpisodeBootLine renders the episode posture. Every number is READ.
func EpisodeBootLine(c EpisodeBootCounts) string {
	k, h := episodeDetectorScope()

	backfill := "backfill n/a (not run)"
	if c.BackfillRan {
		backfill = fmt.Sprintf("backfill recomputed=%d unrecomputable=%d", c.BackfillRecomputed, c.BackfillUnrecomputable)
	}

	return fmt.Sprintf(
		"🎫 episodes: open=%d · closed today=%d (never_reached=%d reached_declined=%d confirmed_not_armed=%d armed_not_filled=%d filled=%d) · %s · k=%.0f[I] Δ=resolved-per-read (kernel.MeanAbsIncrement, the tape's own scale) H=%d[I]",
		c.Open, c.ClosedToday,
		c.NeverReached, c.ReachedDeclined, c.ConfirmedNotArmed, c.ArmedNotFilled, c.Filled,
		backfill, k, h)
}
