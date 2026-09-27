package ninjatrader

import (
	"path/filepath"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// W117-F F7 (ports #117 23c24c6d): missing-account evidence must stay scoped to
// the reconciling trader. The account-agnostic fallback in reconcilePositions
// picked the NEWEST unassigned row across ALL traders — another trader's
// pre-binding row could swallow our account backfill. RED: today's code
// backfills trader-b's row (the newer one); the fix must backfill trader-a's
// own row and never touch the other trader's.
func TestReconcileUnassignedLookupIsTraderScoped(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "scope.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	const traderA = "trader-a"
	const traderB = "trader-b"

	s := ntwire.NewTCPServer(nil)
	s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{
		{Symbol: "MNQ", Side: "LONG", Quantity: 1, AvgPrice: 30000},
	})
	tr := NewTCPTrader(s, "MNQ", "Sim101")

	now := time.Now().UTC().UnixMilli()
	// traderA's own row is OLDER; traderB's unrelated row is NEWER — today's
	// account-agnostic lookup (entry_time DESC, no trader filter) returns B.
	if err := st.GormDB().Create(&store.TraderPosition{
		TraderID: traderA, Symbol: "MNQ", Side: "LONG", Quantity: 1,
		EntryPrice: 30000, EntryTime: now - 60_000, Status: "OPEN",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := st.GormDB().Create(&store.TraderPosition{
		TraderID: traderB, Symbol: "MNQ", Side: "LONG", Quantity: 1,
		EntryPrice: 30000, EntryTime: now - 1_000, Status: "OPEN",
	}).Error; err != nil {
		t.Fatal(err)
	}

	tr.reconcilePositions(traderA, "nt", "ninjatrader", st)

	var b store.TraderPosition
	if err := st.GormDB().Where("trader_id = ?", traderB).First(&b).Error; err != nil {
		t.Fatalf("traderB row read: %v", err)
	}
	if b.Account != "" {
		t.Fatalf("traderB's unassigned row was backfilled with account %q — cross-trader evidence leak", b.Account)
	}
	var a store.TraderPosition
	if err := st.GormDB().Where("trader_id = ?", traderA).First(&a).Error; err != nil {
		t.Fatalf("traderA row read: %v", err)
	}
	if a.Account != "Sim101" {
		t.Fatalf("traderA's own row must be backfilled to Sim101, got %q", a.Account)
	}
}
