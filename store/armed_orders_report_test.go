package store

import (
	"strings"
	"testing"
	"time"
)

// CANCEL-REPORT REGIME (2026-10-03) — the store half, tested at the production
// methods themselves: RecordCancelReport records the AddOn's positive report,
// ConfirmCancelByReport is the ONLY promotion under the regime and it refuses
// everything a report cannot prove.

func newReportStore(t *testing.T) *ArmedOrderStore {
	t.Helper()
	st := NewArmedOrderStore(newArmedTestDB(t))
	r := &ArmedOrderDB{
		TraderID: "t1", PlanID: "2026-10-03:planX", Version: 1, Session: "RTH",
		Scenario: "S1", Side: "long", EntryPx: 100, StopPx: 99, TargetPx: 102,
		State: "armed", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := st.UpsertArm(r); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := st.SetSignal(r.ID, "sig-a"); err != nil {
		t.Fatalf("set signal: %v", err)
	}
	return st
}

func TestRecordCancelReportIsMonotonicLatestWins(t *testing.T) {
	st := newReportStore(t)
	rows, err := st.ListNonTerminal("t1")
	if err != nil || len(rows) != 1 {
		t.Fatalf("seed: %v", err)
	}
	id := rows[0].ID
	if err := st.RecordCancelReport(id, 1000, "cancelled"); err != nil {
		t.Fatalf("first record: %v", err)
	}
	if err := st.RecordCancelReport(id, 2000, "cancelled"); err != nil {
		t.Fatalf("newer record: %v", err)
	}
	// An older duplicate is ignored — the echo and the real transition both
	// report, and either may arrive twice.
	if err := st.RecordCancelReport(id, 500, "cancelled"); err != nil {
		t.Fatalf("older duplicate: %v", err)
	}
	var got ArmedOrderDB
	if err := st.db.First(&got, id).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got.CancelReportMs != 2000 || got.CancelReportState != "cancelled" {
		t.Fatalf("latest report must win, got ms=%d state=%q", got.CancelReportMs, got.CancelReportState)
	}
}

func TestRecordCancelReportRejectsNothingness(t *testing.T) {
	st := newReportStore(t)
	rows, _ := st.ListNonTerminal("t1")
	id := rows[0].ID
	if err := st.RecordCancelReport(id, 0, "cancelled"); err == nil {
		t.Fatalf("a report without a receipt time must be refused")
	}
	if err := st.RecordCancelReport(id, 1000, ""); err == nil {
		t.Fatalf("a report without a state must be refused")
	}
}

func TestConfirmCancelByReportSettlesWithQualifyingReport(t *testing.T) {
	st := newReportStore(t)
	rows, _ := st.ListNonTerminal("t1")
	id := rows[0].ID
	if err := st.RequestCancel(id, "gate changed", 1000); err != nil {
		t.Fatalf("request cancel: %v", err)
	}
	if err := st.RecordCancelReport(id, 1500, "cancelled"); err != nil {
		t.Fatalf("record report: %v", err)
	}
	if err := st.ConfirmCancelByReport(id, "gate changed — report: cancelled"); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	var got ArmedOrderDB
	if err := st.db.First(&got, id).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got.State != StateCancelled {
		t.Fatalf("want cancelled, got %q", got.State)
	}
	if !strings.Contains(got.StateReason, "gate changed") {
		t.Fatalf("the original cancel reason must survive the confirmation, got %q", got.StateReason)
	}
}

func TestConfirmCancelByReportRefusesWithoutAReport(t *testing.T) {
	st := newReportStore(t)
	rows, _ := st.ListNonTerminal("t1")
	id := rows[0].ID
	if err := st.RequestCancel(id, "gate changed", 1000); err != nil {
		t.Fatalf("request cancel: %v", err)
	}
	if err := st.ConfirmCancelByReport(id, "gate changed"); err == nil {
		t.Fatalf("a confirmation with no recorded report must be refused")
	}
	var got ArmedOrderDB
	st.db.First(&got, id)
	if got.State != StateCancelPending {
		t.Fatalf("the row must stay cancel_pending, got %q", got.State)
	}
}

func TestConfirmCancelByReportRefusesReportPredatingTheRequest(t *testing.T) {
	st := newReportStore(t)
	rows, _ := st.ListNonTerminal("t1")
	id := rows[0].ID
	if err := st.RequestCancel(id, "gate changed", 2000); err != nil {
		t.Fatalf("request cancel: %v", err)
	}
	if err := st.RecordCancelReport(id, 1000, "cancelled"); err != nil {
		t.Fatalf("record report: %v", err)
	}
	if err := st.ConfirmCancelByReport(id, "gate changed"); err == nil {
		t.Fatalf("a report that predates the request must be refused (F9)")
	}
}

func TestConfirmCancelByReportRefusesAFilledReport(t *testing.T) {
	st := newReportStore(t)
	rows, _ := st.ListNonTerminal("t1")
	id := rows[0].ID
	if err := st.RequestCancel(id, "gate changed", 1000); err != nil {
		t.Fatalf("request cancel: %v", err)
	}
	if err := st.RecordCancelReport(id, 1500, "filled"); err != nil {
		t.Fatalf("record report: %v", err)
	}
	if err := st.ConfirmCancelByReport(id, "gate changed"); err == nil {
		t.Fatalf("a filled report is the filled path's word, never a cancellation")
	}
	var got ArmedOrderDB
	st.db.First(&got, id)
	if got.State != StateCancelPending {
		t.Fatalf("the row must stay cancel_pending for the filled path to own, got %q", got.State)
	}
}

func TestConfirmCancelByReportRefusesANonPendingRow(t *testing.T) {
	st := newReportStore(t)
	rows, _ := st.ListNonTerminal("t1")
	id := rows[0].ID
	if err := st.RecordCancelReport(id, 1500, "cancelled"); err != nil {
		t.Fatalf("record report: %v", err)
	}
	if err := st.ConfirmCancelByReport(id, "gate changed"); err == nil {
		t.Fatalf("a row that never requested a cancel must not be confirmed")
	}
}
