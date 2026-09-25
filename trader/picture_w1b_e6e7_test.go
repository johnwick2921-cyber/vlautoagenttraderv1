package trader

import (
	"strings"
	"testing"
	"time"

	"nofx/store"
)

// ── WAVE 1b E6 — the run epoch belongs to an INSTANCE, not to a trader id ──
//
// A reload builds a new AutoTrader under the same id in the same process. If
// the new instance Runs before the old one Stops, the old Stop must neither
// clear the new run's epoch (every hand-off and placement would then read
// "trader not running") nor retire the rows the new run recorded. Driven
// through the production Run helper (startPictureRunWith) and the production
// AutoTrader.Stop, in the hazardous order: old.Run → new.Run → old.Stop.

// e6OldInstance is a second AutoTrader under at's id, in the state Run leaves
// it in (running, with a monitor channel Stop can close).
func e6OldInstance(at *AutoTrader) *AutoTrader {
	old := &AutoTrader{id: at.id, exchange: at.exchange, store: at.store, trader: at.trader}
	old.config = at.config
	old.isRunningMutex.Lock()
	old.isRunning = true
	old.isRunningMutex.Unlock()
	old.stopMonitorCh = make(chan struct{})
	return old
}

func TestPictureOldInstanceStopNeverClearsTheNewRunsEpoch(t *testing.T) {
	newAt, st := handOffTrader(t)
	now := handOffNow()
	seedAIPlan(t, st, "active")
	old := e6OldInstance(newAt)
	old.startPictureRunWith(func() {}) // the old instance's Run
	eOld, ok := old.pictureRunEpoch()
	if !ok {
		t.Fatal("fixture: the old run has an epoch")
	}
	time.Sleep(2 * time.Millisecond)     // two runs, two epochs
	newAt.startPictureRunWith(func() {}) // the reload's new instance Runs FIRST
	t.Cleanup(newAt.stopArmedEventLoop)
	eNew, ok := newAt.pictureRunEpoch()
	if !ok || eNew == eOld {
		t.Fatalf("fixture: the new run has its own epoch: %d ok=%v (old %d)", eNew, ok, eOld)
	}

	old.Stop() // ...and the old instance's Stop lands late

	if got, ok := newAt.pictureRunEpoch(); !ok || got != eNew {
		t.Fatalf("the old instance's late Stop cleared the NEW run's epoch: got %d ok=%v, want %d", got, ok, eNew)
	}
	ev := handOffEvidence(now, 1)
	claimHandOff(t, st, ev)
	if err := newAt.pictureHandOffAt(ev, now); err != nil {
		t.Fatalf("the new run's hand-off must be recorded after the old Stop: %v", err)
	}

	// The new instance's own Stop still ends the run (single-owner Stop is
	// unchanged): no epoch survives it.
	newAt.isRunningMutex.Lock() // the event loop reads it through runningNow
	newAt.isRunning = true
	newAt.isRunningMutex.Unlock()
	newAt.stopMonitorCh = make(chan struct{})
	newAt.Stop()
	if _, ok := newAt.pictureRunEpoch(); ok {
		t.Fatal("the owner's Stop must clear its own run epoch")
	}
}

func TestPictureOldInstanceStopNeverRetiresTheNewRunsRow(t *testing.T) {
	r, _ := newPicRig(t, "w1b-e6-rows", nil)
	old := e6OldInstance(r.at)
	old.startPictureRunWith(func() {}) // old.Run
	eOld, _ := old.pictureRunEpoch()
	time.Sleep(2 * time.Millisecond)
	r.at.startPictureRunWith(func() {}) // new.Run (the reload) BEFORE old.Stop
	t.Cleanup(r.at.stopArmedEventLoop)
	eNew, ok := r.at.pictureRunEpoch()
	if !ok || eNew == eOld {
		t.Fatalf("fixture: new epoch %d ok=%v (old %d)", eNew, ok, eOld)
	}
	picPlan(r, picScenario("P1", "opp-e6-new", r.now, eNew, picDefault))
	picPass(r, 0, 99.6) // short of the zone: armed, unplaced
	if row := r.row("P1"); row.State != store.StateArmed {
		t.Fatalf("fixture: P1 armed under the new run: %+v", row)
	}
	// A row the OLD run recorded (same plan, its own scenario and epoch):
	// the old Stop must still retire it — only a live successor's rows are spared.
	scOld := picScenario("P0", "opp-e6-old", r.now, eOld, picDefault)
	seed := &store.ArmedOrderDB{TraderID: r.at.id, PlanID: r.pid, Version: 1, Session: "TEST", Scenario: "P0", Side: "long",
		EntryPx: 100.5, StopPx: 97, TargetPx: 110, State: store.StateArmed, EntryClass: "armed_fill", Kind: "limit",
		Policy: store.ArmPolicyMarketInZone, CreatedAt: r.now, UpdatedAt: r.now}
	stampPictureSource(seed, scOld)
	if err := r.st.ArmedOrders().UpsertArm(seed); err != nil {
		t.Fatal(err)
	}

	old.Stop()

	if row := r.row("P1"); row.State != store.StateArmed {
		t.Fatalf("the old instance's Stop retired the NEW run's row: %s %q", row.State, row.StateReason)
	}
	if row := r.row("P0"); row.State != store.StateCancelled || row.StateReason != "picture: trader stopped — never placed" {
		t.Fatalf("the old run's own row is still retired by its Stop: %s %q", row.State, row.StateReason)
	}
	picPass(r, time.Second, 100.25) // in the zone
	sigs, _ := r.drain()
	if len(sigs) != 1 {
		t.Fatalf("the new run's row places after the old Stop: %d frame(s) %+v", len(sigs), sigs)
	}
	limitOnly(t, sigs)
}

