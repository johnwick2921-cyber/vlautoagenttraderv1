package store

import (
	"fmt"
	"strconv"
	"strings"
)

// ── REPAIR-PARSE (2026-09-02) — RECORDED REPAIR OUTCOMES ─────────────────────
// The repair path is the DEFAULT retry and 18 of 28 of its attempts were
// rejected in the 2026-09-01 journals. That rate was only recoverable by
// parsing logs after the fact; a restart erased the tally. These counters
// record it (the class-35 lesson) so the rate is answerable at any moment.

// RepairOutcomeKey builds the system_config key for one outcome class.
func RepairOutcomeKey(outcome string) string {
	return "repair_outcome_" + strings.TrimSpace(outcome)
}

// IncRepairOutcome bumps one outcome's counter atomically and returns the new
// value. An empty outcome is refused rather than silently counted as "".
func IncRepairOutcome(st *Store, outcome string) (int, error) {
	if st == nil || st.gdb == nil {
		return 0, fmt.Errorf("store required")
	}
	if strings.TrimSpace(outcome) == "" {
		return 0, fmt.Errorf("outcome required")
	}
	key := RepairOutcomeKey(outcome)
	if err := st.gdb.Exec(
		`INSERT INTO system_config (key, value) VALUES (?, '1')
		 ON CONFLICT(key) DO UPDATE SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT)`,
		key).Error; err != nil {
		return 0, err
	}
	return RepairOutcomeCount(st, outcome)
}

// RepairOutcomeCount reads one outcome's counter (0 when unset).
func RepairOutcomeCount(st *Store, outcome string) (int, error) {
	if st == nil || st.gdb == nil {
		return 0, fmt.Errorf("store required")
	}
	var v string
	if err := st.gdb.Raw(`SELECT value FROM system_config WHERE key = ?`, RepairOutcomeKey(outcome)).Scan(&v).Error; err != nil {
		return 0, err
	}
	n, _ := strconv.Atoi(strings.TrimSpace(v))
	return n, nil
}

// RepairOutcomeSummary renders "ok=3 content=7 packaging=1 fragment=0" for the
// boot line and the report. Order is fixed so the line is diffable.
func RepairOutcomeSummary(st *Store) string {
	parts := make([]string, 0, 4)
	for _, o := range []string{"ok", "content", "packaging", "fragment"} {
		n, _ := RepairOutcomeCount(st, o)
		parts = append(parts, fmt.Sprintf("%s=%d", o, n))
	}
	return strings.Join(parts, " ")
}
