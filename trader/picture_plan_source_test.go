package trader

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"nofx/kernel"
	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// ── W-EXEC-TRUTH W5 (builder A) — Picture is a Day Plan scenario SOURCE ─────
//
// Every test drives a production call site: the package-level seam the
// evaluator calls (pictureHtfSubmitSeam), pictureHandOffAt, the executor's
// fold (resolveActivePlanDoc), the D17 sweep. The fixture is W5's own (on the
// MNQ tick grid, no W4-owned field), so W4's evaluator changes cannot move it.

const handOffTraderID = "t-w5a"

// handOffNow is Monday 2026-09-14 10:00:00 CT — inside the NY session.
func handOffNow() time.Time {
	return time.Date(2026, 9, 14, 10, 0, 0, 0, kernel.CTLocation())
}

const handOffDate = "2026-09-14"

func handOffTrader(t *testing.T) (*AutoTrader, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "w5a.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })
	if err := st.MigratePictureHtf(); err != nil {
		t.Fatal(err)
	}
	cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	cfg.RiskControl.MinRiskRewardRatio = 2
	at := &AutoTrader{id: handOffTraderID, exchange: "ninjatrader", store: st}
	at.config.StrategyConfig = &cfg
	at.config.NinjaTraderSymbol = "MNQ"
	at.mcpClient = &planClient{}
	t.Cleanup(func() { at.clearPictureRunEpoch() })
	return at, st
}

// handOffOppKey is an opportunity key shaped like the evaluator's.
func handOffOppKey(n int) string {
	return fmt.Sprintf("%s|sim101|mnq 12-26|long|resistance|1789920000000|%d", handOffTraderID, 1789999200000+int64(n)*3600000)
}

// handOffEvidence is one long Picture opportunity on the MNQ tick grid:
// zone [21528.50, 21530.00], stop 21510, target 21590 → R:R 3.0 at the far
// edge 21530 against a floor of 2.
func handOffEvidence(now time.Time, n int) PictureEvidence {
	return PictureEvidence{
		OppKey: handOffOppKey(n), ClaimID: fmt.Sprintf("picture-htf-%d", now.UnixMilli()+int64(n)),
		TraderID: handOffTraderID, StrategyID: "s1", Contract: "MNQ 12-26", Symbol: "MNQ", Direction: "long",
		Rule: kernel.MachineRulePictureH1CloseBreak, RuleVer: 1,
		LevelRole: "resistance", BodyTop: 21520, BodyBot: 21500, WickHi: 21526, WickLo: 21494,
		LevelBarOpenMs: 1789920000000, LevelKnowableMs: 1789934400000,
		H1PrevClose: 21515, H1NewClose: 21531, H1Boundary: 21520,
		H1OpenMs: now.Add(-time.Hour).UnixMilli(), H1CloseMs: now.Add(-time.Millisecond).UnixMilli(),
		EntryRef: 21530, LatestClose: 21528.5, Stop: 21510, Target: 21590, ATR5m: 6.5,
		StopSource: "5m swing low", TargetZone: "4H body 21585-21600", RREstimate: 3, RRFloor: 2, KnobMinRR: 2,
		WindowOpenMs: now.UnixMilli(), WindowCloseMs: now.Add(10 * time.Second).UnixMilli(), EvalAtMs: now.UnixMilli(),
	}
}

// claimHandOff writes the evaluator's claim for ev: the opportunity row,
// confirmed → place_pending with the synthetic claim id, unstamped.
func claimHandOff(t *testing.T, st *store.Store, ev PictureEvidence) {
	t.Helper()
	if _, ok, err := st.PictureHtfClaim(&store.PictureHtfOpportunityDB{OppKey: ev.OppKey, TraderID: ev.TraderID, Account: "Sim101",
		Symbol: ev.Symbol, Direction: ev.Direction, Stage: "confirmed", WindowOpen: ev.WindowOpenMs, WindowClose: ev.WindowCloseMs}); err != nil || !ok {
		t.Fatalf("claim: ok=%v err=%v", ok, err)
	}
	if won, err := st.PictureHtfClaimSubmission(ev.OppKey, ev.ClaimID); err != nil || !won {
		t.Fatalf("claim submission: won=%v err=%v", won, err)
	}
}

// seedAIPlan writes an ACTIVE AI plan v1 for the NY chain of handOffDate.
func seedAIPlan(t *testing.T, st *store.Store, lifecycle string) *store.PlanDB {
	t.Helper()
	row := &store.PlanDB{PlanID: store.MakePlanIDForTrader(handOffTraderID, handOffDate, "NY"), StrategyID: handOffTraderID,
		TradeDate: handOffDate, Session: "NY", TriggerReason: "NY_scheduled_read", Lifecycle: lifecycle, ModelID: "m", Doc: validTraderPlanJSON}
	if _, err := st.Plan().AppendPlan(row); err != nil {
		t.Fatal(err)
	}
	return row
}

