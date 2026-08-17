package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/market"
	"nofx/store"
)

// H9 — the planner prompt asserted a read-set ("D/4h/1h/15m: structure read")
// while the fetch was hardcoded 1d/1h/5m. Now the configured set, the fetched
// set and the prompt lines can never diverge: each configured TF is fetched and
// each line reports truth — "structure read" or "unavailable".

func TestStructureSummaryLinesFetchConfiguredSet(t *testing.T) {
	var requested []string
	fetch := func(tf string, count int) []market.Kline {
		requested = append(requested, tf)
		now := time.Now().UnixMilli()
		return []market.Kline{{OpenTime: now - 600_000, Close: 100, CloseTime: now - 300_000}}
	}
	lines := structureSummaryLines(fetch, []string{"D", "4h", "1h", "15m"})

	wantReq := []string{"1d", "4h", "1h", "15m"} // "D" maps to the provider's "1d"
	if len(requested) != len(wantReq) {
		t.Fatalf("fetched set = %v, want %v", requested, wantReq)
	}
	for i, r := range wantReq {
		if requested[i] != r {
			t.Fatalf("fetch[%d] = %q, want %q", i, requested[i], r)
		}
	}
	wantLines := []string{"D: structure read", "4h: structure read", "1h: structure read", "15m: structure read"}
	if len(lines) != len(wantLines) {
		t.Fatalf("prompt lines = %v, want %v", lines, wantLines)
	}
	for i, l := range wantLines {
		if lines[i] != l {
			t.Fatalf("line[%d] = %q, want %q", i, lines[i], l)
		}
	}
}

func TestStructureSummaryLinesMissingTFSurfacesUnavailable(t *testing.T) {
	// 4h is dark: the provider has no bars for it. The line must say so — never
	// claim a read that did not happen.
	fetch := func(tf string, count int) []market.Kline {
		if tf == "4h" {
			return nil
		}
		now := time.Now().UnixMilli()
		return []market.Kline{{OpenTime: now - 600_000, Close: 100, CloseTime: now - 300_000}}
	}
	lines := structureSummaryLines(fetch, []string{"D", "4h", "1h"})
	want := []string{"D: structure read", "4h: unavailable", "1h: structure read"}
	for i, l := range want {
		if lines[i] != l {
			t.Fatalf("line[%d] = %q, want %q (all: %v)", i, lines[i], l, lines)
		}
	}
}

func TestStructureSummaryLinesNoProviderAllUnavailable(t *testing.T) {
	lines := structureSummaryLines(nil, []string{"D", "1h"})
	if lines[0] != "D: unavailable" || lines[1] != "1h: unavailable" {
		t.Fatalf("nil provider must mark every TF unavailable, got %v", lines)
	}
}

// TestAssemblePlannerInputHonestStructureLines proves the SEAM end to end: the
// configured planner_timeframes drive the prompt lines that land in the planner
// input package (not a hardcoded header), and a dark TF reads "unavailable".
func TestAssemblePlannerInputHonestStructureLines(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { st.Plan().Close(); _ = st.Close() })

	prev := market.FuturesBarsProvider
	defer func() { market.FuturesBarsProvider = prev }()
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline {
		if tf == "4h" {
			return nil // dark
		}
		now := time.Now().UnixMilli()
		return []market.Kline{{OpenTime: now - 600_000, High: 15610, Low: 15590, Close: 15600, CloseTime: now - 300_000}}
	}

	at := &AutoTrader{
		id: "t1", exchange: "ninjatrader", store: st,
		config: AutoTraderConfig{NinjaTraderSymbol: "MNQ", StrategyConfig: &store.StrategyConfig{
			DayPlan: &store.DayPlanConfig{PlanEnabled: true, PlannerTimeframes: []string{"D", "4h", "1h", "15m"}},
		}},
	}

	input := at.assemblePlannerInput("NY", "2026-08-15")
	got := input.StructureSummary
	want := []string{"D: structure read", "4h: unavailable", "1h: structure read", "15m: structure read"}
	if len(got) != len(want) {
		t.Fatalf("planner input structure lines = %v, want %v", got, want)
	}
	for i, l := range want {
		if got[i] != l {
			t.Fatalf("planner input line[%d] = %q, want %q (all: %v)", i, got[i], l, got)
		}
	}
}
