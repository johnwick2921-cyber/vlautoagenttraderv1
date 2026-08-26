package store

import (
	"path/filepath"
	"testing"
)

// W16/R2 — declining a proposal must be a RECORDED decision.
//
// Before this, "Keep as-is" set local React state only. Two consequences: the
// card claimed "Applied — card updated" for a proposal that changed nothing, and
// the KPI could not distinguish a REJECTED proposal from one the owner never
// answered — both were applied=false.
func TestW16DeclineIsRecordedAndCounted(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "qa.db"))
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	qa := st.PlanQA()
	const trader, plan = "t1", "2026-08-17:NY"

	// A planner PROPOSE-MERGE reply the owner will decline.
	proposalID, err := qa.Append(&PlanQADB{
		TraderID: trader, PlanID: plan, TradeDate: "2026-08-17", Session: "NY",
		Role: "planner", Verdict: "PROPOSE-MERGE", PointClass: "NEW-INFO",
		Patch:   `[{"op":"replace","path":"/levels/0/instruction","value":"fade"}]`,
		Trigger: "owner-edit", ChallengeType: "edit-level",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Baseline: nothing declined yet, and the proposal is NOT applied.
	if n, _ := qa.CountOwnerDeclines(trader, plan, proposalID); n != 0 {
		t.Fatalf("baseline declines = %d, want 0", n)
	}
	base, _ := qa.VerdictStats(trader)
	if base.ProposeMerge != 1 || base.Applied != 0 || base.Declined != 0 {
		t.Fatalf("baseline stats wrong: %+v", base)
	}

	// The owner declines.
	if _, err := qa.Append(&PlanQADB{
		TraderID: trader, PlanID: plan, TradeDate: "2026-08-17", Session: "NY",
		Role: "owner", Verdict: VerdictDeclined, Content: DeclineContentFor(proposalID),
		Trigger: "owner-edit", ChallengeType: "edit-level",
	}); err != nil {
		t.Fatal(err)
	}

	if n, _ := qa.CountOwnerDeclines(trader, plan, proposalID); n != 1 {
		t.Fatalf("declines after decline = %d, want 1 (the idempotency guard reads this)", n)
	}

	after, _ := qa.VerdictStats(trader)
	if after.Declined != 1 {
		t.Errorf("Declined = %d, want 1 — a rejection must be countable, not silent", after.Declined)
	}
	// The planner-reply counters must be UNCHANGED: an owner row may not redefine
	// DEFEND / CONCEDE / PROPOSE-MERGE or inflate Total.
	if after.Total != base.Total || after.ProposeMerge != base.ProposeMerge ||
		after.Defend != base.Defend || after.Concede != base.Concede ||
		after.NewInfo != base.NewInfo || after.Applied != base.Applied {
		t.Errorf("owner decline row polluted the planner counters:\n before %+v\n after  %+v", base, after)
	}

	// A decline of a DIFFERENT reply is counted separately (content is the link).
	if n, _ := qa.CountOwnerDeclines(trader, plan, proposalID+999); n != 0 {
		t.Errorf("decline lookup must be scoped to its own reply, got %d", n)
	}
}
