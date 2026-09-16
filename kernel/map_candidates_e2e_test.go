package kernel

import (
	"strings"
	"testing"
)

// W3 pins E2 (no-target), E3 (reachability order), E4 (the 09-03 projection
// case) and E5 (PWH/PWL from the daily source). Presentation and ordering only
// — no pin here asserts on a score.

// E2 — THE NO-TARGET PIN.
//
// A level with no opposing reference within the resolved minimum is NOT an
// entry candidate, but IS still in the map carrying target/obstacle. Today
// every seated level is offered, so a fade can be authored into a level whose
// first target sits inside the risk.
func TestE2_LevelWithNoPlausibleTargetIsNotAnEntryCandidate(t *testing.T) {
	price := 29650.00
	// 29680 and 29695 are 15 pt apart — under a 30 pt minimum. Neither can
	// offer the other as a target, so neither is an entry candidate.
	in := []ScoredLevel{
		mapFixtureLevel(KindONH, 29680.00, "ONH", "A"),
		mapFixtureLevel(KindPDH, 29695.00, "PDH", "A"),
	}
	got := BuildMapCandidates(in, price, 20.0, MapCandidateOpts{MinTargetDistance: 30.0})

	if len(got) != 2 {
		t.Fatalf("both references must remain on the map; got %d", len(got))
	}
	for _, c := range got {
		if c.EntryCandidate {
			t.Errorf("%.2f (%s) was offered as an entry candidate with its nearest opposing reference under the 30 pt minimum",
				c.Price, c.NamesLine())
		}
		if c.RefusedReason == "" {
			t.Errorf("%.2f refused candidacy without saying why — a refusal must be loud (A9)", c.Price)
		}
		if c.Role != MapRoleTarget && c.Role != MapRoleObstacle {
			t.Errorf("%.2f role = %q, want target or obstacle", c.Price, c.Role)
		}
	}
}

// E2b — the same map with a reachable target DOES produce entry candidates,
// so the pin above is proving the refusal and not a broken builder.
func TestE2b_LevelWithAPlausibleTargetIsAnEntryCandidate(t *testing.T) {
	price := 29650.00
	in := []ScoredLevel{
		mapFixtureLevel(KindONH, 29680.00, "ONH", "A"),
		mapFixtureLevel(KindPDH, 29760.00, "PDH", "A"), // 80 pt beyond — clears 30
	}
	got := BuildMapCandidates(in, price, 20.0, MapCandidateOpts{MinTargetDistance: 30.0})

	var entries int
	for _, c := range got {
		if c.EntryCandidate {
			entries++
			if c.RefusedReason != "" {
				t.Errorf("%.2f is an entry candidate yet carries a refusal reason %q", c.Price, c.RefusedReason)
			}
		}
	}
	if entries == 0 {
		t.Fatal("no entry candidate on a map whose references are 80 pt apart with a 30 pt minimum")
	}
}

// E2c — THE DEFAULT-MINIMUM PIN.
//
// Added because E9 mutation 2 SURVIVED: replacing the resolved default
// `minTarget = MinSLATRMult() * atr5m` with 0.0 changed nothing, because E2 and
// E2b both pass MinTargetDistance explicitly and never exercise the fallback.
// A mutation that passes is not evidence (A8), so this pin drives the DEFAULT
// path and asserts the resolved number reaches the refusal text.
//
// With MinSLATRMult()=1.5 and ATR5m=20 the stop floor is 30 pt, so references
// 15 pt apart cannot offer each other a target.
func TestE2c_DefaultMinimumIsTheResolvedStopFloor(t *testing.T) {
	price := 29650.00
	in := []ScoredLevel{
		mapFixtureLevel(KindONH, 29680.00, "ONH", "A"),
		mapFixtureLevel(KindPDH, 29695.00, "PDH", "A"),
	}
	// No MinTargetDistance — the builder must resolve the stop floor itself.
	got := BuildMapCandidates(in, price, 20.0, MapCandidateOpts{})

	if len(got) != 2 {
		t.Fatalf("want 2 candidates, got %d", len(got))
	}
	var sawResolved bool
	for _, c := range got {
		if c.EntryCandidate {
			t.Errorf("%.2f offered as an entry with only 15 pt to its neighbour and a 30 pt stop floor", c.Price)
		}
		// The refusal must name the RESOLVED minimum. If the default were
		// dropped to 0 the reason becomes "candidacy undecided" instead, which
		// is a different sentence — that is what kills the mutant.
		if strings.Contains(c.RefusedReason, "under the 30 pt minimum") {
			sawResolved = true
		}
		if strings.Contains(c.RefusedReason, "undecided") {
			t.Errorf("%.2f: the minimum failed to resolve from ATR5m; reason %q", c.Price, c.RefusedReason)
		}
	}
	if !sawResolved {
		t.Errorf("no refusal named the resolved 30 pt stop floor; reasons were: %q / %q",
			got[0].RefusedReason, got[1].RefusedReason)
	}
}

