// Command dayplan-arm arms day_plan (plan_enabled + clock fields) on the AI
// strategies — the P2.5 ARM. It is idempotent and DRY-RUN by default: run it once
// to preview, then with --confirm to write.
//
// Run it while the bot is STOPPED, in the flat window at ★ RESTART 1, AFTER
// backing up data/data.db and rebuilding the binary — then start the new binary,
// which reads the armed config (the old binary predates the day_plan codec and
// would strip the field, so never arm while the old binary is running).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"nofx/store"
)

func main() {
	dbPath := flag.String("db", "data/data.db", "path to the sqlite database")
	confirm := flag.Bool("confirm", false, "actually write (default: dry-run preview)")
	lastEntry := flag.String("last-entry", "13:00", "last-entry CT time (14:00 ET)")
	eodFlat := flag.String("eod-flat", "14:45", "eod-flat CT time (15:45 ET)")
	flag.Parse()

	st, err := store.New(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	var strategies []store.Strategy
	if err := st.GormDB().Find(&strategies).Error; err != nil {
		fmt.Fprintf(os.Stderr, "list strategies: %v\n", err)
		os.Exit(1)
	}

	armed := 0
	for i := range strategies {
		s := &strategies[i]
		var cfg store.StrategyConfig
		if err := json.Unmarshal([]byte(s.Config), &cfg); err != nil {
			fmt.Printf("SKIP %s (%s): bad config JSON: %v\n", s.ID, s.Name, err)
			continue
		}
		if cfg.StrategyType == "grid_trading" {
			continue // AI strategies only
		}
		if cfg.DayPlan == nil {
			cfg.DayPlan = store.DefaultDayPlanConfig()
		}
		cfg.DayPlan.PlanEnabled = true
		if *lastEntry != "" {
			cfg.DayPlan.LastEntryCT = *lastEntry
		}
		if *eodFlat != "" {
			cfg.DayPlan.EODFlatCT = *eodFlat
		}
		raw, err := json.Marshal(cfg)
		if err != nil {
			fmt.Printf("SKIP %s (%s): marshal: %v\n", s.ID, s.Name, err)
			continue
		}
		fmt.Printf("ARM %s (%s): plan_enabled=true last_entry=%s eod_flat=%s\n",
			s.ID, s.Name, cfg.DayPlan.LastEntryCT, cfg.DayPlan.EODFlatCT)
		if *confirm {
			s.Config = string(raw)
			if err := st.Strategy().Update(s); err != nil {
				fmt.Printf("  UPDATE FAILED: %v\n", err)
				continue
			}
		}
		armed++
	}

	if *confirm {
		fmt.Printf("\n✅ armed %d AI strateg(ies). Start the new binary; KEY LEVELS will light up on the next cycle.\n", armed)
	} else {
		fmt.Printf("\nDRY-RUN — %d AI strateg(ies) would be armed. BACK UP data/data.db, then re-run with --confirm (bot STOPPED).\n", armed)
	}
}
