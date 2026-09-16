package store

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── SEAM ROWS ARE NEVER GRADED (owner ruling 2026-09-03) ─────────────────────
//
// Row 572 is an ARMED_TEST_SEAM experiment run against the live wire. It sat in
// the graded population carrying a D, and when the 15:02 boot recomputed it, W5
// promoted it to an A — a test harness scoring better than most real trades and
// counting toward the plan-adherence rate. Same family as checklist 29: a test
// row inside a production aggregate.
//
// The rule is enforced at the WRITE, not only at the caller, so a future grader
// cannot reintroduce it by forgetting to check.
func seamStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "seam.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func seedSeamRow(t *testing.T, st *Store, id int64, source string) {
	t.Helper()
	now := time.Now().UnixMilli()
	p := &TraderPosition{
		TraderID: "hoang", ExchangeID: "nt8", ExchangePositionID: "seam" + source + string(rune(id)),
		Symbol: "MNQ", Side: "SHORT", Quantity: 1, EntryQuantity: 1, EntryPrice: 29000,
		EntryTime: now, Leverage: 1, Status: "OPEN", Source: source, CreatedAt: now, UpdatedAt: now,
	}
	if err := st.Position().CreateOpenPosition(p); err != nil {
		t.Fatal(err)
	}
	if err := st.GormDB().Exec(`UPDATE trader_positions SET id=?, status='CLOSED' WHERE id=?`, id, p.ID).Error; err != nil {
		t.Fatal(err)
	}
}

// THE PIN: a seam row never RECEIVES a grade, however it is asked for.
func TestSeamRowNeverReceivesAGrade(t *testing.T) {
	st := seamStore(t)
	seedSeamRow(t, st, 572, CloseReasonTestSeam)
	seedSeamRow(t, st, 900, "system") // a real row, for contrast

	for _, g := range []string{"A", "B", "C", "D", "F"} {
		if err := st.Position().SetAdherence(572, g); err != nil {
			t.Fatalf("SetAdherence must not error on a seam row, got %v", err)
		}
	}
	var got string
	st.GormDB().Raw(`SELECT COALESCE(adherence_grade,'') FROM trader_positions WHERE id=572`).Scan(&got)
	if got != "" && got != SeamExcludedNote {
		t.Errorf("a seam row must never carry a grade, got %q after five attempts", got)
	}
	// A real row still grades normally — the rule must not swallow production.
	if err := st.Position().SetAdherence(900, "A"); err != nil {
		t.Fatal(err)
	}
	st.GormDB().Raw(`SELECT COALESCE(adherence_grade,'') FROM trader_positions WHERE id=900`).Scan(&got)
	if got != "A" {
		t.Errorf("a real row must still be graded, got %q", got)
	}
}

// The predicate covers the known seam source AND any future one by convention,
// so a new harness does not have to be remembered.
func TestSeamSourcePredicate(t *testing.T) {
	for _, s := range []string{CloseReasonTestSeam, "e7_farside_test", "seam_probe", "x_test_seam", "TEST_SEAM"} {
		if !IsSeamSource(s) {
			t.Errorf("%q must be recognised as a seam source", s)
		}
	}
	for _, s := range []string{"system", "reconcile", "armed_entry", ""} {
		if IsSeamSource(s) {
			t.Errorf("%q is a REAL source and must not be treated as a seam", s)
		}
	}
}

// The aggregate excludes seam rows and REPORTS the count — never a silent drop.
func TestAdherenceAggregateExcludesSeamWithCount(t *testing.T) {
	st := seamStore(t)
	seedSeamRow(t, st, 572, CloseReasonTestSeam)
	seedSeamRow(t, st, 901, "system")
	seedSeamRow(t, st, 902, "reconcile")
	_ = st.Position().SetAdherence(901, "A")
	_ = st.Position().SetAdherence(902, "B")

	dist, excluded, err := st.AdherenceDistributionExcludingSeam()
	if err != nil {
		t.Fatal(err)
	}
	if excluded != 1 {
		t.Errorf("the aggregate must COUNT what it excluded, got %d", excluded)
	}
	if strings.Contains(dist, "572") {
		t.Errorf("seam row leaked into the distribution: %s", dist)
	}
	if !strings.Contains(dist, "A=1") || !strings.Contains(dist, "B=1") {
		t.Errorf("real rows must still be counted: %s", dist)
	}
	t.Logf("distribution=%s excluded=%d", dist, excluded)
}

// The boot line reports a REAL count, read from the table.
func TestSeamBootLineIsCounted(t *testing.T) {
	st := seamStore(t)
	seedSeamRow(t, st, 572, CloseReasonTestSeam)
	seedSeamRow(t, st, 903, "system")
	line := st.SeamExclusionBootLine()
	// SUPERSEDED 2026-09-03: "excluded=1" read as "1 row was excluded" when it
	// counted rows MATCHING the predicate. On the 042ff360 boot it printed 3
	// while nothing had been excluded. Matched and stamped are now separate.
	if !strings.Contains(line, "matched=1") {
		t.Errorf("boot line must report what it matched: %s", line)
	}
	if !strings.Contains(line, "stamped=") {
		t.Errorf("boot line must report what it actually stamped: %s", line)
	}
	t.Logf("boot: %s", line)
}
