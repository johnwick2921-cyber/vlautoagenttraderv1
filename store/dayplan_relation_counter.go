package store

import (
	"fmt"
	"strconv"
	"strings"
)

// ── S3 (2026-09-16) — the relation census ────────────────────────────────────
//
// dayplan_scenarios_by_relation:<session>:<relation> — one counter per session
// per validator-stamped relation. Counters RECORD, never infer (canon 35): the
// planner write site increments them from the stamped doc, never from a guess.

// RelationCounterKey renders the counter key for a session + relation.
func RelationCounterKey(session, relation string) string {
	return fmt.Sprintf("dayplan_scenarios_by_relation:%s:%s", session, relation)
}

// IncScenarioRelation bumps the session/relation census counter and returns the
// new count.
func IncScenarioRelation(st *Store, session, relation string) (int, error) {
	if st == nil || st.gdb == nil {
		return 0, fmt.Errorf("store required")
	}
	key := RelationCounterKey(session, relation)
	err := st.gdb.Exec(`INSERT INTO system_config (key, value) VALUES (?, '1')
		ON CONFLICT(key) DO UPDATE SET value = CAST(CAST(value AS INTEGER) + 1 AS TEXT)`, key).Error
	if err != nil {
		return 0, err
	}
	return ScenarioRelationCount(st, session, relation), nil
}

// ScenarioRelationCount reads the recorded count (0 when never bumped).
func ScenarioRelationCount(st *Store, session, relation string) int {
	if st == nil {
		return 0
	}
	v, err := st.GetSystemConfig(RelationCounterKey(session, relation))
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(v))
	return n
}
