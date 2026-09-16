// ── ONE SETUP (dispatch 102, 2026-09-10) — THE STORE SIDE ────────────────────
//
// Three things live here and none of them refuses, cancels or places anything:
//
//	· the SCENARIO RECORD — one system_config row per (trader, plan, version)
//	  holding every armable scenario's latest three-legged verdict, written at
//	  the arm seam and merged into the card's plan response as `one_setup`;
//	· the EPISODE STAMP — W2's pattern: a label written ONCE onto an existing
//	  touch_outcomes row (WHERE one_setup_verdicts IS NULL), never rewritten;
//	· the COUNTERS — the seam's per-session-day classes, read for the boot
//	  line and the strip (class 35: counters record, never infer).
//
// This package never imports kernel (kernel imports store), so the verdict
// arrives already resolved as strings.
package store

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

// D9 boundaries — the boot instants the dispatch names, from the markers on
// dev (2026-09-10, CT): W2's fade permission booted 18:47:07 (marker 4dc0fae1);
// W1's episode contract booted ~12:52 (marker b11659ea 12:52:37). Verdicts are
// recomputed for episodes opened at or after W2's boot (the permission stamp
// exists from there); follow-plans for episodes at or after W1's. Derived
// from the dates in CT so a zone error is visible, not silent.
var (
	OneSetupVerdictEraStart = time.Date(2026, 9, 10, 18, 47, 7, 0, ctLocationForEra())
	FollowPlanEraStart      = time.Date(2026, 9, 10, 12, 52, 37, 0, ctLocationForEra())
)

func ctLocationForEra() *time.Location {
	if loc, err := time.LoadLocation("America/Chicago"); err == nil {
		return loc
	}
	return time.FixedZone("CDT", -5*3600)
}

// OneSetupKey is the scenario record's system_config key — sibling of
// ScenarioMetaKey, so the card reads both for one plan version.
func OneSetupKey(traderID, planID string, version int) string {
	return fmt.Sprintf("one_setup:%s:%s:v%d", traderID, planID, version)
}

// OneSetupScenarioRecord is one scenario's entry in the record.
type OneSetupScenarioRecord struct {
	Allowed     bool    `json:"allowed"`
	Level       string  `json:"level"`
	Play        string  `json:"play"`
	Permission  string  `json:"permission"`
	Reason      string  `json:"reason"`
	BestPrice   float64 `json:"best_price,omitempty"`
	BestNames   string  `json:"best_names,omitempty"`
	BestGrade   string  `json:"best_grade,omitempty"`
	Target      string  `json:"target,omitempty"`  // "first_obstacle@<px>" | "authored(obstacle_missing)" | "authored(obstacle_wrong_side)"
	Waiting     bool    `json:"waiting,omitempty"` // D4: allowed, but another scenario holds the plan's one slot
	Rank        int     `json:"rank,omitempty"`
	EvaluatedMs int64   `json:"evaluated_ms"`
}

// OneSetupRecord is the whole plan version's record.
type OneSetupRecord struct {
	Enabled     bool                              `json:"enabled"`
	MinGrade    string                            `json:"min_grade"`
	EvaluatedMs int64                             `json:"evaluated_ms"`
	Scenarios   map[string]OneSetupScenarioRecord `json:"scenarios"`
}

// SaveOneSetupRecord writes the record for a plan version (last write wins —
// it is the CURRENT verdict, and the episode stamp is the durable one).
func (s *Store) SaveOneSetupRecord(traderID, planID string, version int, r OneSetupRecord) error {
	if s == nil || planID == "" || version <= 0 {
		return nil
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return s.SetSystemConfig(OneSetupKey(traderID, planID, version), string(b))
}

// GetOneSetupRecord reads it; nil when absent or unreadable — the card then
// emits no `one_setup` key rather than an empty one (A24).
func (s *Store) GetOneSetupRecord(traderID, planID string, version int) *OneSetupRecord {
	if s == nil {
		return nil
	}
	raw, err := s.GetSystemConfig(OneSetupKey(traderID, planID, version))
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil
	}
	var r OneSetupRecord
	if json.Unmarshal([]byte(raw), &r) != nil || len(r.Scenarios) == 0 {
		return nil
	}
	return &r
}

// OneSetupStamp is the verdict as the store receives it.
type OneSetupStamp struct {
	Scenario    string
	Verdicts    string // "level=<v> play=<v> permission=<v>"
	Reason      string
	EvaluatedMs int64
	Backfill    string // "" live; "recomputed" for D9
}

