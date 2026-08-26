package kernel

import (
	"os"
	"testing"

	"nofx/market"
)

// B2 (T6) — the unified policy: thresholds per state, any-TF fallback kills
// the {15m,1h}-only blindness.
func TestFeedPolicyThresholds(t *testing.T) {
	os.Unsetenv("FEED_ALERT_S")
	os.Unsetenv("INTRADE_FEED_ALERT_S")
	p := LoadFeedPolicy()
	if p.FlatAlertMs != 600_000 || p.InPositionAlertMs != 120_000 {
		t.Fatalf("defaults = %d/%d, want 600000/120000", p.FlatAlertMs, p.InPositionAlertMs)
	}
	os.Setenv("FEED_ALERT_S", "300")
	defer os.Unsetenv("FEED_ALERT_S")
	if LoadFeedPolicy().FlatAlertMs != 300_000 {
		t.Fatal("FEED_ALERT_S override not honored")
	}
}

func TestStaleEntryFeedAnyTFFallback(t *testing.T) {
	now := int64(1_700_000_900_000)
	// 15m-only selection: newest 15m bar 11 minutes old → the old code was
	// BLIND (fail-open); the fallback now assesses it against the flat policy.
	md := &market.Data{TimeframeData: map[string]*market.TimeframeSeriesData{
		"15m": {Klines: []market.KlineBar{{Time: now - 660_000}}},
	}}
	tf, age, limit, stale := staleEntryFeed(md, now)
	if tf != "15m" || !stale {
		t.Fatalf("15m-only selection must be assessed via the any-TF fallback (tf=%s age=%d limit=%d stale=%v)", tf, age, limit, stale)
	}
	// A fresh 15m bar (2 min old) passes.
	md.TimeframeData["15m"].Klines[0].Time = now - 120_000
	if _, _, _, stale := staleEntryFeed(md, now); stale {
		t.Fatal("a fresh any-TF bar must not read stale")
	}
	// The 1m/5m per-period contract stays PRIMARY and unchanged.
	md.TimeframeData["1m"] = &market.TimeframeSeriesData{Klines: []market.KlineBar{{Time: now - 30_000}}}
	if tf, _, _, stale := staleEntryFeed(md, now); tf != "1m" || stale {
		t.Fatal("1m ladder must take precedence and read fresh")
	}
}