func pictureRow(t *testing.T, st *store.Store, key string) *store.PictureHtfOpportunityDB {
	t.Helper()
	row, ok, err := st.PictureHtfGet(key)
	if err != nil || !ok {
		t.Fatalf("picture row %s: ok=%v err=%v", key, ok, err)
	}
	return row
}

func listOverlays(t *testing.T, st *store.Store, planID string, v int) []*store.PlanOverlayDB {
	t.Helper()
	ovs, err := st.Plan().ListOverlays(planID, v)
	if err != nil {
		t.Fatal(err)
	}
	return ovs
}

// ── the seam ──────────────────────────────────────────────────────────────────

// handOffWire puts a REAL NT8 TCP trader behind the AutoTrader and returns a
// barrier that reports every signal frame the producer wrote so far.
func handOffWire(t *testing.T, at *AutoTrader) func() []ntwire.SignalPayload {
	t.Helper()
	s := ntwire.NewTCPServer(nil)
	s.SetAddrForTest("127.0.0.1:0")
	s.SetAccountsList([]ntwire.AccountInfo{{Name: "Sim101", IsSim: true}}, "Sim101")
	ctx, cancel := context.WithCancel(context.Background())
	if err := s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Stop(); cancel() })
	conn, err := net.Dial("tcp", s.ListenAddrForTest().String())
	if err != nil {
		t.Fatal(err)
	}
	waitAddonRegistered(t, s)
	t.Cleanup(func() { _ = conn.Close() })
	type frame struct {
		sig      *ntwire.SignalPayload
		sentinel string
	}
	ev := make(chan frame, 64)
	go func() {
		for {
			env, err := ntwire.ReadFrame(conn)
			if err != nil {
				return
			}
			switch env.Type {
			case ntwire.FrameSignal:
				var p ntwire.SignalPayload
				if json.Unmarshal(env.Payload, &p) == nil {
					ev <- frame{sig: &p}
				}
			case ntwire.FrameBarsHistoryRequest:
				var p ntwire.BarsHistoryRequestPayload
				if json.Unmarshal(env.Payload, &p) == nil && p.Symbol == "W5A-SENTINEL" {
					ev <- frame{sentinel: p.RequestID}
				}
			}
		}
	}()
	at.trader = ntTrader.NewTCPTrader(s, "MNQ", "Sim101")
	seq := 0
	return func() []ntwire.SignalPayload {
		t.Helper()
		seq++
		id := fmt.Sprintf("w5a-%d", seq)
		if err := s.SendBarsHistoryRequest(ntwire.BarsHistoryRequestPayload{RequestID: id, Symbol: "W5A-SENTINEL"}); err != nil {
			t.Fatalf("sentinel: %v", err)
		}
		var sigs []ntwire.SignalPayload
		deadline := time.After(3 * time.Second)
		for {
			select {
			case f := <-ev:
				if f.sig != nil {
					sigs = append(sigs, *f.sig)
				}
				if f.sentinel == id {
					return sigs
				}
			case <-deadline:
				t.Fatal("sentinel never came back")
				return nil
			}
		}
	}
}

// THE PRODUCTION BINDING: the seam the evaluator calls IS the hand-off (the
// init order puts it after picture_htf_send.go's binding), and calling it
// records a Day Plan scenario while nothing — not one frame — reaches NT8.
func TestPictureSeamIsTheHandOffAndNeverSends(t *testing.T) {
	if reflect.ValueOf(pictureHtfSubmitSeam).Pointer() != reflect.ValueOf(pictureHtfHandOffSeam).Pointer() {
		t.Fatal("the production pictureHtfSubmitSeam must be the Day Plan hand-off (picture_plan_source.go init), not the retired market-entry send")
	}
	at, st := handOffTrader(t)
	signals := handOffWire(t, at)
	now := handOffNow()
	at.markPictureRunEpoch(now)
	ev := handOffEvidence(now, 1)
	claimHandOff(t, st, ev)
	e := &PictureHtfEvaluator{at: at, pendingAdmission: &pictureAdmission{
		EntryRef: ev.EntryRef, LatestClose: ev.LatestClose, Stop: ev.Stop, Target: ev.Target, ATR5m: ev.ATR5m, KnobMinRR: 2}}
	row := &store.PictureHtfOpportunityDB{OppKey: ev.OppKey, SignalID: ev.ClaimID, TraderID: at.id, Symbol: "MNQ", Direction: "long", RuleVer: 1,
		LevelBodyTop: ev.BodyTop, LevelBodyBot: ev.BodyBot, H1NewClose: ev.H1NewClose, H1Boundary: ev.H1Boundary,
		WindowOpen: ev.WindowOpenMs, WindowClose: ev.WindowCloseMs}
	if err := pictureHtfSubmitSeam(e, row, ev.Stop, ev.Target, 0, now); err != nil {
		t.Fatalf("the hand-off must record the opportunity: %v", err)
	}
	if sigs := signals(); len(sigs) != 0 {
		t.Fatalf("the Picture seam must never send — %d signal frame(s) reached NT8: %+v", len(sigs), sigs)
	}
	plan, _ := st.Plan().GetLatestPlanForTraderSession(handOffDate, "NY", at.id)
	if plan == nil || !store.IsMachinePlan(plan) {
		t.Fatalf("no plan existed → the hand-off records a machine plan, got %+v", plan)
	}
	doc, _ := resolveActivePlanDoc(st, plan)
	if sc, ok := kernel.MachineScenarioByRef(doc, ev.OppKey); !ok || sc.ID != "P1" {
		t.Fatalf("the machine scenario must be in plan_final: %+v", doc.Scenarios)
	}
	if got := pictureRow(t, st, ev.OppKey); got.Stage != store.PictureStagePlanned {
		t.Fatalf("the opportunity row settles planned, got %q (%s)", got.Stage, got.StageReason)
	}
}

