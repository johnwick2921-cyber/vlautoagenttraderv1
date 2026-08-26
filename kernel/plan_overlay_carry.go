package kernel

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// ITEM 4 (2026-08-17) — OWNER EDITS SURVIVE A RE-PLAN.
//
// Overlays are keyed (plan_id, plan_version) and every reader resolves them
// against the LATEST version, so appending a new version stranded every edit
// attached to the old one: the owner's levels quietly left the plan the bot was
// trading, with nothing carried, re-pointed or rebased.
//
// THE RULE THAT MAKES THIS SAFE: re-anchor by PRICE IDENTITY, never by array
// index. An RFC-6902 patch like `/levels/3` means "the fourth element", and the
// fourth element of a re-planned doc is a DIFFERENT level. Replaying such a patch
// would move an owner's edit onto the wrong price — worse than losing it. So the
// carry works on the RESOLVED level set, not on the patch text: whatever the
// owner's plan_final actually contained is what gets re-established.
//
// What carries automatically: owner ADDS and owner EDITS of levels, matched by
// price. What does NOT: anything structural (scenarios, bias, no-trade rules) and
// owner DELETES — a delete cannot be replayed safely because the planner may have
// re-added that level deliberately, and silently removing it again would override
// a fresh decision with a stale one. Those surface for review instead.

// carryPriceTolerance is how close two prices must be to count as the same level.
// MNQ ticks are 0.25, so anything inside half a tick is the same level restated.
const carryPriceTolerance = 0.125

// UncarriedEdit is one owner edit that could NOT be re-established on the new
// version, with the reason, so the card can ask for review instead of dropping it.
type UncarriedEdit struct {
	Op     string `json:"op"`
	Path   string `json:"path"`
	Reason string `json:"reason"`
	// Summary is the human sentence the card shows.
	Summary string `json:"summary"`
}

// CarryResult is the outcome of re-anchoring one version's owner edits onto its
// successor.
type CarryResult struct {
	// Patch is an RFC-6902 array that re-establishes the carried levels on the
	// new doc. Empty when there is nothing to carry.
	Patch string `json:"patch"`
	// Carried lists the levels that were re-established, for the log/alert.
	Carried []PlanLevel `json:"carried"`
	// Uncarried lists what needs the owner's eyes.
	Uncarried []UncarriedEdit `json:"uncarried"`
}

// levelName is the label, or a neutral word when the level has none.
func levelName(s string) string {
	if strings.TrimSpace(s) == "" {
		return "level"
	}
	return s
}

func samePrice(a, b float64) bool { return math.Abs(a-b) <= carryPriceTolerance }

func hasLevelAt(levels []PlanLevel, price float64) (PlanLevel, bool) {
	for _, l := range levels {
		if samePrice(l.Price, price) {
			return l, true
		}
	}
	return PlanLevel{}, false
}

// ownerLevels returns the levels the OWNER introduced or changed, by comparing
// the resolved plan_final against the planner's base doc. This is deliberately
// derived from the DOCS rather than parsed out of the patches: it is the only
// representation that cannot be mis-anchored.
func ownerLevels(base, final PlanDoc) []PlanLevel {
	var out []PlanLevel
	for _, fl := range final.Levels {
		bl, found := hasLevelAt(base.Levels, fl.Price)
		switch {
		case !found:
			out = append(out, fl) // the owner added this price
		case bl != fl:
			out = append(out, fl) // same price, owner changed label/grade/instruction
		}
	}
	return out
}

// CarryOwnerEdits re-establishes the previous version's owner edits on the new
// doc, and reports what could not be carried.
//
// oldBase   the planner's doc for the outgoing version (no overlays)
// oldFinal  that version's plan_final (overlays applied) — what the owner saw
// newDoc    the incoming version's doc
// patches   the outgoing version's raw overlay patches, used ONLY to detect
//
//	structural edits that the level comparison cannot see.
func CarryOwnerEdits(oldBase, oldFinal, newDoc PlanDoc, patches []string) CarryResult {
	res := CarryResult{}

	// 1) Levels the owner added or changed, re-anchored by price.
	var ops []map[string]any
	for _, ol := range ownerLevels(oldBase, oldFinal) {
		if existing, found := hasLevelAt(newDoc.Levels, ol.Price); found {
			if existing == ol {
				continue // already exactly right in the new plan — nothing to do
			}
			// The new plan has its OWN idea about this price. Re-applying the
			// owner's version would silently override a fresh decision, so ask.
			res.Uncarried = append(res.Uncarried, UncarriedEdit{
				Op: "replace", Path: fmt.Sprintf("/levels[%g]", ol.Price),
				Reason: "the new plan already has a different level at this price",
				Summary: fmt.Sprintf("your %s at %g differs from the new plan's %s — review",
					levelName(ol.Label), ol.Price, levelName(existing.Label)),
			})
			continue
		}
		ops = append(ops, map[string]any{
			"op": "add", "path": "/levels/-", "value": ol,
		})
		res.Carried = append(res.Carried, ol)
	}

	// 2) Levels the owner DELETED, and anything structural. Neither can be
	//    replayed safely, so both go to review rather than being dropped.
	for _, bl := range oldBase.Levels {
		if _, stillThere := hasLevelAt(oldFinal.Levels, bl.Price); stillThere {
			continue
		}
		if _, reAdded := hasLevelAt(newDoc.Levels, bl.Price); !reAdded {
			continue // the new plan does not carry it either — nothing to review
		}
		res.Uncarried = append(res.Uncarried, UncarriedEdit{
			Op: "remove", Path: fmt.Sprintf("/levels[%g]", bl.Price),
			Reason: "you removed this level and the new plan re-added it",
			Summary: fmt.Sprintf("you removed %s at %g; the new plan added it back — review",
				levelName(bl.Label), bl.Price),
		})
	}
	res.Uncarried = append(res.Uncarried, structuralEdits(patches)...)

	if len(ops) > 0 {
		if b, err := json.Marshal(ops); err == nil {
			res.Patch = string(b)
		}
	}
	return res
}

// structuralEdits finds patch operations that touch anything other than
// /levels. Those cannot be re-anchored by price because they have no price.
func structuralEdits(patches []string) []UncarriedEdit {
	var out []UncarriedEdit
	seen := map[string]bool{}
	for _, p := range patches {
		var ops []struct {
			Op   string `json:"op"`
			Path string `json:"path"`
		}
		if json.Unmarshal([]byte(p), &ops) != nil {
			continue
		}
		for _, op := range ops {
			if strings.HasPrefix(op.Path, "/levels") {
				continue // handled by price identity above
			}
			key := op.Op + " " + op.Path
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, UncarriedEdit{
				Op: op.Op, Path: op.Path,
				Reason:  "structural edits have no price to re-anchor to",
				Summary: fmt.Sprintf("your edit to %s could not carry into the new plan — review", op.Path),
			})
		}
	}
	return out
}
