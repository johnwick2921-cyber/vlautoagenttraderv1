package store

// ── W1 SETTINGS TRUTH (2026-09-23) — THE CONVERSION REPORT ──────────────────
//
// consecutive_loss_halt and day_plan.replan_cap became presence-aware (*int):
// absent inherits, an explicit 0 is 0. Before W1 both were ints whose 0 meant
// "unset" to the runtime (breaker → env/8, strategy replan → 2) while the UI
// and the struct comment called the breaker's 0 OFF.
//
// So the conversion changes an effective value for EXACTLY ONE stored shape: an
// explicit 0 (breaker, or the strategy-level replan cap). No pre-W1 writer
// could store one — every writer re-marshalled the struct and omitempty dropped
// the 0 — so the expected population is empty [B]. The report proves that at
// every boot instead of assuming it, one line per stored strategy.
//
// FAIL-CLOSED, NO DB WRITE AT BOOT. A row whose effective value WOULD change
// is not rewritten: the trader(s) bound to it are REFUSED at load with a named
// reason until the owner re-saves the knob through a W1 writer. A breaker that
// was inherited (ON) is never silently turned OFF by the conversion.
//
// WHICH ZEROS ARE THE OWNER'S. After W1 the Studio's OFF writes an explicit 0 —
// the new meaning — so a stored 0 alone cannot say whether it predates W1.
// The Studio's two writers (POST/PUT /api/strategies — the surface where the
// owner SEES the knob's state) therefore record which knobs each save holds as
// an explicit 0 (RecordExplicitZeros, one system_config key per strategy,
// written at SAVE time by the owner's action, never at boot). A 0 the record
// covers is the owner's and loads; a 0 it does not cover is refused. Raw copies
// (the acceptance-rule migration, an agent update without a config) and the
// agent's config writes never confirm — the agent does not show the owner the
// breaker — so an agent-written 0 loads only after a Studio save. Duplicate
// carries the source's record to the copy, since it copies the same bytes.

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// The knob keys the report and the confirmation record use.
const (
	KnobBreaker        = "risk_control.consecutive_loss_halt"
	KnobReplanStrategy = "day_plan.replan_cap"
)

// settingsTruthSessions — the sessions whose effective replan cap is compared;
// "" is the strategy level (outside every session window).
var settingsTruthSessions = []string{"", "NY", "ASIA", "LONDON"}

// ── THE PRE-W1 RULES, FROZEN ────────────────────────────────────────────────
// Used ONLY by the report to say what the previous binary enforced. Nothing on
// a live path may call these: they are the defect W1 corrects.

// legacyBreakerHaltN is breakerHaltN as it stood before W1: a knob > 0 wins,
// anything else (absent, 0) fell to env BREAKER_HALT_N, else 8.
func legacyBreakerHaltN(stored *int, getenv func(string) string) int {
	if stored != nil && *stored > 0 {
		return *stored
	}
	if n, st := BreakerHaltEnvState(getenv); st == EnvValid {
		return n
	}
	return BreakerHaltDefault
}

// legacyReplanCapFor is ReplanCapFor as it stood before W1: the strategy
// level counted only when > 0; a session override counted at ≥ 0.
func legacyReplanCapFor(c *DayPlanConfig, session string) int {
	n := ReplanCapDefault
	if c != nil && c.ReplanCap != nil && *c.ReplanCap > 0 {
		n = *c.ReplanCap
	}
	if ov := c.SessionOverride(session); ov != nil && ov.ReplanCap != nil && *ov.ReplanCap >= 0 {
		n = *ov.ReplanCap
	}
	return n
}

// ── THE REPORT ──────────────────────────────────────────────────────────────

// SettingsTruthInput is one stored strategy row as the report reads it.
type SettingsTruthInput struct {
	ID     string
	Config string // the RAW stored JSON — presence is read from it
	Bound  int    // traders bound to this strategy
	// Confirmed — knob keys a W1 save recorded as an explicit 0.
	Confirmed map[string]bool
}

// SettingsTruthKnob is one knob of one row: what is stored, what the previous
// binary enforced, what this one enforces.
type SettingsTruthKnob struct {
	Knob    string
	Stored  string // absent | null | <n> | unreadable
	Where   string // where the stored value lives in the JSON
	Before  string
	After   string
	Changed bool
}

