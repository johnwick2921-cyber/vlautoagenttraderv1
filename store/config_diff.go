package store

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ── REPAIR-PARSE E5 (2026-09-02) — CONFIG DIFF ON SAVE ───────────────────────
// The 08:13 CT save on 2026-09-01 changed min_risk_reward_ratio 3 → 2 and
// triggered an in-process trader reload during NY. Nothing said so: the audit
// had to INFER what moved. Third anonymous knob drift of the week (class 8 /
// A11 family — a RESOLVED value changed and no surface announced it).
// Every save now names what changed, in resolved values, and persists it.

// ConfigChange is one knob that moved on one save.
type ConfigChange struct {
	ID       uint   `gorm:"primaryKey"`
	TraderID string `gorm:"index"`
	Strategy string `gorm:"index"`
	Knob     string `gorm:"index"` // dotted path, e.g. risk_control.min_risk_reward_ratio
	OldValue string
	NewValue string
	Source   string    `gorm:"index"` // studio_save | api | migration
	At       time.Time `gorm:"index"`
}

func (ConfigChange) TableName() string { return "config_changes" }

// ConfigChangeStore persists the diff of every save.
type ConfigChangeStore struct{ db *gorm.DB }

// NewConfigChangeStore wires the table.
func NewConfigChangeStore(db *gorm.DB) *ConfigChangeStore {
	if db != nil {
		_ = db.AutoMigrate(&ConfigChange{})
	}
	return &ConfigChangeStore{db: db}
}

// configChangeCap bounds the table (a knob-churn day must not unbound disk).
const configChangeCap = 5000

// Save persists a batch of changes; an empty batch writes nothing.
func (s *ConfigChangeStore) Save(rows []ConfigChange) error {
	if s == nil || s.db == nil || len(rows) == 0 {
		return nil
	}
	if err := s.db.Create(&rows).Error; err != nil {
		return err
	}
	var count int64
	if err := s.db.Model(&ConfigChange{}).Count(&count).Error; err == nil && count > configChangeCap {
		_ = s.db.Exec("DELETE FROM config_changes WHERE id NOT IN (SELECT id FROM config_changes ORDER BY id DESC LIMIT ?)", configChangeCap).Error
	}
	return nil
}

// Recent returns the newest changes, newest first.
func (s *ConfigChangeStore) Recent(limit int) ([]ConfigChange, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("store required")
	}
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	var out []ConfigChange
	err := s.db.Order("id DESC").Limit(limit).Find(&out).Error
	return out, err
}

// DiffStrategyConfig returns every RESOLVED value that differs between two
// saved configs, as dotted paths. It compares the marshalled forms, so what it
// reports is exactly what the trader will reload — not a file default (A11).
func DiffStrategyConfig(before, after StrategyConfig) []ConfigChange {
	flatB := flattenConfig(before)
	flatA := flattenConfig(after)
	keys := map[string]bool{}
	for k := range flatB {
		keys[k] = true
	}
	for k := range flatA {
		keys[k] = true
	}
	names := make([]string, 0, len(keys))
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names)
	var out []ConfigChange
	for _, k := range names {
		o, n := flatB[k], flatA[k]
		if o == n {
			continue
		}
		out = append(out, ConfigChange{Knob: k, OldValue: o, NewValue: n})
	}
	return out
}

// flattenConfig renders a config as dotted path → scalar string. Slices and
// maps are rendered whole at their own path: a reordered list is one change,
// not N.
func flattenConfig(c StrategyConfig) map[string]string {
	b, err := json.Marshal(c)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	out := map[string]string{}
	flattenInto("", m, out)
	return out
}

func flattenInto(prefix string, v any, out map[string]string) {
	switch t := v.(type) {
	case map[string]any:
		for k, sub := range t {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			flattenInto(key, sub, out)
		}
	case nil:
		out[prefix] = "null"
	case []any:
		b, _ := json.Marshal(t)
		out[prefix] = string(b)
	default:
		out[prefix] = fmt.Sprintf("%v", t)
	}
}

// ConfigDiffLine renders ONE change for the journal (pure — fixture-pinned).
func ConfigDiffLine(source string, ch ConfigChange) string {
	return fmt.Sprintf("⚙ config diff (%s): %s %s → %s", source, ch.Knob, ch.OldValue, ch.NewValue)
}

// ConfigDiffSummaryLine renders the header for a save that changed n knobs.
func ConfigDiffSummaryLine(source, strategy string, n int) string {
	if n == 0 {
		return fmt.Sprintf("⚙ config diff (%s): strategy %s saved with NO resolved-value change", source, strategy)
	}
	return fmt.Sprintf("⚙ config diff (%s): strategy %s — %d resolved knob(s) changed, trader reload follows", source, strategy, n)
}

// NormalizeSource keeps the source token short and stable.
func NormalizeSource(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "api"
	}
	return s
}
