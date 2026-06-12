package ninjatrader

import (
	"testing"

	"nofx/store"
)

// TestDefaultAutoBarsTimeframes_MatchesSupported keeps the NT8 auto-subscribe
// set (the timeframes the AddOn actually streams into the BarCache) in lockstep
// with store.SupportedTimeframes, the single source of truth the Strategy Studio
// selector renders. If these drift, the UI could offer a timeframe NT8 never
// streams (→ empty futures klines) or hide one it does — so this parity is a
// hard invariant. There is NO Go-side aggregation: an interval outside this set
// yields no futures bars, which is exactly why the two lists must match.
func TestDefaultAutoBarsTimeframes_MatchesSupported(t *testing.T) {
	if len(defaultAutoBarsTimeframes) != len(store.SupportedTimeframes) {
		t.Fatalf("timeframe count drift: NT8 auto-subscribe has %d (%v), store.SupportedTimeframes has %d (%v)",
			len(defaultAutoBarsTimeframes), defaultAutoBarsTimeframes,
			len(store.SupportedTimeframes), store.SupportedTimeframes)
	}
	for i := range defaultAutoBarsTimeframes {
		if defaultAutoBarsTimeframes[i] != store.SupportedTimeframes[i] {
			t.Fatalf("timeframe drift at index %d: NT8=%q store.SupportedTimeframes=%q\n  NT8=%v\n  store=%v",
				i, defaultAutoBarsTimeframes[i], store.SupportedTimeframes[i],
				defaultAutoBarsTimeframes, store.SupportedTimeframes)
		}
	}
}
