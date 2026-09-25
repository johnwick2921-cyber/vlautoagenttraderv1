package trader

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// ── W-EXEC-TRUTH W0 (Q6/Q7/Q8, defects 2/3/4) — Picture HTF through the one
// admission gate, and the three Picture defects the gate's tests need fixed ─

// admittedPictureEnv is the Picture harness with the trader RUNNING and the
// Day Plan master ON — the preconditions admitEntry checks for Picture since
// W0 (the mode used to run regardless of either, D26).
func admittedPictureEnv(t *testing.T, cfg store.PictureHtfConfig) *pictureHtfTestEnv {
	t.Helper()
	env := newPictureHtfEnv(t, cfg)
	env.at.config.StrategyConfig.DayPlan.PlanEnabled = true
	env.at.isRunningMutex.Lock()
	env.at.isRunning = true
	env.at.isRunningMutex.Unlock()
	env.seedPictureTape()
	return env
}

// DEFECT 4 — an MNQ trader never claims an opportunity from another symbol's
// frames (the live sink fans every symbol out to every trader).
func TestPictureNeverClaimsAnotherSymbolsFrames(t *testing.T) {
	env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.at.config.NinjaTraderSymbol = "MNQ"
	bars := tailOf(market.FuturesBarsProvider("ES", "5m", 28), 1)
	env.eval.OnBars("ES", "5m", bars, env.now)
	res := env.eval.Evaluate("ES", env.now)
	if len(env.submits) != 0 {
		t.Fatalf("an ES frame produced %d submission(s) on an MNQ trader", len(env.submits))
	}
	rows, _ := env.st.PictureHtfByTrader(env.at.id, 10)
	if len(rows) != 0 {
		t.Fatalf("an ES frame claimed an opportunity on an MNQ trader: %+v (res %+v)", rows, res)
	}
}

// DEFECT 3 — refuse() never overwrites a row whose send started: working,
// filled, or place_pending with a submission stamp keep their stage.
func TestPictureRefuseNeverOverwritesASentRow(t *testing.T) {
	env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	for _, tc := range []struct{ key, stage string }{{"opp-working", "working"}, {"opp-filled", "filled"}, {"opp-stamped", "place_pending"}} {
		if _, _, err := env.st.PictureHtfClaim(&store.PictureHtfOpportunityDB{OppKey: tc.key, TraderID: env.at.id, Stage: "confirmed"}); err != nil {
			t.Fatal(err)
		}
		if won, err := env.st.PictureHtfClaimSubmission(tc.key, "claim-"+tc.key); err != nil || !won {
			t.Fatalf("fixture: %v %v", won, err)
		}
		if err := env.st.PictureHtfStampSignal(tc.key, "claim-"+tc.key, "broker-"+tc.key); err != nil {
			t.Fatal(err)
		}
		if tc.stage != "place_pending" {
			if err := env.st.PictureHtfTransition(tc.key, tc.stage, "fixture"); err != nil {
				t.Fatal(err)
			}
		}
		env.eval.refuse(tc.key, "expired", "entry window passed", nil)
		row, _, _ := env.st.PictureHtfGet(tc.key)
		if row.Stage != tc.stage {
			t.Fatalf("%s: refuse() overwrote a sent row %q → %q", tc.key, tc.stage, row.Stage)
		}
	}
}

// Q7 — the R:R floor is max(Picture's own knob, the strategy floor): a knob
// BELOW the strategy floor never loosens it.
func TestPictureRRFloorIsTheStricterOfKnobAndStrategy(t *testing.T) {
	env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 1.0})
	env.at.config.StrategyConfig.RiskControl.MinRiskRewardRatio = 50 // no seeded geometry reaches 50R
	env.eval.markFresh5mReceivedAt(env.now)
	res := env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 0 || res.Stage != "refused" {
		t.Fatalf("a 1.0 knob must not loosen a 50R strategy floor: submits=%d res=%+v", len(env.submits), res)
	}
}

// D12 — no strategy config: no resolvable floor ⇒ refused (fail-closed).
func TestPictureRRFloorFailsClosedWithoutAConfig(t *testing.T) {
	env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true})
	env.at.config.StrategyConfig = nil
	env.eval.markFresh5mReceivedAt(env.now)
	env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 0 {
		t.Fatalf("no strategy config must refuse (no floor), got %d submission(s)", len(env.submits))
	}
}

