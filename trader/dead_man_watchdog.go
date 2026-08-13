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

// watchdogEvent is what a single per-cycle step() produced, so the caller can log
// and act (cancel unfilled entries on reconnect, etc.). Only transitions emit a
// non-none event.
type watchdogEvent int

const (
	wdNone        watchdogEvent = iota // no transition this cycle
	wdWentDown                         // live → disconnected: entries now blocked
	wdReconnected                      // disconnected → awaiting reconcile: sweep unfilled, stay blocked
	wdResumed                          // awaiting reconcile → live: clean reconciliation, entries resumed
)

// step advances the watchdog one trading cycle from the current link status and a
// reconcile probe, returning the event to log/act on. Semantics:
//
//   - A drop (connected=false) → wdWentDown; entries block.
//   - A reconnect → wdReconnected; the caller sweeps unfilled entries, but entries
//     STAY blocked — reconciliation is deferred to a LATER cycle so the link can
//     re-sync before we probe (never resume on the same cycle as the reconnect).
//   - While awaiting reconcile with the link up, a clean probe (reconcileOK()==true)
//     → wdResumed; entries resume. A dirty/failed probe keeps entries blocked and
//     retries next cycle.
//
// reconcileOK is invoked ONLY when the link is up and entries are already blocked
// (awaiting reconciliation), never on the reconnect cycle itself.
func (w *deadManWatchdog) step(connected bool, reconcileOK func() bool) watchdogEvent {
	if _, changed := w.observe(connected); changed {
		switch w.state {
		case dmDisconnected:
			return wdWentDown
		case dmAwaitingReconcile:
			return wdReconnected // reconnected — defer reconcile to a later cycle
		}
		return wdNone
	}
	// No transition; if still awaiting reconcile over a live link, try to reconcile.
	if w.entriesBlocked() && connected && reconcileOK != nil && reconcileOK() && w.reconciled() {
		return wdResumed
	}
	return wdNone
}
