package trader

// ═══════════════════════════════════════════════════════════════════════════
// DEMO PLAN SEED (2026-08-16). Puts a realistic, fully-populated day plan on
// the LIVE dashboard so the owner can click every control before Monday.
//
// NO paid API call: the plan JSON is handcrafted but goes through the SAME
// kernel.ParsePlanDoc/ValidatePlanDoc path a real planner read uses, so it is
// schema-real — anything the validator would reject, this rejects too.
//
// ISOLATION — the demo key is (trade_date 2026-08-16, session NY):
//   • The SCHEDULER can never produce it. maybeRunSessionReads requires
//     kernel.IsCMEOpen(now) AND inSessionReadWindow(08:25→15:00 CT). On a
//     SUNDAY, IsCMEOpen is false until 17:00 (cme_calendar.go:28-29) and by
//     17:00 the NY window is long closed. The two conditions cannot both hold.
//   • The EXECUTOR cannot act on it. runCycle's FIRST gate is
//     cmeSessionClosedSkip() (auto_trader_loop.go), which returns before any
//     context build, AI call or order path for the whole Sunday. After 17:00 CT
//     the active session is ASIA, which is DISABLED, so ActivePlanProvider
//     returns nil.
//   • It self-expires. At Monday 00:00 CT the trade date rolls to 2026-08-17
//     and the demo row is invisible to both the card and the executor. Monday's
//     real key (2026-08-17:NY) is never written here.
//
// GUARDED: skips unless NOFX_DEMO_SEED=1. Back up data/data.db first.
// Undo: docs/superpowers/reports/2026-08-16-demo-plan-seed.md has the cleanup.
// ═══════════════════════════════════════════════════════════════════════════

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

const (
	demoTradeDate = "2026-08-16" // SUNDAY — unreachable by the scheduler
	demoSession   = "NY"         // the only ENABLED session, so the card renders it
	demoSymbol    = "MNQ"
)

// demoPlanV1 is the first version: realistic MNQ structure off the 2026-08-14
// session (PDH 30287.25 / PDL 30025.00 / PDC 30154.75 / RTH-H 30280.75), mixed
// provenance and grades, 3 scenarios in the formal grammar, a T1 red-news
// blackout in no_trade, and one owner-origin level.
//
// NOTE on the 👤/📝 glyphs: they are embedded in the LEVEL LABEL on purpose.
// GET /api/plan/today does not emit the `origin`/`note` fields ZoneTable's real
// markers key off (acceptance-gate finding F-5 — dormant adornments), so the
// label text is the only way to show owner provenance on the card today.
const demoPlanV1 = `{
  "reasoning": "Friday closed at 30154.75 mid-balance after a 30025.00 sweep held and the 08:06 CT Globex spike to 30287.25 was rejected. Value is stacked 30075-30200; the auction is rotational until one edge is accepted through. Long-biased while 30025.00 holds: it is the low of the entire day and the only level tested from both sides. Fade the extremes, buy the reclaim, and stand down through the FOMC minutes.",
  "bias": {"direction": "long", "conviction": "medium", "flip_condition": "2x5m close below 30025.00 flips short toward 29980.00"},
  "levels": [
    {"price": 30287.25, "label": "PDH", "grade": "A", "instruction": "fade"},
    {"price": 30280.75, "label": "RTH-H", "grade": "B", "instruction": "fade"},
    {"price": 30200.00, "label": "RN", "grade": "B", "instruction": "reclaim-long"},
    {"price": 30154.75, "label": "PDC", "grade": "A", "instruction": "hold"},
    {"price": 30120.00, "label": "👤 D-zone 📝 [S1]", "grade": "A", "instruction": "owner: first bid, scale in"},
    {"price": 30075.50, "label": "nPOC·Thu", "grade": "C", "instruction": "reclaim-long"},
    {"price": 30025.00, "label": "PDL", "grade": "A", "instruction": "sweep-reclaim-long"},
    {"price": 29980.00, "label": "ONL", "grade": "C", "instruction": "fade"}
  ],
  "scenarios": [
    {"id": "S1", "trigger": "sweep PDL 30025.00 and reclaim it on a 5m close, holding the 30120.00 owner zone as the first bid", "condition": "sweep_reclaim", "direction": "long", "target_chain": [30075.50, 30154.75, 30200.00], "invalid": "2x5m close below 30025.00", "quality": "A"},
    {"id": "S2", "trigger": "rally into 30280.75-30287.25 and reject it with a 5m RSI roll", "condition": "reject", "direction": "short", "target_chain": [30200.00, 30154.75], "invalid": "2x5m close above 30287.25", "quality": "B"},
    {"id": "S3", "trigger": "acceptance above RN 30200.00 then a held retest of it as support", "condition": "breakout_retest", "direction": "long", "target_chain": [30280.75, 30287.25], "invalid": "5m close back below 30200.00", "quality": "B"}
  ],
  "no_trade": [
    "first 5m (08:30-08:35 CT)",
    "lunch 12:00-13:30 CT",
    "🔴 FOMC Meeting Minutes 13:00 CT ±15m — HARD no-trade (red news)"
  ],
  "death_condition": "2x5m acceptance below 30025.00 or above 30287.25 voids this rotational thesis and demands trend-breakout treatment",
  "day_type": "balance"
}`

