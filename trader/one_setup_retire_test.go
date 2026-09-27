package trader

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// ── ONE SETUP — the gap the first boot found, closed (owner ruling 2026-09-11) ─
//
// One-setup filtered AUTHORIZATION, not the placement of an authorization that
// predates it: the class-33 sweep left rows 150/153 "armed for this process to
// place" and the placement engine would have placed them although their
// scenarios were declined. PIN: at placement time, an authorization whose
// scenario is currently DECLINED is retired with its three verdicts — never
// placed, never sent to the broker (there is nothing at the broker under it);
// an ALLOWED one still places. Driven on the loopback wire so "placed" means a
// signal frame, not a ledger state.
func TestOneSetupDeclinedPreBootAuthorizationRetiredNeverPlaced(t *testing.T) {
	// One fixture clock: 10:00 CT, outside lunch; TEST session is explicit.
	now := time.Date(2026, time.September, 11, 15, 0, 0, 0, time.UTC)
	cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	cfg.RiskControl.MinRiskRewardRatio = 2
	structuralTestPolicy(&cfg, .5)
	at, st, sigs, cancels := shadowWireHarnessAt(t, cfg, now)
	// S1 reject long at 100 — the best level; S2 reject long at 92 — a lower
	// graded level BELOW price (a resting long limit there is placeable, unlike
	// one above price, which today's engine already refuses as marketable),
	// inside the 25-pt placement band, DECLINED level_not_best.
	live := kernel.PlanDoc{Bias: kernel.PlanBias{Direction: "long", Conviction: "low", FlipCondition: "n/a"},
		Levels: []kernel.PlanLevel{{Price: 100, Label: "PDL", Grade: "A", Instruction: "fade"}, {Price: 92, Label: "SWG-L", Grade: "B", Instruction: "fade"}},
		Scenarios: []kernel.PlanScenario{
			{ID: "S1", Trigger: "t", Condition: "reject", Direction: "long", TargetChain: []float64{110}, Invalid: "i", Quality: "B",
				Confirm: &kernel.PlanConfirm{Rule: "touch", RefPrice: 100, Side: "above"},
				Arm:     &kernel.PlanArmSpec{Enabled: true, Entry: 100, Stop: 95, Target: 110}},
			{ID: "S2", Trigger: "t", Condition: "reject", Direction: "long", TargetChain: []float64{120}, Invalid: "i", Quality: "B",
				Confirm: &kernel.PlanConfirm{Rule: "touch", RefPrice: 92, Side: "above"},
				Arm:     &kernel.PlanArmSpec{Enabled: true, Entry: 92, Stop: 87, Target: 104}},
		},
		NoTrade: []string{}, DeathCondition: "n/a",
	}
	structuralTestMap(&live, structuralTestZone{100, 95.5, 100, "PDL"}, structuralTestZone{92, 87.5, 92, "SWG-L"}, structuralTestZone{110, 110, 111, "target"})
	blob, _ := json.Marshal(live)
	pid := shadowPlanAtTime(t, at, st, string(blob), now)
	// Two PRE-BOOT authorizations, never placed (no signal id), authored by a
	// process that is gone — exactly rows 150/153's shape.
	for _, r := range []store.ArmedOrderDB{
		{TraderID: at.id, PlanID: pid, Version: 1, Session: "X", Scenario: "S1", Side: "long", EntryPx: 100, StopPx: 95, TargetPx: 110, State: store.StateArmed, EntryClass: "armed_fill", Condition: "reject", BootID: "old-boot-1"},
		{TraderID: at.id, PlanID: pid, Version: 1, Session: "X", Scenario: "S2", Side: "long", EntryPx: 92, StopPx: 87, TargetPx: 104, State: store.StateArmed, EntryClass: "armed_fill", Condition: "reject", BootID: "old-boot-1"},
	} {
		row := r
		if err := st.ArmedOrders().UpsertArm(&row); err != nil {
			t.Fatal(err)
		}
	}
	at.oneSetupFactsForTest = func(now time.Time) oneSetupTestFacts {
		a, b := *structuralTestIdentity(100, "PDL").ID, *structuralTestIdentity(92, "SWG-L").ID
		return oneSetupTestFacts{Price: 100, BandPts: 50, Candidates: []kernel.MapCandidate{
			{ID: &a, Identity: kernel.PlanLevel{ID: &a, Price: 100}, Price: 100, Names: []string{"PDL"}, Grade: "A", Distance: 0},
			{ID: &b, Identity: kernel.PlanLevel{ID: &b, Price: 92}, Price: 92, Names: []string{"SWG-L"}, Grade: "B", Distance: -8},
		}, Permission: map[string]kernel.FadeVerdict{"S1": {Evaluated: true, Permitted: true}, "S2": {Evaluated: true, Permitted: true}}}
	}
	prev := market.FuturesBarsProvider
	market.FuturesBarsProvider = func(string, string, int) []market.Kline { return shadowBarsNearAt(100, now) }
	t.Cleanup(func() { market.FuturesBarsProvider = prev })

	at.maybeManageArmedOrdersAt(nil, now)

	// The ALLOWED scenario's pre-boot row still places: one signal, S1's.
	select {
	case s := <-sigs:
		if s.SignalID == "" {
			t.Fatalf("empty signal: %+v", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the allowed scenario's pre-boot authorization did NOT place — the retire pass over-reached")
	}
	select {
	case s := <-sigs:
		t.Fatalf("a SECOND placement reached the wire — the declined scenario's pre-boot authorization was placed: %+v", s)
	case <-time.After(300 * time.Millisecond):
	}
	select {
	case c := <-cancels:
		t.Fatalf("a broker cancel was sent for a row with nothing at the broker: %+v", c)
	default:
	}
	rows, _ := st.ArmedOrders().ListForPlan(pid)
	var s1, s2 *store.ArmedOrderDB
	for i := range rows {
		switch rows[i].Scenario {
		case "S1":
			s1 = &rows[i]
		case "S2":
			s2 = &rows[i]
		}
	}
	if s2 == nil || s2.State != store.StateCancelled {
		t.Fatalf("the declined scenario's pre-boot row must be retired (cancelled by ledger state): %+v", s2)
	}
	if !strings.HasPrefix(s2.StateReason, "declined by one-setup; pre-boot authorization not placed") ||
		!strings.Contains(s2.StateReason, "level=level_not_best") || !strings.Contains(s2.StateReason, "play=ok") || !strings.Contains(s2.StateReason, "permission=ok") {
		t.Fatalf("the retirement must carry the owner's reason and all three verdicts: %q", s2.StateReason)
	}
	if s1 == nil || store.IsTerminalArmState(s1.State) || s1.SignalID == "" {
		t.Fatalf("the allowed scenario's row must be placed, not retired: %+v", s1)
	}
}

// OFF: nothing is retired — today's book, including its pre-boot rows.
func TestOneSetupRetireIsInertWhenOff(t *testing.T) {
	g, _, st, _ := driveOneSetupArmPath(t, osOff, func(at *AutoTrader, st *store.Store, pid string) {
		row := store.ArmedOrderDB{TraderID: at.id, PlanID: pid, Version: 1, Session: "X", Scenario: "S3", Side: "short", EntryPx: 29498, StopPx: 29512, TargetPx: 29450, State: store.StateArmed, EntryClass: "armed_fill", Condition: "sweep_reclaim", BootID: "old-boot-1"}
		if err := st.ArmedOrders().UpsertArm(&row); err != nil {
			t.Fatal(err)
		}
	})
	_ = g
	rows, _ := st.ArmedOrders().ListNonTerminal("trader-1")
	found := false
	for _, r := range rows {
		if r.Scenario == "S3" && r.State == store.StateArmed {
			found = true
		}
	}
	if !found {
		t.Fatal("OFF must leave a pre-boot authorization exactly as today's book would")
	}
}

// F13 (port of #117 2dc94a19) — AN UNKNOWN OR UNWRITABLE PERMISSION CANNOT
// PLACE AN OLD AUTHORIZATION. A retire pass that fails must stop the placement
// pass — never let an inherited armed row (or any later row) place without a
// CURRENT permission verdict.
func TestUnknownOneSetupVerdictCannotPlaceOldAuthorization(t *testing.T) {
	t.Run("missing verdict", func(t *testing.T) {
		now := time.Date(2026, time.September, 11, 15, 0, 0, 0, time.UTC)
		cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
		cfg.RiskControl.MinRiskRewardRatio = 2
		structuralTestPolicy(&cfg, .5)
		at, st, _, _ := shadowWireHarnessAt(t, cfg, now)
		live := kernel.PlanDoc{Bias: kernel.PlanBias{Direction: "long", Conviction: "low", FlipCondition: "n/a"},
			Levels: []kernel.PlanLevel{{Price: 100, Label: "PDL", Grade: "A", Instruction: "fade"}},
			Scenarios: []kernel.PlanScenario{
				{ID: "S1", Trigger: "t", Condition: "reject", Direction: "long", TargetChain: []float64{110}, Invalid: "i", Quality: "B",
					Confirm: &kernel.PlanConfirm{Rule: "touch", RefPrice: 100, Side: "above"},
					Arm:     &kernel.PlanArmSpec{Enabled: true, Entry: 100, Stop: 95, Target: 110}},
			},
			NoTrade: []string{}, DeathCondition: "n/a",
		}
		structuralTestMap(&live, structuralTestZone{100, 95.5, 100, "PDL"}, structuralTestZone{110, 110, 111, "target"})
		blob, _ := json.Marshal(live)
		pid := shadowPlanAtTime(t, at, st, string(blob), now)
		row := store.ArmedOrderDB{TraderID: at.id, PlanID: pid, Version: 1, Session: "X", Scenario: "S1",
			Side: "long", EntryPx: 100, StopPx: 95, TargetPx: 110, State: store.StateArmed, BootID: "999-old"}
		if err := st.ArmedOrders().UpsertArm(&row); err != nil {
			t.Fatal(err)
		}
		at.oneSetupFactsForTest = func(time.Time) oneSetupTestFacts { panic("synthetic unavailable permission facts") }
		prev := market.FuturesBarsProvider
		market.FuturesBarsProvider = func(string, string, int) []market.Kline { return shadowBarsNearAt(100, now) }
		t.Cleanup(func() { market.FuturesBarsProvider = prev })

		at.maybeManageArmedOrdersAt(nil, now)

		if err := st.GormDB().First(&row, row.ID).Error; err != nil {
			t.Fatal(err)
		}
		if row.State != store.StateCancelled {
			t.Fatalf("an authorization with no current verdict was not retired: %s", row.State)
		}
		if !strings.Contains(row.StateReason, "level=unknown") || !strings.Contains(row.StateReason, "permission=unknown") {
			t.Fatalf("the retirement must name the unavailable verdict: %q", row.StateReason)
		}
	})

	t.Run("retirement write fails: positive control places without the trigger", func(t *testing.T) {
		at, st, sigs, pid, now := f13RetireWriteFixture(t, false)
		// POSITIVE CONTROL (CTO F13-verify): identical setup WITHOUT the abort
		// trigger — the allowed arm is S1 at PDL@100 (the best level), and it MUST
		// reach the wire, so the negative case below cannot pass vacuously on a
		// fixture where nothing places.
		at.maybeManageArmedOrdersAt(nil, now)
		select {
		case sig := <-sigs:
			if sig.LimitPrice != 100 || sig.TakeProfit != 110 {
				t.Fatalf("positive control placed the wrong arm: %+v", sig)
			}
		case <-time.After(time.Second):
			var rows []store.ArmedOrderDB
			if err := st.GormDB().Where("plan_id = ?", pid).Find(&rows).Error; err != nil {
				t.Fatal(err)
			}
			t.Fatalf("positive control: S1 must place when the retire pass can write; rows=%+v", rows)
		}
	})

	t.Run("retirement write fails", func(t *testing.T) {
		at, _, sigs, _, now := f13RetireWriteFixture(t, true)
		buf := captureTraderLog(t)
		at.maybeManageArmedOrdersAt(nil, now)
		select {
		case sig := <-sigs:
			t.Fatalf("a placement reached the wire while the retire pass could not write: %+v", sig)
		case <-time.After(300 * time.Millisecond):
		}
		// The pass must surface the retirement failure. EITHER F13 mutant (the
		// retire-write error swallowed, or the pass ignoring the error) fails THIS
		// log assertion only: the no-signal assertion above holds under all three
		// because D4's one-arm consult (oneSetupConsult, 'an allowed scenario
		// waits while another holds the plan's one arm') holds the slot for the
		// unretired armed S2 row — the early return is defense-in-depth behind D4,
		// not the only guard.
		if !strings.Contains(buf.String(), "one setup retirement unavailable") {
			t.Fatalf("the pass must log the retirement failure; got logs:\n%s", buf.String())
		}
	})

}

// f13RetireWriteFixture builds the shared F13 error-path fixture: S1 (allowed,
// pre-boot armed, at the BEST level PDL@100 — the one-setup cycle only
// admits the best level) and S2 (declined, pre-boot armed, level_not_best
// SWG-L@92) in one plan. withTrigger installs the fail_unknown_retire
// SQLite trigger that makes the DECLINED row's retirement write ABORT.
// Geometry identical to TestOneSetupDeclinedPreBootAuthorizationRetired-
// NeverPlaced (S1 places there, MinRR 2).
func f13RetireWriteFixture(t *testing.T, withTrigger bool) (*AutoTrader, *store.Store, chan ntwire.SignalPayload, string, time.Time) {
	t.Helper()
	now := time.Date(2026, time.September, 11, 15, 0, 0, 0, time.UTC)
	cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	cfg.RiskControl.MinRiskRewardRatio = 2
	structuralTestPolicy(&cfg, .5)
	at, st, sigs, _ := shadowWireHarnessAt(t, cfg, now)
	live := kernel.PlanDoc{Bias: kernel.PlanBias{Direction: "long", Conviction: "low", FlipCondition: "n/a"},
		Levels: []kernel.PlanLevel{{Price: 100, Label: "PDL", Grade: "A", Instruction: "fade"}, {Price: 92, Label: "SWG-L", Grade: "B", Instruction: "fade"}},
		Scenarios: []kernel.PlanScenario{
			{ID: "S1", Trigger: "t", Condition: "reject", Direction: "long", TargetChain: []float64{110}, Invalid: "i", Quality: "B",
				Confirm: &kernel.PlanConfirm{Rule: "touch", RefPrice: 100, Side: "above"},
				Arm:     &kernel.PlanArmSpec{Enabled: true, Entry: 100, Stop: 95, Target: 110}},
			{ID: "S2", Trigger: "t", Condition: "reject", Direction: "long", TargetChain: []float64{120}, Invalid: "i", Quality: "B",
				Confirm: &kernel.PlanConfirm{Rule: "touch", RefPrice: 92, Side: "above"},
				Arm:     &kernel.PlanArmSpec{Enabled: true, Entry: 92, Stop: 87, Target: 104}},
		},
		NoTrade: []string{}, DeathCondition: "n/a",
	}
	structuralTestMap(&live, structuralTestZone{100, 95.5, 100, "PDL"}, structuralTestZone{92, 87.5, 92, "SWG-L"}, structuralTestZone{110, 110, 111, "target"})
	blob, _ := json.Marshal(live)
	pid := shadowPlanAtTime(t, at, st, string(blob), now)
	for _, r := range []store.ArmedOrderDB{
		{TraderID: at.id, PlanID: pid, Version: 1, Session: "X", Scenario: "S1", Side: "long", EntryPx: 100, StopPx: 95, TargetPx: 110, State: store.StateArmed, BootID: "999-old"},
		{TraderID: at.id, PlanID: pid, Version: 1, Session: "X", Scenario: "S2", Side: "long", EntryPx: 92, StopPx: 87, TargetPx: 104, State: store.StateArmed, BootID: "999-old"},
	} {
		row := r
		if err := st.ArmedOrders().UpsertArm(&row); err != nil {
			t.Fatal(err)
		}
	}
	if withTrigger {
		// The declined row's retirement write ABORTS — the F13 error path.
		if err := st.GormDB().Exec("CREATE TRIGGER fail_unknown_retire BEFORE UPDATE OF state ON armed_orders WHEN NEW.state = 'cancelled' BEGIN SELECT RAISE(ABORT, 'fixture'); END").Error; err != nil {
			t.Fatal(err)
		}
	}
	at.oneSetupFactsForTest = func(now time.Time) oneSetupTestFacts {
		a, b := *structuralTestIdentity(100, "PDL").ID, *structuralTestIdentity(92, "SWG-L").ID
		return oneSetupTestFacts{Price: 100, BandPts: 50, Candidates: []kernel.MapCandidate{
			{ID: &a, Identity: kernel.PlanLevel{ID: &a, Price: 100}, Price: 100, Names: []string{"PDL"}, Grade: "A", Distance: 0},
			{ID: &b, Identity: kernel.PlanLevel{ID: &b, Price: 92}, Price: 92, Names: []string{"SWG-L"}, Grade: "B", Distance: -8},
		}, Permission: map[string]kernel.FadeVerdict{"S1": {Evaluated: true, Permitted: true}, "S2": {Evaluated: true, Permitted: false}}}
	}
	prev := market.FuturesBarsProvider
	market.FuturesBarsProvider = func(string, string, int) []market.Kline { return shadowBarsNearAt(100, now) }
	t.Cleanup(func() { market.FuturesBarsProvider = prev })
	return at, st, sigs, pid, now
}
