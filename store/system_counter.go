package store

import (
	"fmt"
	"strconv"
	"strings"
)

// IncSystemCounter is the ONE generic recorded counter (class-35 lesson: a
// log-only tally evaporates at the next boot, and this repo has grown a
// separate Inc* function per wave). Atomic UPSERT, no read-modify-write race.
func IncSystemCounter(st *Store, key string) (int, error) {
	if st == nil || st.gdb == nil {
		return 0, fmt.Errorf("store required")
	}
	if strings.TrimSpace(key) == "" {
		return 0, fmt.Errorf("counter key required")
	}
	if err := st.gdb.Exec(
		`INSERT INTO system_config (key, value) VALUES (?, '1')
		 ON CONFLICT(key) DO UPDATE SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT)`,
		key).Error; err != nil {
		return 0, err
	}
	return SystemCounter(st, key)
}

// SystemCounter reads one counter (0 when unset).
func SystemCounter(st *Store, key string) (int, error) {
	if st == nil || st.gdb == nil {
		return 0, fmt.Errorf("store required")
	}
	var v string
	if err := st.gdb.Raw(`SELECT value FROM system_config WHERE key = ?`, key).Scan(&v).Error; err != nil {
		return 0, err
	}
	n, _ := strconv.Atoi(strings.TrimSpace(v))
	return n, nil
}
