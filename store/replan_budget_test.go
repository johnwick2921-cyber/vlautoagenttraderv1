package store

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// THE RE-PLAN BUDGET HAS ONE DEFINITION (2026-08-17).
//
// The owner set replan_cap = 4 and saw a row labelled "v6", which reads as "the
// cap didn't work". It did. The cap counts RE-PLANS: v1 is the session's first
// read and costs nothing, so cap N allows real versions v1…v(N+1), and the
// (N+1)th death writes a NO-TRADE marker — which consumes a version number
// because the plans table is append-only. cap=4 therefore ends at a row called
// v6, correctly.

func TestReplanBudgetCeiling(t *testing.T) {
	for _, tc := range []struct {
		cap          int
		lastRealVer  int // the highest version that is a REAL plan
		noTradeAtVer int // the version whose death writes the NO-TRADE marker
	}{
		{cap: 0, lastRealVer: 1, noTradeAtVer: 1}, // no re-plans at all
		{cap: 2, lastRealVer: 3, noTradeAtVer: 3},
		{cap: 4, lastRealVer: 5, noTradeAtVer: 5}, // the owner's ASIA session
	} {
		// Every version up to the ceiling may re-plan…
		for v := 1; v < tc.lastRealVer; v++ {
			if !MayReplan(v, tc.cap) {
				t.Errorf("cap=%d: v%d must be allowed to re-plan (only %d of %d re-plans used)",
					tc.cap, v, ReplansUsed(v), tc.cap)
			}
		}
		// …and the death of the last real version must NOT.
		if MayReplan(tc.noTradeAtVer, tc.cap) {
			t.Errorf("cap=%d: the death of v%d must yield NO-TRADE, not another plan",
				tc.cap, tc.noTradeAtVer)
		}
		if got := ReplansLeftFor(tc.noTradeAtVer, tc.cap); got != 0 {
			t.Errorf("cap=%d: at v%d the card must show 0 re-plans left, got %d",
				tc.cap, tc.noTradeAtVer, got)
		}
	}
}

// The card's "re-plans remaining" must equal what the enforcer will actually
// allow, at every step. These two numbers disagreeing is the exact class of bug
// that told the AI it had 0 re-plans while the card said 2.
func TestCardBudgetMatchesTheEnforcerAtEveryStep(t *testing.T) {
	for _, cap := range []int{0, 1, 2, 4} {
		for v := 1; v <= cap+3; v++ {
			left := ReplansLeftFor(v, cap)
			mayReplan := MayReplan(v, cap)
			if (left > 0) != mayReplan {
				t.Errorf("cap=%d v%d: card says %d left but the enforcer says mayReplan=%v",
					cap, v, left, mayReplan)
			}
		}
	}
}

func TestReplansUsedIgnoresNonsenseVersions(t *testing.T) {
	for _, v := range []int{0, -1, -99} {
		if got := ReplansUsed(v); got != 0 {
			t.Errorf("ReplansUsed(%d) = %d, want 0 — a bad version must never grant budget", v, got)
		}
	}
}

// NO LITERAL BUDGET MAY REAPPEAR IN THE PATH.
//
// installActivePlanProvider once carried `2 - (row.Version - 1)` and
// handler_plan.go carried `dp.ReplanCapFor(session)-(version-1)`. Both are gone;
// this fails if the shape comes back anywhere in the enforcing path.
func TestNoLiteralReplanBudgetInThePath(t *testing.T) {
	// `<something> - (<something>.Version - 1)` / `- (version - 1)` — the hand-rolled
	// budget arithmetic that must now go through ReplansLeftFor.
	pattern := regexp.MustCompile(`-\s*\(\s*\w*\.?[Vv]ersion\s*-\s*1\s*\)`)
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
				if strings.Contains(line, "//") && pattern.MatchString(line) {
					continue // a comment quoting the old bug is fine
				}
				if pattern.MatchString(line) {
					t.Errorf("%s:%d re-derives the re-plan budget by hand — use store.ReplansLeftFor / MayReplan:\n\t%s",
						path, i+1, strings.TrimSpace(line))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
