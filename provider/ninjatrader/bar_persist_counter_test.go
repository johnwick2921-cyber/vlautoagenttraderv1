// Class 35 in waiting — "counters record, never infer".
//
// barPersistSummary used to Swap(0) both counters as it formatted them, so the
// act of reporting destroyed what was reported. The log line was the only
// consumer that could ever see a nonzero value; anything else reading them was
// wrong intermittently and silently, and that is what made
// TestFanOutClosesLastResortIsHonest look like a load flake for weeks.
//
// A counter records. Resetting is a separate, explicit verb.

package ninjatrader

import (
	"testing"
	"time"
)

func resetPersistCounters(t *testing.T) {
	t.Helper()
	persistDropped.Store(0)
	persistDroppedCloses.Store(0)
	persistFlushed.Store(0)
	persistDroppedReported.Store(0)
	persistDroppedClosesReported.Store(0)
	persistLastSum.Store(0)
}

// The defect, stated directly: reporting must not destroy the record.
func TestSummaryDoesNotDestroyTheCounters(t *testing.T) {
	resetPersistCounters(t)
	persistDropped.Add(3)
	persistDroppedCloses.Add(1)

	base := time.Unix(1_800_000_000, 0)
	barPersistSummaryAt(base) // first call arms the window

	if got := persistDropped.Load(); got != 3 {
		t.Errorf("persistDropped = %d after a summary, want 3 — the summary ate the count", got)
	}
	if got := persistDroppedCloses.Load(); got != 1 {
		t.Errorf("persistDroppedCloses = %d after a summary, want 1 — the summary ate the count", got)
	}
}

// A reader racing the 60s boundary must see the same value either way. This is
// the exact race that produced closes_dropped=0 in a branch that increments it.
func TestCounterIsStableAcrossTheSummaryBoundary(t *testing.T) {
	resetPersistCounters(t)
	base := time.Unix(1_800_000_000, 0)
	barPersistSummaryAt(base)

	persistDropped.Add(1)
	persistDroppedCloses.Add(1)

	// Cross the rate-limit boundary — previously this erased both.
	barPersistSummaryAt(base.Add(61 * time.Second))

	if d, c := persistDropped.Load(), persistDroppedCloses.Load(); d != 1 || c != 1 {
		t.Fatalf("after crossing the 60s boundary: dropped=%d closes=%d, want 1 and 1", d, c)
	}
}

// The log keeps its per-interval meaning without destroying anything: the delta
// comes from a separate reported-baseline, not from zeroing the counter.
func TestIntervalDeltaComesFromABaselineNotAReset(t *testing.T) {
	resetPersistCounters(t)
	base := time.Unix(1_800_000_000, 0)
	barPersistSummaryAt(base)

	persistDropped.Add(5)
	d1, c1 := persistIntervalDelta()
	if d1 != 5 || c1 != 0 {
		t.Fatalf("first interval delta = (%d,%d), want (5,0)", d1, c1)
	}

	barPersistSummaryAt(base.Add(61 * time.Second)) // publishes → advances baseline
	persistDropped.Add(2)

	d2, _ := persistIntervalDelta()
	if d2 != 2 {
		t.Errorf("second interval delta = %d, want 2 (not the cumulative 7)", d2)
	}
	if total := persistDropped.Load(); total != 7 {
		t.Errorf("cumulative = %d, want 7 — the total must keep accumulating", total)
	}
}

// Reset exists, but it is its own verb and nothing calls it implicitly.
func TestRolloverIsExplicit(t *testing.T) {
	resetPersistCounters(t)
	persistDropped.Add(4)
	persistDroppedCloses.Add(2)

	rollPersistCounters()

	if d, c := persistDropped.Load(), persistDroppedCloses.Load(); d != 0 || c != 0 {
		t.Fatalf("after an explicit rollover: dropped=%d closes=%d, want 0 and 0", d, c)
	}
	// And the baseline rolls with it, or the next delta would be negative.
	if d, _ := persistIntervalDelta(); d != 0 {
		t.Errorf("interval delta after rollover = %d, want 0", d)
	}
}