// ── (a) an ACTIVE plan gets one machine overlay ─────────────────────────────

func TestPictureHandOffOverlaysAnActivePlan(t *testing.T) {
	at, st := handOffTrader(t)
	now := handOffNow()
	epoch := at.markPictureRunEpoch(now)
	plan := seedAIPlan(t, st, "active")
	ev := handOffEvidence(now, 1)
	claimHandOff(t, st, ev)
	if err := at.pictureHandOffAt(ev, now); err != nil {
		t.Fatalf("hand-off: %v", err)
	}
	// Never a plan VERSION (F4): the AI chain stays at v1.
	if latest, _ := st.Plan().GetLatestPlanForTraderSession(handOffDate, "NY", at.id); latest.Version != 1 || store.IsMachinePlan(latest) {
		t.Fatalf("an overlay never bumps the AI chain's version: %+v", latest)
	}
	ovs := listOverlays(t, st, plan.PlanID, 1)
	if len(ovs) != 1 {
		t.Fatalf("exactly one overlay row, got %d", len(ovs))
	}
	ov := ovs[0]
	if ov.Origin != kernel.MachineOverlayOriginPicture || ov.OverlayID != "picture:"+ev.OppKey || ov.OverlayVersion != 1 {
		t.Fatalf("overlay row = origin %q id %q v%d", ov.Origin, ov.OverlayID, ov.OverlayVersion)
	}
	sc, err := kernel.MachineScenarioFromPatch(ov.Patch)
	if err != nil {
		t.Fatalf("the overlay is exactly one add /scenarios/- of the scenario: %v", err)
	}
	if sc.ID != "P1" || sc.Source != kernel.ScenarioSourcePicture || sc.Condition != "acceptance" || sc.Direction != "long" ||
		sc.Quality != "B" || sc.Confirm != nil || sc.Confirm2 != nil ||
		sc.Arm == nil || !sc.Arm.Enabled || sc.Arm.Policy != kernel.EntryPolicyMarketInZone || sc.Arm.WaitConfirm ||
		sc.Arm.Entry != 21530 || sc.Arm.Stop != 21510 || sc.Arm.Target != 21590 ||
		sc.Economics == nil || len(sc.Economics.EntryZone) != 2 || sc.Economics.EntryZone[0] != 21528.5 || sc.Economics.EntryZone[1] != 21530 ||
		len(sc.TargetChain) != 1 || sc.TargetChain[0] != 21590 {
		t.Fatalf("scenario body = %+v arm=%+v econ=%+v", sc, sc.Arm, sc.Economics)
	}
	m := sc.Machine
	if m == nil || m.Rule != kernel.MachineRulePictureH1CloseBreak || m.RuleVer != 1 || m.Ref != ev.OppKey ||
		m.EligibleFromMs != ev.EvalAtMs || m.EligibleUntilMs != ev.WindowCloseMs || m.RunEpoch != epoch {
		t.Fatalf("machine record = %+v", m)
	}
	var frozen PictureEvidence
	if err := json.Unmarshal(m.Evidence, &frozen); err != nil || frozen.OppKey != ev.OppKey || frozen.H1NewClose != ev.H1NewClose || frozen.RRFloor != ev.RRFloor {
		t.Fatalf("the evidence is frozen into the record: %+v err=%v", frozen, err)
	}
	// The executor's fold: the AI scenarios first, P1 LAST.
	doc, ok := resolveActivePlanDoc(st, plan)
	if !ok || len(doc.Scenarios) != 2 || doc.Scenarios[0].ID != "S1" || doc.Scenarios[1].ID != "P1" {
		t.Fatalf("plan_final = %+v", doc.Scenarios)
	}
	got := pictureRow(t, st, ev.OppKey)
	if got.Stage != store.PictureStagePlanned || got.StageReason != fmt.Sprintf("Day Plan scenario P1 · %s v1 · o1", plan.PlanID) {
		t.Fatalf("picture row = %q %q", got.Stage, got.StageReason)
	}
	// A second opportunity joins as P2.
	ev2 := handOffEvidence(now, 2)
	claimHandOff(t, st, ev2)
	if err := at.pictureHandOffAt(ev2, now); err != nil {
		t.Fatal(err)
	}
	doc, _ = resolveActivePlanDoc(st, plan)
	if len(doc.Scenarios) != 3 || doc.Scenarios[2].ID != "P2" || doc.Scenarios[2].Machine.Ref != ev2.OppKey {
		t.Fatalf("the second opportunity is P2: %+v", doc.Scenarios)
	}
}

