package trader

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// ── ONE SETUP — THE SEAM, driven end to end on the REAL arm path ─────────────

// oneSetupOff sets the switch OFF on a fixture config — for the pre-existing
// wide-book fixtures (fvg_entry, sweep_reclaim splits, R:R at the seam) whose
// subject is not one-setup. E2 proves OFF is today's book byte for byte.
func oneSetupOff(c *store.StrategyConfig) {
	off := false
	if c.DayPlan == nil {
		c.DayPlan = &store.DayPlanConfig{}
	}
	c.DayPlan.OneSetupEnabled = &off
}

func osOff(c *store.StrategyConfig) { oneSetupOff(c) }

// osMap makes the fixture's S1 level (29490, grade A) the best level near
// price and S2's (29500) a lower-graded one; every scenario permitted unless
// the caller overrides.
func osMap(perm map[string]kernel.FadeVerdict) func(*AutoTrader, *store.Store, string) {
	return func(at *AutoTrader, _ *store.Store, _ string) {
		at.oneSetupFactsForTest = func(now time.Time) oneSetupTestFacts {
			a, b := *structuralTestIdentity(29490, "PDL").ID, *structuralTestIdentity(29500, "ONH").ID
			cands := []kernel.MapCandidate{
				{ID: &a, Identity: kernel.PlanLevel{ID: &a, Price: 29490}, Price: 29490, Names: []string{"PDL"}, Grade: "A", Distance: -5},
				{ID: &b, Identity: kernel.PlanLevel{ID: &b, Price: 29500}, Price: 29500, Names: []string{"ONH"}, Grade: "B", Distance: 5},
			}
			if perm == nil {
				perm = map[string]kernel.FadeVerdict{}
				for _, id := range []string{"S1", "S2", "S3"} {
					perm[id] = kernel.FadeVerdict{Evaluated: true, Permitted: true}
				}
			}
			return oneSetupTestFacts{Candidates: cands, Price: 29495, BandPts: 100, Permission: perm}
		}
	}
}

// E2 — OFF PIN. one_setup_enabled=false reproduces the golden generated at the
// base commit 499e4f83 byte for byte: today's book, both arms, no counters.
func TestOneSetupE2OffKeepsSelectionOffWithStructuralGeometry(t *testing.T) {
	want, err := os.ReadFile(oneSetupGoldenPath)
	if err != nil {
		t.Fatal(err)
	}
	var expected armPathGolden
	if err := json.Unmarshal(want, &expected); err != nil {
		t.Fatal(err)
	}
	expected.Rows[0]["target"], expected.Rows[1]["target"] = 29520.0, 29470.0 // explicit structural-target owner ruling
	want = goldenBytes(t, expected)
	g, _, _, _ := driveOneSetupArmPath(t, osOff, nil)
	if got := goldenBytes(t, g); string(got) != string(want) {
		t.Fatalf("OFF must be byte-identical to the base golden:\n--- base\n%s\n--- OFF\n%s", want, got)
	}
}

// E1 (seam) — ON with S1 at the best level: exactly ONE arm (S1), S2 declined
// with all three verdicts recorded, the record written for the card.
func TestOneSetupE1SeamExactlyOneArms(t *testing.T) {
	g, at, st, _ := driveOneSetupArmPath(t, nil, osMap(nil))
	if len(g.Rows) != 1 || g.Rows[0]["scenario"] != "S1" {
		t.Fatalf("exactly S1 must arm, got %+v", g.Rows)
	}
	if g.Counters[store.OneSetupClassArmable] != "1" || g.Counters[store.OneSetupClassLevel] != "1" {
		t.Fatalf("counters must read armable=1 level=1: %+v", g.Counters)
	}
	// E5 — the armed target is the recorded first obstacle (29520), not the authored 29530.
	if g.Rows[0]["target"] != 29520.0 {
		t.Fatalf("E5: the arm's target must be the scenario's first obstacle 29520, got %v", g.Rows[0]["target"])
	}
	// The record the card reads: every armable scenario carries its verdict.
	rec := findOneSetupRecord(t, st, at.id)
	if rec == nil || len(rec.Scenarios) != 3 {
		t.Fatalf("record must carry all three armable scenarios: %+v", rec)
	}
	if !rec.Scenarios["S1"].Allowed || rec.Scenarios["S2"].Allowed || !strings.HasPrefix(rec.Scenarios["S2"].Level, "level_not_best") {
		t.Fatalf("record verdicts wrong: %+v", rec.Scenarios)
	}
	if rec.Scenarios["S3"].Play != "play_not_reject:sweep_reclaim" {
		t.Fatalf("S3's play verdict must be recorded even though wait_confirm keeps it dormant: %+v", rec.Scenarios["S3"])
	}
	if rec.Scenarios["S1"].Target != "first-distinct-eligible-zone" {
		t.Fatalf("record must name the target choice: %q", rec.Scenarios["S1"].Target)
	}
}

