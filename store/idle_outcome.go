package store

import (
	"fmt"
	"sort"
	"strings"
)

// IDLE-BEFORE vs OUTCOME (owner ruling 2026-09-03) — the table that decides
// whether IdleConnTimeout is the fix.
//
// 2026-09-03 08:11:38: a planner stream died to a peer FIN at 283.4s with
// 50,489 reasoning chars in, on a connection REUSED after 101,212ms idle. The
// identical resend succeeded on a connection idle 34,935ms. One observation.
// If cuts cluster above some idle threshold, setting IdleConnTimeout below it
// (or dialing fresh per planner call) is cheap, measurable and needs nothing
// from the provider.
//
// NOTHING IS SET. The owner's ruling is explicit: three more cuts decide. This
// aggregator exists so that decision is read off a table instead of recalled
// from a log, and it is written to make the small-n obvious rather than
// flattering — every row carries its n, an unresolved resend is counted as
// unresolved and never as a loss, and a fresh connection gets its own bucket
// instead of being filed under "0–30s idle" where it would look like evidence
// about idleness.

// IdleOutcomeRow is one idle bucket.
type IdleOutcomeRow struct {
	Bucket string
	N      int // rows in the bucket — every number below rests on it
	Cuts   int // peer-side ends (Kind="cut")
	Fires  int // our own watchdog closes (Kind="watchdog")

	Recovered  int // the identical resend landed
	Lost       int // it did not
	Unresolved int // it has not finished, or was never recorded
}

// idleBucket labels one row. A connection that was NOT reused is its own
// bucket: it had no idle period, and putting it in the lowest idle band would
// make fresh connections read as evidence about short idles.
func idleBucket(idleMs int64, reused bool) string {
	if !reused {
		return "fresh (not reused)"
	}
	switch {
	case idleMs < 30_000:
		return "0–30s"
	case idleMs < 60_000:
		return "30–60s"
	case idleMs < 120_000:
		return "≥60s"
	default:
		return "≥120s"
	}
}

// idleBucketOrder sorts buckets by the idleness they describe.
var idleBucketOrder = map[string]int{
	"fresh (not reused)": 0, "0–30s": 1, "30–60s": 2, "≥60s": 3, "≥120s": 4,
}

// IdleOutcomeTable groups every recorded cut and watchdog fire by the idleness
// of the connection it rode.
func (s *WatchdogFireStore) IdleOutcomeTable() ([]IdleOutcomeRow, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	var rows []WatchdogFireDB
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	agg := map[string]*IdleOutcomeRow{}
	for _, r := range rows {
		b := idleBucket(r.IdleBeforeMs, r.Reused)
		cur := agg[b]
		if cur == nil {
			cur = &IdleOutcomeRow{Bucket: b}
			agg[b] = cur
		}
		cur.N++
		if r.Kind == "cut" {
			cur.Cuts++
		} else {
			cur.Fires++
		}
		switch {
		case !r.Resolved:
			cur.Unresolved++
		case r.ResendOK:
			cur.Recovered++
		default:
			cur.Lost++
		}
	}
	out := make([]IdleOutcomeRow, 0, len(agg))
	for _, v := range agg {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		return idleBucketOrder[out[i].Bucket] < idleBucketOrder[out[j].Bucket]
	})
	return out, nil
}

// RenderIdleOutcomeTable formats the table. It states the total n up front and
// says out loud that no threshold has been set, so the table cannot be read as
// a decision it has not earned.
func RenderIdleOutcomeTable(rows []IdleOutcomeRow) string {
	total := 0
	for _, r := range rows {
		total += r.N
	}
	var b strings.Builder
	fmt.Fprintf(&b, "idle_before vs outcome (n=%d total) — no threshold is set; IdleConnTimeout stays untouched until the evidence is in\n", total)
	fmt.Fprintf(&b, "%-20s %5s %6s %6s %11s %6s %11s\n", "idle before call", "n", "cuts", "fires", "recovered", "lost", "unresolved")
	for _, r := range rows {
		fmt.Fprintf(&b, "%-20s %5d %6d %6d %11d %6d %11d\n",
			r.Bucket, r.N, r.Cuts, r.Fires, r.Recovered, r.Lost, r.Unresolved)
	}
	if total < 4 {
		fmt.Fprintf(&b, "\nn=%d is too few to read a threshold from. Three more cuts were the bar.\n", total)
	}
	return b.String()
}
