package trader

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// ── W1b E9 repair (verifier defect 1) — THE DOOR SAYS WHAT HAPPENED ─────────
//
// OpenManualEntry used to return every failure as one untyped error, and the
// agent told ALL of them as "entry refused by the admission gate" — a broker
// send error (the B3 dupe drop) included, and an entry that OPENED at the
// broker with no bracket included. Three different events, three different
// things the owner must do; the error type is what carries the difference.

// doorBroker is a broker whose open and bracket sets can be made to fail, and
// which records what reached it.
type doorBroker struct {
	stubTrader
	opens            int
	setStopErr       error
	openErr          error
	stopSet, tpSet   float64
	openOrderPayload map[string]interface{}
}

func (b *doorBroker) OpenLong(string, float64, int) (map[string]interface{}, error) {
	b.opens++
	if b.openErr != nil {
		return nil, b.openErr
	}
	return b.openOrderPayload, nil
}
func (b *doorBroker) OpenShort(s string, q float64, l int) (map[string]interface{}, error) {
	return b.OpenLong(s, q, l)
}
func (b *doorBroker) SetStopLoss(_ string, _ string, _ float64, px float64) error {
	if b.setStopErr != nil {
		return b.setStopErr
	}
	b.stopSet = px
	return nil
}
func (b *doorBroker) SetTakeProfit(_ string, _ string, _ float64, px float64) error {
	b.tpSet = px
	return nil
}

func TestManualEntryDoorErrorsSayWhatHappened(t *testing.T) {
	nyMidday := time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC) // 11:00 CT Tuesday

	t.Run("an admission refusal is typed ManualEntryRefusal", func(t *testing.T) {
		w := newAgentDoorWire(t)
		agentTape(t)
		_, err := w.at.OpenManualEntryAt("MNQ", "open_long", 1, 1, 0, 0, nyMidday)
		var ref *ManualEntryRefusal
		if err == nil || !errors.As(err, &ref) || !strings.Contains(ref.Reason, "no explicit stop") {
			t.Fatalf("a stop-less chat entry is an ADMISSION refusal and must be typed as one, got %T %v", err, err)
		}
	})

	t.Run("a broker send error is not an admission refusal", func(t *testing.T) {
		w := newAgentDoorWire(t)
		live := agentTape(t)
		wide := onGrid(kernel.MinSLATRMult()*armSeamATR5m("MNQ") + 10)
		b := &doorBroker{openErr: fmt.Errorf("b3_order_dedup: duplicate entry dropped")}
		w.at.trader = b
		_, err := w.at.OpenManualEntryAt("MNQ", "open_long", 1, 1, live-wide, live+4*wide, nyMidday)
		if err == nil || b.opens != 1 {
			t.Fatalf("fixture: the admitted entry must reach the broker once and fail there: err=%v opens=%d", err, b.opens)
		}
		var ref *ManualEntryRefusal
		var unp *ManualEntryUnprotected
		if errors.As(err, &ref) || errors.As(err, &unp) {
			t.Fatalf("a broker send error was typed as %T — it is neither an admission refusal nor an opened position: %v", err, err)
		}
	})

	t.Run("CME: a bracket that cannot be set sends nothing and is not an admission refusal", func(t *testing.T) {
		w := newAgentDoorWire(t)
		live := agentTape(t)
		wide := onGrid(kernel.MinSLATRMult()*armSeamATR5m("MNQ") + 10)
		b := &doorBroker{setStopErr: fmt.Errorf("maps unavailable")}
		w.at.trader = b
		_, err := w.at.OpenManualEntryAt("MNQ", "open_long", 1, 1, live-wide, live+4*wide, nyMidday)
		var ref *ManualEntryRefusal
		var unp *ManualEntryUnprotected
		if err == nil || b.opens != 0 || errors.As(err, &ref) || errors.As(err, &unp) {
			t.Fatalf("CME: a failed bracket set must send NOTHING and be told as neither a refusal nor an open: err=%T %v opens=%d", err, err, b.opens)
		}
	})

	// Non-CME order is open, then set. The admission chain cannot admit a
	// non-CME entry in a test (its ATR5m seam reads only the NT8 BarCache, and
	// its live price reads the venue), so the send step is driven directly —
	// it is the function OpenManualEntryAt calls after admission.
	t.Run("non-CME: opened but the bracket failed is typed OPENED and UNPROTECTED", func(t *testing.T) {
		at, _ := resetTrader(t, store.StrategyConfig{})
		// W1b FOLD-2: the door runs the AI open's execute path, which reads the
		// venue's market — a crypto venue here (the NT8 venue refuses a non-CME
		// symbol before any send), its network price read stubbed offline.
		at.exchange = "binance"
		prev := openEntryMarketRead
		openEntryMarketRead = func(string, string) (*market.Data, error) { return &market.Data{CurrentPrice: 65000}, nil }
		t.Cleanup(func() { openEntryMarketRead = prev })
		b := &doorBroker{setStopErr: fmt.Errorf("venue rejected the stop"), openOrderPayload: map[string]interface{}{"orderId": "o-1"}}
		at.trader = b
		order, err := at.sendManualEntry("BTCUSDT", "open_long", 1, 1, 60000, 70000)
		var unp *ManualEntryUnprotected
		if err == nil || b.opens != 1 || !errors.As(err, &unp) {
			t.Fatalf("an entry that OPENED with no bracket must be typed ManualEntryUnprotected, got %T %v (opens=%d)", err, err, b.opens)
		}
		if unp.Symbol != "BTCUSDT" || unp.Side != "LONG" || unp.Order["orderId"] != "o-1" || order["orderId"] != "o-1" {
			t.Fatalf("the unprotected error must name the position and carry the broker's order: %+v order=%v", unp, order)
		}
		var ref *ManualEntryRefusal
		if errors.As(err, &ref) {
			t.Fatalf("an opened position must never be typed as an admission refusal: %v", err)
		}
	})
}