// StampOneSetup writes the verdict ONCE onto an episode row. A second call is
// a no-op BY PREDICATE (one_setup_verdicts IS NULL), not by a flag.
func (s *TouchOutcomeStore) StampOneSetup(id uint, v OneSetupStamp) error {
	if s == nil || s.db == nil || id == 0 || v.Verdicts == "" {
		return nil
	}
	fields := map[string]any{
		"one_setup_verdicts": v.Verdicts,
		"one_setup_reason":   v.Reason,
		"one_setup_scenario": v.Scenario,
	}
	if v.EvaluatedMs > 0 {
		fields["one_setup_evaluated_ms"] = v.EvaluatedMs
	}
	if v.Backfill != "" {
		fields["one_setup_backfill"] = v.Backfill
	}
	return s.db.Model(&TouchOutcomeRow{}).
		Where("id = ? AND one_setup_verdicts IS NULL", id).
		Updates(fields).Error
}

// MarkOneSetupUnrecomputable is D9's other state: WHY a row could not be
// judged, leaving the verdict NULL (A30).
func (s *TouchOutcomeStore) MarkOneSetupUnrecomputable(id uint, why string) error {
	if s == nil || s.db == nil || id == 0 {
		return nil
	}
	return s.db.Model(&TouchOutcomeRow{}).
		Where("id = ? AND one_setup_verdicts IS NULL AND one_setup_backfill IS NULL", id).
		Update("one_setup_backfill", "unrecomputable:"+why).Error
}

// UnstampedEpisodesNear returns the plan's episode rows whose level sits
// within tol of levelPrice and that carry no one-setup verdict yet — the rows
// the seam's stamp lands on. The link is PRICE PROXIMITY within the map's own
// merge width, and it is the seam's own link: W1's scenario_nearest column
// resolved 0 of 999 live rows (A15, one-setup report) and cannot be relied on.
func (s *TouchOutcomeStore) UnstampedEpisodesNear(traderID, planID string, levelPrice, tol float64) ([]TouchOutcomeRow, error) {
	if s == nil || s.db == nil || levelPrice <= 0 {
		return nil, nil
	}
	var rows []TouchOutcomeRow
	err := scopeTrader(s.db, traderID).
		Where("plan_id = ? AND one_setup_verdicts IS NULL AND level_price BETWEEN ? AND ?", planID, levelPrice-tol, levelPrice+tol).
		Order("opened_at_ms ASC").Find(&rows).Error
	return rows, err
}

// OneSetupCounts is the boot line's READ set for a trade date.
type OneSetupCounts struct {
	Armable, Declined                    int
	Level, Play, Day, NotEvaluated, Wait int
	ObstacleBelowFloor, ObstacleMissing  int
	DeclinedWhileResting                 int
	Retired                              int
	Readable                             bool
}

// Counter classes (the arm-refusal counter namespace, per session-day).
const (
	OneSetupClassArmable         = "one_setup:armable"
	OneSetupClassLevel           = "one_setup:level"
	OneSetupClassPlay            = "one_setup:play"
	OneSetupClassDay             = "one_setup:day"
	OneSetupClassNotEvaluated    = "one_setup:not_evaluated"
	OneSetupClassWaiting         = "one_setup:waiting"
	OneSetupClassResting         = "one_setup:declined_while_resting"
	OneSetupClassRetired         = "one_setup:retired" // a declined scenario's unplaced authorization retired at placement time (owner ruling 2026-09-11)
	OneSetupClassObstacleFloor   = "obstacle_below_floor"
	OneSetupClassObstacleMissing = "obstacle_missing"
)

// OneSetupCountsFor sums each class across the trade date's sessions. The
// keys are the arm-refusal counter's own; sessions are discovered by prefix,
// never listed by hand.
func OneSetupCountsFor(st *Store, traderID, tradeDate string) OneSetupCounts {
	c := OneSetupCounts{}
	if st == nil || st.gdb == nil {
		return c
	}
	sum := func(class string) int {
		var rows []struct{ Value string }
		prefix := "arm_refusals_0b:" + traderID + ":" + tradeDate + ":"
		if err := st.gdb.Raw("SELECT value FROM system_config WHERE key LIKE ? AND key LIKE ?", prefix+"%", "%:"+class).Scan(&rows).Error; err != nil {
			return 0
		}
		n := 0
		for _, r := range rows {
			var v int
			if _, err := fmt.Sscanf(strings.TrimSpace(r.Value), "%d", &v); err == nil {
				n += v
			}
		}
		return n
	}
	c.Readable = true
	c.Armable = sum(OneSetupClassArmable)
	c.Level, c.Play, c.Day = sum(OneSetupClassLevel), sum(OneSetupClassPlay), sum(OneSetupClassDay)
	c.NotEvaluated, c.Wait = sum(OneSetupClassNotEvaluated), sum(OneSetupClassWaiting)
	c.DeclinedWhileResting = sum(OneSetupClassResting)
	c.Retired = sum(OneSetupClassRetired)
	c.ObstacleBelowFloor, c.ObstacleMissing = sum(OneSetupClassObstacleFloor), sum(OneSetupClassObstacleMissing)
	c.Declined = c.Level + c.Play + c.Day + c.NotEvaluated
	return c
}

