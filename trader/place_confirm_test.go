// THE FOUR PINS the owner named, plus the replay of the incident that produced
// them. Every one drives the REAL ledger; none asserts against a mock of itself.
package trader

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/store"
)

func pcLedger(t *testing.T) (*store.Store, *store.ArmedOrderStore) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "pc.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st, st.ArmedOrders()
}

// seedSent registers the placement before sending, as production does.
func seedSent(t *testing.T, ledger *store.ArmedOrderStore, signal string, entry float64) store.ArmedOrderDB {
	t.Helper()
	row := &store.ArmedOrderDB{
		TraderID: "hoang", PlanID: "2026-09-07:ASIA", Version: 3, Session: "ASIA",
		Scenario: "S3", Side: "short", EntryPx: entry, StopPx: entry + 29.87, TargetPx: entry - 75.25,
		State: store.StateArmed,
	}
	if err := ledger.UpsertArm(row); err != nil {
		t.Fatal(err)
	}
	if err := ledger.BeginPlacement(row.ID, signal); err != nil {
		t.Fatal(err)
	}
	return *row
}

func stateOf(t *testing.T, ledger *store.ArmedOrderStore, id int64) store.ArmedOrderDB {
	t.Helper()
	rows, err := ledger.ListForPlan("2026-09-07:ASIA")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("row %d vanished", id)
	return store.ArmedOrderDB{}
}

// PIN 1 — a REJECT goes terminal with the broker's reason, NOT working.
func TestPin1_RejectIsTerminalWithTheBrokersReason(t *testing.T) {
	_, ledger := pcLedger(t)
	row := seedSent(t, ledger, "9ba63cb5-bb72-454f-8989-a68632530560", 29721.25)

	// NT8's actual words on 2026-09-07, verbatim.
	const brokerReason = "stale signal (age 1824.5s > 60s) — rejecting"
	if err := ledger.RejectPlacement(row.ID, brokerReason); err != nil {
		t.Fatal(err)
	}

	got := stateOf(t, ledger, row.ID)
	if got.State == store.StateWorking {
		t.Fatalf("PIN 1: a REJECTED placement must never read working — this is the 2026-09-07 defect")
	}
	if got.State != store.StateRejected {
		t.Errorf("PIN 1: want state %q, got %q", store.StateRejected, got.State)
	}
	if got.StateReason != brokerReason {
		t.Errorf("PIN 1: the row must carry the BROKER'S words verbatim.\n  want %q\n  got  %q", brokerReason, got.StateReason)
	}
	// And it must not still be counted as live.
	live, _ := ledger.ListNonTerminal("hoang")
	for _, r := range live {
		if r.ID == row.ID {
			t.Errorf("PIN 1: a rejected row must be terminal, but it is still non-terminal")
		}
	}
}

// PIN 2 — send succeeds, NO frame → place_pending, then overdue but still pending. Never working.
func TestPin2_SendWithNoFrameHoldsPendingSlot(t *testing.T) {
	_, ledger := pcLedger(t)
	row := seedSent(t, ledger, "sig-no-frame", 29721.25)

	got := stateOf(t, ledger, row.ID)
	if got.State != store.StatePlacePending {
		t.Fatalf("PIN 2: a send must land in %q, got %q", store.StatePlacePending, got.State)
	}
	if got.State == store.StateWorking {
		t.Fatalf("PIN 2: a send is not a settlement")
	}
	// It is NON-terminal while pending — it may be at the broker.
	live, _ := ledger.ListNonTerminal("hoang")
	found := false
	for _, r := range live {
		if r.ID == row.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("PIN 2: a place_pending row must be NON-terminal — it holds its slot and leg 4 must count it")
	}

	// The bound spends with no frame: named unconfirmed but NONTERMINAL.
	if err := ledger.ExpirePlacement(row.ID, 4*time.Minute); err != nil {
		t.Fatal(err)
	}
	got = stateOf(t, ledger, row.ID)
	if got.State != store.StatePlacePending {
		t.Errorf("PIN 2: want %q, got %q", store.StatePlacePending, got.State)
	}
	if !strings.Contains(got.StateReason, "unconfirmed:no_frame") {
		t.Errorf("PIN 2: the reason must name it unconfirmed:no_frame, got %q", got.StateReason)
	}
	if got.State == store.StateWorking {
		t.Errorf("PIN 2: never silently working")
	}
}

