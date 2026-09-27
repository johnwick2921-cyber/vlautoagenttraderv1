package agent

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
)

// W-ONE-BUTTON M3 (CTO ruling F1) — the Telegram agent's ONE tool, driven at
// its production request builder, can never send the update header: an LLM
// steered by an inbound message cannot make apiCallTool produce a request the
// /api/updates gate admits, even though its JWT carries the owner's user_id.
// (The API side pins that a request without the header is 403 and that
// GetAPIDocs never lists /api/updates: api/handler_updates_test.go.)
func TestAgentAPICallNeverSendsTheUpdateHeader(t *testing.T) {
	var mu sync.Mutex
	var seen []http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Clone())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	tool := newAPICallTool(port, "bot-token-placeholder")

	// Everything an LLM controls: method, path, body — including a body that
	// names the header and a path that tries to smuggle one.
	for _, req := range []*apiRequest{
		{Method: "GET", Path: "/api/updates"},
		{Method: "POST", Path: "/api/updates/install", Body: map[string]any{"X-NOFX-Update": "1", "release_id": "v1"}},
		{Method: "POST", Path: "/api/updates/check?X-NOFX-Update=1"},
		{Method: "GET", Path: "api/updates/jobs/0123456789abcdef"},
	} {
		tool.execute(req)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 4 {
		t.Fatalf("positive control: the tool reached the server %d times, want 4", len(seen))
	}
	for i, h := range seen {
		if h.Get("Authorization") != "Bearer bot-token-placeholder" {
			t.Fatalf("request %d: positive control: the tool's own Authorization header is missing", i)
		}
		if v := h.Values("X-Nofx-Update"); len(v) != 0 {
			t.Fatalf("request %d carried the update header %q", i, v)
		}
		if o := h.Get("Origin"); o != "" {
			t.Fatalf("request %d carried Origin %q", i, o)
		}
	}
}