// ── (b) no plan row → a machine plan v1 ─────────────────────────────────────

func TestPictureHandOffWritesAMachinePlanWhenThereIsNone(t *testing.T) {
	at, st := handOffTrader(t)
	now := handOffNow()
	at.markPictureRunEpoch(now)
	ev := handOffEvidence(now, 1)
	claimHandOff(t, st, ev)
	if err := at.pictureHandOffAt(ev, now); err != nil {
		t.Fatalf("hand-off: %v", err)
	}
	row, _ := st.Plan().GetLatestPlanForTraderSession(handOffDate, "NY", at.id)
	if row == nil || row.Version != 1 || !store.IsMachinePlan(row) || row.TriggerReason != kernel.MachinePlanTriggerPicture ||
		row.ModelID != kernel.MachinePlanModelID || row.Lifecycle != "active" || row.PlanID != store.MakePlanIDForTrader(at.id, handOffDate, "NY") {
		t.Fatalf("machine plan row = %+v", row)
	}
	var doc kernel.PlanDoc
	if err := json.Unmarshal([]byte(row.Doc), &doc); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(doc.Reasoning, "MACHINE-AUTHORED (Picture HTF rule v1) — no AI plan existed for NY at 10:00 CT; superseded by the first AI plan") ||
		doc.Bias.Direction != "neutral" || doc.DeathCondition != "machine plan: ends when an AI plan is written" ||
		doc.Levels == nil || len(doc.Levels) != 0 || len(doc.Scenarios) != 1 || doc.Scenarios[0].ID != "P1" || doc.Scenarios[0].Machine.Ref != ev.OppKey {
		t.Fatalf("machine plan doc = %+v", doc)
	}
	if err := kernel.ValidatePlanDocWithCaps(&doc, kernel.PlanHardMaxLevels, kernel.PlanHardMaxScenarios); err != nil {
		t.Fatalf("the machine plan must be a valid plan: %v", err)
	}
	if ovs := listOverlays(t, st, row.PlanID, 1); len(ovs) != 0 {
		t.Fatalf("the no-plan door writes the plan, not an overlay: %d overlays", len(ovs))
	}
	if got := pictureRow(t, st, ev.OppKey); got.Stage != store.PictureStagePlanned || got.StageReason != fmt.Sprintf("Day Plan scenario P1 · %s v1 · machine plan", row.PlanID) {
		t.Fatalf("picture row = %q %q", got.Stage, got.StageReason)
	}
	// A second opportunity while the machine plan is the chain: an overlay
	// on it, P2 (the machine plan is ACTIVE).
	ev2 := handOffEvidence(now, 2)
	claimHandOff(t, st, ev2)
	if err := at.pictureHandOffAt(ev2, now); err != nil {
		t.Fatal(err)
	}
	final, _ := resolveActivePlanDoc(st, row)
	if len(final.Scenarios) != 2 || final.Scenarios[1].ID != "P2" {
		t.Fatalf("second opportunity on the machine plan: %+v", final.Scenarios)
	}
	if latest, _ := st.Plan().GetLatestPlanForTraderSession(handOffDate, "NY", at.id); latest.Version != 1 {
		t.Fatalf("the machine plan is only ever v1, got v%d", latest.Version)
	}
}

// ── refusals: nothing recorded, the evaluator's "never sent" contract ──────

func TestPictureHandOffRefusesWhenTheDayPlanSaysNo(t *testing.T) {
	for _, lc := range []string{"dormant", "no_trade", "died", "expired", "superseded"} {
		t.Run(lc, func(t *testing.T) {
			at, st := handOffTrader(t)
			now := handOffNow()
			at.markPictureRunEpoch(now)
			plan := seedAIPlan(t, st, lc)
			ev := handOffEvidence(now, 1)
			claimHandOff(t, st, ev)
			err := at.pictureHandOffAt(ev, now)
			if err == nil || !strings.Contains(err.Error(), "Day Plan says no ("+lc+")") {
				t.Fatalf("a %s plan must refuse 'Day Plan says no (%s)', got %v", lc, lc, err)
			}
			if ovs := listOverlays(t, st, plan.PlanID, 1); len(ovs) != 0 {
				t.Fatalf("nothing may be recorded on a %s plan", lc)
			}
			if latest, _ := st.Plan().GetLatestPlanForTraderSession(handOffDate, "NY", at.id); latest.Version != 1 || store.IsMachinePlan(latest) {
				t.Fatalf("no machine plan on top of a %s plan: %+v", lc, latest)
			}
			if got := pictureRow(t, st, ev.OppKey); got.Stage != store.StatePlacePending {
				t.Fatalf("a refused hand-off never settles planned: %q", got.Stage)
			}
		})
	}
}

