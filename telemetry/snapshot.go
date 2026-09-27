package telemetry

import "fmt"

// Snapshot renders the process-lifetime, previously write-only counters into
// one read-only map for /api/telemetry (P2-8: every counter is either exposed
// or deleted — a counter that records into a void is dead wire).
//
// Absent means uncomputed, never a fabricated zero: fields are only set when
// the underlying counter exists.
func Snapshot() map[string]any {
	out := map[string]any{}

	fl, fs, fa := FarArmCounts()
	out["far_arms"] = map[string]any{
		"long":     fl,
		"short":    fs,
		"authored": fa,
	}

	out["repair_regression"] = RepairRegressionCount()
	out["shadowed_arm_refusals"] = ShadowedArmRefusalCount()
	out["research_snapshot"] = map[string]any{
		"drops": ResearchSnapshotDrops(),
		"rows":  ResearchSnapshotRows(),
	}
	out["ga4_send_failures"] = GA4Failures()
	out["weekly"] = fmt.Sprintf("%v", WeeklyCounterSnapshot())
	return out
}
