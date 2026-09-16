package agent

import (
	"encoding/json"
	"testing"

	"nofx/store"
)

// P&L-TRUTH WAVE — F4: the AgentBeta trade tool returns the strict shape
// (figure + resolved n + unresolved count) and it round-trips through JSON.
func TestPnlTruthTradeHistoryToolStrictShape(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	rows := []*store.TraderPosition{
		{ID: 1, Symbol: "MNQ", Side: "LONG", EntryPrice: 29000, ExitPrice: 29050, Quantity: 1, RealizedPnL: 100, PnlCorrected: f(100), CloseReason: "sync", EntryTime: 1_756_700_000_000, ExitTime: 1_756_700_060_000},
		{ID: 2, Symbol: "MNQ", Side: "SHORT", EntryPrice: 29459, ExitPrice: 0, Quantity: 1, RealizedPnL: 0, PnlCorrected: nil, CloseReason: store.CloseReasonUnresolved, EntryTime: 1_756_700_100_000, ExitTime: 1_756_700_160_000},
		{ID: 3, Symbol: "MNQ", Side: "LONG", EntryPrice: 29000, ExitPrice: 28950, Quantity: 1, RealizedPnL: -1458, PnlCorrected: nil, CloseReason: "sync", EntryTime: 1_756_700_200_000, ExitTime: 1_756_700_260_000}, // row-526 shape
		{ID: 4, Symbol: "MNQ", Side: "LONG", EntryPrice: 29000, ExitPrice: 29003, Quantity: 1, RealizedPnL: 6, PnlCorrected: f(6), CloseReason: store.CloseReasonTestSeam, EntryTime: 1_756_700_300_000, ExitTime: 1_756_700_360_000},
	}
	payload := buildTradeHistory([]tradeHistorySource{{Name: "hoang", TID: "8d5c8af5", Positions: rows}}, 10)
	if payload == nil {
		t.Fatal("payload")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var back struct {
		Trades []struct {
			ID       int64    `json:"id"`
			PnL      *float64 `json:"pnl"`
			Resolved bool     `json:"resolved"`
		} `json:"trades"`
		Summary struct {
			TotalTrades        int     `json:"total_trades"`
			ResolvedTrades     int     `json:"resolved_trades"`
			UnresolvedExcluded int     `json:"unresolved_excluded"`
			TotalPnL           float64 `json:"total_pnl"`
			WinRate            string  `json:"win_rate"`
			PnLColumn          string  `json:"pnl_column"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.Summary.ResolvedTrades != 1 || back.Summary.TotalTrades != 1 || back.Summary.UnresolvedExcluded != 2 || back.Summary.TotalPnL != 100 {
		t.Fatalf("summary must be +100 over 1 resolved with 2 unresolved excluded (test-seam quarantined), got %+v", back.Summary)
	}
	if back.Summary.WinRate != "100.0%" || back.Summary.PnLColumn == "" {
		t.Fatalf("summary: %+v", back.Summary)
	}
	if len(back.Trades) != 3 {
		t.Fatalf("test-seam row must be quarantined, got %d rows", len(back.Trades))
	}
	for _, tr := range back.Trades {
		switch tr.ID {
		case 1:
			if !tr.Resolved || tr.PnL == nil || *tr.PnL != 100 {
				t.Fatalf("resolved row: %+v", tr)
			}
		case 2, 3:
			if tr.Resolved || tr.PnL != nil {
				t.Fatalf("unresolved row must carry pnl=null, resolved=false (never raw −1458): %+v", tr)
			}
		}
	}
}
