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

// Planner-speed wave phase 4 (2026-08-31) — the split-deadline contract on the
// SSE path: a live-but-slow stream survives the idle watchdog; a silent stream
// dies at the idle deadline (not 600s); the transport-reset class retries via
// the fixed retry flow; the ai_call line carries ttfb/reasoning/retries.

func streamClient(t *testing.T, url string, retries int) *Client {
	t.Helper()
	c := NewClient().(*Client)
	c.APIKey = "test-key"
	c.BaseURL = url
	c.UseFullURL = true
	c.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	c.Cfg.MaxRetries = retries
	c.Cfg.StreamTries = retries // class 41: the stream path counts CALLS via StreamTries
	c.Cfg.StreamBackoff = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond}
	c.Log = NewMockLogger()
	return c
}

func writeSSE(w http.ResponseWriter, data string) {
	fmt.Fprintf(w, "data: %s\n\n", data)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// TestStreamSlowButAliveSurvivesIdle — chunks keep arriving (gap < idle) so a
// stream longer than the idle window completes; ttfb/reasoning/finish captured.
func TestStreamSlowButAliveSurvivesIdle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		time.Sleep(30 * time.Millisecond) // nonzero queue+think so ttfb is measurable
		writeSSE(w, `{"choices":[{"delta":{"reasoning_content":"thinking hard"},"finish_reason":null}]}`)
		for i := 0; i < 8; i++ { // 8 chunks × 60ms = 480ms > 150ms idle
			time.Sleep(60 * time.Millisecond)
			writeSSE(w, fmt.Sprintf(`{"choices":[{"delta":{"content":"chunk%d"},"finish_reason":null}]}`, i))
		}
		writeSSE(w, `{"choices":[{"delta":{"content":""},"finish_reason":"stop"}],"usage":{"prompt_tokens":100,"completion_tokens":200,"total_tokens":300}}`)
		writeSSE(w, `[DONE]`)
	}))
	defer srv.Close()

	c := streamClient(t, srv.URL, 1)
	req := &Request{Messages: []Message{NewUserMessage("hi")}}
	out, err := c.CallWithRequestStreamRetry(req, nil, 150*time.Millisecond)
	if err != nil {
		t.Fatalf("slow-but-alive stream must survive the idle watchdog, got %v", err)
	}
	if !strings.Contains(out, "chunk7") {
		t.Fatalf("stream text incomplete: %q", out)
	}
	if c.lastTTFBMs.Load() <= 0 {
		t.Fatalf("ttfb not stamped, got %d", c.lastTTFBMs.Load())
	}
	if c.lastReasoningChars.Load() <= 0 {
		t.Fatalf("reasoning chars not captured, got %d", c.lastReasoningChars.Load())
	}
	if f, _ := c.lastFinishReason.Load().(string); f != "stop" {
		t.Fatalf("finish_reason = %q, want stop", f)
	}
	// The retry wrapper emits the enriched ai_call line.
	got := c.Log.(*MockLogger)
	if !logContains(got, "ai_call model=") || !logContains(got, "ttfb_ms=") || !logContains(got, "reasoning_chars=") {
		t.Fatalf("ai_call line missing new fields: %+v", got.Logs)
	}
}

// TestStreamIdleSilenceAborts — a silent stream dies at the idle deadline with
// a context cancel (the retry fires immediately instead of burning 600s).
func TestStreamIdleSilenceAborts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		writeSSE(w, `{"choices":[{"delta":{"content":"first"},"finish_reason":null}]}`)
		time.Sleep(2 * time.Second) // silence ≫ idle
	}))
	defer srv.Close()

	c := streamClient(t, srv.URL, 1)
	req := &Request{Messages: []Message{NewUserMessage("hi")}}
	_, err := c.CallWithRequestStreamIdle(req, nil, 200*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("silent stream must die at the idle deadline (context canceled), got %v", err)
	}
}

