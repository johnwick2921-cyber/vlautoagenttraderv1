package trader

import (
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// ── W-EXEC-TRUTH W5 (builder A) — a machine-only chain is "no plan" to the
// scheduler (F3/D20), and the AI version that supersedes a version carrying a
// LIVE Picture scenario re-appends it (CTO 1790191033566). Driven through
// maybeRunSessionReadsAt → runPlannerRead → the planner write site.

// schedulerTape is a realistic session of 1m bars ending at the wall clock
// (the planner preflight judges bar freshness on the wall clock).
func schedulerTape(t *testing.T) {
	t.Helper()
	prev := market.FuturesBarsProvider
	t.Cleanup(func() { market.FuturesBarsProvider = prev })
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline {
		now := time.Now().UnixMilli()
		bars := make([]market.Kline, 0, 390)
		base := now - 400*60_000
		for i := 0; i < 390; i++ {
			o := base + int64(i)*60_000
			bars = append(bars, market.Kline{OpenTime: o, High: 15650 + float64(i%10), Low: 15550 + float64(i%10), Close: 15600 + float64(i%10), CloseTime: o + 59_000})
		}
		return bars
	}
}

// pinTraderNow pins the lifecycle clock (traderNow) the planner write site
// judges a machine scenario's eligibility with; cleared after the async read
// is joined.
func pinTraderNow(t *testing.T, at time.Time) {
	t.Helper()
	prev := testNow
	testNow = func() time.Time { return at }
	t.Cleanup(func() { testNow = prev })
}

func waitVersion(t *testing.T, st *store.Store, v int) *store.PlanDB {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if row, _ := st.Plan().GetLatestPlanForTraderSession(handOffDate, "NY", handOffTraderID); row != nil && row.Version >= v {
			return row
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil
}

// machinePlanV1 records one Picture opportunity through the production
// hand-off while no plan exists: machine plan v1 with P1.
func machinePlanV1(t *testing.T, at *AutoTrader, st *store.Store, now time.Time) (PictureEvidence, kernel.PlanScenario) {
	t.Helper()
	at.markPictureRunEpoch(now)
	ev := handOffEvidence(now, 1)
	claimHandOff(t, st, ev)
	if err := at.pictureHandOffAt(ev, now); err != nil {
		t.Fatalf("hand-off: %v", err)
	}
	v1, _ := st.Plan().GetLatestPlanForTraderSession(handOffDate, "NY", at.id)
	if v1 == nil || !store.IsMachinePlan(v1) {
		t.Fatalf("fixture: machine plan v1 expected, got %+v", v1)
	}
	doc, _ := resolveActivePlanDoc(st, v1)
	p1, ok := kernel.MachineScenarioByRef(doc, ev.OppKey)
	if !ok {
		t.Fatal("fixture: P1 must be in the machine plan")
	}
	return ev, p1
}

// The AI read still fires on a machine-only chain and lands v2; the machine
// row sees no death / dormant / wake handling; the LIVE P1 is re-appended to
// v2 with the SAME scenario value (same id, same machine record).
func TestMachineOnlyChainStillFiresTheAIReadAndP1RidesIntoV2(t *testing.T) {
	at, st := handOffTrader(t)
	schedulerTape(t)
	now := handOffNow()
	pinTraderNow(t, now.Add(2*time.Second)) // inside P1's 10 s window
	ev, p1 := machinePlanV1(t, at, st, now)

	fired := at.maybeRunSessionReadsAt(now)
	defer drainReReads(t)
	if len(fired) != 1 || fired[0].Session != "NY" || fired[0].TradeDate != handOffDate {
		t.Fatalf("a machine-only chain is 'no plan': the NY read must fire, got %+v", fired)
	}
	v2 := waitVersion(t, st, 2)
	if v2 == nil {
		t.Fatal("the AI read must land v2 on top of the machine plan")
	}
	if store.IsMachinePlan(v2) || v2.TriggerReason != "NY_scheduled_read" || v2.Lifecycle != "active" {
		t.Fatalf("v2 must be the AI's scheduled read (never a death re-plan of the machine row): %+v", v2)
	}
	drainReReads(t)
	// No death / dormant handling ever touched the machine row.
	if log, _ := st.Plan().LifecycleLog(v2.PlanID, 1); len(log) != 0 {
		t.Fatalf("the machine row must see no lifecycle transition: %+v", log)
	}
	if b := store.GetReplanBudget(st, at.id, handOffDate, "NY", 2); b.Used != 0 {
		t.Fatalf("no replan budget is spent on a machine plan: used %d", b.Used)
	}
	// P1 rides into v2: one machine overlay, the SAME scenario value.
	ovs := listOverlays(t, st, v2.PlanID, 2)
	machine := 0
	for _, o := range ovs {
		if kernel.IsMachineOverlayOrigin(o.Origin) {
			machine++
			if o.OverlayID != "picture:"+ev.OppKey {
				t.Fatalf("re-append overlay id = %q", o.OverlayID)
			}
		}
	}
	if machine != 1 {
		t.Fatalf("exactly one machine overlay on v2, got %d (%+v)", machine, ovs)
	}
	doc, _ := resolveActivePlanDoc(st, v2)
	got, ok := kernel.MachineScenarioByRef(doc, ev.OppKey)
	if !ok || doc.Scenarios[len(doc.Scenarios)-1].ID != "P1" {
		t.Fatalf("P1 must be in v2's plan_final, last: %+v", doc.Scenarios)
	}
	a, _ := json.Marshal(p1)
	b, _ := json.Marshal(got)
	if string(a) != string(b) {
		t.Fatalf("the re-appended scenario must be the SAME value:\n v1 %s\n v2 %s", a, b)
	}
}

// A placed (still working) Picture scenario rides into v2 and its ledger row
// stays the ONE row of the opportunity; a finished one (filled) is never
// re-offered; one whose window closed is not carried either.
func TestReappendRespectsTheLedgerAndTheWindow(t *testing.T) {
	cases := map[string]struct {
		state    string
		unplaced bool // the row never reached the broker (no BeginPlacement)
		clockOff time.Duration
		carried  bool
	}{
		"working (placed) → carried": {state: store.StateWorking, clockOff: 2 * time.Second, carried: true},
		"filled → not carried":       {state: store.StateFilled, clockOff: 2 * time.Second, carried: false},
		// F3 (CTO pre-review): v1's P1 UNPLACED and AI v2 active without it.
		"unplaced, window closed → not carried":       {state: "", clockOff: time.Minute, carried: false},
		"unplaced, ledger row terminal → not carried": {state: store.StateCancelled, unplaced: true, clockOff: 2 * time.Second, carried: false},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			at, st := handOffTrader(t)
			schedulerTape(t)
			now := handOffNow()
			pinTraderNow(t, now.Add(c.clockOff))
			ev, _ := machinePlanV1(t, at, st, now)
			pid := store.MakePlanIDForTrader(at.id, handOffDate, "NY")
			var armID int64
			if c.state != "" {
				arm := &store.ArmedOrderDB{TraderID: at.id, PlanID: pid, Version: 1, Session: "NY", Scenario: "P1", Side: "LONG", Kind: "limit",
					EntryPx: 21530, StopPx: 21510, TargetPx: 21590, State: store.StateArmed, Policy: store.ArmPolicyMarketInZone,
					Source: store.ArmSourcePicture, SourceRef: ev.OppKey, SourceRule: kernel.MachineRulePictureH1CloseBreak}
				if err := st.ArmedOrders().UpsertArm(arm); err != nil {
					t.Fatal(err)
				}
				armID = arm.ID
				if !c.unplaced {
					if err := st.ArmedOrders().BeginPlacementEval(armID, "sig-p1", 21530, now.UnixMilli()-60_000, now.UnixMilli()); err != nil {
						t.Fatal(err)
					}
				}
				if err := st.ArmedOrders().SetState(armID, c.state, "fixture"); err != nil {
					t.Fatal(err)
				}
			}
			at.maybeRunSessionReadsAt(now)
			defer drainReReads(t)
			v2 := waitVersion(t, st, 2)
			if v2 == nil {
				t.Fatal("v2 never landed")
			}
			drainReReads(t)
			doc, _ := resolveActivePlanDoc(st, v2)
			_, inV2 := kernel.MachineScenarioByRef(doc, ev.OppKey)
			if inV2 != c.carried {
				t.Fatalf("P1 in v2 = %v, want %v", inV2, c.carried)
			}
			if c.state == store.StateWorking {
				// The executor authors v2's P1 under the SAME opportunity: the
				// ledger keeps ONE row, still working, still its signal.
				next := &store.ArmedOrderDB{TraderID: at.id, PlanID: pid, Version: 2, Session: "NY", Scenario: "P1", Side: "LONG", Kind: "limit",
					EntryPx: 21530, StopPx: 21510, TargetPx: 21590, State: store.StateArmed, Policy: store.ArmPolicyMarketInZone,
					Source: store.ArmSourcePicture, SourceRef: ev.OppKey, SourceRule: kernel.MachineRulePictureH1CloseBreak}
				// A working row is a live broker order: the store declines to
				// rewrite it (D5) — and never mints a second row beside it.
				if err := st.ArmedOrders().UpsertArm(next); err != nil && !strings.Contains(err.Error(), "refusing to rewrite") {
					t.Fatal(err)
				}
				var mine []store.ArmedOrderDB
				for _, r := range mustListForPlan(t, st, pid) {
					if r.SourceRef == ev.OppKey {
						mine = append(mine, r)
					}
				}
				if len(mine) != 1 || mine[0].ID != armID || mine[0].SignalID != "sig-p1" || mine[0].State != store.StateWorking {
					t.Fatalf("one opportunity = one order across versions: %+v", mine)
				}
			}
		})
	}
}