func findOneSetupRecord(t *testing.T, st *store.Store, traderID string) *store.OneSetupRecord {
	t.Helper()
	var rows []struct{ Key, Value string }
	if err := st.GormDB().Raw("SELECT key, value FROM system_config WHERE key LIKE ?", "one_setup:"+traderID+":%").Scan(&rows).Error; err != nil || len(rows) == 0 {
		return nil
	}
	var r store.OneSetupRecord
	if json.Unmarshal([]byte(rows[0].Value), &r) != nil {
		return nil
	}
	return &r
}

// E3 (seam) — permission NULL for S1: nothing arms; declined not_evaluated.
func TestOneSetupE3SeamNullPermissionDeclines(t *testing.T) {
	perm := map[string]kernel.FadeVerdict{"S1": {}, "S2": {Evaluated: true, Permitted: true}, "S3": {Evaluated: true, Permitted: true}}
	g, _, _, _ := driveOneSetupArmPath(t, nil, osMap(perm))
	if len(g.Rows) != 0 {
		t.Fatalf("NULL permission must arm nothing: %+v", g.Rows)
	}
	if g.Counters[store.OneSetupClassNotEvaluated] != "1" {
		t.Fatalf("declined must be counted not_evaluated: %+v", g.Counters)
	}
}

// E5 (floor) — the frozen target zone under the unchanged R:R floor refuses
// with the new geometry reason; the one-setup selection checks are unchanged.
func TestOneSetupE5ObstacleBelowFloorIsTheExistingRefusal(t *testing.T) {
	var d kernel.PlanDoc
	if err := json.Unmarshal([]byte(oneSetupFixtureDoc()), &d); err != nil {
		t.Fatal(err)
	}
	near := 29505.0 // 1.5R from 29490 with a 10-pt stop — under the 2.0 floor
	d.Scenarios[0].Economics.FirstObstacle.Price = &near
	d.Zones.Zones[2].Lo = &near // the actual frozen target now governs admission
	b, _ := json.Marshal(d)
	oneSetupFixtureDocOverride = string(b)
	g, _, _, _ := driveOneSetupArmPath(t, nil, osMap(nil))
	if len(g.Rows) != 0 {
		t.Fatalf("an obstacle target under the floor must not arm: %+v", g.Rows)
	}
	if g.Counters["geometry_rr"] != "1" {
		t.Fatalf("the structural R:R refusal must be recorded as geometry_rr=1: %+v", g.Counters)
	}
}

// E6 — ONE ARM AT A TIME PER PLAN. Two allowed on the same best level: the
// top-ranked (quality A) arms first; the other reads second_setup_waiting;
// after the first goes terminal (by the code's predicate) the second arms.
func TestOneSetupE6OneArmAtATime(t *testing.T) {
	var d kernel.PlanDoc
	if err := json.Unmarshal([]byte(oneSetupFixtureDoc()), &d); err != nil {
		t.Fatal(err)
	}
	// Both rejects long at 29490; S1 is quality B, S2 quality A → S2 ranks first.
	d.Scenarios[0].Quality = "B"
	d.Scenarios[1] = d.Scenarios[0]
	d.Scenarios[1].ID, d.Scenarios[1].Quality = "S2", "A"
	d.Scenarios[1].Arm = &kernel.PlanArmSpec{Enabled: true, Entry: 29490, Stop: 29480, Target: 29530}
	b, _ := json.Marshal(d)
	oneSetupFixtureDocOverride = string(b)
	g, at, st, now := driveOneSetupArmPath(t, nil, osMap(nil))
	if len(g.Rows) != 1 || g.Rows[0]["scenario"] != "S2" {
		t.Fatalf("the top-ranked allowed scenario (S2, quality A) must arm first and alone: %+v", g.Rows)
	}
	if g.Counters[store.OneSetupClassWaiting] != "1" {
		t.Fatalf("S1 must read second_setup_waiting: %+v", g.Counters)
	}
	// S2 goes terminal → the waiting one arms on the next cycle.
	rows, _ := st.ArmedOrders().ListNonTerminal(at.id)
	if err := st.ArmedOrders().SetState(rows[0].ID, "cancelled", "test: terminal"); err != nil {
		t.Fatal(err)
	}
	at.maybeManageArmedOrdersAt(nil, now.Add(time.Second))
	rows, _ = st.ArmedOrders().ListNonTerminal(at.id)
	if len(rows) != 1 || rows[0].Scenario != "S1" {
		t.Fatalf("after S2 is terminal, S1 must arm: %+v", rows)
	}
}

