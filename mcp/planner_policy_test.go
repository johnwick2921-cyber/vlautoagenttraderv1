package mcp

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// E1 — THE HONESTY PIN. Every field on the boot line is compared against the
// function that ENFORCES it. No literal appears in this test except the field
// NAMES. On the class-41 line this fails: watchdog_log, keepalive,
// serialize_executor and resend_identical were string literals, and its
// fixture asserted those same literals, so the line could drift from reality
// undetectably.
func TestClass46BootLineEveryFieldReadFromEnforcer(t *testing.T) {
	line := PlannerClientPolicyLine()

	want := map[string]string{
		"tries":            fmt.Sprintf("%d", StreamRetryTries()),
		"keepalive_set":    DialerKeepAliveSet().String(),
		"resend_identical": fmt.Sprintf("%t", ResendIdenticalOnProviderFailure()),
		"serialize":        fmt.Sprintf("%t", SerializeExecutorDuringPlannerStream()),
		"storm_cap":        fmt.Sprintf("%d", StormCapPerRead()),
		"trace":            fmt.Sprintf("%t", TransportTraceEnabled()),
	}
	for field, v := range want {
		token := field + "=" + v
		if !strings.Contains(line, token) {
			t.Errorf("field %q must be READ from its enforcer: want %q in\n%s", field, token, line)
		}
	}
	// backoff is the schedule, joined from the resolver.
	sched := StreamRetryBackoffSchedule()
	parts := make([]string, 0, len(sched))
	for _, d := range sched {
		parts = append(parts, d.String())
	}
	if !strings.Contains(line, "backoff="+strings.Join(parts, "→")) {
		t.Errorf("backoff not read from StreamRetryBackoffSchedule():\n%s", line)
	}
	// watchdog carries BOTH resolved timers and says it measures data.
	// The line carries the resolver default and, when an override is tighter,
	// the EFFECTIVE limit too (owner ruling 2026-09-02).
	wantWatchdog := fmt.Sprintf("watchdog=pre%ds/post%ds", WatchdogPreTokenSeconds(), WatchdogPostTokenSeconds())
	if !strings.Contains(line, wantWatchdog) {
		t.Errorf("watchdog fields not read from their resolvers:\n%s", line)
	}
	if eff := EffectivePostTokenSeconds(PlannerIdleOverrideSeconds()); eff != WatchdogPostTokenSeconds() &&
		!strings.Contains(line, fmt.Sprintf("→eff%ds", eff)) {
		t.Errorf("a tighter override must show its effective limit:\n%s", line)
	}
	// The observed keepalive is honest about being unobserved.
	if _, ok := ObservedKeepAlive(); !ok && !strings.Contains(line, "observed=n/a") {
		t.Errorf("an unobserved keepalive must print n/a, never the set value:\n%s", line)
	}
}

// The line must MOVE when the enforcers move. A literal cannot do that.
func TestClass46BootLineTracksTheResolvers(t *testing.T) {
	base := PlannerClientPolicyLine()
	t.Setenv("AI_PLAN_STORM_CAP", "9")
	t.Setenv("AI_PLAN_WATCHDOG_POST_SECS", "45")
	t.Setenv("AI_PLAN_STREAM_TRIES", "4")
	t.Setenv("AI_PLAN_TRACE", "off")
	moved := PlannerClientPolicyLine()
	if moved == base {
		t.Fatal("the boot line did not change when its resolvers did — a field is a literal")
	}
	for _, want := range []string{"storm_cap=9", "post45s", "tries=4", "trace=false"} {
		if !strings.Contains(moved, want) {
			t.Errorf("want %q in\n%s", want, moved)
		}
	}
}

// A24 — no bare "on"/"off" words on the line: those were the class-41 tell.
func TestClass46BootLineHasNoLiteralOnOff(t *testing.T) {
	line := PlannerClientPolicyLine()
	for _, banned := range []string{"watchdog_log=on", "serialize_executor=off", "resend_identical=on", "keepalive=30s"} {
		if strings.Contains(line, banned) {
			t.Fatalf("class-41 literal survived: %q in\n%s", banned, line)
		}
	}
	// Every "field=value" pair must have a non-empty value.
	for _, m := range regexp.MustCompile(`(\w+)=(\S*)`).FindAllStringSubmatch(line, -1) {
		if m[2] == "" {
			t.Fatalf("field %q printed an empty value:\n%s", m[1], line)
		}
	}
}

