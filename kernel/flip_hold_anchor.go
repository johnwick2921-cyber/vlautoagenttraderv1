package kernel

import (
	"strings"
)

// W-FLIP-HOLD-ANCHOR (2026-09-17, owner report "it went up all night and
// never flipped") — the G3 flip hysteresis used to measure "plan age" from the
// CURRENT VERSION's created_at. Every level-event wake re-read appends a new
// version (the plans chain is append-only), so the 30-minute hold RESTARTED
// on every re-read: 2026-09-16 ASIA authored v10 23:36, v11 00:13, v12 00:39,
// v13 01:21 — all bias short, flip "above X → long" — price broke up through
// v13's line at 01:30 and the journal read "flip=hold (plan age 815s)" at
// 01:35, 903s at 01:36, 1023s at 01:38. With wakes every 30–40 min the flip
// was held for most of the session.
//
// The hold is now anchored to the plan's STATE, never to a re-read version:
// the latest of the kinds listed in FlipHoldAnchorKinds. A wake re-read that
// keeps the bias (level_event / structure_mss / a scheduled read) never moves
// the anchor. The CONDITION window (bars since the version's birth — the
// touch gate and the confirm closes) is untouched: a version's own flip line
// is still judged only on bars written after that version.

// FlipHoldAnchor is the instant the flip hysteresis counts FROM, with the
// kind of event that set it (one of FlipHoldAnchorKinds, or
// FlipHoldAnchorVersion when the chain could not be read and the evaluator
// fell back to the version's own birth — the pre-fix behaviour, visible on
// the skip line as such).
type FlipHoldAnchor struct {
	SinceMs int64
	Source  string
}

// The anchor kinds, in the order the resolver applies them. The boot line
// prints THIS slice, so what the log says the hold is anchored to is what the
// resolver iterates — never a typed sentence.
const (
	FlipHoldAnchorBirth      = "session-plan-birth" // created_at of the chain's first version
	FlipHoldAnchorReplan     = "replan"             // a version authored by a death re-plan, owner re-read or owner reset (a deliberate fresh plan)
	FlipHoldAnchorBiasChange = "bias-change"        // a version whose bias.direction differs from the last biased version's
	FlipHoldAnchorFlip       = "flip"               // lifecycle → dormant with a dormant:flip: marker
	FlipHoldAnchorRearm      = "rearm"              // lifecycle → active with a rearmed: marker
	FlipHoldAnchorVersion    = "version(fallback)"  // chain unreadable: the version's own created_at (pre-fix semantics)
)

// FlipHoldAnchorKinds is the resolver's table, latest-wins.
var FlipHoldAnchorKinds = []string{FlipHoldAnchorBirth, FlipHoldAnchorReplan, FlipHoldAnchorBiasChange, FlipHoldAnchorFlip, FlipHoldAnchorRearm}

// FlipHoldAnchorLabel renders the kinds for the 🧬 boot line, read from the
// same table the resolver walks.
func FlipHoldAnchorLabel() string {
	return "latest of {" + strings.Join(FlipHoldAnchorKinds, "|") + "}, never a same-bias re-read version"
}

// flipHoldReplanTriggers are the authoring triggers that make a version a
// deliberate fresh plan rather than a wake re-read. Mirrors the store's
// spending classes (death_replan, owner_reread) plus the owner reset baseline.
var flipHoldReplanTriggers = map[string]bool{
	"death_replan": true,
	"owner_reread": true,
	"owner_reset":  true,
}

// PlanVersionFact is the slice of one stored plan version the resolver needs.
type PlanVersionFact struct {
	Version       int
	TriggerReason string
	BiasDirection string // doc bias.direction; "" when the version carries no bias (fail-closed / no-trade rows)
	CreatedAtMs   int64
}

// PlanTransitionFact is one plan_lifecycle_log row.
type PlanTransitionFact struct {
	Version int
	Event   string // the lifecycle it moved TO
	Reason  string // the marker the caller passed (dormant:flip:… / dormant:death:… / rearmed:…)
	AtMs    int64
}

// ResolveFlipHoldAnchor picks the hold anchor for the plan version `current`
// from the chain's versions (ascending) and its lifecycle transitions. Rows
// newer than `current` are ignored. An empty chain falls back to fallbackMs
// tagged FlipHoldAnchorVersion so the skip line says so.
func ResolveFlipHoldAnchor(versions []PlanVersionFact, transitions []PlanTransitionFact, current int, fallbackMs int64) FlipHoldAnchor {
	out := FlipHoldAnchor{SinceMs: fallbackMs, Source: FlipHoldAnchorVersion}
	seeded := false
	lastBias := ""
	for _, v := range versions {
		if current > 0 && v.Version > current {
			continue
		}
		if v.CreatedAtMs <= 0 {
			continue
		}
		if !seeded {
			out = FlipHoldAnchor{SinceMs: v.CreatedAtMs, Source: FlipHoldAnchorBirth}
			seeded = true
			lastBias = v.BiasDirection
			continue
		}
		switch {
		case flipHoldReplanTriggers[strings.TrimSpace(v.TriggerReason)]:
			out = FlipHoldAnchor{SinceMs: v.CreatedAtMs, Source: FlipHoldAnchorReplan}
		case v.BiasDirection != "" && lastBias != "" && v.BiasDirection != lastBias:
			out = FlipHoldAnchor{SinceMs: v.CreatedAtMs, Source: FlipHoldAnchorBiasChange}
		}
		if v.BiasDirection != "" {
			lastBias = v.BiasDirection
		}
	}
	if !seeded {
		return out
	}
	for _, tr := range transitions {
		if current > 0 && tr.Version > current {
			continue
		}
		if tr.AtMs <= out.SinceMs {
			continue // latest wins; the log is append-only so a later id is a later event
		}
		switch {
		case tr.Event == "dormant" && strings.HasPrefix(tr.Reason, "dormant:flip:"):
			out = FlipHoldAnchor{SinceMs: tr.AtMs, Source: FlipHoldAnchorFlip}
		case tr.Event == "active" && strings.HasPrefix(tr.Reason, "rearmed:"):
			out = FlipHoldAnchor{SinceMs: tr.AtMs, Source: FlipHoldAnchorRearm}
		}
	}
	return out
}
