package kernel

import (
	"strings"
	"testing"
	"time"
)

// ── ONE SETUP — the predicate, pure, ONE clock per test (A28/E12) ────────────

func osNow() time.Time { return time.Date(2026, 9, 11, 9, 0, 0, 0, CTLocation()) }

func osCand(id string, price, ref float64, grade string) MapCandidate {
	s := id
	return MapCandidate{ID: &s, Identity: PlanLevel{ID: &s, Price: price}, Price: price, Names: []string{id}, Grade: grade, Distance: price - ref}
}

func osScenario(id, cond string, levelID *string) PlanScenario {
	return PlanScenario{ID: id, Condition: cond, Direction: "long", LevelID: levelID, Confirm: &PlanConfirm{RefPrice: 100}}
}

func osFacts(price float64, cands ...MapCandidate) OneSetupLevelFacts {
	return OneSetupLevelFacts{Price: price, BandPts: 50, Candidates: cands}
}

func permitted() FadeVerdict { return FadeVerdict{Evaluated: true, Permitted: true} }
func excluded() FadeVerdict {
	return FadeVerdict{Evaluated: true, Permitted: false, Exclusions: []FadeExclusion{{Name: FadeExIBBrokenHeld}}}
}
func notEvaluated() FadeVerdict { return FadeVerdict{} }

// E1 — FIVE SCENARIOS, EXACTLY ONE ALLOWED. Every decline names all three verdicts.
func TestOneSetupE1FilterExactlyOneArms(t *testing.T) {
	now := osNow()
	best, lower := "lvl-best", "lvl-lower"
	cands := []MapCandidate{osCand(best, 105, 100, "A"), osCand(lower, 95, 100, "B")}
	cfg := OneSetupConfig{Enabled: true, MinGrade: "B"}
	cases := []struct {
		name string
		sc   PlanScenario
		ref  OneSetupLevelRef
		perm FadeVerdict
		want bool
		lvl  string
		play string
		pm   string
	}{
		{"best-level reject, permitted", osScenario("S1", "reject", &best), OneSetupLevelRef{Price: 105, ID: &best, Basis: LevelBasisCandidateID}, permitted(), true, "ok", "ok", "ok"},
		{"lower-graded reject, permitted", osScenario("S2", "reject", &lower), OneSetupLevelRef{Price: 95, ID: &lower, Basis: LevelBasisCandidateID}, permitted(), false, "level_not_best", "ok", "ok"},
		{"best-level sweep_reclaim", osScenario("S3", "sweep_reclaim", &best), OneSetupLevelRef{Price: 105, ID: &best, Basis: LevelBasisCandidateID}, permitted(), false, "ok", "play_not_reject:sweep_reclaim", "ok"},
		{"reject on an excluded day", osScenario("S4", "reject", &best), OneSetupLevelRef{Price: 105, ID: &best, Basis: LevelBasisCandidateID}, excluded(), false, "ok", "ok", "day_excluded(ib_held)"},
		{"permission NULL", osScenario("S5", "reject", &best), OneSetupLevelRef{Price: 105, ID: &best, Basis: LevelBasisCandidateID}, notEvaluated(), false, "ok", "ok", "not_evaluated"},
	}
	allowed := 0
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := osFacts(100, cands...)
			f.Scenario = c.ref
			v := OneSetupAllowsAt(now, c.sc, f, c.perm, cfg)
			if v.Allowed != c.want {
				t.Fatalf("allowed=%v want %v (%s)", v.Allowed, c.want, v.Reason)
			}
			if !strings.HasPrefix(v.Level, c.lvl) || v.Play != c.play || v.Permission != c.pm {
				t.Fatalf("verdicts level=%q play=%q permission=%q; want %q/%q/%q", v.Level, v.Play, v.Permission, c.lvl, c.play, c.pm)
			}
			if v.Level == "" || v.Play == "" || v.Permission == "" {
				t.Fatal("A9: a verdict with an empty leg — every declined scenario names all three")
			}
			if v.Allowed {
				allowed++
			}
		})
	}
	if allowed != 1 {
		t.Fatalf("E1: %d scenarios allowed, want exactly 1", allowed)
	}
}

// E3 — NULL NEVER PERMITS. A not-evaluated permission is declined, and reads as such.
func TestOneSetupE3NullNeverPermits(t *testing.T) {
	best := "lvl-best"
	f := osFacts(100, osCand(best, 105, 100, "A"))
	f.Scenario = OneSetupLevelRef{Price: 105, ID: &best, Basis: LevelBasisCandidateID}
	v := OneSetupAllowsAt(osNow(), osScenario("S1", "reject", &best), f, notEvaluated(), OneSetupConfig{Enabled: true, MinGrade: "B"})
	if v.Allowed || v.Permission != "not_evaluated" {
		t.Fatalf("NULL permission must decline as not_evaluated; got allowed=%v permission=%q", v.Allowed, v.Permission)
	}
}

