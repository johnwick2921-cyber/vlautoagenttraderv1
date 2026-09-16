package ninjatrader

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// T1 nightly-proof — the retried/once-split writer actually lands rows on a
// temp store (the deployed nightly path used to swallow DB-lock skips with a
// bare continue; the once-runner returns errors instead and the caller retries).
func TestT1RunLevelStatsDayOnceWritesRows(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "ls.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	day := "2026-08-26"
	dayStart := time.Date(2026, 8, 26, 17, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)
	now := dayEnd.Add(time.Hour)

	doc := kernel.PlanDoc{Bias: kernel.PlanBias{Direction: "long"},
		Levels: []kernel.PlanLevel{
			{Price: 29200, Label: "PDC", Grade: "B"},
			{Price: 29250, Label: "ONH", Grade: "A"},
		},
		Scenarios: []kernel.PlanScenario{}, NoTrade: []string{}}
	blob, _ := json.Marshal(doc)
	if _, err := st.Plan().AppendPlan(&store.PlanDB{
		PlanID:    "2026-08-26:NY:8d5c8af5_8ef641a7-815c-4bb5-9798-b070b67d7998_deepseek_1781246265",
		TradeDate: day, Session: "NY", StrategyID: "8d5c8af5_8ef641a7-815c-4bb5-9798-b070b67d7998_deepseek_1781246265",
		Doc: string(blob), Lifecycle: "active", CreatedAt: dayStart,
	}); err != nil {
		t.Fatalf("AppendPlan: %v", err)
	}

	if err := st.BarHistory().Migrate(); err != nil {
		t.Fatalf("bars migrate: %v", err)
	}
	// Seed the persisted 1m window so the outcome evaluation has real klines.
	var seed []store.BarHistoryDB
	for i := int64(0); i < 10; i++ {
		seed = append(seed, store.BarHistoryDB{Contract: "MNQ 09-26", Source: store.BarSourceLive, Symbol: "MNQ", TF: "1m",
			OpenTimeMs: dayStart.UnixMilli() + i*60_000, O: 29200, H: 29205, L: 29195, C: 29202, V: 100})
	}
	if err := st.BarHistory().InsertBars(seed); err != nil {
		t.Fatalf("InsertBars: %v", err)
	}
	if err := st.LevelStats().Migrate(); err != nil {
		t.Fatalf("level_stats migrate: %v", err)
	}
	ls := st.LevelStats()
	n, err := runLevelStatsDayOnce(st, ls, "8d5c8af5_8ef641a7-815c-4bb5-9798-b070b67d7998_deepseek_1781246265",
		day, dayStart.UnixMilli(), dayEnd.UnixMilli(), now.UnixMilli())
	if err != nil {
		t.Fatalf("once-runner returned the skip as an error: %v", err)
	}
	if n < 1 {
		t.Fatalf("evaluated %d rows, want ≥1 (two plan levels)", n)
	}
	// idempotence guard: a second run must not duplicate rows (seen map).
	if _, err := runLevelStatsDayOnce(st, ls, "8d5c8af5_8ef641a7-815c-4bb5-9798-b070b67d7998_deepseek_1781246265",
		day, dayStart.UnixMilli(), dayEnd.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatalf("second run: %v", err)
	}
}