// SettingsTruthRow is one stored strategy.
type SettingsTruthRow struct {
	ID      string
	Bound   int
	Error   string // the config does not parse — nothing resolved
	Breaker SettingsTruthKnob
	Replan  SettingsTruthKnob
	Changed bool
	// Unconfirmed — changed knobs no W1 save has confirmed. Non-empty ⇒ Refuse.
	Unconfirmed []string
	Refuse      string
}

// SettingsTruthResult is the whole report.
type SettingsTruthResult struct {
	EnvState    string // BREAKER_HALT_N: unset | <n> | invalid
	Rows        []SettingsTruthRow
	ChangedRows int
	RefusedRows int
}

// SettingsTruthReport is PURE: rows in, report out. getenv is the environment
// the report evaluates (os.Getenv at boot; a fixed map in tests).
func SettingsTruthReport(rows []SettingsTruthInput, getenv func(string) string) SettingsTruthResult {
	res := SettingsTruthResult{}
	switch n, st := BreakerHaltEnvState(getenv); st {
	case EnvValid:
		res.EnvState = strconv.Itoa(n)
	default:
		res.EnvState = st
	}
	sorted := append([]SettingsTruthInput(nil), rows...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	for _, in := range sorted {
		row := settingsTruthRow(in, getenv)
		if row.Changed {
			res.ChangedRows++
		}
		if row.Refuse != "" {
			res.RefusedRows++
		}
		res.Rows = append(res.Rows, row)
	}
	return res
}

func settingsTruthRow(in SettingsTruthInput, getenv func(string) string) SettingsTruthRow {
	row := SettingsTruthRow{ID: in.ID, Bound: in.Bound}
	var raw map[string]any
	if err := json.Unmarshal([]byte(in.Config), &raw); err != nil {
		row.Error = "config is not a JSON object"
		return row
	}
	// The SAME parse the runtime uses (StrategyConfig.UnmarshalJSON, nested
	// ai_config or the legacy flat keys) — the report resolves what the trader
	// will load, not a second reading of the JSON.
	var cfg StrategyConfig
	if err := json.Unmarshal([]byte(in.Config), &cfg); err != nil {
		row.Error = "config does not parse (" + err.Error() + ")"
		return row
	}

	// Breaker.
	shape, where := breakerRawShape(raw)
	before := legacyBreakerHaltN(cfg.RiskControl.ConsecutiveLossHalt, getenv)
	after, src := ResolveBreakerHaltEnv(&cfg, getenv)
	row.Breaker = SettingsTruthKnob{
		Knob: KnobBreaker, Stored: shape, Where: where,
		Before: breakerWord(before), After: breakerWord(after) + OriginLetter(src),
		Changed: before != after,
	}

	// Replan cap — the strategy level and each session.
	var dpRaw map[string]any
	if dp, ok := raw["day_plan"].(map[string]any); ok {
		dpRaw = dp
	}
	rp := SettingsTruthKnob{Knob: KnobReplanStrategy, Stored: storedIntShape(dpRaw, "replan_cap"), Where: "day_plan"}
	var b, a []string
	for _, s := range settingsTruthSessions {
		old := legacyReplanCapFor(cfg.DayPlan, s)
		n, src := ResolveReplanCap(cfg.DayPlan, s)
		b = append(b, strconv.Itoa(old))
		a = append(a, strconv.Itoa(n)+OriginLetter(src))
		if old != n {
			rp.Changed = true
		}
	}
	rp.Before, rp.After = strings.Join(b, "/"), strings.Join(a, "/")
	row.Replan = rp

	row.Changed = row.Breaker.Changed || row.Replan.Changed
	for _, k := range []SettingsTruthKnob{row.Breaker, row.Replan} {
		if k.Changed && !in.Confirmed[k.Knob] {
			row.Unconfirmed = append(row.Unconfirmed, k.Knob)
		}
	}
	if len(row.Unconfirmed) > 0 {
		row.Refuse = settingsTruthRefusal(in.ID, row)
	}
	return row
}

// settingsTruthRefusal names what the stored 0 meant then and means now.
func settingsTruthRefusal(id string, row SettingsTruthRow) string {
	var parts []string
	for _, k := range row.Unconfirmed {
		switch k {
		case KnobBreaker:
			parts = append(parts, fmt.Sprintf(
				"consecutive_loss_halt=0 from before W1 — it meant 'inherit' then (breaker %s) and 'OFF' now; re-save the strategy in the Studio (its breaker shows OFF there) to confirm OFF, or turn the breaker ON to inherit",
				row.Breaker.Before))
		case KnobReplanStrategy:
			parts = append(parts, fmt.Sprintf(
				"day_plan.replan_cap=0 from before W1 — it meant 'default 2' then (%s) and '0 re-plans' now; re-save the strategy in the Studio to confirm 0, or clear the cap to inherit 2",
				row.Replan.Before))
		}
	}
	return fmt.Sprintf("refused: strategy %s stores %s", id, strings.Join(parts, "; and "))
}

// breakerWord renders a breaker N: off for 0.
func breakerWord(n int) string {
	if n <= 0 {
		return "off"
	}
	return strconv.Itoa(n)
}

// breakerRawShape reads the stored breaker the way UnmarshalJSON does: nested
// under ai_config when ai_config is present and non-null, else the legacy flat
// risk_control key.
func breakerRawShape(raw map[string]any) (shape, where string) {
	if ac, present := raw["ai_config"]; present && ac != nil {
		acm, _ := ac.(map[string]any)
		rc, _ := acm["risk_control"].(map[string]any)
		shape, where = storedIntShape(rc, "consecutive_loss_halt"), "ai_config.risk_control"
		if flat, ok := raw["risk_control"].(map[string]any); ok {
			if _, has := flat["consecutive_loss_halt"]; has {
				where += " (a flat risk_control key is also stored — ignored, ai_config wins)"
			}
		}
		return shape, where
	}
	rc, _ := raw["risk_control"].(map[string]any)
	return storedIntShape(rc, "consecutive_loss_halt"), "risk_control (legacy flat)"
}

// storedIntShape says what a JSON object holds at key: absent, null, an
// integer, or unreadable.
func storedIntShape(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return "absent"
	}
	if v == nil {
		return "null"
	}
	if f, isNum := v.(float64); isNum && f == math.Trunc(f) && !math.IsInf(f, 0) {
		return strconv.FormatInt(int64(f), 10)
	}
	return "unreadable"
}

