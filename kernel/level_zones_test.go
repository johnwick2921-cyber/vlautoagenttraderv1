package kernel

import (
	"encoding/json"
	"nofx/market"
	"os"
	"strings"
	"testing"
	"time"
)

// One frozen NY clock; these are actual detector outputs, not reconstructed chart labels.
func TestLevelZonesOwnerSnapshot(t *testing.T) {
	b, err := os.ReadFile("../docs/superpowers/reports/2026-09-11-level-zones-evidence/snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Now        string  `json:"now"`
		ATR        float64 `json:"atr5m"`
		Price      float64 `json:"price"`
		Candidates []struct {
			ID     int           `json:"id"`
			Origin DetectedLevel `json:"raw_origin"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(b, &fixture); err != nil {
		t.Fatal(err)
	}
	now, err := time.Parse(time.RFC3339Nano, fixture.Now)
	if err != nil {
		t.Fatal(err)
	}
	var raw []DetectedLevel
	for _, r := range fixture.Candidates {
		raw = append(raw, r.Origin)
	}
	before, _ := json.Marshal(raw)
	// Recorded defining candles supplement output-only metadata absent from the old JSON.
	widthData, err := os.ReadFile("../docs/superpowers/reports/2026-09-11-level-zones-evidence/pivot-width-evidence.json")
	if err != nil {
		t.Fatal(err)
	}
	var evidence struct {
		Rows []struct {
			TF  string                                   `json:"tf"`
			ATR float64                                  `json:"atr"`
			Bar struct{ Open, High, Low, Close float64 } `json:"bar"`
		} `json:"rows"`
	}
	if err = json.Unmarshal(widthData, &evidence); err != nil {
		t.Fatal(err)
	}
	inputs := map[string]ZoneWidthInput{}
	for _, l := range raw {
		if l.Kind == KindSWGH && l.Price == 29475 {
			for _, r := range evidence.Rows {
				if r.TF == l.TF {
					wick := ZonePivotWick(market.Kline{Open: r.Bar.Open, High: r.Bar.High, Low: r.Bar.Low, Close: r.Bar.Close}, true)
					inputs[ZoneSourceKey(l)] = ZoneWidthInput{ATR: r.ATR, Wick: &wick}
				}
			}
		}
	}
	view := BuildLevelZones(raw, fixture.Price, fixture.ATR, inputs, DefaultZoneOptions(12), now)
	after, _ := json.Marshal(raw)
	if string(before) != string(after) {
		t.Fatal("render changed detector anchors/bounds")
	}
	if view.Broad != 164 || len(view.Zones) != 498 || view.Merged != 25 {
		t.Fatalf("native-bound census: %+v", view.Counts())
	}
	if view.WidestMerged > fixture.ATR {
		t.Fatalf("width %g", view.WidestMerged)
	}
	var pair, daily bool
	for _, z := range view.Zones {
		if len(z.Sources) > 1 && (z.Incomplete || z.Lo == nil || *z.Hi-*z.Lo > fixture.ATR) {
			t.Fatal("merge width must be known and under cap")
		}
		a, b := false, false
		for _, s := range z.Sources {
			if s.Price == 29475 && s.Kind == KindSWGH {
				a = a || s.TF == "5m"
				b = b || s.TF == "15m"
			}
			if s.Price == 29006.625 && s.Kind == KindDemand {
				daily = z.Broad && len(z.Sources) == 1
			}
		}
		pair = pair || a && b
	}
	if !pair || !daily {
		t.Fatalf("pair=%v daily separate=%v", pair, daily)
	}
	prompt := BuildPlannerPrompt(PlannerInput{Now: now, TradeDate: "2026-09-11", Session: "NY", Price: fixture.Price, Zones: &view})
	t.Logf("%s; rendered bytes=%d", view.Counts(), len(view.Render()))
	for _, name := range []string{"SWG-H·5m", "SWG-H·15m", "Demand·1d", "28810.75", "29202.50"} {
		if !strings.Contains(prompt, name) {
			t.Errorf("model table lost %s", name)
		}
	}
}

func TestLevelZonesSingleShortlistWithLegacyIdentityReferences(t *testing.T) {
	now := time.Date(2026, 9, 11, 15, 30, 0, 0, time.UTC)
	l := DetectedLevel{Kind: KindRound, Price: 100, Lo: 100, Hi: 100, Label: "RN 100"}
	v := BuildLevelZones([]DetectedLevel{l}, 100, 24, nil, DefaultZoneOptions(12), now)
	p := BuildPlannerPrompt(PlannerInput{Now: now, Session: "NY", Price: 100, ATR5m: 24, Levels: []ScoredLevel{{DetectedLevel: l}}, Zones: &v})
	if strings.Count(p, "ENTRY SHORTLIST") != 1 {
		t.Fatal("two incompatible shortlist orderings rendered")
	}
	if !strings.Contains(p, "id=NULL") {
		t.Fatal("legacy identity references discarded")
	}
}

func TestLevelZonesRankingIgnoresHTFAndKeepsAllReferences(t *testing.T) {
	now := time.Date(2026, 9, 11, 15, 30, 0, 0, time.UTC)
	raw := []DetectedLevel{{Kind: KindSWGL, TF: "5m", Price: 100, Lo: 100, Hi: 100, Label: "near"}, {Kind: KindSWGL, TF: "1d", HTF: true, Price: 200, Lo: 200, Hi: 200, Label: "far"}}
	count := 100
	inputs := map[string]ZoneWidthInput{ZoneSourceKey(raw[1]): {PriorTouches: &count}}
	o := DefaultZoneOptions(1)
	v := BuildLevelZones(raw, 100, 24, inputs, o, now)
	if len(v.Zones) != 2 || !v.Zones[1].Shortlisted {
		t.Fatal("touch term not connected to shortlist or loser dropped")
	}
	raw[1].HTF = false
	raw[1].TF = "5m"
	inputs[ZoneSourceKey(raw[1])] = ZoneWidthInput{PriorTouches: &count}
	w := BuildLevelZones(raw, 100, 24, inputs, o, now)
	if *v.Zones[1].RankValue != *w.Zones[1].RankValue {
		t.Fatal("HTF affects order")
	}
	inputs = nil
	w = BuildLevelZones(raw, 100, 24, inputs, o, now)
	if !w.Zones[0].Shortlisted {
		t.Fatal("distance term not connected")
	}
}

func TestLevelZonesBootUsesResolvedParameters(t *testing.T) {
	t.Setenv("LEVEL_ZONE_MERGE_ATR", "0.25")
	t.Setenv("LEVEL_ZONE_MAX_WIDTH_ATR", "0.75")
	t.Setenv("LEVEL_ZONE_BROAD_ATR", "0.6")
	line := ZoneBootLine(ResolveZoneOptions(12))
	for _, s := range []string{"merge=0.25", "max-width=0.75", "broad>0.6", "cap=12", "n/a", "BuildLevelZones"} {
		if !strings.Contains(line, s) {
			t.Errorf("missing %s: %s", s, line)
		}
	}
}

func TestLevelZonesInputsExcludeFutureBarsAndRequireFormation(t *testing.T) {
	now := time.Date(2026, 9, 11, 15, 30, 0, 0, time.UTC)
	start := now.Add(-30 * time.Minute).UnixMilli()
	var bars []market.Kline
	for j := 0; j < 30; j++ {
		bars = append(bars, market.Kline{OpenTime: start + int64(j)*60_000, CloseTime: start + int64(j+1)*60_000 - 1, Open: 100, High: 102, Low: 98, Close: 100 + float64(j%2)})
	}
	wick := 2.
	birth := bars[1].CloseTime
	l := DetectedLevel{Kind: KindSWGL, TF: "1m", Price: 100, ZoneDefiningWick: &wick, FormedCloseMs: &birth}
	raw := []DetectedLevel{l}
	a := LevelZoneInputs(raw, map[string][]market.Kline{"1m": bars}, now)[ZoneSourceKey(l)]
	if a.ATR <= 0 || a.Wick == nil || a.PriorTouches == nil {
		t.Fatalf("known inputs missing: %+v", a)
	}
	bars = append(bars, market.Kline{OpenTime: now.UnixMilli(), CloseTime: now.Add(time.Minute).UnixMilli(), Open: 100, High: 1000, Low: 1, Close: 800})
	b := LevelZoneInputs(raw, map[string][]market.Kline{"1m": bars}, now)[ZoneSourceKey(l)]
	if a.ATR != b.ATR || *a.PriorTouches != *b.PriorTouches {
		t.Fatal("future bar entered read")
	}
	l.FormedCloseMs = nil
	c := LevelZoneInputs([]DetectedLevel{l}, map[string][]market.Kline{"1m": bars}, now)[ZoneSourceKey(l)]
	if c.PriorTouches != nil {
		t.Fatal("unknown birth fabricated a count")
	}
}

func TestLevelZonesCompatibilityAndNoChain(t *testing.T) {
	now := time.Date(2026, 9, 11, 15, 30, 0, 0, time.UTC) // 10:30 NY session, CT
	raw := []DetectedLevel{{Kind: KindEQH, Price: 100, Lo: 99.5, Hi: 100.5, Label: "a"}, {Kind: KindEQH, Price: 112, Lo: 111.5, Hi: 112.5, Label: "b"}, {Kind: KindEQH, Price: 124, Lo: 123.5, Hi: 124.5, Label: "c"}}
	opts := DefaultZoneOptions(12)
	v := BuildLevelZones(raw, 110, 24, nil, opts, now)
	if len(v.Zones) != 2 {
		t.Fatalf("non-transitive clusters=%d", len(v.Zones))
	}
	opts.MergeATR = .25
	if v = BuildLevelZones(raw, 110, 24, nil, opts, now); len(v.Zones) != 3 {
		t.Fatal("m=.25 must not merge 12pt apart")
	}
	opts = DefaultZoneOptions(12)
	opts.MaxWidthATR = .25
	if v = BuildLevelZones(raw, 110, 24, nil, opts, now); len(v.Zones) != 3 {
		t.Fatal("width cap ignored")
	}
}

func TestLevelZonesWidthFamilyAndUnknown(t *testing.T) {
	now := time.Date(2026, 9, 11, 15, 30, 0, 0, time.UTC)
	raw := []DetectedLevel{{Kind: KindSWGL, TF: "1d", Price: 100, Lo: 100, Hi: 100, Label: "daily"}, {Kind: KindSWGL, TF: "4h", Price: 100, Lo: 100, Hi: 100, Label: "4h"}, {Kind: KindSWGL, TF: "1h", Price: 100, Lo: 100, Hi: 100, Label: "1h"}, {Kind: KindPOC, Price: 100, Lo: 100, Hi: 100, Label: "POC"}, {Kind: KindRound, Price: 100, Lo: 100, Hi: 100, Label: "round"}, {Kind: KindPDH, Price: 100, Lo: 100, Hi: 100, Label: "PDH"}, {Kind: KindOB, Price: 100, Lo: 98, Hi: 102, Label: "OB"}}
	wick := 1.0
	known := map[string]ZoneWidthInput{}
	for _, l := range raw {
		known[ZoneSourceKey(l)] = ZoneWidthInput{ATR: 4, Wick: &wick}
	}
	for n, want := range map[int]int{3: 1, 4: 2, 5: 3, 7: 3} {
		part := BuildLevelZones(raw[:n], 100, 24, known, DefaultZoneOptions(12), now)
		if len(part.Zones) != 1 || part.Zones[0].FamilyCount != want {
			t.Fatalf("%d sources: want %d independent families, got %+v", n, want, part.Zones)
		}
	}
	v := BuildLevelZones(raw, 100, 24, known, DefaultZoneOptions(12), now)
	if len(v.Zones) != 1 || v.Zones[0].FamilyCount != 3 || len(v.Zones[0].Families) != 5 || len(v.Zones[0].Sources) != 7 {
		t.Fatalf("families/names: %+v", v.Zones)
	}
	if v.NullWidths != 0 {
		t.Fatalf("missing wick/ATR must be NULL, got %d", v.NullWidths)
	}
	if *v.Zones[0].Lo != 98 || *v.Zones[0].Hi != 102 {
		t.Fatal("native OB bounds lost")
	}
	w := 4.0
	widths := map[string]ZoneWidthInput{ZoneSourceKey(raw[0]): {ATR: 20, Wick: &w}}
	v = BuildLevelZones(raw[:1], 100, 24, widths, DefaultZoneOptions(12), now)
	if *v.Zones[0].Lo != 95 || *v.Zones[0].Hi != 105 {
		t.Fatal("max(wick,k*ATR)/2 not applied")
	}
}

func TestLevelZonesUnknownWidthCannotPassCompatibility(t *testing.T) {
	now := time.Date(2026, 9, 11, 15, 30, 0, 0, time.UTC)
	raw := []DetectedLevel{{Kind: KindSWGH, Price: 100, Lo: 100, Hi: 100, Label: "unknown"}, {Kind: KindOB, Price: 100, Lo: 98, Hi: 102, Label: "known"}}
	v := BuildLevelZones(raw, 100, 24, nil, DefaultZoneOptions(12), now)
	if len(v.Zones) != 2 || v.NullWidths != 1 || v.Merged != 0 {
		t.Fatal("unknown width passed the maximum-width condition")
	}
	if ZonePivotWick(market.Kline{Open: 29434, High: 29475, Low: 29383.25, Close: 29472}, true) != 3 {
		t.Fatal("opposite wick widened a swing high")
	}
}
