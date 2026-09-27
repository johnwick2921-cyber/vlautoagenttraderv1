package retention

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/store"
)

// TestRunDailyPrunesOnlyTheFourTables pins P2-1 at the production function:
// with every knob ON and ancient rows everywhere, ONLY decision_records,
// trader_equity_snapshots, nt8_order_snapshots and level_stats may shrink —
// trades, fills, receipts and plans are never prunable (owner ruling).
func TestRunDailyPrunesOnlyTheFourTables(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "ret.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer st.Close()
	db := st.GormDB()

	old := time.Now().AddDate(0, 0, -400)
	must := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Fatalf("seed %q: %v", sql, err)
		}
	}

	// A trader row so the per-trader prunes have an id to iterate, and the
	// sub-stores whose tables store.New does not auto-migrate.
	must("INSERT INTO traders (id, user_id, name, ai_model_id, exchange_id, strategy_id, initial_balance, created_at, updated_at) VALUES ('t-ret','u','ret','','','',0,'2026-01-01','2026-01-01')")
	if err := st.LevelStats().Migrate(); err != nil {
		t.Fatalf("level_stats migrate: %v", err)
	}

	// Prunable tables, ancient rows.
	must("INSERT INTO decision_records (trader_id, cycle_number, timestamp, system_prompt, input_prompt, success, created_at) VALUES ('t-ret', 1, ?, '', '', 0, ?)", old, old)
	must("INSERT INTO trader_equity_snapshots (trader_id, account, timestamp) VALUES ('t-ret','', ?)", old)
	must("INSERT INTO nt8_order_snapshots (account, symbol, build_id, reason, orders_json, received_at_ms) VALUES ('a','MNQ','b','r','[]', ?)", old.UnixMilli())
	must("INSERT INTO level_stats (trader_id, session_day, price, label, kind, grade, role, family, created_at) VALUES ('t-ret','2026-01-01', 1, 'l', 'k', 'g', 'r', 'f', ?)", old)

	// PROTECTED tables, ancient rows — must survive any knob value.
	must("INSERT INTO trader_positions (trader_id, symbol, side, quantity, entry_price, entry_time, status, created_at) VALUES ('t-ret','MNQ','long', 1, 1, ?, 'OPEN', ?)", old.UnixMilli(), old)
	must("INSERT INTO trader_fills (trader_id, order_id, exchange_order_id, exchange_trade_id, symbol, side, price, quantity, quote_quantity, commission, commission_asset, created_at) VALUES ('t-ret', 1, 'eo', 'et', 'MNQ', 'buy', 1, 1, 1, 0, 'USD', ?)", old)
	must("INSERT INTO plans (plan_id, strategy_id, trade_date, session, version, lifecycle, doc, created_at) VALUES ('p-old','t-ret','2026-01-01','NY', 1, 'active', '{}', ?)", old)

	count := func(table string) int64 {
		var n int64
		if err := db.Table(table).Count(&n).Error; err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		return n
	}
	before := map[string]int64{
		"decision_records":        count("decision_records"),
		"trader_equity_snapshots": count("trader_equity_snapshots"),
		"nt8_order_snapshots":     count("nt8_order_snapshots"),
		"level_stats":             count("level_stats"),
		"trader_positions":        count("trader_positions"),
		"trader_fills":            count("trader_fills"),
		"nt8_exit_receipts":       count("nt8_exit_receipts"),
		"plans":                   count("plans"),
	}

	cfg := Config{DecisionDays: 30, EquityDays: 30, NT8SnapshotDays: 30, LevelStatsDays: 30}
	r := RunDaily(st, time.Now(), cfg)
	if len(r.Errs) > 0 {
		t.Fatalf("RunDaily errors: %v", r.Errs)
	}

	if count("decision_records") != 0 || count("trader_equity_snapshots") != 0 ||
		count("nt8_order_snapshots") != 0 || count("level_stats") != 0 {
		t.Fatalf("prunable tables not pruned: decision=%d equity=%d nt8=%d level=%d",
			count("decision_records"), count("trader_equity_snapshots"),
			count("nt8_order_snapshots"), count("level_stats"))
	}
	for _, protected := range []string{"trader_positions", "trader_fills", "nt8_exit_receipts", "plans"} {
		if count(protected) != before[protected] {
			t.Fatalf("PROTECTED table %s changed: %d -> %d", protected, before[protected], count(protected))
		}
	}
	if r.DecisionPruned < 1 || r.EquityPruned < 1 || r.NT8Pruned < 1 || r.LevelStatsPruned < 1 {
		t.Fatalf("prune report must name what it removed: %+v", r)
	}
}

// TestRunDailyOffPrunesNothing pins the owner ruling: all knobs OFF (0) must
// delete nothing.
func TestRunDailyOffPrunesNothing(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "ret.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer st.Close()
	db := st.GormDB()
	if err := st.LevelStats().Migrate(); err != nil {
		t.Fatalf("level_stats migrate: %v", err)
	}
	old := time.Now().AddDate(0, 0, -400)
	if err := db.Exec("INSERT INTO level_stats (trader_id, session_day, price, label, kind, grade, role, family, created_at) VALUES ('t','2026-01-01', 1,'l','k','g','r','f', ?)", old).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	if r := RunDaily(st, time.Now(), Config{}); r.LevelStatsPruned != 0 {
		t.Fatalf("OFF must prune nothing, pruned %d", r.LevelStatsPruned)
	}
	var n int64
	_ = db.Table("level_stats").Count(&n).Error
	if n != 1 {
		t.Fatalf("OFF deleted rows: %d", n)
	}
}