// Lines renders the report: one line per strategy, then the summary.
func (r SettingsTruthResult) Lines() []string {
	out := make([]string, 0, len(r.Rows)+1)
	for _, row := range r.Rows {
		out = append(out, row.Line())
	}
	out = append(out, fmt.Sprintf(
		"🩺 settings truth (W1): %d strateg%s · BREAKER_HALT_N=%s · rows changed=%d · refused=%d",
		len(r.Rows), map[bool]string{true: "y", false: "ies"}[len(r.Rows) == 1], r.EnvState, r.ChangedRows, r.RefusedRows))
	return out
}

// Line renders one strategy row.
func (row SettingsTruthRow) Line() string {
	id := row.ID
	if id == "" {
		id = "(empty id)" // a real row: the research copy holds one
	}
	if row.Error != "" {
		return fmt.Sprintf("🩺 settings truth [%s] bound=%d · n/a (%s)", id, row.Bound, row.Error)
	}
	breakerStored := row.Breaker.Stored
	if row.Breaker.Where != "ai_config.risk_control" {
		breakerStored += " [" + row.Breaker.Where + "]"
	}
	verdict := "UNCHANGED"
	switch {
	case row.Refuse != "":
		verdict = "CHANGED — " + row.Refuse
	case row.Changed:
		verdict = "CHANGED — confirmed by a W1 save"
	}
	return fmt.Sprintf(
		"🩺 settings truth [%s] bound=%d · consecutive_loss_halt stored=%s before=%s after=%s · replan_cap stored=%s before=%s after=%s (strategy/NY/ASIA/LONDON) · %s",
		id, row.Bound,
		breakerStored, row.Breaker.Before, row.Breaker.After,
		row.Replan.Stored, row.Replan.Before, row.Replan.After,
		verdict)
}

// ── THE CONFIRMATION RECORD ─────────────────────────────────────────────────

// explicitZeroKey is the system_config key holding one strategy's record.
func explicitZeroKey(strategyID string) string { return "settings_truth_zero:" + strategyID }

