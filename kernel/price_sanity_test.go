package kernel

import (
	"testing"

	"nofx/market"
)

func TestPriceSanityViolation(t *testing.T) {
	// entryRef=30000, atr15=20 → 8×ATR = 160. lastPrice=30000.
	cases := []struct {
		name                        string
		entryRef, sl, tp, atr, last float64
		wantBad                     bool
	}{
		{"within bounds", 30000, 29900, 30100, 20, 30000, false}, // dists 100/100 < 160
		{"stop too far (>8×ATR)", 30000, 29800, 30100, 20, 30000, true}, // 200 > 160
		{"target too far (>8×ATR)", 30000, 29900, 30200, 20, 30000, true}, // 200 > 160
		{"stop exactly 8×ATR (boundary, not bad)", 30000, 29840, 30100, 20, 30000, false}, // 160 == 160, > only
		{"entry implausible >1% from last", 30500, 30450, 30550, 20, 30000, true}, // 30500 is 1.67% from 30000
		{"entry exactly 1% (boundary, not bad)", 30300, 30290, 30310, 20, 30000, false}, // exactly 1%, > only
		{"no ATR → ATR check skipped, entry ok", 30000, 20000, 40000, 0, 30000, false}, // atr=0 skips distance
		{"no last → entry check skipped, dist ok", 30000, 29900, 30100, 20, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reason, bad := priceSanityViolation("long", c.entryRef, c.sl, c.tp, c.atr, c.last)
			if bad != c.wantBad {
				t.Fatalf("bad=%v want %v (reason=%q)", bad, c.wantBad, reason)
			}
			if bad && reason == "" {
				t.Fatal("a violation must carry a reason")
			}
		})
	}
}

func TestApplyPriceSanity_NeutralizesAndFailsOpen(t *testing.T) {
	md := &market.Data{
		Symbol:       "MNQ",
		CurrentPrice: 30000,
		TimeframeData: map[string]*market.TimeframeSeriesData{
			"15m": {Timeframe: "15m", ATR14: 20}, // 8×ATR = 160
		},
	}
	ctx := &Context{MarketDataMap: map[string]*market.Data{"MNQ": md}}

	// Absurd stop (400 away > 160) → neutralized to wait.
	fd := &FullDecision{Decisions: []Decision{
		{Symbol: "MNQ", Action: "open_long", StopLoss: 29600, TakeProfit: 30100},
	}}
	applyPriceSanity(fd, ctx)
	if fd.Decisions[0].Action != "wait" {
		t.Fatalf("absurd stop must be neutralized to wait, got %q", fd.Decisions[0].Action)
	}
	if fd.Decisions[0].RefusalReason == "" {
		t.Fatalf("a price-sanity refusal must carry RefusalReason — bare wait is forbidden")
	}

	// Sane stop stays open.
	fd2 := &FullDecision{Decisions: []Decision{
		{Symbol: "MNQ", Action: "open_long", StopLoss: 29900, TakeProfit: 30100},
	}}
	applyPriceSanity(fd2, ctx)
	if fd2.Decisions[0].Action != "open_long" {
		t.Fatalf("sane decision must be untouched, got %q", fd2.Decisions[0].Action)
	}

	// Missing market data → fail-open (decision left intact, no crash).
	fd3 := &FullDecision{Decisions: []Decision{
		{Symbol: "ES", Action: "open_short", StopLoss: 1, TakeProfit: 999999},
	}}
	applyPriceSanity(fd3, &Context{MarketDataMap: map[string]*market.Data{}})
	if fd3.Decisions[0].Action != "open_short" {
		t.Fatalf("missing market data must fail-open (leave decision), got %q", fd3.Decisions[0].Action)
	}
}
