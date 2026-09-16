package kernel

import (
	"math"
	"strings"
	"testing"
	"time"

	"nofx/market"
)

// fvgBars builds a quiet 1m series then a gap pattern at the tail.
// dir: "long" → bullish 3-candle (low[i] > high[i+2]); "short" mirrored.
func fvgBars(start time.Time, dir string) []market.Kline {
	out := make([]market.Kline, 0, 90)
	t := start
	// 75 minutes of quiet 1pt-range bars → small 5m ATR.
	for i := 0; i < 75; i++ {
		out = append(out, market.Kline{
			OpenTime: t.UnixMilli(), Open: 100, High: 100.5, Low: 99.5, Close: 100, Volume: 10,
			CloseTime: t.Add(time.Minute).UnixMilli() - 1,
		})
		t = t.Add(time.Minute)
	}
	if dir == "long" {
		// bullish: newest candle's LOW above the oldest candle's HIGH → gap
		// [oldest.high, newest.low] = [100.5, 104]; impulse body = |105−101| = 4.
		out = append(out, market.Kline{OpenTime: t.UnixMilli(), Open: 100, High: 100.5, Low: 99.5, Close: 100, Volume: 10, CloseTime: t.Add(time.Minute).UnixMilli() - 1})
		t = t.Add(time.Minute)
		out = append(out, market.Kline{OpenTime: t.UnixMilli(), Open: 100, High: 100.5, Low: 99.6, Close: 100, Volume: 10, CloseTime: t.Add(time.Minute).UnixMilli() - 1})
		t = t.Add(time.Minute)
		out = append(out, market.Kline{OpenTime: t.UnixMilli(), Open: 101, High: 106, Low: 104, Close: 105, Volume: 40, CloseTime: t.Add(time.Minute).UnixMilli() - 1})
		t = t.Add(time.Minute)
		// settle above the gap
		out = append(out, market.Kline{OpenTime: t.UnixMilli(), Open: 104, High: 104.5, Low: 103, Close: 104, Volume: 10, CloseTime: t.Add(time.Minute).UnixMilli() - 1})
	} else {
		// bearish mirrored: newest candle's HIGH below the oldest candle's LOW
		// → gap [newest.high, oldest.low] = [103, 105.2]; impulse body = |97−103| = 6.
		out = append(out, market.Kline{OpenTime: t.UnixMilli(), Open: 105, High: 105.6, Low: 105.2, Close: 105.4, Volume: 10, CloseTime: t.Add(time.Minute).UnixMilli() - 1})
		t = t.Add(time.Minute)
		out = append(out, market.Kline{OpenTime: t.UnixMilli(), Open: 105, High: 105.6, Low: 105.3, Close: 105.5, Volume: 10, CloseTime: t.Add(time.Minute).UnixMilli() - 1})
		t = t.Add(time.Minute)
		out = append(out, market.Kline{OpenTime: t.UnixMilli(), Open: 103, High: 103, Low: 96.5, Close: 97, Volume: 40, CloseTime: t.Add(time.Minute).UnixMilli() - 1})
		t = t.Add(time.Minute)
		out = append(out, market.Kline{OpenTime: t.UnixMilli(), Open: 99, High: 99.5, Low: 98, Close: 99, Volume: 10, CloseTime: t.Add(time.Minute).UnixMilli() - 1})
	}
	return out
}

func fvgDoc(dir string) *PlanDoc {
	lo, hi := 100.5, 104.0
	if dir == "short" {
		lo, hi = 103.0, 105.2
	}
	return &PlanDoc{
		Levels: []PlanLevel{{Price: 101.5, Label: "PDL"}, {Price: 106, Label: "PDH"}},
		Scenarios: []PlanScenario{{
			ID: "S1", Condition: "fvg_entry", Direction: dir, Trigger: "retrace into the gap",
			Invalid: "close through the distal edge",
			Quality: "A",
			Fvg: &PlanFvgEntry{Lo: lo, Hi: hi, CE: FvgCe(lo, hi), EntryMode: "edge",
				OriginLevel: "PDL", Direction: dir},
		}},
	}
}

// TestFvgValidateGolden covers the 3-candle math BOTH directions: a declared
// gap that matches the stored bars passes; CE is recomputed.
func TestFvgValidateGolden(t *testing.T) {
	start := time.Date(2026, 8, 26, 14, 0, 0, 0, time.UTC)
	for _, dir := range []string{"long", "short"} {
		bars := fvgBars(start, dir)
		doc := fvgDoc(dir)
		now := time.UnixMilli(bars[len(bars)-1].CloseTime).Add(time.Minute)
		if err := ValidateFvgEntryScenarios(doc, bars, "MNQ", map[string]bool{"PDL": true, "PDH": true}, now); err != nil {
			t.Fatalf("%s golden must validate: %v", dir, err)
		}
	}
}