func TestPictureHandOffRefusesWithoutARunOrARunnableSession(t *testing.T) {
	now := handOffNow()
	cases := map[string]struct {
		prep func(at *AutoTrader)
		at   time.Time
		want string
	}{
		"no run epoch":          {prep: func(at *AutoTrader) {}, at: now, want: "trader not running"},
		"stopped":               {prep: func(at *AutoTrader) { at.markPictureRunEpoch(now); at.clearPictureRunEpoch() }, at: now, want: "trader not running"},
		"day plan off":          {prep: func(at *AutoTrader) { at.markPictureRunEpoch(now); at.config.StrategyConfig.DayPlan.PlanEnabled = false }, at: now, want: "Day Plan is off"},
		"no live session":       {prep: func(at *AutoTrader) { at.markPictureRunEpoch(now) }, at: time.Date(2026, 9, 14, 15, 30, 0, 0, kernel.CTLocation()), want: "no session is live"},
		"session not runnable":  {prep: func(at *AutoTrader) { at.markPictureRunEpoch(now) }, at: time.Date(2026, 9, 14, 3, 0, 0, 0, kernel.CTLocation()), want: "not runnable"},
		"window already closed": {prep: func(at *AutoTrader) { at.markPictureRunEpoch(now) }, at: now, want: "eligibility window closed"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			at, st := handOffTrader(t)
			c.prep(at)
			ev := handOffEvidence(c.at, 1)
			if name == "window already closed" {
				ev.EvalAtMs = ev.WindowCloseMs + 1
			}
			claimHandOff(t, st, ev)
			err := at.pictureHandOffAt(ev, c.at)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want a refusal containing %q, got %v", c.want, err)
			}
			if rows, _ := st.Plan().ListRecentForTrader(at.id, 10); len(rows) != 0 {
				t.Fatalf("a refused hand-off records nothing: %d plan rows", len(rows))
			}
			if got := pictureRow(t, st, ev.OppKey); got.Stage != store.StatePlacePending {
				t.Fatalf("a refused hand-off never settles planned: %q", got.Stage)
			}
		})
	}
}

func TestPictureHandOffRefusesAZoneTooWideOrAnRRBelowTheFloor(t *testing.T) {
	now := handOffNow()
	for name, mut := range map[string]func(*PictureEvidence){
		"zone wider than zone_max_pts": func(ev *PictureEvidence) { ev.LatestClose = ev.EntryRef - 12 },
		"R:R at the far edge below the floor": func(ev *PictureEvidence) {
			ev.RRFloor = 3.5 // 60/20 = 3.0 at the far edge 21530
		},
		"bracket inside the zone": func(ev *PictureEvidence) { ev.Stop = 21529 },
	} {
		t.Run(name, func(t *testing.T) {
			at, st := handOffTrader(t)
			at.markPictureRunEpoch(now)
			seedAIPlan(t, st, "active")
			ev := handOffEvidence(now, 1)
			mut(&ev)
			claimHandOff(t, st, ev)
			if err := at.pictureHandOffAt(ev, now); err == nil {
				t.Fatal("must refuse")
			}
			if ovs := listOverlays(t, st, store.MakePlanIDForTrader(at.id, handOffDate, "NY"), 1); len(ovs) != 0 {
				t.Fatal("a refused scenario is never recorded")
			}
		})
	}
	// The floor is judged at the FAR edge, not the reference: a zone whose far
	// edge takes R:R under the floor refuses although the reference passes.
	at, st := handOffTrader(t)
	at.markPictureRunEpoch(now)
	seedAIPlan(t, st, "active")
	ev := handOffEvidence(now, 1)
	ev.EntryRef, ev.LatestClose = 21528.5, 21531.5 // far 21531.5: 58.5/21.5 = 2.72; ref 21528.5: 61.5/18.5 = 3.32
	ev.RRFloor = 3
	claimHandOff(t, st, ev)
	if err := at.pictureHandOffAt(ev, now); err == nil || !strings.Contains(err.Error(), "far edge") {
		t.Fatalf("R:R is judged at the far edge: %v", err)
	}
}

// ── idempotent on the opportunity ───────────────────────────────────────────

