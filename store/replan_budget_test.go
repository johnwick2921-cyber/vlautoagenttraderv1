package store

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// THE RE-PLAN BUDGET HAS ONE DEFINITION — and it is RECORDED, not inferred.
//
// CLASS 35 (2026-09-01): the budget used to be version − baseline, so every
// appended row (wake reads, dormant flips, fail-closed markers) counted as a
// spent re-plan. Now the two consuming paths record each spend in system_config
// under the chain's baseline; everything else is free. cap N still means N
// death re-plans / owner re-reads, and a NO-TRADE marker still consumes a
// version number (the plans table is append-only) — it just no longer counts.

func newBudgetStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "b.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestReplanBudgetLeftAndMay(t *testing.T) {
	for _, tc := range []struct {
		used, cap, wantLeft int
		wantMay             bool
	}{
		{0, 0, 0, false}, // no re-plans at all: the first death sits out
		{0, 2, 2, true},
		{1, 2, 1, true},
		{2, 2, 0, false},
		{0, 4, 4, true}, // today's LONDON chain: six rows, nothing spent
		{3, 4, 1, true},
		{4, 4, 0, false},
		{9, 4, 0, false}, // over-spent (should be impossible) still clamps at 0
	} {
		b := ReplanBudget{Used: tc.used, Cap: tc.cap}
		if got := b.Left(); got != tc.wantLeft {
			t.Errorf("used=%d cap=%d: Left() = %d, want %d", tc.used, tc.cap, got, tc.wantLeft)
		}
		if got := b.May(); got != tc.wantMay {
			t.Errorf("used=%d cap=%d: May() = %v, want %v", tc.used, tc.cap, got, tc.wantMay)
		}
		// The card's number and the enforcer's verdict can never disagree.
		if (b.Left() > 0) != b.May() {
			t.Errorf("used=%d cap=%d: card says %d left but the enforcer says %v", tc.used, tc.cap, b.Left(), b.May())
		}
	}
}

// A spend is an EVENT the store records; the budget reads it back.
func TestSpendReplanRecordsEachSpend(t *testing.T) {
	st := newBudgetStore(t)
	const tid, date, sess = "tid-1", "2026-09-01", "LONDON"
	if b := GetReplanBudget(st, tid, date, sess, 4); b.Used != 0 || b.Left() != 4 || !b.May() {
		t.Fatalf("fresh chain must have the full budget, got %+v", b)
	}
	for i := 1; i <= 4; i++ {
		used, err := SpendReplan(st, tid, date, sess)
		if err != nil {
			t.Fatalf("spend %d: %v", i, err)
		}
		if used != i {
			t.Fatalf("spend %d returned used=%d", i, used)
		}
		b := GetReplanBudget(st, tid, date, sess, 4)
		if b.Used != i || b.Left() != 4-i {
			t.Fatalf("after %d spends: %+v", i, b)
		}
	}
	if b := GetReplanBudget(st, tid, date, sess, 4); b.May() {
		t.Fatalf("four spends against cap 4 must exhaust the budget, got %+v", b)
	}
	// Per-(trader, date, session) scoping: a sibling session is untouched.
	if b := GetReplanBudget(st, tid, date, "NY", 4); b.Used != 0 {
		t.Errorf("NY must keep its own counter, got %+v", b)
	}
	if b := GetReplanBudget(st, "tid-2", date, sess, 4); b.Used != 0 {
		t.Errorf("another trader must keep its own counter, got %+v", b)
	}
}