// ExplicitZeroKnobs lists the W1 knobs cfg holds as an explicit 0, sorted.
func ExplicitZeroKnobs(cfg *StrategyConfig) []string {
	var out []string
	if cfg == nil {
		return out
	}
	if v := cfg.RiskControl.ConsecutiveLossHalt; v != nil && *v == 0 {
		out = append(out, KnobBreaker)
	}
	if cfg.DayPlan != nil && cfg.DayPlan.ReplanCap != nil && *cfg.DayPlan.ReplanCap == 0 {
		out = append(out, KnobReplanStrategy)
	}
	sort.Strings(out)
	return out
}

// RecordExplicitZeros is called by every W1 writer right after it persists a
// strategy: it records exactly the knobs this save holds as an explicit 0, so
// the conversion check can tell the owner's 0 from one that predates W1. The
// record is REPLACED on every save (a knob turned back to inherit drops out).
func (s *StrategyStore) RecordExplicitZeros(strategyID string, cfg *StrategyConfig) error {
	if s == nil || s.db == nil || strategyID == "" {
		return nil
	}
	return s.db.Exec(`INSERT INTO system_config (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		explicitZeroKey(strategyID), strings.Join(ExplicitZeroKnobs(cfg), ",")).Error
}

// ExplicitZerosConfirmed reads one strategy's record (empty when none).
func (s *StrategyStore) ExplicitZerosConfirmed(strategyID string) map[string]bool {
	out := map[string]bool{}
	if s == nil || s.db == nil || strategyID == "" {
		return out
	}
	var vals []string
	if err := s.db.Raw(`SELECT value FROM system_config WHERE key = ?`, explicitZeroKey(strategyID)).
		Scan(&vals).Error; err != nil || len(vals) == 0 {
		return out
	}
	for _, k := range strings.Split(vals[0], ",") {
		if k = strings.TrimSpace(k); k != "" {
			out[k] = true
		}
	}
	return out
}

// copyExplicitZeros carries a source strategy's record to a duplicate.
func (s *StrategyStore) copyExplicitZeros(fromID, toID string) error {
	conf := s.ExplicitZerosConfirmed(fromID)
	if len(conf) == 0 {
		return nil
	}
	keys := make([]string, 0, len(conf))
	for k := range conf {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return s.db.Exec(`INSERT INTO system_config (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		explicitZeroKey(toID), strings.Join(keys, ",")).Error
}

// ── THE BOOT AND LOAD SEATS ─────────────────────────────────────────────────

// SettingsTruthBootReport reads every stored strategy (READ-ONLY) and returns
// the report the boot block prints — before LoadTradersFromStore, so the
// refusal each trader load applies has already been announced.
func (s *StrategyStore) SettingsTruthBootReport(getenv func(string) string) (SettingsTruthResult, error) {
	if s == nil || s.db == nil {
		return SettingsTruthResult{}, fmt.Errorf("store required")
	}
	var all []Strategy
	if err := s.db.Order("id").Find(&all).Error; err != nil {
		return SettingsTruthResult{}, err
	}
	type boundRow struct {
		StrategyID string
		N          int
	}
	var bound []boundRow
	_ = s.db.Raw(`SELECT strategy_id, COUNT(*) AS n FROM traders WHERE strategy_id IS NOT NULL AND strategy_id <> '' GROUP BY strategy_id`).
		Scan(&bound).Error
	boundBy := map[string]int{}
	for _, b := range bound {
		boundBy[b.StrategyID] = b.N
	}
	in := make([]SettingsTruthInput, 0, len(all))
	for _, st := range all {
		in = append(in, SettingsTruthInput{
			ID: st.ID, Config: st.Config, Bound: boundBy[st.ID],
			Confirmed: s.ExplicitZerosConfirmed(st.ID),
		})
	}
	return SettingsTruthReport(in, getenv), nil
}

// SettingsTruthRefusal is the load-time check for ONE strategy: "" when its
// trader may load, else the named reason it may not.
func (s *StrategyStore) SettingsTruthRefusal(st *Strategy, getenv func(string) string) string {
	if st == nil {
		return ""
	}
	row := settingsTruthRow(SettingsTruthInput{
		ID: st.ID, Config: st.Config, Confirmed: s.ExplicitZerosConfirmed(st.ID),
	}, getenv)
	return row.Refuse
}
