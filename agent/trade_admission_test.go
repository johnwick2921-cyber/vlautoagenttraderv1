package agent

import (
	"fmt"
	"strings"
	"testing"

	"nofx/store"
	"nofx/trader"
)

// ── W-EXEC-TRUTH W0 (CTO Q17) — a chat trade asks the ONE admission gate ────

type admitSelected struct {
	refuse bool
	asked  int
}

func (f *admitSelected) GetStrategyConfig() *store.StrategyConfig { return nil }
func (f *admitSelected) GetAccountInfo() (map[string]interface{}, error) {
	return map[string]interface{}{"total_equity": 100000.0}, nil
}

// OpenManualEntry is the door (W1b E9 repair: the ONLY way a selected trader
// receives a chat entry — there is no AdmitManualEntry+underlying fallback).
func (f *admitSelected) OpenManualEntry(symbol, action string, qty float64, lev int, stop, target float64) (map[string]interface{}, error) {
	f.asked++
	if f.refuse {
		return nil, &trader.ManualEntryRefusal{Reason: "plan_mode: no matched scenario cited (strict mode)"}
	}
	return map[string]interface{}{}, nil
}

type admitUnderlying struct{ opens, closes int }

func (u *admitUnderlying) OpenLong(string, float64, int) (map[string]interface{}, error) {
	u.opens++
	return map[string]interface{}{}, nil
}
func (u *admitUnderlying) OpenShort(string, float64, int) (map[string]interface{}, error) {
	u.opens++
	return map[string]interface{}{}, nil
}
func (u *admitUnderlying) CloseLong(string, float64) (map[string]interface{}, error) {
	u.closes++
	return map[string]interface{}{}, nil
}
func (u *admitUnderlying) CloseShort(string, float64) (map[string]interface{}, error) {
	u.closes++
	return map[string]interface{}{}, nil
}
func (u *admitUnderlying) GetMarketPrice(string) (float64, error) { return 100, nil }

func TestChatEntryIsRefusedWhenTheAdmissionGateRefuses(t *testing.T) {
	sel, und := &admitSelected{refuse: true}, &admitUnderlying{}
	err := executeTradeWith(&TradeAction{Action: "open_long", Symbol: "MNQ", Quantity: 1, Leverage: 1}, false, sel, und)
	if err == nil || !strings.Contains(err.Error(), "admission gate") || !strings.Contains(err.Error(), "plan_mode") {
		t.Fatalf("a refused chat entry must say so, got %v", err)
	}
	if sel.asked != 1 || und.opens != 0 {
		t.Fatalf("the gate must be asked once and nothing sent: asked=%d opens=%d", sel.asked, und.opens)
	}
}

// W1b E9 repair: an admitted chat entry is SENT by the door; the underlying
// broker's OpenLong/OpenShort (which on NT8 read another producer's SL/TP
// maps) never sends one.
func TestChatEntryAdmittedReachesTheBroker(t *testing.T) {
	sel, und := &admitSelected{}, &admitUnderlying{}
	if err := executeTradeWith(&TradeAction{Action: "open_short", Symbol: "MNQ", Quantity: 1, Leverage: 1, StopLoss: 29100, TakeProfit: 28950}, false, sel, und); err != nil {
		t.Fatalf("an admitted chat entry must send: %v", err)
	}
	if sel.asked != 1 || und.opens != 0 {
		t.Fatalf("the door sends, the underlying never does: asked=%d underlying opens=%d", sel.asked, und.opens)
	}
}

// Closes are position management — never admitted, never refused by it.
func TestChatCloseIsNeverAdmitted(t *testing.T) {
	sel, und := &admitSelected{refuse: true}, &admitUnderlying{}
	if err := executeTradeWith(&TradeAction{Action: "close_long", Symbol: "MNQ", Quantity: 1}, false, sel, und); err != nil {
		t.Fatalf("a close must not be refused by the admission gate: %v", err)
	}
	if sel.asked != 0 || und.closes != 1 {
		t.Fatalf("asked=%d closes=%d", sel.asked, und.closes)
	}
}

// ── W1b E9 — the bracket door ───────────────────────────────────────────────
//
// A selected trader that owns the bracket door (the AutoTrader's
// OpenManualEntry) receives EVERY chat open with the chat's own stop and
// target; the underlying broker's OpenLong/OpenShort — which on NT8 reads
// per-(symbol, side) maps another producer left behind — is never called.

type bracketSelected struct {
	admitSelected
	opens        int
	stop, target float64
	refuse       bool
}

func (f *bracketSelected) OpenManualEntry(symbol, action string, qty float64, lev int, stop, target float64) (map[string]interface{}, error) {
	f.opens++
	f.stop, f.target = stop, target
	if f.refuse {
		return nil, &trader.ManualEntryRefusal{Reason: "entry_gate: refused: no explicit stop"}
	}
	return map[string]interface{}{}, nil
}

