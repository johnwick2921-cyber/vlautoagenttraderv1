package researchsnapshot

import (
	"fmt"
	"nofx/telemetry"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Volume control (dispatch 103, 2026-09-16): the archive keeps every fact —
// only the narration changes. Rollups carry rows/objects/drops/queue; drop
// notices are WARN-level and rate-limited to one line per minute with the
// coalesced delta.

// envDur reads a duration env knob (seconds, default fallback).
func envDur(name string, fallback time.Duration) time.Duration {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return fallback
}

func (r *Recorder) noteRows(facts []Fact) {
	r.rowsMu.Lock()
	for _, f := range facts {
		r.rows[f.Object]++
	}
	r.rowsMu.Unlock()
	telemetry.AddResearchSnapshotRows(uint64(len(facts)))
}

// coalesceDrop accumulates a drop into the minute window. The first drop of a
// window arms a short coalescing timer so a burst prints ONE WARN line with the
// full delta instead of one line per drop (the 13.2M-lines/day failure mode).
func (r *Recorder) coalesceDrop(reason string) {
	r.warnMu.Lock()
	if r.warnDelta == 0 {
		r.warnSince = time.Now()
		time.AfterFunc(r.dropWindow, func() { r.emitDropWarn(false) })
	}
	r.warnDelta++
	if r.warnReason == "" {
		r.warnReason = reason
	}
	r.warnMu.Unlock()
}

// emitDropWarn prints at most ONE WARN line per minute. force=true is used by
// the barrier flush and the rollup ticker so pending drops are never lost.
func (r *Recorder) emitDropWarn(force bool) {
	r.warnMu.Lock()
	defer r.warnMu.Unlock()
	if r.warnDelta == 0 {
		return
	}
	now := time.Now()
	if !force && !r.warnAt.IsZero() && now.Sub(r.warnAt) < time.Minute {
		return // rate-limited; the ticker will force it out within 60s
	}
	line := fmt.Sprintf("WARN research snapshot dropped: +%d over %s (reason: %s); total=%d", r.warnDelta, time.Since(r.warnSince).Round(100*time.Millisecond), r.warnReason, r.Dropped())
	r.warnDelta = 0
	r.warnReason = ""
	r.warnAt = now
	if r.warn != nil {
		r.warn(line)
	} else if r.info != nil {
		r.info(line)
	}
}

// emitRollup prints the volume summary: rows per object, the live dropped
// count, and the queue depth. Rate-limited to RESEARCH_LOG_EVERY_S unless
// forced (the administrative Flush barrier forces one so tests and shutdowns
// see a final number).
func (r *Recorder) emitRollup(force bool) {
	r.rowsMu.Lock()
	rows := make(map[string]uint64, len(r.rows))
	var total uint64
	for k, v := range r.rows {
		rows[k] = v
		total += v
	}
	r.rowsMu.Unlock()
	r.rollupMu.Lock()
	defer r.rollupMu.Unlock()
	if !force && !r.lastRollupAt.IsZero() && time.Since(r.lastRollupAt) < r.rollupEvery {
		return
	}
	if !force && total == 0 && r.Dropped() == 0 {
		return
	}
	r.lastRollupAt = time.Now()
	parts := make([]string, 0, len(Objects))
	keys := make([]string, 0, len(Objects))
	for _, k := range Objects {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, rows[k]))
	}
	line := fmt.Sprintf("🗄 research snapshot rollup: rows=%d objects={%s} drops=%d queue=%d", total, strings.Join(parts, " "), r.Dropped(), len(r.queue))
	if r.info != nil {
		r.info(line)
	} else if r.warn != nil {
		r.warn(line)
	}
}

// pruneOldFacts deletes research rows older than retainDays in bounded batches
// (DELETE ... LIMIT), one INFO line per batch, with a short sleep between
// batches so the trading boot is never blocked on a ~77 GB delete. VACUUM is
// NEVER run automatically: on a ~77 GB archive a VACUUM rewrites the whole
// file on the trading DB's disk — the owner decides when to reclaim space.
func pruneOldFacts(a *Archive, retainDays int, emit func(string)) int64 {
	cutoff := time.Now().Add(-time.Duration(retainDays) * 24 * time.Hour).UnixMilli()
	var total int64
	for {
		res, err := a.db.Exec(`DELETE FROM research_facts WHERE id IN (SELECT id FROM research_facts WHERE captured_ms < ? LIMIT 1000)`, cutoff)
		if err != nil {
			if emit != nil {
				emit(fmt.Sprintf("WARN research snapshot prune failed: %v", err))
			}
			return total
		}
		n, _ := res.RowsAffected()
		total += n
		if n == 0 {
			return total
		}
		if emit != nil {
			emit(fmt.Sprintf("research snapshot prune: -%d rows (batch) — %d total", n, total))
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// retainDays: RESEARCH_RETAIN_DAYS UNSET = 0 = NO prune, ever (review B2).
func retainDays() int {
	v := os.Getenv("RESEARCH_RETAIN_DAYS")
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
		return n
	}
	return 0
}

var (
	statusMu   sync.Mutex
	statusNote = "OFF (RESEARCH_SNAPSHOT unset)"
)

func setStatusNote(s string) {
	statusMu.Lock()
	defer statusMu.Unlock()
	statusNote = s
}

func currentStatusNote() string {
	statusMu.Lock()
	defer statusMu.Unlock()
	return statusNote
}
