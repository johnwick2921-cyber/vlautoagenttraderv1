package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/store"
)

// B-fix — maybePruneAckedAlerts wires the previously caller-less
// PruneAckedOlderThan: once per CME session-day it hides ACKED P2 alerts older
// than 7 days. P0/P1 and unacked alerts are never auto-pruned.
func TestAlertFeedPruneWired(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	yes := true
	at := mkTrader("ninjatrader", &yes, "5m")
	at.store = st
	at.id = "trader-1"

	now := time.Now()
	old := now.AddDate(0, 0, -8).Unix()
	fresh := now.AddDate(0, 0, -1).Unix()
	seeds := []struct {
		eventID string
		level   string
		created int64
		acked   bool
	}{
		{"old-acked-p2", "P2", old, true},
		{"fresh-acked-p2", "P2", fresh, true},
		{"old-unacked-p2", "P2", old, false},
		{"old-acked-p1", "P1", old, true},
	}
	for _, s := range seeds {
		if _, err := st.Alert().Emit(&store.AlertDB{
			TraderID: at.id, Level: s.level, Kind: "test", EventID: s.eventID,
			Title: s.eventID, Body: "x", CreatedAt: s.created, Acked: s.acked,
		}); err != nil {
			t.Fatalf("emit %s: %v", s.eventID, err)
		}
	}

	at.maybePruneAckedAlerts(now)
	at.maybePruneAckedAlerts(now) // same CME day → no-op

	// List() hides dismissed rows, so read the raw rows for the assertions.
	var raw []store.AlertDB
	if err := st.GormDB().Where("trader_id = ?", at.id).Find(&raw).Error; err != nil {
		t.Fatalf("read raw alerts: %v", err)
	}
	dismissed := map[string]bool{}
	for _, a := range raw {
		if a.Dismissed {
			dismissed[a.EventID] = true
		}
	}
	if !dismissed["old-acked-p2"] {
		t.Fatalf("old acked P2 must be pruned, got %+v", raw)
	}
	for _, keep := range []string{"fresh-acked-p2", "old-unacked-p2", "old-acked-p1"} {
		if dismissed[keep] {
			t.Fatalf("%s must survive the prune, got %+v", keep, raw)
		}
	}
}
