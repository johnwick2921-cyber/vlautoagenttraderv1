package trader

import (
	"testing"

	"nofx/trader/types"
)

// recordingTrader extends MockTrader to record which close method was invoked,
// so the drawdown/emergency-close routing can be asserted for the UPPERCASE side
// NT8's GetPositions emits.
type recordingTrader struct {
	*MockTrader
	closedLong  bool
	closedShort bool
}

var _ types.Trader = (*recordingTrader)(nil)

func (r *recordingTrader) CloseLong(symbol string, qty float64) (map[string]interface{}, error) {
	r.closedLong = true
	return map[string]interface{}{"orderId": "rec-long"}, nil
}

func (r *recordingTrader) CloseShort(symbol string, qty float64) (map[string]interface{}, error) {
	r.closedShort = true
	return map[string]interface{}{"orderId": "rec-short"}, nil
}

// The sign fix: NT8 side is UPPERCASE; the pre-fix `side == "long"` ran the short
// formula for every long, inverting the sign so the drawdown-close >5% condition
// never fired on a long. positionPnLPct must return the correct sign for either casing.
func TestPositionPnLPct_UppercaseNT8SideNotInverted(t *testing.T) {
	cases := []struct {
		name         string
		side         string
		entry, mark  float64
		wantPositive bool // true → winner (pct>0), false → loser (pct<0)
	}{
		{"UPPER LONG winner → +", "LONG", 29000, 29290, true},
		{"UPPER LONG loser → -", "LONG", 29000, 28710, false},
		{"UPPER SHORT winner → +", "SHORT", 29000, 28710, true},
		{"UPPER SHORT loser → -", "SHORT", 29000, 29290, false},
		{"lower long winner → + (crypto path)", "long", 29000, 29290, true},
		{"lower short winner → + (crypto path)", "short", 29000, 28710, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pct := positionPnLPct(c.side, c.entry, c.mark, 10)
			if c.wantPositive && pct <= 0 {
				t.Fatalf("%s: got %.3f, want > 0 (pre-fix bug inverted this)", c.name, pct)
			}
			if !c.wantPositive && pct >= 0 {
				t.Fatalf("%s: got %.3f, want < 0", c.name, pct)
			}
		})
	}
}

// The switch fix: emergencyClosePosition is called with the UPPERCASE side from
// GetPositions; the pre-fix lowercase-only switch hit default → error, so the
// safety-close could not execute. Now "LONG"/"SHORT" route correctly and don't error.
func TestEmergencyClosePosition_UppercaseSideRoutes(t *testing.T) {
	rtLong := &recordingTrader{MockTrader: &MockTrader{}}
	atLong := &AutoTrader{trader: rtLong}
	if err := atLong.emergencyClosePosition("MNQ", "LONG"); err != nil {
		t.Fatalf("uppercase LONG must not error, got %v", err)
	}
	if !rtLong.closedLong || rtLong.closedShort {
		t.Fatalf("LONG must route to CloseLong (closedLong=%v closedShort=%v)", rtLong.closedLong, rtLong.closedShort)
	}

	rtShort := &recordingTrader{MockTrader: &MockTrader{}}
	atShort := &AutoTrader{trader: rtShort}
	if err := atShort.emergencyClosePosition("MNQ", "SHORT"); err != nil {
		t.Fatalf("uppercase SHORT must not error, got %v", err)
	}
	if !rtShort.closedShort || rtShort.closedLong {
		t.Fatalf("SHORT must route to CloseShort (closedLong=%v closedShort=%v)", rtShort.closedLong, rtShort.closedShort)
	}
}