// The successor skip is Stop's alone (event "stopped"). Day Plan OFF is the
// strategy's master switch, not a run boundary: while the old instance is
// still alive beside its successor, the OLD instance's own pass-head sweep
// (production maybeManageArmedOrdersAt → dayPlanOffPassHead →
// pictureDayPlanOffSweep) must retire every Picture row of the trader —
// including the one the successor recorded under its own epoch.
func TestPictureDayPlanOffOnTheOldInstanceStillRetiresTheNewRunsRow(t *testing.T) {
	r, _ := newPicRig(t, "w1b-e6-dpoff", nil)
	old := e6OldInstance(r.at)
	old.startPictureRunWith(func() {}) // old.Run
	t.Cleanup(old.stopArmedEventLoop)
	eOld, _ := old.pictureRunEpoch()
	time.Sleep(2 * time.Millisecond)
	r.at.startPictureRunWith(func() {}) // new.Run (the reload), old still alive
	t.Cleanup(r.at.stopArmedEventLoop)
	eNew, ok := r.at.pictureRunEpoch()
	if !ok || eNew == eOld {
		t.Fatalf("fixture: new epoch %d ok=%v (old %d)", eNew, ok, eOld)
	}
	if succ, ok := old.pictureOtherInstanceEpoch(); !ok || succ != eNew {
		t.Fatalf("fixture: the old instance sees the new run as another instance's live run: %d ok=%v", succ, ok)
	}
	picPlan(r, picScenario("P1", "opp-e6-dpoff", r.now, eNew, picDefault))
	picPass(r, 0, 99.6) // short of the zone: armed, unplaced
	if row := r.row("P1"); row.State != store.StateArmed || row.SourceRunEpoch == nil || *row.SourceRunEpoch != eNew {
		t.Fatalf("fixture: P1 armed under the new run's epoch: %+v", row)
	}

	r.at.config.StrategyConfig.DayPlan.PlanEnabled = false // shared strategy config: OFF for both instances
	if old.dayPlanEnabled() {
		t.Fatal("fixture: the old instance reads the same Day Plan master")
	}
	at := r.now.Add(time.Second)
	r.setTape(zoneTape(99.6, at, 0))
	old.maybeManageArmedOrdersAt(nil, at) // the OLD instance's pass head, only

	if row := r.row("P1"); row.State != store.StateCancelled || row.StateReason != "picture: "+pictureDayPlanOffReason+" — never placed" {
		t.Fatalf("Day Plan OFF on the old instance must retire the new run's row too (the successor skip is Stop's only): %s %q", row.State, row.StateReason)
	}
}

// ── WAVE 1b E7 — one refusal, one WARN ──────────────────────────────────────
//
// UpsertArm's W5 R13(a) refusal (ErrArmSourceMismatch) is re-hit on every
// pass while the plan carries the other opportunity under the same key. The
// authoring loop (armSourceRefused) WARNs once per change and counts it; the
// store must not add a WARN of its own per refused write. Driven through the
// production pass (maybeManageArmedOrdersAt → refresh UpsertArm).
func TestArmSourceMismatchWarnsOncePerChange(t *testing.T) {
	r, epoch := newPicRig(t, "w1b-e7-warn", nil)
	scA := picScenario("P1", "opp-e7-a", r.now, epoch, picDefault)
	picPlan(r, scA)
	seed := &store.ArmedOrderDB{TraderID: r.at.id, PlanID: r.pid, Version: 1, Session: "TEST", Scenario: "P1", Side: "long",
		EntryPx: 100.5, StopPx: 97, TargetPx: 110, State: store.StateArmed, EntryClass: "armed_fill", Kind: "limit",
		Policy: store.ArmPolicyMarketInZone, CreatedAt: r.now, UpdatedAt: r.now}
	stampPictureSource(seed, scA)
	if err := r.st.ArmedOrders().UpsertArm(seed); err != nil {
		t.Fatal(err)
	}
	g := picDefault
	g.stop = 96.5
	picPlan(r, picScenario("P1", "opp-e7-b", r.now, epoch, g)) // v2: B under A's key
	logs := captureTraderLog(t)
	for i := 1; i <= 5; i++ {
		picPass(r, time.Duration(i*5)*time.Second, 100.25) // in the zone, same refusal each pass
	}
	if sigs, _ := r.drain(); len(sigs) != 0 {
		t.Fatalf("fixture: nothing placed on B's admission of A's row: %+v", sigs)
	}
	out := logs.String()
	if n := strings.Count(out, "arm write refused"); n != 0 {
		t.Fatalf("the store logs the refusal per refused write (%d over 5 passes); the authoring loop owns the once-per-change WARN:\n%s", n, out)
	}
	named := "NOT authored — arm_source_mismatch"
	if n := strings.Count(out, named); n != 1 {
		t.Fatalf("exactly one WARN for one unchanged refusal, got %d:\n%s", n, out)
	}
	// The one WARN still carries what the store WARN said: the row, and both
	// (redacted) opportunity keys — the typed error holds the same fields.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, named) {
			if !strings.Contains(line, "holds ") || !strings.Contains(line, "the write carries ") || !strings.Contains(line, "row #") {
				t.Fatalf("the surviving WARN must name the row and both keys: %s", line)
			}
		}
	}
	if n := store.ArmRefusalCount(r.st, r.at.id, "2026-09-11", "TEST", "arm_source_mismatch"); n != 1 {
		t.Fatalf("counted once per change: %d", n)
	}
}
