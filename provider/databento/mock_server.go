package databento

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// NewMockServer returns an httptest.Server that serves the given fixture
// files at the standard Databento v0 API paths:
//
//	GET /v0/timeseries.get_range  -> ohlcvFixture (NDJSON OHLCV bars)
//	GET /v0/symbology.resolve     -> resolveFixture (JSON object)
//
// Any other path returns 404.
//
// Callers should configure the test Client with BaseURL = srv.URL + "/v0"
// so the path layout matches the real Databento historical API (whose
// canonical base URL also ends in "/v0").
//
// The server is automatically closed via t.Cleanup; the caller does not
// need to invoke srv.Close() explicitly.
//
// Background: the Databento response shape uses a nested "hd" record-header
// with ts_event/rtype inside (not flat). A prior fabricated unit-test
// fixture used a flat shape, which let a parser bug ship undetected
// (caught by the live smoke test on 2026-05-25). This mock exists so all
// tests can exercise the real wire shape end-to-end via fixtures captured
// from a live response.
func NewMockServer(t *testing.T, ohlcvFixture, resolveFixture string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v0/timeseries.get_range":
			serveFixture(t, w, ohlcvFixture, "application/x-ndjson")
		case "/v0/symbology.resolve":
			serveFixture(t, w, resolveFixture, "application/json")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func serveFixture(t *testing.T, w http.ResponseWriter, path, contentType string) {
	t.Helper()
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", contentType)
	if _, err := io.Copy(w, f); err != nil {
		// Already wrote headers; can't change status now. Surface via t.Logf.
		t.Logf("mock_server: copy fixture %s: %v", path, err)
	}
}
