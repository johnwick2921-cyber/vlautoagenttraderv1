package kernel

import (
	"encoding/json"
	"nofx/market"
	"os"
	"strings"
	"testing"
	"time"
)

func truthBase() int64 { return time.Date(2026, 9, 8, 13, 15, 0, 0, time.UTC).UnixMilli() }
func truthScenario() PlanScenario {
	return PlanScenario{ID: "S1", Condition: "sweep_reclaim", Direction: "short", Confirm: &PlanConfirm{Rule: "touch", Side: "above", RefPrice: 100}, Confirm2: &PlanConfirm{Rule: "1x5m_close", Side: "below", RefPrice: 100}}
}

func TestConfirmationTruthLive38329(t *testing.T) {
	var f struct {
		SinceMs  int64          `json:"since_ms"`
		NowMs    int64          `json:"now_ms"`
		Scenario PlanScenario   `json:"scenario"`
		Bars     []market.Kline `json:"bars"`
	}
	b, err := os.ReadFile("testdata/confirmation-london-38329.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	v := EvaluateScenarioConfirm(f.Scenario, f.Bars, f.SinceMs, f.NowMs)
	if v.Met {
		t.Fatalf("decision 38329 at 08:16:59: forming 08:15–08:20 reclaim must NOT MET: %+v", v)
	}
	if !strings.Contains(v.Detail, "08:20") {
		t.Fatalf("verdict must name bucket close 08:20: %s", v.Detail)
	}
}

func TestConfirmationTruthBucketBoundary(t *testing.T) {
	base := truthBase()
	for _, rule := range []string{"1x5m_close", "2x5m_close", "15m_close"} {
		mins := 5
		if rule == "2x5m_close" {
			mins = 10
		}
		if rule == "15m_close" {
			mins = 15
		}
		bars := confirmBars(base, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99)[:mins]
		for _, delta := range []int64{-1000, 0, 1000} {
			t.Run(rule+time.Duration(delta*int64(time.Millisecond)).String(), func(t *testing.T) {
				v := EvaluateConfirm(PlanConfirm{Rule: rule, RefPrice: 100, Side: "below"}, bars, base, base+int64(mins)*60000+delta)
				if v.Met != (delta >= 0) {
					t.Fatalf("%s at boundary %+dms: MET=%v detail=%s", rule, delta, v.Met, v.Detail)
				}
			})
		}
	}
}

func TestConfirmationTruthSequenceOrder(t *testing.T) {
	base := truthBase()
	s := truthScenario()
	bars := confirmBars(base, 99, 99, 99, 99, 99, 101)
	for i := 0; i < 5; i++ {
		bars[i].High = 99.75
	}
	bars[5].Low = 99.75
	v := EvaluateScenarioConfirm(s, bars, base, base+6*60000)
	if v.Met || !strings.Contains(v.Detail, "out of order") {
		t.Fatalf("reclaim BEFORE touch must be out of order: %+v", v)
	}
	// A touch in the first minute followed by the bucket's close is ordered,
	// even though that bucket OPENED before the reference instant.
	ordered := confirmBars(base, 101, 99, 99, 99, 99)
	ordered[0].Low = 99.75
	if v := EvaluateScenarioConfirm(s, ordered, base, base+5*60000); !v.Met {
		t.Fatalf("touch then completed reclaim must MET: %+v", v)
	}
}

func TestConfirmationTruthMissingReference(t *testing.T) {
	base := truthBase()
	s := truthScenario()
	bars := confirmBars(base, 99, 99, 99, 99, 99)
	for i := range bars {
		bars[i].High = 99.75
	}
	v := EvaluateScenarioConfirm(s, bars, base, base+5*60000)
	b, _ := json.Marshal(v)
	if v.Met || !strings.Contains(string(b), `"outcome":"UNKNOWN"`) || !strings.Contains(v.Detail, "reference instant is missing") {
		t.Fatalf("missing touch reference must UNKNOWN, never plan birth: %s", b)
	}
}

func TestConfirmationTruthBreakdownAndVoidConsumers(t *testing.T) {
	base := truthBase()
	t.Setenv("BD_MIN_CLOSES", "1")
	sc := PlanScenario{ID: "S1", Condition: "breakdown_continue", Direction: "short", Breakdown: &PlanBreakdownContinue{Level: 100, EntryMode: "immediate"}}
	bars := confirmBars(base, 99)
	if v := EvaluateScenarioConfirm(sc, bars, base, base+60000); v.Met {
		t.Fatalf("one 1m close cannot satisfy %s: %+v", v.Rule, v)
	}
	bars = confirmBars(base, 99, 99, 99, 99, 99, 101)
	now := base + 6*60000
	if v := EvaluateScenarioConfirm(sc, bars, base, now); !v.Met {
		t.Errorf("minute-only reclaim must not void completed break: %+v", v)
	}
	scope := VoidScope{Bars: bars, SinceMs: base}
	if err := ValidateBreakdownContinueScenarios(&PlanDoc{Scenarios: []PlanScenario{sc}}, scope, 0, 101, now); err != nil {
		t.Errorf("minute-only reclaim must not reject: %v", err)
	}
	if yes, stamp := BreakdownLevelReclaimed(100, true, bars, base, now); yes {
		t.Errorf("void facts must not use minute reclaim %s", stamp)
	}
	rows := ComputeLevelDisplacements([]ScoredLevel{{DetectedLevel: DetectedLevel{Price: 100, Label: "test"}}}, scope, now)
	if len(rows) != 1 || !rows[0].Broken {
		t.Errorf("displacement facts must retain completed break: %+v", rows)
	}
}

func TestConfirmationTruthImmediateDisplacementBeforeClose(t *testing.T) {
	base := truthBase()
	bars := confirmBars(base, 90)
	sc := PlanScenario{ID: "S1", Condition: "breakdown_continue", Direction: "short", Breakdown: &PlanBreakdownContinue{Level: 100, EntryMode: "immediate"}}
	// The explicitly separate 1m-displacement rule must preserve authorability.
	t.Setenv("BD_MAX_LEVEL_DIST_ATR", "100")
	if err := ValidateBreakdownContinueScenarios(&PlanDoc{Scenarios: []PlanScenario{sc}}, VoidScope{Bars: bars, SinceMs: base}, 1, 90, base+60000); err != nil {
		t.Fatalf("immediate displacement authoring before 5m close: %v", err)
	}
	if v := EvaluateScenarioConfirm(sc, bars, base, base+60000); v.Met {
		t.Fatalf("authorability is not confirmation: %+v", v)
	}
	rows := ComputeLevelDisplacements([]ScoredLevel{{DetectedLevel: DetectedLevel{Price: 100, Label: "test"}}}, VoidScope{Bars: bars, SinceMs: base}, base+60000)
	line := RenderDisplacementLines(rows, 1)
	if !strings.Contains(line, "none — no break") || !strings.Contains(line, "1m_displacement: 11.00 pts down") {
		t.Fatalf("planner must see the two distinct facts: %s", line)
	}
}

func TestConfirmationTruthNamedMinuteDisplacement(t *testing.T) {
	base := truthBase()
	for _, short := range []bool{true, false} {
		close := 110.0
		if short {
			close = 90
		}
		bars := confirmBars(base, close)
		before := Evaluate1mDisplacement(bars, 100, short, base, base+59999)
		at := Evaluate1mDisplacement(bars, 100, short, base, base+60000)
		if before.Observed || at.Rule != "1m_displacement" || !at.Observed || at.Pts != 11 || at.Bucket == nil || !at.Bucket.Closed {
			t.Fatalf("separate minute observation boundary short=%t: before=%+v at=%+v", short, before, at)
		}
		if EvaluateBucketClose(base, 5, base+60000).Closed {
			t.Fatal("minute observation cannot change the 5m predicate")
		}
	}
}

func TestConfirmationTruthReferenceFoundAtEpoch(t *testing.T) {
	bars := confirmBars(-60000, 100)
	at, ok := ConfirmReferenceInstant(PlanConfirm{Rule: "touch", RefPrice: 100}, bars, -60000, 1)
	if at != 0 || !ok {
		t.Fatalf("epoch event is distinguishable from missing: at=%d ok=%t", at, ok)
	}
	_, ok = ConfirmReferenceInstant(PlanConfirm{Rule: "touch", RefPrice: 200}, bars, -60000, 1)
	if ok {
		t.Fatal("missing reference must return ok=false")
	}
}

func TestConfirmationTruthSequenceSameInstantIsNotAfter(t *testing.T) {
	base := truthBase()
	bars := confirmBars(base, 99, 99, 99, 99, 99)
	for i := 0; i < 4; i++ {
		bars[i].High = 99.75
	}
	bars[4].High = 101
	v := EvaluateScenarioConfirm(truthScenario(), bars, base, base+5*60000)
	if v.Met || v.Refusal != "out_of_order" {
		t.Fatalf("touch upper bound equals the reclaim close: strictly-after is not established: %+v", v)
	}
}

func TestConfirmationTruthImmediateStored34790(t *testing.T) {
	var corpus struct {
		Plans []struct {
			RowID int     `json:"rowid"`
			Doc   PlanDoc `json:"doc"`
		} `json:"plans"`
		Bars []struct {
			OpenMs        int64 `json:"open_time_ms"`
			O, H, L, C, V float64
		} `json:"bars"`
		Decisions []struct {
			ID  int   `json:"id"`
			Now int64 `json:"snapshot_ms"`
		} `json:"decisions"`
	}
	b, err := os.ReadFile("../docs/superpowers/reports/2026-09-08-confirmation-truth-data/replay-corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &corpus); err != nil {
		t.Fatal(err)
	}
	var sc PlanScenario
	var now int64
	for _, p := range corpus.Plans {
		if p.RowID == 163 {
			for _, s := range p.Doc.Scenarios {
				if s.ID == "S1" {
					sc = s
				}
			}
		}
	}
	for _, d := range corpus.Decisions {
		if d.ID == 34790 {
			now = d.Now
		}
	}
	if sc.Breakdown == nil || now == 0 {
		t.Fatal("stored case missing")
	}
	var tape []market.Kline
	for _, b := range corpus.Bars {
		if b.OpenMs+60000 <= now {
			tape = append(tape, market.Kline{OpenTime: b.OpenMs, CloseTime: b.OpenMs + 59999, Open: b.O, High: b.H, Low: b.L, Close: b.C, Volume: b.V})
		}
	}
	if len(tape) > AISVPBarCount {
		tape = tape[len(tape)-AISVPBarCount:]
	}
	scope := VoidScopeOf(tape, time.UnixMilli(now))
	atr := StaleConfirmATR5m(scope.Bars)
	closed := BreakdownContinueState(sc, scope.Bars, scope.SinceMs, now)
	minute := Evaluate1mDisplacement(scope.Bars, sc.Breakdown.Level, true, scope.SinceMs, now)
	if closed.BreakLegPts >= bdMinDispATR()*atr || minute.Pts < bdMinDispATR()*atr {
		t.Fatalf("pin must straddle the same resolved floor: five=%g minute=%g floor=%g", closed.BreakLegPts, minute.Pts, bdMinDispATR()*atr)
	}
	if err := ValidateBreakdownContinueScenarios(&PlanDoc{Scenarios: []PlanScenario{sc}}, scope, atr, tape[len(tape)-1].Close, now); err != nil {
		t.Fatalf("approved immediate carve-out must accept stored 163/S1 decision 34790: %v", err)
	}
}

func TestConfirmationTruthAuditReceipt(t *testing.T) {
	var rows []map[string]any
	b, err := os.ReadFile("../docs/superpowers/reports/2026-09-08-confirmation-truth-data/breakdown-observations.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	count := 0
	scenarios := map[float64]bool{}
	for _, r := range rows {
		if r["validation_old"] != "PASS" && r["validation_new"] == "PASS" {
			count++
			scenarios[r["plan_row"].(float64)] = true
		}
	}
	var receipt struct {
		Count     int `json:"validation_reject_to_pass"`
		Scenarios int `json:"validation_scenarios"`
	}
	if err := json.Unmarshal(confirmationReplayReceipt, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Count != count || receipt.Scenarios != len(scenarios) {
		t.Fatalf("boot receipt must read measured replay count: %+v computed=%d/%d", receipt, count, len(scenarios))
	}
	if line := ConfirmationBootLine(); !strings.Contains(line, "audit; evaluated=") || !strings.Contains(line, "unevaluated=") {
		t.Fatalf("audit/live provenance missing: %s", line)
	}
}
