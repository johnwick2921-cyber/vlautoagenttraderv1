// Command bars-export dumps COMPLETED bars for every timeframe the resolver
// can answer, from the PERSISTED bars table, to CSV — the tape the detector
// replay and the bias calibration run on.
//
// BAR-SOURCE WAVE (2026-09-02, B5). Read-only: it opens the store, reads, and
// writes CSVs. It never contacts NT8 and never writes to the database, so it
// is safe to run against the live data file while the bot is running.
//
//	go run ./cmd/bars-export -db data/data.db -symbol MNQ -out docs/superpowers/reports/exports/bars
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"nofx/market"
	"nofx/store"
)

func main() {
	dbPath := flag.String("db", "data/data.db", "path to the SQLite store (opened read-only in effect: no writes are issued)")
	symbol := flag.String("symbol", "MNQ", "symbol to export")
	outDir := flag.String("out", "docs/superpowers/reports/exports/bars", "output directory")
	flag.Parse()

	st, err := store.New(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()
	bh := st.BarHistory()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	now := time.Now()
	tfs := market.LadderTFs()
	sort.Slice(tfs, func(i, j int) bool { return market.TFMinutes(tfs[i]) > market.TFMinutes(tfs[j]) })

	fmt.Printf("bars-export: symbol=%s db=%s out=%s\n", *symbol, *dbPath, *outDir)
	fmt.Printf("%-5s %8s  %-10s → %-10s  %-12s %s\n", "tf", "rows", "earliest", "latest", "convention", "file")
	total := 0
	for _, tf := range tfs {
		rows, err := bh.BarsBetween(*symbol, tf, 0, now.UnixMilli())
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: read failed: %v\n", tf, err)
			continue
		}
		if len(rows) == 0 {
			fmt.Printf("%-5s %8d  %-10s   %-10s  %-12s (skipped — nothing persisted)\n", tf, 0, "-", "-", "-")
			continue
		}
		name := filepath.Join(*outDir, fmt.Sprintf("%s_%s.csv", *symbol, tf))
		f, err := os.Create(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: create: %v\n", tf, err)
			continue
		}
		w := csv.NewWriter(f)
		_ = w.Write([]string{"open_time_ms", "open_time_utc", "open", "high", "low", "close", "volume", "tf", "convention"})
		conv := ""
		for _, r := range rows {
			if conv == "" {
				conv = r.Convention
			}
			_ = w.Write([]string{
				strconv.FormatInt(r.OpenTimeMs, 10),
				time.UnixMilli(r.OpenTimeMs).UTC().Format("2006-01-02T15:04:05Z"),
				strconv.FormatFloat(r.O, 'f', -1, 64), strconv.FormatFloat(r.H, 'f', -1, 64),
				strconv.FormatFloat(r.L, 'f', -1, 64), strconv.FormatFloat(r.C, 'f', -1, 64),
				strconv.FormatFloat(r.V, 'f', -1, 64), r.TF, r.Convention,
			})
		}
		w.Flush()
		_ = f.Close()
		if conv == "" {
			conv = "(unlabelled)"
		}
		fmt.Printf("%-5s %8d  %-10s → %-10s  %-12s %s\n", tf, len(rows),
			time.UnixMilli(rows[0].OpenTimeMs).UTC().Format("2006-01-02"),
			time.UnixMilli(rows[len(rows)-1].OpenTimeMs).UTC().Format("2006-01-02"),
			conv, name)
		total += len(rows)
	}
	fmt.Printf("\ntotal rows exported: %d\n", total)
	if why := market.ExcludedNative("1w"); why != "" {
		fmt.Printf("\nNOTE on %s_1w.csv: convention=fri_thu. These are NT8's NATIVE weekly bars and are\nNOT our Monday weeks — research only, no consumer reads them.\n  %s\n", *symbol, why)
	}
}
