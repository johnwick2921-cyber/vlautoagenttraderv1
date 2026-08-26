package kernel

import (
	"testing"
	"time"

	"nofx/market"
)

// TestSampleKeyLevelsBlock assembles the full P1 pipeline (multi-day + intraday
// + round-number detectors → confluence scorer → renderer) on a realistic MNQ
// fixture and logs the block. This documents the exact text the P1.7 wiring will
// inject into the executor prompt. Run: go test ./kernel -run SampleKeyLevels -v
func TestSampleKeyLevelsBlock(t *testing.T) {
	loc := chicago()
	reg := DefaultSessionRegistry()
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, loc)
	const price, dATR = 15600.0, 120.0

	bars := []market.Kline{
		barAt(loc, 2026, 8, 13, 9, 0, 15540, 15580, 15500, 15540),  // 08-13 RTH
		barAt(loc, 2026, 8, 13, 14, 0, 15500, 15560, 15450, 15550), // 08-13 RTH (low)
		barAt(loc, 2026, 8, 13, 18, 0, 15570, 15620, 15560, 15600), // Asia
		barAt(loc, 2026, 8, 14, 0, 30, 15600, 15610, 15530, 15570), // Asia
		barAt(loc, 2026, 8, 14, 4, 0, 15550, 15590, 15480, 15500),  // London
		barAt(loc, 2026, 8, 14, 8, 30, 15600, 15630, 15590, 15610), // OR + IB
		barAt(loc, 2026, 8, 14, 9, 0, 15610, 15650, 15550, 15600),  // IB
	}

	var all []DetectedLevel
	all = append(all, ExtractMultiDayLevels(bars, reg, now)...)
	all = append(all, RoundNumberLevels(price, dATR, 1.5)...)
	all = append(all, OpeningRangeLevels(bars, reg, now)...)

	scored := ScoreLevels(all, price, dATR, nil, DefaultMaxLevels, 1.5)
	block := RenderKeyLevelsBlock(scored, price)
	if block == "" {
		t.Fatal("expected a non-empty KEY LEVELS block")
	}
	t.Logf("\n%s\n", block)
}
