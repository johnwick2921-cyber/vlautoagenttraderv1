package store

import (
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
)

// ── D2 — TOUCH OUTCOMES (1B, 2026-09-03) ─────────────────────────────────────
//
// One row per D1′ episode, written at episode CLOSE. This is the table that
// replaces two biased instruments: touch_telemetry's "rejection" (the close is
// still on its starting side — ≈0.69 on IID noise BY CONSTRUCTION) and
// level_stats_calc's "reacted" (any ≥reactPts move away, so a blast-through
// scored as a reaction). Every reaction rate ever published from those — 84%,
// 70.3%, 75.1% — is an artifact of the predicate.
//
// AMBIGUOUS ROWS ARE WRITTEN, FLAGGED AND EXCLUDED FROM THE RATE, never
// dropped: a rate that silently discards its hard cases lies about its base.
type TouchOutcomeRow struct {
	ID       uint   `gorm:"primaryKey"`
	TraderID string `gorm:"index"`
	Symbol   string `gorm:"index"`
	// The level this episode belongs to.
	LevelPrice float64 `gorm:"index"`
	LevelKind  string  `gorm:"index"`
	// Was the level SEATED in the plan, or only a candidate? The selection
	// question is off-policy and needs the excluded pool too (B2/B3).
	CandidateSeated bool   `gorm:"index"`
	PlanID          string `gorm:"index"`
	PlanVersion     int
	Session         string `gorm:"index"`
	// Ordinal comes from the STORE, never an in-memory counter (C4): touch
	// numbering must survive a restart.
	Ordinal int `gorm:"index"`
	// The resolved detector scope this episode was judged under, recorded so a
	// later reader knows which instrument produced the verdict.
	K       float64
	Delta   float64
	BandPts float64
	Horizon int
	ExitOn  string
	// The episode itself.
	EntrySide  string // "below" | "above" — the side price came FROM
	ExitSide   string // "below" | "above" | "" when ambiguous
	Outcome    string `gorm:"index"` // hold | break | ambiguous_span | ambiguous_horizon
	Ambiguous  bool   `gorm:"index"`
	BarsToExit int
	MFEPts     float64
	MAEPts     float64
	OpenedAtMs int64 `gorm:"index"`
	ClosedAtMs int64
	// FormedAtMs (WAVE A / D1) is the level's own birth instant, copied from
	// DetectedLevel.FormedAtMs. An episode that opened before it is lookahead
	// by construction — the recorder now refuses to scan there, and this
	// column is what lets a later reader CHECK that, instead of trusting it.
	// 0 means the level did not carry a formation time, not "formed at epoch".
	FormedAtMs int64 `gorm:"index"`
	// Validity (WAVE A / D1e) classifies a row for readers. Rows written
	// before the recorder was fixed are marked, never deleted and never
	// silently repaired — a rate computed over them would be the artifact
	// this wave exists to remove.
	//
	// DELIBERATELY NO SQL DEFAULT. SQLite's ADD COLUMN ... DEFAULT 'valid'
	// would stamp all 677 pre-existing contaminated rows as valid the moment
	// AutoMigrate ran, which is the exact fabrication this column exists to
	// prevent. An empty validity means NOT CERTIFIED and is excluded from
	// every rate until the migration classifies it.
	Validity string `gorm:"index"`

	// ── W1 EPISODE CONTRACT ─────────────────────────────────────────────────
	// All POINTERS on purpose: NULL means NOT CAPTURED and must never render as
	// 0 or "" (A24). The precedent is Validity above, which deliberately takes
	// no SQL default so a migration cannot stamp uncertified rows as certified.

	// ScenarioNearest is the touch → scenario link, and it is NOT identity.
	// PlanScenario carries no level reference at all, so the tie can only ever
	// be nearest-by-price; the column is named for what it is so no reader
	// mistakes it for what the planner meant. NULL whenever two scenarios sit
	// inside the band or nothing is close — ambiguity is NULL, never
	// nearest-wins. ScenarioLinkBasis always states how, or why not.
	// LevelID is the named candidate; NULL until a scenario names a resolvable ID.
	LevelID               *string `gorm:"index"`
	ScenarioNearest       *string `gorm:"index"`
	ScenarioLinkBasis     string
	ScenarioLinkDistPts   *float64
	ScenarioLinkDistDelta *float64

	// OpportunityOutcome is the rung this chance closed on; always set at close
	// (E6), NULL only while the episode is still open.
	OpportunityOutcome *string `gorm:"index"`
	CloseCause         *string

	// AttainableEntry is the price actually available — for a confirmed
	// scenario the first tradeable price AFTER confirmation; for a resting arm
	// its entry, with the assumption named in AttainableEntryBasis. NULL for a
	// scenario that never armed: a touch is not a fill.
	AttainableEntry      *float64
	AttainableEntryBasis *string

	// ── W2 FADE PERMISSION ──────────────────────────────────────────────────
	// FIXED AT THE EPISODE'S OPEN and never rewritten: E3 compares episodes BY
	// this value, so a permission that drifted with the day would corrupt the
	// comparison while leaving the row looking populated.
	//
	// *bool, not bool, for the reason the whole block above is pointers: NULL
	// means NOT EVALUATED, which is a different fact from "evaluated and
	// permitted". A plain bool would render an unevaluated episode as
	// excluded=false, i.e. permitted, which is the plausible zero A24 forbids —
	// and it would do it on every historical row at once.
	FadePermitted   *bool `gorm:"index"`
	FadeExclusions  *string
	FadeMeasured    *string // JSON: exclusion -> {measured, threshold, n}
	FadeEvaluatedMs *int64

	// The terms AT THE MOMENT they became executable. Captured FORWARD, never
	// reconstructed: armed_orders is a MUTATED STATE ROW whose entry/stop/target
	// are updated in place, so a historical row's terms are unrecoverable and
	// the backfill marks them unrecomputable:mutated_in_place.
	TermsCapturedAtMs *int64
	TermsStop         *float64
	TermsTarget       *float64
	TermsR            *float64

	// ── ONE SETUP (dispatch 102, 2026-09-10) — THE VERDICT, RECORDED ────────
	// The arm seam's three-legged verdict for the scenario this episode belongs
	// to, stamped ONCE (W2's pattern: WHERE one_setup_verdicts IS NULL). NULL
	// means the seam never judged this row — a different fact from "allowed".
	// OneSetupEvaluatedMs is the clock the verdict was made with: the arm
	// seam's `now` live, the episode's OPEN for the backfill (D9), so a reader
	// can tell which. OneSetupBackfill is D9's three-state mark.
	OneSetupVerdicts    *string `gorm:"index"` // "level=<v> play=<v> permission=<v>"
	OneSetupReason      *string // "allowed" | the joined decline
	OneSetupScenario    *string `gorm:"index"`
	OneSetupEvaluatedMs *int64
	OneSetupBackfill    *string // recomputed | unrecomputable:<which> | untouched

	// ── THE FOLLOW-PLAN (round 17) — RECORDED ONLY, NEVER ARMED ─────────────
	// Beside every fade-plan the level ALSO carries a follow-plan: break → role
	// reversal → retest, with the would-be entry and its MAE/MFE. Every field
	// is NULL until its event occurs; a level never broken is a ROW with
	// FollowBreakAtMs NULL — a zero-trade outcome, not a missing row. Nothing
	// here reaches the wire (E9). FollowBackfill is D9's three-state mark;
	// FollowState is the recorder's lifecycle (open | no_break |
	// broken_no_retest | retested | complete | unrecomputable:<which>).
	FollowBreakAtMs     *int64  `gorm:"index"`
	FollowBreakDir      *string // "up" | "down"
	FollowRetestAtMs    *int64
	FollowEntryPx       *float64
	FollowEntryBasis    *string
	FollowMAE10         *float64
	FollowMFE10         *float64
	FollowNet10         *float64 // pts at the 10th closed 5m bucket after entry, net of 2-pt friction
	FollowMAE20         *float64
	FollowMFE20         *float64
	FollowNet20         *float64
	FollowRetestOutcome *string // the retest's own detector verdict, joined by level + session-day + ordinal
	FollowRoleReversed  *bool   // did the reversed level HOLD on the retest
	BiasWouldFlipTo     *string // "long" | "short" — the direction the break implies; the live bias is untouched
	PlanBiasFrozen      *string // the plan's bias at the recording, beside it, never into it
	FollowState         *string `gorm:"index"`
	FollowBackfill      *string

	CreatedAt time.Time `gorm:"index"`
}

