package trader

// B5 — dead-man's watchdog. On an NT8 TCP disconnect / missed heartbeats, NEW
// entries are blocked; a reconnect alone does NOT resume trading — the link must
// first pass a clean positions/orders reconciliation (so the bot never opens on top
// of a state it hasn't re-verified after a gap). Then it auto-resumes.
//
// State machine (pure + testable):
//
//	dmLive ──disconnect──▶ dmDisconnected ──reconnect──▶ dmAwaitingReconcile ──reconciled──▶ dmLive
//	  ▲                         │                              │
//	  └──────────── flap back down (reconnect lost) ───────────┘
//
// Entries are blocked in dmDisconnected and dmAwaitingReconcile. Exits /
// open-position management are never gated by the watchdog.

type deadManState int

const (
	dmLive              deadManState = iota // connected + reconciled → entries allowed
	dmDisconnected                          // link down → entries blocked
	dmAwaitingReconcile                     // reconnected but not yet reconciled → blocked
)

type deadManWatchdog struct {
	state deadManState
}

// observe updates the state from the current link status. Returns (newState,
// changed) so the caller can log transitions.
func (w *deadManWatchdog) observe(connected bool) (deadManState, bool) {
	prev := w.state
	switch w.state {
	case dmLive:
		if !connected {
			w.state = dmDisconnected
		}
	case dmDisconnected:
		if connected {
			w.state = dmAwaitingReconcile // reconnected → must reconcile before resuming
		}
	case dmAwaitingReconcile:
		if !connected {
			w.state = dmDisconnected // flapped back down
		}
	}
	return w.state, w.state != prev
}

// reconciled marks a clean positions/orders reconciliation complete. It resumes
// trading ONLY from the awaiting-reconcile state (a reconcile while merely live is
// a no-op). Returns true if it resumed.
func (w *deadManWatchdog) reconciled() bool {
	if w.state == dmAwaitingReconcile {
		w.state = dmLive
		return true
	}
	return false
}

// entriesBlocked reports whether NEW entries are currently blocked.
func (w *deadManWatchdog) entriesBlocked() bool {
	return w.state != dmLive
}