// W5 (was Q6) — under plan_mode=strict Picture is NO LONGER refused: it enters
// as a Day Plan scenario, so the pre-claim admission lets it through to the
// hand-off seam, and the 📷 boot line READS the route and the plan mode —
// never a "refused under strict" text.
func TestPictureAdmittedUnderStrictAsADayPlanScenario(t *testing.T) {
	env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.at.config.StrategyConfig.DayPlan.PlanMode = "strict"
	env.eval.markFresh5mReceivedAt(env.now)
	res := env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 1 {
		t.Fatalf("strict must admit Picture to the hand-off (it is a Day Plan scenario since W5), got %d submission(s): %+v", len(env.submits), res)
	}
	line := env.at.pictureHtfBootLineAt(env.now)
	if !strings.Contains(line, "plan_gate="+PictureRouteDayPlan+" · plan_mode=strict") {
		t.Fatalf("the 📷 boot line must READ the route and the plan mode: %q", line)
	}
	if strings.Contains(line, "refused under strict") {
		t.Fatalf("the 📷 boot line must not claim a strict refusal any more: %q", line)
	}
}

// A transient admission refusal BEFORE the claim leaves no durable row: the
// same hour's opportunity trades once the gate clears (an owner pause here).
func TestPicturePreClaimRefusalIsNotDurable(t *testing.T) {
	env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.at.pauseUntilMs.Store(env.now.Add(time.Hour).UnixMilli())
	env.eval.markFresh5mReceivedAt(env.now)
	res := env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 0 || !strings.Contains(res.Reason, "stop_until") {
		t.Fatalf("an owner pause must refuse Picture before the claim: submits=%d res=%+v", len(env.submits), res)
	}
	if rows, _ := env.st.PictureHtfByTrader(env.at.id, 10); len(rows) != 0 {
		t.Fatalf("a transient refusal must not write a durable row: %+v", rows)
	}
	env.at.pauseUntilMs.Store(0)
	env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 1 {
		t.Fatalf("once the pause clears the same opportunity trades, got %d submission(s)", len(env.submits))
	}
}

// Picture is refused while the trader is stopped, and while the Day Plan
// master is off.
func TestPictureRefusedWhenStoppedOrDayPlanOff(t *testing.T) {
	env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.at.isRunningMutex.Lock()
	env.at.isRunning = false
	env.at.isRunningMutex.Unlock()
	env.eval.markFresh5mReceivedAt(env.now)
	env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 0 {
		t.Fatal("a stopped trader must not send a Picture entry")
	}
	env.at.isRunningMutex.Lock()
	env.at.isRunning = true
	env.at.isRunningMutex.Unlock()
	env.at.config.StrategyConfig.DayPlan.PlanEnabled = false
	env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 0 {
		t.Fatal("Day Plan off must not send a Picture entry")
	}
}

// A send refused BEFORE the submission stamp never reached the wire, so the
// row settles refused — never "send ambiguous" place_pending that blocks every
// later Picture entry and the installation gate.
func TestPictureUnsentSendSettlesRefused(t *testing.T) {
	env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	pictureHtfSubmitSeam = func(e *PictureHtfEvaluator, row *store.PictureHtfOpportunityDB, stopPx, targetPx, qty float64, _ time.Time) error {
		env.submits = append(env.submits, row.OppKey)
		return ntTrader.ErrEntryLatched // refused before the stamp
	}
	env.eval.markFresh5mReceivedAt(env.now)
	res := env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 1 {
		t.Fatalf("fixture: the send must have been attempted once, got %d", len(env.submits))
	}
	row, _, _ := env.st.PictureHtfGet(res.OppKey)
	if row.Stage != "refused" || !strings.Contains(row.StageReason, "never sent") {
		t.Fatalf("an unsent Picture send must settle refused (never sent), got stage=%q reason=%q", row.Stage, row.StageReason)
	}
}

