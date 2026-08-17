package trader

import (
	"time"

	"nofx/kernel"
	"nofx/store"
)

// W6 — in-app alert emit (the audit's dead wire: AlertStore.Emit had zero
// production callers). emitAlert is the single gated + deduped producer. P0 =
// pop-up + banner-until-ack · P1 = feed · P2 = digest. Gated on day_plan (the
// alert center only renders for a futures day-plan trader).
func (at *AutoTrader) emitAlert(level, kind, eventID, title, body string) {
	if at.store == nil || !at.dayPlanEnabled() {
		return
	}
	_, _ = at.store.Alert().Emit(&store.AlertDB{
		TraderID:  at.id,
		Level:     level,
		Kind:      kind,
		EventID:   eventID, // dedupe (idempotent bus) — repeats are silent no-ops
		Title:     title,
		Body:      body,
		CreatedAt: time.Now().Unix(),
	})
}

// ackedAlertPruneDays is how old an ACKED P2 alert must be before the daily
// prune hides it from the feed (the row survives — audit, not amnesia).
const ackedAlertPruneDays = 7

// maybePruneAckedAlerts runs the alert-feed prune ONCE per CME session-day.
// PruneAckedOlderThan has existed since the alert center shipped and had ZERO
// production callers (B-audit 2026-08-18): the feed grew without bound. P0/P1
// are never auto-pruned.
func (at *AutoTrader) maybePruneAckedAlerts(now time.Time) {
	if !at.dayPlanEnabled() || at.store == nil {
		return
	}
	day := kernel.CMESessionDayKey(now)
	if at.lastAlertPruneDay == day {
		return
	}
	at.lastAlertPruneDay = day
	cutoff := now.AddDate(0, 0, -ackedAlertPruneDays).Unix()
	n, err := at.store.Alert().PruneAckedOlderThan(at.id, cutoff, now.Unix())
	if err != nil {
		at.logErrorf("🧹 alert prune failed: %v", err)
		return
	}
	if n > 0 {
		at.logInfof("🧹 alert prune: hid %d acked P2 alert(s) older than %d days", n, ackedAlertPruneDays)
	}
}