// demoPlanV2 is the re-plan the owner sees as v2: the 30025.00 sweep resolved,
// so conviction rises and the S1 target chain extends. Seeding two versions
// lights up the version chip and the /plan/history panel.
const demoPlanV2 = `{
  "reasoning": "RE-PLAN. The 30025.00 sweep completed and reclaimed on the 5m, so S1 is live and the downside tail is spent. Raising conviction to high and extending the chain through 30200.00 into the 30280.75 shelf. The FOMC blackout still stands and takes precedence over any setup.",
  "bias": {"direction": "long", "conviction": "high", "flip_condition": "2x5m close back below 30025.00 voids the reclaim and flips short"},
  "levels": [
    {"price": 30287.25, "label": "PDH", "grade": "A", "instruction": "fade"},
    {"price": 30280.75, "label": "RTH-H", "grade": "A", "instruction": "target / fade"},
    {"price": 30200.00, "label": "RN", "grade": "A", "instruction": "reclaim-long"},
    {"price": 30154.75, "label": "PDC", "grade": "B", "instruction": "hold"},
    {"price": 30120.00, "label": "👤 D-zone 📝 [S1]", "grade": "A", "instruction": "owner: first bid, scale in"},
    {"price": 30075.50, "label": "nPOC·Thu", "grade": "B", "instruction": "hold"},
    {"price": 30025.00, "label": "PDL", "grade": "A", "instruction": "defend — thesis floor"},
    {"price": 29980.00, "label": "ONL", "grade": "C", "instruction": "fade"}
  ],
  "scenarios": [
    {"id": "S1", "trigger": "30025.00 sweep reclaimed — hold 30120.00 and continue up", "condition": "sweep_reclaim", "direction": "long", "target_chain": [30154.75, 30200.00, 30280.75], "invalid": "2x5m close below 30025.00", "quality": "A"},
    {"id": "S2", "trigger": "rally into 30280.75-30287.25 and reject it with a 5m RSI roll", "condition": "reject", "direction": "short", "target_chain": [30200.00, 30154.75], "invalid": "2x5m close above 30287.25", "quality": "B"},
    {"id": "S3", "trigger": "acceptance above RN 30200.00 then a held retest of it as support", "condition": "breakout_retest", "direction": "long", "target_chain": [30280.75, 30287.25], "invalid": "5m close back below 30200.00", "quality": "A"}
  ],
  "no_trade": [
    "first 5m (08:30-08:35 CT)",
    "lunch 12:00-13:30 CT",
    "🔴 FOMC Meeting Minutes 13:00 CT ±15m — HARD no-trade (red news)"
  ],
  "death_condition": "2x5m acceptance below 30025.00 voids the reclaim thesis outright",
  "day_type": "balance"
}`