func TestPictureHandOffIsIdempotentOnTheOpportunity(t *testing.T) {
	now := handOffNow()
	t.Run("overlay", func(t *testing.T) {
		at, st := handOffTrader(t)
		at.markPictureRunEpoch(now)
		plan := seedAIPlan(t, st, "active")
		ev := handOffEvidence(now, 1)
		claimHandOff(t, st, ev)
		var wg sync.WaitGroup
		errs := make([]error, 4)
		for i := range errs {
			wg.Add(1)
			go func(i int) { defer wg.Done(); errs[i] = at.pictureHandOffAt(ev, now) }(i)
		}
		wg.Wait()
		for _, err := range errs {
			if err != nil {
				t.Fatalf("a repeated hand-off of a recorded opportunity is not an error: %v", err)
			}
		}
		if ovs := listOverlays(t, st, plan.PlanID, 1); len(ovs) != 1 {
			t.Fatalf("four hand-offs of one opportunity → ONE overlay, got %d", len(ovs))
		}
	})
	t.Run("machine plan", func(t *testing.T) {
		at, st := handOffTrader(t)
		at.markPictureRunEpoch(now)
		ev := handOffEvidence(now, 1)
		claimHandOff(t, st, ev)
		for i := 0; i < 2; i++ {
			if err := at.pictureHandOffAt(ev, now); err != nil {
				t.Fatal(err)
			}
		}
		row, _ := st.Plan().GetLatestPlanForTraderSession(handOffDate, "NY", at.id)
		if row.Version != 1 {
			t.Fatalf("one machine plan, got v%d", row.Version)
		}
		if ovs := listOverlays(t, st, row.PlanID, 1); len(ovs) != 0 {
			t.Fatalf("the second hand-off finds P1 in the machine plan's own doc — no overlay, got %d", len(ovs))
		}
	})
}

// ── the record comes first, then the poke ───────────────────────────────────

func TestPictureHandOffPokesTheEventLoopOnlyAfterRecording(t *testing.T) {
	now := handOffNow()
	at, st := handOffTrader(t)
	l := &armedEventLoop{kick: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{})}
	at.armedEvent.Store(l)
	// A refusal (no run) records nothing and wakes nothing.
	ev := handOffEvidence(now, 1)
	claimHandOff(t, st, ev)
	if err := at.pictureHandOffAt(ev, now); err == nil {
		t.Fatal("fixture: no run epoch must refuse")
	}
	if at.zoneArmActive.Load() || len(l.kick) != 0 {
		t.Fatal("a refused hand-off must not wake the executor")
	}
	at.markPictureRunEpoch(now)
	plan := seedAIPlan(t, st, "active")
	before := time.Now()
	if err := at.pictureHandOffAt(ev, now); err != nil {
		t.Fatal(err)
	}
	if !at.zoneArmActive.Load() || len(l.kick) != 1 {
		t.Fatalf("the hand-off sets zoneArmActive and pokes the event loop (active=%v kicks=%d)", at.zoneArmActive.Load(), len(l.kick))
	}
	// Recorded BEFORE any placement can exist: the overlay row is committed by
	// the time the executor is woken, and the placement the executor makes
	// afterwards is stamped later (BeginPlacementEval's placed_at_ms).
	ov := listOverlays(t, st, plan.PlanID, 1)[0]
	if ov.CreatedAt.IsZero() || ov.CreatedAt.After(time.Now()) || ov.CreatedAt.Before(before.Add(-time.Second)) {
		t.Fatalf("overlay created_at %v must be stamped at the hand-off", ov.CreatedAt)
	}
	arm := &store.ArmedOrderDB{TraderID: at.id, PlanID: plan.PlanID, Version: 1, Scenario: "P1", Side: "LONG", Kind: "limit", EntryPx: 21530, StopPx: 21510, TargetPx: 21590,
		State: store.StateArmed, Policy: store.ArmPolicyMarketInZone, Source: store.ArmSourcePicture, SourceRef: ev.OppKey}
	if err := st.ArmedOrders().UpsertArm(arm); err != nil {
		t.Fatal(err)
	}
	placedAt := time.Now().UnixMilli()
	if err := st.ArmedOrders().BeginPlacementEval(arm.ID, "sig-p1", 21530, placedAt-60_000, placedAt); err != nil {
		t.Fatal(err)
	}
	var placed store.ArmedOrderDB
	for _, r := range mustListForPlan(t, st, plan.PlanID) {
		if r.SourceRef == ev.OppKey {
			placed = r
		}
	}
	if placed.PlacedAtMs == nil || ov.CreatedAt.UnixMilli() > *placed.PlacedAtMs {
		t.Fatalf("the scenario's record (%d) must precede its placement (%v)", ov.CreatedAt.UnixMilli(), placed.PlacedAtMs)
	}
}

