package api

import (
	"time"

	"nofx/kernel"
)

// planMutationSessionAt resolves the session a plan MUTATION (Ask-Planner
// apply, realign) is about to touch, using the SAME wrap-aware chain date the
// plan READS use — never the wall-clock calendar date. Gaps are a refusal
// (ok=false with nil session), not a nil session handed to downstream readers:
// today's inline form dereferenced sess.Name before checking ok, so a session
// gap panicked the apply route instead of refusing it.
// (F18, WAVE 117 PR-D, ports #117 a6b88b7d.)
func (s *Server) planMutationSessionAt(traderID string, now time.Time) (*kernel.SessionDef, string, bool) {
	session, ok := s.planRegistry().ActiveSession(now)
	if !ok || session == nil {
		return nil, "", false
	}
	date, ok := kernel.PlanChainTradeDate(session, now)
	if !ok {
		return nil, "", false
	}
	// P1 — sessionRunnable, not the raw registry flag: an edit must be allowed
	// for a session the bot actually runs.
	if s.traderManager != nil {
		if at, err := s.traderManager.GetTrader(traderID); err == nil && at != nil {
			if runnable, _ := at.SessionRunnable(session); !runnable {
				return nil, "", false
			}
		}
	}
	return session, date, true
}
