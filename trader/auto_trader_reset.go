package trader

import (
	"fmt"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// P6 (2026-08-17) — THE OWNER RESET. Distinct from ⟳ Re-read: a re-read spends
// ONE budget item on the SAME chain; the reset ABANDONS the current chain,
// restores the whole re-plan budget, clears the NO-TRADE state, and writes a
// fresh plan through the normal read path with trigger_reason "owner reset".
//
// Guards:
//   - confirm-before-spend is the FE's job (the server re-checks eligibility);
//   - disabled with a stated reason when the session is disabled / night /
//     market closed / no plan exists yet;
//   - NEVER touches an open position, its brackets, guardrail counters, or the
//     daily cage — it only appends plan rows + one budget marker;
//   - fail-closed: the normal read path's retries + NO-TRADE fallback govern
//     the actual write, and a failed reset changes nothing about positions.
//
// Append-only: the old chain's rows keep their lifecycles and death reasons.
// The seam is a system_config marker (ResetBaselineKey) recording the version
// the new chain starts measuring from; every budget consumer reads it.

// ResetRefusal explains why an owner reset is not available, in the owner's
// language. Empty Reason means it IS available. Note (when set) is an honest
// post-condition the FE should surface even on success.
type ResetRefusal struct {
	Allowed   bool   `json:"allowed"`
	Reason    string `json:"reason,omitempty"`
	Note      string `json:"note,omitempty"`
	Session   string `json:"session,omitempty"`
	Version   int    `json:"version"`
	ReplanCap int    `json:"replan_cap"`
}

// resetClaimPollInterval / resetClaimWaitMax are the claim-retry cadence the
// reset uses before accepting that a concurrent read won the race. Package vars
// so tests can compress them.
var (
	resetClaimPollInterval = 2 * time.Second
	resetClaimWaitMax      = 30 * time.Second
)

// CanForceReset reports whether an owner reset may run right now, and why not
// when it may not. Read-only: it never writes.
func (at *AutoTrader) CanForceReset(now time.Time) ResetRefusal {
	if !at.dayPlanEnabled() || at.store == nil {
		return ResetRefusal{Reason: "the day plan is off for this trader"}
	}
	reg := at.sessionRegistry(now)
	sess, ok := reg.ActiveSession(now)
	if !ok {
		return ResetRefusal{Reason: "no session is active right now — the planner reads at the session open"}
	}
	if runnable, why := at.sessionRunnable(sess); !runnable {
		reason := why
		if reason == "" {
			reason = fmt.Sprintf("the %s session is not enabled for this strategy", sess.Name)
		}
		return ResetRefusal{Session: sess.Name, Reason: reason}
	}
	if !kernel.IsCMEOpen(now) {
		return ResetRefusal{Session: sess.Name, Reason: "the market is closed (holiday or weekend)"}
	}
	tradeDate := sessionChainDate(sess, now)
	row, err := at.store.Plan().GetLatestPlanForTraderSession(tradeDate, sess.Name, at.id)
	if err != nil {
		return ResetRefusal{Session: sess.Name, Reason: "could not read the current plan"}
	}
	if row == nil {
		// No plan yet: the first read is free and there is no chain to abandon.
		return ResetRefusal{Session: sess.Name, Reason: "no plan has been written yet — the first read costs nothing"}
	}
	out := ResetRefusal{
		Session:   sess.Name,
		Version:   row.Version,
		ReplanCap: at.replanCapFor(sess.Name),
	}
	out.Allowed = true
	return out
}

// ForceReset abandons the current chain, re-arms the budget, and runs a fresh
// planner read with trigger_reason "owner reset". It re-checks eligibility
// itself (the caller's earlier check may be stale). It never touches positions,
// brackets, guardrail counters or the daily cage — plan rows + one marker only.
func (at *AutoTrader) ForceReset(now time.Time) (ResetRefusal, error) {
	gate := at.CanForceReset(now)
	if !gate.Allowed {
		return gate, fmt.Errorf("%s", gate.Reason)
	}
	reg := at.sessionRegistry(now)
	sess, _ := reg.ActiveSession(now)
	tradeDate := sessionChainDate(sess, now)
	row, err := at.store.Plan().GetLatestPlanForTraderSession(tradeDate, gate.Session, at.id)
	if err != nil || row == nil {
		return gate, fmt.Errorf("could not read the current plan")
	}
	// The baseline is the new chain's FIRST version (latest+1, assigned by the
	// single-writer): the abandoned chain ends at vN, and the next read is the new
	// chain's v1 — free, exactly like the original chain's first read — with the
	// full re-plan budget in front of it.
	if err := store.SetResetBaseline(at.store, tradeDate, gate.Session, row.Version+1); err != nil {
		return gate, fmt.Errorf("could not record the reset marker: %w", err)
	}
	at.logInfof("🗓️ OWNER RESET %s %s — chain abandoned at v%d; budget re-armed (%d re-plans).",
		tradeDate, gate.Session, row.Version, gate.ReplanCap)
	at.emitAlert("P1", "owner-reset",
		fmt.Sprintf("reset:%s:%s:v%d", tradeDate, gate.Session, row.Version),
		fmt.Sprintf("%s plan chain reset on request", gate.Session),
		fmt.Sprintf("You reset the %s plan. The old chain (through v%d) is preserved in history; a fresh plan is being read now.",
			gate.Session, row.Version))

	// UI-verification fix (2026-08-18): a concurrent death re-plan can hold the
	// read claim, and the old code silently skipped — the reset reported success
	// while the OTHER read wrote the fresh plan with the wrong trigger_reason.
	// Wait briefly for the claim; if a concurrent read wins the whole window,
	// say so in Note (the outcome — a fresh plan on the re-armed chain — is the
	// same, but the owner is never lied to about who wrote it).
	deadline := time.Now().Add(resetClaimWaitMax)
	for at.PlannerReadInFlight(tradeDate, gate.Session) && time.Now().Before(deadline) {
		time.Sleep(resetClaimPollInterval)
	}
	claimed := at.runPlannerReadWithTriggerClaimed(gate.Session, tradeDate, "owner_reset")
	if !claimed {
		gate.Note = "a concurrent re-plan was already writing the fresh plan; your reset re-armed the budget and that plan is the new chain"
		return gate, nil
	}

	// ITEM 4 — owner sticky levels carry across the seam, re-anchored by price
	// identity (never index), exactly like a re-plan.
	if fresh, fErr := at.store.Plan().GetLatestPlanForTraderSession(tradeDate, gate.Session, at.id); fErr == nil && fresh != nil && fresh.Version > row.Version {
		at.carryOwnerEditsInto(fresh.PlanID, row.Version, fresh.Version)
	}
	return gate, nil
}
