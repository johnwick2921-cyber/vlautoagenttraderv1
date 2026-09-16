package store

import "testing"

// ── W2 D2 — THE STAMP, AND THE ONE RULE THAT MAKES IT USABLE ────────────────
//
// An episode's fade permission is fixed at its OPEN and never rewritten. E3
// compares episodes BY that value, so a permission that drifts with the day
// would make the comparison meaningless — and would do it silently, since the
// row would still look populated.

// E6 — FIXED AT OPEN. The second stamp does not overwrite the first, even when
// the day's state has changed underneath it.
func TestEpisodePermissionIsFixedAtOpen(t *testing.T) {
	st := NewTouchOutcomeStore(newArmedTestDB(t))
	id := openRow(t, st, "S1", 1_700_000_000_000)

	first := FadeStamp{Evaluated: true, Permitted: true, Exclusions: nil}
	if err := st.StampFadePermission(id, first); err != nil {
		t.Fatalf("first stamp: %v", err)
	}

	// the day turns: the IB breaks and holds, so a fresh evaluation would now
	// exclude. The episode opened before that and must keep its own verdict.
	second := FadeStamp{Evaluated: true, Permitted: false, Exclusions: []string{"ib_held"}}
	if err := st.StampFadePermission(id, second); err != nil {
		t.Fatalf("second stamp: %v", err)
	}

	got := fadeReadRow(t, st, id)
	if got.FadePermitted == nil {
		t.Fatalf("permission is NULL after stamping")
	}
	if !*got.FadePermitted {
		t.Errorf("permission changed after open: got permitted=false, want the value fixed at open (true)")
	}
	if got.FadeExclusions != nil && *got.FadeExclusions != "" {
		t.Errorf("exclusions rewritten after open: %q", *got.FadeExclusions)
	}
}

// NULL IS NOT PERMITTED. An unstamped episode must be distinguishable from a
// permitted one — that is why the column is a pointer (A24, C6).
func TestUnstampedEpisodeReadsNullNotPermitted(t *testing.T) {
	st := NewTouchOutcomeStore(newArmedTestDB(t))
	id := openRow(t, st, "S1", 1_700_000_000_000)
	got := fadeReadRow(t, st, id)
	if got.FadePermitted != nil {
		t.Errorf("an unevaluated episode must read NULL, got %v", *got.FadePermitted)
	}
}

// A NOT-EVALUATED VERDICT IS RECORDED AS NOT-EVALUATED, never as permitted.
func TestNotEvaluatedStampLeavesPermissionNull(t *testing.T) {
	st := NewTouchOutcomeStore(newArmedTestDB(t))
	id := openRow(t, st, "S1", 1_700_000_000_000)
	if err := st.StampFadePermission(id, FadeStamp{Evaluated: false}); err != nil {
		t.Fatalf("stamp: %v", err)
	}
	got := fadeReadRow(t, st, id)
	if got.FadePermitted != nil {
		t.Errorf("not-evaluated must leave permission NULL, got %v", *got.FadePermitted)
	}
}

// fadeReadRow re-reads one episode row by id.
func fadeReadRow(t *testing.T, st *TouchOutcomeStore, id uint) TouchOutcomeRow {
	t.Helper()
	var r TouchOutcomeRow
	if err := st.db.First(&r, id).Error; err != nil {
		t.Fatalf("read %d: %v", id, err)
	}
	return r
}