func TestDemoSeed(t *testing.T) {
	if os.Getenv("NOFX_DEMO_SEED") != "1" {
		t.Skip("demo seeder is armed only with NOFX_DEMO_SEED=1 (writes to the live DB)")
	}
	traderID := os.Getenv("NOFX_DEMO_TRADER")
	if traderID == "" {
		t.Fatal("NOFX_DEMO_TRADER must be the live trader id the owner will look at")
	}
	dbPath := os.Getenv("NOFX_DEMO_DB")
	if dbPath == "" {
		dbPath = filepath.Join("..", "data", "data.db")
	}

	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("open live store: %v", err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })

	// ── guard: never overwrite Monday, never collide with a real row ──
	if monday, _ := st.Plan().GetLatestPlanForSession("2026-08-17", "NY"); monday != nil {
		t.Fatalf("REFUSING TO SEED: a real 2026-08-17:NY plan already exists (v%d) — the demo must not run beside it", monday.Version)
	}

	// ── idempotent: a re-run must not stack v3/v4 onto the demo key ──
	if existing, _ := st.Plan().GetLatestPlanForSession(demoTradeDate, demoSession); existing != nil {
		t.Logf("demo plan already present at v%d (trigger %q) — skipping plan writes", existing.Version, existing.TriggerReason)
	} else {
		seedDemoPlans(t, st, traderID)
	}

	latest, err := st.Plan().GetLatestPlanForSession(demoTradeDate, demoSession)
	if err != nil || latest == nil {
		t.Fatalf("read back demo plan: %v", err)
	}
	seedDemoExtras(t, st, traderID, latest)

	// ── receipts ──
	t.Logf("── DEMO SEEDED ── key=(%s, %s) latest=v%d", demoTradeDate, demoSession, latest.Version)
	t.Logf("   card renders while the NY window is active: 08:30–15:00 CT today")
	if monday, _ := st.Plan().GetLatestPlanForSession("2026-08-17", "NY"); monday == nil {
		t.Log("   ✓ Monday 2026-08-17:NY — still EMPTY (untouched)")
	} else {
		t.Error("   ✗ a 2026-08-17:NY row exists — investigate before Monday")
	}
	fmt.Println("demo-seed complete")
}

func seedDemoPlans(t *testing.T, st *store.Store, traderID string) {
	t.Helper()
	for i, raw := range []string{demoPlanV1, demoPlanV2} {
		doc, perr := kernel.ParsePlanDoc(raw) // ParsePlanDoc runs ValidatePlanDoc
		if perr != nil {
			t.Fatalf("demo plan v%d rejected by the real schema validator: %v", i+1, perr)
		}
		docJSON, _ := json.Marshal(doc)
		version, aerr := st.Plan().AppendPlan(&store.PlanDB{
			PlanID:        store.MakePlanID(demoTradeDate, demoSession),
			StrategyID:    traderID,
			TradeDate:     demoTradeDate,
			Session:       demoSession,
			TriggerReason: "demo_seed",
			Lifecycle:     "active",
			ModelID:       "demo-seed (no API call)",
			PromptHash:    "demoseed0001",
			Doc:           string(docJSON),
		})
		if aerr != nil {
			t.Fatalf("append demo plan v%d: %v", i+1, aerr)
		}
		t.Logf("✓ plan %s %s v%d written (%d levels, %d scenarios, %d no-trade)",
			demoTradeDate, demoSession, version, len(doc.Levels), len(doc.Scenarios), len(doc.NoTrade))
	}

}

