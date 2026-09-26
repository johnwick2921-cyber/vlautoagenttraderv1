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

// W1b E10 verifier repair 4 — LedgerClockSlack is load-bearing: updated_at is
// zone-bearing TEXT, and a writer in another zone (-05:00, CT) stores a fresh
// fill that sorts lexically BEFORE a UTC "since" bound. The widened SQL bound
// must still return it (the caller then judges the exact window on the parsed
// time). Both ledger-wide reads are pinned.
func TestLedgerClockSlackReturnsACrossZoneFreshFill(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "slack.db"))
	if err != nil {
		t.Fatal(err)
	}
	ct := time.FixedZone("CT", -5*3600)
	const layout = "2006-01-02 15:04:05.999999999-07:00"
	freshCT := time.Now().Add(-10 * time.Second).In(ct).Format(layout)
	since := time.Now().UTC().Add(-2 * time.Minute)
	if freshCT >= since.Format(layout) {
		t.Fatalf("fixture: the -05:00 text %q must sort before the UTC bound %q", freshCT, since.Format(layout))
	}

	led := st.ArmedOrders()
	r := &ArmedOrderDB{TraderID: "trader-ct", PlanID: "p", Scenario: "S1", Version: 1, State: "armed", Side: "short", EntryPx: 100, StopPx: 101, TargetPx: 98}
	if err := led.UpsertArm(r); err != nil {
		t.Fatal(err)
	}
	if err := led.BeginPlacement(r.ID, "sig-ct"); err != nil {
		t.Fatal(err)
	}
	if err := led.SetState(r.ID, StateFilled, "fixture"); err != nil {
		t.Fatal(err)
	}
	if err := st.GormDB().Exec("UPDATE armed_orders SET updated_at = ? WHERE id = ?", freshCT, r.ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.PictureHtfClaim(&PictureHtfOpportunityDB{OppKey: "ct-pic", TraderID: "trader-ct", Account: "Sim101", Symbol: "MNQ", Direction: "short", Stage: "confirmed"}); err != nil {
		t.Fatal(err)
	}
	if err := st.PictureHtfTransition("ct-pic", StateFilled, "fixture"); err != nil {
		t.Fatal(err)
	}
	if err := st.GormDB().Exec("UPDATE picture_htf_opportunities SET updated_at = ? WHERE opp_key = ?", freshCT, "ct-pic").Error; err != nil {
		t.Fatal(err)
	}

	rows, err := led.ListFilledSinceAllTraders(since)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, x := range rows {
		if x.ID == r.ID {
			found = true
			if x.UpdatedAt.Before(since) {
				t.Fatalf("the cross-zone row parses to %v, before since %v — the fixture is not fresh", x.UpdatedAt, since)
			}
		}
	}
	if !found {
		t.Fatalf("armed: a fresh fill written as %q (-05:00) was dropped by a UTC since bound — LedgerClockSlack must cover the zone gap", freshCT)
	}
	pics, err := st.PictureHtfFilledSinceAll(since)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, p := range pics {
		if p.OppKey == "ct-pic" {
			found = true
		}
	}
	if !found {
		t.Fatalf("picture: a fresh fill written as %q (-05:00) was dropped by a UTC since bound — LedgerClockSlack must cover the zone gap", freshCT)
	}
}