// E7 — RECORD PIN. A declined scenario's episode row carries all three verdicts.
func TestOneSetupE7DeclinedScenarioEpisodeCarriesThreeVerdicts(t *testing.T) {
	var epID uint
	hook := func(at *AutoTrader, st *store.Store, pid string) {
		osMap(nil)(at, st, pid)
		row := &store.TouchOutcomeRow{TraderID: at.id, Symbol: at.futuresSymbol(), LevelPrice: 29500.5, LevelKind: "ONH",
			CandidateSeated: true, PlanID: pid, PlanVersion: 1, Session: "X", Ordinal: 1,
			OpenedAtMs: time.Now().Add(-10 * time.Minute).UnixMilli(), Outcome: "hold", Validity: store.ValidityValid, EntrySide: "below"}
		if err := st.TouchOutcomes().SaveOutcome(row); err != nil {
			t.Fatal(err)
		}
		epID = row.ID
	}
	_, _, st, _ := driveOneSetupArmPath(t, nil, hook)
	var r store.TouchOutcomeRow
	if err := st.GormDB().First(&r, epID).Error; err != nil {
		t.Fatal(err)
	}
	if r.OneSetupVerdicts == nil {
		t.Fatal("E7: the declined scenario's episode row carries no verdict — the stamp is missing")
	}
	v := *r.OneSetupVerdicts
	if !strings.Contains(v, "level=level_not_best") || !strings.Contains(v, "play=ok") || !strings.Contains(v, "permission=ok") {
		t.Fatalf("all three verdicts must be on the row: %q", v)
	}
	if r.OneSetupScenario == nil || *r.OneSetupScenario != "S2" {
		t.Fatalf("the stamp names the scenario: %v", r.OneSetupScenario)
	}
}

// E9 — WIRE PIN. The recorder's pass changes no arm row; the recorder file
// names no placer and no ledger writer.
func TestOneSetupE9RecorderNeverReachesTheWire(t *testing.T) {
	g, at, st, now := driveOneSetupArmPath(t, nil, osMap(nil))
	before := goldenBytes(t, g)
	// A fade episode on S1's level, broken and retested on a tape the recorder scans.
	row := &store.TouchOutcomeRow{TraderID: at.id, Symbol: at.futuresSymbol(), LevelPrice: 29490, LevelKind: "PDL",
		CandidateSeated: true, PlanID: "P", PlanVersion: 1, Session: "X", Ordinal: 1,
		OpenedAtMs: now.Add(-70 * time.Minute).Truncate(time.Minute).UnixMilli(), Outcome: "break", Validity: store.ValidityValid, EntrySide: "above"}
	if err := st.TouchOutcomes().SaveOutcome(row); err != nil {
		t.Fatal(err)
	}
	tape := make([]market.Kline, 0, 80)
	base := now.Add(-80 * time.Minute).Truncate(time.Minute).UnixMilli()
	for i := 0; i < 80; i++ {
		cl := 29495.0
		switch {
		case i >= 15 && i < 30:
			cl = 29480 // closed buckets below → break down
		case i == 30:
			cl = 29490 // retest from below, through by a tick
		case i > 30:
			cl = 29470
		}
		o := base + int64(i)*60_000
		k := market.Kline{OpenTime: o, CloseTime: o + 59_999, Open: cl, High: cl + 1, Low: cl - 1, Close: cl}
		if i == 30 {
			k.High = 29490.25
		}
		tape = append(tape, k)
	}
	scanned, breaks, retests := at.recordFollowPlans(at.futuresSymbol(), tape, now)
	if scanned < 1 || breaks != 1 || retests != 1 {
		t.Fatalf("the recorder must see the break and the retest: scanned=%d breaks=%d retests=%d", scanned, breaks, retests)
	}
	var r store.TouchOutcomeRow
	_ = st.GormDB().First(&r, row.ID).Error
	if r.FollowBreakAtMs == nil || r.FollowRetestAtMs == nil || r.FollowEntryPx == nil || r.BiasWouldFlipTo == nil || *r.BiasWouldFlipTo != "short" {
		t.Fatalf("E8 (wired): the row must carry break, retest, entry and the would-be bias: %+v", r)
	}
	td, sess := "", ""
	if s, ok := at.sessionRegistry(now).ActiveSession(now); ok {
		sess = s.Name
		td, _ = kernel.PlanChainTradeDate(s, now)
	}
	after := goldenBytes(t, snapshotArmPath(t, at, st, td, sess))
	if string(before) != string(after) {
		t.Fatalf("E9: the recorder changed the arm path:\n--- before\n%s\n--- after\n%s", before, after)
	}
	src, err := os.ReadFile("follow_plan_wiring.go")
	if err != nil {
		t.Fatal(err)
	}
	if re := regexp.MustCompile(`PlaceLimitEntry|PlaceStopEntry|UpsertArm|CancelOrder|ArmedOrders\(\)`); re.Match(src) {
		t.Fatal("E9: the recorder names a placer or the ledger — a follow-plan is a row, never a placement")
	}
}

