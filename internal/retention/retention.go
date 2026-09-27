// Package retention wires the existing zero-caller prune functions
// (decision_records, equity snapshots, NT8 order snapshots, level_stats) to a
// daily job behind RETENTION_*_DAYS knobs that DEFAULT OFF (0 = keep
// forever). Owner ruling 2026-09-26: "keep the database, we are testing" —
// the job ships WIRED but DISABLED; every knob value is an owner decision.
// Trades / fills / receipts / plans are never prunable — the pin test proves
// it against a seeded store.
package retention

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"nofx/safe"
	"nofx/store"
)

// Config carries the four retention knobs in days. 0 = OFF.
type Config struct {
	DecisionDays    int
	EquityDays      int
	NT8SnapshotDays int
	LevelStatsDays  int
}

func envDays(name string) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// ResolveConfig reads the knobs (0 = OFF for each).
func ResolveConfig() Config {
	return Config{
		DecisionDays:    envDays("RETENTION_DECISION_DAYS"),
		EquityDays:      envDays("RETENTION_EQUITY_DAYS"),
		NT8SnapshotDays: envDays("RETENTION_NT8_SNAPSHOT_DAYS"),
		LevelStatsDays:  envDays("RETENTION_LEVEL_STATS_DAYS"),
	}
}

// Report is one run's outcome: rows pruned per table (only the four prunable
// tables) and the LIVE row counts read after the run (for the boot line —
// counts are READ, never literal).
type Report struct {
	DecisionPruned   int64
	EquityPruned     int64
	NT8Pruned        int64
	LevelStatsPruned int64
	DecisionRows     int64
	EquityRows       int64
	NT8Rows          int64
	LevelStatsRows   int64
	Errs             []error
}

// BootLine renders the boot line for the knobs' resolved state, with live
// row counts. Off knobs render "off (keep all)".
func (r Report) BootLine(cfg Config) string {
	part := func(name string, days int, pruned, rows int64) string {
		if days <= 0 {
			return fmt.Sprintf("%s=off(keep all, rows=%d)", name, rows)
		}
		return fmt.Sprintf("%s=%dd(pruned=%d, rows=%d)", name, days, pruned, rows)
	}
	return fmt.Sprintf("🧹 retention: %s · %s · %s · %s",
		part("decision_records", cfg.DecisionDays, r.DecisionPruned, r.DecisionRows),
		part("equity_snapshots", cfg.EquityDays, r.EquityPruned, r.EquityRows),
		part("nt8_order_snapshots", cfg.NT8SnapshotDays, r.NT8Pruned, r.NT8Rows),
		part("level_stats", cfg.LevelStatsDays, r.LevelStatsPruned, r.LevelStatsRows))
}

// RunDaily executes the daily prune for the resolved config. OFF knobs prune
// nothing. It calls ONLY the four existing prune functions — trades, fills,
// receipts and plans are never in the prune set.
func RunDaily(st *store.Store, now time.Time, cfg Config) Report {
	r := Report{}
	db := st.GormDB()

	if cfg.DecisionDays > 0 {
		for _, traderID := range traderIDs(db) {
			n, err := st.Decision().CleanOldRecords(traderID, cfg.DecisionDays)
			if err != nil {
				r.Errs = append(r.Errs, fmt.Errorf("decision prune %s: %w", traderID, err))
			}
			r.DecisionPruned += n
		}
	}
	if cfg.EquityDays > 0 {
		for _, traderID := range traderIDs(db) {
			n, err := st.Equity().CleanOldRecords(traderID, cfg.EquityDays)
			if err != nil {
				r.Errs = append(r.Errs, fmt.Errorf("equity prune %s: %w", traderID, err))
			}
			r.EquityPruned += n
		}
	}
	if cfg.NT8SnapshotDays > 0 {
		n, err := st.NT8OrderSnapshots().PruneBefore(now.AddDate(0, 0, -cfg.NT8SnapshotDays))
		if err != nil {
			r.Errs = append(r.Errs, fmt.Errorf("nt8 snapshot prune: %w", err))
		}
		r.NT8Pruned += n
	}
	if cfg.LevelStatsDays > 0 {
		n, err := st.LevelStats().PruneOlderThan(now.AddDate(0, 0, -cfg.LevelStatsDays).UnixMilli())
		if err != nil {
			r.Errs = append(r.Errs, fmt.Errorf("level_stats prune: %w", err))
		}
		r.LevelStatsPruned += n
	}

	// Live counts, READ after the run (the boot line quotes what exists).
	count := func(model any) int64 {
		var n int64
		if err := db.Model(model).Count(&n).Error; err != nil {
			r.Errs = append(r.Errs, err)
		}
		return n
	}
	r.DecisionRows = count(&store.DecisionRecordDB{})
	r.EquityRows = count(&store.EquitySnapshot{})
	r.NT8Rows = count(&store.NT8OrderSnapshot{})
	r.LevelStatsRows = count(&store.LevelStatsDB{})
	return r
}

// StartDaily runs the prune once per calendar day on a 1-minute ticker
// (catch-up stamp, like the agent scheduler: a missed minute still runs).
// Errors are collected into the Report; nothing is retried within the day.
func StartDaily(ctx context.Context, st *store.Store) {
	safe.GoNamed("retention-daily-ticker", func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		lastDay := time.Now().Format("2006-01-02")
		// First run happens on the next minute tick; counts are re-read then.
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				day := now.Format("2006-01-02")
				if day == lastDay {
					continue
				}
				lastDay = day
				RunDaily(st, now, ResolveConfig())
			}
		}
	})
}

// traderIDs returns every trader id in the store (the per-trader prune
// functions are trader-scoped; a WHERE-scoped delete is the only kind this
// job ever issues).
func traderIDs(db *gorm.DB) []string {
	var ids []string
	if err := db.Raw("SELECT id FROM traders").Scan(&ids).Error; err != nil {
		return nil
	}
	return ids
}
