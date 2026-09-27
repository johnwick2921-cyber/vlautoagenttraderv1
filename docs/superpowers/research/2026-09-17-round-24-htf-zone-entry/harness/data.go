//go:build r24harness

package main

// data.go — loads MNQ bars from the DB COPY in the worktree (never the live
// store; the live store stays untouched — A3/D7). Bars are grouped by
// (contract, tf) and kept sorted by open time. The seam law is enforced here:
// every window is built from ONE contract's series; nothing ever concatenates
// two contracts' bars.

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"time"

	_ "nofx/store/sqlitedriver" // the ONE sqlite registration site (DS-102 fold, CTO 1790305899255)

	"nofx/kernel"
	"nofx/market"
)

// tfMillis maps a timeframe label to its bar width in milliseconds.
var tfMillis = map[string]int64{
	"1m": 60_000, "3m": 180_000, "5m": 300_000, "10m": 600_000, "15m": 900_000,
	"30m": 1_800_000, "1h": 3_600_000, "2h": 7_200_000, "4h": 14_400_000,
	"6h": 21_600_000, "8h": 28_800_000, "12h": 43_200_000, "1d": 86_400_000,
}

// seriesKey is (contract, tf).
type seriesKey struct{ contract, tf string }

// barRow is a raw DB row.
type barRow struct {
	contract      string
	openMs        int64
	o, h, l, c, v float64
	source        string
}

type barDB struct {
	byKey map[seriesKey][]barRow
	// merged 1m index: (openMs, contract) sorted by openMs — ContractAt emulation.
	merged1m []struct {
		openMs   int64
		contract string
	}
}

// openCopy opens the DB copy read-only.
func openCopy(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("db copy missing: %w", err)
	}
	dsn := "file:" + path + "?mode=ro&_pragma=query_only(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

