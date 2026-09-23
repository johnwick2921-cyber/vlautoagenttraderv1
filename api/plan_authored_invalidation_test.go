package api

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// W2 A1/A2 — the card's authored_invalidation block is READ from the row the
// handler serves (GetLatestPlanForTraderSession, as handlePlanToday reads it):
// a W2 row carries its policy, clocks and groups; a pre-W2 row says n/a.
func TestAuthoredInvalidationViewReadsTheRow(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })
	read, publish := time.UnixMilli(1790145027000), time.UnixMilli(1790146307000)
	check := kernel.EvaluateBornCheck(&kernel.PlanDoc{}, nil, read, publish).Check
	const tid = "trader-1"
	if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: store.MakePlanIDForTrader(tid, "2026-09-22", "ASIA"), StrategyID: tid, TradeDate: "2026-09-22", Session: "ASIA", Doc: "{}"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: store.MakePlanIDForTrader(tid, "2026-09-23", "LONDON"), StrategyID: tid, TradeDate: "2026-09-23", Session: "LONDON", Doc: "{}",
		ReadClockMs: check.ReadClockPtr(), PublishClockMs: check.PublishClockPtr(), BornCheck: (&check).JSONPtr()}); err != nil {
		t.Fatal(err)
	}
	w2, _ := st.Plan().GetLatestPlanForTraderSession("2026-09-23", "LONDON", tid)
	b, _ := json.Marshal(authoredInvalidationFor(w2))
	if got := string(b); got != `{"recorded":true,"policy":"enforced (grammar)","read_clock_ms":1790145027000,"publish_clock_ms":1790146307000,"groups":[1790145000000,1790145300000,1790145600000,1790145900000]}` {
		t.Fatalf("W2 row view: %s", got)
	}
	pre, _ := st.Plan().GetLatestPlanForTraderSession("2026-09-22", "ASIA", tid)
	b, _ = json.Marshal(authoredInvalidationFor(pre))
	if got := string(b); got != `{"recorded":false,"read_clock_ms":null,"publish_clock_ms":null}` || strings.Contains(got, "policy") {
		t.Fatalf("a pre-W2 row must say recorded=false with null clocks, never an inferred policy: %s", got)
	}
}
