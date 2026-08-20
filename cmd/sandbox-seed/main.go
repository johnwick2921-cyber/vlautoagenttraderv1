// cmd/sandbox-seed — populate an ISOLATED sandbox database with realistic
// day-plan content so every panel has something to show and every button has
// something to act on.
//
// It refuses to run against anything that looks like the live database, and it
// only ever writes to the DB handed to it via -db (default data/sandbox.db).
//
//	go run ./cmd/sandbox-seed -db data/sandbox.db -trader <trader_id> [-symbol MNQ]
//
// Idempotent: re-running replaces the seeded day-plan rows (scripts/sandbox-reset.sh
// deletes the file first for a truly clean slate).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/store"
)

func main() {
	dbPath := flag.String("db", "data/sandbox.db", "sandbox database path")
	traderID := flag.String("trader", "", "trader_id to attach the seed to (required)")
	symbol := flag.String("symbol", "MNQ", "instrument symbol")
	flag.Parse()

	// GUARD: never seed the live database.
	if strings.Contains(*dbPath, "data.db") || !strings.Contains(*dbPath, "sandbox") {
		fmt.Fprintf(os.Stderr, "refusing to seed %q — the path must be a sandbox db\n", *dbPath)
		os.Exit(2)
	}
	if strings.TrimSpace(*traderID) == "" {
		fmt.Fprintln(os.Stderr, "-trader is required")
		os.Exit(2)
	}

	st, err := store.New(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open %s: %v\n", *dbPath, err)
		os.Exit(1)
	}
	defer st.Close()
	defer st.Plan().Close()

	now := time.Now()
	loc, _ := time.LoadLocation("America/Chicago")
	if loc == nil {
		loc = time.UTC
	}
	tradeDate := now.In(loc).Format("2006-01-02")
	sess := kernel.SessionNY

	// ── the session registry: NY active all day so the card is always visible,
	// ASIA present-but-disabled so the owner can see the night/disabled tab.
	reg := `{"sessions":[
	  {"name":"NY","window_start_ct":"00:00","window_end_ct":"23:59","read_ct":"08:25","flat_ct":"14:45",
	   "killzones":[{"name":"ny_am","start_ct":"08:30","end_ct":"11:00"},{"name":"ny_pm","start_ct":"13:00","end_ct":"14:45"}],
	   "enabled":true},
	  {"name":"ASIA","window_start_ct":"17:00","window_end_ct":"02:00","read_ct":"16:55","flat_ct":"02:00","enabled":false},
	  {"name":"LONDON","window_start_ct":"02:00","window_end_ct":"08:30","read_ct":"01:55","flat_ct":"08:30","enabled":false}]}`
	must("session registry", st.SetSystemConfig(kernel.SessionRegistryConfigKey, reg))

	// ── PLAN v1 → v2 (so version chips + history have something to show) ────────
	v1 := kernel.PlanDoc{
		Reasoning: "Balance under PDH. Overnight swept ONL and reclaimed; value migrating up.",
		Bias:      kernel.PlanBias{Direction: "long", Conviction: "medium", FlipCondition: "flips short on 2×5m closes below 30148"},
		Levels: []kernel.PlanLevel{
			{Price: 30288.00, Label: "PDH", Grade: "A", Instruction: "target only — trim into it, never chase"},
			{Price: 30246.25, Label: "RTH-H", Grade: "B", Instruction: "supply — long only on acceptance above (2×5m)"},
			{Price: 30214.25, Label: "nPOC·Tue", Grade: "A", Instruction: "magnet — expect touch; no entries at it"},
		},
		Scenarios: []kernel.PlanScenario{
			{ID: "S1", Trigger: "sweep 30160–52 then reclaim", Condition: "sweep_reclaim", Direction: "long",
				TargetChain: []float64{30214.25, 30288}, Invalid: "2×5m closes < 30148", Quality: "A+"},
		},
		NoTrade:        []string{"first 5 min", "lunch 12:00–13:30 CT"},
		DeathCondition: "2×5m closes < 30148",
		DayType:        "balance",
	}
	v2 := v1
	v2.Reasoning = "Re-read after the demand shelf held twice. Adding the supply-acceptance play and the bear alt."
	v2.DayType = "balance→trend?"
	v2.Levels = []kernel.PlanLevel{
		{Price: 30288.00, Label: "PDH", Grade: "A", Instruction: "target only — trim into it, never chase"},
		{Price: 30246.25, Label: "RTH-H", Grade: "B", Instruction: "fade first touch"}, // AI half of the ⚡ pair
		{Price: 30214.25, Label: "nPOC·Tue", Grade: "A", Instruction: "magnet — expect touch; no entries at it"},
		{Price: 30188.50, Label: "ONL", Grade: "B", Instruction: "overnight low — sweep candidate"},
		{Price: 30156.00, Label: "D-4h", Grade: "A", Instruction: "demand — the long POI: sweep + reclaim = entry"},
		{Price: 30124.75, Label: "EQH", Grade: "C", Instruction: "equal highs — liquidity only, confluence"},
		{Price: 30246.00, Label: "watch reclaim", Grade: "B", Instruction: "watch reclaim"}, // owner half of the ⚡ pair
		{Price: 30100.00, Label: "RN", Grade: "C", Instruction: "round number — below = bear day"},
		{Price: 30061.25, Label: "PDL", Grade: "B", Instruction: "prior-day low — last support before the flush"},
	}
	v2.Scenarios = []kernel.PlanScenario{
		{ID: "S1", Trigger: "sweep 30160–52 then reclaim", Condition: "sweep_reclaim", Direction: "long",
			TargetChain: []float64{30214.25, 30288}, Invalid: "2×5m closes < 30148", Quality: "A+"},
		{ID: "S2", Trigger: "acceptance above 30246 (2×5m)", Condition: "acceptance", Direction: "long",
			TargetChain: []float64{30288, 30330}, Invalid: "close back inside the zone", Quality: "B"},
		{ID: "S3", Trigger: "acceptance below 30148", Condition: "acceptance", Direction: "short",
			TargetChain: []float64{30100, 30061.25}, Invalid: "reclaim 30160", Quality: "B"},
		{ID: "S4", Trigger: "reject the EQH shelf", Condition: "reject", Direction: "short",
			TargetChain: []float64{30100}, Invalid: "acceptance above 30130", Quality: "B"},
	}
	v2.NoTrade = []string{
		"first 5 min", "lunch 12:00–13:30 CT",
		"🔴 red-news blackout 13:15–13:45 CT (FOMC minutes)",
	}

	planID := store.MakePlanIDForTrader(*traderID, tradeDate, sess)
	for i, doc := range []kernel.PlanDoc{v1, v2} {
		js, _ := json.Marshal(doc)
		trigger := "NY_scheduled_read"
		if i == 1 {
			trigger = "death_condition_replan"
		}
		if _, err := st.Plan().AppendPlan(&store.PlanDB{
			PlanID: planID, StrategyID: *traderID, TradeDate: tradeDate, Session: sess,
			TriggerReason: trigger, Lifecycle: "active", ModelID: "deepseek-v4-pro",
			PromptHash: fmt.Sprintf("sbx%03d", i+1), Doc: string(js),
			IndicatorsBlock: "### 1d\nEMA20: [29071.9000]\nRSI14: [58.92, 61.30]\nATR14: 595.8473",
			AIConfigHash:    "5f2c9a71bd0e4c33",
		}); err != nil {
			fmt.Fprintf(os.Stderr, "append plan v%d: %v\n", i+1, err)
		}
	}

	// ── OWNER LEVELS (origin=OWNER → 👤 + 📝 note + [S-tag] on the card) ──────
	// Includes the ⚡ CONFLICT PAIR: an owner level at 30246 ("watch reclaim")
	// against the AI's 30246.25 ("fade first touch") — opposing instructions at
	// the same price, so the chip fires and the AI row ghosts. And a STICKY level
	// carried from an earlier session (owner levels persist across reads).
	ownerLevels := []struct {
		price       float64
		label, note string
		tag         string
		ageMin      int
	}{
		{30156.00, "D-4h", "strong 4h OB — tôi tin zone này", "S1", 90},
		{30288.00, "PDH", "PDH magnet — trim, never chase", "S2", 75},
		{30100.00, "RN", "", "", 60},
		{30246.00, "watch reclaim", "my read: reclaim, not a fade", "S2", 45}, // ⚡ vs AI 30246.25
		{30061.25, "PDL·sticky", "carried from yesterday — still my line", "S3", 1500},
	}
	for _, ol := range ownerLevels {
		must("owner level", st.OwnerLevel().Save(&store.OwnerLevelDB{
			Symbol: *symbol, Price: ol.price, Label: ol.label, Note: ol.note,
			ScenarioTag: ol.tag, CreatedAt: now.Add(-time.Duration(ol.ageMin) * time.Minute).UnixMilli(),
		}))
	}

	// ── LEVEL STATE: a BURNED level (consumed in an earlier session) + a tested one
	for _, ls := range []struct {
		price float64
		typ   string
		fresh string
		cons  bool
		tests int
	}{
		{30100.00, "round-number", store.FreshnessDone, true, 3}, // burned
		{30246.25, "PDH/PDL/PDC", store.FreshnessB, false, 1},    // tested once
	} {
		bin := kernel.LevelBinIndex(ls.price)
		key := store.MakeLevelKey("", *symbol, ls.typ, "", bin)
		_ = st.LevelState().EnsureLevel(&store.LevelStateDB{
			Symbol: *symbol, LevelType: ls.typ, BinIndex: bin, Price: ls.price, Freshness: ls.fresh,
		})
		if ls.cons {
			_ = st.LevelState().MarkConsumed(key)
		}
		for i := 0; i < ls.tests; i++ {
			_ = st.LevelState().RecordTest(key)
		}
	}

	// ── ALERTS: P0 unacked (banner) + P1s + P2 ────────────────────────────────
	alerts := []struct{ level, kind, title, body string }{
		{"P0", "fill", "Filled LONG MNQ @ 30162.25", "S1 sweep+reclaim — 1 contract"},
		{"P0", "halt", "🛑 Consecutive-loss halt", "2 losing trades this session — entries blocked until the next session"},
		{"P1", "armed", "NY plan v2 armed", "model deepseek-v4-pro"},
		{"P1", "level-burned", "Burned level re-touched: RN 30100", "consumed in a prior session — no fresh play"},
		{"P1", "triggered", "S1 triggered", "sweep 30160–52 reclaimed at 08:52 CT"},
		{"P2", "digest", "Session digest written", "NY 2026-08-16"},
		{"P2", "digest", "Daily roll-up written", "3 sessions · +$204.00"},
	}
	for i, a := range alerts {
		_, _ = st.Alert().Emit(&store.AlertDB{
			TraderID: *traderID, Level: a.level, Kind: a.kind,
			EventID: fmt.Sprintf("sbx:%s:%d", a.kind, i), Title: a.title, Body: a.body,
			CreatedAt: now.Add(-time.Duration(60-i*7) * time.Minute).Unix(),
		})
	}
	// ack the P2s so the badge shows a realistic unacked count
	if rows, _ := st.Alert().List(*traderID, 50); len(rows) > 0 {
		for _, r := range rows {
			if r.Level == "P2" {
				_ = st.Alert().Ack(r.ID)
			}
		}
	}

	// ── CLOSED TRADES with MAE/MFE + adherence grades ─────────────────────────
	trades := []struct {
		side             string
		entry, exit, pnl float64
		mae, mfe         float64
		grade, scenario  string
		matched          bool
		minsAgo          int
	}{
		{"LONG", 30162.25, 30214.25, 104.0, -8.5, 62.0, "A", "S1", true, 210},
		{"LONG", 30251.00, 30239.50, -23.0, -18.0, 6.5, "C", "S2", true, 140},
		{"SHORT", 30146.00, 30158.75, -25.5, -14.0, 3.0, "F", "off-plan", false, 70},
	}
	for _, tr := range trades {
		entryMs := now.Add(-time.Duration(tr.minsAgo) * time.Minute).UnixMilli()
		pos := &store.TraderPosition{
			TraderID: *traderID, Symbol: *symbol, Side: tr.side,
			EntryPrice: tr.entry, EntryQuantity: 1, EntryTime: entryMs,
			CreatedAt: entryMs, UpdatedAt: entryMs,
		}
		if err := st.Position().Create(pos); err != nil {
			continue
		}
		_, _ = st.Position().ClosePosition(pos.ID, tr.exit, "sbx", tr.pnl, 0, "target")
		_ = st.Position().UpdateExcursion(pos.ID, tr.mae, tr.mfe)
		_ = st.Position().SetAdherence(pos.ID, tr.grade)
		_ = st.Position().SetPlanLink(pos.ID, 2, tr.scenario, tr.matched, "")
	}

	// ── DIGESTS: this session + daily + a tapered week ────────────────────────
	_, _ = st.Digest().SaveIfAbsent(&store.DigestDB{
		Symbol: *symbol, TradeDate: tradeDate, Session: sess, Kind: "session",
		Text:      "NY " + tradeDate + " — 3 entries · +$55.50 · S1 paid, S2 chopped, one off-plan short.",
		CreatedAt: now.UnixMilli(),
	})
	for i := 1; i <= 5; i++ {
		d := now.AddDate(0, 0, -i).In(loc).Format("2006-01-02")
		txt := fmt.Sprintf("%s — 2 sessions · %+.2f · balance day, PDH respected.", d, float64(60-i*35))
		if i > 3 {
			txt = fmt.Sprintf("%s — %+.2f (one-liner)", d, float64(40-i*20))
		}
		_, _ = st.Digest().SaveIfAbsent(&store.DigestDB{
			Symbol: *symbol, TradeDate: d, Kind: "daily", Text: txt,
			CreatedAt: now.AddDate(0, 0, -i).UnixMilli(),
		})
	}

	// ── CALENDAR: this week's slices incl. a T1 (red) blackout today ──────────
	type ev struct {
		Time     time.Time `json:"time"`
		Currency string    `json:"currency"`
		Title    string    `json:"title"`
		Impact   string    `json:"impact"`
	}
	for i := 0; i < 5; i++ {
		d := now.AddDate(0, 0, i-1)
		key := d.In(loc).Format("2006-01-02")
		evs := []ev{{Time: time.Date(d.Year(), d.Month(), d.Day(), 18, 30, 0, 0, time.UTC), Currency: "USD", Title: "Unemployment Claims", Impact: "T2"}}
		if i == 1 {
			evs = append(evs, ev{Time: time.Date(d.Year(), d.Month(), d.Day(), 19, 0, 0, 0, time.UTC), Currency: "USD", Title: "FOMC Minutes", Impact: "T1"})
		}
		js, _ := json.Marshal(evs)
		_, _ = st.Calendar().SaveSliceIfAbsent(&store.CalendarSliceDB{
			TradeDate: key, Source: "sandbox", EventsJSON: string(js), CreatedAt: now.UnixMilli(),
		})
	}

	// ── SCENARIO LIVE STATUS (the four dot states) ────────────────────────────
	must("scenario status", st.SetSystemConfig(store.ScenarioStatusKey(*traderID, planID),
		`{"S1":"triggered","S2":"armed","S3":"waiting","S4":"invalidated"}`))

	fmt.Printf("✅ sandbox seeded: %s\n   plan %s v1+v2 · 8 levels · 3 scenarios · owner level · %d alerts · 3 graded trades · digests · calendar · level-state\n",
		*dbPath, planID, len(alerts))
}

func must(what string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed %s: %v\n", what, err)
	}
}