// ── THE FOLLOW-PLAN STORE ─────────────────────────────────────────────────────

// FollowPlanStamp is the recorder's output as the store receives it. Pointer
// fields are written only when non-nil, so a partial plan (broken, not yet
// retested) writes exactly the fields it knows and leaves the rest NULL.
type FollowPlanStamp struct {
	BreakAtMs      *int64
	BreakDir       *string
	RetestAtMs     *int64
	EntryPx        *float64
	EntryBasis     *string
	MAE10, MFE10   *float64
	Net10          *float64
	MAE20, MFE20   *float64
	Net20          *float64
	RetestOutcome  *string
	RoleReversed   *bool
	BiasWouldFlip  *string
	PlanBiasFrozen *string
	State          string
	Backfill       string
}

// StampFollowPlan writes the follow-plan fields that are KNOWN. Known fields
// are written once (a NULL column takes the value; a non-NULL one is kept —
// an event's first observation is the record). State is always updated.
func (s *TouchOutcomeStore) StampFollowPlan(id uint, f FollowPlanStamp) error {
	if s == nil || s.db == nil || id == 0 {
		return nil
	}
	set := func(col string, v any) {
		q := s.db.Model(&TouchOutcomeRow{}).Where("id = ? AND "+col+" IS NULL", id)
		_ = q.Update(col, v).Error
	}
	if f.BreakAtMs != nil {
		set("follow_break_at_ms", *f.BreakAtMs)
	}
	if f.BreakDir != nil {
		set("follow_break_dir", *f.BreakDir)
	}
	if f.RetestAtMs != nil {
		set("follow_retest_at_ms", *f.RetestAtMs)
	}
	if f.EntryPx != nil {
		set("follow_entry_px", *f.EntryPx)
	}
	if f.EntryBasis != nil {
		set("follow_entry_basis", *f.EntryBasis)
	}
	if f.MAE10 != nil {
		set("follow_mae10", *f.MAE10)
	}
	if f.MFE10 != nil {
		set("follow_mfe10", *f.MFE10)
	}
	if f.Net10 != nil {
		set("follow_net10", *f.Net10)
	}
	if f.MAE20 != nil {
		set("follow_mae20", *f.MAE20)
	}
	if f.MFE20 != nil {
		set("follow_mfe20", *f.MFE20)
	}
	if f.Net20 != nil {
		set("follow_net20", *f.Net20)
	}
	if f.RetestOutcome != nil {
		set("follow_retest_outcome", *f.RetestOutcome)
	}
	if f.RoleReversed != nil {
		set("follow_role_reversed", *f.RoleReversed)
	}
	if f.BiasWouldFlip != nil {
		set("bias_would_flip_to", *f.BiasWouldFlip)
	}
	if f.PlanBiasFrozen != nil {
		set("plan_bias_frozen", *f.PlanBiasFrozen)
	}
	fields := map[string]any{}
	if f.State != "" {
		fields["follow_state"] = f.State
	}
	if f.Backfill != "" {
		fields["follow_backfill"] = f.Backfill
	}
	if len(fields) == 0 {
		return nil
	}
	return s.db.Model(&TouchOutcomeRow{}).Where("id = ?", id).Updates(fields).Error
}

// OpenFollowPlans returns the rows the recorder still watches: opened at or
// after sinceMs and not in a terminal follow state. A NULL follow_state is an
// episode the recorder has never seen — those are the rows it opens.
func (s *TouchOutcomeStore) OpenFollowPlans(traderID, symbol string, sinceMs int64) ([]TouchOutcomeRow, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var rows []TouchOutcomeRow
	err := scopeTrader(s.db, traderID).
		Where("symbol = ? AND opened_at_ms >= ? AND (follow_state IS NULL OR follow_state IN ('open','broken_no_retest','retested'))", symbol, sinceMs).
		Order("opened_at_ms ASC").Find(&rows).Error
	return rows, err
}

// RetestVerdictFor joins the retest's OWN detector verdict: the next episode
// on the same level (price within tol) whose open is at or after retestMs,
// in the same session-day. ok=false when the detector has not written it yet.
func (s *TouchOutcomeStore) RetestVerdictFor(traderID, symbol string, levelPrice, tol float64, retestMs, sessionDayMs, sessionDayEndMs int64, excludeID uint) (outcome string, entrySide string, ok bool) {
	if s == nil || s.db == nil {
		return "", "", false
	}
	var r TouchOutcomeRow
	err := scopeTrader(s.db, traderID).
		Where("symbol = ? AND id <> ? AND level_price BETWEEN ? AND ? AND opened_at_ms >= ? AND opened_at_ms >= ? AND opened_at_ms < ?",
			symbol, excludeID, levelPrice-tol, levelPrice+tol, retestMs-60_000, sessionDayMs, sessionDayEndMs).
		Order("opened_at_ms ASC").First(&r).Error
	if err != nil || r.ID == 0 {
		return "", "", false
	}
	return r.Outcome, r.EntrySide, true
}

