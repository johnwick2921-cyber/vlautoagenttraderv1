package store

import (
	"path/filepath"
	"testing"
	"time"
)

// W1b E10 — the ledger-wide "filled since" reads cover EVERY trader and, after
// the caller's exact re-check, only the window. The SQL bound is widened by
// LedgerClockSlack (zone-bearing text compare), so an old row may be returned;
// a row inside the window must never be missed.
func TestListFilledSinceAllTradersIsLedgerWide(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "lf.db"))
	if err != nil {
		t.Fatal(err)
	}
	led := st.ArmedOrders()
	mk := func(trader, scen string, state string, age time.Duration) int64 {
		r := &ArmedOrderDB{TraderID: trader, PlanID: "p", Scenario: scen, Version: 1, State: "armed", Side: "short", EntryPx: 100, StopPx: 101, TargetPx: 98}
		if err := led.UpsertArm(r); err != nil {
			t.Fatal(err)
		}
		if err := led.BeginPlacement(r.ID, "sig-"+scen); err != nil {
			t.Fatal(err)
		}
		if err := led.SetState(r.ID, state, "fixture"); err != nil {
			t.Fatal(err)
		}
		if err := led.DB().Model(&ArmedOrderDB{}).Where("id = ?", r.ID).UpdateColumn("updated_at", time.Now().Add(-age)).Error; err != nil {
			t.Fatal(err)
		}
		return r.ID
	}
	fresh := mk("trader-a", "S1", StateFilled, 10*time.Second)
	other := mk("trader-b-stopped", "S2", StateFilled, 30*time.Second)
	old := mk("trader-a", "S3", StateFilled, 72*time.Hour)
	mk("trader-b-stopped", "S4", StateWorking, time.Second)

	since := time.Now().Add(-2 * time.Minute)
	rows, err := led.ListFilledSinceAllTraders(since)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int64]bool{}
	for _, r := range rows {
		if r.State != StateFilled {
			t.Fatalf("non-filled row returned: #%d %s", r.ID, r.State)
		}
		if !r.UpdatedAt.Before(since) {
			seen[r.ID] = true
		}
	}
	if !seen[fresh] || !seen[other] || len(seen) != 2 {
		t.Fatalf("inside the window, every trader's fills (and only those): got %v want #%d #%d", seen, fresh, other)
	}
	for _, r := range rows {
		if r.ID == old {
			t.Fatalf("a fill 72h old is outside even the widened SQL bound: #%d", old)
		}
	}
}

func TestPictureHtfFilledSinceAllIsLedgerWide(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	mk := func(key, trader, stage string, age time.Duration) {
		if _, _, err := st.PictureHtfClaim(&PictureHtfOpportunityDB{OppKey: key, TraderID: trader, Account: "Sim101", Symbol: "MNQ", Direction: "short", Stage: "confirmed"}); err != nil {
			t.Fatal(err)
		}
		if err := st.PictureHtfTransition(key, stage, "fixture"); err != nil {
			t.Fatal(err)
		}
		if err := st.GormDB().Model(&PictureHtfOpportunityDB{}).Where("opp_key = ?", key).UpdateColumn("updated_at", time.Now().Add(-age)).Error; err != nil {
			t.Fatal(err)
		}
	}
	mk("fresh-a", "trader-a", StateFilled, 5*time.Second)
	mk("fresh-b", "trader-b-stopped", StateFilled, 40*time.Second)
	mk("old-a", "trader-a", StateFilled, 72*time.Hour)
	mk("working-b", "trader-b-stopped", StateWorking, time.Second)
	since := time.Now().Add(-2 * time.Minute)
	rows, err := st.PictureHtfFilledSinceAll(since)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, p := range rows {
		if p.Stage != StateFilled || p.OppKey == "old-a" {
			t.Fatalf("unexpected row: %s %s", p.OppKey, p.Stage)
		}
		if !p.UpdatedAt.Before(since) {
			seen[p.OppKey] = true
		}
	}
	if !seen["fresh-a"] || !seen["fresh-b"] || len(seen) != 2 {
		t.Fatalf("every trader's fresh Picture fills: %v", seen)
	}
}
