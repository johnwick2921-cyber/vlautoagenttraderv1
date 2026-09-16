package trader

import (
	"encoding/json"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// ENTRY-MECHANICS E4 (2026-08-30) — sweep-reclaim SPLIT entry: two child
// orders, shared plan lineage, independent OCO brackets; EITHER leg's
// stop-out cancels the sibling's unfilled order (no doubling into a failed
// level). Session-end/dormant cancel paths ride the existing cancel-all
// machinery (they cover BOTH legs by construction).

func splitSweepDoc() string {
	// Sweep of 29494.75: leg 1 rests AT the sweep ref (touch leg), leg 2
	// chains on the 1m-MSS. Short direction (sweep-reclaim long is the mirror;
	// this fixture shorts a reclaim-failure).
	d := kernel.PlanDoc{Bias: kernel.PlanBias{Direction: "short", Conviction: "low", FlipCondition: "n/a"},
		Levels: []kernel.PlanLevel{{Price: 29494.75, Label: "EQH", Grade: "A", Instruction: "fade"}},
		Scenarios: []kernel.PlanScenario{{
			ID: "S1", Trigger: "sweep of 29494.75 reclaim fails", Condition: "sweep_reclaim",
			Direction: "short", TargetChain: []float64{29420}, Invalid: "invalid above 29494.75", Quality: "B",
			Confirm:  &kernel.PlanConfirm{Rule: "touch", RefPrice: 29494.75, Side: "below"},
			Confirm2: &kernel.PlanConfirm{Rule: "1x5m_close", RefPrice: 29494.75, Side: "below"},
			Arm: &kernel.PlanArmSpec{Enabled: true, Entry: 29494.75, Stop: 29510.75, Target: 29450.75,
				Legs: []kernel.PlanArmLeg{
					{Entry: 29494.75, Stop: 29510.75, Target: 29450.75, Size: 1},                                        // leg 1: the touch leg
					{Entry: 29486.00, Stop: 29504.00, Target: 29440.00, Size: 1, WaitConfirm: true, Rule: "1x5m_close"}, // leg 2: chains on the reclaim close
				}},
		}},
		NoTrade: []string{}, DeathCondition: "n/a",
	}
	blob, _ := json.Marshal(d)
	return string(blob)
}

func TestArmSpecSplitContractValidation(t *testing.T) {
	// Legal split (1m_mss leg-2 variant for the arm-spec contract only).
	var d kernel.PlanDoc
	if err := json.Unmarshal([]byte(splitSweepDoc()), &d); err != nil {
		t.Fatal(err)
	}
	d.Scenarios[0].Arm.Legs[1].Rule = "1m_mss"
	d.Scenarios[0].Confirm2.Rule = "1m_mss"
	if err := kernel.ArmSpecValid(d.Scenarios[0]); err != nil {
		t.Fatalf("legal sweep split must pass: %v", err)
	}
	// The 1x5m_close leg-2 alternative is also legal.
	var dA kernel.PlanDoc
	if err := json.Unmarshal([]byte(splitSweepDoc()), &dA); err != nil {
		t.Fatal(err)
	}
	if err := kernel.ArmSpecValid(dA.Scenarios[0]); err != nil {
		t.Fatalf("1x5m_close leg-2 split must pass: %v", err)
	}
	// Leg 2 with 2x5m → rejected.
	d.Scenarios[0].Arm.Legs[1].Rule = "2x5m_close"
	d.Scenarios[0].Confirm2.Rule = "2x5m_close"
	if err := kernel.ArmSpecValid(d.Scenarios[0]); err == nil {
		t.Fatal("leg-2 2x5m must be rejected (sweep_leg2_requires_mss_or_1x5m)")
	}
	// Leg 1 chained → rejected (it must rest ON the touch).
	d.Scenarios[0].Arm.Legs[0].WaitConfirm = true
	if err := kernel.ArmSpecValid(d.Scenarios[0]); err == nil {
		t.Fatal("chained leg 1 must be rejected (it fills ON the touch)")
	}
	// Legs on a non-sweep condition → rejected.
	var d2 kernel.PlanDoc
	if err := json.Unmarshal([]byte(armedDoc()), &d2); err != nil {
		t.Fatal(err)
	}
	d2.Scenarios[0].Arm.Legs = []kernel.PlanArmLeg{{Entry: 100, Stop: 95, Target: 110}, {Entry: 99, Stop: 94, Target: 111}}
	if err := kernel.ArmSpecValid(d2.Scenarios[0]); err == nil {
		t.Fatal("legs on fvg_entry must be rejected (arm_legs_sweep_reclaim_only)")
	}
}

// TestSplitArmWritesTwoLedgerRows — the executor writes ONE ROW PER LEG with
// the shared lineage (plan/scenario) and the leg index distinguishing them.
func TestSplitArmWritesTwoLedgerRows(t *testing.T) {
	// FIX 5 (class 27): a 2-leg split is only authorable when the account's
	// leg capacity is ≥ 2 — the explicit max_contracts_per_order declares it.
	cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true},
		RiskControl: store.RiskControlConfig{MaxContractsPerOrder: 2}}
	// ONE SETUP (dispatch 102, 2026-09-11): this fixture exercises the WIDE book
	// (a non-reject play / no map); the switch OFF restores it byte-identically (E2).
	oneSetupOff(&cfg)
	cfg.RiskControl.MinRiskRewardRatio = 2 // R1 (2026-09-03): the arm floor is the Studio value; this fixture arms at R:R 2.0
	at, st := resetTrader(t, cfg)
	now := time.Now()
	sess, ok := at.sessionRegistry(now).ActiveSession(now)
	if !ok {
		t.Skip("no active session right now")
	}
	// RESTORED TO THE WALL CLOCK 2026-09-10, plus one skip.
	//
	// This test drives the REAL arm path, and that path's plan provider
	// (installActivePlanProvider, auto_trader_planner.go:2671-2673) resolves the
	// active session and the chain trade date from time.Now() with NO seam. So
	// the fixture's clock and the arm path's clock must be the SAME clock: any
	// injected time desynchronises them and the plan lookup returns nil, which
	// surfaces as "got 0 legs" with not one refusal line — silence, because
	// nothing refused, there was simply no plan.
	//
	// I replaced this skip with a SEARCHED clock earlier today to fix a real
	// failure inside the lunch band. That fixed the band and broke every hour
	// outside it. The band needs a skip, not a different clock:
	if kernel.InLunchNoTrade(now) {
		ls, le := kernel.LunchWindowCT()
		t.Skipf("inside the lunch no-trade window (%s–%s CT) — the arm path refuses by design, which is not the behaviour this test measures", ls, le)
	}
	//
	// OWED, and the real fix: a clock seam through the plan provider, so a test
	// can drive the arm path at a moment of its choosing. That belongs to whoever
	// owns the planner. Until then this test runs when the wall clock permits and
	// says plainly when it cannot.
	cfg.DayPlan.SessionsEnabled = []string{sess.Name}
	trueV := true
	cfg.DayPlan.Sessions = []store.DayPlanSessionOverride{{Session: sess.Name, Enable: &trueV}}

	// THE FIXTURE STATES ITS OWN SESSION — and this asserts that it took.
	//
	// The two lines above are how this test declares the live session runnable
	// for itself: DayPlan is a POINTER shared with the trader, so mutating it
	// after resetTrader does reach at's config, and a per-session override with
	// Enable=true beats the registry default inside sessionRunnable.
	//
	// It must be checked HERE, not earlier. I first put this check straight after
	// ActiveSession and it reported "ASIA not enabled in the session registry" —
	// true at that instant and meaningless, because the override two lines down
	// had not been applied yet. A gate read before the setup it gates is not a
	// finding about the system, it is a finding about the reader. That misreading
	// also produced a comment claiming the enablement was a no-op, which was
	// wrong: it works, it just had not happened yet.
	if runnable, why := at.sessionRunnable(sess); !runnable {
		t.Skipf("%s did not become runnable even after this fixture declared it (%s)", sess.Name, why)
	}
	td, _ := kernel.PlanChainTradeDate(sess, now)
	pid := store.MakePlanIDForTrader(at.id, td, sess.Name)
	if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: pid, TradeDate: td, Session: sess.Name, StrategyID: at.id, Lifecycle: "active", Doc: splitSweepDoc(), CreatedAt: now.Add(-30 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	installActivePlanProvider(at, st)

	// A bar tape where the leg-2 confirm (1x5m_close below 29494.75) is ALREADY
	// MET so both legs arm. The tape ends in the PAST (all buckets closed).
	prevProvider := market.FuturesBarsProvider
	market.FuturesBarsProvider = func(symbol string, tf string, n int) []market.Kline {
		base := now.Add(-80 * time.Minute).Truncate(time.Minute).UnixMilli()
		out := make([]market.Kline, 0, 80)
		for i := 0; i < 80; i++ {
			cl := 29490.0
			if i < 70 {
				cl = 29495.0 // quiet base, slightly ABOVE the ref
			}
			if i >= 75 {
				cl = 29480.0 // a full 5m bucket beyond (below) the ref → leg-2 chain MET
			}
			o := base + int64(i)*60_000
			out = append(out, market.Kline{OpenTime: o, CloseTime: o + 59_999, Open: cl, High: cl + 1, Low: cl - 1, Close: cl})
		}
		return out
	}
	t.Cleanup(func() { market.FuturesBarsProvider = prevProvider })

	// THE SAME CLOCK, PASSED EXPLICITLY. `now` is time.Now() — this test must use
	// the wall clock because the plan provider does — but it goes through the seam
	// rather than around it, so the fixture and the arm path are provably reading
	// one clock instead of two that happen to agree.
	at.maybeManageArmedOrdersAt(nil, now)

	rows, err := st.ArmedOrders().ListNonTerminal(at.id)
	if err != nil {
		t.Fatal(err)
	}
	byLeg := map[int]*store.ArmedOrderDB{}
	for i := range rows {
		byLeg[rows[i].LegIndex] = &rows[i]
	}
	if len(byLeg) != 2 {
		t.Fatalf("split arm must write 2 ledger rows (legs), got %d (%+v)", len(byLeg), rows)
	}
	if byLeg[0] == nil || byLeg[1] == nil {
		t.Fatalf("missing leg rows: %+v", rows)
	}
	if byLeg[0].LegCount != 2 || byLeg[1].LegCount != 2 {
		t.Fatalf("both legs must carry LegCount=2: %+v", rows)
	}
	if byLeg[0].EntryPx != 29494.75 || byLeg[1].EntryPx != 29486.00 {
		t.Fatalf("leg entries wrong: %+v", rows)
	}
}

