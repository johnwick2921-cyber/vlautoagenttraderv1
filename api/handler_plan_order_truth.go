package api

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"nofx/kernel"
	nt "nofx/provider/ninjatrader"
	"nofx/store"
	"nofx/trader"
)

type orderPrices struct {
	Entry  *float64 `json:"entry"`
	Stop   *float64 `json:"stop"`
	Target *float64 `json:"target"`
	Source string   `json:"source"`
	Reason string   `json:"reason,omitempty"`
}

type planOrderLeg struct {
	State             string      `json:"state"`
	Reason            string      `json:"reason,omitempty"`
	LegIndex          int         `json:"leg_index"`
	Kind              string      `json:"kind,omitempty"`
	RowID             int64       `json:"row_id,omitempty"`
	Version           int         `json:"version,omitempty"`
	ArmedUnderVersion int         `json:"armed_under_version,omitempty"`
	PlacementSeq      int         `json:"placement_seq"`
	SignalID          string      `json:"signal_id,omitempty"`
	Side              string      `json:"side,omitempty"`
	Intended          orderPrices `json:"intended"`
	Composed          orderPrices `json:"composed"`
	Accepted          orderPrices `json:"accepted"`
	BookReceivedAtMs  int64       `json:"book_received_at_ms,omitempty"`
	BookAgeMs         int64       `json:"book_age_ms"`
	BuildID           string      `json:"build_id"`
}

type planArmView struct {
	EntryPx           *float64       `json:"entry_px,omitempty"`
	ArmedUnderVersion int            `json:"armed_under_version,omitempty"`
	Side              string         `json:"side,omitempty"`
	FillQuantity      int            `json:"fill_quantity"`
	State             string         `json:"state"`
	Reason            string         `json:"reason,omitempty"`
	Legs              []planOrderLeg `json:"legs"`
}

func displayPrice(price float64) *float64 {
	if price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) {
		return nil
	}
	return &price
}

// armedMapFor selects a placement per leg within the displayed version.
// Version is the last authorization touch; ArmedUnderVersion remains separate
// original provenance. Reusing an S# never imports an older version's arm.
func (s *Server) armedMapFor(planID string, version int, doc kernel.PlanDoc, book trader.OrderBookDisplay) map[string]planArmView {
	selected := map[string]map[int]*store.ArmedOrderDB{}
	ledgerReason := "no arm recorded for this plan version"
	if s.store == nil {
		ledgerReason = "arm ledger unavailable"
	} else if rows, err := s.store.ArmedOrders().ListForPlan(planID); err != nil {
		ledgerReason = "arm ledger unavailable"
	} else {
		for _, row := range rows {
			if row.Version != version {
				continue
			}
			if selected[row.Scenario] == nil {
				selected[row.Scenario] = map[int]*store.ArmedOrderDB{}
			}
			old := selected[row.Scenario][row.LegIndex]
			if old == nil || row.PlacementSeq > old.PlacementSeq || (row.PlacementSeq == old.PlacementSeq && row.ID > old.ID) {
				selected[row.Scenario][row.LegIndex] = &row
			}
		}
	}
	out := map[string]planArmView{}
	for _, scenario := range doc.Scenarios {
		indices := map[int]bool{0: true}
		if scenario.Arm != nil {
			for i := range scenario.Arm.Legs {
				indices[i] = true
			}
		}
		for i := range selected[scenario.ID] {
			indices[i] = true
		}
		ordered := []int{}
		for i := range indices {
			ordered = append(ordered, i)
		}
		sort.Ints(ordered)
		view := planArmView{State: "UNKNOWN", Reason: ledgerReason, Legs: []planOrderLeg{}}
		for _, i := range ordered {
			leg := planOrderLeg{State: "UNKNOWN", Reason: ledgerReason, LegIndex: i,
				Intended:         orderPrices{Source: fmt.Sprintf("displayed plan v%d (with overlays)", version), Reason: "no intended arm terms"},
				Composed:         orderPrices{Source: "armed_orders", Reason: ledgerReason},
				BookReceivedAtMs: book.ReceivedAtMs, BookAgeMs: book.AgeMs, BuildID: book.BuildID}
			if arm := scenario.Arm; arm != nil {
				if len(arm.Legs) > i && i >= 0 {
					p := arm.Legs[i]
					leg.Intended.Entry, leg.Intended.Stop, leg.Intended.Target = displayPrice(p.Entry), displayPrice(p.Stop), displayPrice(p.Target)
					leg.Intended.Reason = ""
				} else if i == 0 && len(arm.Legs) == 0 {
					leg.Intended.Entry, leg.Intended.Stop, leg.Intended.Target = displayPrice(arm.Entry), displayPrice(arm.Stop), displayPrice(arm.Target)
					leg.Intended.Reason = ""
				}
			}
			row := selected[scenario.ID][i]
			if row != nil {
				leg.RowID, leg.Version, leg.ArmedUnderVersion, leg.PlacementSeq = row.ID, row.Version, row.ArmedUnderVersion, row.PlacementSeq
				leg.State, leg.Reason, leg.Kind, leg.SignalID, leg.Side = row.State, row.StateReason, row.Kind, row.SignalID, row.Side
				leg.Composed = orderPrices{Entry: displayPrice(row.EntryPx), Stop: displayPrice(row.StopPx), Target: displayPrice(row.TargetPx), Source: fmt.Sprintf("armed_orders row %d · placement %d · last touched v%d", row.ID, row.PlacementSeq, row.Version)}
			}
			leg.Accepted = acceptedOrderPrices(leg.SignalID, book)
			if len(view.Legs) == 0 {
				view.State, view.Reason = leg.State, leg.Reason
				if row != nil {
					view.EntryPx, view.ArmedUnderVersion, view.Side, view.FillQuantity = displayPrice(row.EntryPx), row.ArmedUnderVersion, row.Side, row.FillQuantity
				}
			} else if view.State != leg.State {
				view.State, view.Reason = "mixed", "see each leg"
			}
			view.Legs = append(view.Legs, leg)
		}
		out[scenario.ID] = view
	}
	return out
}

