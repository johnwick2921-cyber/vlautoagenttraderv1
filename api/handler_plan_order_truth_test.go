package api

import (
	"nofx/kernel"
	nt "nofx/provider/ninjatrader"
	"nofx/store"
	"nofx/trader"
	"path/filepath"
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
