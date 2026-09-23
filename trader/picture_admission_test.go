package trader

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

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
	env.eval.freshest5mAt = env.now
	res := env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 0 || res.Stage != "refused" {
		t.Fatalf("a 1.0 knob must not loosen a 50R strategy floor: submits=%d res=%+v", len(env.submits), res)
	}
}

// D12 — no strategy config: no resolvable floor ⇒ refused (fail-closed).
func TestPictureRRFloorFailsClosedWithoutAConfig(t *testing.T) {
	env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true})
	env.at.config.StrategyConfig = nil
	env.eval.freshest5mAt = env.now
	env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 0 {
		t.Fatalf("no strategy config must refuse (no floor), got %d submission(s)", len(env.submits))
	}
}

// Q6 — under plan_mode=strict Picture is REFUSED, and the refusal is VISIBLE:
// the evaluation says so and the 📷 boot line carries the same text.
func TestPictureRefusedUnderStrictVisibly(t *testing.T) {
	env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.at.config.StrategyConfig.DayPlan.PlanMode = "strict"
	env.eval.freshest5mAt = env.now
	res := env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 0 {
		t.Fatalf("strict must refuse Picture, got %d submission(s)", len(env.submits))
	}
	if !strings.Contains(res.Reason, PictureStrictRefusal) {
		t.Fatalf("the evaluation must SAY why: %+v", res)
	}
	if line := env.at.pictureHtfBootLineAt(env.now); !strings.Contains(line, PictureStrictRefusal) {
		t.Fatalf("the 📷 boot line must carry the strict refusal: %q", line)
	}
}

// A transient admission refusal BEFORE the claim leaves no durable row: the
// same hour's opportunity trades once the gate clears (an owner pause here).
func TestPicturePreClaimRefusalIsNotDurable(t *testing.T) {
	env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.at.pauseUntilMs.Store(env.now.Add(time.Hour).UnixMilli())
	env.eval.freshest5mAt = env.now
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
	env.eval.freshest5mAt = env.now
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
	env.eval.freshest5mAt = env.now
	res := env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 1 {
		t.Fatalf("fixture: the send must have been attempted once, got %d", len(env.submits))
	}
	row, _, _ := env.st.PictureHtfGet(res.OppKey)
	if row.Stage != "refused" || !strings.Contains(row.StageReason, "never sent") {
		t.Fatalf("an unsent Picture send must settle refused (never sent), got stage=%q reason=%q", row.Stage, row.StageReason)
	}
}

// DEFECT 2 + the CTO's pin — the REAL send: the claim id reaches the stamp, so
// Picture's wire path works, and a Picture claim does NOT latch its own send
// (the entry latch is wired and the claim row is place_pending, unstamped).
func TestPictureClaimDoesNotLatchItsOwnSend(t *testing.T) {
	env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
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
	t.Cleanup(func() { _ = conn.Close() })
	frames := make(chan ntwire.FrameType, 64)
	go func() {
		for {
			f, err := ntwire.ReadFrame(conn)
			if err != nil {
				return
			}
			frames <- f.Type
		}
	}()
	if err := ntwire.WriteFrame(conn, ntwire.FrameHeartbeat, ntwire.HeartbeatPayload{BuildID: ntwire.MinAddonBuildPictureHtf}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 400 && !ntwire.FarSideProven(s.FarSideBuildID(), ntwire.MinAddonBuildPictureHtf); i++ {
		time.Sleep(5 * time.Millisecond)
	}
	nt := ntTrader.NewTCPTrader(s, "MNQ", "Sim101")
	env.at.trader = nt
	env.at.config.NinjaTraderSymbol = "MNQ"
	wireNT8EntryLatch(env.at, nt)
	s.OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: []ntwire.NT8Order{}}, time.Now())
	pictureHtfSubmitSeam = pictureHtfSend

	env.eval.freshest5mAt = env.now
	res := env.eval.Evaluate("MNQ", env.now) // the PRODUCTION call site → claim → the real send
	if res.Stage != "submitted" || strings.Contains(res.Reason, "ambiguous") {
		t.Fatalf("the Picture send must reach the wire (claim id stamped, own claim not latched): %+v", res)
	}
	sent := false
	for deadline := time.After(2 * time.Second); !sent; {
		select {
		case f := <-frames:
			sent = f == ntwire.FrameSignal
		case <-deadline:
			t.Fatal("no signal frame reached the wire")
		}
	}
	got, _, _ := env.st.PictureHtfGet(res.OppKey)
	if got.SubmittedAt == 0 || got.SignalID == "" || strings.HasPrefix(got.SignalID, "picture-htf-") {
		t.Fatalf("the broker signal must be stamped on the row: %+v", got)
	}
}

// Q6 — the plan card's read: on an NT8 trader with Picture on, strict carries
// the refusal text; advisory carries none.
func TestPicturePlanGateViewIsTheGatesRead(t *testing.T) {
	env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.at.exchange = "ninjatrader"
	env.at.config.StrategyConfig.DayPlan.PictureHtf = &store.PictureHtfConfig{Enabled: true, MinRR: 2.5}
	env.at.config.StrategyConfig.DayPlan.PlanMode = "strict"
	if v := env.at.PicturePlanGateAt(env.now); !v.Enabled || v.Refusal != PictureStrictRefusal {
		t.Fatalf("strict: the card must read enabled + the refusal, got %+v", v)
	}
	env.at.config.StrategyConfig.DayPlan.PlanMode = "advisory"
	if v := env.at.PicturePlanGateAt(env.now); !v.Enabled || v.Refusal != "" {
		t.Fatalf("advisory: enabled, no refusal, got %+v", v)
	}
}