// Current accepted terms come only from uniquely identified live broker orders.
// Never backfill from intended/composed prices or the legacy acceptance journal:
// that journal includes historical dying-order records (ORDER-TRUTH DATA-2).
func acceptedOrderPrices(signal string, book trader.OrderBookDisplay) orderPrices {
	out := orderPrices{Source: "current NT8 order_snapshot", Reason: book.Reason}
	if out.Reason != "" {
		return out
	}
	if signal == "" {
		out.Reason = "no signal linked to this placement"
		return out
	}
	if book.ReceivedAtMs == 0 {
		out.Reason = "no dated broker snapshot"
		return out
	}
	missing := []string{}
	for _, leg := range []struct {
		name, label string
		dest        **float64
	}{{signal, "entry", &out.Entry}, {signal + "-sl", "stop", &out.Stop}, {signal + "-tp", "target", &out.Target}} {
		matches := []nt.NT8Order{}
		for _, order := range book.Orders {
			if order.Name == leg.name {
				matches = append(matches, order)
			}
		}
		if len(matches) != 1 {
			missing = append(missing, leg.label+": absent or ambiguous")
			continue
		}
		order := matches[0]
		if nt.ClassifyOrderState(order.State) != nt.LivenessLive {
			missing = append(missing, leg.label+": "+order.State)
			continue
		}
		price := order.LimitPrice
		if leg.label == "stop" || (leg.label == "entry" && strings.Contains(strings.ToLower(order.Type), "stop")) {
			price = order.StopPrice
		}
		*leg.dest = displayPrice(price)
		if *leg.dest == nil {
			missing = append(missing, leg.label+": no quoted price")
		}
	}
	out.Reason = strings.Join(missing, "; ")
	return out
}

func (s *Server) planOrderBook(traderID string, now time.Time) trader.OrderBookDisplay {
	if s.traderManager != nil {
		if at, err := s.traderManager.GetTrader(traderID); err == nil && at != nil {
			return at.OrderBookForDisplayAt(now)
		}
	}
	return trader.OrderBookDisplay{BuildID: "UNKNOWN", Reason: "broker connection unavailable"}
}