// The owner reset moves the baseline; the counter is keyed UNDER the baseline,
// so the new chain starts at 0 without the reset path knowing about counters.
func TestResetBaselineRearmsTheRecordedBudget(t *testing.T) {
	st := newBudgetStore(t)
	const tid, date, sess = "tid-1", "2026-09-01", "NY"
	for i := 0; i < 4; i++ {
		if _, err := SpendReplan(st, tid, date, sess); err != nil {
			t.Fatal(err)
		}
	}
	if b := GetReplanBudget(st, tid, date, sess, 4); b.May() || b.Baseline != 1 {
		t.Fatalf("exhausted original chain expected, got %+v", b)
	}
	if err := SetResetBaseline(st, tid, date, sess, 7); err != nil {
		t.Fatal(err)
	}
	b := GetReplanBudget(st, tid, date, sess, 4)
	if b.Baseline != 7 || b.Used != 0 || b.Left() != 4 || !b.May() {
		t.Fatalf("reset chain must start with the full budget, got %+v", b)
	}
	// The abandoned chain's spends are still on record under baseline 1.
	if raw, _ := st.GetSystemConfig(ReplansUsedKey(tid, date, sess, 1)); strings.TrimSpace(raw) != "4" {
		t.Errorf("original chain counter must survive the reset, got %q", raw)
	}
}

// A malformed counter can never panic the loop or silently exhaust a session:
// it reads as 0 (full budget) and is logged loudly.
func TestMalformedReplanCounterReadsAsZero(t *testing.T) {
	st := newBudgetStore(t)
	const tid, date, sess = "tid-1", "2026-09-01", "NY"
	for _, raw := range []string{"not-a-number", "-3", "  "} {
		if err := st.SetSystemConfig(ReplansUsedKey(tid, date, sess, 1), raw); err != nil {
			t.Fatal(err)
		}
		if b := GetReplanBudget(st, tid, date, sess, 4); b.Used != 0 {
			t.Errorf("raw %q: Used = %d, want 0", raw, b.Used)
		}
	}
	// A nil store is a full budget, never a nil deref.
	if b := GetReplanBudget(nil, tid, date, sess, 2); b.Used != 0 || b.Cap != 2 {
		t.Errorf("nil store: %+v", b)
	}
	if _, err := SpendReplan(nil, tid, date, sess); err == nil {
		t.Error("spending against a nil store must error, not pretend")
	}
}

// Exactly two trigger classes spend. Everything else — the session's scheduled
// read, level_event / structure_mss wakes (fast-market is a reasoning MODE of
// those wakes, not a class), owner_reset, dormant/rearm markers — is free.
func TestTriggerSpendsReplanClasses(t *testing.T) {
	for _, spend := range []string{TriggerDeathReplan, TriggerOwnerReread, " owner_reread "} {
		if !TriggerSpendsReplan(spend) {
			t.Errorf("%q must spend", spend)
		}
	}
	for _, free := range []string{"", "NY_scheduled_read", "LONDON_scheduled_read", "level_event", "structure_mss", "owner_reset", "planner_fail_closed", "replans_exhausted", "dormant:flip:x", "rearmed:x", "sunday_weekly_read"} {
		if TriggerSpendsReplan(free) {
			t.Errorf("%q must NOT spend", free)
		}
	}
}

// NO INFERRED BUDGET MAY REAPPEAR IN THE PATH.
//
// installActivePlanProvider once carried `2 - (row.Version - 1)`, handler_plan.go
// carried `dp.ReplanCapFor(session)-(version-1)`, and store carried
// `version - baseline` (class 35). All three are gone; this fails if any shape
// comes back anywhere in the enforcing path.
func TestNoLiteralReplanBudgetInThePath(t *testing.T) {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`-\s*\(\s*\w*\.?[Vv]ersion\s*-\s*1\s*\)`),
		regexp.MustCompile(`[Vv]ersion\s*-\s*baseline`),
		regexp.MustCompile(`ReplansUsedFrom|MayReplanFrom|ReplansLeftFrom|ReplansLeftFor\(`),
	}
	roots := []string{"../trader", "../api", "../kernel", "."}
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil // tests may quote the old shape to document it
			}
			b, rErr := os.ReadFile(path)
			if rErr != nil {
				return nil
			}
			for i, line := range strings.Split(string(b), "\n") {
				code := line
				if idx := strings.Index(line, "//"); idx >= 0 {
					code = line[:idx] // a comment quoting the old bug is fine
				}
				for _, p := range patterns {
					if p.MatchString(code) {
						t.Errorf("%s:%d re-derives the re-plan budget — use store.GetReplanBudget / SpendReplan:\n\t%s",
							path, i+1, strings.TrimSpace(line))
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
