package store

import "testing"

// ── ONE SETUP — the store side: stamp once, link by price, follow-plan fields ─

// E7 (store half) — the verdict lands on the scenario's episode row, ONCE.
func TestOneSetupStampIsWrittenOnce(t *testing.T) {
	st := NewTouchOutcomeStore(newArmedTestDB(t))
	id := openRow(t, st, "", 1_700_000_000_000)

	rows, err := st.UnstampedEpisodesNear("t1", "P1", 29001.5, 3.0)
	if err != nil || len(rows) != 1 || rows[0].ID != id {
		t.Fatalf("price-proximity link within the map width must find the row: rows=%d err=%v", len(rows), err)
	}
	if rows, _ := st.UnstampedEpisodesNear("t1", "P1", 29010, 3.0); len(rows) != 0 {
		t.Fatal("a level 10 pts away is not this episode's level")
	}

	first := OneSetupStamp{Scenario: "S1", Verdicts: "level=ok play=play_not_reject:sweep_reclaim permission=ok", Reason: "declined", EvaluatedMs: 1_700_000_100_000}
	if err := st.StampOneSetup(id, first); err != nil {
		t.Fatal(err)
	}
	second := OneSetupStamp{Scenario: "S1", Verdicts: "level=ok play=ok permission=ok", Reason: "allowed", EvaluatedMs: 1_700_000_200_000}
	if err := st.StampOneSetup(id, second); err != nil {
		t.Fatal(err)
	}
	got := fadeReadRow(t, st, id)
	if got.OneSetupVerdicts == nil || *got.OneSetupVerdicts != first.Verdicts {
		t.Fatalf("stamp rewritten or missing: %v", got.OneSetupVerdicts)
	}
	if got.OneSetupEvaluatedMs == nil || *got.OneSetupEvaluatedMs != first.EvaluatedMs {
		t.Fatal("the clock of the first verdict must be kept")
	}
	if rows, _ := st.UnstampedEpisodesNear("t1", "P1", 29000, 3.0); len(rows) != 0 {
		t.Fatal("a stamped row must not be offered for stamping again")
	}
}

// NULL IS NOT A VERDICT. An unstamped row reads NULL, never "allowed".
func TestOneSetupUnstampedReadsNull(t *testing.T) {
	st := NewTouchOutcomeStore(newArmedTestDB(t))
	id := openRow(t, st, "", 1_700_000_000_000)
	got := fadeReadRow(t, st, id)
	if got.OneSetupVerdicts != nil || got.OneSetupReason != nil || got.FollowBreakAtMs != nil || got.FollowRoleReversed != nil {
		t.Fatal("fresh row must carry NULL one-setup and follow-plan fields")
	}
	if err := st.MarkOneSetupUnrecomputable(id, "no_plan"); err != nil {
		t.Fatal(err)
	}
	got = fadeReadRow(t, st, id)
	if got.OneSetupVerdicts != nil || got.OneSetupBackfill == nil || *got.OneSetupBackfill != "unrecomputable:no_plan" {
		t.Fatalf("unrecomputable must leave the verdict NULL and say why: %v %v", got.OneSetupVerdicts, got.OneSetupBackfill)
	}
}

// E8 (store half) — a follow-plan writes exactly the fields it knows; a later
// pass fills the rest without rewriting the first observation; a never-broken
// level is a ROW with break_at NULL.
func TestFollowPlanStampFillsForward(t *testing.T) {
	st := NewTouchOutcomeStore(newArmedTestDB(t))
	id := openRow(t, st, "", 1_700_000_000_000)
	never := openRow(t, st, "", 1_700_000_600_000)

	brk, dir, bias, frozen := int64(1_700_000_300_000), "down", "short", "long"
	if err := st.StampFollowPlan(id, FollowPlanStamp{BreakAtMs: &brk, BreakDir: &dir, BiasWouldFlip: &bias, PlanBiasFrozen: &frozen, State: "broken_no_retest"}); err != nil {
		t.Fatal(err)
	}
	got := fadeReadRow(t, st, id)
	if got.FollowBreakAtMs == nil || *got.FollowBreakAtMs != brk || got.FollowRetestAtMs != nil || got.FollowRoleReversed != nil {
		t.Fatalf("break-only pass wrote the wrong fields: %+v", got)
	}
	if got.BiasWouldFlipTo == nil || *got.BiasWouldFlipTo != "short" || got.PlanBiasFrozen == nil || *got.PlanBiasFrozen != "long" {
		t.Fatal("bias_would_flip_to is written BESIDE the frozen plan bias, never into it")
	}
	// A second pass with a different break must not move the first observation.
	later := brk + 60_000
	rt, px, basis, rev, out := int64(1_700_000_420_000), 29000.0, "follow:passive_limit_at_level:through_1_tick", true, "hold"
	if err := st.StampFollowPlan(id, FollowPlanStamp{BreakAtMs: &later, RetestAtMs: &rt, EntryPx: &px, EntryBasis: &basis, RoleReversed: &rev, RetestOutcome: &out, State: "complete"}); err != nil {
		t.Fatal(err)
	}
	got = fadeReadRow(t, st, id)
	if *got.FollowBreakAtMs != brk {
		t.Fatal("first observation of the break must be kept")
	}
	if got.FollowRetestAtMs == nil || *got.FollowRetestAtMs != rt || got.FollowRoleReversed == nil || !*got.FollowRoleReversed || got.FollowState == nil || *got.FollowState != "complete" {
		t.Fatalf("retest pass did not fill forward: %+v", got)
	}
	c := st.CountFollowPlans("t1", 0)
	if !c.Readable || c.Rows != 2 || c.Breaks != 1 || c.Retests != 1 || c.RetestJudged != 1 || c.Reversed != 1 {
		t.Fatalf("counts READ from the table: %+v", c)
	}
	// The never-broken level is still a row with break_at NULL.
	if got := fadeReadRow(t, st, never); got.FollowBreakAtMs != nil {
		t.Fatal("never-broken level must keep break_at NULL")
	}
	open, _ := st.OpenFollowPlans("t1", "MNQ", 0)
	if len(open) != 1 || open[0].ID != never {
		t.Fatalf("only the unfinished row stays open: %d", len(open))
	}
}

func TestResolveOneSetupDefaults(t *testing.T) {
	en, g, es, gs := ResolveOneSetup(nil)
	if !en || g != "B" || es != SourceShippedDefault || gs != SourceShippedDefault {
		t.Fatalf("nil config → ON, B, shipped defaults; got %v %s %s %s", en, g, es, gs)
	}
	off := false
	cfg := &StrategyConfig{DayPlan: &DayPlanConfig{OneSetupEnabled: &off, OneSetupMinGrade: "a"}}
	en, g, es, gs = ResolveOneSetup(cfg)
	if en || g != "A" || es != SourceSaved || gs != SourceSaved {
		t.Fatalf("saved false / a → OFF, A, saved; got %v %s %s %s", en, g, es, gs)
	}
	cfg.DayPlan.OneSetupMinGrade = "gold"
	_, g, _, gs = ResolveOneSetup(cfg)
	if g != "B" || gs == SourceSaved {
		t.Fatalf("a non-grade must fall back to B and say so: %s %s", g, gs)
	}
}
