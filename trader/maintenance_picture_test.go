package trader

import (
	"errors"
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

// pictureHtfSend's own re-check runs FIRST, before any other re-check or the
// wire, and its error carries ErrMaintenanceHold.
func TestPictureHtfSendRefusesWhileHeld(t *testing.T) {
	dir := withMaintenanceDir(t)
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	setHold(t, dir, "job-send")
	row := &store.PictureHtfOpportunityDB{OppKey: "k", Symbol: "MNQ", Direction: "long", WindowClose: time.Now().Add(time.Hour).UnixMilli()}
	env.eval.markFresh5mReceivedAt(time.Now())
	err := pictureHtfSend(env.eval, row, 95, 110, 1, time.Now())
	if !errors.Is(err, ntTrader.ErrMaintenanceHold) {
		t.Fatalf("held: pictureHtfSend must refuse with ErrMaintenanceHold, got %v", err)
	}
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
