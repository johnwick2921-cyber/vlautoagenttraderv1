package trader

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// ── S1 — THE CALL SITE, on TODAY's bars (store, read-only export 2026-09-16) ──
//
// Fixture: MNQ 12-26 NT8-only rows the store held at 2026-09-16 (1h=61,
// 4h=22, 1d=10 — the young contract; the live ring the planner reads is
// deeper, this is what the store could give). The production function
// structureMapForRead runs with the knob ON and OFF; the map must be absent
// when off, present with all three TFs when on, and the prompt/doc/log
// surfaces must carry it — through the same functions the read uses.

func loadStructureFixture(t *testing.T, tf string) []market.Kline {
	t.Helper()
	b, err := os.ReadFile("testdata/structure_mnq_12-26_" + tf + "_2026-09-16.json")
	if err != nil {
		t.Fatal(err)
	}
	// sqlite -json emits lower-case keys t/o/h/l/c/v
	var raw []map[string]float64
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	out := make([]market.Kline, 0, len(raw))
	for _, r := range raw {
		out = append(out, market.Kline{OpenTime: int64(r["t"]), Open: r["o"], High: r["h"], Low: r["l"], Close: r["c"], Volume: r["v"]})
	}
	return out
}

func withStructureFixtureProvider(t *testing.T) {
	t.Helper()
	prev := market.FuturesBarsProvider
	fx := map[string][]market.Kline{"1h": loadStructureFixture(t, "1h"), "4h": loadStructureFixture(t, "4h"), "1d": loadStructureFixture(t, "1d")}
	market.FuturesBarsProvider = func(symbol, tf string, n int) []market.Kline { return fx[tf] }
	t.Cleanup(func() { market.FuturesBarsProvider = prev })
}

func TestStructureMapAtTheReadIsAbsentOffAndPresentOn(t *testing.T) {
	withStructureFixtureProvider(t)
	now := time.UnixMilli(1789600000000) // 2026-09-16 ~17:0x CT, after the newest fixture bar
	pool := []kernel.ScoredLevel{{DetectedLevel: kernel.DetectedLevel{Kind: kernel.LevelKind("OB"), Lo: 29100, Hi: 29130, TF: "4h", HTF: true, Label: "OB·4h"}, Score: 6, Fresh: "fresh"}}

	off := &AutoTrader{config: AutoTraderConfig{StrategyConfig: &store.StrategyConfig{DayPlan: &store.DayPlanConfig{}}}}
	if m := off.structureMapForRead("MNQ", "MNQ 12-26", pool, 29200, now); m != nil {
		t.Fatalf("knob OFF must yield nil (absent), got %+v", m)
	}
	if en, known := off.structureMapEnabled(); en || !known {
		t.Fatalf("off/known: %v/%v", en, known)
	}

	on := true
	at := &AutoTrader{config: AutoTraderConfig{StrategyConfig: &store.StrategyConfig{DayPlan: &store.DayPlanConfig{StructureMap: &on}}}}
	m := at.structureMapForRead("MNQ", "MNQ 12-26", pool, 29200, now)
	if m == nil {
		t.Fatal("knob ON with bars on every TF must yield a map")
	}
	if m.Contract != "MNQ 12-26" {
		t.Fatalf("contract=%q", m.Contract)
	}
	for _, tf := range kernel.StructureMapTFs {
		st, ok := m.TFs[tf]
		if !ok {
			t.Fatalf("%s absent", tf)
		}
		if st.Trend != "up" && st.Trend != "down" && st.Trend != "range" {
			t.Fatalf("%s trend=%q", tf, st.Trend)
		}
	}
	// what the store could give today: 61 1h / 22 4h / 10 1d closed bars
	if m.TFs["1h"].Bars < 55 || m.TFs["4h"].Bars < 18 || m.TFs["D"].Bars < 8 {
		t.Fatalf("bars read: 1h=%d 4h=%d D=%d", m.TFs["1h"].Bars, m.TFs["4h"].Bars, m.TFs["D"].Bars)
	}
	if m.TFs["1h"].LastSwingHigh == nil || m.TFs["1h"].LastSwingLow == nil {
		t.Fatalf("61 1h bars must yield both last swings: %+v", m.TFs["1h"])
	}
	if len(m.TFs["4h"].Zones) != 1 || m.TFs["4h"].Zones[0].Kind != "OB" {
		t.Fatalf("4h zones=%+v (the pool's own OB·4h, label intact)", m.TFs["4h"].Zones)
	}
	// the surfaces: prompt section, doc field, 🗺 line
	p := kernel.BuildPlannerPrompt(kernel.PlannerInput{TradeDate: "2026-09-16", Session: "NY", Price: 29200, Structure: m})
	if !strings.Contains(p, "## STRUCTURE — bias only, not entries") || !strings.Contains(p, "OB 29100.00–29130.00 (fresh)") {
		t.Fatalf("prompt lacks the section with the fixture's zone:\n%s", p[:min(len(p), 1200)])
	}
	doc := kernel.PlanDoc{Structure: m}
	js, _ := json.Marshal(doc)
	if !strings.Contains(string(js), `"structure":{"as_of_ms":1789600000000,"contract":"MNQ 12-26","tfs":{`) {
		t.Fatalf("doc JSON lacks the structure block: %s", string(js)[:min(len(js), 300)])
	}
	jsOff, _ := json.Marshal(kernel.PlanDoc{})
	if strings.Contains(string(jsOff), `"structure"`) {
		t.Fatalf("knob off → the doc must not carry a structure key: %s", jsOff)
	}
	line := kernel.StructureLogLine(m, "NY")
	if !strings.HasPrefix(line, "🗺 structure @NY: D=") || !strings.Contains(line, "zones=1") {
		t.Fatalf("log line: %s", line)
	}
	if kernel.StructureBootLine(at.structureMapEnabled()) != "🗺 structure: on(D/4h/1h)" {
		t.Fatalf("boot line: %s", kernel.StructureBootLine(at.structureMapEnabled()))
	}
}
