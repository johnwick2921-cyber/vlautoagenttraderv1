package trader

import (
	"os"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// ── W-EXEC-TRUTH W5 — the seams the lane joined, proven at the call site ────
//
//   - the D17 interrupted-hand-off sweep runs from the executor's own pass
//     (the hook is declared beside the executor and bound by the plan source);
//   - a PLACED Picture scenario carried into the next plan version (the
//     re-append, CTO 1790191033566) never places again: one opportunity, one
//     order — the executor and the ledger's source pin together.

func TestThePassRunsTheInterruptedHandOffSweep(t *testing.T) {
	at, st := handOffTrader(t)
	now := handOffNow()
	ev := handOffEvidence(now, 1)
	claimHandOff(t, st, ev) // place_pending under the claim, never recorded, never sent
	at.maybeManageArmedOrdersAt(nil, time.UnixMilli(ev.WindowCloseMs+1))
	got := pictureRow(t, st, ev.OppKey)
	if got.Stage != "refused" || got.StageReason != "hand-off interrupted — never sent" {
		t.Fatalf("an ordinary pass must run the D17 sweep: stage %q reason %q", got.Stage, got.StageReason)
	}
}

func TestAPlacedPictureScenarioCarriedIntoTheNextVersionNeverPlacesAgain(t *testing.T) {
	for _, tc := range []struct {
		name  string
		after func(r *zoneRig, ref string)
	}{
		{"resting at the broker", func(*zoneRig, string) {}},
		{"filled", func(r *zoneRig, ref string) {
			if err := r.st.GormDB().Model(&store.ArmedOrderDB{}).Where("source_ref = ?", ref).
				Updates(map[string]any{"state": store.StateFilled, "fill_price": 100.5, "fill_quantity": 1}).Error; err != nil {
				r.t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ref := "opp-carry-" + tc.name
			r, epoch := newPicRig(t, "w5-carry", nil)
			sc := picScenario("P1", ref, r.now, epoch, picDefault)
			picPlan(r, sc) // v1 carries P1
			picPass(r, 0, 100.25)
			if sigs, _ := r.drain(); len(sigs) != 1 {
				t.Fatalf("fixture: v1 places P1 exactly once, got %d", len(sigs))
			}
			tc.after(r, ref)
			picPlan(r, sc) // v2 carries the SAME P1 (the re-append's value)
			picPass(r, 5*time.Second, 100.25)
			picPass(r, 10*time.Second, 100.25)
			if sigs, _ := r.drain(); len(sigs) != 0 {
				t.Fatalf("a placed opportunity carried into v2 must never place again: %d frame(s) %+v", len(sigs), sigs)
			}
			n := 0
			for _, row := range r.rows() {
				if row.SourceRef == ref {
					n++
				}
			}
			if n != 1 {
				t.Fatalf("one opportunity, one ledger row across versions: got %d", n)
			}
		})
	}
}

// L12 at the call site: the hand-off's 🖼 lines name the opportunity with its
// account segment redacted — the full key stays in the plan doc and ledger.
func TestHandOffLinesNeverCarryTheAccountName(t *testing.T) {
	at, st := handOffTrader(t)
	now := handOffNow()
	at.markPictureRunEpoch(now)
	seedAIPlan(t, st, "active")
	ev := handOffEvidence(now, 1)
	if !strings.Contains(ev.OppKey, "|sim101|") {
		t.Fatalf("fixture: the key must carry an account segment: %s", ev.OppKey)
	}
	claimHandOff(t, st, ev)
	logs := captureTraderLog(t)
	if err := at.pictureHandOffAt(ev, now); err != nil {
		t.Fatal(err)
	}
	out := logs.String()
	if !strings.Contains(out, "🖼 picture → Day Plan scenario") {
		t.Fatalf("fixture: the hand-off must log its 🖼 line:\n%s", out)
	}
	if strings.Contains(strings.ToLower(out), "sim101") {
		t.Fatalf("a W5 log line carries the account name:\n%s", out)
	}
	if !strings.Contains(out, store.RedactPictureOppKey(ev.OppKey)) {
		t.Fatalf("the line names the opportunity through the redactor:\n%s", out)
	}
}

// W5 R1 (CTO review): the hand-off reads the latest plan version, and the
// planner appends the next version before the hand-off's overlay lands. The
// overlay must end on the NEW version (the one the executor reads), never on
// the superseded one where nothing would ever trade it.
func TestPictureHandOffNeverLandsOnASupersededVersion(t *testing.T) {
	at, st := handOffTrader(t)
	now := handOffNow()
	at.markPictureRunEpoch(now)
	v1 := seedAIPlan(t, st, "active")
	ev := handOffEvidence(now, 1)
	claimHandOff(t, st, ev)
	moved := false
	pictureHandOffBeforeAppendForTest = func() {
		if moved {
			return
		}
		moved = true
		if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: v1.PlanID, TradeDate: v1.TradeDate, Session: v1.Session, StrategyID: v1.StrategyID,
			TriggerReason: "NY_scheduled_read", Lifecycle: "active", ModelID: "m", Doc: validTraderPlanJSON}); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { pictureHandOffBeforeAppendForTest = nil })
	if err := at.pictureHandOffAt(ev, now); err != nil {
		t.Fatalf("the hand-off must re-read and record on the new version: %v", err)
	}
	if !moved {
		t.Fatal("fixture: the planner append never ran between the read and the append")
	}
	if ovs := listOverlays(t, st, v1.PlanID, 1); len(ovs) != 0 {
		t.Fatalf("the overlay landed on the SUPERSEDED v1 (the executor never reads it): %d row(s)", len(ovs))
	}
	if ovs := listOverlays(t, st, v1.PlanID, 2); len(ovs) != 1 {
		t.Fatalf("the overlay must land on v2, the latest: %d row(s)", len(ovs))
	}
}