// A NO-TRADE version (the planner failed closed) gets no Picture scenario:
// the Day Plan said no.
func TestReappendNeverLandsOnANoTradeVersion(t *testing.T) {
	at, st := handOffTrader(t)
	schedulerTape(t)
	now := handOffNow()
	pinTraderNow(t, now.Add(2*time.Second))
	ev, _ := machinePlanV1(t, at, st, now)
	at.mcpClient = &fakeDecisionClient{} // every attempt returns nothing → fail-closed
	at.maybeRunSessionReadsAt(now)
	defer drainReReads(t)
	v2 := waitVersion(t, st, 2)
	if v2 == nil || v2.Lifecycle != "no_trade" {
		t.Fatalf("fixture: a fail-closed NO-TRADE v2 expected, got %+v", v2)
	}
	drainReReads(t)
	for _, o := range listOverlays(t, st, v2.PlanID, 2) {
		if strings.Contains(o.OverlayID, ev.OppKey) || kernel.IsMachineOverlayOrigin(o.Origin) {
			t.Fatalf("a NO-TRADE version must never carry a Picture scenario: %+v", o)
		}
	}
}

// The AI decision path stays exactly where "no plan" left it while only a
// machine plan exists: entries refused, and no death predicate runs on it.
func TestExecutorPlanDeadReasonTreatsAMachinePlanAsNoAIPlan(t *testing.T) {
	at, st := handOffTrader(t)
	now := handOffNow()
	pinTraderNow(t, now)
	machinePlanV1(t, at, st, now)
	got := at.executorPlanDeadReason()
	if !strings.Contains(got, "no AI day plan for this session yet") {
		t.Fatalf("a machine plan must refuse AI decision entries like no plan does, got %q", got)
	}
}