// E10 — BIAS PIN. The plan's live bias is byte-identical with the recorder
// on; bias_would_flip_to sits beside it on the episode row, never in the plan.
func TestOneSetupE10LiveBiasUntouched(t *testing.T) {
	_, at, st, now := driveOneSetupArmPath(t, nil, osMap(nil))
	plan := kernel.ActivePlanFor(at.id, at.futuresSymbol())
	if plan == nil {
		t.Skip("no active plan resolved")
	}
	before, _ := st.Plan().GetPlan(plan.PlanID, plan.Version)
	row := &store.TouchOutcomeRow{TraderID: at.id, Symbol: at.futuresSymbol(), LevelPrice: 29490, LevelKind: "PDL", PlanID: plan.PlanID, PlanVersion: plan.Version,
		OpenedAtMs: now.Add(-70 * time.Minute).Truncate(time.Minute).UnixMilli(), Outcome: "break", Validity: store.ValidityValid, EntrySide: "above"}
	_ = st.TouchOutcomes().SaveOutcome(row)
	tape := make([]market.Kline, 0, 40)
	base := now.Add(-80 * time.Minute).Truncate(time.Minute).UnixMilli()
	for i := 0; i < 40; i++ {
		cl := 29495.0
		if i >= 15 {
			cl = 29480
		}
		o := base + int64(i)*60_000
		tape = append(tape, market.Kline{OpenTime: o, CloseTime: o + 59_999, Open: cl, High: cl + 1, Low: cl - 1, Close: cl})
	}
	at.recordFollowPlans(at.futuresSymbol(), tape, now)
	after, _ := st.Plan().GetPlan(plan.PlanID, plan.Version)
	if before.Doc != after.Doc {
		t.Fatal("E10: the plan document changed — the live bias must never be written")
	}
	var r store.TouchOutcomeRow
	_ = st.GormDB().First(&r, row.ID).Error
	if r.BiasWouldFlipTo == nil || *r.BiasWouldFlipTo != "short" || r.PlanBiasFrozen == nil || *r.PlanBiasFrozen != "long" {
		t.Fatalf("bias_would_flip_to=short must sit beside the frozen plan bias (long): %v / %v", r.BiasWouldFlipTo, r.PlanBiasFrozen)
	}
}

// The test seam is nil in production construction.
func TestOneSetupTestSeamIsNilInProduction(t *testing.T) {
	at, _ := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}})
	if at.oneSetupFactsForTest != nil {
		t.Fatal("the test seam must be nil unless a test sets it")
	}
	src, err := os.ReadFile("one_setup_wiring.go")
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(src), "at.oneSetupFactsForTest"); n != 2 {
		t.Fatalf("the seam is read at exactly one place (the nil check and the call): %d uses", n)
	}
}
