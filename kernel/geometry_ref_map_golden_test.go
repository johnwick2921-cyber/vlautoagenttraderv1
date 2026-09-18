package kernel

import (
	"strings"
	"testing"
)

// TestGeometryRefIDsMapRenderOnOffParity (F5) — the shipped default (knob ON,
// ref ids rendered in the map block) has no golden without this: a fixture with
// an ONH line lacking a formation close renders id=NULL OFF and id=ref|… ON,
// and the REST of the prompt is byte-identical between the two (the knob only
// changes the id token — OFF stays the pinned behaviour).
func TestGeometryRefIDsMapRenderOnOffParity(t *testing.T) {
	lvl := ScoredLevel{DetectedLevel: DetectedLevel{Kind: "ONH", Price: 29897, Lo: 29897, Hi: 29897, Label: "ONH", OriginDate: "2026-09-17", TF: "1m"}}
	build := func(on bool) string {
		return BuildPlannerPrompt(PlannerInput{
			TradeDate: "2026-09-18", Session: SessionNY, Price: 29890, DATR: 120, ATR5m: 20,
			Levels: []ScoredLevel{lvl}, GeometryRefIDs: on,
		})
	}
	off := build(false)
	on := build(true)

	if !strings.Contains(off, "id=NULL") {
		t.Fatalf("OFF must render the NULL id for the ONH line (today's map); got:\n%s", off)
	}
	if !strings.Contains(on, "id=ref|") {
		t.Fatalf("ON must render the stable ref| id for the ONH line; got:\n%s", on)
	}

	// Byte-identical except the id token: replacing the ref id with NULL must
	// reproduce the OFF render exactly.
	idTok := ""
	if i := strings.Index(on, "id=ref|"); i >= 0 {
		if j := strings.IndexAny(on[i:], " \n"); j >= 0 {
			idTok = on[i : i+j]
		}
	}
	if idTok == "" {
		t.Fatal("could not extract the ref id token from the ON render")
	}
	replaced := strings.ReplaceAll(on, idTok, "id=NULL")
	if replaced != off {
		t.Fatalf("the map block must be byte-identical apart from the id token\n--- OFF ---\n%s\n--- ON with id=NULL ---\n%s", off, replaced)
	}
}