// Validity values. Only ValidityValid rows may be used for a rate.
const (
	// ValidityValid — written by the fixed recorder: scan started at
	// max(FormedAtMs, watermark) and the episode key is unique.
	ValidityValid = "valid"
	// ValidityPreFormation — the episode opened before the level existed.
	ValidityPreFormation = "invalid:pre_formation"
	// ValidityDuplicate — a second (or eleventh) copy of an episode key.
	ValidityDuplicate = "invalid:duplicate"
	// ValidityLegacy — written by the old recorder and provably neither of
	// the above. UNVERIFIED, not blessed: it was still scanned over a whole
	// 33 h void scope with a day-scoped ordinal.
	ValidityLegacy = "legacy:unverified"
	// ValidityNoFormation — de-duplicated correctly, but the level carried NO
	// formation time, so this episode CANNOT be shown to post-date the level.
	//
	// This is not a corner case: FormedAtMs is set only by kernel/levels_zones.go
	// (DEMAND/SUPPLY/OB/FVG). Every LINE level — VWAP, RTH-L, RTH-H, PDH, PDL,
	// PDC, ONH, ONL, POC, OR-H/L, SWG — is built by lineLevel (kernel/levels.go:93-95),
	// which never sets it: 503 of the 677 live rows, 74.3%, including all 140
	// RTH-L rows that are the premise's own evidence.
	//
	// A24 forbids dressing that as a guarantee. The formation floor is applied
	// where it CAN be applied and these rows say plainly that it could not be.
	// They are excluded from every rate until line levels carry a birth time.
	ValidityNoFormation = "unverified:no_formation"
)