func TestPictureHandOffBeforeAppendHookIsNilInProduction(t *testing.T) {
	if pictureHandOffBeforeAppendForTest != nil {
		t.Fatal("the R1 test seam must be nil in production")
	}
}

// W5 R2: a frame that reaches the hand-off the instant the trader registers
// for live bars finds a LIVE run epoch — never a durable "not running".
func TestPictureRunMarksTheEpochBeforeRegistering(t *testing.T) {
	at, st := handOffTrader(t)
	now := handOffNow()
	seedAIPlan(t, st, "active")
	ev := handOffEvidence(now, 1)
	claimHandOff(t, st, ev)
	called := false
	var herr error
	at.startPictureRunWith(func() {
		called = true
		herr = at.pictureHandOffAt(ev, now)
	})
	t.Cleanup(at.stopArmedEventLoop)
	if !called {
		t.Fatal("fixture: the registration never ran")
	}
	if herr != nil {
		t.Fatalf("a frame at registration must find a live run epoch, not a refusal: %v", herr)
	}
}

// Run itself goes through the ordered helper, never the bare registration.
func TestRunStartsPictureThroughTheOrderedHelper(t *testing.T) {
	src, err := os.ReadFile("auto_trader.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)
	i := strings.Index(s, "func (at *AutoTrader) Run() error {")
	if i < 0 {
		t.Fatal("Run not found")
	}
	body := s[i:]
	if j := strings.Index(body, "\n}\n"); j > 0 {
		body = body[:j]
	}
	if !strings.Contains(body, "at.startPictureRun()") || strings.Contains(body, "at.registerPictureHtf()") || strings.Contains(body, "at.startArmedEventLoop()") {
		t.Fatal("Run must start Picture through startPictureRun (epoch first, then registration), never the two calls bare")
	}
}

// W5 R3: the re-append path's 🖼 lines (appended / could NOT be re-appended)
// name the opportunity through the redactor too — never the account.
func TestReappendLinesNeverCarryTheAccountName(t *testing.T) {
	at, st := handOffTrader(t)
	schedulerTape(t)
	now := handOffNow()
	pinTraderNow(t, now.Add(2*time.Second))
	ev, _ := machinePlanV1(t, at, st, now)
	if !strings.Contains(ev.OppKey, "|sim101|") {
		t.Fatalf("fixture: the key must carry an account segment: %s", ev.OppKey)
	}
	logs := captureTraderLog(t)
	at.maybeRunSessionReadsAt(now)
	defer drainReReads(t)
	if waitVersion(t, st, 2) == nil {
		t.Fatal("fixture: the AI read must land v2")
	}
	drainReReads(t)
	out := logs.String()
	if !strings.Contains(out, "re-appended to") {
		t.Fatalf("fixture: the re-append must log its 🖼 line:\n%s", out)
	}
	if strings.Contains(strings.ToLower(out), "sim101") {
		t.Fatalf("a re-append line carries the account name:\n%s", out)
	}
}

// W5 R13(b) (CTO round 2): opportunity A lives on v1 as P1; the planner
// appends v2 and opportunity B is handed off onto v2 BEFORE A's re-append.
// B must not take P1 — P ids are minted past every id an earlier version
// used — so A's re-append lands on v2 under its own P1 and the two
// opportunities keep distinct ids (and therefore distinct ledger keys).
func TestPictureIDsNeverCollideAcrossVersions(t *testing.T) {
	at, st := handOffTrader(t)
	now := handOffNow()
	at.markPictureRunEpoch(now)
	v1 := seedAIPlan(t, st, "active")
	evA := handOffEvidence(now, 1)
	claimHandOff(t, st, evA)
	if err := at.pictureHandOffAt(evA, now); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: v1.PlanID, TradeDate: v1.TradeDate, Session: v1.Session, StrategyID: v1.StrategyID,
		TriggerReason: "NY_scheduled_read", Lifecycle: "active", ModelID: "m", Doc: validTraderPlanJSON}); err != nil {
		t.Fatal(err)
	}
	evB := handOffEvidence(now, 2)
	claimHandOff(t, st, evB)
	if err := at.pictureHandOffAt(evB, now); err != nil {
		t.Fatal(err)
	}
	at.reappendLiveMachineScenarios(v1.PlanID, 2, now)
	ids := map[string]string{}
	for _, o := range listOverlays(t, st, v1.PlanID, 2) {
		if !kernel.IsMachineOverlayOrigin(o.Origin) {
			continue
		}
		s, err := kernel.MachineScenarioFromPatch(o.Patch)
		if err != nil || s.Machine == nil {
			t.Fatalf("v2 machine overlay unreadable: %v", err)
		}
		ids[s.Machine.Ref] = s.ID
	}
	if ids[evA.OppKey] != "P1" {
		t.Fatalf("A must be re-appended to v2 under its own P1 (a collision refuses the re-append): ids=%v", redactIDs(ids))
	}
	if ids[evB.OppKey] == "" || ids[evB.OppKey] == "P1" {
		t.Fatalf("B handed off onto v2 before A's re-append must not take P1 — distinct ids per opportunity: ids=%v", redactIDs(ids))
	}
}

