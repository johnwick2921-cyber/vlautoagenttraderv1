package ninjatrader

import (
	"testing"

	"nofx/store"
)

func TestBuildExitFill(t *testing.T) {
	owner := &store.TraderPosition{ID: 527, TraderID: "hoang"}
	// LONG close → SELL fill.
	f := buildExitFill(owner, "ninjatrader", "futures", "MNQ", "LONG", 30323.75, 1, -68.5, 1724000000000, "sig-1")
	if f == nil {
		t.Fatal("nil fill")
	}
	if f.Side != "SELL" {
		t.Errorf("LONG close must be SELL, got %s", f.Side)
	}
	if f.ExchangeTradeID != "nt8-exit-527" {
		t.Errorf("deterministic trade id = %q", f.ExchangeTradeID)
	}
	if f.Quantity != 1 || f.Price != 30323.75 || f.RealizedPnL != -68.5 || f.CreatedAt != 1724000000000 {
		t.Errorf("field mismatch: %+v", f)
	}
	if f.Symbol != "MNQ" {
		t.Errorf("symbol must normalize to MNQ, got %q", f.Symbol)
	}
	// SHORT close → BUY fill; deterministic id must be stable across calls
	// (idempotency anchor for CreateFill dedupe).
	f2 := buildExitFill(owner, "ninjatrader", "futures", "MNQ", "SHORT", 30000, 2, 10, 1724000000001, "sig-1")
	if f2.Side != "BUY" {
		t.Errorf("SHORT close must be BUY, got %s", f2.Side)
	}
	if f2.ExchangeTradeID != "nt8-exit-527" {
		t.Errorf("same row id must yield the same trade id, got %q", f2.ExchangeTradeID)
	}
	if buildExitFill(nil, "", "", "MNQ", "LONG", 1, 1, 0, 0, "") != nil {
		t.Error("nil owner must yield nil fill")
	}
}
