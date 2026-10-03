package trader

import (
	"strings"
	"testing"
	"time"
)

// F1 — the planner-in-flight claims store the claim-started time and the
// installation gate's planner_in_flight leg renders it, at the production call
// site (InstallationGateStatus). The updater shows this text as the live
// blocker while it waits.
func TestPlannerLegReadsClaimStart(t *testing.T) {
	if !claimPlannerRead("f1-test-trader:2026-10-02:am") {
		t.Fatal("claim must succeed on an empty map")
	}
	if _, ok := plannerClaimStarts(); !ok {
		t.Fatal("plannerClaimStarts must see the claimed read")
	}
	g := InstallationGateStatus(map[string]*AutoTrader{}, nil)
	var detail string
	var pass bool
	found := false
	for _, l := range g.Legs {
		if l.Name == "planner_in_flight" {
			found, detail, pass = true, l.Detail, l.Pass
		}
	}
	if !found {
		t.Fatal("planner_in_flight leg missing from InstallationGateStatus")
	}
	if pass || !strings.Contains(detail, "waiting for the AI plan (started ") {
		t.Fatalf("leg must fail with the waiting text, got pass=%v detail=%q", pass, detail)
	}
	if _, err := time.Parse("15:04:05", strings.TrimPrefix(detail, "waiting for the AI plan (started ")[:8]); err != nil {
		t.Fatalf("detail must carry an hh:mm:ss start, got %q: %v", detail, err)
	}
	releasePlannerRead("f1-test-trader:2026-10-02:am")

	g2 := InstallationGateStatus(map[string]*AutoTrader{}, nil)
	for _, l := range g2.Legs {
		if l.Name == "planner_in_flight" && !l.Pass {
			t.Fatalf("leg must pass once the claim is released: %+v", l)
		}
	}
}

// The weekly claim stores a time the same way (F1 covers all four maps via
// plannerClaimStarts).
func TestWeeklyClaimStoresStart(t *testing.T) {
	if !claimWeeklyRead("weekly:f1-test-trader:2026-10-05") {
		t.Fatal("weekly claim must succeed on an empty map")
	}
	defer releaseWeeklyRead("weekly:f1-test-trader:2026-10-05")
	if _, ok := plannerClaimStarts(); !ok {
		t.Fatal("plannerClaimStarts must see the weekly claim")
	}
}
