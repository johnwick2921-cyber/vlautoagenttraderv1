package api

import (
	"encoding/json"
	"math"
	"nofx/kernel"
	nt "nofx/provider/ninjatrader"
	"nofx/store"
	"nofx/trader"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlanOrderTruthSelectsVersionAndPlacement(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "truth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rows := []store.ArmedOrderDB{
		{ID: 1, PlanID: "p", Scenario: "S1", Version: 2, ArmedUnderVersion: 2, PlacementSeq: 99, Side: "long", State: "filled", SignalID: "old", EntryPx: 99},
		{ID: 2, PlanID: "p", Scenario: "S1", Version: 6, ArmedUnderVersion: 5, PlacementSeq: 1, Side: "short", State: "cancelled", SignalID: "first", EntryPx: 101},
		{ID: 3, PlanID: "p", Scenario: "S1", Version: 6, ArmedUnderVersion: 5, PlacementSeq: 2, Side: "short", State: "working", SignalID: "chosen", EntryPx: 102, StopPx: 112, TargetPx: 92},
		{ID: 4, PlanID: "p", Scenario: "S1", Version: 6, ArmedUnderVersion: 6, PlacementSeq: 1, LegIndex: 1, State: "armed", EntryPx: 103},
		{ID: 5, PlanID: "p", Scenario: "S2", Version: 2, PlacementSeq: 1, State: "working", EntryPx: 999},
	}
	if err := st.GormDB().Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	doc := kernel.PlanDoc{Scenarios: []kernel.PlanScenario{{ID: "S1", Arm: &kernel.PlanArmSpec{Entry: 100, Stop: 110, Target: 90}}, {ID: "S2"}}}
	book := trader.OrderBookDisplay{ReceivedAtMs: time.Now().UnixMilli(), BuildID: "h1", Orders: []nt.NT8Order{
		{Name: "chosen", State: "Working", Type: "limit", LimitPrice: 102.25},
		{Name: "chosen-sl", State: "Accepted", Type: "stop", StopPrice: 112.25},
		{Name: "chosen-tp", State: "Working", Type: "limit", LimitPrice: 92.25},
		{Name: "old-sl", State: "Working", StopPrice: 888},
	}}
	s := &Server{store: st}
	got := s.armedMapFor("p", 6, doc, book)
	leg := got["S1"].Legs[0]
	if leg.RowID != 3 || leg.ArmedUnderVersion != 5 || leg.Version != 6 || leg.PlacementSeq != 2 || leg.Side != "short" {
		t.Fatalf("wrong provenance: %+v", leg)
	}
	if got["S1"].State != "mixed" || len(got["S1"].Legs) != 2 || got["S1"].Legs[1].RowID != 4 {
		t.Fatalf("lost split leg: %+v", got["S1"])
	}
	for _, p := range []struct {
		got  *float64
		want float64
	}{{leg.Intended.Entry, 100}, {leg.Composed.Entry, 102}, {leg.Accepted.Entry, 102.25}, {leg.Intended.Stop, 110}, {leg.Composed.Stop, 112}, {leg.Accepted.Stop, 112.25}, {leg.Accepted.Target, 92.25}} {
		if p.got == nil || *p.got != p.want {
			t.Fatalf("wrong price: got %v want %v", p.got, p.want)
		}
	}
	if got["S2"].State != "UNKNOWN" || got["S2"].Legs[0].Composed.Entry != nil {
		t.Fatal("imported old-version S2")
	}
	// A newer terminal placement wins over an older working row; history is not
	// silently chosen by liveness priority. The unique placement constraint remains intact.
	newer := store.ArmedOrderDB{ID: 6, PlanID: "p", Scenario: "S1", Version: 6, PlacementSeq: 3, State: "cancelled", SignalID: "newer"}
	if err := st.GormDB().Create(&newer).Error; err != nil {
		t.Fatal(err)
	}
	if next := s.armedMapFor("p", 6, doc, book)["S1"].Legs[0]; next.RowID != 6 || next.Accepted.Entry != nil {
		t.Fatalf("wrong latest placement: %+v", next)
	}
}

