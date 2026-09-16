package kernel

import (
	"strings"
	"testing"
)

// ── LABEL PROVENANCE × W3's MERGED NAMES (owner dispatch 2026-09-11) ─────────
//
// W3 renders a merged reference as EVERY member's label, strongest first,
// joined by " · " (MapCandidate.NamesLine → "RTH-H · EQL·4h · EQH·15m"), and
// the model copies that line as the level's label. The provenance validator
// (P0.4-H) split the label at the first "·" and compared "RTH-H " — trailing
// space — against the machine table's "RTH-H": every merged label read as a
// re-invented anchor and burned a repair round on every read (LONDON and NY,
// 2026-09-11). RULE: a merged label passes when its PRIMARY component matches
// the machine table; a wrong primary still fails; the LONDON v1 phantom
// ("PDH" over a zone row) still fails.
func TestProvenanceAcceptsMergedLabelWithMatchingPrimary(t *testing.T) {
	machine := map[float64]string{29275.25: "RTH-H", 29038.00: "PDL", 29297.75: "Supply·1h"}
	doc := &PlanDoc{Levels: []PlanLevel{
		{Price: 29275.25, Label: "RTH-H · EQL·4h · EQH·15m"},          // the live rejection, NY 09-11 attempt 1
		{Price: 29038.00, Label: "PDL · SWG-L·15m · EQL·1h · EQL·15m"}, // the live rejection, second level
	}}
	if mis := MislabeledStructuralLevels(doc, machine); len(mis) != 0 {
		t.Fatalf("a merged label whose PRIMARY matches the machine table must pass; got: %s", strings.Join(mis, "; "))
	}
}

func TestProvenanceStillRejectsWrongPrimary(t *testing.T) {
	machine := map[float64]string{29275.25: "RTH-H", 29297.75: "Supply·1h"}
	doc := &PlanDoc{Levels: []PlanLevel{
		{Price: 29275.25, Label: "EQH·15m · RTH-H"}, // the structural anchor demoted behind another — wrong primary
		{Price: 29297.75, Label: "PDH"},              // LONDON v1's phantom: a structural label over a zone row
	}}
	mis := MislabeledStructuralLevels(doc, machine)
	if len(mis) != 2 {
		t.Fatalf("a wrong primary and a phantom anchor must both still fail; got %d: %s", len(mis), strings.Join(mis, "; "))
	}
}

func TestStructuralPrefixTrimsTheMergedSeparator(t *testing.T) {
	for label, want := range map[string]string{
		"RTH-H · EQL·4h · EQH·15m": "RTH-H",
		"PDL · SWG-L·15m":          "PDL",
		"EQH·4h":                   "EQH",
		"RTH-H":                    "RTH-H",
		"Supply·1h":                "",
		"EQH·15m · RTH-H":          "EQH",
	} {
		if got := structuralPrefix(label); got != want {
			t.Fatalf("structuralPrefix(%q) = %q, want %q", label, got, want)
		}
	}
}
