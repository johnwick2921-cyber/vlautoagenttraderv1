package store

import "fmt"

// DeathRereadKey is the per-session-day counter key for W-DEATH-REREAD
// (2026-09-18): one key per (trader, trade_date, session), recorded ONLY when
// a death re-read's fresh version LANDED in the store (the same "no plan row,
// no budget consumed / no counter" rule as the class-35 spend).
func DeathRereadKey(traderID, tradeDate, session string) string {
	return "death_reread:" + traderID + ":" + tradeDate + ":" + session
}

// IncDeathReread records ONE landed death re-read. Returns the new count.
func IncDeathReread(st *Store, traderID, tradeDate, session string) (int, error) {
	if st == nil || st.gdb == nil {
		return 0, fmt.Errorf("store required")
	}
	key := DeathRereadKey(traderID, tradeDate, session)
	if err := st.gdb.Exec(`INSERT INTO system_config (key, value) VALUES (?, '1')
		ON CONFLICT(key) DO UPDATE SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT)`, key).Error; err != nil {
		return 0, err
	}
	return CountFromSystemConfig(st, key), nil
}

// DeathRereadCount reads the recorded count for a session-day.
func DeathRereadCount(st *Store, traderID, tradeDate, session string) int {
	return CountFromSystemConfig(st, DeathRereadKey(traderID, tradeDate, session))
}
