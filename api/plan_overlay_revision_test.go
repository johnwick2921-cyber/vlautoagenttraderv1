package api

import (
	"encoding/json"
	"testing"
	"time"

	"nofx/store"
)

// F17 (WAVE 117 PR-D, ports #117 09e24a08) — an owner overlay must bind to the
// plan revision the user VIEWED (expected_plan_id + expected_plan_version +
// expected_overlay_version). A stale draft must be refused, never applied on
// top of a newer edit.

func TestOverlayEditRevisionRefusesStaleDraft(t *testing.T) {
	s, _ := newPicturePlanGateServer(t, `{"day_plan":{"plan_enabled":true}}`)
	id := store.MakePlanIDForTrader(ppgTrader, "2026-09-14", "NY")
	doc := `{"reasoning":"review","bias":{"direction":"neutral","conviction":"low","flip_condition":"n/a"},"levels":[],"scenarios":[{"id":"S1","trigger":"t","condition":"reclaim","direction":"long","target_chain":[15600],"invalid":"i","quality":"B"}],"no_trade":[],"death_condition":"dead"}`
	seed := func() {
		t.Helper()
		if _, err := s.store.Plan().AppendPlan(&store.PlanDB{PlanID: id, StrategyID: ppgTrader, TradeDate: "2026-09-14", Session: "NY", Lifecycle: "active", Doc: doc}); err != nil {
			t.Fatal(err)
		}
	}
	seed()
	zero := 0
	expected := &planOverlayRevision{PlanID: id, PlanVersion: 1, OverlayVersion: &zero}
	// A draft viewed at the CURRENT revision applies.
	if _, _, code, msg := s.applyPlanOverlay(ppgTrader, "MNQ", `[{"op":"replace","path":"/reasoning","value":"owner edit"}]`, "owner", w5aDoorNow, expected); code != 0 {
		t.Fatalf("fresh edit refused: %d %s", code, msg)
	}
	// The same draft is now STALE: the row carries one overlay (revision 1)
	// but the draft still claims overlay_version 0 — it must be refused, not
	// applied on top of the newer overlay.
	if _, _, code, _ := s.applyPlanOverlay(ppgTrader, "MNQ", `[{"op":"replace","path":"/reasoning","value":"stale edit"}]`, "owner", w5aDoorNow, expected); code != 409 {
		t.Fatalf("stale overlay revision accepted: %d", code)
	}
	// A draft viewed at plan version 1 must not apply after a replan wrote v2.
	seed()
	one := 1
	expected.OverlayVersion = &one
	if _, _, code, _ := s.applyPlanOverlay(ppgTrader, "MNQ", `[{"op":"replace","path":"/reasoning","value":"v1 draft"}]`, "owner", w5aDoorNow, expected); code != 409 {
		t.Fatalf("stale plan version accepted: %d", code)
	}
	// The newer row was never touched by the stale draft.
	rows, err := s.store.Plan().ListOverlays(id, 2)
	if err != nil || len(rows) != 0 {
		t.Fatalf("new plan mutated: %v %v", rows, err)
	}
}

// handlePlanToday must tell the card WHICH revision it is viewing, or the
// client has nothing to echo back as the expected revision.
func TestPlanTodayCarriesPlanIDAndOverlayVersion(t *testing.T) {
	s, tok := newPicturePlanGateServer(t, `{"day_plan":{"plan_enabled":true}}`)
	row := w5aSeedPlan(t, s.store, w5aNYDate(t, time.Now()), w5aPlanDoc, "NY_scheduled_read")
	w5aOverlay(t, s.store, row, "owner", "o-1", `[{"op":"replace","path":"/reasoning","value":"edited"}]`)
	rec, _ := olDo(t, s, tok, "GET", "/api/plan/today?trader_id="+ppgTrader+"&session=NY", "")
	if rec.Code != 200 {
		t.Fatalf("plan/today: %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		PlanID         string `json:"plan_id"`
		OverlayVersion int    `json:"overlay_version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.PlanID != row.PlanID || out.OverlayVersion != 1 {
		t.Fatalf("plan/today must carry plan_id=%s overlay_version=1, got %q %d\n%s", row.PlanID, out.PlanID, out.OverlayVersion, rec.Body.String())
	}
}