func TestAcceptedOrderTruthRejectsUnprovenTerms(t *testing.T) {
	for _, state := range []string{"Submitted", "CancelPending", "CancelSubmitted", "TriggerPending", "Unknown", "Filled", "Cancelled"} {
		book := trader.OrderBookDisplay{ReceivedAtMs: 1, Orders: []nt.NT8Order{{Name: "s-sl", State: state, StopPrice: 112}}}
		if got := acceptedOrderPrices("s", book); got.Stop != nil {
			t.Errorf("%s became accepted: %+v", state, got)
		}
	}
	book := trader.OrderBookDisplay{ReceivedAtMs: 1, Orders: []nt.NT8Order{{Name: "s", State: "Accepted", Type: "stop", StopPrice: 102, LimitPrice: 999}, {Name: "s-sl", State: "Accepted", StopPrice: 112}, {Name: "s-tp", State: "Working", LimitPrice: 92}}}
	if got := acceptedOrderPrices("s", book); got.Entry == nil || *got.Entry != 102 {
		t.Fatal("stop entry must use trigger price")
	}
	book.Orders = append(book.Orders, nt.NT8Order{Name: "s-sl", State: "Cancelled", StopPrice: 111})
	if got := acceptedOrderPrices("s", book); got.Stop != nil {
		t.Fatal("ambiguous stop accepted")
	}
	book.Reason = "broker snapshot is stale"
	if got := acceptedOrderPrices("s", book); got.Entry != nil || got.Stop != nil || got.Target != nil {
		t.Fatal("stale terms accepted")
	}
}

// legacyTruthRows are rows exactly as the ledger held them before W3: no
// policy. Row 12 also carries W3 receipt columns WITHOUT a policy — the shape a
// receipts writer that stamps every fill would leave — and must still render
// as legacy.
func legacyTruthRows() []store.ArmedOrderDB {
	f := func(v float64) *float64 { return &v }
	ms := func(v int64) *int64 { return &v }
	return []store.ArmedOrderDB{
		{ID: 11, PlanID: "p", Scenario: "S1", Version: 4, ArmedUnderVersion: 3, PlacementSeq: 1, Side: "long", Kind: "limit", State: "working", SignalID: "leg-w", EntryPx: 31010, StopPx: 30990, TargetPx: 31050},
		{ID: 12, PlanID: "p", Scenario: "S2", Version: 4, ArmedUnderVersion: 4, PlacementSeq: 2, Side: "short", Kind: "limit", State: "filled", StateReason: "fill@31020.5", SignalID: "leg-f", EntryPx: 31020, StopPx: 31040, TargetPx: 30980, FillPrice: 31020.5, FillQuantity: 1,
			FilledAtMs: ms(1790190000000), FillSlippageTicks: f(2), EvalPrice: f(31019), ZoneLo: f(31015), ZoneHi: f(31020)},
		{ID: 13, PlanID: "p", Scenario: "S3", Version: 4, ArmedUnderVersion: 4, PlacementSeq: 1, LegIndex: 1, LegCount: 2, Side: "long", Kind: "stop_entry", State: "cancelled", StateReason: "price through the trigger", EntryPx: 30950},
	}
}

