package trader

import (
	"strings"
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

// L12 at the call site: the hand-off's 🖼 lines name the opportunity with its
// account segment redacted — the full key stays in the plan doc and ledger.
func TestHandOffLinesNeverCarryTheAccountName(t *testing.T) {
	at, st := handOffTrader(t)
	now := handOffNow()
	at.markPictureRunEpoch(now)
	seedAIPlan(t, st, "active")
	ev := handOffEvidence(now, 1)
	if !strings.Contains(ev.OppKey, "|sim101|") {
		t.Fatalf("fixture: the key must carry an account segment: %s", ev.OppKey)
	}
	claimHandOff(t, st, ev)
	logs := captureTraderLog(t)
	if err := at.pictureHandOffAt(ev, now); err != nil {
		t.Fatal(err)
	}
	out := logs.String()
	if !strings.Contains(out, "🖼 picture → Day Plan scenario") {
		t.Fatalf("fixture: the hand-off must log its 🖼 line:\n%s", out)
	}
	if strings.Contains(strings.ToLower(out), "sim101") {
		t.Fatalf("a W5 log line carries the account name:\n%s", out)
	}
	if !strings.Contains(out, store.RedactPictureOppKey(ev.OppKey)) {
		t.Fatalf("the line names the opportunity through the redactor:\n%s", out)
	}
}

// W5 R1 (CTO review): the hand-off reads the latest plan version, and the
// planner appends the next version before the hand-off's overlay lands. The
// overlay must end on the NEW version (the one the executor reads), never on
// the superseded one where nothing would ever trade it.
func TestPictureHandOffNeverLandsOnASupersededVersion(t *testing.T) {
	at, st := handOffTrader(t)
	now := handOffNow()
	at.markPictureRunEpoch(now)
	v1 := seedAIPlan(t, st, "active")
	ev := handOffEvidence(now, 1)
	claimHandOff(t, st, ev)
	moved := false
	pictureHandOffBeforeAppendForTest = func() {
		if moved {
			return
		}
		moved = true
		if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: v1.PlanID, TradeDate: v1.TradeDate, Session: v1.Session, StrategyID: v1.StrategyID,
			TriggerReason: "NY_scheduled_read", Lifecycle: "active", ModelID: "m", Doc: validTraderPlanJSON}); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { pictureHandOffBeforeAppendForTest = nil })
	if err := at.pictureHandOffAt(ev, now); err != nil {
		t.Fatalf("the hand-off must re-read and record on the new version: %v", err)
	}
	if !moved {
		t.Fatal("fixture: the planner append never ran between the read and the append")
	}
	if ovs := listOverlays(t, st, v1.PlanID, 1); len(ovs) != 0 {
		t.Fatalf("the overlay landed on the SUPERSEDED v1 (the executor never reads it): %d row(s)", len(ovs))
	}
	if ovs := listOverlays(t, st, v1.PlanID, 2); len(ovs) != 1 {
		t.Fatalf("the overlay must land on v2, the latest: %d row(s)", len(ovs))
	}
}

func TestPictureHandOffBeforeAppendHookIsNilInProduction(t *testing.T) {
	if pictureHandOffBeforeAppendForTest != nil {
		t.Fatal("the R1 test seam must be nil in production")
	}
}
