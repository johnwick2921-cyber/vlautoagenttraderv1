package databento

import (
	"io"
	"net/http"
	"os"
	"testing"
)

// TestMockServer_ServesOHLCVVerbatim asserts the mock streams the OHLCV
// fixture bytes-identically — so any test that asserts on raw NDJSON shape
// is exercising the real captured payload, not a re-encoded approximation.
func TestMockServer_ServesOHLCVVerbatim(t *testing.T) {
	const fixture = "fixtures/nq-ohlcv-1m-real.json"
	srv := NewMockServer(t, fixture, "fixtures/resolve-nqm6.json")

	resp, err := http.Get(srv.URL + "/v0/timeseries.get_range")
	if err != nil {
		t.Fatalf("GET timeseries.get_range: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/x-ndjson" {
		t.Errorf("Content-Type = %q, want application/x-ndjson", ct)
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	want, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("body does not match fixture verbatim (got %d bytes, want %d bytes)", len(got), len(want))
	}
}

// TestMockServer_ServesResolve confirms the symbology.resolve route serves
// the resolve fixture and sets a JSON content type.
func TestMockServer_ServesResolve(t *testing.T) {
	srv := NewMockServer(t, "fixtures/nq-ohlcv-1m-real.json", "fixtures/resolve-nqm6.json")

	resp, err := http.Get(srv.URL + "/v0/symbology.resolve")
	if err != nil {
		t.Fatalf("GET symbology.resolve: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("empty response body")
	}
}

// TestMockServer_UnknownPath asserts that any non-allowlisted path 404s,
// so accidental endpoint typos in tests fail loudly instead of silently.
func TestMockServer_UnknownPath(t *testing.T) {
	srv := NewMockServer(t, "fixtures/nq-ohlcv-1m-real.json", "fixtures/resolve-nqm6.json")

	resp, err := http.Get(srv.URL + "/v0/does-not-exist")
	if err != nil {
		t.Fatalf("GET unknown: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}
