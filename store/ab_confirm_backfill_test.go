package store

import (
	"path/filepath"
	"testing"
)

// E8 BACKFILL (data-integrity wave, D6) — E6.
//
// Measured on the live store, direction taken from the PLAN (all 188 rows
// resolve to their plan scenario) and never inferred from geometry:
//
//	long   67 rows ·   0 with RR < −0.9 ·  0 with |MAE| > 1000
//	short 121 rows · 109 with RR < −0.9 · 46 with |MAE| > 1000 · 109 recomputable
//
// net_pnl is broken the same way and by the same cause:
//	40 rows < −1000  — the exit was the MIRRORED target, so
//	                   (−29418.62 − 29413.00) × 2 = −117 664, i.e. −(target+fill)×pv
//	 4 rows = 0 beside a RESOLVED outcome
//	88 rows = 0 on an OPEN outcome — CORRECT, nothing resolved; not touched
//	56 rows plausible
//
// THREE STAGES, in order, because stage 1 invalidates stage 2's input: the
// short close-rule compared a real close against a NEGATED ref
// (`b.Close > -ref`), which is true for every bar, so the FILL BAR itself was
// wrong — not just the arithmetic performed on it.

func abFixture(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "e8.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestAbConfirmBackfillRecomputesShortsAndMarksTheRest(t *testing.T) {
	st := abFixture(t)
	ab := st.AbConfirm()

	// row-166 shape: a short with complete inputs and poisoned outputs
	broken := &AbConfirmLogDB{
		TraderID: "t", PlanID: "p", Version: 1, Session: "NY", Scenario: "S1",
		Rule: "1x5m_close", FillPx: 29204.50, StopPx: 29226.00, TargetPx: 29132.50,
		RR: -0.998399808319285, MAE: 58409.25, MFE: 0, NetPnL: -1.0, Outcome: "open",
	}
	if err := ab.Upsert(broken); err != nil {
		t.Fatalf("seed broken: %v", err)
	}
	// a short with NO stop — unrecomputable, must never be guessed
	noInputs := &AbConfirmLogDB{
		TraderID: "t", PlanID: "p", Version: 1, Session: "NY", Scenario: "S2",
		Rule: "touch", FillPx: 29204.50, StopPx: 0, TargetPx: 29132.50, Outcome: "open",
	}
	if err := ab.Upsert(noInputs); err != nil {
		t.Fatalf("seed no-inputs: %v", err)
	}
	// a long — clean, and must come back byte-identical
	long := &AbConfirmLogDB{
		TraderID: "t", PlanID: "p", Version: 1, Session: "NY", Scenario: "S3",
		Rule: "touch", FillPx: 29204.00, StopPx: 29180.00, TargetPx: 29280.00,
		RR: 3.1666666666666665, MAE: 4, MFE: 20, NetPnL: 100, Outcome: "target",
	}
	if err := ab.Upsert(long); err != nil {
		t.Fatalf("seed long: %v", err)
	}

	dirOf := map[string]string{"S1": "short", "S2": "short", "S3": "long"}
	res, err := ab.BackfillShortRows(func(planID string, version int, scenario string) (string, bool) {
		d, ok := dirOf[scenario]
		return d, ok
	})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if res.Recomputed != 1 {
		t.Errorf("recomputed = %d, want 1 (only S1 has complete inputs)", res.Recomputed)
	}
	if res.Unrecomputable() != 1 {
		t.Errorf("unrecomputable = %d, want 1 (S2 has no stop)", res.Unrecomputable())
	}
	if res.LongsUntouched != 1 {
		t.Errorf("longs untouched = %d, want 1", res.LongsUntouched)
	}

	rows, err := ab.ListForPlan("p")
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	by := map[string]AbConfirmLogDB{}
	for _, r := range rows {
		by[r.Scenario] = r
	}

	// S1: risk 21.50, reward 72.00 → RR 3.348837…
	want := (29204.50 - 29132.50) / (29226.00 - 29204.50)
	if got := by["S1"].RR; got < want-1e-9 || got > want+1e-9 {
		t.Errorf("S1 RR = %.9f, want %.9f", got, want)
	}
	if by["S1"].MAE > 21.5 || by["S1"].MAE < 0 {
		t.Errorf("S1 MAE = %v, want within [0, 21.5]", by["S1"].MAE)
	}
	if by["S1"].Direction != "short" {
		t.Errorf("S1 direction = %q — the column exists so the next reader never infers it", by["S1"].Direction)
	}

	// S2: untouched numbers, marked, NEVER guessed
	if by["S2"].RR != 0 || by["S2"].MAE != 0 {
		t.Errorf("S2 must not be given numbers: rr=%v mae=%v", by["S2"].RR, by["S2"].MAE)
	}
	if by["S2"].Recompute != "unrecomputable:no-inputs" {
		t.Errorf("S2 recompute = %q, want \"unrecomputable:no-inputs\"", by["S2"].Recompute)
	}

	// S3: the long is byte-identical
	if by["S3"].RR != long.RR || by["S3"].MAE != long.MAE || by["S3"].NetPnL != long.NetPnL {
		t.Errorf("the long row moved: %+v", by["S3"])
	}

	// IDEMPOTENT
	res2, err := ab.BackfillShortRows(func(planID string, version int, scenario string) (string, bool) {
		d, ok := dirOf[scenario]
		return d, ok
	})
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if res2.Recomputed != 0 {
		t.Errorf("second run recomputed %d, want 0 — the backfill must be idempotent", res2.Recomputed)
	}
}

// A row whose direction the PLAN cannot supply is unrecomputable — direction is
// never inferred from geometry, because a short whose fill drifted past its
// target does not look like one (that mistake gave 55 shorts instead of 121).
func TestAbConfirmBackfillNeverInfersDirection(t *testing.T) {
	st := abFixture(t)
	ab := st.AbConfirm()
	if err := ab.Upsert(&AbConfirmLogDB{
		TraderID: "t", PlanID: "gone", Version: 9, Session: "NY", Scenario: "S9",
		Rule: "touch", FillPx: 29204.50, StopPx: 29226.00, TargetPx: 29132.50,
		RR: -0.9984, Outcome: "open",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	res, err := ab.BackfillShortRows(func(string, int, string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if res.Recomputed != 0 || res.Unrecomputable() != 1 {
		t.Errorf("no plan direction → unrecomputable; got recomputed=%d unrecomputable=%d", res.Recomputed, res.Unrecomputable())
	}
}
