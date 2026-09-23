package trader

import (
	"testing"
	"time"

	"nofx/store"
)

// ── W-EXEC-TRUTH W5 — the seams the lane joined, proven at the call site ────
//
//   - the D17 interrupted-hand-off sweep runs from the executor's own pass
//     (the hook is declared beside the executor and bound by the plan source);
//   - a PLACED Picture scenario carried into the next plan version (the
//     re-append, CTO 1790191033566) never places again: one opportunity, one
//     order — the executor and the ledger's source pin together.

func TestThePassRunsTheInterruptedHandOffSweep(t *testing.T) {
	at, st := handOffTrader(t)
	now := handOffNow()
	ev := handOffEvidence(now, 1)
	claimHandOff(t, st, ev) // place_pending under the claim, never recorded, never sent
	at.maybeManageArmedOrdersAt(nil, time.UnixMilli(ev.WindowCloseMs+1))
	got := pictureRow(t, st, ev.OppKey)
	if got.Stage != "refused" || got.StageReason != "hand-off interrupted — never sent" {
		t.Fatalf("an ordinary pass must run the D17 sweep: stage %q reason %q", got.Stage, got.StageReason)
	}
}

func TestAPlacedPictureScenarioCarriedIntoTheNextVersionNeverPlacesAgain(t *testing.T) {
	for _, tc := range []struct {
		name  string
		after func(r *zoneRig, ref string)
	}{
		{"resting at the broker", func(*zoneRig, string) {}},
		{"filled", func(r *zoneRig, ref string) {
			if err := r.st.GormDB().Model(&store.ArmedOrderDB{}).Where("source_ref = ?", ref).
				Updates(map[string]any{"state": store.StateFilled, "fill_price": 100.5, "fill_quantity": 1}).Error; err != nil {
				r.t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ref := "opp-carry-" + tc.name
			r, epoch := newPicRig(t, "w5-carry", nil)
			sc := picScenario("P1", ref, r.now, epoch, picDefault)
			picPlan(r, sc) // v1 carries P1
			picPass(r, 0, 100.25)
			if sigs, _ := r.drain(); len(sigs) != 1 {
				t.Fatalf("fixture: v1 places P1 exactly once, got %d", len(sigs))
			}
			tc.after(r, ref)
			picPlan(r, sc) // v2 carries the SAME P1 (the re-append's value)
			picPass(r, 5*time.Second, 100.25)
			picPass(r, 10*time.Second, 100.25)
			if sigs, _ := r.drain(); len(sigs) != 0 {
				t.Fatalf("a placed opportunity carried into v2 must never place again: %d frame(s) %+v", len(sigs), sigs)
			}
			n := 0
			for _, row := range r.rows() {
				if row.SourceRef == ref {
					n++
				}
			}
			if n != 1 {
				t.Fatalf("one opportunity, one ledger row across versions: got %d", n)
			}
		})
	}
}