func seedDemoExtras(t *testing.T, st *store.Store, traderID string, latest *store.PlanDB) {
	t.Helper()

	// ── an owner overlay on top of v2, so overlay_count > 0 and the edit /
	// conflict path has real content to show (RFC-6902, same applier the API uses)
	patch := `[{"op":"replace","path":"/levels/4/instruction","value":"owner: first bid — size up here"}]`
	if _, oerr := st.Plan().AppendOverlay(&store.PlanOverlayDB{
		PlanID:      latest.PlanID,
		PlanVersion: latest.Version,
		Origin:      "owner",
		Patch:       patch,
		CreatedAt:   time.Now(),
	}); oerr != nil {
		t.Logf("⚠ overlay seed failed (non-fatal): %v", oerr)
	} else {
		t.Log("✓ owner overlay appended to v2 (overlay_count → 1)")
	}

	// ── 3a. alert center: one unacked P0 (banner + toast), one P1, one P2 ──
	alerts := []struct{ level, kind, event, title, body string }{
		{"P0", "read-fail", "demo:p0:" + demoTradeDate,
			"DEMO — planner fail-closed", "Seeded P0: this is the unacked banner. Click Ack to clear it."},
		{"P1", "armed", "demo:p1:" + demoTradeDate,
			"DEMO — NY plan v2 armed", "Seeded P1: plan re-planned after the 30025.00 sweep reclaimed."},
		{"P2", "level-burned", "demo:p2:" + demoTradeDate,
			"DEMO — nPOC·Thu consumed", "Seeded P2: digest-level note, collapses into the count line."},
	}
	for _, a := range alerts {
		wrote, aerr := st.Alert().Emit(&store.AlertDB{
			TraderID: traderID, Level: a.level, EventID: a.event,
			Kind: a.kind, Title: a.title, Body: a.body,
			CreatedAt: time.Now().UnixMilli(),
		})
		if aerr != nil {
			t.Logf("⚠ alert %s seed failed (non-fatal): %v", a.level, aerr)
			continue
		}
		t.Logf("✓ alert %s seeded (new=%v)", a.level, wrote)
	}

	// ── 3b. sticky owner level (feeds the planner input's 👤 row + edit sheet) ──
	ownerLvl := &store.OwnerLevelDB{
		Symbol: demoSymbol, Price: 30120.00,
		Label: "DEMO D-zone", Note: "owner demo level — delete with the cleanup",
		ScenarioTag: "S1", CreatedAt: time.Now().Unix(),
	}
	if oerr := st.OwnerLevel().Save(ownerLvl); oerr != nil {
		t.Logf("⚠ owner level seed failed (non-fatal): %v", oerr)
	} else {
		t.Logf("✓ owner level seeded id=%d @30120.00 (👤 + 📝 + [S1])", ownerLvl.ID)
	}

	// ── 3c. a completed, graded trade so the review path renders ──
	// Written straight through GORM as a historical row, not a live position.
	// Column types matter here: id is an autoincrement INTEGER (omit it),
	// entry/exit/created/updated are INTEGER unix-millis, and side/status are
	// stored UPPERCASE ("LONG"/"CLOSED") like every real row.
	var already int64
	st.GormDB().Table("trader_positions").Where("source = ?", "demo_seed").Count(&already)
	if already > 0 {
		t.Logf("demo trade already present (%d row) — skipping", already)
		return
	}
	now := time.Now()
	entry := now.Add(-3 * time.Hour)
	exit := now.Add(-90 * time.Minute)
	demoPos := map[string]any{
		"trader_id":         traderID,
		"exchange_type":     "ninjatrader",
		"symbol":            demoSymbol,
		"side":              "LONG",
		"entry_quantity":    1.0,
		"quantity":          0.0,
		"entry_price":       30040.25,
		"entry_time":        entry.UnixMilli(),
		"exit_price":        30152.50,
		"exit_time":         exit.UnixMilli(),
		"realized_pnl":      224.50, // (30152.50-30040.25) × $2/pt × 1 contract
		"fee":               1.24,
		"leverage":          1,
		"status":            "CLOSED",
		"close_reason":      "take_profit",
		"source":            "demo_seed",
		"account":           "Sim101",
		"mae":               -18.75,
		"mfe":               126.50,
		"entry_confidence":  78,
		"plan_version":      latest.Version,
		"cited_scenario_id": "S1",
		"plan_matched":      true,
		"adherence_grade":   "A",
		"created_at":        entry.UnixMilli(),
		"updated_at":        exit.UnixMilli(),
	}
	if terr := st.GormDB().Table("trader_positions").Create(demoPos).Error; terr != nil {
		t.Errorf("demo trade seed FAILED: %v", terr)
	} else {
		t.Log("✓ demo trade seeded: S1 LONG 30040.25→30152.50, +$224.50, MAE -18.75 / MFE +126.50, adherence A")
	}
}
