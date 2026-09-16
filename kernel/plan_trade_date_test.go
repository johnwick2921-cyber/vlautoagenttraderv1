package kernel

import (
	"testing"
	"time"
)

// S3 (mega-research 2026-08-26) — PlanTradeDateFor derives the trade date from
// the plan id prefix when it parses, else from the CME session-day of BirthMs.
func TestPlanTradeDateFor(t *testing.T) {
	if got := PlanTradeDateFor(nil); got != "" {
		t.Fatalf("nil plan must yield empty date, got %q", got)
	}
	// Plan id prefix parses ("2026-08-15:NY").
	ap := &ActivePlan{PlanID: "2026-08-15:NY", Session: "NY"}
	if got := PlanTradeDateFor(ap); got != "2026-08-15" {
		t.Fatalf("id-prefix date = %q, want 2026-08-15", got)
	}
	// Non-parsing prefix → BirthMs session-day fallback (17:00 CT roll).
	// BirthMs = 2026-08-26 12:00 UTC → 07:00 CT → session-day 2026-08-25.
	birth := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC).UnixMilli()
	ap2 := &ActivePlan{PlanID: "odd:id", BirthMs: birth}
	if got := PlanTradeDateFor(ap2); got != "2026-08-25" {
		t.Fatalf("birth fallback date = %q, want 2026-08-25", got)
	}
}
