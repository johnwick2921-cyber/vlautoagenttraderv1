package trader

import (
	"sort"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"vl/store"
	"vl/store/sqlitedriver"
)

// TestMentorSeedAtStart is the P0 splice call-site test (CTO 1791030462901):
// a COLD store seeds with nothing and the placement gate refuses with the
// NAMED source; a SEEDED store passes and the per-source depth seam serves the
// seed's own numbers. Mutant: removing the seam wiring in mentorSeedAtStart
// makes the seeded case refuse (n/a) → RED.
func TestMentorSeedAtStart(t *testing.T) {
	t.Run("cold store refuses with the named source", func(t *testing.T) {
		st := mentorSeedStore(t)
		at := &AutoTrader{
			id: "t-seed-cold",
			config: AutoTraderConfig{
				StrategyConfig: &store.StrategyConfig{
					RiskControl: store.RiskControlConfig{MentorMode: true},
				},
			},
			store: st,
		}
		wireMentorPlacementSeams(t)
		at.mentorSeedAtStart()

		if ms := at.mentorEval.SourcesMissing(); len(ms) == 0 {
			t.Fatalf("a cold store must leave the evaluator seeded-but-missing, got none")
		}
		missing := strings.Join(at.mentorSourcesMissing(), ", ")
		for _, want := range []string{
			"4h EMA34 (0/102)", "1m EMA34 (0/102)",
			"1h level set (0/2)", "today session (0/1)", "closed 15m (0/1)",
		} {
			if !textHas(missing, want) {
				t.Fatalf("cold store refusal must name %q: %q", want, missing)
			}
		}
	})

	t.Run("seeded store passes", func(t *testing.T) {
		st := mentorSeedStore(t)
		now := time.Now().UnixMilli()
		bh := store.NewBarHistoryStore(st.GormDB())
		rows1m := mentorSeedBars1m(now)
		rows1h := mentorSeedBars1h(now)
		rows := append(rows1m, rows1h...)
		if err := bh.InsertBars(rows); err != nil {
			t.Fatalf("InsertBars: %v", err)
		}
		// the exact closed 1m count the seed must report (the fixture seam
		// returns 9999 — a mutant that fails to wire the seed's depths keeps
		// that value and must go RED here).
		n1mClosed := 0
		for _, r := range rows1m {
			if r.OpenTimeMs+60_000 <= now {
				n1mClosed++
			}
		}
		at := &AutoTrader{
			id: "t-seed-warm",
			config: AutoTraderConfig{
				StrategyConfig: &store.StrategyConfig{
					RiskControl: store.RiskControlConfig{MentorMode: true},
				},
			},
			store: st,
		}
		wireMentorPlacementSeams(t)
		at.mentorSeedAtStart()

		if ms := at.mentorEval.SourcesMissing(); len(ms) != 0 {
			t.Fatalf("a seeded store must leave no missing source: %v", ms)
		}
		if missing := strings.Join(at.mentorSourcesMissing(), ", "); textHas(missing, "history:") {
			t.Fatalf("a seeded store must not refuse on history depth: %q", missing)
		}
		// the seam serves the seed's own depths (not the test fallback 9999)
		if d, ok := mentorSourceDepth("1m EMA34"); !ok || d != n1mClosed {
			t.Fatalf("1m EMA34 depth from the seed: %d/%v, want %d/true", d, ok, n1mClosed)
		}
		if d, ok := mentorSourceDepth("4h EMA34"); !ok || d < 102 || d > 110 {
			t.Fatalf("4h EMA34 depth from the seed: %d/%v, want 102..110/true", d, ok)
		}
	})
}

// mentorSeedStore builds an in-memory Store with ONLY the bars table (read-only
// seed path; data.db is never involved).
func mentorSeedStore(t *testing.T) *store.Store {
	t.Helper()
	db, err := gorm.Open(sqlitedriver.GormDialector(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm: %v", err)
	}
	if err := store.NewBarHistoryStore(db).Migrate(); err != nil {
		t.Fatalf("bars migrate: %v", err)
	}
	st, err := store.NewFromGorm(db)
	if err != nil {
		t.Fatalf("NewFromGorm: %v", err)
	}
	return st
}

// mentorSeedBars1m builds 200 closed 1m bars ending at now−1m, plus one bar on
// the current CT day (00:00 CT = 05:00 UTC) so the "today session" source is
// present at ANY time of day — the engine skips unclosed bars everywhere
// except the today-session scan, which reads OpenTime only.
func mentorSeedBars1m(now int64) []store.BarHistoryDB {
	const dayMs = int64(24 * 3600_000)
	ctDay := now - ((now - 5*3600_000) % dayMs) // UTC start of the current CT day
	var rows []store.BarHistoryDB
	start := now - 201*60_000
	for k := int64(0); k < 200; k++ {
		o := start + k*60_000
		base := 30000.0 + float64(k)*0.25
		rows = append(rows, store.BarHistoryDB{
			Symbol: "MNQ", TF: "1m", Contract: "MNQ 12-26",
			Source: store.BarSourceLive, OpenTimeMs: o,
			O: base, H: base + 5, L: base - 5, C: base + 2, V: 1,
		})
	}
	extra := ctDay + 5*60_000
	have := false
	for _, r := range rows {
		if r.OpenTimeMs == extra {
			have = true
			break
		}
	}
	if !have {
		rows = append(rows, store.BarHistoryDB{
			Symbol: "MNQ", TF: "1m", Contract: "MNQ 12-26",
			Source: store.BarSourceLive, OpenTimeMs: extra,
			O: 29999, H: 30004, L: 29994, C: 30001, V: 1,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].OpenTimeMs < rows[j].OpenTimeMs })
	return rows
}

// mentorSeedBars1h builds 430 closed 1h bars ending one hour before the current
// 4h bucket start (22:00 UTC = 17:00 CT anchor) — 107 fully closed 4h buckets,
// above the 102 warm-up — with alternating candle colours so the 1H RTH level
// set draws ≥2 candles.
func mentorSeedBars1h(now int64) []store.BarHistoryDB {
	const hourMs = int64(3600_000)
	const bucketMs = int64(4 * 3600_000)
	b0 := now - ((now - 22*hourMs) % bucketMs) // current 4h bucket start
	var rows []store.BarHistoryDB
	for k := int64(1); k <= 430; k++ {
		o := b0 - k*hourMs
		base := 29000.0 - float64(k)*0.5
		green := k%2 == 0
		c := base + 3
		if !green {
			c = base - 3
		}
		rows = append(rows, store.BarHistoryDB{
			Symbol: "MNQ", TF: "1h", Contract: "MNQ 12-26",
			Source: store.BarSourceLive, OpenTimeMs: o,
			O: base, H: base + 10, L: base - 10, C: c, V: 10,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].OpenTimeMs < rows[j].OpenTimeMs })
	return rows
}
