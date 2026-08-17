package kernel

import (
	"encoding/json"
	"strings"
	"testing"
)

// ITEM 4 (2026-08-17) — OWNER EDITS SURVIVE A RE-PLAN.
//
// A re-plan silently stranded every owner edit: overlays are keyed (plan_id,
// plan_version) and every reader resolves against the LATEST version. These pin
// the contract that replaces that silence — and, above all, that no edit is ever
// applied to the WRONG element, which is worse than losing it.

func lv(price float64, label, grade, instr string) PlanLevel {
	return PlanLevel{Price: price, Label: label, Grade: grade, Instruction: instr}
}

// A sticky owner level carries into the new version, with its label and grade.
func TestOwnerLevelSurvivesAReplan(t *testing.T) {
	oldBase := PlanDoc{Levels: []PlanLevel{lv(30200, "ONH", "A", "fade")}}
	oldFinal := PlanDoc{Levels: []PlanLevel{
		lv(30200, "ONH", "A", "fade"),
		lv(30250, "my line", "A", "reclaim-long"), // the owner added this
	}}
	newDoc := PlanDoc{Levels: []PlanLevel{lv(30200, "ONH", "A", "fade"), lv(30180, "VWAP", "C", "magnet")}}

	res := CarryOwnerEdits(oldBase, oldFinal, newDoc, nil)

	if len(res.Carried) != 1 || res.Carried[0].Price != 30250 {
		t.Fatalf("the owner's level must carry, got %+v", res.Carried)
	}
	if res.Carried[0].Label != "my line" || res.Carried[0].Instruction != "reclaim-long" {
		t.Errorf("the level must carry with its own label and instruction: %+v", res.Carried[0])
	}

	// The patch must be an ADD by identity, never an index-anchored replace.
	var ops []map[string]any
	if err := json.Unmarshal([]byte(res.Patch), &ops); err != nil {
		t.Fatalf("patch is not valid RFC-6902: %v (%s)", err, res.Patch)
	}
	if len(ops) != 1 || ops[0]["op"] != "add" || ops[0]["path"] != "/levels/-" {
		t.Errorf("the carry must APPEND by identity, got %s", res.Patch)
	}
	// Applying it must actually put the level on the new doc.
	base, _ := json.Marshal(newDoc)
	out, errs := ApplyOverlayPatches(base, []string{res.Patch})
	if len(errs) > 0 {
		t.Fatalf("the carried patch does not apply: %v", errs)
	}
	var merged PlanDoc
	if err := json.Unmarshal(out, &merged); err != nil {
		t.Fatal(err)
	}
	if _, ok := hasLevelAt(merged.Levels, 30250); !ok {
		t.Error("after carrying, the owner's price is missing from plan_final")
	}
}

// THE SAFETY PROPERTY: an index-based patch is never replayed, so an owner edit
// can never land on a different level.
func TestCarryNeverMovesAnEditOntoAnotherLevel(t *testing.T) {
	// The owner edited what was index 0 in v1 — a level at 30200.
	oldBase := PlanDoc{Levels: []PlanLevel{lv(30200, "ONH", "B", "fade")}}
	oldFinal := PlanDoc{Levels: []PlanLevel{lv(30200, "ONH", "A", "fade")}} // grade B→A
	// v2 reorders: 30200 is now index 1, and index 0 is a DIFFERENT price.
	newDoc := PlanDoc{Levels: []PlanLevel{lv(29900, "PDL", "B", "target_support"), lv(30200, "ONH", "B", "fade")}}

	res := CarryOwnerEdits(oldBase, oldFinal, newDoc, []string{
		`[{"op":"replace","path":"/levels/0/grade","value":"A"}]`,
	})

	// The patch text targeted /levels/0, which in v2 is 29900. Nothing may touch it.
	if strings.Contains(res.Patch, "29900") {
		t.Fatalf("the carry re-pointed an index patch onto a different level: %s", res.Patch)
	}
	for _, u := range res.Uncarried {
		if strings.Contains(u.Path, "29900") {
			t.Fatalf("a review item names the wrong level: %+v", u)
		}
	}
	// The conflicting edit at the SAME price is surfaced, not force-applied.
	if len(res.Uncarried) == 0 {
		t.Fatal("an owner grade change the new plan disagrees with must be raised for review")
	}
	if !strings.Contains(res.Uncarried[0].Summary, "30200") {
		t.Errorf("the review item must name the owner's own level: %+v", res.Uncarried[0])
	}
}

