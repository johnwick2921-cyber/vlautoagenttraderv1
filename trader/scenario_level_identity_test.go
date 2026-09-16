package trader

import (
	"encoding/json"
	"nofx/kernel"
	"nofx/market"
	"nofx/store"
	"strings"
	"testing"
	"time"
)

func identityTestLevel(now time.Time, price float64) kernel.DetectedLevel {
	closeMs := now.Add(-10 * time.Hour).UnixMilli()
	return kernel.DetectedLevel{Kind: kernel.KindPDL, Price: price, Lo: price, Hi: price, Label: "PDL", OriginDate: "2026-09-09", FormedAtMs: closeMs - 60000, FormedCloseMs: &closeMs, FormationTF: "1m", IdentitySymbol: "MNQ", FormationLookback: 900}
}
func TestIdentityE1AuthoringToEpisodeAndE3WarnProduction(t *testing.T) {
	now := time.Date(2026, 9, 10, 20, 0, 0, 0, kernel.CTLocation())
	for _, mode := range []string{"named", "unnamed", "unresolved"} {
		t.Run(mode, func(t *testing.T) {
			at := plannerTestTrader(t)
			l := identityTestLevel(now, 15475)
			candidates := kernel.BuildMapCandidates([]kernel.ScoredLevel{{DetectedLevel: l, Grade: "A", Score: 1}}, 15480, 10, kernel.MapCandidateOpts{})
			if len(candidates) != 1 || candidates[0].ID == nil {
				t.Fatal("map did not expose identity")
			}
			id := *candidates[0].ID
			raw := validTraderPlanJSON
			if mode == "unresolved" {
				id = "not-on-map"
			}
			if mode != "unnamed" {
				raw = strings.Replace(raw, `"id": "S1"`, `"id": "S1", "level_id": "`+id+`"`, 1)
			}
			version, lc, err := at.runPlannerReadCoreObserved(func() time.Time { return now }, nil, "NY", "2026-09-10", "", "fixture", "hash", "", "", "", "fixture", kernel.PlanFacts{IdentityMap: candidates}, nil, nil, nil, true, func(string) (string, error) { return raw, nil })
			if err != nil || lc != "active" || version != 1 {
				t.Fatalf("WARN refused authoring: v%d %s %v", version, lc, err)
			}
			p, err := at.store.Plan().GetLatestPlanForTraderSession("2026-09-10", "NY", at.id)
			if err != nil {
				t.Fatal(err)
			}
			var doc kernel.PlanDoc
			if err = json.Unmarshal([]byte(p.Doc), &doc); err != nil {
				t.Fatal(err)
			}
			c, err := at.store.LevelIdentityCounts(at.id)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "unnamed" && c.Unnamed != 1 || mode == "unresolved" && c.Unresolved != 1 {
				t.Fatalf("WARN not recorded: %+v", c)
			}
			if mode != "named" {
				return
			}
			resolved, ok := kernel.LevelByID(doc.Scenarios[0].LevelID, doc.IdentityLevels)
			if !ok || resolved.Price != 15475 || c.Named != 1 || c.HeuristicDisagreed != 1 {
				t.Fatalf("lost named evidence %+v %+v", resolved, c)
			}
			tape := oscillatingTape(15475, now.Add(-9*time.Hour), 500)
			previous := market.FuturesBarsProvider
			market.FuturesBarsProvider = func(string, string, int) []market.Kline { return tape }
			defer func() { market.FuturesBarsProvider = previous }()
			// The legacy proximity answer points at S-other while the plan names S1.
			at.recordDetectorOutputs("MNQ", p.PlanID, "NY", version, []kernel.DetectedLevel{l}, []kernel.ScoredLevel{{DetectedLevel: l, Grade: "A", Score: 1}}, 15475, 100, 2, 12, now, []store.ScenarioAnchor{{ID: "S-other", Price: 15475}})
			rows, err := at.store.TouchOutcomes().AllOutcomes()
			if err != nil || len(rows) == 0 {
				t.Fatalf("no production episode %v", err)
			}
			for _, r := range rows {
				if r.LevelID == nil || *r.LevelID != id || r.ScenarioNearest == nil || *r.ScenarioNearest != "S-other" {
					t.Fatalf("row lost ID or corroboration %+v", r)
				}
				if sc := kernel.EpisodeScenarioByID(r.LevelID, &doc); sc == nil || *sc != "S1" {
					t.Fatal("episode authority fell back to heuristic")
				}
			}
			backfill, err := at.store.BackfillLevelIdentity(at.id)
			if err != nil || backfill.Recomputed != len(rows) || backfill.Unrecomputable != 0 {
				t.Fatalf("complete frozen inputs did not recompute %+v %v", backfill, err)
			}
			// Polling the same version is one measured disagreement, not N loop ticks.
			at.observeScenarioIdentity(&doc, p.PlanID, version, []kernel.ScenarioEval{{ID: "S1", Anchor: 15480, HasAnchor: true}}, now)
			c, _ = at.store.LevelIdentityCounts(at.id)
			if c.HeuristicDisagreed != 1 {
				t.Fatalf("counter inferred polling ticks %+v", c)
			}
		})
	}
}

func TestIdentityE6LegacyAndBootReads(t *testing.T) {
	at := plannerTestTrader(t)
	now := time.Date(2026, 9, 10, 20, 0, 0, 0, kernel.CTLocation())
	if _, err := at.store.Plan().AppendPlan(&store.PlanDB{PlanID: "legacy", StrategyID: at.id, TradeDate: "2026-09-10", Session: "NY", Doc: validTraderPlanJSON}); err != nil {
		t.Fatal(err)
	}
	row := store.TouchOutcomeRow{TraderID: at.id, Symbol: "MNQ", LevelKind: "PDL", CreatedAt: now.Add(-24 * time.Hour)}
	if err := at.store.TouchOutcomes().SaveOutcome(&row); err != nil {
		t.Fatal(err)
	}
	b, err := at.store.BackfillLevelIdentity(at.id)
	if err != nil || b.Untouched != 1 || b.Recomputed != 0 {
		t.Fatalf("legacy changed %+v %v", b, err)
	}
	var doc kernel.PlanDoc
	_ = json.Unmarshal([]byte(validTraderPlanJSON), &doc)
	r := kernel.ScenarioIdentities(&doc)["S1"]
	if r.LevelID != nil || r.Level != nil {
		t.Fatal("legacy identity guessed")
	}
	c := store.LevelIdentityCounts{Named: 7, Unnamed: 3, Unresolved: 2, HeuristicDisagreed: 1}
	text := levelIdentityBootLine(&doc, &c, &b)
	for _, want := range []string{"map ids=n/a", "named=7", "unnamed=3[WARN]", "unresolved=2[WARN]", "heuristic-disagreed=1", "untouched=1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("boot did not read %s: %s", want, text)
		}
	}
}

func TestIdentityRecordingPanicContained(t *testing.T) {
	at := plannerTestTrader(t)
	returned := false
	func() { defer at.containLevelIdentity(); panic("fixture recording failure") }()
	returned = true
	if !returned {
		t.Fatal("recording panic escaped")
	}
}
