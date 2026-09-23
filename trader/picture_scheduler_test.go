package trader

import (
	"encoding/json"
	"strings"
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
		clockOff time.Duration
		carried  bool
	}{
		"working (placed) → carried":  {state: store.StateWorking, clockOff: 2 * time.Second, carried: true},
		"filled → not carried":        {state: store.StateFilled, clockOff: 2 * time.Second, carried: false},
		"window closed → not carried": {state: "", clockOff: time.Minute, carried: false},
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
				if err := st.ArmedOrders().BeginPlacementEval(armID, "sig-p1", 21530, now.UnixMilli()-60_000, now.UnixMilli()); err != nil {
					t.Fatal(err)
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