// countingPlanClient is the planner's model: it counts calls and holds each
// one until released, so a test can keep ONE read in flight.
type countingPlanClient struct {
	planClient
	calls   atomic.Int32
	release chan struct{}
}

func (c *countingPlanClient) CallWithMessages(_, user string) (string, error) {
	c.calls.Add(1)
	<-c.release
	return mapCompliantPlanJSON(user), nil
}

// F2 (CTO pre-review) — while the newest row is a machine plan the scheduler
// fires `go at.runPlannerRead` on EVERY tick, exactly like the no-plan branch.
// The in-flight guard that makes that ONE planner call is claimPlannerRead
// (runPlannerReadWithTriggerClaimedCtx, keyed MakePlanIDForTrader(trader,
// date, session), process-wide plannerReadInFlight): a second tick during the
// read is refused before any client call.
func TestTwoTicksOnAMachineOnlyChainMakeOnePlannerCall(t *testing.T) {
	at, st := handOffTrader(t)
	schedulerTape(t)
	now := handOffNow()
	pinTraderNow(t, now.Add(2*time.Second))
	machinePlanV1(t, at, st, now)
	c := &countingPlanClient{release: make(chan struct{})}
	var once sync.Once
	release := func() { once.Do(func() { close(c.release) }) }
	at.mcpClient = c
	defer drainReReads(t)
	defer release()

	if fired := at.maybeRunSessionReadsAt(now); len(fired) != 1 {
		t.Fatalf("tick 1 fires the read on a machine-only chain, got %+v", fired)
	}
	deadline := time.Now().Add(5 * time.Second)
	for c.calls.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if c.calls.Load() != 1 {
		t.Fatalf("fixture: the first read must be in flight at the model, calls=%d", c.calls.Load())
	}
	key := store.MakePlanIDForTrader(at.id, handOffDate, "NY")
	if _, held := plannerReadInFlight.Load(key); !held {
		t.Fatalf("the in-flight guard must hold %q while the read runs", key)
	}
	// Tick 2 during the read: the chain is still machine-only, so the
	// scheduler fires again — and the guard makes it no call.
	if fired := at.maybeRunSessionReadsAt(now.Add(time.Second)); len(fired) != 1 {
		t.Fatalf("tick 2 still fires (the chain is still machine-only), got %+v", fired)
	}
	for end := time.Now().Add(time.Second); time.Now().Before(end); time.Sleep(10 * time.Millisecond) {
		if n := c.calls.Load(); n != 1 {
			t.Fatalf("two ticks during one in-flight read made %d planner calls — exactly one expected", n)
		}
	}
	release()
	if v2 := waitVersion(t, st, 2); v2 == nil || store.IsMachinePlan(v2) {
		t.Fatalf("the one read lands v2: %+v", v2)
	}
	drainReReads(t)
	if n := c.calls.Load(); n != 1 {
		t.Fatalf("exactly one planner call in total, got %d", n)
	}
	if latest, _ := st.Plan().GetLatestPlanForTraderSession(handOffDate, "NY", at.id); latest.Version != 2 {
		t.Fatalf("one read → one version, got v%d", latest.Version)
	}
}

