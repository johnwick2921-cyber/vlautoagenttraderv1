package api

import (
	"errors"

	"nofx/store"
)

// errOverlayRevisionMoved — the writer-side check saw an overlay land for the
// plan version between the caller's validation and the append.
var errOverlayRevisionMoved = errors.New("overlay revision moved under the write")

// planOverlayRevision is what the client viewed when it drafted an overlay
// edit: the plan row and the highest overlay version of that row. The overlay
// door refuses the edit when the server's current row/overlay state has moved
// past it — a stale draft must never silently overwrite a newer edit.
// (F17, WAVE 117 PR-D, ports #117 09e24a08.)
type planOverlayRevision struct {
	PlanID         string `json:"expected_plan_id"`
	PlanVersion    int    `json:"expected_plan_version"`
	OverlayVersion *int   `json:"expected_overlay_version"`
}

// latestOverlayRevision returns the highest OverlayVersion in rows (0 when
// there are none) — the value handlePlanToday hands the card so it can echo
// it back as expected_overlay_version.
func latestOverlayRevision(rows []*store.PlanOverlayDB) int {
	latest := 0
	for _, row := range rows {
		if row.OverlayVersion > latest {
			latest = row.OverlayVersion
		}
	}
	return latest
}