// TestSplitSiblingStopOutCancelsUnfilledLeg — EITHER leg's stop-out cancels
// the sibling's unfilled order (pure decision + ledger state via the manager
// machinery's cancel-all cousin).
func TestSplitSiblingStopOutCancelsUnfilledLeg(t *testing.T) {
	tick := 0.25
	pair := []store.ArmedOrderDB{
		{ID: 1, TraderID: "t1", PlanID: "p:NY", Scenario: "S1", Side: "short", LegIndex: 0, LegCount: 2,
			State: "filled", SignalID: "sig-a", EntryPx: 29494.75, StopPx: 29510.75, TargetPx: 29450.75},
		{ID: 2, TraderID: "t1", PlanID: "p:NY", Scenario: "S1", Side: "short", LegIndex: 1, LegCount: 2,
			State: "armed", EntryPx: 29486.00, StopPx: 29504.00, TargetPx: 29440.00},
	}
	// Leg 0's position closed AT the stop → the unfilled leg 1 must cancel.
	closed := []store.TraderPosition{{TraderID: "t1", Status: "CLOSED", EntryOrderID: "sig-a", ExitPrice: 29510.75, Side: "short"}}
	cancel := splitSiblingCancelDecision(pair, closed, tick)
	if len(cancel) != 1 || cancel[0].ID != 2 {
		t.Fatalf("stop-out must cancel the unfilled sibling, got %+v", cancel)
	}
	// Target-out → NOTHING cancels (the level worked; the sibling stays).
	closed[0].ExitPrice = 29450.75
	if cancel = splitSiblingCancelDecision(pair, closed, tick); len(cancel) != 0 {
		t.Fatalf("target-out must NOT cancel the sibling, got %+v", cancel)
	}
	// No closed position → nothing cancels.
	if cancel = splitSiblingCancelDecision(pair, nil, tick); len(cancel) != 0 {
		t.Fatalf("no closed position must not cancel, got %+v", cancel)
	}
	// A legacy single arm (LegCount 0) is never a pair — no pairing, no cancel.
	legacy := []store.ArmedOrderDB{{ID: 3, TraderID: "t1", PlanID: "p:NY", Scenario: "S2", State: "filled", SignalID: "sig-b", StopPx: 29510.75, TargetPx: 29450.75}}
	if cancel = splitSiblingCancelDecision(legacy, closed, tick); len(cancel) != 0 {
		t.Fatalf("legacy rows must never pair, got %+v", cancel)
	}
}