// Structural edits have no price to anchor to → review, never silent drop.
func TestStructuralEditsGoToReview(t *testing.T) {
	doc := PlanDoc{Levels: []PlanLevel{lv(30200, "ONH", "A", "fade")}}
	res := CarryOwnerEdits(doc, doc, doc, []string{
		`[{"op":"replace","path":"/bias/direction","value":"short"}]`,
		`[{"op":"add","path":"/no_trade/-","value":"lunch"}]`,
	})
	if len(res.Uncarried) != 2 {
		t.Fatalf("both structural edits must surface, got %+v", res.Uncarried)
	}
	paths := res.Uncarried[0].Path + " " + res.Uncarried[1].Path
	if !strings.Contains(paths, "/bias/direction") || !strings.Contains(paths, "/no_trade/-") {
		t.Errorf("the review list must name the paths: %v", paths)
	}
	if res.Patch != "" {
		t.Errorf("structural edits must NOT be auto-applied, got patch %s", res.Patch)
	}
}

// A delete the planner has undone is a disagreement, not a silent re-delete.
func TestOwnerDeleteThePlannerUndidGoesToReview(t *testing.T) {
	oldBase := PlanDoc{Levels: []PlanLevel{lv(30200, "ONH", "A", "fade"), lv(30100, "junk", "C", "magnet")}}
	oldFinal := PlanDoc{Levels: []PlanLevel{lv(30200, "ONH", "A", "fade")}} // owner deleted 30100
	newDoc := PlanDoc{Levels: []PlanLevel{lv(30200, "ONH", "A", "fade"), lv(30100, "junk", "C", "magnet")}}

	res := CarryOwnerEdits(oldBase, oldFinal, newDoc, nil)
	if len(res.Uncarried) != 1 || res.Uncarried[0].Op != "remove" {
		t.Fatalf("the undone delete must surface for review, got %+v", res.Uncarried)
	}
	if !strings.Contains(res.Uncarried[0].Summary, "added it back") {
		t.Errorf("the review item must explain the disagreement: %q", res.Uncarried[0].Summary)
	}
}

// Nothing to do when the new plan already contains the owner's level exactly.
func TestNoDuplicateWhenTheNewPlanAlreadyAgrees(t *testing.T) {
	owner := lv(30250, "my line", "A", "reclaim-long")
	oldBase := PlanDoc{Levels: []PlanLevel{lv(30200, "ONH", "A", "fade")}}
	oldFinal := PlanDoc{Levels: []PlanLevel{lv(30200, "ONH", "A", "fade"), owner}}
	newDoc := PlanDoc{Levels: []PlanLevel{lv(30200, "ONH", "A", "fade"), owner}}

	res := CarryOwnerEdits(oldBase, oldFinal, newDoc, nil)
	if res.Patch != "" || len(res.Carried) != 0 {
		t.Errorf("a level the new plan already has must not be added twice: %s / %+v", res.Patch, res.Carried)
	}
	if len(res.Uncarried) != 0 {
		t.Errorf("agreement is not a review item: %+v", res.Uncarried)
	}
}

// Half a tick apart is the same level restated, not a new one.
func TestPriceIdentityToleratesSubTickNoise(t *testing.T) {
	if !samePrice(30200, 30200.1) {
		t.Error("prices within half a tick must be the same level")
	}
	if samePrice(30200, 30200.25) {
		t.Error("a full tick apart is a DIFFERENT level")
	}
}