// TestStreamTransportResetRetryEngages — the first connection is reset
// mid-stream; the fixed retry flow re-sends and wins (phase 4.4).
func TestStreamTransportResetRetryEngages(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		if n == 1 {
			writeSSE(w, `{"choices":[{"delta":{"content":"partial"},"finish_reason":null}]}`)
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, _ := hj.Hijack()
				_ = conn.Close() // transport reset, mid-stream
				return
			}
			return
		}
		writeSSE(w, `{"choices":[{"delta":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}`)
		writeSSE(w, `[DONE]`)
	}))
	defer srv.Close()

	c := streamClient(t, srv.URL, 2)
	req := &Request{Messages: []Message{NewUserMessage("hi")}}
	out, err := c.CallWithRequestStreamRetry(req, nil, 5*time.Second)
	if err != nil {
		t.Fatalf("transport reset must retry and win, got %v", err)
	}
	if !strings.Contains(out, "ok") {
		t.Fatalf("retried stream text = %q", out)
	}
	if calls.Load() < 2 {
		t.Fatalf("expected ≥2 server hits, got %d", calls.Load())
	}
	got := c.Log.(*MockLogger)
	if !logContains(got, "AI API stream failed, retrying (2/2)") {
		t.Fatalf("retry warn missing: %+v", got.Logs)
	}
}

func logContains(m *MockLogger, sub string) bool {
	for _, e := range m.Logs {
		if strings.Contains(e.Message, sub) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Class 37 (2026-09-01) — the whole-request ceiling on a LIVE stream.
// The 4.4 fixtures above used Timeout=10s against ~0.5s streams and never
// crossed http.Client.Timeout; these do.
// ---------------------------------------------------------------------------

// liveStreamServer streams `chunks` reasoning deltas `gap` apart, then a stop
// frame — a LIVE, never-idle stream whose wall time is chunks×gap.
func liveStreamServer(t *testing.T, chunks int, gap time.Duration, requestID string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if requestID != "" {
			w.Header().Set("X-Request-Id", requestID)
		}
		for i := 0; i < chunks; i++ {
			select {
			case <-r.Context().Done():
				return
			default:
			}
			time.Sleep(gap)
			writeSSE(w, fmt.Sprintf(`{"choices":[{"delta":{"reasoning_content":"r%d"},"finish_reason":null}]}`, i))
		}
		writeSSE(w, `{"choices":[{"delta":{"content":"done"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}`)
		writeSSE(w, `[DONE]`)
	}))
}

// TestClass37LiveStreamBeyondHTTPTimeoutSurvivesUnderTotal — a live stream
// that outlives http.Client.Timeout completes when the planner total deadline
// is the ceiling. FAILS on the pre-class-37 code (Client.Timeout killed it at
// 300ms with reasoning still flowing — the 2026-08-30..09-01 shape).
func TestClass37LiveStreamBeyondHTTPTimeoutSurvivesUnderTotal(t *testing.T) {
	srv := liveStreamServer(t, 12, 60*time.Millisecond, "req-class37") // ~720ms live
	defer srv.Close()
	c := streamClient(t, srv.URL, 1)
	c.HTTPClient.Timeout = 300 * time.Millisecond // the executor's ceiling — must NOT bound this call
	req := &Request{Messages: []Message{NewUserMessage("hi")}}
	out, err := c.CallWithRequestStreamRetryDeadlines(req, nil, 200*time.Millisecond, 5*time.Second)
	if err != nil {
		t.Fatalf("live stream beyond http.Client.Timeout must survive under the planner total, got %v", err)
	}
	if !strings.Contains(out, "done") {
		t.Fatalf("stream text = %q", out)
	}
	if c.HTTPClient.Timeout != 300*time.Millisecond {
		t.Fatalf("shared client Timeout was mutated to %v — the executor's ceiling must survive", c.HTTPClient.Timeout)
	}
	if got := c.lastRequestIDString(); got != "req-class37" {
		t.Fatalf("provider request id not captured: %q", got)
	}
	if c.lastHTTPStatus.Load() != 200 {
		t.Fatalf("http status not captured: %d", c.lastHTTPStatus.Load())
	}
	got := c.Log.(*MockLogger)
	if !logContains(got, "ok=true") || !logContains(got, `request_id="req-class37"`) {
		t.Fatalf("ai_call success line must carry request_id: %+v", got.Logs)
	}
}

