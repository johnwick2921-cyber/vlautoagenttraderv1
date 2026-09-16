package store

import (
	"fmt"
	"strconv"
	"strings"
)

// ── 0B (owner rulings 2026-09-02) — RECORDED COUNTERS ────────────────────────
//
// Class 35's law: counters RECORD events; they do not infer them, and a
// log-only tally evaporates at the next boot. Both 0B rulings need a number
// that survives restarts:
//
//   1. The 3.0×ATR dead-zone bound is a PROVISIONAL [I] default, reviewed when
//      `stop_unanchored` reaches n≥30 occurrences. Without a durable count
//      there is no way to know when that review is due.
//   2. Wider stops mean more arm refusals at ARM_MIN_RR 2.0 — the accepted
//      COST side of the stop-floor decision. Recorded per session-day and per
//      refusal class so it can be quoted against the benefit later.

// StopUnanchoredKey counts arms composed with NO seated level on the risk side
// within the dead-zone bound (the ATR floor governed instead).
const StopUnanchoredKey = "arm_stop_unanchored_0b"

// StopUnanchoredReviewN is the owner's review trigger for the provisional
// ARM_STOP_ANCHOR_MAX_ATR default.
const StopUnanchoredReviewN = 30

// IncStopUnanchored bumps the dead-zone counter atomically and returns the new
// value (one UPSERT — no read-modify-write race).
func IncStopUnanchored(st *Store) (int, error) {
	if st == nil || st.gdb == nil {
		return 0, fmt.Errorf("store required")
	}
	if err := st.gdb.Exec(`INSERT INTO system_config (key, value) VALUES (?, '1')
		ON CONFLICT(key) DO UPDATE SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT)`, StopUnanchoredKey).Error; err != nil {
		return 0, err
	}
	return CountFromSystemConfig(st, StopUnanchoredKey), nil
}

// ArmRefusalKey is the per-session-day, per-class arm-refusal counter key.
// One key per (trader, trade_date, session, class) so a session's cost can be
// read back without scanning logs — and so the R:R share is separable from
// every other refusal class.
func ArmRefusalKey(traderID, tradeDate, session, class string) string {
	return "arm_refusals_0b:" + traderID + ":" + tradeDate + ":" + session + ":" + class
}

// IncArmRefusal records ONE distinct refused arm-spec (the caller dedups by
// plan:version:scenario:leg, so this counts arms refused, never cycles spent
// re-refusing the same arm). Returns the new per-session count for that class.
func IncArmRefusal(st *Store, traderID, tradeDate, session, class string) (int, error) {
	if st == nil || st.gdb == nil {
		return 0, fmt.Errorf("store required")
	}
	key := ArmRefusalKey(traderID, tradeDate, session, class)
	if err := st.gdb.Exec(`INSERT INTO system_config (key, value) VALUES (?, '1')
		ON CONFLICT(key) DO UPDATE SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT)`, key).Error; err != nil {
		return 0, err
	}
	return CountFromSystemConfig(st, key), nil
}

// ArmRefusalCount reads one class's recorded count for a session-day.
func ArmRefusalCount(st *Store, traderID, tradeDate, session, class string) int {
	return CountFromSystemConfig(st, ArmRefusalKey(traderID, tradeDate, session, class))
}

// CountFromSystemConfig reads an integer counter; 0 when absent or malformed.
// A24: a malformed row reads as 0 and is never a fabricated figure — the
// caller's log line carries the count with its key so the row can be checked.
func CountFromSystemConfig(st *Store, key string) int {
	if st == nil {
		return 0
	}
	v, err := st.GetSystemConfig(key)
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(v))
	return n
}
