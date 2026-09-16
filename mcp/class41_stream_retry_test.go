package mcp

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// CLASS 41 M1 — the stream path's client retries follow an EXPONENTIAL
// schedule (default 2s → 15s → 45s), not RetryWaitBase×attempt; tries count
// CALLS (AI_PLAN_STREAM_TRIES, default 3 = two retries).
func TestClass41StreamBackoffDefaultSchedule(t *testing.T) {
	t.Setenv("AI_PLAN_STREAM_BACKOFF", "")
	t.Setenv("AI_PLAN_STREAM_TRIES", "")
	got := StreamRetryBackoffSchedule()
	want := []time.Duration{2 * time.Second, 15 * time.Second, 45 * time.Second}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("default schedule: got %v want %v", got, want)
	}
	if StreamRetryTries() != 3 {
		t.Fatalf("default tries: got %d want 3", StreamRetryTries())
	}
	// beyond the schedule the last value repeats; never zero
	if d := streamBackoffFor(5, got); d != 45*time.Second {
		t.Fatalf("beyond schedule must repeat last: %v", d)
	}
}

func TestClass41StreamBackoffEnvOverride(t *testing.T) {
	t.Setenv("AI_PLAN_STREAM_BACKOFF", "1s, 3s ,junk,7s")
	t.Setenv("AI_PLAN_STREAM_TRIES", "4")
	got := StreamRetryBackoffSchedule()
	if fmt.Sprint(got) != fmt.Sprint([]time.Duration{time.Second, 3 * time.Second, 7 * time.Second}) {
		t.Fatalf("env schedule: got %v", got)
	}
	if StreamRetryTries() != 4 {
		t.Fatalf("env tries: got %d", StreamRetryTries())
	}
}

// Two peer FINs mid-body then success: THREE calls, waits follow the injected
// schedule in order, and the ai_call line reports retries=3.
func TestClass41StreamRetryLoopFollowsSchedule(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		writeSSE(w, `{"choices":[{"delta":{"content":"a"}}]}`)
		if n < 3 {
			hj, _ := w.(http.Hijacker)
			conn, _, _ := hj.Hijack()
			conn.Close() // peer FIN mid-body → "unexpected EOF"
			return
		}
		writeSSE(w, `{"choices":[{"delta":{"content":"b"},"finish_reason":"stop"}]}`)
		writeSSE(w, "[DONE]")
	}))
	defer srv.Close()
	c := streamClient(t, srv.URL, 1)
	c.Cfg.StreamTries = 3
	c.Cfg.StreamBackoff = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond}
	out, err := c.CallWithRequestStreamRetryDeadlines(&Request{}, nil, 5*time.Second, 0)
	if err != nil || out != "ab" {
		t.Fatalf("want success on call 3: out=%q err=%v", out, err)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("want 3 calls, got %d", calls)
	}
	var waits []string
	for _, e := range c.Log.(*MockLogger).GetLogs() {
		if strings.Contains(e.Message, "before retry") {
			waits = append(waits, e.Message)
		}
	}
	if len(waits) != 2 || !strings.Contains(waits[0], "10ms") || !strings.Contains(waits[1], "20ms") {
		t.Fatalf("waits must follow the schedule in order, got %v", waits)
	}
}

// CLASS 41 M2 — the idle watchdog LOGS when it fires, with the measured gap.
func TestClass41WatchdogFireIsLogged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		writeSSE(w, `{"choices":[{"delta":{"content":"a"}}]}`)
		time.Sleep(1500 * time.Millisecond)
	}))
	defer srv.Close()
	c := streamClient(t, srv.URL, 1)
	_, err := c.CallWithRequestStreamDeadlines(&Request{}, nil, 300*time.Millisecond, 0)
	if classifyAIError(err) != string(ClassIdle) { // class 46: the vocabulary collapsed idle_deadline → idle
		t.Fatalf("want class idle, got %v", err)
	}
	found := false
	for _, e := range c.Log.(*MockLogger).GetLogs() {
		// CLASS 46 D4: the wording changed with the mechanism — the watchdog no
		// longer measures "since last SSE line" (heartbeats reset that), it
		// measures the mode's own gap and names which timer fired.
		if strings.Contains(e.Message, "⏱ watchdog fired:") && strings.Contains(e.Message, "gap=") {
			found = true
		}
	}
	if !found {
		t.Fatalf("watchdog fire must be logged with the measured gap; logs=%v", c.Log.(*MockLogger).GetLogs())
	}
}

// CLASS 41 M0 amendment (overnight 2026-09-02 01:15 CT): a 5xx / 429 from the
// provider ("Server Overloaded" 503 ×3 attempts in 9 s) is a PROVIDER failure —
// the model never answered — so it must resend identical, not append. 4xx
// request errors stay non-provider (the request itself is wrong).
func TestClass41ProviderFailureClasses(t *testing.T) {
	cases := map[string]bool{
		"still failed after 2 retries: stream interrupted: unexpected EOF":                                               true,
		"stream interrupted: read tcp 1.2.3.4:1->5.6.7.8:443: read: connection reset by peer":                            true,
		"stream idle deadline exceeded (idle 30s of silence, context canceled): stream interrupted: x":                   true,
		"failed to read response: context deadline exceeded (Client.Timeout or context cancellation while reading body)": true,
		"still failed after 2 retries: API error (status 503): {\"error\":{\"message\":\"Server Overloaded\"}}":          true,
		"API returned error (status 502): bad gateway":                                                                   true,
		"API error (status 429): rate limited":                                                                           true,
		"API error (status 400): invalid request":                                                                        false,
		"API error (status 401): unauthorized":                                                                           false,
		"no JSON object found in planner output":                                                                         false,
		"S2 breakdown_continue: a close came back across 29021.25":                                                       false,
	}
	for msg, want := range cases {
		if got := IsProviderFailure(errors.New(msg)); got != want {
			t.Errorf("IsProviderFailure(%q) = %v want %v (class=%s)", msg, got, want, ClassifyAIError(errors.New(msg)))
		}
	}
}
