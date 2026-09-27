package store

import (
	"testing"
)

// ── W-EXEC-TRUTH W5 (builder A) — the Picture hand-off settle (D16/D17).

// handOffClaimed is the evaluator's claim at the store: an opportunity row,
// confirmed → place_pending with the synthetic claim id, unstamped.
func handOffClaimed(t *testing.T, st *Store, key, claim string) {
	t.Helper()
	if _, ok, err := st.PictureHtfClaim(&PictureHtfOpportunityDB{OppKey: key, TraderID: "t1", Account: "Sim101", Symbol: "MNQ", Direction: "long", Stage: "confirmed", WindowClose: 1790190010000}); err != nil || !ok {
		t.Fatalf("claim %s: ok=%v err=%v", key, ok, err)
	}
	if won, err := st.PictureHtfClaimSubmission(key, claim); err != nil || !won {
		t.Fatalf("claim submission %s: won=%v err=%v", key, won, err)
	}
}

func TestPictureHtfHandOffIsACompareAndSetOnTheClaim(t *testing.T) {
	st := newPictureHtfStore(t)
	handOffClaimed(t, st, "opp-a", "picture-htf-1")
	if moved, err := st.PictureHtfHandOff("opp-a", "picture-htf-OTHER", "x"); err != nil || moved {
		t.Fatalf("another owner's claim id must never settle the row: moved=%v err=%v", moved, err)
	}
	moved, err := st.PictureHtfHandOff("opp-a", "picture-htf-1", "Day Plan scenario P1 · p v1 · machine plan")
	if err != nil || !moved {
		t.Fatalf("the claim owner settles the row: moved=%v err=%v", moved, err)
	}
	row, _, _ := st.PictureHtfGet("opp-a")
	if row.Stage != PictureStagePlanned || row.StageReason != "Day Plan scenario P1 · p v1 · machine plan" || row.SignalID != "picture-htf-1" || row.SubmittedAt != 0 {
		t.Fatalf("settled row = %+v", row)
	}
	if again, _ := st.PictureHtfHandOff("opp-a", "picture-htf-1", "again"); again {
		t.Fatal("a settled row never settles twice")
	}
	// A row whose send STARTED (the legacy send's stamp) is never settled.
	handOffClaimed(t, st, "opp-b", "picture-htf-2")
	if err := st.PictureHtfStampSignal("opp-b", "picture-htf-2", "broker-uuid"); err != nil {
		t.Fatal(err)
	}
	if moved, _ := st.PictureHtfHandOff("opp-b", "broker-uuid", "x"); moved {
		t.Fatal("a stamped (sent) row must never be settled planned")
	}
}

// "planned" is outside every reader of an unresolved send: the installation
// gate / reconcile / latch (Recoverable*), the pending list, the send-started
// predicate — a planned row blocks nothing.
func TestPictureHtfPlannedIsNotAnUnresolvedSend(t *testing.T) {
	st := newPictureHtfStore(t)
	handOffClaimed(t, st, "opp-p", "picture-htf-9")
	if _, err := st.PictureHtfHandOff("opp-p", "picture-htf-9", "planned"); err != nil {
		t.Fatal(err)
	}
	if rows, _ := st.PictureHtfRecoverableAll(); len(rows) != 0 {
		t.Fatalf("RecoverableAll must not list a planned row: %+v", rows)
	}
	if rows, _ := st.PictureHtfRecoverableByTrader("t1"); len(rows) != 0 {
		t.Fatalf("RecoverableByTrader must not list a planned row: %+v", rows)
	}
	if rows, _ := st.PictureHtfPendingByTrader("t1"); len(rows) != 0 {
		t.Fatalf("PendingByTrader must not list a planned row: %+v", rows)
	}
	row, _, _ := st.PictureHtfGet("opp-p")
	if PictureSendStarted(*row) {
		t.Fatal("a planned row's send never started")
	}
	planned, err := st.PictureHtfPlannedByTrader("t1")
	if err != nil || len(planned) != 1 || planned[0].OppKey != "opp-p" {
		t.Fatalf("PlannedByTrader = %+v err=%v", planned, err)
	}
	none, err := st.PictureHtfPlannedByTrader("nobody")
	if err != nil || none == nil || len(none) != 0 {
		t.Fatalf("an empty computed list is [], never null: %#v err=%v", none, err)
	}
}

