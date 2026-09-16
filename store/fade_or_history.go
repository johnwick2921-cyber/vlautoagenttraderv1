// ── W2 — the prior-session opening-range median, READ from the tape ─────────
//
// The threshold for exclusion (a) is set against this tape's own distribution
// (C5), never doctrine. The median is over the PRIOR n sessions' 08:30 5m bar
// and returns the n it actually found, because the dispatch asked for 20 and
// the store holds 13 at this boot: a caller that prints "20" would be printing
// a literal (A11/A21).
package store

import (
	"sort"
	"time"
)

// PriorSessionORMedian returns the median OR (5m, session-open bar) over the
// n most recent COMPLETE sessions strictly before `before`, and the count it
// found. The session open is resolved from `before`'s own clock-of-day so the
// SAME session window is compared each day (A28) — never today-vs-today.
func (s *BarHistoryStore) PriorSessionORMedian(symbol string, before time.Time, n int) (float64, int) {
	if s == nil || s.db == nil || n <= 0 {
		return 0, 0
	}
	loc := before.Location()
	hh, mm := before.Hour(), before.Minute()
	var rows []BarHistoryDB
	// Enough 5m rows to cover n sessions with margin; filtered below.
	if err := s.db.Where("symbol = ? AND tf = ? AND open_time_ms < ?", symbol, "5m", before.UnixMilli()).
		Order("open_time_ms DESC").Limit(n * 24 * 12 * 2).Find(&rows).Error; err != nil {
		return 0, 0
	}
	var ors []float64
	seen := map[string]bool{}
	for _, r := range rows {
		t := time.UnixMilli(r.OpenTimeMs).In(loc)
		if t.Hour() != hh || t.Minute() != mm {
			continue
		}
		day := t.Format("2006-01-02")
		if seen[day] {
			continue
		}
		seen[day] = true
		ors = append(ors, r.H-r.L)
		if len(ors) >= n {
			break
		}
	}
	if len(ors) == 0 {
		return 0, 0
	}
	sort.Float64s(ors)
	m := len(ors)
	if m%2 == 1 {
		return ors[m/2], m
	}
	return (ors[m/2-1] + ors[m/2]) / 2, m
}
