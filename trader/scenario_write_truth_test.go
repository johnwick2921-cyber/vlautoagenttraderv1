package trader

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// W-EXEC-TRUTH W2 A3 (identity ≠ price = REFUSE) + A4 (obstacle-chain
// contract). Every test below drives the PRODUCTION write loop
// (runPlannerReadCoreWithFactsGradesClock → runPlannerReadCoreObserved) with a
// stubbed model reply, unless it says otherwise. Fixtures: plans rowid 452 and
// 455 (trader/testdata/w2-write-truth/*.json carry the provenance line).

type w2Fixture struct {
	IdentityLevels []kernel.PlanLevel `json:"identity_levels"`
	Levels         []kernel.PlanLevel `json:"levels"`
	Bias           json.RawMessage    `json:"bias"`
	Death          json.RawMessage    `json:"death"`
	Flip           json.RawMessage    `json:"flip"`
	DeathCondition string             `json:"death_condition"`
	DayType        string             `json:"day_type"`
	PriceAtWrite   float64            `json:"price_at_write"`
	Scenarios      []map[string]any   `json:"scenarios"`
}

func loadW2Fixture(t *testing.T, row int) w2Fixture {
	t.Helper()
	b, err := os.ReadFile(fmt.Sprintf("testdata/w2-write-truth/plan_row_%d_subset.json", row))
	if err != nil {
		t.Fatal(err)
	}
	var f w2Fixture
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

// candidates rebuilds the frozen seated map exactly as the stored doc carries
// it (identity_levels is StampAuthoredIdentity's projection of facts.IdentityMap).
func (f w2Fixture) candidates() []kernel.MapCandidate {
	out := make([]kernel.MapCandidate, 0, len(f.IdentityLevels))
	for _, l := range f.IdentityLevels {
		out = append(out, kernel.MapCandidate{ID: l.ID, Identity: l, Price: l.Price, Names: append([]string(nil), l.Names...)})
	}
	return out
}

func (f w2Fixture) idAt(t *testing.T, price float64) string {
	t.Helper()
	for _, l := range f.IdentityLevels {
		if l.ID != nil && l.Price > price-0.01 && l.Price < price+0.01 {
			return *l.ID
		}
	}
	t.Fatalf("fixture has no identity level at %.2f", price)
	return ""
}

// scenario returns a deep copy of an authored scenario with the machine-only
// stamps removed (economics.version is parser-stamped, arm_disabled_reason is
// written by the write site — neither is model output).
func (f w2Fixture) scenario(t *testing.T, id string) map[string]any {
	t.Helper()
	for _, s := range f.Scenarios {
		if s["id"] != id {
			continue
		}
		b, _ := json.Marshal(s)
		var c map[string]any
		_ = json.Unmarshal(b, &c)
		if e, ok := c["economics"].(map[string]any); ok {
			delete(e, "version")
		}
		if a, ok := c["arm"].(map[string]any); ok {
			delete(a, "arm_disabled_reason")
		}
		return c
	}
	t.Fatalf("fixture has no scenario %s", id)
	return nil
}

func (f w2Fixture) planJSON(t *testing.T, scenarios ...map[string]any) string {
	t.Helper()
	levels := make([]map[string]any, 0, len(f.Levels))
	for _, l := range f.Levels {
		levels = append(levels, map[string]any{"price": l.Price, "label": l.Label, "grade": l.Grade, "instruction": l.Instruction})
	}
	doc := map[string]any{"reasoning": "W2 fixture", "bias": f.Bias, "levels": levels, "scenarios": scenarios,
		"no_trade": []string{"first 5m"}, "death_condition": f.DeathCondition, "death": f.Death, "flip": f.Flip, "day_type": f.DayType}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func (f w2Fixture) facts() kernel.PlanFacts {
	return kernel.PlanFacts{Price: f.PriceAtWrite, DATR: 300, IdentityMap: f.candidates()}
}

func w2Trader(t *testing.T) *AutoTrader {
	t.Helper()
	at := plannerTestTrader(t)
	at.config.StrategyConfig.DayPlan.MaxLevels = 12
	return at
}

// w2Write drives the production write loop with one stubbed reply per attempt
// (the last reply repeats) and returns the prompts the model saw.
func w2Write(t *testing.T, at *AutoTrader, session, tradeDate string, facts kernel.PlanFacts, replies ...string) (prompts []string, lc string) {
	t.Helper()
	_, lc, err := at.runPlannerReadCoreWithFactsGradesClock(feasClock(), session, tradeDate, "owner_reset",
		"deepseek-v4-pro", "hashW2", "", "", "", "FULLPROMPT", facts, nil, nil, nil, true,
		func(userPrompt string) (string, error) {
			prompts = append(prompts, userPrompt)
			i := len(prompts) - 1
			if i >= len(replies) {
				i = len(replies) - 1
			}
			return replies[i], nil
		})
	if err != nil {
		t.Fatalf("write loop error: %v", err)
	}
	return prompts, lc
}

func w2StoredDoc(t *testing.T, at *AutoTrader, session, tradeDate string) kernel.PlanDoc {
	t.Helper()
	row, err := at.store.Plan().GetLatestPlanForSession(tradeDate, session)
	if err != nil || row == nil {
		t.Fatalf("no stored plan: %v", err)
	}
	var doc kernel.PlanDoc
	if err := json.Unmarshal([]byte(row.Doc), &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func w2Counter(t *testing.T, at *AutoTrader, what string) int {
	t.Helper()
	n, err := store.SystemCounter(at.store, writeTruthCounterKey(at.id, what))
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// A3 RED→GREEN on row 452 S2 (log 90563): level_id names PDH 30917.50 while
// trigger/confirm trade 31009.75. Attempt 1 is REFUSED at write (never
// "recorded; evaluator unchanged"); the attempt-2 prompt carries the reason
// naming both ids and both prices; attempt 2 with the SWG-H·5m id publishes.
func TestW2A3Row452S2RefusedThenRepublishedWithTheRightID(t *testing.T) {
	f := loadW2Fixture(t, 452)
	at := w2Trader(t)
	pdhID, swgID := f.idAt(t, 30917.5), f.idAt(t, 31009.75)
	bad := f.scenario(t, "S2")
	if bad["level_id"] != pdhID {
		t.Fatalf("fixture drift: row 452 S2 level_id=%v, want the PDH id", bad["level_id"])
	}
	good := f.scenario(t, "S2")
	good["level_id"] = swgID
	prompts, lc := w2Write(t, at, "ASIA", "2026-09-22", f.facts(), f.planJSON(t, bad), f.planJSON(t, good))
	if lc != "active" || len(prompts) != 2 {
		t.Fatalf("want refused attempt 1 then published attempt 2: lc=%q calls=%d", lc, len(prompts))
	}
	for _, want := range []string{"S2 identity≠price", pdhID, swgID, "30917.50", "31009.75", kernel.RepairIdentityPriceLaw} {
		if !strings.Contains(prompts[1], want) {
			t.Fatalf("attempt-2 prompt lacks %q", want)
		}
	}
	doc := w2StoredDoc(t, at, "ASIA", "2026-09-22")
	if doc.Scenarios[0].LevelID == nil || *doc.Scenarios[0].LevelID != swgID {
		t.Fatalf("published doc must carry the re-authored id, got %v", doc.Scenarios[0].LevelID)
	}
	c, err := at.store.LevelIdentityCounts(at.id)
	if err != nil {
		t.Fatal(err)
	}
	if c.HeuristicDisagreed != 0 || c.Named != 1 {
		t.Fatalf("a refused disagreement must never be recorded as a published one: %+v", c)
	}
	rej, err := at.store.PlannerRejected().Latest()
	if err != nil || rej == nil || rej.Attempt != 1 || !strings.Contains(rej.RejectReason, "S2 identity≠price") {
		t.Fatalf("planner_rejected_prompts row missing or wrong: %+v %v", rej, err)
	}
	if n := w2Counter(t, at, "refused:"+kernel.WriteTruthIdentityPrice); n != 1 {
		t.Fatalf("identity_price refusal counter = %d, want 1 (recorded per refused attempt)", n)
	}
	if n := w2Counter(t, at, "checked"); n != 2 {
		t.Fatalf("checked counter = %d, want 2 (both attempts judged)", n)
	}
}

// A3 always-bad → the existing fail-closed NO-TRADE path after 3 attempts.
func TestW2A3AlwaysDisagreeingFailsClosed(t *testing.T) {
	f := loadW2Fixture(t, 452)
	at := w2Trader(t)
	prompts, lc := w2Write(t, at, "ASIA", "2026-09-22", f.facts(), f.planJSON(t, f.scenario(t, "S2")))
	if lc != "no_trade" || len(prompts) != 3 {
		t.Fatalf("an always-disagreeing plan must fail closed after 3 attempts: lc=%q calls=%d", lc, len(prompts))
	}
	if n := w2Counter(t, at, "refused:"+kernel.WriteTruthIdentityPrice); n != 3 {
		t.Fatalf("identity_price refusals = %d, want 3", n)
	}
}

// A3 tightening (CTO msg 1790178967603 + 1790178997353): an id that does not
// resolve in the frozen map — an invented strict id, or a ref| id whose digest
// was altered (the 09-22 London row-443 shape) — is REFUSED at write and
// re-authorable within the attempts.
func TestW2A3UnresolvedIDsRefusedThenRepublished(t *testing.T) {
	f := loadW2Fixture(t, 452)
	swgID := f.idAt(t, 31009.75)
	mangled := []byte(swgID) // one digest digit altered (the row-443 shape)
	if mangled[5] == 'e' {
		mangled[5] = 'f'
	} else {
		mangled[5] = 'e'
	}
	for _, tc := range []struct{ name, id string }{
		{"invented strict id", strings.Repeat("ab", 32)},
		{"ref| id with an altered digest", "ref|" + string(mangled)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			at := w2Trader(t)
			bad := f.scenario(t, "S2")
			bad["level_id"] = tc.id
			good := f.scenario(t, "S2")
			good["level_id"] = swgID
			prompts, lc := w2Write(t, at, "ASIA", "2026-09-22", f.facts(), f.planJSON(t, bad), f.planJSON(t, good))
			if lc != "active" || len(prompts) != 2 {
				t.Fatalf("unresolved id must be refused then re-authored: lc=%q calls=%d", lc, len(prompts))
			}
			for _, want := range []string{"S2 identity unresolved", tc.id} {
				if !strings.Contains(prompts[1], want) {
					t.Fatalf("attempt-2 prompt lacks %q", want)
				}
			}
			if n := w2Counter(t, at, "refused:"+kernel.WriteTruthIdentityUnresolved); n != 1 {
				t.Fatalf("identity_unresolved refusals = %d, want 1", n)
			}
			c, _ := at.store.LevelIdentityCounts(at.id)
			if c.Unresolved != 0 {
				t.Fatalf("an unresolved id must never be published: %+v", c)
			}
		})
	}
}

// A4 on row 455: S2 names RTH-H 31050 as its first obstacle while SWG-H·15m
// 31059 sits first on the short path; S4 omits SWG-H·5m 31043 four points from
// entry. Attempt 1 (the stored doc) is refused naming both; attempt 2 fixes
// both and publishes.
func TestW2A4Row455S2AndS4RefusedByNameThenRepublished(t *testing.T) {
	f := loadW2Fixture(t, 455)
	at := w2Trader(t)
	s2, s4 := f.scenario(t, "S2"), f.scenario(t, "S4")
	prompts, lc := w2Write(t, at, "LONDON", "2026-09-23", f.facts(), f.planJSON(t, s2, s4), f.planJSON(t, w2Row455S2Fixed(t, f), w2Row455S4Fixed(t, f)))
	if lc != "active" || len(prompts) != 2 {
		t.Fatalf("want refused attempt 1 then published attempt 2: lc=%q calls=%d", lc, len(prompts))
	}
	for _, want := range []string{
		"S2 obstacle chain: first_obstacle 31050.00 is not the nearest seated level on the path — SWG-H·15m 31059.00",
		"S4 obstacle chain: omits SWG-H·5m 31043.00 (4.00 pts from entry)",
		kernel.RepairObstacleChainLaw,
	} {
		if !strings.Contains(prompts[1], want) {
			t.Fatalf("attempt-2 prompt lacks %q", want)
		}
	}
	doc := w2StoredDoc(t, at, "LONDON", "2026-09-23")
	if o := doc.Scenarios[0].Economics.FirstObstacle; o == nil || *o.Price != 31059 {
		t.Fatalf("published S2 first obstacle = %+v, want 31059", o)
	}
	if len(doc.Scenarios[1].Economics.PathLevels) != 3 {
		t.Fatalf("published S4 must carry its path_levels: %+v", doc.Scenarios[1].Economics)
	}
	if n := w2Counter(t, at, "refused:"+kernel.WriteTruthObstacleNotNearest); n != 2 {
		t.Fatalf("obstacle_not_nearest refusals = %d, want 2 (S2 and S4 on attempt 1)", n)
	}
	if n := w2Counter(t, at, "refused:"+kernel.WriteTruthPathLevelMissing); n != 2 {
		t.Fatalf("path_level_missing refusals = %d, want 2", n)
	}
}

func w2Row455S2Fixed(t *testing.T, f w2Fixture) map[string]any {
	s := f.scenario(t, "S2")
	e := s["economics"].(map[string]any)
	e["first_obstacle"] = map[string]any{"price": 31059, "level": "SWG-H·15m", "family": "swing", "response": "pass_through"}
	e["r_to_obstacle"] = (31066.32 - 31059) / 23
	e["path_levels"] = []map[string]any{
		{"price": 31050, "level": "RTH-H", "role": "pass_through"},
		{"price": 31043, "level": "SWG-H·5m", "role": "pass_through"},
		{"price": 31035.25, "level": "PDC", "role": "pass_through"},
		{"price": 31030.27, "level": "eVWAP", "role": "pass_through"},
		{"price": 31021.5, "level": "SWG-L·5m", "role": "exit"},
	}
	return s
}

func w2Row455S4Fixed(t *testing.T, f w2Fixture) map[string]any {
	s := f.scenario(t, "S4")
	e := s["economics"].(map[string]any)
	e["first_obstacle"] = map[string]any{"price": 31043, "level": "SWG-H·5m", "family": "swing", "response": "pass_through"}
	e["r_to_obstacle"] = 4.0 / 23
	e["path_levels"] = []map[string]any{
		{"price": 31035.25, "level": "PDC", "role": "pass_through"},
		{"price": 31030.27, "level": "eVWAP", "role": "pass_through"},
		{"price": 31021.5, "level": "SWG-L·5m", "role": "pass_through"},
	}
	return s
}

// Multi-anchor on row 455 S3 (sweep ONL 30983.25 → reclaim VWAP−2σ 30998.26):
// a legit two-anchor declaration publishes; the same id twice, or an id at an
// unrelated price, is refused; the stored doc carries the two ids.
func TestW2A3MultiAnchorSweepReclaim(t *testing.T) {
	f := loadW2Fixture(t, 455)
	onl, vwap2, pdc := f.idAt(t, 30983.25), f.idAt(t, 30998.26), f.idAt(t, 31035.25)
	legit := func() map[string]any {
		s := f.scenario(t, "S3")
		s["economics"].(map[string]any)["path_levels"] = []map[string]any{
			{"price": 31012.5, "level": "SWG-L·15m", "role": "pass_through"},
			{"price": 31021.5, "level": "SWG-L·5m", "role": "pass_through"},
			{"price": 31030.27, "level": "eVWAP", "role": "pass_through"},
			{"price": 31035.25, "level": "PDC", "role": "pass_through"},
			{"price": 31043, "level": "SWG-H·5m", "role": "pass_through"},
			{"price": 31050, "level": "RTH-H", "role": "pass_through"},
			{"price": 31059, "level": "SWG-H·15m", "role": "exit"},
		}
		s["sweep_level_id"] = onl
		s["reclaim_level_id"] = vwap2
		return s
	}
	t.Run("legit two anchors publish", func(t *testing.T) {
		at := w2Trader(t)
		prompts, lc := w2Write(t, at, "LONDON", "2026-09-23", f.facts(), f.planJSON(t, legit()))
		if lc != "active" || len(prompts) != 1 {
			t.Fatalf("legit two-anchor sweep_reclaim must publish on attempt 1: lc=%q calls=%d", lc, len(prompts))
		}
		doc := w2StoredDoc(t, at, "LONDON", "2026-09-23")
		if s := doc.Scenarios[0]; s.SweepLevelID == nil || *s.SweepLevelID != onl || s.ReclaimLevelID == nil || *s.ReclaimLevelID != vwap2 {
			t.Fatalf("stored two-anchor ids lost: %+v", s)
		}
	})
	// level_id naming the RECLAIM leg's level (not the trigger's first price):
	// legitimate only because the two-anchor fields vouch for it; without
	// them it is an identity≠price refusal.
	t.Run("level_id at a validated leg", func(t *testing.T) {
		at := w2Trader(t)
		s := legit()
		s["level_id"] = vwap2
		prompts, lc := w2Write(t, at, "LONDON", "2026-09-23", f.facts(), f.planJSON(t, s))
		if lc != "active" || len(prompts) != 1 {
			t.Fatalf("level_id at a declared, validated leg must publish: lc=%q calls=%d", lc, len(prompts))
		}
		at2 := w2Trader(t)
		bare := legit()
		bare["level_id"] = vwap2
		delete(bare, "sweep_level_id")
		delete(bare, "reclaim_level_id")
		prompts, lc = w2Write(t, at2, "LONDON", "2026-09-23", f.facts(), f.planJSON(t, bare), f.planJSON(t, legit()))
		if lc != "active" || len(prompts) != 2 || !strings.Contains(prompts[1], "S3 identity≠price") {
			t.Fatalf("without the two-anchor fields the same level_id is identity≠price: lc=%q calls=%d", lc, len(prompts))
		}
	})
	for _, tc := range []struct {
		name, class, want string
		mutate            func(map[string]any)
	}{
		{"same id reused", kernel.WriteTruthAnchorReuse, "sweep_level_id and reclaim_level_id are the same id", func(s map[string]any) { s["reclaim_level_id"] = onl }},
		{"unrelated reclaim id", kernel.WriteTruthAnchorUnrelated, "reclaim_level_id", func(s map[string]any) { s["reclaim_level_id"] = pdc }},
		{"unresolved sweep id", kernel.WriteTruthIdentityUnresolved, "sweep_level_id", func(s map[string]any) { s["sweep_level_id"] = "ref|" + strings.Repeat("0", 64) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			at := w2Trader(t)
			bad := legit()
			tc.mutate(bad)
			prompts, lc := w2Write(t, at, "LONDON", "2026-09-23", f.facts(), f.planJSON(t, bad), f.planJSON(t, legit()))
			if lc != "active" || len(prompts) != 2 {
				t.Fatalf("%s must be refused then re-authored: lc=%q calls=%d", tc.name, lc, len(prompts))
			}
			if !strings.Contains(prompts[1], "S3 identity") || !strings.Contains(prompts[1], tc.want) {
				t.Fatalf("attempt-2 prompt does not name the defect %q", tc.want)
			}
			if n := w2Counter(t, at, "refused:"+tc.class); n != 1 {
				t.Fatalf("%s counter = %d, want 1", tc.class, n)
			}
		})
	}
}

// History untouched: a stored legacy doc (row 452, S2 still naming PDH, no
// sweep/reclaim/path fields) reads exactly as before — no new key appears on
// re-marshal, the stored-doc validator never runs A3/A4, and the read-side
// projection still reports the recorded disagreement.
func TestW2LegacyStoredDocReadsUnchanged(t *testing.T) {
	b, err := os.ReadFile("testdata/w2-write-truth/plan_row_452_subset.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc kernel.PlanDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	out, _ := json.Marshal(doc)
	for _, k := range []string{"sweep_level_id", "reclaim_level_id", "path_levels"} {
		if strings.Contains(string(out), k) {
			t.Fatalf("legacy re-marshal grew %q", k)
		}
	}
	if err := kernel.ValidatePlanDocWithCaps(&doc, 12, 5); err != nil && strings.Contains(err.Error(), "identity") {
		t.Fatalf("stored-doc validation must not judge identity: %v", err)
	}
	r := kernel.ScenarioIdentities(&doc)["S2"]
	if r.Level == nil || r.Level.Price != 30917.5 || !r.Disagreed {
		t.Fatalf("stored disagreement must still read as recorded: %+v", r)
	}
}

// No false refusals: a reject at an S/D zone EDGE naming its own zone
// candidate (candidate price = zone midpoint, 5 pts from the edge anchor) is
// admitted — the level's own zone is the tolerance.
func TestW2A3ZoneEdgeRejectNamingItsZoneIsAdmitted(t *testing.T) {
	at := plannerTestTrader(t)
	feasOff := false
	at.config.StrategyConfig.DayPlan.WriteTimeFeasibility = &feasOff
	now := time.Date(2026, 9, 10, 20, 0, 0, 0, kernel.CTLocation())
	closeMs := now.Add(-10 * time.Hour).UnixMilli()
	zone := kernel.DetectedLevel{Kind: kernel.KindDemand, Price: 15475, Lo: 15470, Hi: 15480, Label: "Demand·1h", OriginDate: "2026-09-09",
		FormedAtMs: closeMs - 3600000, FormedCloseMs: &closeMs, FormationTF: "1h", IdentitySymbol: "MNQ", FormationLookback: 400}
	cs := kernel.BuildMapCandidates([]kernel.ScoredLevel{{DetectedLevel: zone, Grade: "A", Score: 1}}, 15550, 10, kernel.MapCandidateOpts{})
	if len(cs) != 1 || cs[0].ID == nil {
		t.Fatalf("zone candidate has no identity: %+v", cs)
	}
	plan := strings.Replace(class39LegsPlanJSON("15550"), `"condition": "reject", "direction": "long"`,
		`"condition": "reject", "level_id": "`+*cs[0].ID+`", "direction": "long"`, 1)
	prompts, lc := w2Write(t, at, "NY", "2026-08-14", kernel.PlanFacts{Price: 15550, DATR: 300, IdentityMap: cs}, plan)
	if lc != "active" || len(prompts) != 1 {
		t.Fatalf("a reject at its own zone's edge must be admitted on attempt 1: lc=%q calls=%d", lc, len(prompts))
	}
}

// A4 on the compact class-39 universe (long reject at PWL 15480, arm target
// 15550, one extra seated level at 15520): unsorted chain refused (short mirror
// passes), reduce on a single arm refused, an off-tick seated midpoint still
// matches after rounding, and the capacity-cut pool is accepted, never required.
func TestW2A4ChainContractOnTheWriteLoop(t *testing.T) {
	now := time.Date(2026, 9, 10, 20, 0, 0, 0, kernel.CTLocation())
	pwl := identityTestLevel(now, 15480)
	pwl.Label = "PWL"
	mid := identityTestLevel(now, 15520.1) // an off-tick seated price (a midpoint)
	mid.Label = "RN 15525"
	seated := kernel.BuildMapCandidates([]kernel.ScoredLevel{{DetectedLevel: pwl, Grade: "A", Score: 2}, {DetectedLevel: mid, Grade: "B", Score: 1}}, 15550, 10, kernel.MapCandidateOpts{})
	if len(seated) != 2 {
		t.Fatalf("seated map: %+v", seated)
	}
	base := class39LegsPlanJSON("15550")
	withObstacle := func(raw, obstacle, response string, r float64) string {
		return strings.Replace(raw, `"first_obstacle":{"price":15550,"level":"fixture reference","family":"reference","response":"pass_through"},"r_to_obstacle":7`,
			fmt.Sprintf(`"first_obstacle":{"price":%s,"level":"RN 15525","family":"round","response":"%s"},"r_to_obstacle":%v`, obstacle, response, r), 1)
	}
	good := withObstacle(base, "15520.10", "pass_through", 4.01)
	if good == base {
		t.Fatal("fixture mutation missed")
	}
	facts := kernel.PlanFacts{Price: 15550, DATR: 300, IdentityMap: seated}
	run := func(t *testing.T, replies ...string) ([]string, string, *AutoTrader) {
		at := plannerTestTrader(t)
		feasOff := false
		at.config.StrategyConfig.DayPlan.WriteTimeFeasibility = &feasOff
		p, lc := w2Write(t, at, "NY", "2026-08-14", facts, replies...)
		return p, lc, at
	}
	t.Run("off-tick obstacle and seated midpoint normalized and admitted", func(t *testing.T) {
		p, lc, at := run(t, good)
		if lc != "active" || len(p) != 1 {
			t.Fatalf("lc=%q calls=%d", lc, len(p))
		}
		doc := w2StoredDoc(t, at, "NY", "2026-08-14")
		if got := *doc.Scenarios[0].Economics.FirstObstacle.Price; got != 15520 {
			t.Fatalf("first_obstacle must be normalized to the tick grid: %.4f", got)
		}
		if n := w2Counter(t, at, "tick_normalized"); n != 1 {
			t.Fatalf("tick_normalized counter = %d, want 1", n)
		}
	})
	t.Run("obstacle skipping the seated level is refused", func(t *testing.T) {
		p, lc, _ := run(t, base, good)
		if lc != "active" || len(p) != 2 || !strings.Contains(p[1], "S1 obstacle chain: first_obstacle 15550.00 is not the nearest seated level on the path — RN 15525 15520.00") {
			t.Fatalf("lc=%q calls=%d", lc, len(p))
		}
	})
	t.Run("unsorted chain refused", func(t *testing.T) {
		bad := strings.Replace(good, `"target_chain": [15550, 15620]`, `"target_chain": [15620, 15550]`, 1)
		p, lc, _ := run(t, bad, good)
		if lc != "active" || len(p) != 2 || !strings.Contains(p[1], "S1 obstacle chain: target_chain [15620.00 15550.00] is not sorted") {
			t.Fatalf("lc=%q calls=%d", lc, len(p))
		}
	})
	t.Run("reduce on a single arm refused", func(t *testing.T) {
		bad := withObstacle(base, "15520.10", "reduce", 4.01)
		p, lc, _ := run(t, bad, good)
		if lc != "active" || len(p) != 2 || !strings.Contains(p[1], "S1 obstacle chain: reduce at first_obstacle 15520.00 is infeasible") {
			t.Fatalf("lc=%q calls=%d", lc, len(p))
		}
	})
	t.Run("capacity-cut candidate nearer on the path accepted, never required", func(t *testing.T) {
		cutLvl := identityTestLevel(now, 15500)
		cutLvl.Label = "RN 15500"
		cut := kernel.BuildMapCandidates([]kernel.ScoredLevel{{DetectedLevel: cutLvl, Grade: "C", Score: 0.1}}, 15550, 10, kernel.MapCandidateOpts{})
		f2 := facts
		f2.CapacityCut = cut
		cutFirst := strings.Replace(withObstacle(base, "15500", "pass_through", 2.0), `"r_to_arm_target":7}`,
			`"r_to_arm_target":7,"path_levels":[{"price":15520,"level":"RN 15525","role":"pass_through"}]}`, 1)
		if !strings.Contains(cutFirst, "path_levels") || !strings.Contains(cutFirst, `"price":15500`) {
			t.Fatal("fixture mutation missed")
		}
		for _, raw := range []string{cutFirst, good} {
			at := plannerTestTrader(t)
			feasOff := false
			at.config.StrategyConfig.DayPlan.WriteTimeFeasibility = &feasOff
			p, lc := w2Write(t, at, "NY", "2026-08-14", f2, raw)
			if lc != "active" || len(p) != 1 {
				t.Fatalf("capacity-cut: lc=%q calls=%d", lc, len(p))
			}
		}
	})
	t.Run("nil identity map skips the checks (UNKNOWN, not [])", func(t *testing.T) {
		at := plannerTestTrader(t)
		feasOff := false
		at.config.StrategyConfig.DayPlan.WriteTimeFeasibility = &feasOff
		p, lc := w2Write(t, at, "NY", "2026-08-14", kernel.PlanFacts{Price: 15550, DATR: 300}, base)
		if lc != "active" || len(p) != 1 || w2Counter(t, at, "checked") != 0 {
			t.Fatalf("nil map must not judge: lc=%q calls=%d", lc, len(p))
		}
	})
}

// A4 short mirror: the same chain sorted DESCENDING on a short passes.
func TestW2A4ShortMirrorSortedDescendingPasses(t *testing.T) {
	d := &kernel.PlanDoc{Scenarios: []kernel.PlanScenario{{ID: "S1", Direction: "short", TargetChain: []float64{15520, 15480},
		Economics: &kernel.ScenarioEconomics{EntryZone: []float64{15550, 15550}, Geometry: &kernel.ScenarioGeometry{Entry: 15550, Stop: 15560, Target: 15480},
			FirstObstacle: &kernel.ScenarioObstacle{Price: w2f(15480), Level: "PWL", Family: "pivot", Response: "exit"}}}}}
	v := kernel.CheckScenarioWriteTruth(d, []kernel.MapCandidate{}, nil, 0.25)
	if err := v.Err(); err != nil {
		t.Fatalf("short chain sorted descending must pass: %v", err)
	}
	d.Scenarios[0].TargetChain = []float64{15480, 15520}
	if err := kernel.CheckScenarioWriteTruth(d, []kernel.MapCandidate{}, nil, 0.25).Err(); err == nil || !strings.Contains(err.Error(), "not sorted") {
		t.Fatalf("an ascending chain on a short must be refused: %v", err)
	}
}

func w2f(v float64) *float64 { return &v }

// Shadow A/B parity: shadowVerdictFor runs the SAME check and reports the
// same refusal; the corrected document is legal.
func TestW2ShadowVerdictParity(t *testing.T) {
	f := loadW2Fixture(t, 452)
	at := w2Trader(t)
	good := f.scenario(t, "S2")
	good["level_id"] = f.idAt(t, 31009.75)
	legal, reasons := at.shadowVerdictFor(f.planJSON(t, f.scenario(t, "S2")), 12, 3, f.facts(), nil, nil, "")
	if legal || !strings.Contains(strings.Join(reasons, " | "), "S2 identity≠price") {
		t.Fatalf("shadow must refuse the disagreeing S2: legal=%v reasons=%v", legal, reasons)
	}
	legal, reasons = at.shadowVerdictFor(f.planJSON(t, good), 12, 3, f.facts(), nil, nil, "")
	if !legal {
		t.Fatalf("shadow must accept the corrected S2: %v", reasons)
	}
	if n := w2Counter(t, at, "checked"); n != 0 {
		t.Fatalf("the shadow verdict must record nothing, checked=%d", n)
	}
}

// Boot line: n/a before the first recorded check; recorded values after.
func TestW2WriteTruthBootLine(t *testing.T) {
	at := plannerTestTrader(t)
	if line := writeTruthBootLine(at.store, at.id); !strings.Contains(line, "checked=n/a") || !strings.Contains(line, "identity_price=n/a") || !strings.Contains(line, "tick-normalized=n/a") {
		t.Fatalf("before any check the boot line must print n/a: %s", line)
	}
	if line := writeTruthBootLine(nil, at.id); !strings.Contains(line, "checked=n/a") {
		t.Fatalf("no store → n/a: %s", line)
	}
	at.incWriteTruth("checked")
	at.incWriteTruth("refused:" + kernel.WriteTruthPathLevelMissing)
	line := writeTruthBootLine(at.store, at.id)
	for _, want := range []string{"checked=1", "path_level_missing=1", "identity_price=0", "tick-normalized=0"} {
		if !strings.Contains(line, want) {
			t.Fatalf("boot line lacks %q: %s", want, line)
		}
	}
}
