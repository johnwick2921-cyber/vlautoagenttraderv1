// ── W2 D2 — THE STAMP WRITER. THE PRODUCTION CALL PATH. ─────────────────────
//
// A31: this writes a LABEL onto a row that already exists. It refuses nothing,
// cancels nothing and changes no order.
//
// PINNED BY PATH, NOT BY NAME (class 113, nofx-8e 2026-09-10). The predicate
// kernel.FadePermissionAt is pure and that purity is checkable — but purity of
// the predicate says nothing about THIS writer, and the writer is where a
// wall-clock read actually hurts: reading time.Now() here would stamp "now"
// instead of "then" while the predicate stayed provably pure. So the caller
// hands the evaluation moment in, and the A29 gate pins the call SITE.
package store

import (
	"encoding/json"
	"strings"

	"gorm.io/gorm"
)

// FadeStamp is the verdict as the store receives it — already resolved by the
// caller, so this package never imports kernel (kernel imports store).
type FadeStamp struct {
	Evaluated  bool
	Permitted  bool
	Exclusions []string
	// Measured is exclusion -> {"measured":x,"threshold":y,"n":z}. Absent
	// rather than zero when nothing was measured.
	Measured map[string]map[string]float64
	// AtMs is the moment the evaluation was made — the EPISODE'S OPEN, passed
	// in, never read from a clock here.
	AtMs int64
}

// StampFadePermission writes the label ONCE. A second call is a no-op by
// PREDICATE — it selects on fade_permitted IS NULL — not by a flag some other
// path could clear. An episode that opened before the day turned keeps the
// verdict it opened with.
//
// A not-evaluated stamp deliberately leaves fade_permitted NULL: recording
// "we could not tell" as false would be indistinguishable from "excluded".
func (s *TouchOutcomeStore) StampFadePermission(id uint, v FadeStamp) error {
	if s == nil || s.db == nil || id == 0 {
		return nil
	}
	if !v.Evaluated {
		// Nothing to record but the attempt; permission stays NULL.
		return nil
	}

	fields := map[string]any{
		"fade_permitted":  v.Permitted,
		"fade_exclusions": strings.Join(v.Exclusions, ","),
	}
	if v.AtMs > 0 {
		fields["fade_evaluated_ms"] = v.AtMs
	}
	if len(v.Measured) > 0 {
		if b, err := json.Marshal(v.Measured); err == nil {
			fields["fade_measured"] = string(b)
		}
	}

	// FIXED AT OPEN: only rows with no verdict yet are touched.
	return s.db.Model(&TouchOutcomeRow{}).
		Where("id = ? AND fade_permitted IS NULL", id).
		Updates(fields).Error
}

// CountFadeLabels reports today's label distribution for the boot line and the
// desk strip. Every number is READ from the table; none is inferred.
func (s *TouchOutcomeStore) CountFadeLabels(traderID string, sinceMs int64) (permitted, excluded, notEvaluated int64, err error) {
	if s == nil || s.db == nil {
		return 0, 0, 0, nil
	}
	q := func() *gorm.DB { return scopeTrader(s.db.Model(&TouchOutcomeRow{}), traderID) }
	if err = q().Where("fade_permitted = ? AND opened_at_ms >= ?", true, sinceMs).Count(&permitted).Error; err != nil {
		return
	}
	if err = q().Where("fade_permitted = ? AND opened_at_ms >= ?", false, sinceMs).Count(&excluded).Error; err != nil {
		return
	}
	err = q().Where("fade_permitted IS NULL AND opened_at_ms >= ?", sinceMs).Count(&notEvaluated).Error
	return
}

// FadeBackfillResult is the D6 three-state report. Ran distinguishes "ran and
// found nothing" from "has not run" — a zero and an unknown must never render
// alike (A24/A30).
type FadeBackfillResult struct {
	Ran            bool
	Recomputed     int
	Unrecomputable int
	Untouched      int
}

// OpenFadeCandidates returns episodes with no fade verdict yet, oldest first.
func (s *TouchOutcomeStore) OpenFadeCandidates(traderID string, sinceMs int64) ([]TouchOutcomeRow, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var rows []TouchOutcomeRow
	err := scopeTrader(s.db, traderID).
		Where("fade_permitted IS NULL AND fade_exclusions IS NULL").
		Order("opened_at_ms ASC").Find(&rows).Error
	return rows, err
}

// MarkFadeUnrecomputable records WHY a row could not be labelled, in the
// exclusions column with an "unrecomputable:" prefix, leaving fade_permitted
// NULL. A row nobody could recompute must not be indistinguishable from one
// nobody has looked at yet (A30).
func (s *TouchOutcomeStore) MarkFadeUnrecomputable(id uint, why string) error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Model(&TouchOutcomeRow{}).
		Where("id = ? AND fade_permitted IS NULL", id).
		Update("fade_exclusions", "unrecomputable:"+why).Error
}