// F6 (CTO pre-review) — the readers that act on the plan's bias or levels are
// INERT on a machine plan (bias neutral, levels []), each driven with the
// machine plan the provider serves; the same readers act on an AI plan
// (the control), so "inert" is the machine doc's doing, not a dead reader.
func TestMachinePlanIsInertToTheBiasAndLevelReaders(t *testing.T) {
	at, st := handOffTrader(t)
	now := handOffNow()
	pinTraderNow(t, now)
	_, p1 := machinePlanV1(t, at, st, now)
	installActivePlanProviderAt(at, st, func() time.Time { return now })
	t.Cleanup(func() { kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{}) })
	plan := kernel.ActivePlanFor(at.id, at.futuresSymbol())
	if plan == nil || plan.Doc.Bias.Direction != "neutral" || len(plan.Doc.Levels) != 0 {
		t.Fatalf("fixture: the provider serves the machine plan (neutral, no levels), got %+v", plan)
	}
	after := time.Now().Add(time.Minute).UnixMilli() // born after the plan row
	counterEvents := func() *kernel.Context {
		return &kernel.Context{Structure: map[string]kernel.StructureState{"15m": {LastEvents: []kernel.StructureEvent{
			{Type: "CHoCH", Dir: "down", Price: 21500, TimeMs: after}, {Type: "MSS", Dir: "up", Price: 21540, TimeMs: after},
		}}}}
	}
	matched := func() int {
		counts, err := st.MatchedRandom().CountsByType()
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, c := range counts {
			n += c.Touches
		}
		return n
	}
	closed := &store.TraderPosition{EntryPrice: 15600, EntryTime: now.UnixMilli()}

	t.Run("transition stand-down", func(t *testing.T) {
		ctx := counterEvents()
		at.observeTransitionStanddownAt(now, ctx)
		if ctx.TransitionActive || at.transition.Active {
			t.Fatalf("a neutral machine plan never opens a transition stand-down: %+v", at.transition)
		}
	})
	t.Run("HTF veto", func(t *testing.T) {
		// The machine plan's bias vetoes nothing; the veto reads only the
		// structure, exactly as for any scenario (gate parity, D12).
		if v := at.armGateVerdict(p1, biasDirectionFor(plan.Doc.Bias.Direction), nil, 6.5, "", at.config.StrategyConfig, "NY"); v != "" {
			t.Fatalf("P1 under the machine plan (advisory, no HTF trend) must pass the arm gates, got %q", v)
		}
		snap := map[string]kernel.StructureState{"1h": {Trend: "TRENDING_DOWN"}, "4h": {Trend: "TRENDING_DOWN"}}
		if v := at.armGateVerdict(p1, biasDirectionFor(plan.Doc.Bias.Direction), snap, 6.5, "", at.config.StrategyConfig, "NY"); !strings.HasPrefix(v, "HTF veto") {
			t.Fatalf("the HTF veto still reads the structure for P1 (parity), got %q", v)
		}
	})
	t.Run("matched-random", func(t *testing.T) {
		at.recordMatchedRandomForClose(closed, kernel.Excursion{MFE: 10, MAE: 2})
		if n := matched(); n != 0 {
			t.Fatalf("a machine plan (no levels) records no matched-random touch, got %d", n)
		}
	})

	// CONTROL: an AI plan (bias long, levels) supersedes it — the same readers act.
	if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: store.MakePlanIDForTrader(at.id, handOffDate, "NY"), StrategyID: at.id,
		TradeDate: handOffDate, Session: "NY", TriggerReason: "NY_scheduled_read", Lifecycle: "active", Doc: validTraderPlanJSON}); err != nil {
		t.Fatal(err)
	}
	ctx := counterEvents()
	at.observeTransitionStanddownAt(now, ctx)
	if !ctx.TransitionActive {
		t.Fatal("control: the same counter-trend events open a stand-down on a long AI plan — the reader is live")
	}
	at.recordMatchedRandomForClose(closed, kernel.Excursion{MFE: 10, MAE: 2})
	if n := matched(); n != 1 {
		t.Fatalf("control: an AI plan with levels records the touch, got %d", n)
	}
}
