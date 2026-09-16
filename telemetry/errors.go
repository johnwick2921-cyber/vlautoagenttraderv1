package telemetry

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// P0-CLEANUP (2026-08-19) — SELF-ANNOUNCING ERRORS.
//
// Every error path that costs a decision or a trade records ONE structured
// event here: a stable type, a plain-language cause, and its cost. The daily
// digest line ("errors today: N (types: …), decisions lost: N") and the
// /api/risk/errors panel both read this table, so a NEW error class announces
// itself the same day instead of hiding in logs for a week.
//
// In-memory per CME session-day (same lifetime as the gate-block counters —
// the session table resets at the 17:00 CT rollover). The first occurrence of
// an error TYPE in a session returns firstSeen=true, which callers use to emit
// a P1 alert so the feed (not just the table) announces the new class.

// ErrorCost is what the event cost: a decision, a trade, or nothing.
type ErrorCost string

const (
	CostDecisionLost ErrorCost = "decision_lost"
	CostTradeLost    ErrorCost = "trade_lost"
	CostNone         ErrorCost = "none"
)

type errorEventKey struct {
	trader string
	typ    string
	cause  string
	cost   string
}

type errorEventAgg struct {
	count         int
	lastOccurred  int64
	decisionsLost int // incremented per event whose cost == decision_lost
	tradesLost    int
}

var (
	errorMu     sync.Mutex
	errorDay    int64 // active session-day (ms at the 17:00 CT rollover); 0 = unset
	errorEvents = map[errorEventKey]*errorEventAgg{}
	errorSeen   = map[string]bool{} // per-session-day: trader|type seen

	// ErrorAnnounceFunc is called on the FIRST occurrence of an error type in
	// a session-day — the trader installs it to emit a P1 alert so a NEW error
	// class announces itself the same day instead of a week later.
	ErrorAnnounceFunc func(trader, typ, cause string, cost ErrorCost)
)

// SetErrorSessionDay adopts/rolls the error-event day (called from the same
// rollover hook as RolloverGateBlocks).
func SetErrorSessionDay(sessionDayMs int64) {
	errorMu.Lock()
	defer errorMu.Unlock()
	if errorDay != sessionDayMs {
		errorDay = sessionDayMs
		errorEvents = map[errorEventKey]*errorEventAgg{}
		errorSeen = map[string]bool{}
	}
}

// RecordError records one structured error event. Returns firstSeen=true the
// FIRST time this (trader,type) appears in the current session-day — the
// caller's signal to announce the new class.
func RecordError(trader, typ, cause string, cost ErrorCost) (firstSeen bool) {
	if typ == "" {
		typ = "unknown"
	}
	errorMu.Lock()
	defer errorMu.Unlock()
	seenKey := trader + "|" + typ
	if !errorSeen[seenKey] {
		errorSeen[seenKey] = true
		firstSeen = true
	}
	k := errorEventKey{trader: trader, typ: typ, cause: cause, cost: string(cost)}
	a := errorEvents[k]
	if a == nil {
		a = &errorEventAgg{}
		errorEvents[k] = a
	}
	a.count++
	if cost == CostDecisionLost {
		a.decisionsLost++
	}
	if cost == CostTradeLost {
		a.tradesLost++
	}
	ann := ErrorAnnounceFunc
	if firstSeen && ann != nil {
		ann(trader, typ, cause, cost)
	}
	return firstSeen
}

// ErrorEventRow is one aggregated row for the panel / digest.
type ErrorEventRow struct {
	Trader        string `json:"trader"`
	Type          string `json:"type"`
	Cause         string `json:"cause"`
	Cost          string `json:"cost"`
	Count         int    `json:"count"`
	DecisionsLost int    `json:"decisions_lost"`
	TradesLost    int    `json:"trades_lost"`
}

// ErrorSummary returns the aggregated rows for the current session-day, sorted
// by count desc. trader "" returns all traders.
func ErrorSummary(trader string) []ErrorEventRow {
	errorMu.Lock()
	defer errorMu.Unlock()
	out := make([]ErrorEventRow, 0, len(errorEvents))
	for k, a := range errorEvents {
		if trader != "" && k.trader != trader {
			continue
		}
		out = append(out, ErrorEventRow{
			Trader: k.trader, Type: k.typ, Cause: k.cause, Cost: k.cost,
			Count: a.count, DecisionsLost: a.decisionsLost, TradesLost: a.tradesLost,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	return out
}

// ErrorDigestLine renders the daily-digest error line:
// "errors today: N (types: t1, t2, …), decisions lost: N".
func ErrorDigestLine(trader string) string {
	rows := ErrorSummary(trader)
	if len(rows) == 0 {
		return ""
	}
	total, lost := 0, 0
	seen := map[string]bool{}
	var types []string
	for _, r := range rows {
		total += r.Count
		lost += r.DecisionsLost
		if !seen[r.Type] {
			seen[r.Type] = true
			types = append(types, r.Type)
		}
	}
	return fmt.Sprintf("errors today: %d (types: %s), decisions lost: %d", total, strings.Join(types, ", "), lost)
}