func redactIDs(ids map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range ids {
		out[store.RedactPictureOppKey(k)] = v
	}
	return out
}

// W5 R13(a) at the pass (the authoring loop's UpsertArm call sites): A's
// unplaced row holds P1; a later version carries a DIFFERENT opportunity
// under P1. The ledger refuses by type, the loop names the refusal (WARN +
// counter, once per change) and withdraws the admit, and A's row keeps its
// opportunity, version and prices — nothing is placed on B's admission.
func TestAReusedPIDNeverRewritesAnotherOpportunitysRow(t *testing.T) {
	r, epoch := newPicRig(t, "w5-r13a", nil)
	scA := picScenario("P1", "opp-r13-a", r.now, epoch, picDefault)
	picPlan(r, scA) // v1 carries A as P1
	// A's authored, never-placed row on v1 — exactly what the authoring loop
	// writes (stampPictureSource), before any placement.
	seed := &store.ArmedOrderDB{TraderID: r.at.id, PlanID: r.pid, Version: 1, Session: "TEST", Scenario: "P1", Side: "long",
		EntryPx: 100.5, StopPx: 97, TargetPx: 110, State: store.StateArmed, EntryClass: "armed_fill", Kind: "limit",
		Policy: store.ArmPolicyMarketInZone, CreatedAt: r.now, UpdatedAt: r.now}
	stampPictureSource(seed, scA)
	if err := r.st.ArmedOrders().UpsertArm(seed); err != nil {
		t.Fatal(err)
	}
	a := r.rows()[0]
	if a.SourceRef != "opp-r13-a" || a.State != store.StateArmed {
		t.Fatalf("fixture: A must be armed and unplaced on v1, rows=%+v", r.rows())
	}
	g := picDefault
	g.stop = 96.5
	scB := picScenario("P1", "opp-r13-b", r.now, epoch, g)
	picPlan(r, scB) // v2: B under A's id
	logs := captureTraderLog(t)
	picPass(r, 5*time.Second, 100.25) // in the zone
	picPass(r, 10*time.Second, 100.25)
	if sigs, _ := r.drain(); len(sigs) != 0 {
		t.Fatalf("nothing may be placed on B's admission of A's row: %d frame(s) %+v", len(sigs), sigs)
	}
	var after store.ArmedOrderDB
	nB := 0
	for _, row := range r.rows() {
		if row.ID == a.ID {
			after = row
		}
		if row.SourceRef == "opp-r13-b" {
			nB++
		}
	}
	if after.SourceRef != "opp-r13-a" || after.Version != a.Version || after.StopPx != a.StopPx || after.EntryPx != a.EntryPx ||
		after.EligibleUntilMs == nil || *after.EligibleUntilMs != *a.EligibleUntilMs {
		t.Fatalf("A's row must be untouched: before %+v after %+v", a, after)
	}
	if nB != 0 {
		t.Fatalf("B must not get a row under A's key: %d", nB)
	}
	out := logs.String()
	named := "NOT authored — arm_source_mismatch"
	if strings.Count(out, named) != 1 {
		t.Fatalf("the loop names the refusal once per change (WARN), got %d:\n%s", strings.Count(out, named), out)
	}
	if n := store.ArmRefusalCount(r.st, r.at.id, "2026-09-11", "TEST", "arm_source_mismatch"); n != 1 {
		t.Fatalf("the refusal is counted once per change: %d", n)
	}
}