// TestLogShadowABWritesOnlyItsOwnTable — E8 zero-real-effect law at the store
// level: the shadow logger writes ONLY ab_confirm_log (never the armed
// ledger, never a plan row).
func TestLogShadowABWritesOnlyItsOwnTable(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}})
	sc := kernel.PlanScenario{
		ID: "S1", Condition: "reject", Direction: "long",
		Confirm: &kernel.PlanConfirm{Rule: "touch", RefPrice: 100, Side: "above"},
		Arm:     &kernel.PlanArmSpec{Enabled: true, Entry: 100, Stop: 95, Target: 110},
	}
	base := time.Now().Add(-30 * time.Minute).Truncate(time.Minute).UnixMilli()
	closes := []float64{99, 100.5, 99, 99, 99, 101, 101, 101, 101, 101, 102, 102, 102, 102, 102, 104, 106, 108, 110, 110}
	bars := make([]market.Kline, 0, len(closes))
	for i, cl := range closes {
		o := base + int64(i)*60_000
		bars = append(bars, market.Kline{OpenTime: o, CloseTime: o + 59_999, Open: cl, High: cl + 0.5, Low: cl - 0.5, Close: cl})
	}
	plan := &kernel.ActivePlan{PlanID: "2026-08-28:NY:trader-1", Version: 3, Session: "NY", BirthMs: base - 1}
	at.logShadowAB(plan, sc, bars, 1.0, bars[len(bars)-1].CloseTime+1)

	// The 4-rule counterfactual rows exist (touch/1x5m/2x5m present; MSS absent
	// on this tape) — and NOTHING else changed.
	var n int64
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM ab_confirm_log").Scan(&n); err != nil || n < 3 {
		t.Fatalf("ab_confirm_log rows = %d err=%v, want ≥3", n, err)
	}
	arms, _ := st.ArmedOrders().ListNonTerminal(at.id)
	if len(arms) != 0 {
		t.Fatalf("shadow logger touched the armed ledger: %+v", arms)
	}
	if err := st.AbConfirm().Upsert(&store.AbConfirmLogDB{PlanID: plan.PlanID, Version: plan.Version,
		Scenario: "S1", Rule: "touch", FillPx: 100, Outcome: "target"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	var n2 int64
	if err := st.DB().QueryRow("SELECT COUNT(*) FROM ab_confirm_log").Scan(&n2); err != nil || n2 != n {
		t.Fatalf("idempotent upsert created a duplicate: %d → %d", n, n2)
	}
}

// TestStopEntryFallbackWindow — E7 pure twins: the breakout-retest fallback
// goes live ONLY after RETEST_WAIT_BARS bars with no retest touch.
func TestStopEntryFallbackWindow(t *testing.T) {
	base := time.Now().Add(-30 * time.Minute).Truncate(time.Minute).UnixMilli()
	mk := func(closes []float64) []market.Kline {
		out := make([]market.Kline, 0, len(closes))
		for i, cl := range closes {
			o := base + int64(i)*60_000
			// ±0.25 range: a close of 101 never straddles 100 (low 100.75);
			// a close of 100.2 DOES (low 99.95).
			out = append(out, market.Kline{OpenTime: o, CloseTime: o + 59_999, Open: cl, High: cl + 0.25, Low: cl - 0.25, Close: cl})
		}
		return out
	}
	// 8 quiet bars since birth, no touch of 100 → fallback DUE.
	bars := mk([]float64{101, 101, 101, 101, 101, 101, 101, 101})
	now := bars[len(bars)-1].CloseTime + 1
	if !stopEntryFallbackDue(bars, 100, bars[0].OpenTime, now) {
		t.Fatal("no-retest window elapsed → fallback must be DUE")
	}
	// One bar straddles 100 in-window → the retest came → NOT due.
	bars2 := mk([]float64{101, 101, 101, 101, 101, 100.2, 101, 101})
	now2 := bars2[len(bars2)-1].CloseTime + 1
	if stopEntryFallbackDue(bars2, 100, bars2[0].OpenTime, now2) {
		t.Fatal("a retest touch in-window must block the fallback")
	}
	// Too few bars since birth → not due (the window has not elapsed).
	bars3 := mk([]float64{101, 101, 101})
	now3 := bars3[len(bars3)-1].CloseTime + 1
	if stopEntryFallbackDue(bars3, 100, bars3[0].OpenTime, now3) {
		t.Fatal("fewer than RETEST_WAIT_BARS bars → fallback must NOT be due")
	}
}

// TestStopEntryKnobDefaults — the E7 knob shipped values.
func TestStopEntryKnobDefaults(t *testing.T) {
	if stopEntryOffsetTicks() != 2 {
		t.Fatalf("STOP_ENTRY_OFFSET_TICKS default = %d, want 2", stopEntryOffsetTicks())
	}
	if retestWaitBars() != 6 {
		t.Fatalf("RETEST_WAIT_BARS default = %d, want 6", retestWaitBars())
	}
	if stopEntrySeamOn() {
		t.Fatal("STOP_ENTRY_SEAM must default OFF (D-rule: unproven C# paths never ship on)")
	}
}

// TestSplitArmSessionEndCancelsBothLegs — the existing cancel-first machinery
// (dormant/session-end) covers BOTH legs: two rows in, zero non-terminal out.
func TestSplitArmSessionEndCancelsBothLegs(t *testing.T) {
	cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	cfg.RiskControl.MinRiskRewardRatio = 2 // R1 (2026-09-03): the arm floor is the Studio value; this fixture arms at R:R 2.0
	at, st := resetTrader(t, cfg)
	now := time.Now()
	sess, ok := at.sessionRegistry(now).ActiveSession(now)
	if !ok {
		t.Skip("no active session right now")
	}
	cfg.DayPlan.SessionsEnabled = []string{sess.Name}
	trueV := true
	cfg.DayPlan.Sessions = []store.DayPlanSessionOverride{{Session: sess.Name, Enable: &trueV}}
	td, _ := kernel.PlanChainTradeDate(sess, now)
	pid := store.MakePlanIDForTrader(at.id, td, sess.Name)
	if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: pid, TradeDate: td, Session: sess.Name, StrategyID: at.id, Lifecycle: "active", Doc: splitSweepDoc(), CreatedAt: now.Add(-30 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	for _, leg := range []store.ArmedOrderDB{
		{TraderID: at.id, PlanID: pid, Version: 1, Session: sess.Name, Scenario: "S1", Side: "short",
			EntryPx: 29494.75, StopPx: 29510.75, TargetPx: 29450.75, State: "armed", LegIndex: 0, LegCount: 2},
		{TraderID: at.id, PlanID: pid, Version: 1, Session: sess.Name, Scenario: "S1", Side: "short",
			EntryPx: 29486.00, StopPx: 29504.00, TargetPx: 29440.00, State: "armed", LegIndex: 1, LegCount: 2},
	} {
		if err := st.ArmedOrders().UpsertArm(&leg); err != nil {
			t.Fatal(err)
		}
	}
	// Go dormant → the manager cancels ALL arms (both legs) instantly.
	if err := st.Plan().UpdatePlanLifecycle(pid, 1, "dormant", "dormant:test"); err != nil {
		t.Fatal(err)
	}
	installActivePlanProvider(at, st)
	at.maybeManageArmedOrdersAt(nil, armTestClock(t, at))
	rows, err := st.ArmedOrders().ListNonTerminal(at.id)
	if err != nil || len(rows) != 0 {
		t.Fatalf("dormant must cancel BOTH split legs (rows=%d err=%v)", len(rows), err)
	}
}
