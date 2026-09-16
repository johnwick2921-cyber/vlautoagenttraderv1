package store

import (
	"fmt"
	"strconv"
	"strings"
)

// ShadowABKey (ROOT-FIX part B, 2026-09-02) — the RECORDED count of shadow
// fast-mode A/B calls made. The pre-registered criterion is judged at n≥10, so
// the sample size must survive restarts: a log-only tally would reset the
// experiment at every boot (the class-35 lesson).
const ShadowABKey = "shadow_ab_calls_rootfix"

// IncShadowAB bumps the sample counter atomically and returns the new value.
func IncShadowAB(st *Store, n int) (int, error) {
	if st == nil || st.gdb == nil {
		return 0, fmt.Errorf("store required")
	}
	if n <= 0 {
		return ShadowABCount(st)
	}
	if err := st.gdb.Exec(
		`INSERT INTO system_config (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = CAST(CAST(value AS INTEGER) + ? AS TEXT)`,
		ShadowABKey, strconv.Itoa(n), n).Error; err != nil {
		return 0, err
	}
	return ShadowABCount(st)
}

// ShadowABCount reads the sample counter (0 when unset).
func ShadowABCount(st *Store) (int, error) {
	if st == nil || st.gdb == nil {
		return 0, fmt.Errorf("store required")
	}
	var v string
	if err := st.gdb.Raw(`SELECT value FROM system_config WHERE key = ?`, ShadowABKey).Scan(&v).Error; err != nil {
		return 0, err
	}
	n, _ := strconv.Atoi(strings.TrimSpace(v))
	return n, nil
}
