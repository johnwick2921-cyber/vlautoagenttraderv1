package trader

import (
	"fmt"
	"strings"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// ── W-ONE-BUTTON M2 site 3 — Picture HTF ───────────────────────────────────
//
// The hold refuses BEFORE the admission claim: a claim followed by a refused
// send would leave the row place_pending ("ambiguous"), which blocks re-entry
// AND the installation gate until reconciled. The refusal is DURABLE (stage
// refused): an opportunity seen during maintenance never trades, not even
// after the hold clears (fail-closed; the entry window is seconds long).

func TestPictureHtfRefusedBeforeTheClaimWhileHeld(t *testing.T) {
	dir := withMaintenanceDir(t)
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	setHold(t, dir, "job-picture")
	before := gateBlocks(env.at.id, "maintenance_hold")

	env.eval.markFresh5mReceivedAt(env.now)
	res := env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 0 {
		t.Fatalf("held: the submit seam must never run, got %d call(s)", len(env.submits))
	}
	if res.Stage != "refused" || !strings.Contains(res.Reason, "maintenance hold") {
		t.Fatalf("held: want a refused verdict naming the maintenance hold, got %+v", res)
	}
	row, ok, _ := env.st.PictureHtfGet(res.OppKey)
	if !ok || row.Stage != "refused" {
		t.Fatalf("the refusal must be durable (stage refused), got %+v", row)
	}
	if p, _ := env.st.PictureHtfPendingByTrader(env.at.id); len(p) != 0 {
		t.Fatalf("a held refusal must never leave a place_pending row: %+v", p)
	}
	// counted once per opportunity, not once per frame
	_ = env.eval.Evaluate("MNQ", env.now)
	if got := gateBlocks(env.at.id, "maintenance_hold"); got != before+1 {
		t.Fatalf("want exactly one maintenance_hold block for one opportunity, got %d", got-before)
	}

	// After the hold clears the opportunity stays dead (fail-closed).
	if err := store.ClearMaintenanceHold(dir, "job-picture"); err != nil {
		t.Fatal(err)
	}
	_ = env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 0 {
		t.Fatal("an opportunity refused during maintenance must not trade after the hold clears")
	}
}

// The hold landed after the admission check (between the claim and the send):
// the send is refused with ErrMaintenanceHold, which PROVES nothing reached the
// wire, so the row is settled as refused — never left ambiguous place_pending.
func TestPictureHtfHoldRefusalAfterTheClaimSettlesRefusedNotPending(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	orig := pictureHtfSubmitSeam
	pictureHtfSubmitSeam = func(e *PictureHtfEvaluator, row *store.PictureHtfOpportunityDB, stopPx, targetPx, qty float64, _ time.Time) error {
		env.submits = append(env.submits, row.OppKey)
		return fmt.Errorf("picture_htf: market entry refused: %w", ntTrader.ErrMaintenanceHold)
	}
	defer func() { pictureHtfSubmitSeam = orig }()
	env.seedPictureTape()
	env.eval.markFresh5mReceivedAt(env.now)
	res := env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 1 {
		t.Fatalf("fixture: the seam must run once, got %d", len(env.submits))
	}
	row, ok, _ := env.st.PictureHtfGet(env.submits[0])
	if !ok || row.Stage != "refused" || !strings.Contains(row.StageReason, "maintenance hold") {
		t.Fatalf("a hold refusal at send must settle the row refused, got %+v (result %+v)", row, res)
	}
	if p, _ := env.st.PictureHtfPendingByTrader(env.at.id); len(p) != 0 {
		t.Fatalf("no place_pending row may survive a provably-unsent hold refusal: %+v", p)
	}
}

// W5 — the retired send's own hold re-check has no send left to live in: a
// Picture opportunity reaches the wire only as a Day Plan scenario the armed
// executor places. The hold is proven where it now lives, at both ends:
//
//	(1) the evaluator refuses BEFORE the seam — under the hold the production
//	    hand-off never runs and no scenario is recorded (the control, with no
//	    hold, records one: the negative is not vacuous);
//	(2) a Picture scenario's armed row is refused at the executor's send point
//	    (maintenance_hold, counted) and stays armed and unstamped — and places
//	    once the hold clears.
func TestPictureHtfHoldRefusesBeforeTheHandOffRecordsAScenario(t *testing.T) {
	record := func(held bool) (EvaluateResult, *store.PlanDB, *store.PictureHtfOpportunityDB) {
		t.Helper()
		dir := withMaintenanceDir(t)
		prod := pictureHtfSubmitSeam
		env := admittedPictureEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
		pictureHtfSubmitSeam = prod // the harness recorder out; the production hand-off in (its cleanup restores prod)
		env.seed(pictureBars4H(), pictureBarsH1(), pictureOnGrid5M())
		env.at.markPictureRunEpoch(env.now)
		t.Cleanup(env.at.clearPictureRunEpoch)
		if held {
			setHold(t, dir, "job-picture-handoff")
		}
		env.eval.markFresh5mReceivedAt(env.now)
		res := env.eval.Evaluate("MNQ", env.now)
		sess, ok := env.at.sessionRegistry(env.now).ActiveSession(env.now)
		if !ok || sess == nil {
			t.Fatal("fixture: a live session at the Picture instant")
		}
		plan, err := env.st.Plan().GetLatestPlanForTraderSession(sessionChainDate(sess, env.now), sess.Name, env.at.id)
		if err != nil {
			t.Fatal(err)
		}
		var row *store.PictureHtfOpportunityDB
		if res.OppKey != "" {
			row, _, _ = env.st.PictureHtfGet(res.OppKey)
		}
		return res, plan, row
	}
	// Control: no hold → the production hand-off records the scenario.
	res, plan, row := record(false)
	if res.Stage != store.PictureStagePlanned || plan == nil || !store.IsMachinePlan(plan) || row == nil || row.Stage != store.PictureStagePlanned {
		t.Fatalf("control: with no hold the hand-off records a machine plan and the row settles planned: res=%+v plan=%+v row=%+v", res, plan, row)
	}
	// Held: refused before the seam — nothing recorded, the row durably refused.
	res, plan, row = record(true)
	if res.Stage != "refused" || !strings.Contains(res.Reason, "maintenance hold") {
		t.Fatalf("held: the evaluator must refuse naming the hold, got %+v", res)
	}
	if plan != nil {
		t.Fatalf("held: the hand-off must never run — no Day Plan scenario may be recorded, got plan %+v", plan)
	}
	if row == nil || row.Stage != "refused" || row.SubmittedAt != 0 {
		t.Fatalf("held: the opportunity row is refused and never sent, got %+v", row)
	}
}

func TestPictureScenarioRowRefusedAtTheSendPointWhileHeld(t *testing.T) {
	dir := withMaintenanceDir(t)
	r, epoch := newPicRig(t, "w5c-hold", nil)
	picPlan(r, picScenario("P1", "opp-hold", r.now, epoch, picDefault))
	setHold(t, dir, "job-picture-row")
	before := gateBlocks(r.at.id, "maintenance_hold")
	picPass(r, 0, 100.25) // inside the zone: only the hold stands between the row and the wire
	if sigs, _ := r.drain(); len(sigs) != 0 {
		t.Fatalf("held: a Picture scenario's row must never reach the wire: %+v", sigs)
	}
	if row := r.row("P1"); row.State != store.StateArmed || row.SignalID != "" {
		t.Fatalf("held: the row is refused at its send point — armed and unstamped: %+v", row)
	}
	if gateBlocks(r.at.id, "maintenance_hold") != before+1 {
		t.Fatal("held: the refusal is counted as maintenance_hold")
	}
	if err := store.ClearMaintenanceHold(dir, "job-picture-row"); err != nil {
		t.Fatal(err)
	}
	picPass(r, time.Second, 100.25)
	sigs, _ := r.drain()
	if len(sigs) != 1 {
		t.Fatalf("once the hold clears the same row places (the refusal was the hold's), got %d frame(s)", len(sigs))
	}
	limitOnly(t, sigs)
}

// Review F3: the Picture caller must NOT settle an AMBIGUOUS own drop (a write
// had started — it may be at NT8) as refused; the row stays place_pending.
func TestPictureHtfAmbiguousDropStaysPending(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	orig := pictureHtfSubmitSeam
	pictureHtfSubmitSeam = func(e *PictureHtfEvaluator, row *store.PictureHtfOpportunityDB, stopPx, targetPx, qty float64, _ time.Time) error {
		env.submits = append(env.submits, row.OppKey)
		return fmt.Errorf("picture_htf: market entry refused: send signal: %w", ntwire.ErrEntryDropAmbiguous)
	}
	defer func() { pictureHtfSubmitSeam = orig }()
	env.seedPictureTape()
	env.eval.markFresh5mReceivedAt(env.now)
	_ = env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 1 {
		t.Fatalf("fixture: one submit, got %d", len(env.submits))
	}
	row, ok, _ := env.st.PictureHtfGet(env.submits[0])
	if !ok || row.Stage != "place_pending" {
		t.Fatalf("an ambiguous drop must leave the row place_pending (it may be at NT8), got %+v", row)
	}
}

// M2.1 (review 3 F8): production's state is a CONFIGURED data dir with NO hold
// file — not the unconfigured "" every older fixture runs with. There, Picture
// admits and submits exactly as before (mutation: refuse whenever configured).
func TestPictureHtfSubmitsWhenConfiguredAndNoHoldFile(t *testing.T) {
	withMaintenanceDir(t) // configured, no hold file
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	env.seedPictureTape()
	env.eval.markFresh5mReceivedAt(env.now)
	_ = env.eval.Evaluate("MNQ", env.now)
	if len(env.submits) != 1 {
		t.Fatalf("configured + no hold file must submit as before, got %d submit(s)", len(env.submits))
	}
}