// PIN 3 — a FRAME arrives → working, and the row names the frame that did it.
func TestPin3_FrameConfirmsAndIsNamed(t *testing.T) {
	_, ledger := pcLedger(t)
	row := seedSent(t, ledger, "sig-confirmed", 29721.25)

	const frame = "order_snapshot #11040 (Working, limit)"
	if err := ledger.ConfirmPlacement(row.ID, frame); err != nil {
		t.Fatal(err)
	}
	got := stateOf(t, ledger, row.ID)
	if got.State != store.StateWorking {
		t.Fatalf("PIN 3: a confirming frame must promote to working, got %q", got.State)
	}
	if !strings.Contains(got.StateReason, frame) {
		t.Errorf("PIN 3: the row must NAME the frame that promoted it (A21).\n  want to contain %q\n  got %q", frame, got.StateReason)
	}
}

// PIN 3b — a late frame cannot resurrect a row the broker already refused.
func TestPin3b_ConfirmCannotResurrectARejectedRow(t *testing.T) {
	_, ledger := pcLedger(t)
	row := seedSent(t, ledger, "sig-late", 29721.25)
	if err := ledger.RejectPlacement(row.ID, "rejected by NT8"); err != nil {
		t.Fatal(err)
	}
	if err := ledger.ConfirmPlacement(row.ID, "order_snapshot #99999"); err != nil {
		t.Fatal(err)
	}
	if got := stateOf(t, ledger, row.ID); got.State != store.StateRejected {
		t.Errorf("PIN 3b: a late frame must not revive a rejected row; state is %q", got.State)
	}
}

// PIN 4 — THE 22:47:24 REPLAY. Arm 117 must never read working.
//
// The real sequence: the send returns nil, the ledger records the placement, and
// NT8's rejection arrives in the SAME SECOND on another goroutine. Under the old
// code the row read `working` for 33 minutes. Here it must be terminal with the
// broker's reason, and it must never have been working at any point.
func TestPin4_Replay20260907_Arm117NeverReadsWorking(t *testing.T) {
	_, ledger := pcLedger(t)

	// 22:47:24.523 — the send returned nil and the ledger recorded it.
	row := seedSent(t, ledger, "9ba63cb5-bb72-454f-8989-a68632530560", 29721.25)
	afterSend := stateOf(t, ledger, row.ID)
	if afterSend.State == store.StateWorking {
		t.Fatalf("PIN 4: at 22:47:24.523 the row must NOT read working — a socket write is not a broker acceptance")
	}
	if afterSend.State != store.StatePlacePending {
		t.Fatalf("PIN 4: want %q after the send, got %q", store.StatePlacePending, afterSend.State)
	}

	// 22:47:24.528 — NT8's rejection, its own words, five milliseconds later.
	const nt8Said = "stale signal (age 1824.5s > 60s) — rejecting"
	if err := ledger.RejectPlacement(row.ID, nt8Said); err != nil {
		t.Fatal(err)
	}

	final := stateOf(t, ledger, row.ID)
	if final.State == store.StateWorking {
		t.Fatalf("PIN 4: arm 117 must NEVER read working — that is the whole defect")
	}
	if final.State != store.StateRejected {
		t.Errorf("PIN 4: want %q, got %q", store.StateRejected, final.State)
	}
	if final.StateReason != nt8Said {
		t.Errorf("PIN 4: the row must carry NT8's own sentence, not our summary.\n  want %q\n  got  %q", nt8Said, final.StateReason)
	}
	// And the flat gate must not see it as live.
	live, _ := ledger.ListNonTerminal("hoang")
	if len(live) != 0 {
		t.Errorf("PIN 4: after the rejection nothing is live; leg 4 sees %d row(s)", len(live))
	}
}

// An empty broker reason is still recorded honestly — never an empty string that
// reads like no reason existed (A24).
func TestRejectWithNoReasonTextIsStillNamed(t *testing.T) {
	_, ledger := pcLedger(t)
	row := seedSent(t, ledger, "sig-noreason", 29721.25)
	if err := ledger.RejectPlacement(row.ID, "   "); err != nil {
		t.Fatal(err)
	}
	got := stateOf(t, ledger, row.ID)
	if got.StateReason == "" {
		t.Errorf("an unexplained rejection must still say so, not carry an empty reason")
	}
	if !strings.Contains(got.StateReason, "reason unavailable") {
		t.Errorf("want the honest fallback, got %q", got.StateReason)
	}
}
