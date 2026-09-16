package api

import (
	"path/filepath"
	"testing"

	"nofx/store"
)

// CLASS 35 (2026-09-01) — the card's replans_left is the RECORDED budget, the
// same number the death gate and the executor prompt read. The API used to
// compute cap − (version − baseline) from the row it was serving, so today's
// LONDON chain (six rows, nothing spent) rendered "0 re-reads left".
func TestClass35APIReplansLeftIsTheRecordedBudget(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	s := &Server{store: st} // no traderManager → the shipped default cap, resolved nil-safely
	const tid, date, sess = "trader-1", "2026-09-01", "LONDON"
	for _, c := range []struct{ trigger, lifecycle string }{
		{"planner_fail_closed", "no_trade"}, {"level_event", "active"}, {"dormant:flip:x", "dormant"},
		{"level_event", "active"}, {"level_event", "active"}, {"level_event", "active"},
	} {
		if _, err := st.Plan().AppendPlan(&store.PlanDB{
			PlanID: store.MakePlanID(date, sess), StrategyID: tid, TradeDate: date, Session: sess,
			TriggerReason: c.trigger, Lifecycle: c.lifecycle, Doc: "{}",
		}); err != nil {
			t.Fatal(err)
		}
	}
	_, _, left, cap := s.planRulesWithCap(tid, sess, date)
	if cap <= 0 {
		t.Fatalf("resolved cap = %d — a resolver must return the shipped default, never 0", cap)
	}
	if left != cap {
		t.Errorf("replans_left = %d, want the full cap %d: six rows, zero death re-plans, zero owner re-reads", left, cap)
	}
	// One recorded spend → the API moves by exactly one; the chip's "used" is
	// cap − replans_left, so card and API can no longer disagree.
	if _, err := store.SpendReplan(st, tid, date, sess); err != nil {
		t.Fatal(err)
	}
	if _, _, left2, cap2 := s.planRulesWithCap(tid, sess, date); left2 != cap-1 || cap2 != cap {
		t.Errorf("after one spend: replans_left = %d cap = %d, want %d / %d", left2, cap2, cap-1, cap)
	}
}
