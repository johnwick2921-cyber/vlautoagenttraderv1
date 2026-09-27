package trader

import (
	"testing"
	"time"
)

// ── W-ONE-BUTTON M2.1 (review F15 / N7) — a maintenance-hold refusal never
// starts the flip / death re-read launch clock. Both reads refused cutoff and
// stream-guard cases BEFORE the clock ("refusals never start this clock"), but
// the hold refused only inside the launched read — after flipRereadLaunchAt
// and lastPlannerWakeAt were set, so a short hold parked the retry for a whole
// wake_min_interval. The hold now refuses up front, like every other refusal.

func launchClockEntries(at *AutoTrader) int {
	n := 0
	at.flipRereadLaunchAt.Range(func(_, _ any) bool { n++; return true })
	return n
}

func TestFlipRereadRefusedByTheHoldNeverStartsTheClock(t *testing.T) {
	dir := withMaintenanceDir(t)
	at, _, client := realPathTrader(t, true, func(int, string) (string, error) { return validShortPlanJSON, nil })
	now := time.Date(2026, 8, 18, 14, 0, 0, 0, time.UTC)
	flipRereadTestNow(t, now)
	td := "2026-08-18"
	row := seedActivePlan(t, at, td, "NY", now.Add(-40*time.Minute), flipFixtureDoc())
	seedFlipBars(15500, 15470, 6*time.Minute, now)
	setHold(t, dir, "job-flip")
	wakeBefore := at.lastPlannerWakeAt
	before := gateBlocks(at.id, "maintenance_hold")

	at.maybeRereadAfterFlip(now, "NY", td, row, "flip-condition: 2x5m close below 15480.00 → bias short")
	defer drainReReads(t)              // CTO M4: join the async re-read before the seam resets
	time.Sleep(200 * time.Millisecond) // a launched read would be running by now

	if n := launchClockEntries(at); n != 0 {
		t.Fatalf("a hold refusal must not start the flip launch clock (%d entry)", n)
	}
	if !at.lastPlannerWakeAt.Equal(wakeBefore) {
		t.Fatal("a hold refusal must not move lastPlannerWakeAt")
	}
	if client.calls() != 0 {
		t.Fatalf("no AI call under the hold, got %d", client.calls())
	}
	if gateBlocks(at.id, "maintenance_hold") != before+1 {
		t.Fatal("the refusal must be counted as maintenance_hold")
	}
}

func TestDeathRereadRefusedByTheHoldNeverStartsTheClock(t *testing.T) {
	dir := withMaintenanceDir(t)
	at, _, client := realPathTrader(t, true, func(int, string) (string, error) { return validShortPlanJSON, nil })
	now := time.Date(2026, 8, 18, 14, 0, 0, 0, time.UTC)
	flipRereadTestNow(t, now)
	td := "2026-08-18"
	row := seedActivePlan(t, at, td, "NY", now.Add(-40*time.Minute), flipFixtureDoc())
	seedFlipBars(15500, 15470, 6*time.Minute, now)
	setHold(t, dir, "job-death")
	wakeBefore := at.lastPlannerWakeAt

	at.maybeRereadAfterDeath(now, "NY", td, row, "death-condition: close below 15400", 15395)
	defer drainReReads(t) // CTO M4: join the async re-read before the seam resets
	time.Sleep(200 * time.Millisecond)

	if n := launchClockEntries(at); n != 0 {
		t.Fatalf("a hold refusal must not start the death launch clock (%d entry)", n)
	}
	if !at.lastPlannerWakeAt.Equal(wakeBefore) {
		t.Fatal("a hold refusal must not move lastPlannerWakeAt")
	}
	if client.calls() != 0 {
		t.Fatalf("no AI call under the hold, got %d", client.calls())
	}
}