// TestClass37LegacyIdleCallKeepsHTTPTimeoutCeiling — total=0 (agent paths,
// CallWithRequestStream) keeps the legacy behaviour: http.Client.Timeout
// still kills the live stream, classified client_timeout — the exact
// 2026-08-30..09-01 signature, pinned so the class is recognizable forever.
func TestClass37LegacyIdleCallKeepsHTTPTimeoutCeiling(t *testing.T) {
	srv := liveStreamServer(t, 12, 60*time.Millisecond, "")
	defer srv.Close()
	c := streamClient(t, srv.URL, 1)
	c.HTTPClient.Timeout = 300 * time.Millisecond
	req := &Request{Messages: []Message{NewUserMessage("hi")}}
	_, err := c.CallWithRequestStreamRetry(req, nil, 200*time.Millisecond)
	if err == nil {
		t.Fatalf("legacy path must still be bounded by http.Client.Timeout")
	}
	if !strings.Contains(err.Error(), "Client.Timeout") {
		t.Fatalf("expected the Client.Timeout signature, got %v", err)
	}
	if classifyAIError(err) != string(ClassTotalDeadline) /* class 46: the vocabulary has no client_timeout — an http.Client ceiling IS a total deadline */ {
		t.Fatalf("class = %q, want total_deadline", classifyAIError(err))
	}
	if !logContains(c.Log.(*MockLogger), "class=total_deadline") {
		t.Fatalf("ai_call line must carry class=total_deadline (class 46 vocabulary): %+v", c.Log.(*MockLogger).Logs)
	}
	if c.IsRetryableError(err) {
		t.Fatalf("a Client.Timeout kill must NOT be retried at the client level (the planner loop owns that retry; observed 11 kills / 0 client retries)")
	}
}

// TestClass37TotalDeadlineAbortsWithClass — a live-but-endless stream dies at
// the planner total deadline; the error is the sentinel, the ai_call line
// names class=total_deadline, and the kill is not retried by the client.
func TestClass37TotalDeadlineAbortsWithClass(t *testing.T) {
	srv := liveStreamServer(t, 200, 50*time.Millisecond, "") // ~10s live if never cut
	defer srv.Close()
	c := streamClient(t, srv.URL, 2)
	req := &Request{Messages: []Message{NewUserMessage("hi")}}
	start := time.Now()
	_, err := c.CallWithRequestStreamRetryDeadlines(req, nil, 1*time.Second, 400*time.Millisecond)
	if err == nil || !errors.Is(err, ErrStreamTotalDeadline) {
		t.Fatalf("want ErrStreamTotalDeadline, got %v", err)
	}
	if el := time.Since(start); el > 3*time.Second {
		t.Fatalf("total deadline did not bound the call: %v", el)
	}
	got := c.Log.(*MockLogger)
	if !logContains(got, "class=total_deadline") {
		t.Fatalf("ai_call line must carry class=total_deadline: %+v", got.Logs)
	}
	if logContains(got, "AI API stream failed, retrying") {
		t.Fatalf("a total-deadline kill must not be retried at the client level: %+v", got.Logs)
	}
	if c.IsRetryableError(err) {
		t.Fatalf("total-deadline error must not match the retryable tokens")
	}
	if got := LastErrClass(c); got != "total_deadline" {
		t.Fatalf("LastErrClass = %q", got)
	}
}

// TestClass37IdleDeadlineClass — the idle kill keeps its legacy text
// ("context canceled") and is now classified idle_deadline via the sentinel.
func TestClass37IdleDeadlineClass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		writeSSE(w, `{"choices":[{"delta":{"content":"first"},"finish_reason":null}]}`)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer srv.Close()
	c := streamClient(t, srv.URL, 1)
	req := &Request{Messages: []Message{NewUserMessage("hi")}}
	_, err := c.CallWithRequestStreamRetryDeadlines(req, nil, 200*time.Millisecond, 5*time.Second)
	if err == nil || !errors.Is(err, ErrWatchdogIdle) {
		t.Fatalf("silent stream must die with ErrWatchdogIdle (class 46), got %v", err)
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("legacy 'context canceled' text must survive for the existing grep: %v", err)
	}
	if !logContains(c.Log.(*MockLogger), "class=idle") {
		t.Fatalf("ai_call line must carry class=idle (class 46 vocabulary): %+v", c.Log.(*MockLogger).Logs)
	}
}

