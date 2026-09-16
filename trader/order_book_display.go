package trader

import (
	"strings"
	"time"

	nt "nofx/provider/ninjatrader"
)

// OrderBookDisplay is one received account/symbol-scoped frame for read-only
// display. No account identity leaves this projection. Missing/stale books do
// not become an empty accepted book.
type OrderBookDisplay struct {
	Orders       []nt.NT8Order
	ReceivedAtMs int64
	AgeMs        int64
	BuildID      string
	Reason       string
}

func (at *AutoTrader) OrderBookForDisplayAt(now time.Time) OrderBookDisplay {
	view := OrderBookDisplay{BuildID: "UNKNOWN", Reason: "no broker snapshot received"}
	cache, account, symbol := at.brokerBook()
	if cache == nil || strings.TrimSpace(account) == "" || strings.TrimSpace(symbol) == "" {
		return view
	}
	snap, received, ok := cache.LatestReceived(account)
	if !ok || received.IsZero() {
		return view
	}
	view.ReceivedAtMs, view.AgeMs = received.UnixMilli(), now.Sub(received).Milliseconds()
	if build := strings.TrimSpace(snap.BuildID); build != "" {
		view.BuildID = build
	}
	if view.AgeMs < 0 {
		view.Reason = "broker receipt is ahead of server clock"
		return view
	}
	if now.Sub(received) > 2*OrderSnapshotInterval() {
		view.Reason = "broker snapshot is stale"
		return view
	}
	view.Reason = ""
	view.Orders = []nt.NT8Order{}
	for _, order := range snap.Orders {
		if strings.EqualFold(strings.TrimSpace(order.Symbol), strings.TrimSpace(symbol)) {
			view.Orders = append(view.Orders, order)
		}
	}
	return view
}
