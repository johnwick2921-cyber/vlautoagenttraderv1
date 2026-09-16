// W1 item 1 — the touch → scenario link, as the heuristic it actually is.
//
// WHY THIS IS NOT IDENTITY, and cannot be made so in this wave. PlanScenario
// carries NO level reference: ID, Trigger, Condition, Direction, TargetChain,
// Invalid, Confirm, Quality, Fvg, Breakdown, ChainAfter, Arm — not one names a
// level, and Trigger/Invalid are free text the planner wrote. The codebase
// already states the consequence at kernel/scenario_state.go:19-22: "To evaluate
// a scenario we must first decide WHICH LEVEL it is about, and that resolution
// is a heuristic."
//
// Resolving it at authoring rather than at read does NOT make it a fact — it
// only performs the same guess earlier and stores it where it will be read as
// one. So the column is NEAREST-BY-PRICE, it always carries its basis, and it
// is NULL whenever the answer is ambiguous. The rule is the one already written
// beside the heuristic this replaces: a confidently-wrong answer is a NEW lie
// replacing the old one.
//
// The wave that turns this into identity is the scenario-schema change that
// makes the planner NAME its level by candidate id — a prompt/validator change,
// out of scope here, and filed as the next wave's basis.

package store

import "math"

// The basis vocabulary. Every value states HOW the link was made, or why it
// could not be — there is no silent NULL.
const (
	ScenarioLinkPriceProximity = "price_proximity"
	ScenarioLinkAmbiguous      = "unresolved:two_scenarios_within_band"
	ScenarioLinkOutsideBand    = "unresolved:nearest_outside_band"
	ScenarioLinkNoScenario     = "unresolved:no_scenario_at_seat"
)

// ScenarioAnchor is a scenario reduced to the only thing that can be compared
// against a level: the price the planner wrote for it. Deriving that price is
// the caller's job and is itself imperfect.
type ScenarioAnchor struct {
	ID    string
	Price float64
}

// ScenarioLink is the recorded result: a nearest scenario or NULL, always with
// a basis, and — when linked — the distance in BOTH points and Δ, so a reader
// can judge the link instead of trusting it.
type ScenarioLink struct {
	Scenario  *string
	Basis     string
	DistPts   *float64
	DistDelta *float64
}

// ResolveScenarioLink picks the nearest anchor within band, or returns NULL
// with the reason.
//
// AMBIGUITY IS NULL, not nearest-wins: two anchors inside the band means the
// record genuinely cannot say which scenario the touch belongs to, and a
// tie-break would manufacture certainty the data does not contain.
func ResolveScenarioLink(levelPrice float64, anchors []ScenarioAnchor, delta, band float64) ScenarioLink {
	if len(anchors) == 0 {
		return ScenarioLink{Basis: ScenarioLinkNoScenario}
	}

	best, bestDist := -1, math.Inf(1)
	inBand := 0
	for i, a := range anchors {
		d := math.Abs(a.Price - levelPrice)
		if band > 0 && d <= band {
			inBand++
		}
		if d < bestDist {
			best, bestDist = i, d
		}
	}

	if inBand > 1 {
		return ScenarioLink{Basis: ScenarioLinkAmbiguous}
	}
	if band > 0 && bestDist > band {
		return ScenarioLink{Basis: ScenarioLinkOutsideBand}
	}

	id := anchors[best].ID
	pts := bestDist
	out := ScenarioLink{Scenario: &id, Basis: ScenarioLinkPriceProximity, DistPts: &pts}

	// Δ is resolved per read and is 0 when the tape cannot supply it. A 0 Δ
	// leaves the Δ-distance NULL rather than dividing by zero or inventing a
	// ratio — the points distance still stands on its own.
	if delta > 0 {
		dd := pts / delta
		out.DistDelta = &dd
	}
	return out
}
