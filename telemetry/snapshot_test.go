package telemetry

import "testing"

// TestSnapshotReadsCounters pins P2-8: every previously write-only counter is
// readable through Snapshot() (the /api/telemetry payload).
func TestSnapshotReadsCounters(t *testing.T) {
	IncFarArm("long")
	IncFarArm("short")
	IncArmAuthored()
	IncRepairRegression("t-snap")
	IncShadowedArmRefusal()
	IncResearchSnapshotDrop()
	AddResearchSnapshotRows(3)
	IncGA4Failure()

	snap := Snapshot()
	for _, key := range []string{"far_arms", "repair_regression", "shadowed_arm_refusals", "research_snapshot", "ga4_send_failures", "weekly"} {
		if _, ok := snap[key]; !ok {
			t.Fatalf("Snapshot() missing %q", key)
		}
	}
	fa := snap["far_arms"].(map[string]any)
	if fa["long"].(int64) < 1 || fa["short"].(int64) < 1 || fa["authored"].(int64) < 1 {
		t.Fatalf("far_arms not read from live counters: %+v", fa)
	}
	if snap["ga4_send_failures"].(int64) < 1 {
		t.Fatalf("ga4 counter not read")
	}
}
