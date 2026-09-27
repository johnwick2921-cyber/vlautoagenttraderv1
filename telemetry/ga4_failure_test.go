package telemetry

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestGA4Non2xxCounted pins P2-9 at the production call site: a GA4 pipe that
// answers non-2xx (or errors) must be counted, never silently swallowed.
func TestGA4Non2xxCounted(t *testing.T) {
	before := GA4Failures()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	oldClient := httpClient
	oldEndpoint := telemetryEndpoint
	httpClient = srv.Client()
	telemetryEndpoint = srv.URL
	defer func() {
		httpClient = oldClient
		telemetryEndpoint = oldEndpoint
	}()

	Init(true, "test-installation")
	TrackTrade(TradeEvent{Exchange: "binance", Symbol: "MNQ", TraderID: "t9"})

	deadline := time.Now().Add(2 * time.Second)
	for GA4Failures() == before && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if GA4Failures() == before {
		t.Fatalf("a 500 from the GA4 endpoint was not counted (failures=%d)", GA4Failures())
	}
}

// TestGA4TransportErrorCounted: a dead endpoint (connection refused) must count.
func TestGA4TransportErrorCounted(t *testing.T) {
	before := GA4Failures()
	oldClient := httpClient
	oldEndpoint := telemetryEndpoint
	httpClient = &http.Client{Timeout: 500 * time.Millisecond}
	telemetryEndpoint = "http://127.0.0.1:1/mp/collect" // nothing listens
	defer func() {
		httpClient = oldClient
		telemetryEndpoint = oldEndpoint
	}()

	Init(true, "test-installation")
	TrackStartup("rev-x")

	deadline := time.Now().Add(3 * time.Second)
	for GA4Failures() == before && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if GA4Failures() == before {
		t.Fatalf("a transport error was not counted (failures=%d)", GA4Failures())
	}
}