func mustListForPlan(t *testing.T, st *store.Store, planID string) []store.ArmedOrderDB {
	t.Helper()
	rows, err := st.ArmedOrders().ListForPlan(planID)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// ── D17 — the interrupted hand-off sweep ─────────────────────────────────────

func TestPictureHandOffSweepSettlesFromTheRecord(t *testing.T) {
	now := handOffNow()
	t.Run("recorded but never settled → planned", func(t *testing.T) {
		at, st := handOffTrader(t)
		at.markPictureRunEpoch(now)
		plan := seedAIPlan(t, st, "active")
		ev := handOffEvidence(now, 1)
		claimHandOff(t, st, ev)
		// The settle is lost (a claim id the row does not carry): the hand-off
		// still succeeds — the scenario is recorded — and leaves the row.
		lost := ev
		lost.ClaimID = "picture-htf-lost"
		if err := at.pictureHandOffAt(lost, now); err != nil {
			t.Fatalf("after the record a hand-off never errors: %v", err)
		}
		if got := pictureRow(t, st, ev.OppKey); got.Stage != store.StatePlacePending {
			t.Fatalf("fixture: the settle must have been lost, got %q", got.Stage)
		}
		at.sweepInterruptedPictureHandOffsAt(now.Add(time.Second))
		got := pictureRow(t, st, ev.OppKey)
		if got.Stage != store.PictureStagePlanned || !strings.Contains(got.StageReason, "Day Plan scenario P1 · "+plan.PlanID+" v1 · o1") {
			t.Fatalf("the sweep settles a recorded opportunity planned: %q %q", got.Stage, got.StageReason)
		}
	})
	t.Run("never recorded, window open → left alone; closed → refused", func(t *testing.T) {
		at, st := handOffTrader(t)
		ev := handOffEvidence(now, 1)
		claimHandOff(t, st, ev)
		at.sweepInterruptedPictureHandOffsAt(now.Add(time.Second))
		if got := pictureRow(t, st, ev.OppKey); got.Stage != store.StatePlacePending {
			t.Fatalf("an open window may still be handed off — the sweep must not judge it: %q", got.Stage)
		}
		at.sweepInterruptedPictureHandOffsAt(time.UnixMilli(ev.WindowCloseMs + 1))
		got := pictureRow(t, st, ev.OppKey)
		if got.Stage != "refused" || got.StageReason != "hand-off interrupted — never sent" {
			t.Fatalf("an unrecorded claim whose window closed is refused (never sent): %q %q", got.Stage, got.StageReason)
		}
	})
	t.Run("recorded in a superseded version → planned", func(t *testing.T) {
		at, st := handOffTrader(t)
		at.markPictureRunEpoch(now)
		ev := handOffEvidence(now, 1)
		claimHandOff(t, st, ev)
		lost := ev
		lost.ClaimID = "picture-htf-lost"
		if err := at.pictureHandOffAt(lost, now); err != nil { // machine plan v1
			t.Fatal(err)
		}
		// An AI v2 lands WITHOUT the scenario (its window closed before it).
		if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: store.MakePlanIDForTrader(at.id, handOffDate, "NY"), StrategyID: at.id,
			TradeDate: handOffDate, Session: "NY", TriggerReason: "NY_scheduled_read", Doc: validTraderPlanJSON}); err != nil {
			t.Fatal(err)
		}
		at.sweepInterruptedPictureHandOffsAt(time.UnixMilli(ev.WindowCloseMs + 1))
		if got := pictureRow(t, st, ev.OppKey); got.Stage != store.PictureStagePlanned {
			t.Fatalf("a record in ANY version of the chain is a record — never refused as 'never sent': %q %q", got.Stage, got.StageReason)
		}
	})
}

// ── the executor's fold is byte-identical with no machine overlay ──────────

// legacyResolveActivePlanDoc is resolveActivePlanDoc as it stood before W5
// (auto_trader_planner.go at eb7294c9), kept verbatim as the identity oracle.
func legacyResolveActivePlanDoc(st *store.Store, row *store.PlanDB) (kernel.PlanDoc, bool) {
	var base kernel.PlanDoc
	if json.Unmarshal([]byte(row.Doc), &base) != nil {
		return kernel.PlanDoc{}, false
	}
	overlays, _ := st.Plan().ListOverlays(row.PlanID, row.Version)
	if len(overlays) == 0 {
		return base, true
	}
	patches := make([]string, 0, len(overlays))
	for _, o := range overlays {
		patches = append(patches, o.Patch)
	}
	final, _ := kernel.ApplyOverlayPatches([]byte(row.Doc), patches)
	var merged kernel.PlanDoc
	if json.Unmarshal(final, &merged) == nil && kernel.ValidatePlanDocWithCaps(&merged, kernel.PlanHardMaxLevels, kernel.PlanHardMaxScenarios) == nil {
		return merged, true
	}
	return base, true
}

// foldFixtures are the overlay shapes the old fold handled: none, a good
// owner edit, a bad (skipped) patch beside a good one, and a stack that fails
// re-validation (the base comes back).
func foldFixtures() map[string][]string {
	sc := func(i int) string {
		return fmt.Sprintf(`{"id":"S%d","trigger":"t","condition":"reclaim","direction":"long","target_chain":[15600],"invalid":"i","quality":"B"}`, i)
	}
	return map[string][]string{
		"no overlays": nil,
		"good owner level": {`[{"op":"add","path":"/levels/-","value":{"price":15600,"label":"OWN","grade":"A","instruction":"fade"}}]`},
		"bad patch skipped": {
			`[{"op":"replace","path":"/levels/99/price","value":1}]`,
			`[{"op":"replace","path":"/bias/conviction","value":"high"}]`,
		},
		"fold fails → base": {
			`[{"op":"add","path":"/scenarios/-","value":` + sc(2) + `},{"op":"add","path":"/scenarios/-","value":` + sc(3) + `},{"op":"add","path":"/scenarios/-","value":` + sc(4) + `},{"op":"add","path":"/scenarios/-","value":` + sc(5) + `},{"op":"add","path":"/scenarios/-","value":` + sc(6) + `}]`,
		},
	}
}

