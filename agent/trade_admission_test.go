package agent

import (
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
