package store

import (
	"path/filepath"
	"testing"
)

// PRE-ERA LABEL (data-integrity wave, D5) — E5.
//
// CountUnstampedClosed had NO era filter: it counted every CLOSED row with an
// empty plan_id, and the boot line then rendered them as
// "unstamped-closed=516 (pre-era history)" — which calls the same rows
// unstamped AND pre-era in one breath. They are pre-era: there was never a
// plan to stamp them with, and the converge deliberately leaves them alone.
//
// An unstamped row AT OR AFTER DayPlanEraStart is a live defect and must not be
// hidden inside a number that is 99% history.
func TestAttributionSplitsPreEraFromUnstamped(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "a.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ps := st.Position()

	eraMs := DayPlanEraStart.UnixMilli()
	mk := func(createdAt int64, planID string) {
		p := &TraderPosition{
			TraderID: "t", Symbol: "MNQ", Side: "SHORT", Quantity: 1,
			EntryPrice: 29285, EntryTime: createdAt, Status: "OPEN",
			PlanID: planID, CreatedAt: createdAt, UpdatedAt: createdAt,
		}
		if err := ps.Create(p); err != nil {
			t.Fatalf("create: %v", err)
		}
		if _, err := ps.ClosePosition(p.ID, 29115, "", -140, 0, "stop"); err != nil {
			t.Fatalf("close: %v", err)
		}
	}
	// three PRE-era rows with no plan — history, not a defect
	mk(eraMs-3*86_400_000, "")
	mk(eraMs-2*86_400_000, "")
	mk(eraMs-1*86_400_000, "")
	// one era row WITH a plan — neither bucket
	mk(eraMs+86_400_000, "2026-09-03:NY:t")

	preEra, err := ps.CountPreEraUnstamped()
	if err != nil {
		t.Fatalf("pre-era count: %v", err)
	}
	if preEra != 3 {
		t.Errorf("pre-era = %d, want 3", preEra)
	}

	unstamped, err := ps.CountUnstampedClosed()
	if err != nil {
		t.Fatalf("unstamped count: %v", err)
	}
	if unstamped != 0 {
		t.Errorf("unstamped-closed = %d, want 0 — pre-era rows are HISTORY, and counting them here hides a real defect inside a big honest number", unstamped)
	}

	// now a genuine defect: an era row that never got stamped
	mk(eraMs+2*86_400_000, "")
	unstamped, _ = ps.CountUnstampedClosed()
	if unstamped != 1 {
		t.Errorf("unstamped-closed = %d, want 1 — an era row with no plan_id IS the defect this counter exists for", unstamped)
	}

	line := st.AttributionBootLine()
	for _, want := range []string{"pre-era=3", "unstamped-closed=1"} {
		if !contains(line, want) {
			t.Errorf("boot line missing %q:\n  %s", want, line)
		}
	}
}
