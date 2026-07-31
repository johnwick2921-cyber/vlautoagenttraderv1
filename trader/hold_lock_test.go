package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// buildHoldLockTrader spins up a real store with one OPEN position and an
// AutoTrader whose strategy has the given hold-discipline toggle.
func buildHoldLockTrader(t *testing.T, holdOn *bool, withOpenPosition bool) *AutoTrader {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "holdlock.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	const id = "hl-trader"
	if withOpenPosition {
		if err := st.Position().Create(&store.TraderPosition{
			TraderID:   id,
			Symbol:     "MNQ",
			Side:       "LONG",
			Quantity:   1,
			EntryPrice: 30352,
			EntryTime:  time.Now().UTC().UnixMilli(),
			Status:     "OPEN",
		}); err != nil {
			t.Fatalf("create position: %v", err)
		}
	}
	return &AutoTrader{
		id:       id,
		exchange: "ninjatrader",
		store:    st,
		config: AutoTraderConfig{
			StrategyConfig: &store.StrategyConfig{
				RiskControl: store.RiskControlConfig{HoldDisciplineEnabled: holdOn},
			},
		},
	}
}

// Default OFF must be byte-identical to today: closes are NOT suppressed.
func TestHoldLock_DefaultOffAllowsClose(t *testing.T) {
	at := buildHoldLockTrader(t, nil, true) // toggle nil = OFF, position open
	d := &kernel.Decision{Symbol: "MNQ", Action: "close_long"}
	if at.holdLockSuppressesClose(d, &store.DecisionAction{}) {
		t.Fatal("hold-lock OFF (nil) must NOT suppress the close (byte-identical default)")
	}
	off := false
	at2 := buildHoldLockTrader(t, &off, true) // explicit false
	if at2.holdLockSuppressesClose(d, &store.DecisionAction{}) {
		t.Fatal("hold-lock explicit false must NOT suppress the close")
	}
}

// ON + an open position of the matching side → the AI close is suppressed.
func TestHoldLock_OnSuppressesCloseWhenPositionOpen(t *testing.T) {
	on := true
	at := buildHoldLockTrader(t, &on, true)
	rec := &store.DecisionAction{}
	if !at.holdLockSuppressesClose(&kernel.Decision{Symbol: "MNQ", Action: "close_long"}, rec) {
		t.Fatal("hold-lock ON with an open LONG must suppress close_long")
	}
	if !rec.Success {
		t.Fatal("a suppressed close should be recorded as a deliberate no-op (Success=true)")
	}
	if rec.Error != "" {
		t.Fatalf("a suppressed close is not an error, got %q", rec.Error)
	}
}

// ON but flat for that side → nothing to hold, the close is allowed through.
func TestHoldLock_OnButFlatAllowsClose(t *testing.T) {
	on := true
	at := buildHoldLockTrader(t, &on, false) // toggle on, NO open position
	if at.holdLockSuppressesClose(&kernel.Decision{Symbol: "MNQ", Action: "close_long"}, &store.DecisionAction{}) {
		t.Fatal("hold-lock ON but flat must allow the (no-op) close — nothing to hold")
	}
}

// Only close actions are ever gated; opens/holds pass straight through.
func TestHoldLock_OnlyGatesCloseActions(t *testing.T) {
	on := true
	at := buildHoldLockTrader(t, &on, true) // toggle on, position open
	for _, act := range []string{"open_long", "open_short", "hold", "wait", "close_short"} {
		// close_short has no open SHORT position → also not suppressed.
		if at.holdLockSuppressesClose(&kernel.Decision{Symbol: "MNQ", Action: act}, &store.DecisionAction{}) {
			t.Fatalf("action %q must not be suppressed by hold-lock", act)
		}
	}
}
