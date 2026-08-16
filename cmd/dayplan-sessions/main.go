// Command dayplan-sessions applies per-session day-plan overrides to existing
// strategies through the STORE layer (the same hand-rolled codec the app uses) —
// never raw SQL, so a value that cannot survive persistence fails here too.
//
// Why it exists: the researched per-session risk defaults
// (docs/VL-DAYPLAN-FULL-SPEC.md — "Risk defaults (evidence: Asia range exceeded
// 11.8% of days vs NY 37.8%; reopen spreads ~2.4×): ASIA min_grade=A +
// max_trades=1, LONDON min_grade=A") had drifted to min_grade B / max_trades 3,
// and there was no non-SQL way to set them outside the authenticated UI.
//
// SAFETY:
//   - DRY-RUN BY DEFAULT. Nothing is written without -confirm.
//   - It only TIGHTENS: it will refuse to loosen a grade or raise a trade cap
//     unless -allow-loosen is passed. A tool that silently widens risk is the
//     thing this whole class of bug is about.
//   - It only touches strategies that ALREADY have day_plan enabled and ALREADY
//     have an entry for that session. It never enables a session, never creates a
//     day_plan block, and never touches the guardrails master.
//   - Idempotent: re-running after a successful apply reports "already correct".
//
// Usage:
//
//	go run ./cmd/dayplan-sessions                 # dry-run preview
//	go run ./cmd/dayplan-sessions -confirm        # apply
//
// The running bot caches strategy config at trader-load, so a write here takes
// effect on the next trader reload or restart — i.e. at the owner's next deploy.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"nofx/store"
)

// gradeRank orders the quality floor so "tighter" is unambiguous. A is the
// strictest floor (fewest levels qualify).
var gradeRank = map[string]int{"A": 3, "B": 2, "C": 1}

func tighterGrade(current, want string) bool {
	return gradeRank[strings.ToUpper(want)] > gradeRank[strings.ToUpper(current)]
}

func main() {
	dbPath := flag.String("db", "data/data.db", "path to the sqlite database")
	confirm := flag.Bool("confirm", false, "actually write (default: dry-run preview)")
	allowLoosen := flag.Bool("allow-loosen", false, "permit a change that widens risk (default: refuse)")
	asiaGrade := flag.String("asia-min-grade", "A", "ASIA min_grade")
	asiaMaxTrades := flag.Int("asia-max-trades", 1, "ASIA max_trades")
	londonGrade := flag.String("london-min-grade", "A", "LONDON min_grade")
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

	if !*confirm {
		fmt.Println("DRY RUN — nothing will be written. Re-run with -confirm to apply.")
	}
	changed, skipped := 0, 0

	for i := range strategies {
		s := &strategies[i]
		var cfg store.StrategyConfig
		if err := json.Unmarshal([]byte(s.Config), &cfg); err != nil {
			fmt.Printf("SKIP %s (%s): bad config JSON: %v\n", s.ID, s.Name, err)
			skipped++
			continue
		}
		// Only strategies that already run the day plan.
		if cfg.DayPlan == nil || !cfg.DayPlan.PlanEnabled {
			continue
		}

		edits := []string{}
		for _, want := range []struct {
			session   string
			grade     string
			maxTrades *int
		}{
			{"ASIA", *asiaGrade, asiaMaxTrades},
			{"LONDON", *londonGrade, nil},
		} {
			ov := cfg.DayPlan.SessionOverride(want.session)
			if ov == nil {
				// Never invent a session block — that would silently enable behavior
				// the owner did not configure.
				continue
			}
			cur := ""
			if ov.MinGrade != nil {
				cur = *ov.MinGrade
			}
			if cur != strings.ToUpper(want.grade) {
				if cur != "" && !tighterGrade(cur, want.grade) && !*allowLoosen {
					fmt.Printf("  REFUSE %s min_grade %s → %s (would LOOSEN; pass -allow-loosen)\n", want.session, cur, want.grade)
				} else {
					g := strings.ToUpper(want.grade)
					ov.MinGrade = &g
					edits = append(edits, fmt.Sprintf("%s.min_grade %s→%s", want.session, orDash(cur), g))
				}
			}
			if want.maxTrades != nil {
				curMax := -1
				if ov.MaxTrades != nil {
					curMax = *ov.MaxTrades
				}
				if curMax != *want.maxTrades {
					if curMax >= 0 && *want.maxTrades > curMax && !*allowLoosen {
						fmt.Printf("  REFUSE %s max_trades %d → %d (would LOOSEN; pass -allow-loosen)\n", want.session, curMax, *want.maxTrades)
					} else {
						n := *want.maxTrades
						ov.MaxTrades = &n
						edits = append(edits, fmt.Sprintf("%s.max_trades %s→%d", want.session, orDashInt(curMax), n))
					}
				}
			}
		}

		if len(edits) == 0 {
			fmt.Printf("OK   %s (%s): already correct\n", short(s.ID), s.Name)
			continue
		}

		raw, err := json.Marshal(cfg) // through the production codec
		if err != nil {
			fmt.Printf("SKIP %s (%s): marshal: %v\n", short(s.ID), s.Name, err)
			skipped++
			continue
		}
		// Re-read what we are about to store: a field that cannot survive the
		// round trip must never be written as if it had.
		var verify store.StrategyConfig
		if err := json.Unmarshal(raw, &verify); err != nil {
			fmt.Printf("SKIP %s (%s): round-trip failed: %v\n", short(s.ID), s.Name, err)
			skipped++
			continue
		}

		fmt.Printf("EDIT %s (%s): %s\n", short(s.ID), s.Name, strings.Join(edits, " · "))
		if *confirm {
			s.Config = string(raw)
			if err := st.Strategy().Update(s); err != nil {
				fmt.Printf("  UPDATE FAILED: %v\n", err)
				skipped++
				continue
			}
		}
		changed++
	}

	verb := "would change"
	if *confirm {
		verb = "changed"
	}
	fmt.Printf("\n%s %d strateg(ies), skipped %d.\n", verb, changed, skipped)
	if *confirm && changed > 0 {
		fmt.Println("The running bot caches strategy config at trader-load — this takes effect at the next reload/restart.")
	}
}

func short(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
func orDash(s string) string {
	if s == "" {
		return "(unset)"
	}
	return s
}
func orDashInt(n int) string {
	if n < 0 {
		return "(unset)"
	}
	return fmt.Sprintf("%d", n)
}
