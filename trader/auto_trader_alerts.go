package trader

import (
	"time"

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