// Resolvers clamp instead of trusting junk (A11).
func TestClass46PolicyResolversClamp(t *testing.T) {
	t.Setenv("AI_PLAN_STORM_CAP", "0")
	if StormCapPerRead() != 5 {
		t.Fatalf("storm cap 0 must fall back to the default, got %d", StormCapPerRead())
	}
	t.Setenv("AI_PLAN_STORM_CAP", "999")
	if StormCapPerRead() != 5 {
		t.Fatalf("storm cap 999 is out of range and must fall back, got %d", StormCapPerRead())
	}
	t.Setenv("AI_PLAN_WATCHDOG_PRE_SECS", "junk")
	if WatchdogPreTokenSeconds() != 600 {
		t.Fatalf("junk must fall back, got %d", WatchdogPreTokenSeconds())
	}
	// The pre-token limit must not exceed DeepSeek's ~10 min queue close.
	if WatchdogPreTokenSeconds() > 1200 {
		t.Fatal("pre-token limit exceeds the resolver's own ceiling")
	}
}

// E4 — the watchdog measures DATA, not lines. A stream that emits only
// heartbeat comments AFTER the first token is a stalled generation and must
// die at the post-token limit; the same comments BEFORE the first token are a
// live queue and must not kill it.
func TestClass46WatchdogPostTokenIgnoresHeartbeats(t *testing.T) {
	t.Setenv("AI_PLAN_WATCHDOG_POST_SECS", "15") // resolver floor; the call idle overrides tighter
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		writeSSE(w, `{"choices":[{"delta":{"content":"a"}}]}`) // first token
		for i := 0; i < 40; i++ {                              // heartbeat comments only — a stalled generation
			fmt.Fprint(w, ": keep-alive\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(50 * time.Millisecond)
		}
	}))
	defer srv.Close()
	c := streamClient(t, srv.URL, 1)
	start := time.Now()
	_, err := c.CallWithRequestStreamDeadlines(&Request{}, nil, 300*time.Millisecond, 0)
	if !errors.Is(err, ErrWatchdogIdle) {
		t.Fatalf("heartbeats must NOT keep a stalled generation alive; got %v", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("the watchdog took %v — it is still resetting on comment lines", time.Since(start))
	}
	if got := ClassifyFailure(err, 200); got != ClassIdle {
		t.Fatalf("class = %q want idle", got)
	}
	var fired string
	for _, e := range c.Log.(*MockLogger).GetLogs() {
		if strings.Contains(e.Message, "⏱ watchdog fired:") {
			fired = e.Message
		}
	}
	if !strings.Contains(fired, "post gap=") {
		t.Fatalf("the POST timer must be the one that fired: %q", fired)
	}
}

// Before the first token, heartbeats DO keep it alive — a queued request is
// alive, and DeepSeek's own queue close is the real limit.
func TestClass46WatchdogPreTokenSurvivesHeartbeats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		for i := 0; i < 8; i++ { // queued, heartbeating, no token yet
			fmt.Fprint(w, ": keep-alive\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(60 * time.Millisecond)
		}
		writeSSE(w, `{"choices":[{"delta":{"content":"ok"},"finish_reason":"stop"}]}`)
		writeSSE(w, "[DONE]")
	}))
	defer srv.Close()
	c := streamClient(t, srv.URL, 1)
	out, err := c.CallWithRequestStreamDeadlines(&Request{}, nil, 300*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("a heartbeating QUEUE must survive to its own limit, got %v", err)
	}
	if out != "ok" {
		t.Fatalf("out = %q", out)
	}
}

// E5 — three consecutive 503s: the tries are SPACED by the schedule and the
// per-read cap holds across attempts. Observed 2026-09-02 01:15 CT: 9 calls in
// ~7 s at an edge that was already shedding load.
func TestClass46StormCapBoundsProviderCalls(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(503)
		fmt.Fprint(w, `{"error":{"message":"Server Overloaded"}}`)
	}))
	defer srv.Close()
	c := streamClient(t, srv.URL, 3)
	c.Cfg.StreamBackoff = []time.Duration{20 * time.Millisecond, 40 * time.Millisecond}
	c.ResetStormCounter()

	start := time.Now()
	_, err := c.CallWithRequestStreamRetryDeadlines(&Request{}, nil, 2*time.Second, 0)
	if err == nil {
		t.Fatal("three 503s must fail the call")
	}
	if got := ClassifyFailure(err, 503); got != ClassHTTP5xx {
		t.Fatalf("class = %q want http_5xx", got)
	}
	// Spacing: the schedule was actually waited, not fired back to back.
	if el := time.Since(start); el < 60*time.Millisecond {
		t.Fatalf("tries were not spaced by the backoff: %v", el)
	}
	if c.StormCount() != 3 {
		t.Fatalf("storm counter = %d, want 3", c.StormCount())
	}

	// The cap holds ACROSS attempts: the counter is not reset between them.
	before := calls.Load()
	for i := 0; i < 4; i++ {
		_, _ = c.CallWithRequestStreamRetryDeadlines(&Request{}, nil, 2*time.Second, 0)
	}
	if extra := calls.Load() - before; extra > int32(StormCapPerRead()) {
		t.Fatalf("the cap did not hold: %d further provider calls past a cap of %d", extra, StormCapPerRead())
	}
	var capped bool
	for _, e := range c.Log.(*MockLogger).GetLogs() {
		if strings.Contains(e.Message, "🌩 storm cap reached") {
			capped = true
		}
	}
	if !capped {
		t.Fatal("hitting the cap must be logged loudly, not silently")
	}
	// A fresh read gets a fresh budget.
	c.ResetStormCounter()
	if c.StormCount() != 0 {
		t.Fatal("ResetStormCounter must start a new read's budget")
	}
}