// TestFvgValidateRejects — fake gap, weak displacement, missing origin, ce on a
// narrow gap, and a lying ce all fail the validator.
func TestFvgValidateRejects(t *testing.T) {
	start := time.Date(2026, 8, 26, 14, 0, 0, 0, time.UTC)
	bars := fvgBars(start, "long")
	now := time.UnixMilli(bars[len(bars)-1].CloseTime).Add(time.Minute)
	origins := map[string]bool{"PDL": true, "PDH": true}

	fake := fvgDoc("long")
	fake.Scenarios[0].Fvg.Lo, fake.Scenarios[0].Fvg.Hi = 120, 122 // no such gap
	fake.Scenarios[0].Fvg.CE = 121
	if err := ValidateFvgEntryScenarios(fake, bars, "MNQ", origins, now); err == nil || !strings.Contains(err.Error(), "fake/stale") {
		t.Fatalf("fake gap must fail with fake/stale, got %v", err)
	}

	weak := fvgDoc("long")
	// Claim a tiny displacement: validator recomputes ~4.0 body / ATR5 ≈ 2+ → fails the floor.
	weak.Scenarios[0].Fvg.DisplacementATR = 0.2
	if err := ValidateFvgEntryScenarios(weak, bars, "MNQ", origins, now); err == nil || !strings.Contains(err.Error(), "displacement_atr") {
		t.Fatalf("lying displacement must fail, got %v", err)
	}

	noOrigin := fvgDoc("long")
	noOrigin.Scenarios[0].Fvg.OriginLevel = "GHOST"
	if err := ValidateFvgEntryScenarios(noOrigin, bars, "MNQ", origins, now); err == nil || !strings.Contains(err.Error(), "origin_level") {
		t.Fatalf("missing origin must fail, got %v", err)
	}

	ceNarrow := fvgDoc("long")
	ceNarrow.Scenarios[0].Fvg.EntryMode = "ce" // gap 2.5 < 20 → ce invalid
	if err := ValidateFvgEntryScenarios(ceNarrow, bars, "MNQ", origins, now); err == nil || !strings.Contains(err.Error(), "entry_mode=ce") {
		t.Fatalf("ce on narrow gap must fail, got %v", err)
	}

	ceLie := fvgDoc("long")
	ceLie.Scenarios[0].Fvg.CE = 105 // midpoint is 102.75
	if err := ValidateFvgEntryScenarios(ceLie, bars, "MNQ", origins, now); err == nil || !strings.Contains(err.Error(), "ce") {
		t.Fatalf("lying ce must fail, got %v", err)
	}
}

// TestFvgConfirmStates covers ABOVE / IN_ZONE (edge+ce MET rules) /
// FILLED_INVALID and the touch numbering (freshness demote signal).
func TestFvgConfirmStates(t *testing.T) {
	f := &PlanFvgEntry{Lo: 101.5, Hi: 104, CE: 102.75, EntryMode: "edge", Direction: "long"}
	base := time.Date(2026, 8, 26, 14, 0, 0, 0, time.UTC)
	bars := touchBars(base, [][5]float64{
		{106, 106.5, 105.5, 106, 10}, // ABOVE the gap
		{103, 103.5, 102.5, 103, 10}, // IN ZONE → touch #1
		{103, 103.5, 102.8, 103.2, 10},
		{102, 102.5, 101.8, 102, 10},
		{101, 101.5, 100.5, 101, 10}, // closes through distal 101.5 → FILLED_INVALID
	})
	since := base.UnixMilli()
	v := EvaluateFvgEntry(bars, f, since, base.Add(1*time.Minute).UnixMilli())
	if v.State != "ABOVE" {
		t.Fatalf("state = %s, want ABOVE", v.State)
	}
	v = EvaluateFvgEntry(bars, f, since, base.Add(3*time.Minute).UnixMilli())
	if v.State != "IN_ZONE" || !v.Met || v.TouchNumber != 1 {
		t.Fatalf("in-zone = %s met=%v touch=%d, want IN_ZONE/true/1", v.State, v.Met, v.TouchNumber)
	}
	v = EvaluateFvgEntry(bars, f, since, base.Add(5*time.Minute).UnixMilli())
	if v.State != "FILLED_INVALID" {
		t.Fatalf("state = %s, want FILLED_INVALID (closed through distal)", v.State)
	}
	// ce mode: MET only near the midpoint.
	fce := &PlanFvgEntry{Lo: 98, Hi: 122, CE: 110, EntryMode: "ce", Direction: "long"}
	barsCe := touchBars(base, [][5]float64{
		{105, 106, 104, 105, 10}, // in zone, 5pt off CE of a 24pt gap → band 2.4 → not met
		{110, 111, 109, 110, 10}, // AT CE → met
	})
	v = EvaluateFvgEntry(barsCe, fce, since, base.Add(1*time.Minute).UnixMilli())
	if v.State != "IN_ZONE" || v.Met {
		t.Fatalf("ce off-midpoint must be IN_ZONE and NOT met, got %s met=%v", v.State, v.Met)
	}
	v = EvaluateFvgEntry(barsCe, fce, since, base.Add(2*time.Minute).UnixMilli())
	if !v.Met {
		t.Fatalf("ce at midpoint must be MET")
	}
}