func TestPictureHtfHandOffPendingListsOnlyUnstampedClaims(t *testing.T) {
	st := newPictureHtfStore(t)
	handOffClaimed(t, st, "opp-open", "picture-htf-1")
	handOffClaimed(t, st, "opp-sent", "picture-htf-2")
	if err := st.PictureHtfStampSignal("opp-sent", "picture-htf-2", "broker-uuid"); err != nil {
		t.Fatal(err)
	}
	handOffClaimed(t, st, "opp-done", "picture-htf-3")
	if _, err := st.PictureHtfHandOff("opp-done", "picture-htf-3", "planned"); err != nil {
		t.Fatal(err)
	}
	rows, err := st.PictureHtfHandOffPendingByTrader("t1")
	if err != nil || len(rows) != 1 || rows[0].OppKey != "opp-open" {
		t.Fatalf("only the unstamped, unsettled claim is a pending hand-off: %+v err=%v", rows, err)
	}
}

func TestPictureHandOffRecordedForFindsOverlayAndMachinePlan(t *testing.T) {
	st := newPictureHtfStore(t)
	// A machine plan whose doc carries the scenario for ref-plan.
	mp := &PlanDB{PlanID: MakePlanIDForTrader("t1", "2026-09-14", "NY"), StrategyID: "t1", TradeDate: "2026-09-14", Session: "NY",
		TriggerReason: MachinePlanTriggerPrefix + "picture_htf", ModelID: "machine",
		Doc: `{"scenarios":[{"id":"P1","source":"picture","machine":{"rule":"h1_close_break","rule_ver":1,"ref":"ref-plan","eligible_from_ms":1,"eligible_until_ms":2}}]}`}
	if wrote, err := st.Plan().AppendPlanIfAbsent(mp); err != nil || !wrote {
		t.Fatalf("machine plan: %v %v", wrote, err)
	}
	// An AI version with a machine overlay for ref-ov.
	if _, err := st.Plan().AppendPlan(&PlanDB{PlanID: mp.PlanID, StrategyID: "t1", TradeDate: "2026-09-14", Session: "NY", TriggerReason: "NY_scheduled_read", Doc: `{"scenarios":[]}`}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Plan().AppendOverlay(&PlanOverlayDB{PlanID: mp.PlanID, PlanVersion: 2, OverlayID: "picture:ref-ov", Origin: "machine:picture_htf", Patch: `[]`}); err != nil {
		t.Fatal(err)
	}
	// An OWNER overlay that happens to carry the id is not a machine record.
	if _, err := st.Plan().AppendOverlay(&PlanOverlayDB{PlanID: mp.PlanID, PlanVersion: 2, OverlayID: "picture:ref-owner", Origin: "owner", Patch: `[]`}); err != nil {
		t.Fatal(err)
	}
	if rec, ok, err := st.PictureHandOffRecordedFor("t1", "ref-ov"); err != nil || !ok || rec.PlanVersion != 2 || rec.OverlayVersion != 1 {
		t.Fatalf("overlay record = %+v ok=%v err=%v", rec, ok, err)
	}
	if rec, ok, err := st.PictureHandOffRecordedFor("t1", "ref-plan"); err != nil || !ok || rec.PlanVersion != 1 || rec.OverlayVersion != 0 {
		t.Fatalf("machine plan record = %+v ok=%v err=%v", rec, ok, err)
	}
	for _, ref := range []string{"ref-owner", "ref-none", ""} {
		if _, ok, err := st.PictureHandOffRecordedFor("t1", ref); err != nil || ok {
			t.Fatalf("%q is not a machine record: ok=%v err=%v", ref, ok, err)
		}
	}
	if _, ok, _ := st.PictureHandOffRecordedFor("t2", "ref-ov"); ok {
		t.Fatal("another trader's record never counts")
	}
}