// TestClass37FailureResetsStaleTelemetry — a failed call must not inherit the
// previous call's ttfb/reasoning (2026-09-01 06:10:36: an executor reset logged
// reasoning_chars=7092 from the 06:09:33 planner stream).
func TestClass37FailureResetsStaleTelemetry(t *testing.T) {
	ok := liveStreamServer(t, 3, 10*time.Millisecond, "req-ok")
	defer ok.Close()
	c := streamClient(t, ok.URL, 1)
	req := &Request{Messages: []Message{NewUserMessage("hi")}}
	if _, err := c.CallWithRequestStreamRetryDeadlines(req, nil, time.Second, 5*time.Second); err != nil {
		t.Fatalf("setup call failed: %v", err)
	}
	if c.lastReasoningChars.Load() == 0 || c.lastRequestIDString() != "req-ok" {
		t.Fatalf("setup telemetry missing: chars=%d id=%q", c.lastReasoningChars.Load(), c.lastRequestIDString())
	}
	// Now a transport failure on the same client: the server hijacks and
	// closes the connection before writing a byte (deterministic — a dial to a
	// closed port is not: WSL2 swallows the RST and the idle watchdog fires).
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hj, ok := w.(http.Hijacker); ok {
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
		}
	}))
	defer dead.Close()
	c.BaseURL = dead.URL
	_, err := c.CallWithRequestStreamRetryDeadlines(&Request{Messages: []Message{NewUserMessage("hi")}}, nil, time.Second, 5*time.Second)
	if err == nil {
		t.Fatalf("expected a transport failure")
	}
	if c.lastReasoningChars.Load() != 0 || c.lastTTFBMs.Load() != 0 || c.lastHTTPStatus.Load() != 0 || c.lastRequestIDString() != "" {
		t.Fatalf("failed call inherited stale telemetry: chars=%d ttfb=%d status=%d id=%q", c.lastReasoningChars.Load(), c.lastTTFBMs.Load(), c.lastHTTPStatus.Load(), c.lastRequestIDString())
	}
	if classifyAIError(err) != "transport" {
		t.Fatalf("class = %q, want transport (%v)", classifyAIError(err), err)
	}
}

// TestClass37ClassifyAIErrorTable — the observed 2026-08-30..09-01 error
// strings map to the classes the report names (verbatim journal text).
func TestClass37ClassifyAIErrorTable(t *testing.T) {
	cases := []struct {
		msg, want string
	}{
		{"stream interrupted: context deadline exceeded (Client.Timeout or context cancellation while reading body)", string(ClassTotalDeadline) /* class 46: the vocabulary has no client_timeout — an http.Client ceiling IS a total deadline */},
		{"failed to read response: context deadline exceeded (Client.Timeout or context cancellation while reading body)", string(ClassTotalDeadline) /* class 46: the vocabulary has no client_timeout — an http.Client ceiling IS a total deadline */},
		{"failed to read response: read tcp 10.0.0.141:45938->3.173.21.63:443: read: connection reset by peer", "transport"},
		{"stream interrupted: read tcp 10.0.0.141:47328->3.173.21.63:443: read: connection reset by peer", "transport"},
		// class 46: http_status split into http_5xx / http_4xx so the retry
		// policy can tell "the provider is overloaded" from "our request is wrong".
		{"API error (status 429): rate limited", string(ClassHTTP5xx)},
		{"API returned error (status 502): bad gateway", string(ClassHTTP5xx)},
		{"API error (status 400): invalid request", string(ClassHTTP4xx)},
		{"streaming request failed: Post \"https://x\": dial tcp: lookup api.deepseek.com: no such host", "transport"},
		{"AI API key not set", "auth_config"},
		{"stream interrupted: context canceled", "context"},
		{"something else entirely", "other"},
	}
	for _, tc := range cases {
		if got := classifyAIError(errors.New(tc.msg)); got != tc.want {
			t.Errorf("%q → %q, want %q", tc.msg, got, tc.want)
		}
	}
	if classifyAIError(fmt.Errorf("%w (total 20m0s, stream was live): x", ErrStreamTotalDeadline)) != "total_deadline" {
		t.Errorf("wrapped total sentinel not classified")
	}
	if classifyAIError(fmt.Errorf("%w (idle 30s of silence): x", ErrStreamIdleDeadline)) != string(ClassIdle) {
		t.Errorf("wrapped idle sentinel not classified")
	}
}