// E4 — UNRESOLVED NEVER RESOLVES. basis=unresolved:* declines; nearest never wins.
func TestOneSetupE4UnresolvedNeverNearestWins(t *testing.T) {
	best := "lvl-best"
	f := osFacts(100, osCand(best, 105, 100, "A"))
	unknown := "lvl-unknown"
	f.Scenario = OneSetupLevelRef{Price: 105, ID: &unknown, Basis: "unresolved:unknown_level_id"}
	v := OneSetupAllowsAt(osNow(), osScenario("S1", "reject", &unknown), f, permitted(), OneSetupConfig{Enabled: true, MinGrade: "B"})
	if v.Allowed || !strings.HasPrefix(v.Level, "level_unresolved") {
		t.Fatalf("unresolved identity must decline as level_unresolved; got allowed=%v level=%q", v.Allowed, v.Level)
	}
	// A legacy scenario (no level_id) resolves by price proximity — that is the
	// dispatch's fallback, and it is labelled as such, never as an identity.
	f.Scenario = OneSetupLevelRef{Price: 105, Basis: LevelBasisPriceProximity}
	v = OneSetupAllowsAt(osNow(), osScenario("S1", "reject", nil), f, permitted(), OneSetupConfig{Enabled: true, MinGrade: "B"})
	if !v.Allowed {
		t.Fatalf("price-proximity basis on the best level must allow; got %s", v.Reason)
	}
}

// The best level: grade first (≥ min), distance second; band respected; any
// tf, any kind; a projection is never the best (it is never an entry).
func TestOneSetupBestCandidate(t *testing.T) {
	a, b, c, far, proj := "a", "b", "c", "far", "proj"
	cands := []MapCandidate{osCand(b, 98, 100, "B"), osCand(a, 110, 100, "A"), osCand(c, 101, 100, "C"), osCand(far, 170, 100, "A+"), osCand(proj, 102, 100, "A+")}
	cands[4].Projection = true
	got, ok := OneSetupBestCandidate(cands, 100, 50, "B")
	if !ok || *got.ID != a {
		t.Fatalf("best should be the nearest A within the band (a@110), got %v ok=%v", got.Names, ok)
	}
	// Min grade A+ leaves nothing eligible inside the band → no candidate.
	if _, ok := OneSetupBestCandidate(cands, 100, 50, "A+"); ok {
		t.Fatal("no candidate at/above min grade inside the band must report none, never the next best")
	}
	// Two A-grade candidates: the nearer wins.
	a2 := "a2"
	cands = append(cands, osCand(a2, 104, 100, "A"))
	got, _ = OneSetupBestCandidate(cands, 100, 50, "B")
	if *got.ID != a2 {
		t.Fatalf("grade tie → nearest wins; got %v", got.Names)
	}
}

// OFF: the predicate allows everything and says so — the seam then behaves
// byte-identically to today (E2 pins the seam; this pins the predicate).
func TestOneSetupOffAllows(t *testing.T) {
	best := "lvl-best"
	f := osFacts(100, osCand(best, 105, 100, "A"))
	f.Scenario = OneSetupLevelRef{Price: 105, ID: &best, Basis: LevelBasisCandidateID}
	v := OneSetupAllowsAt(osNow(), osScenario("S3", "sweep_reclaim", &best), f, notEvaluated(), OneSetupConfig{Enabled: false, MinGrade: "B"})
	if !v.Allowed || v.Reason != "one_setup=off" {
		t.Fatalf("OFF must allow with reason one_setup=off; got %+v", v)
	}
}

// D4 — rank: quality first, then doc order.
func TestOneSetupRank(t *testing.T) {
	s1 := PlanScenario{ID: "S1", Quality: "B"}
	s2 := PlanScenario{ID: "S2", Quality: "A"}
	s3 := PlanScenario{ID: "S3", Quality: "A"}
	order := OneSetupOrder([]PlanScenario{s1, s2, s3}, map[string]bool{"S1": true, "S2": true, "S3": true})
	if got := order[0].ID + order[1].ID + order[2].ID; got != "S2S3S1" {
		t.Fatalf("rank order want S2 S3 S1, got %s", got)
	}
	// Declined scenarios keep doc order after the allowed ones; nothing is dropped.
	order = OneSetupOrder([]PlanScenario{s1, s2, s3}, map[string]bool{"S1": true})
	if got := order[0].ID + order[1].ID + order[2].ID; got != "S1S2S3" {
		t.Fatalf("allowed first then doc order, got %s", got)
	}
}
