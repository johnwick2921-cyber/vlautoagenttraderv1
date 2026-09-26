package trader

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"nofx/store"
)

// W1 — a per-session setting means what the owner set, whatever the case of
// the session name it was saved under (class 28: one canonicalizer, called
// where the value enters). store.DayPlanConfig.SessionOverride is that
// canonicalizer: EqualFold, first match. Two planner lookups compared the
// name with == and kept the LAST match, so an override saved as "ny" was
// silently ignored for the NY session, and a duplicated session resolved
// differently from every other path.

func TestResolveSessionPlanCfgFindsTheOverrideWhateverItsCase(t *testing.T) {
	a := "A"
	dp := &store.DayPlanConfig{Sessions: []store.DayPlanSessionOverride{{Session: "ny", MinGrade: &a}}}
	if _, _, _, got, _ := resolveSessionPlanCfg(dp, "NY"); got != "A" {
		t.Fatalf("a min-grade override saved under %q must apply to NY; got %q", "ny", got)
	}
}

func TestResolveSessionPlanCfgTakesTheSameOverrideAsSessionOverride(t *testing.T) {
	a, b := "A", "B"
	dp := &store.DayPlanConfig{Sessions: []store.DayPlanSessionOverride{
		{Session: "NY", MinGrade: &a},
		{Session: "NY", MinGrade: &b},
	}}
	want := *dp.SessionOverride("NY").MinGrade
	if _, _, _, got, _ := resolveSessionPlanCfg(dp, "NY"); got != want {
		t.Fatalf("the planner must resolve the same override every other path does (%q); got %q", want, got)
	}
}

// No production code compares a day-plan session name by hand: every lookup
// goes through SessionOverride / at.sessionOverride.
func TestNoHandRolledCaseSensitiveSessionLookup(t *testing.T) {
	re := regexp.MustCompile(`\.Session\s*==\s*session\b`)
	for _, dir := range []string{".", "../kernel", "../api", "../store"} {
		files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			for i, line := range strings.Split(string(b), "\n") {
				if re.MatchString(line) {
					t.Errorf("%s:%d compares a session name by hand — use SessionOverride (EqualFold): %s", f, i+1, strings.TrimSpace(line))
				}
			}
		}
	}
}
