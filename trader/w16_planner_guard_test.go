package trader

import (
	"sync"
	"testing"

	"nofx/store"
)

// W16/R6 — one planner call per (trade_date, session) at a time.
//
// maybeRunSessionReads dedupes on "does a plan row already exist", but that check
// and the AppendPlan that satisfies it are separated by a full AI round trip. The
// exposure is CROSS-TRADER: plan identity is session-global, so two day-plan
// traders on the same symbol both saw "no plan yet" and both paid for a call.
func TestW16PlannerReadInFlightGuard(t *testing.T) {
	key := store.MakePlanID("2026-08-17", "NY")
	releasePlannerRead(key) // start clean regardless of test order

	// Two traders racing for the SAME session: exactly one wins.
	if !claimPlannerRead(key) {
		t.Fatal("the first claim must win")
	}
	if claimPlannerRead(key) {
		t.Fatal("a second claim for the same session must LOSE while the first is in flight")
	}

	// A DIFFERENT session is unaffected — the guard must not serialize the whole
	// planner, only same-session duplicates.
	other := store.MakePlanID("2026-08-17", "ASIA")
	releasePlannerRead(other)
	if !claimPlannerRead(other) {
		t.Fatal("a different session must be claimable concurrently")
	}
	releasePlannerRead(other)

	// Releasing lets the next read (e.g. the re-plan-on-death path) proceed.
	releasePlannerRead(key)
	if !claimPlannerRead(key) {
		t.Fatal("after release the session must be claimable again")
	}
	releasePlannerRead(key)
}

// Under real concurrency exactly ONE caller may proceed.
func TestW16PlannerGuardIsRaceSafe(t *testing.T) {
	key := store.MakePlanID("2026-08-18", "NY")
	releasePlannerRead(key)

	const racers = 32
	var wg sync.WaitGroup
	var mu sync.Mutex
	winners := 0
	wg.Add(racers)
	for i := 0; i < racers; i++ {
		go func() {
			defer wg.Done()
			if claimPlannerRead(key) {
				mu.Lock()
				winners++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if winners != 1 {
		t.Fatalf("%d goroutines claimed the same planner read; want exactly 1 (that is %d duplicate LLM calls)", winners, winners-1)
	}
	releasePlannerRead(key)
}
