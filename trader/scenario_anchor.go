// W1 — a scenario's price anchor, and the band the link compares against.
//
// PlanScenario names no level (kernel/plan_doc.go). The only prices it carries
// are the confirm reference and, when armed, the arm entry. Confirm wins,
// because that is the price the scenario is ABOUT; the arm entry is merely
// where it would be entered. A scenario with neither cannot be anchored, and
// that is a COUNTED outcome — never a 0 anchor, which would "match" a level at
// zero and quietly attach a touch to an unrelated setup.

package trader

import (
	"nofx/kernel"
	"nofx/store"
)

// scenarioAnchorFor reduces a scenario to the one price a level can be compared
// against. ok=false means UNANCHORABLE.
func scenarioAnchorFor(sc kernel.PlanScenario) (store.ScenarioAnchor, bool) {
	if sc.Confirm != nil && sc.Confirm.RefPrice > 0 {
		return store.ScenarioAnchor{ID: sc.ID, Price: sc.Confirm.RefPrice}, true
	}
	if sc.Arm != nil && sc.Arm.Enabled && sc.Arm.Entry > 0 {
		return store.ScenarioAnchor{ID: sc.ID, Price: sc.Arm.Entry}, true
	}
	return store.ScenarioAnchor{}, false
}

// scenarioAnchorsFrom derives every anchorable scenario in a plan, and reports
// how many could NOT be anchored so the caller can count them rather than lose
// them silently.
func scenarioAnchorsFrom(doc *kernel.PlanDoc) (anchors []store.ScenarioAnchor, unanchorable int) {
	if doc == nil {
		return nil, 0
	}
	for _, sc := range doc.Scenarios {
		if a, ok := scenarioAnchorFor(sc); ok {
			anchors = append(anchors, a)
			continue
		}
		unanchorable++
	}
	return anchors, unanchorable
}

// scenarioLinkBand is the MAP'S OWN cluster width — the same tolerance the
// level merge uses to decide that two references are the same reference
// (kernel.LevelClusterTicks, documented as "levels closer than this are the
// SAME reference on an intraday map, not separate seats").
//
// Read from there rather than restated, so the link's notion of "same level"
// cannot drift away from the map's. Two tolerances that agree today are two
// readers that happen to agree (class 97).
func scenarioLinkBand() float64 { return float64(kernel.LevelClusterTicks) * 0.25 }
