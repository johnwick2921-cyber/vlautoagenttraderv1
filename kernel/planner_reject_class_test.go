package kernel

import "testing"

// The classifier is pinned against the REAL reject reasons of rows 339..370
// (DS-106-copy0924.db, the 09-24 evidence set) and must reproduce
// planner-wave-plan-0924.md §0's cited grouping exactly.
func TestPlannerRejectItemClassMatchesPlanFileTable(t *testing.T) {
	cases := []struct {
		id     int64
		reason string // the stored reject_reason, verbatim (head)
		want   string
	}{
		{340, "write-time feasibility: S2 would be refused at arm — geometry: no_provenance", "A1"},
		{350, "write-time feasibility: S3 would be refused at arm — geometry: no_provenance (cost model)", "A1"},
		{342, "write-time feasibility: S2 would be refused at arm — geometry: net_nonpositive (gain=2.0000 cost=2.0000 net=0.0000)", "A2"},
		{347, "write-time feasibility: S1 would be refused at arm — geometry: rr (gain=6.5000 risk=4.5000 rr=1.444444 min=2.000000)", "A2"},
		{368, "write-time feasibility: S2 would be refused at arm — market_in_zone at the zone's far edge", "A2"},
		{346, "scenario[0].confirm.side \"at\" invalid (above|below)", "A3"},
		{356, "arm on S1 needs EXACTLY 2 legs (split contract), got 1", "A3"},
		{361, "arm on S1: entry policy planned_order is not legal on breakdown_continue (planned_order is for reject, fvg_entry and sweep_reclaim leg 0; use market_in_zone)", "A3"},
		{357, "gap reachability: price below PDL — a short trigger ≤ price is required", "A4"},
		{348, "scenario level identity: the named level is not in the frozen map", "A5"},
		{351, "obstacle chain: S4 omits the seated level between entry and target", "A5"},
		{339, "born-dead authored scenario: S1: authored condition \"2x5m cl\"", "A6"},
		{341, "flip condition already met during the read", "A6"},
	}
	for _, c := range cases {
		if got := PlannerRejectItemClass(c.reason); got != c.want {
			t.Errorf("row %d: classified %q, want %q (reason: %q)", c.id, got, c.want, c.reason)
		}
	}
}
