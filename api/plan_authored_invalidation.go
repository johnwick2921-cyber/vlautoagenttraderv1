package api

import (
	"encoding/json"

	"nofx/kernel"
	"nofx/store"
)

// authoredInvalidationView (W-EXEC-TRUTH W2 A1/A2) is the plan card's
// "invalidation" line, READ FROM THE ROW: the policy the write site enforced,
// the read and publish clocks, and the 5m groups it judged. A row written
// before W2 (or a fail-closed NO-TRADE row) has no record: recorded=false and
// every field absent/null — the card says n/a, it never infers a policy for a
// row that did not store one. read_clock_ms null on a recorded row = the read
// clock was unknown at write (legacy facts-less writer).
type authoredInvalidationView struct {
	Recorded       bool    `json:"recorded"`
	Policy         string  `json:"policy,omitempty"`
	ReadClockMs    *int64  `json:"read_clock_ms"`
	PublishClockMs *int64  `json:"publish_clock_ms"`
	Groups         []int64 `json:"groups,omitempty"`
}

func authoredInvalidationFor(row *store.PlanDB) authoredInvalidationView {
	if row == nil || row.BornCheck == nil {
		return authoredInvalidationView{}
	}
	var bc kernel.BornCheck
	if json.Unmarshal([]byte(*row.BornCheck), &bc) != nil || bc.Policy == "" {
		return authoredInvalidationView{}
	}
	return authoredInvalidationView{Recorded: true, Policy: bc.Policy, ReadClockMs: row.ReadClockMs, PublishClockMs: row.PublishClockMs, Groups: bc.Groups}
}