// E6 — the trace names who closed. A peer close mid-body is peer_fin; our own
// watchdog is local_close with its cause.
func TestClass46TraceNamesWhoClosed(t *testing.T) {
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		writeSSE(w, `{"choices":[{"delta":{"content":"a"}}]}`)
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer peer.Close()
	c := streamClient(t, peer.URL, 1)
	c.ResetStormCounter()
	_, _ = c.CallWithRequestStreamDeadlines(&Request{}, nil, 5*time.Second, 0)
	var line string
	for _, e := range c.Log.(*MockLogger).GetLogs() {
		if strings.Contains(e.Message, "🔌 conn trace:") {
			line = e.Message
		}
	}
	if !strings.Contains(line, "closed_by=peer_fin") {
		t.Fatalf("a peer close mid-body must read peer_fin: %q", line)
	}
	for _, want := range []string{"reused=", "bytes=", "elapsed=", "INFERRED"} {
		if !strings.Contains(line, want) {
			t.Fatalf("trace line missing %q: %s", want, line)
		}
	}

	stall := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		writeSSE(w, `{"choices":[{"delta":{"content":"a"}}]}`)
		time.Sleep(1500 * time.Millisecond)
	}))
	defer stall.Close()
	c2 := streamClient(t, stall.URL, 1)
	c2.ResetStormCounter()
	_, _ = c2.CallWithRequestStreamDeadlines(&Request{}, nil, 200*time.Millisecond, 0)
	line = ""
	for _, e := range c2.Log.(*MockLogger).GetLogs() {
		if strings.Contains(e.Message, "🔌 conn trace:") {
			line = e.Message
		}
	}
	if !strings.Contains(line, "closed_by=local_close") || !strings.Contains(line, "cause=") {
		t.Fatalf("our own watchdog close must read local_close with its cause: %q", line)
	}
}

// OWNER RULING 2026-09-02 — the boot line must carry the EFFECTIVE post-token
// limit, not just the resolver default. The first live fire logged "limit 30s"
// while the line advertised post90s: the planner's 30 s override is tighter and
// is what actually closes the stream.
func TestClass46EffectivePostTokenOnBootLine(t *testing.T) {
	t.Setenv("AI_PLAN_WATCHDOG_POST_SECS", "90")
	t.Setenv("AI_PLAN_STREAM_IDLE_SECS", "30")
	if got := EffectivePostTokenSeconds(PlannerIdleOverrideSeconds()); got != 30 {
		t.Fatalf("effective = %d, want the tighter 30", got)
	}
	line := PlannerClientPolicyLine()
	if !strings.Contains(line, "post90s→eff30s(data)") {
		t.Fatalf("the line must show BOTH the default and the effective limit:\n%s", line)
	}
	// No override tighter than the default → no arrow, no noise.
	t.Setenv("AI_PLAN_STREAM_IDLE_SECS", "300")
	if got := EffectivePostTokenSeconds(PlannerIdleOverrideSeconds()); got != 90 {
		t.Fatalf("a looser override must not widen the default: %d", got)
	}
	if l2 := PlannerClientPolicyLine(); strings.Contains(l2, "→eff") {
		t.Fatalf("no arrow when the default already governs:\n%s", l2)
	}
	// Both values are READ: move the default and the line moves.
	t.Setenv("AI_PLAN_WATCHDOG_POST_SECS", "45")
	t.Setenv("AI_PLAN_STREAM_IDLE_SECS", "20")
	if l3 := PlannerClientPolicyLine(); !strings.Contains(l3, "post45s→eff20s(data)") {
		t.Fatalf("both halves must be read from their resolvers:\n%s", l3)
	}
	if EffectivePostTokenSeconds(0) != 45 {
		t.Fatal("override 0 means no override")
	}
}
