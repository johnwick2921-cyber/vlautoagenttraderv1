package main
// Command dayplan-level-repair clears FALSE burned marks from the level_state
// table (2026-08-17 NY: 12 of 12 levels consumed/done, written by the pre-fix
// windowless logic) and lets the bot's P1c windowed evaluator rebuild honest
// consumption from bars on its next cycle.
//
// It is DRY-RUN by default. Procedure:
//
//  1. STOP the bot (or accept that a running PRE-fix binary will re-burn).
//  2. sqlite3 data/data.db ".backup ~/nofx-backups/<name>/level-repair.db"
//  3. go run ./cmd/dayplan-level-repair -db data/data.db        (preview)
//  4. go run ./cmd/dayplan-level-repair -db data/data.db -confirm
//  5. start the NEW binary (P1c windowed consumption) — the next cycle
//     re-evaluates every level on bars SINCE ITS ROW'S BIRTH.
//
// Rows are reset to freshness=C (tested×2), never back to fresh A: a repair
// decays, it does not resurrect. -cutoff-min limits the reset to rows last
// updated before N minutes ago (default 0 = every burned row).
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"nofx/store"
)

func main() {
	dbPath := flag.String("db", "data/data.db", "path to the sqlite database")
	confirm := flag.Bool("confirm", false, "actually write (default: dry-run preview)")
	cutoffMin := flag.Int64("cutoff-min", 0, "only reset rows updated > N minutes ago (0 = all burned rows)")
	flag.Parse()

	st, err := store.New(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	var rows []store.LevelStateDB
	if err := st.GormDB().Where("consumed = ? OR freshness = ?", true, store.FreshnessDone).
		Order("updated_at DESC").Find(&rows).Error; err != nil {
		fmt.Fprintf(os.Stderr, "list level_state: %v\n", err)
		os.Exit(1)
	}
	if len(rows) == 0 {
		fmt.Println("level_state: no burned rows. Nothing to repair.")
		return
	}

	var cutoffMs int64
	if *cutoffMin > 0 {
		cutoffMs = time.Now().Add(-time.Duration(*cutoffMin) * time.Minute).UnixMilli()
	}

	fmt.Printf("level_state burned rows: %d\n", len(rows))
	inScope, outScope := 0, 0
	for _, r := range rows {
		age := time.Since(r.UpdatedAt).Round(time.Minute)
		scope := cutoffMs == 0 || r.UpdatedAt.UnixMilli() < cutoffMs
		if !scope {
			outScope++
			continue
		}
		inScope++
		fmt.Printf("  RESET %-12s bin=%-6d price=%-10.2f freshness=%s updated=%s (%s ago)\n",
			r.LevelType, r.BinIndex, r.Price, r.Freshness,
			r.UpdatedAt.Format("2006-01-02 15:04:05"), age)
	}
	fmt.Printf("in scope (%d) → consumed=false, freshness=C; out of cutoff scope: %d\n", inScope, outScope)

	if !*confirm {
		fmt.Println("\nDRY-RUN — no writes. Add -confirm to execute.")
		return
	}

	n, err := st.LevelState().ResetBurns(cutoffMs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reset: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n✅ reset %d burned row(s). Rebuild happens in-process: the next bot cycle\n   (P1c windowed consumption) re-evaluates each level on bars since its row's birth.\n", n)
}