// FollowPlanCounts is the boot line's READ set: breaks, retests, role-reversed
// of retested, since sinceMs.
type FollowPlanCounts struct {
	Rows, Breaks, Retests, Reversed, RetestJudged int64
	Readable                                      bool
}

func (s *TouchOutcomeStore) CountFollowPlans(traderID string, sinceMs int64) FollowPlanCounts {
	c := FollowPlanCounts{}
	if s == nil || s.db == nil {
		return c
	}
	q := func() *gorm.DB {
		return scopeTrader(s.db.Model(&TouchOutcomeRow{}), traderID).Where("opened_at_ms >= ?", sinceMs)
	}
	if err := q().Count(&c.Rows).Error; err != nil {
		return c
	}
	_ = q().Where("follow_break_at_ms IS NOT NULL").Count(&c.Breaks).Error
	_ = q().Where("follow_retest_at_ms IS NOT NULL").Count(&c.Retests).Error
	_ = q().Where("follow_role_reversed IS NOT NULL").Count(&c.RetestJudged).Error
	_ = q().Where("follow_role_reversed = ?", true).Count(&c.Reversed).Error
	c.Readable = true
	return c
}

// FollowBackfillResult is D9's three-state report for the follow-plans.
type FollowBackfillResult struct {
	Ran                                   bool
	Recomputed, Unrecomputable, Untouched int
}

// OneSetupBackfillResult is D9's three-state report for the verdicts.
type OneSetupBackfillResult struct {
	Ran                                   bool
	Recomputed, Unrecomputable, Untouched int
}

// CountEpisodesBefore is the "untouched" count: rows older than the boundary
// that the backfill deliberately never looks at.
func (s *TouchOutcomeStore) CountEpisodesBefore(traderID string, beforeMs int64) int64 {
	if s == nil || s.db == nil {
		return 0
	}
	var n int64
	_ = scopeTrader(s.db.Model(&TouchOutcomeRow{}), traderID).Where("opened_at_ms < ?", beforeMs).Count(&n).Error
	return n
}

// OpenOneSetupCandidates returns rows since sinceMs with no verdict and no
// backfill mark yet, oldest first.
func (s *TouchOutcomeStore) OpenOneSetupCandidates(traderID string, sinceMs int64) ([]TouchOutcomeRow, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var rows []TouchOutcomeRow
	err := scopeTrader(s.db, traderID).
		Where("opened_at_ms >= ? AND one_setup_verdicts IS NULL AND one_setup_backfill IS NULL", sinceMs).
		Order("opened_at_ms ASC").Find(&rows).Error
	return rows, err
}

// FollowNet is the follow's mark at a horizon, net of friction: (close −
// entry) × direction − friction. Direction +1 long, −1 short.
func FollowNet(entry, closeAt float64, dir int, frictionPts float64) float64 {
	return (closeAt-entry)*float64(dir) - frictionPts
}

// FollowFrictionPts is round 17's 2-pt round-trip friction assumption [T].
const FollowFrictionPts = 2.0

var _ = math.Abs

// PoolReadBefore returns the candidate-pool rows of the LAST read at or before
// atMs for a trader/symbol, and the read's instant. ok=false when no read
// exists in the window [sinceMs, atMs] — the backfill then says
// unrecomputable:no_pool_read rather than judging against a later map (A24).
func (s *CandidatePoolStore) PoolReadBefore(traderID, symbol string, sinceMs, atMs int64) (rows []CandidatePoolRow, readAtMs int64, ok bool) {
	if s == nil || s.db == nil {
		return nil, 0, false
	}
	var at struct{ ReadAtMs int64 }
	err := s.db.Model(&CandidatePoolRow{}).Select("MAX(read_at_ms) AS read_at_ms").
		Where("trader_id = ? AND symbol = ? AND read_at_ms >= ? AND read_at_ms <= ?", traderID, symbol, sinceMs, atMs).Scan(&at).Error
	if err != nil || at.ReadAtMs == 0 {
		return nil, 0, false
	}
	if err := s.db.Where("trader_id = ? AND symbol = ? AND read_at_ms = ?", traderID, symbol, at.ReadAtMs).Find(&rows).Error; err != nil || len(rows) == 0 {
		return nil, 0, false
	}
	return rows, at.ReadAtMs, true
}
