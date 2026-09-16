package kernel

import "testing"

// Class 37 (2026-09-01) — the planner stream total deadline resolver is pinned:
// default 1200 (from the 2026-08-30..09-01 distribution), env override, junk →
// default, and the invariant total > idle (a total at or below idle would fire
// first and misreport the class).
func TestClass37PlannerStreamTotalDefaultAndOverride(t *testing.T) {
	t.Setenv("AI_PLAN_TOTAL_DEADLINE_SECS", "")
	t.Setenv("AI_PLAN_STREAM_IDLE_SECS", "")
	if got := PlannerStreamTotalSeconds(); got != 1200 {
		t.Fatalf("default total = %d, want 1200", got)
	}
	if got := PlannerStreamIdleSeconds(); got != 30 {
		t.Fatalf("default idle = %d, want 30 (unchanged by class 37)", got)
	}
	t.Setenv("AI_PLAN_TOTAL_DEADLINE_SECS", "900")
	if got := PlannerStreamTotalSeconds(); got != 900 {
		t.Fatalf("override total = %d, want 900", got)
	}
	for _, junk := range []string{"abc", "0", "-5", " "} {
		t.Setenv("AI_PLAN_TOTAL_DEADLINE_SECS", junk)
		if got := PlannerStreamTotalSeconds(); got != 1200 {
			t.Fatalf("junk %q → %d, want default 1200", junk, got)
		}
	}
}

func TestClass37PlannerStreamTotalAlwaysExceedsIdle(t *testing.T) {
	t.Setenv("AI_PLAN_STREAM_IDLE_SECS", "2000")
	t.Setenv("AI_PLAN_TOTAL_DEADLINE_SECS", "1200")
	if got := PlannerStreamTotalSeconds(); got != 2060 {
		t.Fatalf("total <= idle must resolve to idle+60, got %d", got)
	}
	t.Setenv("AI_PLAN_STREAM_IDLE_SECS", "30")
	t.Setenv("AI_PLAN_TOTAL_DEADLINE_SECS", "30")
	if got := PlannerStreamTotalSeconds(); got != 90 {
		t.Fatalf("total == idle must resolve to idle+60, got %d", got)
	}
	// The deadline arithmetic the report quotes: 600s was the old ceiling,
	// 1200 = 2× the observed max success (599.5s) and ≥ 65536 tok / 65 tok/s.
	if PlannerStreamTotalSeconds() < 31 {
		t.Fatalf("resolved total must always exceed idle")
	}
}
