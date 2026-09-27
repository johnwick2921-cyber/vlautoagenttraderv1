package api

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"nofx/kernel"
	"nofx/store"
	"nofx/trader"
)

// W-EXEC-TRUTH W5 (display) — the plan API's order-truth leg carries the
// machine source of a Picture scenario's ledger row: source, source_ref (the
// opportunity key), rule (source_rule), method (policy + kind) and the
// eligibility deadline. Copied ONLY when the row carries a Source, so every
// planner row — legacy or W3 policy — serves byte-identical JSON.

// plannerPolicyTruthJSON is a W3 planner policy row (market_in_zone, no
// source) run through armedMapFor — the view handlePlanToday serves as "armed".
func plannerPolicyTruthJSON(t *testing.T) string {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "planner-policy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	f := func(v float64) *float64 { return &v }
	ms := func(v int64) *int64 { return &v }
	rows := []store.ArmedOrderDB{
		{ID: 31, PlanID: "p", Scenario: "S1", Version: 3, ArmedUnderVersion: 3, PlacementSeq: 1, Side: "long", Kind: "limit", Condition: "reclaim", State: "working", SignalID: "miz-w", EntryPx: 31010, StopPx: 30990, TargetPx: 31050,
			Policy: "market_in_zone", ZoneLo: f(31000), ZoneHi: f(31010), ZoneProvenance: "entry_geometry subrange", PlannedEntryPx: f(31005), EvalPrice: f(31004.25), EvalBarMs: ms(1790190000000),
			PlacedAtMs: ms(1790190001000), LastVerdict: "inside", LastVerdictMs: ms(1790190001000)},
	}
	if err := st.GormDB().Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	doc := kernel.PlanDoc{Scenarios: []kernel.PlanScenario{{ID: "S1", Arm: &kernel.PlanArmSpec{Entry: 31005, Stop: 30990, Target: 31050}}}}
	b, err := json.Marshal((&Server{store: st}).armedMapFor("p", 3, doc, trader.OrderBookDisplay{ReceivedAtMs: 1790189990000, AgeMs: 4000, BuildID: "h1"}))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// L4 — a PLANNER policy row is byte-identical to the base. The literal was
// captured by running this exact fixture through armedMapFor at 46f529f2 (the
// W5 foundation, before any W5 field existed on planOrderLeg).
func TestPlanOrderTruthPlannerPolicyJSONByteIdentical(t *testing.T) {
	got := plannerPolicyTruthJSON(t)
	const want = `{"S1":{"entry_px":31010,"armed_under_version":3,"side":"long","fill_quantity":0,"state":"working","legs":[{"state":"working","leg_index":0,"kind":"limit","row_id":31,"version":3,"armed_under_version":3,"placement_seq":1,"signal_id":"miz-w","side":"long","intended":{"entry":31005,"stop":30990,"target":31050,"source":"displayed plan v3 (with overlays)"},"composed":{"entry":31010,"stop":30990,"target":31050,"source":"armed_orders row 31 · placement 1 · last touched v3"},"accepted":{"entry":null,"stop":null,"target":null,"source":"current NT8 order_snapshot","reason":"entry: absent or ambiguous; stop: absent or ambiguous; target: absent or ambiguous"},"book_received_at_ms":1790189990000,"book_age_ms":4000,"build_id":"h1","policy":"market_in_zone","planned_entry":31005,"zone_lo":31000,"zone_hi":31010,"eval_price":31004.25,"eval_bar_ms":1790190000000,"placed_at_ms":1790190001000,"verdict":"inside"}]}}`
	if got != want {
		t.Fatalf("planner policy plan-API JSON drifted\n got: %s\nwant: %s", got, want)
	}
}

// sourceTruthRows: P1 is a Picture scenario's ledger row (source set, policy
// market_in_zone, kind limit) ARMED with its deadline; P2 a filled one; S1 a
// planner row in the same plan.
func sourceTruthRows() []store.ArmedOrderDB {
	f := func(v float64) *float64 { return &v }
	ms := func(v int64) *int64 { return &v }
	return []store.ArmedOrderDB{
		{ID: 41, PlanID: "p", Scenario: "P1", Version: 2, ArmedUnderVersion: 2, Side: "long", Kind: "limit", Condition: "acceptance", State: "armed", EntryPx: 31012,
			StopPx: 30990, TargetPx: 31060, Policy: "market_in_zone", ZoneLo: f(31004), ZoneHi: f(31012), PlannedEntryPx: f(31004),
			Source: "picture", SourceRef: "t1|acct|MNQ|long|resistance|1790186400000|1790190000000", SourceRule: "h1_close_break",
			EligibleUntilMs: ms(1790190010000), SourceRunEpoch: ms(7)},
		{ID: 42, PlanID: "p", Scenario: "P2", Version: 2, ArmedUnderVersion: 2, PlacementSeq: 1, Side: "short", Kind: "limit", Condition: "acceptance", State: "filled", SignalID: "pic-2", EntryPx: 31100,
			Policy: "market_in_zone", Source: "picture", SourceRef: "t1|acct|MNQ|short|support|1790186400000|1790193600000", SourceRule: "h1_close_break"},
		{ID: 43, PlanID: "p", Scenario: "S1", Version: 2, ArmedUnderVersion: 2, Side: "long", Kind: "limit", State: "armed", EntryPx: 31010, Policy: "market_in_zone"},
	}
}

// A Picture row's leg carries source, source_ref, rule, method and its
// deadline — through armedMapFor, the view handlePlanToday serves as "armed".
// A deadline the ledger does not hold is ABSENT (never 0); a planner row in
// the same plan carries none of the keys.
func TestPlanOrderTruthMachineSourceFields(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "source.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rows := sourceTruthRows()
	if err := st.GormDB().Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	doc := kernel.PlanDoc{Scenarios: []kernel.PlanScenario{{ID: "S1"}, {ID: "P1"}, {ID: "P2"}}}
	got := (&Server{store: st}).armedMapFor("p", 2, doc, trader.OrderBookDisplay{ReceivedAtMs: 1, BuildID: "h1"})

	p1 := got["P1"].Legs[0]
	if p1.Source != "picture" || p1.SourceRef != rows[0].SourceRef || p1.Rule != "h1_close_break" || p1.Method != "market_in_zone limit" {
		t.Fatalf("P1 source receipt not carried: %+v", p1)
	}
	if p1.EligibleUntilMs == nil || *p1.EligibleUntilMs != 1790190010000 {
		t.Fatalf("P1 deadline: got %v want 1790190010000", p1.EligibleUntilMs)
	}
	b, err := json.Marshal(p1)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"source":"picture"`, `"source_ref":"t1|acct|MNQ|long|resistance|1790186400000|1790190000000"`,
		`"rule":"h1_close_break"`, `"method":"market_in_zone limit"`, `"eligible_until_ms":1790190010000`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("P1 leg JSON lacks %s: %s", want, b)
		}
	}

	// P2: no deadline in the ledger → the key is absent, never 0.
	b, err = json.Marshal(got["P2"].Legs[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"source":"picture"`) || strings.Contains(string(b), `"eligible_until_ms"`) {
		t.Fatalf("P2: source must be present and the unknown deadline absent: %s", b)
	}

	// S1: a planner row beside them — none of the W5 keys.
	b, err = json.Marshal(got["S1"].Legs[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"source_ref", "rule", "method", "eligible_until_ms"} {
		if strings.Contains(string(b), `"`+key+`"`) {
			t.Errorf("planner leg fabricated %s: %s", key, b)
		}
	}
	// "source" also names orderPrices.source; the leg-level key is the one
	// that must not appear — the leg JSON carries it only as a top-level key.
	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		t.Fatal(err)
	}
	if _, ok := top["source"]; ok {
		t.Errorf("planner leg carries a top-level source: %s", b)
	}
}

// The method is read from the ledger, never guessed: a sourced row with no
// policy names none; a policy row with no kind names the policy alone.
func TestArmPlacementMethodReadsTheLedger(t *testing.T) {
	var leg planOrderLeg
	withMachineSource(&leg, &store.ArmedOrderDB{Source: "picture", Kind: "limit"})
	if leg.Source != "picture" || leg.Method != "" {
		t.Fatalf("no policy must name no method: %+v", leg)
	}
	if m := armPlacementMethod(&store.ArmedOrderDB{Policy: "market_in_zone"}); m != "market_in_zone" {
		t.Fatalf("policy without kind: got %q", m)
	}
	var planner planOrderLeg
	withMachineSource(&planner, &store.ArmedOrderDB{Policy: "market_in_zone", Kind: "limit", SourceRef: "stray", SourceRule: "stray"})
	if planner.Source != "" || planner.SourceRef != "" || planner.Rule != "" || planner.Method != "" || planner.EligibleUntilMs != nil {
		t.Fatalf("a row with no Source copied machine fields: %+v", planner)
	}
}
