package ninjatrader

import (
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"nofx/telemetry"
)

// FIX-DOUBLE-ENTRY (CTO ruling, 2026-09-26): an attempted entry frame — one
// whose write was STARTED and may have reached the AddOn — is NEVER resent
// blindly. On reconnect, before flushing an attempted ENTRY frame, the guard
// requires FRESH broker truth taken after the reconnect:
//
//   - an order named signal_id is present in a post-reconnect snapshot, or a
//     fill echoed the signal_id → SETTLE: drop the frame; the broker owns it.
//   - absent from a fresh snapshot, no echo, inside the original age window →
//     resend ONCE.
//   - no fresh snapshot within attemptedEntryVerifyWait → DROP + refusal
//     (telemetry.IncGateBlock "attempted_entry_unverified") + WARN.
//
// A MISSED entry is acceptable; a DOUBLE entry is not. Fail closed.
//
// Non-entry frames (cancel, close_position, move_stop) are never queued in
// s.pending — they ride immediate writes (SendCancelOrder/SendClosePosition/
// SendMoveStop) and keep today's behaviour: a replayed cancel is idempotent
// at NT8 and settles via the cancel-confirmation book check; a replayed
// close is idempotent flattening. Entry frames are the only queued kind, so
// the guard keys on attempted + entry signal.

const (
	attemptedEntryVerifyWaitEnv = "ATTEMPTED_ENTRY_VERIFY_WAIT_S"
	attemptedVerifyWaitDefault  = 10 * time.Second
	attemptedGateName           = "attempted_entry_unverified"
	// attemptedEchoTTL bounds the fill-echo memory: an echo older than this
	// cannot prove the post-reconnect state of an attempted frame.
	attemptedEchoTTL = 10 * time.Minute
)

// attemptedReplayGuard carries the reconnect-relative broker-truth state the
// flush path consults for attempted entry frames.
type attemptedReplayGuard struct {
	mu          sync.Mutex
	reconnectAt time.Time            // last accept instant
	echoed      map[string]time.Time // signal_id -> fill-receipt instant

	recheckMu    sync.Mutex
	recheckArmed bool
}

func (g *attemptedReplayGuard) noteReconnect(now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.reconnectAt = now
	if g.echoed == nil {
		g.echoed = map[string]time.Time{}
	} else {
		for id, at := range g.echoed {
			if now.Sub(at) > attemptedEchoTTL {
				delete(g.echoed, id)
			}
		}
	}
}

func (g *attemptedReplayGuard) noteEcho(signalID string, now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.echoed == nil {
		g.echoed = map[string]time.Time{}
	}
	g.echoed[signalID] = now
	if len(g.echoed) > 1024 {
		for id, at := range g.echoed { // bounded: drop the oldest half
			if now.Sub(at) > attemptedEchoTTL || len(g.echoed) > 512 {
				delete(g.echoed, id)
			}
		}
	}
}

func (g *attemptedReplayGuard) echoedAfter(signalID string, since time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	at, ok := g.echoed[signalID]
	return ok && !at.Before(since)
}

func (g *attemptedReplayGuard) reconnectTime() time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.reconnectAt
}

// attemptedVerifyWait reads ATTEMPTED_ENTRY_VERIFY_WAIT_S (default 10s). The
// guard itself is ALWAYS ON (CTO ruling C): this knob tunes the wait only,
// and there is no knob that turns the guard off.
var attemptedEnvWarnOnce sync.Once

func attemptedVerifyWait() time.Duration {
	if v := strings.TrimSpace(os.Getenv(attemptedEntryVerifyWaitEnv)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			return time.Duration(n) * time.Second
		}
		// B1 (2026-09-26): a knob value the guard refuses must say so once, out loud.
		attemptedEnvWarnOnce.Do(func() {
			slog.Warn("tcp_server: ATTEMPTED_ENTRY_VERIFY_WAIT_S unparseable or <1s — using the default",
				"op", "attempted_wait_parse", "raw", v, "default", attemptedVerifyWaitDefault.String())
		})
	}
	return attemptedVerifyWaitDefault
}

// attemptedWait is the effective hold window (test override first).
func (s *TCPServer) attemptedWait() time.Duration {
	if s.attemptedVerifyWaitOverride > 0 {
		return s.attemptedVerifyWaitOverride
	}
	return attemptedVerifyWait()
}

type attemptedDecision int

const (
	attemptedResend attemptedDecision = iota
	attemptedHold
	attemptedSettle
	attemptedDrop
)

