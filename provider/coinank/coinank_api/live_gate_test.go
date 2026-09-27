package coinank_api

import (
	"context"
	"os"
	"testing"
	"time"
)

// liveNetworkGate is the opt-in for this package's INTEGRATION probes of the
// real CoinAnk API (REST + WS). A unit test must never call a live external
// service (CI killer 2026-09-26: TestBaseCoinSymbolsNoArgs hung the Backend and
// Go-coverage jobs on a slow API with context.TODO() and no timeout). These
// tests are NOT unit tests — they probe the live endpoint — so they run ONLY
// with NOFX_LIVE_TESTS=1 and under a BOUNDED context.
func liveNetworkGate(t *testing.T) context.Context {
	t.Helper()
	if os.Getenv("NOFX_LIVE_TESTS") != "1" {
		t.Skip("live-network integration test — set NOFX_LIVE_TESTS=1 to run")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	return ctx
}
