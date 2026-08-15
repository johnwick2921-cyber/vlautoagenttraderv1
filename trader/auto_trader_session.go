package trader

import (
	"fmt"
	"time"

	"nofx/kernel"
)

// P3.1 — SESSION GATES. Wire the P0 registry to live entry gating: entries are
// allowed ONLY inside an ENABLED session's window (NY-only by default), which
// closes the interim/overnight window, plus the spec's no-trade sub-windows
// (first 5m of the session, lunch 12:00–13:30 CT). GATED on day_plan → dormant
// by default. Session-flat at the enabled session's boundary is covered by P2.3's
// eod_flat (the NY flat 14:45 CT sits inside the NY window); multi-session flats
// arrive when ASIA/LONDON earn enablement.
//
// NOTE: the registry's `Killzones` currently hold high-probability ACTIVE windows
// (ny_am/ny_pm), so they are NOT used to block here — the dispatch's "no-trade
// sub-windows" are the spec-card-#5 windows below. Reconciling the killzone
// semantics is a P4 admin decision.

// sessionEntryBlocked reports (reason, blocked): whether NEW entries are blocked
// because we are outside an enabled session window or inside a no-trade
// sub-window. Gated on day_plan → dormant by default.
func (at *AutoTrader) sessionEntryBlocked() (string, bool) {
	if !at.dayPlanEnabled() {
		return "", false
	}
	// TODO(P4): load the admin registry from system_config instead of the default.
	return sessionGateDecision(kernel.DefaultSessionRegistry(), time.Now())
}

// sessionGateDecision is the pure session-gate logic: entries only inside an
// ENABLED session window and outside the no-trade sub-windows (first-5m, lunch).
func sessionGateDecision(reg kernel.SessionRegistry, now time.Time) (string, bool) {
	sess, ok := reg.ActiveSession(now)
	if !ok {
		return "outside all session windows (overnight/interim)", true
	}
	if !sess.Enabled {
		return fmt.Sprintf("%s session not enabled", sess.Name), true
	}
	if inSessionFirst5m(sess, now) {
		return fmt.Sprintf("%s first-5m no-trade window", sess.Name), true
	}
	if kernel.InBlackoutWindow(now, "12:00", "13:30") {
		return "lunch no-trade window (12:00–13:30 CT)", true
	}
	return "", false
}

// inSessionFirst5m reports whether now falls in the first 5 minutes of the
// session window (a no-trade sub-window).
func inSessionFirst5m(sess *kernel.SessionDef, now time.Time) bool {
	start, ok := hhmmToMin(sess.WindowStartCT)
	if !ok {
		return false
	}
	cur := ctMinutesNow(now)
	return cur >= start && cur < start+5
}
