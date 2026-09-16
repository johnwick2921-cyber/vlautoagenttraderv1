// Command excursions is the read-only view of trade_excursions (wave 1A, E6),
// and the runner for the backfill (E5).
//
// It writes to the excursion table only. It never touches a position, an
// order, a plan or a gate — this wave measures; it does not act.
//
//	go run ./cmd/excursions -db data/data.db
//	go run ./cmd/excursions -db data/data.db -backfill 2026-08-15
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"nofx/store"
	"nofx/trader"
)

func main() {
	db := flag.String("db", "data/data.db", "path to the SQLite store")
	backfillFrom := flag.String("backfill", "", "run the backfill for closed positions entered on or after this date (YYYY-MM-DD)")
	symbol := flag.String("symbol", "MNQ", "symbol to scan")
	traderID := flag.String("trader", "", "restrict to one trader id (default: all)")
	flag.Parse()

	st, err := store.New(*db)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open store:", err)
		os.Exit(1)
	}
	defer func() { _ = st.Close() }()

	if *backfillFrom != "" {
		from, err := time.Parse("2006-01-02", *backfillFrom)
		if err != nil {
			fmt.Fprintln(os.Stderr, "bad -backfill date:", err)
			os.Exit(2)
		}
		res, err := trader.BackfillExcursions(st, *symbol, *traderID, from, time.Now())
		if err != nil {
			fmt.Fprintln(os.Stderr, "backfill:", err)
			os.Exit(1)
		}
		fmt.Printf("backfill from %s: scanned=%d computed=%d no_coverage=%d levels_resolved=%d\n\n",
			*backfillFrom, res.Scanned, res.Computed, res.NoCoverage, res.LevelsFound)
	}

	fmt.Println(st.TradeExcursions().ExcursionBootLine())
	fmt.Println()
	for _, dim := range []string{"condition", "session", "scenario", "side"} {
		buckets, err := st.TradeExcursions().ExcursionDistribution(dim)
		if err != nil {
			fmt.Fprintln(os.Stderr, dim+":", err)
			continue
		}
		fmt.Println(store.RenderExcursionDistribution(dim, buckets))
	}
}
