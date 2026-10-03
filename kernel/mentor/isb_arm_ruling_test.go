package mentor

import (
	"strings"
	"testing"

	"vl/market"
)

// TestISBOneArmFiveCandleSequence — CTO parity ruling 1791008332386 #1: ONE
// active ISB arm per side; the reference is the FIRST ISB. A 5-candle inside
// sequence gives exactly ONE arm, extended twice (the 2nd and 3rd inside
// candles), then cancelled (the 4th).
func TestISBOneArmFiveCandleSequence(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	head := market.Kline{Open: 121, High: 121.5, Low: 120.5, Close: 120, CloseTime: 1}
	mother := market.Kline{Open: 98, High: 106, Low: 97, Close: 105, CloseTime: 60_000 - 1} // green
	c1 := market.Kline{Open: 103, High: 104, Low: 99, Close: 100, CloseTime: 119_999}       // red, body inside mother
	c2 := market.Kline{Open: 100, High: 101.5, Low: 99.5, Close: 101, CloseTime: 179_999}
	c3 := market.Kline{Open: 100, High: 101.5, Low: 99.5, Close: 101, CloseTime: 239_999}
	c4 := market.Kline{Open: 100, High: 101.5, Low: 99.5, Close: 101, CloseTime: 299_999}
	bars := []market.Kline{head, mother, c1, c2, c3, c4}

	e := New(cfg)
	e.State.Trigger = TriggerLine{Dir: SideLong, Price: 90}
	e.State.HTF = HTF{FourH: TriggerLine{Dir: SideLong, Price: 90}}
	e.State.ORB = ORB{Day: 0, High: 90, Low: 85, Drawn: true, Escaped: SideLong}

	placements, extends, cancels := 0, 0, 0
	for i := 2; i <= len(bars); i++ {
		now := bars[i-1].CloseTime + 1
		for _, in := range e.Tick(bars[:i], now) {
			switch {
			case in.Action == PlaceStopLimitEntry:
				placements++
			case in.Action == ExtendArm:
				extends++
			case in.Action == CancelArm && strings.HasPrefix(in.Reason, "ISB stacking"):
				cancels++
			}
		}
	}
	if placements != 1 {
		t.Fatalf("placements = %d, want exactly ONE arm for the sequence", placements)
	}
	if extends != 2 {
		t.Fatalf("extends = %d, want 2 (the 2nd and 3rd inside candles)", extends)
	}
	if cancels != 1 {
		t.Fatalf("cancels = %d, want 1 (the 4th inside candle)", cancels)
	}
}