func TestResolveActivePlanDocIsByteIdenticalWithoutMachineOverlays(t *testing.T) {
	for name, patches := range foldFixtures() {
		t.Run(name, func(t *testing.T) {
			_, st := handOffTrader(t)
			row := seedAIPlan(t, st, "active")
			for _, p := range patches {
				if _, err := st.Plan().AppendOverlay(&store.PlanOverlayDB{PlanID: row.PlanID, PlanVersion: 1, Patch: p, Origin: "owner"}); err != nil {
					t.Fatal(err)
				}
			}
			got, ok1 := resolveActivePlanDoc(st, row)
			want, ok2 := legacyResolveActivePlanDoc(st, row)
			gb, _ := json.Marshal(got)
			wb, _ := json.Marshal(want)
			if ok1 != ok2 || string(gb) != string(wb) {
				t.Fatalf("the fold moved with no machine overlay:\n got %s\nwant %s", gb, wb)
			}
		})
	}
}

// F2 pin at the executor's fold: a FULL plan (5 scenarios, the hard cap) with
// an owner overlay and a machine scenario keeps the owner's edit active — the
// machine scenario never counts against the cap.
func TestExecutorFoldKeepsOwnerOverlaysBesideAMachineScenario(t *testing.T) {
	at, st := handOffTrader(t)
	now := handOffNow()
	at.markPictureRunEpoch(now)
	var doc kernel.PlanDoc
	if err := json.Unmarshal([]byte(validTraderPlanJSON), &doc); err != nil {
		t.Fatal(err)
	}
	for i := 2; len(doc.Scenarios) < kernel.PlanHardMaxScenarios; i++ {
		s := doc.Scenarios[0]
		s.ID = fmt.Sprintf("S%d", i)
		doc.Scenarios = append(doc.Scenarios, s)
	}
	blob, _ := json.Marshal(doc)
	row := &store.PlanDB{PlanID: store.MakePlanIDForTrader(at.id, handOffDate, "NY"), StrategyID: at.id, TradeDate: handOffDate, Session: "NY",
		TriggerReason: "NY_scheduled_read", Lifecycle: "active", Doc: string(blob)}
	if _, err := st.Plan().AppendPlan(row); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Plan().AppendOverlay(&store.PlanOverlayDB{PlanID: row.PlanID, PlanVersion: 1, Origin: "owner",
		Patch: `[{"op":"add","path":"/levels/-","value":{"price":15600,"label":"OWN","grade":"A","instruction":"fade"}}]`}); err != nil {
		t.Fatal(err)
	}
	ev := handOffEvidence(now, 1)
	claimHandOff(t, st, ev)
	if err := at.pictureHandOffAt(ev, now); err != nil {
		t.Fatalf("a full plan still takes a machine scenario: %v", err)
	}
	final, _ := resolveActivePlanDoc(st, row)
	hasOwner := false
	for _, l := range final.Levels {
		hasOwner = hasOwner || l.Label == "OWN"
	}
	if !hasOwner || len(final.Scenarios) != kernel.PlanHardMaxScenarios+1 || final.Scenarios[len(final.Scenarios)-1].ID != "P1" {
		t.Fatalf("owner overlay active + P1 last expected: owner=%v scenarios=%d", hasOwner, len(final.Scenarios))
	}
}

// F5 — the owner-edit carry never sees a machine overlay: a re-plan of a
// version carrying only a Picture scenario raises no "overlays-need-review".
func TestCarryOwnerEditsSkipsMachineOverlays(t *testing.T) {
	at, st := handOffTrader(t)
	now := handOffNow()
	at.markPictureRunEpoch(now)
	row := seedAIPlan(t, st, "active")
	ev := handOffEvidence(now, 1)
	claimHandOff(t, st, ev)
	if err := at.pictureHandOffAt(ev, now); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: row.PlanID, StrategyID: at.id, TradeDate: handOffDate, Session: "NY", TriggerReason: "structure_mss", Doc: validTraderPlanJSON}); err != nil {
		t.Fatal(err)
	}
	at.carryOwnerEditsInto(row.PlanID, 1, 2)
	for _, ov := range listOverlays(t, st, row.PlanID, 2) {
		if ov.Origin == "owner-carried" {
			t.Fatalf("a machine overlay was carried as an owner edit: %+v", ov)
		}
	}
	if blob, _ := st.GetSystemConfig(store.UncarriedEditsKey(row.PlanID, 2)); blob != "" {
		t.Fatalf("a machine overlay must never be parked as an uncarried owner edit: %s", blob)
	}
	alerts, _ := st.Alert().List(at.id, 20)
	for _, a := range alerts {
		if a.Kind == "overlays-need-review" || a.Kind == "overlays-orphaned" {
			t.Fatalf("false P1 alert for a machine overlay: %+v", a)
		}
	}
}
