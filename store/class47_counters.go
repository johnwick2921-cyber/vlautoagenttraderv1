package store

import "fmt"

// ── CLASS 47 (2026-09-02) — WARN-FIRST WAKE COUNTERS ────────────────────────
//
// F1/F2 are WARN-first: the wake still runs, and we RECORD what a suppression
// would have skipped. A week of these counts is what the owner rules on — not a
// week of impressions. Class-35 law: a log-only tally evaporates at the next
// boot, so these live in system_config.

// Counter keys. Scoped per (trader, session-day, session) so a ruling can be
// made per session rather than on one blended number.
const (
	WakeWouldSkipCutoffKind   = "cutoff"
	WakeWouldSkipCooldownKind = "cooldown"
	WakeStreamDeferKind       = "stream_defer"
	ArmSupersededKey          = "arm_superseded_unplaced_class47"
)

// WakeCounterKey builds the per-session counter key.
func WakeCounterKey(traderID, tradeDate, session, kind string) string {
	return "wake_" + kind + "_class47:" + traderID + ":" + tradeDate + ":" + session
}

// IncWakeCounter records ONE event and returns the new count.
func IncWakeCounter(st *Store, traderID, tradeDate, session, kind string) (int, error) {
	if st == nil || st.gdb == nil {
		return 0, fmt.Errorf("store required")
	}
	key := WakeCounterKey(traderID, tradeDate, session, kind)
	if err := st.gdb.Exec(`INSERT INTO system_config (key, value) VALUES (?, '1')
		ON CONFLICT(key) DO UPDATE SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT)`, key).Error; err != nil {
		return 0, err
	}
	return CountFromSystemConfig(st, key), nil
}

// WakeCounterCount reads one counter (0 when absent or malformed — never a
// fabricated figure).
func WakeCounterCount(st *Store, traderID, tradeDate, session, kind string) int {
	return CountFromSystemConfig(st, WakeCounterKey(traderID, tradeDate, session, kind))
}

// IncArmSuperseded records one unplaced arm expired by F4.
func IncArmSuperseded(st *Store) (int, error) {
	if st == nil || st.gdb == nil {
		return 0, fmt.Errorf("store required")
	}
	if err := st.gdb.Exec(`INSERT INTO system_config (key, value) VALUES (?, '1')
		ON CONFLICT(key) DO UPDATE SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT)`, ArmSupersededKey).Error; err != nil {
		return 0, err
	}
	return CountFromSystemConfig(st, ArmSupersededKey), nil
}

// SupersedeUnplacedArms (CLASS 47 F4, owner-ruled) moves every NEVER-PLACED arm
// row (no broker signal id) belonging to an OLDER plan version to the terminal
// state `superseded`.
//
// Why: on 2026-09-02 one unplaced v5 arm stayed non-terminal across six plan
// versions and held the class-33 cutover gate's leg 4 shut for ~5 hours. A row
// whose plan version has been superseded describes a setup the planner has
// already replaced; with no signal id there is nothing at the broker to cancel,
// so nothing can be orphaned by retiring it.
//
// PLACED rows are UNTOUCHED — they have a live broker order and belong to the
// sweep / stale-window reconcile paths, not to this one.
//
// Returns the ids it retired (A21: the caller logs them).
func (s *ArmedOrderStore) SupersedeUnplacedArms(traderID, planID string, currentVersion int) ([]int64, error) {
	var rows []ArmedOrderDB
	if err := s.db.Where(
		// state='armed' ONLY: a WORKING row is by definition placed and is the
		// sweep / stale-reconcile paths' business, never this one (stop-line:
		// "Placed/working rows untouched"). The signal-id predicate is belt and
		// braces on top of that.
		"trader_id = ? AND plan_id = ? AND version < ? AND (signal_id IS NULL OR signal_id = '') AND state = 'armed'",
		traderID, planID, currentVersion).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("superseded scan: %w", err)
	}
	var ids []int64
	for _, r := range rows {
		reason := fmt.Sprintf("superseded: never placed (no signal id) and plan moved v%d → v%d (class 47)", r.Version, currentVersion)
		if err := s.SetState(r.ID, "superseded", reason); err != nil {
			return ids, fmt.Errorf("superseded write id=%d: %w", r.ID, err)
		}
		ids = append(ids, r.ID)
	}
	return ids, nil
}