// decideAttemptedEntry answers what the flush may do with an attempted entry
// frame at now. reconnectAt must be non-zero (an accept happened); if the
// server has never had a connection the frame is simply flushed (no reconnect
// context exists to require fresh truth — today's behaviour).
func (s *TCPServer) decideAttemptedEntry(sig SignalPayload, now time.Time) attemptedDecision {
	recAt := s.attempted.reconnectTime()
	if recAt.IsZero() {
		return attemptedResend
	}
	// 1. A fill/order_update echoing the signal after the reconnect → broker
	// already executed it. Settle.
	if s.attempted.echoedAfter(sig.SignalID, recAt) {
		return attemptedSettle
	}
	// 2. Fresh broker truth taken AFTER the reconnect — from the frame's OWN
	// account book (P1, DS-101 adversarial review, 2026-09-26): another
	// account's fresh book must never decide this account's frame. A fresh
	// EMPTY book for B resend-deciding A's frame would re-open the double
	// exactly when A's book lags. Only a legacy frame with NO account may use
	// the any-account book, and that path is logged at WARN.
	snap, recvAt, ok, legacy := s.freshBookFor(sig, recAt)
	if ok && recvAt.After(recAt) {
		if legacy {
			s.logger.Warn("tcp_server: attempted-entry decision used the any-account book (frame carries no account)",
				"op", "attempted_entry_legacy_book", "signal_id", sig.SignalID)
		}
		for _, o := range snap.Orders {
			if strings.EqualFold(strings.TrimSpace(o.Name), strings.TrimSpace(sig.SignalID)) {
				return attemptedSettle
			}
		}
		// A fresh OWN book that does NOT hold the order: the frame never landed.
		// Resend once (checkSignalAge still gates at the write).
		return attemptedResend
	}
	// 3. The frame's own book is not fresh (or absent): bounded wait, then
	// fail closed. Another account's evidence does not shorten this.
	if now.Sub(recAt) > s.attemptedWait() {
		return attemptedDrop
	}
	return attemptedHold
}

// freshBookFor picks the book the guard may trust for this frame: the frame's
// own account book when Account is set (P1), the any-account book only for
// legacy empty-account frames (legacy=true).
func (s *TCPServer) freshBookFor(sig SignalPayload, recAt time.Time) (OrderSnapshotPayload, time.Time, bool, bool) {
	if strings.TrimSpace(sig.Account) != "" {
		snap, recvAt, ok := s.orderSnaps.LatestReceived(sig.Account)
		return snap, recvAt, ok, false
	}
	snap, recvAt, ok := s.orderSnaps.LatestReceivedAny()
	return snap, recvAt, ok, true
}

// scheduleAttemptedRecheck arms the one-shot re-flush for held frames. It is
// idempotent per wait window; the accept-flush and every SendSignal-flush call
// the guard too, so the timer only needs to cover the quiet period.
func (s *TCPServer) scheduleAttemptedRecheck() {
	s.attempted.recheckMu.Lock()
	if s.attempted.recheckArmed {
		s.attempted.recheckMu.Unlock()
		return
	}
	s.attempted.recheckArmed = true
	s.attempted.recheckMu.Unlock()
	time.AfterFunc(s.attemptedWait()+500*time.Millisecond, func() {
		s.attempted.recheckMu.Lock()
		s.attempted.recheckArmed = false
		s.attempted.recheckMu.Unlock()
		if err := s.flushPending(); err != nil {
			// B1 (2026-09-26): the re-flush refused or failed is a WARN, never silent.
			s.logger.Warn("tcp_server: attempted-entry recheck flush failed",
				"op", "attempted_entry_recheck_flush", "err", err)
		}
	})
}

// isEntrySignal reports whether a queued frame is an ENTRY (the only kind the
// guard treats specially). Today every SignalPayload is an entry frame;
// non-entry commands (cancel/close/move_stop) never enter s.pending.
func isEntrySignal(sig SignalPayload) bool {
	return true
}

// attemptedAddonDedupeStatus reads the far-side build id for the boot line.
// Until an AddOn build shipping the seen-signal dedupe exists, the honest
// value is n/a — the guard must not claim the far side dedupes.
func attemptedAddonDedupeStatus(s *TCPServer) string {
	bid, _ := s.farSideBuild.Load().(string)
	if strings.TrimSpace(bid) == "" {
		return "n/a"
	}
	return "no (far-side build " + bid + " predates the seen-signal dedupe)"
}

// --- test seams (nil in production) ---

// SetWriteFrameHookForTest overrides the frame write in the flush path.
func (s *TCPServer) SetWriteFrameHookForTest(fn func(c net.Conn, sig SignalPayload) error) {
	s.writeFrameHook = fn
}

// SetAttemptedVerifyWaitForTest overrides the hold window.
func (s *TCPServer) SetAttemptedVerifyWaitForTest(d time.Duration) {
	s.attemptedVerifyWaitOverride = d
}

// SetReconnectAtForTest stamps the reconnect instant directly.
func (s *TCPServer) SetReconnectAtForTest(t time.Time) {
	s.attempted.noteReconnect(t)
}

// NoteEchoForTest feeds a fill echo straight into the guard's echo memory —
// the production path is the FrameFill case in the read loop; the seam skips
// the socket (P2 pin, DS-101 adversarial review 2026-09-26).
func (s *TCPServer) NoteEchoForTest(signalID string, now time.Time) {
	s.attempted.noteEcho(signalID, now)
}

// GateBlockCountForTest exposes the counted refusal for the gate name.
func GateBlockCountForTest(trader, gate string) int {
	_, table := telemetry.GateBlockSnapshot()
	return table[trader][gate]
}