// DEFECT 2 + the CTO's pin, re-pointed by W5 — the REAL path from the
// PRODUCTION seam: Evaluate claims the opportunity and the bound seam (the Day
// Plan hand-off) records it as a machine scenario; the evaluator reports the
// seam's word (F1: "planned", never "submitted" — nothing was sent), the row
// settles planned under the claim id, and NO frame reaches the wire from the
// evaluator. Then the one door to the wire, the armed pass, places the
// scenario as ONE LIMIT frame through the WIRED entry latch: the Picture
// claim never latches its own scenario's order.
func TestPictureClaimDoesNotLatchItsOwnScenario(t *testing.T) {
	prod := pictureHtfSubmitSeam
	env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	pictureHtfSubmitSeam = prod // the harness recorder out; the production hand-off in (its cleanup restores prod)
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
	waitAddonRegistered(t, s) // CTO M7: the producer must not race the accept
	t.Cleanup(func() { _ = conn.Close() })
	ev := make(chan zoneFrame, 64)
	done := make(chan struct{})
	t.Cleanup(func() { close(done) })
	go readZoneFrames(conn, ev, done)
	nt := ntTrader.NewTCPTrader(s, "MNQ", "Sim101")
	env.at.trader = nt
	env.at.config.NinjaTraderSymbol = "MNQ"
	wireNT8EntryLatch(env.at, nt)
	if !nt.EntryLatchWired() {
		t.Fatal("fixture: the entry latch must be wired as production wires it")
	}
	s.OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: []ntwire.NT8Order{}}, time.Now())
	// The Picture ladders on the tick grid, and a 1m tape whose newest close
	// (101.50) is the entry: the zone is the one price 101.50.
	passAt := env.now.Add(time.Second) // inside the 10 s eligibility window
	oneMin := zoneTape(101.5, env.now, 0)
	fourH, oneH, fiveM := pictureBars4H(), pictureBarsH1(), pictureOnGrid5M()
	market.FuturesBarsProvider = func(_ string, tf string, count int) []market.Kline {
		switch tf {
		case "4h":
			return tailOf(fourH, count)
		case "1h":
			return tailOf(oneH, count)
		case "5m":
			return tailOf(fiveM, count)
		case "1m":
			return tailOf(oneMin, count)
		}
		return nil
	}
	env.at.markPictureRunEpoch(env.now)
	t.Cleanup(env.at.clearPictureRunEpoch)
	r := &zoneRig{t: t, at: env.at, st: env.st, srv: s, conn: conn, ev: ev, now: passAt}

	env.eval.markFresh5mReceivedAt(env.now)
	res := env.eval.Evaluate("MNQ", env.now) // the PRODUCTION call site → claim → the production seam
	if res.Stage != store.PictureStagePlanned || res.Reason != "" || res.OppKey == "" {
		t.Fatalf("the evaluator must report the seam's word %q after a clean hand-off (F1 — nothing was submitted): %+v", store.PictureStagePlanned, res)
	}
	if sigs, _ := r.drain(); len(sigs) != 0 {
		t.Fatalf("the evaluator never reaches the wire — %d frame(s): %+v", len(sigs), sigs)
	}
	row, _, _ := env.st.PictureHtfGet(res.OppKey)
	if row == nil || row.Stage != store.PictureStagePlanned || row.SubmittedAt != 0 || row.SignalID != "picture-htf-"+strconv.FormatInt(env.now.UnixMilli(), 10) {
		t.Fatalf("the row settles planned under the claim id, never stamped as sent: %+v", row)
	}

	installActivePlanProviderAt(env.at, env.st, func() time.Time { return passAt })
	env.at.maybeManageArmedOrdersAt(nil, passAt) // the ONE door to the wire
	sigs, _ := r.drain()
	if len(sigs) != 1 {
		t.Fatalf("the armed pass must place the Picture scenario once — its own claim must not latch it: %d frame(s) %+v", len(sigs), sigs)
	}
	limitOnly(t, sigs)
	plan := kernel.ActivePlanFor(env.at.id, env.at.futuresSymbol())
	if plan == nil {
		t.Fatal("fixture: the recorded plan must be active at the pass")
	}
	r.pid = plan.PlanID
	armed := r.row("P1")
	if armed.Source != store.ArmSourcePicture || armed.SourceRef != res.OppKey || armed.SignalID != sigs[0].SignalID || sigs[0].LimitPrice != 101.5 {
		t.Fatalf("the frame is the P1 scenario's LIMIT at 101.50 (source picture, ref = the opportunity): row=%+v frame=%+v", armed, sigs[0])
	}
}

// W5 — the plan card's read: on an NT8 trader with Picture on, the ROUTE is
// the Day Plan scenario route under every plan mode (strict included) and the
// refusal carries only a real refusal — the running / Day Plan checks
// admitChain applies to Picture's source, in their own words.
func TestPicturePlanGateViewIsTheGatesRead(t *testing.T) {
	env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.at.exchange = "ninjatrader"
	env.at.config.StrategyConfig.DayPlan.PictureHtf = &store.PictureHtfConfig{Enabled: true, MinRR: 2.5}
	for _, mode := range []string{"strict", "advisory"} {
		env.at.config.StrategyConfig.DayPlan.PlanMode = mode
		if v := env.at.PicturePlanGateAt(env.now); !v.Enabled || v.Route != PictureRouteDayPlan || v.Refusal != "" {
			t.Fatalf("%s: the card must read enabled + the route, no refusal, got %+v", mode, v)
		}
	}
	env.at.config.StrategyConfig.DayPlan.PlanEnabled = false
	if v := env.at.PicturePlanGateAt(env.now); v.Refusal != pictureRefusalDayPlanOff {
		t.Fatalf("Day Plan off is a real refusal, got %+v", v)
	}
	env.at.config.StrategyConfig.DayPlan.PlanEnabled = true
	env.at.isRunningMutex.Lock()
	env.at.isRunning = false
	env.at.isRunningMutex.Unlock()
	if v := env.at.PicturePlanGateAt(env.now); v.Refusal != pictureRefusalStopped {
		t.Fatalf("a stopped trader is a real refusal, got %+v", v)
	}
	env.at.config.StrategyConfig.DayPlan.PictureHtf = &store.PictureHtfConfig{Enabled: false}
	if v := env.at.PicturePlanGateAt(env.now); v.Enabled || v.Route != "" || v.Refusal != "" {
		t.Fatalf("Picture off: no route, no refusal, got %+v", v)
	}
}