func legacyTruthJSON(t *testing.T) string {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rows := legacyTruthRows()
	if err := st.GormDB().Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	doc := kernel.PlanDoc{Scenarios: []kernel.PlanScenario{
		{ID: "S1", Arm: &kernel.PlanArmSpec{Entry: 31005, Stop: 30990, Target: 31050}},
		{ID: "S2", Arm: &kernel.PlanArmSpec{Entry: 31018, Stop: 31040, Target: 30980}},
		{ID: "S3"},
	}}
	book := trader.OrderBookDisplay{ReceivedAtMs: 1790189990000, AgeMs: 4000, BuildID: "h1", Orders: []nt.NT8Order{
		{Name: "leg-w", State: "Working", Type: "limit", LimitPrice: 31010},
		{Name: "leg-w-sl", State: "Accepted", Type: "stop", StopPrice: 30990},
		{Name: "leg-w-tp", State: "Working", Type: "limit", LimitPrice: 31050},
	}}
	b, err := json.Marshal((&Server{store: st}).armedMapFor("p", 4, doc, book))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// W3 (h) — L4: a LEGACY row's plan-API JSON is BYTE-IDENTICAL to the base.
// The literal below was captured by running this exact fixture through
// armedMapFor at e74fce17 (before any W3 field existed on planOrderLeg).
func TestPlanOrderTruthLegacyJSONByteIdentical(t *testing.T) {
	got := legacyTruthJSON(t)
	const want = `{"S1":{"entry_px":31010,"armed_under_version":3,"side":"long","fill_quantity":0,"state":"working","legs":[{"state":"working","leg_index":0,"kind":"limit","row_id":11,"version":4,"armed_under_version":3,"placement_seq":1,"signal_id":"leg-w","side":"long","intended":{"entry":31005,"stop":30990,"target":31050,"source":"displayed plan v4 (with overlays)"},"composed":{"entry":31010,"stop":30990,"target":31050,"source":"armed_orders row 11 · placement 1 · last touched v4"},"accepted":{"entry":31010,"stop":30990,"target":31050,"source":"current NT8 order_snapshot"},"book_received_at_ms":1790189990000,"book_age_ms":4000,"build_id":"h1"}]},"S2":{"entry_px":31020,"armed_under_version":4,"side":"short","fill_quantity":1,"state":"filled","reason":"fill@31020.5","legs":[{"state":"filled","reason":"fill@31020.5","leg_index":0,"kind":"limit","row_id":12,"version":4,"armed_under_version":4,"placement_seq":2,"signal_id":"leg-f","side":"short","intended":{"entry":31018,"stop":31040,"target":30980,"source":"displayed plan v4 (with overlays)"},"composed":{"entry":31020,"stop":31040,"target":30980,"source":"armed_orders row 12 · placement 2 · last touched v4"},"accepted":{"entry":null,"stop":null,"target":null,"source":"current NT8 order_snapshot","reason":"entry: absent or ambiguous; stop: absent or ambiguous; target: absent or ambiguous"},"book_received_at_ms":1790189990000,"book_age_ms":4000,"build_id":"h1"}]},"S3":{"fill_quantity":0,"state":"mixed","reason":"see each leg","legs":[{"state":"UNKNOWN","reason":"no arm recorded for this plan version","leg_index":0,"placement_seq":0,"intended":{"entry":null,"stop":null,"target":null,"source":"displayed plan v4 (with overlays)","reason":"no intended arm terms"},"composed":{"entry":null,"stop":null,"target":null,"source":"armed_orders","reason":"no arm recorded for this plan version"},"accepted":{"entry":null,"stop":null,"target":null,"source":"current NT8 order_snapshot","reason":"no signal linked to this placement"},"book_received_at_ms":1790189990000,"book_age_ms":4000,"build_id":"h1"},{"state":"cancelled","reason":"price through the trigger","leg_index":1,"kind":"stop_entry","row_id":13,"version":4,"armed_under_version":4,"placement_seq":1,"side":"long","intended":{"entry":null,"stop":null,"target":null,"source":"displayed plan v4 (with overlays)","reason":"no intended arm terms"},"composed":{"entry":30950,"stop":null,"target":null,"source":"armed_orders row 13 · placement 1 · last touched v4"},"accepted":{"entry":null,"stop":null,"target":null,"source":"current NT8 order_snapshot","reason":"no signal linked to this placement"},"book_received_at_ms":1790189990000,"book_age_ms":4000,"build_id":"h1"}]}}`
	if got != want {
		t.Fatalf("legacy plan-API JSON drifted\n got: %s\nwant: %s", got, want)
	}
}

// W3 (h) — a market_in_zone row's receipts reach the card through the view the
// plan handler serves (armedMapFor, handler_plan.go "armed"). Absent stays
// absent (never 0); a measured 0 slippage is a value and stays.
func TestPlanOrderTruthMarketInZoneReceipts(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "miz.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	f := func(v float64) *float64 { return &v }
	ms := func(v int64) *int64 { return &v }
	rows := []store.ArmedOrderDB{
		{ID: 21, PlanID: "p", Scenario: "S1", Version: 2, ArmedUnderVersion: 2, PlacementSeq: 1, Side: "long", Kind: "limit", Condition: "reclaim", State: "filled", StateReason: "fill@31006", SignalID: "miz-1", EntryPx: 31010, StopPx: 30990, TargetPx: 31050, FillPrice: 31006, FillQuantity: 1,
			Policy: "market_in_zone", ZoneLo: f(31000), ZoneHi: f(31010), ZoneProvenance: "entry_geometry subrange", PlannedEntryPx: f(31005), EvalPrice: f(31004.25), EvalBarMs: ms(1790190000000),
			PlacedAtMs: ms(1790190001000), FilledAtMs: ms(1790190001500), FillSlippageTicks: f(0), LastVerdict: "inside", LastVerdictMs: ms(1790190001000)},
		{ID: 22, PlanID: "p", Scenario: "S2", Version: 2, ArmedUnderVersion: 2, Side: "short", Kind: "limit", State: "armed", EntryPx: 31100, Policy: "market_in_zone"},
	}
	if err := st.GormDB().Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	doc := kernel.PlanDoc{Scenarios: []kernel.PlanScenario{{ID: "S1"}, {ID: "S2"}}}
	got := (&Server{store: st}).armedMapFor("p", 2, doc, trader.OrderBookDisplay{ReceivedAtMs: 1, BuildID: "h1"})

	leg := got["S1"].Legs[0]
	if leg.Policy != "market_in_zone" || leg.Verdict != "inside" {
		t.Fatalf("policy/verdict not carried: %+v", leg)
	}
	for _, p := range []struct {
		name string
		got  *float64
		want float64
	}{{"planned_entry", leg.PlannedEntry, 31005}, {"zone_lo", leg.ZoneLo, 31000}, {"zone_hi", leg.ZoneHi, 31010}, {"eval_price", leg.EvalPrice, 31004.25}, {"fill_price", leg.FillPrice, 31006}, {"fill_slippage_ticks", leg.FillSlippageTicks, 0}, {"composed.entry (the limit)", leg.Composed.Entry, 31010}} {
		if p.got == nil || *p.got != p.want {
			t.Fatalf("%s: got %v want %v", p.name, p.got, p.want)
		}
	}
	for _, p := range []struct {
		name string
		got  *int64
		want int64
	}{{"eval_bar_ms", leg.EvalBarMs, 1790190000000}, {"placed_at_ms", leg.PlacedAtMs, 1790190001000}, {"filled_at_ms", leg.FilledAtMs, 1790190001500}} {
		if p.got == nil || *p.got != p.want {
			t.Fatalf("%s: got %v want %v", p.name, p.got, p.want)
		}
	}
	b, err := json.Marshal(leg)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"policy":"market_in_zone"`, `"planned_entry":31005`, `"zone_lo":31000`, `"zone_hi":31010`, `"eval_price":31004.25`, `"eval_bar_ms":1790190000000`,
		`"fill_price":31006`, `"fill_slippage_ticks":0`, `"placed_at_ms":1790190001000`, `"filled_at_ms":1790190001500`, `"verdict":"inside"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("policy leg JSON lacks %s: %s", want, b)
		}
	}

	// S2: armed, nothing measured — the policy is present, every receipt is
	// ABSENT from the JSON (never 0, never "").
	b, err = json.Marshal(got["S2"].Legs[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"policy":"market_in_zone"`) {
		t.Fatalf("policy missing on an unmeasured policy row: %s", b)
	}
	for _, key := range []string{"planned_entry", "zone_lo", "zone_hi", "eval_price", "eval_bar_ms", "fill_price", "fill_slippage_ticks", "placed_at_ms", "filled_at_ms", "verdict"} {
		if strings.Contains(string(b), `"`+key+`"`) {
			t.Errorf("absent %s was fabricated: %s", key, b)
		}
	}

	// A value JSON cannot carry never breaks the whole plan response and is
	// never turned into 0.
	nan := math.NaN()
	var odd planOrderLeg
	withEntryPolicy(&odd, &store.ArmedOrderDB{Policy: "market_in_zone", FillSlippageTicks: &nan, ZoneLo: f(0), EvalBarMs: ms(0)})
	if odd.FillSlippageTicks != nil || odd.ZoneLo != nil || odd.EvalBarMs != nil {
		t.Fatalf("non-values surfaced: %+v", odd)
	}
	if _, err := json.Marshal(odd); err != nil {
		t.Fatalf("NaN slippage broke the JSON: %v", err)
	}
}
