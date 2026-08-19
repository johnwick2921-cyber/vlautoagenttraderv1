package trader

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"nofx/store"
	"nofx/trader/types"
)

// P0 hotfix (2026-08-19) — the skip gate defers to live broker truth.

// stubTrader is a minimal types.Trader whose only meaningful method is
// GetPositions; everything else returns zero values.
type stubTrader struct {
	positions []map[string]interface{}
	err       error
}

var _ types.Trader = (*stubTrader)(nil)

func (s *stubTrader) GetBalance() (map[string]interface{}, error)   { return nil, nil }
func (s *stubTrader) GetPositions() ([]map[string]interface{}, error) { return s.positions, s.err }
func (s *stubTrader) OpenLong(string, float64, int) (map[string]interface{}, error) {
	return nil, nil
}
func (s *stubTrader) OpenShort(string, float64, int) (map[string]interface{}, error) {
	return nil, nil
}
func (s *stubTrader) CloseLong(string, float64) (map[string]interface{}, error)  { return nil, nil }
func (s *stubTrader) CloseShort(string, float64) (map[string]interface{}, error) { return nil, nil }
func (s *stubTrader) SetLeverage(string, int) error                              { return nil }
func (s *stubTrader) SetMarginMode(string, bool) error                           { return nil }
func (s *stubTrader) GetMarketPrice(string) (float64, error)                     { return 0, nil }
func (s *stubTrader) SetStopLoss(string, string, float64, float64) error         { return nil }
func (s *stubTrader) SetTakeProfit(string, string, float64, float64) error       { return nil }
func (s *stubTrader) CancelStopLossOrders(string) error                          { return nil }
func (s *stubTrader) CancelTakeProfitOrders(string) error                        { return nil }
func (s *stubTrader) CancelAllOrders(string) error                               { return nil }
func (s *stubTrader) CancelStopOrders(string) error                              { return nil }
func (s *stubTrader) FormatQuantity(_ string, q float64) (string, error)         { return "", nil }
func (s *stubTrader) GetOrderStatus(string, string) (map[string]interface{}, error) {
	return nil, nil
}
func (s *stubTrader) GetClosedPnL(time.Time, int) ([]types.ClosedPnLRecord, error) {
	return nil, nil
}
func (s *stubTrader) GetOpenOrders(string) ([]types.OpenOrder, error) { return nil, nil }

func desyncTrader(t *testing.T) (*AutoTrader, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "d.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	at := &AutoTrader{
		id: "td1", exchange: "ninjatrader", store: st,
	}
	at.config.Exchange = "ninjatrader"
	at.config.StrategyConfig = &store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	openAPosition(t, st, "td1") // store says HOLDING (MNQ SHORT? -> helper uses LONG)
	return at, st
}

// V3 — forced desync: store holding, broker flat → the gate must NOT skip.
func TestSkipGateDesyncBrokerFlat(t *testing.T) {
	at, _ := desyncTrader(t)
	at.trader = &stubTrader{positions: nil} // broker: FLAT

	if skip, _ := at.skipWhileOpen(); skip {
		t.Fatal("broker-flat vs store-open must NOT skip (position_state_desync)")
	}
}

// V4 — genuine hold: broker confirms the row → skip works exactly as before
// (no regression on the #49 in-position heartbeat design).
func TestSkipGateGenuineHoldStillSkips(t *testing.T) {
	at, _ := desyncTrader(t)
	at.trader = &stubTrader{positions: []map[string]interface{}{
		{"symbol": "MNQ", "side": "LONG"}, // matches openAPosition's row
	}}
	skip, why := at.skipWhileOpen()
	if !skip || why == "" {
		t.Fatalf("a broker-confirmed hold must still skip, got skip=%v why=%q", skip, why)
	}
}

// Fail-safe posture: no usable broker truth keeps the existing skip.
func TestSkipGateFailSafeKeepsSkip(t *testing.T) {
	at, _ := desyncTrader(t)

	at.trader = nil // no broker bound
	if skip, _ := at.skipWhileOpen(); !skip {
		t.Fatal("nil trader = no truth → keep skipping")
	}

	at.trader = &stubTrader{err: errors.New("wire hiccup")} // truth errored
	if skip, _ := at.skipWhileOpen(); !skip {
		t.Fatal("GetPositions error = no truth → keep skipping")
	}
}

// Env kill-switch: POSITION_RECONCILE=off restores pre-hotfix behavior.
func TestSkipGateEnvOff(t *testing.T) {
	at, _ := desyncTrader(t)
	at.trader = &stubTrader{positions: nil} // broker flat (would desync)
	t.Setenv("POSITION_RECONCILE", "off")
	if skip, _ := at.skipWhileOpen(); !skip {
		t.Fatal("POSITION_RECONCILE=off must disable the cross-check")
	}
}

// Side-normalization: broker "SHORT" vs store "SHORT" in any casing matches.
func TestSkipGateSideCaseInsensitive(t *testing.T) {
	at, st := desyncTrader(t)
	// Add a SHORT row too; broker holds only the short, lowercase side.
	if err := st.Position().Create(&store.TraderPosition{
		TraderID: "td1", Account: "Sim101", Symbol: "MNQ", Side: "SHORT",
		Quantity: 1, EntryPrice: 29466.5, Status: "OPEN", EntryTime: 2, CreatedAt: 2, UpdatedAt: 2,
	}); err != nil {
		t.Fatal(err)
	}
	at.trader = &stubTrader{positions: []map[string]interface{}{
		{"symbol": "mnq", "side": "short"},
	}}
	if skip, _ := at.skipWhileOpen(); !skip {
		t.Fatal("one live row (case-insensitive match) is a genuine hold → skip")
	}
}
