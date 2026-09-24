package agent

import (
	"fmt"
	"strings"
	"testing"

	"nofx/store"
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
func (f *admitSelected) AdmitManualEntry(symbol, action string) (string, bool) {
	f.asked++
	if f.refuse {
		return "plan_mode: no matched scenario cited (strict mode)", true
	}
	return "", false
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

func TestChatEntryAdmittedReachesTheBroker(t *testing.T) {
	sel, und := &admitSelected{}, &admitUnderlying{}
	if err := executeTradeWith(&TradeAction{Action: "open_short", Symbol: "MNQ", Quantity: 1, Leverage: 1}, false, sel, und); err != nil {
		t.Fatalf("an admitted chat entry must send: %v", err)
	}
	if sel.asked != 1 || und.opens != 1 {
		t.Fatalf("asked=%d opens=%d", sel.asked, und.opens)
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
		return nil, fmt.Errorf("entry_gate: refused: no explicit stop")
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

// A chat entry that CARRIES a stop, on a selected trader with no bracket door,
// is refused: the underlying broker could only send it on whatever its maps
// hold (fail-closed).
func TestChatEntryWithAStopAndNoBracketDoorIsRefused(t *testing.T) {
	sel, und := &admitSelected{}, &admitUnderlying{}
	err := executeTradeWith(&TradeAction{Action: "open_long", Symbol: "MNQ", Quantity: 1, Leverage: 1, StopLoss: 28950, TakeProfit: 29100}, false, sel, und)
	if err == nil || !strings.Contains(err.Error(), "own stop") || und.opens != 0 {
		t.Fatalf("a bracket the trader cannot send must be refused, nothing sent: err=%v opens=%d", err, und.opens)
	}
}
