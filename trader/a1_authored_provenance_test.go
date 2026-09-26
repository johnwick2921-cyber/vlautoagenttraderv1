package trader

// WAVE PLANNER Lane A, item A1 — reference-less levels get usable zone
// provenance.
//
// The W3 ruling reads: provenance is RECORDED, not refused, when the authored
// economics.entry_zone passes the sanity checks (contains the level price).
// Today a NULL-WIDTH map line whose kind is NOT in referenceAnchorKinds (PDC,
// PDH, PDL — prior-day lines) falls through to
// `entry_zone_edges_or_provenance_unusable`, so NY v4 S2 (PDC) and S4 (PDH)
// were written with arm.enabled=false and geometry_no_provenance.
//
// Class-250 probe: THIS TEST IS THE CALLER — the fixture is the REAL NY v4 doc
// (2026-09-24, plan 8d5c8af5…, rows S2/S4) trimmed to its zones, identity
// levels, levels and the two scenarios, run through the production
// ComposeLevelFadeGeometryWith call site (the executor arm path / write-time
// feasibility compose). The mutation that proves it: re-adding the
// referenceAnchorKind gate on the authored path re-refuses both scenarios.
//
// Knob contract (wave RULING): geometry_reference_levels ON (the shipped
// default) admits with provenance "authored:<label>"; OFF reproduces today's
// refusal byte-identically.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nofx/kernel"
	"nofx/store"
)

func TestA1AuthoredEntryZoneAdmitsNullWidthMapLines(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "planner_a1_nyv4_fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc kernel.PlanDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	policy := store.StructuralStopPolicy{
		BufferPoints: 4.5, BufferKnown: true,
		CostPoints: 2, CostKnown: true,
		// MinRR 0.1: the R:R verdict is A2's item — this test pins the A1
		// ADMISSION (provenance authored:<label> + composed stop/target), so
		// the later min-RR gate must not mask it.
		MinRR: 0.1,
	}
	for _, id := range []string{"S2", "S4"} {
		var sc kernel.PlanScenario
		for _, s := range doc.Scenarios {
			if s.ID == id {
				sc = s
			}
		}
		if sc.ID == "" {
			t.Fatalf("fixture scenario %s missing", id)
		}
		leg := kernel.PlanArmLeg{Entry: sc.Arm.Entry}
		r := ComposeLevelFadeGeometryWith(&doc, sc, leg, policy, 20, 0.25, 2, true)
		if r.Reason != "" {
			t.Fatalf("%s: knob ON must ADMIT the authored-entry-zone null-width line, got refusal: %s (%s)",
				id, r.Reason, r.Detail)
		}
		if !strings.HasPrefix(r.StopSource, "authored:") {
			t.Fatalf("%s: provenance must be recorded as authored:<label>, got %q", id, r.StopSource)
		}
		if r.Stop == nil || r.Target == nil {
			t.Fatalf("%s: the admitted scenario must compose a stop and target, got stop=%v target=%v", id, r.Stop, r.Target)
		}
		// knob OFF = today's refusal, byte-identical
		rOff := ComposeLevelFadeGeometryWith(&doc, sc, leg, policy, 20, 0.25, 2, false)
		if rOff.Reason == "" {
			t.Fatalf("%s: knob OFF must refuse exactly as today", id)
		}
		t.Logf("%s: ADMITTED stop_source=%s stop=%.2f target=%.2f", id, r.StopSource, *r.Stop, *r.Target)
	}

	// CTO A1 fold 1 — WIDTH: a 60-pt authored zone on the same PDC line must
	// stay refused (day_plan.zone_max_pts shipped default 10.0), on EVERY
	// path — the authored band may not become frozen geometry past the cap.
	var wide kernel.PlanDoc
	if err := json.Unmarshal(b, &wide); err != nil {
		t.Fatal(err)
	}
	for i := range wide.Scenarios {
		if wide.Scenarios[i].ID == "S2" && wide.Scenarios[i].Economics != nil {
			wide.Scenarios[i].Economics.EntryZone = []float64{30626.5, 30686.5} // 60 pts
		}
	}
	for _, s := range wide.Scenarios {
		if s.ID != "S2" {
			continue
		}
		rw := ComposeLevelFadeGeometryWith(&wide, s, kernel.PlanArmLeg{Entry: s.Arm.Entry}, policy, 20, 0.25, 2, true)
		if rw.Reason == "" {
			t.Fatalf("a 60-pt authored zone (zone_max_pts=10) must stay REFUSED, got admit: %+v", rw)
		}
		t.Logf("S2 60-pt zone refused: %s (%s)", rw.Reason, rw.Detail)
	}
}
