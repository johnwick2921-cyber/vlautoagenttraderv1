package telemetry

import "sync/atomic"

// repairRegression counts planner repair attempts whose reject reason repeats
// the PREVIOUS attempt's reason (the whack-a-mole tell, planner-speed wave
// 3.4, 2026-08-31): the Sep-9 measure of whether repair beats re-author.
var repairRegression atomic.Int64

// IncRepairRegression bumps the whack-a-mole counter (called at the reject
// site when attempt N repeats attempt N-1's defect).
func IncRepairRegression(traderID string) {
	repairRegression.Add(1)
}

// RepairRegressionCount returns the running count (log/UI visibility).
func RepairRegressionCount() int64 {
	return repairRegression.Load()
}
