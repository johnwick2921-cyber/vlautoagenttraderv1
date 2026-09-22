package ninjatrader

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// W-PICTURE-HTF (2026-09-20) — the two-picture market entry with its
// protective bracket. Proven on the loopback: the frame is a plain MARKET
// order (no order_type — the AddOn's oldest, proven path), the bracket rides
// stop_loss/take_profit, the entry is the SL/TP midpoint reference, and the
// beforeSend stamp runs BEFORE the wire (a failed stamp never sends).

func TestMarketEntryWithProtectionFrameOnLoopback(t *testing.T) {
	for _, tc := range []struct {
		side string
		sl   float64
		tp   float64
	}{
		{"long", 98.25, 110.00},
		{"short", 104.00, 95.00},
	} {
		s, _, conn, frames := stopEntryServer(t)
		tr := NewTCPTrader(s, "MNQ", "Sim101")

		stampOrder := make(chan string, 1)
		sid, err := tr.MarketEntryWithProtection(tc.side, 1, tc.sl, tc.tp, func(brokerSignalID string) error {
			stampOrder <- brokerSignalID
			return nil
		})
		if err != nil {
			t.Fatalf("%s: market entry with protection failed: %v", tc.side, err)
		}
		stamped := <-stampOrder
		select {
		case p := <-frames:
			if p.OrderType != "" {
				t.Fatalf("%s: order_type=%q on a market entry, want empty (the proven path)", tc.side, p.OrderType)
			}
			if p.StopPrice != 0 || p.LimitPrice != 0 {
				t.Fatalf("%s: stop/limit slots must be empty on a market entry", tc.side)
			}
			if p.SignalID != sid || stamped != sid {
				t.Fatalf("%s: signal identity broken: sid=%q stamped=%q frame=%q", tc.side, sid, stamped, p.SignalID)
			}
			if p.StopLoss != tc.sl || p.TakeProfit != tc.tp {
				t.Fatalf("%s: bracket wrong: SL %.2f TP %.2f", tc.side, p.StopLoss, p.TakeProfit)
			}
			wantEntry := RoundToTick((tc.sl+tc.tp)/2, InstrumentTickSize("MNQ"))
			if p.Entry != wantEntry {
				t.Fatalf("%s: wire entry must be the tick-rounded midpoint reference %.2f, got %.2f", tc.side, wantEntry, p.Entry)
			}
			if p.Account != "Sim101" || p.TraderID != tr.traderID {
				t.Fatalf("%s: identity stamp missing: acct=%q trader=%q", tc.side, p.Account, p.TraderID)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s: no market-entry frame", tc.side)
		}
		_ = conn
	}
}

func TestMarketEntryWithProtectionRefusesIncompleteBracket(t *testing.T) {
	s, _, _, frames := stopEntryServer(t)
	tr := NewTCPTrader(s, "MNQ", "Sim101")
	if _, err := tr.MarketEntryWithProtection("long", 1, 0, 110); err == nil || !strings.Contains(err.Error(), "bracket incomplete") {
		t.Fatalf("a missing stop must refuse with the bracket reason, got %v", err)
	}
	select {
	case <-frames:
		t.Fatal("incomplete bracket must never reach the wire")
	case <-time.After(30 * time.Millisecond):
	}
}

func TestMarketEntryWithProtectionStampFailureNeverSends(t *testing.T) {
	s, _, _, frames := stopEntryServer(t)
	tr := NewTCPTrader(s, "MNQ", "Sim101")
	_, err := tr.MarketEntryWithProtection("long", 1, 98.25, 110, func(string) error {
		return errors.New("fixture ledger unavailable")
	})
	if err == nil {
		t.Fatal("stamp failure ignored")
	}
	select {
	case <-frames:
		t.Fatal("unstamped picture entry sent")
	case <-time.After(30 * time.Millisecond):
	}
}