func (TouchOutcomeRow) TableName() string { return "touch_outcomes" }

// TouchOutcomeStore persists one row per closed episode.
type TouchOutcomeStore struct{ db *gorm.DB }

// NewTouchOutcomeStore wires the table via AutoMigrate.
func NewTouchOutcomeStore(db *gorm.DB) *TouchOutcomeStore {
	if db != nil {
		_ = db.AutoMigrate(&TouchOutcomeRow{})
	}
	return &TouchOutcomeStore{db: db}
}

// NextOrdinal reads the next touch ordinal for a level FROM THE STORE (C4), so
// numbering survives a restart. Scoped per (trader, symbol, level, session-day).
//
// WAVE A / D1c — sessionDayMs IS THE EPISODE'S OWN SESSION-DAY, not the day of
// the read. The caller used to pass the CURRENT day, so every episode that
// opened earlier scored MAX(ordinal)=0 and was written as ordinal 1 on every
// read: 471 of 677 live rows read 1, which is why no ordinal stratum is
// usable. This function was always right; it was being asked the wrong
// question. It is the ONE ordinal implementation (A24) — do not add a second.
func (s *TouchOutcomeStore) NextOrdinal(traderID, symbol string, level float64, sessionDayMs int64) int {
	if s == nil || s.db == nil {
		return 1
	}
	var maxOrd int
	if err := s.db.Model(&TouchOutcomeRow{}).
		Where("trader_id = ? AND symbol = ? AND level_price = ? AND opened_at_ms >= ?",
			traderID, symbol, level, sessionDayMs).
		Select("COALESCE(MAX(ordinal), 0)").Scan(&maxOrd).Error; err != nil {
		return 1
	}
	return maxOrd + 1
}

