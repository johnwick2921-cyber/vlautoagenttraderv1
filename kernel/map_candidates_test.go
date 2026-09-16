package kernel

import (
	"reflect"
	"testing"
)

// W3 (candidates-not-entitlements, 2026-09-09) — the map-candidate view.
//
// The owner ruled on 2026-09-09 that this wave is PRESENTATION AND ORDERING
// ONLY: the merge produces a render-time view, the scored `conf` term is left
// exactly as it is, and kernel/testdata/stage_a_score_legacy.json stays
// byte-identical (E7). Every pin below therefore asserts on MapCandidate, never
// on ScoredLevel and never on a score.
//
// Clock (A28): these tests take their own `now` and never read the wall clock.

// mapFixtureLevel builds a scored level at a price with a kind + label, so a
// pin can state its own map without depending on the detector universe.
func mapFixtureLevel(kind LevelKind, price float64, label, grade string) ScoredLevel {
	return ScoredLevel{
		DetectedLevel: DetectedLevel{Kind: kind, Price: price, Label: label},
		Grade:         grade,
		Fresh:         "fresh",
		Score:         1.0,
	}
}

// E1 — THE MERGE PIN.
//
// Three references within one zone-width are ONE candidate carrying all three
// names, and they contribute ONE merged credit — not three. Today there is no
// merged view at all, so the map hands the model three separate rows for one
// price and the confluence column reads them as three confirmations.
func TestE1_ThreeReferencesInOneZoneWidthMergeToOneCandidate(t *testing.T) {
	price := 29650.00
	// 29657.38 / 29657.50 / 29658.00 — a 0.62 pt spread, well inside the
	// 3.00 pt collapse width (LevelClusterTicks=12 × 0.25).
	in := []ScoredLevel{
		mapFixtureLevel(KindSupply, 29657.38, "Supply·1h", "A"),
		mapFixtureLevel(KindPDC, 29657.50, "PDC", "A"),
		mapFixtureLevel(KindVWAP, 29658.00, "VWAP+1σ", "B"),
	}

	got := BuildMapCandidates(in, price, 20.0, MapCandidateOpts{})

	if len(got) != 1 {
		t.Fatalf("three references within one zone-width must merge to ONE candidate; got %d: %+v", len(got), got)
	}
	c := got[0]
	if len(c.Names) != 3 {
		t.Fatalf("merged candidate must carry ALL THREE names; got %d: %v", len(c.Names), c.Names)
	}
	for _, want := range []string{"Supply·1h", "PDC", "VWAP+1σ"} {
		found := false
		for _, n := range c.Names {
			if n == want {
				found = true
			}
		}
		if !found {
			t.Errorf("merged candidate lost the name %q; carries %v", want, c.Names)
		}
	}
	// ONE credit, not three: the merged candidate counts once.
	if c.MergedCount != 3 {
		t.Errorf("MergedCount = %d, want 3 (the number of references folded in)", c.MergedCount)
	}
	if c.MergedCredit != 1 {
		t.Errorf("MergedCredit = %d, want 1 — three names for one price count ONCE", c.MergedCredit)
	}
	// The display string the card and the model both get.
	if disp := c.NamesLine(); disp != "Supply·1h · PDC · VWAP+1σ" {
		t.Errorf("NamesLine() = %q, want %q", disp, "Supply·1h · PDC · VWAP+1σ")
	}
}

// E7 GUARD (in this file so it fails beside the merge it protects) — the merge
// must never write back into the scored input. The parity golden marshals the
// full []ScoredLevel; a mutation here would break it byte-for-byte.
func TestE7_BuildMapCandidatesDoesNotMutateItsInput(t *testing.T) {
	in := []ScoredLevel{
		mapFixtureLevel(KindSupply, 29657.38, "Supply·1h", "A"),
		mapFixtureLevel(KindPDC, 29657.50, "PDC", "A"),
	}
	before := make([]ScoredLevel, len(in))
	copy(before, in)

	_ = BuildMapCandidates(in, 29650.00, 20.0, MapCandidateOpts{})

	for i := range in {
		// DeepEqual, not !=: DetectedLevel now carries a slice (CollapsedNames,
		// fix/collapse-keeps-names) and a struct with a slice is not comparable
		// with ==. The assertion's meaning is unchanged — the input is untouched.
		if !reflect.DeepEqual(in[i], before[i]) {
			t.Fatalf("BuildMapCandidates mutated its input at %d:\n got  %+v\n want %+v", i, in[i], before[i])
		}
	}
}

// E6 — EXCLUSION IS NOT INVALIDATION (class 93, AUDIT-CHECKLIST.md:2776).
//
// A candidate refused entry candidacy is still IN the map, carrying a role.
// Nothing is ever dropped from the returned set.
func TestE6_RefusedCandidacyStaysInTheMap(t *testing.T) {
	price := 29650.00
	// Two references 1.5 pt apart (merge) and one far away with nothing beyond
	// it — the far one has no plausible target and cannot be an entry, but it
	// MUST still be returned.
	in := []ScoredLevel{
		mapFixtureLevel(KindPDC, 29655.00, "PDC", "A"),
		mapFixtureLevel(KindVWAP, 29656.00, "VWAP+1σ", "A"),
		mapFixtureLevel(KindONH, 29900.00, "ONH", "A"),
	}
	got := BuildMapCandidates(in, price, 20.0, MapCandidateOpts{MinTargetDistance: 30.0})

	if len(got) != 2 {
		t.Fatalf("map must keep every merged candidate; got %d, want 2", len(got))
	}
	var far *MapCandidate
	for i := range got {
		if got[i].Price >= 29899.0 {
			far = &got[i]
		}
	}
	if far == nil {
		t.Fatal("the far candidate was DROPPED from the map — exclusion is not invalidation (class 93)")
	}
	if far.Role == "" {
		t.Error("a candidate excluded from the entry shortlist must still carry a map role")
	}
}