// TestFvgMinSLInteraction — the scenario anchor IS the distal edge, so an SL
// beyond distal +2 ticks satisfies the level-clearance leg naturally, and a
// too-narrow gap still trips the ATR leg.
func TestFvgMinSLInteraction(t *testing.T) {
	s := PlanScenario{ID: "S1", Condition: "fvg_entry", Direction: "long",
		Fvg: &PlanFvgEntry{Lo: 101.5, Hi: 104, Direction: "long"}}
	anchor, ok := ScenarioAnchor(s, []PlanLevel{{Price: 104, Label: "PDH"}})
	if !ok || math.Abs(anchor-101.5) > 0.01 {
		t.Fatalf("fvg anchor must be the distal edge 101.5, got %.2f ok=%v", anchor, ok)
	}
	// The ATR leg: an SL 2 ticks beyond the distal edge that is closer than
	// 1.0×ATR from the entry still refuses (the gate is anchor-agnostic on
	// distance). Entry at CE 102.75, SL at 101.5+0.5=102.0 → dist 0.75 < 1.0×ATR
	// with ATR 2.0 → refuse.
	if refuse, _ := MinSLVerdict("open_long", 0.75, 2.0, 1.0); !refuse {
		t.Fatal("too-tight SL (dist < 1×ATR) must refuse")
	}
	if refuse, _ := MinSLVerdict("open_long", 2.75, 2.0, 1.0); refuse {
		t.Fatal("wide-enough SL must pass the ATR leg")
	}
}

// TestFvgPromptRender — the planner contract carries the playbook, and the
// executor render quotes the band + state.
func TestFvgPromptRender(t *testing.T) {
	p := BuildPlannerPrompt(PlannerInput{TradeDate: "2026-08-26", Session: "NY", Price: 103, DATR: 100,
		MaxLevels: 8, ScenarioCap: 3,
		Levels: []ScoredLevel{{DetectedLevel: DetectedLevel{Kind: KindPDH, Price: 106, Label: "PDH"}, Grade: "A", Distance: 3}}})
	for _, want := range []string{"FVG ENTRY (fvg_entry)", "fvg_lo", "DISTAL edge", "MSS research"} {
		if !strings.Contains(p, want) {
			t.Fatalf("planner contract missing %q", want)
		}
	}
	doc := fvgDoc("long")
	bars := fvgBars(time.Date(2026, 8, 26, 14, 0, 0, 0, time.UTC), "long")
	line := RenderFvgEntryLines(*doc, bars, 0, time.UnixMilli(bars[len(bars)-1].CloseTime).Add(time.Minute).UnixMilli())
	for _, want := range []string{"S1 fvg_entry", "gap 100.50–104.00", "CE 102.25"} {
		if !strings.Contains(line, want) {
			t.Fatalf("fvg line %q missing %q", line, want)
		}
	}
}

// TestFvgRoleWarn — an fvg_entry anchored on a liquidity_break origin carries
// the sweep-reclaim caution WARN (no exemption).
func TestFvgRoleWarn(t *testing.T) {
	doc := fvgDoc("long")
	doc.Scenarios[0].Fvg.OriginLevel = "ONH"
	ms := RoleMismatches(doc)
	if len(ms) != 1 || !strings.Contains(ms[0], "liquidity_break") || !strings.Contains(ms[0], "sweep-reclaim") {
		t.Fatalf("fvg liquidity origin must WARN, got %v", ms)
	}
}