// E3 — THE REACHABILITY PIN.
//
// Among entry candidates the NEARER is listed first even when it scores lower,
// and the score is carried on both rows.
func TestE3_EntryShortlistOrdersByReachabilityNotScore(t *testing.T) {
	price := 29650.00
	near := mapFixtureLevel(KindONH, 29680.00, "ONH", "B") // 30 pt away
	near.Score = 0.80
	far := mapFixtureLevel(KindPDH, 29760.00, "PDH", "A") // 110 pt away
	far.Score = 1.60

	got := BuildMapCandidates([]ScoredLevel{far, near}, price, 20.0, MapCandidateOpts{MinTargetDistance: 30.0})
	if len(got) < 2 {
		t.Fatalf("want both candidates, got %d", len(got))
	}
	if got[0].Price != 29680.00 {
		t.Errorf("shortlist head = %.2f (score %.2f); the NEARER 29680.00 must rank first even though it scores lower",
			got[0].Price, got[0].Score)
	}
	// The score is carried and shown on every row (D4), it just ranks second.
	for _, c := range got {
		if c.Score <= 0 {
			t.Errorf("%.2f lost its score in the map view; the score is carried on every row", c.Price)
		}
	}
}

// E3b — reachability is measured in ATR5m when ATR is available, and the label
// says n/a rather than 0 when it is not (A24 / canon 49).
func TestE3b_DistanceInATRIsLabelledNotFabricated(t *testing.T) {
	price := 29650.00
	in := []ScoredLevel{mapFixtureLevel(KindONH, 29690.00, "ONH", "A")}

	withATR := BuildMapCandidates(in, price, 20.0, MapCandidateOpts{MinTargetDistance: 30.0})
	if !withATR[0].HasATR {
		t.Fatal("ATR was supplied but HasATR is false")
	}
	if got := withATR[0].DistanceATRLabel(); got != "2" {
		t.Errorf("40 pt at ATR5m 20 = 2 ATR; DistanceATRLabel() = %q", got)
	}

	noATR := BuildMapCandidates(in, price, 0, MapCandidateOpts{MinTargetDistance: 30.0})
	if noATR[0].HasATR {
		t.Error("no ATR was supplied yet HasATR is true")
	}
	if got := noATR[0].DistanceATRLabel(); got != "n/a" {
		t.Errorf("an uncomputed ATR distance must print n/a, never a number; got %q", got)
	}
}

// E4 — THE 09-03 PIN.
//
// On 2026-09-03 the market ran +483 pts; once price left the highest seated
// level there was no target and the fade book kept selling. With price ABOVE
// every mapped reference the map must carry at least one PROJECTION above it,
// labelled as such with its method — and no projection may be offered as an
// entry.
func TestE4_PriceBeyondTheMapCarriesAProjectionAboveIt(t *testing.T) {
	// The 09-03 shape: price has run past the highest reference.
	price := 29585.00
	mapped := []ScoredLevel{
		mapFixtureLevel(KindPDH, 29255.00, "PDH", "A"),
		mapFixtureLevel(KindONH, 29293.00, "ONH", "A"),
		mapFixtureLevel(KindRTHH, 29498.50, "RTH-H", "A"),
	}
	projections := []MapCandidate{
		{
			Price: 29750.00, Names: []string{"PWH"}, Kinds: []LevelKind{KindPWH},
			Grade: "A", Score: 1.0, MergedCount: 1, MergedCredit: 1,
			Projection: true, ProjectionMethod: "prior-week high (daily bars)",
		},
	}

	got := BuildMapWithProjections(mapped, projections, price, 20.0, MapCandidateOpts{MinTargetDistance: 30.0})

	var above []MapCandidate
	for _, c := range got {
		if c.Price > price {
			above = append(above, c)
		}
	}
	if len(above) == 0 {
		t.Fatal("price is beyond every mapped reference and the map carries NOTHING above it — the 09-03 case")
	}
	sawProjection := false
	for _, c := range above {
		if !c.Projection {
			continue
		}
		sawProjection = true
		if c.ProjectionMethod == "" {
			t.Errorf("projection at %.2f carries no method — a projection must say how it was derived", c.Price)
		}
		if c.EntryCandidate {
			t.Errorf("projection at %.2f was offered as an ENTRY candidate; projections are targets and obstacles only", c.Price)
		}
		if !strings.Contains(strings.ToLower(string(c.Role)), "target") &&
			!strings.Contains(strings.ToLower(string(c.Role)), "obstacle") {
			t.Errorf("projection at %.2f has role %q, want target or obstacle", c.Price, c.Role)
		}
	}
	if !sawProjection {
		t.Error("nothing above price is labelled a projection")
	}
	// D1 — the mapped references are all still there.
	if len(got) < len(mapped) {
		t.Errorf("map lost references when projections were added: %d < %d", len(got), len(mapped))
	}
}
