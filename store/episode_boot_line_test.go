// W1 / D6 — the episode boot line. READ, never literal (A11/A24).
//
// The owner's ruling: NAME THE RESOLVER, never freeze a per-read value. Δ is
// re-derived from the tape each read (kernel.MeanAbsIncrement — "the tape's OWN
// scale, re-derived per period, never a const"), so a number for it at boot is
// a literal wearing a resolver's clothes, and the reader will quote it as the
// constant. The existing detector line already does this and is the form we
// match: "detector: D1′ k=%.0f Δ=resolved-per-read band=k×Δ H=%d …".
//
// AND THIS TEST IS MUTATION-PROVABLE, which the last boot-line pin in this repo
// was not: it moves the resolver and requires the line to move with it. A guard
// that derives its expectation from the thing it guards can only confirm today's
// agreement (checklist class 93).

package store

import (
	"strings"
	"testing"
)

func TestEpisodeBootLineNamesResolversNotValues(t *testing.T) {
	line := EpisodeBootLine(EpisodeBootCounts{
		Open: 2, ClosedToday: 7,
		NeverReached: 3, ReachedDeclined: 2, ConfirmedNotArmed: 1, ArmedNotFilled: 1, Filled: 0,
		BackfillRan: true, BackfillRecomputed: 41, BackfillUnrecomputable: 966,
	})

	// Δ must be named, never numbered.
	if !strings.Contains(line, "Δ=resolved-per-read") {
		t.Errorf("Δ must be named as resolved-per-read, not frozen at boot: %s", line)
	}
	if !strings.Contains(line, "MeanAbsIncrement") {
		t.Errorf("the line must name Δ's SOURCE so a reader can check it: %s", line)
	}
	// The invented parameters must carry their label.
	for _, want := range []string{"k=", "H=", "[I]"} {
		if !strings.Contains(line, want) {
			t.Errorf("missing %q: %s", want, line)
		}
	}
	// Every outcome rung must be countable from the line.
	for _, want := range []string{"never_reached=3", "reached_declined=2", "confirmed_not_armed=1", "armed_not_filled=1", "filled=0"} {
		if !strings.Contains(line, want) {
			t.Errorf("missing %q: %s", want, line)
		}
	}
	// A24: a rate without n is forbidden — the backfill states BOTH counts.
	if !strings.Contains(line, "recomputed=41") || !strings.Contains(line, "unrecomputable=966") {
		t.Errorf("backfill must state both counts, never a percentage alone: %s", line)
	}
}

// MUTATION: move the resolver, the line MUST move. This is the check the
// previous boot-line pin in this repo lacked.
func TestEpisodeBootLineFollowsItsResolver(t *testing.T) {
	before := EpisodeBootLine(EpisodeBootCounts{})
	t.Setenv("DETECTOR_K", "5")
	t.Setenv("DETECTOR_HORIZON_BARS", "20")
	after := EpisodeBootLine(EpisodeBootCounts{})

	if before == after {
		t.Fatalf("the line did not move when its resolvers did — it is reporting literals:\n  before %s\n  after  %s", before, after)
	}
	if !strings.Contains(after, "k=5") || !strings.Contains(after, "H=20") {
		t.Errorf("the line must render the RESOLVED values: %s", after)
	}
}

// An uncomputed count is stated, never silently zero (A24). A zero backfill and
// an unrun backfill must not read the same.
func TestEpisodeBootLineDistinguishesZeroFromUnrun(t *testing.T) {
	unrun := EpisodeBootLine(EpisodeBootCounts{BackfillRan: false})
	ran := EpisodeBootLine(EpisodeBootCounts{BackfillRan: true})
	if unrun == ran {
		t.Fatal("an unrun backfill must not render identically to one that ran and found nothing")
	}
	if !strings.Contains(unrun, "n/a") {
		t.Errorf("a field the process cannot know yet prints n/a: %s", unrun)
	}
}
