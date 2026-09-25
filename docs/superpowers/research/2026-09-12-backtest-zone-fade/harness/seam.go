package main

// seam.go — C10: the seam assertion set. Every window in this backtest is read
// PER CONTRACT (BarsBetweenOn semantics emulated by loading each contract's
// series separately); the harness asserts in code that no series mixes
// contracts and that the import census matches wave 101's D5 table.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type seamResult struct {
	Summary                    string            `json:"summary"`
	ImportCensus               []censusRow       `json:"import_census"`
	D5JSONL                   map[string]map[string]int64 `json:"d5_jsonl_totals"`
	D5Delta                   map[string]map[string]int64 `json:"d5_delta_db_minus_jsonl"`
	D5Matches                  bool              `json:"d5_matches"`
	Standalone0622Rows         int64             `json:"standalone_mnq_0622_rows"`
	SeriesSingleContractAssert bool              `json:"series_single_contract_assert"`
	AssertionText              string            `json:"assertion_text"`
}

type censusRow struct {
	Contract string `json:"contract"`
	TF       string `json:"tf"`
	Rows     int64  `json:"rows"`
}

// seamChecks runs the C10 assertions against the DB copy and the wave-101 D5
// artifact in the worktree.
func seamChecks(db *sql.DB, d *barDB) seamResult {
	r := seamResult{D5JSONL: map[string]map[string]int64{}, D5Delta: map[string]map[string]int64{}}

	// 1) import census from the copy.
	rows, err := db.Query(`SELECT contract, tf, COUNT(*) FROM bars
		WHERE symbol='MNQ' AND source='historical_import' GROUP BY contract, tf ORDER BY contract, tf`)
	if err == nil {
		for rows.Next() {
			var c censusRow
			if err := rows.Scan(&c.Contract, &c.TF, &c.Rows); err == nil {
				r.ImportCensus = append(r.ImportCensus, c)
			}
		}
		rows.Close()
	}
	dbCensus := map[string]map[string]int64{}
	for _, c := range r.ImportCensus {
		if dbCensus[c.Contract] == nil {
			dbCensus[c.Contract] = map[string]int64{}
		}
		dbCensus[c.Contract][c.TF] += c.Rows
	}

	// 2) wave-101 D5 totals from the import-results.jsonl in this checkout.
	if b, err := os.ReadFile(importResultsPath); err == nil {
		lines := strings.Split(strings.TrimSpace(string(b)), "\n")
		for _, ln := range lines {
			var rec struct {
				Contract string `json:"contract"`
				Results  []struct {
					Timeframe string `json:"timeframe"`
					Status    string `json:"status"`
					Imported  int64  `json:"imported"`
				} `json:"results"`
			}
			if json.Unmarshal([]byte(ln), &rec) != nil {
				continue
			}
			if r.D5JSONL[rec.Contract] == nil {
				r.D5JSONL[rec.Contract] = map[string]int64{}
			}
			for _, res := range rec.Results {
				r.D5JSONL[rec.Contract][res.Timeframe] += res.Imported
			}
		}
	}
	// delta: db minus jsonl. The standalone MNQ 06-22 import (66,450 rows, run
	// before the jsonl loop) is expected to be the only nonzero delta.
	keys := map[string]bool{}
	for c := range dbCensus {
		keys[c] = true
	}
	for c := range r.D5JSONL {
		keys[c] = true
	}
	var keyList []string
	for k := range keys {
		keyList = append(keyList, k)
	}
	sort.Strings(keyList)
	match := true
	for _, c := range keyList {
		r.D5Delta[c] = map[string]int64{}
		for _, tf := range []string{"1m", "5m", "15m", "1h"} {
			dv := dbCensus[c][tf]
			jv := r.D5JSONL[c][tf]
			if dv != jv {
				r.D5Delta[c][tf] = dv - jv
				if c != "MNQ 06-22" {
					match = false
				}
			}
		}
	}
	r.Standalone0622Rows = dbCensus["MNQ 06-22"]["1m"] + dbCensus["MNQ 06-22"]["5m"] + dbCensus["MNQ 06-22"]["15m"] + dbCensus["MNQ 06-22"]["1h"]
	if r.Standalone0622Rows != 66450 {
		match = false
	}
	r.D5Matches = match

	// 3) every loaded series carries exactly its own contract (formal assert).
	r.SeriesSingleContractAssert = true
	for k, rowsList := range d.byKey {
		for _, rr := range rowsList {
			if rr.contract != k.contract {
				r.SeriesSingleContractAssert = false
			}
		}
	}
	r.AssertionText = "assert: every bar of every window was fetched from ONE contract's series; " +
		"a window is built by lastClosed(contract, tf, n, readTime) and sessionBars1m(contract, …) " +
		"which read only that contract's rows — no window ever concatenates two contracts' bars."
	r.Summary = fmt.Sprintf("D5 match=%v · standalone 06-22=%d rows · single-contract assert=%v · import census rows=%d",
		r.D5Matches, r.Standalone0622Rows, r.SeriesSingleContractAssert, len(r.ImportCensus))
	return r
}

// importResultsPath is the wave-101 D5 artifact (pinned on dev @ 84eaf69f).
var importResultsPath = "docs/superpowers/reports/2026-09-11-historical-backfill-data/import-results.jsonl"