// SaveOutcome writes one episode. Telemetry may WARN, never panic (A10) — a
// failed write must not stop the loop.
func (s *TouchOutcomeStore) SaveOutcome(r *TouchOutcomeRow) error {
	if s == nil || s.db == nil || r == nil {
		return nil
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	return s.db.Create(r).Error
}

// AllOutcomes returns every persisted episode, oldest first. Used by the
// WAVE A migration (which must classify rows it may never delete) and by the
// recorder's own tests. Read-only.
func (s *TouchOutcomeStore) AllOutcomes() ([]TouchOutcomeRow, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var rows []TouchOutcomeRow
	if err := s.db.Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CountByValidity is the boot line's breakdown — every figure READ (A11).
// Returns a map from validity value to row count; the empty key is
// "not certified" (rows the migration has not classified).
func (s *TouchOutcomeStore) CountByValidity() map[string]int64 {
	out := map[string]int64{}
	if s == nil || s.db == nil {
		return out
	}
	type vr struct {
		Validity string
		N        int64
	}
	var rows []vr
	if err := s.db.Model(&TouchOutcomeRow{}).
		Select("COALESCE(validity,'') AS validity, COUNT(*) AS n").
		Group("validity").Scan(&rows).Error; err != nil {
		return out
	}
	for _, r := range rows {
		out[r.Validity] = r.N
	}
	return out
}

// CountOutcomes is the boot line's figure — READ, never a literal.
func (s *TouchOutcomeStore) CountOutcomes() int64 {
	if s == nil || s.db == nil {
		return 0
	}
	var n int64
	_ = s.db.Model(&TouchOutcomeRow{}).Count(&n).Error
	return n
}

// HoldRateBy returns hold/(hold+break) with n and the excluded ambiguous count
// for a filtered slice of the table. group is a column name ("level_kind",
// "session", "ordinal") or "" for the whole table.
type OutcomeRate struct {
	Group     string
	Hold      int
	Break     int
	Ambiguous int
}

// N is the rate's base — hold+break, ambiguous EXCLUDED.
func (r OutcomeRate) N() int { return r.Hold + r.Break }

// P is hold/(hold+break); 0 at n=0, and callers must check N before quoting it.
func (r OutcomeRate) P() float64 {
	if r.N() == 0 {
		return 0
	}
	return float64(r.Hold) / float64(r.N())
}

// RatesBy groups the table and returns hold/break/ambiguous per group.
func (s *TouchOutcomeStore) RatesBy(column string) ([]OutcomeRate, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	sel := "'' AS grp"
	grp := ""
	if column != "" {
		sel = column + " AS grp"
		grp = column
	}
	type row struct {
		Grp     string
		Outcome string
		N       int
	}
	var rows []row
	// D1e — ONLY CERTIFIED ROWS MAY FORM A RATE. Everything else (legacy,
	// duplicate, pre-formation, formation-unknown) is recorded and excluded.
	// This is the single chokepoint: DetectorReport and every caller of
	// RatesBy inherit it, so there is one implementation of "usable" (A24).
	q := s.db.Model(&TouchOutcomeRow{}).Select(sel+", outcome, COUNT(*) AS n").
		Where("validity = ?", ValidityValid)
	if grp != "" {
		q = q.Group(grp + ", outcome").Order(grp)
	} else {
		q = q.Group("outcome")
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	byGroup := map[string]*OutcomeRate{}
	var order []string
	for _, r := range rows {
		g, ok := byGroup[r.Grp]
		if !ok {
			g = &OutcomeRate{Group: r.Grp}
			byGroup[r.Grp] = g
			order = append(order, r.Grp)
		}
		switch r.Outcome {
		case "hold":
			g.Hold += r.N
		case "break":
			g.Break += r.N
		default:
			g.Ambiguous += r.N
		}
	}
	out := make([]OutcomeRate, 0, len(order))
	for _, g := range order {
		out = append(out, *byGroup[g])
	}
	return out, nil
}

// ── D6 — THE READ-ONLY REPORT ────────────────────────────────────────────────
//
// p(hold) per kind / session / ordinal, each with n, a Wilson interval and the
// ambiguous share. Below the floor every line says DESCRIPTIVE ONLY: at n<200 a
// rate is a description of a sample, not an estimate of a property.
const TouchRateFloor = 200

// DetectorReport renders the D6 table. Empty table → says so, never a zero rate.
func (s *TouchOutcomeStore) DetectorReport() string {
	if s == nil || s.db == nil {
		return "detector report: store unavailable"
	}
	var b []byte
	add := func(f string, a ...any) { b = append(b, []byte(sprintf(f, a...))...) }
	total := s.CountOutcomes()
	add("touch_outcomes: %d row(s)\n", total)
	if total == 0 {
		add("  (empty — the detector has recorded no episodes yet; every figure below would be a plausible zero)\n")
		return string(b)
	}
	for _, dim := range []struct{ label, col string }{
		{"ALL", ""}, {"by kind", "level_kind"}, {"by session", "session"}, {"by ordinal", "ordinal"},
	} {
		rates, err := s.RatesBy(dim.col)
		if err != nil {
			add("  %s: unavailable (%v)\n", dim.label, err)
			continue
		}
		add("  %s:\n", dim.label)
		for _, r := range rates {
			g := r.Group
			if g == "" {
				g = "(all)"
			}
			if r.N() == 0 {
				add("    %-14s n=0 — no resolved episodes (ambiguous=%d)\n", g, r.Ambiguous)
				continue
			}
			lo, hi := wilson(r.P(), r.N())
			note := ""
			if r.N() < TouchRateFloor {
				note = sprintf("  — n<%d, DESCRIPTIVE ONLY", TouchRateFloor)
			}
			ambShare := float64(r.Ambiguous) / float64(r.N()+r.Ambiguous) * 100
			add("    %-14s p(hold)=%.3f [%.3f, %.3f] n=%d · ambiguous=%d (%.1f%%)%s\n",
				g, r.P(), lo, hi, r.N(), r.Ambiguous, ambShare, note)
		}
	}
	return string(b)
}

// TouchOutcomesBootLine reports the recorded row count — READ, never a literal.
func (s *TouchOutcomeStore) TouchOutcomesBootLine() int64 { return s.CountOutcomes() }

func sprintf(f string, a ...any) string { return fmt.Sprintf(f, a...) }

// wilson is the 95% score interval — the same formula the detector uses, kept
// here so the report cannot quote a bare proportion.
func wilson(p float64, n int) (lo, hi float64) {
	if n <= 0 {
		return 0, 0
	}
	const z = 1.959963984540054
	nf := float64(n)
	den := 1 + z*z/nf
	c := (p + z*z/(2*nf)) / den
	h := z * math.Sqrt(p*(1-p)/nf+z*z/(4*nf*nf)) / den
	return c - h, c + h
}

// LastOpenedAtMs is the watermark that makes the per-read hook idempotent: the
// newest episode already recorded for this level at or after sinceMs. 0 = none.
//
// WAVE A / D1a — sinceMs IS THE SCAN WINDOW, NOT THE SESSION DAY. The caller
// used to pass the CURRENT session-day start, so an episode that opened before
// 17:00 CT today never matched this filter, the watermark read 0, and the whole
// set was written again on every read. That is how 677 rows became 423
// episodes and RTH-L became 140 rows of 14. The watermark must cover exactly
// the window the caller is about to scan.
func (s *TouchOutcomeStore) LastOpenedAtMs(traderID, symbol string, level float64, sinceMs int64) int64 {
	if s == nil || s.db == nil {
		return 0
	}
	var last int64
	if err := s.db.Model(&TouchOutcomeRow{}).
		Where("trader_id = ? AND symbol = ? AND level_price = ? AND opened_at_ms >= ?",
			traderID, symbol, level, sinceMs).
		Select("COALESCE(MAX(opened_at_ms), 0)").Scan(&last).Error; err != nil {
		return 0
	}
	return last
}
