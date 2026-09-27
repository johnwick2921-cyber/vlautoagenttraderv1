package main

// eratrend.go — S4c (b)/(c): the ERA-WIDE trend series.
//
// Why (root cause (a), [A]): the port's trendFor keyed on the 1m-derived
// contract label (contractAt). The store holds deep series ONLY under
// "MNQ 09-26" (1d = 1,896 rows 2019-05-02→2026-09-01, 4h = 2,036 rows
// 2025-05-18→2026-09-10) — every other contract label has ZERO daily rows and
// ~0–21 4h rows. Reads in 2022–2025 resolved to young contracts → empty daily
// series → 0 labelled swings → "range" everywhere. The data was deep; the
// filter was wrong.
//
// The era series (direction/swings ONLY — prices are internal and de-stepped,
// never published as levels):
//   1d = MNQ 09-26 rows (through open 1788238800000) + MNQ 12-26 rows
//        (from open 1788325200000) shifted by −632.50 (measured close-to-close
//        step: row 09-26-last@1788238800000 c=29186.25 → row
//        12-26-first@1788325200000 c=29818.75).
//   4h = MNQ 09-26 rows (through open 1789092000000) + MNQ 12-26 rows
//        (from open 1789106400000) shifted by −380.00 (step matches the CTO's
//        stated 4h roll step exactly).
// A constant tail shift preserves fractal swing order statistics — the join
// cannot fabricate a swing.
//
// -eratrend reads trends.jsonl (day, session, read_at_ms) and writes
// era-trends.jsonl with s1Trend computed on the era series at each read time
// (same closed-bar filter, same port, lookahead rule unchanged: bars close
// strictly before the read).

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"nofx/market"
)

type eraBar struct {
	openMs     int64
	o, h, l, c float64
}

type eraTrendRow struct {
	Day      string `json:"day"`
	Session  string `json:"session"`
	ReadAtMs int64  `json:"read_at_ms"`
	DTrend   string `json:"d_trend"`
	DSwings  int    `json:"d_swings"`
	DClosed  int    `json:"d_closed"`
	H4Trend  string `json:"h4_trend"`
	H4Swings int    `json:"h4_swings"`
	H4Closed int    `json:"h4_closed"`
}

const (
	dailyJoinOpenMs = int64(1788238800000) // last 09-26 daily row open
	dailyStep       = 632.50
	h4JoinOpenMs    = int64(1789092000000) // last 09-26 4h row open
	h4Step          = 380.00
)

// loadEraSeries builds the de-stepped continuous daily (tf=1d) and 4h series.
func loadEraSeries(db *sql.DB) (d1d, h4 []eraBar, err error) {
	load := func(tf string) ([]eraBar, error) {
		rows, err := db.Query(`SELECT contract, open_time_ms, o, h, l, c FROM bars
			WHERE symbol='MNQ' AND tf=? AND COALESCE(source,'') NOT IN ('mixed','replay:off-scale')
			  AND (contract='MNQ 09-26' OR contract='MNQ 12-26')
			ORDER BY open_time_ms`, tf)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		step := dailyStep
		joinOpen := dailyJoinOpenMs
		if tf == "4h" {
			step = h4Step
			joinOpen = h4JoinOpenMs
		}
		var out []eraBar
		for rows.Next() {
			var c string
			var b eraBar
			if err := rows.Scan(&c, &b.openMs, &b.o, &b.h, &b.l, &b.c); err != nil {
				return nil, err
			}
			if c == "MNQ 12-26" && b.openMs > joinOpen {
				b.o -= step
				b.h -= step
				b.l -= step
				b.c -= step
			}
			out = append(out, b)
		}
		return out, rows.Err()
	}
	if d1d, err = load("1d"); err != nil {
		return nil, nil, err
	}
	if h4, err = load("4h"); err != nil {
		return nil, nil, err
	}
	return d1d, h4, nil
}

func eraToKline(rows []eraBar) []market.Kline {
	out := make([]market.Kline, 0, len(rows))
	for _, r := range rows {
		out = append(out, market.Kline{
			OpenTime: r.openMs, Open: r.o, High: r.h, Low: r.l, Close: r.c,
		})
	}
	return out
}

// runEraTrend reads the S4 trends.jsonl and recomputes D/4h trends on the era
// series at every read time.
func runEraTrend(db *sql.DB, trendsPath, outPath string) error {
	d1d, h4, err := loadEraSeries(db)
	if err != nil {
		return err
	}
	if len(d1d) == 0 || len(h4) == 0 {
		return fmt.Errorf("era series empty: 1d=%d 4h=%d", len(d1d), len(h4))
	}
	fmt.Printf("era series: 1d=%d bars (%d→%d), 4h=%d bars (%d→%d)\n",
		len(d1d), d1d[0].openMs, d1d[len(d1d)-1].openMs, len(h4), h4[0].openMs, h4[len(h4)-1].openMs)
	dK := eraToKline(d1d)
	hK := eraToKline(h4)

	in, err := os.Open(trendsPath)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()
	w := bufio.NewWriterSize(out, 1<<20)
	defer w.Flush()

	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	n := 0
	for sc.Scan() {
		var t struct {
			Day      string `json:"day"`
			Session  string `json:"session"`
			ReadAtMs int64  `json:"read_at_ms"`
		}
		if err := json.Unmarshal(sc.Bytes(), &t); err != nil {
			continue
		}
		row := eraTrendRow{Day: t.Day, Session: t.Session, ReadAtMs: t.ReadAtMs}
		row.DTrend, row.DSwings, row.DClosed = s1Trend(dK, 1440, t.ReadAtMs)
		row.H4Trend, row.H4Swings, row.H4Closed = s1Trend(hK, 240, t.ReadAtMs)
		line, err := json.Marshal(row)
		if err != nil {
			return err
		}
		w.Write(line)
		w.WriteByte('\n')
		n++
	}
	fmt.Printf("era-trend rows written: %d\n", n)
	return sc.Err()
}

var _ = filepath.Join
var _ = sort.Search