// loadBars reads every usable MNQ bar from the copy and groups it by key.
// Excludes exactly what BarsBetweenOn excludes: source in ('mixed','replay:off-scale').
func loadBars(db *sql.DB, tfs []string) (*barDB, error) {
	out := &barDB{byKey: map[seriesKey][]barRow{}}
	for _, tf := range tfs {
		q := `SELECT contract, open_time_ms, o, h, l, c, v, COALESCE(source,'')
		      FROM bars
		      WHERE symbol='MNQ' AND tf=? AND contract<>'' AND contract<>'unrecomputable:spans_roll'
		        AND COALESCE(source,'') NOT IN ('mixed','replay:off-scale')
		      ORDER BY contract, open_time_ms`
		rows, err := db.Query(q, tf)
		if err != nil {
			return nil, fmt.Errorf("query bars tf=%s: %w", tf, err)
		}
		n := 0
		for rows.Next() {
			var r barRow
			if err := rows.Scan(&r.contract, &r.openMs, &r.o, &r.h, &r.l, &r.c, &r.v, &r.source); err != nil {
				rows.Close()
				return nil, err
			}
			k := seriesKey{r.contract, tf}
			out.byKey[k] = append(out.byKey[k], r)
			n++
			if tf == "1m" {
				out.merged1m = append(out.merged1m, struct {
					openMs   int64
					contract string
				}{r.openMs, r.contract})
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		fmt.Printf("loaded tf=%s rows=%d\n", tf, n)
	}
	sort.Slice(out.merged1m, func(i, j int) bool { return out.merged1m[i].openMs < out.merged1m[j].openMs })
	return out, nil
}

// contractAt emulates store.ContractAt: the contract of the newest usable 1m
// bar at or before ms. ok=false when none exists.
func (b *barDB) contractAt(ms int64) (string, bool) {
	idx := sort.Search(len(b.merged1m), func(i int) bool { return b.merged1m[i].openMs > ms })
	if idx == 0 {
		return "", false
	}
	return b.merged1m[idx-1].contract, true
}

// toKline converts rows to market.Kline with the timeframe's proper close time
// (the live ring's convention; the store reader's +59_999 convention is 1m-only).
func toKline(rows []barRow, tf string) []market.Kline {
	width := tfMillis[tf]
	if width <= 0 {
		width = 60_000
	}
	out := make([]market.Kline, 0, len(rows))
	for _, r := range rows {
		out = append(out, market.Kline{
			OpenTime: r.openMs, CloseTime: r.openMs + width - 1,
			Open: r.o, High: r.h, Low: r.l, Close: r.c, Volume: r.v,
		})
	}
	return out
}

// lastClosed returns the last n bars of tf for contract with CloseTime < before,
// ascending. This matches the live provider's newest-n closed-bars view.
func (b *barDB) lastClosed(contract, tf string, n int, before time.Time) []market.Kline {
	rows := b.byKey[seriesKey{contract, tf}]
	if len(rows) == 0 {
		return nil
	}
	limit := before.UnixMilli()
	// rows are sorted by openMs; the bar closes at openMs+width-1, so the last
	// closed bar has openMs <= limit-width. Walk back from the end.
	width := tfMillis[tf]
	hi := sort.Search(len(rows), func(i int) bool { return rows[i].openMs+width-1 >= limit })
	if hi > len(rows) {
		hi = len(rows)
	}
	lo := hi - n
	if lo < 0 {
		lo = 0
	}
	return toKline(rows[lo:hi], tf)
}

// sessionBars1m returns 1m bars of the contract with windowStartMs <= openMs <
// windowEndMs, plus the single bar immediately before windowStartMs as touch
// context (its range is the "previous bar" for a touch on the very first bar).
func (b *barDB) sessionBars1m(contract string, winStartMs, winEndMs int64) (prev *market.Kline, bars []market.Kline) {
	rows := b.byKey[seriesKey{contract, "1m"}]
	start := sort.Search(len(rows), func(i int) bool { return rows[i].openMs >= winStartMs })
	if start > 0 {
		pr := toKline(rows[start-1:start], "1m")
		prev = &pr[0]
	}
	end := sort.Search(len(rows), func(i int) bool { return rows[i].openMs >= winEndMs })
	if end > start {
		bars = toKline(rows[start:end], "1m")
	}
	return prev, bars
}

// sessionTouchesWindow is the [start,end) window a session's touches live in.
// ASIA 17:00→02:00 next day (read 16:30); LONDON 02:00→08:30 (read 01:30);
// NY 08:30→14:45 (read 08:00) — the DefaultSessionRegistry values, CT-anchored.
func sessionWindow(day time.Time, session string) (readTime, winStart, winEnd time.Time) {
	loc := kernel.CTLocation()
	switch session {
	case "ASIA":
		readTime = time.Date(day.Year(), day.Month(), day.Day(), 16, 30, 0, 0, loc)
		winStart = time.Date(day.Year(), day.Month(), day.Day(), 17, 0, 0, 0, loc)
		winEnd = time.Date(day.Year(), day.Month(), day.Day()+1, 2, 0, 0, 0, loc)
	case "LONDON":
		readTime = time.Date(day.Year(), day.Month(), day.Day(), 1, 30, 0, 0, loc)
		winStart = time.Date(day.Year(), day.Month(), day.Day(), 2, 0, 0, 0, loc)
		winEnd = time.Date(day.Year(), day.Month(), day.Day(), 8, 30, 0, 0, loc)
	case "NY":
		readTime = time.Date(day.Year(), day.Month(), day.Day(), 8, 0, 0, 0, loc)
		winStart = time.Date(day.Year(), day.Month(), day.Day(), 8, 30, 0, 0, loc)
		winEnd = time.Date(day.Year(), day.Month(), day.Day(), 14, 45, 0, 0, loc)
	}
	return readTime, winStart, winEnd
}

// flatMsOf is the session's flat instant in unix ms.
func flatMsOf(winEnd time.Time) int64 { return winEnd.UnixMilli() }
