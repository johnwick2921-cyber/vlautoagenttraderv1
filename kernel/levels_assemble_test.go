package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
)

func TestBuildKeyLevelsBlock(t *testing.T) {
	loc := chicago()
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, loc)
	bars := []market.Kline{
		barAt(loc, 2026, 8, 13, 9, 0, 15540, 15580, 15500, 15540),
		barAt(loc, 2026, 8, 13, 14, 0, 15500, 15560, 15450, 15550),
		barAt(loc, 2026, 8, 13, 18, 0, 15570, 15620, 15560, 15600),
		barAt(loc, 2026, 8, 14, 4, 0, 15550, 15590, 15480, 15500),
		barAt(loc, 2026, 8, 14, 9, 0, 15590, 15650, 15550, 15600),
	}
	block := BuildKeyLevelsBlock("", bars, DefaultSessionRegistry(), "MNQ", 8, now, 1.5)
	if !strings.HasPrefix(block, "KEY LEVELS (map, nearest-first;") {
		t.Fatalf("assembled block header wrong:\n%s", block)
	}
	if !strings.Contains(block, "Anchor:") {
		t.Fatalf("assembled block missing anchor:\n%s", block)
	}
	// No closed bars → empty block (dormant / warming).
	if BuildKeyLevelsBlock("", nil, DefaultSessionRegistry(), "MNQ", 8, now, 1.5) != "" {
		t.Fatalf("no bars must render an empty block")
	}
}

func TestDailyATRProxy(t *testing.T) {
	loc := chicago()
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, loc)
	// Two completed session-days with 100-pt ranges → proxy ≈ 100.
	bars := []market.Kline{
		barAt(loc, 2026, 8, 11, 12, 0, 15000, 15050, 14950, 15020), // day A range 100
		barAt(loc, 2026, 8, 12, 12, 0, 15100, 15150, 15050, 15120), // day B range 100
		barAt(loc, 2026, 8, 14, 9, 0, 15590, 15650, 15550, 15600),  // developing (skipped)
	}
	if d := DailyATRProxy(bars, now); d < 90 || d > 110 {
		t.Fatalf("daily ATR proxy = %v want ≈100", d)
	}
}
