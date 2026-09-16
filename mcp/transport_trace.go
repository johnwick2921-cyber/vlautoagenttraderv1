package mcp

import (
	"errors"
	"fmt"
	"net/http/httptrace"
	"sync/atomic"
	"time"
)

// ── CLASS 46 D6 (2026-09-02) — WHO CLOSED THE SOCKET, FROM INSIDE ───────────
// The sockwatch bash loop (unsupervised, 250 ms `ss` poll) never once caught a
// FIN or RST state — blind to the single thing it was built for — and there is
// no tcpdump on the box. This answers the same question from inside the
// process, where the information actually is.
//
// What it can and cannot say, stated plainly: httptrace does not expose TCP
// flags, so `closed_by` is INFERRED, not observed. It is `local_close` when
// OUR cancel cause fired (watchdog or a deadline) and `peer_fin` when the read
// died with no local cause. That inference is sound because those are the only
// two ways this reader stops early — but it is an inference, and the field name
// says peer_fin rather than pretending to have seen a FIN packet.

// ConnTrace is what one streaming call observed about its connection.
type ConnTrace struct {
	Reused     bool
	WasIdleMs  int64
	DialMs     int64
	FirstByte  time.Duration
	Bytes      int64
	Elapsed    time.Duration
	ClosedBy   string // peer_fin | local_close | clean
	LocalCause string // the sentinel when we closed
}

// TraceLine renders the one line (pure — fixture-pinned).
func (t ConnTrace) TraceLine() string {
	return fmt.Sprintf("🔌 conn trace: closed_by=%s reused=%t idle_before=%dms dial=%dms ttfb=%.1fs bytes=%d elapsed=%.1fs%s (class 46; closed_by is INFERRED — httptrace sees no TCP flags)",
		t.ClosedBy, t.Reused, t.WasIdleMs, t.DialMs, t.FirstByte.Seconds(), t.Bytes, t.Elapsed.Seconds(),
		func() string {
			if t.LocalCause != "" {
				return " cause=" + t.LocalCause
			}
			return ""
		}())
}

// newConnTrace wires httptrace hooks onto a request context and returns the
// collector. Nothing here can fail the call: every hook only records.
func newConnTrace() (*ConnTrace, *httptrace.ClientTrace, *atomic.Int64) {
	tr := &ConnTrace{}
	bytes := &atomic.Int64{}
	var dialStart time.Time
	ct := &httptrace.ClientTrace{
		GotConn: func(i httptrace.GotConnInfo) {
			tr.Reused = i.Reused
			tr.WasIdleMs = i.IdleTime.Milliseconds()
		},
		ConnectStart: func(_, _ string) { dialStart = time.Now() },
		ConnectDone: func(_, _ string, err error) {
			if !dialStart.IsZero() {
				tr.DialMs = time.Since(dialStart).Milliseconds()
			}
		},
	}
	return tr, ct, bytes
}

// finish classifies who ended the stream. cause is context.Cause(ctx).
func (t *ConnTrace) finish(readErr, cause error, bytes int64, ttfb time.Duration, elapsed time.Duration) {
	t.Bytes = bytes
	t.FirstByte = ttfb
	t.Elapsed = elapsed
	switch {
	case cause != nil && (errors.Is(cause, ErrWatchdogIdle) || errors.Is(cause, ErrStreamIdleDeadline) ||
		errors.Is(cause, ErrStreamTotalDeadline)):
		t.ClosedBy = "local_close"
		t.LocalCause = cause.Error()
	case readErr != nil:
		t.ClosedBy = "peer_fin"
	default:
		t.ClosedBy = "clean"
	}
}