func TestChatEntryWithAStopGoesThroughTheBracketDoor(t *testing.T) {
	sel, und := &bracketSelected{}, &admitUnderlying{}
	trade := &TradeAction{Action: "open_long", Symbol: "MNQ", Quantity: 1, Leverage: 1, StopLoss: 28950, TakeProfit: 29100}
	if err := executeTradeWith(trade, false, sel, und); err != nil {
		t.Fatalf("an admitted bracket entry must send: %v", err)
	}
	if sel.opens != 1 || sel.stop != 28950 || sel.target != 29100 || und.opens != 0 {
		t.Fatalf("the chat's own bracket must reach the door and nothing else may send: door opens=%d SL=%.2f TP=%.2f underlying opens=%d",
			sel.opens, sel.stop, sel.target, und.opens)
	}
	// Refused by the door → the error names the admission gate, nothing sent.
	sel2, und2 := &bracketSelected{refuse: true}, &admitUnderlying{}
	err := executeTradeWith(&TradeAction{Action: "open_short", Symbol: "MNQ", Quantity: 1, Leverage: 1}, false, sel2, und2)
	if err == nil || !strings.Contains(err.Error(), "admission gate") || !strings.Contains(err.Error(), "no explicit stop") || und2.opens != 0 {
		t.Fatalf("a refused bracket-door entry must say so and send nothing: err=%v underlying opens=%d", err, und2.opens)
	}
}

// (W1b E9 repair) A selected trader with no bracket door no longer exists:
// the door is in tradeSelectedTrader, so the old fallback — admit, then send
// through the underlying broker's leftover SL/TP maps — cannot compile, and
// the test that pinned its refusal of a stop is gone with it.

// ── W1b E9 repair (verifier defect 1) — the reply says what HAPPENED ────────
//
// Only the door's typed admission refusal is told as "refused by the
// admission gate". A broker send error is a send failure; an entry the broker
// OPENED whose own bracket failed to set is a live, UNPROTECTED position —
// never "refused", never "failed".

type errDoor struct {
	admitSelected
	err error
}

func (d *errDoor) OpenManualEntry(string, string, float64, int, float64, float64) (map[string]interface{}, error) {
	return nil, d.err
}

func TestChatEntryErrorsSayWhatHappened(t *testing.T) {
	open := func(err error) error {
		trade := &TradeAction{Action: "open_long", Symbol: "MNQ", Quantity: 1, Leverage: 1, StopLoss: 28950, TakeProfit: 29100}
		return executeTradeWith(trade, false, &errDoor{err: err}, &admitUnderlying{})
	}

	if err := open(&trader.ManualEntryRefusal{Reason: "entry_gate: refused: no explicit stop"}); err == nil ||
		!strings.Contains(err.Error(), "admission gate") || !strings.Contains(err.Error(), "no explicit stop") {
		t.Fatalf("an admission refusal must be told as one, got %v", err)
	}

	sendErr := open(fmt.Errorf("b3_order_dedup: duplicate entry dropped"))
	if sendErr == nil || strings.Contains(sendErr.Error(), "admission gate") || !strings.Contains(sendErr.Error(), "send failed") {
		t.Fatalf("a broker send error must be told as a SEND failure, never as an admission refusal, got %v", sendErr)
	}

	unp := &trader.ManualEntryUnprotected{Symbol: "BTCUSDT", Side: "LONG", Err: fmt.Errorf("set stop 60000.00: venue rejected the stop")}
	openErr := open(unp)
	if openErr == nil || !strings.Contains(openErr.Error(), "OPENED") || !strings.Contains(openErr.Error(), "UNPROTECTED") ||
		strings.Contains(openErr.Error(), "refused") || strings.Contains(openErr.Error(), "send failed") {
		t.Fatalf("an entry that opened without its bracket must be told as OPENED and UNPROTECTED, got %v", openErr)
	}

	// The confirm reply (the owner's chat line) and the trade record.
	trade := &TradeAction{ID: "trade_1", Action: "open_long", Symbol: "BTCUSDT", Quantity: 1}
	reply := tradeFailureReply(trade, openErr, "en")
	if !strings.Contains(reply, "OPENED") || !strings.Contains(reply, "UNPROTECTED") || strings.Contains(reply, "failed:") || trade.Status != "executed" {
		t.Fatalf("the owner must read an OPENED, UNPROTECTED position — never 'execution failed': reply=%q status=%q", reply, trade.Status)
	}
	trade2 := &TradeAction{ID: "trade_2", Action: "open_long", Symbol: "MNQ", Quantity: 1}
	if reply := tradeFailureReply(trade2, sendErr, "en"); !strings.Contains(reply, "failed") || trade2.Status != "failed" {
		t.Fatalf("a send failure stays a failure: reply=%q status=%q", reply, trade2.Status)
	}
}
