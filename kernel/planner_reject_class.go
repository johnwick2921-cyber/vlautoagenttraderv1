package kernel

import "strings"

// PlannerRejectItemClass maps a planner reject reason to the WAVE PLANNER item
// class the plan file groups it under (planner-wave-plan-0924.md §0): A1
// no_provenance, A2 write-time feasibility, A3 schema/legality, A4 gap
// reachability, A5 identity/obstacle chain, A6 born-dead/flip-met. The SAME
// classifier feeds cmd/planner_replay's before/after table and the executor's
// 🧭 per-read line, so the two can never disagree (one canonicalizer, class
// 28). The mapping is pinned against the plan file's cited row ids in
// planner_reject_class_test.go.
func PlannerRejectItemClass(reason string) string {
	r := strings.ToLower(reason)
	switch {
	case strings.Contains(r, "born-dead"), strings.Contains(r, "born dead"),
		strings.Contains(r, "flip"), strings.Contains(r, "already met"):
		return "A6"
	case strings.Contains(r, "no_provenance"), strings.Contains(r, "provenance"):
		return "A1"
	case strings.Contains(r, "gap"), strings.Contains(r, "reach"):
		return "A4"
	case strings.Contains(r, "identity"), strings.Contains(r, "obstacle"):
		return "A5"
	case strings.HasPrefix(r, "write-time feasibility"), strings.Contains(r, "insufficient balance"):
		return "A2"
	default:
		return "A3" // schema / legality — everything else the write site refused
	}
}
