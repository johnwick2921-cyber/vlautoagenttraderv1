package mcp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// PHASE 4 (T5–T7) — the AI-call timeout contract, against a real HTTP server.
// Durations are scaled (seconds, not minutes): the CONTRACT under test is
// identical — a response slower than the old literal completes; a response
// slower than the CONFIGURED ceiling fails naming the deadline that fired; a
// scan tick during an in-flight call cancels nothing.

func slowAIServer(t *testing.T, delay time.Duration) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// testClient builds a client pointed at the mock server with a PLAIN http.Client
// (the production SafeHTTPClient blocks loopback by SSRF design), carrying the
// env-resolved timeout exactly as production does.
func testClient(t *testing.T, url string) *Client {
	t.Helper()
	c := NewClient().(*Client)
	c.APIKey = "test-key"
	c.BaseURL = url
	c.UseFullURL = true
	c.HTTPClient = &http.Client{Timeout: ResolvedAITimeout()}
	c.Cfg.MaxRetries = 1
	return c
}

// T5 — a response slower than the OLD 180s literal (scaled: slower than a
// too-small ceiling would have allowed) completes when the configured timeout
// accommodates it. ok=true with a real duration.
func TestT5SlowResponseCompletesWithinConfiguredTimeout(t *testing.T) {
	t.Setenv("AI_HTTP_TIMEOUT_SECONDS", "8")
	srv := slowAIServer(t, 2*time.Second) // "the 200s call", scaled
	c := testClient(t, srv.URL)

	start := time.Now()
	out, err := c.CallWithMessages("sys", "user")
	if err != nil {
		t.Fatalf("a slow-but-legal response must complete, got %v", err)
	}
	if out != "ok" {
		t.Fatalf("unexpected body %q", out)
	}
	if d := time.Since(start); d < 2*time.Second {
		t.Fatalf("duration %v — the server delay was not exercised", d)
	}
}

// T6 — a response exceeding AI_HTTP_TIMEOUT_SECONDS fails, and the failure is
// classified: the http.Client ceiling is the thing that fired.
func TestT6ExceedingConfiguredTimeoutNamesTheDeadline(t *testing.T) {
	t.Setenv("AI_HTTP_TIMEOUT_SECONDS", "1")
	srv := slowAIServer(t, 3*time.Second)
	c := testClient(t, srv.URL)

	_, err := c.CallWithMessages("sys", "user")
	if err == nil {
		t.Fatal("a call past the configured ceiling must fail")
	}
	// The classifier the structured log uses: Client.Timeout → source=client.
	if !strings.Contains(err.Error(), "Client.Timeout") &&
		!strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("the error must carry the deadline signature, got %q", err.Error())
	}
	if c.HTTPClient.Timeout != 1*time.Second {
		t.Fatalf("deadline_s in the log would be wrong: client timeout is %v", c.HTTPClient.Timeout)
	}
}

// T7 — a scan tick firing while a call is in flight cancels NOTHING. The call
// path takes no context from the loop (CallWithMessages has no ctx parameter —
// its only ceiling is the client timeout), and the trader loop is single-
// goroutine (tickOnce runs synchronously inside the ticker select, so a tick
// during a cycle WAITS; missed ticks are dropped). Here: a ticker fires 20+
// times during a 2s call and the call still completes.
func TestT7TickDuringInFlightCallCancelsNothing(t *testing.T) {
	t.Setenv("AI_HTTP_TIMEOUT_SECONDS", "8")
	srv := slowAIServer(t, 2*time.Second)
	c := testClient(t, srv.URL)

	var ticks atomic.Int64
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-tick.C:
				ticks.Add(1) // the scan cadence firing away — with no handle on the call
			case <-done:
				return
			}
		}
	}()

	out, err := c.CallWithMessages("sys", "user")
	close(done)
	if err != nil {
		t.Fatalf("ticks during an in-flight call must never cancel it, got %v", err)
	}
	if out != "ok" {
		t.Fatalf("unexpected body %q", out)
	}
	if ticks.Load() < 20 {
		t.Fatalf("only %d ticks fired — the concurrency under test did not happen", ticks.Load())
	}
}
