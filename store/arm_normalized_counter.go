package store

import (
	"fmt"
	"strconv"
	"strings"
)

// ArmsNormalizedKey — CLASS 39 (owner ruling 2026-09-01) recorded counter:
// every normalize-don't-reject event (legs dropped from a non-sweep arm) bumps
// it in system_config, so the count survives restarts. The class-35 lesson:
// counters RECORD events; log-only tallies evaporate at the next boot.
const ArmsNormalizedKey = "arms_normalized_class39"

// IncArmsNormalized bumps the counter atomically (one UPSERT, no
// read-modify-write race) and returns the new value.
func IncArmsNormalized(st *Store) (int, error) {
	if st == nil || st.gdb == nil {
		return 0, fmt.Errorf("store required")
	}
	err := st.gdb.Exec(`INSERT INTO system_config (key, value) VALUES (?, '1')
		ON CONFLICT(key) DO UPDATE SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT)`, ArmsNormalizedKey).Error
	if err != nil {
		return 0, err
	}
	return ArmsNormalizedCount(st), nil
}

// ArmsNormalizedCount reads the recorded counter (0 when never bumped).
func ArmsNormalizedCount(st *Store) int {
	if st == nil {
		return 0
	}
	v, err := st.GetSystemConfig(ArmsNormalizedKey)
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(v))
	return n
}
